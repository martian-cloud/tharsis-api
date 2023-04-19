// Package group package
package group

import (
	"context"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetGroupsInput is the input for querying a list of groups
type GetGroupsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.GroupSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// ParentGroup filters the groups by the parent group
	ParentGroup *models.Group
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
	// GetGroupByFullPath returns a group by full path
	GetGroupByFullPath(ctx context.Context, path string) (*models.Group, error)
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
}

type service struct {
	logger                     logger.Logger
	dbClient                   *db.Client
	namespaceMembershipService namespacemembership.Service
	activityService            activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	namespaceMembershipService namespacemembership.Service,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:                     logger,
		dbClient:                   dbClient,
		namespaceMembershipService: namespaceMembershipService,
		activityService:            activityService,
	}
}

func (s *service) GetGroupsByIDs(ctx context.Context, idList []string) ([]models.Group, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{Filter: &db.GroupFilter{GroupIDs: idList}})
	if err != nil {
		return nil, err
	}

	paths := []string{}
	for _, g := range resp.Groups {
		paths = append(paths, g.FullPath)
	}

	// Verify user has access to all returned groups
	if len(paths) > 0 {
		err = caller.RequirePermission(ctx, permissions.ViewGroupPermission, auth.WithNamespacePaths(paths))
		if err != nil {
			return nil, err
		}
	}

	return resp.Groups, nil
}

func (s *service) GetGroups(ctx context.Context, input *GetGroupsInput) (*db.GroupsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	dbInput := db.GetGroupsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            &db.GroupFilter{},
	}

	if input.ParentGroup != nil {
		err = caller.RequirePermission(ctx, permissions.ViewGroupPermission, auth.WithNamespacePath(input.ParentGroup.FullPath))
		if err != nil {
			return nil, err
		}
		dbInput.Filter.ParentID = &input.ParentGroup.Metadata.ID
	} else {
		// Only return groups that the caller is a member of
		policy, err := caller.GetNamespaceAccessPolicy(ctx)
		if err != nil {
			return nil, err
		}

		if !policy.AllowAll {
			dbInput.Filter.NamespaceIDs = policy.RootNamespaceIDs
		} else {
			dbInput.Filter.RootOnly = true
		}
	}

	return s.dbClient.Groups.GetGroups(ctx, &dbInput)
}

func (s *service) GetGroupByID(ctx context.Context, id string) (*models.Group, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if group == nil {
		return nil, errors.New(
			errors.ENotFound,
			"Group with id %s not found", id,
		)
	}

	err = caller.RequirePermission(ctx, permissions.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *service) GetGroupByFullPath(ctx context.Context, path string) (*models.Group, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByFullPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if group == nil {
		return nil, errors.New(
			errors.ENotFound,
			"Group with path %s not found", path,
		)
	}

	err = caller.RequirePermission(ctx, permissions.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *service) DeleteGroup(ctx context.Context, input *DeleteGroupInput) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteGroupPermission, auth.WithGroupID(input.Group.Metadata.ID))
	if err != nil {
		return err
	}

	s.logger.Infow("Requested deletion of a group.",
		"caller", caller.GetSubject(),
		"fullPath", input.Group.FullPath,
		"groupID", input.Group.Metadata.ID,
	)

	if !input.Force {
		// Check if this group has any sub-groups or workspaces

		subgroups, gErr := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{Filter: &db.GroupFilter{ParentID: &input.Group.Metadata.ID}})
		if gErr != nil {
			return gErr
		}

		if len(subgroups.Groups) > 0 {
			return errors.New(
				errors.EConflict,
				"This group can't be deleted because it contains subgroups, "+
					"use the force option to automatically delete all subgroups.",
			)
		}

		workspaces, wErr := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{GroupID: &input.Group.Metadata.ID}})
		if wErr != nil {
			return wErr
		}

		if len(workspaces.Workspaces) > 0 {
			return errors.New(
				errors.EConflict,
				"This group can't be deleted because it contains workspaces, "+
					"use the force option to automatically delete all workspaces in this group.",
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for DeleteGroup: %v", txErr)
		}
	}()

	// The foreign key with on cascade delete should remove activity events whose target ID is this group.

	// This will return an error if the group has nested groups or workspaces
	err = s.dbClient.Groups.DeleteGroup(txContext, input.Group)
	if err != nil {
		return err
	}

	// If this group is nested, create an activity event for removal of this group from its parent.
	if input.Group.ParentID != "" {
		parentPath := input.Group.GetParentPath()
		if _, err = s.activityService.CreateActivityEvent(txContext,
			&activityevent.CreateActivityEventInput{
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
			return err
		}
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateGroup(ctx context.Context, input *models.Group) (*models.Group, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if input.ParentID != "" {
		err = caller.RequirePermission(ctx, permissions.CreateGroupPermission, auth.WithGroupID(input.ParentID))
		if err != nil {
			return nil, err
		}
	} else {
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New(errors.EForbidden, "Unsupported caller type, only users are allowed to create top-level groups")
		}
		// Only admins are allowed to create top level groups
		if !userCaller.User.Admin {
			return nil, errors.New(errors.EForbidden, "Only system admins can create top-level groups")
		}
	}

	// Validate model
	if err = input.Validate(); err != nil {
		return nil, err
	}

	input.CreatedBy = caller.GetSubject()

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateGroup: %v", txErr)
		}
	}()

	group, err := s.dbClient.Groups.CreateGroup(txContext, input)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	// Add owner namespace membership if this is a top level group
	if input.ParentID == "" {
		// Create namespace membership for caller with owner access level
		namespaceMembershipInput := &namespacemembership.CreateNamespaceMembershipInput{
			NamespacePath: group.FullPath,
			RoleID:        models.OwnerRoleID.String(),
			User:          caller.(*auth.UserCaller).User,
		}

		// This call to CreateNamespaceMembership creates the activity event for the namespace membership,
		// so don't create another activity event from this module or there will be duplicates.
		if _, err := s.namespaceMembershipService.CreateNamespaceMembership(txContext, namespaceMembershipInput); err != nil {
			return nil, err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	s.logger.Infow("Created a new group.",
		"caller", caller.GetSubject(),
		"fullPath", group.FullPath,
		"groupID", group.Metadata.ID,
	)
	return group, nil
}

func (s *service) UpdateGroup(ctx context.Context, group *models.Group) (*models.Group, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateGroupPermission, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Validate model
	if err = group.Validate(); err != nil {
		return nil, err
	}

	s.logger.Infow("Requested an update to a group.",
		"caller", caller.GetSubject(),
		"fullPath", group.FullPath,
		"groupID", group.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateGroup: %v", txErr)
		}
	}()

	updatedGroup, err := s.dbClient.Groups.UpdateGroup(txContext, group)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedGroup.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetGroup,
			TargetID:      updatedGroup.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedGroup, nil
}

func (s *service) MigrateGroup(ctx context.Context, groupID string, newParentID *string) (*models.Group, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get the group to be moved.
	group, err := s.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New(
			errors.ENotFound,
			"Group with id %s not found", groupID,
		)
	}

	// Caller must have UpdateGroupPermission in the group being moved.
	err = caller.RequirePermission(ctx, permissions.DeleteGroupPermission, auth.WithNamespacePath(group.FullPath))
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
			return nil, nErr
		}
		if newParent == nil {
			return nil, errors.New(
				errors.ENotFound,
				"Group with id %s not found", *newParentID,
			)
		}

		// In case a user gets confused or otherwise tries to do a no-op move, detect and bail out.
		// Because nothing gets done, it's safe to do this before the authorization check on the new parent.
		if group.ParentID == newParent.Metadata.ID {
			// Return BadRequest.
			return nil, errors.New(errors.EInvalid, "group already has the specified parent")
		}

		// Make sure the group to be moved and the new parent group aren't exactly the same group.
		if newParent.FullPath == group.FullPath {
			return nil, errors.New(errors.EInvalid, "cannot move a group to be its own parent")
		}

		// Make sure the group to be moved and the new parent group aren't respective ancestor and descendant.
		if strings.HasPrefix(newParent.FullPath, (group.FullPath + "/")) {
			return nil, errors.New(errors.EInvalid, "cannot move a group under one of its descendants")
		}

		// If there is a new parent, the caller must have CreateGroupPermission in the new parent.
		err = caller.RequirePermission(ctx, permissions.CreateGroupPermission, auth.WithNamespacePath(newParent.FullPath))
		if err != nil {
			return nil, err
		}

		newParentPath = newParent.FullPath
	} else {

		// Return BadRequest if the user tries to move a root group to root.
		if group.ParentID == "" {
			// Return BadRequest.
			return nil, errors.New(errors.EInvalid, "group is already a top-level group")
		}

		// If moving to root, the caller must be admin, because only admins are allowed to create new root groups.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New(errors.EForbidden,
				"Unsupported caller type, only users are allowed to move groups to top-level")
		}
		if !userCaller.User.Admin {
			return nil, errors.New(errors.EForbidden, "Only system admins can move groups to top-level")
		}
		// Leave newParentPath empty for the log message.
	}

	// Because the group to be moved and the new parent group have been fetched from the DB,
	// there's no need to validate them.

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer MigrateGroup: %v", txErr)
		}
	}()

	s.logger.Infow("Requested a group migration.",
		"caller", caller.GetSubject(),
		"fullPath", group.FullPath, // This is the full path of the group prior to migration.
		"groupID", group.Metadata.ID,
		"newParentPath", newParentPath,
	)

	// Now that all checks have passed and the transaction is open, do the actual work of the migration.
	migratedGroup, err := s.dbClient.Groups.MigrateGroup(txContext, group, newParent)
	if err != nil {
		return nil, err
	}

	// For now, generate an activity event on the group that was migrated--but without a custom payload.
	// The old parent (if any) and the new parent (if any) might have (also) wanted an activity event.
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &migratedGroup.FullPath,
			Action:        models.ActionMigrate,
			TargetType:    models.TargetGroup,
			TargetID:      migratedGroup.Metadata.ID,
			Payload: &models.ActivityEventMigrateGroupPayload{
				PreviousGroupPath: group.FullPath,
			},
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return migratedGroup, nil
}
