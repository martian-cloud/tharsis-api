package db

//go:generate go tool mockery --name Runs --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Runs encapsulates the logic to access runs from the database
type Runs interface {
	GetRunByID(ctx context.Context, id string) (*models.Run, error)
	GetRunByTRN(ctx context.Context, trn string) (*models.Run, error)
	GetRunByPlanID(ctx context.Context, planID string) (*models.Run, error)
	GetRunByApplyID(ctx context.Context, applyID string) (*models.Run, error)
	CreateRun(ctx context.Context, run *models.Run) (*models.Run, error)
	UpdateRun(ctx context.Context, run *models.Run) (*models.Run, error)
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

func (sf RunSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case RunSortableFieldCreatedAtAsc, RunSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "runs", Col: "created_at"}
	case RunSortableFieldUpdatedAtAsc, RunSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "runs", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf RunSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// RunFilter contains the supported fields for filtering Run resources
type RunFilter struct {
	TimeRangeStart      *time.Time
	PlanID              *string
	ApplyID             *string
	WorkspaceID         *string
	GroupID             *string
	UserMemberID        *string
	RunIDs              []string
	WorkspaceAssessment *bool
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
	Runs     []models.Run
}

type runs struct {
	dbClient *Client
}

var runFieldList = append(
	metadataFieldList,
	"status",
	"is_destroy",
	"has_changes",
	"workspace_id",
	"configuration_version_id",
	"created_by",
	"plan_id",
	"apply_id",
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
)

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

func (r *runs) GetRunByTRN(ctx context.Context, trn string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunByTRN")
	defer span.End()

	path, err := types.RunModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("a run TRN must have the workspace path and run GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return r.getRun(ctx, goqu.Ex{
		"runs.id":         gid.FromGlobalID(path[lastSlashIndex+1:]),
		"namespaces.path": path[:lastSlashIndex],
	})
}

func (r *runs) GetRunByPlanID(ctx context.Context, planID string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunByPlanID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sort := RunSortableFieldUpdatedAtDesc
	result, err := r.GetRuns(ctx, &GetRunsInput{
		Sort: &sort,
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &RunFilter{
			PlanID: &planID,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get run for plan")
		return nil, errors.Wrap(err, "failed to get run for plan")
	}

	if len(result.Runs) == 0 {
		tracing.RecordError(span, nil, "Failed to get run for plan")
		return nil, errors.New(
			"Failed to get run for plan",
		)
	}

	return &result.Runs[0], nil
}

func (r *runs) GetRunByApplyID(ctx context.Context, applyID string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.GetRunByApplyID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sort := RunSortableFieldUpdatedAtDesc
	result, err := r.GetRuns(ctx, &GetRunsInput{
		Sort: &sort,
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &RunFilter{
			ApplyID: &applyID,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get run for apply")
		return nil, errors.Wrap(err, "failed to get run for apply")
	}

	if len(result.Runs) == 0 {
		tracing.RecordError(span, nil, "Failed to get run for apply")
		return nil, errors.New(
			"Failed to get run for apply",
		)
	}

	return &result.Runs[0], nil
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

		if input.Filter.PlanID != nil {
			ex = ex.Append(goqu.I("runs.plan_id").Eq(*input.Filter.PlanID))
		}

		if input.Filter.ApplyID != nil {
			ex = ex.Append(goqu.I("runs.apply_id").Eq(*input.Filter.ApplyID))
		}

		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("runs.workspace_id").Eq(*input.Filter.WorkspaceID))
		}

		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("workspaces.group_id").Eq(*input.Filter.GroupID))
		}

		if input.Filter.UserMemberID != nil {
			ex = ex.Append(namespaceMembershipFilterQuery("namespace_memberships.user_id", *input.Filter.UserMemberID))
		}

		if input.Filter.TimeRangeStart != nil {
			// Must use UTC here otherwise, queries will return unexpected results.
			ex = ex.Append(goqu.I("runs.created_at").Gte(input.Filter.TimeRangeStart.UTC()))
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
	results := []models.Run{}
	for rows.Next() {
		item, err := scanRun(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
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

	timestamp := currentTime()

	targets, err := json.Marshal(run.TargetAddresses)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal target addresses")
		return nil, err
	}

	sql, args, err := dialect.From("runs").
		Prepared(true).
		With("runs",
			dialect.Insert("runs").
				Rows(goqu.Record{
					"id":                        newResourceID(),
					"version":                   initialResourceVersion,
					"created_at":                timestamp,
					"updated_at":                timestamp,
					"status":                    run.Status,
					"is_destroy":                run.IsDestroy,
					"has_changes":               run.HasChanges,
					"workspace_id":              run.WorkspaceID,
					"configuration_version_id":  run.ConfigurationVersionID,
					"created_by":                run.CreatedBy,
					"plan_id":                   nullableString(run.PlanID),
					"apply_id":                  nullableString(run.ApplyID),
					"module_source":             run.ModuleSource,
					"module_version":            run.ModuleVersion,
					"module_digest":             run.ModuleDigest,
					"force_canceled_by":         run.ForceCanceledBy,
					"force_cancel_available_at": run.ForceCancelAvailableAt,
					"force_canceled":            run.ForceCanceled,
					"comment":                   run.Comment,
					"auto_apply":                false,
					"terraform_version":         run.TerraformVersion,
					"targets":                   targets,
					"refresh":                   run.Refresh,
					"refresh_only":              run.RefreshOnly,
					"is_assessment_run":         run.IsAssessmentRun,
				}).Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdRun, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		r.dbClient.logger.WithContextFields(ctx).Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return createdRun, nil
}

// UpdateRun updates an existing run by ID
func (r *runs) UpdateRun(ctx context.Context, run *models.Run) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("runs").
		Prepared(true).
		With("runs",
			dialect.Update("runs").
				Set(
					goqu.Record{
						"version":                   goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":                timestamp,
						"status":                    run.Status,
						"has_changes":               run.HasChanges,
						"plan_id":                   nullableString(run.PlanID),
						"apply_id":                  nullableString(run.ApplyID),
						"module_source":             run.ModuleSource,
						"module_version":            run.ModuleVersion,
						"module_digest":             run.ModuleDigest,
						"force_canceled_by":         run.ForceCanceledBy,
						"force_cancel_available_at": run.ForceCancelAvailableAt,
						"force_canceled":            run.ForceCanceled,
					},
				).Where(goqu.Ex{"id": run.Metadata.ID, "version": run.Metadata.Version}).
				Returning("*"),
		).Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedRun, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		r.dbClient.logger.WithContextFields(ctx).Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return updatedRun, nil
}

func (r *runs) getRun(ctx context.Context, ex goqu.Ex) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "db.getRun")
	defer span.End()

	sql, args, err := dialect.From("runs").
		Prepared(true).
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex).
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

	return run, nil
}

func (r *runs) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range runFieldList {
		selectFields = append(selectFields, fmt.Sprintf("runs.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanRun(row scanner) (*models.Run, error) {
	var planID sql.NullString
	var applyID sql.NullString
	var workspacePath string
	run := &models.Run{}
	run.TargetAddresses = []string{}

	err := row.Scan(
		&run.Metadata.ID,
		&run.Metadata.CreationTimestamp,
		&run.Metadata.LastUpdatedTimestamp,
		&run.Metadata.Version,
		&run.Status,
		&run.IsDestroy,
		&run.HasChanges,
		&run.WorkspaceID,
		&run.ConfigurationVersionID,
		&run.CreatedBy,
		&planID,
		&applyID,
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
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	if planID.Valid {
		run.PlanID = planID.String
	}

	if applyID.Valid {
		run.ApplyID = applyID.String
	}

	run.Metadata.TRN = types.RunModelType.BuildTRN(workspacePath, run.GetGlobalID())

	return run, nil
}
