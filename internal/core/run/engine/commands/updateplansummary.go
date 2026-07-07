package commands

import (
	"bytes"
	"context"
	"encoding/json"

	tfjson "github.com/hashicorp/terraform-json"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// UpdatePlanSummaryInput carries everything UpdatePlanSummary needs across its
// Prepare and Execute phases.
type UpdatePlanSummaryInput struct {
	PlanID            string
	TFPlan            *tfjson.Plan
	TFProviderSchemas *tfjson.ProviderSchemas
}

// UpdatePlanSummary parses the Terraform plan into a diff/summary in Prepare
// (before the transaction is opened), then in Execute records the summary on the
// plan node and uploads the plan diff and JSON to the artifact store. The uploads
// are idempotent by storage key, so OLE retries are safe.
type UpdatePlanSummary struct {
	dbClient      *db.Client
	planParser    plan.Parser
	artifactStore workspace.ArtifactStore
	in            *UpdatePlanSummaryInput

	// Populated by Prepare.
	runID    string
	summary  models.PlanSummary
	planDiff []byte
	planJSON []byte
}

// Prepare resolves the run, parses the plan diff, and computes the summary. It
// runs before the transaction is opened.
func (c *UpdatePlanSummary) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByNodeID(ctx, c.in.PlanID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by plan ID")
	}
	if run == nil {
		return errors.New("run with plan ID %s not found", c.in.PlanID, errors.WithErrorCode(errors.ENotFound))
	}

	diff, err := c.planParser.Parse(c.in.TFPlan, c.in.TFProviderSchemas)
	if err != nil {
		return errors.Wrap(err, "failed to create plan diff")
	}

	planDiff, err := json.Marshal(diff)
	if err != nil {
		return errors.Wrap(err, "failed to marshal plan diff")
	}

	// Calculate summary
	var summary models.PlanSummary
	for _, change := range diff.Resources {
		switch change.Action {
		case action.Create:
			summary.ResourceAdditions++
		case action.Update:
			summary.ResourceChanges++
		case action.Delete:
			summary.ResourceDestructions++
		case action.CreateThenDelete, action.DeleteThenCreate:
			summary.ResourceAdditions++
			summary.ResourceDestructions++
		}
		if change.Imported {
			summary.ResourceImports++
		}
		if change.Drifted {
			summary.ResourceDrift++
		}
	}
	for _, change := range diff.Outputs {
		switch change.Action {
		case action.Create:
			summary.OutputAdditions++
		case action.Update:
			summary.OutputChanges++
		case action.Delete:
			summary.OutputDestructions++
		}
	}

	planJSON, err := json.Marshal(c.in.TFPlan)
	if err != nil {
		return errors.Wrap(err, "failed to marshal plan json")
	}

	c.runID = run.Metadata.ID
	c.summary = summary
	c.planDiff = planDiff
	c.planJSON = planJSON
	return nil
}

// Execute records the plan summary on the plan node and uploads the plan diff
// and JSON to the artifact store.
func (c *UpdatePlanSummary) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.runID)
	if err != nil {
		return err
	}

	// HasChanges is intentionally not set here: the run's authoritative has-changes
	// result comes from UpdatePlan (Terraform's detailed-exitcode), which the executor
	// calls right after this command. Computing it from the summary here risked
	// disagreeing with Terraform (e.g. import/drift-only plans).
	planNode := &run.Plan
	planNode.Summary = c.summary
	planNode.DiffSize = len(c.planDiff)

	if err := c.artifactStore.UploadPlanDiff(ctx, run, bytes.NewReader(c.planDiff)); err != nil {
		return errors.Wrap(err, "failed to write plan diff to object storage")
	}

	if err := c.artifactStore.UploadPlanJSON(ctx, run, bytes.NewReader(c.planJSON)); err != nil {
		return errors.Wrap(err, "failed to write plan json to object storage")
	}

	return nil
}
