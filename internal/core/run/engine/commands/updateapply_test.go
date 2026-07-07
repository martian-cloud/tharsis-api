package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestUpdateApply_Prepare(t *testing.T) {
	ctx := context.Background()
	msg := "apply failed"

	t.Run("resolves run and sanitizes message", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "apply-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		cmd := &UpdateApply{dbClient: &db.Client{Runs: mockRuns}, ApplyID: "apply-1", ErrorMessage: &msg}
		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, "run-1", cmd.runID)
		require.NotNil(t, cmd.sanitizedMessage)
		assert.Equal(t, "apply failed", *cmd.sanitizedMessage)
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "apply-1").Return(nil, nil)

		cmd := &UpdateApply{dbClient: &db.Client{Runs: mockRuns}, ApplyID: "apply-1"}
		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestUpdateApply_Execute(t *testing.T) {
	ctx := context.Background()
	msg := "boom"

	t.Run("sets error message on apply node", func(t *testing.T) {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyRunning},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &UpdateApply{runID: "run-1", sanitizedMessage: &msg}
		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		require.NotNil(t, cmd.Updated)
		require.NotNil(t, cmd.Updated.ErrorMessage)
		assert.Equal(t, "boom", *cmd.Updated.ErrorMessage)
	})

	t.Run("no apply node yields invalid", func(t *testing.T) {
		run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, Apply: nil}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &UpdateApply{runID: "run-1"}
		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})
}
