package commands

import (
	"context"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// UpdateApply updates an apply node's status. It accepts the apply node ID and
// resolves the owning run in Prepare.
type UpdateApply struct {
	dbClient     *db.Client
	ApplyID      string
	ErrorMessage *string

	// Populated by Prepare.
	runID            string
	sanitizedMessage *string

	// Updated is populated with the apply once Execute succeeds.
	Updated *models.Apply
}

// Prepare resolves the run owning the apply node and sanitizes the error message.
// It runs before the transaction is opened.
func (c *UpdateApply) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByNodeID(ctx, c.ApplyID)
	if err != nil {
		return errors.Wrap(err, "failed to get run by apply ID")
	}
	if run == nil {
		return errors.New("apply with id %s not found", c.ApplyID, errors.WithErrorCode(errors.ENotFound))
	}

	c.runID = run.Metadata.ID
	if c.ErrorMessage != nil {
		c.sanitizedMessage = corerun.SanitizeAndTruncateErrorMessage(*c.ErrorMessage)
	}
	return nil
}

// Execute executes the update apply command.
func (c *UpdateApply) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.runID)
	if err != nil {
		return err
	}

	applyNode := run.Apply
	if applyNode == nil {
		return errors.New("run does not have an apply node", errors.WithErrorCode(errors.EInvalid))
	}

	if c.sanitizedMessage != nil {
		applyNode.ErrorMessage = c.sanitizedMessage
	}

	c.Updated = applyNode
	return nil
}
