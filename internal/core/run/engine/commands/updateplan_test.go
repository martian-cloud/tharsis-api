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

func TestUpdatePlan_Prepare(t *testing.T) {
	ctx := context.Background()
	msg := "plan failed"

	t.Run("resolves run id and sanitizes message", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		cmd := &UpdatePlan{dbClient: &db.Client{Runs: mockRuns}, PlanID: "plan-1", ErrorMessage: &msg}
		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, "run-1", cmd.runID)
		require.NotNil(t, cmd.sanitizedMessage)
		assert.Equal(t, "plan failed", *cmd.sanitizedMessage)
	})

	t.Run("nil error message leaves sanitized message nil", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		cmd := &UpdatePlan{dbClient: &db.Client{Runs: mockRuns}, PlanID: "plan-1"}
		require.NoError(t, cmd.Prepare(ctx))
		assert.Nil(t, cmd.sanitizedMessage)
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(nil, nil)

		cmd := &UpdatePlan{dbClient: &db.Client{Runs: mockRuns}, PlanID: "plan-1"}
		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestUpdatePlan_Execute(t *testing.T) {
	ctx := context.Background()
	msg := "boom"

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanRunning},
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &UpdatePlan{runID: "run-1", HasChanges: true, sanitizedMessage: &msg}
	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

	require.NotNil(t, cmd.Updated)
	assert.True(t, cmd.Updated.HasChanges)
	require.NotNil(t, cmd.Updated.ErrorMessage)
	assert.Equal(t, "boom", *cmd.Updated.ErrorMessage)
}
