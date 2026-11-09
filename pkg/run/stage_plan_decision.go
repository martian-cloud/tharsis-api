package run

import (
	"context"
	"fmt"
	"time"

	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// planDecisionStage is the step between a finished plan and the apply phase, where the plan is
// approved or discarded. On the server it is the planned run status rather than a node of its own.
type planDecisionStage struct {
	f                 *follower
	stage             StageRef
	reportedNoChanges bool
	reportedAutoApply bool
	// deferred is set once the decision was left to be made elsewhere, until the run leaves planned.
	deferred bool
	// applyRequested is set once the run was applied, until the run is seen out of planned. A
	// snapshot that still shows planned is then out of date, and the plan isn't asked about again.
	applyRequested bool
}

func newPlanDecisionStage(f *follower) *planDecisionStage {
	return &planDecisionStage{f: f, stage: StageRef{Name: "Plan approval", Kind: StagePlanApproval}}
}

func (s *planDecisionStage) ref() StageRef {
	return s.stage
}

func (s *planDecisionStage) sync(_ context.Context, run *pb.Run) (stageStatus, error) {
	if run.Status != pb.RunStatus_PLANNED {
		// The run moved on; should it come back to planned (an undiscard), it's asked about again.
		s.applyRequested = false
		s.deferred = false
	}

	if applyPhaseStarted(run) {
		if run.AutoApply && !s.reportedAutoApply && !s.f.shouldSkipFinished() {
			s.f.emit(AutoApproving{})
		}
		s.reportedAutoApply = true
		return stageDone, nil
	}

	switch run.Status {
	case pb.RunStatus_PLANNED:
		if run.AutoApply {
			// The server starts the apply of an auto-apply run itself.
			s.f.waiting(s.stage, WaitApplyStart, "")
			return stageWaiting, nil
		}
		if s.applyRequested {
			// The run was already applied. If it stays planned, though (the apply conflicted for another
			// reason), the wait timeout bounds the wait.
			s.f.waiting(s.stage, WaitApplyStart, "")
			return stageWaitingOnDecision, nil
		}
		if s.f.decider == nil && !s.deferred {
			s.deferred = true
			s.f.emit(DecisionDeferred{Stage: s.stage})
		}
		if s.deferred {
			s.f.waiting(s.stage, WaitDecisionElsewhere, "")
			return stageWaitingOnDecision, nil
		}
		return stageNeedsDecision, nil
	case pb.RunStatus_PLANNED_AND_FINISHED:
		if !s.reportedNoChanges {
			s.reportedNoChanges = true
			s.f.emit(NoChanges{})
		}
		return stageDone, nil
	case pb.RunStatus_APPLIED:
		// Applied without its apply phase showing as started (the apply isn't in the snapshot); there
		// is nothing left to decide.
		return stageDone, nil
	case pb.RunStatus_CANCELED, pb.RunStatus_ERRORED, pb.RunStatus_DISCARDED:
		return stageFailed, nil
	default:
		return stageWaiting, nil
	}
}

func (s *planDecisionStage) decide(ctx context.Context, run *pb.Run) (decisionResult, error) {
	req := &PlanRequest{Run: run, IsDestroy: run.IsDestroy}
	decision, how, err := decide(ctx, s.f, run,
		func(latest *pb.Run) bool { return latest.Status == pb.RunStatus_PLANNED },
		func(ctx context.Context) (PlanDecision, error) { return s.f.decider.DecidePlan(ctx, req) },
	)
	if err != nil {
		return decisionActed, err
	}
	switch how {
	case abandoned:
		s.f.emit(DecisionAbandoned{Stage: s.stage})
		return decisionActed, nil
	case timedOut:
		return decisionTimedOut, nil
	}

	switch decision.Action {
	case PlanApply:
		return s.apply(ctx, run)
	case PlanDiscard:
		return s.discard(ctx, run)
	case PlanLeave:
		s.f.leaveChosen, s.f.leaveReason = decision.Chosen, decision.Reason
		return decisionDetach, nil
	case PlanWaitElsewhere:
		s.deferred = true
		s.f.emit(DecisionDeferred{Stage: s.stage, Reason: decision.Reason})
		return decisionActed, nil
	default:
		return decisionActed, fmt.Errorf("unknown plan decision %d", decision.Action)
	}
}

func (s *planDecisionStage) apply(ctx context.Context, run *pb.Run) (decisionResult, error) {
	if _, err := s.f.client.RunsClient.ApplyRun(ctx, &pb.ApplyRunRequest{RunId: run.Metadata.Id}); err != nil {
		if !isStateConflict(err) {
			return decisionActed, fmt.Errorf("failed to apply run: %w", err)
		}
		// Already applied (e.g. a retried request, or approved elsewhere), or changed by someone else
		// at the same moment; re-read the run.
	}
	s.applyRequested = true
	// The wait for the apply to start is timed afresh, not from when the decision was first asked.
	s.f.blockedSince = time.Now()
	return decisionActed, nil
}

func (s *planDecisionStage) discard(ctx context.Context, run *pb.Run) (decisionResult, error) {
	if _, err := s.f.client.RunsClient.DiscardRun(ctx, &pb.DiscardRunRequest{RunId: run.Metadata.Id}); err != nil {
		if isStateConflict(err) {
			s.f.emit(ActionConflict{Stage: s.stage, Action: "discard"})
			return decisionActed, nil
		}
		return decisionActed, fmt.Errorf("failed to discard run: %w", err)
	}
	return decisionDiscarded, nil
}

// applyPhaseStarted reports whether the run has gone past plan approval: its pre-apply stage or its
// apply has been readied. This reads the nodes rather than the run status so a run that later
// errors in its apply phase still counts as approved.
func applyPhaseStarted(run *pb.Run) bool {
	if stage := findTaskStage(run, pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_APPLY); stage != nil {
		switch stage.Status {
		case pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_UNSPECIFIED,
			pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_CREATED,
			pb.RunTaskStageStatus_RUN_TASK_STAGE_STATUS_SKIPPED:
		default:
			return true
		}
	}
	if run.Apply != nil {
		switch run.Apply.Status {
		case pb.ApplyStatus_UNSPECIFIED, pb.ApplyStatus_CREATED, pb.ApplyStatus_SKIPPED:
		default:
			return true
		}
	}
	return false
}
