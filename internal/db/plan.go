package db

//go:generate go tool mockery --name Plans --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Plans encapsulates the logic to access plans from the database
type Plans interface {
	// GetPlanByID returns a plan by ID
	GetPlanByID(ctx context.Context, id string) (*models.Plan, error)
	// GetPlanByTRN returns a plan by TRN
	GetPlanByTRN(ctx context.Context, trn string) (*models.Plan, error)
	// GetPlans returns a list of plans
	GetPlans(ctx context.Context, input *GetPlansInput) (*PlansResult, error)
	// CreatePlan will create a new plan
	CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error)
	// UpdatePlan updates an existing plan
	UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error)
}

// PlanSortableField represents the fields that a plan can be sorted by
type PlanSortableField string

// PlanSortableField constants
const (
	PlanSortableFieldUpdatedAtAsc  PlanSortableField = "UPDATED_AT_ASC"
	PlanSortableFieldUpdatedAtDesc PlanSortableField = "UPDATED_AT_DESC"
)

func (sf PlanSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case PlanSortableFieldUpdatedAtAsc, PlanSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "plans", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf PlanSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// PlanFilter contains the supported fields for filtering Plan resources
type PlanFilter struct {
	PlanIDs []string
}

// GetPlansInput is the input for listing workspaces
type GetPlansInput struct {
	// Sort specifies the field to sort on and direction
	Sort *PlanSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *PlanFilter
}

// PlansResult contains the response data and page information
type PlansResult struct {
	PageInfo *pagination.PageInfo
	Plans    []models.Plan
}

type plans struct {
	dbClient *Client
}

var planFieldList = append(
	metadataFieldList,
	"workspace_id",
	"status",
	"error_message",
	"has_changes",
	"resource_additions",
	"resource_changes",
	"resource_destructions",
	"resource_imports",
	"resource_drift",
	"output_additions",
	"output_changes",
	"output_destructions",
	"diff_size",
)

// NewPlans returns an instance of the Plan interface
func NewPlans(dbClient *Client) Plans {
	return &plans{dbClient: dbClient}
}

// GetPlanByID returns a plan by name
func (p *plans) GetPlanByID(ctx context.Context, id string) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "db.GetPlanByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return p.getPlan(ctx, goqu.Ex{"plans.id": id})
}

func (p *plans) GetPlanByTRN(ctx context.Context, trn string) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "db.GetPlanByTRN")
	defer span.End()

	path, err := types.PlanModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("a plan TRN must have the workspace path and plan GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return p.getPlan(ctx, goqu.Ex{
		"plans.id":        gid.FromGlobalID(path[lastSlashIndex+1:]),
		"namespaces.path": path[:lastSlashIndex],
	})
}

// GetPlans returns a list of plans
func (p *plans) GetPlans(ctx context.Context, input *GetPlansInput) (*PlansResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetPlans")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.PlanIDs != nil {
			ex["plans.id"] = input.Filter.PlanIDs
		}
	}

	query := dialect.From("plans").
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"plans.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "plans", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, p.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Plan{}
	for rows.Next() {
		item, err := scanPlan(rows)
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

	result := PlansResult{
		PageInfo: rows.GetPageInfo(),
		Plans:    results,
	}

	return &result, nil
}

// CreatePlan creates a new plan by name
func (p *plans) CreatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "db.CreatePlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("plans").
		Prepared(true).
		With("plans",
			dialect.Insert("plans").
				Rows(goqu.Record{
					"id":                    newResourceID(),
					"version":               initialResourceVersion,
					"created_at":            timestamp,
					"updated_at":            timestamp,
					"workspace_id":          plan.WorkspaceID,
					"status":                plan.Status,
					"error_message":         plan.ErrorMessage,
					"has_changes":           plan.HasChanges,
					"resource_additions":    plan.Summary.ResourceAdditions,
					"resource_changes":      plan.Summary.ResourceChanges,
					"resource_destructions": plan.Summary.ResourceDestructions,
					"resource_imports":      plan.Summary.ResourceImports,
					"resource_drift":        plan.Summary.ResourceDrift,
					"output_additions":      plan.Summary.OutputAdditions,
					"output_changes":        plan.Summary.OutputChanges,
					"output_destructions":   plan.Summary.OutputDestructions,
					"diff_size":             plan.PlanDiffSize,
				}).Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"plans.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdPlan, err := scanPlan(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		p.dbClient.logger.Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return createdPlan, nil
}

// UpdatePlan updates an existing plan
func (p *plans) UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "db.UpdatePlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("plans").
		Prepared(true).
		With("plans",
			dialect.Update("plans").
				Set(
					goqu.Record{
						"version":               goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":            timestamp,
						"status":                plan.Status,
						"error_message":         plan.ErrorMessage,
						"has_changes":           plan.HasChanges,
						"resource_additions":    plan.Summary.ResourceAdditions,
						"resource_changes":      plan.Summary.ResourceChanges,
						"resource_destructions": plan.Summary.ResourceDestructions,
						"resource_imports":      plan.Summary.ResourceImports,
						"resource_drift":        plan.Summary.ResourceDrift,
						"output_additions":      plan.Summary.OutputAdditions,
						"output_changes":        plan.Summary.OutputChanges,
						"output_destructions":   plan.Summary.OutputDestructions,
						"diff_size":             plan.PlanDiffSize,
					},
				).Where(goqu.Ex{"id": plan.Metadata.ID, "version": plan.Metadata.Version}).
				Returning("*"),
		).Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"plans.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedPlan, err := scanPlan(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		p.dbClient.logger.Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return updatedPlan, nil
}

func (p *plans) getPlan(ctx context.Context, ex goqu.Ex) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "db.getPlan")
	defer span.End()

	sql, args, err := dialect.From("plans").
		Prepared(true).
		Select(p.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"plans.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	plan, err := scanPlan(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return plan, nil
}

func (p *plans) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range planFieldList {
		selectFields = append(selectFields, fmt.Sprintf("plans.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanPlan(row scanner) (*models.Plan, error) {
	var workspacePath string
	plan := &models.Plan{}

	err := row.Scan(
		&plan.Metadata.ID,
		&plan.Metadata.CreationTimestamp,
		&plan.Metadata.LastUpdatedTimestamp,
		&plan.Metadata.Version,
		&plan.WorkspaceID,
		&plan.Status,
		&plan.ErrorMessage,
		&plan.HasChanges,
		&plan.Summary.ResourceAdditions,
		&plan.Summary.ResourceChanges,
		&plan.Summary.ResourceDestructions,
		&plan.Summary.ResourceImports,
		&plan.Summary.ResourceDrift,
		&plan.Summary.OutputAdditions,
		&plan.Summary.OutputChanges,
		&plan.Summary.OutputDestructions,
		&plan.PlanDiffSize,
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	plan.Metadata.TRN = types.PlanModelType.BuildTRN(workspacePath, plan.GetGlobalID())

	return plan, nil
}
