package run

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestFindLatestApplyRunForWorkspace(t *testing.T) {
	ctx := context.Background()
	const workspaceID = "ws-1"

	runID := "run-1"

	tests := []struct {
		name           string
		workspace      *models.Workspace
		getWorkspace   error
		stateVersion   *models.StateVersion
		getStateVer    error
		sourceRun      *models.Run
		getRun         error
		expectStateVer bool // whether GetStateVersionByID should be stubbed
		expectGetRun   bool // whether GetRunByID should be stubbed
		wantCode       errors.CodeType
		wantRunID      string
	}{
		{
			name:         "workspace lookup fails",
			getWorkspace: errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			wantCode:     errors.EInternal,
		},
		{
			name:      "workspace not found",
			workspace: nil,
			wantCode:  errors.ENotFound,
		},
		{
			name:      "workspace has no current state version",
			workspace: &models.Workspace{CurrentStateVersionID: ""},
			wantCode:  errors.EConflict,
		},
		{
			name:           "state version not found",
			workspace:      &models.Workspace{CurrentStateVersionID: "sv-1"},
			stateVersion:   nil,
			expectStateVer: true,
			wantCode:       errors.ENotFound,
		},
		{
			name:           "state version created manually has no run",
			workspace:      &models.Workspace{CurrentStateVersionID: "sv-1"},
			stateVersion:   &models.StateVersion{RunID: nil},
			expectStateVer: true,
			wantCode:       errors.EConflict,
		},
		{
			name:           "run associated with state version not found",
			workspace:      &models.Workspace{CurrentStateVersionID: "sv-1"},
			stateVersion:   &models.StateVersion{RunID: &runID},
			expectStateVer: true,
			sourceRun:      nil,
			expectGetRun:   true,
			wantCode:       errors.ENotFound,
		},
		{
			name:           "happy path returns the source run",
			workspace:      &models.Workspace{CurrentStateVersionID: "sv-1"},
			stateVersion:   &models.StateVersion{RunID: &runID},
			expectStateVer: true,
			sourceRun:      &models.Run{Metadata: models.ResourceMetadata{ID: runID}},
			expectGetRun:   true,
			wantRunID:      runID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockStateVersions := db.NewMockStateVersions(t)
			mockRuns := db.NewMockRuns(t)

			mockWorkspaces.On("GetWorkspaceByID", ctx, workspaceID).Return(tt.workspace, tt.getWorkspace)
			if tt.expectStateVer {
				mockStateVersions.On("GetStateVersionByID", ctx, "sv-1").Return(tt.stateVersion, tt.getStateVer)
			}
			if tt.expectGetRun {
				mockRuns.On("GetRunByID", ctx, runID).Return(tt.sourceRun, tt.getRun)
			}

			dbClient := &db.Client{
				Workspaces:    mockWorkspaces,
				StateVersions: mockStateVersions,
				Runs:          mockRuns,
			}

			source, err := FindLatestApplyRunForWorkspace(ctx, dbClient, workspaceID)

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, errors.ErrorCode(err))
				assert.Nil(t, source)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, source)
			assert.Equal(t, tt.wantRunID, source.Metadata.ID)
		})
	}
}
