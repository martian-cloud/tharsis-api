package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// CreateReconcileRunInput is the input for creating a reconcile run. The remaining run
// configuration is derived from the workspace's current state in Prepare.
type CreateReconcileRunInput struct {
	Subject     string
	WorkspaceID string
}

// CreateReconcileRun creates a reconcile run from the workspace's current state,
// re-running the configuration that produced it.
type CreateReconcileRun struct {
	dbClient           *db.Client
	variablesBuilder   *runvariables.Builder
	moduleResolver     registry.ModuleResolver
	createRun          createRunFunc
	uploadRunVariables uploadRunVariablesFunc

	in                *CreateReconcileRunInput
	createInput       *corerun.CreateRunInput
	variablesRetainFn db.RetainObjectRefFunc

	// Created is populated with the persisted run once Execute succeeds.
	Created *models.Run
}

// Prepare derives the run inputs from the workspace's current state, then builds the core
// run-creation input. It runs before the transaction is opened.
func (c *CreateReconcileRun) Prepare(ctx context.Context) error {
	source, err := corerun.FindLatestApplyRunForWorkspace(ctx, c.dbClient, c.in.WorkspaceID)
	if err != nil {
		return err
	}

	// The source run's variables are reused as-is, with their namespace paths cleared so
	// they are stored against the new run rather than the namespace they were inherited from.
	variables, err := c.variablesBuilder.Get(ctx, source, true)
	if err != nil {
		return errors.Wrap(err, "failed to get run variables for run with ID %s", source.Metadata.ID)
	}
	for i := range variables {
		variables[i].NamespacePath = nil
	}

	resolvedModule, err := corerun.ResolveModule(ctx, c.dbClient, c.moduleResolver,
		c.in.WorkspaceID, source.ModuleSource, source.ModuleVersion, false, variables)
	if err != nil {
		return err
	}

	variablesRetainFn, variablesObjectKey, err := c.uploadRunVariables(ctx, c.in.WorkspaceID, variables)
	if err != nil {
		return errors.Wrap(err, "failed to upload run variables")
	}
	c.variablesRetainFn = variablesRetainFn

	c.createInput = &corerun.CreateRunInput{
		Subject:                 c.in.Subject,
		WorkspaceID:             c.in.WorkspaceID,
		ConfigurationVersionID:  source.ConfigurationVersionID,
		ModuleSource:            source.ModuleSource,
		ModuleVersion:           resolvedModule.Version,
		ModuleDigest:            resolvedModule.Digest,
		ModuleRegistrySource:    resolvedModule.Source,
		Refresh:                 true,
		VariablesObjectStoreKey: variablesObjectKey,
	}
	return nil
}

// Execute creates the run via the core pure function, then enqueues it.
func (c *CreateReconcileRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	created, err := c.createRun(ctx, c.createInput)
	if err != nil {
		return err
	}

	if err = c.variablesRetainFn(ctx, created.Metadata.ID); err != nil {
		return errors.Wrap(err, "failed to link run variables object store ref")
	}

	input.RunStore.AddRun(created)
	changes, err := statemachine.SetRunStatus(created, models.RunQueuing)
	if err != nil {
		return errors.Wrap(err, "failed to initialize run state")
	}
	if err = input.RunStore.AddRunChanges(created, changes...); err != nil {
		return err
	}

	c.Created = created
	return nil
}
