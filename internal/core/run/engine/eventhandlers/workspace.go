// Package eventhandlers provides run event handling functionality.
package eventhandlers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
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
// Acquisition of the workspace (setting CurrentApplyRunID) happens in the
// admission transformer when a non-speculative node is queued. This handler
// releases it (when the run completes, or when a manual run reaches planned with
// auto-apply off and is awaiting approval) and enqueues a work item so the
// work item consumer re-evaluates the workspace.
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
			// Release the workspace when the run completes, or when a manual run is
			// parked at planned awaiting approval.
			releaseWorkspace := run.IsComplete() ||
				(run.Status == models.RunPlanned && !run.AutoApply)

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

		// Ask the work item consumer to re-evaluate the workspace when it was just freed (so
		// the next run can proceed) or when this change moved the run into a queuing state
		// (a node became ready but is not yet admitted, so it needs to be queued). Enqueuing
		// on the transition — rather than whenever the run merely remains queuing — avoids a
		// redundant work item on every later update. It also closes the race where the
		// workspace is freed by another instance before this queuing run commits: the work
		// item is committed with the run, so it is always processed after the run is visible.
		if freedWorkspace || transitionedToQueuing(change) {
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

// transitionedToQueuing reports whether this change set moved the run into a queuing
// state: queuing (its plan is now waiting to be admitted) or queuing_apply (its apply
// is). That transition is the moment the run needs the work item consumer to
// (re-)evaluate the workspace. Keying off the transition rather than the run's current
// status avoids re-enqueuing a redundant work item on every later update while the run
// merely remains queuing.
func transitionedToQueuing(change types.RunChange) bool {
	for _, sc := range change.NodeStatusChanges {
		rc, ok := sc.(statemachine.RunStatusChange)
		if !ok {
			continue
		}
		if rc.NewStatus == models.RunQueuing || rc.NewStatus == models.RunQueuingApply {
			return true
		}
	}
	return false
}
