package commands

import (
	"context"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// UpdatePlan updates a plan node's status. It accepts the plan node ID and
// resolves the owning run in Prepare.
type UpdatePlan struct {
	dbClient     *db.Client
	PlanID       string
	HasChanges   bool
	ErrorMessage *string

	// Populated by Prepare.
	runID            string
	sanitizedMessage *string

	// Updated is populated with the plan once Execute succeeds.
	Updated *models.Plan
}

// Prepare resolves the run owning the plan node and sanitizes the error message.
// It runs before the transaction is opened.
func (c *UpdatePlan) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByNodeID(ctx, c.PlanID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by plan ID")
	}
	if run == nil {
		return errors.New("plan with id %s not found", c.PlanID, errors.WithErrorCode(errors.ENotFound))
	}

	c.runID = run.Metadata.ID
	if c.ErrorMessage != nil {
		c.sanitizedMessage = corerun.SanitizeAndTruncateErrorMessage(*c.ErrorMessage)
	}
	return nil
}

// Execute executes the update plan command.
func (c *UpdatePlan) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.runID)
	if err != nil {
		return err
	}

	planNode := &run.Plan

	if c.sanitizedMessage != nil {
		planNode.ErrorMessage = c.sanitizedMessage
	}
	planNode.HasChanges = c.HasChanges

	c.Updated = planNode
	return nil
}
