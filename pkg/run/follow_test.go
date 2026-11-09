package run

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/run/runtest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// recorder is an Observer that records what it is told.
type recorder struct {
	mu     sync.Mutex
	events []Event
	logs   []string
	logErr error
}

func (r *recorder) Event(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) Logs(_ StageRef, chunk string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, chunk)
	return r.logErr
}

// kinds lists the recorded events' types, such as "run.StageStarted".
func (r *recorder) kinds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.events))
	for _, e := range r.events {
		out = append(out, fmt.Sprintf("%T", e))
	}
	return out
}

// find returns the recorded events of type T.
func find[T Event](r *recorder) []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []T
	for _, e := range r.events {
		if t, ok := e.(T); ok {
			out = append(out, t)
		}
	}
	return out
}

// scriptedDecider answers from scripted lists, recording each request; when a list runs out, it
// blocks until its context is canceled.
type scriptedDecider struct {
	mu        sync.Mutex
	plans     []PlanDecision
	gates     []GateDecision
	gateReqs  []GateRequest
	planCalls int
}

func (d *scriptedDecider) DecidePlan(ctx context.Context, _ *PlanRequest) (PlanDecision, error) {
	d.mu.Lock()
	d.planCalls++
	if len(d.plans) > 0 {
		next := d.plans[0]
		d.plans = d.plans[1:]
		d.mu.Unlock()
		return next, nil
	}
	d.mu.Unlock()
	<-ctx.Done()
	return PlanDecision{}, ctx.Err()
}

func (d *scriptedDecider) DecideGates(ctx context.Context, req *GateRequest) (GateDecision, error) {
	d.mu.Lock()
	d.gateReqs = append(d.gateReqs, *req)
	if len(d.gates) > 0 {
		next := d.gates[0]
		d.gates = d.gates[1:]
		d.mu.Unlock()
		return next, nil
	}
	d.mu.Unlock()
	<-ctx.Done()
	return GateDecision{}, ctx.Err()
}

type interrupterFunc func(context.Context, *pb.Run) InterruptAction

func (f interrupterFunc) OnInterrupt(ctx context.Context, run *pb.Run) InterruptAction {
	return f(ctx, run)
}

type harness struct {
	runs *runtest.Runs
	jobs *runtest.Jobs
	rec  *recorder
}

func newHarness(runs ...*pb.Run) *harness {
	return &harness{runs: runtest.NewRuns(runs...), jobs: &runtest.Jobs{}, rec: &recorder{}}
}

// options returns fast-polling options observed by the recorder.
func (h *harness) options(decider Decider) *Options {
	opts := &Options{Observer: h.rec, PollInterval: 5 * time.Millisecond, RetryDelay: time.Millisecond}
	if decider != nil {
		opts.Decider = decider
	}
	return opts
}

func (h *harness) follow(t *testing.T, opts *Options) *Outcome {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	outcome, err := Follow(ctx, runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING, h.created()...), opts)
	require.NoError(t, err)
	require.NotEqual(t, OutcomeInterrupted, outcome.Kind, "timed out following the run; events: %v", h.rec.kinds())
	return outcome
}

// created gives the created run the task stages of the first snapshot, since the stages followed
// are fixed from the run's shape when following begins.
func (h *harness) created() []runtest.RunOpt {
	first := h.runs.Queue[0]
	var opts []runtest.RunOpt
	for _, s := range first.TaskStages {
		opts = append(opts, runtest.WithStage(s.StageName, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED))
	}
	if first.Speculative {
		opts = append(opts, runtest.Speculative())
	}
	return opts
}

func planned(opts ...runtest.RunOpt) *pb.Run {
	return runtest.NewRun(pb.RunStatus_PLANNED, append([]runtest.RunOpt{runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job")}, opts...)...)
}

func awaitingPreApply() *pb.Run {
	return runtest.NewRun(pb.RunStatus_PRE_APPLY_AWAITING_DECISION,
		runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"),
		runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_AWAITING_OVERRIDE,
			runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_SOFT_FAILED, "check-job")))
}

func clearedPreApply() *pb.Run {
	return runtest.AppliedRun(runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED,
		runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_OVERRIDDEN, "check-job")))
}

func TestFollow_RequiresObserver(t *testing.T) {
	_, err := Follow(context.Background(), nil, runtest.NewRun(pb.RunStatus_PENDING), &Options{})
	assert.Error(t, err)
}

func TestFollow_Events(t *testing.T) {
	t.Run("reports each stage, its logs and the outcome", func(t *testing.T) {
		h := newHarness(runtest.AppliedRun())

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, []string{"logs for plan-job", "logs for apply-job"}, h.rec.logs)
		started := find[StageStarted](h.rec)
		require.Len(t, started, 2)
		assert.Equal(t, StagePlan, started[0].Stage.Kind)
		assert.Equal(t, StageApply, started[1].Stage.Kind)
	})

	t.Run("a waiting reason is reported once until something else is", func(t *testing.T) {
		pending := runtest.NewRun(pb.RunStatus_PLAN_QUEUED, runtest.WithPlan(pb.PlanStatus_QUEUED, ""))
		h := newHarness(pending, pending, pending, pending, runtest.AppliedRun())

		h.follow(t, h.options(nil))

		waits := find[Waiting](h.rec)
		require.NotEmpty(t, waits)
		assert.Equal(t, WaitRunner, waits[0].Reason)
		assert.Len(t, waits, 1, "the same reason isn't repeated")
	})

	t.Run("an observer's log error stops following", func(t *testing.T) {
		h := newHarness(runtest.AppliedRun())
		h.rec.logErr = errors.New("disk full")

		_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), h.options(nil))

		assert.ErrorContains(t, err, "disk full")
	})

	t.Run("SkipFinished passes over stages that had finished", func(t *testing.T) {
		applying := runtest.NewRun(pb.RunStatus_APPLYING,
			runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"), runtest.WithApply(pb.ApplyStatus_RUNNING, "apply-job"))
		h := newHarness(applying, runtest.AppliedRun())
		opts := h.options(nil)
		opts.SkipFinished = true

		outcome := h.follow(t, opts)

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, []string{"apply-job"}, h.jobs.Streamed(), "the plan isn't replayed")
		started := find[StageStarted](h.rec)
		require.Len(t, started, 1)
		assert.Equal(t, StageApply, started[0].Stage.Kind)
	})
}

func TestFollow_PlanDecision(t *testing.T) {
	t.Run("without a decider the plan is waited on from elsewhere", func(t *testing.T) {
		h := newHarness(planned(), planned(), runtest.AppliedRun())

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 0, h.runs.Applied)
		require.Len(t, find[DecisionDeferred](h.rec), 1)
		assert.Equal(t, StagePlanApproval, find[DecisionDeferred](h.rec)[0].Stage.Kind)
	})

	t.Run("apply is performed by the core", func(t *testing.T) {
		h := newHarness(planned())
		h.runs.OnApply = func(f *runtest.Runs) { f.Set(runtest.AppliedRun()) }
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanApply}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 1, h.runs.Applied)
		assert.Equal(t, 1, d.planCalls)
	})

	t.Run("discard is performed by the core", func(t *testing.T) {
		h := newHarness(planned())
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanDiscard}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDiscarded, outcome.Kind)
		assert.Equal(t, 1, h.runs.Discarded)
	})

	t.Run("leave detaches at plan approval", func(t *testing.T) {
		h := newHarness(planned())
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanLeave, Chosen: true}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		assert.Equal(t, StagePlanApproval, outcome.Stage.Kind)
		assert.True(t, outcome.LeaveChosen)
		assert.Empty(t, outcome.LeaveReason)
	})

	t.Run("a leave's reason is passed on in the outcome", func(t *testing.T) {
		h := newHarness(planned())
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanLeave, Reason: "nobody is there"}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		assert.False(t, outcome.TimedOut)
		assert.False(t, outcome.LeaveChosen, "a decision is the decider's own unless it says it was chosen")
		assert.Equal(t, "nobody is there", outcome.LeaveReason)
	})

	t.Run("wait elsewhere passes on the reason and isn't asked again", func(t *testing.T) {
		h := newHarness(planned(), planned(), planned(), runtest.AppliedRun())
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanWaitElsewhere, Reason: "nobody is there"}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 1, d.planCalls)
		deferred := find[DecisionDeferred](h.rec)
		require.Len(t, deferred, 1)
		assert.Equal(t, "nobody is there", deferred[0].Reason)
	})

	t.Run("the decider's context is canceled when the run moves on", func(t *testing.T) {
		h := newHarness(planned(), planned(), runtest.AppliedRun())
		d := &scriptedDecider{} // blocks until canceled

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Len(t, find[DecisionAbandoned](h.rec), 1)
	})

	t.Run("a conflicting apply isn't asked about again while the snapshot lags", func(t *testing.T) {
		h := newHarness(planned())
		h.runs.ApplyErr = status.Error(codes.AlreadyExists, "the apply phase has already been started")
		h.runs.OnApply = func(f *runtest.Runs) { f.Set(planned(), planned(), runtest.AppliedRun()) }
		d := &scriptedDecider{plans: []PlanDecision{{Action: PlanApply}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 1, d.planCalls)
	})

	t.Run("the wait for a decision made elsewhere times out", func(t *testing.T) {
		h := newHarness(planned())
		opts := h.options(nil)
		opts.WaitTimeout = 30 * time.Millisecond

		outcome := h.follow(t, opts)

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		assert.True(t, outcome.TimedOut)
	})
}

// planDeciderFunc is a Decider whose plan decisions come from a function.
type planDeciderFunc func(context.Context) (PlanDecision, error)

func (f planDeciderFunc) DecidePlan(ctx context.Context, _ *PlanRequest) (PlanDecision, error) {
	return f(ctx)
}
func (planDeciderFunc) DecideGates(context.Context, *GateRequest) (GateDecision, error) {
	return GateDecision{Action: GateLeave}, nil
}

func TestFollow_WaitTimeout(t *testing.T) {
	t.Run("an unanswered decider times out", func(t *testing.T) {
		h := newHarness(planned())
		opts := h.options(&scriptedDecider{}) // blocks until its context ends
		opts.WaitTimeout = 30 * time.Millisecond

		outcome := h.follow(t, opts)

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		assert.True(t, outcome.TimedOut)
		assert.Equal(t, StagePlanApproval, outcome.Stage.Kind)
		assert.Empty(t, find[DecisionAbandoned](h.rec), "timing out isn't the run moving on")
	})

	t.Run("one clock covers asking and then waiting elsewhere", func(t *testing.T) {
		h := newHarness(planned())
		opts := h.options(planDeciderFunc(func(context.Context) (PlanDecision, error) {
			time.Sleep(40 * time.Millisecond)
			return PlanDecision{Action: PlanWaitElsewhere}, nil
		}))
		opts.WaitTimeout = 60 * time.Millisecond

		start := time.Now()
		outcome := h.follow(t, opts)

		assert.True(t, outcome.TimedOut)
		assert.Less(t, time.Since(start), 100*time.Millisecond, "the wait elsewhere gets only what the prompt left")
	})

	t.Run("a decider's own error isn't taken for the timeout", func(t *testing.T) {
		h := newHarness(planned())
		ioErr := errors.New("terminal read failed")
		opts := h.options(planDeciderFunc(func(ctx context.Context) (PlanDecision, error) {
			<-ctx.Done()
			return PlanDecision{}, ioErr
		}))
		opts.WaitTimeout = 30 * time.Millisecond

		_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), opts)

		assert.ErrorIs(t, err, ioErr)
	})

	t.Run("a decider's own error isn't taken for the run moving on", func(t *testing.T) {
		h := newHarness(planned(), planned(), runtest.AppliedRun())
		ioErr := errors.New("terminal read failed")
		opts := h.options(planDeciderFunc(func(ctx context.Context) (PlanDecision, error) {
			<-ctx.Done()
			return PlanDecision{}, ioErr
		}))

		_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), opts)

		assert.ErrorIs(t, err, ioErr)
		assert.Empty(t, find[DecisionAbandoned](h.rec))
	})

	t.Run("without a timeout the default applies", func(t *testing.T) {
		h := newHarness(planned())
		var deadline time.Time
		opts := h.options(planDeciderFunc(func(ctx context.Context) (PlanDecision, error) {
			deadline, _ = ctx.Deadline()
			return PlanDecision{Action: PlanLeave}, nil
		}))
		opts.WaitTimeout = 0

		_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), opts)
		require.NoError(t, err)

		require.False(t, deadline.IsZero(), "a decision is never waited on forever")
		assert.WithinDuration(t, time.Now().Add(DefaultWaitTimeout), deadline, time.Minute)
	})
}

func TestFollow_GateDecision(t *testing.T) {
	approvedGate := func() *pb.RunGate {
		g := runtest.SoftFailedResults("c1", 1, 1).RunGate
		g.Status = pb.RunGateStatus_RUN_GATE_STATUS_APPROVED
		return g
	}

	t.Run("approve is performed by the core and reported", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		h.runs.OnApprove = func(f *runtest.Runs) *pb.RunGate {
			f.Set(clearedPreApply())
			return approvedGate()
		}
		d := &scriptedDecider{gates: []GateDecision{{Action: GateApprove}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Equal(t, 1, h.runs.Approved)
		require.Len(t, d.gateReqs, 1)
		assert.True(t, d.gateReqs[0].CanApprove)
		assert.False(t, d.gateReqs[0].CanOverride)
		require.Len(t, find[GatesPending](h.rec), 1)
		approved := find[GateApproved](h.rec)
		require.Len(t, approved, 1)
		assert.True(t, approved[0].AllApprovalsIn)
	})

	t.Run("a denied approval asks again without it", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		h.runs.ApproveErr = status.Error(codes.PermissionDenied, "not an eligible approver")
		d := &scriptedDecider{gates: []GateDecision{{Action: GateApprove}, {Action: GateLeave}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDetached, outcome.Kind)
		require.Len(t, d.gateReqs, 2)
		assert.False(t, d.gateReqs[1].CanApprove, "the denied gate isn't offered again")
		assert.Len(t, find[GateDenied](h.rec), 1)
	})

	t.Run("without a decider the gates are shown and waited on", func(t *testing.T) {
		h := newHarness(awaitingPreApply(), awaitingPreApply(), clearedPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 2, 1)

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
		assert.Len(t, find[GatesPending](h.rec), 1)
		waits := find[Waiting](h.rec)
		require.NotEmpty(t, waits)
		var elsewhere *Waiting
		for i := range waits {
			if waits[i].Reason == WaitDecisionElsewhere {
				elsewhere = &waits[i]
			}
		}
		require.NotNil(t, elsewhere)
		assert.Equal(t, "1 of 2 approvals", elsewhere.Detail)
	})

	t.Run("discard is performed by the core", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		h.runs.Results["c1"] = runtest.SoftFailedResults("c1", 1, 0)
		d := &scriptedDecider{gates: []GateDecision{{Action: GateDiscard}}}

		outcome := h.follow(t, h.options(d))

		assert.Equal(t, OutcomeDiscarded, outcome.Kind)
		assert.Equal(t, 1, h.runs.Discarded)
	})

	t.Run("an approve with nothing to approve is an error", func(t *testing.T) {
		h := newHarness(awaitingPreApply())
		results := runtest.SoftFailedResults("c1", 1, 0)
		results.RunGate.ApprovalRules = nil // only an override clears it
		h.runs.Results["c1"] = results
		d := &scriptedDecider{gates: []GateDecision{{Action: GateApprove}}}

		_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING, h.created()...), h.options(d))

		assert.ErrorContains(t, err, "no gate can be approved")
	})

	t.Run("advisory failures are collected on the outcome", func(t *testing.T) {
		h := newHarness(runtest.AppliedRun(runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_COMPLETED,
			runtest.NewCheck("c1", pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED, "check-job"))))
		results := runtest.SoftFailedResults("c1", 1, 0)
		results.PolicyCheck.Status = pb.PolicyCheckStatus_POLICY_CHECK_STATUS_PASSED
		results.PolicyResults[0].Policy.EnforcementLevel = pb.PolicyEnforcementLevel_POLICY_ENFORCEMENT_LEVEL_ADVISORY
		results.RunGate = nil
		h.runs.Results["c1"] = results

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, []string{"group/encrypted-buckets"}, outcome.Advisories)
		checks := find[PolicyCheckFinished](h.rec)
		require.Len(t, checks, 1)
		assert.True(t, checks[0].First)
	})
}

func TestFollow_Interrupt(t *testing.T) {
	interrupt := func(t *testing.T, h *harness, opts *Options) *Outcome {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		outcome, err := Follow(ctx, runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), opts)
		require.NoError(t, err)
		require.Equal(t, OutcomeInterrupted, outcome.Kind)
		return outcome
	}
	queued := func() *pb.Run {
		return runtest.NewRun(pb.RunStatus_PLAN_QUEUED, runtest.WithPlan(pb.PlanStatus_QUEUED, ""))
	}
	always := func(a InterruptAction) Interrupter {
		return interrupterFunc(func(context.Context, *pb.Run) InterruptAction { return a })
	}

	t.Run("without an interrupter the run is left going", func(t *testing.T) {
		h := newHarness(queued())

		outcome := interrupt(t, h, h.options(nil))

		assert.False(t, outcome.RunCanceled)
		assert.Equal(t, 0, h.runs.Count(&h.runs.Canceled))
		assert.Len(t, find[Interrupted](h.rec), 1)
	})

	t.Run("cancel cancels a run doing work", func(t *testing.T) {
		h := newHarness(queued())
		opts := h.options(nil)
		opts.Interrupter = always(InterruptCancelRunIfInProgress)

		outcome := interrupt(t, h, opts)

		assert.True(t, outcome.RunCanceled)
		assert.Equal(t, 1, h.runs.Count(&h.runs.Canceled))
	})

	t.Run("cancel leaves a run waiting on a decision", func(t *testing.T) {
		h := newHarness(planned())
		opts := h.options(nil)
		opts.Interrupter = always(InterruptCancelRunIfInProgress)

		outcome := interrupt(t, h, opts)

		assert.False(t, outcome.RunCanceled)
		assert.False(t, outcome.RunDiscarded)
		failed := find[InterruptFailed](h.rec)
		require.Len(t, failed, 1, "the interrupter is told the run was not canceled")
		assert.ErrorIs(t, failed[0].Err, errRunNotCancelable)
	})

	t.Run("end discards a run waiting on a decision", func(t *testing.T) {
		h := newHarness(planned())
		opts := h.options(nil)
		opts.Interrupter = always(InterruptEnd)

		outcome := interrupt(t, h, opts)

		assert.True(t, outcome.RunDiscarded)
		assert.Equal(t, 1, h.runs.Count(&h.runs.Discarded))
	})

	t.Run("end cancels a run whose decision was made as it was interrupted", func(t *testing.T) {
		h := newHarness(planned())
		h.runs.DiscardErr = status.Error(codes.AlreadyExists, "run is no longer planned")
		h.runs.OnDiscard = func(f *runtest.Runs) {
			f.Set(runtest.NewRun(pb.RunStatus_APPLY_QUEUED, runtest.WithPlan(pb.PlanStatus_FINISHED, "plan-job"), runtest.WithApply(pb.ApplyStatus_QUEUED, "")))
		}
		opts := h.options(nil)
		opts.Interrupter = always(InterruptEnd)

		outcome := interrupt(t, h, opts)

		assert.False(t, outcome.RunDiscarded)
		assert.True(t, outcome.RunCanceled)
	})

	t.Run("end reports nothing when the run ended as it was interrupted", func(t *testing.T) {
		h := newHarness(planned())
		h.runs.DiscardErr = status.Error(codes.AlreadyExists, "run is no longer planned")
		h.runs.OnDiscard = func(f *runtest.Runs) {
			f.Set(runtest.AppliedRun())
		}
		opts := h.options(nil)
		opts.Interrupter = always(InterruptEnd)

		outcome := interrupt(t, h, opts)

		assert.False(t, outcome.RunDiscarded)
		assert.False(t, outcome.RunCanceled)
		assert.Empty(t, find[InterruptFailed](h.rec))
	})

	t.Run("cancel leaves a run that moved to a decision while the interrupter decided", func(t *testing.T) {
		h := newHarness(queued())
		opts := h.options(nil)
		opts.Interrupter = interrupterFunc(func(context.Context, *pb.Run) InterruptAction {
			h.runs.Set(planned()) // the plan finished while the user was being asked
			return InterruptCancelRunIfInProgress
		})

		outcome := interrupt(t, h, opts)

		assert.False(t, outcome.RunCanceled)
		assert.Equal(t, 0, h.runs.Count(&h.runs.Canceled))
		assert.Equal(t, pb.RunStatus_PLANNED, outcome.Run.Status)
		failed := find[InterruptFailed](h.rec)
		require.Len(t, failed, 1)
		assert.ErrorIs(t, failed[0].Err, errRunNotCancelable)
	})

	t.Run("a failed cancel is reported", func(t *testing.T) {
		h := newHarness(queued())
		h.runs.CancelErr = status.Error(codes.Internal, "boom")
		opts := h.options(nil)
		opts.Interrupter = always(InterruptCancelRunIfInProgress)

		outcome := interrupt(t, h, opts)

		assert.False(t, outcome.RunCanceled)
		failed := find[InterruptFailed](h.rec)
		require.Len(t, failed, 1)
		assert.Equal(t, "cancel", failed[0].Action)
	})

	t.Run("a canceled cleanup context leaves the run going", func(t *testing.T) {
		h := newHarness(queued())
		cleanup, cancel := context.WithCancel(context.Background())
		opts := h.options(nil)
		opts.CleanupContext = cleanup
		opts.Interrupter = interrupterFunc(func(context.Context, *pb.Run) InterruptAction {
			cancel() // a second interrupt while deciding
			return InterruptCancelRunIfInProgress
		})

		outcome := interrupt(t, h, opts)

		assert.False(t, outcome.RunCanceled)
	})

	t.Run("a run that is over isn't handed to the interrupter", func(t *testing.T) {
		h := newHarness(runtest.AppliedRun())
		h.runs.HoldUntil = func() bool { return false }
		called := false
		opts := h.options(nil)
		opts.Interrupter = interrupterFunc(func(context.Context, *pb.Run) InterruptAction {
			called = true
			return InterruptCancelRunIfInProgress
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		outcome, err := Follow(ctx, runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), opts)

		require.NoError(t, err)
		assert.Equal(t, OutcomeInterrupted, outcome.Kind)
		assert.False(t, called)
	})
}

func TestFollow_Retries(t *testing.T) {
	t.Run("transient read errors are retried", func(t *testing.T) {
		h := newHarness(runtest.AppliedRun())
		h.runs.GetErrs = []error{status.Error(codes.Unavailable, "draining"), status.Error(codes.Aborted, "optimistic lock")}

		outcome := h.follow(t, h.options(nil))

		assert.Equal(t, OutcomeCompleted, outcome.Kind)
	})

	t.Run("other read errors end following", func(t *testing.T) {
		h := newHarness(runtest.AppliedRun())
		h.runs.GetErrs = []error{status.Error(codes.PermissionDenied, "no")}

		_, err := Follow(context.Background(), runtest.Client(h.runs, h.jobs), runtest.NewRun(pb.RunStatus_PENDING), h.options(nil))

		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})
}

func TestBuildPipeline(t *testing.T) {
	names := func(stages []stage) []string {
		out := []string{}
		for _, s := range stages {
			out = append(out, s.ref().Name)
		}
		return out
	}
	f := &follower{opts: &Options{}}

	t.Run("a speculative run has no apply phase", func(t *testing.T) {
		run := runtest.NewRun(pb.RunStatus_PENDING, runtest.Speculative(),
			runtest.WithStage(runtest.PostPlan, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED))
		assert.Equal(t, []string{"Plan", "Post-plan stage"}, names(buildPipeline(f, run)))
	})

	t.Run("only the stages the run has are added, in run order", func(t *testing.T) {
		run := runtest.NewRun(pb.RunStatus_PENDING,
			runtest.WithStage(runtest.PostApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED),
			runtest.WithStage(runtest.PrePlan, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED),
			runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED))
		assert.Equal(t, []string{
			"Pre-plan stage", "Plan", "Plan approval", "Pre-apply stage", "Apply", "Post-apply stage",
		}, names(buildPipeline(f, run)))
	})
}

func TestApplyPhaseStarted(t *testing.T) {
	tests := []struct {
		name string
		run  *pb.Run
		want bool
	}{
		{name: "planned", run: runtest.NewRun(pb.RunStatus_PLANNED), want: false},
		{name: "apply pending", run: runtest.NewRun(pb.RunStatus_APPLY_QUEUING, runtest.WithApply(pb.ApplyStatus_PENDING, "")), want: true},
		{name: "apply skipped", run: runtest.NewRun(pb.RunStatus_PLANNED_AND_FINISHED, runtest.WithApply(pb.ApplyStatus_SKIPPED, "")), want: false},
		{
			name: "pre-apply stage readied",
			run:  runtest.NewRun(pb.RunStatus_PRE_APPLY_QUEUING, runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_PENDING)),
			want: true,
		},
		{
			name: "pre-apply stage canceled by a discard",
			run:  runtest.NewRun(pb.RunStatus_DISCARDED, runtest.WithStage(runtest.PreApply, pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CANCELED)),
			want: true,
		},
		{name: "speculative", run: runtest.NewRun(pb.RunStatus_PLANNED_AND_FINISHED, runtest.Speculative()), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, applyPhaseStarted(tt.run))
		})
	}
}

func TestAdvisoryPolicies(t *testing.T) {
	f := &follower{}
	f.recordAdvisories("c1", []string{"group/a", "group/b"})
	f.recordAdvisories("c2", []string{"group/b", "group/c"})
	assert.Equal(t, []string{"group/a", "group/b", "group/c"}, f.advisoryPolicies(), "each policy is listed once, in order")

	// A re-evaluated check replaces what it reported before.
	f.recordAdvisories("c1", nil)
	assert.Equal(t, []string{"group/b", "group/c"}, f.advisoryPolicies())
}

func TestRunStatusHelpers(t *testing.T) {
	assert.True(t, IsRunFinal(pb.RunStatus_APPLIED))
	assert.True(t, IsRunFinal(pb.RunStatus_PLANNED_AND_FINISHED))
	assert.True(t, IsRunFinal(pb.RunStatus_DISCARDED))
	assert.False(t, IsRunFinal(pb.RunStatus_PLANNED))
	assert.False(t, IsRunCancelable(pb.RunStatus_PLANNED), "a run waiting on a decision is discarded instead")
	assert.True(t, IsRunCancelable(pb.RunStatus_APPLYING))
	assert.False(t, IsRunCancelable(pb.RunStatus_APPLIED))
}

func TestFollower_RefreshRun(t *testing.T) {
	withVersion := func(v int64) *pb.Run {
		r := runtest.NewRun(pb.RunStatus_PLANNED)
		r.Metadata.Version = v
		return r
	}
	newFollower := func(queue ...*pb.Run) (*follower, *runtest.Runs) {
		runs := runtest.NewRuns(queue...)
		return &follower{
			client: runtest.Client(runs, &runtest.Jobs{}),
			opts:   &Options{},
			logger: hclog.NewNullLogger(),
		}, runs
	}
	refresh := func(t *testing.T, f *follower, since *pb.Run, forceRead bool) int64 {
		t.Helper()
		got, err := f.refreshRun(context.Background(), since, forceRead)
		require.NoError(t, err)
		return got.Metadata.Version
	}

	t.Run("uses an event's run that is newer without reading the run", func(t *testing.T) {
		f, runs := newFollower(withVersion(1))
		f.newestRunCache.offer(withVersion(2))

		assert.Equal(t, int64(2), refresh(t, f, withVersion(1), false))
		assert.Zero(t, runs.Count(&runs.Gets))
	})

	t.Run("reads the run when the event's run is not newer", func(t *testing.T) {
		f, runs := newFollower(withVersion(3))
		f.newestRunCache.offer(withVersion(2))

		assert.Equal(t, int64(3), refresh(t, f, withVersion(2), false))
		assert.Equal(t, 1, runs.Count(&runs.Gets))
	})

	t.Run("reads the run when no event delivered one", func(t *testing.T) {
		f, runs := newFollower(withVersion(2))
		f.newestRunCache.offer(nil)
		f.newestRunCache.offer(&pb.Run{})

		assert.Equal(t, int64(2), refresh(t, f, withVersion(1), false))
		assert.Equal(t, 1, runs.Count(&runs.Gets))
	})

	t.Run("reads the run when a read is forced", func(t *testing.T) {
		f, runs := newFollower(withVersion(4))
		f.newestRunCache.offer(withVersion(3))

		assert.Equal(t, int64(4), refresh(t, f, withVersion(1), true))
		assert.Equal(t, 1, runs.Count(&runs.Gets))
	})

	t.Run("returns an event's run newer than the read", func(t *testing.T) {
		f, _ := newFollower(withVersion(2))
		f.newestRunCache.offer(withVersion(5))

		assert.Equal(t, int64(5), refresh(t, f, withVersion(1), true))
	})

	t.Run("keeps the newest of several events", func(t *testing.T) {
		f, runs := newFollower(withVersion(1))
		f.newestRunCache.offer(withVersion(3))
		f.newestRunCache.offer(withVersion(2))

		assert.Equal(t, int64(3), refresh(t, f, withVersion(1), false))
		assert.Zero(t, runs.Count(&runs.Gets))
	})

	t.Run("callers don't take an event's run from each other", func(t *testing.T) {
		f, runs := newFollower(withVersion(1))
		f.newestRunCache.offer(withVersion(3))

		assert.Equal(t, int64(3), refresh(t, f, withVersion(1), false), "the decision goroutine sees it")
		assert.Equal(t, int64(3), refresh(t, f, withVersion(2), false), "and so does the driver")
		assert.Zero(t, runs.Count(&runs.Gets))
	})

	t.Run("forgets an event's run when the subscription is re-established", func(t *testing.T) {
		f, runs := newFollower(withVersion(5))
		f.newestRunCache.offer(withVersion(2))
		f.newestRunCache.clear()

		assert.Equal(t, int64(5), refresh(t, f, withVersion(1), false))
		assert.Equal(t, 1, runs.Count(&runs.Gets))
	})
}
