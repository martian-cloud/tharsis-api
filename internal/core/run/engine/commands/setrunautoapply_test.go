package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestSetRunAutoApply_Prepare(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves the workspace namespace path", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{FullPath: "groupA/ws"}, nil)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &SetRunAutoApplyInput{RunID: "run-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, "groupA/ws", cmd.namespacePath)
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").Return(nil, nil)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{Runs: mockRuns},
			in:       &SetRunAutoApplyInput{RunID: "run-1"},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("missing workspace yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(nil, nil)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &SetRunAutoApplyInput{RunID: "run-1"},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestSetRunAutoApply_Execute(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), auth.NewServiceAccountCaller("sa-1", "sa/path", nil, nil, nil))

	t.Run("enables auto-apply on a planning run and records an enable activity event", func(t *testing.T) {
		mockActivityEvents := db.NewMockActivityEvents(t)
		mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
			var payload models.ActivityEventUpdateRunPayload
			if err := json.Unmarshal(in.Payload, &payload); err != nil {
				return false
			}
			return in.Action == models.ActionUpdate &&
				in.TargetType == models.TargetRun &&
				payload.Type == string(models.RunUpdateTypeEnableAutoApply)
		})).Return(&models.ActivityEvent{}, nil)

		run := &models.Run{
			Metadata:  models.ResourceMetadata{ID: "run-1"},
			Status:    models.RunPlanning,
			AutoApply: false,
			Plan:      models.Plan{ID: "plan-1", Status: models.PlanRunning},
			Apply:     &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &SetRunAutoApply{
			dbClient:      &db.Client{ActivityEvents: mockActivityEvents},
			in:            &SetRunAutoApplyInput{RunID: "run-1", AutoApply: true},
			namespacePath: "groupA/ws",
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		require.NotNil(t, cmd.Updated)
		assert.True(t, cmd.Updated.AutoApply)
	})

	t.Run("disabling auto-apply records a disable activity event", func(t *testing.T) {
		mockActivityEvents := db.NewMockActivityEvents(t)
		mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
			var payload models.ActivityEventUpdateRunPayload
			if err := json.Unmarshal(in.Payload, &payload); err != nil {
				return false
			}
			return payload.Type == string(models.RunUpdateTypeDisableAutoApply)
		})).Return(&models.ActivityEvent{}, nil)

		run := &models.Run{
			Metadata:  models.ResourceMetadata{ID: "run-1"},
			Status:    models.RunPlanning,
			AutoApply: true,
			Plan:      models.Plan{ID: "plan-1", Status: models.PlanRunning},
			Apply:     &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{ActivityEvents: mockActivityEvents},
			in:       &SetRunAutoApplyInput{RunID: "run-1", AutoApply: false},
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		require.NotNil(t, cmd.Updated)
		assert.False(t, cmd.Updated.AutoApply)
	})

	t.Run("no-op when the setting already matches and records no activity event", func(t *testing.T) {
		// NewMockActivityEvents(t) asserts no unexpected calls, so a CreateActivityEvent would fail.
		mockActivityEvents := db.NewMockActivityEvents(t)

		run := &models.Run{
			Metadata:  models.ResourceMetadata{ID: "run-1"},
			Status:    models.RunPlanning,
			AutoApply: true,
			Plan:      models.Plan{ID: "plan-1", Status: models.PlanRunning},
			Apply:     &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{ActivityEvents: mockActivityEvents},
			in:       &SetRunAutoApplyInput{RunID: "run-1", AutoApply: true},
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		require.NotNil(t, cmd.Updated)
		assert.True(t, cmd.Updated.AutoApply)
	})

	t.Run("rejects a speculative run with no apply node", func(t *testing.T) {
		mockActivityEvents := db.NewMockActivityEvents(t)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlanning,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanRunning},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{ActivityEvents: mockActivityEvents},
			in:       &SetRunAutoApplyInput{RunID: "run-1", AutoApply: true},
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
		assert.Nil(t, cmd.Updated)
	})

	t.Run("rejects when the apply has already started", func(t *testing.T) {
		mockActivityEvents := db.NewMockActivityEvents(t)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunApplying,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyRunning},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{ActivityEvents: mockActivityEvents},
			in:       &SetRunAutoApplyInput{RunID: "run-1", AutoApply: true},
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
		assert.Nil(t, cmd.Updated)
	})

	t.Run("rejects once the plan has completed (planned)", func(t *testing.T) {
		mockActivityEvents := db.NewMockActivityEvents(t)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlanned,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &SetRunAutoApply{
			dbClient: &db.Client{ActivityEvents: mockActivityEvents},
			in:       &SetRunAutoApplyInput{RunID: "run-1", AutoApply: true},
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
		assert.Nil(t, cmd.Updated)
	})
}
