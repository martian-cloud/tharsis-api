package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// StartApplyInput carries everything StartApply needs across its Prepare and
// Execute phases.
type StartApplyInput struct {
	RunID       string
	TriggeredBy models.Actor
	Comment     string
}

// StartApply transitions a planned run to apply-pending. It verifies the
// workspace's managed-identity rules permit the apply in Prepare (before the
// transaction is opened) and transitions the apply node in Execute.
type StartApply struct {
	dbClient       *db.Client
	moduleResolver registry.ModuleResolver
	ruleEnforcer   rules.RuleEnforcer
	in             *StartApplyInput

	// Populated by Prepare: the identities that passed the apply-stage rules, recorded on the apply
	// node so the apply job can only use those.
	managedIdentityIDs []string

	// Updated is populated with the run once Execute succeeds.
	Updated *models.Run
}

// Prepare verifies the workspace's managed-identity rules allow the apply. It
// performs registry/network I/O and reads, so it runs before the transaction is
// opened.
func (c *StartApply) Prepare(ctx context.Context) error {
	dbClient := c.dbClient

	run, err := dbClient.Runs.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return errors.Wrap(err, "failed to get run")
	}
	if run == nil {
		return errors.New("run with ID %s not found", c.in.RunID, errors.WithErrorCode(errors.ENotFound))
	}

	c.managedIdentityIDs, err = enforceApplyRules(ctx, dbClient, c.moduleResolver, c.ruleEnforcer, run)
	return err
}

// enforceApplyRules verifies the managed-identity apply-stage rules for every identity assigned to
// the run's workspace, against the caller in ctx, and returns the IDs of the identities it checked
// (never nil) for the caller to record on the apply node. It performs registry/network I/O, so
// callers run it from Prepare, before the transaction is opened.
func enforceApplyRules(
	ctx context.Context,
	dbClient *db.Client,
	moduleResolver registry.ModuleResolver,
	ruleEnforcer rules.RuleEnforcer,
	run *models.Run,
) ([]string, error) {
	managedIdentities, err := dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, run.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get managed identities assigned to workspace")
	}
	if len(managedIdentities) == 0 {
		return []string{}, nil
	}

	ws, err := dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace by ID")
	}
	if ws == nil {
		return nil, errors.New("failed to get workspace ID %s associated with run ID %s", run.WorkspaceID, run.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	var currentStateVersionID *string
	if ws.CurrentStateVersionID != "" {
		currentStateVersionID = &ws.CurrentStateVersionID
	}

	var moduleSource registry.ModuleRegistrySource
	if run.ModuleSource != nil && run.ModuleDigest != nil {
		moduleSource, err = moduleResolver.ParseModuleRegistrySource(
			ctx, *run.ModuleSource, runvariables.ModuleRegistryToken(nil), corerun.GetFederatedRegistry(dbClient, ws))
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse module registry source")
		}
	}

	runDetails := &rules.RunDetails{
		RunStage:              models.JobApplyType,
		ModuleDigest:          run.ModuleDigest,
		CurrentStateVersionID: currentStateVersionID,
		ModuleSource:          moduleSource,
		ModuleSemanticVersion: run.ModuleVersion,
	}
	return ruleEnforcer.EnforceRules(ctx, managedIdentities, runDetails)
}

// Execute executes the start apply command.
func (c *StartApply) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return err
	}

	applyNode := run.Apply
	if applyNode == nil {
		return errors.New("run does not have an apply node", errors.WithErrorCode(errors.EConflict))
	}
	if applyNode.Status != models.ApplyCreated {
		return errors.New("the apply phase has already been started for this run", errors.WithErrorCode(errors.EConflict))
	}

	applyNode.TriggeredBy = new(c.in.TriggeredBy)
	applyNode.Comment = c.in.Comment
	applyNode.ManagedIdentityIDs = c.managedIdentityIDs

	switch run.Status {
	case models.RunPostPlanRunning, models.RunPostPlanAwaitingDecision, models.RunPostPlanCompleted:
		// To support backward compatibility with older clients which do not support task stages, we'll set the
		// run to auto apply here which will result in the apply being automatically started after the task stages have completed
		run.AutoApply = true
	case models.RunPlanned:
		// The human approval this run was parked at planned for has now been given, so advance it. The run
		// node decides what that means: it readies a pre-apply policy stage if the run has one, and otherwise
		// the apply itself — so a pre-apply gate cannot be bypassed here. Either way the admission transformer
		// then attempts to acquire the workspace; if it is busy the readied node stays pending and the work
		// item consumer admits it when the workspace frees up.
		changes, err := statemachine.AdvanceRun(run)
		if err != nil {
			return errors.Wrap(err, "failed to advance run to its apply phase")
		}
		if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
			return err
		}
	default:
		return errors.New("run must be in planned state to start apply", errors.WithErrorCode(errors.EConflict))
	}

	c.Updated = run
	return nil
}
