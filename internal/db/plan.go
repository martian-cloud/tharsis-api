package db

//go:generate mockery --name Plans --inpackage --case underscore

import (
	"context"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Plans encapsulates the logic to access plans from the database
type Plans interface {
	// GetPlan returns a plan by ID
	GetPlan(ctx context.Context, id string) (*models.Plan, error)
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

var planFieldList = append(metadataFieldList, "workspace_id", "status", "has_changes", "resource_additions", "resource_changes", "resource_destructions")

// NewPlans returns an instance of the Plan interface
func NewPlans(dbClient *Client) Plans {
	return &plans{dbClient: dbClient}
}

// GetPlan returns a plan by name
func (p *plans) GetPlan(ctx context.Context, id string) (*models.Plan, error) {

	sql, args, err := dialect.From("plans").
		Prepared(true).
		Select(planFieldList...).
		Where(goqu.Ex{"id": id}).
		ToSQL()

	if err != nil {
		return nil, err
	}

	plan, err := scanPlan(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return plan, nil
}

// GetPlans returns a list of plans
func (p *plans) GetPlans(ctx context.Context, input *GetPlansInput) (*PlansResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.PlanIDs != nil {
			ex["plans.id"] = input.Filter.PlanIDs
		}
	}

	query := dialect.From("plans").
		Select(planFieldList...).
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
		sortBy,
		sortDirection,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, p.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Plan{}
	for rows.Next() {
		item, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
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
	timestamp := currentTime()

	sql, args, err := dialect.Insert("plans").
		Prepared(true).
		Rows(goqu.Record{
			"id":                    newResourceID(),
			"version":               initialResourceVersion,
			"created_at":            timestamp,
			"updated_at":            timestamp,
			"workspace_id":          plan.WorkspaceID,
			"status":                plan.Status,
			"has_changes":           plan.HasChanges,
			"resource_additions":    plan.ResourceAdditions,
			"resource_changes":      plan.ResourceChanges,
			"resource_destructions": plan.ResourceDestructions,
		}).
		Returning(planFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdPlan, err := scanPlan(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		p.dbClient.logger.Error(err)
		return nil, err
	}
	return createdPlan, nil
}

// UpdatePlan updates an existing plan
func (p *plans) UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Update("plans").
		Prepared(true).
		Set(
			goqu.Record{
				"version":               goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":            timestamp,
				"status":                plan.Status,
				"has_changes":           plan.HasChanges,
				"resource_additions":    plan.ResourceAdditions,
				"resource_changes":      plan.ResourceChanges,
				"resource_destructions": plan.ResourceDestructions,
			},
		).Where(goqu.Ex{"id": plan.Metadata.ID, "version": plan.Metadata.Version}).Returning(planFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedPlan, err := scanPlan(p.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		p.dbClient.logger.Error(err)
		return nil, err
	}
	return updatedPlan, nil
}

func scanPlan(row scanner) (*models.Plan, error) {
	plan := &models.Plan{}

	err := row.Scan(
		&plan.Metadata.ID,
		&plan.Metadata.CreationTimestamp,
		&plan.Metadata.LastUpdatedTimestamp,
		&plan.Metadata.Version,
		&plan.WorkspaceID,
		&plan.Status,
		&plan.HasChanges,
		&plan.ResourceAdditions,
		&plan.ResourceChanges,
		&plan.ResourceDestructions,
	)
	if err != nil {
		return nil, err
	}

	return plan, nil
}
