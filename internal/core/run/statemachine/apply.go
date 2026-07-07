package statemachine

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// ApplyNode represents the apply node in a run.
type ApplyNode struct {
	nodeBase
	path          string
	status        models.ApplyStatus
	statusChanges []ApplyStatusChange
	autoApply     bool
}

// NewApplyNode creates a new apply node. The autoApply flag indicates the apply
// is pre-approved and is therefore ready to be queued as soon as the plan
// finishes with changes; manual runs leave the apply in created until approved.
func NewApplyNode(path string, status models.ApplyStatus, autoApply bool) *ApplyNode {
	return &ApplyNode{
		path:          path,
		status:        status,
		statusChanges: []ApplyStatusChange{},
		autoApply:     autoApply,
	}
}

// Status returns the current apply status.
func (n *ApplyNode) Status() models.ApplyStatus { return n.status }

// AutoApply reports whether the apply is pre-approved.
func (n *ApplyNode) AutoApply() bool { return n.autoApply }

// SetStatus transitions the apply to a new status and fires its listeners. It is
// used for externally driven changes (e.g. job-status updates); the run node uses
// setStatusSilently when it drives the apply itself.
func (n *ApplyNode) SetStatus(status models.ApplyStatus) error {
	if err := n.transition(status); err != nil {
		return err
	}
	return n.nodeBase.fireEvent(string(status))
}

// setStatusSilently transitions the apply WITHOUT firing its listeners. The run
// node uses this when it drives the apply itself (e.g. cancellation cleanup), so
// the change does not re-project back onto the run. The status change is still
// recorded.
func (n *ApplyNode) setStatusSilently(status models.ApplyStatus) error {
	return n.transition(status)
}

// transition validates and applies a status change, recording it. Only transitions
// that are part of the apply lifecycle are allowed; anything else (including setting
// the status the node already holds) is rejected, since the apply should never be
// asked to move to a state it cannot legitimately reach from its current one.
func (n *ApplyNode) transition(status models.ApplyStatus) error {
	if !canTransitionTo(applyTransitions, n.status, status) {
		return fmt.Errorf("invalid apply status transition from %q to %q", n.status, status)
	}
	n.statusChanges = append(n.statusChanges, ApplyStatusChange{
		OldStatus: n.status,
		NewStatus: status,
		Path:      n.path,
	})
	n.status = status
	return nil
}

// GetStatusChanges returns all status changes recorded on this node.
func (n *ApplyNode) GetStatusChanges() []NodeStatusChange {
	changes := make([]NodeStatusChange, len(n.statusChanges))
	for i, c := range n.statusChanges {
		changes[i] = c
	}
	return changes
}
