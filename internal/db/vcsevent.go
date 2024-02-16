package db

//go:generate mockery --name VCSEvents --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// VCSEvents encapsulates the logic for accessing vcs events form the database.
type VCSEvents interface {
	GetEventByID(ctx context.Context, id string) (*models.VCSEvent, error)
	GetEvents(ctx context.Context, input *GetVCSEventsInput) (*VCSEventsResult, error)
	CreateEvent(ctx context.Context, event *models.VCSEvent) (*models.VCSEvent, error)
	UpdateEvent(ctx context.Context, event *models.VCSEvent) (*models.VCSEvent, error)
}

// VCSEventSortableField represents the fields that a vcs event can be sorted by.
type VCSEventSortableField string

// VCSEventSortableField constants.
const (
	VCSEventSortableFieldCreatedAtAsc  VCSEventSortableField = "CREATED_AT_ASC"
	VCSEventSortableFieldCreatedAtDesc VCSEventSortableField = "CREATED_AT_DESC"
	VCSEventSortableFieldUpdatedAtAsc  VCSEventSortableField = "UPDATED_AT_ASC"
	VCSEventSortableFieldUpdatedAtDesc VCSEventSortableField = "UPDATED_AT_DESC"
)

func (sf VCSEventSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case VCSEventSortableFieldCreatedAtAsc, VCSEventSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "vcs_events", Col: "created_at"}
	case VCSEventSortableFieldUpdatedAtAsc, VCSEventSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "vcs_events", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf VCSEventSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// VCSEventFilter contains the supported fields for filtering vcs event resources
type VCSEventFilter struct {
	WorkspaceID *string
	VCSEventIDs []string
}

// GetVCSEventsInput is the input for listing vcs events
type GetVCSEventsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *VCSEventSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter contains the supported fields for filtering VCSEvent resources
	Filter *VCSEventFilter
}

// VCSEventsResult contains the response data and page information
type VCSEventsResult struct {
	PageInfo  *pagination.PageInfo
	VCSEvents []models.VCSEvent
}

type vcsEvents struct {
	dbClient *Client
}

// NewVCSEvents returns an instance of the VCSEvents interface.
func NewVCSEvents(dbClient *Client) VCSEvents {
	return &vcsEvents{dbClient: dbClient}
}

var vcsEventsFieldList = append(
	metadataFieldList,
	"commit_id",
	"source_reference_name",
	"workspace_id",
	"type",
	"status",
	"repository_url",
	"error_message",
)

func (ve *vcsEvents) GetEventByID(ctx context.Context, id string) (*models.VCSEvent, error) {
	ctx, span := tracer.Start(ctx, "db.GetEventByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return ve.getEvent(ctx, goqu.Ex{"vcs_events.id": id})
}

func (ve *vcsEvents) GetEvents(ctx context.Context, input *GetVCSEventsInput) (*VCSEventsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetEvents")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()
	if input.Filter != nil {
		if input.Filter.VCSEventIDs != nil {
			ex = ex.Append(goqu.I("vcs_events.id").In(input.Filter.VCSEventIDs))
		}

		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("vcs_events.workspace_id").Eq(input.Filter.WorkspaceID))
		}
	}

	query := dialect.From("vcs_events").
		Select(ve.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "vcs_events", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, ve.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.VCSEvent{}
	for rows.Next() {
		item, err := scanVCSEvent(rows)
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

	result := VCSEventsResult{
		PageInfo:  rows.GetPageInfo(),
		VCSEvents: results,
	}

	return &result, nil
}

func (ve *vcsEvents) CreateEvent(ctx context.Context, event *models.VCSEvent) (*models.VCSEvent, error) {
	ctx, span := tracer.Start(ctx, "db.CreateEvent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("vcs_events").
		Prepared(true).
		Rows(goqu.Record{
			"id":                    newResourceID(),
			"version":               initialResourceVersion,
			"created_at":            timestamp,
			"updated_at":            timestamp,
			"commit_id":             event.CommitID,
			"source_reference_name": event.SourceReferenceName,
			"workspace_id":          event.WorkspaceID,
			"type":                  event.Type,
			"status":                event.Status,
			"repository_url":        event.RepositoryURL,
			"error_message":         event.ErrorMessage,
		}).
		Returning(vcsEventsFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdEvent, err := scanVCSEvent(ve.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_workspace_id":
					tracing.RecordError(span, nil, "workspace does not exist")
					return nil, errors.New("workspace does not exist", errors.WithErrorCode(errors.ENotFound))
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdEvent, nil
}

func (ve *vcsEvents) UpdateEvent(ctx context.Context, event *models.VCSEvent) (*models.VCSEvent, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateEvent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("vcs_events").
		Prepared(true).
		Set(
			goqu.Record{
				"version":       goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":    timestamp,
				"status":        event.Status,
				"error_message": event.ErrorMessage,
			},
		).Where(goqu.Ex{"id": event.Metadata.ID, "version": event.Metadata.Version}).
		Returning(vcsEventsFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedEvent, err := scanVCSEvent(ve.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedEvent, nil
}

func (ve *vcsEvents) getEvent(ctx context.Context, exp goqu.Ex) (*models.VCSEvent, error) {
	sql, args, err := dialect.From(goqu.T("vcs_events")).
		Prepared(true).
		Select(ve.getSelectFields()...).
		Where(exp).
		ToSQL()

	if err != nil {
		return nil, err
	}

	event, err := scanVCSEvent(ve.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return event, nil
}

func (ve *vcsEvents) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range vcsEventsFieldList {
		selectFields = append(selectFields, fmt.Sprintf("vcs_events.%s", field))
	}

	return selectFields
}

func scanVCSEvent(row scanner) (*models.VCSEvent, error) {
	ve := &models.VCSEvent{}

	fields := []interface{}{
		&ve.Metadata.ID,
		&ve.Metadata.CreationTimestamp,
		&ve.Metadata.LastUpdatedTimestamp,
		&ve.Metadata.Version,
		&ve.CommitID,
		&ve.SourceReferenceName,
		&ve.WorkspaceID,
		&ve.Type,
		&ve.Status,
		&ve.RepositoryURL,
		&ve.ErrorMessage,
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	return ve, nil
}
