// Package statemachine contains the state machine logic for a run
// in the form of a hierarchical graph: Run → Plan → Apply
package statemachine

type listenerFunc func() error

type nodeBase struct {
	listeners map[string]listenerFunc
}

func (n *nodeBase) registerListener(status string, l listenerFunc) {
	if n.listeners == nil {
		n.listeners = map[string]listenerFunc{}
	}
	n.listeners[status] = l
}

func (n *nodeBase) fireEvent(status string) error {
	if listener, ok := n.listeners[status]; ok {
		return listener()
	}
	return nil
}
