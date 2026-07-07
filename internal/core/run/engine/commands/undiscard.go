package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// UndiscardRunInput carries everything UndiscardRun needs across its Prepare and Execute
// phases.
type UndiscardRunInput struct {
	RunID string
}

// UndiscardRun reverses a discard, moving a run from the terminal discarded status back to
// planned (and recording an undiscard activity event) in a single transaction. A run can only
// be undiscarded while it is in the discarded state (enforced by Execute).
type UndiscardRun struct {
	dbClient *db.Client
	in       *UndiscardRunInput

	// Populated by Prepare.
	namespacePath string

	// Updated is populated with the run once Execute succeeds.
	Updated *models.Run
}

// Prepare resolves the workspace namespace path used for the undiscard activity event. It
// runs before the transaction is opened.
func (c *UndiscardRun) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return errors.Wrap(err, "failed to get run")
	}
	if run == nil {
		return errors.New("run with ID %s not found", c.in.RunID, errors.WithErrorCode(errors.ENotFound))
	}

	ws, err := c.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace by ID")
	}
	if ws == nil {
		return errors.New("failed to get workspace ID %s associated with run ID %s", run.WorkspaceID, run.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
	}

	c.namespacePath = ws.FullPath
	return nil
}

// Execute undiscards the run, validating it is in the discarded state, and records the
// activity event. The state machine performs the transition back to planned (restoring the
// skipped apply node to created).
func (c *UndiscardRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return err
	}

	if run.Status != models.RunDiscarded {
		return errors.New("run can only be undiscarded when it is in the discarded state", errors.WithErrorCode(errors.EConflict))
	}

	changes, err := statemachine.SetRunStatus(run, models.RunPlanned)
	if err != nil {
		return err
	}

	if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
		return err
	}

	if _, err := activity.CreateActivityEvent(ctx, c.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &c.namespacePath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetRun,
		TargetID:      run.Metadata.ID,
		Payload: &models.ActivityEventUpdateRunPayload{
			Type: string(models.RunUpdateTypeUndiscard),
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create activity event")
	}

	c.Updated = run
	return nil
}
