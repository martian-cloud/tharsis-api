package run

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/run/runtest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// followErr follows the run to the end, returning the error that ended following.
func followErr(h *harness, opts *Options) error {
	_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING, h.created()...), opts)
	return err
}

// overrideOnlyResults is a blocked check that only an override clears, as it has no approval rules.
func overrideOnlyResults() *pb.PolicyCheckResults {
	results := runtest.SoftFailedResults("c1", 1, 0)
	results.RunGate.ApprovalRules = nil
	return results
}

func overriddenGate() *pb.RunGate {
	g := overrideOnlyResults().RunGate
	g.Status = pb.RunGateStatus_RUN_GATE_STATUS_OVERRIDDEN
	return g
}

func TestFollow_GateOverride(t *testing.T) {
	t.Run("override is performed by the core and reported", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = overrideOnlyResults()
		h.runs.OnOverride = func(f *runtest.Runs) *pb.RunGate {
			f.Set(clearedPreApply())
			return overriddenGate()
		}
		d := &scriptedDecider{gates: []GateDecision{{Action: GateOverride}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 1, h.runs.Overridden)
		require.Len(t, d.gateReqs, 1)
		assert.True(t, d.gateReqs[0].CanOverride)
		assert.False(t, d.gateReqs[0].CanApprove)
		require.Len(t, find[GateOverridden](h.rec), 1)
	})

	t.Run("a denied override asks again without it", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = overrideOnlyResults()
		h.runs.OverrideErr = status.Error(codes.PermissionDenied, "not allowed to override")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateOverride}, {Action: GateLeave}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		require.Len(t, d.gateReqs, 2)
		assert.False(t, d.gateReqs[1].CanOverride, "the denied gate isn't offered again")
		denied := find[GateDenied](h.rec)
		require.Len(t, denied, 1)
		assert.Equal(t, GateOverride, denied[0].Action)
	})

	t.Run("an override with nothing to override is an error", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0) // has approval rules
		d := &scriptedDecider{gates: []GateDecision{{Action: GateOverride}}}

		err := followErr(h, h.options(d))

		assert.ErrorContains(t, err, "no gate can be overridden")
	})

	t.Run("a conflicting override re-reads the run", func(t *testing.T) {
		h := newHarness(awaitingPreApply(), awaitingPreApply(), awaitingPreApply(), clearedPreApply())
		h.runs.Results["c1"] = overrideOnlyResults()
		h.runs.OverrideErr = status.Error(codes.FailedPrecondition, "the gate was already cleared")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateOverride}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Empty(t, find[GateOverridden](h.rec))
	})

	t.Run("any other override failure ends following", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = overrideOnlyResults()
		h.runs.OverrideErr = status.Error(codes.Internal, "boom")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateOverride}}}

		err := followErr(h, h.options(d))

		assert.ErrorContains(t, err, "failed to override run gate")
	})
}

func TestFollow_GateApproveFailures(t *testing.T) {
	t.Run("a conflicting approval re-reads the run", func(t *testing.T) {
		h := newHarness(awaitingPreApply(), awaitingPreApply(), awaitingPreApply(), clearedPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		h.runs.ApproveErr = status.Error(codes.AlreadyExists, "the gate was already approved")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateApprove}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Empty(t, find[GateApproved](h.rec))
	})

	t.Run("any other approval failure ends following", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		h.runs.ApproveErr = status.Error(codes.Internal, "boom")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateApprove}}}

		err := followErr(h, h.options(d))

		assert.ErrorContains(t, err, "failed to approve run gate")
	})

	t.Run("approvals still needed from others are waited on", func(t *testing.T) {
		h := newHarness(awaitingPreApply(), awaitingPreApply(), awaitingPreApply(), awaitingPreApply(), clearedPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 2, 0)
		h.runs.OnApprove = func(f *runtest.Runs) *pb.RunGate {
			f.Results["c1"] = runtest.SoftFailedResults("c1", 2, 1) // the approval shows in the results
			return f.Results["c1"].RunGate                          // still pending
		}
		d := &scriptedDecider{gates: []GateDecision{{Action: GateApprove}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 1, h.runs.Approved)
		assert.Len(t, d.gateReqs, 1, "this caller isn't asked again for a gate it approved")
		approved := find[GateApproved](h.rec)
		require.Len(t, approved, 1)
		assert.False(t, approved[0].AllApprovalsIn)
		var others *Waiting
		for _, w := range find[Waiting](h.rec) {
			if w.Reason == WaitOtherApprovers {
				others = &w
			}
		}
		require.NotNil(t, others, "waits on the other approvers")
		assert.Equal(t, "1 of 2 approvals", others.Detail)
	})
}

func TestFollow_DiscardFailures(t *testing.T) {
	conflict := status.Error(codes.AlreadyExists, "the run is no longer waiting on a decision")

	t.Run("a conflicting plan discard is reported and following goes on", func(t *testing.T) {
		h := newHarness(planned(), planned(), runtest.AppliedRun())
		h.runs.DiscardErr = conflict
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanDiscard}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		conflicts := find[ActionConflict](h.rec)
		require.Len(t, conflicts, 1)
		assert.Equal(t, "discard", conflicts[0].Action)
	})

	t.Run("any other plan discard failure ends following", func(t *testing.T) {
		h := newHarness(planned())
		h.runs.DiscardErr = status.Error(codes.Internal, "boom")
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanDiscard}}}

		err := followErr(h, h.options(d))

		assert.ErrorContains(t, err, "failed to discard run")
	})

	t.Run("a conflicting gate discard is reported and following goes on", func(t *testing.T) {
		h := newHarness(awaitingPreApply(), awaitingPreApply(), awaitingPreApply(), clearedPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		h.runs.DiscardErr = conflict
		d := &scriptedDecider{gates: []GateDecision{{Action: GateDiscard}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		require.Len(t, find[ActionConflict](h.rec), 1)
	})

	t.Run("any other gate discard failure ends following", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		h.runs.DiscardErr = status.Error(codes.Internal, "boom")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateDiscard}}}

		err := followErr(h, h.options(d))

		assert.ErrorContains(t, err, "failed to discard run")
	})

	t.Run("an unknown decision ends following", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		d := &scriptedDecider{gates: []GateDecision{{Action: GateAction(99)}}}

		err := followErr(h, h.options(d))

		assert.ErrorContains(t, err, "unknown gate decision")
	})
}

func TestFollow_TaskStages(t *testing.T) {
	const completed = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED
	const running = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_RUNNING

	applying := func(stageStatus pb.RunTaskStageStatus, checks ...*pb.PolicyCheck) *pb.Run {
		return runtest.NewRun(pb.RunStatus_APPLYING,
			runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"),
			runtest.WithApply(pb.ApplyStatus_RUNNING, "apply-job"),
			runtest.WithStage(runtest.PreApply, stageStatus, checks...))
	}
	passed := func(jobID string) *pb.PolicyCheck {
		return runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED, jobID)
	}
	passedResults := func() *pb.PolicyCheckResults {
		results := runtest.SoftFailedResults("c1", 1, 0)
		results.PolicyCheck.Status = pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED
		results.RunGate = nil
		return results
	}
	preApplyStarts := func(h *harness) []StageStarted {
		var out []StageStarted
		for _, e := range find[StageStarted](h.rec) {
			if e.Stage.Kind == StageTask && e.Stage.TaskStage == runtest.PreApply {
				out = append(out, e)
			}
		}
		return out
	}

	t.Run("a stage that evaluates again is shown again as a retry", func(t *testing.T) {
		// The first snapshot is served twice: the plan's logs stream while it is read, so the stages
		// after the plan only see the next one.
		h := newHarness(
			applying(completed, passed("check-job-1")),
			applying(completed, passed("check-job-1")),
			applying(running, runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_RUNNING, "check-job-2")),
			runtest.AppliedRun(runtest.WithStage(runtest.PreApply, completed, passed("check-job-2"))),
		)
		h.runs.Results["c1"] = passedResults()

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		starts := preApplyStarts(h)
		require.Len(t, starts, 2)
		assert.False(t, starts[0].Retry)
		assert.True(t, starts[1].Retry)
		assert.Len(t, find[PolicyCheckFinished](h.rec), 2, "the check is reported again for its new evaluation")
	})

	t.Run("a check run again between two reads is noticed by its job", func(t *testing.T) {
		h := newHarness(
			applying(completed, passed("check-job-1")),
			applying(completed, passed("check-job-1")),
			applying(completed, passed("check-job-2")),
			runtest.AppliedRun(runtest.WithStage(runtest.PreApply, completed, passed("check-job-2"))),
		)
		h.runs.Results["c1"] = passedResults()

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		starts := preApplyStarts(h)
		require.Len(t, starts, 2)
		assert.True(t, starts[1].Retry)
	})

	t.Run("a stage with no visible tasks is reported once", func(t *testing.T) {
		h := newHarness(applying(running), applying(running), runtest.AppliedRun(runtest.WithStage(runtest.PreApply, completed)))

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Len(t, find[NoVisibleTasks](h.rec), 1)
	})

	t.Run("a check of an unsupported type is reported once and not read", func(t *testing.T) {
		unsupported := runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED, "check-job")
		unsupported.CheckType = pb.PolicyCheckType_POLICY_CHECK_TYPE_UNSPECIFIED
		h := newHarness(
			applying(running, unsupported),
			applying(running, unsupported),
			runtest.AppliedRun(runtest.WithStage(runtest.PreApply, completed, unsupported)),
		)

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Len(t, find[UnsupportedPolicyCheck](h.rec), 1)
		assert.Empty(t, find[PolicyCheckFinished](h.rec))
	})

	t.Run("a stage blocked on something that can't be shown is waited on", func(t *testing.T) {
		unsupported := runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_SOFT_FAILED, "check-job")
		unsupported.CheckType = pb.PolicyCheckType_POLICY_CHECK_TYPE_UNSPECIFIED
		blocked := runtest.NewRun(pb.RunStatus_PRE_APPLY_AWAITING_DECISION,
			runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"),
			runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE, unsupported))
		h := newHarness(blocked, blocked, runtest.AppliedRun(runtest.WithStage(runtest.PreApply, completed)))

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		var waited bool
		for _, w := range find[Waiting](h.rec) {
			waited = waited || w.Reason == WaitUnsupportedTask
		}
		assert.True(t, waited)
	})

	t.Run("a stage blocked on something that can't be shown asks the decider with no gates", func(t *testing.T) {
		unsupported := runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_SOFT_FAILED, "check-job")
		unsupported.CheckType = pb.PolicyCheckType_POLICY_CHECK_TYPE_UNSPECIFIED
		blocked := runtest.NewRun(pb.RunStatus_PRE_APPLY_AWAITING_DECISION,
			runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"),
			runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE, unsupported))
		h := newHarness(blocked)
		d := &scriptedDecider{gates: []GateDecision{{Action: GateLeave, Reason: "nobody is there"}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		assert.False(t, outcome.LeaveChosen)
		assert.Equal(t, "nobody is there", outcome.LeaveReason)
		require.Len(t, d.gateReqs, 1)
		assert.Empty(t, d.gateReqs[0].Pending)
		assert.False(t, d.gateReqs[0].CanApprove)
		assert.False(t, d.gateReqs[0].CanOverride)
		assert.Empty(t, find[GatesPending](h.rec), "there are no gates to show")
	})

	t.Run("a pending stage waits on the workspace", func(t *testing.T) {
		h := newHarness(
			applying(pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_PENDING),
			applying(pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_PENDING),
			runtest.AppliedRun(runtest.WithStage(runtest.PreApply, completed)),
		)

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		var waited bool
		for _, w := range find[Waiting](h.rec) {
			waited = waited || w.Reason == WaitWorkspace
		}
		assert.True(t, waited)
	})

	t.Run("an errored stage fails the run at that stage", func(t *testing.T) {
		errored := runtest.NewRun(pb.RunStatus_ERRORED,
			runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"),
			runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_ERRORED))
		h := newHarness(errored)

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeFailed, outcome.Kind)
		assert.Equal(t, StageTask, outcome.Stage.Kind)
	})
}

func TestTerraformStates(t *testing.T) {
	t.Run("plan", func(t *testing.T) {
		for status, want := range map[pb.PlanStatus]nodeState{
			pb.PlanStatus_UNSPECIFIED: nodeNotStarted,
			pb.PlanStatus_CREATED:     nodeNotStarted,
			pb.PlanStatus_PENDING:     nodePending,
			pb.PlanStatus_QUEUED:      nodeQueued,
			pb.PlanStatus_RUNNING:     nodeRunning,
			pb.PlanStatus_FINISHED:    nodeFinished,
			pb.PlanStatus_ERRORED:     nodeFailed,
			pb.PlanStatus_CANCELED:    nodeFailed,
		} {
			assert.Equal(t, want, planState(status), status.String())
		}
	})

	t.Run("apply", func(t *testing.T) {
		for status, want := range map[pb.ApplyStatus]nodeState{
			pb.ApplyStatus_UNSPECIFIED: nodeNotStarted,
			pb.ApplyStatus_CREATED:     nodeNotStarted,
			pb.ApplyStatus_PENDING:     nodePending,
			pb.ApplyStatus_QUEUED:      nodeQueued,
			pb.ApplyStatus_RUNNING:     nodeRunning,
			pb.ApplyStatus_FINISHED:    nodeFinished,
			pb.ApplyStatus_ERRORED:     nodeFailed,
			pb.ApplyStatus_CANCELED:    nodeFailed,
			pb.ApplyStatus_SKIPPED:     nodeSkipped,
		} {
			assert.Equal(t, want, applyState(status), status.String())
		}
	})
}

func TestWaitReason(t *testing.T) {
	for runStatus, want := range map[pb.RunStatus]WaitReason{
		pb.RunStatus_PENDING:            WaitRunStarting,
		pb.RunStatus_PRE_PLAN_QUEUING:   WaitWorkspace,
		pb.RunStatus_PLAN_QUEUING:       WaitWorkspace,
		pb.RunStatus_PRE_APPLY_QUEUING:  WaitWorkspace,
		pb.RunStatus_APPLY_QUEUING:      WaitWorkspace,
		pb.RunStatus_PLAN_QUEUED:        WaitRunner,
		pb.RunStatus_APPLY_QUEUED:       WaitRunner,
		pb.RunStatus_PRE_PLAN_RUNNING:   WaitTasksRunning,
		pb.RunStatus_POST_PLAN_RUNNING:  WaitTasksRunning,
		pb.RunStatus_PRE_APPLY_RUNNING:  WaitTasksRunning,
		pb.RunStatus_POST_APPLY_RUNNING: WaitTasksRunning,
		pb.RunStatus_PLANNING:           0,
		pb.RunStatus_PLANNED:            0,
		pb.RunStatus_APPLIED:            0,
		pb.RunStatus_UNSPECIFIED:        0,
	} {
		assert.Equal(t, want, waitReason(runStatus), runStatus.String())
	}
}

func TestErrorClassification(t *testing.T) {
	for code, want := range map[codes.Code]struct{ conflict, transient bool }{
		codes.AlreadyExists:      {conflict: true},
		codes.FailedPrecondition: {conflict: true},
		codes.Aborted:            {conflict: true, transient: true},
		codes.Unavailable:        {transient: true},
		codes.ResourceExhausted:  {transient: true},
		codes.DeadlineExceeded:   {transient: true},
		codes.Internal:           {},
		codes.NotFound:           {},
		codes.PermissionDenied:   {},
		codes.InvalidArgument:    {},
		codes.OK:                 {},
	} {
		err := status.Error(code, "x")
		assert.Equal(t, want.conflict, isStateConflict(err), "conflict: %s", code)
		assert.Equal(t, want.transient, isTransient(err), "transient: %s", code)
	}
	assert.True(t, isPermissionDenied(status.Error(codes.PermissionDenied, "x")))
	assert.False(t, isPermissionDenied(status.Error(codes.Internal, "x")))
}

func TestStageRestarted(t *testing.T) {
	const (
		created  = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED
		pending  = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_PENDING
		running  = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_RUNNING
		awaiting = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE
		done     = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED
		errored  = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_ERRORED
		canceled = pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CANCELED
	)
	for _, prev := range []pb.RunTaskStageStatus{awaiting, done, errored, canceled} {
		for _, cur := range []pb.RunTaskStageStatus{created, pending, running} {
			assert.True(t, isStageRestarted(prev, cur), "%s then %s", prev, cur)
		}
		assert.False(t, isStageRestarted(prev, done), "%s then completed", prev)
	}
	for _, prev := range []pb.RunTaskStageStatus{created, pending, running} {
		assert.False(t, isStageRestarted(prev, running), "%s then running", prev)
	}
}

func TestCheckHelpers(t *testing.T) {
	assert.True(t, isSupportedCheckType(pb.PolicyCheckType_POLICY_CHECK_TYPE_OPA))
	assert.True(t, isSupportedCheckType(pb.PolicyCheckType_POLICY_CHECK_TYPE_MODULE_ATTESTATION))
	assert.False(t, isSupportedCheckType(pb.PolicyCheckType_POLICY_CHECK_TYPE_UNSPECIFIED))

	for s, want := range map[pb.PolicyCheckStatus]bool{
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED:      true,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_SOFT_FAILED: true,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_OVERRIDDEN:  true,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_ERRORED:     true,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_CREATED:     false,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_QUEUED:      false,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_RUNNING:     false,
		pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PENDING:     false,
	} {
		assert.Equal(t, want, isCheckFinal(s), s.String())
	}
}

func TestFinalOutcome(t *testing.T) {
	stage := StageRef{Name: "Plan", Kind: StagePlan}

	failed := finalOutcome(runtest.NewRun(pb.RunStatus_ERRORED), stage)
	assert.Equal(t, OutcomeFailed, failed.Kind)
	assert.Equal(t, stage, failed.Stage)

	completed := finalOutcome(runtest.AppliedRun(), stage)
	assert.Equal(t, OutcomeCompleted, completed.Kind)
	assert.Equal(t, StageRef{}, completed.Stage)
}
