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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

const (

	// forceCancelWait is how long a run must be soft-canceled before it is allowed to be forcefully canceled.
	forceCancelWait = 30 * time.Minute
)

// ClaimJobResponse is returned when a runner claims a Job
type ClaimJobResponse struct {
	Job   *models.Job
	Token string
}

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
	PaginationOptions *db.PaginationOptions
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
	IsDestroy              bool
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
	ClaimJob(ctx context.Context, runnerID string) (*ClaimJobResponse, error)
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
	idp             *auth.IdentityProvider
	jobService      job.Service
	cliService      cli.Service
	runStateManager *runStateManager
	activityService activityevent.Service
	moduleService   moduleregistry.Service
	moduleResolver  ModuleResolver
	ruleEnforcer    rules.RuleEnforcer
}

var (
	planExecutionTime  = metric.NewHistogram("plan_execution_time", "Amount of time a plan took to execute.", 1, 2, 10)
	applyExecutionTime = metric.NewHistogram("apply_execution_time", "Amount of time a plan took to apply.", 1, 2, 10)

	planFinished  = metric.NewCounter("plan_completed_count", "Amount of times a plan is completed.")
	applyFinished = metric.NewCounter("apply_completed_count", "Amount of times an apply is completed.")
	runFinished   = metric.NewCounter("run_completed_count", "Amount of times a run is completed.")
)

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	artifactStore workspace.ArtifactStore,
	eventManager *events.EventManager,
	idp *auth.IdentityProvider,
	jobService job.Service,
	cliService cli.Service,
	activityService activityevent.Service,
	moduleService moduleregistry.Service,
	moduleResolver ModuleResolver,
) Service {
	return newService(
		logger,
		dbClient,
		artifactStore,
		eventManager,
		idp,
		jobService,
		cliService,
		activityService,
		moduleService,
		moduleResolver,
		newRunStateManager(dbClient, logger),
		rules.NewRuleEnforcer(dbClient),
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	artifactStore workspace.ArtifactStore,
	eventManager *events.EventManager,
	idp *auth.IdentityProvider,
	jobService job.Service,
	cliService cli.Service,
	activityService activityevent.Service,
	moduleService moduleregistry.Service,
	moduleResolver ModuleResolver,
	runStateManager *runStateManager,
	ruleEnforcer rules.RuleEnforcer,
) Service {
	return &service{
		logger,
		dbClient,
		artifactStore,
		eventManager,
		idp,
		jobService,
		cliService,
		runStateManager,
		activityService,
		moduleService,
		moduleResolver,
		ruleEnforcer,
	}
}

func (s *service) ClaimJob(ctx context.Context, runnerID string) (*ClaimJobResponse, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Only allow system caller for now until runner registration is supported
	if _, ok := caller.(*auth.SystemCaller); !ok {
		return nil, errors.NewError(errors.EForbidden, fmt.Sprintf("Subject %s is not authorized to claim jobs", caller.GetSubject()))
	}

	for {
		job, err := s.jobService.GetNextAvailableQueuedJob(ctx, runnerID)
		if err != nil {
			return nil, err
		}

		// Attempt to claim job
		job, err = s.claimJob(ctx, job, runnerID)
		if err != nil {
			return nil, err
		}

		if job != nil {
			maxJobDuration := time.Duration(job.MaxJobDuration) * time.Minute
			expiration := time.Now().Add(maxJobDuration + time.Hour)
			token, err := s.idp.GenerateToken(ctx, &auth.TokenInput{
				// Expiration is job timeout plus 1 hour to give the job time to gracefully exit
				Expiration: &expiration,
				Subject:    fmt.Sprintf("job-%s", job.Metadata.ID),
				Claims: map[string]string{
					"job_id":       gid.ToGlobalID(gid.JobType, job.Metadata.ID),
					"run_id":       gid.ToGlobalID(gid.RunType, job.RunID),
					"workspace_id": gid.ToGlobalID(gid.WorkspaceType, job.WorkspaceID),
					"type":         auth.JobTokenType,
				},
			})
			if err != nil {
				return nil, err
			}

			s.logger.Infow("Claimed a job.",
				"caller", caller.GetSubject(),
				"workspaceID", job.WorkspaceID,
				"jobID", job.Metadata.ID,
			)
			return &ClaimJobResponse{Job: job, Token: string(token)}, nil
		}
	}
}

func (s *service) claimJob(ctx context.Context, job *models.Job, runnerID string) (*models.Job, error) {
	// Start transaction
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for claimJob: %v", txErr)
		}
	}()

	now := time.Now()
	job.Timestamps.PendingTimestamp = &now
	job.Status = models.JobPending
	job.RunnerID = runnerID

	job, err = s.runStateManager.updateJob(txContext, job)
	if err != nil && err != db.ErrOptimisticLockError {
		return nil, err
	}

	if err == db.ErrOptimisticLockError {
		return nil, nil
	}

	// Get run associated with job
	run, err := s.dbClient.Runs.GetRun(ctx, job.RunID)
	if err != nil {
		return nil, err
	}

	if run == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Run with ID %s not found", job.RunID))
	}

	switch job.Type {
	case models.JobPlanType:
		plan, err := s.dbClient.Plans.GetPlan(ctx, run.PlanID)
		if err != nil {
			return nil, err
		}

		plan.Status = models.PlanPending
		if _, err := s.runStateManager.updatePlan(txContext, plan); err != nil {
			return nil, err
		}
	case models.JobApplyType:
		apply, err := s.dbClient.Applies.GetApply(ctx, run.ApplyID)
		if err != nil {
			return nil, err
		}

		apply.Status = models.ApplyPending
		if _, err := s.runStateManager.updateApply(txContext, apply); err != nil {
			return nil, err
		}
	default:
		return nil, errors.NewError(errors.EInternal, fmt.Sprintf("Invalid job type: %s", job.Type))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return job, nil
}

func (s *service) SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if options.WorkspaceID == nil {
		return nil, errors.NewError(errors.EInvalid, "WorkspaceID option is required")
	}

	if err := caller.RequireAccessToWorkspace(ctx, *options.WorkspaceID, models.ViewerRole); err != nil {
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
				if err != context.Canceled {
					s.logger.Errorf("Error occurred while waiting for run events: %v", err)
				}
				return
			}

			run, err := s.getRun(ctx, event.ID)
			if err != nil {
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
	err := options.Validate()
	if err != nil {
		return nil, err
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, options.WorkspaceID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Build run variables
	runVariables, err := s.buildRunVariables(ctx, options.WorkspaceID, options.Variables)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"failed to build run variables",
			errors.WithErrorErr(err),
		)
	}

	// Filter out the non-environmental variables.
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
			return nil, errors.NewError(errors.EInvalid, fmt.Sprintf("Failed to resolve module source: %v", err))
		}

		// registry source will be nil if this is a remote module source that doesn't use the terraform module registry protocol
		if moduleRegistrySource != nil {
			var resolvedVersion string
			resolvedVersion, err = s.moduleResolver.ResolveModuleVersion(ctx, moduleRegistrySource, options.ModuleVersion, runEnvVars)
			if err != nil {
				return nil, errors.NewError(errors.EInvalid, fmt.Sprintf("Failed to resolve module source: %v", err))
			}
			moduleVersion = &resolvedVersion

			// If this is a module stored in the local tharsis registry, we need to get the module version digest to pin the run to it to
			// prevent the module package from changing after the run has been created. This is an additional protection that is only available
			// for modules in the tharsis module registry
			if moduleRegistrySource.ModuleID != nil {
				var versionsResponse *db.ModuleVersionsResult
				versionsResponse, err = s.moduleService.GetModuleVersions(ctx, &moduleregistry.GetModuleVersionsInput{
					PaginationOptions: &db.PaginationOptions{
						First: ptr.Int32(1),
					},
					ModuleID:        *moduleRegistrySource.ModuleID,
					SemanticVersion: &resolvedVersion,
				})
				if err != nil {
					return nil, err
				}

				if len(versionsResponse.ModuleVersions) == 0 {
					return nil, errors.NewError(errors.EInternal, fmt.Sprintf("unable to find the module package for module %s with semantic version %s", *options.ModuleSource, resolvedVersion))
				}

				moduleDigest = versionsResponse.ModuleVersions[0].SHASum
			}
		}
	}

	// Retrieve workspace to find Terraform version and max job duration.
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, options.WorkspaceID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get workspace associated with run",
			errors.WithErrorErr(err),
		)
	}

	if ws == nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get workspace associated with run",
		)
	}

	// Check if Terraform version is supported. Use workspace's value by default.
	terraformVersion := ws.TerraformVersion
	if options.TerraformVersion != "" {
		versions, tErr := s.cliService.GetTerraformCLIVersions(ctx)
		if tErr != nil {
			return nil, tErr
		}

		if err = versions.Supported(options.TerraformVersion); err != nil {
			return nil, err
		}

		terraformVersion = options.TerraformVersion
	}

	// Enforce the workspace's option to prevent a destroy run.
	if options.IsDestroy && ws.PreventDestroyPlan {
		return nil, errors.NewError(
			errors.EForbidden,
			"Workspace does not allow destroy plan",
		)
	}

	// Check if any managed identities are assigned to this workspace
	managedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, options.WorkspaceID)
	if err != nil {
		return nil, err
	}

	runDetails := &rules.RunDetails{
		RunStage:     models.JobPlanType,
		ModuleDigest: moduleDigest,
	}

	if moduleRegistrySource != nil {
		runDetails.ModuleID = moduleRegistrySource.ModuleID
	}

	// Verify that subject has permission to create a plan for all of the assigned managed identities
	if err = s.enforceManagedIdentityRules(ctx, managedIdentities, runDetails); err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
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
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to create plan",
			errors.WithErrorErr(err),
		)
	}

	// If there is a module source, make the plan job _not_ speculative.
	isSpeculative := false

	// If there is a configuration version, get it and let it decide whether the run is speculative.
	if options.ConfigurationVersionID != nil {
		configVersion, gcvErr := s.dbClient.ConfigurationVersions.GetConfigurationVersion(txContext, *options.ConfigurationVersionID)
		if gcvErr != nil {
			return nil, errors.NewError(
				errors.EInternal,
				"Failed to get configuration version associated with run",
				errors.WithErrorErr(gcvErr),
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
	}

	if options.Comment != nil {
		createRunOptions.Comment = *options.Comment
	}

	if !isSpeculative {
		// Create apply resource
		apply, aErr := s.dbClient.Applies.CreateApply(txContext, &models.Apply{Status: models.ApplyCreated, WorkspaceID: options.WorkspaceID})

		if aErr != nil {
			return nil, errors.NewError(
				errors.EInternal,
				"Failed to create apply",
				errors.WithErrorErr(aErr),
			)
		}

		createRunOptions.ApplyID = apply.Metadata.ID
	}

	run, err := s.dbClient.Runs.CreateRun(txContext, &createRunOptions)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to create run",
			errors.WithErrorErr(err),
		)
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &ws.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetRun,
			TargetID:      run.Metadata.ID,
		}); err != nil {
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
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to create job",
			errors.WithErrorErr(err),
		)
	}

	// Save run variables.
	data, err := json.Marshal(runVariables)
	if err != nil {
		return run, err
	}
	if err := s.artifactStore.UploadRunVariables(ctx, run, bytes.NewReader(data)); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Check if any managed identities are assigned to this workspace
	managedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, run.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if len(managedIdentities) > 0 {
		runDetails := &rules.RunDetails{
			RunStage:     models.JobApplyType,
			ModuleDigest: run.ModuleDigest,
		}

		var moduleSource *ModuleRegistrySource
		if run.ModuleSource != nil {
			moduleSource, err = s.moduleResolver.ParseModuleRegistrySource(ctx, *run.ModuleSource)
			if err != nil {
				return nil, err
			}

			if moduleSource != nil {
				runDetails.ModuleID = moduleSource.ModuleID
			}
		}

		// Verify that subject has permission to create a plan for all of the assigned managed identities
		if err = s.enforceManagedIdentityRules(ctx, managedIdentities, runDetails); err != nil {
			return nil, err
		}
	}

	// Get apply resource
	apply, err := s.dbClient.Applies.GetApply(ctx, run.ApplyID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get apply resource",
			errors.WithErrorErr(err),
		)
	}

	apply.Status = models.ApplyQueued
	apply.TriggeredBy = caller.GetSubject()

	if comment != nil {
		apply.Comment = *comment
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for ApplyRun: %v", txErr)
		}
	}()

	_, err = s.runStateManager.updateApply(txContext, apply)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to update apply resource",
			errors.WithErrorErr(err),
		)
	}

	// Retrieve workspace to find max job duration.
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(txContext, run.WorkspaceID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get workspace associated with run",
			errors.WithErrorErr(err),
		)
	}

	if ws == nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get workspace associated with run",
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
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to create job",
			errors.WithErrorErr(err),
		)
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: Remember to do something with the options.Comment field.

	run, err := s.GetRun(ctx, options.RunID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.DeployerRole); err != nil {
		return nil, err
	}

	// Verify run is in a valid state to be canceled
	switch run.Status {
	case models.RunPlannedAndFinished:
		// If a run is in RunPlannedAndFinished state, meaning the run was for plan
		// only and that plan has finished, the job cannot be canceled, so return a
		// bad request error aka EInvalid.
		return nil, errors.NewError(
			errors.EInvalid,
			"run has been planned and finished, so it cannot be canceled",
		)
	case models.RunApplied:
		// If a run is in RunApplied state, meaning the run was for apply and that plan has finished,
		// the job cannot be canceled, so return a bad request error aka EInvalid.
		return nil, errors.NewError(
			errors.EInvalid,
			"run has been applied, so it cannot be canceled",
		)
	case models.RunCanceled:
		return nil, errors.NewError(
			errors.EInvalid,
			"run has already been canceled",
		)
	}

	// If this is a force cancel request, verify graceful cancel was already attempted
	if options.Force {
		// Verify that graceful cancel was already attempted
		if run.ForceCancelAvailableAt == nil {
			return nil, errors.NewError(
				errors.EInvalid,
				"run has not already received a graceful request to cancel",
			)
		}

		// Error out with errors.EInvalid if not yet eligible.
		if time.Now().Before(*run.ForceCancelAvailableAt) {
			return nil, errors.NewError(
				errors.EInvalid,
				fmt.Sprintf(
					"insufficient time has elapsed since graceful cancel request; force cancel will be available at %s",
					*run.ForceCancelAvailableAt,
				),
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
			return nil, errors.NewError(
				errors.EInternal,
				"failed to get the apply object to cancel a planned run",
				errors.WithErrorErr(aErr),
			)
		}

		apply.Status = models.ApplyCanceled
		_, err = s.runStateManager.updateApply(ctx, apply)
		if err != nil {
			return nil, errors.NewError(
				errors.EInternal,
				"failed to update the apply to cancel a planned run",
				errors.WithErrorErr(err),
			)
		}

		return run, nil
	case models.RunPlanQueued:
		plan, pErr := s.GetPlan(ctx, run.PlanID)
		if pErr != nil {
			return nil, errors.NewError(
				errors.EInternal,
				"failed to get the plan to cancel a queued run",
				errors.WithErrorErr(pErr),
			)
		}

		plan.Status = models.PlanCanceled
		_, err = s.runStateManager.updatePlan(ctx, plan)
		if err != nil {
			return nil, errors.NewError(
				errors.EInternal,
				"failed to update the plan to cancel a queued run",
				errors.WithErrorErr(err),
			)
		}

		return run, nil
	}

	// Wrap all the DB updates in a transaction, whether the cancel is forced or graceful.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"failed to create a transaction to cancel a run",
			errors.WithErrorErr(err),
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
		return nil, cancelErr
	}

	workspace, wErr := s.dbClient.Workspaces.GetWorkspaceByID(ctx, updatedRun.WorkspaceID)
	if wErr != nil {
		return nil, wErr
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &workspace.FullPath,
			Action:        models.ActionCancel,
			TargetType:    models.TargetRun,
			TargetID:      updatedRun.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"failed to commit the transaction to cancel a run",
			errors.WithErrorErr(err),
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
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get latest job for run",
			errors.WithErrorErr(err),
		)
	}

	if job == nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Run has no job",
		)
	}

	now := time.Now()
	job.CancelRequested = true
	job.CancelRequestedTimestamp = &now

	_, err = s.runStateManager.updateJob(ctx, job)
	if err != nil {
		return nil, err
	}

	return s.runStateManager.updateRun(ctx, run)
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

	updatedRun, err := s.runStateManager.updateRun(ctx, run)
	if err != nil {
		return nil, err
	}

	// Cancel latest job associated with run
	job, err := s.jobService.GetLatestJobForRun(ctx, run)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get latest job for run",
			errors.WithErrorErr(err),
		)
	}

	if job == nil {
		return nil, errors.NewError(
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
			return nil, errors.NewError(
				errors.EInternal,
				"failed to get the plan object to cancel a run",
				errors.WithErrorErr(err),
			)
		}

		plan.Status = models.PlanCanceled
		_, err = s.runStateManager.updatePlan(ctx, plan)
		if err != nil {
			// This error does not need to be wrapped.
			return nil, err
		}
	case models.JobApplyType:
		apply, err := s.GetApply(ctx, run.ApplyID)
		if err != nil {
			return nil, errors.NewError(
				errors.EInternal,
				"failed to get an apply object to cancel a run",
				errors.WithErrorErr(err),
			)
		}

		apply.Status = models.ApplyCanceled
		_, err = s.runStateManager.updateApply(ctx, apply)
		if err != nil {
			// This error does not need to be wrapped.
			return nil, err
		}
	}

	return updatedRun, nil
}

// GetRun returns a run
func (s *service) GetRun(ctx context.Context, runID string) (*models.Run, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	return run, nil
}

func (s *service) GetRuns(ctx context.Context, input *GetRunsInput) (*db.RunsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	filter := &db.RunFilter{}

	if input.Workspace != nil {
		if err = caller.RequireAccessToNamespace(ctx, input.Workspace.FullPath, models.ViewerRole); err != nil {
			return nil, err
		}
		filter.WorkspaceID = &input.Workspace.Metadata.ID
	} else if input.Group != nil {
		if err = caller.RequireAccessToNamespace(ctx, input.Group.FullPath, models.ViewerRole); err != nil {
			return nil, err
		}
		filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			return nil, napErr
		}
		if !policy.AllowAll {
			return nil, errors.NewError(errors.EInvalid, "either a workspace or group must be specified when querying for runs")
		}
	}

	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get runs",
			errors.WithErrorErr(err),
		)
	}

	return result, nil
}

func (s *service) GetRunsByIDs(ctx context.Context, idList []string) ([]models.Run, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Filter: &db.RunFilter{
			RunIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get runs",
			errors.WithErrorErr(err),
		)
	}

	for _, run := range result.Runs {
		if err := caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
			return nil, err
		}
	}

	return result.Runs, nil
}

func (s *service) GetPlansByIDs(ctx context.Context, idList []string) ([]models.Plan, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.Plans.GetPlans(ctx, &db.GetPlansInput{
		Filter: &db.PlanFilter{
			PlanIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get plans",
			errors.WithErrorErr(err),
		)
	}

	for _, plan := range result.Plans {
		if err := caller.RequireAccessToWorkspace(ctx, plan.WorkspaceID, models.ViewerRole); err != nil {
			return nil, err
		}
	}

	return result.Plans, nil
}

// GetPlan returns a tfe plan
func (s *service) GetPlan(ctx context.Context, planID string) (*models.Plan, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	plan, err := s.dbClient.Plans.GetPlan(ctx, planID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get plan",
			errors.WithErrorErr(err),
		)
	}

	if plan == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Plan with ID %s not found", planID))
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *service) UpdatePlan(ctx context.Context, plan *models.Plan) (*models.Plan, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err := caller.RequirePlanWriteAccess(ctx, plan.Metadata.ID); err != nil {
		return nil, err
	}

	return s.runStateManager.updatePlan(ctx, plan)
}

func (s *service) DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetPlanCache(ctx, run)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get plan cache from artifact store",
			errors.WithErrorErr(err),
		)
	}

	return result, nil
}

func (s *service) GetRunVariables(ctx context.Context, runID string) ([]Variable, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRun(ctx, runID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run",
			errors.WithErrorErr(err),
		)
	}

	if run == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Run with ID %s not found", runID))
	}

	// Only include variable values if the caller has run write access or deployer access on the workspace
	includeValues := false
	if err = caller.RequireRunWriteAccess(ctx, runID); err == nil {
		includeValues = true
	} else if err = caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.DeployerRole); err == nil {
		includeValues = true
	} else if err = caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetRunVariables(ctx, run)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run variables from object store",
			errors.WithErrorErr(err),
		)
	}

	defer result.Close()

	var variables []Variable
	if err := json.NewDecoder(result).Decode(&variables); err != nil {
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
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err = caller.RequirePlanWriteAccess(ctx, planID); err != nil {
		return err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		return err
	}

	if err := s.artifactStore.UploadPlanCache(ctx, run, reader); err != nil {
		return errors.NewError(
			errors.EInternal,
			"Failed to write plan cache to object storage",
			errors.WithErrorErr(err),
		)
	}

	return nil
}

func (s *service) GetAppliesByIDs(ctx context.Context, idList []string) ([]models.Apply, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.Applies.GetApplies(ctx, &db.GetAppliesInput{
		Filter: &db.ApplyFilter{
			ApplyIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to list applies",
			errors.WithErrorErr(err),
		)
	}

	for _, apply := range result.Applies {
		if err := caller.RequireAccessToWorkspace(ctx, apply.WorkspaceID, models.ViewerRole); err != nil {
			return nil, err
		}
	}

	return result.Applies, nil
}

// GetApply returns a tfe apply
func (s *service) GetApply(ctx context.Context, applyID string) (*models.Apply, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	apply, err := s.dbClient.Applies.GetApply(ctx, applyID)
	if err != nil {
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get apply",
			errors.WithErrorErr(err),
		)
	}

	if apply == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Apply with ID %s not found", applyID))
	}

	run, err := s.dbClient.Runs.GetRunByApplyID(ctx, applyID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	return apply, nil
}

func (s *service) UpdateApply(ctx context.Context, apply *models.Apply) (*models.Apply, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireApplyWriteAccess(ctx, apply.Metadata.ID); err != nil {
		return nil, err
	}

	return s.runStateManager.updateApply(ctx, apply)
}

func (s *service) GetLatestJobForPlan(ctx context.Context, planID string) (*models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	return s.getLatestJobByRunAndType(ctx, run.Metadata.ID, models.JobPlanType)
}

func (s *service) GetLatestJobForApply(ctx context.Context, applyID string) (*models.Job, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByApplyID(ctx, applyID)
	if err != nil {
		return nil, err
	}

	if err := caller.RequireAccessToWorkspace(ctx, run.WorkspaceID, models.ViewerRole); err != nil {
		return nil, err
	}

	return s.getLatestJobByRunAndType(ctx, run.Metadata.ID, models.JobApplyType)
}

func (s *service) buildRunVariables(ctx context.Context, workspaceID string, runVariables []Variable) ([]Variable, error) {
	// Get Workspace
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
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
			return nil, errors.NewError(errors.EInvalid, "HCL variables are not supported for the environment category")
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
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Latest %s job for run %s not found", jobType, runID))
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
		return nil, errors.NewError(
			errors.EInternal,
			"Failed to get run",
			errors.WithErrorErr(err),
		)
	}

	if run == nil {
		return nil, errors.NewError(errors.ENotFound, fmt.Sprintf("Run with ID %s not found", runID))
	}

	return run, nil
}
