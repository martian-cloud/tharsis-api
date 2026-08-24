package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestDiscard verifies a planned run moves to the terminal discarded status (marking
// the never-started apply node skipped) and that the transition map rejects discarding
// a non-planned run (the discard command gates the user-facing conflict error).
func TestDiscard(t *testing.T) {
	t.Run("discards a planned run", func(t *testing.T) {
		r := newTestRun()
		finishTestPlan(t, r) // run planned, apply created
		require.Equal(t, models.RunPlanned, r.Status)

		changes, err := SetRunStatus(r, models.RunDiscarded)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.RunDiscarded, r.Status)
		// The never-started apply node is marked skipped.
		assert.Equal(t, models.ApplySkipped, r.Apply.Status)
	})

	t.Run("cannot discard a run that is not planned", func(t *testing.T) {
		r := newTestRun() // pending
		_, err := SetRunStatus(r, models.RunDiscarded)
		require.Error(t, err)
	})
}

func TestUndiscard(t *testing.T) {
	t.Run("undiscards a discarded run back to planned and restores the apply", func(t *testing.T) {
		r := newTestRun()
		finishTestPlan(t, r) // run planned, apply created
		_, err := SetRunStatus(r, models.RunDiscarded)
		require.NoError(t, err)
		require.Equal(t, models.RunDiscarded, r.Status)
		require.Equal(t, models.ApplySkipped, r.Apply.Status)

		changes, err := UndiscardRun(r)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.RunPlanned, r.Status)
		// The skipped apply is restored to created so the run can be applied or discarded again.
		assert.Equal(t, models.ApplyCreated, r.Apply.Status)
	})

	t.Run("cannot undiscard from a state that cannot reach planned", func(t *testing.T) {
		// The discarded-only precondition is enforced by the UndiscardRun command; at the
		// state-machine level only the transition guard applies. A pending run (no post-plan gate)
		// undiscards toward planned, which pending cannot transition to, so it errors. (The command
		// rejects any non-discarded run before reaching here.)
		r := newTestRun() // pending
		_, err := UndiscardRun(r)
		require.Error(t, err)
	})
}

// TestUndiscard_RerunsPostPlanGate verifies the model round-trip for discarding and then undiscarding
// a run blocked at post_plan_awaiting_decision: the discard cancels the gate — its stage and the
// soft-failed check under it — along with skipping the never-started apply, and the undiscard re-runs
// that gate rather than restoring its verdict. The check returns to pending and its ungated stage
// restarts directly on the plan's slot, so the run is back at post_plan_running with the apply
// restored and the gate still to be satisfied before it can be applied.
func TestUndiscard_RerunsPostPlanGate(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-post", StageName: models.RunTaskStageNamePostPlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Drive the plan to finished (queues the post-plan check), then soft-fail the check so the run
	// blocks awaiting a human override.
	for _, s := range []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished} {
		_, err := SetPlanStatus(r, s)
		require.NoError(t, err)
	}
	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err := SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckSoftFailed)
	require.NoError(t, err)
	require.Equal(t, models.RunPostPlanAwaitingDecision, r.Status)
	require.Equal(t, models.RunTaskStageAwaitingOverride, r.TaskStages[0].Status)

	// Discard from the awaiting-decision state: the abandoned gate is canceled and the apply skipped.
	_, err = SetRunStatus(r, models.RunDiscarded)
	require.NoError(t, err)
	assert.Equal(t, models.RunDiscarded, r.Status)
	assert.Equal(t, models.RunTaskStageCanceled, r.TaskStages[0].Status, "the abandoned gate stage is canceled")
	assert.Equal(t, models.PolicyCheckCanceled, r.AllPolicyChecks()[0].Status, "the gate's check is canceled with it")
	assert.Equal(t, models.ApplySkipped, r.Apply.Status)

	// Undiscard re-runs the gate: the check is re-queued (its stage is ungated, so it restarts straight
	// to running) and the apply is restored.
	_, err = UndiscardRun(r)
	require.NoError(t, err)
	assert.Equal(t, models.RunPostPlanRunning, r.Status)
	assert.Equal(t, models.RunTaskStageRunning, r.TaskStages[0].Status)
	assert.Equal(t, models.PolicyCheckQueued, r.AllPolicyChecks()[0].Status, "the check is re-queued for a fresh evaluation")
	assert.Equal(t, models.ApplyCreated, r.Apply.Status)

	// The re-evaluation decides the run: a check that now passes clears the gate and plans the run.
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckPassed)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanned, r.Status)
	assert.Equal(t, models.ApplyCreated, r.Apply.Status)
}

// TestUndiscard_RerunsPrePlanGate verifies the model round-trip for discarding and then undiscarding a
// run blocked at pre_plan_awaiting_decision: the discard cancels the pre-plan gate and skips both the
// never-run plan and the unstarted apply, and the undiscard re-runs the gate. The stage is
// workspace-gated, so it restarts at pending and the run waits at pre_plan_queuing to be re-admitted
// before the check is evaluated again — the plan still does not run until the gate clears.
func TestUndiscard_RerunsPrePlanGate(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-pre", StageName: models.RunTaskStageNamePrePlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePrePlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Advance the fresh run, which readies its workspace-gated pre-plan stage, then admit the stage (standing in
	// for the admitter acquiring the slot) so its check is queued. Soft-fail the check so the run blocks
	// awaiting a human override before the plan runs.
	_, err := AdvanceRun(r)
	require.NoError(t, err)
	require.Equal(t, models.RunPrePlanQueuing, r.Status)
	_, err = AdvanceRun(r)
	require.NoError(t, err)
	require.Equal(t, models.RunPrePlanRunning, r.Status)
	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckSoftFailed)
	require.NoError(t, err)
	require.Equal(t, models.RunPrePlanAwaitingDecision, r.Status)
	require.Equal(t, models.RunTaskStageAwaitingOverride, r.TaskStages[0].Status)

	// Discard from the pre-plan awaiting-decision state: the abandoned gate is canceled, and the
	// never-run plan and the unstarted apply are both skipped.
	_, err = SetRunStatus(r, models.RunDiscarded)
	require.NoError(t, err)
	assert.Equal(t, models.RunDiscarded, r.Status)
	assert.Equal(t, models.RunTaskStageCanceled, r.TaskStages[0].Status, "the abandoned pre-plan gate stage is canceled")
	assert.Equal(t, models.PolicyCheckCanceled, r.AllPolicyChecks()[0].Status, "the gate's check is canceled with it")
	assert.Equal(t, models.PlanSkipped, r.Plan.Status, "the never-run plan is skipped")
	assert.Equal(t, models.ApplySkipped, r.Apply.Status)

	// Undiscard re-runs the pre-plan gate and restores the skipped plan and apply to created. The stage
	// is gated, so it waits at pending for the slot rather than evaluating immediately.
	_, err = UndiscardRun(r)
	require.NoError(t, err)
	assert.Equal(t, models.RunPrePlanQueuing, r.Status)
	assert.Equal(t, models.RunTaskStagePending, r.TaskStages[0].Status)
	assert.Equal(t, models.PolicyCheckPending, r.AllPolicyChecks()[0].Status, "the check waits for re-admission before re-evaluating")
	assert.Equal(t, models.PlanCreated, r.Plan.Status)
	assert.Equal(t, models.ApplyCreated, r.Apply.Status)

	// Re-admitted, the stage evaluates again; a check that now passes clears the gate, readying the plan
	// and returning the run to plan_queuing to await admission.
	_, err = AdvanceRun(r)
	require.NoError(t, err)
	require.Equal(t, models.RunPrePlanRunning, r.Status)
	require.Equal(t, models.PolicyCheckQueued, r.AllPolicyChecks()[0].Status)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckPassed)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanQueuing, r.Status)
	assert.Equal(t, models.PlanPending, r.Plan.Status)
}

// TestUndiscard_ClearedPostPlanGate verifies that a run discarded from planned after its post-plan
// gate had already cleared (stage completed) undiscards back to planned: the completed stage is
// preserved through discard and undiscard derives planned from it, restoring the apply.
func TestUndiscard_ClearedPostPlanGate(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-post", StageName: models.RunTaskStageNamePostPlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Drive the plan to finished and pass the post-plan check: the gate clears and the run is planned.
	for _, s := range []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished} {
		_, err := SetPlanStatus(r, s)
		require.NoError(t, err)
	}
	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err := SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckPassed)
	require.NoError(t, err)
	require.Equal(t, models.RunPlanned, r.Status)
	require.Equal(t, models.RunTaskStageCompleted, r.TaskStages[0].Status)

	// Discard, then undiscard: the completed gate is preserved and the run returns to planned.
	_, err = SetRunStatus(r, models.RunDiscarded)
	require.NoError(t, err)
	require.Equal(t, models.RunTaskStageCompleted, r.TaskStages[0].Status, "completed gate is preserved through discard")
	require.Equal(t, models.ApplySkipped, r.Apply.Status)

	_, err = UndiscardRun(r)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanned, r.Status)
	assert.Equal(t, models.ApplyCreated, r.Apply.Status)
}

func newTestRun() *models.Run {
	return &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
	}
}

func newSpeculativeTestRun() *models.Run {
	return &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated},
	}
}

func startTestRun(t *testing.T, r *models.Run) {
	_, err := SetPlanStatus(r, models.PlanPending)
	require.NoError(t, err)
}

func finishTestPlan(t *testing.T, r *models.Run) {
	startTestRun(t, r)
	for _, s := range []models.PlanStatus{models.PlanQueued, models.PlanRunning, models.PlanFinished} {
		_, err := SetPlanStatus(r, s)
		require.NoError(t, err)
	}
}

// TestAdvanceRun covers the three things advancing a run can mean, all chosen from the run's own state:
// beginning the plan phase for a freshly-created run, admitting a node that was waiting for the
// workspace slot, and beginning the apply phase for a manual run parked at planned. It is also a no-op
// when the run is not waiting on anything, so a redundant call cannot corrupt it.
func TestAdvanceRun(t *testing.T) {
	t.Run("a pending run begins its plan phase", func(t *testing.T) {
		r := newTestRun() // no pre-plan stage, so the plan itself is readied

		changes, err := AdvanceRun(r)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.RunPlanQueuing, r.Status, "the readied plan now waits for the slot")
		assert.Equal(t, models.PlanPending, r.Plan.Status)
	})

	t.Run("a pending plan is admitted", func(t *testing.T) {
		r := newTestRun()
		_, err := AdvanceRun(r) // begins the plan phase; the plan is now pending
		require.NoError(t, err)

		changes, err := AdvanceRun(r)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.PlanQueued, r.Plan.Status)
		// Admission is a real run-status transition: plan_queuing -> plan_queued records the moment the
		// run won the workspace slot and is now waiting only for a runner.
		assert.Equal(t, models.RunPlanQueued, r.Status)
		assert.False(t, r.Status.IsQueuing())
	})

	t.Run("a manual run parked at planned begins its apply phase", func(t *testing.T) {
		r := newTestRun()
		finishTestPlan(t, r) // run planned, apply created and awaiting approval
		require.False(t, r.Status.IsQueuing(), "waiting on a human, not the workspace")

		changes, err := AdvanceRun(r)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.RunApplyQueuing, r.Status)
		assert.Equal(t, models.ApplyPending, r.Apply.Status)
	})

	t.Run("a run that is not waiting is a no-op", func(t *testing.T) {
		r := newTestRun()
		for _, s := range []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning} {
			_, err := SetPlanStatus(r, s)
			require.NoError(t, err)
		}
		require.Equal(t, models.RunPlanning, r.Status)

		changes, err := AdvanceRun(r)
		require.NoError(t, err)
		assert.Empty(t, changes)
		assert.Equal(t, models.RunPlanning, r.Status)
	})
}

// TestSetPlanStatus_TransitionsRun verifies that updating the plan node's
// status through SetPlanStatus transitions the run (and cascades to the apply node)
// to the correct status.
func TestSetPlanStatus_TransitionsRun(t *testing.T) {
	tests := []struct {
		name      string
		planSeq   []models.PlanStatus
		wantRun   models.RunStatus
		wantApply models.ApplyStatus
	}{
		{"pending moves run to plan_queuing", []models.PlanStatus{models.PlanPending}, models.RunPlanQueuing, models.ApplyCreated},
		{"queued moves run to plan_queued", []models.PlanStatus{models.PlanPending, models.PlanQueued}, models.RunPlanQueued, models.ApplyCreated},
		{"running moves run to planning", []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning}, models.RunPlanning, models.ApplyCreated},
		{"finished plans run and leaves apply created", []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished}, models.RunPlanned, models.ApplyCreated},
		{"errored errors run and skips apply", []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanErrored}, models.RunErrored, models.ApplySkipped},
		{"canceled cancels run and skips apply", []models.PlanStatus{models.PlanPending, models.PlanCanceled}, models.RunCanceled, models.ApplySkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRun()

			var last models.PlanStatus
			for _, s := range tt.planSeq {
				_, err := SetPlanStatus(r, s)
				require.NoError(t, err)
				last = s
			}

			assert.Equal(t, tt.wantRun, r.Status)
			assert.Equal(t, last, r.Plan.Status)
			assert.Equal(t, tt.wantApply, r.Apply.Status)
		})
	}
}

// TestSetApplyStatus_TransitionsRun verifies that updating the apply node's
// status through SetApplyStatus transitions the run to the correct status.
func TestSetApplyStatus_TransitionsRun(t *testing.T) {
	tests := []struct {
		name     string
		applySeq []models.ApplyStatus
		wantRun  models.RunStatus
	}{
		{"pending moves run to apply_queuing", []models.ApplyStatus{models.ApplyPending}, models.RunApplyQueuing},
		{"queued moves run to apply_queued", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued}, models.RunApplyQueued},
		{"running moves run to applying", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning}, models.RunApplying},
		{"finished applies run", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyFinished}, models.RunApplied},
		{"errored errors run", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyErrored}, models.RunErrored},
		{"canceled cancels run", []models.ApplyStatus{models.ApplyPending, models.ApplyCanceled}, models.RunCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRun()
			finishTestPlan(t, r)

			var last models.ApplyStatus
			for _, s := range tt.applySeq {
				_, err := SetApplyStatus(r, s)
				require.NoError(t, err)
				last = s
			}

			assert.Equal(t, tt.wantRun, r.Status)
			assert.Equal(t, last, r.Apply.Status)
		})
	}
}

// TestSetPlanStatus_SpeculativeFinished verifies a run without an apply node
// reaches planned_and_finished when the plan finishes.
func TestSetPlanStatus_SpeculativeFinished(t *testing.T) {
	r := newSpeculativeTestRun()
	startTestRun(t, r)

	_, err := SetPlanStatus(r, models.PlanQueued)
	require.NoError(t, err)
	_, err = SetPlanStatus(r, models.PlanRunning)
	require.NoError(t, err)
	_, err = SetPlanStatus(r, models.PlanFinished)
	require.NoError(t, err)

	assert.Equal(t, models.RunPlannedAndFinished, r.Status)
	assert.Equal(t, models.PlanFinished, r.Plan.Status)
}

// TestPlanFinishedNoChangesFinishesRun verifies that a non-speculative run
// whose plan finishes with no changes finishes immediately rather than waiting
// on an apply, with the never-started apply marked skipped.
func TestPlanFinishedNoChangesFinishesRun(t *testing.T) {
	r := newTestRun()
	r.Plan.HasChanges = false

	finishTestPlan(t, r)

	assert.Equal(t, models.RunPlannedAndFinished, r.Status)
	assert.Equal(t, models.ApplySkipped, r.Apply.Status)
}

// TestSetApplyStatus_NoApplyNodeErrors verifies updating an apply status on a
// speculative run is an error.
func TestSetApplyStatus_NoApplyNodeErrors(t *testing.T) {
	r := newSpeculativeTestRun()

	_, err := SetApplyStatus(r, models.ApplyRunning)
	assert.Error(t, err)
}

// TestSetPlanStatus_ReturnsRunChange verifies the caller receives the run's
// status transition alongside the plan change.
func TestSetPlanStatus_ReturnsRunChange(t *testing.T) {
	r := newTestRun()
	startTestRun(t, r)
	_, err := SetPlanStatus(r, models.PlanQueued)
	require.NoError(t, err)

	changes, err := SetPlanStatus(r, models.PlanRunning)
	require.NoError(t, err)

	var sawRunPlanning bool
	for _, c := range changes {
		if rc, ok := c.(RunStatusChange); ok && rc.NewStatus == models.RunPlanning {
			sawRunPlanning = true
		}
	}
	assert.True(t, sawRunPlanning, "expected a run transition to planning in the returned changes")
}

// TestHappyPath walks a standard run through its full lifecycle via the package
// functions, asserting the run status after each child update.
func TestHappyPath(t *testing.T) {
	r := newTestRun()

	_, err := SetPlanStatus(r, models.PlanPending)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanQueuing, r.Status) // plan waiting for the workspace slot

	_, err = SetPlanStatus(r, models.PlanQueued)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanQueued, r.Status)

	_, err = SetPlanStatus(r, models.PlanRunning)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanning, r.Status)

	_, err = SetPlanStatus(r, models.PlanFinished)
	require.NoError(t, err)
	assert.Equal(t, models.RunPlanned, r.Status)
	assert.Equal(t, models.ApplyCreated, r.Apply.Status)

	_, err = SetApplyStatus(r, models.ApplyPending)
	require.NoError(t, err)
	assert.Equal(t, models.RunApplyQueuing, r.Status) // apply waiting for the workspace slot

	_, err = SetApplyStatus(r, models.ApplyQueued)
	require.NoError(t, err)
	assert.Equal(t, models.RunApplyQueued, r.Status)

	_, err = SetApplyStatus(r, models.ApplyRunning)
	require.NoError(t, err)
	assert.Equal(t, models.RunApplying, r.Status)

	_, err = SetApplyStatus(r, models.ApplyFinished)
	require.NoError(t, err)
	assert.Equal(t, models.RunApplied, r.Status)
}

// TestCancelViaPlanStatus verifies that cancelling the plan node (how the cancel
// command stops a non-running plan) cancels the run and marks the never-started
// apply node skipped.
func TestCancelViaPlanStatus(t *testing.T) {
	r := newTestRun()
	startTestRun(t, r)
	_, err := SetPlanStatus(r, models.PlanQueued)
	require.NoError(t, err)

	_, err = SetPlanStatus(r, models.PlanCanceled)
	require.NoError(t, err)

	assert.Equal(t, models.RunCanceled, r.Status)
	assert.Equal(t, models.PlanCanceled, r.Plan.Status)
	assert.Equal(t, models.ApplySkipped, r.Apply.Status)
}

// TestSpeculativeRunWithChecks_QueuesThenFinishes verifies the model round-trip for a speculative
// run with a post-plan check: the plan finishing queues the check (persisted back to the model),
// and once the check passes the run finishes with the check status persisted.
func TestSpeculativeRunWithChecks_QueuesThenFinishes(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-post", StageName: models.RunTaskStageNamePostPlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Driving the plan to finished enters post_plan_running and queues the check, even though
	// the run is speculative (no apply node).
	for _, s := range []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished} {
		_, err := SetPlanStatus(r, s)
		require.NoError(t, err)
	}
	require.Equal(t, models.RunPostPlanRunning, r.Status)
	require.Equal(t, models.PolicyCheckQueued, r.AllPolicyChecks()[0].Status)

	// The check runs and passes; the speculative run finishes with nothing to apply.
	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err := SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckPassed)
	require.NoError(t, err)

	assert.Equal(t, models.RunPlannedAndFinished, r.Status)
	assert.Equal(t, models.PolicyCheckPassed, r.AllPolicyChecks()[0].Status)
}

// TestPlanFinishedNoChangesSkipsChecks verifies the model round-trip for a no-change plan: the
// post-plan check is skipped (persisted), the run finishes, and the never-started apply is skipped.
func TestPlanFinishedNoChangesSkipsChecks(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: false},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-post", StageName: models.RunTaskStageNamePostPlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePostPlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	for _, s := range []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished} {
		_, err := SetPlanStatus(r, s)
		require.NoError(t, err)
	}

	assert.Equal(t, models.RunPlannedAndFinished, r.Status)
	assert.Equal(t, models.PolicyCheckSkipped, r.AllPolicyChecks()[0].Status)
	assert.Equal(t, models.ApplySkipped, r.Apply.Status)
}

// TestPrePlanRunWithChecks_GatesThenReleasesPlan verifies the model round-trip for a run with a
// pre-plan check: queuing starts the lock-free pre-plan stage and gates the plan (run ->
// pre_plan_running, check queued, plan still created), and once the check passes the run returns to
// queuing with the plan readied to pending (awaiting workspace admission) — all persisted back to
// the model.
func TestPrePlanRunWithChecks_GatesThenReleasesPlan(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-pre", StageName: models.RunTaskStageNamePrePlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePrePlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Advancing the fresh run readies its workspace-gated pre-plan stage; the stage's check is only queued once
	// the stage is admitted, and the plan stays gated behind the stage throughout.
	_, err := AdvanceRun(r)
	require.NoError(t, err)
	require.Equal(t, models.RunPrePlanQueuing, r.Status)
	require.Equal(t, models.PolicyCheckCreated, r.AllPolicyChecks()[0].Status, "no check is queued before admission")

	_, err = AdvanceRun(r)
	require.NoError(t, err)
	require.Equal(t, models.RunPrePlanRunning, r.Status)
	require.Equal(t, models.PolicyCheckQueued, r.AllPolicyChecks()[0].Status)
	require.Equal(t, models.PlanCreated, r.Plan.Status, "plan is gated until pre-plan clears")

	// The check runs and passes; the plan is readied to pending, projecting the run onto plan_queued to
	// await admission for the plan itself.
	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckPassed)
	require.NoError(t, err)

	assert.Equal(t, models.RunPlanQueuing, r.Status)
	assert.Equal(t, models.PolicyCheckPassed, r.AllPolicyChecks()[0].Status)
	assert.Equal(t, models.PlanPending, r.Plan.Status)
}

// TestPrePlanRunHardFail_ErrorsBeforePlan verifies the model round-trip for a pre-plan check that
// errors (hard-mandatory failure): the run errors before the plan runs and the apply is skipped.
func TestPrePlanRunHardFail_ErrorsBeforePlan(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-pre", StageName: models.RunTaskStageNamePrePlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "check-1", StageName: models.RunTaskStageNamePrePlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Advancing the fresh run readies its pre-plan stage; admitting the stage queues its check.
	_, err := AdvanceRun(r)
	require.NoError(t, err)
	_, err = AdvanceRun(r)
	require.NoError(t, err)

	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)
	_, err = SetPolicyCheckStatus(r, checkPath, models.PolicyCheckErrored)
	require.NoError(t, err)

	assert.Equal(t, models.RunErrored, r.Status)
	assert.Equal(t, models.PlanSkipped, r.Plan.Status, "the plan never ran, so it is skipped")
	assert.Equal(t, models.ApplySkipped, r.Apply.Status)
}
