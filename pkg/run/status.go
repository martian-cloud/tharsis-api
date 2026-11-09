package run

import (
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsRunFailed reports whether the run ended without completing.
func IsRunFailed(s pb.RunStatus) bool {
	switch s {
	case pb.RunStatus_CANCELED, pb.RunStatus_ERRORED, pb.RunStatus_DISCARDED:
		return true
	default:
		return false
	}
}

// IsRunFinal reports whether the run is over: it completed, or ended without completing.
func IsRunFinal(s pb.RunStatus) bool {
	switch s {
	case pb.RunStatus_APPLIED, pb.RunStatus_PLANNED_AND_FINISHED:
		return true
	default:
		return IsRunFailed(s)
	}
}

// IsRunCancelable reports whether the run is doing work that canceling would stop. A run waiting on
// a decision (plan approval or a policy gate) could be canceled too, but declining it is done by
// discarding it, and a run that is over can't be canceled.
func IsRunCancelable(s pb.RunStatus) bool {
	return !IsRunAwaitingDecision(s) && !IsRunFinal(s)
}

// IsRunAwaitingDecision reports whether the run is waiting on a decision: plan approval or a policy
// gate. Such a run is applied or discarded rather than canceled.
func IsRunAwaitingDecision(s pb.RunStatus) bool {
	switch s {
	case pb.RunStatus_PLANNED,
		pb.RunStatus_PRE_PLAN_AWAITING_DECISION,
		pb.RunStatus_POST_PLAN_AWAITING_DECISION,
		pb.RunStatus_PRE_APPLY_AWAITING_DECISION:
		return true
	default:
		return false
	}
}

// waitReason is what a run in the given status is waiting on, or zero for a transient status between
// stages that isn't worth reporting.
func waitReason(s pb.RunStatus) WaitReason {
	switch s {
	case pb.RunStatus_PENDING:
		return WaitRunStarting
	case pb.RunStatus_PRE_PLAN_QUEUING, pb.RunStatus_PLAN_QUEUING,
		pb.RunStatus_PRE_APPLY_QUEUING, pb.RunStatus_APPLY_QUEUING:
		return WaitWorkspace
	case pb.RunStatus_PLAN_QUEUED, pb.RunStatus_APPLY_QUEUED:
		return WaitRunner
	case pb.RunStatus_PRE_PLAN_RUNNING, pb.RunStatus_POST_PLAN_RUNNING,
		pb.RunStatus_PRE_APPLY_RUNNING, pb.RunStatus_POST_APPLY_RUNNING:
		return WaitTasksRunning
	default:
		return 0
	}
}

// isStateConflict reports whether an action failed because the run or gate had already moved on,
// or was changed by someone else at the same moment (Aborted, the server's optimistic lock), in
// which case the right response is to re-read the run.
func isStateConflict(err error) bool {
	switch status.Code(err) {
	case codes.AlreadyExists, codes.FailedPrecondition, codes.Aborted:
		return true
	default:
		return false
	}
}

// isPermissionDenied reports whether an action failed because the caller isn't allowed to take it.
func isPermissionDenied(err error) bool {
	return status.Code(err) == codes.PermissionDenied
}

// isTransient reports whether a request failed in a way that retrying it may fix.
func isTransient(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}
