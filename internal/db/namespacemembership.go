package db

//go:generate go tool mockery --name NamespaceMemberships --inpackage --case underscore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// CreateNamespaceMembershipInput is the input for creating a new namespace membership
type CreateNamespaceMembershipInput struct {
	UserID           *string
	ServiceAccountID *string
	TeamID           *string
	NamespacePath    string
	RoleID           string
}

// NamespaceMemberships encapsulates the logic to access namespace memberships from the database
type NamespaceMemberships interface {
	GetNamespaceMemberships(ctx context.Context, input *GetNamespaceMembershipsInput) (*NamespaceMembershipResult, error)
	GetNamespaceMembershipByID(ctx context.Context, id string) (*models.NamespaceMembership, error)
	CreateNamespaceMembership(ctx context.Context, input *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error)
	UpdateNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) (*models.NamespaceMembership, error)
	DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error
}

// NamespaceMembershipSortableField represents the fields that a namespace membership can be sorted by
type NamespaceMembershipSortableField string

// NamespaceMembershipSortableField constants
const (
	NamespaceMembershipSortableFieldUpdatedAtAsc      NamespaceMembershipSortableField = "UPDATED_AT_ASC"
	NamespaceMembershipSortableFieldUpdatedAtDesc     NamespaceMembershipSortableField = "UPDATED_AT_DESC"
	NamespaceMembershipSortableFieldNamespacePathAsc  NamespaceMembershipSortableField = "NAMESPACE_PATH_ASC"
	NamespaceMembershipSortableFieldNamespacePathDesc NamespaceMembershipSortableField = "NAMESPACE_PATH_DESC"
)

func (sf NamespaceMembershipSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case NamespaceMembershipSortableFieldUpdatedAtAsc, NamespaceMembershipSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "namespace_memberships", Col: "updated_at"}
	case NamespaceMembershipSortableFieldNamespacePathAsc, NamespaceMembershipSortableFieldNamespacePathDesc:
		return &pagination.FieldDescriptor{Key: "namespace_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (sf NamespaceMembershipSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// NamespaceMembershipFilter contains the supported fields for filtering NamespaceMembership resources
type NamespaceMembershipFilter struct {
	UserID                 *string
	ServiceAccountID       *string
	TeamID                 *string
	GroupID                *string
	WorkspaceID            *string
	NamespacePathPrefix    *string
	RoleID                 *string
	NamespacePaths         []string
	NamespaceMembershipIDs []string
}

// GetNamespaceMembershipsInput is the input for listing namespace memberships
type GetNamespaceMembershipsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *NamespaceMembershipSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *NamespaceMembershipFilter
}

// NamespaceMembershipResult contains the response data and page information
type NamespaceMembershipResult struct {
	PageInfo             *pagination.PageInfo
	NamespaceMemberships []models.NamespaceMembership
}

type namespaceMemberships struct {
	dbClient *Client
}

var namespaceMembershipFieldList = append(metadataFieldList, "role_id", "user_id", "service_account_id", "team_id")

// NewNamespaceMemberships returns an instance of the NamespaceMemberships interface
func NewNamespaceMemberships(dbClient *Client) NamespaceMemberships {
	return &namespaceMemberships{dbClient: dbClient}
}

func (m *namespaceMemberships) GetNamespaceMembershipByID(ctx context.Context, id string) (*models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespaceMembershipByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("namespace_memberships").
		Prepared(true).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
		Select(m.getSelectFields()...).
		Where(goqu.Ex{"namespace_memberships.id": id}).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	namespaceMembership, err := scanNamespaceMembership(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return namespaceMembership, nil
}

func (m *namespaceMemberships) CreateNamespaceMembership(ctx context.Context,
	input *CreateNamespaceMembershipInput,
) (*models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "db.CreateNamespaceMembership")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	namespace, err := getNamespaceByPath(ctx, m.dbClient.getConnection(ctx), input.NamespacePath)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace by path")
		return nil, err
	}

	if namespace == nil {
		tracing.RecordError(span, nil, "Namespace not found")
		return nil, errors.New("Namespace not found", errors.WithErrorCode(errors.ENotFound))
	}

	timestamp := currentTime()

	record := goqu.Record{
		"id":           newResourceID(),
		"version":      initialResourceVersion,
		"created_at":   timestamp,
		"updated_at":   timestamp,
		"namespace_id": namespace.id,
		"role_id":      input.RoleID,
	}

	// Should be that exactly one of these takes effect.
	switch {
	case input.UserID != nil:
		record["user_id"] = input.UserID
	case input.ServiceAccountID != nil:
		record["service_account_id"] = input.ServiceAccountID
	case input.TeamID != nil:
		record["team_id"] = input.TeamID
	}

	sql, args, err := dialect.Insert("namespace_memberships").
		Prepared(true).
		Rows(record).
		Returning(namespaceMembershipFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdNamespaceMembership, err := scanNamespaceMembership(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "member already exists")
				return nil, errors.New("member already exists", errors.WithErrorCode(errors.EConflict))
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_memberships_user_id":
					tracing.RecordError(span, nil, "user does not exist")
					return nil, errors.New("user does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_namespace_memberships_service_account_id":
					tracing.RecordError(span, nil, "service account does not exist")
					return nil, errors.New("service account does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_namespace_memberships_team_id":
					tracing.RecordError(span, nil, "team does not exist")
					return nil, errors.New("team does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_namespace_memberships_namespace_id":
					tracing.RecordError(span, nil, "namespace does not exist")
					return nil, errors.New("namespace does not exist", errors.WithErrorCode(errors.ENotFound))
				case "fk_namespace_memberships_role_id":
					tracing.RecordError(span, nil, "role does not exist")
					return nil, errors.New("role does not exist", errors.WithErrorCode(errors.ENotFound))
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	createdNamespaceMembership.Namespace.Path = input.NamespacePath
	createdNamespaceMembership.Namespace.GroupID = &namespace.groupID
	createdNamespaceMembership.Namespace.WorkspaceID = &namespace.workspaceID

	return createdNamespaceMembership, nil
}

func (m *namespaceMemberships) UpdateNamespaceMembership(ctx context.Context,
	namespaceMembership *models.NamespaceMembership,
) (*models.NamespaceMembership, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateNamespaceMembership")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("namespace_memberships").
		Prepared(true).
		Set(goqu.Record{
			"version":    goqu.L("? + ?", goqu.C("version"), 1),
			"updated_at": timestamp,
			"role_id":    namespaceMembership.RoleID,
		}).
		Where(goqu.Ex{"id": namespaceMembership.Metadata.ID, "version": namespaceMembership.Metadata.Version}).Returning(namespaceMembershipFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedNamespaceMembership, err := scanNamespaceMembership(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "member already exists")
				return nil, errors.New("member already exists", errors.WithErrorCode(errors.EConflict))
			}
			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_namespace_memberships_role_id":
					tracing.RecordError(span, nil, "role does not exist")
					return nil, errors.New("role does not exist", errors.WithErrorCode(errors.ENotFound))
				}
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	updatedNamespaceMembership.Namespace = namespaceMembership.Namespace

	return updatedNamespaceMembership, nil
}

func (m *namespaceMemberships) DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error {
	ctx, span := tracer.Start(ctx, "db.DeleteNamespaceMembership")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("namespace_memberships").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      namespaceMembership.Metadata.ID,
				"version": namespaceMembership.Metadata.Version,
			},
		).Returning(namespaceMembershipFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanNamespaceMembership(m.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

// GetNamespaceMemberships queries for namespaces visible by or connected to the specified entities.
//
// In the case of a user ID, this method returns both direct membership and indirect membership via
// a team member relationship.
func (m *namespaceMemberships) GetNamespaceMemberships(ctx context.Context,
	input *GetNamespaceMembershipsInput,
) (*NamespaceMembershipResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetNamespaceMemberships")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {

		if input.Filter.UserID != nil {
			ex = ex.Append(goqu.Or(

				// This filters for direct namespace membership.
				goqu.I("namespace_memberships.user_id").Eq(*input.Filter.UserID),

				// This filters for indirect via the user being a team member.
				goqu.I("namespace_memberships.team_id").In(
					dialect.From("team_members").
						Select("team_id").
						Where(goqu.I("team_members.user_id").Eq(*input.Filter.UserID))),
			))
		}
		if input.Filter.RoleID != nil {
			ex = ex.Append(goqu.I("namespace_memberships.role_id").Eq(*input.Filter.RoleID))
		}
		if input.Filter.ServiceAccountID != nil {
			ex = ex.Append(goqu.I("namespace_memberships.service_account_id").Eq(*input.Filter.ServiceAccountID))
		}
		if input.Filter.TeamID != nil {
			ex = ex.Append(goqu.I("namespace_memberships.team_id").Eq(*input.Filter.TeamID))
		}
		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("namespaces.group_id").Eq(*input.Filter.GroupID))
		}
		if input.Filter.WorkspaceID != nil {
			ex = ex.Append(goqu.I("namespaces.workspace_id").Eq(*input.Filter.WorkspaceID))
		}
		if input.Filter.NamespacePaths != nil {
			// This check avoids an SQL syntax error if an empty slice is provided.
			if len(input.Filter.NamespacePaths) > 0 {
				ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.NamespacePaths))
			}
		}

		if input.Filter.NamespacePathPrefix != nil {
			ex = ex.Append(goqu.Or(
				goqu.I("namespaces.path").Eq(*input.Filter.NamespacePathPrefix),
				goqu.I("namespaces.path").Like(*input.Filter.NamespacePathPrefix+"/%"),
			))
		}

		if input.Filter.NamespaceMembershipIDs != nil {
			// This check avoids an SQL syntax error if an empty slice is provided.
			if len(input.Filter.NamespaceMembershipIDs) > 0 {
				ex = ex.Append(goqu.I("namespace_memberships.id").In(input.Filter.NamespaceMembershipIDs))
			}
		}
	}

	query := dialect.From("namespace_memberships").
		Select(m.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "namespace_memberships", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, m.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.NamespaceMembership{}
	for rows.Next() {
		item, err := scanNamespaceMembership(rows, true)
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

	result := NamespaceMembershipResult{
		PageInfo:             rows.GetPageInfo(),
		NamespaceMemberships: results,
	}

	return &result, nil
}

func (m *namespaceMemberships) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range namespaceMembershipFieldList {
		selectFields = append(selectFields, fmt.Sprintf("namespace_memberships.%s", field))
	}

	selectFields = append(selectFields, "namespaces.id")
	selectFields = append(selectFields, "namespaces.path")
	selectFields = append(selectFields, "namespaces.group_id")
	selectFields = append(selectFields, "namespaces.workspace_id")

	return selectFields
}

func scanNamespaceMembership(row scanner, withNamespacePath bool) (*models.NamespaceMembership, error) {
	namespaceMembership := &models.NamespaceMembership{}

	var namespaceID, namespacePath string
	var groupID, workspaceID sql.NullString
	var userID sql.NullString
	var serviceAccountID sql.NullString
	var teamID sql.NullString

	fields := []interface{}{
		&namespaceMembership.Metadata.ID,
		&namespaceMembership.Metadata.CreationTimestamp,
		&namespaceMembership.Metadata.LastUpdatedTimestamp,
		&namespaceMembership.Metadata.Version,
		&namespaceMembership.RoleID,
		&userID,
		&serviceAccountID,
		&teamID,
	}

	if withNamespacePath {
		fields = append(fields, &namespaceID)
		fields = append(fields, &namespacePath)
		fields = append(fields, &groupID)
		fields = append(fields, &workspaceID)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if withNamespacePath {
		namespaceMembership.Namespace.ID = namespaceID
		namespaceMembership.Namespace.Path = namespacePath
	}

	if groupID.Valid {
		namespaceMembership.Namespace.GroupID = &groupID.String
	}

	if workspaceID.Valid {
		namespaceMembership.Namespace.WorkspaceID = &workspaceID.String
	}

	if userID.Valid {
		namespaceMembership.UserID = &userID.String
	}

	if serviceAccountID.Valid {
		namespaceMembership.ServiceAccountID = &serviceAccountID.String
	}

	if teamID.Valid {
		namespaceMembership.TeamID = &teamID.String
	}

	return namespaceMembership, nil
}

type namespaceMembershipExpressionBuilder struct {
	userID           *string
	serviceAccountID *string
}

func (n namespaceMembershipExpressionBuilder) build() exp.Expression {
	var whereEx exp.Expression

	if n.userID != nil {
		// If dealing with a user ID, must also check team member relationships.
		whereEx = goqu.Or().
			Append(goqu.I("namespace_memberships.user_id").Eq(*n.userID)).
			Append(
				goqu.I("namespace_memberships.team_id").In(
					dialect.From("team_members").
						Select("team_id").
						Where(goqu.I("team_members.user_id").Eq(*n.userID))))
	} else {
		whereEx = goqu.I("namespace_memberships.service_account_id").Eq(*n.serviceAccountID)
	}

	return goqu.Or(
		goqu.I("namespaces.path").In(
			dialect.From("namespace_memberships").
				Select(goqu.L("path")).
				InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
				Where(whereEx),
		),
		goqu.I("namespaces.path").Like(goqu.Any(
			dialect.From("namespace_memberships").
				Select(goqu.L("path || '/%'")).
				InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
				Where(whereEx),
		)),
	)
}
