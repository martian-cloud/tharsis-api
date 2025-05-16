package db

//go:generate go tool mockery --name NotificationPreferences --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// NotificationPreferences encapsulates the logic to access preferences from the database
type NotificationPreferences interface {
	GetNotificationPreferences(ctx context.Context, input *GetNotificationPreferencesInput) (*NotificationPreferencesResult, error)
	UpdateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) (*models.NotificationPreference, error)
	CreateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) (*models.NotificationPreference, error)
	DeleteNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error
}

// NotificationPreferenceSortableField represents the fields that a preference can be sorted by
type NotificationPreferenceSortableField string

// GroupSortableField constants
const (
	NotificationPreferenceSortableFieldCreatedAtAsc  NotificationPreferenceSortableField = "CREATED_AT_ASC"
	NotificationPreferenceSortableFieldCreatedAtDesc NotificationPreferenceSortableField = "CREATED_AT_DESC"
	NotificationPreferenceSortableFieldUpdatedAtAsc  NotificationPreferenceSortableField = "UPDATED_AT_ASC"
	NotificationPreferenceSortableFieldUpdatedAtDesc NotificationPreferenceSortableField = "UPDATED_AT_DESC"
)

func (sf NotificationPreferenceSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case NotificationPreferenceSortableFieldCreatedAtAsc, NotificationPreferenceSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "notification_preferences", Col: "created_at"}
	case NotificationPreferenceSortableFieldUpdatedAtAsc, NotificationPreferenceSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "notification_preferences", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf NotificationPreferenceSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// NotificationPreferenceFilter contains the supported fields for filtering NotificationPreference resources
type NotificationPreferenceFilter struct {
	NotificationPreferenceIDs []string
	UserIDs                   []string
	NamespacePath             *string
	Global                    *bool
}

// GetNotificationPreferencesInput is the input for listing preferences
type GetNotificationPreferencesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *NotificationPreferenceSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *NotificationPreferenceFilter
}

// NotificationPreferencesResult contains the response data and page information
type NotificationPreferencesResult struct {
	PageInfo                *pagination.PageInfo
	NotificationPreferences []models.NotificationPreference
}

type notificationPreferences struct {
	dbClient *Client
}

var notificationPreferenceFieldList = append(metadataFieldList, "user_id", "scope", "custom_events")

// NewNotificationPreferences returns an instance of the NotificationPreferences interface
func NewNotificationPreferences(dbClient *Client) NotificationPreferences {
	return &notificationPreferences{dbClient: dbClient}
}

func (np *notificationPreferences) GetNotificationPreferences(ctx context.Context, input *GetNotificationPreferencesInput) (*NotificationPreferencesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetNotificationPreferences")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.NotificationPreferenceIDs != nil {
			ex = ex.Append(goqu.I("notification_preferences.id").In(input.Filter.NotificationPreferenceIDs))
		}
		if input.Filter.UserIDs != nil {
			ex = ex.Append(goqu.I("notification_preferences.user_id").In(input.Filter.UserIDs))
		}
		if input.Filter.NamespacePath != nil {
			ex = ex.Append(goqu.I("namespaces.path").Eq(*input.Filter.NamespacePath))
		}
		if input.Filter.Global != nil {
			if *input.Filter.Global {
				ex = ex.Append(goqu.I("notification_preferences.namespace_id").IsNull())
			} else {
				ex = ex.Append(goqu.I("notification_preferences.namespace_id").IsNotNull())
			}
		}
	}

	query := dialect.From(goqu.T("notification_preferences")).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"notification_preferences.namespace_id": goqu.I("namespaces.id")})).
		Select(np.getSelectFields()...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "notification_preferences", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, np.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.NotificationPreference{}
	for rows.Next() {
		item, err := scanNotificationPreference(rows)
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

	result := NotificationPreferencesResult{
		PageInfo:                rows.GetPageInfo(),
		NotificationPreferences: results,
	}

	return &result, nil
}

func (np *notificationPreferences) UpdateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) (*models.NotificationPreference, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateNotificationPreference")
	defer span.End()

	var customEvents []byte
	if preference.CustomEvents != nil {
		events, err := json.Marshal(preference.CustomEvents)
		if err != nil {
			return nil, err
		}
		customEvents = events
	}

	timestamp := currentTime()

	sql, args, err := dialect.From("notification_preferences").
		Prepared(true).
		With("notification_preferences",
			dialect.Update("notification_preferences").
				Set(
					goqu.Record{
						"version":       goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":    timestamp,
						"scope":         preference.Scope,
						"custom_events": customEvents,
					},
				).
				Where(goqu.Ex{"id": preference.Metadata.ID, "version": preference.Metadata.Version}).
				Returning("*"),
		).Select(np.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"notification_preferences.namespace_id": goqu.I("namespaces.id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedNotificationPreference, err := scanNotificationPreference(np.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedNotificationPreference, nil
}

func (np *notificationPreferences) CreateNotificationPreference(ctx context.Context, preference *models.NotificationPreference) (*models.NotificationPreference, error) {
	ctx, span := tracer.Start(ctx, "db.CreateNotificationPreference")
	defer span.End()

	var namespaceID *string
	if preference.NamespacePath != nil {

		namespace, err := getNamespaceByPath(ctx, np.dbClient.getConnection(ctx), *preference.NamespacePath)
		if err != nil {
			tracing.RecordError(span, err, "failed to get namespace by path")
			return nil, err
		}

		if namespace == nil {
			tracing.RecordError(span, nil, "Namespace not found")
			return nil, errors.New("Namespace not found", errors.WithErrorCode(errors.EInvalid))
		}

		namespaceID = &namespace.id
	}

	var customEvents []byte
	if preference.CustomEvents != nil {
		events, err := json.Marshal(preference.CustomEvents)
		if err != nil {
			return nil, err
		}
		customEvents = events
	}

	timestamp := currentTime()

	sql, args, err := dialect.From("notification_preferences").
		Prepared(true).
		With("notification_preferences",
			dialect.Insert("notification_preferences").
				Rows(goqu.Record{
					"id":            newResourceID(),
					"version":       initialResourceVersion,
					"created_at":    timestamp,
					"updated_at":    timestamp,
					"user_id":       preference.UserID,
					"scope":         preference.Scope,
					"custom_events": customEvents,
					"namespace_id":  namespaceID,
				}).
				Returning("*"),
		).Select(np.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"notification_preferences.namespace_id": goqu.I("namespaces.id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdNotificationPreference, err := scanNotificationPreference(np.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("user preference already exists", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_id":
					return nil, errors.New("namespace not found", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
				case "fk_user_id":
					return nil, errors.New("user not found", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
				default:
					return nil, errors.New("invalid foreign key", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
				}
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdNotificationPreference, nil
}

func (np *notificationPreferences) DeleteNotificationPreference(ctx context.Context, preference *models.NotificationPreference) error {
	ctx, span := tracer.Start(ctx, "db.DeleteNotificationPreference")
	defer span.End()

	sql, args, err := dialect.From("notification_preferences").
		Prepared(true).
		With("notification_preferences",
			dialect.Delete("notification_preferences").
				Where(
					goqu.Ex{
						"id":      preference.Metadata.ID,
						"version": preference.Metadata.Version,
					},
				).
				Returning("*"),
		).Select(np.getSelectFields()...).
		LeftJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"notification_preferences.namespace_id": goqu.I("namespaces.id")})).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanNotificationPreference(np.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (np *notificationPreferences) getSelectFields() []interface{} {
	selectFields := []interface{}{}

	for _, field := range notificationPreferenceFieldList {
		selectFields = append(selectFields, fmt.Sprintf("notification_preferences.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanNotificationPreference(row scanner) (*models.NotificationPreference, error) {
	preference := &models.NotificationPreference{}

	fields := []interface{}{
		&preference.Metadata.ID,
		&preference.Metadata.CreationTimestamp,
		&preference.Metadata.LastUpdatedTimestamp,
		&preference.Metadata.Version,
		&preference.UserID,
		&preference.Scope,
		&preference.CustomEvents,
		&preference.NamespacePath,
	}

	err := row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if preference.NamespacePath != nil {
		preference.Metadata.TRN = types.NotificationPreferenceModelType.BuildTRN(*preference.NamespacePath, preference.GetGlobalID())
	} else {
		preference.Metadata.TRN = types.NotificationPreferenceModelType.BuildTRN(preference.GetGlobalID())
	}

	return preference, nil
}
