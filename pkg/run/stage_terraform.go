package run

import (
	"context"

	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// nodeState is a Terraform plan or apply node's status, reduced to what its stage acts on.
type nodeState int

const (
	nodeNotStarted nodeState = iota
	// nodePending is readied: the node has started but is waiting for the workspace.
	nodePending
	nodeQueued
	nodeRunning
	nodeFinished
	nodeFailed
	nodeSkipped
)

// terraformStage follows a run's Terraform plan or apply: it streams each of the node's jobs once
// and reports the node's outcome.
type terraformStage struct {
	f     *follower
	stage StageRef
	// node reads the stage's node from a run snapshot.
	node      func(run *pb.Run) (nodeState, *string)
	streamed  map[string]bool
	announced bool
}

func newPlanStage(f *follower) *terraformStage {
	return &terraformStage{
		f:     f,
		stage: StageRef{Name: "Plan", Kind: StagePlan},
		node: func(run *pb.Run) (nodeState, *string) {
			if run.Plan == nil {
				return nodeNotStarted, nil
			}
			return planState(run.Plan.Status), run.Plan.LatestJobId
		},
		streamed: map[string]bool{},
	}
}

func newApplyStage(f *follower) *terraformStage {
	return &terraformStage{
		f:     f,
		stage: StageRef{Name: "Apply", Kind: StageApply},
		node: func(run *pb.Run) (nodeState, *string) {
			if run.Apply == nil {
				return nodeSkipped, nil
			}
			return applyState(run.Apply.Status), run.Apply.LatestJobId
		},
		streamed: map[string]bool{},
	}
}

func (s *terraformStage) ref() StageRef {
	return s.stage
}

func (s *terraformStage) sync(ctx context.Context, run *pb.Run) (stageStatus, error) {
	state, jobID := s.node(run)

	switch state {
	case nodeSkipped:
		return stageDone, nil
	case nodeNotStarted:
		s.f.waiting(s.stage, waitReason(run.Status), "")
		return stageWaiting, nil
	}

	if state == nodeFinished && s.f.shouldSkipFinished() {
		// Finished before following began: don't show it, but remember its job so a retry is shown.
		s.announced = true
		if jobID != nil {
			s.streamed[*jobID] = true
		}
		return stageDone, nil
	}

	if !s.announced {
		s.announced = true
		s.f.emit(StageStarted{Stage: s.stage})
	}

	// Stream each job once; a new job means the node was retried.
	if jobID != nil && !s.streamed[*jobID] {
		if len(s.streamed) > 0 {
			s.f.emit(StageStarted{Stage: s.stage, Retry: true})
		}
		s.streamed[*jobID] = true
		if err := s.f.streamJobLogs(ctx, s.stage, *jobID); err != nil {
			return stageWaiting, err
		}
		return stageStale, nil
	}

	switch state {
	case nodeFinished:
		return stageDone, nil
	case nodeFailed:
		return stageFailed, nil
	default:
		s.f.waiting(s.stage, waitReason(run.Status), "")
		return stageWaiting, nil
	}
}

func planState(s pb.PlanStatus) nodeState {
	switch s {
	case pb.PlanStatus_PENDING:
		return nodePending
	case pb.PlanStatus_QUEUED:
		return nodeQueued
	case pb.PlanStatus_RUNNING:
		return nodeRunning
	case pb.PlanStatus_FINISHED:
		return nodeFinished
	case pb.PlanStatus_ERRORED, pb.PlanStatus_CANCELED:
		return nodeFailed
	default:
		// CREATED, and UNSPECIFIED. A skipped plan has no wire value yet (the server maps it to
		// UNSPECIFIED); it only happens on a run that ended, which the driver handles.
		return nodeNotStarted
	}
}

func applyState(s pb.ApplyStatus) nodeState {
	switch s {
	case pb.ApplyStatus_PENDING:
		return nodePending
	case pb.ApplyStatus_QUEUED:
		return nodeQueued
	case pb.ApplyStatus_RUNNING:
		return nodeRunning
	case pb.ApplyStatus_FINISHED:
		return nodeFinished
	case pb.ApplyStatus_ERRORED, pb.ApplyStatus_CANCELED:
		return nodeFailed
	case pb.ApplyStatus_SKIPPED:
		return nodeSkipped
	default:
		return nodeNotStarted
	}
}
