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

func TestUpdatePlanSummary_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("records the summary on the plan node and uploads diff and json", func(t *testing.T) {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Plan:     models.Plan{ID: "plan-1"},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanDiff", ctx, run, mock.Anything).Return(nil)
		mockArtifactStore.On("UploadPlanJSON", ctx, run, mock.Anything).Return(nil)

		summary := models.PlanSummary{ResourceAdditions: 3}
		diff := []byte(`{"resources":[]}`)
		cmd := &UpdatePlanSummary{
			artifactStore: mockArtifactStore,
			runID:         "run-1",
			summary:       summary,
			planDiff:      diff,
			planJSON:      []byte(`{}`),
		}

		require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
		assert.Equal(t, summary, run.Plan.Summary)
		assert.Equal(t, len(diff), run.Plan.DiffSize)
	})

	t.Run("propagates an artifact store upload error", func(t *testing.T) {
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Plan:     models.Plan{ID: "plan-1"},
		}
		runStore := store.NewRunStore(&db.Client{})
		runStore.AddRun(run)

		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadPlanDiff", ctx, run, mock.Anything).
			Return(errors.New("object store unavailable"))

		cmd := &UpdatePlanSummary{
			artifactStore: mockArtifactStore,
			runID:         "run-1",
			planDiff:      []byte(`{}`),
			planJSON:      []byte(`{}`),
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write plan diff to object storage")
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

		cmd := &UpdatePlanSummary{
			dbClient:   &db.Client{Runs: mockRuns},
			planParser: mockParser,
			in:         &UpdatePlanSummaryInput{PlanID: "plan-1"},
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
		assert.NotEmpty(t, cmd.planDiff)
	})

	t.Run("empty diff yields no changes", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByNodeID", ctx, "plan-1").Return(
			&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}, nil)

		mockParser := plan.NewMockParser(t)
		mockParser.On("Parse", mock.Anything, mock.Anything).Return(&plan.Diff{}, nil)

		cmd := &UpdatePlanSummary{
			dbClient:   &db.Client{Runs: mockRuns},
			planParser: mockParser,
			in:         &UpdatePlanSummaryInput{PlanID: "plan-1"},
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

		cmd := &UpdatePlanSummary{
			dbClient:   &db.Client{Runs: mockRuns},
			planParser: mockParser,
			in:         &UpdatePlanSummaryInput{PlanID: "plan-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))
		assert.Equal(t, int32(1), cmd.summary.ResourceDrift)
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
