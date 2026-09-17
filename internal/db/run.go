package db

//go:generate go tool mockery --name Runs --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// Runs encapsulates the logic to access runs from the database
type Runs interface {
	GetRunByID(ctx context.Context, id string) (*models.Run, error)
	GetRunByTRN(ctx context.Context, trnValue string) (*models.Run, error)
	GetRunByNodeID(ctx context.Context, nodeID string) (*models.Run, error)
	GetWorkspaceIDForRun(ctx context.Context, id string) (string, error)
	CreateRun(ctx context.Context, run *models.Run) (*models.Run, error)
	UpdateRun(ctx context.Context, run *models.Run, nodeIDs ...string) (*models.Run, error)
	GetRuns(ctx context.Context, input *GetRunsInput) (*RunsResult, error)
	DeleteRunBatch(ctx context.Context, input *DeleteRunBatchInput) ([]string, error)
}

// RunSortableField represents the fields that a workspace can be sorted by
type RunSortableField string

// GroupSortableField constants
const (
	RunSortableFieldCreatedAtAsc  RunSortableField = "CREATED_AT_ASC"
	RunSortableFieldCreatedAtDesc RunSortableField = "CREATED_AT_DESC"
	RunSortableFieldUpdatedAtAsc  RunSortableField = "UPDATED_AT_ASC"
	RunSortableFieldUpdatedAtDesc RunSortableField = "UPDATED_AT_DESC"
)

func (r RunSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch r {
	case RunSortableFieldCreatedAtAsc, RunSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "runs", Col: "created_at"}
	case RunSortableFieldUpdatedAtAsc, RunSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "runs", Col: "updated_at"}
	default:
		return nil
	}
}

func (r RunSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(r), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// DeleteRunBatchInput configures safety guards for batch run deletion.
type DeleteRunBatchInput struct {
	// Runs is the set of runs to delete.
	Runs []*models.Run
}

// RunFilter contains the supported fields for filtering Run resources
type RunFilter struct {
	TimeRangeStart *time.Time
	UpdatedBefore  *time.Time
	WorkspaceID    *string
	GroupID        *string
	// RootNamespaceMemberships limits results to runs in workspaces at or under one of the
	// caller's root member namespace paths. Non-nil empty = no memberships (matches nothing);
	// nil = no membership filter.
	RootNamespaceMemberships []models.MembershipNamespace
	Statuses                 []models.RunStatus
	RunIDs                   []string
	NodeIDs                  []string
	WorkspaceAssessment      *bool
	IncludeNestedRuns        *bool
	HasStateVersion          *bool
}

// GetRunsInput is the input for listing runs
type GetRunsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *RunSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *RunFilter
}

// RunsResult contains the response data and page information
type RunsResult struct {
	PageInfo *pagination.PageInfo
	Runs     []*models.Run
}

type nodeIDFilterSet map[string]struct{}

// includes returns true when the set is empty (update all) or contains the given ID.
func (s nodeIDFilterSet) includes(id string) bool {
	if len(s) == 0 {
		return true
	}
	_, exists := s[id]
	return exists
}

// runNode is the internal DB representation of a run_nodes row. All plan_* columns are nullable
// (only plan nodes populate them), so the type-specific fields are pointers.
type runNode struct {
	LatestJobID          *string
	ErrorMessage         *string
	TriggeredBy          *string
	Comment              *string
	CheckType            *string
	StageName            *string
	Policies             []byte
	MessagesSummary      []byte
	CacheObjectStoreKey  *string
	JSONObjectStoreKey   *string
	DiffObjectStoreKey   *string
	HasChanges           *bool
	DiffSize             *int
	ResourceAdditions    *int32
	ResourceChanges      *int32
	ResourceDestructions *int32
	ResourceImports      *int32
	ResourceDrift        *int32
	OutputAdditions      *int32
	OutputChanges        *int32
	OutputDestructions   *int32
	ID                   string
	RunID                string
	Type                 string
	Status               string
}

// runNodeInstance represents a single run_nodes row of a given kind on a run. A kind may yield
// zero, one, or many instances (a run has at most one plan/apply, but any number of policy
// checks), which is how the single run_nodes table backs both singleton and collection nodes.
type runNodeInstance struct {
	// id is the instance's current node id ("" if not yet assigned).
	id string
	// ensureID assigns a new id if the instance has none and returns the id (create path).
	ensureID func() string
	// contentColumns returns all writable content columns for this instance (status,
	// latest_job_id, error_message and the type-specific columns). Structural columns (id,
	// run_id, type, sort_order) are added by the caller.
	contentColumns func() goqu.Record
	// sortOrder is the row's persisted position, written once on create (it never changes). It
	// places a run's task stages and their checks in execution order so hydration can return them
	// correctly ordered via a single ORDER BY, without an in-memory re-sort.
	sortOrder int
}

// runNodeKind centralizes everything the db layer needs to persist and load one run-node
// type. Adding a node type means adding one entry here; the create/update/read/hydrate paths
// all dispatch through this table instead of per-type branches. instances enumerates the run's
// nodes of this kind (0, 1, or many) operating on the concrete run.Plan / run.Apply /
// run.TaskStages (and the checks they own) fields directly (no models.RunNode interface, no type assertions);
// load builds a model node from a scanned wide row and assigns/appends it onto the run.
type runNodeKind struct {
	typeName  string
	instances func(run *models.Run) []runNodeInstance
	load      func(run *models.Run, n *runNode) error
}

var runNodeKinds = []runNodeKind{
	{
		typeName: "plan",
		instances: func(run *models.Run) []runNodeInstance {
			p := &run.Plan
			return []runNodeInstance{{
				id: p.ID,
				ensureID: func() string {
					if p.ID == "" {
						p.ID = newResourceID()
					}
					return p.ID
				},
				contentColumns: func() goqu.Record {
					return goqu.Record{
						"status":                      string(p.Status),
						"latest_job_id":               p.LatestJobID,
						"error_message":               p.ErrorMessage,
						"plan_has_changes":            p.HasChanges,
						"plan_diff_size":              p.DiffSize,
						"plan_resource_additions":     p.Summary.ResourceAdditions,
						"plan_resource_changes":       p.Summary.ResourceChanges,
						"plan_resource_destructions":  p.Summary.ResourceDestructions,
						"plan_resource_imports":       p.Summary.ResourceImports,
						"plan_resource_drift":         p.Summary.ResourceDrift,
						"plan_output_additions":       p.Summary.OutputAdditions,
						"plan_output_changes":         p.Summary.OutputChanges,
						"plan_output_destructions":    p.Summary.OutputDestructions,
						"plan_cache_object_store_key": p.CacheObjectStoreKey,
						"plan_json_object_store_key":  p.JSONObjectStoreKey,
						"plan_diff_object_store_key":  p.DiffObjectStoreKey,
					}
				},
				sortOrder: sortOrderPlan,
			}}
		},
		load: func(run *models.Run, n *runNode) error {
			plan := models.Plan{
				ID:           n.ID,
				Status:       models.PlanStatus(n.Status),
				LatestJobID:  n.LatestJobID,
				ErrorMessage: n.ErrorMessage,
				Summary: models.PlanSummary{
					ResourceAdditions:    ptr.ToInt32(n.ResourceAdditions),
					ResourceChanges:      ptr.ToInt32(n.ResourceChanges),
					ResourceDestructions: ptr.ToInt32(n.ResourceDestructions),
					ResourceImports:      ptr.ToInt32(n.ResourceImports),
					ResourceDrift:        ptr.ToInt32(n.ResourceDrift),
					OutputAdditions:      ptr.ToInt32(n.OutputAdditions),
					OutputChanges:        ptr.ToInt32(n.OutputChanges),
					OutputDestructions:   ptr.ToInt32(n.OutputDestructions),
				},
			}
			if n.HasChanges != nil {
				plan.HasChanges = *n.HasChanges
			}
			if n.DiffSize != nil {
				plan.DiffSize = *n.DiffSize
			}
			plan.CacheObjectStoreKey = n.CacheObjectStoreKey
			plan.JSONObjectStoreKey = n.JSONObjectStoreKey
			plan.DiffObjectStoreKey = n.DiffObjectStoreKey
			run.Plan = plan
			return nil
		},
	},
	{
		typeName: "apply",
		instances: func(run *models.Run) []runNodeInstance {
			if run.Apply == nil {
				return nil
			}
			a := run.Apply
			return []runNodeInstance{{
				id: a.ID,
				ensureID: func() string {
					if a.ID == "" {
						a.ID = newResourceID()
					}
					return a.ID
				},
				contentColumns: func() goqu.Record {
					return goqu.Record{
						"status":             string(a.Status),
						"latest_job_id":      a.LatestJobID,
						"error_message":      a.ErrorMessage,
						"apply_triggered_by": nullableString(a.TriggeredBy),
						"apply_comment":      nullableString(a.Comment),
					}
				},
				sortOrder: sortOrderApply,
			}}
		},
		load: func(run *models.Run, n *runNode) error {
			apply := &models.Apply{
				ID:           n.ID,
				Status:       models.ApplyStatus(n.Status),
				LatestJobID:  n.LatestJobID,
				ErrorMessage: n.ErrorMessage,
			}
			if n.TriggeredBy != nil {
				apply.TriggeredBy = *n.TriggeredBy
			}
			if n.Comment != nil {
				apply.Comment = *n.Comment
			}
			run.Apply = apply
			return nil
		},
	},
	{
		typeName: "task_stage",
		instances: func(run *models.Run) []runNodeInstance {
			out := make([]runNodeInstance, 0, len(run.TaskStages))
			for _, stage := range run.TaskStages {
				stage := stage
				out = append(out, runNodeInstance{
					id: stage.ID,
					ensureID: func() string {
						if stage.ID == "" {
							stage.ID = newResourceID()
						}
						return stage.ID
					},
					contentColumns: func() goqu.Record {
						return goqu.Record{
							"status":     string(stage.Status),
							"stage_name": string(stage.StageName),
						}
					},
					sortOrder: sortOrderTaskStage,
				})
			}
			return out
		},
		load: func(run *models.Run, n *runNode) error {
			var stageName models.RunTaskStageName
			if n.StageName != nil {
				stageName = models.RunTaskStageName(*n.StageName)
			}
			run.TaskStages = append(run.TaskStages, &models.RunTaskStage{
				StageName: stageName,
				ID:        n.ID,
				Status:    models.RunTaskStageStatus(n.Status),
			})
			return nil
		},
	},
	{
		typeName: "policy_check",
		instances: func(run *models.Run) []runNodeInstance {
			checks := run.AllPolicyChecks()
			out := make([]runNodeInstance, 0, len(checks))
			for _, check := range checks {
				check := check
				out = append(out, runNodeInstance{
					id: check.ID,
					ensureID: func() string {
						if check.ID == "" {
							check.ID = newResourceID()
						}
						return check.ID
					},
					contentColumns: func() goqu.Record {
						// PolicyCheckPolicy fields are all strings/enums and the summary is
						// strings and a bool, so marshal cannot fail; a nil result (empty
						// slice) is written as SQL NULL.
						var policies []byte
						if len(check.Policies) > 0 {
							policies, _ = json.Marshal(check.Policies)
						}
						var messagesSummary []byte
						if check.MessagesSummary != nil {
							messagesSummary, _ = json.Marshal(check.MessagesSummary)
						}
						return goqu.Record{
							"status":                        string(check.Status),
							"latest_job_id":                 check.LatestJobID,
							"policy_check_type":             string(check.CheckType),
							"stage_name":                    string(check.StageName),
							"policy_check_policies":         policies,
							"policy_check_messages_summary": messagesSummary,
						}
					},
					sortOrder: sortOrderPolicyCheck,
				})
			}
			return out
		},
		load: func(run *models.Run, n *runNode) error {
			check := &models.PolicyCheck{
				ID:          n.ID,
				Status:      models.PolicyCheckStatus(n.Status),
				LatestJobID: n.LatestJobID,
			}
			if n.CheckType != nil {
				check.CheckType = models.PolicyKind(*n.CheckType)
			}
			if n.StageName != nil {
				check.StageName = models.RunTaskStageName(*n.StageName)
			}
			if len(n.Policies) > 0 {
				_ = json.Unmarshal(n.Policies, &check.Policies)
			}
			if len(n.MessagesSummary) > 0 {
				_ = json.Unmarshal(n.MessagesSummary, &check.MessagesSummary)
			}
			// The load order guarantees every task_stage row hydrates before any policy_check row (see
			// sortOrder constants), so the owning stage should already be present. A missing stage means
			// an inconsistent run graph (e.g. a check row whose stage row was lost); fail loudly rather
			// than dereferencing nil and panicking while reading the run.
			stage := run.TaskStageByStageName(check.StageName)
			if stage == nil {
				return errors.New("policy check %s references task stage %q that is not present on run %s",
					check.ID, check.StageName, run.Metadata.ID)
			}
			stage.PolicyChecks = append(stage.PolicyChecks, check)
			return nil
		},
	},
}

// run_nodes.sort_order positions. All task_stage rows share sortOrderTaskStage — the only
// invariant is that every task_stage row hydrates before every policy_check row (so that
// ensureTaskStage can find the stage when a check's load path runs). Plan and apply have no
// children, so their positions relative to task stages don't matter for correctness.
const (
	sortOrderTaskStage   = 0
	sortOrderPlan        = 1
	sortOrderApply       = 2
	sortOrderPolicyCheck = 3
)

var runNodeKindByType = func() map[string]runNodeKind {
	m := make(map[string]runNodeKind, len(runNodeKinds))
	for _, k := range runNodeKinds {
		m[k.typeName] = k
	}
	return m
}()

var runFieldList = append(
	metadataFieldList,
	"status",
	"is_destroy",
	"workspace_id",
	"configuration_version_id",
	"created_by",
	"module_source",
	"module_version",
	"module_digest",
	"force_canceled_by",
	"force_cancel_available_at",
	"force_canceled",
	"comment",
	"auto_apply",
	"terraform_version",
	"targets",
	"refresh",
	"refresh_only",
	"is_assessment_run",
	"variables_object_store_key",
	"has_advisory_failures",
	"annotations",
)

// runNodeColumns is the single ordered source of truth for run_nodes columns, shared by the SELECT
// field list (getRunNodeSelectFields) and the row scan (scanRunNode), so the two can never drift.
// dest returns the scan target for each column on the given runNode.
var runNodeColumns = []struct {
	name string
	dest func(n *runNode) any
}{
	{"id", func(n *runNode) any { return &n.ID }},
	{"run_id", func(n *runNode) any { return &n.RunID }},
	{"type", func(n *runNode) any { return &n.Type }},
	{"status", func(n *runNode) any { return &n.Status }},
	{"latest_job_id", func(n *runNode) any { return &n.LatestJobID }},
	{"error_message", func(n *runNode) any { return &n.ErrorMessage }},
	{"plan_has_changes", func(n *runNode) any { return &n.HasChanges }},
	{"plan_diff_size", func(n *runNode) any { return &n.DiffSize }},
	{"plan_resource_additions", func(n *runNode) any { return &n.ResourceAdditions }},
	{"plan_resource_changes", func(n *runNode) any { return &n.ResourceChanges }},
	{"plan_resource_destructions", func(n *runNode) any { return &n.ResourceDestructions }},
	{"plan_resource_imports", func(n *runNode) any { return &n.ResourceImports }},
	{"plan_resource_drift", func(n *runNode) any { return &n.ResourceDrift }},
	{"plan_output_additions", func(n *runNode) any { return &n.OutputAdditions }},
	{"plan_output_changes", func(n *runNode) any { return &n.OutputChanges }},
	{"plan_output_destructions", func(n *runNode) any { return &n.OutputDestructions }},
	{"apply_triggered_by", func(n *runNode) any { return &n.TriggeredBy }},
	{"apply_comment", func(n *runNode) any { return &n.Comment }},
	{"policy_check_type", func(n *runNode) any { return &n.CheckType }},
	{"stage_name", func(n *runNode) any { return &n.StageName }},
	{"policy_check_policies", func(n *runNode) any { return &n.Policies }},
	{"policy_check_messages_summary", func(n *runNode) any { return &n.MessagesSummary }},
	{"plan_cache_object_store_key", func(n *runNode) any { return &n.CacheObjectStoreKey }},
	{"plan_json_object_store_key", func(n *runNode) any { return &n.JSONObjectStoreKey }},
	{"plan_diff_object_store_key", func(n *runNode) any { return &n.DiffObjectStoreKey }},
}

type runs struct {
	dbClient *Client
}

// NewRuns returns an instance of the Run interface
func NewRuns(dbClient *Client) Runs {
	return &runs{dbClient: dbClient}
}

// GetRunByID returns a run by ID
func (r *runs) GetRunByID(ctx context.Context, id string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return r.getRun(ctx, goqu.Ex{"runs.id": id})
}

// GetWorkspaceIDForRun returns only the workspace_id for a run. It is faster than GetRunByID
// because it selects a single column and skips the namespaces join and run_nodes hydration.
func (r *runs) GetWorkspaceIDForRun(ctx context.Context, id string) (string, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceIDForRun")
	defer span.End()

	sql, args, err := toSQLWithTag("run.GetWorkspaceIDForRun", dialect.From("runs").
		Prepared(true).
		Select(goqu.C("workspace_id")).
		Where(goqu.Ex{"id": id}))
	if err != nil {
		return "", errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	var workspaceID string
	if err = r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...).Scan(&workspaceID); err != nil {
		if err == pgx.ErrNoRows {
			return "", errors.New("run with id %s not found", id,
				errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return "", ErrInvalidID
			}
		}
		return "", errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return workspaceID, nil
}

func (r *runs) GetRunByTRN(ctx context.Context, trnValue string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunByTRN")
	defer span.End()

	parsed, err := trn.TypeRun.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if !parsed.HasParent() {
		return nil, errors.New("a run TRN must have the workspace path and run GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return r.getRun(ctx, goqu.Ex{
		"runs.id":         gid.FromGlobalID(parsed.BaseName()),
		"namespaces.path": parsed.ParentPath(),
	})
}

func (r *runs) GetRunByNodeID(ctx context.Context, nodeID string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunByNodeID")
	defer span.End()

	sql, args, err := dialect.From("runs").
		Prepared(true).
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})).
		InnerJoin(goqu.T("run_nodes"), goqu.On(goqu.Ex{"runs.id": goqu.I("run_nodes.run_id")})).
		Where(goqu.Ex{"run_nodes.id": nodeID}).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	run, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if err := r.hydrateRunNodes(ctx, r.dbClient.getConnection(ctx), []*models.Run{run}); err != nil {
		return nil, err
	}

	return run, nil
}

func (r *runs) GetRuns(ctx context.Context, input *GetRunsInput) (*RunsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRuns")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	selectEx := dialect.From("runs").
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("workspaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("workspaces.id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")}))

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.RunIDs != nil {
			ex = ex.Append(goqu.I("runs.id").In(input.Filter.RunIDs))
		}

		if input.Filter.NodeIDs != nil {
			ex = ex.Append(goqu.I("runs.id").In(
				dialect.From("run_nodes").Select("run_id").Where(goqu.I("id").In(input.Filter.NodeIDs)),
			))
		}

		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("runs.workspace_id").Eq(*input.Filter.WorkspaceID))
		}

		if input.Filter.Statuses != nil {
			statuses := make([]string, len(input.Filter.Statuses))
			for i, status := range input.Filter.Statuses {
				statuses[i] = string(status)
			}
			ex = ex.Append(goqu.I("runs.status").In(statuses))
		}

		if input.Filter.GroupID != nil {
			includeNested := input.Filter.IncludeNestedRuns != nil && *input.Filter.IncludeNestedRuns
			if includeNested {
				ex = ex.Append(goqu.I("namespaces.path").Like(goqu.Any(
					dialect.From("namespaces").Select(goqu.L("path || '/%'")).Where(goqu.Ex{"group_id": *input.Filter.GroupID}))))
			} else {
				ex = ex.Append(goqu.I("workspaces.group_id").Eq(*input.Filter.GroupID))
			}
		}

		if input.Filter.RootNamespaceMemberships != nil {
			ex = ex.Append(membershipFilterByRootNamespaces(input.Filter.RootNamespaceMemberships))
		}

		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("runs.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
		}

		if input.Filter.UpdatedBefore != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("runs.updated_at").Lt(input.Filter.UpdatedBefore.UTC()))
		}

		if input.Filter.WorkspaceAssessment != nil {
			ex = ex.Append(goqu.I("runs.is_assessment_run").Eq(*input.Filter.WorkspaceAssessment))
		}

		if input.Filter.HasStateVersion != nil {
			stateVersionSub := goqu.From("state_versions").Select(goqu.L("1")).Where(goqu.I("state_versions.run_id").Eq(goqu.I("runs.id")))
			if *input.Filter.HasStateVersion {
				ex = ex.Append(goqu.L("EXISTS ?", stateVersionSub))
			} else {
				ex = ex.Append(goqu.L("NOT EXISTS ?", stateVersionSub))
			}
		}
	}

	query := selectEx.Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "runs", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("run.GetRuns"),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, r.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []*models.Run{}
	for rows.Next() {
		item, err := scanRun(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	// Hydrate run nodes for all results
	if err := r.hydrateRunNodes(ctx, r.dbClient.getConnection(ctx), results); err != nil {
		return nil, errors.Wrap(err, "failed to hydrate run nodes", errors.WithSpan(span))
	}

	result := RunsResult{
		PageInfo: rows.GetPageInfo(),
		Runs:     results,
	}

	return &result, nil
}

// CreateRun creates a new run
func (r *runs) CreateRun(ctx context.Context, run *models.Run) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.CreateRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	tx, err := r.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}
	// Rollback is safe to call even if the tx is already closed, so if the tx
	// commits successfully, this is a no-op.
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			r.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateRun: %v", txErr)
		}
	}()

	timestamp := currentTime()

	targets, err := json.Marshal(run.TargetAddresses)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal target addresses", errors.WithSpan(span))
	}

	annotations, err := json.Marshal(run.Annotations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal annotations", errors.WithSpan(span))
	}

	sql, args, err := toSQLWithTag("run.CreateRun", dialect.From("runs").
		Prepared(true).
		With("runs",
			dialect.Insert("runs").
				Rows(goqu.Record{
					"id":                         newResourceID(),
					"version":                    initialResourceVersion,
					"created_at":                 timestamp,
					"updated_at":                 timestamp,
					"status":                     run.Status,
					"is_destroy":                 run.IsDestroy,
					"workspace_id":               run.WorkspaceID,
					"configuration_version_id":   run.ConfigurationVersionID,
					"created_by":                 run.CreatedBy,
					"module_source":              run.ModuleSource,
					"module_version":             run.ModuleVersion,
					"module_digest":              run.ModuleDigest,
					"force_canceled_by":          run.ForceCanceledBy,
					"force_cancel_available_at":  run.ForceCancelAvailableAt,
					"force_canceled":             run.ForceCanceled,
					"comment":                    run.Comment,
					"auto_apply":                 run.AutoApply,
					"terraform_version":          run.TerraformVersion,
					"targets":                    targets,
					"refresh":                    run.Refresh,
					"refresh_only":               run.RefreshOnly,
					"is_assessment_run":          run.IsAssessmentRun,
					"variables_object_store_key": run.VariablesObjectStoreKey,
					"has_advisory_failures":      run.HasAdvisoryFailures,
					"annotations":                annotations,
				}).Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})))

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdRun, err := scanRun(tx.QueryRow(ctx, sql, args...))

	if err != nil {
		r.dbClient.logger.WithContextFields(ctx).Error(err)
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	// Create run nodes
	run.Metadata.ID = createdRun.Metadata.ID
	if err := r.createRunNodes(ctx, tx, run); err != nil {
		return nil, errors.Wrap(err, "failed to create run nodes", errors.WithSpan(span))
	}
	// Hydrate nodes onto the returned run
	if err := r.hydrateRunNodes(ctx, tx, []*models.Run{createdRun}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return createdRun, nil
}

// UpdateRun updates an existing run by ID
func (r *runs) UpdateRun(ctx context.Context, run *models.Run, nodeIDs ...string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	tx, err := r.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}
	// Rollback is safe to call even if the tx is already closed, so if the tx
	// commits successfully, this is a no-op.
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			r.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for UpdateRun: %v", txErr)
		}
	}()

	nodeIDSet := nodeIDFilterSet{}
	for _, id := range nodeIDs {
		nodeIDSet[id] = struct{}{}
	}

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("run.UpdateRun", dialect.From("runs").
		Prepared(true).
		With("runs",
			dialect.Update("runs").
				Set(
					goqu.Record{
						"version":                    goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":                 timestamp,
						"status":                     run.Status,
						"module_source":              run.ModuleSource,
						"module_version":             run.ModuleVersion,
						"module_digest":              run.ModuleDigest,
						"auto_apply":                 run.AutoApply,
						"force_canceled_by":          run.ForceCanceledBy,
						"force_cancel_available_at":  run.ForceCancelAvailableAt,
						"force_canceled":             run.ForceCanceled,
						"variables_object_store_key": run.VariablesObjectStoreKey,
						"has_advisory_failures":      run.HasAdvisoryFailures,
					},
				).Where(goqu.Ex{"id": run.Metadata.ID, "version": run.Metadata.Version}).
				Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})))

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedRun, err := scanRun(tx.QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		r.dbClient.logger.WithContextFields(ctx).Error(err)
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	// Update run nodes
	if err := r.updateRunNodes(ctx, tx, run, nodeIDSet); err != nil {
		return nil, errors.Wrap(err, "failed to update run nodes", errors.WithSpan(span))
	}
	// Re-hydrate nodes onto the returned run
	if err := r.hydrateRunNodes(ctx, tx, []*models.Run{updatedRun}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return updatedRun, nil
}

// DeleteRunBatch deletes the given runs, matching each on (id, version) for optimistic locking. If
// fewer rows are deleted than were requested, it returns ErrOptimisticLockError alongside whatever was
// actually deleted.
func (r *runs) DeleteRunBatch(ctx context.Context, input *DeleteRunBatchInput) ([]string, error) {
	ctx, span := tracer.Start(ctx, "db.DeleteRunBatch")
	defer span.End()

	if len(input.Runs) == 0 {
		return nil, nil
	}

	candidateEx := make([]goqu.Expression, len(input.Runs))
	for i, run := range input.Runs {
		candidateEx[i] = goqu.And(
			goqu.C("id").Eq(run.Metadata.ID),
			goqu.C("version").Eq(run.Metadata.Version),
		)
	}

	sql, args, err := dialect.Delete("runs").
		Prepared(true).
		Where(goqu.Or(candidateEx...)).
		Returning("id").
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := r.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	deletedIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		deletedIDs = append(deletedIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read rows", errors.WithSpan(span))
	}

	if len(deletedIDs) != len(input.Runs) {
		return deletedIDs, ErrOptimisticLockError
	}

	return deletedIDs, nil
}
func (r *runs) getRun(ctx context.Context, ex goqu.Ex) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.getRun")
	defer span.End()

	sql, args, err := toSQLWithTag("run.getRun", dialect.From("runs").
		Prepared(true).
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex))

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	run, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	if err := r.hydrateRunNodes(ctx, r.dbClient.getConnection(ctx), []*models.Run{run}); err != nil {
		return nil, err
	}

	return run, nil
}

// hydrateRunNodes queries run_nodes for the given runs and populates their Plan/Apply fields.
func (r *runs) hydrateRunNodes(ctx context.Context, con connection, runs []*models.Run) error {
	if len(runs) == 0 {
		return nil
	}

	idList := []string{}
	for _, run := range runs {
		idList = append(idList, run.Metadata.ID)
	}

	// Order by sort_order so each run's task stages load in canonical execution order and every
	// task_stage row is seen before every policy_check row (id breaks ties deterministically, e.g.
	// among sibling checks that share the policy-check slot).
	query, args, err := dialect.From(goqu.T("run_nodes")).
		Prepared(true).
		Select(r.getRunNodeSelectFields()...).
		Where(goqu.I("run_id").In(idList)).
		Order(goqu.I("run_nodes.sort_order").Asc(), goqu.I("run_nodes.id").Asc()).
		ToSQL()
	if err != nil {
		return err
	}

	rows, err := con.Query(ctx, query, args...)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}
	defer rows.Close()

	runMap := map[string]*models.Run{}
	for _, run := range runs {
		runMap[run.Metadata.ID] = run
	}

	for rows.Next() {
		node, sErr := scanRunNode(rows)
		if sErr != nil {
			return sErr
		}

		kind, ok := runNodeKindByType[node.Type]
		if !ok {
			return errors.New("unexpected run node type %s", node.Type)
		}

		run, ok := runMap[node.RunID]
		if !ok {
			return errors.New("failed to find run %s while hydrating run nodes", node.RunID)
		}

		if err := kind.load(run, node); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

// updateRunNodes updates run_nodes rows for a run based on the current model state. Only nodes
// whose ID is in nodeIDs are updated.
func (r *runs) updateRunNodes(ctx context.Context, con connection, run *models.Run, nodeIDs nodeIDFilterSet) error {
	for _, kind := range runNodeKinds {
		for _, inst := range kind.instances(run) {
			id := inst.id
			if id == "" || !nodeIDs.includes(id) {
				continue
			}

			record := inst.contentColumns()

			query, args, err := dialect.Update("run_nodes").
				Prepared(true).
				Set(record).
				Where(goqu.Ex{"id": id}).
				ToSQL()
			if err != nil {
				return err
			}
			if _, err := con.Exec(ctx, query, args...); err != nil {
				return err
			}
		}
	}

	return nil
}

// createRunNodes inserts run_nodes rows for a newly created run in a single multi-row insert,
// one row per present node. A multi-row insert requires every row to declare the same columns, so
// the type-specific columns a node doesn't own — all nullable — are padded with NULL to the union
// of columns across the rows being inserted.
func (r *runs) createRunNodes(ctx context.Context, con connection, run *models.Run) error {
	var records []goqu.Record

	for _, kind := range runNodeKinds {
		for _, inst := range kind.instances(run) {
			record := goqu.Record{
				"id":         inst.ensureID(),
				"run_id":     run.Metadata.ID,
				"type":       kind.typeName,
				"sort_order": inst.sortOrder,
			}
			maps.Copy(record, inst.contentColumns())
			records = append(records, record)
		}
	}

	allColumns := map[string]struct{}{}
	for _, record := range records {
		for col := range record {
			allColumns[col] = struct{}{}
		}
	}
	rows := make([]any, len(records))
	for i, record := range records {
		for col := range allColumns {
			if _, ok := record[col]; !ok {
				record[col] = nil
			}
		}
		rows[i] = record
	}

	query, args, err := dialect.Insert("run_nodes").
		Prepared(true).
		Rows(rows...).
		ToSQL()
	if err != nil {
		return err
	}
	_, err = con.Exec(ctx, query, args...)
	return err
}

func (r *runs) getRunNodeSelectFields() []any {
	fields := make([]any, len(runNodeColumns))
	for i, c := range runNodeColumns {
		fields[i] = fmt.Sprintf("run_nodes.%s", c.name)
	}
	return fields
}

func scanRunNode(row scanner) (*runNode, error) {
	n := &runNode{}

	dest := make([]any, len(runNodeColumns))
	for i, c := range runNodeColumns {
		dest[i] = c.dest(n)
	}

	if err := row.Scan(dest...); err != nil {
		return nil, err
	}

	return n, nil
}

func (r *runs) getSelectFields() []any {
	selectFields := []any{}
	for _, field := range runFieldList {
		selectFields = append(selectFields, fmt.Sprintf("runs.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanRun(row scanner) (*models.Run, error) {
	var (
		workspacePath           string
		variablesObjectStoreKey *string
		annotationsJSON         []byte
	)
	run := &models.Run{}
	run.TargetAddresses = []string{}

	err := row.Scan(
		&run.Metadata.ID,
		&run.Metadata.CreationTimestamp,
		&run.Metadata.LastUpdatedTimestamp,
		&run.Metadata.Version,
		&run.Status,
		&run.IsDestroy,
		&run.WorkspaceID,
		&run.ConfigurationVersionID,
		&run.CreatedBy,
		&run.ModuleSource,
		&run.ModuleVersion,
		&run.ModuleDigest,
		&run.ForceCanceledBy,
		&run.ForceCancelAvailableAt,
		&run.ForceCanceled,
		&run.Comment,
		&run.AutoApply,
		&run.TerraformVersion,
		&run.TargetAddresses,
		&run.Refresh,
		&run.RefreshOnly,
		&run.IsAssessmentRun,
		&variablesObjectStoreKey,
		&run.HasAdvisoryFailures,
		&annotationsJSON,
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	run.VariablesObjectStoreKey = variablesObjectStoreKey

	if len(annotationsJSON) > 0 {
		if uErr := json.Unmarshal(annotationsJSON, &run.Annotations); uErr != nil {
			return nil, errors.Wrap(uErr, "failed to unmarshal annotations")
		}
	}

	run.Metadata.TRN = trn.TypeRun.Build(workspacePath, run.GetGlobalID())

	return run, nil
}
