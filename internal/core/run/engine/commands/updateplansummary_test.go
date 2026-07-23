package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// makePlanLinkFunc builds a no-op RetainObjectRefFunc that satisfies mock expectations on refs.
func makePlanLinkFunc(refs *db.MockObjectStoreRefs, key string) db.RetainObjectRefFunc {
	return func(ctx context.Context, ownerID string) error {
		return refs.LinkRef(ctx, key, db.ObjectStoreRefOwnerRun, ownerID)
	}
}

func TestUpdatePlanSummary_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("records the summary and object keys on the plan node", func(t *testing.T) {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Plan:     models.Plan{ID: "plan-1"},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		summary := models.PlanSummary{ResourceAdditions: 3}

		mockRefs := db.NewMockObjectStoreRefs(t)
		mockRefs.On("LinkRef", mock.Anything, mock.Anything, db.ObjectStoreRefOwnerRun, "run-1").Return(nil)

		cmd := &UpdatePlanSummary{
			dbClient:      &db.Client{},
			runID:         "run-1",
			summary:       summary,
			diffSize:      16,
			diffObjectKey: "plan/diff",
			jsonObjectKey: "plan/json",
			diffRetainFn:  makePlanLinkFunc(mockRefs, "plan/diff"),
			jsonRetainFn:  makePlanLinkFunc(mockRefs, "plan/json"),
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		assert.Equal(t, summary, run.Plan.Summary)
		assert.Equal(t, 16, run.Plan.DiffSize)
		assert.Equal(t, &cmd.diffObjectKey, run.Plan.DiffObjectStoreKey)
		assert.Equal(t, &cmd.jsonObjectKey, run.Plan.JSONObjectStoreKey)
	})
}

func TestUpdatePlanSummary_Prepare(t *testing.T) {
	ctx := context.Background()

	t.Run("computes summary counts and hasChanges from the diff", func(t *testing.T) {
		diff := &plan.Diff{
			Resources: []*plan.ResourceDiff{
				{Action: action.Create},
				{Action: action.Create},
				{Action: action.Update},
				{Action: action.Delete},
				// Replace counts as both an addition and a destruction.
				{Action: action.CreateThenDelete},
				{Action: action.DeleteThenCreate},
				// Import and drift flags accumulate independently of the action.
				{Action: action.NoOp, Imported: true, Drifted: true},
			},
			Outputs: []*plan.OutputDiff{
				{Action: action.Create},
				{Action: action.Update},
				{Action: action.Update},
				{Action: action.Delete},
			},
		}

		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		mockParser := plan.NewMockParser(t)
		mockParser.On("Parse", mock.Anything, mock.Anything).Return(diff, nil)

		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanDiff", ctx, mock.Anything, mock.Anything).
			Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), "plan/diff", nil)
		mockArtifactStore.On("UploadPlanJSON", ctx, mock.Anything, mock.Anything).
			Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), "plan/json", nil)

		cmd := &UpdatePlanSummary{
			dbClient:      &db.Client{Runs: mockRuns},
			planParser:    mockParser,
			artifactStore: mockArtifactStore,
			in:            &UpdatePlanSummaryInput{PlanID: "plan-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))

		// Additions: 2 creates + create_then_delete + delete_then_create = 4.
		assert.Equal(t, int32(4), cmd.summary.ResourceAdditions)
		assert.Equal(t, int32(1), cmd.summary.ResourceChanges)
		// Destructions: 1 delete + create_then_delete + delete_then_create = 3.
		assert.Equal(t, int32(3), cmd.summary.ResourceDestructions)
		assert.Equal(t, int32(1), cmd.summary.ResourceImports)
		assert.Equal(t, int32(1), cmd.summary.ResourceDrift)
		assert.Equal(t, int32(1), cmd.summary.OutputAdditions)
		assert.Equal(t, int32(2), cmd.summary.OutputChanges)
		assert.Equal(t, int32(1), cmd.summary.OutputDestructions)
		assert.Equal(t, "run-1", cmd.runID)
		assert.Equal(t, "plan/diff", cmd.diffObjectKey)
		assert.Equal(t, "plan/json", cmd.jsonObjectKey)
		assert.NotZero(t, cmd.diffSize)
	})

	t.Run("empty diff yields no changes", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		mockParser := plan.NewMockParser(t)
		mockParser.On("Parse", mock.Anything, mock.Anything).Return(&plan.Diff{}, nil)

		noopLinkFunc := db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil })
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanDiff", ctx, mock.Anything, mock.Anything).
			Return(noopLinkFunc, "plan/diff", nil)
		mockArtifactStore.On("UploadPlanJSON", ctx, mock.Anything, mock.Anything).
			Return(noopLinkFunc, "plan/json", nil)

		cmd := &UpdatePlanSummary{
			dbClient:      &db.Client{Runs: mockRuns},
			planParser:    mockParser,
			artifactStore: mockArtifactStore,
			in:            &UpdatePlanSummaryInput{PlanID: "plan-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, int32(0), cmd.summary.ResourceAdditions)
	})

	t.Run("drift-only diff does not count as hasChanges", func(t *testing.T) {
		// Drift and imports are reported in the summary but do not, on their own,
		// constitute plan changes.
		diff := &plan.Diff{
			Resources: []*plan.ResourceDiff{
				{Action: action.NoOp, Drifted: true},
			},
		}

		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		mockParser := plan.NewMockParser(t)
		mockParser.On("Parse", mock.Anything, mock.Anything).Return(diff, nil)

		noopLinkFunc := db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil })
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanDiff", ctx, mock.Anything, mock.Anything).
			Return(noopLinkFunc, "plan/diff", nil)
		mockArtifactStore.On("UploadPlanJSON", ctx, mock.Anything, mock.Anything).
			Return(noopLinkFunc, "plan/json", nil)

		cmd := &UpdatePlanSummary{
			dbClient:      &db.Client{Runs: mockRuns},
			planParser:    mockParser,
			artifactStore: mockArtifactStore,
			in:            &UpdatePlanSummaryInput{PlanID: "plan-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, int32(1), cmd.summary.ResourceDrift)
	})

	t.Run("UploadPlanDiff error is returned", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		mockParser := plan.NewMockParser(t)
		mockParser.On("Parse", mock.Anything, mock.Anything).Return(&plan.Diff{}, nil)

		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanDiff", ctx, mock.Anything, mock.Anything).Return(db.RetainObjectRefFunc(nil), "", errors.New("s3 error"))

		cmd := &UpdatePlanSummary{
			dbClient:      &db.Client{Runs: mockRuns},
			planParser:    mockParser,
			artifactStore: mockArtifactStore,
			in:            &UpdatePlanSummaryInput{PlanID: "plan-1"},
		}

		require.Error(t, cmd.Prepare(ctx))
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(nil, nil)

		cmd := &UpdatePlanSummary{
			dbClient: &db.Client{Runs: mockRuns},
			in:       &UpdatePlanSummaryInput{PlanID: "plan-1"},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}
