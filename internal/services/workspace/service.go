// Package workspace provides the workspace service.
package workspace

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/terraform"
	coreworkspace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
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

	// maxStateVersionOutputSizeBytes is the maximum allowed size in bytes of a single
	// Terraform state version output value.
	maxStateVersionOutputSizeBytes = 2 * 1024 * 1024 // 2 MiB

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

	// Error returned when a workspace lock is attempted but it has an apply run in progress.
	ErrWorkspaceHasCurrentApplyRun = errors.New("cannot lock workspace with an apply run in progress", errors.WithErrorCode(errors.EConflict))
)

// EventType identifies the kind of workspace event, encoding both the resource
// that changed and the action. Mirrors the semantic event-type pattern used by agent.EventType.
type EventType string

// EventType values.
const (
	WorkspaceEventCreated           EventType = "WORKSPACE_CREATED"
	WorkspaceEventUpdated           EventType = "WORKSPACE_UPDATED"
	WorkspaceEventAssessmentCreated EventType = "WORKSPACE_ASSESSMENT_CREATED"
	WorkspaceEventAssessmentUpdated EventType = "WORKSPACE_ASSESSMENT_UPDATED"
)

// Event represents a workspace event
type Event struct {
	Type      EventType
	Workspace models.Workspace
}

// workspaceEventType maps a database event's table and action to the semantic workspace
// event type. The second return is false for any combination the subscription does not emit.
func workspaceEventType(table, action string) (EventType, bool) {
	switch events.SubscriptionType(table) {
	case events.WorkspaceSubscription:
		switch events.SubscriptionAction(action) {
		case events.CreateAction:
			return WorkspaceEventCreated, true
		case events.UpdateAction:
			return WorkspaceEventUpdated, true
		}
	case events.WorkspaceAssessmentSubscription:
		switch events.SubscriptionAction(action) {
		case events.CreateAction:
			return WorkspaceEventAssessmentCreated, true
		case events.UpdateAction:
			return WorkspaceEventAssessmentUpdated, true
		}
	}
	return "", false
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

// StateVersionInventory contains all state-derived data resolved from a single artifact download.
type StateVersionInventory struct {
	Resources    []*StateVersionResource
	Dependencies []*StateVersionDependency
	CheckResults []*corerun.CheckResult
}

// GetWorkspacesInput is the input for querying a list of workspaces
type GetWorkspacesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.WorkspaceSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// GroupID filters the workspaces by the specified group
	GroupID *string
	// AssignedManagedIdentityID filters the workspaces by the specified managed identity
	AssignedManagedIdentityID *string
	// Search is used to search for a workspace by name or namespace path
	Search *string
	// WorkspacePath is used to search for a workspace by its full path
	WorkspacePath *string
	// LabelFilters filters workspaces by labels
	LabelFilters []db.WorkspaceLabelFilter
	// Favorites filters to only return user's favorite workspaces
	Favorites *bool
	// ExcludeFavorites excludes the user's favorite workspaces from the results
	ExcludeFavorites *bool
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

// GetConfigurationVersionContentOutput is the result of GetConfigurationVersionContent.
type GetConfigurationVersionContentOutput struct {
	Body          io.ReadCloser
	ContentLength int64
}

// SetWorkspaceRoleBindingInput is the input for creating, changing, or removing a workspace's role
// binding. Setting RoleID to nil removes the binding.
type SetWorkspaceRoleBindingInput struct {
	WorkspaceID string
	RoleID      *string
}

// Service implements all workspace related functionality
type Service interface {
	SubscribeToWorkspaceEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error)
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	GetWorkspaceByTRN(ctx context.Context, trn string) (*models.Workspace, error)
	GetWorkspaces(ctx context.Context, input *GetWorkspacesInput) (*db.WorkspacesResult, error)
	GetWorkspacesByIDs(ctx context.Context, idList []string) ([]models.Workspace, error)
	CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	DeleteWorkspace(ctx context.Context, workspace *models.Workspace, force bool) error
	LockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	UnlockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error)
	GetCurrentStateVersion(ctx context.Context, workspaceID string) (*models.StateVersion, error)
	CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion, data string, jsonData *string) (*models.StateVersion, error)
	GetStateVersionByID(ctx context.Context, stateVersionID string) (*models.StateVersion, error)
	GetStateVersionByTRN(ctx context.Context, trn string) (*models.StateVersion, error)
	GetStateVersions(ctx context.Context, input *GetStateVersionsInput) (*db.StateVersionsResult, error)
	GetStateVersionContent(ctx context.Context, stateVersionID string) (io.ReadCloser, error)
	UploadStateVersionJSON(ctx context.Context, stateVersionID string, reader io.Reader) error
	GetStateVersionJSONContent(ctx context.Context, stateVersionID string) (io.ReadCloser, error)
	GetStateVersionsByIDs(ctx context.Context, idList []string) ([]models.StateVersion, error)
	GetWorkspaceAssessmentByID(ctx context.Context, id string) (*models.WorkspaceAssessment, error)
	GetWorkspaceAssessmentByTRN(ctx context.Context, trn string) (*models.WorkspaceAssessment, error)
	GetWorkspaceAssessmentsByWorkspaceIDs(ctx context.Context, idList []string) ([]models.WorkspaceAssessment, error)
	CreateConfigurationVersion(ctx context.Context, options *CreateConfigurationVersionInput) (*models.ConfigurationVersion, error)
	GetConfigurationVersionByID(ctx context.Context, configurationVersionID string) (*models.ConfigurationVersion, error)
	GetConfigurationVersionByTRN(ctx context.Context, configurationVersionID string) (*models.ConfigurationVersion, error)
	UploadConfigurationVersion(ctx context.Context, configurationVersionID string, reader io.Reader) error
	GetConfigurationVersionContent(ctx context.Context, configurationVersionID string) (*GetConfigurationVersionContentOutput, error)
	GetConfigurationVersionsByIDs(ctx context.Context, idList []string) ([]models.ConfigurationVersion, error)
	GetStateVersionOutputByID(ctx context.Context, id string) (*models.StateVersionOutput, error)
	GetStateVersionOutputByTRN(ctx context.Context, trn string) (*models.StateVersionOutput, error)
	GetStateVersionOutputs(ctx context.Context, stateVersionID string) ([]models.StateVersionOutput, error)
	GetStateVersionInventory(ctx context.Context, stateVersion *models.StateVersion) (*StateVersionInventory, error)
	MigrateWorkspace(ctx context.Context, workspaceID string, newGroupID string) (*models.Workspace, error)
	GetWorkspaceRoleBindingByID(ctx context.Context, id string) (*models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindingByTRN(ctx context.Context, trn string) (*models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindingByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindingsByWorkspaceIDs(ctx context.Context, idList []string) ([]models.WorkspaceRoleBinding, error)
	GetWorkspaceRoleBindingsByIDs(ctx context.Context, idList []string) ([]models.WorkspaceRoleBinding, error)
	SetWorkspaceRoleBinding(ctx context.Context, options *SetWorkspaceRoleBindingInput) (*models.WorkspaceRoleBinding, error)
	GetRunnerTagsSetting(ctx context.Context, workspace *models.Workspace) (*namespace.RunnerTagsSetting, error)
	GetDriftDetectionEnabledSetting(ctx context.Context, workspace *models.Workspace) (*namespace.DriftDetectionEnabledSetting, error)
	GetProviderMirrorEnabledSetting(ctx context.Context, workspace *models.Workspace) (*namespace.ProviderMirrorEnabledSetting, error)
	GetOutputVisibilitySetting(ctx context.Context, workspace *models.Workspace) (*namespace.OutputVisibilitySetting, error)
}

type handleCallerFunc func(
	ctx context.Context,
	userHandler func(ctx context.Context, caller *auth.UserCaller) error,
	serviceAccountHandler func(ctx context.Context, caller *auth.ServiceAccountCaller) error,
) error

type service struct {
	logger                        logger.Logger
	dbClient                      *db.Client
	limitChecker                  limits.LimitChecker
	artifactStore                 coreworkspace.ArtifactStore
	eventManager                  *events.EventManager
	terraformCLIVersionConstraint string
	inheritedSettingsResolver     namespace.InheritedSettingResolver
	handleCaller                  handleCallerFunc
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	artifactStore coreworkspace.ArtifactStore,
	eventManager *events.EventManager,
	terraformCLIVersionConstraint string,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
) Service {
	return newService(
		logger,
		dbClient,
		limitChecker,
		artifactStore,
		eventManager,
		terraformCLIVersionConstraint,
		inheritedSettingsResolver,
		auth.HandleCaller,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	artifactStore coreworkspace.ArtifactStore,
	eventManager *events.EventManager,
	terraformCLIVersionConstraint string,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
	handleCaller handleCallerFunc,
) Service {
	return &service{
		logger:                        logger,
		dbClient:                      dbClient,
		limitChecker:                  limitChecker,
		artifactStore:                 artifactStore,
		eventManager:                  eventManager,
		terraformCLIVersionConstraint: terraformCLIVersionConstraint,
		inheritedSettingsResolver:     inheritedSettingsResolver,
		handleCaller:                  handleCaller,
	}
}

func (s *service) SubscribeToWorkspaceEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToWorkspaceEvents")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	subscriptions := []events.Subscription{
		{
			Type: events.WorkspaceSubscription,
			ID:   options.WorkspaceID, // Subscribe to specific workspace ID
			Actions: []events.SubscriptionAction{
				events.CreateAction,
				events.UpdateAction,
			},
		},
		{
			// Also fire when the workspace's assessment is created or updated. Assessment
			// events key on the assessment ID, not the workspace ID, so scope by the
			// workspace_id carried in the event data instead of the subscription ID.
			Type: events.WorkspaceAssessmentSubscription,
			Actions: []events.SubscriptionAction{
				events.CreateAction,
				events.UpdateAction,
			},
			Filter: func(data json.RawMessage) bool {
				var d db.WorkspaceAssessmentEventData
				return json.Unmarshal(data, &d) == nil && d.WorkspaceID == options.WorkspaceID
			},
		},
	}
	subscriber := s.eventManager.Subscribe(subscriptions)

	outgoing := make(chan *Event)
	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		// Wait for workspace updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) && !errors.IsDeadlineExceededError(err) {
					s.logger.WithContextFields(ctx).Errorf("error occurred while waiting for workspace events: %v", err)
				}
				return
			}

			// Both subscriptions are scoped to this workspace (the workspace event by ID,
			// the assessment event by the workspace_id filter), so every delivered event
			// pertains to options.WorkspaceID — resolve by it rather than event.ID, which
			// for an assessment event is the assessment ID.
			eventType, ok := workspaceEventType(event.Table, event.Action)
			if !ok {
				// Defensive: the subscriptions only register create/update on the workspace and
				// workspace_assessment tables, so any other combination is unexpected.
				s.logger.WithContextFields(ctx).Errorf("received unexpected workspace event for table %q action %q", event.Table, event.Action)
				continue
			}

			ws, err := s.getWorkspaceByID(ctx, options.WorkspaceID)
			if err != nil {
				if errors.IsContextCanceledError(err) || errors.IsDeadlineExceededError(err) {
					return
				}
				s.logger.WithContextFields(ctx).Errorf("error occurred while querying for workspace associated with workspace event %s: %v", event.ID, err)
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &Event{Type: eventType, Workspace: *ws}:
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
		return nil, err
	}

	resp, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{Filter: &db.WorkspaceFilter{WorkspaceIDs: idList}})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspaces", errors.WithSpan(span))
	}

	wsPaths := []string{}
	for _, ws := range resp.Workspaces {
		wsPaths = append(wsPaths, ws.FullPath)
	}

	// Verify caller has access to all returned workspaces.
	if len(wsPaths) > 0 {
		err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePaths(wsPaths))
		if err != nil {
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
		return nil, err
	}

	dbInput := db.GetWorkspacesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.WorkspaceFilter{
			Search:                    input.Search,
			AssignedManagedIdentityID: input.AssignedManagedIdentityID,
			WorkspacePath:             input.WorkspacePath,
			LabelFilters:              input.LabelFilters,
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

	if input.GroupID != nil {
		err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithGroupID(*input.GroupID))
		if err != nil {
			return nil, err
		}
		dbInput.Filter.GroupID = input.GroupID
	} else if !caller.IsAdminModeActivated(ctx) {
		rootNamespaces, rErr := caller.GetRootNamespaceMemberships(ctx)
		if rErr != nil {
			return nil, errors.Wrap(rErr, "failed to get root namespaces", errors.WithSpan(span))
		}
		dbInput.Filter.RootNamespaceMemberships = rootNamespaces
	}

	workspacesResult, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &dbInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspaces", errors.WithSpan(span))
	}

	return workspacesResult, nil
}

func (s *service) GetWorkspaceByTRN(ctx context.Context, trn string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceByTRN")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	workspace, err := s.dbClient.Workspaces.GetWorkspaceByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by full path", errors.WithSpan(span))
	}

	if workspace == nil {
		return nil, errors.New(
			"Workspace with TRN %s not found", trn,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *service) GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	workspace, err := s.getWorkspaceByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by ID", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *service) DeleteWorkspace(ctx context.Context, workspace *models.Workspace, force bool) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteWorkspace")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return err
	}

	if !force && workspace.CurrentStateVersionID != "" {
		sv, gErr := s.getStateVersionByID(ctx, workspace.CurrentStateVersionID)
		if gErr != nil {
			return errors.Wrap(gErr, "failed to get state version", errors.WithSpan(span))
		}

		// A state version could be created by something other than a run e.g. 'terraform import'.
		if sv.RunID == nil {
			return errors.New(
				"current state version was not created by a destroy run",
				errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
		}

		run, rErr := s.dbClient.Runs.GetRunByID(ctx, *sv.RunID)
		if rErr != nil {
			return errors.Wrap(rErr, "failed to get run", errors.WithSpan(span))
		}

		if run == nil {
			return errors.New("run with ID %s not found", *sv.RunID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}

		// Check to keep from accidentally deleting a workspace when resources are still deployed.
		if !run.IsDestroy {
			return errors.New("run associated with the current state version was not a destroy run", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
		}
	}

	s.logger.WithContextFields(ctx).Infow("Requested deletion of a workspace.",
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
		"force", force,
		"currentStateVersionID", workspace.CurrentStateVersionID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeleteWorkspace: %v", txErr)
		}
	}()

	// The foreign key with on cascade delete should remove activity events whose target ID is this group.
	err = s.dbClient.Workspaces.DeleteWorkspace(txContext, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to delete workspace")
	}

	parentGroupPath := workspace.GetGroupPath()
	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
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
		return errors.Wrap(err, "failed to create activity event for workspace deletion")
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit transaction for workspace deletion")
	}

	return nil
}

func (s *service) CreateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateWorkspacePermission, auth.WithGroupID(workspace.GroupID))
	if err != nil {
		return nil, err
	}

	// Validate model (includes label validation)
	if wErr := workspace.Validate(); wErr != nil {
		return nil, errors.Wrap(wErr, "failed to validate workspace model", errors.WithSpan(span))
	}

	workspace.CreatedBy = caller.GetSubject()

	if d := workspace.MaxJobDuration; d != nil {
		if vErr := validateMaxJobDuration(*d); vErr != nil {
			return nil, errors.Wrap(vErr, "failed to validate max job duration", errors.WithSpan(span))
		}
	} else {
		duration := int32(defaultMaxJobDuration.Minutes())
		workspace.MaxJobDuration = &duration
	}

	// Get a list of all the supported Terraform versions.
	versions, err := terraform.GetCLIVersions(ctx, s.terraformCLIVersionConstraint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Terraform CLI versions", errors.WithSpan(span))
	}

	// Check if requested Terraform version is supported.
	if workspace.TerraformVersion != "" {
		if terr := versions.Supported(workspace.TerraformVersion); terr != nil {
			return nil, errors.Wrap(terr, "requested Terraform version is not supported", errors.WithSpan(span))
		}
	}

	// If nothing is specified use the latest version available.
	if workspace.TerraformVersion == "" {
		workspace.TerraformVersion = versions.Latest()
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateWorkspace: %v", txErr)
		}
	}()

	s.logger.WithContextFields(ctx).Infow("Requested creation of a new workspace.",
		"groupID", workspace.GroupID,
		"workspaceName", workspace.Name,
	)
	createdWorkspace, err := s.dbClient.Workspaces.CreateWorkspace(txContext, workspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to get group's workspaces", errors.WithSpan(span))
	}
	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitWorkspacesPerGroup, newWorkspaces.PageInfo.TotalCount); err != nil {
		return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
	}

	// Create activity event with label information if labels exist
	activityEventInput := &activity.CreateActivityEventInput{
		NamespacePath: &createdWorkspace.FullPath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetWorkspace,
		TargetID:      createdWorkspace.Metadata.ID,
		Payload: &models.ActivityEventCreateWorkspacePayload{
			Labels: createdWorkspace.Labels,
		},
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, activityEventInput); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return createdWorkspace, nil
}

func (s *service) UpdateWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Validate model.
	if wErr := workspace.Validate(); wErr != nil {
		return nil, errors.Wrap(wErr, "failed to validate workspace model", errors.WithSpan(span))
	}

	if vErr := validateMaxJobDuration(*workspace.MaxJobDuration); vErr != nil {
		return nil, errors.Wrap(vErr, "failed to validate max job duration", errors.WithSpan(span))
	}

	// Get a list of all the supported versions.
	versions, err := terraform.GetCLIVersions(ctx, s.terraformCLIVersionConstraint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of supported Terraform CLI versions", errors.WithSpan(span))
	}

	// Check if requested Terraform version is supported.
	if err = versions.Supported(workspace.TerraformVersion); err != nil {
		return nil, errors.Wrap(err, "requested Terraform CLI version is not supported", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Requested an update to a workspace.",
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	// Get the current workspace to detect label changes
	currentWorkspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspace.Metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current workspace", errors.WithSpan(span))
	}

	if currentWorkspace == nil {
		return nil, errors.New("workspace with ID %s not found", workspace.Metadata.ID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateWorkspace: %v", txErr)
		}
	}()

	// Detect label changes
	labelChanges := detectLabelChanges(currentWorkspace.Labels, workspace.Labels)

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update workspace", errors.WithSpan(span))
	}

	// Create activity event with label change information if there are changes
	activityEventInput := &activity.CreateActivityEventInput{
		NamespacePath: &updatedWorkspace.FullPath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetWorkspace,
		TargetID:      updatedWorkspace.Metadata.ID,
	}

	// Include label changes in activity event payload if there are any changes
	if labelChanges != nil && (len(labelChanges.Added) > 0 || len(labelChanges.Updated) > 0 || len(labelChanges.Removed) > 0) {
		activityEventInput.Payload = &models.ActivityEventUpdateWorkspacePayload{
			LabelChanges: labelChanges,
		}
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, activityEventInput); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return updatedWorkspace, nil
}

func (s *service) LockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.LockWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Check if workspace is already locked.
	if workspace.Locked {
		return nil, ErrWorkspaceLocked
	}

	// Cannot lock a workspace that has an apply run in progress.
	if workspace.CurrentApplyRunID != nil {
		return nil, ErrWorkspaceHasCurrentApplyRun
	}

	// Update the field.
	workspace.Locked = true

	s.logger.WithContextFields(ctx).Infow("Requested a lock on workspace.",
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer LockWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update workspace", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionLock,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return updatedWorkspace, nil
}

func (s *service) UnlockWorkspace(ctx context.Context, workspace *models.Workspace) (*models.Workspace, error) {
	ctx, span := tracer.Start(ctx, "svc.UnlockWorkspace")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateWorkspacePermission, auth.WithWorkspaceID(workspace.Metadata.ID))
	if err != nil {
		return nil, err
	}

	// Check if workspace is already unlocked.
	if !workspace.Locked {
		return nil, ErrWorkspaceUnlocked
	}

	// Update the field.
	workspace.Locked = false

	s.logger.WithContextFields(ctx).Infow("Requested an unlock on workspace.",
		"fullPath", workspace.FullPath,
		"workspaceID", workspace.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UnlockWorkspace: %v", txErr)
		}
	}()

	updatedWorkspace, err := s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update workspace", errors.WithSpan(span))
	}

	// Now that the workspace is unlocked, ask the run work item consumer to re-evaluate it
	// so any runs that were waiting on the lock get queued (the work item consumer runs the
	// QueueRun command for the workspace's pending runs).
	if _, err = s.dbClient.WorkItemsQueue.AddWorkItemToQueue(txContext, &db.AddWorkItemToQueueInput{
		Type:    db.QueuePendingRunsForWorkspaceType,
		Payload: &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: updatedWorkspace.Metadata.ID},
	}); err != nil {
		return nil, errors.Wrap(err, "failed to enqueue pending runs work item", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &updatedWorkspace.FullPath,
			Action:        models.ActionUnlock,
			TargetType:    models.TargetWorkspace,
			TargetID:      updatedWorkspace.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return updatedWorkspace, nil
}

func (s *service) GetCurrentStateVersion(ctx context.Context, workspaceID string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetCurrentStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	workspace, err := s.getWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by ID", errors.WithSpan(span))
	}

	if workspace == nil || workspace.CurrentStateVersionID == "" {
		return nil, nil
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	return s.getStateVersionByID(ctx, workspace.CurrentStateVersionID)
}

func (s *service) GetStateVersionInventory(ctx context.Context, stateVersion *models.StateVersion) (*StateVersionInventory, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionInventory")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	reader, err := s.artifactStore.GetStateVersion(ctx, stateVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version", errors.WithSpan(span))
	}
	defer reader.Close()

	var state stateV4
	if err := json.NewDecoder(reader).Decode(&state); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal decoded data", errors.WithSpan(span))
	}

	if state.Version != version4 {
		return nil, errors.New("expected stateVersionV4, got %d", state.Version, errors.WithSpan(span))
	}

	// Extract resources
	resources := []*StateVersionResource{}
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
			return nil, errors.New(
				"invalid provider config encountered when parsing state version resources %s", r.ProviderConfig,
				errors.WithSpan(span))
		}
		endIndex := strings.LastIndex(r.ProviderConfig, "\"]")
		if endIndex == -1 {
			return nil, errors.New(
				"invalid provider config encountered when parsing state version resources %s", r.ProviderConfig,
				errors.WithSpan(span))
		}

		resource.Provider = r.ProviderConfig[startIndex+2 : endIndex]

		resources = append(resources, &resource)
	}

	// Extract dependencies
	dependencies := []*StateVersionDependency{}
	for _, r := range state.Resources {
		if r.ProviderConfig == tharsisTerraformProviderConfig && r.Type == tharsisWorkspaceOutputsDatasourceName && len(r.Instances) > 0 {
			attributes := map[string]interface{}{}
			if err := json.Unmarshal(r.Instances[0].AttributesRaw, &attributes); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal attributes for tharsis terraform provider", errors.WithSpan(span))
			}

			fullPathVal, exists := attributes["full_path"]
			if !exists {
				return nil, errors.New("full_path attribute missing from %s resource %s", r.Type, r.Name, errors.WithSpan(span))
			}

			stateVersionIDVal, exists := attributes["state_version_id"]
			if !exists {
				return nil, errors.New("state_version_id attribute missing from %s resource %s", r.Type, r.Name, errors.WithSpan(span))
			}

			workspaceIDVal, exists := attributes["workspace_id"]
			if !exists {
				return nil, errors.New("workspace_id attribute missing from %s resource %s", r.Type, r.Name, errors.WithSpan(span))
			}

			// Skip resources with nil attribute values (e.g., stale data source references).
			if fullPathVal == nil || stateVersionIDVal == nil || workspaceIDVal == nil {
				continue
			}

			fullPath, ok := fullPathVal.(string)
			if !ok {
				return nil, errors.New("full_path attribute is not a string in %s resource %s", r.Type, r.Name, errors.WithSpan(span))
			}

			stateVersionID, ok := stateVersionIDVal.(string)
			if !ok {
				return nil, errors.New("state_version_id attribute is not a string in %s resource %s", r.Type, r.Name, errors.WithSpan(span))
			}

			workspaceID, ok := workspaceIDVal.(string)
			if !ok {
				return nil, errors.New("workspace_id attribute is not a string in %s resource %s", r.Type, r.Name, errors.WithSpan(span))
			}

			dependencies = append(dependencies, &StateVersionDependency{
				WorkspacePath:  fullPath,
				WorkspaceID:    gid.FromGlobalID(workspaceID),
				StateVersionID: gid.FromGlobalID(stateVersionID),
			})
		}
	}

	// Extract check results
	checkResults := []*corerun.CheckResult{}
	for _, cr := range state.CheckResults {
		objects := []corerun.CheckResultObject{}
		for _, obj := range cr.Objects {
			objects = append(objects, corerun.CheckResultObject{
				Address:         obj.ObjectAddr,
				Status:          corerun.NormalizeCheckStatus(obj.Status),
				FailureMessages: obj.FailureMessages,
			})
		}
		checkResults = append(checkResults, &corerun.CheckResult{
			Name:    cr.ConfigAddr,
			Status:  corerun.NormalizeCheckStatus(cr.Status),
			Objects: objects,
		})
	}

	return &StateVersionInventory{
		Resources:    resources,
		Dependencies: dependencies,
		CheckResults: checkResults,
	}, nil
}

func (s *service) CreateStateVersion(ctx context.Context, stateVersion *models.StateVersion, data string, jsonData *string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	// We need to decode the base64 encoded string
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decoded base64-encoded state version", errors.WithSpan(span))
	}

	// Decode and validate the state, and pre-compute which outputs to persist,
	// before opening the DB transaction. Filtering ahead of time keeps the
	// transaction short: only the already-approved outputs are written inside
	// it, rather than decoding and enforcing limits while holding a DB
	// connection.
	var state stateV4
	if err = json.Unmarshal(decoded, &state); err != nil {
		return nil, errors.New("failed to unmarshal decoded data: %s", err, errors.WithSpan(span))
	}
	if state.Version != version4 {
		return nil, errors.New("expected stateVersionV4, got %d", state.Version, errors.WithSpan(span))
	}

	// jsonData is the optional "terraform show -json" rendering of the same state, base64 encoded like
	// data. Decoding it here, before anything is written, means a malformed payload costs nothing.
	var decodedJSON []byte
	if jsonData != nil && *jsonData != "" {
		decodedJSON, err = base64.StdEncoding.DecodeString(*jsonData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode base64-encoded state version JSON",
				errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
	}

	// Collect limit violations. Outputs that violate a limit are not persisted
	// as individually fetchable outputs, but the full state blob is still
	// uploaded below. Failing to upload the state would corrupt the workspace
	// because infrastructure changes have already been applied at this point.
	var limitErrors error

	// Look up the max-outputs-per-state-version limit value. This is a read, so
	// it does not need the transaction. We fetch the value directly (rather than
	// using limitChecker.CheckLimit) because enforcement keeps up to that many
	// outputs, so we need the number itself. If this lookup fails, we log it and
	// treat it as "no limit configured" rather than aborting: a failure here
	// usually indicates a broader DB issue, in which case the subsequent writes
	// in this function (state version, workspace update, outputs) will likely
	// fail too and surface a real error; aborting early here would only pay off
	// in the narrower case where this lookup fails but every later DB write
	// still succeeds. This is an intentional deviation from limitChecker.CheckLimit's
	// fail-closed pattern — fail-open is acceptable here because state must always
	// be persisted after an apply.
	outputCountLimit, limitErr := s.dbClient.ResourceLimits.GetResourceLimit(ctx, string(limits.ResourceLimitOutputsPerStateVersion))
	if limitErr != nil {
		s.logger.WithContextFields(ctx).Errorf(
			"failed to look up %s; skipping output count enforcement for this state version: %v",
			limits.ResourceLimitOutputsPerStateVersion, limitErr,
		)
		outputCountLimit = nil
	}

	// Sort output names so enforcement is deterministic: map iteration order in
	// Go is randomized, so without sorting, "the first N outputs" would be a
	// different set on every run. Sorting by name guarantees the same outputs
	// are kept each time.
	outputNames := make([]string, 0, len(state.RootOutputs))
	for outputName := range state.RootOutputs {
		outputNames = append(outputNames, outputName)
	}
	sort.Strings(outputNames)

	// Build the filtered list of outputs to persist. StateVersionID is filled in
	// later, once the state version has been created inside the transaction.
	outputsToStore := make([]models.StateVersionOutput, 0, len(outputNames))
	for _, outputName := range outputNames {
		outputInfo := state.RootOutputs[outputName]

		// Skip an output that exceeds the size limit. The value is still present
		// in the uploaded state blob; it is only omitted from the individually
		// fetchable outputs returned over the size-limited gRPC transport.
		// Oversized outputs do not consume a slot toward the count limit.
		if len(outputInfo.ValueRaw) > maxStateVersionOutputSizeBytes {
			limitErrors = goerrors.Join(limitErrors, fmt.Errorf(
				"output %q size %d exceeds maximum allowed size of %d bytes", outputName, len(outputInfo.ValueRaw), maxStateVersionOutputSizeBytes,
			))
			continue
		}

		// Enforce the count limit: keep up to outputCountLimit.Value size-valid
		// outputs (in sorted order) and skip the rest. The skipped values remain
		// in the uploaded state blob; they are only omitted from the individually
		// fetchable outputs.
		if outputCountLimit != nil && len(outputsToStore) >= outputCountLimit.Value {
			continue
		}

		outputsToStore = append(outputsToStore, models.StateVersionOutput{
			Name:      outputName,
			Value:     outputInfo.ValueRaw,
			Type:      outputInfo.ValueTypeRaw,
			Sensitive: outputInfo.Sensitive,
		})
	}

	// Report a count-limit violation if the total number of outputs (including
	// oversized ones) exceeds the configured limit.
	if outputCountLimit != nil && len(state.RootOutputs) > outputCountLimit.Value {
		limitErrors = goerrors.Join(limitErrors, fmt.Errorf(
			"state version has %d outputs, exceeding the limit of %d; kept the first %d sorted by name",
			len(state.RootOutputs), outputCountLimit.Value, len(outputsToStore),
		))
	}

	var jsonRetainFn db.RetainObjectRefFunc
	if decodedJSON != nil {
		retainFn, jsonKey, jsonErr := s.artifactStore.UploadStateVersionJSON(ctx, stateVersion, bytes.NewBuffer(decodedJSON))
		if jsonErr != nil {
			return nil, errors.Wrap(jsonErr, "failed to write state version JSON to object storage", errors.WithSpan(span))
		}
		jsonRetainFn = retainFn
		stateVersion.JSONObjectStoreKey = &jsonKey
	}

	// Upload before the transaction so the pending ref is committed immediately.
	// If the TX rolls back, the janitor will clean up the orphaned S3 object.
	svRetainFn, svKey, err := s.artifactStore.UploadStateVersion(ctx, stateVersion, bytes.NewBuffer(decoded))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to write state version to object storage", errors.WithSpan(span))
	}
	stateVersion.ObjectStoreKey = svKey

	// Wrap a transaction around persisting the state version and the state version outputs.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateStateVersion: %v", txErr)
		}
	}()

	// Update the CreatedBy field since a state version could be created manually.
	stateVersion.CreatedBy = caller.GetSubject()

	createdStateVersion, err := s.dbClient.StateVersions.CreateStateVersion(txContext, stateVersion)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create state version", errors.WithSpan(span))
	}

	if err = svRetainFn(txContext, createdStateVersion.Metadata.ID); err != nil {
		return nil, errors.Wrap(err, "failed to link state version object store ref", errors.WithSpan(span))
	}

	if jsonRetainFn != nil {
		if err = jsonRetainFn(txContext, createdStateVersion.Metadata.ID); err != nil {
			return nil, errors.Wrap(err, "failed to link state version JSON object store ref", errors.WithSpan(span))
		}
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
		return nil, errors.Wrap(err, "failed to get workspace's state versions", errors.WithSpan(span))
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitStateVersionsPerWorkspacePerTimePeriod, recentStateVersions.PageInfo.TotalCount); err != nil {
		return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
	}

	// Update the current state version field on the workspace.
	// This is a read-only operation, so there's no need to use the transaction context.
	workspace, wErr := s.getWorkspaceByID(ctx, createdStateVersion.WorkspaceID)
	if wErr != nil {
		return nil, errors.Wrap(wErr, "failed to get workspace by ID", errors.WithSpan(span))
	}

	workspace.DirtyState = false
	workspace.CurrentStateVersionID = createdStateVersion.Metadata.ID

	// Update the workspace and ignore the returned model since its not needed.
	_, err = s.dbClient.Workspaces.UpdateWorkspace(txContext, workspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update workspace", errors.WithSpan(span))
	}

	// Persist the pre-filtered outputs. All decoding and limit enforcement was
	// done before the transaction was opened; here we only write the approved
	// outputs, keeping the transaction short.
	for i := range outputsToStore {
		outputsToStore[i].StateVersionID = createdStateVersion.Metadata.ID

		// There's nothing that needs to be done with the stored new output, so ignore it.
		if _, err = s.dbClient.StateVersionOutputs.CreateStateVersionOutput(txContext, &outputsToStore[i]); err != nil {
			return nil, errors.Wrap(err, "failed to create state version output", errors.WithSpan(span))
		}
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetStateVersion,
			TargetID:      createdStateVersion.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	// Commit the transaction here.  If the upload fails, the transaction will be aborted.
	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a new state version",
		"stateVersionID", createdStateVersion.Metadata.ID,
		"workspaceID", createdStateVersion.WorkspaceID,
		"workspaceFullPath", workspace.FullPath,
	)

	// Return limit violations after the state version is safely persisted. The
	// state version is not returned here (unlike other successful paths) even
	// though it was created: the gRPC server handler and generated client stub
	// both discard the response value whenever err is non-nil, so no caller can
	// ever observe a non-nil value on this path. Returning it would only be
	// meaningful if a caller could receive both a value and an error together,
	// which isn't supported without a proto/API change (see MR discussion).
	if limitErrors != nil {
		return nil, errors.New(
			"%s: %s", coreworkspace.StateVersionOutputLimitViolationMsg, limitErrors.Error(),
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	return createdStateVersion, nil
}

func (s *service) GetStateVersionByID(ctx context.Context, stateVersionID string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, stateVersionID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to query state version from the database", errors.WithSpan(span))
	}

	if sv == nil {
		return nil, errors.New("state version with ID %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return sv, nil
}

func (s *service) GetStateVersionByTRN(ctx context.Context, trn string) (*models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersionByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version")
	}

	if sv == nil {
		return nil, errors.New(
			"state version with TRN %s not found", trn,
			errors.WithErrorCode(errors.ENotFound),
			errors.WithSpan(span),
		)
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
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
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(input.Workspace.Metadata.ID))
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
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionContent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, stateVersionID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to query state version from the database", errors.WithSpan(span))
	}

	if sv == nil {
		return nil, errors.New("state version with ID %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionDataPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetStateVersion(ctx, sv)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get state version from artifact store", errors.WithSpan(span))
	}

	return result, nil
}

// UploadStateVersionJSON stores the "terraform show -json" rendering of a state version that already
// exists. It is a separate request from CreateStateVersion on purpose: that call carries the raw state
// base64-encoded inside a single gRPC message, so bundling a second, typically larger, payload into it
// would halve the state size the transport can carry and would couple the two — an oversized rendering
// would fail the write of the state itself, at a point where infrastructure has already changed.
func (s *service) UploadStateVersionJSON(ctx context.Context, stateVersionID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadStateVersionJSON")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, stateVersionID)
	if err != nil {
		return errors.Wrap(err, "failed to query state version from the database", errors.WithSpan(span))
	}

	if sv == nil {
		return errors.New("state version with ID %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Writing the rendering is part of recording state, so it takes the same permission as creating the
	// state version rather than a weaker view permission.
	if err = caller.RequirePermission(ctx, models.CreateStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID)); err != nil {
		return err
	}

	// Upload before the transaction so the pending ref is committed immediately. If the transaction
	// rolls back, the janitor collects the orphaned object.
	retainFn, key, err := s.artifactStore.UploadStateVersionJSON(ctx, sv, reader)
	if err != nil {
		return errors.Wrap(err, "failed to write state version JSON to object storage", errors.WithSpan(span))
	}

	sv.JSONObjectStoreKey = &key

	// Recording the key and retaining the object have to happen together: a key with no ref would let
	// the janitor collect an object the row still points at, and a ref with no key would leak an object
	// nothing can reach.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for UploadStateVersionJSON: %v", txErr)
		}
	}()

	if _, err = s.dbClient.StateVersions.UpdateStateVersion(txContext, sv); err != nil {
		return errors.Wrap(err, "failed to update state version", errors.WithSpan(span))
	}

	if err = retainFn(txContext, sv.Metadata.ID); err != nil {
		return errors.Wrap(err, "failed to link state version JSON object store ref", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return nil
}

func (s *service) GetStateVersionJSONContent(ctx context.Context, stateVersionID string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionJSONContent")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, stateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query state version from the database", errors.WithSpan(span))
	}

	if sv == nil {
		return nil, errors.New("state version with ID %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, models.ViewStateVersionDataPermission, auth.WithWorkspaceID(sv.WorkspaceID)); err != nil {
		return nil, err
	}

	// A missing rendering is an expected state, not a failure: it means nothing ever uploaded one. It
	// surfaces as ENotFound so callers can distinguish "no rendering" from a storage error and decide
	// for themselves whether to fall back.
	if sv.JSONObjectStoreKey == nil {
		return nil, errors.New(
			"state version with ID %s has no JSON rendering", stateVersionID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span),
		)
	}

	result, err := s.artifactStore.GetStateVersionJSON(ctx, sv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version JSON from artifact store", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) GetWorkspaceAssessmentByID(ctx context.Context, id string) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceAssessmentByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	assessment, err := s.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if assessment == nil {
		return nil, errors.New("workspace assessment with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	if err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(assessment.WorkspaceID)); err != nil {
		return nil, err
	}

	return assessment, nil
}

func (s *service) GetWorkspaceAssessmentByTRN(ctx context.Context, trn string) (*models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceAssessmentByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	assessment, err := s.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace assessment", errors.WithSpan(span))
	}

	if assessment == nil {
		return nil, errors.New(
			"workspace assessment with TRN %s not found", trn,
			errors.WithErrorCode(errors.ENotFound),
			errors.WithSpan(span),
		)
	}

	if err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(assessment.WorkspaceID)); err != nil {
		return nil, err
	}

	return assessment, nil
}

func (s *service) GetWorkspaceAssessmentsByWorkspaceIDs(ctx context.Context, idList []string) ([]models.WorkspaceAssessment, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceAssessmentsByWorkspaceIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &db.GetWorkspaceAssessmentsInput{
		Filter: &db.WorkspaceAssessmentFilter{
			WorkspaceIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get workspace assessments",
			errors.WithSpan(span),
		)
	}

	for _, a := range result.WorkspaceAssessments {
		err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithWorkspaceID(a.WorkspaceID))
		if err != nil {
			return nil, err
		}
	}

	return result.WorkspaceAssessments, nil
}

func (s *service) GetStateVersionsByIDs(ctx context.Context,
	idList []string,
) ([]models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

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
		return nil, errors.Wrap(
			err,
			"Failed to get state versions", errors.WithSpan(span))
	}

	for _, sv := range result.StateVersions {
		err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
		if err != nil {
			return nil, err
		}
	}

	return result.StateVersions, nil
}

func (s *service) GetConfigurationVersionContent(ctx context.Context, configurationVersionID string) (*GetConfigurationVersionContentOutput, error) {
	ctx, span := tracer.Start(ctx, "svc.GetConfigurationVersionContent")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	cv, err := s.GetConfigurationVersionByID(ctx, configurationVersionID)
	if err != nil {
		return nil, err
	}

	body, contentLength, err := s.artifactStore.GetConfigurationVersion(ctx, cv)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get configuration version from artifact store", errors.WithSpan(span))
	}

	return &GetConfigurationVersionContentOutput{
		Body:          body,
		ContentLength: contentLength,
	}, nil
}

// CreateConfigurationVersion creates a new configuration version
func (s *service) CreateConfigurationVersion(ctx context.Context, options *CreateConfigurationVersionInput) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateConfigurationVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateConfigurationVersionPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	// Wrap a transaction around persisting the new configuration version.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateConfigurationVersion: %v", txErr)
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
		return nil, errors.Wrap(err, "failed to create configuration version", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to get workspace's configuration versions", errors.WithSpan(span))
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitConfigurationVersionsPerWorkspacePerTimePeriod, recentCVs.PageInfo.TotalCount); err != nil {
		return nil, errors.Wrap(err, "limit check failed", errors.WithSpan(span))
	}

	// Commit the transaction here.
	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a configuration version.",
		"workspaceID", options.WorkspaceID,
		"configurationVersionID", cv.Metadata.ID,
	)
	return cv, nil
}

// GetConfigurationVersionByID returns a tfe configuration version
func (s *service) GetConfigurationVersionByID(ctx context.Context, configurationVersionID string) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetConfigurationVersionByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	cv, err := s.dbClient.ConfigurationVersions.GetConfigurationVersionByID(ctx, configurationVersionID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get configuration version", errors.WithSpan(span))
	}

	if cv == nil {
		return nil, errors.New(
			"Configuration version with ID %s not found", configurationVersionID,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return cv, nil
}

func (s *service) GetConfigurationVersionByTRN(ctx context.Context, configurationVersionID string) (*models.ConfigurationVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetConfigurationVersionByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	cv, err := s.dbClient.ConfigurationVersions.GetConfigurationVersionByTRN(ctx, configurationVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get configuration version", errors.WithSpan(span))
	}

	if cv == nil {
		return nil, errors.New("configuration version with TRN %s not found",
			configurationVersionID,
			errors.WithErrorCode(errors.ENotFound),
			errors.WithSpan(span),
		)
	}

	if err = caller.RequirePermission(ctx, models.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID)); err != nil {
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
		return nil, err
	}

	result, err := s.dbClient.ConfigurationVersions.GetConfigurationVersions(ctx, &db.GetConfigurationVersionsInput{
		Filter: &db.ConfigurationVersionFilter{
			ConfigurationVersionIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get configuration versions", errors.WithSpan(span))
	}

	for _, cv := range result.ConfigurationVersions {
		err = caller.RequirePermission(ctx, models.ViewConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
		if err != nil {
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
		return err
	}

	cv, err := s.GetConfigurationVersionByID(ctx, configurationVersionID)
	if err != nil {
		return errors.Wrap(err, "failed to get configuration version", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.UpdateConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID))
	if err != nil {
		return err
	}

	if cv.Status != models.ConfigurationPending {
		return errors.New(
			"configuration version is already uploaded",
			errors.WithErrorCode(errors.EConflict),
			errors.WithSpan(span),
		)
	}

	cvRetainFn, cvKey, err := s.artifactStore.UploadConfigurationVersion(ctx, cv, reader)
	if err != nil {
		return errors.Wrap(err, "Failed to write configuration version to object storage", errors.WithSpan(span))
	}

	cv.ObjectStoreKey = cvKey
	// Update status of configuration version to uploaded
	cv.Status = models.ConfigurationUploaded

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}
	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for UploadConfigurationVersion: %v", txErr)
		}
	}()

	if _, err := s.dbClient.ConfigurationVersions.UpdateConfigurationVersion(txContext, *cv); err != nil {
		return errors.Wrap(
			err,
			"Failed to to update configuration version", errors.WithSpan(span))
	}

	if err := cvRetainFn(txContext, cv.Metadata.ID); err != nil {
		return errors.Wrap(err, "failed to link configuration version object store ref", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Uploaded a configuration version.",
		"workspaceID", cv.WorkspaceID,
		"configurationVersionID", cv.Metadata.ID,
	)
	return nil
}

func (s *service) GetStateVersionOutputByID(ctx context.Context, id string) (*models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionOutputByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	output, err := s.dbClient.StateVersionOutputs.GetStateVersionOutputByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version output", errors.WithSpan(span))
	}

	if output == nil {
		return nil, errors.New("state version output with ID %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	stateVersion, err := s.getStateVersionByID(ctx, output.StateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (s *service) GetStateVersionOutputByTRN(ctx context.Context, trn string) (*models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionOutputByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	output, err := s.dbClient.StateVersionOutputs.GetStateVersionOutputByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version output", errors.WithSpan(span))
	}

	if output == nil {
		return nil, errors.New("state version output with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	stateVersion, err := s.getStateVersionByID(ctx, output.StateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(stateVersion.WorkspaceID))
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (s *service) GetStateVersionOutputs(ctx context.Context, stateVersionID string) ([]models.StateVersionOutput, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionOutputs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := s.getStateVersionByID(ctx, stateVersionID)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewStateVersionPermission, auth.WithWorkspaceID(sv.WorkspaceID))
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.StateVersionOutputs.GetStateVersionOutputs(ctx, stateVersionID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"failed to list state version outputs", errors.WithSpan(span))
	}

	return result, nil
}

// GetRunnerTagsSetting returns the (inherited or direct) runner tags setting for a workspace.
func (s *service) GetRunnerTagsSetting(ctx context.Context, workspace *models.Workspace) (*namespace.RunnerTagsSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunnerTagsSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	return s.inheritedSettingsResolver.GetRunnerTags(ctx, workspace)
}

// GetDetectionEnabledSetting returns the (inherited or direct) setting for a group.
func (s *service) GetDriftDetectionEnabledSetting(ctx context.Context, workspace *models.Workspace) (*namespace.DriftDetectionEnabledSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetDriftDetectionEnabledSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	return s.inheritedSettingsResolver.GetDriftDetectionEnabled(ctx, workspace)
}

// GetProviderMirrorEnabledSetting returns the (inherited or direct) provider mirror setting for a workspace.
func (s *service) GetProviderMirrorEnabledSetting(ctx context.Context, workspace *models.Workspace) (*namespace.ProviderMirrorEnabledSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderMirrorEnabledSetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	return s.inheritedSettingsResolver.GetProviderMirrorEnabled(ctx, workspace)
}

// GetOutputVisibilitySetting returns the (inherited or direct) output visibility setting for a workspace.
func (s *service) GetOutputVisibilitySetting(ctx context.Context, workspace *models.Workspace) (*namespace.OutputVisibilitySetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetOutputVisibilitySetting")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(workspace.FullPath))
	if err != nil {
		return nil, err
	}

	setting, err := s.inheritedSettingsResolver.GetOutputVisibility(ctx, workspace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get output visibility setting", errors.WithSpan(span))
	}

	return setting, nil
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
		return nil, err
	}

	// The caller must have CreateWorkspacePermission in the new parent.
	err = caller.RequirePermission(ctx, models.CreateWorkspacePermission, auth.WithGroupID(newGroupID))
	if err != nil {
		return nil, err
	}

	// The caller must also have CreateNamespaceMembershipPermission in the new parent. Moving a
	// workspace in introduces principals the destination's administrator never approved, and it
	// changes output visibility relationships, which are derived from namespace paths and group IDs.
	// Both are access decisions that belong to whoever controls access at the destination, so this is
	// checked in addition to (not instead of) CreateWorkspacePermission.
	//
	// Only the destination is checked. Moving a workspace out already requires
	// DeleteWorkspacePermission on it, and a caller who can delete the workspace outright gains
	// nothing by moving it.
	err = caller.RequirePermission(ctx, models.CreateNamespaceMembershipPermission, auth.WithGroupID(newGroupID))
	if err != nil {
		return nil, err
	}

	// Caller must have DeleteWorkspacePermission in the workspace being moved.
	err = caller.RequirePermission(ctx, models.DeleteWorkspacePermission, auth.WithWorkspaceID(workspaceID))
	if err != nil {
		return nil, err
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

	s.logger.WithContextFields(ctx).Infow("Requested a workspace migration.",
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
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer MigrateWorkspace: %v", txErr)
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
	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
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

func (s *service) GetWorkspaceRoleBindingByID(ctx context.Context, id string) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceRoleBindingByID")
	defer span.End()

	binding, err := s.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindingByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace role binding by ID", errors.WithSpan(span))
	}

	if binding == nil {
		return nil, errors.New("workspace role binding with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ViewWorkspaceRoleBindingPermission, auth.WithWorkspaceID(binding.WorkspaceID)); err != nil {
		return nil, err
	}

	return binding, nil
}

func (s *service) GetWorkspaceRoleBindingByTRN(ctx context.Context, trn string) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceRoleBindingByTRN")
	defer span.End()

	binding, err := s.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindingByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace role binding by TRN", errors.WithSpan(span))
	}

	if binding == nil {
		return nil, errors.New("workspace role binding with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ViewWorkspaceRoleBindingPermission, auth.WithWorkspaceID(binding.WorkspaceID)); err != nil {
		return nil, err
	}

	return binding, nil
}

func (s *service) GetWorkspaceRoleBindingByWorkspaceID(ctx context.Context, workspaceID string) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceRoleBindingByWorkspaceID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ViewWorkspaceRoleBindingPermission, auth.WithWorkspaceID(workspaceID)); err != nil {
		return nil, err
	}

	binding, err := s.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindingByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace role binding by workspace ID", errors.WithSpan(span))
	}

	// A workspace legitimately has no binding; that is not an error.
	return binding, nil
}

// GetWorkspaceRoleBindingsByWorkspaceIDs returns the role bindings for a batch of workspaces, used
// by the WorkspaceRoleBinding dataloader. Workspaces with no binding are simply absent from the
// result; that is not an error, and callers must not treat a missing entry as EnotFound.
func (s *service) GetWorkspaceRoleBindingsByWorkspaceIDs(ctx context.Context, idList []string) ([]models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceRoleBindingsByWorkspaceIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindings(ctx, &db.GetWorkspaceRoleBindingsInput{
		Filter: &db.WorkspaceRoleBindingFilter{
			WorkspaceIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace role bindings by workspace IDs", errors.WithSpan(span))
	}

	if err = s.requireViewWorkspaceRoleBindingAccess(ctx, caller, result.WorkspaceRoleBindings); err != nil {
		return nil, err
	}

	return result.WorkspaceRoleBindings, nil
}

// GetWorkspaceRoleBindingsByIDs returns the role bindings with the given IDs, used by the
// WorkspaceRoleBinding-as-activity-event-target loader, which looks a binding up starting from its
// own ID (the activity event's target ID) rather than from its workspace. Unlike
// GetWorkspaceRoleBindingsByWorkspaceIDs, a missing entry here IS meaningful to the caller (the
// binding no longer exists, e.g. it was later removed), so this returns exactly the bindings found
// with no guarantee every requested ID is present — the caller (the activity event resolver) treats
// an absent ID as "target no longer exists," the same as it does for every other target type.
func (s *service) GetWorkspaceRoleBindingsByIDs(ctx context.Context, idList []string) ([]models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceRoleBindingsByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindings(ctx, &db.GetWorkspaceRoleBindingsInput{
		Filter: &db.WorkspaceRoleBindingFilter{
			IDs: idList,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace role bindings by IDs", errors.WithSpan(span))
	}

	if err = s.requireViewWorkspaceRoleBindingAccess(ctx, caller, result.WorkspaceRoleBindings); err != nil {
		return nil, err
	}

	return result.WorkspaceRoleBindings, nil
}

// SetWorkspaceRoleBinding creates, changes, or removes the role bound to a workspace. The bound
// role's permissions become available to the workspace's job caller at the workspace's DIRECT
// PARENT namespace (see models.WorkspaceRoleBinding for why the binding carries no namespace of its
// own).
//
// Authorization is two-part and BOTH checks are against the workspace's PARENT namespace, never
// against the workspace itself:
//
//  1. The caller must hold a WorkspaceRoleBinding permission (Create/Update/Delete as appropriate)
//     at the parent namespace. This is the permission that governs conferring a binding, and by
//     default only Owner holds it.
//  2. The caller must hold every permission contained in the role being bound, at the parent
//     namespace. This is what stops a caller from binding a role broader than their own access —
//     the actual escalation-prevention check.
//
// Checking (1) with WithWorkspaceID instead of WithGroupID(workspace.GroupID) would be a critical
// bug: it would let a caller whose membership is scoped to just the workspace confer authority over
// the workspace's PARENT group, which is a namespace they may have no access to at all.
//
// A caller changing an existing binding to a different role must satisfy both checks again for the
// new role. There is no in-place role swap that skips re-validation.
func (s *service) SetWorkspaceRoleBinding(ctx context.Context, options *SetWorkspaceRoleBindingInput) (*models.WorkspaceRoleBinding, error) {
	ctx, span := tracer.Start(ctx, "svc.SetWorkspaceRoleBinding")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, options.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by ID", errors.WithSpan(span))
	}

	if workspace == nil {
		return nil, errors.New("workspace with id %s not found", options.WorkspaceID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	existing, err := s.dbClient.WorkspaceRoleBindings.GetWorkspaceRoleBindingByWorkspaceID(ctx, options.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing workspace role binding", errors.WithSpan(span))
	}

	if options.RoleID == nil {
		return s.removeWorkspaceRoleBinding(ctx, caller, workspace, existing)
	}

	return s.createOrUpdateWorkspaceRoleBinding(ctx, caller, workspace, existing, *options.RoleID)
}

// createOrUpdateWorkspaceRoleBinding performs both halves of the escalation check against the
// workspace's parent namespace, then creates or updates the binding row.
func (s *service) createOrUpdateWorkspaceRoleBinding(
	ctx context.Context,
	caller auth.Caller,
	workspace *models.Workspace,
	existing *models.WorkspaceRoleBinding,
	roleID string,
) (*models.WorkspaceRoleBinding, error) {
	bindingPerm := models.CreateWorkspaceRoleBindingPermission
	action := models.ActionCreate
	if existing != nil {
		bindingPerm = models.UpdateWorkspaceRoleBindingPermission
		action = models.ActionUpdate
	}

	// Part 1: the caller must hold the permission that governs conferring a binding, at the PARENT
	// namespace. WithGroupID, never WithWorkspaceID.
	if err := caller.RequirePermission(ctx, bindingPerm, auth.WithGroupID(workspace.GroupID)); err != nil {
		return nil, err
	}

	role, err := s.dbClient.Roles.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get role by ID")
	}

	if role == nil {
		return nil, errors.New("role with id %s not found", roleID, errors.WithErrorCode(errors.ENotFound))
	}

	// Part 2: the caller must hold every permission the role would confer, at the PARENT namespace.
	// This is the check that prevents a caller from binding a role broader than their own access.
	if err := s.requireEffectivePermissionSuperset(ctx, caller, workspace.GroupID, role); err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction")
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer SetWorkspaceRoleBinding: %v", txErr)
		}
	}()

	var result *models.WorkspaceRoleBinding
	var previousRoleID string

	if existing == nil {
		created, cErr := s.dbClient.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(txContext, &models.WorkspaceRoleBinding{
			WorkspaceID: workspace.Metadata.ID,
			RoleID:      roleID,
			CreatedBy:   caller.GetSubject(),
		})
		if cErr != nil {
			return nil, errors.Wrap(cErr, "failed to create workspace role binding")
		}
		result = created
	} else {
		previousRoleID = existing.RoleID
		toUpdate := *existing
		toUpdate.RoleID = roleID

		updated, uErr := s.dbClient.WorkspaceRoleBindings.UpdateWorkspaceRoleBinding(txContext, &toUpdate)
		if uErr != nil {
			return nil, errors.Wrap(uErr, "failed to update workspace role binding")
		}
		result = updated
	}

	groupPath := workspace.GetGroupPath()
	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        action,
			TargetType:    models.TargetWorkspaceRoleBinding,
			TargetID:      result.Metadata.ID,
			Payload: &models.ActivityEventSetWorkspaceRoleBindingPayload{
				PreviousRoleID: previousRoleID,
				NewRoleID:      roleID,
			},
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create an activity event")
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit a DB transaction")
	}

	s.logger.WithContextFields(ctx).Infow("Set workspace role binding.",
		"workspaceID", workspace.Metadata.ID,
		"roleID", roleID,
	)

	return result, nil
}

// removeWorkspaceRoleBinding removes a workspace's binding. Removing a binding never grants
// anything, so it is gated solely on DeleteWorkspaceRoleBindingPermission at the parent namespace —
// there is no role to subset-check against.
func (s *service) removeWorkspaceRoleBinding(
	ctx context.Context,
	caller auth.Caller,
	workspace *models.Workspace,
	existing *models.WorkspaceRoleBinding,
) (*models.WorkspaceRoleBinding, error) {
	if existing == nil {
		return nil, errors.New("workspace with id %s has no role binding to remove", workspace.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	if err := caller.RequirePermission(ctx, models.DeleteWorkspaceRoleBindingPermission, auth.WithGroupID(workspace.GroupID)); err != nil {
		return nil, err
	}

	role, err := s.dbClient.Roles.GetRoleByID(ctx, existing.RoleID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get role by ID")
	}

	roleName := existing.RoleID
	if role != nil {
		roleName = role.Name
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction")
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer SetWorkspaceRoleBinding: %v", txErr)
		}
	}()

	// Delete first, then record a DeleteChildResource event against the WORKSPACE the binding
	// belonged to — not against the binding itself, which no longer exists once deleted.
	if err = s.dbClient.WorkspaceRoleBindings.DeleteWorkspaceRoleBinding(txContext, existing); err != nil {
		return nil, errors.Wrap(err, "failed to delete workspace role binding")
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetWorkspace,
			TargetID:      workspace.Metadata.ID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: roleName,
				ID:   existing.Metadata.ID,
				Type: string(models.TargetWorkspaceRoleBinding),
			},
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create an activity event")
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit a DB transaction")
	}

	s.logger.WithContextFields(ctx).Infow("Removed workspace role binding.",
		"workspaceID", workspace.Metadata.ID,
	)

	return existing, nil
}

// requireEffectivePermissionSuperset returns an error unless the caller holds every permission in
// role, at namespacePath. This is the subset check: it must be evaluated against the role's actual
// permission set, not against a role ID comparison, so a custom role's permissions are checked the
// same way a default role's are.
func (s *service) requireEffectivePermissionSuperset(ctx context.Context, caller auth.Caller, groupID string, role *models.Role) error {
	if caller.IsAdminModeActivated(ctx) {
		return nil
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, groupID)
	if err != nil {
		return errors.Wrap(err, "failed to get group by ID")
	}

	if group == nil {
		return errors.New("group with id %s not found", groupID, errors.WithErrorCode(errors.ENotFound))
	}

	effective, err := caller.GetNamespacePermissions(ctx, group.FullPath)
	if err != nil {
		return errors.Wrap(err, "failed to get effective permissions")
	}

	missing := []string{}
	for _, perm := range role.GetPermissions() {
		required := perm
		satisfied := false
		for _, heldPerm := range effective {
			if heldPerm.GTE(&required) {
				satisfied = true
				break
			}
		}
		if !satisfied {
			missing = append(missing, required.String())
		}
	}

	if len(missing) > 0 {
		return errors.New(
			"cannot bind a role that grants permissions you do not hold at %s: missing %s",
			group.FullPath, strings.Join(missing, ", "),
			errors.WithErrorCode(errors.EForbidden),
		)
	}

	return nil
}

func (s *service) getStateVersionByID(ctx context.Context, stateVersionID string) (*models.StateVersion, error) {
	sv, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, stateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query state version by id")
	}

	if sv == nil {
		return nil, errors.New("state version with id %s not found", stateVersionID, errors.WithErrorCode(errors.ENotFound))
	}

	return sv, nil
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

// detectLabelChanges compares old and new labels and returns the changes
func detectLabelChanges(oldLabels, newLabels map[string]string) *models.LabelChangePayload {
	if oldLabels == nil {
		oldLabels = make(map[string]string)
	}
	if newLabels == nil {
		newLabels = make(map[string]string)
	}

	changes := &models.LabelChangePayload{
		Added:   make(map[string]string),
		Updated: make(map[string]string),
		Removed: []string{},
	}

	// Find added and updated labels
	for key, newValue := range newLabels {
		if oldValue, exists := oldLabels[key]; exists {
			if oldValue != newValue {
				changes.Updated[key] = newValue
			}
		} else {
			changes.Added[key] = newValue
		}
	}

	// Find removed labels
	for key := range oldLabels {
		if _, exists := newLabels[key]; !exists {
			changes.Removed = append(changes.Removed, key)
		}
	}

	// Return nil if no changes detected
	if len(changes.Added) == 0 && len(changes.Updated) == 0 && len(changes.Removed) == 0 {
		return nil
	}

	return changes
}

// requireViewWorkspaceRoleBindingAccess verifies the caller can view the given role bindings,
// batching the workspace-ID-to-path resolution and permission check into a single RequirePermission
// call (matching the pattern used by GetWorkspacesByIDs) instead of checking WithWorkspaceID for
// each binding individually, which would otherwise issue one path-resolving DB lookup and one
// permission check per binding for what is a single dataloader batch.
func (s *service) requireViewWorkspaceRoleBindingAccess(ctx context.Context, caller auth.Caller, bindings []models.WorkspaceRoleBinding) error {
	if len(bindings) == 0 {
		return nil
	}

	workspaceIDs := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		workspaceIDs = append(workspaceIDs, binding.WorkspaceID)
	}

	wsResult, err := s.dbClient.Workspaces.GetWorkspaces(ctx, &db.GetWorkspacesInput{
		Filter: &db.WorkspaceFilter{WorkspaceIDs: workspaceIDs},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get workspaces for role bindings")
	}

	wsPaths := make([]string, 0, len(wsResult.Workspaces))
	for _, ws := range wsResult.Workspaces {
		wsPaths = append(wsPaths, ws.FullPath)
	}

	if len(wsPaths) == 0 {
		return nil
	}

	return caller.RequirePermission(ctx, models.ViewWorkspaceRoleBindingPermission, auth.WithNamespacePaths(wsPaths))
}
