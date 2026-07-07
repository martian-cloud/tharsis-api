package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// RunNode represents the root node of a run, containing plan and apply child nodes.
type RunNode struct {
	nodeBase
	plan          *PlanNode
	apply         *ApplyNode
	status        models.RunStatus
	statusChanges []RunStatusChange
}

// NewRunNode creates a new run node.
func NewRunNode(status models.RunStatus) *RunNode {
	return &RunNode{
		status:        status,
		statusChanges: []RunStatusChange{},
	}
}

// SetPlanNode sets the plan node on the run.
func (n *RunNode) SetPlanNode(p *PlanNode) {
	n.plan = p
}

// SetApplyNode sets the optional apply node on the run.
func (n *RunNode) SetApplyNode(a *ApplyNode) {
	n.apply = a
}

// Plan returns the plan node.
func (n *RunNode) Plan() *PlanNode { return n.plan }

// Apply returns the apply node (nil for speculative runs).
func (n *RunNode) Apply() *ApplyNode { return n.apply }

// Status returns the current run status.
func (n *RunNode) Status() models.RunStatus { return n.status }

// GetStatusChanges returns all status changes recorded on the run node.
func (n *RunNode) GetStatusChanges() []NodeStatusChange {
	changes := make([]NodeStatusChange, len(n.statusChanges))
	for i, c := range n.statusChanges {
		changes[i] = c
	}
	return changes
}

func (n *RunNode) init() {
	// A node entering the pending state (ready/approved but not yet admitted)
	// projects onto the run as queuing (plan waiting to be queued) or
	// queuing_apply (apply waiting to be queued) — in the forward flow and in the
	// retry resets alike.
	if n.plan != nil {
		n.plan.registerListener(string(models.PlanPending), n.handlePlanPending)
		n.plan.registerListener(string(models.PlanQueued), n.handlePlanQueued)
		n.plan.registerListener(string(models.PlanRunning), n.handlePlanRunning)
		n.plan.registerListener(string(models.PlanFinished), n.handlePlanSucceeded)
		n.plan.registerListener(string(models.PlanErrored), n.handlePlanErrored)
		n.plan.registerListener(string(models.PlanCanceled), n.handlePlanCanceled)
	}
	if n.apply != nil {
		n.apply.registerListener(string(models.ApplyPending), n.handleApplyPending)
		n.apply.registerListener(string(models.ApplyQueued), n.handleApplyQueued)
		n.apply.registerListener(string(models.ApplyRunning), n.handleApplyRunning)
		n.apply.registerListener(string(models.ApplyFinished), n.handleApplySucceeded)
		n.apply.registerListener(string(models.ApplyErrored), n.handleApplyErrored)
		n.apply.registerListener(string(models.ApplyCanceled), n.handleApplyCanceled)
	}

	// The run node also listens to its own transitions: a run that reaches a final
	// state with its apply never started marks the apply skipped, and queuing
	// readies the plan (and resets a skipped apply on a plan retry).
	n.registerListener(string(models.RunPlannedAndFinished), n.skipUnstartedApply)
	n.registerListener(string(models.RunErrored), n.skipUnstartedApply)
	n.registerListener(string(models.RunCanceled), n.skipUnstartedApply)
	n.registerListener(string(models.RunDiscarded), n.skipUnstartedApply)
	n.registerListener(string(models.RunQueuing), n.handleRunQueuing)
	// A run (re-)entering planned must have an apply that again awaits approval: undiscard
	// returns a discarded run to planned, so restore its skipped apply to created. In the
	// normal plan-finished flow the apply is already created/pending, so this is a no-op.
	n.registerListener(string(models.RunPlanned), n.unskipApply)
}

// handleRunQueuing reacts to the run entering the queuing state. When the run is
// queued directly (run-driven, at creation) it drives the still-created plan to
// pending so admission can pick it up — silently, since firing the plan's listeners
// would re-project onto the run. When queuing is reached via a plan retry the plan
// is already pending, and the reset instead returns a skipped apply to created so
// it again awaits the new plan's outcome.
func (n *RunNode) handleRunQueuing() error {
	if n.plan != nil && n.plan.Status() == models.PlanCreated {
		if err := n.plan.setStatusSilently(models.PlanPending); err != nil {
			return err
		}
	}
	return n.unskipApply()
}

// skipUnstartedApply moves an apply that never started (still in created) to skipped
// when the run reaches a final state, so it is explicit that the apply will never run.
// The run drives the apply here, so it is set silently to avoid re-projecting onto the
// run; the change is still recorded for persistence. An apply that already progressed
// (running, canceled, errored, ...) is left untouched — its own status conveys the outcome.
func (n *RunNode) skipUnstartedApply() error {
	if n.apply == nil || n.apply.Status() != models.ApplyCreated {
		return nil
	}
	return n.apply.setStatusSilently(models.ApplySkipped)
}

// unskipApply returns a skipped apply to created when the run leaves a final state — via
// the plan-retry reset (run back to queuing) or an undiscard (run back to planned). The
// apply again awaits its outcome, and downstream flows (auto-apply advancement, the
// start-apply command) expect a not-yet-started apply to be in created.
func (n *RunNode) unskipApply() error {
	if n.apply == nil || n.apply.Status() != models.ApplySkipped {
		return nil
	}
	return n.apply.setStatusSilently(models.ApplyCreated)
}

// handlePlanPending projects a pending plan onto the run: the plan is waiting to be
// queued (admitted to the workspace), so the run is queuing. This covers the forward
// flow (pending -> queuing) and the plan retry (errored/canceled -> queuing) alike.
func (n *RunNode) handlePlanPending() error {
	return n.SetStatus(models.RunQueuing)
}

// handleApplyPending projects a pending apply onto the run: the apply is approved and
// waiting to be queued, so the run is queuing_apply. This covers the manual start-apply
// flow (planned -> queuing_apply) and the apply retry (errored/canceled -> queuing_apply)
// alike.
func (n *RunNode) handleApplyPending() error {
	return n.SetStatus(models.RunQueuingApply)
}

// SetStatus performs a run status transition. The run status is a projection of its
// children, so in practice it is driven by the child-node listeners (which is why it
// shares the SetStatus shape of the plan and apply nodes). Setting the status the run
// already holds is a no-op — the pending listeners re-assert the current status in the
// normal forward flow — while any other transition outside the run lifecycle is an error.
func (n *RunNode) SetStatus(status models.RunStatus) error {
	if n.status == status {
		return nil
	}
	if !canTransitionTo(runTransitions, n.status, status) {
		return fmt.Errorf("invalid run status transition from %q to %q", n.status, status)
	}
	n.statusChanges = append(n.statusChanges, RunStatusChange{
		OldStatus: n.status,
		NewStatus: status,
	})
	n.status = status
	return n.nodeBase.fireEvent(string(status))
}

func (n *RunNode) handlePlanQueued() error {
	return n.SetStatus(models.RunPlanQueued)
}

func (n *RunNode) handlePlanRunning() error {
	return n.SetStatus(models.RunPlanning)
}

func (n *RunNode) handlePlanSucceeded() error {
	// A run with no apply node, or whose plan produced no changes, has nothing
	// to apply and finishes immediately.
	if n.apply == nil || !n.plan.hasChanges {
		return n.SetStatus(models.RunPlannedAndFinished)
	}
	// An auto-apply run is pre-approved, so its apply is ready to be queued as
	// soon as the plan finishes with changes. A manual run leaves the apply in
	// created until it is approved (the start-apply command moves it to pending).
	if n.apply.autoApply {
		// The run drives the apply here, so set it silently to avoid re-projecting,
		// then record the projection itself: the run passes through planned (the
		// plan completed) straight into queuing_apply (the apply awaits admission).
		if err := n.apply.setStatusSilently(models.ApplyPending); err != nil {
			return err
		}
		if err := n.SetStatus(models.RunPlanned); err != nil {
			return err
		}
		return n.SetStatus(models.RunQueuingApply)
	}
	return n.SetStatus(models.RunPlanned)
}

func (n *RunNode) handlePlanErrored() error {
	// The run ends here; the run node's own terminal listener marks the
	// never-started apply as skipped.
	return n.SetStatus(models.RunErrored)
}

func (n *RunNode) handlePlanCanceled() error {
	// As with a plan error, the never-started apply is marked skipped.
	return n.SetStatus(models.RunCanceled)
}

func (n *RunNode) handleApplyQueued() error {
	return n.SetStatus(models.RunApplyQueued)
}

func (n *RunNode) handleApplyRunning() error {
	return n.SetStatus(models.RunApplying)
}

func (n *RunNode) handleApplySucceeded() error {
	return n.SetStatus(models.RunApplied)
}

func (n *RunNode) handleApplyErrored() error {
	return n.SetStatus(models.RunErrored)
}

func (n *RunNode) handleApplyCanceled() error {
	return n.SetStatus(models.RunCanceled)
}
