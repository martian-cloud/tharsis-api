package db

//go:generate mockery --name Users --inpackage --case underscore

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

func (js UserSortableField) getFieldDescriptor() *fieldDescriptor {
	switch js {
	case UserSortableFieldUpdatedAtAsc, UserSortableFieldUpdatedAtDesc:
		return &fieldDescriptor{key: "updated_at", table: "users", col: "updated_at"}
	default:
		return nil
	}
}

func (js UserSortableField) getSortDirection() SortDirection {
	if strings.HasSuffix(string(js), "_DESC") {
		return DescSort
	}
	return AscSort
}

// UserFilter contains the supported fields for filtering User resources
type UserFilter struct {
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
	PaginationOptions *PaginationOptions
	// Filter is used to filter the results
	Filter *UserFilter
}

// UsersResult contains the response data and page information
type UsersResult struct {
	PageInfo *PageInfo
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
	return u.getUser(ctx, goqu.Ex{"users.id": id})
}

func (u *users) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return u.getUser(ctx, goqu.Ex{"users.email": email})
}

func (u *users) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return u.getUser(ctx, goqu.Ex{"users.username": username})
}

func (u *users) GetUserBySCIMExternalID(ctx context.Context, scimExternalID string) (*models.User, error) {
	return u.getUser(ctx, goqu.Ex{"users.scim_external_id": scimExternalID})
}

func (u *users) GetUserByExternalID(ctx context.Context, issuer string, externalID string) (*models.User, error) {
	query := dialect.From(goqu.T("users")).
		Select(u.getSelectFields()...).
		InnerJoin(goqu.T("user_external_identities"), goqu.On(goqu.Ex{"users.id": goqu.I("user_external_identities.user_id")})).
		Where(goqu.Ex{"user_external_identities.external_id": externalID, "user_external_identities.issuer": issuer})

	sql, _, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	user, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(
		ctx, sql))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (u *users) LinkUserWithExternalID(ctx context.Context, issuer string, externalID string, userID string) error {
	timestamp := currentTime()

	sql, _, err := dialect.Insert("user_external_identities").
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
		return err
	}

	_, err = u.dbClient.getConnection(ctx).Exec(ctx, sql)

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return errors.NewError(errors.EConflict, fmt.Sprintf("user with external id %s already exists for issuer %s", externalID, issuer))
			}
		}
		return err
	}

	return nil
}

func (u *users) GetUsers(ctx context.Context, input *GetUsersInput) (*UsersResult, error) {
	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.UserIDs != nil {
			ex["users.id"] = input.Filter.UserIDs
		}
		if input.Filter.UsernamePrefix != nil && *input.Filter.UsernamePrefix != "" {
			ex["users.username"] = goqu.Op{"like": *input.Filter.UsernamePrefix + "%%"}
		}
		if input.Filter.SCIMExternalID {
			ex["users.scim_external_id"] = goqu.Op{"isNot": nil}
		}
		if input.Filter.Active {
			ex["users.active"] = input.Filter.Active // Return only active users.
		}
	}

	query := dialect.From(goqu.T("users")).Select(userFieldList...).Where(ex)

	sortDirection := AscSort

	var sortBy *fieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := newPaginatedQueryBuilder(
		input.PaginationOptions,
		&fieldDescriptor{key: "id", table: "users", col: "id"},
		sortBy,
		sortDirection,
		userFieldResolver,
	)

	if err != nil {
		return nil, err
	}

	rows, err := qBuilder.execute(ctx, u.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.User{}
	for rows.Next() {
		item, err := scanUser(rows)
		if err != nil {
			return nil, err
		}

		results = append(results, *item)
	}

	if err := rows.finalize(&results); err != nil {
		return nil, err
	}

	result := UsersResult{
		PageInfo: rows.getPageInfo(),
		Users:    results,
	}

	return &result, nil
}

func (u *users) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Update("users").Set(
		goqu.Record{
			"version":          goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at":       timestamp,
			"username":         user.Username,
			"email":            user.Email,
			"scim_external_id": nullableString(user.SCIMExternalID),
			"active":           user.Active,
		},
	).Where(goqu.Ex{"id": user.Metadata.ID, "version": user.Metadata.Version}).Returning(userFieldList...).ToSQL()

	if err != nil {
		return nil, err
	}

	updatedUser, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, fmt.Sprintf("user with username %s already exists", user.Username))
			}
		}
		return nil, err
	}

	return updatedUser, nil
}

func (u *users) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	timestamp := currentTime()

	sql, _, err := dialect.Insert("users").
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
		return nil, err
	}

	createdUser, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql))

	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.NewError(errors.EConflict, fmt.Sprintf("user with username %s already exists", user.Username))
			}
		}
		return nil, err
	}

	return createdUser, nil
}

func (u *users) DeleteUser(ctx context.Context, user *models.User) error {
	sql, _, err := dialect.Delete("users").Where(
		goqu.Ex{
			"id":      user.Metadata.ID,
			"version": user.Metadata.Version,
		},
	).Returning(userFieldList...).ToSQL()

	if err != nil {
		return err
	}

	if _, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(ctx, sql)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return err
	}

	return nil
}

func (u *users) getUser(ctx context.Context, exp goqu.Ex) (*models.User, error) {
	query := dialect.From(goqu.T("users")).
		Select(userFieldList...).Where(exp)

	sql, _, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	user, err := scanUser(u.dbClient.getConnection(ctx).QueryRow(
		ctx, sql))

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

func userFieldResolver(key string, model interface{}) (string, error) {
	user, ok := model.(*models.User)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Expected user type, got %T", model))
	}

	val, ok := metadataFieldResolver(key, &user.Metadata)
	if !ok {
		return "", errors.NewError(errors.EInternal, fmt.Sprintf("Invalid field key requested %s", key))
	}

	return val, nil
}
