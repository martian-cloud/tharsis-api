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

		changes, err := SetRunStatus(r, models.RunPlanned)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.RunPlanned, r.Status)
		// The skipped apply is restored to created so the run can be applied or discarded again.
		assert.Equal(t, models.ApplyCreated, r.Apply.Status)
	})

	t.Run("cannot undiscard from a state that cannot reach planned", func(t *testing.T) {
		// The discarded-only precondition is enforced by the UndiscardRun command; at the
		// state-machine level only the transition guard applies. A pending run cannot
		// transition to planned, so Undiscard errors. (The command rejects any non-discarded
		// run before reaching here.)
		r := newTestRun() // pending
		_, err := SetRunStatus(r, models.RunPlanned)
		require.Error(t, err)
	})
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

// TestQueue verifies queuing a newly created run: the run moves to queuing and
// the state machine drives the still-created plan to pending so admission can pick
// it up. Queuing a run that is not pending is rejected.
func TestQueue(t *testing.T) {
	t.Run("queues a pending run and readies the plan", func(t *testing.T) {
		r := newTestRun()

		changes, err := SetRunStatus(r, models.RunQueuing)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
		assert.Equal(t, models.RunQueuing, r.Status)
		assert.Equal(t, models.PlanPending, r.Plan.Status)
	})

	t.Run("cannot queue a run that is not pending", func(t *testing.T) {
		r := newTestRun()
		finishTestPlan(t, r) // run planned
		_, err := SetRunStatus(r, models.RunQueuing)
		require.Error(t, err)
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
		{"pending moves run to queuing", []models.PlanStatus{models.PlanPending}, models.RunQueuing, models.ApplyCreated},
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
		{"pending moves run to queuing_apply", []models.ApplyStatus{models.ApplyPending}, models.RunQueuingApply},
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
	assert.Equal(t, models.RunQueuing, r.Status) // plan waiting to be queued

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
	assert.Equal(t, models.RunQueuingApply, r.Status) // apply waiting to be queued

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
