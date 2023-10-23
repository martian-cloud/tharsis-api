package run

//go:generate mockery --name Service --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// forceCancelWait is how long a run must be soft-canceled before it is allowed to be forcefully canceled.
	forceCancelWait = 30 * time.Minute
)

// Variable represents a run variable
type Variable struct {
	Value         *string                 `json:"value"`
	NamespacePath *string                 `json:"namespacePath"`
	Key           string                  `json:"key"`
	Category      models.VariableCategory `json:"category"`
	Hcl           bool                    `json:"hcl"`
}

// Event represents a run event
type Event struct {
	Action string
	Run    models.Run
}

// EventSubscriptionOptions provides options for subscribing to run events
type EventSubscriptionOptions struct {
	WorkspaceID *string
	RunID       *string // RunID is optional
}

// GetRunsInput is the input for querying a list of runs
type GetRunsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.RunSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Workspace filters the runs by the specified workspace
	Workspace *models.Workspace
	// Group filters the runs by the specified group
	Group *models.Group
}

// CreateRunInput is the input for creating a new run
type CreateRunInput struct {
	ConfigurationVersionID *string
	Comment                *string
	ModuleSource           *string
	ModuleVersion          *string
	WorkspaceID            string
	TerraformVersion       string
	Variables              []Variable
	TargetAddresses        []string
	IsDestroy              bool
	Refresh                bool
}

// Validate attempts to ensure the CreateRunInput structure is in good form and able to be used.
func (c CreateRunInput) Validate() error {

	// Check that there is at least one of configuration version and module source.
	if (c.ConfigurationVersionID == nil) && (c.ModuleSource == nil) {
		return fmt.Errorf("must supply either configuration version ID or module source")
	}

	// Check that there is no more than one of configuration version and module source.
	if (c.ConfigurationVersionID != nil) && (c.ModuleSource != nil) {
		return fmt.Errorf("must supply configuration version ID or module source but not both")
	}

	// Check that there is no more than one of configuration version and module version.
	if (c.ConfigurationVersionID != nil) && (c.ModuleVersion != nil) {
		return fmt.Errorf("must supply configuration version ID or module version but not both")
	}

	// Make sure module version is not specified without module source.
	if (c.ModuleSource == nil) && (c.ModuleVersion != nil) {
		return fmt.Errorf("module version is not allowed without module source")
	}

	// If a module version is specified, make sure it's a valid semver.
	if c.ModuleVersion != nil {
		_, err := semver.StrictNewVersion(*c.ModuleVersion)
		if err != nil {
			return fmt.Errorf("module version is not a valid semver string: %v", err)
		}
	}

	return nil
}

// CancelRunInput is the input for canceling a run.
type CancelRunInput struct {
	Comment *string
	RunID   string
	Force   bool
}

// Service encapsulates Terraform Enterprise Support
type Service interface {
	GetRun(ctx context.Context, runID string) (*models.Run, error)
	GetRuns(ctx context.Context, input *GetRunsInput) (*db.RunsResult, error)
	GetRunsByIDs(ctx context.Context, idList []string) ([]models.Run, error)
	CreateRun(ctx context.Context, options *CreateRunInput) (*models.Run, error)
	ApplyRun(ctx context.Context, runID string, comment *string) (*models.Run, error)
	CancelRun(ctx context.Context, options *CancelRunInput) (*models.Run, error)
	GetRunVariables(ctx context.Context, runID string) ([]Variable, error)
	GetPlansByIDs(ctx context.Context, idList []string) ([]models.Plan, error)
	GetPlan(ctx context.Context, planID string) (*models.Plan, error)
	UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error)
	DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error)
	UploadPlan(ctx context.Context, planID string, reader io.Reader) error
	GetAppliesByIDs(ctx context.Context, idList []string) ([]models.Apply, error)
	GetApply(ctx context.Context, applyID string) (*models.Apply, error)
	UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error)
	GetLatestJobForPlan(ctx context.Context, planID string) (*models.Job, error)
	GetLatestJobForApply(ctx context.Context, applyID string) (*models.Job, error)
	SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error)
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	artifactStore   workspace.ArtifactStore
	eventManager    *events.EventManager
	jobService      job.Service
	cliService      cli.Service
	runStateManager *state.RunStateManager
	activityService activityevent.Service
	moduleService   moduleregistry.Service
	moduleResolver  ModuleResolver
	ruleEnforcer    rules.RuleEnforcer
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	artifactStore workspace.ArtifactStore,
	eventManager *events.EventManager,
	jobService job.Service,
	cliService cli.Service,
	activityService activityevent.Service,
	moduleService moduleregistry.Service,
	moduleResolver ModuleResolver,
	runStateManager *state.RunStateManager,
) Service {
	return newService(
		logger,
		dbClient,
		artifactStore,
		eventManager,
		jobService,
		cliService,
		activityService,
		moduleService,
		moduleResolver,
		runStateManager,
		rules.NewRuleEnforcer(dbClient),
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	artifactStore workspace.ArtifactStore,
	eventManager *events.EventManager,
	jobService job.Service,
	cliService cli.Service,
	activityService activityevent.Service,
	moduleService moduleregistry.Service,
	moduleResolver ModuleResolver,
	runStateManager *state.RunStateManager,
	ruleEnforcer rules.RuleEnforcer,
) Service {
	return &service{
		logger,
		dbClient,
		artifactStore,
		eventManager,
		jobService,
		cliService,
		runStateManager,
		activityService,
		moduleService,
		moduleResolver,
		ruleEnforcer,
	}
}

func (s *service) SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error) {
	ctx, span := tracer.Start(ctx, "svc.SubscribeToRunEvents")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if options.WorkspaceID == nil {
		return nil, errors.New(errors.EInvalid, "WorkspaceID option is required")
	}

	err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithWorkspaceID(*options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	outgoing := make(chan *Event)

	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)

		subscription := events.Subscription{
			Type: events.RunSubscription,
			Actions: []events.SubscriptionAction{
				events.CreateAction,
				events.UpdateAction,
			},
		}
		subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

		defer s.eventManager.Unsubscribe(subscriber)

		// Wait for run updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) {
					s.logger.Errorf("Error occurred while waiting for run events: %v", err)
				}
				return
			}

			run, err := s.getRun(ctx, event.ID)
			if err != nil {
				if errors.IsContextCanceledError(err) {
					return
				}
				s.logger.Errorf("Error occurred while querying for run associated with run event %s: %v", event.ID, err)
				continue
			}

			// Check if run is associated with the desired workspace
			if run.WorkspaceID != *options.WorkspaceID {
				continue
			}

			if options.RunID != nil && run.Metadata.ID != *options.RunID {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &Event{Action: event.Action, Run: *run}:
			}
		}
	}()

	return outgoing, nil
}

// CreateRun creates a new run and associates a Plan with it
func (s *service) CreateRun(ctx context.Context, options *CreateRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	err := options.Validate()
	if err != nil {
		tracing.RecordError(span, err, "failed to validate create run options")
		return nil, err
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Build run variables
	runVariables, err := s.buildRunVariables(ctx, options.WorkspaceID, options.Variables)
	if err != nil {
		tracing.RecordError(span, err, "failed to build run variables")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"failed to build run variables",
		)
	}

	// Filter out the environment variables.
	runEnvVars := []Variable{}
	for _, variable := range runVariables {
		if variable.Category == models.EnvironmentVariableCategory {
			runEnvVars = append(runEnvVars, variable)
		}
	}

	// If a module source (and a registry-style source), resolve the module version.
	// This requires the run variables in order to have the token(s) for getting version numbers.
	// Handle the case where the run uses a module source rather than a configuration version.
	// If this fails, the transaction will be rolled back, so everything is safe.
	var moduleVersion *string
	var moduleDigest []byte
	var moduleRegistrySource *ModuleRegistrySource
	if options.ModuleSource != nil {
		moduleRegistrySource, err = s.moduleResolver.ParseModuleRegistrySource(ctx, *options.ModuleSource)
		if err != nil {
			tracing.RecordError(span, err, "failed to parse/resolve module source")
			return nil, errors.Wrap(err, errors.EInvalid, "failed to resolve module source")
		}

		// registry source will be nil if this is a remote module source that doesn't use the terraform module registry protocol
		if moduleRegistrySource != nil {
			var resolvedVersion string
			resolvedVersion, err = s.moduleResolver.ResolveModuleVersion(ctx, moduleRegistrySource, options.ModuleVersion, runEnvVars)
			if err != nil {
				tracing.RecordError(span, err, "failed to resolve module source")
				return nil, errors.Wrap(err, errors.EInvalid, "failed to resolve module source")
			}
			moduleVersion = &resolvedVersion

			// If this is a module stored in the local tharsis registry, we need to get the module version digest to pin the run to it to
			// prevent the module package from changing after the run has been created. This is an additional protection that is only available
			// for modules in the tharsis module registry
			if moduleRegistrySource.ModuleID != nil {
				var versionsResponse *db.ModuleVersionsResult
				versionsResponse, err = s.moduleService.GetModuleVersions(ctx, &moduleregistry.GetModuleVersionsInput{
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
					ModuleID:        *moduleRegistrySource.ModuleID,
					SemanticVersion: &resolvedVersion,
				})
				if err != nil {
					tracing.RecordError(span, err, "failed to get module versions")
					return nil, err
				}

				if len(versionsResponse.ModuleVersions) == 0 {
					return nil, errors.New(errors.EInternal, "unable to find the module package for module %s with semantic version %s", *options.ModuleSource, resolvedVersion)
				}

				moduleDigest = versionsResponse.ModuleVersions[0].SHASum
			}
		}
	}

	// Retrieve workspace to find Terraform version and max job duration.
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, options.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get workspace associated with run")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get workspace associated with run",
		)
	}

	if ws == nil {
		return nil, errors.New(
			errors.EInternal,
			"Failed to get workspace associated with run",
		)
	}

	// Check if Terraform version is supported. Use workspace's value by default.
	terraformVersion := ws.TerraformVersion
	if options.TerraformVersion != "" {
		versions, tErr := s.cliService.GetTerraformCLIVersions(ctx)
		if tErr != nil {
			tracing.RecordError(span, tErr, "failed to get terraform CLI versions")
			return nil, tErr
		}

		if err = versions.Supported(options.TerraformVersion); err != nil {
			tracing.RecordError(span, err, "failed to get supported terraform version")
			return nil, err
		}

		terraformVersion = options.TerraformVersion
	}

	// Enforce the workspace's option to prevent a destroy run.
	if options.IsDestroy && ws.PreventDestroyPlan {
		return nil, errors.New(
			errors.EForbidden,
			"Workspace does not allow destroy plan",
		)
	}

	// Check if any managed identities are assigned to this workspace
	managedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, options.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identities for workspace")
		return nil, err
	}

	var currentStateVersionID *string
	if ws.CurrentStateVersionID != "" {
		currentStateVersionID = &ws.CurrentStateVersionID
	}

	runDetails := &rules.RunDetails{
		RunStage:              models.JobPlanType,
		ModuleDigest:          moduleDigest,
		CurrentStateVersionID: currentStateVersionID,
		ModuleSource:          options.ModuleSource,
	}

	if moduleRegistrySource != nil {
		runDetails.ModuleID = moduleRegistrySource.ModuleID
	}

	// Verify that subject has permission to create a plan for all of the assigned managed identities
	if err = s.enforceManagedIdentityRules(ctx, managedIdentities, runDetails); err != nil {
		tracing.RecordError(span, err, "failed to verify subject can enforce managed identity rules")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for claimJob: %v", txErr)
		}
	}()

	// Create plan resource
	plan, err := s.dbClient.Plans.CreatePlan(txContext, &models.Plan{Status: models.PlanQueued, WorkspaceID: options.WorkspaceID})
	if err != nil {
		tracing.RecordError(span, err, "failed to create plan")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create plan",
		)
	}

	// If there is a module source, make the plan job _not_ speculative.
	isSpeculative := false

	// If there is a configuration version, get it and let it decide whether the run is speculative.
	if options.ConfigurationVersionID != nil {
		configVersion, gcvErr := s.dbClient.ConfigurationVersions.GetConfigurationVersion(txContext, *options.ConfigurationVersionID)
		if gcvErr != nil {
			tracing.RecordError(span, gcvErr, "failed to get configuration version")
			return nil, errors.Wrap(
				gcvErr,
				errors.EInternal,
				"Failed to get configuration version associated with run",
			)
		}
		isSpeculative = configVersion.Speculative
	}

	createRunOptions := models.Run{
		WorkspaceID:            options.WorkspaceID,
		ConfigurationVersionID: options.ConfigurationVersionID,
		IsDestroy:              options.IsDestroy,
		Status:                 models.RunPlanQueued,
		CreatedBy:              caller.GetSubject(),
		ModuleSource:           options.ModuleSource,
		ModuleVersion:          moduleVersion,
		ModuleDigest:           moduleDigest,
		PlanID:                 plan.Metadata.ID,
		TerraformVersion:       terraformVersion,
		TargetAddresses:        options.TargetAddresses,
		Refresh:                options.Refresh,
	}

	if options.Comment != nil {
		createRunOptions.Comment = *options.Comment
	}

	if !isSpeculative {
		// Create apply resource
		apply, aErr := s.dbClient.Applies.CreateApply(txContext, &models.Apply{Status: models.ApplyCreated, WorkspaceID: options.WorkspaceID})

		if aErr != nil {
			tracing.RecordError(span, aErr, "failed to create apply")
			return nil, errors.Wrap(
				aErr,
				errors.EInternal,
				"Failed to create apply",
			)
		}

		createRunOptions.ApplyID = apply.Metadata.ID
	}

	run, err := s.dbClient.Runs.CreateRun(txContext, &createRunOptions)
	if err != nil {
		tracing.RecordError(span, err, "failed to create run")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create run",
		)
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &ws.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetRun,
			TargetID:      run.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	now := time.Now()

	// Create job for initial plan
	job := models.Job{
		Status:          models.JobQueued,
		Type:            models.JobPlanType,
		WorkspaceID:     options.WorkspaceID,
		RunID:           run.Metadata.ID,
		CancelRequested: false,
		Timestamps: models.JobTimestamps{
			QueuedTimestamp: &now,
		},
		MaxJobDuration: *ws.MaxJobDuration,
	}

	// Create Job
	if _, err = s.dbClient.Jobs.CreateJob(txContext, &job); err != nil {
		tracing.RecordError(span, err, "failed to create job")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create job",
		)
	}

	// Save run variables.
	data, err := json.Marshal(runVariables)
	if err != nil {
		tracing.RecordError(span, err, "failed to marshal run variables")
		return run, err
	}
	if err := s.artifactStore.UploadRunVariables(ctx, run, bytes.NewReader(data)); err != nil {
		tracing.RecordError(span, err, "failed to upload run variables")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a new run.",
		"caller", caller.GetSubject(),
		"workspaceID", run.WorkspaceID,
		"runID", run.Metadata.ID,
	)
	return run, nil
}

// ApplyRun executes the apply action on an existing run
func (s *service) ApplyRun(ctx context.Context, runID string, comment *string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.ApplyRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.getRun(ctx, runID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateRunPermission, auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Check if any managed identities are assigned to this workspace
	managedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, run.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get managed identities assigned to workspace")
		return nil, err
	}

	// Retrieve workspace to find max job duration and current state version ID.
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace by ID")
		return nil, err
	}

	if ws == nil {
		return nil, fmt.Errorf("failed to get workspace ID %s associated with run ID %s", run.WorkspaceID, run.Metadata.ID)
	}

	var currentStateVersionID *string
	if ws.CurrentStateVersionID != "" {
		currentStateVersionID = &ws.CurrentStateVersionID
	}

	if len(managedIdentities) > 0 {
		runDetails := &rules.RunDetails{
			RunStage:              models.JobApplyType,
			ModuleDigest:          run.ModuleDigest,
			CurrentStateVersionID: currentStateVersionID,
		}

		var moduleSource *ModuleRegistrySource
		if run.ModuleSource != nil {
			moduleSource, err = s.moduleResolver.ParseModuleRegistrySource(ctx, *run.ModuleSource)
			if err != nil {
				tracing.RecordError(span, err, "failed to parse module registry source")
				return nil, err
			}

			if moduleSource != nil {
				runDetails.ModuleID = moduleSource.ModuleID
				runDetails.ModuleSource = run.ModuleSource
			}
		}

		// Verify that subject has permission to create a plan for all of the assigned managed identities
		if err = s.enforceManagedIdentityRules(ctx, managedIdentities, runDetails); err != nil {
			tracing.RecordError(span, err, "failed to verify subject can enforce managed identity rules")
			return nil, err
		}
	}

	// Get apply resource
	apply, err := s.dbClient.Applies.GetApply(ctx, run.ApplyID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get apply")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get apply resource",
		)
	}

	apply.Status = models.ApplyQueued
	apply.TriggeredBy = caller.GetSubject()

	if comment != nil {
		apply.Comment = *comment
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for ApplyRun: %v", txErr)
		}
	}()

	_, err = s.runStateManager.UpdateApply(txContext, apply)
	if err != nil {
		tracing.RecordError(span, err, "failed to update apply")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to update apply resource",
		)
	}

	now := time.Now()

	// Create job for apply
	job := models.Job{
		Status:          models.JobQueued,
		Type:            models.JobApplyType,
		WorkspaceID:     run.WorkspaceID,
		RunID:           run.Metadata.ID,
		CancelRequested: false,
		Timestamps: models.JobTimestamps{
			QueuedTimestamp: &now,
		},
		MaxJobDuration: *ws.MaxJobDuration,
	}

	// Create Job
	if _, err := s.dbClient.Jobs.CreateJob(txContext, &job); err != nil {
		tracing.RecordError(span, err, "failed to create job")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create job",
		)
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Applied a run.",
		"caller", caller.GetSubject(),
		"workspaceID", run.WorkspaceID,
		"runStatus", run.Status,
		"runID", runID,
	)
	return run, nil
}

func (s *service) CancelRun(ctx context.Context, options *CancelRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CancelRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// TODO: Remember to do something with the options.Comment field.

	run, err := s.GetRun(ctx, options.RunID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run")
		return nil, err
	}

	// Since UpdateRunPermission means run write access, we must use CreateRunPermission
	// instead i.e. if caller can create a run, they must be able to cancel it as well.
	err = caller.RequirePermission(ctx, permissions.CreateRunPermission, auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Verify run is in a valid state to be canceled
	switch run.Status {
	case models.RunPlannedAndFinished:
		// If a run is in RunPlannedAndFinished state, meaning the run was for plan
		// only and that plan has finished, the job cannot be canceled, so return a
		// bad request error aka EInvalid.
		return nil, errors.New(
			errors.EInvalid,
			"run has been planned and finished, so it cannot be canceled",
		)
	case models.RunApplied:
		// If a run is in RunApplied state, meaning the run was for apply and that plan has finished,
		// the job cannot be canceled, so return a bad request error aka EInvalid.
		return nil, errors.New(
			errors.EInvalid,
			"run has been applied, so it cannot be canceled",
		)
	case models.RunCanceled:
		return nil, errors.New(
			errors.EInvalid,
			"run has already been canceled",
		)
	}

	// If this is a force cancel request, verify graceful cancel was already attempted
	if options.Force {
		// Verify that graceful cancel was already attempted
		if run.ForceCancelAvailableAt == nil {
			return nil, errors.New(
				errors.EInvalid,
				"run has not already received a graceful request to cancel",
			)
		}

		// Error out with errors.EInvalid if not yet eligible.
		if time.Now().Before(*run.ForceCancelAvailableAt) {
			return nil, errors.New(
				errors.EInvalid,
				"insufficient time has elapsed since graceful cancel request; force cancel will be available at %s",
				*run.ForceCancelAvailableAt,
			)
		}
	}

	switch run.Status {
	case models.RunPlanned:
		// If a run is in RunPlanned state, meaning the plan job has finished but
		// the apply job has not yet been queued, cancel the run by simply doing
		// updateApply on it.

		apply, aErr := s.GetApply(ctx, run.ApplyID)
		if aErr != nil {
			tracing.RecordError(span, aErr, "failed to get apply")
			return nil, errors.Wrap(
				aErr,
				errors.EInternal,
				"failed to get the apply object to cancel a planned run",
			)
		}

		apply.Status = models.ApplyCanceled
		_, err = s.runStateManager.UpdateApply(ctx, apply)
		if err != nil {
			tracing.RecordError(span, err, "failed to update apply")
			return nil, errors.Wrap(
				err,
				errors.EInternal,
				"failed to update the apply to cancel a planned run",
			)
		}

		return run, nil
	case models.RunPlanQueued:
		plan, pErr := s.GetPlan(ctx, run.PlanID)
		if pErr != nil {
			tracing.RecordError(span, pErr, "failed to get the plan to cancel a queued run")
			return nil, errors.Wrap(
				pErr,
				errors.EInternal,
				"failed to get the plan to cancel a queued run",
			)
		}

		plan.Status = models.PlanCanceled
		_, err = s.runStateManager.UpdatePlan(ctx, plan)
		if err != nil {
			tracing.RecordError(span, err, "failed to update the plan to cancel a queued run")
			return nil, errors.Wrap(
				err,
				errors.EInternal,
				"failed to update the plan to cancel a queued run",
			)
		}

		return run, nil
	}

	// Wrap all the DB updates in a transaction, whether the cancel is forced or graceful.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction to cancel a run")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"failed to create a transaction to cancel a run",
		)
	}
	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CancelRun: %v", txErr)
		}
	}()

	var updatedRun *models.Run
	var cancelErr error
	if options.Force {
		updatedRun, cancelErr = s.forceCancelRun(txContext, run)
	} else {
		updatedRun, cancelErr = s.gracefullyCancelRun(txContext, run)
	}

	if cancelErr != nil {
		tracing.RecordError(span, cancelErr, "failed to cancel a run")
		return nil, cancelErr
	}

	workspace, wErr := s.dbClient.Workspaces.GetWorkspaceByID(ctx, updatedRun.WorkspaceID)
	if wErr != nil {
		tracing.RecordError(span, wErr, "failed to get workspace by ID")
		return nil, wErr
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionCancel,
			TargetType:    models.TargetRun,
			TargetID:      updatedRun.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction to cancel a run")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"failed to commit the transaction to cancel a run",
		)
	}

	return updatedRun, nil
}

func (s *service) gracefullyCancelRun(ctx context.Context, run *models.Run) (*models.Run, error) {

	// Update run's ForceCancelAvailableAt.
	if run.ForceCancelAvailableAt == nil {
		now := time.Now()
		whenForceCancelAllowed := now.Add(forceCancelWait)
		run.ForceCancelAvailableAt = &whenForceCancelAllowed
	}

	// Cancel latest job associated with run
	job, err := s.jobService.GetLatestJobForRun(ctx, run)
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get latest job for run",
		)
	}

	if job == nil {
		return nil, errors.New(
			errors.EInternal,
			"Run has no job",
		)
	}

	now := time.Now()
	job.CancelRequested = true
	job.CancelRequestedTimestamp = &now

	_, err = s.runStateManager.UpdateJob(ctx, job)
	if err != nil {
		return nil, err
	}

	return s.runStateManager.UpdateRun(ctx, run)
}

func (s *service) forceCancelRun(ctx context.Context, run *models.Run) (*models.Run, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Update Run fields.
	subject := caller.GetSubject()
	run.ForceCanceled = true
	run.ForceCanceledBy = &subject

	updatedRun, err := s.runStateManager.UpdateRun(ctx, run)
	if err != nil {
		return nil, err
	}

	// Cancel latest job associated with run
	job, err := s.jobService.GetLatestJobForRun(ctx, run)
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get latest job for run",
		)
	}

	if job == nil {
		return nil, errors.New(
			errors.EInternal,
			"Run has no job",
		)
	}

	// If a forced cancel, update the state of the plan or apply directly
	// and mark the workspace as having dirty state.  Do this before tell the
	// job to cancel itself to avoid risk of not recording the dirty state.

	// Update the plan or apply directly.
	switch job.Type {
	case models.JobPlanType:
		plan, err := s.GetPlan(ctx, run.PlanID)
		if err != nil {
			return nil, errors.Wrap(
				err,
				errors.EInternal,
				"failed to get the plan object to cancel a run",
			)
		}

		plan.Status = models.PlanCanceled
		_, err = s.runStateManager.UpdatePlan(ctx, plan)
		if err != nil {
			// This error does not need to be wrapped.
			return nil, err
		}
	case models.JobApplyType:
		apply, err := s.GetApply(ctx, run.ApplyID)
		if err != nil {
			return nil, errors.Wrap(
				err,
				errors.EInternal,
				"failed to get an apply object to cancel a run",
			)
		}

		apply.Status = models.ApplyCanceled
		_, err = s.runStateManager.UpdateApply(ctx, apply)
		if err != nil {
			// This error does not need to be wrapped.
			return nil, err
		}
	}

	return updatedRun, nil
}

// GetRun returns a run
func (s *service) GetRun(ctx context.Context, runID string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRun")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.getRun(ctx, runID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return run, nil
}

func (s *service) GetRuns(ctx context.Context, input *GetRunsInput) (*db.RunsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRuns")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	filter := &db.RunFilter{}

	if input.Workspace != nil {
		err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithNamespacePath(input.Workspace.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		filter.WorkspaceID = &input.Workspace.Metadata.ID
	} else if input.Group != nil {
		err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			tracing.RecordError(span, napErr, "failed to get namespace access policy")
			return nil, napErr
		}
		if !policy.AllowAll {
			return nil, errors.New(errors.EInvalid, "either a workspace or group must be specified when querying for runs")
		}
	}

	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		tracing.RecordError(span, err, "Failed to get runs")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get runs",
		)
	}

	return result, nil
}

func (s *service) GetRunsByIDs(ctx context.Context, idList []string) ([]models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Filter: &db.RunFilter{
			RunIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "Failed to get runs")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get runs",
		)
	}

	for _, run := range result.Runs {
		err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.Runs, nil
}

func (s *service) GetPlansByIDs(ctx context.Context, idList []string) ([]models.Plan, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPlansByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.Plans.GetPlans(ctx, &db.GetPlansInput{
		Filter: &db.PlanFilter{
			PlanIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "Failed to get plans")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get plans",
		)
	}

	for _, plan := range result.Plans {
		err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithWorkspaceID(plan.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.Plans, nil
}

// GetPlan returns a tfe plan
func (s *service) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	plan, err := s.dbClient.Plans.GetPlan(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get plan")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get plan",
		)
	}

	if plan == nil {
		return nil, errors.New(errors.ENotFound, "plan with ID %s not found", planID)
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return plan, nil
}

func (s *service) UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdatePlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdatePlanPermission, auth.WithPlanID(plan.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return s.runStateManager.UpdatePlan(ctx, plan)
}

func (s *service) DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "svc.DownloadPlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	result, err := s.artifactStore.GetPlanCache(ctx, run)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get plan cache from artifact store")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get plan cache from artifact store",
		)
	}

	return result, nil
}

func (s *service) GetRunVariables(ctx context.Context, runID string) ([]Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunVariables")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRun(ctx, runID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get run")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get run",
		)
	}

	if run == nil {
		return nil, errors.New(errors.ENotFound, "run with ID %s not found", runID)
	}

	// Only include variable values if the caller has UpdateRunPermission or ViewVariableValuePermission on workspace.
	includeValues := false
	if err = caller.RequirePermission(ctx, permissions.ViewVariableValuePermission, auth.WithWorkspaceID(run.WorkspaceID)); err == nil {
		includeValues = true
	} else if err = caller.RequirePermission(ctx, permissions.ViewVariablePermission, auth.WithWorkspaceID(run.WorkspaceID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	result, err := s.artifactStore.GetRunVariables(ctx, run)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run variables from object store")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get run variables from object store",
		)
	}

	defer result.Close()

	var variables []Variable
	if err := json.NewDecoder(result).Decode(&variables); err != nil {
		tracing.RecordError(span, err, "failed to decode run variables")
		return nil, err
	}

	if !includeValues {
		for i := range variables {
			variables[i].Value = nil
		}
	}

	// Sort variable list
	sort.Slice(variables, func(i, j int) bool {
		var v int
		if variables[i].NamespacePath != nil && variables[j].NamespacePath != nil {
			v = strings.Compare(*variables[i].NamespacePath, *variables[j].NamespacePath)
		} else if variables[i].NamespacePath != nil && variables[j].NamespacePath == nil {
			v = 1
		} else {
			v = -1
		}

		if v == 0 {
			return strings.Compare(variables[i].Key, variables[j].Key) < 0
		}
		return v < 0
	})

	return variables, nil
}

func (s *service) UploadPlan(ctx context.Context, planID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadPlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdatePlanPermission, auth.WithPlanID(planID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return err
	}

	if err := s.artifactStore.UploadPlanCache(ctx, run, reader); err != nil {
		tracing.RecordError(span, err, "Failed to write plan cache to object storage")
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to write plan cache to object storage",
		)
	}

	return nil
}

func (s *service) GetAppliesByIDs(ctx context.Context, idList []string) ([]models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.GetAppliesByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.Applies.GetApplies(ctx, &db.GetAppliesInput{
		Filter: &db.ApplyFilter{
			ApplyIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "Failed to list applies")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to list applies",
		)
	}

	for _, apply := range result.Applies {
		err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithWorkspaceID(apply.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.Applies, nil
}

// GetApply returns a tfe apply
func (s *service) GetApply(ctx context.Context, applyID string) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.GetApply")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	apply, err := s.dbClient.Applies.GetApply(ctx, applyID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get apply")
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get apply",
		)
	}

	if apply == nil {
		return nil, errors.New(errors.ENotFound, "apply with ID %s not found", applyID)
	}

	run, err := s.dbClient.Runs.GetRunByApplyID(ctx, applyID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by apply ID")
		return nil, err
	}

	if run == nil {
		return nil, fmt.Errorf("failed to get run associated with apply id %s", applyID)
	}

	err = caller.RequirePermission(ctx, permissions.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return apply, nil
}

func (s *service) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateApply")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateApplyPermission, auth.WithApplyID(apply.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return s.runStateManager.UpdateApply(ctx, apply)
}

func (s *service) GetLatestJobForPlan(ctx context.Context, planID string) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "svc.GetLatestJobForPlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return nil, err
	}

	job, err := s.getLatestJobByRunAndType(ctx, run.Metadata.ID, models.JobPlanType)
	if err != nil {
		tracing.RecordError(span, err, "failed to get latest job by run and type")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithWorkspaceID(run.WorkspaceID), auth.WithJobID(job.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return job, nil
}

func (s *service) GetLatestJobForApply(ctx context.Context, applyID string) (*models.Job, error) {
	ctx, span := tracer.Start(ctx, "svc.GetLatestJobForApply")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByApplyID(ctx, applyID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by apply ID")
		return nil, err
	}

	if run == nil {
		return nil, fmt.Errorf("failed to get run associated with apply id %s", applyID)
	}

	job, err := s.getLatestJobByRunAndType(ctx, run.Metadata.ID, models.JobApplyType)
	if err != nil {
		tracing.RecordError(span, err, "failed to get latest job by run and type")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.ViewJobPermission, auth.WithWorkspaceID(run.WorkspaceID), auth.WithJobID(job.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return job, nil
}

func (s *service) buildRunVariables(ctx context.Context, workspaceID string, runVariables []Variable) ([]Variable, error) {
	// Get Workspace
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if ws == nil {
		return nil, fmt.Errorf("workspace with id %s not found", workspaceID)
	}

	pathParts := strings.Split(ws.FullPath, "/")

	namespacePaths := []string{}
	for len(pathParts) > 0 {
		namespacePaths = append(namespacePaths, strings.Join(pathParts, "/"))
		// Remove last element
		pathParts = pathParts[:len(pathParts)-1]
	}

	// Use a descending sort so the variables from the closest ancestor will take precedence
	sortBy := db.VariableSortableFieldNamespacePathDesc
	result, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: namespacePaths,
		},
		Sort: &sortBy,
	})
	if err != nil {
		return nil, err
	}

	variableMap := map[string]Variable{}

	buildMapKey := func(key string, category string) string {
		return fmt.Sprintf("%s::%s", key, category)
	}

	// Add run variables first since they have the highest precedence
	for _, v := range runVariables {
		if v.Category == models.EnvironmentVariableCategory && v.Hcl {
			return nil, errors.New(errors.EInvalid, "HCL variables are not supported for the environment category")
		}

		variableMap[buildMapKey(v.Key, string(v.Category))] = Variable{
			Key:      v.Key,
			Value:    v.Value,
			Category: v.Category,
			Hcl:      v.Hcl,
		}
	}

	for _, v := range result.Variables {
		vCopy := v

		keyAndCategory := buildMapKey(v.Key, string(v.Category))
		if _, ok := variableMap[keyAndCategory]; !ok {
			variableMap[keyAndCategory] = Variable{
				Key:           v.Key,
				Value:         v.Value,
				Category:      v.Category,
				Hcl:           v.Hcl,
				NamespacePath: &vCopy.NamespacePath,
			}
		}
	}

	variables := []Variable{}
	for _, v := range variableMap {
		variables = append(variables, v)
	}

	return variables, nil
}

func (s *service) getLatestJobByRunAndType(ctx context.Context, runID string, jobType models.JobType) (*models.Job, error) {
	job, err := s.dbClient.Jobs.GetLatestJobByType(ctx, runID, jobType)
	if err != nil {
		return nil, err
	}

	if job == nil {
		return nil, errors.New(errors.ENotFound, "latest %s job for run %s not found", jobType, runID)
	}

	return job, nil
}

func (s *service) enforceManagedIdentityRules(ctx context.Context, managedIdentities []models.ManagedIdentity, runDetails *rules.RunDetails) error {
	for _, mi := range managedIdentities {
		miCopy := mi
		if err := s.ruleEnforcer.EnforceRules(ctx, &miCopy, runDetails); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) getRun(ctx context.Context, runID string) (*models.Run, error) {
	run, err := s.dbClient.Runs.GetRun(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get run",
		)
	}

	if run == nil {
		return nil, errors.New(errors.ENotFound, "run with ID %s not found", runID)
	}

	return run, nil
}
