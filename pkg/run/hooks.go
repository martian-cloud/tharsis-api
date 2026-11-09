// Package run follows a Tharsis run through each of its stages: pre-plan, plan, post-plan, plan
// approval, pre-apply, apply and post-apply.
//
// The package holds the generic part of following a run: it reads the run, works out where each
// stage stands, and makes every API call (applying, discarding, approving or overriding a gate,
// canceling). How the run is shown, how decisions are made and what happens on an interrupt are
// left to hooks that the caller supplies:
//
//   - an Observer is told everything worth showing, as typed events and job log chunks;
//   - a Decider is asked when a stage is blocked on a decision;
//   - an Interrupter decides what happens to the run when following is interrupted.
//
// Hooks describe and decide; they never call the API themselves, so conflict handling, retries and
// the rule that the run is re-read after every action live in one place.
package run

import (
	"context"
	"time"

	"github.com/hashicorp/go-hclog"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// DefaultWaitTimeout is how long a stage may stay blocked on a decision when Options.WaitTimeout
// isn't set.
const DefaultWaitTimeout = time.Hour

// Observer is told everything worth showing while a run is followed. Its methods are called from
// the goroutine that called Follow, one at a time, so an implementation needs no locking.
type Observer interface {
	// Event reports something that happened. Implementations should ignore event types they don't
	// know, so new events can be added without breaking them.
	Event(e Event)
	// Logs receives a chunk of a job's logs as it streams. An error stops following.
	Logs(stage StageRef, chunk string) error
}

// Decider is asked when a stage is blocked on a decision. The ctx passed to each method is canceled
// as soon as the decision is no longer needed (the run moved on, or following was interrupted), so
// an implementation that waits for an answer only has to watch ctx. When ctx ends before an answer,
// an implementation should return ctx.Err() (or an error wrapping it); any other error is treated as
// the decision failing. A decision chosen by whoever the decider asked (a user, an agent, a service
// account) should set Chosen, so that leaving the run waiting by choice isn't mistaken for the
// decider leaving it because it couldn't decide.
type Decider interface {
	// DecidePlan is asked when the run's plan is waiting for approval. ctx also expires when
	// Options.WaitTimeout passes.
	DecidePlan(ctx context.Context, req *PlanRequest) (PlanDecision, error)
	// DecideGates is asked when a policy stage is blocked on gates this caller may still act on, or
	// on work that can't be shown (an unsupported check), in which case the request has no pending
	// gates. It is asked again, with an updated request, if the decision didn't clear any gate (for
	// example, the caller wasn't allowed to approve it).
	DecideGates(ctx context.Context, req *GateRequest) (GateDecision, error)
}

// Interrupter decides what happens to the run when the context passed to Follow is canceled.
type Interrupter interface {
	// OnInterrupt is called with the latest run, unless the run is already over. ctx is
	// Options.CleanupContext.
	OnInterrupt(ctx context.Context, run *pb.Run) InterruptAction
}

// InterruptAction is what to do with the run when following is interrupted.
type InterruptAction int

const (
	// InterruptLeave leaves the run going.
	InterruptLeave InterruptAction = iota
	// InterruptCancelRunIfInProgress cancels the run if it is in progress, and otherwise leaves it:
	// one waiting on a decision or already over is not canceled.
	InterruptCancelRunIfInProgress
	// InterruptEnd ends the run: a run waiting on a decision is discarded, and a run doing work is
	// canceled. Use it when nobody will be left to look after the run, as when a CI job is canceled.
	InterruptEnd
)

// Options controls how a run is followed.
type Options struct {
	// Observer is told what happens. Required.
	Observer Observer
	// Decider makes decisions. When nil, every decision is left to be made elsewhere (the Tharsis UI)
	// and waited on.
	Decider Decider
	// Interrupter decides what happens to the run when following is interrupted. When nil, the run
	// is left going.
	Interrupter Interrupter
	// CleanupContext is the context the interrupt handling runs under, since the context passed to
	// Follow is already done by then. When nil, it is a minute-long context detached from Follow's.
	CleanupContext context.Context
	// WaitTimeout is how long a stage may stay blocked on a decision (plan approval or a policy
	// gate), whether the decider is being asked or the decision is being made elsewhere, before
	// following stops and leaves the run waiting (OutcomeDetached with TimedOut set). Zero or less
	// uses DefaultWaitTimeout; there is no way to wait forever.
	WaitTimeout time.Duration
	// SkipFinished, when true, doesn't show the stages that had already finished when following
	// began, such as the plan when following a run from its apply.
	SkipFinished bool
	// PollInterval is how often the run is re-read when no run event arrives, as a fallback in case
	// the event stream stalls without erroring. Defaults to 30 seconds.
	PollInterval time.Duration
	// RetryDelay is the first delay before a read that failed with a transient error is retried;
	// each retry doubles it, up to 15 seconds. Defaults to a second.
	RetryDelay time.Duration
	// Logger receives debug logs. Defaults to a null logger.
	Logger hclog.Logger
}

// StageKind is the kind of a stage.
type StageKind int

const (
	// StagePlan is the Terraform plan.
	StagePlan StageKind = iota
	// StagePlanApproval is the plan approval step between the plan and the apply phase.
	StagePlanApproval
	// StageApply is the Terraform apply.
	StageApply
	// StageTask is a task stage, holding policy checks: pre-plan, post-plan, pre-apply or post-apply.
	StageTask
)

// StageRef identifies a stage of the run.
type StageRef struct {
	// Name is a display name for the stage, such as "Plan" or "Pre-apply stage".
	Name string
	Kind StageKind
	// TaskStage names the task stage, when Kind is StageTask.
	TaskStage pb.RunTaskStageName
}

// OutcomeKind is how following a run ended.
type OutcomeKind int

const (
	// OutcomeCompleted means every stage finished: the run applied, finished with no changes, or (for
	// a speculative run) finished planning.
	OutcomeCompleted OutcomeKind = iota
	// OutcomeDetached means the run is waiting on a decision and following stopped: the decider
	// chose to leave it waiting, or (with TimedOut set) Options.WaitTimeout passed.
	OutcomeDetached
	// OutcomeDiscarded means the run was discarded on the decider's decision.
	OutcomeDiscarded
	// OutcomeFailed means the run errored, was canceled, or was discarded elsewhere.
	OutcomeFailed
	// OutcomeInterrupted means the context passed to Follow was canceled. The run keeps going unless
	// RunCanceled or RunDiscarded is set.
	OutcomeInterrupted
)

// Outcome describes how following a run ended.
type Outcome struct {
	Kind OutcomeKind
	// Run is the last snapshot of the run.
	Run *pb.Run
	// Stage is the stage following stopped at: the one the run is waiting on when detached, or the
	// one that failed. It is the zero StageRef when the run completed or following was interrupted.
	Stage StageRef
	// RunCanceled is true when following was interrupted and the run was canceled.
	RunCanceled bool
	// RunDiscarded is true when following was interrupted and the run was discarded.
	RunDiscarded bool
	// TimedOut is true when following stopped because Options.WaitTimeout passed.
	TimedOut bool
	// LeaveChosen is true when following stopped on a PlanLeave or GateLeave decision that was
	// chosen (Chosen set). It is false when the decider left the run waiting on its own, such as when
	// no one could be asked (see LeaveReason), and when following timed out.
	LeaveChosen bool
	// LeaveReason is the decider's Reason for leaving the run waiting on its own, when it gave one.
	LeaveReason string
	// Advisories names each policy that failed at the advisory level in the checks shown, in the
	// order they were shown, each once. A failed advisory policy doesn't block the run.
	Advisories []string
}

// PlanRequest asks for a decision on a plan waiting for approval.
type PlanRequest struct {
	Run *pb.Run
	// IsDestroy is true for a destroy run.
	IsDestroy bool
}

// PlanAction is what to do with a plan waiting for approval.
type PlanAction int

const (
	// PlanApply approves the plan and applies the run.
	PlanApply PlanAction = iota
	// PlanDiscard declines the plan and discards the run.
	PlanDiscard
	// PlanLeave stops following and leaves the run waiting for approval.
	PlanLeave
	// PlanWaitElsewhere keeps following, waiting for the plan to be decided elsewhere.
	PlanWaitElsewhere
)

// PlanDecision is a decision on a plan waiting for approval.
type PlanDecision struct {
	Action PlanAction
	// Chosen is true when whoever the decider asked chose this decision, and false when the decider
	// made it on its own, such as when no one could be asked. For PlanLeave it is passed on in
	// Outcome.LeaveChosen, so a caller can tell a choice to leave the run waiting from the decider
	// giving up.
	Chosen bool
	// Reason says why the decision isn't being made here: for PlanWaitElsewhere it is passed on in
	// the DecisionDeferred event, and for PlanLeave in Outcome.LeaveReason.
	Reason string
}

// GateInfo is a pending gate, with the results of the policy check it blocks.
type GateInfo struct {
	Gate    *pb.RunGate
	Results *pb.PolicyCheckResults
	// NeedsOverride is true when the gate has no approval rules, so only an override clears it.
	NeedsOverride bool
}

// GateRequest asks for a decision on a policy stage blocked on gates.
type GateRequest struct {
	Run   *pb.Run
	Stage StageRef
	// Pending is every pending gate in the stage. It is empty when the stage is blocked only on work
	// that can't be shown, such as a check of an unsupported type (reported in UnsupportedPolicyCheck);
	// then only GateDiscard, GateLeave and GateWaitElsewhere apply.
	Pending []GateInfo
	// CanApprove and CanOverride say whether GateApprove and GateOverride would act on any gate:
	// there is a pending gate of that kind that this caller hasn't already approved and hasn't
	// been denied on.
	CanApprove, CanOverride bool
}

// GateAction is what to do with a policy stage blocked on gates.
type GateAction int

const (
	// GateApprove approves each pending gate this caller may still approve.
	GateApprove GateAction = iota
	// GateOverride overrides each pending gate that can only be overridden.
	GateOverride
	// GateDiscard discards the run.
	GateDiscard
	// GateLeave stops following and leaves the run waiting.
	GateLeave
	// GateWaitElsewhere keeps following, waiting for the gates to be decided elsewhere.
	GateWaitElsewhere
)

// GateDecision is a decision on a policy stage blocked on gates.
type GateDecision struct {
	Action GateAction
	// Chosen is true when whoever the decider asked chose this decision, and false when the decider
	// made it on its own, such as when no one could be asked. For GateLeave it is passed on in
	// Outcome.LeaveChosen, so a caller can tell a choice to leave the run waiting from the decider
	// giving up.
	Chosen bool
	// Reason says why the decision isn't being made here: for GateWaitElsewhere it is passed on in
	// the DecisionDeferred event, and for GateLeave in Outcome.LeaveReason.
	Reason string
}
