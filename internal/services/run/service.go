// Package run provides the run service for creating and managing runs
package run

//go:generate go tool mockery --name Service --inpackage --case underscore

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
	tfjson "github.com/hashicorp/terraform-json"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// forceCancelWait is how long a run must be soft-canceled before it is allowed to be forcefully canceled.
	forceCancelWait = 1 * time.Minute
	// Max error message length for plan and apply errors.
	maxErrorMessageLength = 2048
)

// Variable represents a run variable
type Variable struct {
	VersionID     *string                 `json:"version_id"`
	Value         *string                 `json:"value"`
	NamespacePath *string                 `json:"namespacePath"`
	Key           string                  `json:"key"`
	Category      models.VariableCategory `json:"category"`
	Sensitive     bool                    `json:"sensitive"`
	// DEPRECATED: Hcl is deprecated and will be removed in a future release
	Hcl                bool  `json:"hcl"`
	IncludedInTFConfig *bool `json:"includedInTFConfig"`
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

// SetVariablesIncludedInTFConfigInput is the input for setting variables
// that are included in the Terraform config.
type SetVariablesIncludedInTFConfigInput struct {
	RunID        string
	VariableKeys []string
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
	// WorkspaceAssessment can be used to filter for only assessment runs or to exclude assessment runs
	WorkspaceAssessment *bool
}

// CreateRunInput is the input for creating a new run
type CreateRunInput struct {
	ConfigurationVersionID *string
	Comment                *string
	ModuleSource           *string
	ModuleVersion          *string
	Speculative            *bool // optional field, defaults to false unless using a speculative configuration version
	WorkspaceID            string
	TerraformVersion       string
	Variables              []Variable
	TargetAddresses        []string
	IsDestroy              bool
	Refresh                bool
	RefreshOnly            bool
}

// CreateDestroyRunForWorkspaceInput is the input for creating a destroy run using the current
// configuration version or module that is applied.
type CreateDestroyRunForWorkspaceInput struct {
	WorkspaceID string
}

// CreateAssessmentRunForWorkspaceInput is the input for creating an assessment run
type CreateAssessmentRunForWorkspaceInput struct {
	WorkspaceID             string
	LatestAssessmentVersion *int
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

	// Don't allow refresh_only in combination with other options that would conflict.
	if c.RefreshOnly && c.IsDestroy {
		return fmt.Errorf("refresh_only is not allowed with destroy")
	}

	return nil
}

// CancelRunInput is the input for canceling a run.
type CancelRunInput struct {
	Comment *string
	RunID   string
	Force   bool
}

// UpdateApplyInput is the input for updating an apply
type UpdateApplyInput struct {
	ApplyID         string
	ErrorMessage    *string
	MetadataVersion *int
	Status          models.ApplyStatus
}

// UpdatePlanInput is the input for updating a plan
type UpdatePlanInput struct {
	ErrorMessage    *string
	HasChanges      bool
	MetadataVersion *int
	PlanID          string
	Status          models.PlanStatus
}

type createRunInput struct {
	ConfigurationVersionID *string
	Comment                *string
	ModuleSource           *string
	ModuleVersion          *string
	Speculative            *bool // optional field, defaults to false unless using a speculative configuration version
	WorkspaceID            string
	TerraformVersion       string
	Variables              []Variable
	TargetAddresses        []string
	IsDestroy              bool
	Refresh                bool
	RefreshOnly            bool
	IsAssessmentRun        bool
}

// Service encapsulates Terraform Enterprise Support
type Service interface {
	GetRunByID(ctx context.Context, runID string) (*models.Run, error)
	GetRunByTRN(ctx context.Context, trn string) (*models.Run, error)
	GetRuns(ctx context.Context, input *GetRunsInput) (*db.RunsResult, error)
	GetRunsByIDs(ctx context.Context, idList []string) ([]models.Run, error)
	CreateRun(ctx context.Context, options *CreateRunInput) (*models.Run, error)
	ApplyRun(ctx context.Context, runID string, comment *string) (*models.Run, error)
	CancelRun(ctx context.Context, options *CancelRunInput) (*models.Run, error)
	GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]Variable, error)
	CreateAssessmentRunForWorkspace(ctx context.Context, options *CreateAssessmentRunForWorkspaceInput) (*models.Run, error)
	CreateDestroyRunForWorkspace(ctx context.Context, options *CreateDestroyRunForWorkspaceInput) (*models.Run, error)
	SetVariablesIncludedInTFConfig(ctx context.Context, input *SetVariablesIncludedInTFConfigInput) error
	GetPlansByIDs(ctx context.Context, idList []string) ([]models.Plan, error)
	GetPlanByID(ctx context.Context, planID string) (*models.Plan, error)
	GetPlanByTRN(ctx context.Context, trn string) (*models.Plan, error)
	GetPlanDiff(ctx context.Context, planID string) (*plan.Diff, error)
	UpdatePlan(ctx context.Context, input *UpdatePlanInput) (*models.Plan, error)
	DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error)
	UploadPlanBinary(ctx context.Context, planID string, reader io.Reader) error
	ProcessPlanData(ctx context.Context, planID string, plan *tfjson.Plan, providerSchemas *tfjson.ProviderSchemas) error
	GetAppliesByIDs(ctx context.Context, idList []string) ([]models.Apply, error)
	GetApplyByID(ctx context.Context, applyID string) (*models.Apply, error)
	GetApplyByTRN(ctx context.Context, trn string) (*models.Apply, error)
	UpdateApply(ctx context.Context, input *UpdateApplyInput) (*models.Apply, error)
	GetLatestJobForPlan(ctx context.Context, planID string) (*models.Job, error)
	GetLatestJobForApply(ctx context.Context, applyID string) (*models.Job, error)
	SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error)
	GetStateVersionsByRunIDs(ctx context.Context, idList []string) ([]models.StateVersion, error)
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	artifactStore   workspace.ArtifactStore
	eventManager    *events.EventManager
	jobService      job.Service
	cliService      cli.Service
	runStateManager state.RunStateManager
	activityService activityevent.Service
	moduleResolver  registry.ModuleResolver
	ruleEnforcer    rules.RuleEnforcer
	limitChecker    limits.LimitChecker
	planParser      plan.Parser
	secretManager   secret.Manager
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
	moduleResolver registry.ModuleResolver,
	runStateManager state.RunStateManager,
	limitChecker limits.LimitChecker,
	secretManager secret.Manager,
) Service {
	return newService(
		logger,
		dbClient,
		artifactStore,
		eventManager,
		jobService,
		cliService,
		activityService,
		moduleResolver,
		runStateManager,
		rules.NewRuleEnforcer(dbClient),
		limitChecker,
		plan.NewParser(),
		secretManager,
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
	moduleResolver registry.ModuleResolver,
	runStateManager state.RunStateManager,
	ruleEnforcer rules.RuleEnforcer,
	limitChecker limits.LimitChecker,
	planParser plan.Parser,
	secretManager secret.Manager,
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
		moduleResolver,
		ruleEnforcer,
		limitChecker,
		planParser,
		secretManager,
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

	var userMemberID *string
	switch {
	case options.WorkspaceID != nil:
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithWorkspaceID(*options.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	default:
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New("only users can subscribe to run events without a WorkspaceID filter", errors.WithErrorCode(errors.EForbidden))
		}

		if !userCaller.User.Admin {
			userMemberID = &userCaller.User.Metadata.ID
		}
	}

	subscription := events.Subscription{
		Type: events.RunSubscription,
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

		// Wait for run updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) && !errors.IsDeadlineExceededError(err) {
					s.logger.WithContextFields(ctx).Errorf("Error occurred while waiting for run events: %v", err)
				}
				return
			}

			eventData, err := event.ToRunEventData()
			if err != nil {
				s.logger.WithContextFields(ctx).Errorf("failed to get run event data in run event subscription: %v", err)
				continue
			}

			if options.RunID != nil && eventData.ID != *options.RunID {
				// Not the run we're looking for.
				continue
			}

			if options.WorkspaceID != nil && eventData.WorkspaceID != *options.WorkspaceID {
				// Not the workspace we're looking for.
				continue
			}

			runsResult, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
				},
				Filter: &db.RunFilter{
					WorkspaceID:  options.WorkspaceID,
					UserMemberID: userMemberID,
					RunIDs:       []string{event.ID},
				},
			})
			if err != nil {
				if errors.IsContextCanceledError(err) || errors.IsDeadlineExceededError(err) {
					return
				}
				s.logger.WithContextFields(ctx).Errorf("Error occurred while querying for run associated with run event %s: %v", event.ID, err)
				continue
			}

			if runsResult.PageInfo.TotalCount == 0 {
				// Run isn't for the target workspace or user.
				continue
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &Event{Action: event.Action, Run: runsResult.Runs[0]}:
			}
		}
	}()

	return outgoing, nil
}

func (s *service) CreateAssessmentRunForWorkspace(ctx context.Context, options *CreateAssessmentRunForWorkspaceInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateAssessmentRunForWorkspace")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Get the workspace
	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, options.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace with ID %s", options.WorkspaceID, errors.WithSpan(span))
	}

	if workspace == nil {
		return nil, errors.New("workspace not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if workspace.CurrentStateVersionID == "" {
		return nil, errors.New(
			"assessment run cannot be created because workspace has no current state version",
			errors.WithErrorCode(errors.EConflict),
			errors.WithSpan(span),
		)
	}

	// Get the current state version for the workspace
	stateVersion, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, workspace.CurrentStateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version with ID %s", workspace.CurrentStateVersionID, errors.WithSpan(span))
	}

	if stateVersion == nil {
		return nil, errors.New("assessment run cannot be created because state version not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if stateVersion.RunID == nil {
		return nil, errors.New(
			"cannot create assessment run because state version was created manually and has no module or configuration version associated",
			errors.WithErrorCode(errors.EConflict),
			errors.WithSpan(span),
		)
	}

	// Get the run
	latestRun, err := s.dbClient.Runs.GetRunByID(ctx, *stateVersion.RunID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run with ID %s", *stateVersion.RunID, errors.WithSpan(span))
	}

	if latestRun == nil {
		return nil, errors.New("assessment run cannot be created because run associated with workspace was not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if latestRun.IsDestroy {
		return nil, errors.New(
			"cannot create assessment run because latest run is a destroy run",
			errors.WithErrorCode(errors.EConflict),
			errors.WithSpan(span),
		)
	}

	// Get run variables
	variables, err := s.getRunVariables(ctx, latestRun, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run variables for run with ID %s", latestRun.Metadata.ID, errors.WithSpan(span))
	}

	// Clear namespace path from variables
	for i := range variables {
		variables[i].NamespacePath = nil
	}

	assessment, err := s.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, options.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace assessment for workspace with ID %s", options.WorkspaceID, errors.WithSpan(span))
	}

	// Start transaction
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateAssessmentRunForWorkspace: %v", txErr)
		}
	}()

	if assessment == nil {
		if options.LatestAssessmentVersion != nil {
			return nil, errors.New(
				"cannot create assessment run because latest assessment version is not nil",
				errors.WithErrorCode(errors.EConflict),
				errors.WithSpan(span),
			)
		}
		// Create assessment
		if _, err = s.dbClient.WorkspaceAssessments.CreateWorkspaceAssessment(txContext, &models.WorkspaceAssessment{
			WorkspaceID:        options.WorkspaceID,
			StartedAtTimestamp: time.Now().UTC(),
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create assessment for workspace %q", options.WorkspaceID, errors.WithSpan(span))
		}
	} else {
		if options.LatestAssessmentVersion != nil && *options.LatestAssessmentVersion != assessment.Metadata.Version {
			return nil, errors.New(
				"cannot create assessment run because latest assessment version does not match",
				errors.WithErrorCode(errors.EConflict),
				errors.WithSpan(span),
			)
		}

		// Check if an assessment is already in progress
		if assessment.CompletedAtTimestamp == nil {
			return nil, errors.New(
				"cannot create assessment run because an assessment is already in progress",
				errors.WithErrorCode(errors.EConflict),
				errors.WithSpan(span),
			)
		}

		// Update assessment
		assessment.StartedAtTimestamp = time.Now().UTC()
		assessment.CompletedAtTimestamp = nil
		if _, err = s.dbClient.WorkspaceAssessments.UpdateWorkspaceAssessment(txContext, assessment); err != nil {
			return nil, errors.Wrap(err, "failed to update workspace assessment with ID %q", assessment.Metadata.ID, errors.WithSpan(span))
		}
	}

	assessmentRunInput := &createRunInput{
		IsAssessmentRun:        true,
		Speculative:            ptr.Bool(true),
		RefreshOnly:            true,
		Refresh:                true,
		WorkspaceID:            options.WorkspaceID,
		ConfigurationVersionID: latestRun.ConfigurationVersionID,
		ModuleSource:           latestRun.ModuleSource,
		ModuleVersion:          latestRun.ModuleVersion,
		Variables:              variables,
	}

	run, err := s.createRun(txContext, assessmentRunInput)
	if err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return run, nil
}

func (s *service) CreateDestroyRunForWorkspace(ctx context.Context, options *CreateDestroyRunForWorkspaceInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateDestroyRunForWorkspace")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Get the workspace
	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, options.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace with ID %s", options.WorkspaceID, errors.WithSpan(span))
	}

	if workspace == nil {
		return nil, errors.New("workspace not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if workspace.CurrentStateVersionID == "" {
		return nil, errors.New(
			"destroy run cannot be created because workspace has no current state version",
			errors.WithErrorCode(errors.EConflict),
			errors.WithSpan(span),
		)
	}

	// Get the current state version for the workspace
	stateVersion, err := s.dbClient.StateVersions.GetStateVersionByID(ctx, workspace.CurrentStateVersionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get state version with ID %s", workspace.CurrentStateVersionID, errors.WithSpan(span))
	}

	if stateVersion == nil {
		return nil, errors.New("state version not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if stateVersion.RunID == nil {
		return nil, errors.New(
			"cannot create destroy run because state version was created manually and has no module or configuration version associated",
			errors.WithErrorCode(errors.EConflict),
			errors.WithSpan(span),
		)
	}

	// Get the run
	run, err := s.dbClient.Runs.GetRunByID(ctx, *stateVersion.RunID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run with ID %s", *stateVersion.RunID, errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Get run variables
	variables, err := s.getRunVariables(ctx, run, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run variables for run with ID %s", run.Metadata.ID, errors.WithSpan(span))
	}

	// Clear namespace path from variables
	for i := range variables {
		variables[i].NamespacePath = nil
	}

	destroyRunInput := &createRunInput{
		IsDestroy:              true,
		Refresh:                true,
		WorkspaceID:            options.WorkspaceID,
		ConfigurationVersionID: run.ConfigurationVersionID,
		ModuleSource:           run.ModuleSource,
		ModuleVersion:          run.ModuleVersion,
		Variables:              variables,
	}

	return s.createRun(ctx, destroyRunInput)
}

// CreateRun creates a new run and associates a Plan with it
func (s *service) CreateRun(ctx context.Context, options *CreateRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateRun")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if err = options.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate create run options")
		return nil, err
	}

	// Build run variables
	runVariables, err := s.buildRunVariables(ctx, options.WorkspaceID, options.Variables)
	if err != nil {
		tracing.RecordError(span, err, "failed to build run variables")
		return nil, errors.Wrap(
			err,
			"failed to build run variables",
		)
	}

	return s.createRun(ctx, &createRunInput{
		ConfigurationVersionID: options.ConfigurationVersionID,
		Comment:                options.Comment,
		ModuleSource:           options.ModuleSource,
		ModuleVersion:          options.ModuleVersion,
		Speculative:            options.Speculative,
		WorkspaceID:            options.WorkspaceID,
		TerraformVersion:       options.TerraformVersion,
		Variables:              runVariables,
		TargetAddresses:        options.TargetAddresses,
		IsDestroy:              options.IsDestroy,
		Refresh:                options.Refresh,
		RefreshOnly:            options.RefreshOnly,
	})
}

// CreateRun creates a new run and associates a Plan with it
func (s *service) createRun(ctx context.Context, options *createRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.createRun")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	runVariables := options.Variables

	// Filter out the environment variables.
	runEnvVars := []Variable{}
	for _, variable := range runVariables {
		if variable.Category == models.EnvironmentVariableCategory {
			runEnvVars = append(runEnvVars, variable)
		}
	}

	// Retrieve workspace to find Terraform version and max job duration.
	// ... also to have the workspace path in order to create federated registry tokens.
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, options.WorkspaceID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get workspace associated with run")
		return nil, errors.Wrap(
			err,
			"failed to get workspace (ID %s) associated with run",
			options.WorkspaceID,
			errors.WithSpan(span),
		)
	}

	if ws == nil {
		return nil, errors.New(
			"failed to get workspace associated with run",
			errors.WithErrorCode(errors.ENotFound))
	}

	// If a module source (and a registry-style source), resolve the module version.
	// This requires the run variables in order to have the token(s) for getting version numbers.
	// Handle the case where the run uses a module source rather than a configuration version.
	// If this fails, the transaction will be rolled back, so everything is safe.
	var moduleVersion *string
	var moduleDigest []byte
	var moduleRegistrySource registry.ModuleRegistrySource
	if options.ModuleSource != nil {
		moduleRegistrySource, err = s.moduleResolver.ParseModuleRegistrySource(ctx, *options.ModuleSource, getModuleRegistryToken(runEnvVars), getFederatedRegistry(s.dbClient, ws))
		if err != nil && err != registry.ErrRemoteModuleSource {
			return nil, errors.Wrap(err, "failed to resolve module source", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}

		// registry source will be nil if this is a remote module source that doesn't use the terraform module registry protocol
		if moduleRegistrySource != nil {
			module, err := moduleRegistrySource.LocalRegistryModule(ctx)
			if err != nil {
				return nil, err
			}
			// If module is not nil and private, verify that the caller has authorization to use it
			if module != nil && module.Private {
				err = caller.RequireAccessToInheritableResource(ctx, types.TerraformModuleModelType, auth.WithGroupID(module.GroupID))
				if err != nil {
					return nil, errors.Wrap(err, "caller not authorized to use module %s", *options.ModuleSource, errors.WithSpan(span))
				}
			}

			var resolvedVersion string
			resolvedVersion, err = moduleRegistrySource.ResolveSemanticVersion(ctx, options.ModuleVersion)
			if err != nil {
				return nil, errors.Wrap(err, "failed to resolve module source", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
			}

			moduleVersion = &resolvedVersion

			moduleDigest, err = moduleRegistrySource.ResolveDigest(ctx, *moduleVersion)
			if err != nil {
				return nil, err
			}
		}
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
			"Workspace does not allow destroy plan",
			errors.WithErrorCode(errors.EForbidden))
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
		ModuleSource:          moduleRegistrySource,
		ModuleSemanticVersion: moduleVersion,
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
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for claimJob: %v", txErr)
		}
	}()

	// Create plan resource
	plan, err := s.dbClient.Plans.CreatePlan(txContext, &models.Plan{Status: models.PlanQueued, WorkspaceID: options.WorkspaceID})
	if err != nil {
		tracing.RecordError(span, err, "failed to create plan")
		return nil, errors.Wrap(
			err,
			"Failed to create plan",
		)
	}

	// If there is a module source, default speculative to false unless the option is specified.
	isSpeculative := false
	if (options.ModuleSource != nil) && (options.Speculative != nil) {
		isSpeculative = *options.Speculative
	}

	// If there is a configuration version, get it and let it decide whether the run is speculative.
	if options.ConfigurationVersionID != nil {
		configVersion, gcvErr := s.dbClient.ConfigurationVersions.GetConfigurationVersionByID(txContext, *options.ConfigurationVersionID)
		if gcvErr != nil {
			tracing.RecordError(span, gcvErr, "failed to get configuration version")
			return nil, errors.Wrap(
				gcvErr,
				"Failed to get configuration version associated with run",
			)
		}

		// Do not allow the options to set speculative to false if the configuration version has it set to true.
		if configVersion.Speculative && (options.Speculative != nil) && !*options.Speculative {
			return nil, errors.New(
				"Speculative configuration version does not allow non-speculative runs",
				errors.WithErrorCode(errors.EInvalid))
		}

		// Otherwise, the speculative option can override the configuration version.
		isSpeculative = configVersion.Speculative
		if options.Speculative != nil {
			isSpeculative = *options.Speculative
		}
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
		RefreshOnly:            options.RefreshOnly,
		IsAssessmentRun:        options.IsAssessmentRun,
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
			"Failed to create run",
		)
	}

	// Get the number of recent runs for this workspace to check whether we just violated the limit.
	newRuns, err := s.dbClient.Runs.GetRuns(txContext, &db.GetRunsInput{
		Filter: &db.RunFilter{
			TimeRangeStart: ptr.Time(run.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			WorkspaceID:    &options.WorkspaceID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get workspace's runs")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitRunsPerWorkspacePerTimePeriod, newRuns.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
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

	runnerTags, err := s.getJobTags(txContext, ws)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner tags for workspace")
	}

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
		Tags:           runnerTags,
	}

	// Create Job
	createdJob, err := s.dbClient.Jobs.CreateJob(txContext, &job)
	if err != nil {
		tracing.RecordError(span, err, "failed to create job")
		return nil, errors.Wrap(
			err,
			"Failed to create job",
		)
	}

	_, err = s.dbClient.LogStreams.CreateLogStream(txContext, &models.LogStream{
		JobID: &createdJob.Metadata.ID,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create log stream for plan job")
		return nil, errors.Wrap(
			err,
			"Failed to create log stream for plan job",
		)
	}

	// When saving run variables we'll remove the sensitive values.
	// This is done to prevent the sensitive values from being stored in object storage.
	for i, variable := range runVariables {
		if variable.Sensitive {
			runVariables[i].Value = nil
		}
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

	s.logger.WithContextFields(ctx).Infow("Created a new run.",
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

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(run.WorkspaceID))
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
		var moduleSource registry.ModuleRegistrySource

		// Create module source if this is a module and if the module digest is not nil since the module digest is required
		// for enforcing managed identity rules.
		if run.ModuleSource != nil && run.ModuleDigest != nil {
			moduleSource, err = s.moduleResolver.ParseModuleRegistrySource(ctx, *run.ModuleSource, getModuleRegistryToken([]Variable{}), getFederatedRegistry(s.dbClient, ws))
			if err != nil {
				tracing.RecordError(span, err, "failed to parse module registry source")
				return nil, err
			}
		}

		runDetails := &rules.RunDetails{
			RunStage:              models.JobApplyType,
			ModuleDigest:          run.ModuleDigest,
			CurrentStateVersionID: currentStateVersionID,
			ModuleSource:          moduleSource,
			ModuleSemanticVersion: run.ModuleVersion,
		}

		// Verify that subject has permission to create a plan for all of the assigned managed identities
		if err = s.enforceManagedIdentityRules(ctx, managedIdentities, runDetails); err != nil {
			tracing.RecordError(span, err, "failed to verify subject can enforce managed identity rules")
			return nil, err
		}
	}

	// Get apply resource
	apply, err := s.dbClient.Applies.GetApplyByID(ctx, run.ApplyID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get apply")
		return nil, errors.Wrap(
			err,
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
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for ApplyRun: %v", txErr)
		}
	}()

	_, err = s.runStateManager.UpdateApply(txContext, apply)
	if err != nil {
		tracing.RecordError(span, err, "failed to update apply")
		return nil, errors.Wrap(
			err,
			"Failed to update apply resource",
		)
	}

	now := time.Now()

	runnerTags, err := s.getJobTags(txContext, ws)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner tags for workspace")
	}

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
		Tags:           runnerTags,
	}

	// Create Job
	createdJob, err := s.dbClient.Jobs.CreateJob(txContext, &job)
	if err != nil {
		tracing.RecordError(span, err, "failed to create job")
		return nil, errors.Wrap(
			err,
			"Failed to create job",
		)
	}

	_, err = s.dbClient.LogStreams.CreateLogStream(txContext, &models.LogStream{
		JobID: &createdJob.Metadata.ID,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create log stream for apply job")
		return nil, errors.Wrap(
			err,
			"Failed to create log stream for apply job",
		)
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Applied a run.",
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

	run, err := s.GetRunByID(ctx, options.RunID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run")
		return nil, err
	}

	// Since UpdateRunPermission means run write access, we must use CreateRunPermission
	// instead i.e. if caller can create a run, they must be able to cancel it as well.
	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(run.WorkspaceID))
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
			"run has been planned and finished, so it cannot be canceled",
			errors.WithErrorCode(errors.EInvalid))
	case models.RunApplied:
		// If a run is in RunApplied state, meaning the run was for apply and that plan has finished,
		// the job cannot be canceled, so return a bad request error aka EInvalid.
		return nil, errors.New(
			"run has been applied, so it cannot be canceled",
			errors.WithErrorCode(errors.EInvalid))
	case models.RunCanceled:
		return nil, errors.New(
			"run has already been canceled",
			errors.WithErrorCode(errors.EInvalid))
	}

	// If this is a force cancel request, verify graceful cancel was already attempted
	if options.Force {
		// Verify that graceful cancel was already attempted
		if run.ForceCancelAvailableAt == nil {
			return nil, errors.New(
				"run has not already received a graceful request to cancel",
				errors.WithErrorCode(errors.EInvalid))
		}

		// Error out with errors.EInvalid if not yet eligible.
		if time.Now().Before(*run.ForceCancelAvailableAt) {
			return nil, errors.New(
				"insufficient time has elapsed since graceful cancel request; force cancel will be available at %s",
				*run.ForceCancelAvailableAt,
				errors.WithErrorCode(errors.EInvalid),
			)
		}
	}

	switch run.Status {
	case models.RunPlanned:
		// If a run is in RunPlanned state, meaning the plan job has finished but
		// the apply job has not yet been queued, cancel the run by simply doing
		// updateApply on it.

		apply, aErr := s.GetApplyByID(ctx, run.ApplyID)
		if aErr != nil {
			tracing.RecordError(span, aErr, "failed to get apply")
			return nil, errors.Wrap(
				aErr,
				"failed to get the apply object to cancel a planned run",
			)
		}

		apply.Status = models.ApplyCanceled
		_, err = s.runStateManager.UpdateApply(ctx, apply)
		if err != nil {
			tracing.RecordError(span, err, "failed to update apply")
			return nil, errors.Wrap(
				err,
				"failed to update the apply to cancel a planned run",
			)
		}

		return run, nil
	case models.RunPlanQueued:
		plan, pErr := s.GetPlanByID(ctx, run.PlanID)
		if pErr != nil {
			tracing.RecordError(span, pErr, "failed to get the plan to cancel a queued run")
			return nil, errors.Wrap(
				pErr,
				"failed to get the plan to cancel a queued run",
			)
		}

		plan.Status = models.PlanCanceled
		_, err = s.runStateManager.UpdatePlan(ctx, plan)
		if err != nil {
			tracing.RecordError(span, err, "failed to update the plan to cancel a queued run")
			return nil, errors.Wrap(
				err,
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
			"failed to create a transaction to cancel a run",
		)
	}
	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CancelRun: %v", txErr)
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
			"failed to commit the transaction to cancel a run",
		)
	}

	return updatedRun, nil
}

func (s *service) gracefullyCancelRun(ctx context.Context, run *models.Run) (*models.Run, error) {

	// Update run's ForceCancelAvailableAt.
	if run.ForceCancelAvailableAt == nil {
		now := time.Now().UTC()
		whenForceCancelAllowed := now.Add(forceCancelWait)
		run.ForceCancelAvailableAt = &whenForceCancelAllowed
	}

	// Cancel latest job associated with run
	job, err := s.jobService.GetLatestJobForRun(ctx, run)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get latest job for run",
		)
	}

	if job == nil {
		return nil, errors.New(
			"Run has no job",
			errors.WithErrorCode(errors.EInternal))
	}

	now := time.Now().UTC()
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
			"Failed to get latest job for run",
		)
	}

	if job == nil {
		return nil, errors.New(
			"Run has no job",
			errors.WithErrorCode(errors.EInternal))
	}

	// If a forced cancel, update the state of the plan or apply directly
	// and mark the workspace as having dirty state.  Do this before tell the
	// job to cancel itself to avoid risk of not recording the dirty state.

	// Update the plan or apply directly.
	switch job.Type {
	case models.JobPlanType:
		plan, err := s.GetPlanByID(ctx, run.PlanID)
		if err != nil {
			return nil, errors.Wrap(
				err,
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
		apply, err := s.GetApplyByID(ctx, run.ApplyID)
		if err != nil {
			return nil, errors.Wrap(
				err,
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

// GetRunByID returns a run by ID
func (s *service) GetRunByID(ctx context.Context, runID string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunByID")
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

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return run, nil
}

// GetRunByTRN returns a run by TRN
func (s *service) GetRunByTRN(ctx context.Context, trn string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by TRN")
		return nil, errors.Wrap(err, "failed to get run by TRN", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
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

	filter := &db.RunFilter{
		WorkspaceAssessment: input.WorkspaceAssessment,
	}

	switch {
	case input.Workspace != nil:
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithNamespacePath(input.Workspace.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		filter.WorkspaceID = &input.Workspace.Metadata.ID
	case input.Group != nil:
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		filter.GroupID = &input.Group.Metadata.ID
	default:
		// Otherwise, only return runs the user caller has access to.
		userCaller, ok := caller.(*auth.UserCaller)
		if !ok {
			return nil, errors.New("only users can query for runs without a workspace or group filter", errors.WithErrorCode(errors.EForbidden))
		}

		if !userCaller.IsAdmin() {
			// Add filter is user isn't an admin.
			filter.UserMemberID = &userCaller.User.Metadata.ID
		}
	}

	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get runs", errors.WithSpan(span))
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
			"Failed to get runs",
		)
	}

	for _, run := range result.Runs {
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
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
			"Failed to get plans",
		)
	}

	for _, plan := range result.Plans {
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithWorkspaceID(plan.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.Plans, nil
}

// GetPlan returns a tfe plan
func (s *service) GetPlanByID(ctx context.Context, planID string) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPlanByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	plan, err := s.dbClient.Plans.GetPlanByID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get plan")
		return nil, errors.Wrap(
			err,
			"Failed to get plan",
		)
	}

	if plan == nil {
		return nil, errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return plan, nil
}

func (s *service) GetPlanByTRN(ctx context.Context, trn string) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPlanByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	plan, err := s.dbClient.Plans.GetPlanByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plan by TRN", errors.WithSpan(span))
	}

	if plan == nil {
		return nil, errors.New("plan with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, plan.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return plan, nil
}

func (s *service) UpdatePlan(ctx context.Context, input *UpdatePlanInput) (*models.Plan, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdatePlan")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdatePlanPermission, auth.WithPlanID(input.PlanID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	plan, err := s.dbClient.Plans.GetPlanByID(ctx, input.PlanID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get current plan")
		return nil, err
	}

	if plan == nil {
		return nil, errors.New("plan with id %s not found", input.PlanID)
	}

	plan.Status = input.Status
	plan.HasChanges = input.HasChanges

	if input.MetadataVersion != nil {
		plan.Metadata.Version = *input.MetadataVersion
	}

	if input.ErrorMessage != nil {
		plan.ErrorMessage = sanitizeAndTruncateErrorMessage(*input.ErrorMessage)
	}

	if err := plan.Validate(); err != nil {
		tracing.RecordError(span, err, "updated plan is not valid")
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

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	result, err := s.artifactStore.GetPlanCache(ctx, run)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get plan cache from artifact store")
		return nil, errors.Wrap(
			err,
			"Failed to get plan cache from artifact store",
		)
	}

	return result, nil
}

func (s *service) GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunVariables")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByID(ctx, runID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get run")
		return nil, errors.Wrap(
			err,
			"Failed to get run",
		)
	}

	if run == nil {
		return nil, errors.New("run with ID %s not found", runID, errors.WithErrorCode(errors.ENotFound))
	}

	// Only include variable values if the caller has UpdateRunPermission or ViewVariableValuePermission on workspace.
	includeValues := false
	if err = caller.RequirePermission(ctx, models.ViewVariableValuePermission, auth.WithWorkspaceID(run.WorkspaceID)); err == nil {
		includeValues = true
	} else if err = caller.RequirePermission(ctx, models.ViewVariablePermission, auth.WithWorkspaceID(run.WorkspaceID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if !includeValues && includeSensitiveValues {
		return nil, errors.New("caller does not have permission to view sensitive variable values", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	variables, err := s.getRunVariables(ctx, run, includeSensitiveValues)
	if err != nil {
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
		if variables[i].NamespacePath == variables[j].NamespacePath {
			v = 0
		} else if variables[i].NamespacePath != nil && variables[j].NamespacePath != nil {
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

func (s *service) SetVariablesIncludedInTFConfig(ctx context.Context, input *SetVariablesIncludedInTFConfigInput) error {
	ctx, span := tracer.Start(ctx, "svc.SetVariablesIncludedInTFConfig")
	span.SetAttributes(attribute.String("run_id", input.RunID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	run, err := s.dbClient.Runs.GetRunByID(ctx, input.RunID)
	if err != nil {
		return errors.Wrap(err, "failed to get run", errors.WithSpan(span))
	}

	if run == nil {
		return errors.New("run with ID %s not found", input.RunID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Since variables should only be updated during the plan operation, we're requiring that permission here.
	if err = caller.RequirePermission(ctx, models.UpdatePlanPermission, auth.WithPlanID(run.PlanID)); err != nil {
		return errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if len(input.VariableKeys) == 0 {
		// Nothing to do.
		return nil
	}

	result, err := s.artifactStore.GetRunVariables(ctx, run)
	if err != nil {
		return errors.Wrap(err, "failed to get run variables from object store", errors.WithSpan(span))
	}
	defer result.Close()

	var variables []Variable
	if err = json.NewDecoder(result).Decode(&variables); err != nil {
		return errors.Wrap(err, "failed to decode run variables", errors.WithSpan(span))
	}

	variablesIncludedInTFConfig := make(map[string]struct{}, len(input.VariableKeys))
	for _, key := range input.VariableKeys {
		variablesIncludedInTFConfig[key] = struct{}{}
	}

	for i, variable := range variables {
		if variable.Category != models.TerraformVariableCategory {
			// We only need to filter for terraform vars.
			continue
		}

		_, hasUsage := variablesIncludedInTFConfig[variable.Key]
		variables[i].IncludedInTFConfig = &hasUsage
	}

	data, err := json.Marshal(variables)
	if err != nil {
		return errors.Wrap(err, "failed to marshal variables", errors.WithSpan(span))
	}

	if err = s.artifactStore.UploadRunVariables(ctx, run, bytes.NewReader(data)); err != nil {
		return errors.Wrap(err, "failed to upload run variables", errors.WithSpan(span))
	}

	return nil
}

func (s *service) UploadPlanBinary(ctx context.Context, planID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadPlanBinary")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdatePlanPermission, auth.WithPlanID(planID))
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
			"Failed to write plan cache to object storage",
		)
	}

	return nil
}

func (s *service) ProcessPlanData(ctx context.Context, planID string, tfPlan *tfjson.Plan, tfProviderSchemas *tfjson.ProviderSchemas) error {
	ctx, span := tracer.Start(ctx, "svc.ProcessPlanData")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdatePlanPermission, auth.WithPlanID(planID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	run, err := s.dbClient.Runs.GetRunByPlanID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return err
	}

	if run == nil {
		return errors.New("run with plan ID %s not found", planID)
	}

	diff, err := s.planParser.Parse(tfPlan, tfProviderSchemas)
	if err != nil {
		return errors.Wrap(
			err,
			"failed to create plan diff",
		)
	}

	planDiff, err := json.Marshal(diff)
	if err != nil {
		return errors.Wrap(
			err,
			"failed to marshal plan diff",
		)
	}

	planModel, err := s.dbClient.Plans.GetPlanByID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get plan")
		return err
	}

	if planModel == nil {
		return errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	// Update plan summary
	for _, change := range diff.Resources {
		switch change.Action {
		case action.Create:
			planModel.Summary.ResourceAdditions++
		case action.Update:
			planModel.Summary.ResourceChanges++
		case action.Delete:
			planModel.Summary.ResourceDestructions++
		case action.CreateThenDelete, action.DeleteThenCreate:
			planModel.Summary.ResourceAdditions++
			planModel.Summary.ResourceDestructions++
		}

		if change.Imported {
			planModel.Summary.ResourceImports++
		}

		if change.Drifted {
			planModel.Summary.ResourceDrift++
		}
	}
	for _, change := range diff.Outputs {
		switch change.Action {
		case action.Create:
			planModel.Summary.OutputAdditions++
		case action.Update:
			planModel.Summary.OutputChanges++
		case action.Delete:
			planModel.Summary.OutputDestructions++
		}
	}

	planModel.PlanDiffSize = len(planDiff)

	if _, err = s.runStateManager.UpdatePlan(ctx, planModel); err != nil {
		return errors.Wrap(
			err,
			"failed to update plan",
		)
	}

	if err = s.artifactStore.UploadPlanDiff(ctx, run, bytes.NewReader(planDiff)); err != nil {
		return errors.Wrap(
			err,
			"Failed to write plan diff to object storage",
			errors.WithSpan(span),
		)
	}

	planJSON, err := json.Marshal(tfPlan)
	if err != nil {
		return errors.Wrap(
			err,
			"failed to marshal plan json",
		)
	}

	if err := s.artifactStore.UploadPlanJSON(ctx, run, bytes.NewReader(planJSON)); err != nil {
		return errors.Wrap(
			err,
			"Failed to write plan json to object storage",
			errors.WithSpan(span),
		)
	}

	return nil
}

// GetPlanDiff returns the plan diff
func (s *service) GetPlanDiff(ctx context.Context, planID string) (*plan.Diff, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPlanDiff")
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

	if run == nil {
		return nil, errors.New("run with plan ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	reader, err := s.artifactStore.GetPlanDiff(ctx, run)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get plan diff from artifact store",
		)
	}
	defer reader.Close()

	var diff *plan.Diff
	if err := json.NewDecoder(reader).Decode(&diff); err != nil {
		return nil, errors.Wrap(
			err,
			"failed to decode plan diff",
		)
	}

	return diff, nil
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
			"Failed to list applies",
		)
	}

	for _, apply := range result.Applies {
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithWorkspaceID(apply.WorkspaceID))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
	}

	return result.Applies, nil
}

// GetApplyByID returns a tfe apply
func (s *service) GetApplyByID(ctx context.Context, applyID string) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.GetApplyByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	apply, err := s.dbClient.Applies.GetApplyByID(ctx, applyID)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get apply")
		return nil, errors.Wrap(
			err,
			"Failed to get apply",
		)
	}

	if apply == nil {
		return nil, errors.New("apply with ID %s not found", applyID, errors.WithErrorCode(errors.ENotFound))
	}

	run, err := s.dbClient.Runs.GetRunByApplyID(ctx, applyID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by apply ID")
		return nil, err
	}

	if run == nil {
		return nil, fmt.Errorf("failed to get run associated with apply id %s", applyID)
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return apply, nil
}

func (s *service) GetApplyByTRN(ctx context.Context, trn string) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.GetApplyByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	apply, err := s.dbClient.Applies.GetApplyByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get apply by TRN", errors.WithSpan(span))
	}

	if apply == nil {
		return nil, errors.New("apply with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	run, err := s.dbClient.Runs.GetRunByApplyID(ctx, apply.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by apply ID")
		return nil, err
	}

	if run == nil {
		return nil, fmt.Errorf("failed to get run associated with apply id %s", apply.Metadata.ID)
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return apply, nil
}

func (s *service) UpdateApply(ctx context.Context, input *UpdateApplyInput) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateApply")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateApplyPermission, auth.WithApplyID(input.ApplyID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	apply, err := s.dbClient.Applies.GetApplyByID(ctx, input.ApplyID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get current apply")
		return nil, err
	}

	if apply == nil {
		return nil, errors.New("apply with id %s not found", input.ApplyID)
	}

	apply.Status = input.Status

	if input.MetadataVersion != nil {
		apply.Metadata.Version = *input.MetadataVersion
	}

	if input.ErrorMessage != nil {
		apply.ErrorMessage = sanitizeAndTruncateErrorMessage(*input.ErrorMessage)
	}

	if err := apply.Validate(); err != nil {
		tracing.RecordError(span, err, "updated apply is not valid")
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

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithWorkspaceID(run.WorkspaceID), auth.WithJobID(job.Metadata.ID))
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

	err = caller.RequirePermission(ctx, models.ViewJobPermission, auth.WithWorkspaceID(run.WorkspaceID), auth.WithJobID(job.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return job, nil
}

func (s *service) GetStateVersionsByRunIDs(ctx context.Context, runIDs []string) ([]models.StateVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetStateVersionsByRunIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	runsResult, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Filter: &db.RunFilter{
			RunIDs: runIDs,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get runs", errors.WithSpan(span))
	}

	for _, run := range runsResult.Runs {
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
		if err != nil {
			return nil, err
		}
	}

	if runsResult.PageInfo.TotalCount > 0 {
		result, err := s.dbClient.StateVersions.GetStateVersions(ctx, &db.GetStateVersionsInput{
			Filter: &db.StateVersionFilter{
				RunIDs: runIDs,
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get state versions", errors.WithSpan(span))
		}

		return result.StateVersions, nil
	}

	return []models.StateVersion{}, nil
}

func (s *service) getRunVariables(ctx context.Context, run *models.Run, includeSensitiveValues bool) ([]Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.getRunVariables")
	defer span.End()

	result, err := s.artifactStore.GetRunVariables(ctx, run)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run variables from object store")
		return nil, errors.Wrap(
			err,
			"Failed to get run variables from object store",
		)
	}
	defer result.Close()

	var variables []Variable
	if err := json.NewDecoder(result).Decode(&variables); err != nil {
		tracing.RecordError(span, err, "failed to decode run variables")
		return nil, err
	}

	if includeSensitiveValues {
		runID := run.Metadata.ID

		// Extract variable version IDs for sensitive variables
		variableVersionIDs := make([]string, 0, len(variables))
		for _, v := range variables {
			if v.Sensitive {
				if v.VersionID == nil {
					return nil, errors.New("variable version ID is missing for sensitive variable %q in run %q", v.Key, runID, errors.WithSpan(span))
				}
				variableVersionIDs = append(variableVersionIDs, *v.VersionID)
			}
		}

		if len(variableVersionIDs) > 0 {
			// Query for variable versions
			variableVersionsResp, err := s.dbClient.VariableVersions.GetVariableVersions(ctx, &db.GetVariableVersionsInput{
				Filter: &db.VariableVersionFilter{
					VariableVersionIDs: variableVersionIDs,
				},
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to query for variable versions associated with run %q", runID, errors.WithSpan(span))
			}

			// Ensure that we recieved all the requested variable versions
			if len(variableVersionsResp.VariableVersions) != len(variableVersionIDs) {
				return nil, errors.New("some of the requested variable versions are missing for run %q", runID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}

			// Build map of secret values
			secretValues := make(map[string]string, len(variableVersionsResp.VariableVersions))
			for _, v := range variableVersionsResp.VariableVersions {
				// Use secret manager to get the secret value
				value, err := s.secretManager.Get(ctx, v.Key, v.SecretData)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get secret value for variable version with ID %q", v.Metadata.ID, errors.WithSpan(span))
				}
				secretValues[v.Metadata.ID] = value
			}

			// Populate sensitive variable values
			for i, v := range variables {
				if v.Sensitive {
					if value, ok := secretValues[*v.VersionID]; ok {
						variables[i].Value = &value
					} else {
						return nil, errors.New("failed to populate secret value for variable version %q because secret value was not found", *v.VersionID, errors.WithSpan(span))
					}
				}
			}
		}
	}

	return variables, nil
}

func (s *service) buildRunVariables(ctx context.Context, workspaceID string, runVariables []Variable) ([]Variable, error) {
	variableMap := map[string]Variable{}

	buildMapKey := func(key string, category string) string {
		return fmt.Sprintf("%s::%s", key, category)
	}

	// Add run variables first since they have the highest precedence
	for _, v := range runVariables {
		if v.Category == models.EnvironmentVariableCategory && v.Hcl {
			return nil, errors.New("HCL variables are not supported for the environment category", errors.WithErrorCode(errors.EInvalid))
		}

		variableMap[buildMapKey(v.Key, string(v.Category))] = Variable{
			Key:       v.Key,
			Value:     v.Value,
			Category:  v.Category,
			Hcl:       v.Hcl,
			Sensitive: false,
		}
	}

	// Get Workspace
	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	if ws == nil {
		return nil, fmt.Errorf("workspace with id %s not found", workspaceID)
	}

	// Use a descending sort so the variables from the closest ancestor will take precedence
	sortBy := db.VariableSortableFieldNamespacePathDesc
	result, err := s.dbClient.Variables.GetVariables(ctx, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: ws.ExpandPath(),
		},
		Sort: &sortBy,
	})
	if err != nil {
		return nil, err
	}

	for _, v := range result.Variables {
		v := v

		keyAndCategory := buildMapKey(v.Key, string(v.Category))
		if _, ok := variableMap[keyAndCategory]; !ok {
			value := v.Value
			// Get secret value if variable is sensitive
			if v.Sensitive {
				// Use secret manager to get the secret value
				secret, err := s.secretManager.Get(ctx, v.Key, v.SecretData)
				if err != nil {
					return nil, errors.Wrap(err, "failed to get secret value for variable %q when saving run variables %q", v.Key)
				}
				value = &secret
			}

			variableMap[keyAndCategory] = Variable{
				Key:           v.Key,
				Value:         value,
				Category:      v.Category,
				Hcl:           v.Hcl,
				NamespacePath: &v.NamespacePath,
				Sensitive:     v.Sensitive,
				VersionID:     &v.LatestVersionID,
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
		return nil, errors.New("latest %s job for run %s not found", jobType, runID, errors.WithErrorCode(errors.ENotFound))
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
	run, err := s.dbClient.Runs.GetRunByID(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get run",
		)
	}

	if run == nil {
		return nil, errors.New("run with ID %s not found", runID, errors.WithErrorCode(errors.ENotFound))
	}

	return run, nil
}

// getJobTags gets the applicable runner tags from the workspace or the lowest level ancestor group with tags set.
func (s *service) getJobTags(ctx context.Context, workspace *models.Workspace) ([]string, error) {
	setting, err := namespace.NewInheritedSettingResolver(s.dbClient).GetRunnerTags(ctx, workspace)
	if err != nil {
		return nil, err
	}

	return setting.Value, nil
}

// sanitizeAndTruncateErrorMessage sanitizes UTF-8 characters and truncates if needed
func sanitizeAndTruncateErrorMessage(errorMessage string) *string {
	// First sanitize UTF-8 - replace invalid sequences with replacement character
	sanitized := strings.ToValidUTF8(errorMessage, "�")

	if len(sanitized) > maxErrorMessageLength {
		truncated := fmt.Sprintf(
			"%s...\n%s",
			sanitized[:maxErrorMessageLength],
			ansi.Colorize("Error message has been truncated, check the logs for the full error message", ansi.Yellow),
		)
		return &truncated
	}
	return &sanitized
}

func getModuleRegistryToken(envVars []Variable) registry.TokenGetterFunc {
	return func(_ context.Context, hostname string) (string, error) {
		seeking, err := module.BuildTokenEnvVar(hostname)
		if err == nil {
			for _, variable := range envVars {
				if variable.Key == seeking {
					return *variable.Value, nil
				}
			}
		}
		return "", nil
	}
}

func getFederatedRegistry(dbClient *db.Client, workspace *models.Workspace) registry.FederatedRegistryGetterFunc {
	// Search all parent group paths for a federated registry that matches this host
	return func(ctx context.Context, hostname string) (*models.FederatedRegistry, error) {
		federatedRegistries, err := registry.GetFederatedRegistries(ctx, &registry.GetFederatedRegistriesInput{
			DBClient:  dbClient,
			GroupPath: workspace.GetGroupPath(),
			Hostname:  &hostname,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get federated registries")
		}

		if len(federatedRegistries) > 0 {
			return federatedRegistries[0], nil
		}

		return nil, nil
	}
}
