package statemachine

// StateMachine encapsulates run state updates.
type StateMachine struct {
	run *RunNode
}

// New creates a new StateMachine instance.
func New(run *RunNode) *StateMachine {
	run.init()
	return &StateMachine{run: run}
}

// GetRunNode returns the root run node.
func (s *StateMachine) GetRunNode() *RunNode {
	return s.run
}

// GetStatusChanges returns all status changes from all nodes in tree order.
func (s *StateMachine) GetStatusChanges() []NodeStatusChange {
	changes := s.run.GetStatusChanges()
	if s.run.plan != nil {
		changes = append(changes, s.run.plan.GetStatusChanges()...)
	}
	if s.run.apply != nil {
		changes = append(changes, s.run.apply.GetStatusChanges()...)
	}
	return changes
}
