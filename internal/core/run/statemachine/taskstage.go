package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TaskStageNode represents a stage of a run (pre_plan, post_plan, ...) in the state machine. Its
// status is the aggregate verdict of its child policy check nodes (and, in future, run task
// results): the per-stage verdict logic lives here once, so a stage projects a single status onto
// the run regardless of how many checks it owns or which stage it is.
type TaskStageNode struct {
	nodeBase
	id            string
	stageName     models.RunTaskStageName
	status        models.RunTaskStageStatus
	policyChecks  []*PolicyCheckNode
	statusChanges []TaskStageStatusChange
}

// NewTaskStageNode creates a new task stage node.
func NewTaskStageNode(id string, stageName models.RunTaskStageName, status models.RunTaskStageStatus) *TaskStageNode {
	return &TaskStageNode{
		id:            id,
		stageName:     stageName,
		status:        status,
		statusChanges: []TaskStageStatusChange{},
	}
}

// AddPolicyCheckNode appends a policy check node as a child of this stage.
func (n *TaskStageNode) AddPolicyCheckNode(c *PolicyCheckNode) {
	n.policyChecks = append(n.policyChecks, c)
}

// ID returns the stage node ID.
func (n *TaskStageNode) ID() string { return n.id }

// StageName returns the run stage this node evaluates.
func (n *TaskStageNode) StageName() models.RunTaskStageName { return n.stageName }

// Status returns the current stage status.
func (n *TaskStageNode) Status() models.RunTaskStageStatus { return n.status }

// Path returns the run-relative path identifying this stage (the stage is unique per run).
func (n *TaskStageNode) Path() string { return string(n.stageName) }

// PolicyChecks returns the stage's child policy check nodes.
func (n *TaskStageNode) PolicyChecks() []*PolicyCheckNode { return n.policyChecks }

// init registers this stage's listeners on its child checks: a pending child re-enters the stage, a
// queued child moves the stage to running (projecting onto the run), and a child reaching a terminal
// or blocking status re-evaluates the aggregate verdict.
func (n *TaskStageNode) init() {
	for _, check := range n.policyChecks {
		check.registerListener(string(models.PolicyCheckPending), n.handleCheckPending)
		check.registerListener(string(models.PolicyCheckQueued), n.handleCheckQueued)
		check.registerListener(string(models.PolicyCheckPassed), n.handleVerdict)
		check.registerListener(string(models.PolicyCheckOverridden), n.handleVerdict)
		check.registerListener(string(models.PolicyCheckErrored), n.handleVerdict)
		check.registerListener(string(models.PolicyCheckSoftFailed), n.handleVerdict)
	}
}

// queueReadyChecks queues every check that has not yet started (its policy-eval job is then created
// by the job-creation transformer). It is driven from SetStatus when the stage enters running, so a
// caller starts a stage's checks simply by moving the stage to running.
func (n *TaskStageNode) queueReadyChecks() error {
	for _, check := range n.policyChecks {
		if check.Status().NotStarted() {
			if err := check.SetStatus(models.PolicyCheckQueued); err != nil {
				return err
			}
		}
	}
	return nil
}

// skipUnstartedChecks skips every check that never started. Called when the stage enters skipped so
// the stage fully settles its children without the run node having to iterate them directly.
func (n *TaskStageNode) skipUnstartedChecks() error {
	for _, check := range n.policyChecks {
		if check.Status().NotStarted() {
			if err := check.SetStatus(models.PolicyCheckSkipped); err != nil {
				return err
			}
		}
	}
	return nil
}

// cancelActiveChecks settles every check when the stage is canceled: active checks (started but not
// yet final) are canceled; unstarted ones are skipped. The distinction matches the stage and check
// lifecycles — "skipped" records a check that will never run for this outcome, while "canceled"
// records one that was interrupted mid-flight.
func (n *TaskStageNode) cancelActiveChecks() error {
	for _, check := range n.policyChecks {
		if check.Status().IsFinalStatus() {
			continue
		}
		if check.Status().NotStarted() {
			if err := check.SetStatus(models.PolicyCheckSkipped); err != nil {
				return err
			}
		} else {
			if err := check.SetStatus(models.PolicyCheckCanceled); err != nil {
				return err
			}
		}
	}
	return nil
}

// handleCheckPending re-enters the stage for a retried child. A check goes errored/canceled/soft_failed
// -> pending when it is retried, and the stage it belongs to is errored/canceled/awaiting_override with
// it; this restarts the stage, which re-queues the check (and so creates a fresh policy-eval job).
//
// The stage decides how to restart, because only the stage knows whether it is workspace-gated: a
// gated stage returns to pending and waits to be re-admitted, since the run released its slot when it
// terminated, and a job must not be created for it before then. An ungated stage (post-plan) goes
// straight back to running on the slot the plan already holds. Keeping that choice here is what lets
// a retry be expressed as nothing more than "set the check to pending".
func (n *TaskStageNode) handleCheckPending() error {
	if n.stageName.IsWorkspaceGated() {
		return n.SetStatus(models.RunTaskStagePending)
	}
	return n.SetStatus(models.RunTaskStageRunning)
}

// handleCheckQueued projects a queued child onto the stage as running. This is a no-op in the flows
// the stage itself drives (SetStatus queues the checks after the stage enters running, and a retry
// arrives via handleCheckPending), so it covers a check queued from outside the stage — a job-status
// sync reporting a job back to queued — by keeping the stage's status consistent with its children.
func (n *TaskStageNode) handleCheckQueued() error {
	return n.SetStatus(models.RunTaskStageRunning)
}

// handleVerdict re-evaluates the aggregate verdict across ALL of the stage's checks whenever any
// check reaches a terminal or blocking status:
//   - any errored check fails the stage;
//   - if all remaining checks are awaiting_override (none still running/queued), the stage is
//     awaiting_override (human approval needed);
//   - once every check has cleared (passed or overridden), the stage is completed;
//   - if any check is still running or queued, no change (stage stays running).
//
// skipped and canceled checks are settled by the run (a check is only skipped/canceled at run
// termination, which drives the stage status directly), so they neither block the verdict nor count
// as cleared — the verdict is computed over the stage's remaining checks. Ignoring them keeps a
// multi-check stage from hanging in running if one check is settled by the run while a sibling clears.
//
// The stage's status listeners project each of these onto the run.
func (n *TaskStageNode) handleVerdict() error {
	for _, check := range n.policyChecks {
		if check.Status() == models.PolicyCheckErrored {
			return n.SetStatus(models.RunTaskStageErrored)
		}
	}

	allCleared := true
	for _, check := range n.policyChecks {
		switch check.Status() {
		case models.PolicyCheckPassed, models.PolicyCheckOverridden:
			// cleared
		case models.PolicyCheckSoftFailed:
			allCleared = false
		case models.PolicyCheckSkipped, models.PolicyCheckCanceled:
			// Settled by the run; neither blocks the verdict nor counts as cleared.
		default:
			// A check is still running or queued: no verdict yet.
			return nil
		}
	}

	if !allCleared {
		return n.SetStatus(models.RunTaskStageAwaitingOverride)
	}
	return n.SetStatus(models.RunTaskStageCompleted)
}

// restartCanceled re-runs a stage that was canceled with its decision still outstanding — the gate a
// discarded run was abandoned at, which undiscard revives (UndiscardRun). Its canceled checks return to
// pending, the same edge a retry uses: handleCheckPending restarts the stage (waiting for the workspace
// slot first if the stage is gated), the restarted stage re-queues the check, and a fresh policy-eval
// job re-evaluates it. Re-evaluating rather than restoring the previous verdict is deliberate — the run
// gate the check was blocked on was canceled when the run was discarded, and the policies may have
// changed since — and it is why this is expressed as the check-level retry rather than as a reversal.
// Checks that had already cleared (passed/overridden) were never canceled and are left as they are.
func (n *TaskStageNode) restartCanceled() error {
	for _, check := range n.policyChecks {
		if check.Status() == models.PolicyCheckCanceled {
			if err := check.SetStatus(models.PolicyCheckPending); err != nil {
				return err
			}
		}
	}
	return nil
}

// resetIfSkipped returns a skipped stage — and the checks it owns — to created, so the stage can run
// again when the run is retried past the point that skipped it (the stage was skipped because it
// never started when an earlier node failed or the run was canceled). It is a no-op for a stage that
// is not skipped. created carries no listeners, so this restores state without projecting onto the
// run; the stage's checks are re-queued later when the stage next enters running.
func (n *TaskStageNode) resetIfSkipped() error {
	if n.status != models.RunTaskStageSkipped {
		return nil
	}
	for _, check := range n.policyChecks {
		if check.Status() == models.PolicyCheckSkipped {
			if err := check.SetStatus(models.PolicyCheckCreated); err != nil {
				return err
			}
		}
	}
	return n.SetStatus(models.RunTaskStageCreated)
}

// SetStatus transitions the stage to a new status and fires its listeners, which project the stage
// verdict onto the run. Entering running additionally queues the stage's not-yet-started checks, so
// starting a stage's checks is just a matter of moving the stage to running. Setting the status the
// node already holds is a no-op.
func (n *TaskStageNode) SetStatus(status models.RunTaskStageStatus) error {
	if n.status == status {
		return nil
	}
	if !canTransitionTo(taskStageTransitions, n.status, status) {
		return fmt.Errorf("invalid task stage status transition from %q to %q", n.status, status)
	}
	n.statusChanges = append(n.statusChanges, TaskStageStatusChange{
		OldStatus: n.status,
		NewStatus: status,
		StageID:   n.id,
		Path:      n.Path(),
	})
	n.status = status
	if err := n.nodeBase.fireEvent(string(status)); err != nil {
		return err
	}
	// Cascade the new status onto the stage's child checks after fireEvent so the stage first projects
	// its own status onto the run before touching children.
	switch status {
	case models.RunTaskStageRunning:
		return n.queueReadyChecks()
	case models.RunTaskStageSkipped:
		return n.skipUnstartedChecks()
	case models.RunTaskStageCanceled:
		return n.cancelActiveChecks()
	case models.RunTaskStageErrored:
		// One check erroring fails the whole stage (handleVerdict), which leaves this stage's other
		// checks stranded mid-flight in a multi-check stage. Settle them the same way a cancel does —
		// active checks canceled, unstarted ones skipped — so no sibling hangs in running/queued after
		// the stage has already errored. The errored check itself is final and is left untouched.
		return n.cancelActiveChecks()
	}
	return nil
}

// GetStatusChanges returns all status changes recorded on this node.
func (n *TaskStageNode) GetStatusChanges() []NodeStatusChange {
	changes := make([]NodeStatusChange, len(n.statusChanges))
	for i, c := range n.statusChanges {
		changes[i] = c
	}
	return changes
}
