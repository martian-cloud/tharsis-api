package workspace

//go:generate mockery --name Service --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// defaultMaxJobDuration is the default amount of minutes a job can run for before being gracefully cancelled.
	defaultMaxJobDuration = time.Hour * 12

	// upperLimitMaxJobDuration is the highest duration in minutes MaxJobDuration field can be assigned.
	upperLimitMaxJobDuration = time.Hour * 24

	// lowerLimitMaxJobDuration is the lowest value MaxJobDuration field can be assigned.
	lowerLimitMaxJobDuration = time.Minute

	tharsisTerraformProviderConfig        = "provider[\"registry.terraform.io/martian-cloud/tharsis\"]"
	tharsisWorkspaceOutputsDatasourceName = "tharsis_workspace_outputs"
)

// These error messages must be translated to TFE equivalent by caller.
var (
	// Error returned when workspace is already locked.
	WorkspaceLockedError = errors.NewError(errors.EConflict, "workspace locked")

	// Error returned when workspace is already unlocked.
	WorkspaceUnlockedError = errors.NewError(errors.EConflict, "workspace unlocked")

	// Error returned when a workspace unlock is attempted but it's locked by a run.
	WorkspaceLockedByRunError = errors.NewError(errors.EConflict, "workspace locked by run")
)

// Event represents a workspace event
type Event struct {
	Action    string
	Workspace models.Workspace
}

// EventSubscriptionOptions provides options for subscribing to workspace events
type EventSubscriptionOptions struct {
	WorkspaceID string
}

// StateVersionResource represents a resource from a workspace state version
type StateVersionResource struct {
	Module   string
	Mode     string
	Type     string
	Name     string
	Provider string
}

// StateVersionDependency represents a workspace dependency
type StateVersionDependency struct {
	WorkspacePath  string
	WorkspaceID    string
	StateVersionID string
}

// GetWorkspacesInput is the input for querying a list of workspaces
type GetWorkspacesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.WorkspaceSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Group filters the workspaces by the specified group
	Group *models.Group
	// Search is used to search for a workspace by name or namespace path
	Search *string
}

// GetStateVersionsInput is the input for querying a list of state versions
type GetStateVersionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.StateVersionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Workspace filters state versions by the specified workspace
	Workspace *models.Workspace
}

// CreateConfigurationVersionInput is the input for creating a new configuration version
type CreateConfigurationVersionInput struct {
	VCSEventID  *string
	WorkspaceID string
	Speculative bool
}

// Service implements all workspace related functionality
type Service interface {
	SubscribeToWorkspaceEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error)
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	GetWorkspaceByFullPath(ctx context.Context, path string) (*models.Workspace, error)
	GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*db.WorkspacesResult, error)
	GetWorkspacesByIDs(ctx context.Context, idList []string) ([]models.Workspace, error)
	CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	DeleteWorkspace(ctx context.Context, workspace *models.Workspace, force bool) error
	LockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	UnlockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	GetCurrentStateVersion(ctx context.Context, workspaceID string) (*models.StateVersion, error)
	CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion, data *string) (*models.StateVersion, error)
	GetStateVersion(ctx context.Context, stateVersionID string) (*models.StateVersion, error)
	GetStateVersions(ctx context.Context, input *GetStateVersionsInput) (*db.StateVersionsResult, error)
	GetStateVersionContent(ctx context.Context, stateVersionID string) (io.ReadCloser, error)
	GetStateVersionsByIDs(ctx context.Context, idList []string) ([]models.StateVersion, error)
	CreateConfigurationVersion(ctx context.Context, options *CreateConfigurationVersionInput) (*models.ConfigurationVersion, error)
	GetConfigurationVersion(ctx context.Context, configurationVersionID string) (*models.ConfigurationVersion, error)
	UploadConfigurationVersion(ctx context.Context, configurationVersionID string, reader io.Reader) error
	GetConfigurationVersionContent(ctx context.Context, configurationVersionID string) (io.ReadCloser, error)
	GetConfigurationVersionsByIDs(ctx context.Context, idList []string) ([]models.ConfigurationVersion, error)
	GetStateVersionOutputs(context context.Context, stateVersionID string) ([]models.StateVersionOutput, error)
	GetStateVersionResources(ctx context.Context, stateVersion *models.StateVersion) ([]StateVersionResource, error)
	GetStateVersionDependencies(ctx context.Context, stateVersion *models.StateVersion) ([]StateVersionDependency, error)
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	artifactStore   ArtifactStore
	eventManager    *events.EventManager
	cliService      cli.Service
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	artifactStore ArtifactStore,
	eventManager *events.EventManager,
	cliService cli.Service,
	activityService activityevent.Service,
) Service {
	return &service{
		logger,
		dbClient,
		artifactStore,
		eventManager,
		cliService,
		activityService,
	}
}

func (s *service) SubscribeToWorkspaceEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *Event)

	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)

		subscription := events.Subscription{
			Type: events.WorkspaceSubscription,
			ID:   options.WorkspaceID, // Subscribe to specific workspace ID
			Actions: []events.SubscriptionAction{
				events.CreateAction,
				events.UpdateAction,
			},
		}
		subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

		defer s.eventManager.Unsubscribe(subscriber)

		// Wait for workspace updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if err != context.Canceled {
					s.logger.Errorf("Error occurred while waiting for workspace events: %v", err)
				}
				return
			}

			ws, err := s.getWorkspaceByID(ctx, event.ID)
			if err != nil {
				s.logger.Errorf("Error occurred while querying for workspace associated with workspace event %s: %v", event.ID, err)
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &Event{Action: event.Action, Workspace: *ws}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) GetWorkspacesByIDs(ctx context.Context, idList []string) ([]models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{WorkspaceIDs: idList}})
	if err != nil {
		return nil, err
	}

	wsPaths := []string{}
	for _, ws := range resp.Workspaces {
		wsPaths = append(wsPaths, ws.FullPath)
	}

	// Verify caller has access to all returned workspaces.
	if len(wsPaths) > 0 {
		err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithNamespacePaths(wsPaths))
		if err != nil {
			return nil, err
		}
	}

	return resp.Workspaces, nil
}

func (s *service) GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*db.WorkspacesResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	dbInput := db.GetWorkspacesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.WorkspaceFilter{
			Search: input.Search,
		},
	}

	if input.Group != nil {
		err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			return nil, err
		}
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			return nil, napErr
		}

		if !policy.AllowAll {
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

	workspacesResult, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &dbInput)
	if err != nil {
		return nil, err
	}

	return workspacesResult, nil
}

func (s *service) GetWorkspaceByFullPath(ctx context.Context, path string) (*models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithNamespacePath(path))
	if err != nil {
		return nil, err
	}

	workspace, err := s.dbClient.Workspaces.GetWorkspaceByFullPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if workspace == nil {
		return nil, errors.NewError(
			errors.ENotFound,
			fmt.Sprintf("Workspace with path %s not found", path),
		)
	}

	return workspace, nil
}

func (s *service) GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	workspace, err := s.getWorkspaceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *service) DeleteWorkspace(ctx context.Context, workspace *models.Workspace, force bool) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return err
	}

	if !force && workspace.CurrentStateVersionID != "" {
		sv, gErr := s.GetStateVersion(ctx, workspace.CurrentStateVersionID)
		if gErr != nil {
			return gErr
		}

		// A state version could be created by something other than a run e.g. 'terraform import'.
		if sv.RunID == nil {
			return errors.NewError(
				errors.EConflict,
				"current state version was not created by a destroy run")
		}

		run, rErr := s.dbClient.Runs.GetRun(ctx, *sv.RunID)
		if rErr != nil {
			return rErr
		}

		if run == nil {
			return errors.NewError(errors.ENotFound, fmt.Sprintf("Run with ID %s not found", *sv.RunID))
		}

		// Check to keep from accidentally deleting a workspace when resources are still deployed.
		if !run.IsDestroy {
			return errors.NewError(
				errors.EConflict,
				"run associated with the current state version was not a destroy run")
		}
	}

	s.logger.Infow("Requested deletion of a workspace.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteServiceAccount: %v", txErr)
		}
	}()

	// The foreign key with on cascade delete should remove activity events whose target ID is this group.

	err = s.dbClient.Workspaces.DeleteWorkspace(txContext, workspace)
	if err != nil {
		return err
	}

	parentGroupPath := workspace.GetGroupPath()
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &parentGroupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      workspace.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: workspace.Name,
				ID:   workspace.Metadata.ID,
				Type: string(models.TargetWorkspace),
			},
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateWorkspacePermission, auth.WithGroupID(workspace.GroupID))
	if err != nil {
		return nil, err
	}

	// Validate model
	if wErr := workspace.Validate(); wErr != nil {
		return nil, wErr
	}

	workspace.CreatedBy = caller.GetSubject()

	if d := workspace.MaxJobDuration; d != nil {
		if vErr := validateMaxJobDuration(*d); vErr != nil {
			return nil, vErr
		}
	} else {
		duration := int32(defaultMaxJobDuration.Minutes())
		workspace.MaxJobDuration = &duration
	}

	// Get a list of all the supported Terraform versions.
	versions, err := s.cliService.GetTerraformCLIVersions(ctx)
	if err != nil {
		return nil, err
	}

	// Check if requested Terraform version is supported.
	if workspace.TerraformVersion != "" {
		if terr := versions.Supported(workspace.TerraformVersion); terr != nil {
			return nil, terr
		}
	}

	// If nothing is specified use the latest version available.
	if workspace.TerraformVersion == "" {
		workspace.TerraformVersion = versions.Latest()
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateWorkspace: %v", txErr)
		}
	}()

	s.logger.Infow("Requested creation of a new workspace.",
		"caller", caller.GetSubject(),
		"groupID", workspace.GroupID,
		"workspaceName", workspace.Name,
	)
	createdWorkspace, err := s.dbClient.Workspaces.CreateWorkspace(txContext, workspace)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &createdWorkspace.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetWorkspace,
			TargetID:      createdWorkspace.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return createdWorkspace, nil
}

func (s *service) UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Validate model.
	if wErr := workspace.Validate(); wErr != nil {
		return nil, wErr
	}

	if vErr := validateMaxJobDuration(*workspace.MaxJobDuration); vErr != nil {
		return nil, vErr
	}

	// Get a list of all the supported versions.
	versions, err := s.cliService.GetTerraformCLIVersions(ctx)
	if err != nil {
		return nil, err
	}

	// Check if requested Terraform version is supported.
	if err = versions.Supported(workspace.TerraformVersion); err != nil {
		return nil, err
	}

	s.logger.Infow("Requested an update to a workspace.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedWorkspace, nil
}

func (s *service) LockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Check if workspace is already locked.
	if workspace.Locked {
		return nil, WorkspaceLockedError
	}

	// Update the field.
	workspace.Locked = true

	s.logger.Infow("Requested a lock on workspace.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer LockWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionLock,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedWorkspace, nil
}

func (s *service) UnlockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Check if workspace is already unlocked.
	if !workspace.Locked {
		return nil, WorkspaceUnlockedError
	}

	// Check if workspace is locked by a run.
	if workspace.CurrentJobID != "" {
		return nil, WorkspaceLockedByRunError
	}

	// Update the field.
	workspace.Locked = false

	s.logger.Infow("Requested an unlock on workspace.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UnlockWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionUnlock,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedWorkspace, nil
}

func (s *service) GetCurrentStateVersion(ctx context.Context, workspaceID string) (*models.StateVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	workspace, err := s.getWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if workspace == nil || workspace.CurrentStateVersionID == "" {
		return nil, nil
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	return s.GetStateVersion(ctx, workspace.CurrentStateVersionID)
}

func (s *service) GetStateVersionResources(ctx context.Context, stateVersion *models.StateVersion) ([]StateVersionResource, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	reader, err := s.artifactStore.GetStateVersion(ctx, stateVersion)
	if err != nil {
		return nil, err
	}

	// Attempt to unmarshal to a stateV4:
	var state stateV4
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decoded data: %s", err)
	}

	if state.Version != version4 {
		return nil, fmt.Errorf("expected stateVersionV4, got %d", state.Version)
	}

	response := []StateVersionResource{}

	for _, r := range state.Resources {
		resource := StateVersionResource{
			Mode:   r.Mode,
			Type:   r.Type,
			Name:   r.Name,
			Module: r.Module,
		}

		if resource.Module == "" {
			resource.Module = "root"
		}

		startIndex := strings.Index(r.ProviderConfig, "[\"")
		if startIndex == -1 {
			return nil, fmt.Errorf("invalid provider config encountered when parsing state version resources %s", r.ProviderConfig)
		}
		endIndex := strings.LastIndex(r.ProviderConfig, "\"]")
		if endIndex == -1 {
			return nil, fmt.Errorf("invalid provider config encountered when parsing state version resources %s", r.ProviderConfig)
		}

		resource.Provider = r.ProviderConfig[startIndex+2 : endIndex]

		response = append(response, resource)
	}

	return response, nil
}

func (s *service) GetStateVersionDependencies(ctx context.Context, stateVersion *models.StateVersion) ([]StateVersionDependency, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	reader, err := s.artifactStore.GetStateVersion(ctx, stateVersion)
	if err != nil {
		return nil, err
	}

	// Attempt to unmarshal to a stateV4:
	var state stateV4
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decoded data: %s", err)
	}

	if state.Version != version4 {
		return nil, fmt.Errorf("expected stateVersionV4, got %d", state.Version)
	}

	response := []StateVersionDependency{}

	for _, r := range state.Resources {
		if r.ProviderConfig == tharsisTerraformProviderConfig && r.Type == tharsisWorkspaceOutputsDatasourceName {
			if len(r.Instances) != 1 {
				return nil, fmt.Errorf("expected one instance for %s but found %d", r.Type, len(r.Instances))
			}

			attributes := map[string]interface{}{}
			if err := json.Unmarshal(r.Instances[0].AttributesRaw, &attributes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal attributes for tharsis terraform provider %v", err)
			}

			fullPath, ok := attributes["full_path"]
			if !ok {
				return nil, fmt.Errorf("full_path attribute missing from %s resource %s", r.Type, r.Name)
			}

			stateVersionID, ok := attributes["state_version_id"]
			if !ok {
				return nil, fmt.Errorf("state_version_id attribute missing from %s resource %s", r.Type, r.Name)
			}

			workspaceID, ok := attributes["workspace_id"]
			if !ok {
				return nil, fmt.Errorf("workspace_id attribute missing from %s resource %s", r.Type, r.Name)
			}

			response = append(response, StateVersionDependency{
				WorkspacePath:  fullPath.(string),
				WorkspaceID:    gid.FromGlobalID(workspaceID.(string)),
				StateVersionID: gid.FromGlobalID(stateVersionID.(string)),
			})
		}
	}

	return response, nil
}

func (s *service) CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion, data *string) (*models.StateVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	// We need to decode the base64 encoded string
	decoded, err := base64.StdEncoding.DecodeString(*data)
	if err != nil {
		return nil, err
	}

	// Wrap a transaction around persisting the state version and the state version outputs.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateStateVersion: %v", txErr)
		}
	}()

	// Update the CreatedBy field since a state version could be created manually.
	stateVersion.CreatedBy = caller.GetSubject()

	createdStateVersion, err := s.dbClient.StateVersions.CreateStateVersion(txContext, stateVersion)
	if err != nil {
		return nil, err
	}

	// Update the current state version field on the workspace.
	// This is a read-only operation, so there's no need to use the transaction context.
	workspace, wErr := s.getWorkspaceByID(ctx, createdStateVersion.WorkspaceID)
	if wErr != nil {
		return nil, wErr
	}

	workspace.DirtyState = false
	workspace.CurrentStateVersionID = createdStateVersion.Metadata.ID

	// Update the workspace and ignore the returned model since its not needed.
	_, err = s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, err
	}

	// Attempt to unmarshal to a stateV4:
	var state stateV4
	err = json.Unmarshal(decoded, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal decoded data: %s", err)
	}
	if state.Version != version4 {
		return nil, fmt.Errorf("expected stateVersionV4, got %d", state.Version)
	}

	for outputName, outputInfo := range state.RootOutputs {

		newOutput := models.StateVersionOutput{
			Name:           outputName,
			Value:          outputInfo.ValueRaw,
			Type:           outputInfo.ValueTypeRaw,
			Sensitive:      outputInfo.Sensitive,
			StateVersionID: createdStateVersion.Metadata.ID,
		}

		// There's nothing that needs to be done with the stored new output, so ignore it.
		_, err = s.dbClient.StateVersionOutputs.CreateStateVersionOutput(txContext, &newOutput)
		if err != nil {
			return nil, err
		}

	}

	// Upload state version data to object store
	// Does not touch the DB, so no need to use the transaction context.
	if err = s.artifactStore.UploadStateVersion(ctx, createdStateVersion, bytes.NewBuffer(decoded)); err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to write state version to object storage",
			errors.WithErrorErr(err),
		)
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetStateVersion,
			TargetID:      createdStateVersion.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	// Commit the transaction here.  If the upload fails, the transaction will be aborted.
	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	s.logger.Infow("Created a state version with ID",
		"caller", caller.GetSubject(),
		"stateVersionID", createdStateVersion.Metadata.ID,
	)

	return createdStateVersion, nil
}

// GetStateVersion returns a state version by ID
func (s *service) GetStateVersion(ctx context.Context, stateVersionID string) (*models.StateVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersion(ctx, stateVersionID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to query state version from the database",
			errors.WithErrorErr(err),
		)
	}

	if sv == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("State version with ID %s not found", stateVersionID))
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return sv, nil
}

func (s *service) GetStateVersions(ctx context.Context, input *GetStateVersionsInput) (*db.StateVersionsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(input.Workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	return s.dbClient.StateVersions.GetStateVersions(ctx, &db.GetStateVersionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.StateVersionFilter{
			WorkspaceID: &input.Workspace.Metadata.ID,
		},
	})
}

// GetStateVersionContent returns the contents of the state version file
func (s *service) GetStateVersionContent(ctx context.Context, stateVersionID string) (io.ReadCloser, error) {
	sv, err := s.GetStateVersion(ctx, stateVersionID)
	if err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetStateVersion(ctx, sv)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get state version from artifact store",
			errors.WithErrorErr(err),
		)
	}

	return result, nil
}

func (s *service) GetStateVersionsByIDs(ctx context.Context,
	idList []string) ([]models.StateVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.StateVersions.GetStateVersions(ctx, &db.GetStateVersionsInput{
		Filter: &db.StateVersionFilter{
			StateVersionIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get state versions",
			errors.WithErrorErr(err),
		)
	}

	for _, sv := range result.StateVersions {
		err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
		if err != nil {
			return nil, err
		}
	}

	return result.StateVersions, nil
}

func (s *service) GetConfigurationVersionContent(ctx context.Context, configurationVersionID string) (io.ReadCloser, error) {
	cv, err := s.GetConfigurationVersion(ctx, configurationVersionID)
	if err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetConfigurationVersion(ctx, cv)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get configuration version from artifact store",
			errors.WithErrorErr(err),
		)
	}

	return result, nil
}

// CreateConfigurationVersion creates a new configuration version
func (s *service) CreateConfigurationVersion(ctx context.Context, options *CreateConfigurationVersionInput) (*models.ConfigurationVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateConfigurationVersionPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	cv, err := s.dbClient.ConfigurationVersions.CreateConfigurationVersion(ctx, models.ConfigurationVersion{
		VCSEventID:  options.VCSEventID,
		WorkspaceID: options.WorkspaceID,
		Speculative: options.Speculative,
		Status:      models.ConfigurationPending,
		CreatedBy:   caller.GetSubject(),
	})
	if err != nil {
		return nil, err
	}

	s.logger.Infow("Created a configuration version.",
		"caller", caller.GetSubject(),
		"workspaceID", options.WorkspaceID,
		"configurationVersionID", cv.Metadata.ID,
	)
	return cv, nil
}

// GetConfigurationVersion returns a tfe configuration version
func (s *service) GetConfigurationVersion(ctx context.Context, configurationVersionID string) (*models.ConfigurationVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	cv, err := s.dbClient.ConfigurationVersions.GetConfigurationVersion(ctx, configurationVersionID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get configuration version",
			errors.WithErrorErr(err),
		)
	}

	if cv == nil {
		return nil, errors.NewError(
			errors.ENotFound,
			fmt.Sprintf("Configuration version with ID %s not found", configurationVersionID),
		)
	}

	err = caller.RequirePermission(ctx, permissions.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return cv, nil
}

func (s *service) GetConfigurationVersionsByIDs(ctx context.Context, idList []string) ([]models.ConfigurationVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.ConfigurationVersions.GetConfigurationVersions(ctx, &db.GetConfigurationVersionsInput{
		Filter: &db.ConfigurationVersionFilter{
			ConfigurationVersionIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get configuration versions",
			errors.WithErrorErr(err),
		)
	}

	for _, cv := range result.ConfigurationVersions {
		err = caller.RequirePermission(ctx, permissions.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
		if err != nil {
			return nil, err
		}
	}

	return result.ConfigurationVersions, nil
}

// UploadConfigurationVersion uploads a new configuration version file
func (s *service) UploadConfigurationVersion(ctx context.Context, configurationVersionID string, reader io.Reader) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	cv, err := s.GetConfigurationVersion(ctx, configurationVersionID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
	if err != nil {
		return err
	}

	if err := s.artifactStore.UploadConfigurationVersion(ctx, cv, reader); err != nil {
		return errors.NewError(
			errors.EInternal,
			"Failed to write configuration version to object storage",
			errors.WithErrorErr(err),
		)
	}

	// Update status of configuration version to uploaded
	cv.Status = models.ConfigurationUploaded
	if _, err := s.dbClient.ConfigurationVersions.UpdateConfigurationVersion(ctx, *cv); err != nil {
		return errors.NewError(
			errors.EInternal,
			"Failed to to update configuration version",
			errors.WithErrorErr(err),
		)
	}

	return nil
}

func (s *service) GetStateVersionOutputs(ctx context.Context, stateVersionID string) ([]models.StateVersionOutput, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// sv is needed for access check
	sv, err := s.dbClient.StateVersions.GetStateVersion(ctx, stateVersionID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"failed to query state version from the database",
			errors.WithErrorErr(err),
		)
	}
	if sv == nil {
		return nil, errors.NewError(
			errors.EInternal,
			fmt.Sprintf("state version ID %s does not exist.", stateVersionID),
		)
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.StateVersionOutputs.GetStateVersionOutputs(ctx, stateVersionID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"failed to list state version outputs",
			errors.WithErrorErr(err),
		)
	}

	return result, nil
}

func (s *service) getWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if workspace == nil {
		return nil, errors.NewError(
			errors.ENotFound,
			fmt.Sprintf("Workspace with id %s not found", id),
		)
	}

	return workspace, nil
}

// validateMaxJobDuration validates if duration is within MaxJobDuration limits.
func validateMaxJobDuration(duration int32) error {
	if duration < int32(lowerLimitMaxJobDuration.Minutes()) || duration > int32(upperLimitMaxJobDuration.Minutes()) {
		return errors.NewError(errors.EInvalid,
			fmt.Sprintf("Invalid maxJobDuration. Must be between %d and %d.",
				int32(lowerLimitMaxJobDuration.Minutes()),
				int32(upperLimitMaxJobDuration.Minutes()),
			),
		)
	}

	return nil
}

// The End.
