package db

//go:generate mockery --name VCSEvents --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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

func (sf VCSEventSortableField) getFieldDescriptor() *fieldDescriptor {
	switch sf {
	case VCSEventSortableFieldCreatedAtAsc, VCSEventSortableFieldCreatedAtDesc:
		return &fieldDescriptor{key: "created_at", table: "vcs_events", col: "created_at"}
	case VCSEventSortableFieldUpdatedAtAsc, VCSEventSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "vcs_events", col: "updated_at"}
	default:
		return nil
	}
}

func (sf VCSEventSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return DescSort
	}
	return AscSort
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
	PaginationOptions *PaginationOptions
	// Filter contains the supported fields for filtering VCSEvent resources
	Filter *VCSEventFilter
}

// VCSEventsResult contains the response data and page information
type VCSEventsResult struct {
	PageInfo  *PageInfo
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
	return ve.getEvent(ctx, goqu.Ex{"vcs_events.id": id})
}

func (ve *vcsEvents) GetEvents(ctx context.Context, input *GetVCSEventsInput) (*VCSEventsResult, error) {

	ex := goqu.And()
	if input.Filter != nil {
		if input.Filter.VCSEventIDs != nil {
			ex = ex.Append(goqu.I("vcs_events.id").In(input.Filter.VCSEventIDs))
		}

		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("vcs_events.workspace_id").Eq(input.Filter.WorkspaceID))
		}
	}

	query := dialect.From("vcs_events").Select(ve.getSelectFields()...).Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "vcs_events", col: "id"},
		sortBy,
		sortDirection,
		vcsEventFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, ve.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.VCSEvent{}
	for rows.Next() {
		item, err := scanVCSEvent(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := VCSEventsResult{
		PageInfo:  rows.getPageInfo(),
		VCSEvents: results,
	}

	return &result, nil
}

func (ve *vcsEvents) CreateEvent(ctx context.Context, event *models.VCSEvent) (*models.VCSEvent, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Insert("vcs_events").
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
		return nil, err
	}

	createdEvent, err := scanVCSEvent(ve.dbClient.getConnection(ctx).QueryRow(ctx, sql))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_workspace_id":
					return nil, errors.NewError(errors.ENotFound, "workspace does not exist")
				}
			}
		}
		return nil, err
	}

	return createdEvent, nil
}

func (ve *vcsEvents) UpdateEvent(ctx context.Context, event *models.VCSEvent) (*models.VCSEvent, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Update("vcs_events").Set(
		goqu.Record{
			"version":       goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":    timestamp,
			"status":        event.Status,
			"error_message": event.ErrorMessage,
		},
	).Where(goqu.Ex{"id": event.Metadata.ID, "version": event.Metadata.Version}).
		Returning(vcsEventsFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedEvent, err := scanVCSEvent(ve.dbClient.getConnection(ctx).QueryRow(ctx, sql))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, err
	}

	return updatedEvent, nil
}

func (ve *vcsEvents) getEvent(ctx context.Context, exp goqu.Ex) (*models.VCSEvent, error) {
	sql, _, err := dialect.From(goqu.T("vcs_events")).Select(ve.getSelectFields()...).Where(exp).ToSQL()
	if err != nil {
		return nil, err
	}

	event, err := scanVCSEvent(ve.dbClient.getConnection(ctx).QueryRow(ctx, sql))
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

func vcsEventFieldResolver(key string, model interface{}) (string, error) {
	vcsEvent, ok := model.(*models.VCSEvent)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected vcs event type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &vcsEvent.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
