package engine

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// reconcileMinInterval/reconcileMaxInterval bound the sweep interval. The actual
	// wait is randomized within this range so multiple API instances don't all sweep
	// the runs table at the same moment.
	reconcileMinInterval = 5 * time.Minute
	reconcileMaxInterval = 10 * time.Minute

	// reconcileStaleThreshold is the minimum age (since last update) a queuing run must
	// reach before the reconciler re-drives it. The event-driven path advances queuing
	// runs within seconds (or one workItemEventTimeout on a missed event); the threshold
	// keeps the reconciler from churning work items for runs that path is already
	// handling. It is not a correctness guard — re-enqueueing is harmless at any age.
	reconcileStaleThreshold = 5 * time.Minute

	// reconcileBatchSize is the number of runs fetched per page during a sweep.
	reconcileBatchSize = 1000
)

var (
	runReconcilerAttempts           = metric.NewCounter("run_reconciler_attempts", "Number of run reconciler sweep attempts.")
	runReconcilerWorkspacesEnqueued = metric.NewCounter("run_reconciler_workspaces_enqueued", "Number of workspaces re-enqueued by the run reconciler.")
)

// Reconciler is a safety net for runs stranded in queuing/queuing_apply. The event-driven
// path (WorkspaceLockManager enqueues a QUEUE_PENDING_RUNS_FOR_WORKSPACE work item that the
// WorkItemConsumer processes) normally advances these runs, but a run is stranded if that
// work item is never enqueued (e.g. a crash between commit and enqueue) or never processed
// (e.g. it exhausts maxWorkItemClaims and is dropped as undeliverable). The reconciler
// periodically sweeps the runs table and re-enqueues the workspace work item so the
// existing consumer logic picks the run back up.
type Reconciler struct {
	logger             logger.Logger
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
}

// NewReconciler creates a new run Reconciler.
func NewReconciler(logger logger.Logger, dbClient *db.Client, maintenanceMonitor maintenance.Monitor) *Reconciler {
	return &Reconciler{logger: logger, dbClient: dbClient, maintenanceMonitor: maintenanceMonitor}
}

// Start launches the reconciliation sweep as a background goroutine that runs on a
// randomized interval until the context is canceled.
func (r *Reconciler) Start(ctx context.Context) {
	r.logger.Info("run reconciler started")

	go func() {
		for {
			// Randomize the sleep within [min, max) to avoid all instances sweeping at once.
			sleep := reconcileMinInterval + time.Duration(rand.Int64N(int64(reconcileMaxInterval-reconcileMinInterval)))

			select {
			case <-time.After(sleep):
				runReconcilerAttempts.Inc()
				if err := r.reconcile(ctx); err != nil {
					r.logger.Errorf("run reconciler sweep failed: %v", err)
				}
			case <-ctx.Done():
				r.logger.Info("run reconciler stopped")
				return
			}
		}
	}()
}

// reconcile paginates every run in queuing/queuing_apply that has been stale for at least
// reconcileStaleThreshold and enqueues one QUEUE_PENDING_RUNS_FOR_WORKSPACE work item per
// distinct workspace. The consumer's handler is idempotent and re-checks workspace
// availability, so re-enqueueing is safe even for runs legitimately waiting on a busy
// workspace.
func (r *Reconciler) reconcile(ctx context.Context) error {
	// Re-enqueueing is a DB write, so skip the sweep while in maintenance mode; the next
	// sweep after maintenance ends re-drives any still-stranded runs.
	inMaintenance, err := r.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		r.logger.Errorf("run reconciler failed to check maintenance mode: %v", err)
		return nil
	}
	if inMaintenance {
		return nil
	}

	cutoff := time.Now().UTC().Add(-reconcileStaleThreshold)
	sort := db.RunSortableFieldCreatedAtAsc

	// One work item per workspace per sweep, deduped across pages. A workspace stays
	// marked even if its enqueue fails, so a transient failure can't be hammered within
	// a single sweep; the next sweep retries.
	enqueued := map[string]struct{}{}
	var cursor *string

	for {
		result, err := r.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
			Sort: &sort,
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(reconcileBatchSize),
				After: cursor,
			},
			Filter: &db.RunFilter{
				Statuses:      []models.RunStatus{models.RunQueuing, models.RunQueuingApply},
				UpdatedBefore: &cutoff,
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to query queuing runs in reconciler")
		}

		for _, run := range result.Runs {
			if _, ok := enqueued[run.WorkspaceID]; ok {
				continue
			}
			enqueued[run.WorkspaceID] = struct{}{}

			if _, err := r.dbClient.WorkItemsQueue.AddWorkItemToQueue(ctx, &db.AddWorkItemToQueueInput{
				Type:    db.QueuePendingRunsForWorkspaceType,
				Payload: &db.QueuePendingRunsForWorkspacePayload{WorkspaceID: run.WorkspaceID},
			}); err != nil {
				// Best-effort: one workspace's enqueue failure must not abort the sweep.
				r.logger.Errorf("run reconciler failed to enqueue work item for workspace %s: %v", run.WorkspaceID, err)
				continue
			}
			runReconcilerWorkspacesEnqueued.Inc()
		}

		if !result.PageInfo.HasNextPage {
			break
		}

		cursor, err = result.PageInfo.Cursor(result.Runs[len(result.Runs)-1])
		if err != nil {
			return errors.Wrap(err, "failed to get next cursor in reconciler")
		}
	}

	return nil
}
