package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestRetryRunNode_Prepare(t *testing.T) {
	bg := context.Background()

	t.Run("resolves the workspace namespace path", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").
			Return(&models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, WorkspaceID: "ws-1"}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", bg, "ws-1").Return(&models.Workspace{FullPath: "groupA/ws"}, nil)

		cmd := &RetryRunNode{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &RetryRunNodeInput{RunID: "run-1", NodePath: "plan"},
		}

		require.NoError(t, cmd.Prepare(bg))
		assert.Equal(t, "groupA/ws", cmd.namespacePath)
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", bg, "run-1").Return(nil, nil)

		cmd := &RetryRunNode{
			dbClient: &db.Client{Runs: mockRuns},
			in:       &RetryRunNodeInput{RunID: "run-1"},
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

		cmd := &RetryRunNode{
			dbClient: &db.Client{Runs: mockRuns, Workspaces: mockWorkspaces},
			in:       &RetryRunNodeInput{RunID: "run-1"},
		}

		err := cmd.Prepare(bg)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestRetryRunNode_Execute(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), auth.NewServiceAccountCaller("sa-1", "sa/path", nil, nil, nil))
	errMsg := "boom"
	forceCanceledBy := "user-1"
	now := time.Now().UTC()

	tests := []struct {
		name     string
		run      *models.Run
		nodePath string
		wantCode errors.CodeType
		wantRun  models.RunStatus
		wantNode string // status to assert on the retried node
	}{
		{
			name: "retry failed plan resets plan to pending and run to queuing",
			run: &models.Run{
				Metadata:               models.ResourceMetadata{ID: "run-1"},
				Status:                 models.RunErrored,
				Plan:                   models.Plan{ID: "plan-1", Status: models.PlanErrored, ErrorMessage: &errMsg},
				Apply:                  &models.Apply{ID: "apply-1", Status: models.ApplySkipped},
				ForceCanceled:          true,
				ForceCanceledBy:        &forceCanceledBy,
				ForceCancelAvailableAt: &now,
			},
			nodePath: "plan",
			wantRun:  models.RunQueuing,
			wantNode: string(models.PlanPending),
		},
		{
			name: "retry canceled apply resets apply to pending and run to queuing_apply",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunCanceled,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCanceled, ErrorMessage: &errMsg},
			},
			nodePath: "apply",
			wantRun:  models.RunQueuingApply,
			wantNode: string(models.ApplyPending),
		},
		{
			name: "retry a finished plan yields conflict",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPlanned,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
			},
			nodePath: "plan",
			wantCode: errors.EConflict,
		},
		{
			name: "invalid node path yields invalid error",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunErrored,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanErrored},
			},
			nodePath: "bogus",
			wantCode: errors.EInvalid,
		},
		{
			name: "retry apply on a speculative run yields invalid error",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunErrored,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanErrored},
			},
			nodePath: "apply",
			wantCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStore := store.NewRunStore(&db.Client{})
			runStore.AddRun(tt.run)

			// On success the retry is recorded as an UPDATE activity event whose payload
			// identifies the retry sub-action and the node path that was retried. On the
			// error cases Execute returns before recording, so NewMockActivityEvents(t)
			// (which fails on any unexpected call) verifies no event is created.
			mockActivityEvents := db.NewMockActivityEvents(t)
			if tt.wantCode == "" {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
					var payload models.ActivityEventUpdateRunPayload
					if err := json.Unmarshal(in.Payload, &payload); err != nil {
						return false
					}
					return in.Action == models.ActionUpdate &&
						in.TargetType == models.TargetRun &&
						payload.Type == string(models.RunUpdateTypeRetry) &&
						payload.NodePath != nil && *payload.NodePath == tt.nodePath
				})).Return(&models.ActivityEvent{}, nil)
			}

			cmd := &RetryRunNode{
				dbClient:      &db.Client{ActivityEvents: mockActivityEvents},
				in:            &RetryRunNodeInput{RunID: "run-1", NodePath: tt.nodePath},
				namespacePath: "groupA/ws",
			}

			err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})

			if tt.wantCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, errors.ErrorCode(err))
				assert.Nil(t, cmd.Updated)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cmd.Updated)
			assert.Equal(t, tt.wantRun, cmd.Updated.Status)
			assert.False(t, cmd.Updated.ForceCanceled, "force-cancellation state should be cleared on retry")
			assert.Nil(t, cmd.Updated.ForceCanceledBy)
			assert.Nil(t, cmd.Updated.ForceCancelAvailableAt)

			switch tt.nodePath {
			case "plan":
				assert.Equal(t, models.PlanStatus(tt.wantNode), cmd.Updated.Plan.Status)
				assert.Nil(t, cmd.Updated.Plan.ErrorMessage, "plan error message should be cleared on retry")
			case "apply":
				assert.Equal(t, models.ApplyStatus(tt.wantNode), cmd.Updated.Apply.Status)
				assert.Nil(t, cmd.Updated.Apply.ErrorMessage, "apply error message should be cleared on retry")
			}
		})
	}
}
