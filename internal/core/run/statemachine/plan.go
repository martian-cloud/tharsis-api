package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// PlanNode represents the plan node in a run.
type PlanNode struct {
	nodeBase
	path          string
	status        models.PlanStatus
	hasChanges    bool
	statusChanges []PlanStatusChange
}

// NewPlanNode creates a new plan node. hasChanges indicates whether the plan
// produced changes, which determines whether the run has anything to apply.
func NewPlanNode(path string, status models.PlanStatus, hasChanges bool) *PlanNode {
	return &PlanNode{
		path:          path,
		status:        status,
		hasChanges:    hasChanges,
		statusChanges: []PlanStatusChange{},
	}
}

// Status returns the current plan status.
func (n *PlanNode) Status() models.PlanStatus { return n.status }

// HasChanges reports whether the plan produced changes.
func (n *PlanNode) HasChanges() bool { return n.hasChanges }

// SetStatus transitions the plan to a new status and fires its listeners, which
// project the change onto the run (e.g. a job-status update advancing the run).
func (n *PlanNode) SetStatus(status models.PlanStatus) error {
	if err := n.transition(status); err != nil {
		return err
	}
	return n.nodeBase.fireEvent(string(status))
}

// setStatusSilently transitions the plan WITHOUT firing its listeners. The run
// node uses this when it drives the plan itself (e.g. readying the plan when the
// run is queued), where firing the listeners would re-project onto the run. The
// change is still recorded so it is persisted.
func (n *PlanNode) setStatusSilently(status models.PlanStatus) error {
	return n.transition(status)
}

// transition validates and applies a status change, recording it. Only transitions
// that are part of the plan lifecycle are allowed; anything else (including setting
// the status the node already holds) is rejected, since the plan should never be
// asked to move to a state it cannot legitimately reach from its current one.
func (n *PlanNode) transition(status models.PlanStatus) error {
	if !canTransitionTo(planTransitions, n.status, status) {
		return fmt.Errorf("invalid plan status transition from %q to %q", n.status, status)
	}
	n.statusChanges = append(n.statusChanges, PlanStatusChange{
		OldStatus: n.status,
		NewStatus: status,
		Path:      n.path,
	})
	n.status = status
	return nil
}

// GetStatusChanges returns all status changes recorded on this node.
func (n *PlanNode) GetStatusChanges() []NodeStatusChange {
	changes := make([]NodeStatusChange, len(n.statusChanges))
	for i, c := range n.statusChanges {
		changes[i] = c
	}
	return changes
}
