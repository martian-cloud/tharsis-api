// Package group package
package group

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/trace"
)

// GetGroupsInput is the input for querying a list of groups
type GetGroupsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.GroupSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// ParentGroupID filters the groups by the parent group
	ParentGroupID *string
	// Search is used to search for a group by name or namespace path
	Search *string
	// Set RootOnly true to get only root groups returned by the query.
	RootOnly bool
	// GroupPath is the path of the group to be used for filtering
	GroupPath *string
	// Favorites filters to only return user's favorite groups
	Favorites *bool
	// ExcludeFavorites excludes the user's favorited groups from the results
	ExcludeFavorites *bool
}

// DeleteGroupInput is the input for deleting a group
type DeleteGroupInput struct {
	Group *models.Group
	Force bool
}

// Service implements all group related functionality
type Service interface {
	// GetGroupByID returns a group by ID
	GetGroupByID(ctx context.Context, id string) (*models.Group, error)
	// GetGroupByTRN returns a group by TRN
	GetGroupByTRN(ctx context.Context, trn string) (*models.Group, error)
	// GetGroupByIDs returns a list of groups by IDs
	GetGroupsByIDs(ctx context.Context, idList []string) ([]models.Group, error)
	// GetGroups returns a list of groups
	GetGroups(ctx context.Context, input *GetGroupsInput) (*db.GroupsResult, error)
	// DeleteGroup deletes a group by name
	DeleteGroup(ctx context.Context, input *DeleteGroupInput) error
	// CreateGroup creates a new group
	CreateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
	// UpdateGroup updates an existing group
	UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error)
	// MigrateGroup migrates an existing group to a new parent (or to root)
	MigrateGroup(ctx context.Context, groupID string, newParentID *string) (*models.Group, error)
	// GetRunnerTagsSetting returns the (inherited or direct) runner tags setting for a group.
	GetRunnerTagsSetting(ctx context.Context, group *models.Group) (*namespace.RunnerTagsSetting, error)
	// GetDriftDetectionEnabledSetting returns the (inherited or direct) drift detection enabled setting for a group.
	GetDriftDetectionEnabledSetting(ctx context.Context, group *models.Group) (*namespace.DriftDetectionEnabledSetting, error)
	// GetProviderMirrorEnabledSetting returns the (inherited or direct) provider mirror enabled setting for a group.
	GetProviderMirrorEnabledSetting(ctx context.Context, group *models.Group) (*namespace.ProviderMirrorEnabledSetting, error)
	// GetOutputVisibilitySetting returns the (inherited or direct) output visibility setting for a group.
	GetOutputVisibilitySetting(ctx context.Context, group *models.Group) (*namespace.OutputVisibilitySetting, error)
}

type service struct {
	logger                     logger.Logger
	dbClient                   *db.Client
	limitChecker               limits.LimitChecker
	namespaceMembershipService namespacemembership.Service
	inheritedSettingsResolver  namespace.InheritedSettingResolver
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	namespaceMembershipService namespacemembership.Service,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
) Service {
	return &service{
		logger:                     logger,
		dbClient:                   dbClient,
		limitChecker:               limitChecker,
		namespaceMembershipService: namespaceMembershipService,
		inheritedSettingsResolver:  inheritedSettingsResolver,
	}
}

func (s *service) GetGroupsByIDs(ctx context.Context, idList []string) ([]models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGroupsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{Filter: &db.GroupFilter{GroupIDs: idList}})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get groups", errors.WithSpan(span))
	}

	paths := []string{}
	for _, g := range resp.Groups {
		paths = append(paths, g.FullPath)
	}

	// Verify user has access to all returned groups
	if len(paths) > 0 {
		err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePaths(paths))
		if err != nil {
			return nil, err
		}
	}

	return resp.Groups, nil
}

func (s *service) GetGroups(ctx context.Context, input *GetGroupsInput) (*db.GroupsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGroups")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if input.ParentGroupID != nil && input.RootOnly {
		return nil, errors.New("RootOnly cannot be true when ParentGroup is specified")
	}

	dbInput := db.GetGroupsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.GroupFilter{
			Search: input.Search,
		},
	}

	// Handle favorites filter
	if input.Favorites != nil && *input.Favorites {
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New("only users can filter by favorites", errors.WithErrorCode(errors.EInvalid))
		}
		dbInput.Filter.FavoriteUserID = &userCaller.User.Metadata.ID
	}

	// Handle exclude favorites filter
	if input.ExcludeFavorites != nil && *input.ExcludeFavorites {
		userCaller, ok := caller.(*auth.UserCaller)
		if ok {
			dbInput.Filter.ExcludeFavoriteUserID = &userCaller.User.Metadata.ID
		}
	}

	if input.GroupPath != nil {
		dbInput.Filter.GroupPaths = []string{*input.GroupPath}
	}

	if input.ParentGroupID != nil {
		// Since parent group is specified we will authorize access based on the parent group
		err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithGroupID(*input.ParentGroupID))
		if err != nil {
			return nil, err
		}
		dbInput.Filter.ParentID = input.ParentGroupID
	} else {
		// Only return groups that the caller is a member of.
		dbInput.Filter.RootOnly = input.RootOnly

		if !caller.IsAdminModeActivated(ctx) {
			rootNamespaces, err := caller.GetRootNamespaceMemberships(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get root namespaces", errors.WithSpan(span))
			}
			// The db layer restricts to these memberships: exact root namespaces when RootOnly,
			// or their descendants otherwise.
			dbInput.Filter.RootNamespaceMemberships = rootNamespaces
		}
	}

	return s.dbClient.Groups.GetGroups(ctx, &dbInput)
}

func (s *service) GetGroupByID(ctx context.Context, id string) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGroupByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get group by ID", errors.WithSpan(span))
	}

	if group == nil {
		return nil, errors.New(
			"group with id %s not found", id,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *service) GetGroupByTRN(ctx context.Context, trn string) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGroupByTRN")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get group by trn", errors.WithSpan(span))
	}

	if group == nil {
		return nil, errors.New(
			"Group with trn %s not found", trn,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *service) DeleteGroup(ctx context.Context, input *DeleteGroupInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteGroupPermission, auth.WithGroupID(input.Group.Metadata.ID))
	if err != nil {
		return err
	}

	s.logger.WithContextFields(ctx).Infow("Requested deletion of a group.",
		"fullPath", input.Group.FullPath,
		"groupID", input.Group.Metadata.ID,
	)

	if !input.Force {
		// Check if this group has any sub-groups or workspaces

		subgroups, gErr := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{Filter: &db.GroupFilter{ParentID: &input.Group.Metadata.ID}})
		if gErr != nil {
			return errors.Wrap(gErr, "failed to get groups", errors.WithSpan(span))
		}

		if len(subgroups.Groups) > 0 {
			return errors.New(
				"This group can't be deleted because it contains subgroups, "+
					"use the force option to automatically delete all subgroups.",
				errors.WithErrorCode(errors.EConflict), errors.WithSpan(span),
			)
		}

		workspaces, wErr := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{GroupID: &input.Group.Metadata.ID}})
		if wErr != nil {
			return errors.Wrap(wErr, "failed to get workspaces", errors.WithSpan(span))
		}

		if len(workspaces.Workspaces) > 0 {
			return errors.New(
				"This group can't be deleted because it contains workspaces, "+
					"use the force option to automatically delete all workspaces in this group.",
				errors.WithErrorCode(errors.EConflict), errors.WithSpan(span),
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin a DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for DeleteGroup: %v", txErr)
		}
	}()

	// The foreign key with on cascade delete should remove activity events whose target ID is this group.

	// This will return an error if the group has nested groups or workspaces
	err = s.dbClient.Groups.DeleteGroup(txContext, input.Group)
	if err != nil {
		return errors.Wrap(err, "failed to delete a group", errors.WithSpan(span))
	}

	// If this group is nested, create an activity event for removal of this group from its parent.
	if input.Group.ParentID != "" {
		parentPath := input.Group.GetParentPath()
		if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
			&activity.CreateActivityEventInput{
				NamespacePath: &parentPath,
				Action:        models.ActionDeleteChildResource,
				TargetType:    models.TargetGroup,
				TargetID:      input.Group.ParentID,
				Payload: &models.ActivityEventDeleteChildResourcePayload{
					Name: input.Group.Name,
					ID:   input.Group.Metadata.ID,
					Type: string(models.TargetGroup),
				},
			}); err != nil {
			return errors.Wrap(err, "failed to create an activity event", errors.WithSpan(span))
		}
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateGroup(ctx context.Context, input *models.Group) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if input.ParentID != "" {
		err = caller.RequirePermission(ctx, models.CreateGroupPermission, auth.WithGroupID(input.ParentID))
		if err != nil {
			return nil, err
		}
	} else {
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New("Unsupported caller type, only users are allowed to create top-level groups", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
		}
		// Only admins with admin mode activated are allowed to create top level groups
		if !userCaller.IsAdminModeActivated(ctx) {
			return nil, errors.New("only admins with admin mode activated can create top-level groups", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
		}
	}

	// Validate model
	if err = input.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate a group model", errors.WithSpan(span))
	}

	input.CreatedBy = caller.GetSubject()

	// Auto-default output visibility for new root groups
	if input.ParentID == "" && input.OutputVisibility == nil {
		v := models.DefaultOutputVisibility
		input.OutputVisibility = &v
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin a DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateGroup: %v", txErr)
		}
	}()

	group, err := s.dbClient.Groups.CreateGroup(txContext, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a group", errors.WithSpan(span))
	}

	// If a nested group, check limits to see whether we just violated them.
	if input.ParentID != "" {

		// Check the limit on number of subgroups per parent.
		err = s.checkParentSubgroupLimit(txContext, span, input.ParentID)
		if err != nil {
			// The error has already been recorded to the tracing span.
			return nil, err
		}

		// Check the limit on depth of the tree.
		if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitGroupTreeDepth, limits.StaticCount(int32(group.GetDepth()))); err != nil {
			return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
		}
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create an activity event", errors.WithSpan(span))
	}

	// Add owner namespace membership if this is a top level group
	if input.ParentID == "" {
		// Create namespace membership for caller with owner access level
		namespaceMembershipInput := &namespacemembership.CreateNamespaceMembershipInput{
			NamespacePath:    group.FullPath,
			RoleID:           models.OwnerRoleID.String(),
			User:             caller.(*auth.UserCaller).User,
			SkipNotification: true,
		}

		// This call to CreateNamespaceMembership creates the activity event for the namespace membership,
		// so don't create another activity event from this module or there will be duplicates.
		if _, err := s.namespaceMembershipService.CreateNamespaceMembership(txContext, namespaceMembershipInput); err != nil {
			return nil, errors.Wrap(err, "failed to create a namespace membership", errors.WithSpan(span))
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit a DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a new group.",
		"fullPath", group.FullPath,
		"groupID", group.Metadata.ID,
	)
	return group, nil
}

func (s *service) UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateGroupPermission, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Validate model
	if err = group.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate a group model", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Requested an update to a group.",
		"fullPath", group.FullPath,
		"groupID", group.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin a DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateGroup: %v", txErr)
		}
	}()

	updatedGroup, err := s.dbClient.Groups.UpdateGroup(txContext, group)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update a group", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &updatedGroup.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetGroup,
			TargetID:      updatedGroup.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create an activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit a DB transaction", errors.WithSpan(span))
	}

	return updatedGroup, nil
}

func (s *service) MigrateGroup(ctx context.Context, groupID string, newParentID *string) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.MigrateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get the group to be moved.
	group, err := s.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a group by ID", errors.WithSpan(span))
	}
	if group == nil {
		return nil, errors.New(
			"group with id %s not found", groupID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Caller must have DeleteGroupPermission in the group being moved.
	err = caller.RequirePermission(ctx, models.DeleteGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	// If supplied, get the new parent group.
	var newParentPath string
	var newParent *models.Group
	var nErr error
	if newParentID != nil {
		newParent, nErr = s.dbClient.Groups.GetGroupByID(ctx, *newParentID)
		if nErr != nil {
			return nil, errors.Wrap(nErr, "failed to get a group by ID", errors.WithSpan(span))
		}
		if newParent == nil {
			return nil, errors.New(
				"group with id %s not found", *newParentID,
				errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}

		// In case a user gets confused or otherwise tries to do a no-op move, detect and bail out.
		// Because nothing gets done, it's safe to do this before the authorization check on the new parent.
		if group.ParentID == newParent.Metadata.ID {
			// Return BadRequest.
			return nil, errors.New("group already has the specified parent", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		// Make sure the group to be moved and the new parent group aren't exactly the same group.
		if newParent.FullPath == group.FullPath {
			return nil, errors.New("cannot move a group to be its own parent", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		// Make sure the group to be moved and the new parent group aren't respective ancestor and descendant.
		if newParent.IsDescendantOfGroup(group.FullPath) {
			return nil, errors.New("cannot move a group under one of its descendants", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		// If there is a new parent, the caller must have CreateGroupPermission in the new parent.
		err = caller.RequirePermission(ctx, models.CreateGroupPermission, auth.WithNamespacePath(newParent.FullPath))
		if err != nil {
			return nil, err
		}

		// The caller must also have CreateNamespaceMembershipPermission in the new parent. Moving a
		// subtree in introduces principals the destination's administrator never approved, and it
		// changes output visibility relationships, which are derived from namespace paths and group
		// IDs. Both are access decisions that belong to whoever controls access at the destination,
		// so this is checked in addition to (not instead of) CreateGroupPermission.
		//
		// Only the destination is checked. Moving a subtree out already requires
		// DeleteGroupPermission at the source, and a caller who can delete the group outright gains
		// nothing by moving it.
		err = caller.RequirePermission(ctx, models.CreateNamespaceMembershipPermission, auth.WithNamespacePath(newParent.FullPath))
		if err != nil {
			return nil, err
		}

		newParentPath = newParent.FullPath
	} else {

		// Return BadRequest if the user tries to move a root group to root.
		if group.ParentID == "" {
			// Return BadRequest.
			return nil, errors.New("group is already a top-level group", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		// If moving to root, the caller must be admin, because only admins are allowed to create new root groups.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New(
				"Unsupported caller type, only users are allowed to move groups to top-level",
				errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span),
			)
		}
		if !userCaller.IsAdminModeActivated(ctx) {
			return nil, errors.New("only admins with admin mode activated can move groups to top-level", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
		}
		// Leave newParentPath empty for the log message.
	}

	// Because the group to be moved and the new parent group have been fetched from the DB,
	// there's no need to validate them.

	s.logger.WithContextFields(ctx).Infow("Requested a group migration.",
		"fullPath", group.FullPath, // This is the full path of the group prior to migration.
		"groupID", group.Metadata.ID,
		"newParentPath", newParentPath,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin a DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer MigrateGroup: %v", txErr)
		}
	}()

	// Now that all checks have passed and the transaction is open, do the actual work of the migration.
	migratedGroup, err := s.dbClient.Groups.MigrateGroup(txContext, group, newParent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to migrate a group", errors.WithSpan(span))
	}

	// If it will be a nested group, check limits to see whether we just committed a violation.
	if newParentID != nil {

		// Check the limit on number of subgroups per parent.
		err = s.checkParentSubgroupLimit(txContext, span, *newParentID)
		if err != nil {
			// The error has already been recorded to the tracing span.
			return nil, err
		}

		// Check the limit on depth of the tree.
		childDepth, cErr := s.dbClient.Groups.GetChildDepth(txContext, migratedGroup)
		if cErr != nil {
			return nil, errors.Wrap(cErr, "failed to get group's depth of descendants", errors.WithSpan(span))
		}

		if err = s.limitChecker.CheckLimit(txContext,
			limits.ResourceLimitGroupTreeDepth, limits.StaticCount(int32(migratedGroup.GetDepth()+childDepth))); err != nil {
			return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
		}
	}

	// For now, generate an activity event on the group that was migrated--but without a custom payload.
	// The old parent (if any) and the new parent (if any) might have (also) wanted an activity event.
	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &migratedGroup.FullPath,
			Action:        models.ActionMigrate,
			TargetType:    models.TargetGroup,
			TargetID:      migratedGroup.Metadata.ID,
			Payload: &models.ActivityEventMigrateGroupPayload{
				PreviousGroupPath: group.FullPath,
			},
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create an activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to create a DB transaction", errors.WithSpan(span))
	}

	return migratedGroup, nil
}

// GetRunnerTagsSetting returns the (inherited or direct) runner tags setting for a group.
func (s *service) GetRunnerTagsSetting(ctx context.Context, group *models.Group) (*namespace.RunnerTagsSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerTagsSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return s.inheritedSettingsResolver.GetRunnerTags(ctx, group)
}

// GetDetectionEnabledSetting returns the (inherited or direct) setting for a group.
func (s *service) GetDriftDetectionEnabledSetting(ctx context.Context, group *models.Group) (*namespace.DriftDetectionEnabledSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetDriftDetectionEnabledsSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return s.inheritedSettingsResolver.GetDriftDetectionEnabled(ctx, group)
}

// GetProviderMirrorEnabledSetting returns the (inherited or direct) setting for a group.
func (s *service) GetProviderMirrorEnabledSetting(ctx context.Context, group *models.Group) (*namespace.ProviderMirrorEnabledSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderMirrorEnabledSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return s.inheritedSettingsResolver.GetProviderMirrorEnabled(ctx, group)
}

// GetOutputVisibilitySetting returns the (inherited or direct) output visibility setting for a group.
func (s *service) GetOutputVisibilitySetting(ctx context.Context, group *models.Group) (*namespace.OutputVisibilitySetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetOutputVisibilitySetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	setting, err := s.inheritedSettingsResolver.GetOutputVisibility(ctx, group)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get output visibility setting", errors.WithSpan(span))
	}

	return setting, nil
}

// checkParentSubgroupLimit checks whether the parent subgroup limit has just been violated.
// This function records any errors on the span.
func (s *service) checkParentSubgroupLimit(ctx context.Context, span trace.Span, parentID string) error {
	children, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
		Filter: &db.GroupFilter{
			ParentID: &parentID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get parent group's children", errors.WithSpan(span))
	}

	if err = s.limitChecker.CheckLimit(ctx, limits.ResourceLimitSubgroupsPerParent, children.PageInfo.TotalCount); err != nil {
		return errors.Wrap(err, "limit check failed", errors.WithSpan(span))
	}

	return nil
}
