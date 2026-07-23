package commands

import (
	"context"
	"time"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// CreateAssessmentRunInput is the input for creating an assessment run. The remaining run
// configuration is derived from the workspace's current state in Prepare.
type CreateAssessmentRunInput struct {
	Subject                 string
	WorkspaceID             string
	LatestAssessmentVersion *int

	// SkipActivityEvent suppresses the run-creation activity event. The assessment
	// scheduler sets it so its automatic, high-frequency assessments don't flood the
	// activity feed; user-initiated assessments leave it false.
	SkipActivityEvent bool
}

// CreateAssessmentRun creates a speculative assessment run from the workspace's
// current state. It also creates or updates the workspace's assessment record; the
// upsert and the run creation happen in the command processor's single
// transaction, so they are persisted atomically.
type CreateAssessmentRun struct {
	dbClient           *db.Client
	variablesBuilder   *runvariables.Builder
	moduleResolver     registry.ModuleResolver
	createRun          createRunFunc
	uploadRunVariables uploadRunVariablesFunc

	in                *CreateAssessmentRunInput
	createInput       *corerun.CreateRunInput
	variablesRetainFn db.RetainObjectRefFunc

	// Created is populated with the persisted run once Execute succeeds.
	Created *models.Run
}

// Prepare derives the run inputs from the workspace's current state, then builds the core
// run-creation input. It runs before the transaction is opened.
func (c *CreateAssessmentRun) Prepare(ctx context.Context) error {
	source, err := corerun.FindLatestApplyRunForWorkspace(ctx, c.dbClient, c.in.WorkspaceID)
	if err != nil {
		return err
	}
	if source.IsDestroy {
		return errors.New("cannot create assessment run because the latest run is a destroy run", errors.WithErrorCode(errors.EConflict))
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
		IsAssessmentRun:         true,
		Speculative:             ptr.Bool(true),
		RefreshOnly:             true,
		Refresh:                 true,
		SkipActivityEvent:       c.in.SkipActivityEvent,
		VariablesObjectStoreKey: variablesObjectKey,
	}
	return nil
}

// Execute upserts the assessment record and then creates the run via the core pure function and
// enqueues it, all within the same transaction.
func (c *CreateAssessmentRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	if err := c.upsertAssessment(ctx); err != nil {
		return err
	}

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

func (c *CreateAssessmentRun) upsertAssessment(ctx context.Context) error {
	workspaceID := c.in.WorkspaceID

	assessment, err := c.dbClient.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace assessment for workspace with ID %s", workspaceID)
	}

	if assessment == nil {
		if c.in.LatestAssessmentVersion != nil {
			return errors.New(
				"cannot create assessment run because latest assessment version is not nil",
				errors.WithErrorCode(errors.EConflict),
			)
		}
		if _, err = c.dbClient.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
			WorkspaceID:        workspaceID,
			StartedAtTimestamp: time.Now().UTC(),
		}); err != nil {
			return errors.Wrap(err, "failed to create assessment for workspace %q", workspaceID)
		}
		return nil
	}

	if c.in.LatestAssessmentVersion != nil && *c.in.LatestAssessmentVersion != assessment.Metadata.Version {
		return errors.New(
			"cannot create assessment run because latest assessment version does not match",
			errors.WithErrorCode(errors.EConflict),
		)
	}

	// An in-progress assessment blocks a new run, unless it has gone stale (its run was
	// abandoned before completing), in which case we restart it below.
	if assessment.CompletedAtTimestamp == nil && !assessment.IsStaleInProgress() {
		return errors.New(
			"cannot create assessment run because an assessment is already in progress",
			errors.WithErrorCode(errors.EConflict),
		)
	}

	assessment.StartedAtTimestamp = time.Now().UTC()
	assessment.CompletedAtTimestamp = nil
	if _, err = c.dbClient.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, assessment); err != nil {
		return errors.Wrap(err, "failed to update workspace assessment with ID %q", assessment.Metadata.ID)
	}
	return nil
}
