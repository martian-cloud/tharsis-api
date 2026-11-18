package commands

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestCreateAssessmentRun_Prepare(t *testing.T) {
	ctx := context.Background()

	// The latest run is a destroy run, so an assessment cannot be derived from it.
	workspaces := db.NewMockWorkspaces(t)
	stateVersions := db.NewMockStateVersions(t)
	runs := db.NewMockRuns(t)
	dbClient := &db.Client{Workspaces: workspaces, StateVersions: stateVersions, Runs: runs}

	workspaces.On("GetWorkspaceByID", ctx, "ws-1").Return(&models.Workspace{CurrentStateVersionID: "sv-1"}, nil)
	stateVersions.On("GetStateVersionByID", ctx, "sv-1").Return(&models.StateVersion{RunID: ptr.String("src")}, nil)
	runs.On("GetRunByID", ctx, "src").Return(&models.Run{Metadata: models.ResourceMetadata{ID: "src"}, IsDestroy: true}, nil)

	cmd := &CreateAssessmentRun{
		dbClient: dbClient,
		in:       &CreateAssessmentRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1"},
	}

	err := cmd.Prepare(ctx)
	require.Error(t, err)
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
}

func TestCreateAssessmentRun_Execute(t *testing.T) {
	completed := time.Now().UTC().Add(-time.Hour)
	stale := time.Now().UTC().Add(-2 * models.AssessmentStaleTimeout)
	recent := time.Now().UTC()

	tests := []struct {
		name             string
		latestVersionArg *int
		existing         *models.WorkspaceAssessment
		expectCreate     bool
		expectUpdate     bool
		wantCode         errors.CodeType
	}{
		{
			name:         "no existing assessment creates one and the run",
			existing:     nil,
			expectCreate: true,
		},
		{
			name:             "no existing assessment but a version supplied is a conflict",
			latestVersionArg: ptr.Int(1),
			existing:         nil,
			wantCode:         errors.EConflict,
		},
		{
			name:             "version mismatch is a conflict",
			latestVersionArg: ptr.Int(3),
			existing:         &models.WorkspaceAssessment{Metadata: models.ResourceMetadata{Version: 5}, CompletedAtTimestamp: &completed},
			wantCode:         errors.EConflict,
		},
		{
			name:             "assessment recently updated and in progress is a conflict",
			latestVersionArg: ptr.Int(5),
			existing:         &models.WorkspaceAssessment{Metadata: models.ResourceMetadata{Version: 5, LastUpdatedTimestamp: &recent}, CompletedAtTimestamp: nil},
			wantCode:         errors.EConflict,
		},
		{
			name:             "stale in-progress assessment is restarted",
			latestVersionArg: ptr.Int(5),
			existing:         &models.WorkspaceAssessment{Metadata: models.ResourceMetadata{ID: "wa-1", Version: 5, LastUpdatedTimestamp: &stale}, CompletedAtTimestamp: nil},
			expectUpdate:     true,
		},
		{
			name:         "completed assessment is updated and the run created",
			existing:     &models.WorkspaceAssessment{Metadata: models.ResourceMetadata{ID: "wa-1", Version: 5}, CompletedAtTimestamp: &completed},
			expectUpdate: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			workspaceAssessments := db.NewMockWorkspaceAssessments(t)
			workspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", ctx, "ws-1").Return(test.existing, nil)
			if test.expectCreate {
				workspaceAssessments.On("CreateWorkspaceAssessment", ctx, mock.Anything).Return(&models.WorkspaceAssessment{}, nil)
			}
			if test.expectUpdate {
				workspaceAssessments.On("UpdateWorkspaceAssessment", ctx, mock.Anything).Return(&models.WorkspaceAssessment{}, nil)
			}

			createCalled := false
			cmd := &CreateAssessmentRun{
				dbClient:          &db.Client{WorkspaceAssessments: workspaceAssessments},
				variablesRetainFn: func(_ context.Context, _ string) error { return nil },
				createRun: func(_ context.Context, _ *corerun.CreateRunInput) (*models.Run, error) {
					createCalled = true
					return &models.Run{
						Metadata: models.ResourceMetadata{ID: "run-new"},
						Status:   models.RunPending,
						Plan:     models.Plan{Status: models.PlanCreated},
					}, nil
				},
				in:          &CreateAssessmentRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1", LatestAssessmentVersion: test.latestVersionArg},
				createInput: &corerun.CreateRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1"},
			}

			runStore := store.NewRunStore(&db.Client{})
			err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})

			if test.wantCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.wantCode, errors.ErrorCode(err))
				assert.Nil(t, cmd.Created)
				// A failed assessment upsert must short-circuit before the run is created.
				assert.False(t, createCalled, "createRun must not run when the assessment upsert fails")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd.Created)
			assert.True(t, createCalled)
			assert.Equal(t, models.RunPlanQueuing, cmd.Created.Status)
		})
	}
}

// TestCreateAssessmentRun_Execute_ClearsErrorMessage covers the other half of recording an
// assessment failure: once resolution succeeds the stored reason is stale and must not linger.
// Reaching Execute is itself the proof, because Prepare resolved the source run and module
// before the transaction opened.
func TestCreateAssessmentRun_Execute_ClearsErrorMessage(t *testing.T) {
	completed := time.Now().UTC().Add(-time.Hour)

	t.Run("a recorded reason is cleared when the next assessment run is created", func(t *testing.T) {
		ctx := context.Background()

		existing := &models.WorkspaceAssessment{
			Metadata:             models.ResourceMetadata{ID: "wa-1", Version: 5},
			CompletedAtTimestamp: &completed,
			HasDrift:             true,
			ErrorMessage:         ptr.String("module not found for source registry/example/aws"),
			AvailableAtTimestamp: ptr.Time(time.Now().UTC().Add(4 * time.Hour)),
			ConsecutiveFailures:  3,
		}

		workspaceAssessments := db.NewMockWorkspaceAssessments(t)
		workspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", ctx, "ws-1").Return(existing, nil)
		workspaceAssessments.On("UpdateWorkspaceAssessment", ctx, mock.MatchedBy(func(a *models.WorkspaceAssessment) bool {
			// The drift verdict is untouched; only the failure reason, the backoff state, and
			// the timestamps move. Creation succeeding ends the run of consecutive failures
			// backoff was tracking, so both reset alongside the error message.
			return a.ErrorMessage == nil && a.CompletedAtTimestamp == nil && a.HasDrift &&
				a.AvailableAtTimestamp == nil && a.ConsecutiveFailures == 0
		})).Return(&models.WorkspaceAssessment{}, nil).Once()

		cmd := &CreateAssessmentRun{
			dbClient:          &db.Client{WorkspaceAssessments: workspaceAssessments},
			variablesRetainFn: func(_ context.Context, _ string) error { return nil },
			createRun: func(_ context.Context, _ *corerun.CreateRunInput) (*models.Run, error) {
				return &models.Run{
					Metadata: models.ResourceMetadata{ID: "run-new"},
					Status:   models.RunPending,
					Plan:     models.Plan{Status: models.PlanCreated},
				}, nil
			},
			in:          &CreateAssessmentRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1"},
			createInput: &corerun.CreateRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1"},
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: store.NewRunStore(&db.Client{})})

		require.NoError(t, err)
		require.NotNil(t, cmd.Created)
	})

	t.Run("the in-progress conflict guard still takes precedence over clearing", func(t *testing.T) {
		ctx := context.Background()
		recent := time.Now().UTC()

		// An in-progress, non-stale assessment must still be rejected, so no update happens and
		// the recorded reason is left alone.
		existing := &models.WorkspaceAssessment{
			Metadata:             models.ResourceMetadata{ID: "wa-1", Version: 5, LastUpdatedTimestamp: &recent},
			CompletedAtTimestamp: nil,
			ErrorMessage:         ptr.String("module not found for source registry/example/aws"),
		}

		workspaceAssessments := db.NewMockWorkspaceAssessments(t)
		workspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", ctx, "ws-1").Return(existing, nil)

		createCalled := false
		cmd := &CreateAssessmentRun{
			dbClient:          &db.Client{WorkspaceAssessments: workspaceAssessments},
			variablesRetainFn: func(_ context.Context, _ string) error { return nil },
			createRun: func(_ context.Context, _ *corerun.CreateRunInput) (*models.Run, error) {
				createCalled = true
				return nil, nil
			},
			in:          &CreateAssessmentRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1"},
			createInput: &corerun.CreateRunInput{CreatedBy: models.Actor{Subject: "u"}, WorkspaceID: "ws-1"},
		}

		err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: store.NewRunStore(&db.Client{})})

		require.Error(t, err)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
		assert.False(t, createCalled)
		require.NotNil(t, existing.ErrorMessage, "a rejected attempt must not clear the recorded reason")
	})
}
