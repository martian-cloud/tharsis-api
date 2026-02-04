package db

//go:generate go tool mockery --name Groups --inpackage --case underscore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Groups encapsulates the logic to access groups from the database
type Groups interface {
	// GetGroupByID returns a group by ID
	GetGroupByID(ctx context.Context, id string) (*models.Group, error)
	// GetGroupByTRN returns a group by trn
	GetGroupByTRN(ctx context.Context, trn string) (*models.Group, error)
	// DeleteGroup deletes a group
	DeleteGroup(ctx context.Context, group *models.Group) error
	// GetGroups returns a list of groups
	GetGroups(ctx context.Context, input *GetGroupsInput) (*GroupsResult, error)
	// CreateGroup creates a new group
	CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
	// UpdateGroup updates an existing group
	UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
	// GetChildDepth returns the depth of tree containing this group and its descendants.
	GetChildDepth(ctx context.Context, group *models.Group) (int, error)
	// MigrateGroup re-parents an existing group
	MigrateGroup(ctx context.Context, group, newParentGroup *models.Group) (*models.Group, error)
}

// GroupFilter contains the supported fields for filtering Group resources
type GroupFilter struct {
	ParentID               *string
	UserMemberID           *string
	ServiceAccountMemberID *string
	Search                 *string
	GroupIDs               []string
	NamespaceIDs           []string
	RootOnly               bool
	GroupPaths             []string
	FavoriteUserID         *string
}

// GroupSortableField represents the fields that a group can be sorted by
type GroupSortableField string

// GroupSortableField constants
const (
	GroupSortableFieldFullPathAsc    GroupSortableField = "FULL_PATH_ASC"
	GroupSortableFieldFullPathDesc   GroupSortableField = "FULL_PATH_DESC"
	GroupSortableFieldGroupLevelAsc  GroupSortableField = "GROUP_LEVEL_ASC"
	GroupSortableFieldGroupLevelDesc GroupSortableField = "GROUP_LEVEL_DESC"
)

func (gs GroupSortableField) getFieldDescriptor() *pagination.FieldDescriptor {
	switch gs {
	case GroupSortableFieldFullPathAsc, GroupSortableFieldFullPathDesc:
		return &pagination.FieldDescriptor{Key: "full_path", Table: "namespaces", Col: "path"}
	case GroupSortableFieldGroupLevelAsc, GroupSortableFieldGroupLevelDesc:
		return &pagination.FieldDescriptor{Key: "full_path", Table: "namespaces", Col: "path"}
	default:
		return nil
	}
}

func (gs GroupSortableField) getSortDirection() pagination.SortDirection {
	if strings.HasSuffix(string(gs), "_DESC") {
		return pagination.DescSort
	}
	return pagination.AscSort
}

func (gs GroupSortableField) getTransformFunc() pagination.SortTransformFunc {
	switch gs {
	case GroupSortableFieldGroupLevelAsc, GroupSortableFieldGroupLevelDesc:
		return func(s string) string {
			return fmt.Sprintf("array_length(string_to_array(%s, '/'), 1)", s)
		}
	default:
		return nil
	}
}

// GetGroupsInput is the input for listing groups
type GetGroupsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *GroupSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Filter is used to filter the results
	Filter *GroupFilter
}

// GroupsResult contains the response data and page information
type GroupsResult struct {
	PageInfo *pagination.PageInfo
	Groups   []models.Group
}

var groupFieldList = append(metadataFieldList, "name", "description", "parent_id", "created_by", "runner_tags", "drift_detection_enabled", "provider_mirror_enabled")

type groups struct {
	dbClient *Client
}

// NewGroups returns an instance of the Groups interface
func NewGroups(dbClient *Client) Groups {
	return &groups{dbClient: dbClient}
}

func (g *groups) GetGroupByID(ctx context.Context, id string) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "db.GetGroupByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	return g.getGroup(ctx, goqu.Ex{"groups.id": id})
}

func (g *groups) GetGroupByTRN(ctx context.Context, trn string) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "db.GetGroupByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	path, err := types.GroupModelType.ResourcePathFromTRN(trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse TRN", errors.WithSpan(span))
	}

	return g.getGroup(ctx, goqu.Ex{"namespaces.path": path})
}

func (g *groups) GetGroups(ctx context.Context, input *GetGroupsInput) (*GroupsResult, error) {
	ctx, span := tracer.Start(ctx, "db.GetGroups")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	ex := goqu.And()

	if input.Filter != nil {
		if input.Filter.RootOnly {
			ex = ex.Append(goqu.I("groups.parent_id").Eq(nil))
		}

		if len(input.Filter.GroupIDs) > 0 {
			ex = ex.Append(goqu.I("groups.id").In(input.Filter.GroupIDs))
		}

		if input.Filter.ParentID != nil {
			ex = ex.Append(goqu.I("groups.parent_id").Eq(*input.Filter.ParentID))
		}

		if input.Filter.NamespaceIDs != nil {
			if len(input.Filter.NamespaceIDs) == 0 {
				return &GroupsResult{
					PageInfo: &pagination.PageInfo{},
					Groups:   []models.Group{},
				}, nil
			}

			ex = ex.Append(goqu.I("namespaces.id").In(input.Filter.NamespaceIDs))
		}

		if input.Filter.UserMemberID != nil {
			ex = ex.Append(
				namespaceMembershipExpressionBuilder{
					userID: input.Filter.UserMemberID,
				}.build(),
			)
		}

		if input.Filter.ServiceAccountMemberID != nil {
			ex = ex.Append(
				namespaceMembershipExpressionBuilder{
					serviceAccountID: input.Filter.ServiceAccountMemberID,
				}.build(),
			)
		}

		if input.Filter.Search != nil && *input.Filter.Search != "" {
			ex = ex.Append(goqu.I("namespaces.path").ILike("%" + *input.Filter.Search + "%"))
		}

		if len(input.Filter.GroupPaths) > 0 {
			ex = ex.Append(goqu.I("namespaces.path").In(input.Filter.GroupPaths))
		}

	}

	query := dialect.From(goqu.T("groups")).
		Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")}))

	if input.Filter != nil && input.Filter.FavoriteUserID != nil {
		query = query.InnerJoin(goqu.T("namespace_favorites"), goqu.On(goqu.And(
			goqu.Ex{"namespace_favorites.group_id": goqu.I("groups.id")},
			goqu.Ex{"namespace_favorites.user_id": *input.Filter.FavoriteUserID},
		)))
	}

	query = query.Where(ex)

	sortDirection := pagination.AscSort

	var sortBy *pagination.FieldDescriptor
	var sortTransformFunc pagination.SortTransformFunc
	if input.Sort != nil {
		sortDirection = input.Sort.getSortDirection()
		sortBy = input.Sort.getFieldDescriptor()
		sortTransformFunc = input.Sort.getTransformFunc()
	}

	qBuilder, err := pagination.NewPaginatedQueryBuilder(
		input.PaginationOptions,
		&pagination.FieldDescriptor{Key: "id", Table: "groups", Col: "id"},
		pagination.WithSortByField(sortBy, sortDirection),
		pagination.WithSortByTransform(sortTransformFunc),
	)
	if err != nil {
		tracing.RecordError(span, err, "failed to build query")
		return nil, err
	}

	rows, err := qBuilder.Execute(ctx, g.dbClient.getConnection(ctx), query)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.Group{}
	for rows.Next() {
		item, err := scanGroup(rows, true)
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

	result := GroupsResult{
		PageInfo: rows.GetPageInfo(),
		Groups:   results,
	}

	return &result, nil
}

func (g *groups) CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "db.CreateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	var runnerTags []byte
	if group.RunnerTags != nil {
		decoded, err := json.Marshal(group.RunnerTags)
		if err != nil {
			return nil, err
		}
		runnerTags = decoded
	}

	// Use transaction to update groups and namespaces tables
	tx, err := g.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			g.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateGroup: %v", txErr)
		}
	}()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("groups").
		Prepared(true).
		Rows(goqu.Record{
			"id":                      newResourceID(),
			"version":                 initialResourceVersion,
			"created_at":              timestamp,
			"updated_at":              timestamp,
			"name":                    group.Name,
			"description":             nullableString(group.Description),
			"parent_id":               nullableString(group.ParentID),
			"created_by":              group.CreatedBy,
			"runner_tags":             runnerTags,
			"drift_detection_enabled": group.EnableDriftDetection,
			"provider_mirror_enabled": group.EnableProviderMirror,
		}).
		Returning(groupFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdGroup, err := scanGroup(tx.QueryRow(ctx, sql, args...), false)
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_parent_id" {
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

	fullPath := group.Name

	// Lookup namespace for parent group if this is a nested group
	if group.ParentID != "" {
		parentNamespace, err := getNamespaceByGroupID(ctx, tx, group.ParentID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get namespace by group ID")
			return nil, err
		}

		fullPath = fmt.Sprintf("%s/%s", parentNamespace.path, fullPath)
	}

	// Create new namespace resource for group
	if _, err := createNamespace(ctx, tx, &namespaceRow{path: fullPath, groupID: createdGroup.Metadata.ID}); err != nil {
		tracing.RecordError(span, err, "failed to create namespace for group")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	createdGroup.FullPath = fullPath
	createdGroup.Metadata.TRN = types.GroupModelType.BuildTRN(fullPath)

	return createdGroup, nil
}

func (g *groups) UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	var runnerTags []byte
	var err error
	if group.RunnerTags != nil {
		runnerTags, err = json.Marshal(group.RunnerTags)
		if err != nil {
			return nil, err
		}
	}

	timestamp := currentTime()

	sql, args, err := dialect.From("groups").
		Prepared(true).
		With("groups",
			dialect.Update("groups").
				Set(
					goqu.Record{
						"version":                 goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at":              timestamp,
						"description":             nullableString(group.Description),
						"runner_tags":             runnerTags,
						"drift_detection_enabled": group.EnableDriftDetection,
						"provider_mirror_enabled": group.EnableProviderMirror,
					},
				).Where(goqu.Ex{"id": group.Metadata.ID, "version": group.Metadata.Version}).
				Returning("*"),
		).Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedGroup, err := scanGroup(g.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedGroup, nil
}

// GetChildDepth returns the depth of the descendant tree, EXCLUDING this group.
func (g *groups) GetChildDepth(ctx context.Context, group *models.Group) (int, error) {
	ctx, span := tracer.Start(ctx, "db.GetChildDepth")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	conn := g.dbClient.getConnection(ctx)
	// Apparently does not need a deferred close.

	depth, err := g.getChildDepth(ctx, conn, span, group.Metadata.ID)
	// any error has already been recorded to the tracing span
	if err != nil {
		return -1, err
	}

	return depth, nil
}

// getChildDepth is self-recursive and returns the depth of the descendant tree, EXCLUDING this group.
func (g *groups) getChildDepth(ctx context.Context, conn connection, span trace.Span, id string) (int, error) {
	// Scan rows
	resp, err := g.GetGroups(ctx, &GetGroupsInput{Filter: &GroupFilter{ParentID: &id}})
	if err != nil {
		return 0, err
	}

	if resp.PageInfo.TotalCount == 0 {
		return 0, nil
	}

	maxChildDepth := 0
	for _, child := range resp.Groups {
		candidate, err := g.getChildDepth(ctx, conn, span, child.Metadata.ID)
		if err != nil {
			tracing.RecordError(span, err, "failed to recurse")
			return 0, err
		}

		if candidate > maxChildDepth {
			maxChildDepth = candidate
		}
	}

	return maxChildDepth + 1, nil
}

// MigrateGroup migrates a group.  If moving group to become a root group, newParentGroup must be set to nil.
func (g *groups) MigrateGroup(ctx context.Context, group, newParentGroup *models.Group) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "db.MigrateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	var newPath, newParentID string
	if newParentGroup == nil {
		// Moving to root group.
		newPath = group.Name
	} else {
		newPath = newParentGroup.FullPath + "/" + group.Name
		newParentID = newParentGroup.Metadata.ID
	}

	tx, err := g.dbClient.getConnection(ctx).Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	// Rollback is safe to call even if the tx is already closed, so if
	// the tx commits successfully, this is a no-op
	defer func() {
		if txErr := tx.Rollback(ctx); txErr != nil && txErr != pgx.ErrTxClosed {
			g.dbClient.logger.WithContextFields(ctx).Errorf("failed to rollback tx for MigrateGroup: %v", txErr)
		}
	}()

	timestamp := currentTime()

	// Substitute the affected paths in the namespaces table first so that the FullPath field below will be set correctly.
	if err = migrateNamespaces(ctx, tx, group.FullPath, newPath); err != nil {
		tracing.RecordError(span, err, "failed to migrate namespaces")
		return nil, fmt.Errorf("failed to migrate namespaces: %v", err)
	}

	// Update the parent_id field in the group being migrated.
	sql, args, err := dialect.From("groups").
		Prepared(true).
		With("groups",
			dialect.Update("groups").
				Set(
					goqu.Record{
						"version":    goqu.L("? + ?", goqu.C("version"), 1),
						"updated_at": timestamp,
						"parent_id":  nullableString(newParentID),
					},
				).Where(goqu.Ex{"id": group.Metadata.ID, "version": group.Metadata.Version}).
				Returning("*"),
		).Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to generate SQL to update the migrating group's parent ID")
		return nil, fmt.Errorf("failed to generate SQL to update the migrating group's parent ID: %v", err)
	}

	migratedGroup, err := scanGroup(tx.QueryRow(ctx, sql, args...), true)
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err,
			"failed to execute query to update the migrating group's parent ID")
		return nil, fmt.Errorf("failed to execute query to update the migrating group's parent ID: %v", err)
	}

	// Delete managed identity assignments to a workspace
	// where the workspace is in the tree being migrated
	// and the home group path of the managed identity is no longer a direct ancestor of the workspace.
	sql, args, err = dialect.Delete("workspace_managed_identity_relation").
		Prepared(true).
		Where(goqu.And(
			goqu.I("workspace_managed_identity_relation.workspace_id").In(
				dialect.From(goqu.T("workspaces")).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.workspace_id": goqu.I("workspaces.id")})).
					Select("workspaces.id").
					Where(
						// Workspace is underneath the new path of the group being migrated.
						// No equals check needed, because a workspace is never at the same path as a group.
						goqu.I("namespaces.path").Like(newPath+"/%"),
					)),
			goqu.I("workspace_managed_identity_relation.managed_identity_id").In(
				dialect.From(goqu.T("managed_identities")).
					InnerJoin(goqu.T("groups"),
						goqu.On(goqu.Ex{"managed_identities.group_id": goqu.I("groups.id")})).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.group_id": goqu.I("groups.id")})).
					Select("managed_identities.id").
					Where(
						// Managed identity's home group path is no longer a direct ancestor of the workspace.
						goqu.I("namespaces.path").NotIn(migratedGroup.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to generate SQL to delete managed identity assignments")
		return nil, fmt.Errorf("failed to generate SQL to delete managed identity assignments: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to delete managed identity assignments")
		return nil, fmt.Errorf("failed to execute query to delete managed identity assignments: %v", err)
	}

	// Delete service accounts assigned to runners
	// where the runner is in the tree being migrated
	// and the group path of the service account is no longer a direct ancestor of the group.
	sql, args, err = dialect.Delete("service_account_runner_relation").
		Prepared(true).
		Where(goqu.And(
			goqu.I("service_account_runner_relation.runner_id").
				In(
					dialect.From(goqu.T("runners")).
						InnerJoin(goqu.T("namespaces"),
							goqu.On(goqu.Ex{"runners.group_id": goqu.I("namespaces.group_id")})).
						Select("runners.id").
						Where(
							// Runner is underneath the new path of the group being migrated.
							goqu.Or(
								goqu.I("namespaces.path").Eq(newPath),
								goqu.I("namespaces.path").Like(newPath+"/%"),
							),
						)),
			goqu.I("service_account_runner_relation.service_account_id").In(
				dialect.From(goqu.T("service_accounts")).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"service_accounts.group_id": goqu.I("namespaces.group_id")})).
					Select("service_accounts.id").
					Where(
						// Service account's group path is no longer a direct ancestor of the runner's group.
						goqu.I("namespaces.path").NotIn(migratedGroup.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to generate SQL to delete runner service account assignments")
		return nil, fmt.Errorf("failed to generate SQL to delete runner service account assignments: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to delete runner service account assignments")
		return nil, fmt.Errorf("failed to execute query to delete runner service account assignments: %v", err)
	}

	// Delete namespace memberships of service accounts
	// where the namespace (group or workspace) is in the tree being migrated
	// and the home group path of the service account is no longer a direct ancestor of the namespace.
	sql, args, err = dialect.Delete("namespace_memberships").
		Prepared(true).
		Where(goqu.And(
			goqu.I("namespace_memberships.namespace_id").In(
				dialect.From(goqu.T("namespaces")).
					Select("id").
					Where(
						// Namespace (group or workspace) is in the tree being migrated.
						goqu.Or(
							goqu.I("path").Eq(newPath),
							goqu.I("path").Like(newPath+"/%"),
						),
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
						goqu.I("namespaces.path").NotIn(migratedGroup.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to generate SQL to delete service account namespace memberships")
		return nil, fmt.Errorf("failed to generate SQL to delete service account namespace memberships: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to delete service account namespace memberships")
		return nil, fmt.Errorf("failed to execute query to delete service account namespace memberships: %v", err)
	}

	// Delete workspace VCS provider links to workspaces
	// where the workspace is in the tree being migrated
	// and the home group path of the VCS provider link is no longer a direct ancestor of the workspace.
	sql, args, err = dialect.Delete("workspace_vcs_provider_links").
		Prepared(true).
		Where(goqu.And(
			goqu.I("workspace_vcs_provider_links.workspace_id").In(
				dialect.From(goqu.T("workspaces")).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.workspace_id": goqu.I("workspaces.id")})).
					Select("workspaces.id").
					Where(
						// Workspace is underneath the new path of the group being migrated.
						// No equals check needed, because a workspace is never at the same path as a group.
						goqu.I("namespaces.path").Like(newPath+"/%"),
					)),
			goqu.I("workspace_vcs_provider_links.provider_id").In(
				dialect.From(goqu.T("vcs_providers")).
					InnerJoin(goqu.T("groups"),
						goqu.On(goqu.Ex{"vcs_providers.group_id": goqu.I("groups.id")})).
					InnerJoin(goqu.T("namespaces"),
						goqu.On(goqu.Ex{"namespaces.group_id": goqu.I("groups.id")})).
					Select("vcs_providers.id").
					Where(
						// Home group of the provider is no longer a direct ancestor of the namespace.
						goqu.I("namespaces.path").NotIn(migratedGroup.ExpandPath()),
					)),
		)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to generate SQL to delete workspace VCS provider links")
		return nil, fmt.Errorf("failed to generate SQL to delete workspace VCS provider links: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to delete workspace VCS provider links")
		return nil, fmt.Errorf("failed to execute query to delete workspace VCS provider links: %v", err)
	}

	// Find the new root group ID.
	newRootGroupRow, err := getNamespaceByPath(ctx, tx, migratedGroup.GetRootGroupPath())
	if err != nil {
		tracing.RecordError(span, err, "failed to get new root group")
		return nil, fmt.Errorf("failed to get new root group: %v", err)
	}
	if newRootGroupRow == nil {
		tracing.RecordError(span, nil, "failed to get new root group")
		return nil, fmt.Errorf("failed to get new root group")
	}
	newRootGroupID := newRootGroupRow.groupID

	// For any affected Terraform providers, find all of them under the new path and update the root_group_id
	// wherever it is not equal to the new root group ID.
	sql, args, err = dialect.Update("terraform_providers").
		Prepared(true).
		Set(
			goqu.Record{
				"version":       goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":    timestamp,
				"root_group_id": newRootGroupID,
			},
		).
		Where(
			goqu.And(
				goqu.I("terraform_providers.group_id").In(
					dialect.From(goqu.T("namespaces")).
						Select("group_id").
						Where(
							// Namespace is a group and is in the tree being migrated.
							goqu.And(
								goqu.I("group_id").Neq(nil),
								goqu.Or(
									goqu.I("path").Eq(newPath),
									goqu.I("path").Like(newPath+"/%"),
								),
							),
						)),
				goqu.I("terraform_providers.root_group_id").Neq(newRootGroupID),
			)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to prepare SQL to update the root group of Terraform providers")
		return nil, fmt.Errorf("failed to prepare SQL to update the root group of Terraform providers: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to update the root group of Terraform providers")
		return nil, fmt.Errorf("failed to execute query to update the root group of Terraform providers: %v", err)
	}

	// For any affected Terraform modules, find all of them under the new path and update the root_group_id
	// wherever it is not equal to the new root group ID.
	sql, args, err = dialect.Update("terraform_modules").
		Prepared(true).
		Set(
			goqu.Record{
				"version":       goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":    timestamp,
				"root_group_id": newRootGroupID,
			},
		).
		Where(
			goqu.And(
				goqu.I("terraform_modules.group_id").In(
					dialect.From(goqu.T("namespaces")).
						Select("group_id").
						Where(
							// Namespace is a group and is in the tree being migrated.
							goqu.And(
								goqu.I("group_id").Neq(nil),
								goqu.Or(
									goqu.I("path").Eq(newPath),
									goqu.I("path").Like(newPath+"/%"),
								),
							),
						)),
				goqu.I("terraform_modules.root_group_id").Neq(newRootGroupID),
			)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to prepare SQL to update the root group of Terraform modules")
		return nil, fmt.Errorf("failed to prepare SQL to update the root group of Terraform modules: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to update the root group of Terraform modules")
		return nil, fmt.Errorf("failed to execute query to update the root group of Terraform modules: %v", err)
	}

	// For any affected Terraform provider mirrors, find all of them under the new path and update the group_id
	// wherever it is not equal to the new root group ID.
	sql, args, err = dialect.Update("terraform_provider_version_mirrors").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"group_id":   newRootGroupID,
			},
		).
		Where(
			goqu.And(
				goqu.I("terraform_provider_version_mirrors.group_id").In(
					dialect.From(goqu.T("namespaces")).
						Select("group_id").
						Where(
							// Namespace is a group and is in the tree being migrated.
							goqu.And(
								goqu.I("group_id").Neq(nil),
								goqu.Or(
									goqu.I("path").Eq(newPath),
									goqu.I("path").Like(newPath+"/%"),
								),
							),
						)),
				goqu.I("terraform_provider_version_mirrors.group_id").Neq(newRootGroupID),
			)).ToSQL()
	if err != nil {
		tracing.RecordError(span, err,
			"failed to prepare SQL to update the root group of Terraform provider version mirrors")
		return nil, fmt.Errorf("failed to prepare SQL to update the group of Terraform provider version mirrors: %v", err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		tracing.RecordError(span, err,
			"failed to execute query to update the group of Terraform provider version mirrors")
		return nil, fmt.Errorf("failed to execute query to update the group of Terraform provider version mirrors: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err, "failed to commit group migration transaction")
		return nil, fmt.Errorf("failed to commit group migration transaction: %v", err)
	}

	return migratedGroup, nil
}

func (g *groups) DeleteGroup(ctx context.Context, group *models.Group) error {
	ctx, span := tracer.Start(ctx, "db.DeleteGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	sql, args, err := dialect.From("groups").
		Prepared(true).
		With("groups",
			dialect.Delete("groups").
				Where(
					goqu.Ex{
						"id":      group.Metadata.ID,
						"version": group.Metadata.Version,
					},
				).Returning("*"),
		).Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")})).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err := scanGroup(g.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true); err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_parent_id" {
				tracing.RecordError(span, nil,
					"all nested groups and workspaces must be deleted before this group can be deleted")
				return errors.New("all nested groups and workspaces must be deleted before this group can be deleted", errors.WithErrorCode(errors.EConflict))
			}
		}

		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func (g *groups) getGroup(ctx context.Context, exp goqu.Ex) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "db.getGroup")
	defer span.End()

	query := dialect.From(goqu.T("groups")).
		Prepared(true).
		Select(g.getSelectFields()...).
		InnerJoin(goqu.T("namespaces"), goqu.On(goqu.Ex{"groups.id": goqu.I("namespaces.group_id")})).
		Where(exp)

	sql, args, err := query.ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	group, err := scanGroup(g.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...), true)
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

	return group, nil
}

func (g *groups) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range groupFieldList {
		selectFields = append(selectFields, fmt.Sprintf("groups.%s", field))
	}
	selectFields = append(selectFields, "namespaces.path")

	return selectFields
}

func scanGroup(row scanner, withFullPath bool) (*models.Group, error) {
	var parentID sql.NullString
	var description sql.NullString
	var err error

	group := &models.Group{}

	fields := []interface{}{
		&group.Metadata.ID,
		&group.Metadata.CreationTimestamp,
		&group.Metadata.LastUpdatedTimestamp,
		&group.Metadata.Version,
		&group.Name,
		&description,
		&parentID,
		&group.CreatedBy,
		&group.RunnerTags,
		&group.EnableDriftDetection,
		&group.EnableProviderMirror,
	}

	if withFullPath {
		fields = append(fields, &group.FullPath)
	}

	err = row.Scan(fields...)

	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		group.ParentID = parentID.String
	}

	if description.Valid {
		group.Description = description.String
	}

	if withFullPath {
		group.Metadata.TRN = types.GroupModelType.BuildTRN(group.FullPath)
	}

	return group, nil
}
