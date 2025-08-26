package db

//go:generate go tool mockery --name Applies --inpackage --case underscore

import (
	"context"
	"database/sql"
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
	"go.opentelemetry.io/otel/attribute"
)

// Applies encapsulates the logic to access applies from the database
type Applies interface {
	// GetApplyByID returns a apply by ID
	GetApplyByID(ctx context.Context, id string) (*models.Apply, error)
	// GetApplyByTRN returns a apply by TRN
	GetApplyByTRN(ctx context.Context, trn string) (*models.Apply, error)
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

func (sf ApplySortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case ApplySortableFieldUpdatedAtAsc, ApplySortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "applies", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf ApplySortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
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
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *ApplyFilter
}

// AppliesResult contains the response data and page information
type AppliesResult struct {
	PageInfo *pagination.PageInfo
	Applies  []models.Apply
}

type applies struct {
	dbClient *Client
}

var applyFieldList = append(metadataFieldList, "workspace_id", "status", "error_message", "comment", "triggered_by")

// NewApplies returns an instance of the Apply interface
func NewApplies(dbClient *Client) Applies {
	return &applies{dbClient: dbClient}
}

func (a *applies) GetApplyByID(ctx context.Context, id string) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "db.GetApplyByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return a.getApply(ctx, goqu.Ex{"applies.id": id})
}

func (a *applies) GetApplyByTRN(ctx context.Context, trn string) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "db.GetApplyByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.ApplyModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	lastSlashIndex := strings.LastIndex(path, "/")
	if lastSlashIndex == -1 {
		return nil, errors.New("a apply TRN must have the workspace path and apply GID separated by a forward slash",
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	return a.getApply(ctx, goqu.Ex{
		"applies.id":      gid.FromGlobalID(path[lastSlashIndex+1:]),
		"namespaces.path": path[:lastSlashIndex],
	})
}

func (a *applies) GetApplies(ctx context.Context, input *GetAppliesInput) (*AppliesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetApplies")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.ApplyIDs != nil {
			ex["applies.id"] = input.Filter.ApplyIDs
		}
	}

	query := dialect.From("applies").
		Select(a.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"applies.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "applies", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, a.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Apply{}
	for rows.Next() {
		item, err := scanApply(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan a row")
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		tracing.RecordError(span, err, "failed to finalize rows")
		return nil, err
	}

	result := AppliesResult{
		PageInfo: rows.GetPageInfo(),
		Applies:  results,
	}

	return &result, nil
}

func (a *applies) CreateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "db.CreateApply")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("applies").
		Prepared(true).
		With("applies",
			dialect.Insert("applies").
				Rows(goqu.Record{
					"id":            newResourceID(),
					"version":       initialResourceVersion,
					"created_at":    timestamp,
					"updated_at":    timestamp,
					"workspace_id":  apply.WorkspaceID,
					"status":        apply.Status,
					"error_message": apply.ErrorMessage,
					"comment":       apply.Comment,
					"triggered_by":  nullableString(apply.TriggeredBy),
				}).Returning("*"),
		).Select(a.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"applies.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdApply, err := scanApply(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		a.dbClient.logger.WithContextFields(ctx).Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return createdApply, nil
}

func (a *applies) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateApply")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("applies").
		Prepared(true).
		With("applies",
			dialect.Update("applies").
				Set(
					goqu.Record{
						"version":       goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":    timestamp,
						"status":        apply.Status,
						"error_message": apply.ErrorMessage,
						"comment":       apply.Comment,
						"triggered_by":  nullableString(apply.TriggeredBy),
					},
				).Where(goqu.Ex{"id": apply.Metadata.ID, "version": apply.Metadata.Version}).
				Returning("*"),
		).Select(a.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"applies.workspace_id": goqu.I("namespaces.workspace_id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedApply, err := scanApply(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		a.dbClient.logger.WithContextFields(ctx).Error(err)
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}
	return updatedApply, nil
}

func (a *applies) getApply(ctx context.Context, ex goqu.Ex) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "db.getApply")
	defer span.End()

	sql, args, err := dialect.From("applies").
		Prepared(true).
		Select(a.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"applies.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex).
		ToSQL()

	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	apply, err := scanApply(a.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return apply, nil
}

func (a *applies) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range applyFieldList {
		selectFields = append(selectFields, fmt.Sprintf("applies.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanApply(row scanner) (*models.Apply, error) {
	var triggeredBy sql.NullString
	var workspacePath string

	apply := &models.Apply{}

	err := row.Scan(
		&apply.Metadata.ID,
		&apply.Metadata.CreationTimestamp,
		&apply.Metadata.LastUpdatedTimestamp,
		&apply.Metadata.Version,
		&apply.WorkspaceID,
		&apply.Status,
		&apply.ErrorMessage,
		&apply.Comment,
		&triggeredBy,
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	if triggeredBy.Valid {
		apply.TriggeredBy = triggeredBy.String
	}

	apply.Metadata.TRN = types.ApplyModelType.BuildTRN(workspacePath, apply.GetGlobalID())

	return apply, nil
}
