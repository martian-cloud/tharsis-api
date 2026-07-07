package statemachine

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// NodeType identifies which node a status change belongs to.
type NodeType string

// NodeType constants
const (
	RunNodeType   NodeType = "run"
	PlanNodeType  NodeType = "plan"
	ApplyNodeType NodeType = "apply"
)

// NodeStatusChange is a discriminated union of status changes across all node types.
type NodeStatusChange interface {
	GetNodeType() NodeType
}

// RunStatusChange represents a status change on the run node.
type RunStatusChange struct {
	OldStatus models.RunStatus
	NewStatus models.RunStatus
}

// GetNodeType returns the node type.
func (RunStatusChange) GetNodeType() NodeType { return RunNodeType }

// PlanStatusChange represents a status change on the plan node.
type PlanStatusChange struct {
	OldStatus models.PlanStatus
	NewStatus models.PlanStatus
	Path      string
}

// GetNodeType returns the node type.
func (PlanStatusChange) GetNodeType() NodeType { return PlanNodeType }

// ApplyStatusChange represents a status change on the apply node.
type ApplyStatusChange struct {
	OldStatus models.ApplyStatus
	NewStatus models.ApplyStatus
	Path      string
}

// GetNodeType returns the node type.
func (ApplyStatusChange) GetNodeType() NodeType { return ApplyNodeType }
