package transformers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestAutoApplyTransformer_Transform(t *testing.T) {
	// applyPendingChange is the node status change the transformer reacts to.
	applyPendingChange := statemachine.ApplyStatusChange{NewStatus: models.ApplyPending}

	tests := []struct {
		name        string
		run         *models.Run
		changes     []statemachine.NodeStatusChange
		wantTrigger string
	}{
		{
			name: "auto-apply non-speculative run with apply pending transition gets system trigger",
			run: &models.Run{
				AutoApply: true,
				Apply:     &models.Apply{},
			},
			changes:     []statemachine.NodeStatusChange{applyPendingChange},
			wantTrigger: "system",
		},
		{
			name: "run without auto-apply is left untouched",
			run: &models.Run{
				AutoApply: false,
				Apply:     &models.Apply{},
			},
			changes:     []statemachine.NodeStatusChange{applyPendingChange},
			wantTrigger: "",
		},
		{
			name: "speculative run (no apply node) is skipped",
			run: &models.Run{
				AutoApply: true,
				Apply:     nil,
			},
			changes:     []statemachine.NodeStatusChange{applyPendingChange},
			wantTrigger: "", // nothing to set; apply is nil
		},
		{
			name: "apply transition to a non-pending status does not set the trigger",
			run: &models.Run{
				AutoApply: true,
				Apply:     &models.Apply{},
			},
			changes:     []statemachine.NodeStatusChange{statemachine.ApplyStatusChange{NewStatus: models.ApplyQueued}},
			wantTrigger: "",
		},
		{
			name: "a plan pending change is ignored (wrong node type)",
			run: &models.Run{
				AutoApply: true,
				Apply:     &models.Apply{},
			},
			changes:     []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanPending}},
			wantTrigger: "",
		},
		{
			name: "no node status changes leaves the trigger untouched",
			run: &models.Run{
				AutoApply: true,
				Apply:     &models.Apply{},
			},
			changes:     nil,
			wantTrigger: "",
		},
	}

	transformer := NewAutoApplyTransformer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := types.RunChange{Run: tt.run, NodeStatusChanges: tt.changes}

			// The transformer is pure with respect to the store for this path, so nil is fine.
			err := transformer.Transform(context.Background(), []types.RunChange{change}, nil)
			assert.NoError(t, err)

			if tt.run.Apply != nil {
				assert.Equal(t, tt.wantTrigger, tt.run.Apply.TriggeredBy)
			}
		})
	}
}

func TestAutoApplyTransformer_Transform_PreservesExistingTrigger(t *testing.T) {
	// When the run does not qualify, an already-set TriggeredBy must be preserved.
	run := &models.Run{
		AutoApply: false,
		Apply:     &models.Apply{TriggeredBy: "user-123"},
	}
	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.ApplyStatusChange{NewStatus: models.ApplyPending}},
	}

	transformer := NewAutoApplyTransformer()
	require := assert.New(t)
	require.NoError(transformer.Transform(context.Background(), []types.RunChange{change}, nil))
	require.Equal("user-123", run.Apply.TriggeredBy)
}
