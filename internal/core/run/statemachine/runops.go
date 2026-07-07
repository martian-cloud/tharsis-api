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

// fromRun constructs a state machine from the run's current state.
func fromRun(r *models.Run) *StateMachine {
	runNode := NewRunNode(r.Status)
	runNode.SetPlanNode(NewPlanNode(r.Plan.GetPath(), r.Plan.Status, r.Plan.HasChanges))
	if r.Apply != nil {
		runNode.SetApplyNode(NewApplyNode(r.Apply.GetPath(), r.Apply.Status, r.AutoApply))
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
}
