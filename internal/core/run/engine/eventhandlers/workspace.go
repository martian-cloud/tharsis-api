// Package eventhandlers provides run event handling functionality.
package eventhandlers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// WorkspaceLockManager manages workspace locking based on run state changes.
type WorkspaceLockManager struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewWorkspaceLockManager creates a new WorkspaceLockManager.
func NewWorkspaceLockManager(logger logger.Logger, dbClient *db.Client) *WorkspaceLockManager {
	return &WorkspaceLockManager{dbClient: dbClient, logger: logger}
}

// HandleRunChanges implements RunChangeHandler.
//
// Acquisition of the workspace (setting CurrentApplyRunID) is performed by the
// admitter when it starts a non-speculative run's next transition — driven by the admission
// transformer on a fresh pending transition (via TryStartNextRunTransitionIfNextInLine, so it
// cannot jump the workspace's queue), or by the QueueRun command for a parked run. This handler
// releases it (when the run completes, or when a run is parked awaiting a human decision) and
// enqueues a work item so the work item consumer re-evaluates the workspace.
//
// The work item is enqueued when this run change either freed the workspace or
// left a node pending (ready but not yet queued). Enqueuing in the same
// transaction as the run change is what makes admission safe under concurrent
// instances: the run is guaranteed to be visible when the work item consumer processes
// the item, so a freed-workspace wakeup can't be consumed before a concurrently
// pending run commits and strand it.
func (h *WorkspaceLockManager) HandleRunChanges(ctx context.Context, changes []types.RunChange) error {
	for _, change := range changes {
		run := change.Run

		freedWorkspace := false

		// Releasing and dirtying the workspace only applies to non-speculative
		// runs, since only they hold the workspace.
		if !run.Speculative() {
			// Release the workspace when the run completes, when a manual run is parked at planned
			// awaiting approval, or when any policy stage is blocked awaiting a manual override. In each
			// of these the run is waiting on a human while holding the workspace, so it should not keep
			// the slot; it is re-acquired when the run resumes, because clearing a gate readies the next
			// workspace-gated node and the admitter has to queue it.
			//
			// Every stage is listed, including pre-plan: a gated stage holds the slot for its own
			// evaluation, so a run blocked at any of these gates has a slot to give up. Note the slot is
			// only released on the human-waits, not while a stage is merely running — an automated stage
			// is short, and releasing it would let another run take the slot and get this one
			// stale-discarded for no benefit.
			releaseWorkspace := run.IsComplete() ||
				(run.Status == models.RunPlanned && !run.AutoApply) ||
				run.Status == models.RunPrePlanAwaitingDecision ||
				run.Status == models.RunPostPlanAwaitingDecision ||
				run.Status == models.RunPreApplyAwaitingDecision

			if releaseWorkspace {
				ws, err := h.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
				if err != nil {
					return err
				}
				if ws == nil {
					continue
				}
				if ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == run.Metadata.ID {
					ws.CurrentApplyRunID = nil
					if _, err := h.dbClient.Workspaces.UpdateWorkspace(ctx, ws); err != nil {
						return err
					}
					freedWorkspace = true
				}
			}

			// Mark workspace dirty on force cancel during apply
			if run.ForceCanceled {
				applyNode := run.Apply
				if applyNode != nil && applyNode.Status == models.ApplyCanceled {
					ws, err := h.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
					if err != nil {
						return err
					}
					if ws == nil {
						continue
					}
					ws.DirtyState = true
					if _, err := h.dbClient.Workspaces.UpdateWorkspace(ctx, ws); err != nil {
						return err
					}
				}
			}
		}

		// Ask the work item consumer to re-evaluate the workspace when it was just freed (so the next
		// run can proceed) or when this change moved the run into one of its *_queuing statuses, meaning
		// a node became ready but is not yet admitted. Keying off the transition into a queuing status —
		// rather than the run merely still being in one — avoids a redundant work item on every later
		// update. It also closes the race where the workspace is freed by another instance before this
		// run commits: the work item is committed with the run, so it is always processed after the run
		// is visible.
		if freedWorkspace || admission.TransitionedToQueuing(change.NodeStatusChanges) {
			if _, err := h.dbClient.WorkItemsQueue.AddWorkItemToQueue(ctx, &db.AddWorkItemToQueueInput{
				Type:    db.QueuePendingRunsForWorkspaceType,
				Payload: &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: run.WorkspaceID},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
