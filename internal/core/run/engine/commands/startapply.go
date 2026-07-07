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
	TriggeredBy string
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

	managedIdentities, err := dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get managed identities assigned to workspace")
	}
	if len(managedIdentities) == 0 {
		return nil
	}

	ws, err := dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace by ID")
	}
	if ws == nil {
		return errors.New("failed to get workspace ID %s associated with run ID %s", run.WorkspaceID, run.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	var currentStateVersionID *string
	if ws.CurrentStateVersionID != "" {
		currentStateVersionID = &ws.CurrentStateVersionID
	}

	var moduleSource registry.ModuleRegistrySource
	if run.ModuleSource != nil && run.ModuleDigest != nil {
		moduleSource, err = c.moduleResolver.ParseModuleRegistrySource(
			ctx, *run.ModuleSource, runvariables.ModuleRegistryToken(nil), corerun.GetFederatedRegistry(dbClient, ws))
		if err != nil {
			return errors.Wrap(err, "failed to parse module registry source")
		}
	}

	runDetails := &rules.RunDetails{
		RunStage:              models.JobApplyType,
		ModuleDigest:          run.ModuleDigest,
		CurrentStateVersionID: currentStateVersionID,
		ModuleSource:          moduleSource,
		ModuleSemanticVersion: run.ModuleVersion,
	}
	for _, mi := range managedIdentities {
		miCopy := mi
		if err := c.ruleEnforcer.EnforceRules(ctx, &miCopy, runDetails); err != nil {
			return err
		}
	}

	return nil
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

	if run.Status != models.RunPlanned {
		return errors.New("run must be in planned state to start apply", errors.WithErrorCode(errors.EConflict))
	}

	applyNode.TriggeredBy = c.in.TriggeredBy
	applyNode.Comment = c.in.Comment

	// Mark the apply pending (approved, waiting). The admission transformer
	// attempts to queue it; if the workspace is busy it stays pending and the
	// work item consumer admits it when the workspace frees up.
	changes, err := statemachine.SetApplyStatus(run, models.ApplyPending)
	if err != nil {
		return errors.Wrap(err, "failed to transition apply node to pending")
	}
	if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
		return err
	}

	c.Updated = run
	return nil
}
