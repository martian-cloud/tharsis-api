package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestPlanNode_Transitions verifies that every lifecycle transition records a
// single change carrying the correct old/new status, node type, and path.
func TestPlanNode_Transitions(t *testing.T) {
	tests := []struct {
		name string
		from models.PlanStatus
		to   models.PlanStatus
	}{
		{"created to pending", models.PlanCreated, models.PlanPending},
		{"pending to queued", models.PlanPending, models.PlanQueued},
		{"queued to running", models.PlanQueued, models.PlanRunning},
		{"running to finished", models.PlanRunning, models.PlanFinished},
		{"running to errored", models.PlanRunning, models.PlanErrored},
		{"running to canceled", models.PlanRunning, models.PlanCanceled},
		{"pending to canceled", models.PlanPending, models.PlanCanceled},
		{"queued to canceled", models.PlanQueued, models.PlanCanceled},
		{"created to canceled", models.PlanCreated, models.PlanCanceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlanNode("plan", tt.from, false)

			require.NoError(t, p.SetStatus(tt.to))
			assert.Equal(t, tt.to, p.Status())

			changes := p.GetStatusChanges()
			require.Len(t, changes, 1)
			c, ok := changes[0].(PlanStatusChange)
			require.True(t, ok)
			assert.Equal(t, tt.from, c.OldStatus)
			assert.Equal(t, tt.to, c.NewStatus)
			assert.Equal(t, "plan", c.Path)
			assert.Equal(t, PlanNodeType, c.GetNodeType())
		})
	}
}

// TestPlanNode_SameStatusRejected verifies that setting the status the node already
// holds is rejected: a node should never be asked to "transition" to its current
// state.
func TestPlanNode_SameStatusRejected(t *testing.T) {
	p := NewPlanNode("plan", models.PlanRunning, false)

	err := p.SetStatus(models.PlanRunning)

	require.Error(t, err)
	assert.Equal(t, models.PlanRunning, p.Status())
	assert.Empty(t, p.GetStatusChanges())
}

// TestPlanNode_InvalidTransition verifies that transitions outside the plan
// lifecycle are rejected with an error and leave the node unchanged.
func TestPlanNode_InvalidTransition(t *testing.T) {
	tests := []struct {
		name string
		from models.PlanStatus
		to   models.PlanStatus
	}{
		{"created to queued skips pending", models.PlanCreated, models.PlanQueued},
		{"created to running", models.PlanCreated, models.PlanRunning},
		{"created to finished", models.PlanCreated, models.PlanFinished},
		{"pending to running skips queued", models.PlanPending, models.PlanRunning},
		{"queued to finished skips running", models.PlanQueued, models.PlanFinished},
		{"running back to queued", models.PlanRunning, models.PlanQueued},
		{"queued back to pending", models.PlanQueued, models.PlanPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlanNode("plan", tt.from, false)

			err := p.SetStatus(tt.to)

			require.Error(t, err)
			assert.Equal(t, tt.from, p.Status())
			assert.Empty(t, p.GetStatusChanges())
		})
	}
}

// TestPlanNode_FinalStatusRejectsTransition verifies that a node in a final state
// rejects further transitions (final states have no outgoing transitions).
func TestPlanNode_FinalStatusRejectsTransition(t *testing.T) {
	for _, final := range []models.PlanStatus{models.PlanFinished, models.PlanErrored, models.PlanCanceled} {
		p := NewPlanNode("plan", final, false)

		err := p.SetStatus(models.PlanRunning)

		require.Error(t, err)
		assert.Equal(t, final, p.Status())
		assert.Empty(t, p.GetStatusChanges())
	}
}
