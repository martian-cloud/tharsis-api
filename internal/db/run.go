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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// Runs encapsulates the logic to access runs from the database
type Runs interface {
	GetRunByID(ctx context.Context, id string) (*models.Run, error)
	GetRunByTRN(ctx context.Context, trnValue string) (*models.Run, error)
	GetRunByNodeID(ctx context.Context, nodeID string) (*models.Run, error)
	CreateRun(ctx context.Context, run *models.Run) (*models.Run, error)
	UpdateRun(ctx context.Context, run *models.Run, nodeIDs ...string) (*models.Run, error)
	GetRuns(ctx context.Context, input *GetRunsInput) (*RunsResult, error)
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
	SortOrder            int
}

// runNodeKind centralizes everything the db layer needs to persist and load one run-node
// type. Adding a node type means adding one entry here; the create/update/read/hydrate paths
// all dispatch through this table instead of per-type branches. The closures operate on the
// concrete run.Plan / run.Apply fields directly (no models.RunNode interface, no type
// assertions), so a misconfigured kind is a compile error rather than a runtime panic.
type runNodeKind struct {
	typeName  string
	sortOrder int
	// present reports whether the run has a node of this kind.
	present func(run *models.Run) bool
	// id returns this kind's node id on the run.
	id func(run *models.Run) string
	// ensureID assigns a new id if the node has none and returns the id (create path).
	ensureID func(run *models.Run) string
	// contentColumns returns all writable content columns for this run's node of this kind (status,
	// latest_job_id, error_message and the type-specific columns). Structural columns (id, run_id,
	// type, sort_order, created_at, updated_at) are added by the caller.
	contentColumns func(run *models.Run) goqu.Record
	// load builds the model node from a scanned wide row and assigns it onto the run.
	load func(run *models.Run, n *runNode)
}

var runNodeKinds = []runNodeKind{
	{
		typeName:  "plan",
		sortOrder: 0,
		present:   func(_ *models.Run) bool { return true },
		id:        func(run *models.Run) string { return run.Plan.ID },
		ensureID: func(run *models.Run) string {
			if run.Plan.ID == "" {
				run.Plan.ID = newResourceID()
			}
			return run.Plan.ID
		},
		contentColumns: func(run *models.Run) goqu.Record {
			p := &run.Plan
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
		load: func(run *models.Run, n *runNode) {
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
		},
	},
	{
		typeName:  "apply",
		sortOrder: 1,
		present:   func(run *models.Run) bool { return run.Apply != nil },
		id:        func(run *models.Run) string { return run.Apply.ID },
		ensureID: func(run *models.Run) string {
			if run.Apply.ID == "" {
				run.Apply.ID = newResourceID()
			}
			return run.Apply.ID
		},
		contentColumns: func(run *models.Run) goqu.Record {
			a := run.Apply
			return goqu.Record{
				"status":             string(a.Status),
				"latest_job_id":      a.LatestJobID,
				"error_message":      a.ErrorMessage,
				"apply_triggered_by": nullableString(a.TriggeredBy),
				"apply_comment":      nullableString(a.Comment),
			}
		},
		load: func(run *models.Run, n *runNode) {
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
		},
	},
}

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
	{"sort_order", func(n *runNode) any { return &n.SortOrder }},
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
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, r.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []*models.Run{}
	for rows.Next() {
		item, err := scanRun(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	// Hydrate run nodes for all results
	if err := r.hydrateRunNodes(ctx, r.dbClient.getConnection(ctx), results); err != nil {
		tracing.RecordError(span, err, "failed to hydrate run nodes")
		return nil, err
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
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
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
		tracing.RecordError(span, err, "failed to marshal target addresses")
		return nil, err
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
				}).Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})))

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdRun, err := scanRun(tx.QueryRow(ctx, sql, args...))

	if err != nil {
		r.dbClient.logger.WithContextFields(ctx).Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Create run nodes
	run.Metadata.ID = createdRun.Metadata.ID
	if err := r.createRunNodes(ctx, tx, run); err != nil {
		tracing.RecordError(span, err, "failed to create run nodes")
		return nil, err
	}
	// Hydrate nodes onto the returned run
	if err := r.hydrateRunNodes(ctx, tx, []*models.Run{createdRun}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
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
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
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
					},
				).Where(goqu.Ex{"id": run.Metadata.ID, "version": run.Metadata.Version}).
				Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})))

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedRun, err := scanRun(tx.QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		r.dbClient.logger.WithContextFields(ctx).Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Update run nodes
	if err := r.updateRunNodes(ctx, tx, run, nodeIDSet); err != nil {
		tracing.RecordError(span, err, "failed to update run nodes")
		return nil, err
	}
	// Re-hydrate nodes onto the returned run
	if err := r.hydrateRunNodes(ctx, tx, []*models.Run{updatedRun}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedRun, nil
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

	query, args, err := dialect.From(goqu.T("run_nodes")).
		Prepared(true).
		Select(r.getRunNodeSelectFields()...).
		Where(goqu.I("run_id").In(idList)).
		Order(goqu.I("sort_order").Asc()).
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

		kind.load(run, node)
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
		if !kind.present(run) {
			continue
		}
		id := kind.id(run)
		if !nodeIDs.includes(id) {
			continue
		}

		record := kind.contentColumns(run)

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

	return nil
}

// createRunNodes inserts run_nodes rows for a newly created run in a single multi-row insert,
// one row per present node. A multi-row insert requires every row to declare the same columns, so
// the type-specific columns a node doesn't own — all nullable — are padded with NULL to the union
// of columns across the rows being inserted.
func (r *runs) createRunNodes(ctx context.Context, con connection, run *models.Run) error {
	var records []goqu.Record

	for _, kind := range runNodeKinds {
		if !kind.present(run) {
			continue
		}

		record := goqu.Record{
			"id":         kind.ensureID(run),
			"run_id":     run.Metadata.ID,
			"type":       kind.typeName,
			"sort_order": kind.sortOrder,
		}
		maps.Copy(record, kind.contentColumns(run))
		records = append(records, record)
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
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	run.VariablesObjectStoreKey = variablesObjectStoreKey

	run.Metadata.TRN = trn.TypeRun.Build(workspacePath, run.GetGlobalID())

	return run, nil
}
