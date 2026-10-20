package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestUploadPlanBinary_Prepare(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Plan:     models.Plan{ID: "plan-1"},
	}

	t.Run("uploads the plan binary and records the run and cache key", func(t *testing.T) {
		body := strings.NewReader("plan bytes")

		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(run, nil)

		retainFn := db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil })
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanCache", ctx, run, body).Return(retainFn, "plan/cache", nil).Once()

		cmd := &UploadPlanBinary{
			dbClient:      &db.Client{Runs: mockRuns},
			artifactStore: mockArtifactStore,
			in:            &UploadPlanBinaryInput{PlanID: "plan-1", Body: body},
		}

		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, "run-1", cmd.runID)
		assert.Equal(t, "plan/cache", cmd.cacheKey)
		assert.NotNil(t, cmd.retainFn)
	})

	t.Run("upload error is returned", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(run, nil)

		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanCache", ctx, run, mock.Anything).
			Return(db.RetainObjectRefFunc(nil), "", errors.New("s3 error"))

		cmd := &UploadPlanBinary{
			dbClient:      &db.Client{Runs: mockRuns},
			artifactStore: mockArtifactStore,
			in:            &UploadPlanBinaryInput{PlanID: "plan-1", Body: strings.NewReader("plan bytes")},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upload plan cache")
		assert.Contains(t, err.Error(), "s3 error")
	})

	t.Run("missing run yields not found and nothing is uploaded", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(nil, nil)

		// No expectations are set on the artifact store, so any upload fails the test.
		cmd := &UploadPlanBinary{
			dbClient:      &db.Client{Runs: mockRuns},
			artifactStore: workspace.NewMockArtifactStore(t),
			in:            &UploadPlanBinaryInput{PlanID: "plan-1", Body: strings.NewReader("plan bytes")},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("database error looking up the run is returned and nothing is uploaded", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(nil, errors.New("db error"))

		cmd := &UploadPlanBinary{
			dbClient:      &db.Client{Runs: mockRuns},
			artifactStore: workspace.NewMockArtifactStore(t),
			in:            &UploadPlanBinaryInput{PlanID: "plan-1", Body: strings.NewReader("plan bytes")},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestUploadPlanBinary_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("records the cache key on the plan node and links the stored object", func(t *testing.T) {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Plan:     models.Plan{ID: "plan-1"},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		mockRefs := db.NewMockObjectStoreRefs(t)
		mockRefs.On("LinkRef", mock.Anything, "plan/cache", db.ObjectStoreRefOwnerRun, "run-1").Return(nil)

		cmd := &UploadPlanBinary{
			runID:    "run-1",
			cacheKey: "plan/cache",
			retainFn: makePlanLinkFunc(mockRefs, "plan/cache"),
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))

		require.NotNil(t, run.Plan.CacheObjectStoreKey)
		assert.Equal(t, "plan/cache", *run.Plan.CacheObjectStoreKey)

		// The command processor only saves the nodes the store reports as changed, so the plan node
		// must be flagged or the cache key would never be persisted.
		changed, err := runStore.GetChangedNodeIDsForRun("run-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"plan-1"}, changed)
	})

	t.Run("link error is returned", func(t *testing.T) {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Plan:     models.Plan{ID: "plan-1"},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		cmd := &UploadPlanBinary{
			runID:    "run-1",
			cacheKey: "plan/cache",
			retainFn: func(_ context.Context, _ string) error { return errors.New("link error") },
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to link plan cache object store ref")
		assert.Contains(t, err.Error(), "link error")
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").Return(nil, nil)

		cmd := &UploadPlanBinary{runID: "run-1", cacheKey: "plan/cache"}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: store.NewRunStore(&db.Client{Runs: mockRuns})})
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}
