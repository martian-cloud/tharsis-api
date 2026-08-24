// Package admission decides whether a run's plan/apply node may be queued given
// the workspace state, and performs the transition (status change and workspace
// acquisition). It is the shared gate used by both the run commands and the
// transformers. Job creation for queued nodes is handled separately by a
// stateful change handler.
package admission

import (
	"context"
	"time"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Admitter gates and performs the transition of a run's next workspace-gated node based on workspace
// state.
//
// Admission rules:
//   - speculative run: always admitted (it only reads state, so the workspace lock and contention
//     don't apply; many may run).
//   - non-speculative run: workspace not locked and no other non-speculative run in progress
//     (CurrentApplyRunID empty or this run).
//
// Which node the run is waiting on does not change those rules — a pre-plan stage, the plan, a pre-apply
// stage and the apply all wait for the same slot — so the admitter never needs to identify it. The
// post-plan stage is not gated here at all: it inherits the slot the plan already holds.
type Admitter struct {
	dbClient *db.Client
}

// New creates a new Admitter.
func New(dbClient *db.Client) *Admitter {
	return &Admitter{dbClient: dbClient}
}

// TransitionedToQueuing reports whether any of the run's status changes moved the run into one of its
// *_queuing statuses — i.e. a workspace-gated node just became ready, so the run is now waiting for the
// slot. This is the signal the admission transformer reacts to (try to admit it now) and the workspace
// lock manager enqueues a work item for (so admission is retried later if that fails).
//
// Each gated node projects onto its own *_queuing status when it becomes pending, so one run-status
// check covers all of them without enumerating node types. Keying off the transition rather than the
// run's current status is what stops a redundant work item on every later update while the run merely
// remains queuing.
func TransitionedToQueuing(changes []statemachine.NodeStatusChange) bool {
	for _, sc := range changes {
		if rc, ok := sc.(statemachine.RunStatusChange); ok && rc.NewStatus.IsQueuing() {
			return true
		}
	}
	return false
}

// TryStartNextRunTransition performs the run's next workspace-gated transition, if one is available.
// The next transition is deterministic from the run's current state — start the pre-plan stage, queue
// the plan, start the pre-apply stage, or queue the apply — and these are sequential phases, so at most
// one applies at a time. It acquires the workspace before transitioning (so a failed acquisition, e.g.
// an OLE because another instance changed the workspace, leaves the node untouched for the caller to
// retry) and reports whether a transition was performed plus, when it was, the resulting node status
// changes. Callers need not know which transition is next.
//
// Which node to move is left entirely to statemachine.AdvanceRun, which starts whichever gated node is
// pending. All this needs to decide is whether the run is waiting on the workspace at all, which is
// exactly what a *_queuing run status means (see models.QueuingRunStatuses) — each gated node projects
// onto its own, so one status check covers them all. The check is not merely an optimisation: advancing
// a run with no pending node instead begins its next phase (starting a plan, or approving a manual
// apply), which is not admission's decision to make, and it would also take the workspace slot for a
// run that is not waiting for one.
//
// A queuing run needs no further precondition check, because a gated node is gated by being held in
// created until the step before it completes, never by being parked in pending. An apply, for instance,
// only reaches pending (and so apply_queuing) once the plan has finished with changes and any pre-apply
// stage has cleared; a run blocked on a policy gate is awaiting_decision, not queuing.
//
// This form performs no queue-ordering check, so it is for callers that have already established that
// it is this run's turn — the work item consumer, which selects the queuing run that has been waiting
// on the workspace longest and drives it through the QueueRun command. Callers reacting only to a
// single run's own transition must use TryStartNextRunTransitionIfNextInLine instead.
func (a *Admitter) TryStartNextRunTransition(ctx context.Context, run *models.Run) (bool, []statemachine.NodeStatusChange, error) {
	return a.tryStartNextRunTransition(ctx, run, false)
}

// TryStartNextRunTransitionIfNextInLine is the queue-order-preserving form of
// TryStartNextRunTransition, for callers that have not established that it is this run's turn — i.e.
// the admission transformer, which reacts to a run's own fresh queuing transition and knows nothing
// about the workspace's other waiting runs.
//
// It behaves identically except that a run competing for a free slot is admitted only when no other
// run was already waiting for that slot. Without this, a run whose queuing transition happens to commit
// while the workspace is momentarily free jumps ahead of runs that have been parked in *_queuing since
// before it existed. That window is not narrow: it spans from the workspace being released (or
// unlocked) until the work item consumer admits the next run in order.
//
// Deferring is always safe. The WorkspaceLockManager enqueues a QueuePendingRunsForWorkspace work item
// in the same transaction as the queuing transition, and the consumer admits the workspace's queuing
// runs in queue-entry order (it sorts by updated_at, the same column this check counts on), so a
// deferred run is guaranteed a later, ordered admission attempt. The cost of an unnecessary deferral is
// therefore one trip through the work-items queue, which is why the check errs toward deferring.
func (a *Admitter) TryStartNextRunTransitionIfNextInLine(ctx context.Context, run *models.Run) (bool, []statemachine.NodeStatusChange, error) {
	return a.tryStartNextRunTransition(ctx, run, true)
}

func (a *Admitter) tryStartNextRunTransition(ctx context.Context, run *models.Run, requireNextInLine bool) (bool, []statemachine.NodeStatusChange, error) {
	if !run.Status.IsQueuing() {
		// The run is not waiting on the workspace: nothing to do, and no workspace lookup needed.
		return false, nil, nil
	}

	ws, err := a.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to get workspace")
	}
	if ws == nil {
		return false, nil, nil
	}

	if !a.workspaceAdmits(ws, run) {
		return false, nil, nil
	}

	// A run competing for a free slot must wait its turn, so it cannot jump ahead of runs that were
	// already waiting on the workspace. The check is skipped for a run that already holds it: it is not
	// competing for the slot, so it cannot jump the queue, and deferring it would strand it — the work
	// item consumer only admits when the workspace is free, which it is not while this run holds it.
	// Speculative runs never take the slot at all, so they have no queue to jump either.
	if requireNextInLine && !run.Speculative() && !holdsWorkspace(ws, run) {
		nextInLine, err := a.isNextInLine(ctx, run)
		if err != nil {
			return false, nil, err
		}
		if !nextInLine {
			return false, nil, nil
		}
	}

	// Acquire the workspace before transitioning the node so a failed acquisition leaves the node
	// untouched. The caller decides whether an OLE should retry.
	if err := a.acquireWorkspace(ctx, run, ws); err != nil {
		return false, nil, err
	}

	// The slot is held, so the run is unblocked: the state machine advances whichever node was pending.
	changes, err := statemachine.AdvanceRun(run)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to advance run")
	}
	return true, changes, nil
}

// WorkspaceAvailable reports whether the workspace can accept a new non-speculative
// run: it is not locked and no non-speculative run is in progress. This is the single
// source of the workspace-availability rule — the work item consumer uses it to pre-filter
// before issuing queue commands, and the admitter's per-run checks build on it.
func WorkspaceAvailable(ws *models.Workspace) bool {
	return !ws.Locked && ws.CurrentApplyRunID == nil
}

// workspaceAvailableForRun is the per-run form of WorkspaceAvailable: it additionally
// admits the run that already occupies the workspace, so a run is never blocked by
// its own earlier acquisition.
func workspaceAvailableForRun(ws *models.Workspace, run *models.Run) bool {
	if WorkspaceAvailable(ws) {
		return true
	}
	return !ws.Locked && holdsWorkspace(ws, run)
}

// holdsWorkspace reports whether the workspace is currently occupied by this run.
func holdsWorkspace(ws *models.Workspace, run *models.Run) bool {
	return ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == run.Metadata.ID
}

// isNextInLine reports whether this run is at the front of its workspace's queue: no other run was
// already waiting on the workspace when this run joined the queue. If one was, admitting this run now
// would jump ahead of it, so the caller must leave the node pending for the work item consumer to admit
// in order.
//
// A count answers this — which runs are ahead doesn't matter, only whether any are — so the query asks
// for no rows and reads the (lazy) total count, matching how run limits are counted elsewhere:
//   - Statuses limits it to runs waiting on the workspace. A run that is not waiting is either not
//     competing for the slot or already holds it, and a holder is handled by workspaceAdmits instead.
//   - UpdatedBefore is the current time, so the count is of runs already waiting as of this instant.
//
// The run itself is not in that count. Its own queuing transition has not been written yet — the run
// store flushes UpdateRun after the transformers run — so its row still holds the status it had when
// the command began, which is not a queuing status precisely because it is transitioning into one now.
// This is the reason the query needs no self-exclusion, and it is safe to depend on: were the run ever
// counted, the count would be non-zero and the run would defer to the work item consumer.
//
// Speculative runs are counted even though they never take the slot, because there is no DB filter for
// them. That only costs an unnecessary deferral, and it is close to unreachable in practice: a
// speculative run is always admitted in the transaction that creates it, so it does not sit in a queuing
// status. Every imprecision here is deliberately in this direction — an over-count defers a run that
// could have gone now (one trip through the work-items queue), while an under-count jumps the queue.
func (a *Admitter) isNextInLine(ctx context.Context, run *models.Run) (bool, error) {
	workspaceID := run.WorkspaceID
	now := time.Now().UTC()

	waiting, err := a.dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(0)},
		Filter: &db.RunFilter{
			WorkspaceID:   &workspaceID,
			Statuses:      models.QueuingRunStatuses,
			UpdatedBefore: &now,
		},
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to query runs waiting on workspace")
	}

	// TotalCount is lazy: asking for zero rows above means this is the only query that runs.
	count, err := waiting.PageInfo.TotalCount(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to count runs waiting on workspace")
	}

	return count == 0, nil
}

// workspaceAdmits reports whether the workspace permits this run's next transition. Speculative runs
// only read state, so they are always admitted — the workspace lock and contention
// (CurrentApplyRunID) don't apply, and many may run at once.
func (a *Admitter) workspaceAdmits(ws *models.Workspace, run *models.Run) bool {
	if run.Speculative() {
		return true
	}
	return workspaceAvailableForRun(ws, run)
}

// acquireWorkspace marks the workspace as occupied by this non-speculative run
// so other non-speculative runs wait. Speculative runs never occupy it.
func (a *Admitter) acquireWorkspace(ctx context.Context, run *models.Run, ws *models.Workspace) error {
	if run.Speculative() || holdsWorkspace(ws, run) {
		return nil
	}
	ws.CurrentApplyRunID = &run.Metadata.ID
	if _, err := a.dbClient.Workspaces.UpdateWorkspace(ctx, ws); err != nil {
		// This may be an OLE if another instance changed the workspace concurrently.
		// It is returned as-is; callers decide how to handle it: the admission
		// transformer swallows it (leaving the node pending for the work-item retry)
		// while QueueRun lets it retry the (cheap) command.
		return errors.Wrap(err, "failed to acquire workspace")
	}
	return nil
}
