package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// opaPostApplyCheckPath is the run-relative path of the OPA post-apply policy check node.
const opaPostApplyCheckPath = "post_apply/opa"

// newPostApplyRun builds a run node with a plan, an apply already running, and — when withStage is
// true — a single post-apply OPA policy check. The apply starts at ApplyRunning so the test can drive
// it to ApplyFinished and observe how the run reacts, mirroring how the plan-phase tests drive their
// gating node to its terminal status.
func newPostApplyRun(withStage bool) *RunNode {
	run := NewRunNode(models.RunApplying)
	run.SetPlanNode(NewPlanNode("plan", models.PlanFinished, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyRunning, false))
	if withStage {
		stage := NewTaskStageNode("post-apply-stage", models.RunTaskStageNamePostApply, models.RunTaskStageCreated)
		stage.AddPolicyCheckNode(NewPolicyCheckNode("post-apply-1", opaPostApplyCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
		run.AddTaskStageNode(stage)
	}
	New(run)
	return run
}

func postApplyCheck(run *RunNode) *PolicyCheckNode {
	return run.TaskStage(models.RunTaskStageNamePostApply).PolicyChecks()[0]
}

// TestApplySucceeded_NoPostApplyStage_FinishesRun verifies that a run with no post-apply stage goes
// straight from applying to applied when the apply finishes, unaffected by this change.
func TestApplySucceeded_NoPostApplyStage_FinishesRun(t *testing.T) {
	run := newPostApplyRun(false)

	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))

	assert.Equal(t, models.RunApplied, run.Status())
}

// TestApplySucceeded_PostApplyStage_QueuesCheck verifies that a run with a post-apply stage moves to
// post_apply_running and queues its check when the apply finishes — mirroring how a post-plan stage
// starts once the plan finishes with changes — rather than settling on applied immediately.
func TestApplySucceeded_PostApplyStage_QueuesCheck(t *testing.T) {
	run := newPostApplyRun(true)

	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))

	assert.Equal(t, models.RunPostApplyRunning, run.Status())
	assert.Equal(t, models.RunTaskStageRunning, run.TaskStage(models.RunTaskStageNamePostApply).Status())
	assert.Equal(t, models.PolicyCheckQueued, postApplyCheck(run).Status())
	// The apply already finished and is untouched by the post-apply stage.
	assert.Equal(t, models.ApplyFinished, run.Apply().Status())
}

// TestPostApplyVerdict_PassFinishesRun verifies that a passed post-apply check clears the stage and
// finishes the run, passing through the transient post_apply_completed status on the way — recorded in
// the run's status-change history, per requirement, but never the run's resting status.
func TestPostApplyVerdict_PassFinishesRun(t *testing.T) {
	run := newPostApplyRun(true)
	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))

	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunApplied, run.Status())

	// The recorded sequence must include post_apply_completed immediately before applied — the
	// transient status is a real recorded step, not merely skipped over.
	changes := run.GetStatusChanges()
	require.NotEmpty(t, changes)
	last := changes[len(changes)-1].(RunStatusChange)
	secondToLast := changes[len(changes)-2].(RunStatusChange)
	assert.Equal(t, models.RunPostApplyCompleted, secondToLast.NewStatus)
	assert.Equal(t, models.RunApplied, last.NewStatus)
}

// TestPostApplyVerdict_AdvisoryFailureStillFinishesRun verifies requirement 2: a post-apply check can
// only be advisory (enforced at the policy layer, not exercised by the state machine itself), and an
// advisory failure is reported as a passed check verdict by the outcome-reporting layer — so from the
// state machine's point of view a "failed" advisory policy still finishes the run normally. This test
// documents that the post-apply stage applies no special-casing: any PolicyCheckPassed clears it,
// regardless of what the check's underlying policies individually reported.
func TestPostApplyVerdict_AdvisoryFailureStillFinishesRun(t *testing.T) {
	run := newPostApplyRun(true)
	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))

	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	// An advisory-only failure never blocks a run, so the outcome-reporting layer computes a "passed"
	// check verdict even though a policy underneath it failed (see computePolicyCheckVerdict).
	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunApplied, run.Status())
}

// TestPostApplyVerdict_HardFailErrorsRun verifies that an errored post-apply check (the eval job
// itself failing, not a policy verdict) errors the run per the normal enforcement-level projection —
// the apply already succeeded, but there is no dedicated status carrying that nuance.
func TestPostApplyVerdict_HardFailErrorsRun(t *testing.T) {
	run := newPostApplyRun(true)
	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))

	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckErrored))

	assert.Equal(t, models.RunErrored, run.Status())
}

// TestPostApplyVerdict_RetryRestartsOnApplySlot verifies that retrying an errored post-apply check
// restarts the stage directly on running — the stage is ungated (it inherits the apply's slot), so the
// retry mirrors the post-plan stage's retry rather than a workspace-gated stage's.
func TestPostApplyVerdict_RetryRestartsOnApplySlot(t *testing.T) {
	run := newPostApplyRun(true)
	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))
	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckRunning))
	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckErrored))
	require.Equal(t, models.RunErrored, run.Status())

	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckPending))

	assert.Equal(t, models.RunPostApplyRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, postApplyCheck(run).Status())
}

// TestPostApplyVerdict_CancelCancelsRun verifies that canceling the run while it is in
// post_apply_running (the path commands.cancelActivePhase drives via statemachine.SetRunStatus once
// every non-final check's job has stopped) cancels the run outright — a successful apply followed by a
// canceled post-apply stage still reports canceled, per the normal projection (no dedicated "applied
// but post-apply canceled" status).
func TestPostApplyVerdict_CancelCancelsRun(t *testing.T) {
	run := newPostApplyRun(true)
	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))
	require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckRunning))

	require.NoError(t, run.SetStatus(models.RunCanceled))

	assert.Equal(t, models.RunCanceled, run.Status())
}

// TestStateMachine_NoRunEverRestsOnTransientCompletedStatuses is the invariant that makes it safe for
// post_plan_completed and post_apply_completed to be absent from the GraphQL and protobuf RunStatus
// enums: every path that sets either status advances the run further within the same pass, so no
// persisted run (whose status is read only after a full pass completes) can ever be observed resting
// on one.
func TestStateMachine_NoRunEverRestsOnTransientCompletedStatuses(t *testing.T) {
	transient := map[models.RunStatus]bool{
		models.RunPostPlanCompleted:  true,
		models.RunPostApplyCompleted: true,
	}

	t.Run("post_plan_completed, manual run with apply", func(t *testing.T) {
		run := newStageRunWith(stageRunConfig{hasChanges: true, withApply: true, autoApply: false})
		drivePlanToFinished(t, run)
		driveCheckToRunning(t, run)
		require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))
		assert.False(t, transient[run.Status()], "run must not rest on %q", run.Status())
	})

	t.Run("post_plan_completed, auto-apply run", func(t *testing.T) {
		run := newStageRunWith(stageRunConfig{hasChanges: true, withApply: true, autoApply: true})
		drivePlanToFinished(t, run)
		driveCheckToRunning(t, run)
		require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))
		assert.False(t, transient[run.Status()], "run must not rest on %q", run.Status())
	})

	t.Run("post_plan_completed, speculative run", func(t *testing.T) {
		run := newStageRunWith(stageRunConfig{hasChanges: true, withApply: false, autoApply: false})
		drivePlanToFinished(t, run)
		driveCheckToRunning(t, run)
		require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))
		assert.False(t, transient[run.Status()], "run must not rest on %q", run.Status())
	})

	t.Run("post_apply_completed", func(t *testing.T) {
		run := newPostApplyRun(true)
		require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))
		require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckRunning))
		require.NoError(t, postApplyCheck(run).SetStatus(models.PolicyCheckPassed))
		assert.False(t, transient[run.Status()], "run must not rest on %q", run.Status())
	})
}
