package db

//go:generate go tool mockery --name Workspaces --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v4"
	"go.opentelemetry.io/otel/attribute"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Workspaces encapsulates the logic to access workspaces from the database
type Workspaces interface {
	GetWorkspaceByTRN(ctx context.Context, trn string) (*models.Workspace, error)
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*WorkspacesResult, error)
	UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	DeleteWorkspace(ctx context.Context, workspace *models.Workspace) error
	GetWorkspacesForManagedIdentity(ctx context.Context, managedIdentityID string) ([]models.Workspace, error)
	MigrateWorkspace(ctx context.Context, workspace *models.Workspace, newParentGroup *models.Group) (*models.Workspace, error)
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
	GroupID                        *string
	UserMemberID                   *string
	ServiceAccountMemberID         *string
	Search                         *string
	AssignedManagedIdentityID      *string
	WorkspaceIDs                   []string
	PathLessThan                   *string
	MinDurationSinceLastAssessment *time.Duration
	Locked                         *bool
	Dirty                          *bool
	HasStateVersion                *bool
	WorkspacePath                  *string
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
	"runner_tags",
	"drift_detection_enabled",
)

// NewWorkspaces returns an instance of the Workspaces interface
func NewWorkspaces(dbClient *Client) Workspaces {
	return &workspaces{dbClient: dbClient}
}

func (w *workspaces) GetWorkspaceByTRN(ctx context.Context, trn string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.GetWorkspaceByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.WorkspaceModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

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
			ex = ex.Append(goqu.I("namespaces.path").ILike("%" + *input.Filter.Search + "%"))
		}

		if input.Filter.WorkspacePath != nil {
			ex = ex.Append(goqu.I("namespaces.path").Eq(*input.Filter.WorkspacePath))
		}

		if input.Filter.PathLessThan != nil {
			ex = ex.Append(goqu.I("namespaces.path").Lt(*input.Filter.PathLessThan))
		}

		if input.Filter.Locked != nil {
			ex = ex.Append(goqu.I("workspaces.locked").Eq(*input.Filter.Locked))
		}

		if input.Filter.Dirty != nil {
			ex = ex.Append(goqu.I("workspaces.dirty_state").Eq(*input.Filter.Dirty))
		}

		if input.Filter.HasStateVersion != nil {
			if *input.Filter.HasStateVersion {
				ex = ex.Append(goqu.I("workspaces.current_state_version_id").IsNotNull())
			} else {
				ex = ex.Append(goqu.I("workspaces.current_state_version_id").IsNull())
			}
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

	if input.Filter != nil && input.Filter.MinDurationSinceLastAssessment != nil {
		query = query.LeftJoin(goqu.T("workspace_assessments"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("workspace_assessments.workspace_id")}))

		maxTime := time.Now().Add(-(*input.Filter.MinDurationSinceLastAssessment))

		ex = ex.Append(goqu.Or(
			goqu.I("workspace_assessments.id").IsNull(),
			goqu.I("workspace_assessments.completed_at").Lte(maxTime.UTC()),
		))
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

	var runnerTags []byte
	var err error
	if workspace.RunnerTags != nil {
		runnerTags, err = json.Marshal(workspace.RunnerTags)
		if err != nil {
			return nil, err
		}
	}

	timestamp := currentTime()

	sql, args, err := dialect.From("workspaces").
		Prepared(true).
		With("workspaces",
			dialect.Update("workspaces").
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
						"runner_tags":              runnerTags,
						"drift_detection_enabled":  workspace.EnableDriftDetection,
					},
				).Where(goqu.Ex{"id": workspace.Metadata.ID, "version": workspace.Metadata.Version}).
				Returning("*"),
		).Select(w.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedWorkspace, err := scanWorkspace(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedWorkspace, nil
}

func (w *workspaces) CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.CreateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	var runnerTags []byte
	var err error
	if workspace.RunnerTags != nil {
		runnerTags, err = json.Marshal(workspace.RunnerTags)
		if err != nil {
			return nil, err
		}
	}

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
			w.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateWorkspace: %v", txErr)
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
			"runner_tags":              runnerTags,
			"drift_detection_enabled":  workspace.EnableDriftDetection,
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
	createdWorkspace.Metadata.TRN = types.WorkspaceModelType.BuildTRN(fullPath)

	return createdWorkspace, nil
}

func (w *workspaces) DeleteWorkspace(ctx context.Context, workspace *models.Workspace) error {
	ctx, span := tracer.Start(ctx, "db.DeleteWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("workspaces").
		Prepared(true).
		With("workspaces",
			dialect.Delete("workspaces").
				Where(
					goqu.Ex{
						"id":      workspace.Metadata.ID,
						"version": workspace.Metadata.Version,
					},
				).Returning("*"),
		).Select(w.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanWorkspace(w.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true); err != nil {
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

// MigrateWorkspace migrates a workspace.
func (w *workspaces) MigrateWorkspace(ctx context.Context, workspace *models.Workspace, newParentGroup *models.Group) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "db.MigrateWorkspace")
	defer span.End()

	newPath := newParentGroup.FullPath + "/" + workspace.Name
	newParentID := newParentGroup.Metadata.ID

	tx, err := w.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			w.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for MigrateWorkspace: %v", txErr)
		}
	}()

	timestamp := currentTime()

	// Substitute the affected paths in the namespaces table first so that the FullPath field below will be set correctly.
	if err = migrateNamespaces(ctx, tx, workspace.FullPath, newPath); err != nil {
		return nil, errors.Wrap(err, "failed to migrate namespaces", errors.WithSpan(span))
	}

	// Update the group_id field in the workspace being migrated.
	sql, args, err := dialect.From("workspaces").
		Prepared(true).
		With("workspaces",
			dialect.Update("workspaces").
				Set(
					goqu.Record{
						"version":    goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at": timestamp,
						"group_id":   newParentID,
					},
				).Where(goqu.Ex{"id": workspace.Metadata.ID, "version": workspace.Metadata.Version}).
				Returning("*"),
		).Select(w.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"workspaces.id": goqu.I("namespaces.workspace_id")})).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL to update the migrating workspace's group ID", errors.WithSpan(span))
	}

	migratedWorkspace, err := scanWorkspace(tx.QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query to update the migrating workspace's group ID", errors.WithSpan(span))
	}

	// Delete managed identity assignments to the workspace being migrated
	// if the home group path of the managed identity is no longer a direct ancestor of the workspace.
	sql, args, err = dialect.Delete("workspace_managed_identity_relation").
		Prepared(true).
		Where(goqu.And(
			goqu.I("workspace_managed_identity_relation.workspace_id").Eq(migratedWorkspace.Metadata.ID),
			goqu.I("workspace_managed_identity_relation.managed_identity_id").In(
				dialect.From(goqu.T("managed_identities")).
					InnerJoin(goqu.T("groups"),
						goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("groups.id")})).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.group_id": goqu.I("groups.id")})).
					Select("managed_identities.id").
					Where(
						// Managed identity's home group path is no longer a direct ancestor of the workspace.
						goqu.I("namespaces.path").NotIn(migratedWorkspace.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL to delete managed identity assignments", errors.WithSpan(span))
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return nil, errors.Wrap(err, "failed to execute query to delete managed identity assignments", errors.WithSpan(span))
	}

	// Delete namespace memberships of service accounts
	// where the namespace (workspace) is being migrated
	// and the home group path of the service account is no longer a direct ancestor of the namespace.
	sql, args, err = dialect.Delete("namespace_memberships").
		Prepared(true).
		Where(goqu.And(
			goqu.I("namespace_memberships.id").In(
				dialect.From(goqu.T("namespace_memberships")).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespace_memberships.namespace_id": goqu.I("namespaces.id")})).
					Select("namespace_memberships.id").
					Where(
						// Namespace membership is to the workspace being migrated.
						goqu.I("namespaces.workspace_id").Eq(migratedWorkspace.Metadata.ID),
					)),
			goqu.I("namespace_memberships.service_account_id").In(
				dialect.From(goqu.T("service_accounts")).
					InnerJoin(goqu.T("groups"),
						goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("groups.id")})).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.group_id": goqu.I("groups.id")})).
					Select("service_accounts.id").
					Where(
						// Home group of the service account is no longer a direct ancestor of the namespace.
						goqu.I("namespaces.path").NotIn(migratedWorkspace.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL to delete service account namespace memberships", errors.WithSpan(span))
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return nil, errors.Wrap(err, "failed to execute query to delete service account namespace memberships", errors.WithSpan(span))
	}

	// Delete workspace VCS provider links to workspaces
	// where the workspace is being migrated
	// and the home group path of the VCS provider link is no longer a direct ancestor of the workspace.
	sql, args, err = dialect.Delete("workspace_vcs_provider_links").
		Prepared(true).
		Where(goqu.And(
			goqu.I("workspace_vcs_provider_links.workspace_id").Eq(migratedWorkspace.Metadata.ID),
			goqu.I("workspace_vcs_provider_links.provider_id").In(
				dialect.From(goqu.T("vcs_providers")).
					InnerJoin(goqu.T("groups"),
						goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("groups.id")})).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.group_id": goqu.I("groups.id")})).
					Select("vcs_providers.id").
					Where(
						// Home group of the provider is no longer a direct ancestor of the namespace.
						goqu.I("namespaces.path").NotIn(migratedWorkspace.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL to delete workspace VCS provider links", errors.WithSpan(span))
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return nil, errors.Wrap(err, "failed to execute query to delete workspace VCS provider links", errors.WithSpan(span))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to commit workspace migration transaction", errors.WithSpan(span))
	}

	return migratedWorkspace, nil
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

		if pgErr := asPgError(err); pgErr != nil {
			if isInvalidIDViolation(pgErr) {
				return nil, ErrInvalidID
			}
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
		&ws.RunnerTags,
		&ws.EnableDriftDetection,
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

	if withFullPath {
		ws.Metadata.TRN = types.WorkspaceModelType.BuildTRN(ws.FullPath)
	}

	return ws, nil
}
