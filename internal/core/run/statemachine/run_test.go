package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestRunNode_PlanPendingMovesRunToQueuing verifies that moving the plan to pending
// moves the run to queuing (waiting to be queued), and admission to queued advances it.
func TestRunNode_PlanPendingMovesRunToQueuing(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)

	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	assert.Equal(t, models.RunQueuing, run.Status())
	assert.Equal(t, models.PlanPending, run.Plan().Status())

	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	assert.Equal(t, models.RunPlanQueued, run.Status())
}

// TestRunNode_QueuingReadiesPlan verifies the run-driven queue path used at run
// creation: transitioning the run itself to queuing drives the still-created plan to
// pending (recorded for persistence, without re-projecting onto the run).
func TestRunNode_QueuingReadiesPlan(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)

	require.NoError(t, run.SetStatus(models.RunQueuing))

	assert.Equal(t, models.RunQueuing, run.Status())
	assert.Equal(t, models.PlanPending, run.Plan().Status())

	// The plan change is recorded for persistence.
	planChanges := run.Plan().GetStatusChanges()
	require.Len(t, planChanges, 1)
	assert.Equal(t, models.PlanPending, planChanges[0].(PlanStatusChange).NewStatus)

	// The run recorded exactly one transition (pending -> queuing): the silent
	// plan set must not have re-projected onto the run.
	runChanges := run.GetStatusChanges()
	require.Len(t, runChanges, 1)
	assert.Equal(t, models.RunQueuing, runChanges[0].(RunStatusChange).NewStatus)
}

// TestRunNode_PlanTransitions verifies how the run reacts as the plan advances
// through its lifecycle. Each case drives the plan through a valid sequence and
// asserts the resulting run status.
func TestRunNode_PlanTransitions(t *testing.T) {
	tests := []struct {
		name      string
		planSeq   []models.PlanStatus
		wantRun   models.RunStatus
		wantApply models.ApplyStatus
	}{
		{"plan pending moves run to queuing", []models.PlanStatus{models.PlanPending}, models.RunQueuing, models.ApplyCreated},
		{"plan queued moves run to plan_queued", []models.PlanStatus{models.PlanPending, models.PlanQueued}, models.RunPlanQueued, models.ApplyCreated},
		{"plan running moves run to planning", []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning}, models.RunPlanning, models.ApplyCreated},
		{"plan finished plans run and leaves apply created", []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished}, models.RunPlanned, models.ApplyCreated},
		{"plan errored errors run and skips apply", []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanErrored}, models.RunErrored, models.ApplySkipped},
		{"plan canceled cancels run and skips apply", []models.PlanStatus{models.PlanPending, models.PlanCanceled}, models.RunCanceled, models.ApplySkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := NewRunNode(models.RunPending)
			run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
			run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
			New(run)

			for _, s := range tt.planSeq {
				require.NoError(t, run.Plan().SetStatus(s))
			}

			assert.Equal(t, tt.wantRun, run.Status())
			assert.Equal(t, tt.wantApply, run.Apply().Status())
		})
	}
}

// TestRunNode_ApplyTransitions verifies how the run reacts as the apply advances
// through its lifecycle, once the plan has finished and the apply is awaiting
// approval. Each case drives the apply through a valid sequence.
func TestRunNode_ApplyTransitions(t *testing.T) {
	tests := []struct {
		name     string
		applySeq []models.ApplyStatus
		wantRun  models.RunStatus
	}{
		{"apply pending moves run to queuing_apply", []models.ApplyStatus{models.ApplyPending}, models.RunQueuingApply},
		{"apply queued moves run to apply_queued", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued}, models.RunApplyQueued},
		{"apply running moves run to applying", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning}, models.RunApplying},
		{"apply finished applies run", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyFinished}, models.RunApplied},
		{"apply errored errors run", []models.ApplyStatus{models.ApplyPending, models.ApplyQueued, models.ApplyRunning, models.ApplyErrored}, models.RunErrored},
		{"apply canceled cancels run", []models.ApplyStatus{models.ApplyPending, models.ApplyCanceled}, models.RunCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := NewRunNode(models.RunPending)
			run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
			run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
			New(run)

			// Drive the plan to finished so the run is planned and the apply is
			// created (awaiting approval).
			require.NoError(t, run.Plan().SetStatus(models.PlanPending))
			require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
			require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
			require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

			for _, s := range tt.applySeq {
				require.NoError(t, run.Apply().SetStatus(s))
			}

			assert.Equal(t, tt.wantRun, run.Status())
		})
	}
}

// TestRunNode_SpeculativePlanFinished verifies a run without an apply node
// finishes once the plan finishes.
func TestRunNode_SpeculativePlanFinished(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	New(run)

	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
	assert.Nil(t, run.Apply())
}

// TestRunNode_PlanFinishedNoChangesFinishesRun verifies that a non-speculative
// run whose plan finishes with no changes goes straight to a final state instead
// of waiting on an apply, and the never-started apply is marked skipped.
func TestRunNode_PlanFinishedNoChangesFinishesRun(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, false)) // no changes
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)

	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status()) // apply will never start
}

// TestRunNode_SpeculativePlanErrored verifies a speculative run errors when the
// plan errors, with no apply cascade to perform.
func TestRunNode_SpeculativePlanErrored(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	New(run)

	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanErrored))

	assert.Equal(t, models.RunErrored, run.Status())
}

// TestRunNode_TerminalLockOnPlanError verifies that a plan error sets the run to
// errored (a terminal state) and marks the never-started apply node skipped.
func TestRunNode_TerminalLockOnPlanError(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))

	require.NoError(t, run.Plan().SetStatus(models.PlanErrored))

	assert.Equal(t, models.RunErrored, run.Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())

	// The run must never have transitioned to canceled.
	for _, c := range run.GetStatusChanges() {
		rc, ok := c.(RunStatusChange)
		require.True(t, ok)
		assert.NotEqual(t, models.RunCanceled, rc.NewStatus)
	}
}

// TestRunNode_PlanFinishLeavesApplyCreated verifies plan completion lands the run
// at planned (with a single transition) and leaves the apply node in its created
// state, awaiting approval.
func TestRunNode_PlanFinishLeavesApplyCreated(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))

	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunPlanned, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())

	plannedCount := 0
	for _, c := range run.GetStatusChanges() {
		if rc, ok := c.(RunStatusChange); ok && rc.NewStatus == models.RunPlanned {
			plannedCount++
		}
	}
	assert.Equal(t, 1, plannedCount)
}

// TestRunNode_AutoApplyPlanFinishPendsApply verifies that when an auto-apply
// run's plan finishes with changes, the apply is moved to pending (ready to be
// queued) and the run passes through planned into queuing_apply.
func TestRunNode_AutoApplyPlanFinishPendsApply(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, true)) // auto-apply
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))

	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunQueuingApply, run.Status())
	assert.Equal(t, models.ApplyPending, run.Apply().Status())

	// The run recorded its passage through planned on the way to queuing_apply.
	var sawPlanned bool
	for _, c := range run.GetStatusChanges() {
		if rc, ok := c.(RunStatusChange); ok && rc.NewStatus == models.RunPlanned {
			sawPlanned = true
		}
	}
	assert.True(t, sawPlanned, "run should record the planned transition before queuing_apply")
}

// TestRunNode_SameStatusIsNoOp verifies that setting the run to the status it already
// holds is a no-op (no error, no recorded change) — the pending projection relies on
// this in the normal forward flow, where the run is already at the target status.
func TestRunNode_SameStatusIsNoOp(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)

	require.NoError(t, run.SetStatus(models.RunPending))
	assert.Empty(t, run.GetStatusChanges())
}

// TestRunNode_StandardHappyPath walks a standard run through its full lifecycle.
func TestRunNode_StandardHappyPath(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	plan := run.Plan()
	apply := run.Apply()

	require.NoError(t, plan.SetStatus(models.PlanPending))
	assert.Equal(t, models.RunQueuing, run.Status()) // plan waiting to be queued

	require.NoError(t, plan.SetStatus(models.PlanQueued))
	assert.Equal(t, models.RunPlanQueued, run.Status())

	require.NoError(t, plan.SetStatus(models.PlanRunning))
	assert.Equal(t, models.RunPlanning, run.Status())

	require.NoError(t, plan.SetStatus(models.PlanFinished))
	assert.Equal(t, models.RunPlanned, run.Status())
	assert.Equal(t, models.ApplyCreated, apply.Status())

	require.NoError(t, apply.SetStatus(models.ApplyPending))
	assert.Equal(t, models.RunQueuingApply, run.Status()) // apply waiting to be queued

	require.NoError(t, apply.SetStatus(models.ApplyQueued))
	assert.Equal(t, models.RunApplyQueued, run.Status())

	require.NoError(t, apply.SetStatus(models.ApplyRunning))
	assert.Equal(t, models.RunApplying, run.Status())

	require.NoError(t, apply.SetStatus(models.ApplyFinished))
	assert.Equal(t, models.RunApplied, run.Status())
}

// TestRunNode_InvalidTransition verifies that run status transitions outside the
// run lifecycle are rejected and leave the run unchanged.
func TestRunNode_InvalidTransition(t *testing.T) {
	tests := []struct {
		name string
		from models.RunStatus
		to   models.RunStatus
	}{
		{"pending to applied", models.RunPending, models.RunApplied},
		{"pending to planning skips plan_queued", models.RunPending, models.RunPlanning},
		{"plan_queued back to pending", models.RunPlanQueued, models.RunPending},
		{"planning to apply_queued skips planned", models.RunPlanning, models.RunApplyQueued},
		{"planned to applying skips apply_queued", models.RunPlanned, models.RunApplying},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := NewRunNode(tt.from)

			err := run.SetStatus(tt.to)

			require.Error(t, err)
			assert.Equal(t, tt.from, run.Status())
			assert.Empty(t, run.GetStatusChanges())
		})
	}
}

// TestRunNode_FinalStatusRejectsTransition verifies that, because the run status is
// fully internally controlled (children driven by the run cancel silently), a
// transition out of a final state is treated as a bug and rejected rather than
// silently absorbed.
func TestRunNode_FinalStatusRejectsTransition(t *testing.T) {
	for _, final := range []models.RunStatus{models.RunApplied, models.RunPlannedAndFinished, models.RunErrored, models.RunCanceled} {
		run := NewRunNode(final)

		err := run.SetStatus(models.RunPlanning)

		require.Error(t, err)
		assert.Equal(t, final, run.Status())
		assert.Empty(t, run.GetStatusChanges())
	}
}

// TestRunNode_PlanErrorSkipsApply verifies that a plan error ends the run and marks
// the never-started apply skipped — recorded as a change for persistence but without
// re-projecting onto the run: the run records exactly one transition (to errored),
// never a canceled transition.
func TestRunNode_PlanErrorSkipsApply(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued)) // plan queued, run plan_queued

	require.NoError(t, run.Plan().SetStatus(models.PlanErrored))

	assert.Equal(t, models.RunErrored, run.Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status()) // apply never started, marked skipped

	// The skip is recorded as an apply change so it is persisted.
	applyChanges := run.Apply().GetStatusChanges()
	require.Len(t, applyChanges, 1)
	assert.Equal(t, models.ApplySkipped, applyChanges[0].(ApplyStatusChange).NewStatus)

	// The plan error must not have re-projected onto the run as a cancel: its last
	// transition is to errored and it never attempts canceled.
	runChanges := run.GetStatusChanges()
	require.NotEmpty(t, runChanges)
	last := runChanges[len(runChanges)-1].(RunStatusChange)
	assert.Equal(t, models.RunErrored, last.NewStatus)
	for _, c := range runChanges {
		assert.NotEqual(t, models.RunCanceled, c.(RunStatusChange).NewStatus)
	}
}

// TestRunNode_PlanRetryUnskipsApply verifies the plan-retry reset: a plan error skips
// the never-started apply, and retrying the plan (back to pending) returns the run to
// queuing and the apply to created so the new plan's outcome decides the apply's fate
// again.
func TestRunNode_PlanRetryUnskipsApply(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))

	require.NoError(t, run.Plan().SetStatus(models.PlanErrored))
	assert.Equal(t, models.RunErrored, run.Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())

	// Retry the plan: errored -> pending resets the run to queuing and un-skips the apply.
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	assert.Equal(t, models.RunQueuing, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())

	// The retried run proceeds normally: the plan finishing with changes lands the run
	// at planned with the apply awaiting approval, exactly like a fresh run.
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))
	assert.Equal(t, models.RunPlanned, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestRunNode_SpeculativeHappyPath walks a speculative run through its lifecycle.
func TestRunNode_SpeculativeHappyPath(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	New(run)
	plan := run.Plan()

	require.NoError(t, plan.SetStatus(models.PlanPending))
	require.NoError(t, plan.SetStatus(models.PlanQueued))
	require.NoError(t, plan.SetStatus(models.PlanRunning))
	require.NoError(t, plan.SetStatus(models.PlanFinished))

	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
}

// TestRunNode_RetryPlan verifies that resetting a failed/canceled plan node to pending
// projects the run back to queuing (the retry case), via the pending listener.
func TestRunNode_RetryPlan(t *testing.T) {
	for _, terminal := range []models.PlanStatus{models.PlanErrored, models.PlanCanceled} {
		t.Run(string(terminal), func(t *testing.T) {
			run := NewRunNode(models.RunPending)
			run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
			run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
			New(run)
			plan := run.Plan()

			require.NoError(t, plan.SetStatus(models.PlanPending))
			require.NoError(t, plan.SetStatus(models.PlanQueued))
			if terminal == models.PlanErrored {
				require.NoError(t, plan.SetStatus(models.PlanRunning))
			}
			require.NoError(t, plan.SetStatus(terminal))
			require.True(t, run.Status().IsFinalStatus(), "run should be terminal after plan %s", terminal)

			// Retry: plan back to pending returns the run to queuing (the plan is
			// again waiting to be queued).
			require.NoError(t, plan.SetStatus(models.PlanPending))
			assert.Equal(t, models.PlanPending, plan.Status())
			assert.Equal(t, models.RunQueuing, run.Status())
		})
	}
}

// TestRunNode_RetryApply verifies that resetting a failed/canceled apply node to
// pending projects the run back to queuing_apply so the apply can be re-queued.
func TestRunNode_RetryApply(t *testing.T) {
	for _, terminal := range []models.ApplyStatus{models.ApplyErrored, models.ApplyCanceled} {
		t.Run(string(terminal), func(t *testing.T) {
			run := NewRunNode(models.RunPending)
			run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
			run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
			New(run)

			// Drive the plan to finished so the run is planned and the apply is created.
			require.NoError(t, run.Plan().SetStatus(models.PlanPending))
			require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
			require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
			require.NoError(t, run.Plan().SetStatus(models.PlanFinished))
			apply := run.Apply()

			require.NoError(t, apply.SetStatus(models.ApplyPending))
			require.NoError(t, apply.SetStatus(models.ApplyQueued))
			if terminal == models.ApplyErrored {
				require.NoError(t, apply.SetStatus(models.ApplyRunning))
			}
			require.NoError(t, apply.SetStatus(terminal))
			require.True(t, run.Status().IsFinalStatus(), "run should be terminal after apply %s", terminal)

			// Retry: apply back to pending returns the run to queuing_apply (the
			// apply is again waiting to be queued).
			require.NoError(t, apply.SetStatus(models.ApplyPending))
			assert.Equal(t, models.ApplyPending, apply.Status())
			assert.Equal(t, models.RunQueuingApply, run.Status())
		})
	}
}

// TestRunNode_PendingProjection verifies the pending listeners project onto the run in
// the normal forward flow: a pending plan moves the run to queuing, and a pending apply
// moves it to queuing_apply.
func TestRunNode_PendingProjection(t *testing.T) {
	// Plan created -> pending at run creation moves the run to queuing.
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	assert.Equal(t, models.RunQueuing, run.Status())

	// Apply created -> pending at start-apply moves the run to queuing_apply.
	run = NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))
	require.NoError(t, run.Apply().SetStatus(models.ApplyPending))
	assert.Equal(t, models.RunQueuingApply, run.Status())
}
