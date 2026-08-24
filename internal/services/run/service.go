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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
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

// ApproveRunGateInput is the input for recording an approve/reject decision on a run gate.
type ApproveRunGateInput struct {
	GateID   string
	Decision models.RunGateDecision
	Comment  *string
}

// GetRunGatesInput is the input for querying a list of run gates.
type GetRunGatesInput struct {
	Sort              *db.RunGateSortableField
	PaginationOptions *pagination.Options
	RunID             string
}

// GetRunGatesAwaitingDecisionInput is the input for the caller's approvals inbox.
type GetRunGatesAwaitingDecisionInput struct {
	Sort              *db.RunGateSortableField
	PaginationOptions *pagination.Options
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

// RetryRunNodeInput is the input for retrying a run's node, identified by RunID and NodePath ("plan",
// "apply", or a policy check path such as "post_plan.opa"). A plan or apply must be failed or canceled;
// a policy check may also be soft-failed.
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

// PolicyOutcome is a single policy's reported result for a policy check, keyed by its
// PolicyCheckPolicy id. Messages holds one entry per violation, in display order, and is empty when
// the policy passed.
type PolicyOutcome struct {
	PolicyID string
	Messages []string
	Passed   bool
}

// ReportRunPolicyOutcomesInput is the input for reporting a policy check's per-policy-set outcomes.
type ReportRunPolicyOutcomesInput struct {
	PolicyCheckID string
	Outcomes      []*PolicyOutcome
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
	GetPolicyCheckPolicyMessages(ctx context.Context, policyCheckID string, policyID string) ([]string, error)
	UpdatePlan(ctx context.Context, input *UpdatePlanInput) (*models.Plan, error)
	ReportRunPolicyOutcomes(ctx context.Context, input *ReportRunPolicyOutcomesInput) error
	DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error)
	DownloadPlanJSON(ctx context.Context, planID string) (io.ReadCloser, error)
	UploadPlanBinary(ctx context.Context, planID string, reader io.Reader) error
	ProcessPlanData(ctx context.Context, planID string, plan *tfjson.Plan, providerSchemas *tfjson.ProviderSchemas) error
	UpdateApply(ctx context.Context, input *UpdateApplyInput) (*models.Apply, error)
	SubscribeToRunEvents(ctx context.Context, options *EventSubscriptionOptions) (<-chan *Event, error)
	GetStateVersionsByRunIDs(ctx context.Context, idList []string) ([]models.StateVersion, error)
	GetRunGateByID(ctx context.Context, id string) (*models.RunGate, error)
	GetRunGateByTRN(ctx context.Context, trn string) (*models.RunGate, error)
	GetRunGates(ctx context.Context, input *GetRunGatesInput) (*db.RunGatesResult, error)
	GetRunGatesByIDs(ctx context.Context, ids []string) ([]*models.RunGate, error)
	GetRunGatesByPolicyCheckIDs(ctx context.Context, policyCheckIDs []string) ([]*models.RunGate, error)
	GetRunGateApprovalByID(ctx context.Context, id string) (*models.RunGateApproval, error)
	GetRunGateApprovalByTRN(ctx context.Context, trn string) (*models.RunGateApproval, error)
	GetRunGateApprovalsByGateID(ctx context.Context, gateID string) ([]models.RunGateApproval, error)
	GetRunGatesAwaitingDecision(ctx context.Context, input *GetRunGatesAwaitingDecisionInput) (*db.RunGatesResult, error)
	ApproveRunGate(ctx context.Context, input *ApproveRunGateInput) (*models.RunGate, error)
	OverrideRunGate(ctx context.Context, gateID string, comment *string) (*models.RunGate, error)
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
				return nil, errors.Wrap(rErr, "failed to get root namespaces", errors.WithSpan(span))
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
			return nil, errors.Wrap(err, "failed to query target group", errors.WithSpan(span))
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
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
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
		return nil, errors.Wrap(err, "failed to create assessment run", errors.WithSpan(span))
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
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	cmd := s.cmdFactory.NewCreateDestroyRun(&commands.CreateDestroyRunInput{
		Subject:     caller.GetSubject(),
		WorkspaceID: options.WorkspaceID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to create destroy run", errors.WithSpan(span))
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
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	cmd := s.cmdFactory.NewCreateReconcileRun(&commands.CreateReconcileRunInput{
		Subject:     caller.GetSubject(),
		WorkspaceID: options.WorkspaceID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to create reconcile run", errors.WithSpan(span))
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
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateRunPermission, auth.WithWorkspaceID(options.WorkspaceID))
	if err != nil {
		return nil, err
	}

	if err = options.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate create run options", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to create run", errors.WithSpan(span))
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
		return nil, errors.Wrap(err, "failed to start apply", errors.WithSpan(span))
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
		return nil, err
	}

	cmd := s.cmdFactory.NewSetRunAutoApply(&commands.SetRunAutoApplyInput{
		RunID:     runID,
		AutoApply: autoApply,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to set run auto-apply", errors.WithSpan(span))
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
		return nil, err
	}

	cmd := s.cmdFactory.NewCancelRun(&commands.CancelRunInput{
		RunID:      options.RunID,
		CanceledBy: caller.GetSubject(),
		Force:      options.Force,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to cancel run", errors.WithSpan(span))
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
		return nil, err
	}

	cmd := s.cmdFactory.NewRetryRunNode(&commands.RetryRunNodeInput{
		RunID:    options.RunID,
		NodePath: options.NodePath,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to retry run node", errors.WithSpan(span))
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
		return nil, err
	}

	cmd := s.cmdFactory.NewDiscardRun(&commands.DiscardRunInput{
		RunID: options.RunID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to discard run", errors.WithSpan(span))
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
		return nil, err
	}

	cmd := s.cmdFactory.NewUndiscardRun(&commands.UndiscardRunInput{
		RunID: options.RunID,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to undiscard run", errors.WithSpan(span))
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
		return nil, err
	}

	run, err := s.getRun(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
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
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by TRN", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
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
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by node ID", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run with node ID %s not found", nodeID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
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
		return nil, err
	}

	filter := &db.RunFilter{
		WorkspaceAssessment: input.WorkspaceAssessment,
	}

	switch {
	case input.Workspace != nil:
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithNamespacePath(input.Workspace.FullPath))
		if err != nil {
			return nil, err
		}
		filter.WorkspaceID = &input.Workspace.Metadata.ID
	case input.Group != nil:
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
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
				return nil, errors.Wrap(rErr, "failed to get root namespaces", errors.WithSpan(span))
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
		return nil, err
	}

	result, err := s.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Filter: &db.RunFilter{
			RunIDs: idList,
		},
	})
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get runs",
			errors.WithSpan(span),
		)
	}

	for _, run := range result.Runs {
		err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
		if err != nil {
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
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, input.PlanID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by plan node ID", errors.WithSpan(span))
	}
	if run == nil {
		return nil, errors.New("plan with ID %s not found", input.PlanID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.UpdateRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithPlanID(input.PlanID))
	if err != nil {
		return nil, err
	}

	cmd := s.cmdFactory.NewUpdatePlan(input.PlanID, input.HasChanges, input.ErrorMessage)
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to update plan node", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a plan.",
		"planID", input.PlanID,
		"planStatus", cmd.Updated.Status,
	)
	return cmd.Updated, nil
}

func (s *service) ReportRunPolicyOutcomes(ctx context.Context, input *ReportRunPolicyOutcomesInput) error {
	ctx, span := tracer.Start(ctx, "svc.ReportRunPolicyOutcomes")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, input.PolicyCheckID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by policy check node ID", errors.WithSpan(span))
	}
	if run == nil || run.PolicyCheckByID(input.PolicyCheckID) == nil {
		return errors.New("policy check node with ID %s not found", input.PolicyCheckID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = caller.RequirePermission(ctx, models.UpdateRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithPolicyCheckID(input.PolicyCheckID)); err != nil {
		return err
	}

	outcomes := make([]commands.RunPolicyOutcome, 0, len(input.Outcomes))
	for _, o := range input.Outcomes {
		outcomes = append(outcomes, commands.RunPolicyOutcome{
			PolicyID: o.PolicyID,
			Messages: o.Messages,
			Passed:   o.Passed,
		})
	}

	cmd := s.cmdFactory.NewReportRunPolicyOutcomes(input.PolicyCheckID, outcomes)
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return errors.Wrap(err, "failed to report stage outcomes", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Reported policy check outcomes.",
		"runID", run.Metadata.ID,
		"runTRN", run.Metadata.TRN,
		"policyCheckID", input.PolicyCheckID,
		"outcomeCount", len(input.Outcomes),
	)
	return nil
}

func (s *service) DownloadPlan(ctx context.Context, planID string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "svc.DownloadPlan")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by plan ID", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetPlanCache(ctx, run)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get plan cache from artifact store",
			errors.WithSpan(span),
		)
	}

	return result, nil
}

func (s *service) DownloadPlanJSON(ctx context.Context, planID string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "svc.DownloadPlanJSON")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by plan ID", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		return nil, err
	}

	result, err := s.artifactStore.GetPlanJSON(ctx, run)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get plan JSON from artifact store", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]runvariables.Variable, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunVariables")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByID(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to get run",
			errors.WithSpan(span),
		)
	}

	if run == nil {
		return nil, errors.New("run with ID %s not found", runID, errors.WithErrorCode(errors.ENotFound))
	}

	// Sensitive variable values are only included if the caller has ViewSensitiveVariableValuePermission
	// on the workspace. Non-sensitive values are visible to any caller that can view variables.
	hasPermissionToViewSensitiveValues := false
	if err = caller.RequirePermission(ctx, models.ViewSensitiveVariableValuePermission, auth.WithWorkspaceID(run.WorkspaceID)); err == nil {
		hasPermissionToViewSensitiveValues = true
	} else if err = caller.RequirePermission(ctx, models.ViewVariablePermission, auth.WithWorkspaceID(run.WorkspaceID)); err != nil {
		return nil, err
	}

	if !hasPermissionToViewSensitiveValues && includeSensitiveValues {
		return nil, errors.New("caller does not have permission to view sensitive variable values", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	variables, err := s.variablesBuilder.Get(ctx, run, includeSensitiveValues)
	if err != nil {
		return nil, err
	}

	if !includeSensitiveValues {
		for i := range variables {
			if variables[i].Sensitive {
				variables[i].Value = nil
			}
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
		return err
	}

	run, err := s.dbClient.Runs.GetRunByID(ctx, input.RunID)
	if err != nil {
		return errors.Wrap(err, "failed to get run", errors.WithSpan(span))
	}

	if run == nil {
		return errors.New("run with ID %s not found", input.RunID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Since variables should only be updated during the plan operation, we're requiring that permission here.
	if err = caller.RequirePermission(ctx, models.UpdateRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithPlanID(run.Plan.GetID())); err != nil {
		return err
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

	retainFn, _, err := s.artifactStore.UploadRunVariables(ctx, run, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "failed to upload run variables", errors.WithSpan(span))
	}

	if err := retainFn(ctx, run.Metadata.ID); err != nil {
		return errors.Wrap(err, "failed to link run variables object store ref", errors.WithSpan(span))
	}

	return nil
}

func (s *service) UploadPlanBinary(ctx context.Context, planID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadPlanBinary")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by plan ID", errors.WithSpan(span))
	}

	if run == nil {
		return errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.UpdateRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithPlanID(planID))
	if err != nil {
		return err
	}

	retainFn, cacheKey, err := s.artifactStore.UploadPlanCache(ctx, run, reader)
	if err != nil {
		return errors.Wrap(err, "failed to upload plan cache", errors.WithSpan(span))
	}

	txCtx, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txCtx); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback UploadPlanBinary tx: %v", txErr)
		}
	}()

	run.Plan.CacheObjectStoreKey = &cacheKey
	if _, err = s.dbClient.Runs.UpdateRun(txCtx, run, run.Plan.GetID()); err != nil {
		return errors.Wrap(err, "failed to update run", errors.WithSpan(span))
	}

	if err = retainFn(txCtx, run.Metadata.ID); err != nil {
		return errors.Wrap(err, "failed to link plan cache object store ref", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txCtx); err != nil {
		return errors.Wrap(err, "failed to commit transaction", errors.WithSpan(span))
	}

	return nil
}

func (s *service) ProcessPlanData(ctx context.Context, planID string, tfPlan *tfjson.Plan, tfProviderSchemas *tfjson.ProviderSchemas) error {
	ctx, span := tracer.Start(ctx, "svc.ProcessPlanData")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by plan node ID", errors.WithSpan(span))
	}
	if run == nil {
		return errors.New("plan with ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.UpdateRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithPlanID(planID))
	if err != nil {
		return err
	}

	cmd := s.cmdFactory.NewUpdatePlanSummary(&commands.UpdatePlanSummaryInput{
		PlanID:            planID,
		TFPlan:            tfPlan,
		TFProviderSchemas: tfProviderSchemas,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return errors.Wrap(err, "failed to update plan summary", errors.WithSpan(span))
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
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, planID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by plan ID", errors.WithSpan(span))
	}

	if run == nil {
		return nil, errors.New("run with plan ID %s not found", planID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID))
	if err != nil {
		return nil, err
	}

	if run.Plan.DiffObjectStoreKey == nil {
		return nil, errors.New("plan diff not available yet", errors.WithErrorCode(errors.ENotFound))
	}

	reader, err := s.artifactStore.GetPlanDiff(ctx, run)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"failed to get plan diff from artifact store",
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
		return nil, err
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
		return nil, err
	}

	if run.Plan.JSONObjectStoreKey == nil {
		return nil, errors.New("plan JSON not available yet", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
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

// GetPolicyCheckPolicyMessages returns the violation messages one policy of a policy check reported.
// The full list lives in object storage, so this is a read per policy — a view showing many checks
// should use the check's MessagesSummary instead.
func (s *service) GetPolicyCheckPolicyMessages(ctx context.Context, policyCheckID string, policyID string) ([]string, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPolicyCheckPolicyMessages")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, policyCheckID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by policy check node ID", errors.WithSpan(span))
	}
	if run == nil {
		return nil, errors.New("policy check with ID %s not found", policyCheckID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, models.ViewRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithWorkspaceID(run.WorkspaceID)); err != nil {
		return nil, err
	}

	check := run.PolicyCheckByID(policyCheckID)
	if check == nil {
		return nil, errors.New("policy check with ID %s not found", policyCheckID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	var policy *models.PolicyCheckPolicy
	for _, p := range check.Policies {
		if p.ID == policyID {
			policy = p
			break
		}
	}
	if policy == nil {
		return nil, errors.New("policy check %s has no policy %s", policyCheckID, policyID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// No key means the policy reported nothing: it passed, or the check has not been evaluated. That
	// is an empty list rather than an error, and it costs no object-storage read.
	if policy.MessagesObjectStoreKey == nil {
		return []string{}, nil
	}

	reader, err := s.artifactStore.GetPolicyCheckPolicyMessages(ctx, policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policy messages from artifact store", errors.WithSpan(span))
	}
	defer reader.Close()

	messages := []string{}
	if err := json.NewDecoder(reader).Decode(&messages); err != nil {
		return nil, errors.Wrap(err, "failed to decode policy messages", errors.WithSpan(span))
	}

	return messages, nil
}

func (s *service) UpdateApply(ctx context.Context, input *UpdateApplyInput) (*models.Apply, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateApply")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	run, err := s.dbClient.Runs.GetRunByNodeID(ctx, input.ApplyID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run by apply node ID", errors.WithSpan(span))
	}
	if run == nil {
		return nil, errors.New("apply with ID %s not found", input.ApplyID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.UpdateRunPermission, auth.WithRunID(run.Metadata.ID), auth.WithApplyID(input.ApplyID))
	if err != nil {
		return nil, err
	}

	cmd := s.cmdFactory.NewUpdateApply(input.ApplyID, input.ErrorMessage)
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to update apply node", errors.WithSpan(span))
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

func (s *service) GetRunGateByID(ctx context.Context, id string) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGateByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	gate, err := s.dbClient.RunGates.GetRunGateByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate by ID", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	return gate, nil
}

func (s *service) GetRunGateByTRN(ctx context.Context, trnValue string) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGateByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	gate, err := s.dbClient.RunGates.GetRunGateByTRN(ctx, trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate by TRN", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with TRN %s not found", trnValue, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	return gate, nil
}

func (s *service) GetRunGates(ctx context.Context, input *GetRunGatesInput) (*db.RunGatesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGates")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = s.requireRunViewAccess(ctx, caller, input.RunID); err != nil {
		return nil, err
	}

	return s.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.RunGateFilter{
			RunID: &input.RunID,
		},
	})
}

// GetRunGatesByIDs returns the run gates with the given IDs. Permission checks are deduplicated per
// run since a batch usually holds several gates from the same run.
func (s *service) GetRunGatesByIDs(ctx context.Context, ids []string) ([]*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGatesByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Filter: &db.RunGateFilter{
			RunGateIDs: ids,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gates", errors.WithSpan(span))
	}

	gates := make([]*models.RunGate, len(result.RunGates))
	checkedRuns := map[string]struct{}{}
	for i := range result.RunGates {
		gate := &result.RunGates[i]
		if _, checked := checkedRuns[gate.RunID]; !checked {
			if err = caller.RequirePermission(ctx, models.ViewRunPermission,
				auth.WithRunID(gate.RunID), auth.WithWorkspaceID(gate.WorkspaceID)); err != nil {
				return nil, err
			}
			checkedRuns[gate.RunID] = struct{}{}
		}
		gates[i] = gate
	}

	return gates, nil
}

// GetRunGatesByPolicyCheckIDs returns the gates governing the given policy-check nodes, at most one
// per check. Unlike the other batched getters this authorizes after the fetch: a policy check ID is a
// run-node ID that cannot be resolved back to its run without a query, so the view check is made
// against each gate's denormalized run and workspace IDs. Nothing is returned until every distinct
// run has passed.
func (s *service) GetRunGatesByPolicyCheckIDs(ctx context.Context, policyCheckIDs []string) ([]*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGatesByPolicyCheckIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Filter: &db.RunGateFilter{
			PolicyCheckIDs: policyCheckIDs,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gates", errors.WithSpan(span))
	}

	gates := make([]*models.RunGate, len(result.RunGates))
	checkedRuns := map[string]struct{}{}
	for i := range result.RunGates {
		gate := &result.RunGates[i]
		if _, checked := checkedRuns[gate.RunID]; !checked {
			if err = caller.RequirePermission(ctx, models.ViewRunPermission,
				auth.WithRunID(gate.RunID), auth.WithWorkspaceID(gate.WorkspaceID)); err != nil {
				return nil, err
			}
			checkedRuns[gate.RunID] = struct{}{}
		}
		gates[i] = gate
	}

	return gates, nil
}

func (s *service) GetRunGateApprovalByID(ctx context.Context, id string) (*models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGateApprovalByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	approval, err := s.dbClient.RunGateApprovals.GetRunGateApprovalByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate approval by ID", errors.WithSpan(span))
	}
	if approval == nil {
		return nil, errors.New("run gate approval with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	gate, err := s.dbClient.RunGates.GetRunGateByID(ctx, approval.RunGateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate by ID", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with id %s not found", approval.RunGateID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	return approval, nil
}

func (s *service) GetRunGateApprovalByTRN(ctx context.Context, trnValue string) (*models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGateApprovalByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	approval, err := s.dbClient.RunGateApprovals.GetRunGateApprovalByTRN(ctx, trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate approval by TRN", errors.WithSpan(span))
	}
	if approval == nil {
		return nil, errors.New("run gate approval with TRN %s not found", trnValue, errors.WithErrorCode(errors.ENotFound))
	}

	gate, err := s.dbClient.RunGates.GetRunGateByID(ctx, approval.RunGateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate by ID", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with id %s not found", approval.RunGateID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	return approval, nil
}

func (s *service) GetRunGateApprovalsByGateID(ctx context.Context, gateID string) ([]models.RunGateApproval, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGateApprovalsByGateID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	gate, err := s.dbClient.RunGates.GetRunGateByID(ctx, gateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate by ID", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with id %s not found", gateID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	return s.dbClient.RunGateApprovals.GetRunGateApprovalsByGateID(ctx, gateID)
}

// GetRunGatesAwaitingDecision returns the caller's approvals inbox: pending run gates the caller is
// an eligible approver for (a directly allowed user or service account, or a member of an allowed
// team) and has not already recorded a decision on. Because one gate is created per policy check and
// the RunGateManager closes out pending gates when a run reaches a terminal status, a pending gate
// maps one-to-one to a run awaiting the caller's decision. Both user and service-account callers are
// supported.
func (s *service) GetRunGatesAwaitingDecision(ctx context.Context, input *GetRunGatesAwaitingDecisionInput) (*db.RunGatesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRunGatesAwaitingDecision")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	eligibility := &db.RunGateEligibilityFilter{}
	if err := auth.HandleCaller(
		ctx,
		func(_ context.Context, c *auth.UserCaller) error {
			// Only the user ID is needed; the db layer derives team eligibility from the user's
			// team memberships via a subquery.
			userID := c.User.Metadata.ID
			eligibility.UserID = &userID
			return nil
		},
		func(_ context.Context, c *auth.ServiceAccountCaller) error {
			saID := c.ServiceAccountID
			eligibility.ServiceAccountID = &saID
			return nil
		},
	); err != nil {
		return nil, err
	}

	// Being a valid approver on a gate does not by itself grant access to the gate's workspace, so
	// additionally scope the inbox to the caller's root member namespaces (and their descendants).
	// A nil slice means no membership filter (an admin sees all); a non-nil (possibly empty) slice
	// restricts to those namespaces.
	var rootNamespaceMemberships []models.MembershipNamespace
	if !caller.IsAdminModeActivated(ctx) {
		rootNamespaces, rErr := caller.GetRootNamespaceMemberships(ctx)
		if rErr != nil {
			return nil, errors.Wrap(rErr, "failed to get root namespaces", errors.WithSpan(span))
		}
		rootNamespaceMemberships = rootNamespaces
	}

	sort := db.RunGateSortableFieldCreatedAtDesc
	if input.Sort != nil {
		sort = *input.Sort
	}

	return s.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Sort:              &sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.RunGateFilter{
			Statuses:                 []models.RunGateStatus{models.RunGatePending},
			Eligibility:              eligibility,
			RootNamespaceMemberships: rootNamespaceMemberships,
		},
	})
}

// ApproveRunGate records an approve/reject decision on a run gate by an eligible approver.
//
// The caller must both be able to view the run and be an eligible approver. Eligibility is decided
// per approval rule: for each of the gate's rules, the caller is eligible if they are a directly
// allowed user/service account or a member of an allowed team (matched against the caller's LIVE
// team memberships), reusing the same eligible-principal logic as managed-identity rule
// enforcement. The decision covers every rule the caller is eligible for, and the caller must be
// eligible for at least one rule.
//
// A reject is advisory: it records the decision and leaves the gate pending and the run parked at
// awaiting_override, so other eligible approvers can still approve the gate. An approve records the
// decision and, once the gate reaches its required number of approvals, marks the gate approved;
// when that approval satisfies the last outstanding gate on the run, the run's blocked post-plan
// stage is auto-overridden (awaiting_override -> overridden) in the same transaction, so no
// separate override call is needed. Both decisions are dispatched as run engine commands.
func (s *service) ApproveRunGate(ctx context.Context, input *ApproveRunGateInput) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "svc.ApproveRunGate")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	var decision commands.GateDecision
	switch input.Decision {
	case models.RunGateDecisionApprove:
		decision = commands.GateDecisionApprove
	case models.RunGateDecisionReject:
		decision = commands.GateDecisionReject
	default:
		return nil, errors.New("run gate decision %s is not supported", input.Decision, errors.WithErrorCode(errors.EInvalid))
	}

	// Eligibility is not access. A policy's approvers are arbitrary users and teams — only approver
	// service accounts are checked against the owning group — so being named on a rule says nothing
	// about the caller's rights in the workspace the gate belongs to. Deciding on a run therefore
	// requires being able to see it, the same view check every run gate read in this service makes:
	// otherwise a gate the caller cannot fetch, and which never appears in their awaiting-decision
	// inbox, could still be approved by ID.
	gate, err := s.dbClient.RunGates.GetRunGateByID(ctx, input.GateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with id %s not found", input.GateID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	cmd := s.cmdFactory.NewSetRunGateDecision(&commands.SetRunGateDecisionInput{
		GateID:   input.GateID,
		Decision: decision,
		Comment:  input.Comment,
		Caller:   caller,
	})
	if err = s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to set run gate decision", errors.WithSpan(span))
	}

	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, cmd.UpdatedGate.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace for activity event", errors.WithSpan(span))
	}
	if ws != nil {
		if _, err = activity.CreateActivityEvent(ctx, s.dbClient, &activity.CreateActivityEventInput{
			NamespacePath: &ws.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetRunGate,
			TargetID:      cmd.UpdatedGate.Metadata.ID,
			Payload: &models.ActivityEventUpdateRunGatePayload{
				Type:    models.ActivityEventRunGateUpdateType(input.Decision),
				Comment: input.Comment,
			},
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
		}
	}

	s.logger.WithContextFields(ctx).Infow("Set a run gate decision.",
		"runGateID", input.GateID,
		"runGateTRN", cmd.UpdatedGate.Metadata.TRN,
		"decision", input.Decision,
		"gateStatus", cmd.UpdatedGate.Status,
	)
	return cmd.UpdatedGate, nil
}

// OverrideRunGate clears a pending run gate, and with it the policy check the gate blocks. Every
// soft-failed check has exactly one gate, so this is the only way to clear one by fiat: a gate with
// approval rules is being bypassed ahead of its approvals, and a gate with none never had approvals to
// collect. The caller must be able to view the run, and — unless admin mode is active — hold
// UpdatePolicyPermission in every group defining a soft-mandatory policy that failed on the check.
func (s *service) OverrideRunGate(ctx context.Context, gateID string, comment *string) (*models.RunGate, error) {
	ctx, span := tracer.Start(ctx, "svc.OverrideRunGate")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	gate, err := s.dbClient.RunGates.GetRunGateByID(ctx, gateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run gate", errors.WithSpan(span))
	}
	if gate == nil {
		return nil, errors.New("run gate with id %s not found", gateID, errors.WithErrorCode(errors.ENotFound))
	}

	// Checked before anything about the gate's state is resolved, so a caller who cannot see the run
	// learns nothing about it — not the gate's status, not whether its policy check is awaiting an
	// override. The override permission below is held in the group owning the policy, which is an
	// ancestor of the workspace, so it says the caller may lift that policy's requirement; it does not
	// say they may read this run. Both are required.
	if err = s.requireRunViewAccess(ctx, caller, gate.RunID); err != nil {
		return nil, err
	}

	if gate.Status != models.RunGatePending {
		return nil, errors.New("run gate is not pending a decision", errors.WithErrorCode(errors.EConflict))
	}

	run, err := s.dbClient.Runs.GetRunByID(ctx, gate.RunID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get run", errors.WithSpan(span))
	}

	check := run.PolicyCheckByID(gate.PolicyCheckID)
	if check == nil || check.Status != models.PolicyCheckSoftFailed {
		return nil, errors.New("run does not have a policy check awaiting override", errors.WithErrorCode(errors.EConflict))
	}

	if !caller.IsAdminModeActivated(ctx) {
		if err = requireGateOverridePermission(ctx, caller, check); err != nil {
			return nil, err
		}
	}

	cmd := s.cmdFactory.NewSetRunGateDecision(&commands.SetRunGateDecisionInput{
		GateID:   gateID,
		Decision: commands.GateDecisionOverride,
		Comment:  comment,
		Caller:   caller,
	})
	if err := s.cmdProcessor.ProcessCommand(ctx, cmd); err != nil {
		return nil, errors.Wrap(err, "failed to override run gate", errors.WithSpan(span))
	}

	ws, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, gate.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace for activity event", errors.WithSpan(span))
	}
	if ws != nil {
		if _, err = activity.CreateActivityEvent(ctx, s.dbClient, &activity.CreateActivityEventInput{
			NamespacePath: &ws.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetRunGate,
			TargetID:      cmd.UpdatedGate.Metadata.ID,
			Payload: &models.ActivityEventUpdateRunGatePayload{
				Type:    models.RunGateUpdateTypeOverride,
				Comment: comment,
			},
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
		}
	}

	s.logger.WithContextFields(ctx).Infow("Overrode a run gate.",
		"runGateID", gateID,
		"runGateTRN", cmd.UpdatedGate.Metadata.TRN,
	)
	return cmd.UpdatedGate, nil
}

// requireGateOverridePermission checks that the caller has UpdatePolicyPermission in every group
// defining a soft-mandatory policy that failed on the gate's check — the policies whose requirement
// the override lifts. It is keyed on the check's failed policies rather than the gate's approval
// rules: a policy that declared no approvers produces no rule, so keying on rules would wave through
// exactly the rule-less gate that has no approvers to answer to.
func requireGateOverridePermission(ctx context.Context, caller auth.Caller, check *models.PolicyCheck) error {
	checkedGroups := map[string]struct{}{}
	matched := false
	for i := range check.Policies {
		policy := check.Policies[i]
		if policy.Status != models.PolicyCheckPolicyFailed ||
			policy.EnforcementLevel != models.PolicyEnforcementSoftMandatory {
			continue
		}
		matched = true
		groupID := policy.Provenance.GroupID
		if _, checked := checkedGroups[groupID]; checked {
			continue
		}
		if err := caller.RequirePermission(ctx, models.UpdatePolicyPermission, auth.WithGroupID(groupID)); err != nil {
			return errors.New("a run gate can only be overridden by callers that have the 'UpdatePolicyPermission' in the groups where the failed soft-mandatory policies are defined", errors.WithErrorCode(errors.EForbidden))
		}
		checkedGroups[groupID] = struct{}{}
	}
	// Fail closed: a gate parks a run precisely because a soft-mandatory policy failed, so an override
	// request against a check with no such policy is an inconsistent state, not a free pass. Granting it
	// would bypass the permission check entirely, so refuse instead.
	if !matched {
		return errors.New("a run gate can only be overridden when a soft-mandatory policy has failed on its check", errors.WithErrorCode(errors.EForbidden))
	}
	return nil
}

func (s *service) requireRunViewAccess(ctx context.Context, caller auth.Caller, runID string) error {
	workspaceID, err := s.dbClient.Runs.GetWorkspaceIDForRun(ctx, runID)
	if err != nil {
		return err
	}
	return caller.RequirePermission(ctx, models.ViewRunPermission,
		auth.WithRunID(runID), auth.WithWorkspaceID(workspaceID))
}
