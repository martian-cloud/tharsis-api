package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// opaPrePlanCheckPath is the run-relative path of the OPA pre-plan policy check node.
const opaPrePlanCheckPath = "pre_plan/opa"

// newPrePlanRun builds a run node with a plan, a single pre-plan OPA policy check, and an apply
// node only when withApply is true (a speculative run has none). The run starts pending, as at
// creation.
func newPrePlanRun(withApply bool) *RunNode {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	if withApply {
		run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	}
	stage := NewTaskStageNode("pre-stage", models.RunTaskStageNamePrePlan, models.RunTaskStageCreated)
	stage.AddPolicyCheckNode(NewPolicyCheckNode("pre-1", opaPrePlanCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(stage)
	New(run)
	return run
}

func prePlanCheck(run *RunNode) *PolicyCheckNode {
	return run.TaskStage(models.RunTaskStageNamePrePlan).PolicyChecks()[0]
}

// driveRunToPrePlanRunning starts a freshly-created run and admits its pre-plan stage, the two steps
// that get a run with a pre-plan stage to pre_plan_running. Advancing the fresh run readies the stage
// (pending -> the run reports pre_plan_queuing); the stage is workspace-gated, so it only starts once the
// admitter moves it to running, which is what queues its checks.
func driveRunToPrePlanRunning(t *testing.T, run *RunNode) {
	t.Helper()
	require.NoError(t, run.advance())
	require.Equal(t, models.RunPrePlanQueuing, run.Status())

	// Stand in for the admitter having acquired the workspace slot.
	require.NoError(t, run.TaskStage(models.RunTaskStageNamePrePlan).SetStatus(models.RunTaskStageRunning))
	require.Equal(t, models.RunPrePlanRunning, run.Status())
}

// TestAdvanceRun_PrePlanAwaitsAdmission verifies that starting a run with a pre-plan check readies the
// stage but does NOT start it: the stage is workspace-gated, so the run waits at pre_plan_queuing and no
// check is queued (which is what stops a policy-eval job being created before the slot is held).
func TestAdvanceRun_PrePlanAwaitsAdmission(t *testing.T) {
	run := newPrePlanRun(true)

	require.NoError(t, run.advance())

	assert.Equal(t, models.RunPrePlanQueuing, run.Status())
	assert.Equal(t, models.RunTaskStagePending, run.TaskStage(models.RunTaskStageNamePrePlan).Status())
	assert.Equal(t, models.PolicyCheckCreated, prePlanCheck(run).Status(), "no check is queued before admission")
	assert.Equal(t, models.PlanCreated, run.Plan().Status())
}

// TestAdvanceRun_PrePlanQueuesChecks verifies that once the pre-plan stage is admitted the run reports
// pre_plan_running and the stage queues its check, without releasing the plan.
func TestAdvanceRun_PrePlanQueuesChecks(t *testing.T) {
	run := newPrePlanRun(true)

	driveRunToPrePlanRunning(t, run)

	assert.Equal(t, models.RunPrePlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, prePlanCheck(run).Status())
	// The plan is gated: it must not advance past created until the pre-plan checks clear.
	assert.Equal(t, models.PlanCreated, run.Plan().Status())
}

// TestPrePlanVerdict_PassReleasesPlan verifies that when the pre-plan check passes the plan is readied
// to pending, projecting the run onto plan_queuing to await admission for the plan itself, and that
// admitting the plan leaves the run on the same status with the plan node now queued.
func TestPrePlanVerdict_PassReleasesPlan(t *testing.T) {
	run := newPrePlanRun(true)
	driveRunToPrePlanRunning(t, run)

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckPassed))

	// The cleared pre-plan stage readies the plan (pending), which projects onto plan_queuing. The stage
	// held the slot only for its own evaluation, so the plan must still be admitted.
	assert.Equal(t, models.RunPlanQueuing, run.Status())
	assert.Equal(t, models.PlanPending, run.Plan().Status())

	// Admitting the plan moves the run on to plan_queued, so the run status alone says which wait it is
	// in: plan_queuing for the workspace slot, plan_queued for a runner.
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	assert.Equal(t, models.RunPlanQueued, run.Status())
	assert.Equal(t, models.PlanQueued, run.Plan().Status())
}

// TestPrePlanVerdict_SoftFailThenOverride verifies that a soft-mandatory pre-plan failure blocks the
// run at pre_plan_awaiting_decision (plan still gated) and that an override releases the plan.
func TestPrePlanVerdict_SoftFailThenOverride(t *testing.T) {
	run := newPrePlanRun(true)
	driveRunToPrePlanRunning(t, run)

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckSoftFailed))

	assert.Equal(t, models.RunPrePlanAwaitingDecision, run.Status())
	assert.Equal(t, models.PlanCreated, run.Plan().Status(), "plan stays gated while awaiting override")

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckOverridden))
	assert.Equal(t, models.RunPlanQueuing, run.Status())
	assert.Equal(t, models.PlanPending, run.Plan().Status())
}

// TestPrePlanVerdict_SoftFailThenRetry verifies the other way out of a pre-plan gate: retrying the
// soft-failed check instead of overriding it. The stage is workspace-gated and the run released its slot
// when it parked at the gate, so the retry leaves the run queuing for re-admission with nothing queued —
// unlike the ungated post-plan stage, which restarts straight into running.
func TestPrePlanVerdict_SoftFailThenRetry(t *testing.T) {
	run := newPrePlanRun(true)
	driveRunToPrePlanRunning(t, run)

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckSoftFailed))
	require.Equal(t, models.RunPrePlanAwaitingDecision, run.Status())

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckPending))

	assert.Equal(t, models.RunPrePlanQueuing, run.Status())
	assert.Equal(t, models.RunTaskStagePending, run.TaskStage(models.RunTaskStageNamePrePlan).Status())
	assert.Equal(t, models.PolicyCheckPending, prePlanCheck(run).Status(),
		"no check is queued before the stage is re-admitted")
	assert.Equal(t, models.PlanCreated, run.Plan().Status(), "the plan stays gated across the retry")

	// Re-admitting the stage queues the check, so a fresh policy-eval job is created only now.
	require.NoError(t, run.advance())
	assert.Equal(t, models.RunPrePlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, prePlanCheck(run).Status())
}

// TestPrePlanVerdict_HardFailErrorsRun verifies that a hard-mandatory pre-plan failure (check
// errored) fails the run before the plan runs, and the never-started apply is skipped.
func TestPrePlanVerdict_HardFailErrorsRun(t *testing.T) {
	run := newPrePlanRun(true)
	driveRunToPrePlanRunning(t, run)

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckErrored))

	assert.Equal(t, models.RunErrored, run.Status())
	// The plan and apply never started, so both are skipped.
	assert.Equal(t, models.PlanSkipped, run.Plan().Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestPrePlan_SpeculativeGatesPlan verifies a speculative run (no apply) still gates the plan on its
// pre-plan checks and releases the plan once they pass.
func TestPrePlan_SpeculativeGatesPlan(t *testing.T) {
	run := newPrePlanRun(false)
	driveRunToPrePlanRunning(t, run)
	assert.Equal(t, models.RunPrePlanRunning, run.Status())

	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunPlanQueuing, run.Status())
	assert.Equal(t, models.PlanPending, run.Plan().Status())
	assert.Nil(t, run.Apply())
}

// TestPrePlanCancel_SettlesCheck verifies that canceling a run during pre_plan_running cancels the
// active pre-plan check and skips the never-started apply.
func TestPrePlanCancel_SettlesCheck(t *testing.T) {
	run := newPrePlanRun(true)
	driveRunToPrePlanRunning(t, run)
	require.NoError(t, prePlanCheck(run).SetStatus(models.PolicyCheckRunning))

	require.NoError(t, run.SetStatus(models.RunCanceled))

	assert.Equal(t, models.RunCanceled, run.Status())
	assert.Equal(t, models.PolicyCheckCanceled, prePlanCheck(run).Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestPrePlanCancel_SkipsDownstreamThenRetryRestores is the regression test for a run carrying both
// stages that is canceled during its pre-plan stage: the active pre-plan nodes are canceled while
// the never-started post-plan stage/check and the apply are SKIPPED (not canceled). Retrying the
// pre-plan check then resets those skipped nodes to created, so once the retried pre-plan clears and
// the plan runs, the post-plan stage evaluates again instead of stalling with a skipped check.
func TestPrePlanCancel_SkipsDownstreamThenRetryRestores(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	preStage := NewTaskStageNode("pre-stage", models.RunTaskStageNamePrePlan, models.RunTaskStageCreated)
	preStage.AddPolicyCheckNode(NewPolicyCheckNode("pre-1", opaPrePlanCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(preStage)
	postStage := NewTaskStageNode("post-stage", models.RunTaskStageNamePostPlan, models.RunTaskStageCreated)
	postStage.AddPolicyCheckNode(NewPolicyCheckNode("post-1", opaCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(postStage)
	New(run)

	preCheck := func() *PolicyCheckNode { return run.TaskStage(models.RunTaskStageNamePrePlan).PolicyChecks()[0] }
	postCheck := func() *PolicyCheckNode { return run.TaskStage(models.RunTaskStageNamePostPlan).PolicyChecks()[0] }

	// Drive to pre_plan_running (start readies the gated stage; admission starts it), then cancel while
	// the pre-plan check is active.
	require.NoError(t, run.advance())
	require.NoError(t, preStage.SetStatus(models.RunTaskStageRunning))
	require.NoError(t, preCheck().SetStatus(models.PolicyCheckRunning))
	require.NoError(t, run.SetStatus(models.RunCanceled))

	// The active pre-plan nodes are canceled; the never-started downstream nodes are skipped.
	assert.Equal(t, models.PolicyCheckCanceled, preCheck().Status())
	assert.Equal(t, models.RunTaskStageCanceled, run.TaskStage(models.RunTaskStageNamePrePlan).Status())
	assert.Equal(t, models.PolicyCheckSkipped, postCheck().Status(), "never-started post-plan check is skipped, not canceled")
	assert.Equal(t, models.RunTaskStageSkipped, run.TaskStage(models.RunTaskStageNamePostPlan).Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())

	// Retrying the pre-plan check is a single edge: setting it to pending returns the gated stage to
	// pending, so the run goes back to pre_plan_queuing to be re-admitted (it released the workspace slot
	// when it was canceled). As the run re-enters that phase it resets the skipped downstream nodes (the
	// post-plan stage/check and the apply) back to created — no explicit reset call is needed.
	require.NoError(t, preCheck().SetStatus(models.PolicyCheckPending))
	assert.Equal(t, models.RunTaskStagePending, run.TaskStage(models.RunTaskStageNamePrePlan).Status())
	assert.Equal(t, models.PolicyCheckPending, preCheck().Status(), "a gated stage does not re-queue its check until admitted")
	assert.Equal(t, models.RunPrePlanQueuing, run.Status())
	assert.Equal(t, models.PolicyCheckCreated, postCheck().Status())
	assert.Equal(t, models.RunTaskStageCreated, run.TaskStage(models.RunTaskStageNamePostPlan).Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())

	// Re-admitting the stage starts it again and re-queues its check.
	require.NoError(t, preStage.SetStatus(models.RunTaskStageRunning))
	assert.Equal(t, models.RunPrePlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, preCheck().Status())

	// The retried pre-plan passes, the plan runs and finishes with changes, and the restored
	// post-plan stage evaluates again (the run does NOT stall at post_plan_running).
	require.NoError(t, preCheck().SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preCheck().SetStatus(models.PolicyCheckPassed))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, postCheck().Status())
}

// TestBothStages_PreThenPlanThenPost verifies a run carrying both a pre-plan and a post-plan check:
// the pre-plan stage gates the plan, and only after it clears does the plan run and the post-plan
// stage evaluate.
func TestBothStages_PreThenPlanThenPost(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	preStage := NewTaskStageNode("pre-stage", models.RunTaskStageNamePrePlan, models.RunTaskStageCreated)
	preStage.AddPolicyCheckNode(NewPolicyCheckNode("pre-1", opaPrePlanCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(preStage)
	postStage := NewTaskStageNode("post-stage", models.RunTaskStageNamePostPlan, models.RunTaskStageCreated)
	postStage.AddPolicyCheckNode(NewPolicyCheckNode("post-1", opaCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(postStage)
	New(run)

	preCheck := func() *PolicyCheckNode { return run.TaskStage(models.RunTaskStageNamePrePlan).PolicyChecks()[0] }
	postCheck := func() *PolicyCheckNode { return run.TaskStage(models.RunTaskStageNamePostPlan).PolicyChecks()[0] }

	// Advancing the fresh run readies its workspace-gated pre-plan stage; advancing again (the admitter having got the
	// slot) starts it and queues its check. The plan is gated on the pre-plan check, and the post-plan
	// check stays created.
	require.NoError(t, run.advance())
	require.Equal(t, models.RunPrePlanQueuing, run.Status())
	require.NoError(t, run.advance())
	assert.Equal(t, models.RunPrePlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, preCheck().Status())
	assert.Equal(t, models.PolicyCheckCreated, postCheck().Status())

	// Pre-plan passes -> plan released -> admitted -> runs -> finishes with changes.
	require.NoError(t, preCheck().SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preCheck().SetStatus(models.PolicyCheckPassed))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	// Now the post-plan stage is the active phase.
	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, postCheck().Status())
}
