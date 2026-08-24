package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// PolicyCheckNode represents a policy check node in a run (e.g. the OPA post-plan check). The stage a
// check belongs to is owned by its parent TaskStageNode, so the check node does not carry it.
type PolicyCheckNode struct {
	nodeBase
	id            string
	path          string
	checkType     models.PolicyKind
	status        models.PolicyCheckStatus
	statusChanges []PolicyCheckStatusChange
}

// NewPolicyCheckNode creates a new policy check node.
func NewPolicyCheckNode(id, path string, checkType models.PolicyKind, status models.PolicyCheckStatus) *PolicyCheckNode {
	return &PolicyCheckNode{
		id:            id,
		path:          path,
		checkType:     checkType,
		status:        status,
		statusChanges: []PolicyCheckStatusChange{},
	}
}

// Status returns the current check status.
func (n *PolicyCheckNode) Status() models.PolicyCheckStatus { return n.status }

// Path returns the run-relative node path identifying this check.
func (n *PolicyCheckNode) Path() string { return n.path }

// SetStatus transitions the check to a new status and fires its listeners, which
// project the verdict onto the run (e.g. clearing policy or erroring the run).
// Setting the status the node already holds is a no-op.
func (n *PolicyCheckNode) SetStatus(status models.PolicyCheckStatus) error {
	if n.status == status {
		return nil
	}
	if err := n.transition(status); err != nil {
		return err
	}
	return n.nodeBase.fireEvent(string(status))
}

// transition validates and applies a status change, recording it. Only transitions that
// are part of the check lifecycle are allowed.
func (n *PolicyCheckNode) transition(status models.PolicyCheckStatus) error {
	if !canTransitionTo(policyCheckTransitions, n.status, status) {
		return fmt.Errorf("invalid policy check status transition from %q to %q", n.status, status)
	}
	n.statusChanges = append(n.statusChanges, PolicyCheckStatusChange{
		OldStatus: n.status,
		NewStatus: status,
		CheckID:   n.id,
		Path:      n.path,
	})
	n.status = status
	return nil
}

// GetStatusChanges returns all status changes recorded on this node.
func (n *PolicyCheckNode) GetStatusChanges() []NodeStatusChange {
	changes := make([]NodeStatusChange, len(n.statusChanges))
	for i, c := range n.statusChanges {
		changes[i] = c
	}
	return changes
}
