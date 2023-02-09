package db

//go:generate mockery --name Runs --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Runs encapsulates the logic to access runs from the database
type Runs interface {
	GetRun(ctx context.Context, id string) (*models.Run, error)
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

func (sf RunSortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case RunSortableFieldCreatedAtAsc, RunSortableFieldCreatedAtDesc:
		return &fieldDescriptor{key: "created_at", table: "runs", col: "created_at"}
	case RunSortableFieldUpdatedAtAsc, RunSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "runs", col: "updated_at"}
	default:
		return nil
	}
}

func (sf RunSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
}

// RunFilter contains the supported fields for filtering Run resources
type RunFilter struct {
	PlanID      *string
	ApplyID     *string
	WorkspaceID *string
	GroupID     *string
	RunIDs      []string
}

// GetRunsInput is the input for listing runs
type GetRunsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *RunSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *RunFilter
}

// RunsResult contains the response data and page information
type RunsResult struct {
	PageInfo *PageInfo
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
)

// NewRuns returns an instance of the Run interface
func NewRuns(dbClient *Client) Runs {
	return &runs{dbClient: dbClient}
}

// GetRun returns a run by ID
func (r *runs) GetRun(ctx context.Context, id string) (*models.Run, error) {
	sql, args, err := dialect.From("runs").
		Prepared(true).
		Select(runFieldList...).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	run, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return run, nil
}

func (r *runs) GetRunByPlanID(ctx context.Context, planID string) (*models.Run, error) {
	sort := RunSortableFieldUpdatedAtDesc
	result, err := r.GetRuns(ctx, &GetRunsInput{
		Sort: &sort,
		PaginationOptions: &PaginationOptions{
			First: ptr.Int32(1),
		},
		Filter: &RunFilter{
			PlanID: &planID,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run for plan",
			errors.WithErrorErr(err),
		)
	}

	if len(result.Runs) == 0 {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run for plan",
		)
	}

	return &result.Runs[0], nil
}

func (r *runs) GetRunByApplyID(ctx context.Context, applyID string) (*models.Run, error) {
	sort := RunSortableFieldUpdatedAtDesc
	result, err := r.GetRuns(ctx, &GetRunsInput{
		Sort: &sort,
		PaginationOptions: &PaginationOptions{
			First: ptr.Int32(1),
		},
		Filter: &RunFilter{
			ApplyID: &applyID,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run for apply",
			errors.WithErrorErr(err),
		)
	}

	if len(result.Runs) == 0 {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run for apply",
		)
	}

	return &result.Runs[0], nil
}

func (r *runs) GetRuns(ctx context.Context, input *GetRunsInput) (*RunsResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.RunIDs != nil {
			ex["runs.id"] = input.Filter.RunIDs
		}

		if input.Filter.PlanID != nil {
			ex["runs.plan_id"] = *input.Filter.PlanID

		}

		if input.Filter.ApplyID != nil {
			ex["runs.apply_id"] = *input.Filter.ApplyID
		}

		if input.Filter.WorkspaceID != nil {
			ex["runs.workspace_id"] = *input.Filter.WorkspaceID
		}

		if input.Filter.GroupID != nil {
			ex["workspaces.group_id"] = *input.Filter.GroupID
		}
	}

	query := dialect.From("runs").
		Select(r.getSelectFields()...).
		InnerJoin(goqu.T("workspaces"), goqu.On(goqu.Ex{"runs.workspace_id": goqu.I("workspaces.id")})).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "runs", col: "id"},
		sortBy,
		sortDirection,
		runFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, r.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Run{}
	for rows.Next() {
		item, err := scanRun(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := RunsResult{
		PageInfo: rows.getPageInfo(),
		Runs:     results,
	}

	return &result, nil
}

// CreateRun creates a new run
func (r *runs) CreateRun(ctx context.Context, run *models.Run) (*models.Run, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Insert("runs").
		Prepared(true).
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
		}).
		Returning(runFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdRun, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		r.dbClient.logger.Error(err)
		return nil, err
	}
	return createdRun, nil
}

// UpdateRun updates an existing run by ID
func (r *runs) UpdateRun(ctx context.Context, run *models.Run) (*models.Run, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Update("runs").
		Prepared(true).
		Set(
			goqu.Record{
				"version":                   goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":                timestamp,
				"status":                    run.Status,
				"is_destroy":                run.IsDestroy,
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
		).Where(goqu.Ex{"id": run.Metadata.ID, "version": run.Metadata.Version}).Returning(r.getSelectFields()...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedRun, err := scanRun(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		r.dbClient.logger.Error(err)
		return nil, err
	}
	return updatedRun, nil
}

func (r *runs) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range runFieldList {
		selectFields = append(selectFields, fmt.Sprintf("runs.%s", field))
	}
	return selectFields
}

func scanRun(row scanner) (*models.Run, error) {
	var configurationVersionID sql.NullString
	var forceCancelAvailableAt sql.NullTime
	var forceCanceledBy sql.NullString
	var planID sql.NullString
	var applyID sql.NullString

	run := &models.Run{}

	err := row.Scan(
		&run.Metadata.ID,
		&run.Metadata.CreationTimestamp,
		&run.Metadata.LastUpdatedTimestamp,
		&run.Metadata.Version,
		&run.Status,
		&run.IsDestroy,
		&run.HasChanges,
		&run.WorkspaceID,
		&configurationVersionID,
		&run.CreatedBy,
		&planID,
		&applyID,
		&run.ModuleSource,
		&run.ModuleVersion,
		&run.ModuleDigest,
		&forceCanceledBy,
		&forceCancelAvailableAt,
		&run.ForceCanceled,
		&run.Comment,
		&run.AutoApply,
		&run.TerraformVersion,
	)
	if err != nil {
		return nil, err
	}

	if configurationVersionID.Valid {
		run.ConfigurationVersionID = &configurationVersionID.String
	}

	if planID.Valid {
		run.PlanID = planID.String
	}

	if applyID.Valid {
		run.ApplyID = applyID.String
	}

	if forceCanceledBy.Valid {
		run.ForceCanceledBy = &forceCanceledBy.String
	}

	if forceCancelAvailableAt.Valid {
		run.ForceCancelAvailableAt = &forceCancelAvailableAt.Time
	}

	return run, nil
}

func runFieldResolver(key string, model interface{}) (string, error) {
	run, ok := model.(*models.Run)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected run type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &run.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
