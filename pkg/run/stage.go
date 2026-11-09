package run

import (
	"context"

	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// stageStatus is where a stage stands against the latest run snapshot.
type stageStatus int

const (
	// stageWaiting means the stage has not started, or is in progress.
	stageWaiting stageStatus = iota
	// stageDone means the stage cleared or was skipped; the driver moves to the next stage.
	stageDone
	// stageNeedsDecision means the stage is blocked on a decision; the driver calls decide.
	stageNeedsDecision
	// stageFailed means the stage's node errored or was canceled, so the run won't continue.
	stageFailed
	// stageStale means the stage did blocking work (it streamed a job's logs), so the snapshot it
	// was given is out of date and the driver should re-read the run straight away.
	stageStale
	// stageWaitingOnDecision means the stage is blocked on a decision being made elsewhere (in the
	// Tharsis UI, or by other approvers). The driver waits as for stageWaiting, and gives up after
	// Options.WaitTimeout.
	stageWaitingOnDecision
)

// stage is one stage of a run: a Terraform plan or apply, a policy task stage, or the plan approval
// step. The server owns the run's state; a stage only reads the snapshots it is given, reports what
// is new to the observer, and says where it stands.
type stage interface {
	ref() StageRef
	// sync brings this stage up to date with the run snapshot. It reports anything new (and may
	// block while streaming a job's logs) and says where the stage stands. The snapshot may show the
	// stage further ahead than last time, or back where it was after a retry or an undiscard.
	sync(ctx context.Context, run *pb.Run) (stageStatus, error)
}

// decisionResult is what came of a decision on a blocked stage.
type decisionResult int

const (
	// decisionActed means an action was taken (or the run moved on); re-read the run and continue.
	decisionActed decisionResult = iota
	// decisionDetach means the decider chose to stop following and leave the run waiting.
	decisionDetach
	// decisionDiscarded means the run was discarded.
	decisionDiscarded
	// decisionTimedOut means Options.WaitTimeout passed before the decider decided.
	decisionTimedOut
)

// decidingStage is implemented by the stages that can block on a decision.
type decidingStage interface {
	decide(ctx context.Context, run *pb.Run) (decisionResult, error)
}

// buildPipeline builds the ordered stages for a run from its shape. The shape is fixed when the run
// is created: a policy task stage only exists when policies apply to it, and a speculative run has
// no apply phase.
func buildPipeline(f *follower, run *pb.Run) []stage {
	stages := []stage{}
	addTaskStage := func(name pb.RunTaskStageName) {
		if findTaskStage(run, name) != nil {
			stages = append(stages, newTaskStage(f, name))
		}
	}

	addTaskStage(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN)
	stages = append(stages, newPlanStage(f))
	addTaskStage(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_PLAN)

	if run.Speculative {
		return stages
	}

	stages = append(stages, newPlanDecisionStage(f))
	addTaskStage(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_APPLY)
	stages = append(stages, newApplyStage(f))
	addTaskStage(pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY)

	return stages
}

// findTaskStage returns the run's task stage with the given name, or nil if the run has none.
func findTaskStage(run *pb.Run, name pb.RunTaskStageName) *pb.RunTaskStage {
	for _, stage := range run.TaskStages {
		if stage.StageName == name {
			return stage
		}
	}
	return nil
}

// TaskStageTitle is the display name of a task stage.
func TaskStageTitle(name pb.RunTaskStageName) string {
	switch name {
	case pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_PLAN:
		return "Pre-plan stage"
	case pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_PLAN:
		return "Post-plan stage"
	case pb.RunTaskStageName_RUN_TASK_STAGE_NAME_PRE_APPLY:
		return "Pre-apply stage"
	case pb.RunTaskStageName_RUN_TASK_STAGE_NAME_POST_APPLY:
		return "Post-apply stage"
	default:
		return "Task stage"
	}
}
