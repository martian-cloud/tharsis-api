package transformers

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestAdmissionTransformer_Transform_QueuesPendingPlan(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		// No apply node => speculative => always admitted, and never acquires the workspace.
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanPending}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	// The admitter transitioned the plan to queued and recorded the change on the store.
	assert.Equal(t, models.PlanQueued, run.Plan.Status)

	storeChanges := runStore.GetChanges()
	assert.Len(t, storeChanges, 1)
}

func TestAdmissionTransformer_Transform_DoesNotQueueWhenWorkspaceLocked(t *testing.T) {
	ctx := context.Background()

	// Non-speculative run (has an apply node) so the workspace lock is enforced.
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, Locked: true}, nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanPending}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	// Plan stays pending, nothing recorded on the store.
	assert.Equal(t, models.PlanPending, run.Plan.Status)
	assert.Empty(t, runStore.GetChanges())
}

func TestAdmissionTransformer_Transform_IgnoresNonPendingTransition(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanQueued},
	}

	// No GetWorkspaceByID expectation: the admitter must not be called for a
	// non-pending transition (e.g. a plan moving to running).
	mockWorkspaces := db.NewMockWorkspaces(t)
	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanRunning}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)
	assert.Empty(t, runStore.GetChanges())
}

func TestAdmissionTransformer_Transform_QueuesPendingApply(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunQueuingApply,
		// Apply admission requires a finished plan that produced changes.
		Plan:  models.Plan{Status: models.PlanFinished, HasChanges: true},
		Apply: &models.Apply{Status: models.ApplyPending},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	// The apply is non-speculative, so the workspace is acquired.
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.MatchedBy(func(ws *models.Workspace) bool {
		return ws.CurrentApplyRunID != nil && *ws.CurrentApplyRunID == "run-1"
	})).Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}, CurrentApplyRunID: ptr.String("run-1")}, nil)

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.ApplyStatusChange{NewStatus: models.ApplyPending}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	assert.Equal(t, models.ApplyQueued, run.Apply.Status)
	assert.Len(t, runStore.GetChanges(), 1)
}

func TestAdmissionTransformer_Transform_SwallowsOptimisticLock(t *testing.T) {
	ctx := context.Background()

	// Non-speculative plan whose workspace acquisition fails with an OLE. The
	// transformer must swallow it, leaving the node pending and the store clean.
	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)
	mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
		Return(nil, errors.New("conflict", errors.WithErrorCode(errors.EOptimisticLock)))

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanPending}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.NoError(t, err)

	assert.Equal(t, models.PlanPending, run.Plan.Status)
	assert.Empty(t, runStore.GetChanges())
}

func TestAdmissionTransformer_Transform_PropagatesNonOLEError(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{},
	}

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").
		Return(nil, errors.New("boom"))

	dbClient := &db.Client{Workspaces: mockWorkspaces}

	runStore := store.NewRunStore(dbClient)
	runStore.AddRun(run)

	transformer := NewAdmissionTransformer(admission.New(dbClient))

	change := types.RunChange{
		Run:               run,
		NodeStatusChanges: []statemachine.NodeStatusChange{statemachine.PlanStatusChange{NewStatus: models.PlanPending}},
	}

	err := transformer.Transform(ctx, []types.RunChange{change}, runStore)
	assert.Error(t, err)
}
