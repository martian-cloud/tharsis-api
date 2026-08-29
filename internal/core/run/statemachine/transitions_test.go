package statemachine

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// nonDiscardableStatuses are the run statuses that may neither be discarded nor be the target of an
// undiscard. Discard is only reachable from planned or a policy gate's awaiting-decision status.
var nonDiscardableStatuses = []models.RunStatus{
	models.RunPending,
	models.RunPrePlanQueuing, models.RunPrePlanRunning, models.RunPrePlanCompleted,
	models.RunPlanQueuing, models.RunPlanQueued, models.RunPlanning,
	models.RunPostPlanRunning,
	models.RunPreApplyQueuing, models.RunPreApplyRunning, models.RunPreApplyCompleted,
	models.RunApplyQueuing, models.RunApplyQueued, models.RunApplying,
	models.RunPostApplyRunning, models.RunApplied,
	models.RunErrored, models.RunCanceled,
}

// undiscardTargets are the statuses an undiscard may return a discarded run to besides planned: a
// discard cancels the gate the run was blocked at, and undiscarding re-runs that gate, so the run
// re-enters where a retry of the stage would put it — its *_queuing status when the stage is
// workspace-gated, post_plan_running when it is not.
var undiscardTargets = []models.RunStatus{
	models.RunPrePlanQueuing, models.RunPostPlanRunning, models.RunPreApplyQueuing,
}

func TestRunTransitions_Discard(t *testing.T) {
	// A planned run may be discarded; no other status may transition to discarded.
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunDiscarded), "planned -> discarded should be allowed")

	for _, s := range nonDiscardableStatuses {
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunDiscarded), "%s -> discarded should not be allowed", s)
	}

	// discarded is reversible (undiscard): back to planned, or to the re-run of the gate the discard
	// canceled. Nothing else — in particular no *_awaiting_decision status, because a discard leaves no
	// gate awaiting a decision to return to.
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPlanned), "discarded -> planned (undiscard) should be allowed")
	for _, s := range undiscardTargets {
		assert.Truef(t, canTransitionTo(runTransitions, models.RunDiscarded, s), "discarded -> %s (gate re-run) should be allowed", s)
	}
	for _, s := range []models.RunStatus{
		models.RunPrePlanAwaitingDecision, models.RunPostPlanAwaitingDecision, models.RunPreApplyAwaitingDecision,
	} {
		assert.Falsef(t, canTransitionTo(runTransitions, models.RunDiscarded, s), "discarded -> %s should not be allowed", s)
	}
	for _, s := range nonDiscardableStatuses {
		if slices.Contains(undiscardTargets, s) {
			continue
		}
		assert.Falsef(t, canTransitionTo(runTransitions, models.RunDiscarded, s), "discarded -> %s should not be allowed", s)
	}
}

func TestRunTransitions_QueuingPhases(t *testing.T) {
	// A run with no pre-plan stage readies its plan at start and lands on plan_queuing; one with a
	// pre-plan stage readies that stage instead and lands on pre_plan_queuing. Either way it starts by
	// waiting for the workspace, never straight on a *_queued status.
	assert.True(t, canTransitionTo(runTransitions, models.RunPending, models.RunPlanQueuing), "pending -> plan_queuing should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPending, models.RunPrePlanQueuing), "pending -> pre_plan_queuing should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPending, models.RunPlanQueued), "pending -> plan_queued should not skip admission")

	// Being admitted is its own transition on each gated node: queuing (waiting for the slot) -> queued
	// (holding the slot, waiting for a runner) -> the node starts.
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanQueuing, models.RunPlanQueued), "plan_queuing -> plan_queued (admitted) should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanQueued, models.RunPlanning), "plan_queued -> planning should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunApplyQueuing, models.RunApplyQueued), "apply_queuing -> apply_queued (admitted) should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunApplyQueued, models.RunApplying), "apply_queued -> applying should be allowed")

	// A node must not start without being admitted first, which is what the queuing status records.
	assert.False(t, canTransitionTo(runTransitions, models.RunPlanQueuing, models.RunPlanning), "plan_queuing -> planning should not skip plan_queued")
	assert.False(t, canTransitionTo(runTransitions, models.RunApplyQueuing, models.RunApplying), "apply_queuing -> applying should not skip apply_queued")
	assert.False(t, canTransitionTo(runTransitions, models.RunPending, models.RunPlanning), "pending -> planning should not skip the plan queue")
	assert.False(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunApplying), "planned -> applying should not skip the apply queue")

	// Admission is one-way: a run that won the slot does not go back to waiting for it without first
	// terminating (which is the retry path asserted below).
	assert.False(t, canTransitionTo(runTransitions, models.RunPlanQueued, models.RunPlanQueuing), "plan_queued -> plan_queuing should not be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunApplyQueued, models.RunApplyQueuing), "apply_queued -> apply_queuing should not be allowed")

	// Every queuing phase can still be canceled while it waits.
	for _, s := range models.QueuingRunStatuses {
		assert.Truef(t, canTransitionTo(runTransitions, s, models.RunCanceled), "%s -> canceled should be allowed", s)
	}

	// An approved apply readies either the apply itself or a pre-apply stage; both wait for the slot.
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunApplyQueuing), "planned -> apply_queuing should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunPreApplyQueuing), "planned -> pre_apply_queuing should be allowed")

	// A retry resets the retried node to pending, so it lands on that node's queuing status — never on a
	// *_queued status (which would mean it still held the slot it gave up when it terminated), and never
	// back on pending or planned.
	for _, s := range []models.RunStatus{models.RunErrored, models.RunCanceled} {
		for _, target := range models.QueuingRunStatuses {
			assert.Truef(t, canTransitionTo(runTransitions, s, target), "%s -> %s (retry) should be allowed", s, target)
		}
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunPlanQueued), "%s -> plan_queued should not skip re-admission", s)
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunApplyQueued), "%s -> apply_queued should not skip re-admission", s)
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunPending), "%s -> pending should not be allowed", s)
		assert.Falsef(t, canTransitionTo(runTransitions, s, models.RunPlanned), "%s -> planned should not be allowed", s)
	}
}

func TestRunTransitions_PrePlan(t *testing.T) {
	// The pre-plan stage is workspace-gated, so a run with one waits at pre_plan_queuing from start until
	// the admitter starts it. It must not reach pre_plan_running without passing through that wait.
	assert.True(t, canTransitionTo(runTransitions, models.RunPending, models.RunPrePlanQueuing), "pending -> pre_plan_queuing should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanQueuing, models.RunPrePlanRunning), "pre_plan_queuing -> pre_plan_running should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPending, models.RunPrePlanRunning), "pending -> pre_plan_running should not skip admission")

	// The stage evaluates (running), may block on a human override, and settles to completed.
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanRunning, models.RunPrePlanCompleted), "pre_plan_running -> pre_plan_completed should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanRunning, models.RunPrePlanAwaitingDecision), "pre_plan_running -> pre_plan_awaiting_decision should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanAwaitingDecision, models.RunPrePlanCompleted), "pre_plan_awaiting_decision -> pre_plan_completed (override) should be allowed")

	// A cleared pre-plan stage readies the plan, so the run moves to plan_queuing to await admission for
	// the plan itself — the stage held the slot only for its own evaluation.
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanCompleted, models.RunPlanQueuing), "pre_plan_completed -> plan_queuing should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPrePlanCompleted, models.RunPlanQueued), "pre_plan_completed -> plan_queued should not skip admission")
	assert.False(t, canTransitionTo(runTransitions, models.RunPrePlanCompleted, models.RunPlanning), "pre_plan_completed -> planning should not skip the plan queue")

	// A pre-plan gate awaiting an override can be discarded (declining the override), and that discard is
	// reversible: the discard cancels the gate, so the undiscard re-runs it and the run returns to
	// pre_plan_queuing to be re-admitted before evaluating again.
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanAwaitingDecision, models.RunDiscarded), "pre_plan_awaiting_decision -> discarded should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPrePlanQueuing), "discarded -> pre_plan_queuing (undiscard re-runs the gate) should be allowed")

	// A hard pre-plan failure errors the run before the plan runs; a stage retry returns an
	// errored/canceled run to pre_plan_queuing, so it is re-admitted before running again.
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanRunning, models.RunErrored), "pre_plan_running -> errored should be allowed")
	for _, s := range []models.RunStatus{models.RunErrored, models.RunCanceled} {
		assert.Truef(t, canTransitionTo(runTransitions, s, models.RunPrePlanQueuing), "%s -> pre_plan_queuing (stage retry) should be allowed", s)
	}
}

func TestRunTransitions_PreApply(t *testing.T) {
	// The pre-apply stage mirrors the pre-plan stage: workspace-gated, so it waits at pre_apply_queuing
	// until admitted, and it must not start without passing through that wait.
	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyQueuing, models.RunPreApplyRunning), "pre_apply_queuing -> pre_apply_running should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPlanned, models.RunPreApplyRunning), "planned -> pre_apply_running should not skip admission")

	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyRunning, models.RunPreApplyCompleted), "pre_apply_running -> pre_apply_completed should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyRunning, models.RunPreApplyAwaitingDecision), "pre_apply_running -> pre_apply_awaiting_decision should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyAwaitingDecision, models.RunPreApplyCompleted), "pre_apply_awaiting_decision -> pre_apply_completed (override) should be allowed")

	// Once the gate clears the apply is readied, so the run waits for the slot again — the stage released
	// it when the gate blocked, and holds it only for its own evaluation otherwise.
	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyCompleted, models.RunApplyQueuing), "pre_apply_completed -> apply_queuing should be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPreApplyCompleted, models.RunApplyQueued), "pre_apply_completed -> apply_queued should not skip admission")
	assert.False(t, canTransitionTo(runTransitions, models.RunPreApplyCompleted, models.RunApplying), "pre_apply_completed -> applying should not skip the apply queue")

	// Declining a pre-apply override discards the run, reversibly: the undiscard re-runs the gate the
	// discard canceled, so the run returns to pre_apply_queuing to be re-admitted.
	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyAwaitingDecision, models.RunDiscarded), "pre_apply_awaiting_decision -> discarded should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPreApplyQueuing), "discarded -> pre_apply_queuing (undiscard re-runs the gate) should be allowed")
}

func TestRunTransitions_PostApply(t *testing.T) {
	// The post-apply stage is not workspace-gated (it inherits the apply's slot, mirroring post-plan),
	// so it starts running directly once the apply finishes — no *_queuing status of its own.
	assert.True(t, canTransitionTo(runTransitions, models.RunApplying, models.RunPostApplyRunning), "applying -> post_apply_running should be allowed")
	// An apply with no post-apply stage finishes the run directly.
	assert.True(t, canTransitionTo(runTransitions, models.RunApplying, models.RunApplied), "applying -> applied should be allowed")

	// post_apply_completed is transient: the stage's completed handler passes straight through it to
	// applied within the same state-machine pass. It is still a legal, recorded transition (the run's
	// status-change history and event stream observe it even though no persisted run rests there).
	assert.True(t, canTransitionTo(runTransitions, models.RunPostApplyRunning, models.RunPostApplyCompleted), "post_apply_running -> post_apply_completed should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPostApplyCompleted, models.RunApplied), "post_apply_completed -> applied should be allowed")

	// post_apply is advisory-only (models.Policy.Validate), so no check can ever soft-fail and block on
	// a human override: there is no post_apply_awaiting_decision status at all, and therefore no
	// discard reachable from a post-apply gate.
	assert.False(t, canTransitionTo(runTransitions, models.RunPostApplyRunning, models.RunDiscarded), "post_apply_running -> discarded should not be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunPostApplyCompleted, models.RunDiscarded), "post_apply_completed -> discarded should not be allowed")

	// A hard failure of the post-apply eval job still errors the run — the apply already wrote state,
	// but there is no dedicated status carrying that nuance, mirroring every other hard-mandatory
	// failure. A cancel while the stage runs cancels the run outright too.
	assert.True(t, canTransitionTo(runTransitions, models.RunPostApplyRunning, models.RunErrored), "post_apply_running -> errored should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPostApplyRunning, models.RunCanceled), "post_apply_running -> canceled should be allowed")
	assert.True(t, canTransitionTo(runTransitions, models.RunPostApplyCompleted, models.RunCanceled), "post_apply_completed -> canceled should be allowed")

	// A retry returns an errored/canceled run straight to post_apply_running, the ungated stage that
	// re-runs directly on the apply's slot — never to a *_queuing/_queued status, and never back to
	// applied or pending.
	for _, s := range []models.RunStatus{models.RunErrored, models.RunCanceled} {
		assert.Truef(t, canTransitionTo(runTransitions, s, models.RunPostApplyRunning), "%s -> post_apply_running (stage retry) should be allowed", s)
	}

	// applied is terminal: nothing transitions out of it, including back into post-apply.
	assert.False(t, canTransitionTo(runTransitions, models.RunApplied, models.RunPostApplyRunning), "applied -> post_apply_running should not be allowed")
	assert.False(t, canTransitionTo(runTransitions, models.RunApplied, models.RunErrored), "applied -> errored should not be allowed")
}

func TestTaskStageTransitions_Admission(t *testing.T) {
	// A gated stage is readied to pending and only reaches running once admitted; an ungated stage
	// (post-plan) goes straight from created to running on the plan's slot.
	assert.True(t, canTransitionTo(taskStageTransitions, models.RunTaskStageCreated, models.RunTaskStagePending), "created -> pending should be allowed")
	assert.True(t, canTransitionTo(taskStageTransitions, models.RunTaskStagePending, models.RunTaskStageRunning), "pending -> running (admitted) should be allowed")
	assert.True(t, canTransitionTo(taskStageTransitions, models.RunTaskStageCreated, models.RunTaskStageRunning), "created -> running should be allowed for an ungated stage")

	// A retried stage may return to pending (a gated stage, re-admitted before running again, since the
	// run released the slot when it terminated or parked at the gate) or straight to running (an ungated
	// stage). The stage picks for itself in handleCheckPending; both edges are legal in this status-keyed
	// map. awaiting_override carries the pair too, because a soft-failed check is retryable.
	for _, s := range []models.RunTaskStageStatus{models.RunTaskStageErrored, models.RunTaskStageCanceled, models.RunTaskStageAwaitingOverride} {
		assert.Truef(t, canTransitionTo(taskStageTransitions, s, models.RunTaskStagePending), "%s -> pending (gated retry) should be allowed", s)
		assert.Truef(t, canTransitionTo(taskStageTransitions, s, models.RunTaskStageRunning), "%s -> running (ungated retry) should be allowed", s)
	}

	// A pending stage can still be settled by the run terminating.
	assert.True(t, canTransitionTo(taskStageTransitions, models.RunTaskStagePending, models.RunTaskStageCanceled), "pending -> canceled should be allowed")
	assert.True(t, canTransitionTo(taskStageTransitions, models.RunTaskStagePending, models.RunTaskStageSkipped), "pending -> skipped should be allowed")
}

func TestApplyTransitions_Skipped(t *testing.T) {
	// Only a never-started (created) apply may be skipped.
	assert.True(t, canTransitionTo(applyTransitions, models.ApplyCreated, models.ApplySkipped), "created -> skipped should be allowed")
	for _, s := range []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled} {
		assert.Falsef(t, canTransitionTo(applyTransitions, s, models.ApplySkipped), "%s -> skipped should not be allowed", s)
	}

	// A skipped apply may only be reset back to created (the plan-retry reset);
	// it can never advance directly into the apply lifecycle.
	assert.True(t, canTransitionTo(applyTransitions, models.ApplySkipped, models.ApplyCreated), "skipped -> created should be allowed")
	for _, s := range []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled} {
		assert.Falsef(t, canTransitionTo(applyTransitions, models.ApplySkipped, s), "skipped -> %s should not be allowed", s)
	}
}
