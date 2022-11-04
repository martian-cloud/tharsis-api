package managedidentity

import (
	"context"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

// GetManagedIdentitiesInput is the input for listing managed identities
type GetManagedIdentitiesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.ManagedIdentitySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *db.PaginationOptions
	// Search returns only the managed identities with a name or resource path that starts with the value of search
	Search *string
	// NamespacePath is the namespace to return service accounts for
	NamespacePath string
	// IncludeInherited includes inherited managed identities in the result
	IncludeInherited bool
}

// DeleteManagedIdentityInput is the input for deleting a managed identity
type DeleteManagedIdentityInput struct {
	ManagedIdentity *models.ManagedIdentity
	Force           bool
}

// Service implements managed identity functionality
type Service interface {
	GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error)
	GetManagedIdentityByPath(ctx context.Context, path string) (*models.ManagedIdentity, error)
	GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*db.ManagedIdentitiesResult, error)
	GetManagedIdentitiesByIDs(ctx context.Context, ids []string) ([]models.ManagedIdentity, error)
	CreateManagedIdentity(ctx context.Context, input *types.CreateManagedIdentityInput) (*models.ManagedIdentity, error)
	UpdateManagedIdentity(ctx context.Context, input *types.UpdateManagedIdentityInput) (*models.ManagedIdentity, error)
	DeleteManagedIdentity(ctx context.Context, input *DeleteManagedIdentityInput) error
	CreateCredentials(ctx context.Context, identity *models.ManagedIdentity) ([]byte, error)
	GetManagedIdentitiesForWorkspace(ctx context.Context, workspaceID string) ([]models.ManagedIdentity, error)
	AddManagedIdentityToWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error
	RemoveManagedIdentityFromWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error
	GetManagedIdentityAccessRules(ctx context.Context, managedIdentity *models.ManagedIdentity) ([]models.ManagedIdentityAccessRule, error)
	GetManagedIdentityAccessRulesByIDs(ctx context.Context, ids []string) ([]models.ManagedIdentityAccessRule, error)
	GetManagedIdentityAccessRule(ctx context.Context, ruleID string) (*models.ManagedIdentityAccessRule, error)
	CreateManagedIdentityAccessRule(ctx context.Context, input *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error)
	UpdateManagedIdentityAccessRule(ctx context.Context, input *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error)
	DeleteManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) error
}

type service struct {
	logger           logger.Logger
	dbClient         *db.Client
	delegateMap      map[models.ManagedIdentityType]Delegate
	workspaceService workspace.Service
	jobService       job.Service
	activityService  activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	managedIdentityDelegateMap map[models.ManagedIdentityType]Delegate,
	workspaceService workspace.Service,
	jobService job.Service,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:           logger,
		dbClient:         dbClient,
		delegateMap:      managedIdentityDelegateMap,
		workspaceService: workspaceService,
		jobService:       jobService,
		activityService:  activityService,
	}
}

func (s *service) GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*db.ManagedIdentitiesResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToNamespace(ctx, input.NamespacePath, models.ViewerRole); err != nil {
		return nil, err
	}

	filter := &db.ManagedIdentityFilter{
		Search: input.Search,
	}

	if input.IncludeInherited {
		pathParts := strings.Split(input.NamespacePath, "/")

		paths := []string{}
		for len(pathParts) > 0 {
			paths = append(paths, strings.Join(pathParts, "/"))
			// Remove last element
			pathParts = pathParts[:len(pathParts)-1]
		}

		filter.NamespacePaths = paths
	} else {
		// This will return an empty result for workspace namespaces because workspaces
		// don't have managed identities directly associated (i.e. only group namespaces do)
		filter.NamespacePaths = []string{input.NamespacePath}
	}

	result, err := s.dbClient.ManagedIdentities.GetManagedIdentities(ctx, &db.GetManagedIdentitiesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *service) DeleteManagedIdentity(ctx context.Context, input *DeleteManagedIdentityInput) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if rErr := caller.RequireAccessToGroup(ctx, input.ManagedIdentity.GroupID, models.DeployerRole); rErr != nil {
		return rErr
	}

	s.logger.Infow("Requested to delete a managed identity.",
		"caller", caller.GetSubject(),
		"groupID", input.ManagedIdentity.GroupID,
		"managedIdentityID", input.ManagedIdentity.Metadata.ID,
	)

	if !input.Force {
		// Verify that managed identity is not assigned to any workspaces
		workspaces, wErr := s.dbClient.Workspaces.GetWorkspacesForManagedIdentity(ctx, input.ManagedIdentity.Metadata.ID)
		if wErr != nil {
			return wErr
		}
		if len(workspaces) > 0 {
			return errors.NewError(
				errors.EConflict,
				fmt.Sprintf("This managed identity can't be deleted because it's currently assigned to %d workspaces. "+
					"Setting force to true will automatically remove this managed identity from all workspaces it's assigned to.", len(workspaces)),
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteManagedIdentity: %v", txErr)
		}
	}()

	err = s.dbClient.ManagedIdentities.DeleteManagedIdentity(txContext, input.ManagedIdentity)
	if err != nil {
		return err
	}

	groupPath := input.ManagedIdentity.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      input.ManagedIdentity.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: input.ManagedIdentity.Name,
				ID:   input.ManagedIdentity.Metadata.ID,
				Type: string(models.TargetManagedIdentity),
			},
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetManagedIdentitiesForWorkspace(ctx context.Context, workspaceID string) ([]models.ManagedIdentity, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, workspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	identities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return identities, nil
}

func (s *service) AddManagedIdentityToWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err = caller.RequireAccessToWorkspace(ctx, workspaceID, models.DeployerRole); err != nil {
		return err
	}

	// Get managed identity that will be added
	identity, err := s.getManagedIdentityByID(ctx, managedIdentityID)
	if err != nil {
		return err
	}

	// Get workspace
	workspace, err := s.workspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return err
	}

	resourcePathParts := strings.Split(identity.ResourcePath, "/")
	groupPath := strings.Join(resourcePathParts[:len(resourcePathParts)-1], "/")

	// Verify that the managed identity's group is in the group hierarchy of the workspace
	if !strings.HasPrefix(workspace.FullPath, fmt.Sprintf("%s/", groupPath)) {
		return fmt.Errorf("managed identity %s is not available to workspace %s", managedIdentityID, workspaceID)
	}

	identitiesInWorkspace, err := s.GetManagedIdentitiesForWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}

	// Verify that only one type of each managed identity can be assigned at a time
	for _, mi := range identitiesInWorkspace {
		if mi.Type == identity.Type {
			return errors.NewError(errors.EInvalid, fmt.Sprintf("Managed identity with type %s already assigned to workspace %s", identity.Type, workspaceID))
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer AddManagedIdentityToWorkspace: %v", txErr)
		}
	}()

	if aErr := s.dbClient.ManagedIdentities.AddManagedIdentityToWorkspace(txContext,
		managedIdentityID, workspaceID); aErr != nil {
		return aErr
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionAdd,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      identity.Metadata.ID,
		}); err != nil {
		return err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return err
	}

	s.logger.Infow("Added a managed identity to a workspace.",
		"caller", caller.GetSubject(),
		"workspaceID", workspace.Metadata.ID,
		"fullPath", workspace.FullPath,
		"managedIdentityID", managedIdentityID,
	)
	return nil
}

func (s *service) RemoveManagedIdentityFromWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if rErr := caller.RequireAccessToWorkspace(ctx, workspaceID, models.DeployerRole); rErr != nil {
		return rErr
	}

	// Get managed identity that will be removed
	identity, err := s.getManagedIdentityByID(ctx, managedIdentityID)
	if err != nil {
		return err
	}

	// Get workspace
	workspace, err := s.workspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer RemoveManagedIdentityFromWorkspace: %v", txErr)
		}
	}()

	if err = s.dbClient.ManagedIdentities.RemoveManagedIdentityFromWorkspace(txContext,
		managedIdentityID, workspaceID); err != nil {
		return err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionRemove,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      identity.Metadata.ID,
		}); err != nil {
		return err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return err
	}

	s.logger.Infow("Deleted a managed identity from workspace.",
		"caller", caller.GetSubject(),
		"workspaceID", workspaceID,
		"managedIdentityID", managedIdentityID,
	)
	return nil
}

func (s *service) GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get identity from DB
	identity, err := s.getManagedIdentityByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToInheritedGroupResource(ctx, identity.GroupID); err != nil {
		return nil, err
	}

	return identity, nil
}

func (s *service) GetManagedIdentityByPath(ctx context.Context, path string) (*models.ManagedIdentity, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if isResourcePathInvalid(path) {
		return nil, errors.NewError(errors.EInvalid, "Invalid path")
	}

	// Get identity from DB
	identity, err := s.dbClient.ManagedIdentities.GetManagedIdentityByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if identity == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Managed identity with path %s not found", path))
	}

	if err := caller.RequireAccessToInheritedGroupResource(ctx, identity.GroupID); err != nil {
		return nil, err
	}

	return identity, nil
}

func (s *service) CreateManagedIdentity(ctx context.Context, input *types.CreateManagedIdentityInput) (*models.ManagedIdentity, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, input.GroupID, models.OwnerRole); err != nil {
		return nil, err
	}

	delegate, err := s.getDelegate(input.Type)
	if err != nil {
		return nil, err
	}

	managedIdentity := &models.ManagedIdentity{
		Type:        input.Type,
		Name:        input.Name,
		Description: input.Description,
		GroupID:     input.GroupID,
		CreatedBy:   caller.GetSubject(),
	}

	// Validate model
	if err = managedIdentity.Validate(); err != nil {
		return nil, err
	}

	s.logger.Infow("Requested to create a new managed identity.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"managedIdentityName", managedIdentity.Name,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateManagedIdentity: %v", txErr)
		}
	}()

	// Store identity in DB
	managedIdentity, err = s.dbClient.ManagedIdentities.CreateManagedIdentity(txContext, managedIdentity)
	if err != nil {
		return nil, err
	}

	if err = delegate.SetManagedIdentityData(ctx, managedIdentity, input.Data); err != nil {
		return nil, errors.NewError(errors.EInvalid, fmt.Sprintf("Failed to create managed identity: %v", err))
	}

	managedIdentity, err = s.dbClient.ManagedIdentities.UpdateManagedIdentity(txContext, managedIdentity)
	if err != nil {
		return nil, err
	}

	groupPath := managedIdentity.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      managedIdentity.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	// Store access rules
	if input.AccessRules != nil {
		for _, rule := range input.AccessRules {
			if err := s.verifyServiceAccountAccessForGroup(ctx, rule.AllowedServiceAccountIDs, managedIdentity.GetGroupPath()); err != nil {
				return nil, err
			}

			_, err := s.dbClient.ManagedIdentities.CreateManagedIdentityAccessRule(txContext,
				&models.ManagedIdentityAccessRule{
					ManagedIdentityID:        managedIdentity.Metadata.ID,
					RunStage:                 rule.RunStage,
					AllowedUserIDs:           rule.AllowedUserIDs,
					AllowedServiceAccountIDs: rule.AllowedServiceAccountIDs,
					AllowedTeamIDs:           rule.AllowedTeamIDs,
				})
			if err != nil {
				return nil, err
			}
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return managedIdentity, nil
}

func (s *service) GetManagedIdentitiesByIDs(ctx context.Context, ids []string) ([]models.ManagedIdentity, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get identity from DB
	results, err := s.dbClient.ManagedIdentities.GetManagedIdentities(ctx, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			ManagedIdentityIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, identity := range results.ManagedIdentities {
		if err := caller.RequireAccessToInheritedGroupResource(ctx, identity.GroupID); err != nil {
			return nil, err
		}
	}

	return results.ManagedIdentities, nil
}

func (s *service) UpdateManagedIdentity(ctx context.Context, input *types.UpdateManagedIdentityInput) (*models.ManagedIdentity, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, managedIdentity.GroupID, models.OwnerRole); err != nil {
		return nil, err
	}

	delegate, err := s.getDelegate(managedIdentity.Type)
	if err != nil {
		return nil, err
	}

	managedIdentity.Description = input.Description

	// Validate model
	if vErr := managedIdentity.Validate(); vErr != nil {
		return nil, vErr
	}

	if sErr := delegate.SetManagedIdentityData(ctx, managedIdentity, input.Data); sErr != nil {
		return nil, errors.NewError(errors.EInvalid, "Failed to create managed identity", errors.WithErrorErr(sErr))
	}

	s.logger.Infow("Updated a managed identity.",
		"caller", caller.GetSubject(),
		"groupID", managedIdentity.GroupID,
		"managedIdentityID", managedIdentity.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateManagedIdentity: %v", txErr)
		}
	}()

	// Store identity in DB
	updatedManagedIdentity, err := s.dbClient.ManagedIdentities.UpdateManagedIdentity(txContext, managedIdentity)
	if err != nil {
		return nil, err
	}

	groupPath := updatedManagedIdentity.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      updatedManagedIdentity.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedManagedIdentity, nil
}

func (s *service) GetManagedIdentityAccessRules(ctx context.Context, managedIdentity *models.ManagedIdentity) ([]models.ManagedIdentityAccessRule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToInheritedGroupResource(ctx, managedIdentity.GroupID); err != nil {
		return nil, err
	}

	resp, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &db.GetManagedIdentityAccessRulesInput{
		Filter: &db.ManagedIdentityAccessRuleFilter{
			ManagedIdentityID: &managedIdentity.Metadata.ID,
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.ManagedIdentityAccessRules, nil
}

func (s *service) GetManagedIdentityAccessRulesByIDs(ctx context.Context,
	ids []string) ([]models.ManagedIdentityAccessRule, error) {
	_, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Get identity from DB
	resp, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &db.GetManagedIdentityAccessRulesInput{
		Filter: &db.ManagedIdentityAccessRuleFilter{
			ManagedIdentityAccessRuleIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	// Get the identity IDs.
	identityIDMap := make(map[string]bool)
	identityIDs := []string{}
	for _, rule := range resp.ManagedIdentityAccessRules {
		identityID := rule.ManagedIdentityID
		if _, ok := identityIDMap[identityID]; !ok {
			identityIDMap[identityID] = true
			identityIDs = append(identityIDs, identityID)
		}
	}

	// Make sure caller has permission to see the affected groups.
	_, err = s.GetManagedIdentitiesByIDs(ctx, identityIDs)
	if err != nil {
		return nil, err
	}

	return resp.ManagedIdentityAccessRules, nil
}

func (s *service) GetManagedIdentityAccessRule(ctx context.Context, ruleID string) (*models.ManagedIdentityAccessRule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	rule, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRule(ctx, ruleID)
	if err != nil {
		return nil, err
	}

	if rule == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Managed identity access rule with ID %s not found", ruleID))
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, rule.ManagedIdentityID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToInheritedGroupResource(ctx, managedIdentity.GroupID); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *service) CreateManagedIdentityAccessRule(ctx context.Context, input *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ManagedIdentityID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, managedIdentity.GroupID, models.OwnerRole); err != nil {
		return nil, err
	}

	if err = s.verifyServiceAccountAccessForGroup(ctx, input.AllowedServiceAccountIDs, managedIdentity.GetGroupPath()); err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateManagedIdentityAccessRule: %v", txErr)
		}
	}()

	rule, err := s.dbClient.ManagedIdentities.CreateManagedIdentityAccessRule(txContext, input)
	if err != nil {
		return nil, err
	}

	groupPath := managedIdentity.GetGroupPath()

	// Activity events for creating managed identity access
	// rules point to the managed identity, not the rule.
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetManagedIdentityAccessRule,
			TargetID:      rule.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *service) UpdateManagedIdentityAccessRule(ctx context.Context, input *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ManagedIdentityID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToGroup(ctx, managedIdentity.GroupID, models.OwnerRole); err != nil {
		return nil, err
	}

	if err = s.verifyServiceAccountAccessForGroup(ctx, input.AllowedServiceAccountIDs, managedIdentity.GetGroupPath()); err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateManagedIdentityAccessRule: %v", txErr)
		}
	}()

	rule, err := s.dbClient.ManagedIdentities.UpdateManagedIdentityAccessRule(txContext, input)
	if err != nil {
		return nil, err
	}

	groupPath := managedIdentity.GetGroupPath()

	// Activity events for updating managed identity access rules point to the rule.
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetManagedIdentityAccessRule,
			TargetID:      rule.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return rule, nil
}

func (s *service) DeleteManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, rule.ManagedIdentityID)
	if err != nil {
		return err
	}

	if rErr := caller.RequireAccessToGroup(ctx, managedIdentity.GroupID, models.OwnerRole); rErr != nil {
		return rErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for DeleteManagedIdentityAccessRule: %v", txErr)
		}
	}()

	err = s.dbClient.ManagedIdentities.DeleteManagedIdentityAccessRule(txContext, rule)
	if err != nil {
		return err
	}

	groupPath := managedIdentity.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      managedIdentity.Metadata.ID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				ID:   rule.Metadata.ID,
				Name: string(rule.RunStage),
				Type: string(models.TargetManagedIdentityAccessRule),
			},
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateCredentials(ctx context.Context, identity *models.ManagedIdentity) ([]byte, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	jobCaller, ok := caller.(*auth.JobCaller)
	if !ok {
		return nil, errors.NewError(errors.EForbidden, "Only job callers can create managed identity credentials")
	}

	// Get Job
	job, err := s.jobService.GetJob(ctx, jobCaller.JobID)
	if err != nil {
		return nil, err
	}

	// Verify job is in a workspace that has access to this managed identity
	identitiesInWorkspace, err := s.GetManagedIdentitiesForWorkspace(ctx, job.WorkspaceID)
	if err != nil {
		return nil, err
	}

	found := false
	for _, mi := range identitiesInWorkspace {
		if identity.Metadata.ID == mi.Metadata.ID {
			found = true
			break
		}
	}

	if !found {
		return nil, errors.NewError(errors.EUnauthorized, fmt.Sprintf("Managed identity %s is not assigned to workspace %s", identity.Metadata.ID, job.WorkspaceID))
	}

	delegate, err := s.getDelegate(identity.Type)
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Created credentials for a managed identity.",
		"caller", caller.GetSubject(),
		"groupID", identity.GroupID,
		"managedIdentityID", identity.Metadata.ID,
	)

	return delegate.CreateCredentials(ctx, identity, job)
}

func (s *service) getDelegate(delegateType models.ManagedIdentityType) (Delegate, error) {
	delegate, ok := s.delegateMap[delegateType]
	if !ok {
		return nil, fmt.Errorf("managed identity with type %s is not supported", delegateType)
	}
	return delegate, nil
}

func (s *service) verifyServiceAccountAccessForGroup(ctx context.Context, serviceAccountIDs []string, groupPath string) error {
	for _, id := range serviceAccountIDs {
		sa, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, id)
		if err != nil {
			return err
		}

		saGroupPath := sa.GetGroupPath()

		if groupPath != saGroupPath && !strings.HasPrefix(groupPath, saGroupPath+"/") {
			return errors.NewError(errors.EInvalid, fmt.Sprintf("Service account %s is outside the scope of group %s", sa.ResourcePath, groupPath))
		}
	}
	return nil
}

func (s *service) getManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error) {
	// Get identity from DB
	identity, err := s.dbClient.ManagedIdentities.GetManagedIdentityByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if identity == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Managed identity with ID %s not found", id))
	}

	return identity, nil
}

// Helper function to determine if a resource path is invalid.
func isResourcePathInvalid(path string) bool {
	return strings.LastIndex(path, "/") == -1 ||
		strings.HasPrefix(path, "/") ||
		strings.HasSuffix(path, "/")
}
