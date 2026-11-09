package run

import (
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// Event is something that happened while a run was followed. It is one of the event types in this
// file.
type Event interface {
	isEvent()
}

// StageStarted reports that a stage started, or started again (Retry): a Terraform node was
// retried and has a new job, or a task stage re-evaluated after a retry or an undiscard.
type StageStarted struct {
	Stage StageRef
	Retry bool
}

// WaitReason is what the run is waiting on.
type WaitReason int

const (
	// WaitRunStarting means the run hasn't started yet.
	WaitRunStarting WaitReason = iota + 1
	// WaitWorkspace means the run is waiting for the workspace to be free.
	WaitWorkspace
	// WaitRunner means a job is waiting for a runner.
	WaitRunner
	// WaitTasksRunning means a task stage's tasks are running.
	WaitTasksRunning
	// WaitApplyStart means the plan was approved and the apply is about to start.
	WaitApplyStart
	// WaitOtherApprovers means this caller approved every pending gate, which still need other
	// approvers. Detail summarizes the approvals still needed.
	WaitOtherApprovers
	// WaitDecisionElsewhere means a decision is being made elsewhere (the Tharsis UI). Detail
	// summarizes the approvals still needed, for a gate.
	WaitDecisionElsewhere
	// WaitUnsupportedTask means a stage is blocked on a task this package can't show or act on.
	WaitUnsupportedTask
)

// Waiting reports what the run is waiting on. It is sent when the reason changes, or when anything
// else has been reported since, not for every read of the run.
type Waiting struct {
	Stage  StageRef
	Reason WaitReason
	Detail string
}

// PolicyCheckFinished reports a policy check's verdict and findings. First is true for the first
// check reported in an evaluation of its stage.
type PolicyCheckFinished struct {
	Stage   StageRef
	Results *pb.PolicyCheckResults
	First   bool
}

// UnsupportedPolicyCheck reports a policy check of a type this package doesn't know how to show; it
// is waited on without being shown. First is as for PolicyCheckFinished.
type UnsupportedPolicyCheck struct {
	Stage     StageRef
	CheckType pb.PolicyCheckType
	First     bool
}

// NoVisibleTasks reports a stage that has started but holds no tasks this package can show.
type NoVisibleTasks struct {
	Stage StageRef
}

// GatesPending reports the gates a blocked policy stage is waiting on, before a decision on them
// is asked for.
type GatesPending struct {
	Stage StageRef
	Gates []GateInfo
}

// DecisionDeferred reports that the decider left a decision to be made elsewhere. It is sent once
// for each time the stage blocks.
type DecisionDeferred struct {
	Stage  StageRef
	Reason string
}

// DecisionAbandoned reports that the run moved on while a decision was being made, so the decision
// was no longer needed.
type DecisionAbandoned struct {
	Stage StageRef
}

// GateApproved reports an approval recorded on a gate. AllApprovalsIn is true when the gate needs
// no more approvals.
type GateApproved struct {
	Stage          StageRef
	Gate           *pb.RunGate
	AllApprovalsIn bool
}

// GateOverridden reports a gate that was overridden.
type GateOverridden struct {
	Stage StageRef
	Gate  *pb.RunGate
}

// GateDenied reports that the caller isn't allowed to take an action on a gate. The gate isn't
// offered for that action again.
type GateDenied struct {
	Stage  StageRef
	Gate   *pb.RunGate
	Action GateAction
}

// ActionConflict reports that an action wasn't taken because the run had already moved on, such
// as discarding a run whose plan was approved elsewhere in the meantime.
type ActionConflict struct {
	Stage StageRef
	// Action is "apply" or "discard".
	Action string
}

// NoChanges reports that the plan had no changes, so the run finished after planning.
type NoChanges struct{}

// AutoApproving reports that an auto-apply run went into its apply phase without asking.
type AutoApproving struct{}

// Interrupted reports that following was interrupted. The interrupt handling follows.
type Interrupted struct{}

// InterruptFailed reports that the action chosen on an interrupt failed. Action is "cancel" or
// "discard".
type InterruptFailed struct {
	Action string
	Err    error
}

func (StageStarted) isEvent()           {}
func (Waiting) isEvent()                {}
func (PolicyCheckFinished) isEvent()    {}
func (UnsupportedPolicyCheck) isEvent() {}
func (NoVisibleTasks) isEvent()         {}
func (GatesPending) isEvent()           {}
func (DecisionDeferred) isEvent()       {}
func (DecisionAbandoned) isEvent()      {}
func (GateApproved) isEvent()           {}
func (GateOverridden) isEvent()         {}
func (GateDenied) isEvent()             {}
func (ActionConflict) isEvent()         {}
func (NoChanges) isEvent()              {}
func (AutoApproving) isEvent()          {}
func (Interrupted) isEvent()            {}
func (InterruptFailed) isEvent()        {}
