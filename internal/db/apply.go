package db

//go:generate mockery --name Applies --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Applies encapsulates the logic to access applies from the database
type Applies interface {
	// GetApply returns a apply by ID
	GetApply(ctx context.Context, id string) (*models.Apply, error)
	// GetApplies returns a list of applies
	GetApplies(ctx context.Context, input *GetAppliesInput) (*AppliesResult, error)
	// CreateApply will create a new apply
	CreateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error)
	// UpdateApply updates an existing apply
	UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error)
}

// ApplySortableField represents the fields that an apply can be sorted by
type ApplySortableField string

// ApplySortableField constants
const (
	ApplySortableFieldUpdatedAtAsc  ApplySortableField = "UPDATED_AT_ASC"
	ApplySortableFieldUpdatedAtDesc ApplySortableField = "UPDATED_AT_DESC"
)

func (sf ApplySortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case ApplySortableFieldUpdatedAtAsc, ApplySortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "applies", col: "updated_at"}
	default:
		return nil
	}
}

func (sf ApplySortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
}

// ApplyFilter contains the supported fields for filtering Apply resources
type ApplyFilter struct {
	ApplyIDs []string
}

// GetAppliesInput is the input for listing workspaces
type GetAppliesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *ApplySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *ApplyFilter
}

// AppliesResult contains the response data and page information
type AppliesResult struct {
	PageInfo *PageInfo
	Applies  []models.Apply
}

type applies struct {
	dbClient *Client
}

var applyFieldList = append(metadataFieldList, "workspace_id", "status", "comment", "triggered_by")

// NewApplies returns an instance of the Apply interface
func NewApplies(dbClient *Client) Applies {
	return &applies{dbClient: dbClient}
}

func (a *applies) GetApply(ctx context.Context, id string) (*models.Apply, error) {
	sql, args, err := dialect.From("applies").
		Prepared(true).
		Select(applyFieldList...).
		Where(goqu.Ex{"id": id}).ToSQL()

	if err != nil {
		return nil, err
	}

	apply, err := scanApply(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return apply, nil
}

func (a *applies) GetApplies(ctx context.Context, input *GetAppliesInput) (*AppliesResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.ApplyIDs != nil {
			ex["applies.id"] = input.Filter.ApplyIDs
		}
	}

	query := dialect.From("applies").
		Select(applyFieldList...).
		Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "applies", col: "id"},
		sortBy,
		sortDirection,
		applyFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Apply{}
	for rows.Next() {
		item, err := scanApply(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := AppliesResult{
		PageInfo: rows.getPageInfo(),
		Applies:  results,
	}

	return &result, nil
}

func (a *applies) CreateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Insert("applies").
		Prepared(true).
		Rows(goqu.Record{
			"id":           newResourceID(),
			"version":      initialResourceVersion,
			"created_at":   timestamp,
			"updated_at":   timestamp,
			"workspace_id": apply.WorkspaceID,
			"status":       apply.Status,
			"comment":      apply.Comment,
			"triggered_by": nullableString(apply.TriggeredBy),
		}).
		Returning(applyFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	createdApply, err := scanApply(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		a.dbClient.logger.Error(err)
		return nil, err
	}
	return createdApply, nil
}

func (a *applies) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	timestamp := currentTime()

	sql, args, err := dialect.Update("applies").
		Prepared(true).
		Set(
			goqu.Record{
				"version":      goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":   timestamp,
				"status":       apply.Status,
				"comment":      apply.Comment,
				"triggered_by": nullableString(apply.TriggeredBy),
			},
		).Where(goqu.Ex{"id": apply.Metadata.ID, "version": apply.Metadata.Version}).Returning(applyFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedApply, err := scanApply(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		a.dbClient.logger.Error(err)
		return nil, err
	}
	return updatedApply, nil
}

func scanApply(row scanner) (*models.Apply, error) {
	var triggeredBy sql.NullString

	apply := &models.Apply{}

	err := row.Scan(
		&apply.Metadata.ID,
		&apply.Metadata.CreationTimestamp,
		&apply.Metadata.LastUpdatedTimestamp,
		&apply.Metadata.Version,
		&apply.WorkspaceID,
		&apply.Status,
		&apply.Comment,
		&triggeredBy,
	)
	if err != nil {
		return nil, err
	}

	if triggeredBy.Valid {
		apply.TriggeredBy = triggeredBy.String
	}

	return apply, nil
}

func applyFieldResolver(key string, model interface{}) (string, error) {
	apply, ok := model.(*models.Apply)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected apply type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &apply.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
