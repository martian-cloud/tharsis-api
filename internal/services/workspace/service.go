package workspace

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
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
	ErrWorkspaceLocked = errors.New("workspace already locked", errors.WithErrorCode(errors.EConflict))

	// Error returned when workspace is already unlocked.
	ErrWorkspaceUnlocked = errors.New("workspace already unlocked", errors.WithErrorCode(errors.EConflict))

	// Error returned when a workspace unlock is attempted but it's locked by a run.
	ErrWorkspaceLockedByRun = errors.New("cannot unlock workspace locked by run", errors.WithErrorCode(errors.EConflict))
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
	// AssignedManagedIdentityID filters the workspaces by the specified managed identity
	AssignedManagedIdentityID *string
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
	MigrateWorkspace(ctx context.Context, workspaceID string, newGroupID string) (*models.Workspace, error)
	GetRunnerTagsSetting(ctx context.Context, workspace *models.Workspace) (*models.RunnerTagsSetting, error)
}

type handleCallerFunc func(
	ctx context.Context,
	userHandler func(ctx context.Context, caller *auth.UserCaller) error,
	serviceAccountHandler func(ctx context.Context, caller *auth.ServiceAccountCaller) error,
) error

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	limitChecker    limits.LimitChecker
	artifactStore   ArtifactStore
	eventManager    *events.EventManager
	cliService      cli.Service
	activityService activityevent.Service
	handleCaller    handleCallerFunc
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	artifactStore ArtifactStore,
	eventManager *events.EventManager,
	cliService cli.Service,
	activityService activityevent.Service,
) Service {
	return newService(
		logger,
		dbClient,
		limitChecker,
		artifactStore,
		eventManager,
		cliService,
		activityService,
		auth.HandleCaller,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	artifactStore ArtifactStore,
	eventManager *events.EventManager,
	cliService cli.Service,
	activityService activityevent.Service,
	handleCaller handleCallerFunc,
) Service {
	return &service{
		logger,
		dbClient,
		limitChecker,
		artifactStore,
		eventManager,
		cliService,
		activityService,
		handleCaller,
	}
}

func (s *service) SubscribeToWorkspaceEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToWorkspaceEvents")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	subscription := events.Subscription{
		Type: events.WorkspaceSubscription,
		ID:   options.WorkspaceID, // Subscribe to specific workspace ID
		Actions: []events.SubscriptionAction{
			events.CreateAction,
			events.UpdateAction,
		},
	}
	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *Event)
	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		// Wait for workspace updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) {
					s.logger.Errorf("error occurred while waiting for workspace events: %v", err)
				}
				return
			}

			ws, err := s.getWorkspaceByID(ctx, event.ID)
			if err != nil {
				if errors.IsContextCanceledError(err) {
					return
				}
				s.logger.Errorf("error occurred while querying for workspace associated with workspace event %s: %v", event.ID, err)
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
	ctx, span := tracer.Start(ctx, "svc.GetWorkspacesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	resp, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{WorkspaceIDs: idList}})
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspaces")
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
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return resp.Workspaces, nil
}

func (s *service) GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*db.WorkspacesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaces")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	dbInput := db.GetWorkspacesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.WorkspaceFilter{
			Search:                    input.Search,
			AssignedManagedIdentityID: input.AssignedManagedIdentityID,
		},
	}

	if input.Group != nil {
		err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			tracing.RecordError(span, napErr, "failed to get namespace access policy")
			return nil, napErr
		}

		if !policy.AllowAll {
			if err = s.handleCaller(
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
				tracing.RecordError(span, err, "failed to set filters for non-admin caller")
				return nil, err
			}
		}
	}

	workspacesResult, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspaces")
		return nil, err
	}

	return workspacesResult, nil
}

func (s *service) GetWorkspaceByFullPath(ctx context.Context, path string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceByFullPath")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithNamespacePath(path))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	workspace, err := s.dbClient.Workspaces.GetWorkspaceByFullPath(ctx, path)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by full path")
		return nil, err
	}

	if workspace == nil {
		tracing.RecordError(span, nil, "Workspace with path %s not found", path)
		return nil, errors.New(
			"Workspace with path %s not found", path,
			errors.WithErrorCode(errors.ENotFound))
	}

	return workspace, nil
}

func (s *service) GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	workspace, err := s.getWorkspaceByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return workspace, nil
}

func (s *service) DeleteWorkspace(ctx context.Context, workspace *models.Workspace, force bool) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if !force && workspace.CurrentStateVersionID != "" {
		sv, gErr := s.GetStateVersion(ctx, workspace.CurrentStateVersionID)
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get state version")
			return gErr
		}

		// A state version could be created by something other than a run e.g. 'terraform import'.
		if sv.RunID == nil {
			tracing.RecordError(span, nil, "current state version was not created by a destroy run")
			return errors.New(
				"current state version was not created by a destroy run",
				errors.WithErrorCode(errors.EConflict),
			)
		}

		run, rErr := s.dbClient.Runs.GetRun(ctx, *sv.RunID)
		if rErr != nil {
			tracing.RecordError(span, rErr, "failed to get run")
			return rErr
		}

		if run == nil {
			tracing.RecordError(span, nil, "run with ID %s not found", *sv.RunID)
			return errors.New("run with ID %s not found", *sv.RunID, errors.WithErrorCode(errors.ENotFound))
		}

		// Check to keep from accidentally deleting a workspace when resources are still deployed.
		if !run.IsDestroy {
			tracing.RecordError(span, nil, "run associated with the current state version was not a destroy run")
			return errors.New("run associated with the current state version was not a destroy run", errors.WithErrorCode(errors.EConflict))
		}
	}

	s.logger.Infow("Requested deletion of a workspace.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteWorkspace: %v", txErr)
		}
	}()

	// The foreign key with on cascade delete should remove activity events whose target ID is this group.

	err = s.dbClient.Workspaces.DeleteWorkspace(txContext, workspace)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete workspace")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateWorkspacePermission, auth.WithGroupID(workspace.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model
	if wErr := workspace.Validate(); wErr != nil {
		tracing.RecordError(span, wErr, "failed to commit DB transaction")
		return nil, wErr
	}

	workspace.CreatedBy = caller.GetSubject()

	if d := workspace.MaxJobDuration; d != nil {
		if vErr := validateMaxJobDuration(*d); vErr != nil {
			tracing.RecordError(span, vErr, "failed to validate max job duration")
			return nil, vErr
		}
	} else {
		duration := int32(defaultMaxJobDuration.Minutes())
		workspace.MaxJobDuration = &duration
	}

	// Get a list of all the supported Terraform versions.
	versions, err := s.cliService.GetTerraformCLIVersions(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to get Terraform CLI versions")
		return nil, err
	}

	// Check if requested Terraform version is supported.
	if workspace.TerraformVersion != "" {
		if terr := versions.Supported(workspace.TerraformVersion); terr != nil {
			tracing.RecordError(span, terr, "requested Terraform version is not supported")
			return nil, terr
		}
	}

	// If nothing is specified use the latest version available.
	if workspace.TerraformVersion == "" {
		workspace.TerraformVersion = versions.Latest()
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to create workspace")
		return nil, err
	}

	// Get the number of workspaces in the group to check whether we just violated the limit.
	newWorkspaces, err := s.dbClient.Workspaces.GetWorkspaces(txContext, &db.GetWorkspacesInput{
		Filter: &db.WorkspaceFilter{
			GroupID: &createdWorkspace.GroupID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's workspaces")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitWorkspacesPerGroup, newWorkspaces.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &createdWorkspace.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetWorkspace,
			TargetID:      createdWorkspace.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdWorkspace, nil
}

func (s *service) UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model.
	if wErr := workspace.Validate(); wErr != nil {
		tracing.RecordError(span, wErr, "failed to validate workspace model")
		return nil, wErr
	}

	if vErr := validateMaxJobDuration(*workspace.MaxJobDuration); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate max job duration")
		return nil, vErr
	}

	// Get a list of all the supported versions.
	versions, err := s.cliService.GetTerraformCLIVersions(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to get list of supported Terraform CLI versions")
		return nil, err
	}

	// Check if requested Terraform version is supported.
	if err = versions.Supported(workspace.TerraformVersion); err != nil {
		tracing.RecordError(span, err, "requested Terraform CLI version is not supported")
		return nil, err
	}

	s.logger.Infow("Requested an update to a workspace.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		tracing.RecordError(span, err, "failed to update workspace")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedWorkspace, nil
}

func (s *service) LockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.LockWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Check if workspace is already locked.
	if workspace.Locked {
		tracing.RecordError(span, nil, "workspace is already locked")
		return nil, ErrWorkspaceLocked
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
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer LockWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		tracing.RecordError(span, err, "failed to update workspace")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionLock,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedWorkspace, nil
}

func (s *service) UnlockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.UnlockWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Check if workspace is already unlocked.
	if !workspace.Locked {
		tracing.RecordError(span, nil, "workspace is already unlocked")
		return nil, ErrWorkspaceUnlocked
	}

	// Check if workspace is locked by a run.
	if workspace.CurrentJobID != "" {
		tracing.RecordError(span, nil, "workspace is locked by a run")
		return nil, ErrWorkspaceLockedByRun
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
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UnlockWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		tracing.RecordError(span, err, "failed to update workspace")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionUnlock,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedWorkspace, nil
}

func (s *service) GetCurrentStateVersion(ctx context.Context, workspaceID string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetCurrentStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	workspace, err := s.getWorkspaceByID(ctx, workspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by ID")
		return nil, err
	}

	if workspace == nil || workspace.CurrentStateVersionID == "" {
		tracing.RecordError(span, nil, "workspace not found or current state version ID is empty")
		return nil, nil
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return s.GetStateVersion(ctx, workspace.CurrentStateVersionID)
}

func (s *service) GetStateVersionResources(ctx context.Context, stateVersion *models.StateVersion) ([]StateVersionResource, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionResources")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	reader, err := s.artifactStore.GetStateVersion(ctx, stateVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to get state version")
		return nil, err
	}

	// Attempt to unmarshal to a stateV4:
	var state stateV4
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		tracing.RecordError(span, nil, "failed to unmarshal decoded data: %s", err)
		return nil, fmt.Errorf("failed to unmarshal decoded data: %s", err)
	}

	if state.Version != version4 {
		tracing.RecordError(span, nil, "expected stateVersionV4, got %d", state.Version)
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
			tracing.RecordError(span, nil,
				"invalid provider config encountered when parsing state version resources %s", r.ProviderConfig)
			return nil, fmt.Errorf("invalid provider config encountered when parsing state version resources %s", r.ProviderConfig)
		}
		endIndex := strings.LastIndex(r.ProviderConfig, "\"]")
		if endIndex == -1 {
			tracing.RecordError(span, nil,
				"invalid provider config encountered when parsing state version resources %s", r.ProviderConfig)
			return nil, fmt.Errorf("invalid provider config encountered when parsing state version resources %s", r.ProviderConfig)
		}

		resource.Provider = r.ProviderConfig[startIndex+2 : endIndex]

		response = append(response, resource)
	}

	return response, nil
}

func (s *service) GetStateVersionDependencies(ctx context.Context, stateVersion *models.StateVersion) ([]StateVersionDependency, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionDependencies")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	reader, err := s.artifactStore.GetStateVersion(ctx, stateVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to get state version")
		return nil, err
	}

	// Attempt to unmarshal to a stateV4:
	var state stateV4
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		tracing.RecordError(span, nil, "failed to unmarshal decoded data: %s", err)
		return nil, fmt.Errorf("failed to unmarshal decoded data: %s", err)
	}

	if state.Version != version4 {
		tracing.RecordError(span, nil, "expected stateVersionV4, got %d", state.Version)
		return nil, fmt.Errorf("expected stateVersionV4, got %d", state.Version)
	}

	response := []StateVersionDependency{}

	for _, r := range state.Resources {
		if r.ProviderConfig == tharsisTerraformProviderConfig && r.Type == tharsisWorkspaceOutputsDatasourceName && len(r.Instances) > 0 {
			attributes := map[string]interface{}{}
			if err := json.Unmarshal(r.Instances[0].AttributesRaw, &attributes); err != nil {
				tracing.RecordError(span, nil,
					"failed to unmarshal attributes for tharsis terraform provider %v", err)
				return nil, fmt.Errorf("failed to unmarshal attributes for tharsis terraform provider %v", err)
			}

			fullPath, ok := attributes["full_path"]
			if !ok {
				tracing.RecordError(span, nil,
					"full_path attribute missing from %s resource %s", r.Type, r.Name)
				return nil, fmt.Errorf("full_path attribute missing from %s resource %s", r.Type, r.Name)
			}

			stateVersionID, ok := attributes["state_version_id"]
			if !ok {
				tracing.RecordError(span, nil,
					"state_version_id attribute missing from %s resource %s", r.Type, r.Name)
				return nil, fmt.Errorf("state_version_id attribute missing from %s resource %s", r.Type, r.Name)
			}

			workspaceID, ok := attributes["workspace_id"]
			if !ok {
				tracing.RecordError(span, nil,
					"workspace_id attribute missing from %s resource %s", r.Type, r.Name)
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
	ctx, span := tracer.Start(ctx, "svc.CreateStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// We need to decode the base64 encoded string
	decoded, err := base64.StdEncoding.DecodeString(*data)
	if err != nil {
		tracing.RecordError(span, err, "failed to decoded base64-encoded state version")
		return nil, err
	}

	// Wrap a transaction around persisting the state version and the state version outputs.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to create state version")
		return nil, err
	}

	// Get the number of recent state versions for this workspace to check whether we just violated the limit.
	recentStateVersions, err := s.dbClient.StateVersions.GetStateVersions(txContext, &db.GetStateVersionsInput{
		Filter: &db.StateVersionFilter{
			TimeRangeStart: ptr.Time(createdStateVersion.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			WorkspaceID:    &stateVersion.WorkspaceID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace's state versions")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitStateVersionsPerWorkspacePerTimePeriod, recentStateVersions.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	// Update the current state version field on the workspace.
	// This is a read-only operation, so there's no need to use the transaction context.
	workspace, wErr := s.getWorkspaceByID(ctx, createdStateVersion.WorkspaceID)
	if wErr != nil {
		tracing.RecordError(span, wErr, "failed to get workspace by ID")
		return nil, wErr
	}

	workspace.DirtyState = false
	workspace.CurrentStateVersionID = createdStateVersion.Metadata.ID

	// Update the workspace and ignore the returned model since its not needed.
	_, err = s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		tracing.RecordError(span, err, "failed to update workspace")
		return nil, err
	}

	// Attempt to unmarshal to a stateV4:
	var state stateV4
	err = json.Unmarshal(decoded, &state)
	if err != nil {
		tracing.RecordError(span, nil, "failed to unmarshal decoded data: %s", err)
		return nil, fmt.Errorf("failed to unmarshal decoded data: %s", err)
	}
	if state.Version != version4 {
		tracing.RecordError(span, nil, "expected stateVersionV4, got %d", state.Version)
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
			tracing.RecordError(span, err, "failed to create state version output")
			return nil, err
		}

	}

	// Upload state version data to object store
	// Does not touch the DB, so no need to use the transaction context.
	if err = s.artifactStore.UploadStateVersion(ctx, createdStateVersion, bytes.NewBuffer(decoded)); err != nil {
		tracing.RecordError(span, err, "failed to upload state version")
		return nil, errors.Wrap(
			err,
			"Failed to write state version to object storage",
		)
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetStateVersion,
			TargetID:      createdStateVersion.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	// Commit the transaction here.  If the upload fails, the transaction will be aborted.
	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a new state version",
		"caller", caller.GetSubject(),
		"stateVersionID", createdStateVersion.Metadata.ID,
		"workspaceID", createdStateVersion.WorkspaceID,
		"workspaceFullPath", workspace.FullPath,
	)

	return createdStateVersion, nil
}

// GetStateVersion returns a state version by ID
func (s *service) GetStateVersion(ctx context.Context, stateVersionID string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersion(ctx, stateVersionID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to query state version from the database")
		return nil, errors.Wrap(
			err,
			"Failed to query state version from the database",
		)
	}

	if sv == nil {
		tracing.RecordError(span, nil, "state version with ID %s not found", stateVersionID)
		return nil, errors.New("state version with ID %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return sv, nil
}

func (s *service) GetStateVersions(ctx context.Context, input *GetStateVersionsInput) (*db.StateVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(input.Workspace.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionContent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersion(ctx, stateVersionID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to query state version from the database")
		return nil, errors.Wrap(
			err,
			"Failed to query state version from the database",
		)
	}

	if sv == nil {
		tracing.RecordError(span, nil, "state version with ID %s not found", stateVersionID)
		return nil, errors.New("state version with ID %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionDataPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	result, err := s.artifactStore.GetStateVersion(ctx, sv)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get state version from artifact store")
		return nil, errors.Wrap(
			err,
			"Failed to get state version from artifact store",
		)
	}

	return result, nil
}

func (s *service) GetStateVersionsByIDs(ctx context.Context,
	idList []string) ([]models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.StateVersions.GetStateVersions(ctx, &db.GetStateVersionsInput{
		Filter: &db.StateVersionFilter{
			StateVersionIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "Failed to get state versions")
		return nil, errors.Wrap(
			err,
			"Failed to get state versions",
		)
	}

	for _, sv := range result.StateVersions {
		err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.StateVersions, nil
}

func (s *service) GetConfigurationVersionContent(ctx context.Context, configurationVersionID string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "svc.GetConfigurationVersionContent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	cv, err := s.GetConfigurationVersion(ctx, configurationVersionID)
	if err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetConfigurationVersion(ctx, cv)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get configuration version from artifact store")
		return nil, errors.Wrap(
			err,
			"Failed to get configuration version from artifact store",
		)
	}

	return result, nil
}

// CreateConfigurationVersion creates a new configuration version
func (s *service) CreateConfigurationVersion(ctx context.Context, options *CreateConfigurationVersionInput) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateConfigurationVersionPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Wrap a transaction around persisting the new configuration version.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateConfigurationVersion: %v", txErr)
		}
	}()

	cv, err := s.dbClient.ConfigurationVersions.CreateConfigurationVersion(txContext, models.ConfigurationVersion{
		VCSEventID:  options.VCSEventID,
		WorkspaceID: options.WorkspaceID,
		Speculative: options.Speculative,
		Status:      models.ConfigurationPending,
		CreatedBy:   caller.GetSubject(),
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create configuration version")
		return nil, err
	}

	// Get the number of recent configuration versions for this workspace to check whether we just violated the limit.
	recentCVs, err := s.dbClient.ConfigurationVersions.GetConfigurationVersions(txContext, &db.GetConfigurationVersionsInput{
		Filter: &db.ConfigurationVersionFilter{
			TimeRangeStart: ptr.Time(cv.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			WorkspaceID:    &options.WorkspaceID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace's configuration versions")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitConfigurationVersionsPerWorkspacePerTimePeriod, recentCVs.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	// Commit the transaction here.
	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.GetConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	cv, err := s.dbClient.ConfigurationVersions.GetConfigurationVersion(ctx, configurationVersionID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get configuration version")
		return nil, errors.Wrap(
			err,
			"Failed to get configuration version",
		)
	}

	if cv == nil {
		tracing.RecordError(span, nil, "Configuration version with ID %s not found", configurationVersionID)
		return nil, errors.New(
			"Configuration version with ID %s not found", configurationVersionID,
			errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return cv, nil
}

func (s *service) GetConfigurationVersionsByIDs(ctx context.Context, idList []string) ([]models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetConfigurationVersionsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.ConfigurationVersions.GetConfigurationVersions(ctx, &db.GetConfigurationVersionsInput{
		Filter: &db.ConfigurationVersionFilter{
			ConfigurationVersionIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "Failed to get configuration versions")
		return nil, errors.Wrap(
			err,
			"Failed to get configuration versions",
		)
	}

	for _, cv := range result.ConfigurationVersions {
		err = caller.RequirePermission(ctx, permissions.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.ConfigurationVersions, nil
}

// UploadConfigurationVersion uploads a new configuration version file
func (s *service) UploadConfigurationVersion(ctx context.Context, configurationVersionID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	cv, err := s.GetConfigurationVersion(ctx, configurationVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get configuration version")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if err := s.artifactStore.UploadConfigurationVersion(ctx, cv, reader); err != nil {
		tracing.RecordError(span, err, "Failed to write configuration version to object storage")
		return errors.Wrap(
			err,
			"Failed to write configuration version to object storage",
		)
	}

	// Update status of configuration version to uploaded
	cv.Status = models.ConfigurationUploaded
	if _, err := s.dbClient.ConfigurationVersions.UpdateConfigurationVersion(ctx, *cv); err != nil {
		tracing.RecordError(span, err, "Failed to to update configuration version")
		return errors.Wrap(
			err,
			"Failed to to update configuration version",
		)
	}

	return nil
}

func (s *service) GetStateVersionOutputs(ctx context.Context, stateVersionID string) ([]models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionOutputs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// sv is needed for access check
	sv, err := s.dbClient.StateVersions.GetStateVersion(ctx, stateVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to query state version from the database")
		return nil, errors.Wrap(
			err,
			"failed to query state version from the database",
		)
	}

	if sv == nil {
		tracing.RecordError(span, nil, "state version with id %s not found", stateVersionID)
		return nil, errors.New("state version with id %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, permissions.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	result, err := s.dbClient.StateVersionOutputs.GetStateVersionOutputs(ctx, stateVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to list state version outputs")
		return nil, errors.Wrap(
			err,
			"failed to list state version outputs",
		)
	}

	return result, nil
}

// GetRunnerTagsSetting returns the (inherited or direct) runner tags setting for a workspace.
func (s *service) GetRunnerTagsSetting(ctx context.Context, workspace *models.Workspace) (*models.RunnerTagsSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerTagsSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewWorkspacePermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// The workspace sets its own tags.
	if workspace.RunnerTags != nil {
		return &models.RunnerTagsSetting{
			Inherited:     false,
			NamespacePath: workspace.FullPath,
			Value:         workspace.RunnerTags,
		}, nil
	}

	sortLowestToHighest := db.GroupSortableFieldFullPathDesc
	parentGroupsResult, err := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
		Sort: &sortLowestToHighest,
		Filter: &db.GroupFilter{
			GroupPaths: workspace.ExpandPath()[1:],
		},
	})
	if err != nil {
		return nil, err
	}

	parentGroups := []*models.Group{}
	for _, g := range parentGroupsResult.Groups {
		copyGroup := g
		parentGroups = append(parentGroups, &copyGroup)
	}

	return models.GetRunnerTagsSetting(parentGroups), nil
}

func (s *service) getWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if workspace == nil {
		return nil, errors.New(
			"workspace with id %s not found", id,
			errors.WithErrorCode(errors.ENotFound))
	}

	return workspace, nil
}

func (s *service) MigrateWorkspace(ctx context.Context, workspaceID string, newGroupID string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.MigrateWorkspace")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	// The caller must have CreateWorkspacePermission in the new parent.
	err = caller.RequirePermission(ctx, permissions.CreateWorkspacePermission, auth.WithGroupID(newGroupID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	// Caller must have DeleteWorkspacePermission in the workspace being moved.
	err = caller.RequirePermission(ctx, permissions.DeleteWorkspacePermission, auth.WithWorkspaceID(workspaceID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	// Get the workspace to be moved.
	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by ID", errors.WithSpan(span))
	}
	if workspace == nil {
		return nil, errors.New(
			"workspace with id %s not found", workspaceID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Get the old parent group.
	oldGroupID := workspace.GroupID
	oldParent, err := s.dbClient.Groups.GetGroupByID(ctx, oldGroupID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get old parent group by ID", errors.WithSpan(span))
	}
	if oldParent == nil {
		return nil, errors.New(
			"Old parent group with id %s not found", oldGroupID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Get the new parent group.
	newGroup, nErr := s.dbClient.Groups.GetGroupByID(ctx, newGroupID)
	if nErr != nil {
		return nil, errors.Wrap(nErr, "failed to get a group by ID", errors.WithSpan(span))
	}
	if newGroup == nil {
		return nil, errors.New(
			"group with id %s not found", newGroupID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// In case a user gets confused or otherwise tries to do a no-op move, detect and bail out.
	// Because nothing gets done, it's safe to do this before the authorization check on the new parent.
	if oldGroupID == newGroupID {
		// Return BadRequest.
		return nil, errors.New("workspace is already in the specified group", errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span))
	}

	// Because the workspace to be moved and the new parent group have been fetched from the DB,
	// there's no need to validate them.

	s.logger.Infow("Requested a workspace migration.",
		"caller", caller.GetSubject(),
		"fullPath", workspace.FullPath, // This is the full path of the workspace prior to migration.
		"workspaceID", workspace.Metadata.ID,
		"newGroupPath", newGroup.FullPath,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin a DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer MigrateWorkspace: %v", txErr)
		}
	}()

	// Now that all checks have passed and the transaction is open, do the actual work of the migration.
	migratedWorkspace, err := s.dbClient.Workspaces.MigrateWorkspace(txContext, workspace, newGroup)
	if err != nil {
		return nil, errors.Wrap(err, "failed to migrate a workspace", errors.WithSpan(span))
	}

	// Check limits to see whether we just committed a violation.
	children, err := s.dbClient.Workspaces.GetWorkspaces(txContext, &db.GetWorkspacesInput{
		Filter: &db.WorkspaceFilter{
			GroupID: &newGroupID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get parent group's children", errors.WithSpan(span))
	}

	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitWorkspacesPerGroup, children.PageInfo.TotalCount); err != nil {
		return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
	}

	// Generate an activity event on the workspace that was migrated.
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &migratedWorkspace.FullPath,
			Action:        models.ActionMigrate,
			TargetType:    models.TargetWorkspace,
			TargetID:      migratedWorkspace.Metadata.ID,
			Payload: &models.ActivityEventMigrateWorkspacePayload{
				PreviousGroupPath: oldParent.FullPath,
			},
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create an activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit a DB transaction", errors.WithSpan(span))
	}

	return migratedWorkspace, nil
}

// validateMaxJobDuration validates if duration is within MaxJobDuration limits.
func validateMaxJobDuration(duration int32) error {
	if duration < int32(lowerLimitMaxJobDuration.Minutes()) || duration > int32(upperLimitMaxJobDuration.Minutes()) {
		return errors.New(
			"invalid maxJobDuration. Must be between %d and %d",
			int32(lowerLimitMaxJobDuration.Minutes()),
			int32(upperLimitMaxJobDuration.Minutes()),
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	return nil
}
