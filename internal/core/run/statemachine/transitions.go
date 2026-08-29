package statemachine

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// runTransitions lists the statuses each run status may legally transition to. The run status is a
// projection of its task stage / plan / apply nodes, so these are the transitions the node listeners
// can drive.
//
// A workspace-gated node passes through pending (ready, waiting for the workspace slot) before being
// admitted, and the run names both halves of that wait: *_queuing while unadmitted, *_queued once the
// job exists and only a runner is missing. So each gated node contributes a queuing -> queued edge
// here (a gated stage has no *_queued: an admitted stage goes straight to *_running, because the
// job-created wait belongs to its child checks).
//
// The terminal applied/planned_and_finished statuses are absorbing; errored/canceled additionally
// permit the retry resets (a node being retried moves back to pending, so the run lands on that
// node's *_queuing status). The three *_awaiting_decision statuses permit the same reset, because a
// soft-failed check can be retried instead of decided. discarded is a run-level outcome (not driven by
// a child node), reachable only from planned or a policy gate, and reversible: undiscard revives the
// run at the point the discard abandoned it — which is why a check is not retryable while the run is
// discarded (see RetryRunNode), the undiscard is the retry.
var runTransitions = map[models.RunStatus][]models.RunStatus{
	// A freshly-created run starts (RunNode.startPlanPhase) into one of two states: a run with a
	// pre-plan policy stage readies that stage and lands on pre_plan_queuing to await admission; a run
	// with no pre-plan stage readies its plan instead and lands on plan_queuing.
	models.RunPending: {models.RunPrePlanQueuing, models.RunPlanQueuing, models.RunCanceled},
	// The pre-plan policy stage. It is workspace-gated, so the run waits at pre_plan_queuing until the
	// slot is acquired and the stage starts (pre_plan_running). The stage then clears — readying the
	// plan, so the run moves to plan_queuing to await admission for the plan itself — or fails a hard
	// gate (errored, the plan never runs). pre_plan_awaiting_decision is entered when all checks have
	// evaluated and at least one requires human approval before the plan; the run releases the slot
	// there, so clearing the gate returns it to plan_queuing to be re-admitted.
	models.RunPrePlanQueuing: {models.RunPrePlanRunning, models.RunCanceled},
	models.RunPrePlanRunning: {models.RunPrePlanCompleted, models.RunPrePlanAwaitingDecision, models.RunErrored, models.RunCanceled},
	// A pre-plan gate awaiting an override can also be discarded: declining the override abandons the
	// run (the same way declining a post-plan override does). Retrying the soft-failed check instead of
	// deciding its gate returns the run to pre_plan_queuing: the check resets to pending, and the stage
	// is gated so it must win the slot again before re-evaluating.
	models.RunPrePlanAwaitingDecision: {
		models.RunPrePlanCompleted, models.RunPrePlanQueuing,
		models.RunErrored, models.RunCanceled, models.RunDiscarded,
	},
	models.RunPrePlanCompleted: {models.RunPlanQueuing, models.RunCanceled},
	// The plan: admitted (queuing -> queued, the run won the slot and its job was created), then
	// claimed by a runner (queued -> planning). A finished plan lands on planned (a manual run, which
	// waits there for approval), planned_and_finished (nothing to apply), post_plan_running (a
	// post-plan stage must clear first) or, for an auto-apply run with no post-plan stage, straight on
	// its apply phase: an auto-apply run is already approved, so it never enters planned.
	models.RunPlanQueuing: {models.RunPlanQueued, models.RunErrored, models.RunCanceled},
	models.RunPlanQueued:  {models.RunPlanning, models.RunErrored, models.RunCanceled},
	models.RunPlanning: {
		models.RunPlanned, models.RunPlannedAndFinished, models.RunPostPlanRunning,
		models.RunPreApplyQueuing, models.RunApplyQueuing,
		models.RunErrored, models.RunCanceled,
	},
	// The post-plan policy stage brackets the run between planning and either planned / the apply
	// phase (policy cleared, run has an apply), planned_and_finished (policy cleared on a speculative
	// run with no apply) or errored (policy failed). It is not separately gated — it inherits the
	// slot the plan already holds — so it has no *_queuing status. post_plan_running covers the
	// automated evaluation phase; post_plan_awaiting_decision is entered when all checks have
	// evaluated and at least one requires human approval. Discard is only reachable from
	// post_plan_awaiting_decision (the "waiting on human" state), not from post_plan_running. Retrying
	// the soft-failed check instead of deciding its gate returns the run to post_plan_running directly:
	// the stage is ungated, so there is no *_queuing status to wait in.
	models.RunPostPlanRunning: {models.RunPostPlanCompleted, models.RunPostPlanAwaitingDecision, models.RunErrored, models.RunCanceled},
	models.RunPostPlanAwaitingDecision: {
		models.RunPostPlanCompleted, models.RunPostPlanRunning,
		models.RunErrored, models.RunCanceled, models.RunDiscarded,
	},
	models.RunPostPlanCompleted: {models.RunPlanned, models.RunPreApplyQueuing, models.RunApplyQueuing, models.RunPlannedAndFinished, models.RunCanceled},
	models.RunPlanned:           {models.RunPreApplyQueuing, models.RunApplyQueuing, models.RunCanceled, models.RunDiscarded},
	// The pre-apply policy stage mirrors the pre-plan stage: workspace-gated (it evaluates the plan it
	// is about to apply, so another run must not apply underneath it), releasing the slot while
	// awaiting a decision, readying the apply once it clears, and returning to pre_apply_queuing to be
	// re-admitted when its soft-failed check is retried instead of decided.
	models.RunPreApplyQueuing: {models.RunPreApplyRunning, models.RunCanceled},
	models.RunPreApplyRunning: {models.RunPreApplyCompleted, models.RunPreApplyAwaitingDecision, models.RunErrored, models.RunCanceled},
	models.RunPreApplyAwaitingDecision: {
		models.RunPreApplyCompleted, models.RunPreApplyQueuing,
		models.RunErrored, models.RunCanceled, models.RunDiscarded,
	},
	models.RunPreApplyCompleted: {models.RunApplyQueuing, models.RunCanceled},
	// The apply, mirroring the plan: admitted (queuing -> queued), then claimed by a runner.
	models.RunApplyQueuing: {models.RunApplyQueued, models.RunErrored, models.RunCanceled},
	models.RunApplyQueued:  {models.RunApplying, models.RunErrored, models.RunCanceled},
	// A finished apply lands on applied directly (no post-apply stage) or on post_apply_running (a
	// post-apply stage must clear first). Mirrors RunPlanning's edge onto RunPostPlanRunning.
	models.RunApplying: {models.RunApplied, models.RunPostApplyRunning, models.RunErrored, models.RunCanceled},
	// The post-apply policy stage brackets the run between applying and applied, mirroring the
	// post-plan stage: not separately gated (it inherits the apply's slot), so no *_queuing status.
	// post_apply_completed is transient — the stage's completed handler sets it and then immediately
	// advances the run to applied within the same state-machine pass (see handleStageCompleted), so a
	// persisted run is never observed resting there. There is deliberately no
	// post_apply_awaiting_decision: a post-apply policy is restricted to advisory enforcement
	// (models.Policy.Validate), so no post-apply check can ever soft-fail and block on a human
	// override.
	models.RunPostApplyRunning:   {models.RunPostApplyCompleted, models.RunErrored, models.RunCanceled},
	models.RunPostApplyCompleted: {models.RunApplied, models.RunCanceled},
	// A retry returns an errored/canceled run to the *_queuing status of whichever node was retried —
	// the node resets to pending, so the run is waiting for the slot again, never straight to *_queued —
	// or to post_plan_running / post_apply_running, the ungated stages that re-run directly on the plan's
	// or apply's slot.
	models.RunErrored: {
		models.RunPlanQueuing, models.RunApplyQueuing,
		models.RunPrePlanQueuing, models.RunPostPlanRunning, models.RunPreApplyQueuing, models.RunPostApplyRunning,
	},
	models.RunCanceled: {
		models.RunPlanQueuing, models.RunApplyQueuing,
		models.RunPrePlanQueuing, models.RunPostPlanRunning, models.RunPreApplyQueuing, models.RunPostApplyRunning,
	},
	// discarded is otherwise terminal, but a discard may be reversed (undiscard): a run discarded from
	// planned returns to planned, and one discarded at a policy gate re-runs that gate — the discard
	// canceled it, so the undiscard restarts its stage and lands on the same status a retry of that
	// stage would (its *_queuing status when gated, post_plan_running when not). There is deliberately
	// no edge back to a *_awaiting_decision status: a discard leaves no gate awaiting a decision to
	// return to.
	models.RunDiscarded: {
		models.RunPlanned, models.RunPrePlanQueuing, models.RunPostPlanRunning, models.RunPreApplyQueuing,
	},
}

// planTransitions lists the statuses each plan status may legally transition to.
// finished is absorbing; errored/canceled permit the retry reset back to pending.
// skipped permits the reset back to created: a plan that never ran (the run terminated at a pre-plan
// gate, or was discarded/canceled before the plan started) is skipped, and reviving the run (a stage
// retry or an undiscard) returns it to created so the run can flow through it again — mirroring the
// apply's skipped reset.
var planTransitions = map[models.PlanStatus][]models.PlanStatus{
	models.PlanCreated:  {models.PlanPending, models.PlanCanceled, models.PlanSkipped},
	models.PlanPending:  {models.PlanQueued, models.PlanCanceled},
	models.PlanQueued:   {models.PlanRunning, models.PlanErrored, models.PlanCanceled},
	models.PlanRunning:  {models.PlanFinished, models.PlanErrored, models.PlanCanceled},
	models.PlanErrored:  {models.PlanPending},
	models.PlanCanceled: {models.PlanPending},
	models.PlanSkipped:  {models.PlanCreated},
}

// applyTransitions lists the statuses each apply status may legally transition to.
// finished is absorbing; errored/canceled permit the retry reset back to pending.
// skipped permits the reset back to created: retrying the plan of an errored/canceled
// run returns the run to pending, and the apply must again await the plan's outcome.
var applyTransitions = map[models.ApplyStatus][]models.ApplyStatus{
	models.ApplyCreated:  {models.ApplyPending, models.ApplyCanceled, models.ApplySkipped},
	models.ApplyPending:  {models.ApplyQueued, models.ApplyCanceled},
	models.ApplyQueued:   {models.ApplyRunning, models.ApplyErrored, models.ApplyCanceled},
	models.ApplyRunning:  {models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled},
	models.ApplyErrored:  {models.ApplyPending},
	models.ApplyCanceled: {models.ApplyPending},
	models.ApplySkipped:  {models.ApplyCreated},
}

// policyCheckTransitions lists the statuses each policy check status may legally
// transition to. The check passes through queued (its policy-eval job has been created,
// awaiting a runner) before running. passed and overridden are absorbing; soft_failed
// is a blocked state that resolves to overridden (a manual override clears it), errored (a
// gate approver rejected the gate) or canceled.
//
// errored, canceled and soft_failed permit the retry reset back to pending, which is the only edge a
// retry needs: the check's stage listens for pending and restarts itself, re-admitting first if it is
// workspace-gated. The reset does not go straight to queued because queueing a check creates its
// policy-eval job immediately, and a retried check on a gated stage must wait for the slot first.
// soft_failed is retryable because a soft failure is a verdict on the policies as they stood, and
// fixing the policy (or bumping its version) makes re-evaluating the right resolution rather than
// overriding a failure that no longer applies. Retrying deletes the check's gate (RunGateManager), so
// any approvals it had collected are discarded along with the verdict they were approving.
//
// skipped permits the reset back to created: a check that never ran is skipped when an earlier
// node fails or the run is canceled, and retrying an earlier node returns it to created so the
// run can flow through it again.
var policyCheckTransitions = map[models.PolicyCheckStatus][]models.PolicyCheckStatus{
	models.PolicyCheckCreated: {models.PolicyCheckQueued, models.PolicyCheckCanceled, models.PolicyCheckSkipped},
	models.PolicyCheckPending: {models.PolicyCheckQueued, models.PolicyCheckCanceled, models.PolicyCheckSkipped},
	models.PolicyCheckQueued:  {models.PolicyCheckRunning, models.PolicyCheckCanceled},
	models.PolicyCheckRunning: {models.PolicyCheckPassed, models.PolicyCheckSoftFailed, models.PolicyCheckErrored, models.PolicyCheckCanceled},
	models.PolicyCheckSoftFailed: {
		models.PolicyCheckOverridden, models.PolicyCheckPending,
		models.PolicyCheckErrored, models.PolicyCheckCanceled,
	},
	models.PolicyCheckErrored:  {models.PolicyCheckPending},
	models.PolicyCheckCanceled: {models.PolicyCheckPending},
	models.PolicyCheckSkipped:  {models.PolicyCheckCreated},
}

// taskStageTransitions lists the statuses each task stage status may legally transition to. A stage
// starts created and is readied to pending when the run reaches it. A workspace-gated stage (pre_plan,
// pre_apply) waits there until the admitter acquires the slot and moves it to running, which queues
// its checks; an ungated stage (post_plan) passes straight from created to running on the slot the
// plan already holds. The stage then settles to completed (all checks cleared), awaiting_override (a
// check blocks on human override), or errored (a check failed a hard gate). skipped is reached when
// the stage never runs (e.g. a post-plan stage on a no-change plan, or any stage that never started
// when an earlier node fails or the run is canceled).
//
// completed is absorbing; errored, canceled and awaiting_override permit the retry reset back to pending
// (a gated stage, so it is re-admitted before running again) or straight to running (an ungated stage,
// which needs no admission). Both edges are legal on each of them because this map is keyed by status
// alone; which one a retry uses is decided by the stage itself, in handleCheckPending. awaiting_override
// carries the pair because a soft-failed check is retryable, which restarts the stage that was blocked
// on it. skipped permits the reset back to created — retrying an earlier node restores a never-started
// stage so the run can flow through it again — mirroring the check-level retry resets.
var taskStageTransitions = map[models.RunTaskStageStatus][]models.RunTaskStageStatus{
	models.RunTaskStageCreated: {models.RunTaskStagePending, models.RunTaskStageRunning, models.RunTaskStageSkipped, models.RunTaskStageCanceled},
	models.RunTaskStagePending: {models.RunTaskStageRunning, models.RunTaskStageSkipped, models.RunTaskStageCanceled},
	models.RunTaskStageRunning: {models.RunTaskStageCompleted, models.RunTaskStageAwaitingOverride, models.RunTaskStageErrored, models.RunTaskStageCanceled},
	models.RunTaskStageAwaitingOverride: {
		models.RunTaskStageCompleted, models.RunTaskStagePending, models.RunTaskStageRunning,
		models.RunTaskStageErrored, models.RunTaskStageCanceled,
	},
	models.RunTaskStageErrored:  {models.RunTaskStagePending, models.RunTaskStageRunning},
	models.RunTaskStageCanceled: {models.RunTaskStagePending, models.RunTaskStageRunning},
	models.RunTaskStageSkipped:  {models.RunTaskStageCreated},
}

// canTransitionTo reports whether a node may move from its current status to next
// according to the given transition map.
func canTransitionTo[S comparable](transitions map[S][]S, from, next S) bool {
	for _, allowed := range transitions[from] {
		if allowed == next {
			return true
		}
	}
	return false
}
