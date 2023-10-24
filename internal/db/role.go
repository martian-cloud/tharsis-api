package db

//go:generate mockery --name Roles --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Roles encapsulates the logic to access Tharsis roles from the database.
type Roles interface {
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	GetRoleByID(ctx context.Context, id string) (*models.Role, error)
	GetRoles(ctx context.Context, input *GetRolesInput) (*RolesResult, error)
	CreateRole(ctx context.Context, role *models.Role) (*models.Role, error)
	UpdateRole(ctx context.Context, role *models.Role) (*models.Role, error)
	DeleteRole(ctx context.Context, role *models.Role) error
}

// RoleSortableField represents the fields that a role can be sorted by
type RoleSortableField string

// RoleSortableField constants
const (
	RoleSortableFieldNameAsc       RoleSortableField = "NAME_ASC"
	RoleSortableFieldNameDesc      RoleSortableField = "NAME_DESC"
	RoleSortableFieldUpdatedAtAsc  RoleSortableField = "UPDATED_AT_ASC"
	RoleSortableFieldUpdatedAtDesc RoleSortableField = "UPDATED_AT_DESC"
)

func (r RoleSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch r {
	case RoleSortableFieldNameAsc, RoleSortableFieldNameDesc:
		return &pagination.FieldDescriptor{Key: "name", Table: "roles", Col: "name"}
	case RoleSortableFieldUpdatedAtAsc, RoleSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "roles", Col: "updated_at"}
	default:
		return nil
	}
}

func (r RoleSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(r), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// RoleFilter contains the supported fields for filtering Role resources
type RoleFilter struct {
	RoleNamePrefix *string
	RoleIDs        []string
}

// GetRolesInput is the input for listing roles
type GetRolesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *RoleSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *RoleFilter
}

// RolesResult contains the response data and page information
type RolesResult struct {
	PageInfo *pagination.PageInfo
	Roles    []models.Role
}

type roles struct {
	dbClient *Client
}

var rolesFieldList = append(metadataFieldList, "created_by", "name", "description", "permissions")

// NewRoles returns an instance of the Roles interface.
func NewRoles(dbClient *Client) Roles {
	return &roles{dbClient: dbClient}
}

func (r *roles) GetRoleByID(ctx context.Context, id string) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "db.GetRoleByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return r.getRole(ctx, goqu.Ex{"roles.id": id})
}

func (r *roles) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "db.GetRoleByName")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return r.getRole(ctx, goqu.Ex{"roles.name": name})
}

func (r *roles) GetRoles(ctx context.Context, input *GetRolesInput) (*RolesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetRoles")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.Ex{}

	if input.Filter != nil {
		if input.Filter.RoleIDs != nil {
			ex["roles.id"] = input.Filter.RoleIDs
		}
		if input.Filter.RoleNamePrefix != nil && *input.Filter.RoleNamePrefix != "" {
			ex["roles.name"] = goqu.Op{"like": *input.Filter.RoleNamePrefix + "%"}
		}
	}

	query := dialect.From(goqu.T("roles")).
		Select(rolesFieldList...).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "roles", Col: "id"},
		sortBy,
		sortDirection,
	)

	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, r.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Role{}
	for rows.Next() {
		item, err := scanRole(rows)
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

	result := RolesResult{
		PageInfo: rows.GetPageInfo(),
		Roles:    results,
	}

	return &result, nil
}

func (r *roles) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "db.CreateRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	permissions, err := r.marshalPermissions(role.GetPermissions())
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal permissions")
		return nil, err
	}

	sql, args, err := dialect.Insert("roles").
		Prepared(true).
		Rows(goqu.Record{
			"id":          newResourceID(),
			"version":     initialResourceVersion,
			"created_at":  timestamp,
			"updated_at":  timestamp,
			"created_by":  role.CreatedBy,
			"name":        role.Name,
			"description": role.Description,
			"permissions": permissions,
		}).
		Returning(rolesFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdRole, err := scanRole(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				tracing.RecordError(span, nil, "role with name %s already exists", role.Name)
				return nil, errors.New("role with name %s already exists", role.Name, errors.WithErrorCode(errors.EConflict))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdRole, nil
}

func (r *roles) UpdateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	permissions, err := r.marshalPermissions(role.GetPermissions())
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal permissions")
		return nil, err
	}

	sql, args, err := dialect.Update("roles").
		Prepared(true).
		Set(
			goqu.Record{
				"version":     goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":  timestamp,
				"description": role.Description,
				"permissions": permissions,
			},
		).Where(goqu.Ex{"id": role.Metadata.ID, "version": role.Metadata.Version}).Returning(rolesFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedRole, err := scanRole(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedRole, nil
}

func (r *roles) DeleteRole(ctx context.Context, role *models.Role) error {
	ctx, span := tracer.Start(ctx, "db.DeleteRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("roles").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      role.Metadata.ID,
				"version": role.Metadata.Version,
			},
		).Returning(rolesFieldList...).ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanRole(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (r *roles) marshalPermissions(input []permissions.Permission) ([]byte, error) {
	permissionsJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal role permissions to JSON: %w", err)
	}

	return permissionsJSON, nil
}

func (r *roles) getRole(ctx context.Context, exp exp.Ex) (*models.Role, error) {
	ctx, span := tracer.Start(ctx, "db.getRole")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("roles").
		Prepared(true).
		Select(rolesFieldList...).
		Where(exp).
		ToSQL()

	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	role, err := scanRole(r.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return role, nil
}

func scanRole(row scanner) (*models.Role, error) {
	r := &models.Role{}
	perms := []permissions.Permission{}

	fields := []interface{}{
		&r.Metadata.ID,
		&r.Metadata.CreationTimestamp,
		&r.Metadata.LastUpdatedTimestamp,
		&r.Metadata.Version,
		&r.CreatedBy,
		&r.Name,
		&r.Description,
		&perms,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	r.SetPermissions(perms)

	return r, nil
}
