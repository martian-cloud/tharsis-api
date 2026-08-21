package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// opaCheckPath is the run-relative path of the OPA post-plan policy check node.
const opaCheckPath = "post_plan/opa"

// stageRunConfig configures the run node built by newStageRunWith.
type stageRunConfig struct {
	hasChanges bool
	withApply  bool
	autoApply  bool
}

// newStageRunWith builds a run node with a plan, a single post-plan OPA policy check, and an
// apply node only when withApply is true (a speculative run has none). hasChanges controls
// whether the plan reports changes.
func newStageRunWith(cfg stageRunConfig) *RunNode {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, cfg.hasChanges))
	if cfg.withApply {
		run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, cfg.autoApply))
	}
	stage := NewTaskStageNode("post-stage", models.RunTaskStageNamePostPlan, models.RunTaskStageCreated)
	stage.AddPolicyCheckNode(NewPolicyCheckNode("check-1", opaCheckPath, models.PolicyKindOPA, models.PolicyCheckCreated))
	run.AddTaskStageNode(stage)
	New(run)
	return run
}

// newStageRun builds a non-speculative run with changes, an apply, and a post-plan OPA check.
func newStageRun(autoApply bool) *RunNode {
	return newStageRunWith(stageRunConfig{hasChanges: true, withApply: true, autoApply: autoApply})
}

// check returns the run's single post-plan policy check node.
func check(run *RunNode) *PolicyCheckNode {
	return run.TaskStage(models.RunTaskStageNamePostPlan).PolicyChecks()[0]
}

func drivePlanToFinished(t *testing.T, run *RunNode) {
	t.Helper()
	require.NoError(t, run.Plan().SetStatus(models.PlanPending))
	require.NoError(t, run.Plan().SetStatus(models.PlanQueued))
	require.NoError(t, run.Plan().SetStatus(models.PlanRunning))
	require.NoError(t, run.Plan().SetStatus(models.PlanFinished))
}

// driveCheckToRunning advances the check from queued (where plan-finished leaves it) to
// running, mirroring the policy-eval job being claimed and started.
func driveCheckToRunning(t *testing.T, run *RunNode) {
	t.Helper()
	require.NoError(t, check(run).SetStatus(models.PolicyCheckRunning))
}

// TestPolicyCheckTransitions verifies the check lifecycle edges (and rejects illegal ones).
func TestPolicyCheckTransitions(t *testing.T) {
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckCreated, models.PolicyCheckQueued))
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckQueued, models.PolicyCheckRunning))
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckQueued, models.PolicyCheckCanceled))
	for _, to := range []models.PolicyCheckStatus{models.PolicyCheckPassed, models.PolicyCheckSoftFailed, models.PolicyCheckErrored, models.PolicyCheckCanceled} {
		assert.Truef(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckRunning, to), "running -> %s should be allowed", to)
	}
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckSoftFailed, models.PolicyCheckOverridden))
	// Retry: an errored/canceled/soft-failed check returns to pending, and its stage re-queues it from
	// there. It must not go straight back to queued — queueing creates the check's job, and a retried
	// check on a workspace-gated stage has to wait for the stage to be re-admitted first. soft_failed is
	// retryable as well as overridable, so a policy that has since been fixed can be re-evaluated.
	for _, from := range []models.PolicyCheckStatus{models.PolicyCheckErrored, models.PolicyCheckCanceled, models.PolicyCheckSoftFailed} {
		assert.Truef(t, canTransitionTo(policyCheckTransitions, from, models.PolicyCheckPending), "%s -> pending (retry) should be allowed", from)
		assert.Falsef(t, canTransitionTo(policyCheckTransitions, from, models.PolicyCheckQueued), "%s -> queued should not skip pending", from)
	}
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckPending, models.PolicyCheckQueued))
	// A retry re-evaluates rather than decides: it can never reach a verdict on its own.
	assert.False(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckSoftFailed, models.PolicyCheckPassed))
	// A pending check can still be settled by the run terminating, but never skips ahead to a verdict.
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckPending, models.PolicyCheckCanceled))
	assert.True(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckPending, models.PolicyCheckSkipped))
	assert.False(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckPending, models.PolicyCheckRunning))
	// Illegal: created cannot jump straight to running or a verdict, and passed is absorbing.
	assert.False(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckCreated, models.PolicyCheckRunning))
	assert.False(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckQueued, models.PolicyCheckPassed))
	assert.False(t, canTransitionTo(policyCheckTransitions, models.PolicyCheckPassed, models.PolicyCheckRunning))
}

// TestRunTransitions_CheckRetry verifies the retry resets that return a run to post_plan_running when
// its policy check node is retried — from an ended run (an errored/canceled check) and from one parked
// at the gate (a soft-failed check retried instead of decided).
func TestRunTransitions_CheckRetry(t *testing.T) {
	assert.True(t, canTransitionTo(runTransitions, models.RunErrored, models.RunPostPlanRunning))
	assert.True(t, canTransitionTo(runTransitions, models.RunCanceled, models.RunPostPlanRunning))
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanAwaitingDecision, models.RunPostPlanRunning))
	// The gated stages re-enter at their *_queuing status instead, because the run gave up its workspace
	// slot when it parked at the gate and must be re-admitted before evaluating again.
	assert.True(t, canTransitionTo(runTransitions, models.RunPrePlanAwaitingDecision, models.RunPrePlanQueuing))
	assert.True(t, canTransitionTo(runTransitions, models.RunPreApplyAwaitingDecision, models.RunPreApplyQueuing))
	assert.False(t, canTransitionTo(runTransitions, models.RunPrePlanAwaitingDecision, models.RunPrePlanRunning),
		"a gated stage must not restart without re-admission")
	// A discarded run re-enters at the same statuses on an undiscard, which re-runs the gate the discard
	// canceled. Retrying a check directly is still rejected by RetryRunNode: the undiscard is the retry,
	// and it is what records the run being revived.
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPostPlanRunning))
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPrePlanQueuing))
	assert.True(t, canTransitionTo(runTransitions, models.RunDiscarded, models.RunPreApplyQueuing))
}

// TestRunTransitions_PostPlan verifies the run-status edges bracketing the policy checks.
func TestRunTransitions_PostPlan(t *testing.T) {
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanning, models.RunPostPlanRunning))
	// Automated evaluation -> blocked waiting for human
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanRunning, models.RunPostPlanAwaitingDecision))
	// Automated evaluation -> all checks cleared
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanRunning, models.RunPostPlanCompleted))
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanRunning, models.RunErrored))
	// Discard is only reachable from the awaiting-decision state, not from automated evaluation
	assert.False(t, canTransitionTo(runTransitions, models.RunPostPlanRunning, models.RunDiscarded))
	// Awaiting decision: human approves/overrides -> completed; or cancel/discard
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanAwaitingDecision, models.RunPostPlanCompleted))
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanAwaitingDecision, models.RunDiscarded))
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanAwaitingDecision, models.RunCanceled))
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanCompleted, models.RunPlanned))
	// An auto-apply run's cleared post-plan stage readies the apply, which then waits for the slot. It
	// does not pass through planned (it is already approved), and the same is true of an auto-apply run
	// with no post-plan stage, whose plan readies the apply directly.
	assert.True(t, canTransitionTo(runTransitions, models.RunPostPlanCompleted, models.RunApplyQueuing))
	assert.True(t, canTransitionTo(runTransitions, models.RunPlanning, models.RunApplyQueuing))
	// The plan can never skip its own post-plan gate, however: policy must clear before the apply.
	assert.False(t, canTransitionTo(runTransitions, models.RunPlanning, models.RunApplying))
}

// TestHandlePlanSucceeded_WithoutChecks verifies the plan-finished routing is unchanged
// when there are no policy checks: a manual run lands at planned, apply left created.
func TestHandlePlanSucceeded_WithoutChecks(t *testing.T) {
	run := NewRunNode(models.RunPending)
	run.SetPlanNode(NewPlanNode("plan", models.PlanCreated, true))
	run.SetApplyNode(NewApplyNode("apply", models.ApplyCreated, false))
	New(run)

	drivePlanToFinished(t, run)

	assert.Equal(t, models.RunPlanned, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestHandlePlanSucceeded_WithChecks verifies that when a policy check is present the run
// moves to post_plan_running and the check is queued, instead of going straight to planned.
func TestHandlePlanSucceeded_WithChecks(t *testing.T) {
	run := newStageRun(false)

	drivePlanToFinished(t, run)

	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, check(run).Status())
	// The apply must not have advanced: policy has not cleared yet.
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestHandlePostPlanVerdict_Passed verifies a passed check clears policy: the run reaches
// post_plan_completed then planned (manual run), leaving the apply awaiting approval.
func TestHandlePostPlanVerdict_Passed(t *testing.T) {
	run := newStageRun(false)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)

	require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunPlanned, run.Status())
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())

	// The run recorded its passage through post_plan_completed.
	var sawCompleted bool
	for _, c := range run.GetStatusChanges() {
		if rc, ok := c.(RunStatusChange); ok && rc.NewStatus == models.RunPostPlanCompleted {
			sawCompleted = true
		}
	}
	assert.True(t, sawCompleted, "run should pass through post_plan_completed")
}

// TestHandlePostPlanVerdict_PassedAutoApply verifies an auto-apply run whose check clears
// advances all the way to queuing_apply with the apply readied.
func TestHandlePostPlanVerdict_PassedAutoApply(t *testing.T) {
	run := newStageRun(true)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)

	require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))

	// The readied apply is pending, so the run reports apply_queuing: it still has to be admitted.
	assert.Equal(t, models.RunApplyQueuing, run.Status())
	assert.Equal(t, models.ApplyPending, run.Apply().Status())

	// The run went from post_plan_completed straight to apply_queuing: planned means "waiting on a
	// human to approve the apply", and a pre-approved run is never in that state.
	var statuses []models.RunStatus
	for _, c := range run.GetStatusChanges() {
		if rc, ok := c.(RunStatusChange); ok {
			statuses = append(statuses, rc.NewStatus)
		}
	}
	assert.NotContains(t, statuses, models.RunPlanned, "an auto-apply run should never enter planned")
	assert.Subset(t, statuses, []models.RunStatus{models.RunPostPlanCompleted, models.RunApplyQueuing})
}

// TestHandlePostPlanVerdict_Overridden verifies an overridden check clears policy just like
// a passed one.
func TestHandlePostPlanVerdict_Overridden(t *testing.T) {
	run := newStageRun(false)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)

	require.NoError(t, check(run).SetStatus(models.PolicyCheckSoftFailed))
	assert.Equal(t, models.RunPostPlanAwaitingDecision, run.Status()) // blocked awaiting human decision

	require.NoError(t, check(run).SetStatus(models.PolicyCheckOverridden))
	assert.Equal(t, models.RunPlanned, run.Status())
}

// TestHandlePostPlanVerdict_Errored verifies a failed check errors the run and the run's
// terminal listener marks the never-started apply skipped.
func TestHandlePostPlanVerdict_Errored(t *testing.T) {
	run := newStageRun(false)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)

	require.NoError(t, check(run).SetStatus(models.PolicyCheckErrored))

	assert.Equal(t, models.RunErrored, run.Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestHandlePostPlanVerdict_AwaitingOverride verifies that awaiting_override advances the run
// to post_plan_awaiting_decision to signal that human approval is needed.
func TestHandlePostPlanVerdict_AwaitingOverride(t *testing.T) {
	run := newStageRun(false)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)
	require.Equal(t, models.RunPostPlanRunning, run.Status())

	require.NoError(t, check(run).SetStatus(models.PolicyCheckSoftFailed))

	assert.Equal(t, models.RunPostPlanAwaitingDecision, run.Status())
	assert.Equal(t, models.PolicyCheckSoftFailed, check(run).Status())
}

// TestHandleCheckPending_RetryFromErrored verifies retrying an errored check returns the run to
// post_plan_running and restores the skipped apply to created. Setting the check to pending is the
// whole retry: the stage reacts to it, and because the post-plan stage is not workspace-gated it goes
// straight back to running, which re-queues the check.
func TestHandleCheckPending_RetryFromErrored(t *testing.T) {
	run := newStageRun(false)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)
	require.NoError(t, check(run).SetStatus(models.PolicyCheckErrored))
	require.Equal(t, models.RunErrored, run.Status())
	require.Equal(t, models.ApplySkipped, run.Apply().Status())

	require.NoError(t, check(run).SetStatus(models.PolicyCheckPending))

	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.RunTaskStageRunning, run.TaskStage(models.RunTaskStageNamePostPlan).Status(),
		"an ungated stage restarts without waiting for admission")
	assert.Equal(t, models.PolicyCheckQueued, check(run).Status(), "the stage re-queues the check")
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestHandleCheckPending_RetryFromSoftFailed verifies retrying a soft-failed check restarts its stage
// from awaiting_override the same way an errored one restarts from errored. The run was parked at
// post_plan_awaiting_decision waiting on a human; the retry takes it back to evaluating instead, which
// is the point — a policy that has since been fixed is re-evaluated rather than overridden.
func TestHandleCheckPending_RetryFromSoftFailed(t *testing.T) {
	run := newStageRun(false)
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)
	require.NoError(t, check(run).SetStatus(models.PolicyCheckSoftFailed))
	require.Equal(t, models.RunPostPlanAwaitingDecision, run.Status())
	require.Equal(t, models.RunTaskStageAwaitingOverride, run.TaskStage(models.RunTaskStageNamePostPlan).Status())

	require.NoError(t, check(run).SetStatus(models.PolicyCheckPending))

	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.RunTaskStageRunning, run.TaskStage(models.RunTaskStageNamePostPlan).Status(),
		"an ungated stage restarts without waiting for admission")
	assert.Equal(t, models.PolicyCheckQueued, check(run).Status(), "the stage re-queues the check")
	// The apply was never skipped here — the run parked at the gate rather than terminating — so it is
	// still awaiting the stage's outcome.
	assert.Equal(t, models.ApplyCreated, run.Apply().Status())
}

// TestSetPolicyCheckStatus_TransitionsRun verifies the package-level entry function drives the
// check node and projects onto the run model.
func TestSetPolicyCheckStatus_TransitionsRun(t *testing.T) {
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
	// Drive the plan to finished so the run is post_plan_running and the check is queued.
	for _, s := range []models.PlanStatus{models.PlanPending, models.PlanQueued, models.PlanRunning, models.PlanFinished} {
		_, err := SetPlanStatus(r, s)
		require.NoError(t, err)
	}
	require.Equal(t, models.RunPostPlanRunning, r.Status)
	require.Equal(t, models.PolicyCheckQueued, r.AllPolicyChecks()[0].Status)

	// The policy-eval job starts (queued -> running), then reports its verdict.
	checkPath := r.AllPolicyChecks()[0].GetPath()
	_, err := SetPolicyCheckStatus(r, checkPath, models.PolicyCheckRunning)
	require.NoError(t, err)

	changes, err := SetPolicyCheckStatus(r, checkPath, models.PolicyCheckPassed)
	require.NoError(t, err)
	assert.NotEmpty(t, changes)
	assert.Equal(t, models.PolicyCheckPassed, r.AllPolicyChecks()[0].Status)
	assert.Equal(t, models.RunPlanned, r.Status)
}

// TestSetPolicyCheckStatus_NoCheckNodeErrors verifies updating a check status on a run without
// a matching check node is an error.
func TestSetPolicyCheckStatus_NoCheckNodeErrors(t *testing.T) {
	r := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPending,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCreated, HasChanges: true},
	}
	_, err := SetPolicyCheckStatus(r, opaCheckPath, models.PolicyCheckRunning)
	require.Error(t, err)
}

// TestHandlePlanSucceeded_SpeculativeWithChecks verifies a speculative run (no apply) whose plan
// finishes with changes still queues its post-plan checks — the policy verdict is meaningful even
// with nothing to apply — and finishes once the checks clear.
func TestHandlePlanSucceeded_SpeculativeWithChecks(t *testing.T) {
	run := newStageRunWith(stageRunConfig{hasChanges: true, withApply: false})

	drivePlanToFinished(t, run)

	// The check is queued and the run is evaluating policy, even without an apply.
	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, check(run).Status())
	assert.Nil(t, run.Apply())

	// Once the check clears there is nothing to apply, so the run finishes.
	driveCheckToRunning(t, run)
	require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
	assert.Equal(t, models.PolicyCheckPassed, check(run).Status())
}

// TestHandlePlanSucceeded_SpeculativeSoftFailOverride verifies a speculative run whose check
// soft-fails blocks awaiting a decision (the normal verdict applies), then finishes once the
// check is overridden.
//
// A speculative run's post-plan policies are resolved to advisory enforcement at creation
// (resolveRunPolicies), so a soft-failed verdict is not reachable for one in practice today. The
// transition stays supported and tested here because enforcement is a creation-time decision and the
// state machine must not encode it.
func TestHandlePlanSucceeded_SpeculativeSoftFailOverride(t *testing.T) {
	run := newStageRunWith(stageRunConfig{hasChanges: true, withApply: false})
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)

	require.NoError(t, check(run).SetStatus(models.PolicyCheckSoftFailed))
	assert.Equal(t, models.RunPostPlanAwaitingDecision, run.Status())

	require.NoError(t, check(run).SetStatus(models.PolicyCheckOverridden))
	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
}

// TestHandlePlanSucceeded_SpeculativeCheckErrored verifies a failed check errors a speculative run,
// just like a non-speculative one.
func TestHandlePlanSucceeded_SpeculativeCheckErrored(t *testing.T) {
	run := newStageRunWith(stageRunConfig{hasChanges: true, withApply: false})
	drivePlanToFinished(t, run)
	driveCheckToRunning(t, run)

	require.NoError(t, check(run).SetStatus(models.PolicyCheckErrored))

	assert.Equal(t, models.RunErrored, run.Status())
}

// TestHandlePlanSucceeded_NoChangesSkipsChecks verifies that when an appliable run's plan produced no
// changes the post-plan checks are skipped (not queued) and the run finishes, with the never-started
// apply marked skipped. There is no apply left for policy to gate, so evaluating it would decide
// nothing.
func TestHandlePlanSucceeded_NoChangesSkipsChecks(t *testing.T) {
	run := newStageRunWith(stageRunConfig{hasChanges: false, withApply: true})

	drivePlanToFinished(t, run)

	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
	assert.Equal(t, models.PolicyCheckSkipped, check(run).Status())
	assert.Equal(t, models.ApplySkipped, run.Apply().Status())
}

// TestHandlePlanSucceeded_SpeculativeNoChangesRunsChecks verifies a speculative run whose plan
// produced no changes still evaluates its post-plan checks: a speculative run's only output is the
// verdict, and the plan document still describes existing resources, provider versions and variables
// whether or not anything changed. The run finishes once the check clears, as it has no apply.
func TestHandlePlanSucceeded_SpeculativeNoChangesRunsChecks(t *testing.T) {
	run := newStageRunWith(stageRunConfig{hasChanges: false, withApply: false})

	drivePlanToFinished(t, run)

	assert.Equal(t, models.RunPostPlanRunning, run.Status())
	assert.Equal(t, models.PolicyCheckQueued, check(run).Status())
	assert.Nil(t, run.Apply())

	driveCheckToRunning(t, run)
	require.NoError(t, check(run).SetStatus(models.PolicyCheckPassed))

	assert.Equal(t, models.RunPlannedAndFinished, run.Status())
}
