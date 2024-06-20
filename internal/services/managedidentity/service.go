package managedidentity

import (
	"context"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetManagedIdentitiesInput is the input for listing managed identities
type GetManagedIdentitiesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.ManagedIdentitySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Search returns only the managed identities with a name or resource path that starts with the value of search
	Search *string
	// AliasSourceID is used to return aliases for a given managed identity
	AliasSourceID *string
	// NamespacePath is the namespace to return managed identities for
	NamespacePath string
	// IncludeInherited includes inherited managed identities in the result
	IncludeInherited bool
}

// DeleteManagedIdentityInput is the input for deleting a managed identity or alias.
type DeleteManagedIdentityInput struct {
	ManagedIdentity *models.ManagedIdentity
	Force           bool
}

// CreateManagedIdentityInput contains the fields for creating a new managed identity
type CreateManagedIdentityInput struct {
	Type        models.ManagedIdentityType
	Name        string
	Description string
	GroupID     string
	Data        []byte
	AccessRules []struct {
		Type                      models.ManagedIdentityAccessRuleType
		RunStage                  models.JobType
		ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
		AllowedUserIDs            []string
		AllowedServiceAccountIDs  []string
		AllowedTeamIDs            []string
		VerifyStateLineage        bool
	}
}

// UpdateManagedIdentityInput contains the fields for updating a managed identity
type UpdateManagedIdentityInput struct {
	ID          string
	Description string
	Data        []byte
}

// CreateManagedIdentityAliasInput is the input for creating a managed identity alias.
type CreateManagedIdentityAliasInput struct {
	Group         *models.Group
	Name          string
	AliasSourceID string
}

// MoveManagedIdentityInput is the input for moving a managed identity to a new group.
type MoveManagedIdentityInput struct {
	ManagedIdentityID string
	NewGroupID        string
}

// Service implements managed identity functionality
type Service interface {
	GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error)
	GetManagedIdentityByPath(ctx context.Context, path string) (*models.ManagedIdentity, error)
	GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*db.ManagedIdentitiesResult, error)
	GetManagedIdentitiesByIDs(ctx context.Context, ids []string) ([]models.ManagedIdentity, error)
	CreateManagedIdentity(ctx context.Context, input *CreateManagedIdentityInput) (*models.ManagedIdentity, error)
	UpdateManagedIdentity(ctx context.Context, input *UpdateManagedIdentityInput) (*models.ManagedIdentity, error)
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
	CreateManagedIdentityAlias(ctx context.Context, input *CreateManagedIdentityAliasInput) (*models.ManagedIdentity, error)
	DeleteManagedIdentityAlias(ctx context.Context, input *DeleteManagedIdentityInput) error
	MoveManagedIdentity(ctx context.Context, input *MoveManagedIdentityInput) (*models.ManagedIdentity, error)
}

type service struct {
	logger           logger.Logger
	dbClient         *db.Client
	limitChecker     limits.LimitChecker
	delegateMap      map[models.ManagedIdentityType]Delegate
	workspaceService workspace.Service
	jobService       job.Service
	activityService  activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	managedIdentityDelegateMap map[models.ManagedIdentityType]Delegate,
	workspaceService workspace.Service,
	jobService job.Service,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:           logger,
		dbClient:         dbClient,
		limitChecker:     limitChecker,
		delegateMap:      managedIdentityDelegateMap,
		workspaceService: workspaceService,
		jobService:       jobService,
		activityService:  activityService,
	}
}

func (s *service) GetManagedIdentities(ctx context.Context, input *GetManagedIdentitiesInput) (*db.ManagedIdentitiesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentities")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.NamespacePath != "" {
		if err = caller.RequirePermission(ctx, permissions.ViewManagedIdentityPermission, auth.WithNamespacePath(input.NamespacePath)); err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else if input.AliasSourceID != nil {
		sourceIdentity, gErr := s.getManagedIdentityByID(ctx, *input.AliasSourceID)
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get managed identity by ID")
			return nil, gErr
		}

		if err = caller.RequireAccessToInheritableResource(ctx, permissions.ManagedIdentityResourceType, auth.WithGroupID(sourceIdentity.GroupID)); err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	} else {
		return nil, errors.New("Either NamespacePath or AliasSourceID must be defined", errors.WithErrorCode(errors.EInvalid))
	}

	filter := &db.ManagedIdentityFilter{
		Search:        input.Search,
		AliasSourceID: input.AliasSourceID,
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
	} else if input.NamespacePath != "" {
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
		tracing.RecordError(span, err, "failed to get managed identities")
		return nil, err
	}

	return result, nil
}

func (s *service) DeleteManagedIdentity(ctx context.Context, input *DeleteManagedIdentityInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	// Don't allow deleting an alias.
	if input.ManagedIdentity.IsAlias() {
		return errors.New("Only a source managed identity can be deleted, not an alias", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, permissions.DeleteManagedIdentityPermission, auth.WithGroupID(input.ManagedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
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
			tracing.RecordError(span, wErr, "failed to get workspaces for managed identity")
			return wErr
		}
		if len(workspaces) > 0 {
			return errors.New(
				"This managed identity can't be deleted because it's currently assigned to %d workspaces. "+
					"Setting force to true will automatically remove this managed identity from all workspaces it's assigned to.", len(workspaces),
				errors.WithErrorCode(errors.EConflict),
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteManagedIdentity: %v", txErr)
		}
	}()

	err = s.dbClient.ManagedIdentities.DeleteManagedIdentity(txContext, input.ManagedIdentity)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete managed identity")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetManagedIdentitiesForWorkspace(ctx context.Context, workspaceID string) ([]models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentitiesForWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewManagedIdentityPermission, auth.WithWorkspaceID(workspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	identities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identities for workspace")
		return nil, err
	}

	return identities, nil
}

func (s *service) AddManagedIdentityToWorkspace(ctx context.Context, managedIdentityID string, workspaceID string) error {
	ctx, span := tracer.Start(ctx, "svc.AddManagedIdentityToWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Get managed identity that will be added
	identity, err := s.getManagedIdentityByID(ctx, managedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return err
	}

	// Get workspace
	workspace, err := s.workspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by ID")
		return err
	}

	resourcePathParts := strings.Split(identity.ResourcePath, "/")
	groupPath := strings.Join(resourcePathParts[:len(resourcePathParts)-1], "/")

	// Verify that the managed identity's group is in the group hierarchy of the workspace
	if !workspace.IsDescendantOfGroup(groupPath) {
		return errors.New("managed identity %s is not available to workspace %s", managedIdentityID, workspaceID, errors.WithErrorCode(errors.EInvalid))
	}

	identitiesInWorkspace, err := s.GetManagedIdentitiesForWorkspace(ctx, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identities for workspace")
		return err
	}

	// Verify that only one type of each managed identity can be assigned at a time
	for _, mi := range identitiesInWorkspace {
		if mi.Type == identity.Type {
			return errors.New("managed identity with type %s already assigned to workspace %s", identity.Type, workspaceID, errors.WithErrorCode(errors.EInvalid))
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer AddManagedIdentityToWorkspace: %v", txErr)
		}
	}()

	if aErr := s.dbClient.ManagedIdentities.AddManagedIdentityToWorkspace(txContext,
		managedIdentityID, workspaceID); aErr != nil {
		tracing.RecordError(span, aErr, "failed to add managed identity to workspace")
		return aErr
	}

	// Get the number of managed identities assigned to a workspace to check whether we just violated the limit.
	newManagedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(txContext, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace's managed identities")
		return err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitAssignedManagedIdentitiesPerWorkspace, int32(len(newManagedIdentities))); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return err
	}

	// Must check the group ID of the managed identity to make sure the managed identity did not get moved.
	newManagedIdentity, err := s.dbClient.ManagedIdentities.GetManagedIdentityByID(ctx, managedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get updated managed identity")
		return err
	}
	if newManagedIdentity.GroupID != identity.GroupID {
		return errors.New("managed identity was moved while adding to workspace", errors.WithErrorCode(errors.EConflict))
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionAdd,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      identity.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.RemoveManagedIdentityFromWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Get managed identity that will be removed
	identity, err := s.getManagedIdentityByID(ctx, managedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return err
	}

	// Get workspace
	workspace, err := s.workspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by ID")
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer RemoveManagedIdentityFromWorkspace: %v", txErr)
		}
	}()

	if err = s.dbClient.ManagedIdentities.RemoveManagedIdentityFromWorkspace(txContext,
		managedIdentityID, workspaceID); err != nil {
		tracing.RecordError(span, err, "failed to remove managed identity from workspace")
		return err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionRemove,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      identity.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	s.logger.Infow("Removed a managed identity from workspace.",
		"caller", caller.GetSubject(),
		"workspaceID", workspaceID,
		"managedIdentityID", managedIdentityID,
	)
	return nil
}

func (s *service) GetManagedIdentityByID(ctx context.Context, id string) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentityByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get identity from DB
	identity, err := s.getManagedIdentityByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.ManagedIdentityResourceType, auth.WithGroupID(identity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource access check failed")
		return nil, err
	}

	return identity, nil
}

func (s *service) GetManagedIdentityByPath(ctx context.Context, path string) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentityByPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if isResourcePathInvalid(path) {
		return nil, errors.New("Invalid path", errors.WithErrorCode(errors.EInvalid))
	}

	// Get identity from DB
	identity, err := s.dbClient.ManagedIdentities.GetManagedIdentityByPath(ctx, path)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by path")
		return nil, err
	}

	if identity == nil {
		return nil, errors.New("managed identity with path %s not found", path, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.ManagedIdentityResourceType, auth.WithGroupID(identity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource access check failed")
		return nil, err
	}

	return identity, nil
}

func (s *service) CreateManagedIdentityAlias(ctx context.Context, input *CreateManagedIdentityAliasInput) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateManagedIdentityAlias")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Require permissions for target group (group being shared to).
	err = caller.RequirePermission(ctx, permissions.CreateManagedIdentityPermission, auth.WithGroupID(input.Group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	aliasSourceIdentity, err := s.getManagedIdentityByID(ctx, input.AliasSourceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return nil, err
	}

	// Make sure an alias isn't being aliased.
	if aliasSourceIdentity.IsAlias() {
		return nil, errors.New("An alias managed identity must not be created from another alias", errors.WithErrorCode(errors.EInvalid))
	}

	sourceGroup, err := s.dbClient.Groups.GetGroupByID(ctx, aliasSourceIdentity.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by ID")
		return nil, err
	}

	// Shouldn't happen.
	if sourceGroup == nil {
		return nil, errors.New("group associated with managed identity ID %s not found", aliasSourceIdentity.Metadata.ID)
	}

	// Require permissions for source group (group source managed identity belongs to).
	err = caller.RequirePermission(ctx, permissions.CreateManagedIdentityPermission, auth.WithGroupID(input.Group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Verify managed identity isn't being aliased within same namespace it's already available in.
	if input.Group.IsDescendantOfGroup(sourceGroup.FullPath) || input.Group.FullPath == sourceGroup.FullPath {
		return nil, errors.New("source managed identity %s is already available within namespace", aliasSourceIdentity.Name, errors.WithErrorCode(errors.EInvalid))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateMangedIdentityAlias: %v", txErr)
		}
	}()

	toCreate := &models.ManagedIdentity{
		GroupID:       input.Group.Metadata.ID,
		AliasSourceID: &aliasSourceIdentity.Metadata.ID,
		Name:          input.Name,
		CreatedBy:     caller.GetSubject(),
	}

	if err = toCreate.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate managed identity model to create")
		return nil, err
	}

	createdAlias, err := s.dbClient.ManagedIdentities.CreateManagedIdentity(txContext, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create managed identity")
		return nil, err
	}

	groupPath := createdAlias.GetGroupPath()

	// Get the number of managed identities in the group to check whether we just violated the limit.
	newManagedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentities(txContext, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			NamespacePaths: []string{groupPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's managed identities")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitManagedIdentitiesPerGroup, newManagedIdentities.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	// Get the number of aliases for the source managed identity to check whether we just violated the limit.
	newAliases, err := s.dbClient.ManagedIdentities.GetManagedIdentities(txContext, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			AliasSourceID: createdAlias.AliasSourceID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity's aliases")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitManagedIdentityAliasesPerManagedIdentity, newAliases.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      createdAlias.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a managed identity alias.",
		"caller", caller.GetSubject(),
		"groupID", input.Group.Metadata.ID,
		"aliasID", createdAlias.Metadata.ID,
	)

	return createdAlias, nil
}

func (s *service) DeleteManagedIdentityAlias(ctx context.Context, input *DeleteManagedIdentityInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteManagedIdentityAlias")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	// Only allow deleting managed identity aliases.
	if !input.ManagedIdentity.IsAlias() {
		return errors.New("Only an alias may be deleted, not a source managed identity", errors.WithErrorCode(errors.EInvalid))
	}

	// First check whether they have permissions for alias' group.
	perm := permissions.DeleteManagedIdentityPermission
	if err = caller.RequirePermission(ctx, perm, auth.WithGroupID(input.ManagedIdentity.GroupID)); err != nil {
		aliasSource, gErr := s.getManagedIdentityByID(ctx, *input.ManagedIdentity.AliasSourceID)
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get managed identity by ID")
			return gErr
		}

		// Now check if they have permissions in group of the source managed identity.
		if err = caller.RequirePermission(ctx, perm, auth.WithGroupID(aliasSource.GroupID)); err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return err
		}

	}

	s.logger.Infow("Requested to delete a managed identity alias.",
		"caller", caller.GetSubject(),
		"groupID", input.ManagedIdentity.GroupID,
		"aliasID", input.ManagedIdentity.Metadata.ID,
	)

	if !input.Force {
		// Verify that managed identity alias is not assigned to any workspaces
		workspaces, wErr := s.dbClient.Workspaces.GetWorkspacesForManagedIdentity(ctx, input.ManagedIdentity.Metadata.ID)
		if wErr != nil {
			tracing.RecordError(span, wErr, "failed to get workspaces for managed identity")
			return wErr
		}
		if len(workspaces) > 0 {
			return errors.New(
				"This managed identity alias can't be deleted because it's currently assigned to %d workspaces. "+
					"Setting force to true will automatically remove this managed identity alias from all workspaces it's assigned to.", len(workspaces),
				errors.WithErrorCode(errors.EConflict),
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteManagedIdentityAlias: %v", txErr)
		}
	}()

	err = s.dbClient.ManagedIdentities.DeleteManagedIdentity(txContext, input.ManagedIdentity)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete managed identity")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateManagedIdentity(ctx context.Context, input *CreateManagedIdentityInput) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateManagedIdentityPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	delegate, err := s.getDelegate(input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get delegate")
		return nil, err
	}

	managedIdentity := &models.ManagedIdentity{
		Type:        input.Type,
		Name:        input.Name,
		Description: input.Description,
		GroupID:     input.GroupID,
		CreatedBy:   caller.GetSubject(),
		Data:        []byte{}, // Required or identity will fail to create.
	}

	// Validate model
	if err = managedIdentity.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate managed identity model")
		return nil, err
	}

	s.logger.Infow("Requested to create a new managed identity.",
		"caller", caller.GetSubject(),
		"groupID", input.GroupID,
		"managedIdentityName", managedIdentity.Name,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to create managed identity")
		return nil, err
	}

	if err = delegate.SetManagedIdentityData(txContext, managedIdentity, input.Data); err != nil {
		tracing.RecordError(span, err, "failed to set managed identity data")
		return nil, errors.Wrap(err, "failed to set managed identity data", errors.WithErrorCode(errors.EInvalid))
	}

	managedIdentity, err = s.dbClient.ManagedIdentities.UpdateManagedIdentity(txContext, managedIdentity)
	if err != nil {
		tracing.RecordError(span, err, "failed to update managed identity")
		return nil, err
	}

	groupPath := managedIdentity.GetGroupPath()

	// Get the number of managed identities in the group to check whether we just violated the limit.
	newManagedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentities(txContext, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			NamespacePaths: []string{groupPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's managed identities")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitManagedIdentitiesPerGroup, newManagedIdentities.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      managedIdentity.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	// Store access rules
	if input.AccessRules != nil {
		for _, rule := range input.AccessRules {
			if err = s.verifyServiceAccountAccessForGroup(ctx, rule.AllowedServiceAccountIDs, managedIdentity.GetGroupPath()); err != nil {
				tracing.RecordError(span, err, "failed to verify service access for group")
				return nil, err
			}

			ruleToCreate := models.ManagedIdentityAccessRule{
				Type:                      rule.Type,
				ManagedIdentityID:         managedIdentity.Metadata.ID,
				RunStage:                  rule.RunStage,
				ModuleAttestationPolicies: rule.ModuleAttestationPolicies,
				AllowedUserIDs:            rule.AllowedUserIDs,
				AllowedServiceAccountIDs:  rule.AllowedServiceAccountIDs,
				AllowedTeamIDs:            rule.AllowedTeamIDs,
				VerifyStateLineage:        rule.VerifyStateLineage,
			}

			if err = ruleToCreate.Validate(); err != nil {
				tracing.RecordError(span, err, "failed to validate managed identity access rule model to create")
				return nil, err
			}

			_, err := s.dbClient.ManagedIdentities.CreateManagedIdentityAccessRule(txContext, &ruleToCreate)
			if err != nil {
				tracing.RecordError(span, err, "failed to create managed identity access rule")
				return nil, err
			}
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return managedIdentity, nil
}

func (s *service) GetManagedIdentitiesByIDs(ctx context.Context, ids []string) ([]models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentitiesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get identity from DB
	results, err := s.dbClient.ManagedIdentities.GetManagedIdentities(ctx, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			ManagedIdentityIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identities")
		return nil, err
	}

	namespacePaths := []string{}
	for _, identity := range results.ManagedIdentities {
		namespacePaths = append(namespacePaths, identity.GetGroupPath())
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.ManagedIdentityResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return results.ManagedIdentities, nil
}

func (s *service) UpdateManagedIdentity(ctx context.Context, input *UpdateManagedIdentityInput) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateManagedIdentity")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return nil, err
	}

	// Don't allow updates to a managed identity alias.
	if managedIdentity.IsAlias() {
		return nil, errors.New("Only a source managed identity can be updated, not an alias", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, permissions.UpdateManagedIdentityPermission, auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	delegate, err := s.getDelegate(managedIdentity.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get delegate")
		return nil, err
	}

	managedIdentity.Description = input.Description

	// Validate model
	if vErr := managedIdentity.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate managed identity model to update")
		return nil, vErr
	}

	if sErr := delegate.SetManagedIdentityData(ctx, managedIdentity, input.Data); sErr != nil {
		tracing.RecordError(span, err, "failed to set managed identity date")
		return nil, errors.Wrap(err, "failed to set managed identity data", errors.WithErrorCode(errors.EInvalid))
	}

	s.logger.Infow("Updated a managed identity.",
		"caller", caller.GetSubject(),
		"groupID", managedIdentity.GroupID,
		"managedIdentityID", managedIdentity.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to update managed identity")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedManagedIdentity, nil
}

func (s *service) GetManagedIdentityAccessRules(ctx context.Context, managedIdentity *models.ManagedIdentity) ([]models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentityAccessRules")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.ManagedIdentityResourceType, auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource access check failed")
		return nil, err
	}

	resp, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &db.GetManagedIdentityAccessRulesInput{
		Filter: &db.ManagedIdentityAccessRuleFilter{
			ManagedIdentityID: &managedIdentity.Metadata.ID,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity access rules")
		return nil, err
	}

	return resp.ManagedIdentityAccessRules, nil
}

func (s *service) GetManagedIdentityAccessRulesByIDs(ctx context.Context,
	ids []string) ([]models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentityAccessRulesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	_, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Get identity from DB
	resp, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &db.GetManagedIdentityAccessRulesInput{
		Filter: &db.ManagedIdentityAccessRuleFilter{
			ManagedIdentityAccessRuleIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity access rules")
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
		tracing.RecordError(span, err, "failed to get managed identities by IDs")
		return nil, err
	}

	return resp.ManagedIdentityAccessRules, nil
}

func (s *service) GetManagedIdentityAccessRule(ctx context.Context, ruleID string) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "svc.GetManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	rule, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRule(ctx, ruleID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity access rule")
		return nil, err
	}

	if rule == nil {
		return nil, errors.New("managed identity access rule with ID %s not found", ruleID, errors.WithErrorCode(errors.ENotFound))
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, rule.ManagedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.ManagedIdentityResourceType, auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "inheritable resource access check failed")
		return nil, err
	}

	return rule, nil
}

func (s *service) CreateManagedIdentityAccessRule(ctx context.Context, input *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if err = input.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate managed identity access rule model")
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ManagedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return nil, err
	}

	// Don't allow creating access rules for an aliased identity.
	if managedIdentity.IsAlias() {
		return nil, errors.New("Access rules can be created only for source managed identities, not for aliases", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, permissions.UpdateManagedIdentityPermission, auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if err = s.verifyServiceAccountAccessForGroup(ctx, input.AllowedServiceAccountIDs, managedIdentity.GetGroupPath()); err != nil {
		tracing.RecordError(span, err, "group service account access check failed")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateManagedIdentityAccessRule: %v", txErr)
		}
	}()

	rule, err := s.dbClient.ManagedIdentities.CreateManagedIdentityAccessRule(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to create managed identity access rule")
		return nil, err
	}

	// Get the number of access rules in the managed identity to check whether we just violated the limit.
	newAccessRules, err := s.dbClient.ManagedIdentities.GetManagedIdentityAccessRules(txContext,
		&db.GetManagedIdentityAccessRulesInput{
			Filter: &db.ManagedIdentityAccessRuleFilter{
				ManagedIdentityID: &rule.ManagedIdentityID,
			},
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(0),
			},
		})
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity's access rules")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitManagedIdentityAccessRulesPerManagedIdentity, newAccessRules.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return rule, nil
}

func (s *service) UpdateManagedIdentityAccessRule(ctx context.Context, input *models.ManagedIdentityAccessRule) (*models.ManagedIdentityAccessRule, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if err = input.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate managed identity access rule model")
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ManagedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return nil, err
	}

	// Don't allow updating access rules for managed identity aliases.
	if managedIdentity.IsAlias() {
		return nil, errors.New("Access rules can be updated only for source managed identities, not for aliases", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, permissions.UpdateManagedIdentityPermission, auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if err = s.verifyServiceAccountAccessForGroup(ctx, input.AllowedServiceAccountIDs, managedIdentity.GetGroupPath()); err != nil {
		tracing.RecordError(span, err, "group service account access check failed")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateManagedIdentityAccessRule: %v", txErr)
		}
	}()

	rule, err := s.dbClient.ManagedIdentities.UpdateManagedIdentityAccessRule(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to update managed identity access rule")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return rule, nil
}

func (s *service) DeleteManagedIdentityAccessRule(ctx context.Context, rule *models.ManagedIdentityAccessRule) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteManagedIdentityAccessRule")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, rule.ManagedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity by ID")
		return err
	}

	// Don't allow access rule deletion for aliases.
	if managedIdentity.IsAlias() {
		return errors.New("Access rules can be deleted only for source managed identities, not for aliases", errors.WithErrorCode(errors.EInvalid))
	}

	err = caller.RequirePermission(ctx, permissions.UpdateManagedIdentityPermission, auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for DeleteManagedIdentityAccessRule: %v", txErr)
		}
	}()

	err = s.dbClient.ManagedIdentities.DeleteManagedIdentityAccessRule(txContext, rule)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete managed identity access rule")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateCredentials(ctx context.Context, identity *models.ManagedIdentity) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateCredentials")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	jobCaller, ok := caller.(*auth.JobCaller)
	if !ok {
		return nil, errors.New("Only job callers can create managed identity credentials", errors.WithErrorCode(errors.EForbidden))
	}

	// Get Job
	job, err := s.jobService.GetJob(ctx, jobCaller.JobID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get job")
		return nil, err
	}

	// Verify job is in a workspace that has access to this managed identity
	identitiesInWorkspace, err := s.GetManagedIdentitiesForWorkspace(ctx, job.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identities for workspace")
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
		return nil, errors.New("managed identity %s is not assigned to workspace %s", identity.Metadata.ID, job.WorkspaceID, errors.WithErrorCode(errors.EUnauthorized))
	}

	delegate, err := s.getDelegate(identity.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to get delegate")
		return nil, err
	}

	s.logger.Infow("Created credentials for a managed identity.",
		"caller", caller.GetSubject(),
		"groupID", identity.GroupID,
		"managedIdentityID", identity.Metadata.ID,
	)

	return delegate.CreateCredentials(ctx, identity, job)
}

func (s *service) MoveManagedIdentity(ctx context.Context, input *MoveManagedIdentityInput) (*models.ManagedIdentity, error) {
	ctx, span := tracer.Start(ctx, "svc.MoveManagedIdentity")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	managedIdentity, err := s.getManagedIdentityByID(ctx, input.ManagedIdentityID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identity")
		return nil, err
	}

	// Caller must be an owner of both the old group and the new group.
	err = caller.RequirePermission(ctx, permissions.DeleteManagedIdentityPermission,
		auth.WithGroupID(managedIdentity.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}
	err = caller.RequirePermission(ctx, permissions.CreateManagedIdentityPermission,
		auth.WithGroupID(input.NewGroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Only non-aliases are allowed to be moved.
	if managedIdentity.IsAlias() {
		return nil, errors.New("Only a source managed identity can be moved, not an alias", errors.WithErrorCode(errors.EInvalid))
	}

	newGroup, err := s.dbClient.Groups.GetGroupByID(ctx, input.NewGroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get new group")
		return nil, err
	}

	if newGroup == nil {
		return nil, errors.New("group with id %s not found", input.NewGroupID, errors.WithErrorCode(errors.ENotFound))
	}

	// Check to ensure there are no aliases of the managed identity in the new group or certain related groups.
	// This is to prevent a situation where a managed identity is moved to a group that contains an alias of itself.
	err = s.checkDisallowedAliases(ctx, managedIdentity, newGroup)
	if err != nil {
		return nil, err
	}

	// Check to ensure there are no assignments of any relevant managed identity to a workspace.
	// If there are any, list the workspaces in the error message.
	err = s.checkWorkspaceAssignments(ctx, managedIdentity, newGroup)
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Requested to move a managed identity.",
		"caller", caller.GetSubject(),
		"managedIdentityName", managedIdentity.Name,
		"groupPath", newGroup.FullPath,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for MoveManagedIdentity: %v", txErr)
		}
	}()

	oldGroupPath := managedIdentity.GetGroupPath()

	// Record the move in the DB.
	managedIdentity.GroupID = input.NewGroupID
	managedIdentity, err = s.dbClient.ManagedIdentities.UpdateManagedIdentity(txContext, managedIdentity)
	if err != nil {
		tracing.RecordError(span, err, "failed to move managed identity")
		return nil, err
	}

	// Get the number of managed identities now in the new group to check whether we just violated the limit.
	newManagedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentities(txContext, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			NamespacePaths: []string{newGroup.FullPath},
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's managed identities")
		return nil, err
	}

	// Check the resource limit.
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitManagedIdentitiesPerGroup, newManagedIdentities.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &newGroup.FullPath,
			Action:        models.ActionMigrate,
			TargetType:    models.TargetManagedIdentity,
			TargetID:      managedIdentity.Metadata.ID,
			Payload: &models.ActivityEventMoveManagedIdentityPayload{
				PreviousGroupPath: oldGroupPath,
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return managedIdentity, nil
}

// Check to ensure there are no aliases of the managed identity in the new group or certain related groups.
// Related groups include descendants of the target group and all ancestors of the target group.
// This is to prevent a situation where a managed identity is moved to a group that contains an alias of itself.
func (s *service) checkDisallowedAliases(ctx context.Context,
	managedIdentity *models.ManagedIdentity, targetGroup *models.Group) error {

	aliases, err := s.dbClient.ManagedIdentities.GetManagedIdentities(ctx, &db.GetManagedIdentitiesInput{
		Filter: &db.ManagedIdentityFilter{
			AliasSourceID: &managedIdentity.Metadata.ID,
		},
	})
	if err != nil {
		return err
	}

	for _, alias := range aliases.ManagedIdentities {

		// If the alias is in the target group, then it's a problem.
		if alias.GroupID == targetGroup.Metadata.ID {
			return errors.New("managed identity %s is an alias of managed identity %s, which is in the target group %s",
				alias.ResourcePath, managedIdentity.ResourcePath, targetGroup.FullPath, errors.WithErrorCode(errors.EInvalid))
		}

		// If the alias is in a descendant of the target group, then it's a problem.
		if models.IsDescendantOfPath(alias.GetGroupPath(), targetGroup.FullPath) {
			return errors.New("managed identity %s is an alias of managed identity %s, which is in a descendant group of the target group %s",
				alias.ResourcePath, managedIdentity.ResourcePath, targetGroup.FullPath, errors.WithErrorCode(errors.EInvalid))
		}

		// If the alias is in an ancestor of the target group, then it's a problem.
		if targetGroup.IsDescendantOfGroup(alias.GetGroupPath()) {
			return errors.New("managed identity %s is an alias of managed identity %s, which is in an ancestor group of the target group %s",
				alias.ResourcePath, managedIdentity.ResourcePath, targetGroup.FullPath, errors.WithErrorCode(errors.EInvalid))
		}
	}

	// If nothing was found, then we're good.
	return nil
}

func (s *service) checkWorkspaceAssignments(ctx context.Context,
	managedIdentity *models.ManagedIdentity, newGroup *models.Group) error {

	workspaces, err := s.dbClient.Workspaces.GetWorkspacesForManagedIdentity(ctx, managedIdentity.Metadata.ID)
	if err != nil {
		return err
	}

	badPaths := []string{}

	for _, workspace := range workspaces {
		if !workspace.IsDescendantOfGroup(newGroup.FullPath) {
			badPaths = append(badPaths, workspace.FullPath)
		}
	}

	if len(badPaths) > 0 {
		return errors.New("managed identity %s is assigned to workspaces %s, which are outside the target group %s",
			managedIdentity.ResourcePath, strings.Join(badPaths, ", "), newGroup.FullPath, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

func (s *service) getDelegate(delegateType models.ManagedIdentityType) (Delegate, error) {
	delegate, ok := s.delegateMap[delegateType]
	if !ok {
		return nil, errors.New("managed identity with type %s is not supported", delegateType, errors.WithErrorCode(errors.EInvalid))
	}
	return delegate, nil
}

func (s *service) verifyServiceAccountAccessForGroup(ctx context.Context, serviceAccountIDs []string, groupPath string) error {
	for _, id := range serviceAccountIDs {
		sa, err := s.dbClient.ServiceAccounts.GetServiceAccountByID(ctx, id)
		if err != nil {
			return err
		}

		if sa == nil {
			return errors.New("service account with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
		}

		saGroupPath := sa.GetGroupPath()

		if groupPath != saGroupPath && !models.IsDescendantOfPath(groupPath, saGroupPath) {
			return errors.New("service account %s is outside the scope of group %s", sa.ResourcePath, groupPath, errors.WithErrorCode(errors.EInvalid))
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
		return nil, errors.New("managed identity with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	return identity, nil
}

// Helper function to determine if a resource path is invalid.
func isResourcePathInvalid(path string) bool {
	return strings.LastIndex(path, "/") == -1 ||
		strings.HasPrefix(path, "/") ||
		strings.HasSuffix(path, "/")
}
