package db

//go:generate go tool mockery --name WorkspaceRoleBindings --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// WorkspaceRoleBindings encapsulates the logic to access workspace role bindings from the database
type WorkspaceRoleBindings interface {
	GetWorkspaceRoleBindingByID(ctx context.Context, id string) (*models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindingByTRN(ctx context.Context, trnValue string) (*models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindingByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindings(ctx context.Context, input *GetWorkspaceRoleBindingsInput) (*WorkspaceRoleBindingsResult, error)
	CreateWorkspaceRoleBinding(ctx context.Context, binding *models.WorkspaceRoleBinding) (*models.WorkspaceRoleBinding, error)
	UpdateWorkspaceRoleBinding(ctx context.Context, binding *models.WorkspaceRoleBinding) (*models.WorkspaceRoleBinding, error)
	DeleteWorkspaceRoleBinding(ctx context.Context, binding *models.WorkspaceRoleBinding) error
}

// WorkspaceRoleBindingSortableField represents the fields that a workspace role binding can be sorted by
type WorkspaceRoleBindingSortableField string

// WorkspaceRoleBindingSortableField constants
const (
	WorkspaceRoleBindingSortableFieldCreatedAtAsc  WorkspaceRoleBindingSortableField = "CREATED_AT_ASC"
	WorkspaceRoleBindingSortableFieldCreatedAtDesc WorkspaceRoleBindingSortableField = "CREATED_AT_DESC"
	WorkspaceRoleBindingSortableFieldUpdatedAtAsc  WorkspaceRoleBindingSortableField = "UPDATED_AT_ASC"
	WorkspaceRoleBindingSortableFieldUpdatedAtDesc WorkspaceRoleBindingSortableField = "UPDATED_AT_DESC"
)

func (sf WorkspaceRoleBindingSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch sf {
	case WorkspaceRoleBindingSortableFieldCreatedAtAsc, WorkspaceRoleBindingSortableFieldCreatedAtDesc:
		return &pagination.FieldDescriptor{Key: "created_at", Table: "workspace_role_bindings", Col: "created_at"}
	case WorkspaceRoleBindingSortableFieldUpdatedAtAsc, WorkspaceRoleBindingSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "workspace_role_bindings", Col: "updated_at"}
	default:
		return nil
	}
}

func (sf WorkspaceRoleBindingSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(sf), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// WorkspaceRoleBindingFilter contains the supported fields for filtering WorkspaceRoleBinding resources
type WorkspaceRoleBindingFilter struct {
	// IDs filters to bindings with any of these IDs. Used by the activity-event target loader, which
	// looks bindings up by their own ID rather than by workspace.
	IDs []string
	// WorkspaceIDs filters to bindings for any of these workspaces.
	WorkspaceIDs []string
	// RoleID filters to bindings that confer this role.
	RoleID *string
}

// GetWorkspaceRoleBindingsInput is the input for listing workspace role bindings
type GetWorkspaceRoleBindingsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *WorkspaceRoleBindingSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *WorkspaceRoleBindingFilter
}

// WorkspaceRoleBindingsResult contains the response data and page information
type WorkspaceRoleBindingsResult struct {
	PageInfo              *pagination.PageInfo
	WorkspaceRoleBindings []models.WorkspaceRoleBinding
}

var workspaceRoleBindingFieldList = append(metadataFieldList,
	"workspace_id",
	"role_id",
	"created_by",
)

type workspaceRoleBindings struct {
	dbClient *Client
}

// NewWorkspaceRoleBindings returns an instance of the WorkspaceRoleBindings interface
func NewWorkspaceRoleBindings(dbClient *Client) WorkspaceRoleBindings {
	return &workspaceRoleBindings{dbClient: dbClient}
}

func (w *workspaceRoleBindings) GetWorkspaceRoleBindingByID(ctx context.Context, id string) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceRoleBindingByID")
	defer span.End()

	return w.getWorkspaceRoleBinding(ctx, goqu.Ex{"workspace_role_bindings.id": id})
}

func (w *workspaceRoleBindings) GetWorkspaceRoleBindingByTRN(ctx context.Context, trnValue string) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceRoleBindingByTRN")
	defer span.End()

	parsed, err := trn.TypeWorkspaceRoleBinding.Parse(trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	// A binding is identified by its workspace, since a workspace has at most one.
	return w.getWorkspaceRoleBinding(ctx, goqu.Ex{"namespaces.path": parsed.Path()})
}

func (w *workspaceRoleBindings) GetWorkspaceRoleBindingByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceRoleBindingByWorkspaceID")
	defer span.End()

	return w.getWorkspaceRoleBinding(ctx, goqu.Ex{"workspace_role_bindings.workspace_id": workspaceID})
}

func (w *workspaceRoleBindings) GetWorkspaceRoleBindings(ctx context.Context, input *GetWorkspaceRoleBindingsInput) (*WorkspaceRoleBindingsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceRoleBindings")
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		// Guard against an empty slice, which would produce invalid SQL.
		if len(input.Filter.IDs) > 0 {
			ex = ex.Append(goqu.I("workspace_role_bindings.id").In(input.Filter.IDs))
		}

		if len(input.Filter.WorkspaceIDs) > 0 {
			ex = ex.Append(goqu.I("workspace_role_bindings.workspace_id").In(input.Filter.WorkspaceIDs))
		}

		if input.Filter.RoleID != nil {
			ex = ex.Append(goqu.I("workspace_role_bindings.role_id").Eq(*input.Filter.RoleID))
		}
	}

	query := dialect.From("workspace_role_bindings").
		Select(w.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspace_role_bindings.workspace_id": goqu.I("namespaces.workspace_id")})).
		Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "workspace_role_bindings", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithQueryTag("workspacerolebinding.GetWorkspaceRoleBindings"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build query", errors.WithSpan(span))
	}

	rows, err := qBuilder.Execute(ctx, w.dbClient.getConnection(ctx), query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	defer rows.Close()

	results := []models.WorkspaceRoleBinding{}
	for rows.Next() {
		item, err := scanWorkspaceRoleBinding(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}

		results = append(results, *item)
	}

	if err := rows.Finalize(&results); err != nil {
		return nil, errors.Wrap(err, "failed to finalize rows", errors.WithSpan(span))
	}

	return &WorkspaceRoleBindingsResult{
		PageInfo:              rows.GetPageInfo(),
		WorkspaceRoleBindings: results,
	}, nil
}

func (w *workspaceRoleBindings) CreateWorkspaceRoleBinding(ctx context.Context, binding *models.WorkspaceRoleBinding) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "db.CreateWorkspaceRoleBinding")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("workspacerolebinding.CreateWorkspaceRoleBinding",
		dialect.From("workspace_role_bindings").
			Prepared(true).
			With("workspace_role_bindings",
				dialect.Insert("workspace_role_bindings").
					Rows(goqu.Record{
						"id":           newResourceID(),
						"version":      initialResourceVersion,
						"created_at":   timestamp,
						"updated_at":   timestamp,
						"workspace_id": binding.WorkspaceID,
						"role_id":      binding.RoleID,
						"created_by":   binding.CreatedBy,
					}).
					Returning("*"),
			).Select(w.getSelectFields()...).
			InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspace_role_bindings.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdBinding, err := scanWorkspaceRoleBinding(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New(
					"this workspace already has a role binding",
					errors.WithErrorCode(errors.EConflict),
					errors.WithSpan(span),
				)
			}

			if isForeignKeyViolation(pgErr) {
				switch pgErr.ConstraintName {
				case "fk_workspace_role_bindings_workspace_id":
					return nil, errors.New("workspace does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				case "fk_workspace_role_bindings_role_id":
					return nil, errors.New("role does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
				}
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdBinding, nil
}

func (w *workspaceRoleBindings) UpdateWorkspaceRoleBinding(ctx context.Context, binding *models.WorkspaceRoleBinding) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateWorkspaceRoleBinding")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("workspacerolebinding.UpdateWorkspaceRoleBinding",
		dialect.From("workspace_role_bindings").
			Prepared(true).
			With("workspace_role_bindings",
				dialect.Update("workspace_role_bindings").
					Set(goqu.Record{
						"version":    goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at": timestamp,
						"role_id":    binding.RoleID,
					}).
					Where(goqu.Ex{"id": binding.Metadata.ID, "version": binding.Metadata.Version}).
					Returning("*"),
			).Select(w.getSelectFields()...).
			InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspace_role_bindings.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedBinding, err := scanWorkspaceRoleBinding(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}

		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) {
				return nil, errors.New("role does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
		}

		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedBinding, nil
}

func (w *workspaceRoleBindings) DeleteWorkspaceRoleBinding(ctx context.Context, binding *models.WorkspaceRoleBinding) error {
	ctx, span := tracer.Start(ctx, "db.DeleteWorkspaceRoleBinding")
	defer span.End()

	sql, args, err := toSQLWithTag("workspacerolebinding.DeleteWorkspaceRoleBinding",
		dialect.From("workspace_role_bindings").
			Prepared(true).
			With("workspace_role_bindings",
				dialect.Delete("workspace_role_bindings").
					Where(goqu.Ex{"id": binding.Metadata.ID, "version": binding.Metadata.Version}).
					Returning("*"),
			).Select(w.getSelectFields()...).
			InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspace_role_bindings.workspace_id": goqu.I("namespaces.workspace_id")})))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = scanWorkspaceRoleBinding(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			return ErrOptimisticLockError
		}

		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (w *workspaceRoleBindings) getWorkspaceRoleBinding(ctx context.Context, ex goqu.Ex) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "db.getWorkspaceRoleBinding")
	defer span.End()

	sql, args, err := toSQLWithTag("workspacerolebinding.getWorkspaceRoleBinding",
		dialect.From("workspace_role_bindings").
			Prepared(true).
			InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspace_role_bindings.workspace_id": goqu.I("namespaces.workspace_id")})).
			Select(w.getSelectFields()...).
			Where(ex))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	binding, err := scanWorkspaceRoleBinding(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
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

	return binding, nil
}

func (w *workspaceRoleBindings) getSelectFields() []interface{} {
	selectFields := []any{}

	for _, field := range workspaceRoleBindingFieldList {
		selectFields = append(selectFields, fmt.Sprintf("workspace_role_bindings.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanWorkspaceRoleBinding(row scanner) (*models.WorkspaceRoleBinding, error) {
	var workspacePath string

	binding := &models.WorkspaceRoleBinding{}

	err := row.Scan(
		&binding.Metadata.ID,
		&binding.Metadata.CreationTimestamp,
		&binding.Metadata.LastUpdatedTimestamp,
		&binding.Metadata.Version,
		&binding.WorkspaceID,
		&binding.RoleID,
		&binding.CreatedBy,
		&workspacePath,
	)
	if err != nil {
		return nil, err
	}

	binding.Metadata.TRN = trn.TypeWorkspaceRoleBinding.Build(workspacePath)

	return binding, nil
}
