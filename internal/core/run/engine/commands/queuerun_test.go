package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestQueueRun_Execute_RunNotFound(t *testing.T) {
	ctx := context.Background()

	mockRuns := db.NewMockRuns(t)
	mockRuns.On("GetRunByID", ctx, "missing").Return(nil, nil)

	runStore := store.NewRunStore(&db.Client{Runs: mockRuns})
	cmd := &QueueRun{admitter: admission.New(&db.Client{}), RunID: "missing"}

	err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
	require.Error(t, err)
}

func TestQueueRun_Execute_NoPendingNodes(t *testing.T) {
	ctx := context.Background()

	// Plan already finished and no apply: nothing to re-queue, so the admitter (and
	// thus the workspace lookup) is never touched.
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlannedAndFinished,
		Plan:     models.Plan{Status: models.PlanFinished},
		Apply:    nil,
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &QueueRun{admitter: admission.New(&db.Client{Workspaces: db.NewMockWorkspaces(t)}), RunID: "run-1"}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Empty(t, runStore.GetChanges())
}

func TestQueueRun_Execute_PlanPending_WorkspaceBusy(t *testing.T) {
	ctx := context.Background()

	mockWorkspaces := db.NewMockWorkspaces(t)
	// Locked workspace -> a non-speculative plan cannot be queued.
	mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{Locked: true}, nil)

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{Status: models.ApplyCreated}, // non-speculative, not yet pending
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &QueueRun{admitter: admission.New(&db.Client{Workspaces: mockWorkspaces}), RunID: "run-1"}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	// Still pending; no changes recorded.
	assert.Equal(t, models.PlanPending, run.Plan.Status)
	assert.Empty(t, runStore.GetChanges())
}

func TestQueueRun_Execute_PlanPending_WorkspaceFree_Queues(t *testing.T) {
	ctx := context.Background()

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{}, nil)
	// Acquiring the workspace persists the CurrentApplyRunID update.
	mockWorkspaces.On("UpdateWorkspace", ctx, mock.Anything).Return(&models.Workspace{}, nil)

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunQueuing,
		Plan:        models.Plan{Status: models.PlanPending},
		Apply:       &models.Apply{Status: models.ApplyCreated},
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &QueueRun{admitter: admission.New(&db.Client{Workspaces: mockWorkspaces}), RunID: "run-1"}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	assert.Equal(t, models.PlanQueued, run.Plan.Status)
	assert.NotEmpty(t, runStore.GetChanges())
}

func TestQueueRun_Execute_ApplyPending_WorkspaceFree_Queues(t *testing.T) {
	ctx := context.Background()

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{}, nil)
	mockWorkspaces.On("UpdateWorkspace", ctx, mock.Anything).Return(&models.Workspace{}, nil)

	run := &models.Run{
		Metadata:    models.ResourceMetadata{ID: "run-1"},
		WorkspaceID: "ws-1",
		Status:      models.RunQueuingApply,
		Plan:        models.Plan{Status: models.PlanFinished, HasChanges: true},
		Apply:       &models.Apply{Status: models.ApplyPending},
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &QueueRun{admitter: admission.New(&db.Client{Workspaces: mockWorkspaces}), RunID: "run-1"}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	assert.Equal(t, models.ApplyQueued, run.Apply.Status)
	assert.NotEmpty(t, runStore.GetChanges())
}
