package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// SetRunStatus transitions the run node's status via the state machine.
func SetRunStatus(r *models.Run, status models.RunStatus) ([]NodeStatusChange, error) {
	sm := fromRun(r)
	if err := sm.GetRunNode().SetStatus(status); err != nil {
		return nil, err
	}
	syncToRun(sm, r)
	return sm.GetStatusChanges(), nil
}

// SetPlanStatus transitions the run's plan node status via the state machine. Every
// run has a plan node (fromRun always builds one), so no nil check is needed.
func SetPlanStatus(r *models.Run, status models.PlanStatus) ([]NodeStatusChange, error) {
	sm := fromRun(r)
	if err := sm.GetRunNode().Plan().SetStatus(status); err != nil {
		return nil, err
	}
	syncToRun(sm, r)
	return sm.GetStatusChanges(), nil
}

// SetApplyStatus transitions the run's apply node status via the state machine.
func SetApplyStatus(r *models.Run, status models.ApplyStatus) ([]NodeStatusChange, error) {
	sm := fromRun(r)
	applyNode := sm.GetRunNode().Apply()
	if applyNode == nil {
		return nil, fmt.Errorf("run does not have an apply node")
	}
	if err := applyNode.SetStatus(status); err != nil {
		return nil, err
	}
	syncToRun(sm, r)
	return sm.GetStatusChanges(), nil
}

// AdvanceRun moves the run on from whatever it was waiting for. Callers say only "the thing this run was
// blocked on has been satisfied" — the run node works out what that means from its own state, so no
// caller has to name a target status or know which node comes next.
//
// There are three blockers a caller can satisfy:
//   - the workspace slot: the admitter calls this once it has acquired the slot, and the run node starts
//     whichever gated node was pending (a task stage, the plan, or the apply);
//   - the run's own creation: the create commands call this on a freshly-created run, and the run node
//     begins the plan phase — readying a pre-plan policy stage if the run has one, rather than the plan;
//   - human approval: StartApply calls this for a manual run parked at planned, and the run node begins
//     the apply phase — readying a pre-apply policy stage if the run has one, rather than the apply.
//
// All three collapse to "advance the run", so this is the only op an external actor needs to unblock one,
// and adding a workspace-gated node needs no new op and no change at any call site. It is a no-op when
// the run is not waiting on anything, so a redundant call is harmless.
func AdvanceRun(r *models.Run) ([]NodeStatusChange, error) {
	sm := fromRun(r)
	if err := sm.GetRunNode().advance(); err != nil {
		return nil, err
	}
	syncToRun(sm, r)
	return sm.GetStatusChanges(), nil
}

// SetPolicyCheckStatus transitions the run's policy check node identified by
// checkPath via the state machine. It errors if the run has no matching check node.
//
// Retrying a check needs no op of its own: setting a failed or canceled check to pending is the whole
// retry. The check's stage listens for pending and restarts itself from there (see
// TaskStageNode.handleCheckPending), re-queueing the check — and so creating a fresh policy-eval job —
// only once it holds the workspace slot.
func SetPolicyCheckStatus(r *models.Run, checkPath string, status models.PolicyCheckStatus) ([]NodeStatusChange, error) {
	sm := fromRun(r)
	var target *PolicyCheckNode
	for _, check := range sm.GetRunNode().AllPolicyChecks() {
		if check.Path() == checkPath {
			target = check
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("run does not have a %q policy check node", checkPath)
	}
	if err := target.SetStatus(status); err != nil {
		return nil, err
	}
	syncToRun(sm, r)
	return sm.GetStatusChanges(), nil
}

// UndiscardRun reverses a discard, reviving the run at the point the discard abandoned it. A run
// discarded at a policy gate had that gate's stage canceled with it, so undiscard re-runs the gate: its
// canceled checks return to pending, which re-evaluates them (and re-creates the run gate) before the
// run can go any further, so the gate is never silently released. A run discarded from planned has no
// canceled gate and simply returns to planned — an unsatisfied gate can never take that branch, since
// the discard would have canceled it. Either way the nodes discard skipped are restored, so the revived
// run can flow through them again.
func UndiscardRun(r *models.Run) ([]NodeStatusChange, error) {
	sm := fromRun(r)
	runNode := sm.GetRunNode()

	// Restore the nodes discard skipped — the never-started plan/apply and any stage that had not run.
	// Done before the transition below so the run's children are already restored by the time it
	// projects a new run status; it is a no-op for anything already in a non-skipped state.
	if err := runNode.resetSkippedChildren(); err != nil {
		return nil, err
	}

	// A canceled gate drives the run status itself: restarting its checks restarts the stage, which
	// projects that stage's *_queuing (gated) or post_plan_running (ungated) status onto the run.
	// Otherwise the run was discarded from planned — the only other status a run can be discarded from —
	// so that is where it returns to, which also restores its skipped apply (the unskipApply listener).
	if gate := canceledRunTaskStage(runNode); gate != nil {
		if err := gate.restartCanceled(); err != nil {
			return nil, err
		}
	} else if err := runNode.SetStatus(models.RunPlanned); err != nil {
		return nil, err
	}

	syncToRun(sm, r)
	return sm.GetStatusChanges(), nil
}

// canceledRunTaskStage returns the stage a discard abandoned mid-decision, or nil if the run was
// discarded from planned. Discarding cancels the stage the run was blocked at (handleRunTerminated),
// and only that one can be canceled on a discardable run: a run parks at planned or a gate's
// awaiting-decision status to be discarded, and it cannot have reached either past a stage an earlier
// cancellation canceled — that stage's checks are the only retryable node at that point, so the stage
// completes before the run moves on. The stages are checked in run order, so the result is
// deterministic even if that ever stops holding.
func canceledRunTaskStage(runNode *RunNode) *TaskStageNode {
	for _, stageName := range []models.RunTaskStageName{
		models.RunTaskStageNamePrePlan,
		models.RunTaskStageNamePostPlan,
		models.RunTaskStageNamePreApply,
	} {
		if stage := runNode.TaskStage(stageName); stage != nil && stage.Status() == models.RunTaskStageCanceled {
			return stage
		}
	}
	return nil
}

// fromRun constructs a state machine from the run's current state.
func fromRun(r *models.Run) *StateMachine {
	runNode := NewRunNode(r.Status)
	runNode.SetPlanNode(NewPlanNode(r.Plan.GetPath(), r.Plan.Status, r.Plan.HasChanges))
	if r.Apply != nil {
		runNode.SetApplyNode(NewApplyNode(r.Apply.GetPath(), r.Apply.Status, r.AutoApply))
	}
	for _, stage := range r.TaskStages {
		stageNode := NewTaskStageNode(stage.ID, stage.StageName, stage.Status)
		for _, check := range stage.PolicyChecks {
			stageNode.AddPolicyCheckNode(NewPolicyCheckNode(check.ID, check.GetPath(), check.CheckType, check.Status))
		}
		runNode.AddTaskStageNode(stageNode)
	}
	return New(runNode)
}

// syncToRun writes state machine statuses back to the run model fields.
func syncToRun(sm *StateMachine, r *models.Run) {
	runNode := sm.GetRunNode()

	r.Status = runNode.Status()

	if planNode := runNode.Plan(); planNode != nil {
		r.Plan.Status = planNode.Status()
	}

	if r.Apply != nil {
		if applyNode := runNode.Apply(); applyNode != nil {
			r.Apply.Status = applyNode.Status()
		}
	}

	// Sync each task stage's status back to the model, and each check's status by path.
	if len(r.TaskStages) > 0 {
		allChecks := runNode.AllPolicyChecks()
		byPath := make(map[string]*PolicyCheckNode, len(allChecks))
		for _, check := range allChecks {
			byPath[check.Path()] = check
		}
		for _, stage := range r.TaskStages {
			if stageNode := runNode.TaskStage(stage.StageName); stageNode != nil {
				stage.Status = stageNode.Status()
			}
			for _, check := range stage.PolicyChecks {
				if node, ok := byPath[check.GetPath()]; ok {
					check.Status = node.Status()
				}
			}
		}
	}
}
