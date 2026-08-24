package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
)

// QueueRun re-attempts a run's next workspace-gated transition (starting its pre-plan stage,
// queuing its plan, or queuing its apply — whichever its state calls for). The work item
// consumer calls it to advance a run that was parked because its workspace was busy. Unlike a
// fresh pending transition (which the admission transformer picks up automatically), a parked
// node has no new transition to react to, so its queue attempt is driven explicitly here.
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

	// Re-attempt the run's next workspace-gated transition (start pre-plan, queue plan, or queue
	// apply — the admitter determines which from the run's state). Errors propagate: retrying this
	// cheap command is acceptable.
	started, changes, err := c.admitter.TryStartNextRunTransition(ctx, run)
	if err != nil {
		return err
	}
	if started {
		if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
			return err
		}
	}

	return nil
}
