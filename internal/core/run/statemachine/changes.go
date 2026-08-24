package statemachine

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// NodeType identifies which node a status change belongs to.
type NodeType string

// NodeType constants
const (
	RunNodeType         NodeType = "run"
	PlanNodeType        NodeType = "plan"
	ApplyNodeType       NodeType = "apply"
	PolicyCheckNodeType NodeType = "policy_check"
	TaskStageNodeType   NodeType = "task_stage"
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

// PolicyCheckStatusChange represents a status change on a policy check node.
type PolicyCheckStatusChange struct {
	OldStatus models.PolicyCheckStatus
	NewStatus models.PolicyCheckStatus
	CheckID   string
	Path      string
}

// GetNodeType returns the node type.
func (PolicyCheckStatusChange) GetNodeType() NodeType { return PolicyCheckNodeType }

// TaskStageStatusChange represents a status change on a task stage node.
type TaskStageStatusChange struct {
	OldStatus models.RunTaskStageStatus
	NewStatus models.RunTaskStageStatus
	StageID   string
	Path      string
}

// GetNodeType returns the node type.
func (TaskStageStatusChange) GetNodeType() NodeType { return TaskStageNodeType }
