package eventhandlers

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// StalePlannedRunDiscarder enqueues a work item to discard a workspace's stale planned
// runs whenever one of its runs is successfully applied.
type StalePlannedRunDiscarder struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewStalePlannedRunDiscarder creates a new StalePlannedRunDiscarder.
func NewStalePlannedRunDiscarder(logger logger.Logger, dbClient *db.Client) *StalePlannedRunDiscarder {
	return &StalePlannedRunDiscarder{dbClient: dbClient, logger: logger}
}

// HandleRunChanges implements RunChangeHandler.
//
// When a run's apply finishes successfully (the run reaches applied), the workspace state
// has changed, so any other runs in that workspace still parked at planned (awaiting
// approval) are now stale. This enqueues a work item, in the same transaction as the
// apply, for the work item consumer to discard those runs asynchronously. It is registered as a
// stateful handler so the work item commits atomically with the run change.
func (h *StalePlannedRunDiscarder) HandleRunChanges(ctx context.Context, changes []types.RunChange) error {
	// This handler runs in-transaction right after the apply transition, so "now" is the
	// apply-completion instant. It is the staleness cutoff: every planned run last updated
	// (i.e. that entered the planned state) before it planned against pre-apply state. The
	// cutoff is frozen here rather than read when the work item is processed, since async
	// processing delay would otherwise sweep in runs that entered planned after the apply.
	now := time.Now().UTC()

	enqueued := map[string]struct{}{}

	for _, change := range changes {
		for _, statusChange := range change.NodeStatusChanges {
			rc, ok := statusChange.(statemachine.RunStatusChange)
			if !ok || rc.NewStatus != models.RunApplied {
				continue
			}

			workspaceID := change.Run.WorkspaceID
			if _, done := enqueued[workspaceID]; done {
				continue
			}
			enqueued[workspaceID] = struct{}{}

			if _, err := h.dbClient.WorkItemsQueue.AddWorkItemToQueue(ctx, &db.AddWorkItemToQueueInput{
				Type: db.DiscardStalePlannedRunsForWorkspaceType,
				Payload: &db.DiscardStalePlannedRunsForWorkspacePayload{
					WorkspaceID:      workspaceID,
					ApplyCompletedAt: now,
				},
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
