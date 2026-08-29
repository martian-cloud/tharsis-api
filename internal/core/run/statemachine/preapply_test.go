package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// opaPreApplyCheckPath is the run-relative path of the OPA pre-apply policy check node.
const opaPreApplyCheckPath = "pre_apply/opa"

// newPreApplyRun builds a manual (non-auto-apply) run already parked at planned, with a plan that
// finished with changes, a created apply, and a single pre-apply OPA policy check. Starting at planned
// mirrors how newPrePlanRun starts at pending: it is the state immediately before the phase this stage
// gates begins, so driveRunToPreApplyRunning can advance it the same way a fresh run is advanced into
// pre_plan_running.
func newPreApplyRun() *RunNode {
	run := NewRunNode(models.RunPlanned)
	run.SetPlanNode(NewPlanNode("plan", models.PlanFinished, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	stage := NewTaskStageNode("pre-apply-stage", models.RunTaskStageNamePreApply, models.RunTaskStageCreated)
	stage.AddPolicyCheckNode(NewPolicyCheckNode("pre-apply-1", opaPreApplyCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(stage)
	New(run)
	return run
}

func preApplyCheck(run *RunNode) *PolicyCheckNode {
	return run.TaskStage(models.RunTaskStageNamePreApply).PolicyChecks()[0]
}

// driveRunToPreApplyRunning approves a run parked at planned and admits its pre-apply stage, the two
// steps that get a run with a pre-apply stage to pre_apply_running. Approving the run readies the stage
// (planned -> the run reports pre_apply_queuing); the stage is workspace-gated, so it only starts once
// the admitter moves it to running, which is what queues its checks.
func driveRunToPreApplyRunning(t *testing.T, run *RunNode) {
	t.Helper()
	require.NoError(t, run.advance())
	require.Equal(t, models.RunPreApplyQueuing, run.Status())

	// Stand in for the admitter having acquired the workspace slot.
	require.NoError(t, run.TaskStage(models.RunTaskStageNamePreApply).SetStatus(models.RunTaskStageRunning))
	require.Equal(t, models.RunPreApplyRunning, run.Status())
}

// TestAdvanceRun_PreApplyAwaitsAdmission verifies that approving a run with a pre-apply check readies
// the stage but does NOT start it: the stage is workspace-gated, so the run waits at pre_apply_queuing
// and no check is queued (which is what stops a policy-eval job being created before the slot is held).
func TestAdvanceRun_PreApplyAwaitsAdmission(t *testing.T) {
	run := newPreApplyRun()

	require.NoError(t, run.advance())

	assert.Equal(t, models.RunPreApplyQueuing, run.Status())
	assert.Equal(t, models.RunTaskStagePending, run.TaskStage(models.RunTaskStageNamePreApply).Status())
	assert.Equal(t, models.PolicyCheckCreated, preApplyCheck(run).Status(), "no check is queued before admission")
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestAdvanceRun_PreApplyQueuesChecks verifies that once the pre-apply stage is admitted the run
// reports pre_apply_running and the stage queues its check, without releasing the apply.
func TestAdvanceRun_PreApplyQueuesChecks(t *testing.T) {
	run := newPreApplyRun()

	driveRunToPreApplyRunning(t, run)

	assert.Equal(t, models.RunPreApplyRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, preApplyCheck(run).Status())
	// The apply is gated: it must not advance past created until the pre-apply checks clear.
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestPreApplyVerdict_PassReleasesApply verifies that when the pre-apply check passes the apply is
// readied to pending, projecting the run onto apply_queuing to await admission for the apply itself,
// and that admitting the apply leaves the run on the same status with the apply node now queued.
func TestPreApplyVerdict_PassReleasesApply(t *testing.T) {
	run := newPreApplyRun()
	driveRunToPreApplyRunning(t, run)

	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckPassed))

	// The cleared pre-apply stage readies the apply (pending), which projects onto apply_queuing. The
	// stage held the slot only for its own evaluation, so the apply must still be admitted.
	assert.Equal(t, models.RunApplyQueuing, run.Status())
	assert.Equal(t, models.ApplyPending, run.Apply().Status())

	// Admitting the apply moves the run on to apply_queued, so the run status alone says which wait it
	// is in: apply_queuing for the workspace slot, apply_queued for a runner.
	require.NoError(t, run.Apply().SetStatus(models.ApplyQueued))
	assert.Equal(t, models.RunApplyQueued, run.Status())
	assert.Equal(t, models.ApplyQueued, run.Apply().Status())
}

// TestPreApplyVerdict_SoftFailThenOverride verifies that a soft-mandatory pre-apply failure blocks the
// run at pre_apply_awaiting_decision (apply still gated) and that an override releases the apply.
func TestPreApplyVerdict_SoftFailThenOverride(t *testing.T) {
	run := newPreApplyRun()
	driveRunToPreApplyRunning(t, run)

	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckSoftFailed))

	assert.Equal(t, models.RunPreApplyAwaitingDecision, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status(), "apply stays gated while awaiting override")

	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckOverridden))
	assert.Equal(t, models.RunApplyQueuing, run.Status())
	assert.Equal(t, models.ApplyPending, run.Apply().Status())
}

// TestPreApplyVerdict_SoftFailThenRetry verifies the other way out of a pre-apply gate: retrying the
// soft-failed check instead of overriding it. The stage is workspace-gated and the run released its
// slot when it parked at the gate, so the retry leaves the run queuing for re-admission with nothing
// queued — unlike the ungated post-apply stage, which restarts straight into running.
func TestPreApplyVerdict_SoftFailThenRetry(t *testing.T) {
	run := newPreApplyRun()
	driveRunToPreApplyRunning(t, run)

	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckSoftFailed))
	require.Equal(t, models.RunPreApplyAwaitingDecision, run.Status())

	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckPending))

	assert.Equal(t, models.RunPreApplyQueuing, run.Status())
	assert.Equal(t, models.RunTaskStagePending, run.TaskStage(models.RunTaskStageNamePreApply).Status())
	assert.Equal(t, models.PolicyCheckPending, preApplyCheck(run).Status(),
		"no check is queued before the stage is re-admitted")
	assert.Equal(t, models.ApplyCreated, run.Apply().Status(), "the apply stays gated across the retry")

	// Re-admitting the stage queues the check, so a fresh policy-eval job is created only now.
	require.NoError(t, run.advance())
	assert.Equal(t, models.RunPreApplyRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, preApplyCheck(run).Status())
}

// TestPreApplyVerdict_HardFailErrorsRun verifies that a hard-mandatory pre-apply failure (check
// errored) fails the run before the apply runs, and the never-started apply is skipped.
func TestPreApplyVerdict_HardFailErrorsRun(t *testing.T) {
	run := newPreApplyRun()
	driveRunToPreApplyRunning(t, run)

	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckErrored))

	assert.Equal(t, models.RunErrored, run.Status())
	// The apply never started, so it is skipped.
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestPreApplyCancel_SettlesCheck verifies that canceling a run during pre_apply_running cancels the
// active pre-apply check and skips the never-started apply.
func TestPreApplyCancel_SettlesCheck(t *testing.T) {
	run := newPreApplyRun()
	driveRunToPreApplyRunning(t, run)
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckRunning))

	require.NoError(t, run.SetStatus(models.RunCanceled))

	assert.Equal(t, models.RunCanceled, run.Status())
	assert.Equal(t, models.PolicyCheckCanceled, preApplyCheck(run).Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestPreApply_AutoApplyGatesApply verifies an auto-apply run's post-plan stage clearing takes the run
// straight into the pre-apply gate (never resting on planned, which means "waiting on a human"), and
// that the pre-apply check still has to clear before the apply is admitted.
func TestPreApply_AutoApplyGatesApply(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, true))
	preApplyStage := NewTaskStageNode("pre-apply-stage", models.RunTaskStageNamePreApply, models.RunTaskStageCreated)
	preApplyStage.AddPolicyCheckNode(NewPolicyCheckNode("pre-apply-1", opaPreApplyCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(preApplyStage)
	New(run)

	// Drive the plan straight to finished; there is no post-plan stage, so the plan's own success
	// listener starts the apply phase, and startApplyPhase readies the pre-apply gate rather than the
	// apply, per the auto-apply precedence documented on projectPlanFinishedToApply.
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunPreApplyQueuing, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status(), "the apply stays gated behind pre-apply")

	// The run never rested on planned: an auto-apply run is pre-approved and must never report the
	// status that means "waiting on a human".
	var statuses []models.RunStatus
	for _, c := range run.GetStatusChanges() {
		if rc, ok := c.(RunStatusChange); ok {
			statuses = append(statuses, rc.NewStatus)
		}
	}
	assert.NotContains(t, statuses, models.RunPlanned, "an auto-apply run should never enter planned")

	require.NoError(t, preApplyStage.SetStatus(models.RunTaskStageRunning))
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, preApplyCheck(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunApplyQueuing, run.Status())
	assert.Equal(t, models.ApplyPending, run.Apply().Status())
}
