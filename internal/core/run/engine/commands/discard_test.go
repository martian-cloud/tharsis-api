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

func TestDiscardRun_Prepare(t *testing.T) {
	bg := context.Background()

	t.Run("resolves the workspace namespace path", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", bg, "ws-1").Return(&models.Workspace{FullPath: "groupA/ws"}, nil)

		cmd := &DiscardRun{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &DiscardRunInput{RunID: "run-1"},
		}

		require.NoError(t, cmd.Prepare(bg))
		assert.Equal(t, "groupA/ws", cmd.namespacePath)
	})

	t.Run("SkipActivityEvent short-circuits without any lookups", func(t *testing.T) {
		// With the activity event suppressed there is no namespace to resolve, so
		// Prepare is a no-op. NewMockRuns(t) fails on any unexpected call, so leaving
		// the db client empty (and never stubbing GetRunByID) verifies nothing is read.
		cmd := &DiscardRun{
			dbClient: &db.Client{Runs: db.NewMockRuns(t)},
			in:       &DiscardRunInput{RunID: "run-1", SkipActivityEvent: true},
		}

		require.NoError(t, cmd.Prepare(bg))
		assert.Empty(t, cmd.namespacePath)
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").Return(nil, nil)

		cmd := &DiscardRun{
			dbClient: &db.Client{Runs: mockRuns},
			in:       &DiscardRunInput{RunID: "run-1"},
		}

		err := cmd.Prepare(bg)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("missing workspace yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", bg, "ws-1").Return(nil, nil)

		cmd := &DiscardRun{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &DiscardRunInput{RunID: "run-1"},
		}

		err := cmd.Prepare(bg)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestDiscardRun_Execute(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), auth.NewServiceAccountCaller("sa-1", "sa/path", nil, nil, nil))

	t.Run("discards a planned run and records an activity event", func(t *testing.T) {
		mockActivityEvents := db.NewMockActivityEvents(t)
		// The discard is recorded as an UPDATE activity event whose payload identifies
		// the run-update sub-action (discard).
		mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
			var payload models.ActivityEventUpdateRunPayload
			if err := json.Unmarshal(in.Payload, &payload); err != nil {
				return false
			}
			return in.Action == models.ActionUpdate &&
				in.TargetType == models.TargetRun &&
				payload.Type == string(models.RunUpdateTypeDiscard)
		})).Return(&models.ActivityEvent{}, nil)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlanned,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &DiscardRun{
			dbClient:      &db.Client{ActivityEvents: mockActivityEvents},
			in:            &DiscardRunInput{RunID: "run-1"},
			namespacePath: "groupA/ws",
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		require.NotNil(t, cmd.Updated)
		assert.Equal(t, models.RunDiscarded, cmd.Updated.Status)
		// The never-started apply node is marked skipped.
		assert.Equal(t, models.ApplySkipped, cmd.Updated.Apply.Status)
	})

	t.Run("discarding a non-planned run is a conflict and records no activity event", func(t *testing.T) {
		// NewMockActivityEvents(t) asserts no unexpected calls, so a CreateActivityEvent
		// here would fail the test.
		mockActivityEvents := db.NewMockActivityEvents(t)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunApplying,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyRunning},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &DiscardRun{
			dbClient:      &db.Client{ActivityEvents: mockActivityEvents},
			in:            &DiscardRunInput{RunID: "run-1"},
			namespacePath: "groupA/ws",
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
		assert.Nil(t, cmd.Updated)
	})

	t.Run("SkipActivityEvent discards without recording an activity event", func(t *testing.T) {
		// NewMockActivityEvents(t) fails the test on any unexpected call, so a
		// CreateActivityEvent here would fail — verifying the event is suppressed.
		mockActivityEvents := db.NewMockActivityEvents(t)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlanned,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &DiscardRun{
			dbClient: &db.Client{ActivityEvents: mockActivityEvents},
			in:       &DiscardRunInput{RunID: "run-1", SkipActivityEvent: true},
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		require.NotNil(t, cmd.Updated)
		assert.Equal(t, models.RunDiscarded, cmd.Updated.Status)
		mockActivityEvents.AssertNotCalled(t, "CreateActivityEvent", mock.Anything, mock.Anything)
	})
}
