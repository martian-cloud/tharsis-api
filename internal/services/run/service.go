// Package run provides the run service for creating and managing runs
package run

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"
	tfjson "github.com/hashicorp/terraform-json"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/commands"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	namespaceutils "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

// Event represents a run event
type Event struct {
	Action string
	Run    *models.Run
}

// EventSubscriptionOptions provides options for subscribing to run events
type EventSubscriptionOptions struct {
	WorkspaceID     *string
	RunID           *string // RunID is optional
	AncestorGroupID *string
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
	// IncludeNestedRuns indicates whether to include runs from nested namespaces
	IncludeNestedRuns *bool
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
	AutoApply              bool  // when true, the apply starts automatically after a plan finishes with changes
	WorkspaceID            string
	TerraformVersion       string
	Variables              []runvariables.Variable
	TargetAddresses        []string
	IsDestroy              bool
	// Refresh is optional; nil means "not explicitly set" and resolves to true
	// (Terraform's default) during run creation.
	Refresh     *bool
	RefreshOnly bool
	// IncludeModulePrereleases, when true and ModuleVersion is nil or a
	// constraint range, allows prerelease module versions to be selected as
	// "latest". Has no effect when ModuleVersion is an exact match (which
	// already resolves to that version regardless of prerelease status).
	// Requires ModuleSource to be set.
	// TODO: pair this with a workspace-level default (e.g.,
	// Workspace.PreferModulePrereleases) so users can opt in once per
	// workspace instead of on every run; per-run flag should override the
	// workspace default.
	IncludeModulePrereleases bool
}

// CreateDestroyRunForWorkspaceInput is the input for creating a destroy run using the current
// configuration version or module that is applied.
type CreateDestroyRunForWorkspaceInput struct {
	WorkspaceID string
}

// CreateReconcileRunForWorkspaceInput is the input for creating a reconcile run using the current
// configuration version or module that is applied.
type CreateReconcileRunForWorkspaceInput struct {
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
		return errors.New("must supply either configuration version ID or module source", errors.WithErrorCode(errors.EInvalid))
	}

	// Check that there is no more than one of configuration version and module source.
	if (c.ConfigurationVersionID != nil) && (c.ModuleSource != nil) {
		return errors.New("must supply configuration version ID or module source but not both", errors.WithErrorCode(errors.EInvalid))
	}

	// Check that there is no more than one of configuration version and module version.
	if (c.ConfigurationVersionID != nil) && (c.ModuleVersion != nil) {
		return errors.New("must supply configuration version ID or module version but not both", errors.WithErrorCode(errors.EInvalid))
	}

	// Make sure module version is not specified without module source.
	if (c.ModuleSource == nil) && (c.ModuleVersion != nil) {
		return errors.New("module version is not allowed without module source", errors.WithErrorCode(errors.EInvalid))
	}

	// Make sure includeModulePrereleases is not set without a module source.
	if (c.ModuleSource == nil) && c.IncludeModulePrereleases {
		return errors.New("includeModulePrereleases is not allowed without module source", errors.WithErrorCode(errors.EInvalid))
	}

	// If a module version is specified, validate it.
	if c.ModuleVersion != nil {
		if *c.ModuleVersion == "" {
			return errors.New("module version cannot be empty; please specify a valid semantic version", errors.WithErrorCode(errors.EInvalid))
		}
		if *c.ModuleVersion == "latest" {
			return errors.New("'latest' is not a valid module version; please specify a valid semantic version", errors.WithErrorCode(errors.EInvalid))
		}

		// Make sure it's a valid semver version or constraint expression.
		_, err := version.NewConstraint(*c.ModuleVersion)
		if err != nil {
			return errors.New("module version is not a valid semver version or constraint expression", errors.WithErrorCode(errors.EInvalid))
		}
	}

	// Don't allow refresh_only in combination with other options that would conflict.
	if c.RefreshOnly && c.IsDestroy {
		return errors.New("refresh_only is not allowed with destroy", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// CancelRunInput is the input for canceling a run.
type CancelRunInput struct {
	Comment *string
	RunID   string
	Force   bool
}

// RetryRunNodeInput is the input for retrying a run's plan or apply node, identified
// by RunID and NodePath ("plan" or "apply").
type RetryRunNodeInput struct {
	RunID    string
	NodePath string
}

// DiscardRunInput is the input for discarding a planned run.
type DiscardRunInput struct {
	RunID string
}

// UndiscardRunInput is the input for undiscarding a discarded run.
type UndiscardRunInput struct {
	RunID string
}

// UpdateApplyInput is the input for updating an apply
type UpdateApplyInput struct {
	ApplyID         string
	ErrorMessage    *string
	MetadataVersion *int
}

// UpdatePlanInput is the input for updating a plan
type UpdatePlanInput struct {
	ErrorMessage    *string
	HasChanges      bool
	MetadataVersion *int
	PlanID          string
}

// Service encapsulates Terraform Enterprise Support
type Service interface {
	GetRunByID(ctx context.Context, runID string) (*models.Run, error)
	GetRunByTRN(ctx context.Context, trn string) (*models.Run, error)
	GetRunByNodeID(ctx context.Context, nodeID string) (*models.Run, error)
	GetRuns(ctx context.Context, input *GetRunsInput) (*db.RunsResult, error)
	GetRunsByIDs(ctx context.Context, idList []string) ([]*models.Run, error)
	CreateRun(ctx context.Context, options *CreateRunInput) (*models.Run, error)
	ApplyRun(ctx context.Context, runID string, comment *string) (*models.Run, error)
	SetRunAutoApply(ctx context.Context, runID string, autoApply bool) (*models.Run, error)
	CancelRun(ctx context.Context, options *CancelRunInput) (*models.Run, error)
	RetryRunNode(ctx context.Context, options *RetryRunNodeInput) (*models.Run, error)
	DiscardRun(ctx context.Context, options *DiscardRunInput) (*models.Run, error)
	UndiscardRun(ctx context.Context, options *UndiscardRunInput) (*models.Run, error)
	GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]runvariables.Variable, error)
	CreateAssessmentRunForWorkspace(ctx context.Context, options *CreateAssessmentRunForWorkspaceInput) (*models.Run, error)
	CreateDestroyRunForWorkspace(ctx context.Context, options *CreateDestroyRunForWorkspaceInput) (*models.Run, error)
	CreateReconcileRunForWorkspace(ctx context.Context, options *CreateReconcileRunForWorkspaceInput) (*models.Run, error)
	SetVariablesIncludedInTFConfig(ctx context.Context, input *SetVariablesIncludedInTFConfigInput) error
	GetPlanDiff(ctx context.Context, planID string) (*plan.Diff, error)
	GetPlanCheckResults(ctx context.Context, planID string) ([]corerun.CheckResult, error)
	UpdatePlan(ctx context.Context, input *UpdatePlanInput) (*models.Plan, error)
	DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error)
	UploadPlanBinary(ctx context.Context, planID string, reader io.Reader) error
	ProcessPlanData(ctx context.Context, planID string, plan *tfjson.Plan, providerSchemas *tfjson.ProviderSchemas) error
	UpdateApply(ctx context.Context, input *UpdateApplyInput) (*models.Apply, error)
	SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error)
	GetStateVersionsByRunIDs(ctx context.Context, idList []string) ([]models.StateVersion, error)
}

type service struct {
	logger           logger.Logger
	dbClient         *db.Client
	cmdProcessor     engine.CmdProcessor
	cmdFactory       *commands.Factory
	artifactStore    workspace.ArtifactStore
	eventManager     *events.EventManager
	variablesBuilder *runvariables.Builder
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	cmdProcessor engine.CmdProcessor,
	cmdFactory *commands.Factory,
	artifactStore workspace.ArtifactStore,
	eventManager *events.EventManager,
	variablesBuilder *runvariables.Builder,
) Service {
	return &service{
		logger:           logger,
		dbClient:         dbClient,
		cmdProcessor:     cmdProcessor,
		cmdFactory:       cmdFactory,
		artifactStore:    artifactStore,
		eventManager:     eventManager,
		variablesBuilder: variablesBuilder,
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

	var rootNamespaceMemberships []models.MembershipNamespace
	// checkRootNamespaceMemberships is true when each run's workspace must be verified against
	// the caller's root namespace memberships. It stays false when the caller has admin mode
	// activated (sees everything) or is filtering by a specific workspace (ViewRun permission
	// already verified below).
	checkRootNamespaceMemberships := false
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

		if !userCaller.IsAdminModeActivated(ctx) {
			rootNamespaces, rErr := userCaller.GetRootNamespaceMemberships(ctx)
			if rErr != nil {
				tracing.RecordError(span, rErr, "failed to get root namespaces")
				return nil, rErr
			}
			rootNamespaceMemberships = rootNamespaces
			checkRootNamespaceMemberships = true
		}
	}

	// Pre-fetch target group if group filtering is enabled
	var ancestorGroupPath string
	if options.AncestorGroupID != nil {
		targetGroup, err := s.dbClient.Groups.GetGroupByID(ctx, *options.AncestorGroupID)
		if err != nil {
			tracing.RecordError(span, err, "failed to query target group")
			return nil, err
		}
		if targetGroup == nil {
			return nil, errors.New("target group not found", errors.WithErrorCode(errors.ENotFound))
		}
		ancestorGroupPath = targetGroup.FullPath
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
			// Manually verify the caller can view the run by checking the run's workspace path
			// against the caller's root namespace memberships before querying the run itself,
			// avoiding a wasted run query when the caller doesn't have access. Skipped when the
			// caller has admin mode activated or is filtering by a specific workspace (already
			// permission checked).
			if checkRootNamespaceMemberships {
				ws, wErr := s.dbClient.Workspaces.GetWorkspaceByID(ctx, eventData.WorkspaceID)
				if wErr != nil {
					if errors.IsContextCanceledError(wErr) || errors.IsDeadlineExceededError(wErr) {
						return
					}
					s.logger.WithContextFields(ctx).Errorf("Error occurred while querying for workspace %s associated with run event %s: %v", eventData.WorkspaceID, event.ID, wErr)
					continue
				}

				if ws == nil || !callerHasRootNamespaceAccess(ws.FullPath, rootNamespaceMemberships) {
					// Caller doesn't have access to the run's workspace.
					continue
				}
			}

			run, err := s.dbClient.Runs.GetRunByID(ctx, event.ID)
			if err != nil {
				if errors.IsContextCanceledError(err) || errors.IsDeadlineExceededError(err) {
					return
				}
				s.logger.WithContextFields(ctx).Errorf("Error occurred while querying for run associated with run event %s: %v", event.ID, err)
				continue
			}

			if run == nil {
				// Run no longer exists.
				continue
			}

			// Group filtering: check if run's workspace belongs to the specified group using TRN
			if options.AncestorGroupID != nil {
				var isInGroup bool
				isInGroup, err = s.isRunInTargetGroup(run, ancestorGroupPath)
				if err != nil {
					s.logger.WithContextFields(ctx).Errorf("Error checking run group membership for run %s: %v", run.Metadata.ID, err)
					continue
				}

				if !isInGroup {
					continue
				}
			}

			select {
			case <-ctx.Done():
				return
			case outgoing <- &Event{Action: event.Action, Run: run}:
			}
		}
	}()

	return outgoing, nil
}

// callerHasRootNamespaceAccess returns true if the workspace path is at or under one of the
// caller's root namespace memberships, mirroring the DB-level root namespace membership filter.
func callerHasRootNamespaceAccess(workspacePath string, rootNamespaceMemberships []models.MembershipNamespace) bool {
	for _, ns := range rootNamespaceMemberships {
		if workspacePath == ns.Path || namespaceutils.IsDescendantOfPath(workspacePath, ns.Path) {
			return true
		}
	}
	return false
}

// isRunInTargetGroup checks if a run's workspace belongs to the specified group or its descendants
func (s *service) isRunInTargetGroup(run *models.Run, ancestorGroupPath string) (bool, error) {
	workspaceGroupPath := run.GetGroupPath()

	// Check if workspace's group matches or is descendant of target group
	return workspaceGroupPath == ancestorGroupPath || namespaceutils.IsDescendantOfPath(workspaceGroupPath, ancestorGroupPath), nil
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

	// The assessment command derives its inputs (source run + variables) from the
	// workspace's latest applied run in its Prepare phase, then upserts the
	// assessment record and creates the run in a single transaction.
	cmd := s.cmdFactory.NewCreateAssessmentRun(&commands.CreateAssessmentRunInput{
		Subject:                 caller.GetSubject(),
		WorkspaceID:             options.WorkspaceID,
		LatestAssessmentVersion: options.LatestAssessmentVersion,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to create assessment run")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Created an assessment run.",
		"workspaceID", options.WorkspaceID,
		"runID", cmd.Created.Metadata.ID,
	)
	return cmd.Created, nil
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

	cmd := s.cmdFactory.NewCreateDestroyRun(&commands.CreateDestroyRunInput{
		Subject:     caller.GetSubject(),
		WorkspaceID: options.WorkspaceID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to create destroy run")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Created a destroy run.",
		"workspaceID", options.WorkspaceID,
		"runID", cmd.Created.Metadata.ID,
	)
	return cmd.Created, nil
}

func (s *service) CreateReconcileRunForWorkspace(ctx context.Context, options *CreateReconcileRunForWorkspaceInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateReconcileRunForWorkspace")
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

	cmd := s.cmdFactory.NewCreateReconcileRun(&commands.CreateReconcileRunInput{
		Subject:     caller.GetSubject(),
		WorkspaceID: options.WorkspaceID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to create reconcile run")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Created a reconcile run.",
		"workspaceID", options.WorkspaceID,
		"runID", cmd.Created.Metadata.ID,
	)
	return cmd.Created, nil
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

	// The command's Prepare phase builds the run variables (merging the
	// workspace's inherited variables), normalizes the module version, and
	// resolves the module source — all before the transaction is opened.
	cmd := s.cmdFactory.NewRun(&commands.NewRunInput{
		Subject:                  caller.GetSubject(),
		WorkspaceID:              options.WorkspaceID,
		ConfigurationVersionID:   options.ConfigurationVersionID,
		Comment:                  options.Comment,
		ModuleSource:             options.ModuleSource,
		ModuleVersion:            options.ModuleVersion,
		Speculative:              options.Speculative,
		AutoApply:                options.AutoApply,
		TerraformVersion:         options.TerraformVersion,
		Variables:                options.Variables,
		TargetAddresses:          options.TargetAddresses,
		IsDestroy:                options.IsDestroy,
		Refresh:                  options.Refresh,
		RefreshOnly:              options.RefreshOnly,
		IncludeModulePrereleases: options.IncludeModulePrereleases,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to create run")
		return nil, err
	}

	run := cmd.Created
	s.logger.WithContextFields(ctx).Infow("Created a new run.",
		"workspaceID", run.WorkspaceID,
		"runID", run.Metadata.ID,
		"configurationVersionID", options.ConfigurationVersionID,
		"moduleSource", options.ModuleSource,
		"requestedModuleVersion", options.ModuleVersion,
		"resolvedModuleVersion", run.ModuleVersion,
		"terraformVersion", options.TerraformVersion,
		"targetAddresses", options.TargetAddresses,
		"refresh", run.Refresh,
		"refreshOnly", options.RefreshOnly,
		"speculative", options.Speculative,
		"autoApply", options.AutoApply,
		"isDestroy", options.IsDestroy,
		"includeModulePrereleases", options.IncludeModulePrereleases,
	)
	return run, nil
}

// ApplyRun executes the apply action on an existing run
// authorizeRunMutation authorizes the caller for a run mutation command: it resolves
// the caller, fetches the run, and requires CreateRunPermission on the run's
// workspace. All run-mutation entry points (apply, cancel, retry, discard) share it
// so they authorize identically.
func (s *service) authorizeRunMutation(ctx context.Context, runID string) (auth.Caller, *models.Run, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, nil, err
	}

	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, nil, err
	}

	if err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(run.WorkspaceID)); err != nil {
		return nil, nil, err
	}

	return caller, run, nil
}

func (s *service) ApplyRun(ctx context.Context, runID string, comment *string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.ApplyRun")
	defer span.End()

	caller, run, err := s.authorizeRunMutation(ctx, runID)
	if err != nil {
		tracing.RecordError(span, err, "run mutation authorization failed")
		return nil, err
	}

	commentStr := ""
	if comment != nil {
		commentStr = *comment
	}

	cmd := s.cmdFactory.NewStartApply(&commands.StartApplyInput{
		RunID:       runID,
		TriggeredBy: caller.GetSubject(),
		Comment:     commentStr,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to start apply")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Applied a run.",
		"workspaceID", run.WorkspaceID,
		"runStatus", cmd.Updated.Status,
		"runID", runID,
	)
	return cmd.Updated, nil
}

func (s *service) SetRunAutoApply(ctx context.Context, runID string, autoApply bool) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.SetRunAutoApply")
	defer span.End()

	_, run, err := s.authorizeRunMutation(ctx, runID)
	if err != nil {
		tracing.RecordError(span, err, "run mutation authorization failed")
		return nil, err
	}

	cmd := s.cmdFactory.NewSetRunAutoApply(&commands.SetRunAutoApplyInput{
		RunID:     runID,
		AutoApply: autoApply,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to set run auto-apply")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Set run auto-apply.",
		"workspaceID", run.WorkspaceID,
		"autoApply", autoApply,
		"runID", runID,
	)
	return cmd.Updated, nil
}

func (s *service) CancelRun(ctx context.Context, options *CancelRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.CancelRun")
	defer span.End()

	caller, _, err := s.authorizeRunMutation(ctx, options.RunID)
	if err != nil {
		tracing.RecordError(span, err, "run mutation authorization failed")
		return nil, err
	}

	cmd := s.cmdFactory.NewCancelRun(&commands.CancelRunInput{
		RunID:      options.RunID,
		CanceledBy: caller.GetSubject(),
		Force:      options.Force,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to cancel run")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Canceled a run.",
		"runID", options.RunID,
		"runStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
}

func (s *service) RetryRunNode(ctx context.Context, options *RetryRunNodeInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.RetryRunNode")
	defer span.End()

	if _, _, err := s.authorizeRunMutation(ctx, options.RunID); err != nil {
		tracing.RecordError(span, err, "run mutation authorization failed")
		return nil, err
	}

	cmd := s.cmdFactory.NewRetryRunNode(&commands.RetryRunNodeInput{
		RunID:    options.RunID,
		NodePath: options.NodePath,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to retry run node")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Retried a run node.",
		"runID", options.RunID,
		"nodePath", options.NodePath,
		"runStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
}

func (s *service) DiscardRun(ctx context.Context, options *DiscardRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.DiscardRun")
	defer span.End()

	if _, _, err := s.authorizeRunMutation(ctx, options.RunID); err != nil {
		tracing.RecordError(span, err, "run mutation authorization failed")
		return nil, err
	}

	cmd := s.cmdFactory.NewDiscardRun(&commands.DiscardRunInput{
		RunID: options.RunID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to discard run")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Discarded a run.",
		"runID", options.RunID,
		"runStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
}

func (s *service) UndiscardRun(ctx context.Context, options *UndiscardRunInput) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.UndiscardRun")
	defer span.End()

	if _, _, err := s.authorizeRunMutation(ctx, options.RunID); err != nil {
		tracing.RecordError(span, err, "run mutation authorization failed")
		return nil, err
	}

	cmd := s.cmdFactory.NewUndiscardRun(&commands.UndiscardRunInput{
		RunID: options.RunID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to undiscard run")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Undiscarded a run.",
		"runID", options.RunID,
		"runStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
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

// GetRunByNodeID returns a run by a plan or apply node ID
func (s *service) GetRunByNodeID(ctx context.Context, nodeID string) (*models.Run, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunByNodeID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, nodeID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by node ID")
		return nil, errors.Wrap(err, "failed to get run by node ID", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run with node ID %s not found", nodeID, errors.WithErrorCode(errors.ENotFound))
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

		if !userCaller.IsAdminModeActivated(ctx) {
			// Restrict to runs in the user's member namespaces (and descendants).
			rootNamespaces, rErr := userCaller.GetRootNamespaceMemberships(ctx)
			if rErr != nil {
				tracing.RecordError(span, rErr, "failed to get root namespaces")
				return nil, rErr
			}
			filter.RootNamespaceMemberships = rootNamespaces
		}
	}
	if input.IncludeNestedRuns != nil && *input.IncludeNestedRuns && input.Group == nil {
		return nil, errors.New("IncludeNestedRuns can only be used with Group filter", errors.WithErrorCode(errors.EInvalid))
	}
	filter.IncludeNestedRuns = input.IncludeNestedRuns

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

func (s *service) GetRunsByIDs(ctx context.Context, idList []string) ([]*models.Run, error) {
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

	cmd := s.cmdFactory.NewUpdatePlan(input.PlanID, input.HasChanges, input.ErrorMessage)
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to update plan node")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Updated a plan.",
		"planID", input.PlanID,
		"planStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
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

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return nil, err
	}

	if run == nil {
		return nil, errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
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

func (s *service) GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]runvariables.Variable, error) {
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

	variables, err := s.variablesBuilder.Get(ctx, run, includeSensitiveValues)
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
	if err = caller.RequirePermission(ctx, models.UpdatePlanPermission, auth.WithPlanID(run.Plan.GetID())); err != nil {
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

	var variables []runvariables.Variable
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

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get run by plan ID")
		return err
	}

	if run == nil {
		return errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
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

	cmd := s.cmdFactory.NewUpdatePlanSummary(&commands.UpdatePlanSummaryInput{
		PlanID:            planID,
		TFPlan:            tfPlan,
		TFProviderSchemas: tfProviderSchemas,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to update plan summary")
		return err
	}

	s.logger.WithContextFields(ctx).Infow("Processed plan data.",
		"planID", planID,
	)
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

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
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

// GetPlanCheckResults returns check results from the plan JSON
func (s *service) GetPlanCheckResults(ctx context.Context, planID string) ([]corerun.CheckResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPlanCheckResults")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by plan ID", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run with plan ID %s not found", planID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	reader, err := s.artifactStore.GetPlanJSON(ctx, run)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get plan JSON from artifact store", errors.WithSpan(span))
	}
	defer reader.Close()

	var tfPlan tfjson.Plan
	if err := json.NewDecoder(reader).Decode(&tfPlan); err != nil {
		return nil, errors.Wrap(err, "failed to decode plan JSON", errors.WithSpan(span))
	}

	results := []corerun.CheckResult{}
	for _, check := range tfPlan.Checks {
		objects := []corerun.CheckResultObject{}
		for _, instance := range check.Instances {
			var failureMessages []string
			for _, problem := range instance.Problems {
				failureMessages = append(failureMessages, problem.Message)
			}
			objects = append(objects, corerun.CheckResultObject{
				Address:         instance.Address.ToDisplay,
				Status:          corerun.NormalizeCheckStatus(string(instance.Status)),
				FailureMessages: failureMessages,
			})
		}
		results = append(results, corerun.CheckResult{
			Name:    check.Address.ToDisplay,
			Status:  corerun.NormalizeCheckStatus(string(check.Status)),
			Objects: objects,
		})
	}

	return results, nil
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

	cmd := s.cmdFactory.NewUpdateApply(input.ApplyID, input.ErrorMessage)
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		tracing.RecordError(span, err, "failed to update apply node")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Updated an apply.",
		"applyID", input.ApplyID,
		"applyStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
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

	if runsResult.PageInfo.HasResults {
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
