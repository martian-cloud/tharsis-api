package admission

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestCanQueuePlan(t *testing.T) {
	// A run with no apply node is speculative; one with an apply node is not.
	speculativeRun := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}
	nonSpeculativeRun := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, Apply: &models.Apply{}}

	tests := []struct {
		name string
		ws   *models.Workspace
		run  *models.Run
		want bool
	}{
		{"speculative plan admitted even when the workspace is locked", &models.Workspace{Locked: true}, speculativeRun, true},
		{"speculative plan admitted while an apply is in progress", &models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, speculativeRun, true},
		{"non-speculative plan blocked when the workspace is locked", &models.Workspace{Locked: true}, nonSpeculativeRun, false},
		{"non-speculative plan blocked when another run holds the workspace", &models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, nonSpeculativeRun, false},
		{"non-speculative plan admitted when the workspace is free", &models.Workspace{}, nonSpeculativeRun, true},
	}

	a := &Admitter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.canQueuePlan(tt.ws, tt.run))
		})
	}
}

// finishedPlanRun builds a non-speculative run whose plan finished with changes
// (the precondition for queuing an apply), owned for workspace-contention checks.
func finishedPlanRun() *models.Run {
	return &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Plan:     models.Plan{Status: models.PlanFinished, HasChanges: true},
		Apply:    &models.Apply{},
	}
}

func TestCanQueueApply(t *testing.T) {
	noChanges := finishedPlanRun()
	noChanges.Plan.HasChanges = false

	planNotFinished := finishedPlanRun()
	planNotFinished.Plan.Status = models.PlanRunning

	tests := []struct {
		name string
		ws   *models.Workspace
		run  *models.Run
		want bool
	}{
		{"plan not finished is not admitted", &models.Workspace{}, planNotFinished, false},
		{"plan finished without changes is not admitted", &models.Workspace{}, noChanges, false},
		{"apply blocked when the workspace is locked", &models.Workspace{Locked: true}, finishedPlanRun(), false},
		{"apply blocked when another run holds the workspace", &models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, finishedPlanRun(), false},
		{"apply admitted when the workspace is free", &models.Workspace{}, finishedPlanRun(), true},
		{"apply admitted when this run already holds the workspace", &models.Workspace{CurrentApplyRunID: ptr.String("run-1")}, finishedPlanRun(), true},
	}

	a := &Admitter{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, a.canQueueApply(tt.ws, tt.run))
		})
	}
}

func TestWorkspaceAvailable(t *testing.T) {
	assert.True(t, WorkspaceAvailable(&models.Workspace{}), "unlocked and unoccupied is available")
	assert.False(t, WorkspaceAvailable(&models.Workspace{Locked: true}), "locked is not available")
	assert.False(t, WorkspaceAvailable(&models.Workspace{CurrentApplyRunID: ptr.String("run-1")}), "occupied is not available")
}

func TestWorkspaceAvailableForRun(t *testing.T) {
	run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}

	assert.True(t, workspaceAvailableForRun(&models.Workspace{}, run), "empty CurrentApplyRunID is available")
	assert.True(t, workspaceAvailableForRun(&models.Workspace{CurrentApplyRunID: ptr.String("run-1")}, run), "held by this run is available for it")
	assert.False(t, workspaceAvailableForRun(&models.Workspace{CurrentApplyRunID: ptr.String("other-run")}, run), "held by another run is not available")
	assert.False(t, workspaceAvailableForRun(&models.Workspace{Locked: true, CurrentApplyRunID: ptr.String("run-1")}, run), "locked is not available even for the holding run")
}
