package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// TestApplyNode_Transitions verifies that every lifecycle transition records a
// single change carrying the correct old/new status, node type, and path.
func TestApplyNode_Transitions(t *testing.T) {
	tests := []struct {
		name string
		from models.ApplyStatus
		to   models.ApplyStatus
	}{
		{"created to pending", models.ApplyCreated, models.ApplyPending},
		{"pending to queued", models.ApplyPending, models.ApplyQueued},
		{"queued to running", models.ApplyQueued, models.ApplyRunning},
		{"running to finished", models.ApplyRunning, models.ApplyFinished},
		{"running to errored", models.ApplyRunning, models.ApplyErrored},
		{"running to canceled", models.ApplyRunning, models.ApplyCanceled},
		{"pending to canceled", models.ApplyPending, models.ApplyCanceled},
		{"queued to canceled", models.ApplyQueued, models.ApplyCanceled},
		{"created to canceled", models.ApplyCreated, models.ApplyCanceled},
		{"created to skipped", models.ApplyCreated, models.ApplySkipped},
		{"skipped back to created", models.ApplySkipped, models.ApplyCreated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApplyNode("apply", tt.from, false)

			require.NoError(t, a.SetStatus(tt.to))
			assert.Equal(t, tt.to, a.Status())

			changes := a.GetStatusChanges()
			require.Len(t, changes, 1)
			c, ok := changes[0].(ApplyStatusChange)
			require.True(t, ok)
			assert.Equal(t, tt.from, c.OldStatus)
			assert.Equal(t, tt.to, c.NewStatus)
			assert.Equal(t, "apply", c.Path)
			assert.Equal(t, ApplyNodeType, c.GetNodeType())
		})
	}
}

// TestApplyNode_SameStatusRejected verifies that setting the status the node already
// holds is rejected: a node should never be asked to "transition" to its current
// state.
func TestApplyNode_SameStatusRejected(t *testing.T) {
	a := NewApplyNode("apply", models.ApplyRunning, false)

	err := a.SetStatus(models.ApplyRunning)

	require.Error(t, err)
	assert.Equal(t, models.ApplyRunning, a.Status())
	assert.Empty(t, a.GetStatusChanges())
}

// TestApplyNode_InvalidTransition verifies that transitions outside the apply
// lifecycle are rejected with an error and leave the node unchanged.
func TestApplyNode_InvalidTransition(t *testing.T) {
	tests := []struct {
		name string
		from models.ApplyStatus
		to   models.ApplyStatus
	}{
		{"created to queued skips pending", models.ApplyCreated, models.ApplyQueued},
		{"created to running", models.ApplyCreated, models.ApplyRunning},
		{"created to finished", models.ApplyCreated, models.ApplyFinished},
		{"pending to running skips queued", models.ApplyPending, models.ApplyRunning},
		{"queued to finished skips running", models.ApplyQueued, models.ApplyFinished},
		{"running back to queued", models.ApplyRunning, models.ApplyQueued},
		{"queued back to pending", models.ApplyQueued, models.ApplyPending},
		{"skipped to pending", models.ApplySkipped, models.ApplyPending},
		{"pending to skipped", models.ApplyPending, models.ApplySkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApplyNode("apply", tt.from, false)

			err := a.SetStatus(tt.to)

			require.Error(t, err)
			assert.Equal(t, tt.from, a.Status())
			assert.Empty(t, a.GetStatusChanges())
		})
	}
}

// TestApplyNode_FinalStatusRejectsTransition verifies that a node in a final state
// rejects further transitions (final states have no outgoing transitions).
func TestApplyNode_FinalStatusRejectsTransition(t *testing.T) {
	for _, final := range []models.ApplyStatus{models.ApplyFinished, models.ApplyErrored, models.ApplyCanceled, models.ApplySkipped} {
		a := NewApplyNode("apply", final, false)

		err := a.SetStatus(models.ApplyRunning)

		require.Error(t, err)
		assert.Equal(t, final, a.Status())
		assert.Empty(t, a.GetStatusChanges())
	}
}
