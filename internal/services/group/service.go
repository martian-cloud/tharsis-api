package group

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
)

// GetGroupsInput is the input for querying a list of groups
type GetGroupsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.GroupSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
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

	// Verify user has access to all returned groups
	if err := caller.RequireViewerAccessToGroups(ctx, resp.Groups); err != nil {
		return nil, err
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
		if err := caller.RequireAccessToNamespace(ctx, input.ParentGroup.FullPath, models.ViewerRole); err != nil {
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
		return nil, errors.NewError(
			errors.ENotFound,
			fmt.Sprintf("Group with id %s not found", id),
		)
	}

	if err := caller.RequireAccessToNamespace(ctx, group.FullPath, models.ViewerRole); err != nil {
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
		return nil, errors.NewError(
			errors.ENotFound,
			fmt.Sprintf("Group with path %s not found", path),
		)
	}

	if err := caller.RequireAccessToNamespace(ctx, group.FullPath, models.ViewerRole); err != nil {
		return nil, err
	}

	return group, nil
}

func (s *service) DeleteGroup(ctx context.Context, input *DeleteGroupInput) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if input.Group.ParentID == "" {
		// Require owner role to delete top level groups
		if err := caller.RequireAccessToNamespace(ctx, input.Group.FullPath, models.OwnerRole); err != nil {
			return err
		}
	} else {
		// Require deployer role to delete nested groups
		if err := caller.RequireAccessToNamespace(ctx, input.Group.FullPath, models.DeployerRole); err != nil {
			return err
		}
	}

	s.logger.Infow("Requested deletion of a group.",
		"caller", caller.GetSubject(),
		"fullPath", input.Group.FullPath,
		"groupID", input.Group.Metadata.ID,
	)

	if !input.Force {
		// Check if this group has any sub-groups or workspaces

		subgroups, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{Filter: &db.GroupFilter{ParentID: &input.Group.Metadata.ID}})
		if err != nil {
			return err
		}

		if len(subgroups.Groups) > 0 {
			return errors.NewError(
				errors.EConflict,
				fmt.Sprintf("This group can't be deleted because it contains subgroups, "+
					"use the force option to automatically delete all subgroups."),
			)
		}

		workspaces, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{GroupID: &input.Group.Metadata.ID}})
		if err != nil {
			return err
		}

		if len(workspaces.Workspaces) > 0 {
			return errors.NewError(
				errors.EConflict,
				fmt.Sprintf("This group can't be deleted because it contains workspaces, "+
					"use the force option to automatically delete all workspaces in this group."),
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
		if err = caller.RequireAccessToGroup(ctx, input.ParentID, models.DeployerRole); err != nil {
			return nil, err
		}
	} else {
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.NewError(errors.EForbidden, "Unsupported caller type, only users are allowed to create top-level groups")
		}
		// Only admins are allowed to create top level groups
		if !userCaller.User.Admin {
			return nil, errors.NewError(errors.EForbidden, "Only system admins can create top-level groups")
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
			Role:          models.OwnerRole,
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

	if err := caller.RequireAccessToNamespace(ctx, group.FullPath, models.DeployerRole); err != nil {
		return nil, err
	}

	// Validate model
	if err := group.Validate(); err != nil {
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
