// Package group package
package group

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
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
}

type service struct {
	logger                     logger.Logger
	dbClient                   *db.Client
	limitChecker               limits.LimitChecker
	namespaceMembershipService namespacemembership.Service
	activityService            activityevent.Service
	inheritedSettingsResolver  namespace.InheritedSettingResolver
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	namespaceMembershipService namespacemembership.Service,
	activityService activityevent.Service,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
) Service {
	return &service{
		logger:                     logger,
		dbClient:                   dbClient,
		limitChecker:               limitChecker,
		namespaceMembershipService: namespaceMembershipService,
		activityService:            activityService,
		inheritedSettingsResolver:  inheritedSettingsResolver,
	}
}

func (s *service) GetGroupsByIDs(ctx context.Context, idList []string) ([]models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.GetGroupsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	resp, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{Filter: &db.GroupFilter{GroupIDs: idList}})
	if err != nil {
		tracing.RecordError(span, err, "failed to get groups")
		return nil, err
	}

	paths := []string{}
	for _, g := range resp.Groups {
		paths = append(paths, g.FullPath)
	}

	// Verify user has access to all returned groups
	if len(paths) > 0 {
		err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePaths(paths))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
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
		tracing.RecordError(span, err, "caller authorization failed")
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

	if input.GroupPath != nil {
		dbInput.Filter.GroupPaths = []string{*input.GroupPath}
	}

	if input.ParentGroupID != nil {
		// Since parent group is specified we will authorize access based on the parent group
		err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithGroupID(*input.ParentGroupID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		dbInput.Filter.ParentID = input.ParentGroupID
	} else {
		// Only return groups that the caller is a member of
		policy, err := caller.GetNamespaceAccessPolicy(ctx)
		if err != nil {
			tracing.RecordError(span, err, "failed to get namespace access policy")
			return nil, err
		}

		if policy.AllowAll {
			// Policy is set to allow all so no need for additional authorization
			dbInput.Filter.RootOnly = input.RootOnly
		} else {
			if input.RootOnly {
				// RootOnly is true so filter by root namesapce IDs from the policy
				dbInput.Filter.NamespaceIDs = policy.RootNamespaceIDs
			} else {
				// RootOnly if false so filter by group memberships
				if err = auth.HandleCaller(
					ctx,
					func(_ context.Context, c *auth.UserCaller) error {
						dbInput.Filter.UserMemberID = &c.User.Metadata.ID
						return nil
					},
					func(_ context.Context, c *auth.ServiceAccountCaller) error {
						dbInput.Filter.ServiceAccountMemberID = &c.ServiceAccountID
						return nil
					},
				); err != nil {
					return nil, err
				}
			}
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
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by ID")
		return nil, err
	}

	if group == nil {
		tracing.RecordError(span, nil, "group with id %s not found", id)
		return nil, errors.New(
			"group with id %s not found", id,
			errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by trn")
		return nil, err
	}

	if group == nil {
		tracing.RecordError(span, nil, "Group with trn %s not found", trn)
		return nil, errors.New(
			"Group with trn %s not found", trn,
			errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteGroupPermission, auth.WithGroupID(input.Group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
			tracing.RecordError(span, gErr, "failed to get groups")
			return gErr
		}

		if len(subgroups.Groups) > 0 {
			tracing.RecordError(span, nil,
				"This group can't be deleted because it contains subgroups, "+
					"use the force option to automatically delete all subgroups.")
			return errors.New(
				"This group can't be deleted because it contains subgroups, "+
					"use the force option to automatically delete all subgroups.",
				errors.WithErrorCode(errors.EConflict),
			)
		}

		workspaces, wErr := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{GroupID: &input.Group.Metadata.ID}})
		if wErr != nil {
			tracing.RecordError(span, wErr, "failed to get workspaces")
			return wErr
		}

		if len(workspaces.Workspaces) > 0 {
			tracing.RecordError(span, nil,
				"This group can't be deleted because it contains workspaces, "+
					"use the force option to automatically delete all workspaces in this group.")
			return errors.New(
				"This group can't be deleted because it contains workspaces, "+
					"use the force option to automatically delete all workspaces in this group.",
				errors.WithErrorCode(errors.EConflict),
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin a DB transaction")
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
		tracing.RecordError(span, err, "failed to delete a group")
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
			tracing.RecordError(span, err, "failed to create an activity event")
			return err
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
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.ParentID != "" {
		err = caller.RequirePermission(ctx, models.CreateGroupPermission, auth.WithGroupID(input.ParentID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else {
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			tracing.RecordError(span, nil, "Unsupported caller type, only users are allowed to create top-level groups")
			return nil, errors.New("Unsupported caller type, only users are allowed to create top-level groups", errors.WithErrorCode(errors.EForbidden))
		}
		// Only admins are allowed to create top level groups
		if !userCaller.User.Admin {
			tracing.RecordError(span, nil, "Only system admins can create top-level groups")
			return nil, errors.New("Only system admins can create top-level groups", errors.WithErrorCode(errors.EForbidden))
		}
	}

	// Validate model
	if err = input.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate a group model")
		return nil, err
	}

	input.CreatedBy = caller.GetSubject()

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin a DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateGroup: %v", txErr)
		}
	}()

	group, err := s.dbClient.Groups.CreateGroup(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to create a group")
		return nil, err
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
		if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitGroupTreeDepth, int32(group.GetDepth())); err != nil {
			tracing.RecordError(span, err, "limit check failed")
			return nil, err
		}
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create an activity event")
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
			tracing.RecordError(span, err, "failed to create a namespace membership")
			return nil, err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit a DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.UpdateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateGroupPermission, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model
	if err = group.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate a group model")
		return nil, err
	}

	s.logger.Infow("Requested an update to a group.",
		"caller", caller.GetSubject(),
		"fullPath", group.FullPath,
		"groupID", group.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin a DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateGroup: %v", txErr)
		}
	}()

	updatedGroup, err := s.dbClient.Groups.UpdateGroup(txContext, group)
	if err != nil {
		tracing.RecordError(span, err, "failed to update a group")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedGroup.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetGroup,
			TargetID:      updatedGroup.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create an activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit a DB transaction")
		return nil, err
	}

	return updatedGroup, nil
}

func (s *service) MigrateGroup(ctx context.Context, groupID string, newParentID *string) (*models.Group, error) {
	ctx, span := tracer.Start(ctx, "svc.MigrateGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get the group to be moved.
	group, err := s.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get a group by ID")
		return nil, err
	}
	if group == nil {
		tracing.RecordError(span, nil, "group with id %s not found", groupID)
		return nil, errors.New(
			"group with id %s not found", groupID,
			errors.WithErrorCode(errors.ENotFound))
	}

	// Caller must have DeleteGroupPermission in the group being moved.
	err = caller.RequirePermission(ctx, models.DeleteGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// If supplied, get the new parent group.
	var newParentPath string
	var newParent *models.Group
	var nErr error
	if newParentID != nil {
		newParent, nErr = s.dbClient.Groups.GetGroupByID(ctx, *newParentID)
		if nErr != nil {
			tracing.RecordError(span, nErr, "failed to get a group by ID")
			return nil, nErr
		}
		if newParent == nil {
			tracing.RecordError(span, nil, "group with id %s not found", *newParentID)
			return nil, errors.New(
				"group with id %s not found", *newParentID,
				errors.WithErrorCode(errors.ENotFound))
		}

		// In case a user gets confused or otherwise tries to do a no-op move, detect and bail out.
		// Because nothing gets done, it's safe to do this before the authorization check on the new parent.
		if group.ParentID == newParent.Metadata.ID {
			// Return BadRequest.
			tracing.RecordError(span, nil, "group already has the specified parent")
			return nil, errors.New("group already has the specified parent", errors.WithErrorCode(errors.EInvalid))
		}

		// Make sure the group to be moved and the new parent group aren't exactly the same group.
		if newParent.FullPath == group.FullPath {
			tracing.RecordError(span, nil, "cannot move a group to be its own parent")
			return nil, errors.New("cannot move a group to be its own parent", errors.WithErrorCode(errors.EInvalid))
		}

		// Make sure the group to be moved and the new parent group aren't respective ancestor and descendant.
		if newParent.IsDescendantOfGroup(group.FullPath) {
			tracing.RecordError(span, nil, "cannot move a group under one of its descendants")
			return nil, errors.New("cannot move a group under one of its descendants", errors.WithErrorCode(errors.EInvalid))
		}

		// If there is a new parent, the caller must have CreateGroupPermission in the new parent.
		err = caller.RequirePermission(ctx, models.CreateGroupPermission, auth.WithNamespacePath(newParent.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}

		newParentPath = newParent.FullPath
	} else {

		// Return BadRequest if the user tries to move a root group to root.
		if group.ParentID == "" {
			// Return BadRequest.
			tracing.RecordError(span, nil, "group is already a top-level group")
			return nil, errors.New("group is already a top-level group", errors.WithErrorCode(errors.EInvalid))
		}

		// If moving to root, the caller must be admin, because only admins are allowed to create new root groups.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			tracing.RecordError(span, nil, "Unsupported caller type, only users are allowed to move groups to top-level")
			return nil, errors.New(
				"Unsupported caller type, only users are allowed to move groups to top-level",
				errors.WithErrorCode(errors.EForbidden),
			)
		}
		if !userCaller.User.Admin {
			tracing.RecordError(span, nil, "Only system admins can move groups to top-level")
			return nil, errors.New("Only system admins can move groups to top-level", errors.WithErrorCode(errors.EForbidden))
		}
		// Leave newParentPath empty for the log message.
	}

	// Because the group to be moved and the new parent group have been fetched from the DB,
	// there's no need to validate them.

	s.logger.Infow("Requested a group migration.",
		"caller", caller.GetSubject(),
		"fullPath", group.FullPath, // This is the full path of the group prior to migration.
		"groupID", group.Metadata.ID,
		"newParentPath", newParentPath,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin a DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer MigrateGroup: %v", txErr)
		}
	}()

	// Now that all checks have passed and the transaction is open, do the actual work of the migration.
	migratedGroup, err := s.dbClient.Groups.MigrateGroup(txContext, group, newParent)
	if err != nil {
		tracing.RecordError(span, err, "failed to migrate a group")
		return nil, err
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
			tracing.RecordError(span, cErr, "failed to get group's depth of descendants")
			return nil, cErr
		}

		if err = s.limitChecker.CheckLimit(txContext,
			limits.ResourceLimitGroupTreeDepth, int32(migratedGroup.GetDepth()+childDepth)); err != nil {
			tracing.RecordError(span, err, "limit check failed")
			return nil, err
		}
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
		tracing.RecordError(span, err, "failed to create an activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to create a DB transaction")
		return nil, err
	}

	return migratedGroup, nil
}

// GetRunnerTagsSetting returns the (inherited or direct) runner tags setting for a group.
func (s *service) GetRunnerTagsSetting(ctx context.Context, group *models.Group) (*namespace.RunnerTagsSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerTagsSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(group.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return s.inheritedSettingsResolver.GetDriftDetectionEnabled(ctx, group)
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
		tracing.RecordError(span, err, "failed to get parent group's children")
		return err
	}

	if err = s.limitChecker.CheckLimit(ctx, limits.ResourceLimitSubgroupsPerParent, children.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return err
	}

	return nil
}
