package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// RunNode represents the root node of a run, containing plan, apply, and task stage
// child nodes (each task stage owning its policy checks).
type RunNode struct {
	nodeBase
	plan          *PlanNode
	apply         *ApplyNode
	taskStages    []*TaskStageNode
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

// AddTaskStageNode appends a task stage node to the run.
func (n *RunNode) AddTaskStageNode(s *TaskStageNode) {
	n.taskStages = append(n.taskStages, s)
}

// Plan returns the plan node.
func (n *RunNode) Plan() *PlanNode { return n.plan }

// Apply returns the apply node (nil for speculative runs).
func (n *RunNode) Apply() *ApplyNode { return n.apply }

// TaskStages returns the run's task stage nodes.
func (n *RunNode) TaskStages() []*TaskStageNode { return n.taskStages }

// TaskStage returns the run's task stage node for the given stage, or nil if it has none.
func (n *RunNode) TaskStage(stage models.RunTaskStageName) *TaskStageNode {
	for _, s := range n.taskStages {
		if s.StageName() == stage {
			return s
		}
	}
	return nil
}

// AllPolicyChecks returns every policy check node on the run across all stages.
func (n *RunNode) AllPolicyChecks() []*PolicyCheckNode {
	var all []*PolicyCheckNode
	for _, stage := range n.taskStages {
		all = append(all, stage.PolicyChecks()...)
	}
	return all
}

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
	// A node entering the pending state (ready/approved but not yet admitted to the workspace) projects
	// onto the run as that node's *_queuing status, in the forward flow and in the retry resets alike.
	// Being admitted then projects onto its *_queued status, so the run status names which of the two
	// waits it is in and records the moment it won the slot.
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
	for _, stage := range n.taskStages {
		stage := stage
		// Wire the stage's own listeners on its child checks (verdict aggregation lives on the
		// stage node), then project the stage's status transitions onto the run. The stage
		// pending/running/awaiting_override/completed/errored statuses map to the run's per-stage
		// statuses; canceled/skipped are settled by the run (handleRunTerminated /
		// skipCreatedNodes) and are deliberately not projected.
		stage.init()
		stage.registerListener(string(models.RunTaskStagePending), func() error { return n.handleStagePending(stage) })
		stage.registerListener(string(models.RunTaskStageRunning), func() error { return n.handleStageRunning(stage) })
		stage.registerListener(string(models.RunTaskStageAwaitingOverride), func() error { return n.handleStageAwaitingOverride(stage) })
		stage.registerListener(string(models.RunTaskStageCompleted), func() error { return n.handleStageCompleted(stage) })
		stage.registerListener(string(models.RunTaskStageErrored), func() error { return n.handleStageErrored(stage) })
	}

	// The run node also listens to its own transitions: a run that reaches a final
	// state settles the nodes that never started, and re-entering a queuing phase
	// resets any node skipped by an earlier termination (a retry).
	//
	// planned_and_finished ends the run with its apply phase never entered — a no-change plan, or a
	// speculative run — so the apply and every stage that has not run (the post-plan stage on a
	// no-change plan, plus the pre-apply/post-apply stages, which gate an apply that will never
	// happen) are skipped. Leaving them in created would advertise evaluations that are never coming.
	n.registerListener(string(models.RunPlannedAndFinished), n.skipCreatedNodes)
	// A run that errors skips every node that never started (still created); canceled and discarded
	// additionally cancel any node that was active (started but not final). Discarding a run blocked at
	// a policy gate therefore cancels that gate's stage and the checks it was blocked on: the run is
	// abandoned, so a stage left reporting awaiting_override would be advertising a decision that no
	// longer has anything to decide (its run gate is canceled with it, see RunGateManager). Undiscard
	// revives such a run by re-running the canceled gate (UndiscardRun). On a retry the skipped nodes
	// are reset to created (resetSkippedChildren) so the run can flow through them again.
	n.registerListener(string(models.RunErrored), n.skipCreatedNodes)
	n.registerListener(string(models.RunCanceled), n.handleRunTerminated)
	n.registerListener(string(models.RunDiscarded), n.handleRunTerminated)
	// Every workspace-gated phase the run can re-enter from errored/canceled on a retry. A retry resets
	// the retried node to pending, so the run re-enters at that node's *_queuing status — never at
	// *_queued, which is only reachable by being admitted. resetSkippedChildren is idempotent and a
	// no-op in the forward flow (nothing is skipped there), so registering it on all of them costs
	// nothing and means a retry of any gated node restores the nodes downstream of it.
	for _, queuing := range models.QueuingRunStatuses {
		n.registerListener(string(queuing), n.handleRunQueuing)
	}
	// A run (re-)entering planned must have an apply that again awaits approval: undiscard
	// returns a discarded run to planned, so restore its skipped apply to created. In the
	// normal plan-finished flow the apply is already created/pending, so this is a no-op.
	n.registerListener(string(models.RunPlanned), n.unskipApply)
}

// advance moves the run on from whatever it was waiting for, choosing the next transition from the run's
// own state so no caller has to. It is the single way an external actor unblocks a run, and there are
// three things a caller can have satisfied:
//
//   - the workspace slot, for a gated node sitting at pending: start it. This is the admitter's case.
//     The gated nodes are checked in run order and at most one is ever pending (they are sequential
//     phases), so the first match is the only one. Starting a gated stage is what queues its checks,
//     which is why a stage must not be started before the slot is held.
//   - the run's own creation, for a run at pending: begin the plan phase. This is the create commands'
//     case.
//   - human approval, for a manual run parked at planned: begin the apply phase.
//
// The last two are the same shape — a phase boundary where the run holds no pending node — which is why
// they sit together below. The admitter can never trip either one: it only calls advance when a node is
// pending, so it can neither start a run nor auto-approve an apply.
func (n *RunNode) advance() error {
	for _, stage := range n.taskStages {
		if stage.Status() == models.RunTaskStagePending {
			return stage.SetStatus(models.RunTaskStageRunning)
		}
	}
	if n.plan != nil && n.plan.Status() == models.PlanPending {
		return n.plan.SetStatus(models.PlanQueued)
	}
	if n.apply != nil && n.apply.Status() == models.ApplyPending {
		return n.apply.SetStatus(models.ApplyQueued)
	}

	switch n.status {
	case models.RunPending:
		return n.startPlanPhase()
	case models.RunPlanned:
		return n.startApplyPhase()
	}

	// Nothing is waiting: advancing is a no-op rather than an error, so a redundant call (e.g. a
	// redelivered work item) is harmless.
	return nil
}

// startPlanPhase begins the plan phase of a freshly-created run. Its first node depends on the run's
// shape: a run with a pre-plan policy stage readies that stage (so the run reports pre_plan_queuing and
// waits for the slot, with the plan left created until the stage clears), and a run without one readies
// the plan itself (reporting plan_queuing). A stage is always created with at least one check (see run
// create.go), so a created stage has checks to queue once started.
//
// This mirrors startApplyPhase: both ready the first node of a phase, and both leave the node that the
// phase's gate protects in created.
//
// The stage/plan are driven normally (not silently) so their projections onto the run fire, taking the
// run straight to pre_plan_queuing or plan_queuing without a spurious intermediate status.
func (n *RunNode) startPlanPhase() error {
	if preStage := n.TaskStage(models.RunTaskStageNamePrePlan); preStage != nil && preStage.Status() == models.RunTaskStageCreated {
		return preStage.SetStatus(models.RunTaskStagePending)
	}
	if n.plan != nil && n.plan.Status() == models.PlanCreated {
		return n.plan.SetStatus(models.PlanPending)
	}
	return nil
}

// handleRunQueuing reacts to the run entering one of its *_queuing phases — the phase's node is ready
// but not yet admitted to the workspace. In the forward flow the node was just readied, so there is
// nothing more to do. A *_queuing status is also where a retry re-enters (the retried node resets to
// pending): as the run re-enters an active phase, restore any nodes that were skipped when it
// previously terminated so the retried run can flow through its later stages again — a no-op in the
// forward flow, where nothing is skipped.
func (n *RunNode) handleRunQueuing() error {
	return n.resetSkippedChildren()
}

// handleStagePending projects a workspace-gated stage that is ready but unadmitted onto the run as
// that stage's *_queuing status. The ungated post-plan stage never reaches pending (it is driven
// straight to running on the plan's slot), so it has no case here and no *_queuing run status.
func (n *RunNode) handleStagePending(stage *TaskStageNode) error {
	switch stage.StageName() {
	case models.RunTaskStageNamePrePlan:
		return n.SetStatus(models.RunPrePlanQueuing)
	case models.RunTaskStageNamePreApply:
		return n.SetStatus(models.RunPreApplyQueuing)
	}
	return nil
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

// skipUnstartedPlan moves a plan that never started (still in created) to skipped when the run
// reaches a final state before the plan could run — a pre-plan policy gate errored, the run was
// canceled, or it was discarded while awaiting a pre-plan override. Like the apply, the run drives
// the plan here, so it is set silently to avoid re-projecting onto the run; the change is still
// recorded for persistence. A plan that already progressed (pending/running/finished/...) is left
// untouched — its own status conveys the outcome.
func (n *RunNode) skipUnstartedPlan() error {
	if n.plan == nil || n.plan.Status() != models.PlanCreated {
		return nil
	}
	return n.plan.setStatusSilently(models.PlanSkipped)
}

// handleRunTerminated fires when the run is canceled or discarded: it cancels any node that was
// active (started but not yet final) and skips any node that never started, settling every child to a
// terminal state consistent with the run outcome without requiring callers to drive each child
// explicitly. A retry later resets the skipped nodes (resetSkippedChildren), and an undiscard re-runs
// the canceled gate (UndiscardRun). An errored run uses skipCreatedNodes alone: the node that failed
// already carries the outcome.
func (n *RunNode) handleRunTerminated() error {
	if err := n.cancelActiveNodes(); err != nil {
		return err
	}
	return n.skipCreatedNodes()
}

// skipCreatedNodes marks every task stage that never started (still in created) as skipped when the
// run reaches a terminal state (planned_and_finished, errored, canceled, or discarded). Each stage
// cascades the skip to its own created checks. A created node was never evaluated, so skipped records
// that it will not run for this outcome — canceled would misrepresent a node that never started. On a
// retry these skipped nodes are reset to created (resetSkippedChildren) so the run can flow through
// them again.
func (n *RunNode) skipCreatedNodes() error {
	if err := n.skipUnstartedPlan(); err != nil {
		return err
	}
	if err := n.skipUnstartedApply(); err != nil {
		return err
	}
	for _, stage := range n.taskStages {
		if stage.Status() == models.RunTaskStageCreated {
			if err := stage.SetStatus(models.RunTaskStageSkipped); err != nil {
				return err
			}
		}
	}
	return nil
}

// cancelActiveNodes transitions every task stage that was active (started but not yet final) to
// canceled when the run is canceled or discarded — including a stage awaiting an override, which is
// exactly the state a discard abandons. Each stage cascades to its own checks: active checks are
// canceled and created checks are skipped. Stages that never started (still created) are settled
// by skipCreatedNodes instead, so they are skipped rather than canceled.
func (n *RunNode) cancelActiveNodes() error {
	for _, stage := range n.taskStages {
		if stage.Status() == models.RunTaskStageCreated || stage.Status().IsFinalStatus() {
			continue
		}
		if err := stage.SetStatus(models.RunTaskStageCanceled); err != nil {
			return err
		}
	}
	return nil
}

// resetSkippedChildren returns the run's skipped children — its plan, its task stages (and the checks
// they own), and its apply — to created. It runs when the run re-enters an active phase after having
// terminated (a retry) or is undiscarded: a node is skipped only when it never ran because an earlier
// node failed or the run was canceled/discarded, so once the run is revived past that point the node
// must run again. created carries no listeners, so the reset restores node state without projecting
// onto the run, and in the forward flow (nothing skipped) it is a no-op.
func (n *RunNode) resetSkippedChildren() error {
	if err := n.unskipPlan(); err != nil {
		return err
	}
	if err := n.unskipApply(); err != nil {
		return err
	}
	for _, stage := range n.taskStages {
		if err := stage.resetIfSkipped(); err != nil {
			return err
		}
	}
	return nil
}

// unskipApply returns a skipped apply to created when the run leaves a final state — via
// the plan-retry reset (run back to plan_queuing) or an undiscard (run back to planned). The
// apply again awaits its outcome, and downstream flows (auto-apply advancement, the
// start-apply command) expect a not-yet-started apply to be in created.
func (n *RunNode) unskipApply() error {
	if n.apply == nil || n.apply.Status() != models.ApplySkipped {
		return nil
	}
	return n.apply.setStatusSilently(models.ApplyCreated)
}

// unskipPlan returns a skipped plan to created when the run leaves a final state — a stage-retry reset
// or an undiscard — so the plan can run once its gating stage clears again. It is a no-op when the
// plan is not skipped.
func (n *RunNode) unskipPlan() error {
	if n.plan == nil || n.plan.Status() != models.PlanSkipped {
		return nil
	}
	return n.plan.setStatusSilently(models.PlanCreated)
}

// handlePlanPending projects a pending plan onto the run: the plan is ready and waiting to be admitted
// to the workspace, so the run is plan_queuing. This covers the forward flow (pending or
// pre_plan_completed -> plan_queuing) and the plan retry (errored/canceled -> plan_queuing) alike. The
// run leaves plan_queuing only when the plan is admitted, which is the queued projection below.
func (n *RunNode) handlePlanPending() error {
	return n.SetStatus(models.RunPlanQueuing)
}

// handleApplyPending projects a pending apply onto the run: the apply is approved and waiting to be
// admitted, so the run is apply_queuing. This covers the manual start-apply flow (planned ->
// apply_queuing), the pre-apply stage clearing (pre_apply_completed -> apply_queuing) and the apply
// retry (errored/canceled -> apply_queuing) alike.
func (n *RunNode) handleApplyPending() error {
	return n.SetStatus(models.RunApplyQueuing)
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

// handlePlanQueued projects an admitted plan onto the run: the run has won the workspace slot and its
// plan job exists, so it is now waiting only for a runner. This is the transition that records when the
// run stopped queuing for the workspace, which is why the two waits are separate run statuses.
func (n *RunNode) handlePlanQueued() error {
	return n.SetStatus(models.RunPlanQueued)
}

func (n *RunNode) handlePlanRunning() error {
	return n.SetStatus(models.RunPlanning)
}

func (n *RunNode) handlePlanSucceeded() error {
	postStage := n.TaskStage(models.RunTaskStageNamePostPlan)
	// A plan that produced no changes has nothing to apply, so a run that could have applied one is
	// finished here. Policy is not bypassed by finishing: there is no apply left for it to gate, and
	// the stages that will now never run — the post-plan stage and the apply-phase stages — are settled
	// as skipped when the run reaches planned_and_finished (skipCreatedNodes), each cascading the skip
	// to its own checks.
	//
	// A speculative run is the exception and still evaluates, because it never had an apply to gate in
	// the first place — its whole output is the verdict.
	if !n.plan.hasChanges && n.apply != nil {
		return n.SetStatus(models.RunPlannedAndFinished)
	}
	// The plan produced changes, or produced none on a speculative run. A run with a post-plan stage
	// must clear policy before it can proceed — this holds even for a speculative run with no apply,
	// whose policy verdict is still meaningful. The stage moves to running and queues its checks (their
	// policy-eval jobs are created by the job-creation transformer), projecting the run onto
	// post_plan_running. Advancing toward the apply, or finishing a speculative run, is deferred until
	// the stage completes (handleStageCompleted), so policy is never bypassed.
	if postStage != nil {
		return postStage.SetStatus(models.RunTaskStageRunning)
	}
	// No policy checks: advance toward the apply, or finish a speculative run.
	return n.projectPlanFinishedToApply()
}

// projectPlanFinishedToApply advances a run whose plan finished with changes (and whose
// post-plan policy, if any, has cleared). A speculative run has no apply, so there is
// nothing to apply and the run finishes. Otherwise a manual run stops at planned with the apply left
// in created until it is approved.
//
// An auto-apply run is already approved, so it does not enter planned at all and begins its apply
// phase directly (landing on pre_apply_queuing or apply_queuing). planned means "the plan is done and
// the run is waiting on a human": it is what the start-apply and discard commands require, what
// releases the workspace slot, and what the CLI's confirm prompt keys off. A pre-approved run waits
// for none of that, so passing it through planned would report a state the run was never in.
//
// This runs after the plan finishes (no stage) or after the post-plan stage clears; the two paths
// behave the same way, so an auto-apply run's history never shows planned regardless of whether it
// has a post-plan stage.
func (n *RunNode) projectPlanFinishedToApply() error {
	if n.apply == nil {
		// Speculative run: nothing to apply, so the run finishes.
		return n.SetStatus(models.RunPlannedAndFinished)
	}
	if n.apply.autoApply {
		// A pre-apply stage takes precedence over readying the apply — see startApplyPhase.
		return n.startApplyPhase()
	}
	return n.SetStatus(models.RunPlanned)
}

// startApplyPhase begins the run's apply phase for an approved (or auto-approved) apply. A run with a
// pre-apply policy stage readies that stage instead of the apply: the stage is workspace-gated, so the
// run waits at pre_apply_queuing until admitted, and the apply stays created until the stage clears.
// This is the same shape as the pre-plan stage gating the plan, and it is what keeps the apply from
// being admitted past an unsatisfied pre-apply gate — the admitter only ever acts on a pending node, so
// the apply must not become pending until its gate has cleared.
func (n *RunNode) startApplyPhase() error {
	// A run revived past an earlier failure (a node retry, or an undiscard) can still carry nodes that
	// were skipped when it terminated, so restore them before choosing the phase's first node: a
	// skipped pre-apply stage would otherwise be passed over below and its gate silently bypassed, and
	// a skipped apply would leave the run parked with nothing left to advance it. An auto-apply run no
	// longer passes through planned, whose listener previously did this; it is a no-op in the forward
	// flow, where nothing is skipped.
	if err := n.resetSkippedChildren(); err != nil {
		return err
	}
	if preApply := n.TaskStage(models.RunTaskStageNamePreApply); preApply != nil && preApply.Status() == models.RunTaskStageCreated {
		return preApply.SetStatus(models.RunTaskStagePending)
	}
	// No pre-apply gate: ready the apply. Driven normally so the projection onto apply_queuing fires.
	if n.apply.Status() == models.ApplyCreated {
		return n.apply.SetStatus(models.ApplyPending)
	}
	return nil
}

// handleStageRunning projects a stage entering running onto the run: pre_plan_running,
// post_plan_running, pre_apply_running or post_apply_running by stage. In the forward flow a gated
// stage was just admitted (its *_queuing status already fired), so this records the move to
// *_running. It also restores any nodes that were skipped downstream of an earlier failure (the later
// stages and the apply) so a retried run can flow through them again — a no-op in the forward flow,
// where nothing is skipped.
func (n *RunNode) handleStageRunning(stage *TaskStageNode) error {
	if err := n.resetSkippedChildren(); err != nil {
		return err
	}
	switch stage.StageName() {
	case models.RunTaskStageNamePrePlan:
		return n.SetStatus(models.RunPrePlanRunning)
	case models.RunTaskStageNamePostPlan:
		return n.SetStatus(models.RunPostPlanRunning)
	case models.RunTaskStageNamePreApply:
		return n.SetStatus(models.RunPreApplyRunning)
	case models.RunTaskStageNamePostApply:
		return n.SetStatus(models.RunPostApplyRunning)
	}
	return nil
}

// handleStageAwaitingOverride projects a stage blocked on a human override onto the run as that stage's
// *_awaiting_decision status. The run releases its workspace slot at each of these (see the workspace
// lock manager), so clearing the gate returns the run to the next gated node's *_queuing status to be
// re-admitted. There is no post_apply case: a post-apply policy is restricted to advisory enforcement
// (models.Policy.Validate), so a post-apply check can never soft-fail and reach awaiting_override in
// the first place.
func (n *RunNode) handleStageAwaitingOverride(stage *TaskStageNode) error {
	switch stage.StageName() {
	case models.RunTaskStageNamePrePlan:
		return n.SetStatus(models.RunPrePlanAwaitingDecision)
	case models.RunTaskStageNamePostPlan:
		return n.SetStatus(models.RunPostPlanAwaitingDecision)
	case models.RunTaskStageNamePreApply:
		return n.SetStatus(models.RunPreApplyAwaitingDecision)
	}
	return nil
}

// handleStageCompleted projects a stage whose policy has cleared onto the run and advances the run:
//   - pre_plan: the run moves to pre_plan_completed and the still-created plan is readied, projecting
//     the run onto plan_queuing so the admitter acquires the workspace and queues the plan;
//   - post_plan: the run moves to post_plan_completed and advances toward the apply (manual ->
//     planned, auto-apply -> the apply phase, speculative -> planned_and_finished);
//   - pre_apply: the run moves to pre_apply_completed and the still-created apply is readied,
//     projecting onto apply_queuing for admission;
//   - post_apply: the run moves to post_apply_completed and then straight to applied. Both are
//     recorded status changes, but post_apply_completed is never the run's resting status — this
//     handler always advances past it in the same pass, which is what makes it safe for the status to
//     be absent from the GraphQL and protobuf RunStatus enums despite living in models.AllRunStatuses.
func (n *RunNode) handleStageCompleted(stage *TaskStageNode) error {
	switch stage.StageName() {
	case models.RunTaskStageNamePrePlan:
		if err := n.SetStatus(models.RunPrePlanCompleted); err != nil {
			return err
		}
		// Pre-plan policy is satisfied: ready the plan. The stage held the slot only for its own
		// evaluation, so the plan takes the same waiting-for-admission path a run with no pre-plan
		// stage takes. Driven normally (not silently) so the projection onto plan_queuing fires.
		if n.plan != nil && n.plan.Status() == models.PlanCreated {
			return n.plan.SetStatus(models.PlanPending)
		}
		return nil
	case models.RunTaskStageNamePostPlan:
		if err := n.SetStatus(models.RunPostPlanCompleted); err != nil {
			return err
		}
		return n.projectPlanFinishedToApply()
	case models.RunTaskStageNamePreApply:
		if err := n.SetStatus(models.RunPreApplyCompleted); err != nil {
			return err
		}
		// Pre-apply policy is satisfied: the apply may now become pending, which is what allows the
		// admitter to queue it. Until this point it was deliberately left created.
		if n.apply != nil && n.apply.Status() == models.ApplyCreated {
			return n.apply.SetStatus(models.ApplyPending)
		}
		return nil
	case models.RunTaskStageNamePostApply:
		if err := n.SetStatus(models.RunPostApplyCompleted); err != nil {
			return err
		}
		// The apply already succeeded (this stage only ever starts from handleApplySucceeded), and
		// post-apply's advisory-only enforcement means a failure here never blocks the run, so the run
		// always finishes.
		return n.SetStatus(models.RunApplied)
	}
	return nil
}

// handleStageErrored projects a stage that failed a hard gate onto the run as errored (the run
// node's own terminal listener skips the never-started apply). The plan never starts on a pre-plan
// failure.
func (n *RunNode) handleStageErrored(_ *TaskStageNode) error {
	return n.SetStatus(models.RunErrored)
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

// handleApplyQueued projects an admitted apply onto the run, mirroring handlePlanQueued: the run holds
// the workspace slot and its apply job exists, so it is waiting only for a runner.
func (n *RunNode) handleApplyQueued() error {
	return n.SetStatus(models.RunApplyQueued)
}

func (n *RunNode) handleApplyRunning() error {
	return n.SetStatus(models.RunApplying)
}

func (n *RunNode) handleApplySucceeded() error {
	postApplyStage := n.TaskStage(models.RunTaskStageNamePostApply)
	// A run with a post-apply stage must run it before settling on applied — the stage moves to
	// running and queues its checks (their policy-eval jobs are created by the job-creation
	// transformer), projecting the run onto post_apply_running. Reaching applied is deferred until the
	// stage completes (handleStageCompleted), mirroring how a post-plan stage defers a finished plan.
	if postApplyStage != nil {
		return postApplyStage.SetStatus(models.RunTaskStageRunning)
	}
	// No post-apply policy: the run finishes here.
	return n.SetStatus(models.RunApplied)
}

func (n *RunNode) handleApplyErrored() error {
	return n.SetStatus(models.RunErrored)
}

func (n *RunNode) handleApplyCanceled() error {
	return n.SetStatus(models.RunCanceled)
}
