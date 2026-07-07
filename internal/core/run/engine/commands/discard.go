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

// DiscardRunInput carries everything DiscardRun needs across its Prepare and
// Execute phases.
type DiscardRunInput struct {
	RunID string
	// SkipActivityEvent suppresses the discard activity event. System-initiated discards
	// (e.g. auto-discarding stale planned runs) set this, since there is no user actor to
	// attribute and no activity event is wanted.
	SkipActivityEvent bool
}

// DiscardRun discards a planned run, moving it to the terminal discarded status (and,
// unless suppressed, recording a discard activity event) in a single transaction. A run
// can only be discarded while it is in the planned state (enforced by Execute).
type DiscardRun struct {
	dbClient *db.Client
	in       *DiscardRunInput

	// Populated by Prepare.
	namespacePath string

	// Updated is populated with the run once Execute succeeds.
	Updated *models.Run
}

// Prepare resolves the workspace namespace path used for the discard activity event.
// It runs before the transaction is opened. When the activity event is suppressed there
// is nothing to resolve, so it's a no-op (Execute validates the run exists).
func (c *DiscardRun) Prepare(ctx context.Context) error {
	if c.in.SkipActivityEvent {
		return nil
	}

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

// Execute discards the run, validating it is in the planned state, and records the
// activity event. The state machine performs the transition to discarded (marking the
// never-started apply node skipped).
func (c *DiscardRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return err
	}

	if run.Status != models.RunPlanned {
		return errors.New("run can only be discarded when it is in the planned state", errors.WithErrorCode(errors.EConflict))
	}

	changes, err := statemachine.SetRunStatus(run, models.RunDiscarded)
	if err != nil {
		return err
	}

	if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
		return err
	}

	if !c.in.SkipActivityEvent {
		if _, err := activity.CreateActivityEvent(ctx, c.dbClient, &activity.CreateActivityEventInput{
			NamespacePath: &c.namespacePath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetRun,
			TargetID:      run.Metadata.ID,
			Payload: &models.ActivityEventUpdateRunPayload{
				Type: string(models.RunUpdateTypeDiscard),
			},
		}); err != nil {
			return errors.Wrap(err, "failed to create activity event")
		}
	}

	c.Updated = run
	return nil
}
