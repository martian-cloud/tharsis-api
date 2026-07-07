// Package admission decides whether a run's plan/apply node may be queued given
// the workspace state, and performs the transition (status change and workspace
// acquisition). It is the shared gate used by both the run commands and the
// transformers. Job creation for queued nodes is handled separately by a
// stateful change handler.
package admission

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Admitter gates and performs the transition of a run's plan/apply node to
// queued based on workspace state.
//
// Admission rules:
//   - speculative plan: always admitted (it only reads state, so the workspace lock
//     and contention don't apply; many may run).
//   - non-speculative plan: workspace not locked and no other non-speculative
//     run in progress (CurrentApplyRunID empty or this run).
//   - apply: plan finished with changes, workspace not locked, and no other
//     non-speculative run in progress.
type Admitter struct {
	dbClient *db.Client
}

// New creates a new Admitter.
func New(dbClient *db.Client) *Admitter {
	return &Admitter{dbClient: dbClient}
}

// TryQueuePlan queues the run's plan node if the workspace allows it. It reports
// whether the plan was queued and, when it was, the resulting node status changes
// so the caller can record them on the run store.
func (a *Admitter) TryQueuePlan(ctx context.Context, run *models.Run) (bool, []statemachine.NodeStatusChange, error) {
	ws, err := a.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to get workspace")
	}
	if ws == nil || !a.canQueuePlan(ws, run) {
		return false, nil, nil
	}

	// Acquire the workspace before transitioning the node so a failed acquisition
	// (e.g. an OLE because another instance changed the workspace) leaves the node
	// untouched and pending. The caller decides whether an OLE should retry.
	if err := a.acquireWorkspace(ctx, run, ws); err != nil {
		return false, nil, err
	}

	changes, err := statemachine.SetPlanStatus(run, models.PlanQueued)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to transition plan node to queued")
	}
	return true, changes, nil
}

// TryQueueApply queues the run's apply node if the workspace allows it. It
// reports whether the apply was queued and, when it was, the resulting node
// status changes so the caller can record them on the run store.
func (a *Admitter) TryQueueApply(ctx context.Context, run *models.Run) (bool, []statemachine.NodeStatusChange, error) {
	if run.Apply == nil {
		return false, nil, nil
	}
	ws, err := a.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to get workspace")
	}
	if ws == nil || !a.canQueueApply(ws, run) {
		return false, nil, nil
	}

	// Acquire the workspace before transitioning the node so a failed acquisition
	// (e.g. an OLE because another instance changed the workspace) leaves the node
	// untouched and pending. The caller decides whether an OLE should retry.
	if err := a.acquireWorkspace(ctx, run, ws); err != nil {
		return false, nil, err
	}

	changes, err := statemachine.SetApplyStatus(run, models.ApplyQueued)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to transition apply node to queued")
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
	return !ws.Locked && ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == run.Metadata.ID
}

func (a *Admitter) canQueuePlan(ws *models.Workspace, run *models.Run) bool {
	// Speculative plans only read state, so they are always admitted — the workspace
	// lock and contention (CurrentApplyRunID) don't apply.
	if run.Speculative() {
		return true
	}
	return workspaceAvailableForRun(ws, run)
}

func (a *Admitter) canQueueApply(ws *models.Workspace, run *models.Run) bool {
	if run.Plan.Status != models.PlanFinished || !run.Plan.HasChanges {
		return false
	}
	return workspaceAvailableForRun(ws, run)
}

// acquireWorkspace marks the workspace as occupied by this non-speculative run
// so other non-speculative runs wait. Speculative runs never occupy it.
func (a *Admitter) acquireWorkspace(ctx context.Context, run *models.Run, ws *models.Workspace) error {
	if run.Speculative() || (ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == run.Metadata.ID) {
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
