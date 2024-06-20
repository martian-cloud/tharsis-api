package db

//go:generate mockery --name Workspaces --inpackage --case underscore

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

// Workspaces encapsulates the logic to access workspaces from the database
type Workspaces interface {
	GetWorkspaceByFullPath(ctx context.Context, path string) (*models.Workspace, error)
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*WorkspacesResult, error)
	UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	DeleteWorkspace(ctx context.Context, workspace *models.Workspace) error
	GetWorkspacesForManagedIdentity(ctx context.Context, managedIdentityID string) ([]models.Workspace, error)
}

// WorkspaceSortableField represents the fields that a workspace can be sorted by
type WorkspaceSortableField string

// WorkspaceSortableField constants
const (
	WorkspaceSortableFieldFullPathAsc   WorkspaceSortableField = "FULL_PATH_ASC"
	WorkspaceSortableFieldFullPathDesc  WorkspaceSortableField = "FULL_PATH_DESC"
	WorkspaceSortableFieldUpdatedAtAsc  WorkspaceSortableField = "UPDATED_AT_ASC"
	WorkspaceSortableFieldUpdatedAtDesc WorkspaceSortableField = "UPDATED_AT_DESC"
)

func (gs WorkspaceSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch gs {
	case WorkspaceSortableFieldFullPathAsc, WorkspaceSortableFieldFullPathDesc:
		return &pagination.FieldDescriptor{Key: "full_path", Table: "namespaces", Col: "path"}
	case WorkspaceSortableFieldUpdatedAtAsc, WorkspaceSortableFieldUpdatedAtDesc:
		return &pagination.FieldDescriptor{Key: "updated_at", Table: "workspaces", Col: "updated_at"}
	default:
		return nil
	}
}

func (gs WorkspaceSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(gs), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

// WorkspaceFilter contains the supported fields for filtering Workspace resources
type WorkspaceFilter struct {
	GroupID                   *string
	UserMemberID              *string
	ServiceAccountMemberID    *string
	Search                    *string
	AssignedManagedIdentityID *string
	WorkspaceIDs              []string
}

// GetWorkspacesInput is the input for listing workspaces
type GetWorkspacesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *WorkspaceSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *WorkspaceFilter
}

// WorkspacesResult contains the response data and page information
type WorkspacesResult struct {
	PageInfo   *pagination.PageInfo
	Workspaces []models.Workspace
}

type workspaces struct {
	dbClient *Client
}

var workspaceFieldList = append(
	metadataFieldList,
	"name",
	"group_id",
	"description",
	"current_job_id",
	"current_state_version_id",
	"dirty_state",
	"locked",
	"max_job_duration",
	"created_by",
	"terraform_version",
	"prevent_destroy_plan",
)

// NewWorkspaces returns an instance of the Workspaces interface
func NewWorkspaces(dbClient *Client) Workspaces {
	return &workspaces{dbClient: dbClient}
}

func (w *workspaces) GetWorkspaceByFullPath(ctx context.Context, path string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceByFullPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return w.getWorkspace(ctx, goqu.Ex{"namespaces.path": path})
}

func (w *workspaces) GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return w.getWorkspace(ctx, goqu.Ex{"workspaces.id": id})
}

func (w *workspaces) GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*WorkspacesResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaces")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.WorkspaceIDs != nil {
			// This check avoids an SQL syntax error if an empty slice is provided.
			if len(input.Filter.WorkspaceIDs) > 0 {
				ex = ex.Append(goqu.I("workspaces.id").In(input.Filter.WorkspaceIDs))
			}
		}

		if input.Filter.GroupID != nil {
			ex = ex.Append(goqu.I("workspaces.group_id").Eq(*input.Filter.GroupID))
		}

		if input.Filter.UserMemberID != nil {
			ex = ex.Append(namespaceMembershipFilterQuery("namespace_memberships.user_id", *input.Filter.UserMemberID))
		}

		if input.Filter.ServiceAccountMemberID != nil {
			ex = ex.Append(namespaceMembershipFilterQuery("namespace_memberships.service_account_id", *input.Filter.ServiceAccountMemberID))
		}

		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(goqu.I("namespaces.path").Like("%" + *input.Filter.Search + "%"))
		}
	}

	query := dialect.From(goqu.T("workspaces")).
		Select(w.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")}))

	// Since managed identities is a many to many relationship only join them when we are looking for exactly one.
	// Otherwise duplicates will result.
	if input.Filter != nil && input.Filter.AssignedManagedIdentityID != nil {
		query = query.InnerJoin(goqu.T("workspace_managed_identity_relation"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("workspace_managed_identity_relation.workspace_id")}))

		ex = ex.Append(goqu.Ex{"workspace_managed_identity_relation.managed_identity_id": input.Filter.AssignedManagedIdentityID})
	}

	query = query.Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "workspaces", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, w.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Workspace{}
	for rows.Next() {
		item, err := scanWorkspace(rows, true)
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

	result := WorkspacesResult{
		PageInfo:   rows.GetPageInfo(),
		Workspaces: results,
	}

	return &result, nil
}

func (w *workspaces) UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("workspaces").
		Prepared(true).
		Set(
			goqu.Record{
				"version":                  goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":               timestamp,
				"description":              nullableString(workspace.Description),
				"current_job_id":           nullableString(workspace.CurrentJobID),
				"current_state_version_id": nullableString(workspace.CurrentStateVersionID),
				"dirty_state":              workspace.DirtyState,
				"locked":                   workspace.Locked,
				"max_job_duration":         workspace.MaxJobDuration,
				"terraform_version":        workspace.TerraformVersion,
				"prevent_destroy_plan":     workspace.PreventDestroyPlan,
			},
		).Where(goqu.Ex{"id": workspace.Metadata.ID, "version": workspace.Metadata.Version}).Returning(workspaceFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedWorkspace, err := scanWorkspace(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	namespace, err := getNamespaceByWorkspaceID(ctx, w.dbClient.getConnection(ctx), updatedWorkspace.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace by workspace ID")
		return nil, err
	}

	updatedWorkspace.FullPath = namespace.path

	return updatedWorkspace, nil
}

func (w *workspaces) CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.CreateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Use transaction to update workspaces and namespaces tables
	tx, err := w.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			w.dbClient.logger.Errorf("failed to rollback tx for CreateWorkspace: %v", txErr)
		}
	}()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("workspaces").
		Prepared(true).
		Rows(goqu.Record{
			"id":                       newResourceID(),
			"version":                  initialResourceVersion,
			"created_at":               timestamp,
			"updated_at":               timestamp,
			"name":                     workspace.Name,
			"group_id":                 workspace.GroupID,
			"description":              nullableString(workspace.Description),
			"current_job_id":           nullableString(workspace.CurrentJobID),
			"current_state_version_id": nullableString(workspace.CurrentStateVersionID),
			"dirty_state":              workspace.DirtyState,
			"locked":                   workspace.Locked,
			"max_job_duration":         workspace.MaxJobDuration,
			"created_by":               workspace.CreatedBy,
			"terraform_version":        workspace.TerraformVersion,
			"prevent_destroy_plan":     workspace.PreventDestroyPlan,
		}).
		Returning(workspaceFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdWorkspace, err := scanWorkspace(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_group_id" {
				tracing.RecordError(span, nil,
					"invalid group parent: the specified parent group does not exist")
				return nil, errors.New("invalid group parent: the specified parent group does not exist", errors.WithErrorCode(errors.EConflict))
			}

			if isInvalidIDViolation(pgErr) {
				tracing.RecordError(span, pgErr, "invalid ID")
				return nil, ErrInvalidID
			}
		}

		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	// Lookup namespace for parent group
	parentNamespace, err := getNamespaceByGroupID(ctx, tx, workspace.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get namespace by group ID")
		return nil, err
	}

	fullPath := fmt.Sprintf("%s/%s", parentNamespace.path, workspace.Name)

	// Create new namespace resource for workspace
	if _, err := createNamespace(ctx, tx, &namespaceRow{path: fullPath, workspaceID: createdWorkspace.Metadata.ID}); err != nil {
		tracing.RecordError(span, err, "failed to create namespace")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	createdWorkspace.FullPath = fullPath

	return createdWorkspace, nil
}

func (w *workspaces) DeleteWorkspace(ctx context.Context, workspace *models.Workspace) error {
	ctx, span := tracer.Start(ctx, "db.DeleteWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.Delete("workspaces").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      workspace.Metadata.ID,
				"version": workspace.Metadata.Version,
			},
		).Returning(workspaceFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanWorkspace(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), false); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (w *workspaces) GetWorkspacesForManagedIdentity(ctx context.Context, managedIdentityID string) ([]models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspacesForManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("workspaces").
		Prepared(true).
		Select(w.getSelectFields()...).
		InnerJoin(goqu.T("workspace_managed_identity_relation"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("workspace_managed_identity_relation.workspace_id")})).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")})).
		Where(goqu.Ex{"workspace_managed_identity_relation.managed_identity_id": managedIdentityID}).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	rows, err := w.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Workspace{}
	for rows.Next() {
		item, err := scanWorkspace(rows, true)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	return results, nil
}

func (w *workspaces) getWorkspace(ctx context.Context, exp goqu.Ex) (*models.Workspace, error) {
	query := dialect.From(goqu.T("workspaces")).
		Prepared(true).
		Select(w.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	workspace, err := scanWorkspace(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return workspace, nil
}

// TODO: Remove this function and use namespaceMembershipExpressionBuilder after DB integration tests have been merged
func namespaceMembershipFilterQuery(col string, id string) exp.Expression {
	// The base column ID comparison, to be ORed with a sub-query based on team member relationships.
	whereExOr := goqu.Or()
	whereExOr = whereExOr.Append(goqu.I(col).Eq(id))

	// If dealing with a user ID, must also check team member relationships.
	if strings.HasSuffix(col, ".user_id") {
		// This is a logical OR with the base column ID comparison.
		whereExOr = whereExOr.Append(
			goqu.I("namespace_memberships.team_id").In(
				dialect.From("team_members").
					Select("team_id").
					Where(goqu.I("team_members.user_id").Eq(id))))
	}

	return goqu.Or(
		goqu.I("namespaces.path").Like(goqu.Any(
			dialect.From("namespace_memberships").
				Select(goqu.L("path || '/%'")).
				InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
				Where(whereExOr, goqu.I("namespaces.workspace_id").IsNull()),
		)),
		goqu.I("namespaces.path").In(
			dialect.From("namespace_memberships").
				Select("path").
				InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
				Where(whereExOr, goqu.I("namespaces.group_id").IsNull()),
		),
	)
}

func (w *workspaces) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range workspaceFieldList {
		selectFields = append(selectFields, fmt.Sprintf("workspaces.%s", field))
	}

	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanWorkspace(row scanner, withFullPath bool) (*models.Workspace, error) {
	var description sql.NullString
	var currentJobID sql.NullString
	var currentStateVersionID sql.NullString

	ws := &models.Workspace{}

	fields := []interface{}{
		&ws.Metadata.ID,
		&ws.Metadata.CreationTimestamp,
		&ws.Metadata.LastUpdatedTimestamp,
		&ws.Metadata.Version,
		&ws.Name,
		&ws.GroupID,
		&description,
		&currentJobID,
		&currentStateVersionID,
		&ws.DirtyState,
		&ws.Locked,
		&ws.MaxJobDuration,
		&ws.CreatedBy,
		&ws.TerraformVersion,
		&ws.PreventDestroyPlan,
	}

	if withFullPath {
		fields = append(fields, &ws.FullPath)
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		ws.Description = description.String
	}

	if currentJobID.Valid {
		ws.CurrentJobID = currentJobID.String
	}

	if currentStateVersionID.Valid {
		ws.CurrentStateVersionID = currentStateVersionID.String
	}

	return ws, nil
}
