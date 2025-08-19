package db

//go:generate go tool mockery --name UserSessions --inpackage --case underscore

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

// UserSessions encapsulates the logic to access user sessions from the database
type UserSessions interface {
	GetUserSessionByID(ctx context.Context, id string) (*models.UserSession, error)
	GetUserSessionByTRN(ctx context.Context, trn string) (*models.UserSession, error)
	GetUserSessions(ctx context.Context, input *GetUserSessionsInput) (*UserSessionsResult, error)
	CreateUserSession(ctx context.Context, session *models.UserSession) (*models.UserSession, error)
	UpdateUserSession(ctx context.Context, session *models.UserSession) (*models.UserSession, error)
	DeleteUserSession(ctx context.Context, session *models.UserSession) error
}

// UserSessionSortableField represents the fields that user sessions can be sorted by
type UserSessionSortableField string

// UserSessionSortableField constants
const (
	UserSessionSortableFieldCreatedAtAsc   UserSessionSortableField = "CREATED_AT_ASC"
	UserSessionSortableFieldCreatedAtDesc  UserSessionSortableField = "CREATED_AT_DESC"
	UserSessionSortableFieldExpirationAsc  UserSessionSortableField = "EXPIRATION_ASC"
	UserSessionSortableFieldExpirationDesc UserSessionSortableField = "EXPIRATION_DESC"
)

func (us UserSessionSortableField) getValue() string {
	return string(us)
}

func (us UserSessionSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch us {
	case UserSessionSortableFieldCreatedAtAsc, UserSessionSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "user_sessions", Col: "created_at"}
	case UserSessionSortableFieldExpirationAsc, UserSessionSortableFieldExpirationDesc:
		return &pagination.FieldDescriptor{Key: "expiration", Table: "user_sessions", Col: "expiration"}
	default:
		return nil
	}
}

func (us UserSessionSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(us), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// UserSessionFilter contains the supported fields for filtering UserSession resources
type UserSessionFilter struct {
	UserID         *string
	UserSessionIDs []string
	RefreshTokenID *string
}

// GetUserSessionsInput is the input for listing user sessions
type GetUserSessionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *UserSessionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *UserSessionFilter
}

// UserSessionsResult contains the response data and page information
type UserSessionsResult struct {
	PageInfo     *pagination.PageInfo
	UserSessions []models.UserSession
}

type userSessions struct {
	dbClient *Client
}

var userSessionFieldList = append(metadataFieldList, "user_id", "refresh_token_id", "expiration", "user_agent")

// NewUserSessions returns an instance of the UserSessions interface
func NewUserSessions(dbClient *Client) UserSessions {
	return &userSessions{dbClient: dbClient}
}

func (u *userSessions) GetUserSessionByID(ctx context.Context, id string) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserSessionByID")
	defer span.End()

	return u.getUserSession(ctx, goqu.Ex{"user_sessions.id": id})
}

func (u *userSessions) GetUserSessionByTRN(ctx context.Context, trn string) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserSessionByTRN")
	defer span.End()

	resourcePath, err := types.UserSessionModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	// Parse the resource path: username/session_global_id
	parts := strings.Split(resourcePath, "/")
	if len(parts) != 2 {
		return nil, errors.New("invalid user session TRN format: expected username/session_id", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	username := parts[0]
	sessionGlobalID := parts[1]

	// Parse the global ID to get the actual session ID
	parsedGID, err := gid.ParseGlobalID(sessionGlobalID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse session global ID", errors.WithSpan(span))
	}

	// Query by username and session ID
	return u.getUserSession(ctx, goqu.Ex{
		"users.username":   username,
		"user_sessions.id": parsedGID.ID,
	})
}

func (u *userSessions) GetUserSessions(ctx context.Context, input *GetUserSessionsInput) (*UserSessionsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserSessions")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.UserID != nil {
			ex = ex.Append(goqu.I("user_sessions.user_id").Eq(*input.Filter.UserID))
		}

		if len(input.Filter.UserSessionIDs) > 0 {
			ex = ex.Append(goqu.I("user_sessions.id").In(input.Filter.UserSessionIDs))
		}

		if input.Filter.RefreshTokenID != nil {
			ex = ex.Append(goqu.I("user_sessions.refresh_token_id").Eq(*input.Filter.RefreshTokenID))
		}
	}

	query := dialect.From(goqu.T("user_sessions")).
		Select(u.getSelectFields()...).
		InnerJoin(goqu.T("users"), goqu.On(goqu.Ex{"user_sessions.user_id": goqu.I("users.id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "user_sessions", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, u.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	// Scan rows
	results := []models.UserSession{}
	for rows.Next() {
		item, err := scanUserSession(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	result := UserSessionsResult{
		PageInfo:     rows.GetPageInfo(),
		UserSessions: results,
	}

	return &result, nil
}

func (u *userSessions) CreateUserSession(ctx context.Context, session *models.UserSession) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "db.CreateUserSession")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("user_sessions").
		Prepared(true).
		With("user_sessions",
			dialect.Insert("user_sessions").Rows(
				goqu.Record{
					"id":               newResourceID(),
					"version":          initialResourceVersion,
					"created_at":       timestamp,
					"updated_at":       timestamp,
					"user_id":          session.UserID,
					"refresh_token_id": session.RefreshTokenID,
					"expiration":       session.Expiration,
					"user_agent":       session.UserAgent,
				}).Returning("*"),
		).Select(u.getSelectFields()...).
		InnerJoin(goqu.T("users"), goqu.On(goqu.Ex{"user_sessions.user_id": goqu.I("users.id")})).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdUserSession, err := scanUserSession(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("invalid user ID", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}
			if isUniqueViolation(pgErr) {
				return nil, errors.New("session ID already exists", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdUserSession, nil
}

func (u *userSessions) UpdateUserSession(ctx context.Context, session *models.UserSession) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateUserSession")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.From("user_sessions").
		Prepared(true).
		With("user_sessions",
			dialect.Update("user_sessions").
				Set(goqu.Record{
					"version":          goqu.L("? + ?", goqu.C("version"), 1),
					"updated_at":       timestamp,
					"refresh_token_id": session.RefreshTokenID,
					"expiration":       session.Expiration,
				}).Where(goqu.Ex{"id": session.Metadata.ID, "version": session.Metadata.Version}).
				Returning("*"),
		).Select(u.getSelectFields()...).
		InnerJoin(goqu.T("users"), goqu.On(goqu.Ex{"user_sessions.user_id": goqu.I("users.id")})).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedUserSession, err := scanUserSession(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedUserSession, nil
}

func (u *userSessions) DeleteUserSession(ctx context.Context, session *models.UserSession) error {
	ctx, span := tracer.Start(ctx, "db.DeleteUserSession")
	defer span.End()

	sql, args, err := dialect.From("user_sessions").
		Prepared(true).
		With("user_sessions",
			dialect.Delete("user_sessions").
				Where(goqu.Ex{"id": session.Metadata.ID, "version": session.Metadata.Version}).
				Returning("*"),
		).Select(u.getSelectFields()...).
		InnerJoin(goqu.T("users"), goqu.On(goqu.Ex{"user_sessions.user_id": goqu.I("users.id")})).
		ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	_, err = scanUserSession(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (u *userSessions) getUserSession(ctx context.Context, exp goqu.Ex) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "db.getUserSession")
	defer span.End()

	query := dialect.From(goqu.T("user_sessions")).
		Prepared(true).
		Select(u.getSelectFields()...).
		InnerJoin(goqu.T("users"), goqu.On(goqu.Ex{"user_sessions.user_id": goqu.I("users.id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	session, err := scanUserSession(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, errors.Wrap(pgErr, "invalid ID; %s", pgErr.Message, errors.WithSpan(span), errors.WithErrorCode(errors.EInvalid))
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return session, nil
}

func (*userSessions) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range userSessionFieldList {
		selectFields = append(selectFields, fmt.Sprintf("user_sessions.%s", field))
	}

	selectFields = append(selectFields, "users.username")

	return selectFields
}

func scanUserSession(row scanner) (*models.UserSession, error) {
	session := &models.UserSession{}
	var username string

	fields := []interface{}{
		&session.Metadata.ID,
		&session.Metadata.CreationTimestamp,
		&session.Metadata.LastUpdatedTimestamp,
		&session.Metadata.Version,
		&session.UserID,
		&session.RefreshTokenID,
		&session.Expiration,
		&session.UserAgent,
		&username,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	session.Metadata.TRN = types.UserSessionModelType.BuildTRN(username, session.GetGlobalID())

	return session, nil
}
