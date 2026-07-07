package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// SetRunAutoApplyInput carries everything SetRunAutoApply needs across its Prepare and
// Execute phases.
type SetRunAutoApplyInput struct {
	RunID     string
	AutoApply bool
}

// SetRunAutoApply changes a run's auto-apply setting after creation, recording a run-update
// activity event. Auto-apply is only consumed when the plan finishes (the state machine
// decides whether to auto-advance the apply), so the change is only permitted before the plan
// completes — while the run is still pending/queuing/plan_queued/planning with its apply node
// not yet started. After that the setting no longer has any effect.
type SetRunAutoApply struct {
	dbClient *db.Client
	in       *SetRunAutoApplyInput

	// Populated by Prepare.
	namespacePath string

	// Updated is populated with the run once Execute succeeds.
	Updated *models.Run
}

// Prepare resolves the workspace namespace path used for the activity event. It runs before
// the transaction is opened.
func (c *SetRunAutoApply) Prepare(ctx context.Context) error {
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

// Execute validates the run can still have its auto-apply setting changed, updates it, and
// records the activity event. No state machine transition occurs — this is a pure run-field
// update, persisted via the run's shallow-compare diff.
func (c *SetRunAutoApply) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.in.RunID)
	if err != nil {
		return err
	}

	if run.Apply == nil {
		return errors.New("speculative runs do not have an apply phase, so auto-apply cannot be set", errors.WithErrorCode(errors.EInvalid))
	}
	if run.Apply.Status != models.ApplyCreated {
		return errors.New("the apply phase has already started, so auto-apply can no longer be changed", errors.WithErrorCode(errors.EConflict))
	}
	switch run.Status {
	case models.RunPending, models.RunQueuing, models.RunPlanQueued, models.RunPlanning:
		// Auto-apply still controls what happens when the plan finishes.
	default:
		return errors.New("auto-apply can only be changed before the plan completes", errors.WithErrorCode(errors.EConflict))
	}

	// Nothing to do if the setting already matches the request.
	if run.AutoApply == c.in.AutoApply {
		c.Updated = run
		return nil
	}

	run.AutoApply = c.in.AutoApply

	updateType := models.RunUpdateTypeEnableAutoApply
	if !c.in.AutoApply {
		updateType = models.RunUpdateTypeDisableAutoApply
	}
	if _, err := activity.CreateActivityEvent(ctx, c.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &c.namespacePath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetRun,
		TargetID:      run.Metadata.ID,
		Payload: &models.ActivityEventUpdateRunPayload{
			Type: string(updateType),
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create activity event")
	}

	c.Updated = run
	return nil
}
