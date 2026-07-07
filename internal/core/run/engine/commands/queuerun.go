package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// QueueRun re-attempts admission for a run's pending plan/apply nodes. The
// work item consumer calls it to advance a run that was parked because its workspace was
// busy. Unlike a fresh pending transition (which the admission transformer picks
// up automatically), a parked node has no new transition to react to, so its
// queue attempt is driven explicitly here.
type QueueRun struct {
	admitter *admission.Admitter
	RunID    string
}

// Execute executes the queue run command.
func (c *QueueRun) Execute(ctx context.Context, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, c.RunID)
	if err != nil {
		return err
	}

	// Re-attempt admission for a plan waiting on the workspace.
	if run.Plan.Status == models.PlanPending {
		queued, changes, err := c.admitter.TryQueuePlan(ctx, run)
		if err != nil {
			return err
		}
		if queued {
			if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
				return err
			}
		}
	}

	// Re-attempt admission for an approved apply waiting on the workspace.
	if run.Apply != nil && run.Apply.Status == models.ApplyPending {
		queued, changes, err := c.admitter.TryQueueApply(ctx, run)
		if err != nil {
			return err
		}
		if queued {
			if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
				return err
			}
		}
	}

	return nil
}
