package db

//go:generate mockery --name Users --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Users encapsulates the logic to access users from the database
type Users interface {
	GetUserBySCIMExternalID(ctx context.Context, scimExternalID string) (*models.User, error)
	GetUserByExternalID(ctx context.Context, issuer string, externalID string) (*models.User, error)
	LinkUserWithExternalID(ctx context.Context, issuer string, externalID string, userID string) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUsers(ctx context.Context, input *GetUsersInput) (*UsersResult, error)
	UpdateUser(ctx context.Context, user *models.User) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	DeleteUser(ctx context.Context, user *models.User) error
}

// UserSortableField represents the fields that a user can be sorted by
type UserSortableField string

// UserSortableField constants
const (
	UserSortableFieldUpdatedAtAsc  UserSortableField = "UPDATED_AT_ASC"
	UserSortableFieldUpdatedAtDesc UserSortableField = "UPDATED_AT_DESC"
)

func (js UserSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch js {
	case UserSortableFieldUpdatedAtAsc, UserSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "users", Col: "updated_at"}
	default:
		return nil
	}
}

func (js UserSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(js), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// UserFilter contains the supported fields for filtering User resources
type UserFilter struct {
	Search         *string
	UsernamePrefix *string
	UserIDs        []string
	SCIMExternalID bool
	Active         bool
}

// GetUsersInput is the input for listing users
type GetUsersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *UserSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *UserFilter
}

// UsersResult contains the response data and page information
type UsersResult struct {
	PageInfo *pagination.PageInfo
	Users    []models.User
}

type users struct {
	dbClient *Client
}

var userFieldList = append(metadataFieldList, "username", "email", "admin", "scim_external_id", "active")

// NewUsers returns an instance of the Users interface
func NewUsers(dbClient *Client) Users {
	return &users{dbClient: dbClient}
}

func (u *users) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return u.getUser(ctx, goqu.Ex{"users.id": id})
}

func (u *users) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserByEmail")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return u.getUser(ctx, goqu.Ex{"users.email": email})
}

func (u *users) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserByUsername")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return u.getUser(ctx, goqu.Ex{"users.username": username})
}

func (u *users) GetUserBySCIMExternalID(ctx context.Context, scimExternalID string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserBySCIMExternalID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return u.getUser(ctx, goqu.Ex{"users.scim_external_id": scimExternalID})
}

func (u *users) GetUserByExternalID(ctx context.Context, issuer string, externalID string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.GetUserByExternalID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	query := dialect.From(goqu.T("users")).
		Prepared(true).
		Select(u.getSelectFields()...).
		InnerJoin(goqu.T("user_external_identities"), goqu.On(goqu.Ex{"users.id": goqu.I("user_external_identities.user_id")})).
		Where(goqu.Ex{"user_external_identities.external_id": externalID, "user_external_identities.issuer": issuer})

	sql, args, err := query.ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	user, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return user, nil
}

func (u *users) LinkUserWithExternalID(ctx context.Context, issuer string, externalID string, userID string) error {
	ctx, span := tracer.Start(ctx, "db.LinkUserWithExternalID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("user_external_identities").
		Prepared(true).
		Rows(goqu.Record{
			"id":          newResourceID(),
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"issuer":      issuer,
			"external_id": externalID,
			"user_id":     userID,
		}).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	_, err = u.dbClient.getConnection(ctx).Exec(ctx, sql, args...)

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"user with external id %s already exists for issuer %s", externalID, issuer)
				return errors.New("user with external id %s already exists for issuer %s", externalID, issuer, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (u *users) GetUsers(ctx context.Context, input *GetUsersInput) (*UsersResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetUsers")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.UserIDs != nil {
			ex = ex.Append(goqu.I("users.id").In(input.Filter.UserIDs))
		}
		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(goqu.I("users.username").ILike("%" + *input.Filter.Search + "%"))
		}
		if input.Filter.UsernamePrefix != nil && *input.Filter.UsernamePrefix != "" {
			ex = ex.Append(goqu.I("users.username").Like(*input.Filter.UsernamePrefix + "%"))
		}
		if input.Filter.SCIMExternalID {
			ex = ex.Append(goqu.I("users.scim_external_id").IsNotNull())
		}
		if input.Filter.Active {
			ex = ex.Append(goqu.I("users.active").IsTrue()) // Return only active users.
		}
	}

	query := dialect.From(goqu.T("users")).
		Select(userFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "users", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, u.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.User{}
	for rows.Next() {
		item, err := scanUser(rows)
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

	result := UsersResult{
		PageInfo: rows.GetPageInfo(),
		Users:    results,
	}

	return &result, nil
}

func (u *users) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateUser")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("users").
		Prepared(true).
		Set(
			goqu.Record{
				"version":          goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":       timestamp,
				"username":         user.Username,
				"email":            user.Email,
				"scim_external_id": nullableString(user.SCIMExternalID),
				"active":           user.Active,
				"admin":            user.Admin,
			},
		).Where(goqu.Ex{"id": user.Metadata.ID, "version": user.Metadata.Version}).Returning(userFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedUser, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil,
					"user with username %s already exists", user.Username)
				return nil, errors.New("user with username %s already exists", user.Username, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedUser, nil
}

func (u *users) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "db.CreateUser")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("users").
		Prepared(true).
		Rows(goqu.Record{
			"id":               newResourceID(),
			"version":          initialResourceVersion,
			"created_at":       timestamp,
			"updated_at":       timestamp,
			"username":         user.Username,
			"email":            user.Email,
			"admin":            user.Admin,
			"scim_external_id": nullableString(user.SCIMExternalID),
			"active":           user.Active,
		}).
		Returning(userFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdUser, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "user with username %s already exists", user.Username)
				return nil, errors.New("user with username %s already exists", user.Username, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdUser, nil
}

func (u *users) DeleteUser(ctx context.Context, user *models.User) error {
	ctx, span := tracer.Start(ctx, "db.DeleteUser")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("users").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      user.Metadata.ID,
				"version": user.Metadata.Version,
			},
		).Returning(userFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (u *users) getUser(ctx context.Context, exp goqu.Ex) (*models.User, error) {
	query := dialect.From(goqu.T("users")).
		Prepared(true).
		Select(userFieldList...).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	user, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (u *users) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range userFieldList {
		selectFields = append(selectFields, fmt.Sprintf("users.%s", field))
	}

	return selectFields
}

func scanUser(row scanner) (*models.User, error) {
	var scimExternalID sql.NullString
	user := &models.User{}

	fields := []interface{}{
		&user.Metadata.ID,
		&user.Metadata.CreationTimestamp,
		&user.Metadata.LastUpdatedTimestamp,
		&user.Metadata.Version,
		&user.Username,
		&user.Email,
		&user.Admin,
		&scimExternalID,
		&user.Active,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if scimExternalID.Valid {
		user.SCIMExternalID = scimExternalID.String
	}

	return user, nil
}
