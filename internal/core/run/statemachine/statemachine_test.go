package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestStateMachine_GetRunNode verifies the wrapper exposes the root run node.
func TestStateMachine_GetRunNode(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	sm := New(run)

	got := sm.GetRunNode()

	require.NotNil(t, got)
	assert.Equal(t, models.RunPending, got.Status())
}

// TestStateMachine_GetStatusChangesOrdering verifies aggregated changes are
// ordered run-first, then plan, then apply, and carry the expected values.
func TestStateMachine_GetStatusChangesOrdering(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	sm := New(run)

	// Drive the plan through its lifecycle to queued; the pending and queued
	// transitions both advance the run, and collected changes are run-first then plan.
	require.NoError(t, sm.GetRunNode().Plan().SetStatus(models.PlanPending))
	require.NoError(t, sm.GetRunNode().Plan().SetStatus(models.PlanQueued))

	changes := sm.GetStatusChanges()
	// run: pending -> queuing -> plan_queued; plan: created -> pending -> queued.
	require.Len(t, changes, 4)

	runChange1, ok := changes[0].(RunStatusChange)
	require.True(t, ok)
	assert.Equal(t, models.RunPending, runChange1.OldStatus)
	assert.Equal(t, models.RunQueuing, runChange1.NewStatus)
	assert.Equal(t, RunNodeType, runChange1.GetNodeType())

	runChange2, ok := changes[1].(RunStatusChange)
	require.True(t, ok)
	assert.Equal(t, models.RunQueuing, runChange2.OldStatus)
	assert.Equal(t, models.RunPlanQueued, runChange2.NewStatus)

	planChange1, ok := changes[2].(PlanStatusChange)
	require.True(t, ok)
	assert.Equal(t, models.PlanCreated, planChange1.OldStatus)
	assert.Equal(t, models.PlanPending, planChange1.NewStatus)
	assert.Equal(t, "plan", planChange1.Path)

	planChange2, ok := changes[3].(PlanStatusChange)
	require.True(t, ok)
	assert.Equal(t, models.PlanPending, planChange2.OldStatus)
	assert.Equal(t, models.PlanQueued, planChange2.NewStatus)
	assert.Equal(t, "plan", planChange2.Path)
}

// TestStateMachine_StatusChangesCarryPath verifies that across a full run, every
// plan and apply change carries its node's relative path.
func TestStateMachine_StatusChangesCarryPath(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	sm := New(run)

	// Drive the plan through to finished so the apply is created and awaiting approval.
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))
	require.NoError(t, run.Apply().SetStatus(models.ApplyPending))
	require.NoError(t, run.Apply().SetStatus(models.ApplyQueued))
	require.NoError(t, run.Apply().SetStatus(models.ApplyRunning))
	require.NoError(t, run.Apply().SetStatus(models.ApplyFinished))

	sawPlan, sawApply := false, false
	for _, c := range sm.GetStatusChanges() {
		switch change := c.(type) {
		case PlanStatusChange:
			assert.Equal(t, "plan", change.Path)
			sawPlan = true
		case ApplyStatusChange:
			assert.Equal(t, "apply", change.Path)
			sawApply = true
		}
	}
	assert.True(t, sawPlan, "expected at least one plan change")
	assert.True(t, sawApply, "expected at least one apply change")
}
