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

// NewRunInput carries the run configuration used to create a run. Variables are
// resolved by each command (merged from the workspace for user runs, or read from a
// source run for derived runs), and the command passes the final variables to
// corerun.Create, so this struct does not say how the variables were produced.
type NewRunInput struct {
	Subject                string
	WorkspaceID            string
	TerraformVersion       string
	ConfigurationVersionID *string
	ModuleSource           *string
	ModuleVersion          *string
	Comment                *string
	Speculative            *bool
	AutoApply              bool
	TargetAddresses        []string
	IsDestroy              bool
	// Refresh is optional; nil means "not explicitly set" and resolves to true
	// (Terraform's default) when the core run-creation input is built in Prepare.
	Refresh                  *bool
	RefreshOnly              bool
	IsAssessmentRun          bool
	IncludeModulePrereleases bool

	// Variables are the caller-provided run variables (used by user-initiated
	// runs, which merge them with the workspace's inherited variables).
	Variables []runvariables.Variable
}

// CreateRun is the user-initiated run creation command. It merges the workspace's
// inherited variables with the caller's variables in Prepare, then creates the run
// via createRun and enqueues it in Execute.
type CreateRun struct {
	dbClient         *db.Client
	variablesBuilder *runvariables.Builder
	moduleResolver   registry.ModuleResolver
	createRun        createRunFunc

	in          *NewRunInput
	createInput *corerun.CreateRunInput

	// Created is populated with the persisted run once Execute succeeds.
	Created *models.Run
}

// Prepare merges the workspace's inherited variables with the caller's variables,
// then builds the core run-creation input. It runs before the transaction is opened.
func (c *CreateRun) Prepare(ctx context.Context) error {
	variables, err := c.variablesBuilder.Build(ctx, c.in.WorkspaceID, c.in.Variables)
	if err != nil {
		return errors.Wrap(err, "failed to build run variables")
	}

	resolvedModule, err := corerun.ResolveModule(ctx, c.dbClient, c.moduleResolver,
		c.in.WorkspaceID, c.in.ModuleSource, c.in.ModuleVersion, c.in.IncludeModulePrereleases, variables)
	if err != nil {
		return err
	}

	// Refresh defaults to true (Terraform's default) unless the caller explicitly set it.
	refresh := true
	if c.in.Refresh != nil {
		refresh = *c.in.Refresh
	}

	c.createInput = &corerun.CreateRunInput{
		Subject:                c.in.Subject,
		WorkspaceID:            c.in.WorkspaceID,
		TerraformVersion:       c.in.TerraformVersion,
		ConfigurationVersionID: c.in.ConfigurationVersionID,
		ModuleSource:           c.in.ModuleSource,
		ModuleVersion:          resolvedModule.Version,
		ModuleDigest:           resolvedModule.Digest,
		ModuleRegistrySource:   resolvedModule.Source,
		Comment:                c.in.Comment,
		Speculative:            c.in.Speculative,
		AutoApply:              c.in.AutoApply,
		TargetAddresses:        c.in.TargetAddresses,
		IsDestroy:              c.in.IsDestroy,
		Refresh:                refresh,
		RefreshOnly:            c.in.RefreshOnly,
		IsAssessmentRun:        c.in.IsAssessmentRun,
		Variables:              variables,
	}
	return nil
}

// Execute creates the run via the core pure function, then enqueues it.
func (c *CreateRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	created, err := c.createRun(ctx, c.createInput)
	if err != nil {
		return err
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
