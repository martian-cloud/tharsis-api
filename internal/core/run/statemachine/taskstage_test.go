package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// newMultiCheckPostPlanRun builds a non-speculative run whose single post-plan task stage owns TWO
// policy checks, to exercise the stage node's aggregate verdict across more than one child (the
// single-check tests in policycheck_test.go cannot).
func newMultiCheckPostPlanRun() (*RunNode, *TaskStageNode) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	stage := NewTaskStageNode("post-stage", models.RunTaskStageNamePostPlan, models.RunTaskStageCreated)
	stage.AddPolicyCheckNode(NewPolicyCheckNode("check-a", "post_plan.opa", models.PolicyKindOPA, models.PolicyCheckCreated))
	stage.AddPolicyCheckNode(NewPolicyCheckNode("check-b", "post_plan.opa2", models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(stage)
	New(run)
	return run, stage
}

// TestTaskStage_AggregateVerdict_AllClearedCompletes verifies a stage with multiple checks only
// completes (and clears the run) once EVERY check has cleared: the run stays post_plan_running while
// one check is still passing.
func TestTaskStage_AggregateVerdict_AllClearedCompletes(t *testing.T) {
	run, stage := newMultiCheckPostPlanRun()
	drivePlanToFinished(t, run)

	require.Equal(t, models.RunPostPlanRunning, run.Status())
	require.Equal(t, models.RunTaskStageRunning, stage.Status())

	a, b := stage.PolicyChecks()[0], stage.PolicyChecks()[1]
	require.NoError(t, a.SetStatus(models.PolicyCheckRunning))
	require.NoError(t, a.SetStatus(models.PolicyCheckPassed))

	// One check cleared, one still queued: no verdict yet.
	assert.Equal(t, models.RunTaskStageRunning, stage.Status())
	assert.Equal(t, models.RunPostPlanRunning, run.Status())

	require.NoError(t, b.SetStatus(models.PolicyCheckRunning))
	require.NoError(t, b.SetStatus(models.PolicyCheckPassed))

	// Both cleared: the stage completes and the run advances to planned.
	assert.Equal(t, models.RunTaskStageCompleted, stage.Status())
	assert.Equal(t, models.RunPlanned, run.Status())
}

// TestTaskStage_AggregateVerdict_OneErroredFailsStage verifies that a single errored check fails the
// whole stage (and the run), regardless of the other checks' status.
func TestTaskStage_AggregateVerdict_OneErroredFailsStage(t *testing.T) {
	run, stage := newMultiCheckPostPlanRun()
	drivePlanToFinished(t, run)

	a := stage.PolicyChecks()[0]
	require.NoError(t, a.SetStatus(models.PolicyCheckRunning))
	require.NoError(t, a.SetStatus(models.PolicyCheckErrored))

	assert.Equal(t, models.RunTaskStageErrored, stage.Status())
	assert.Equal(t, models.RunErrored, run.Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestTaskStage_AggregateVerdict_SoftFailAwaits verifies that with one check passed and another
// soft-failed the stage blocks on a human decision (awaiting_decision), projecting the run onto
// post_plan_awaiting_decision.
func TestTaskStage_AggregateVerdict_SoftFailAwaits(t *testing.T) {
	run, stage := newMultiCheckPostPlanRun()
	drivePlanToFinished(t, run)

	a, b := stage.PolicyChecks()[0], stage.PolicyChecks()[1]
	require.NoError(t, a.SetStatus(models.PolicyCheckRunning))
	require.NoError(t, a.SetStatus(models.PolicyCheckPassed))
	require.NoError(t, b.SetStatus(models.PolicyCheckRunning))
	require.NoError(t, b.SetStatus(models.PolicyCheckSoftFailed))

	assert.Equal(t, models.RunTaskStageAwaitingOverride, stage.Status())
	assert.Equal(t, models.RunPostPlanAwaitingDecision, run.Status())

	// Overriding the soft-failed check clears the stage.
	require.NoError(t, b.SetStatus(models.PolicyCheckOverridden))
	assert.Equal(t, models.RunTaskStageCompleted, stage.Status())
	assert.Equal(t, models.RunPlanned, run.Status())
}

// TestGetStatusChanges_IncludesPrePlanChanges guards the fix for GetStatusChanges previously
// iterating only the post-plan checks: a pre-plan run's stage AND check status changes must be
// returned so they are persisted.
func TestGetStatusChanges_IncludesPrePlanChanges(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		TaskStages: []*models.RunTaskStage{
			{ID: "stage-pre", StageName: models.RunTaskStageNamePrePlan, Status: models.RunTaskStageCreated, PolicyChecks: []*models.PolicyCheck{
				{ID: "pre-1", StageName: models.RunTaskStageNamePrePlan, CheckType: models.PolicyKindOPA, Status: models.PolicyCheckCreated},
			}},
		},
	}

	// Advancing the freshly-created run readies its workspace-gated pre-plan stage (created -> pending);
	// that change so it is persisted, and no check is queued yet.
	changes, err := AdvanceRun(r)
	require.NoError(t, err)

	var sawStagePending, sawCheckQueued bool
	for _, c := range changes {
		switch sc := c.(type) {
		case TaskStageStatusChange:
			if sc.Path == string(models.RunTaskStageNamePrePlan) && sc.NewStatus == models.RunTaskStagePending {
				sawStagePending = true
			}
		case PolicyCheckStatusChange:
			if sc.NewStatus == models.PolicyCheckQueued {
				sawCheckQueued = true
			}
		}
	}
	assert.True(t, sawStagePending, "pre-plan task stage pending change should be returned")
	assert.False(t, sawCheckQueued, "no check is queued before the stage is admitted")

	// Advancing the run (the admitter having acquired the slot) starts the stage and queues its check,
	// and those changes are returned too.
	changes, err = AdvanceRun(r)
	require.NoError(t, err)

	var sawStageRunning bool
	sawCheckQueued = false
	for _, c := range changes {
		switch sc := c.(type) {
		case TaskStageStatusChange:
			if sc.Path == string(models.RunTaskStageNamePrePlan) && sc.NewStatus == models.RunTaskStageRunning {
				sawStageRunning = true
			}
		case PolicyCheckStatusChange:
			if sc.NewStatus == models.PolicyCheckQueued {
				sawCheckQueued = true
			}
		}
	}
	assert.True(t, sawStageRunning, "pre-plan task stage running change should be returned")
	assert.True(t, sawCheckQueued, "pre-plan policy check status change should be returned")
}
