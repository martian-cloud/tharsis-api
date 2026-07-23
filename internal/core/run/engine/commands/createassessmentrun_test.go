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
		in:       &CreateAssessmentRunInput{Subject: "u", WorkspaceID: "ws-1"},
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
				in:          &CreateAssessmentRunInput{Subject: "u", WorkspaceID: "ws-1", LatestAssessmentVersion: test.latestVersionArg},
				createInput: &corerun.CreateRunInput{Subject: "u", WorkspaceID: "ws-1"},
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
			assert.Equal(t, models.RunQueuing, cmd.Created.Status)
		})
	}
}
