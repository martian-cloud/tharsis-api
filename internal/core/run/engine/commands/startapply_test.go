package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestStartApply_Prepare(t *testing.T) {
	ctx := context.Background()

	t.Run("no managed identities returns early without enforcing rules", func(t *testing.T) {
		// With no managed identities assigned, Prepare returns before fetching the
		// workspace or enforcing any rules. NewMock*(t) fail on unexpected calls, so
		// not stubbing those interfaces verifies they are never touched.
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").Return(&models.Run{
			Metadata:    models.ResourceMetadata{ID: "run-1"},
			WorkspaceID: "ws-1",
		}, nil)

		mockManagedIdentities := db.NewMockManagedIdentities(t)
		mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").
			Return([]models.ManagedIdentity{}, nil)

		cmd := &StartApply{
			dbClient:     &db.Client{Runs: mockRuns, ManagedIdentities: mockManagedIdentities},
			ruleEnforcer: rules.NewMockRuleEnforcer(t),
			in:           &StartApplyInput{RunID: "run-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))
	})

	t.Run("enforces rules for each managed identity on the happy path", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").Return(&models.Run{
			Metadata:    models.ResourceMetadata{ID: "run-1"},
			WorkspaceID: "ws-1",
			// ModuleSource/ModuleDigest nil -> ParseModuleRegistrySource is not called.
		}, nil)

		mockManagedIdentities := db.NewMockManagedIdentities(t)
		mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return([]models.ManagedIdentity{
			{Metadata: models.ResourceMetadata{ID: "mi-1"}},
			{Metadata: models.ResourceMetadata{ID: "mi-2"}},
		}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").
			Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

		mockRuleEnforcer := rules.NewMockRuleEnforcer(t)
		// Each assigned managed identity must be checked, with an apply-stage RunDetails.
		mockRuleEnforcer.On("EnforceRules", ctx, mock.MatchedBy(func(mi *models.ManagedIdentity) bool {
			return mi.Metadata.ID == "mi-1"
		}), mock.MatchedBy(func(d *rules.RunDetails) bool {
			return d.RunStage == models.JobApplyType
		})).Return(nil)
		mockRuleEnforcer.On("EnforceRules", ctx, mock.MatchedBy(func(mi *models.ManagedIdentity) bool {
			return mi.Metadata.ID == "mi-2"
		}), mock.Anything).Return(nil)

		cmd := &StartApply{
			dbClient:     &db.Client{Runs: mockRuns, ManagedIdentities: mockManagedIdentities, Workspaces: mockWorkspaces},
			ruleEnforcer: mockRuleEnforcer,
			in:           &StartApplyInput{RunID: "run-1"},
		}

		require.NoError(t, cmd.Prepare(ctx))
	})

	t.Run("rule enforcement denial is returned", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").Return(&models.Run{
			Metadata:    models.ResourceMetadata{ID: "run-1"},
			WorkspaceID: "ws-1",
		}, nil)

		mockManagedIdentities := db.NewMockManagedIdentities(t)
		mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", ctx, "ws-1").Return([]models.ManagedIdentity{
			{Metadata: models.ResourceMetadata{ID: "mi-1"}},
		}, nil)

		mockWorkspaces := db.NewMockWorkspaces(t)
		mockWorkspaces.On("GetWorkspaceByID", ctx, "ws-1").
			Return(&models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}, nil)

		mockRuleEnforcer := rules.NewMockRuleEnforcer(t)
		mockRuleEnforcer.On("EnforceRules", ctx, mock.Anything, mock.Anything).
			Return(errors.New("rule denied", errors.WithErrorCode(errors.EForbidden)))

		cmd := &StartApply{
			dbClient:     &db.Client{Runs: mockRuns, ManagedIdentities: mockManagedIdentities, Workspaces: mockWorkspaces},
			ruleEnforcer: mockRuleEnforcer,
			in:           &StartApplyInput{RunID: "run-1"},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})

	t.Run("missing run yields not found", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockRuns.On("GetRunByID", ctx, "run-1").Return(nil, nil)

		cmd := &StartApply{
			dbClient: &db.Client{Runs: mockRuns},
			in:       &StartApplyInput{RunID: "run-1"},
		}

		err := cmd.Prepare(ctx)
		require.Error(t, err)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})
}

func TestStartApply_Execute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		run       *models.Run
		wantCode  errors.CodeType
		wantApply models.ApplyStatus // expected apply status on success
	}{
		{
			name: "no apply node yields conflict",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPlanned,
				Apply:    nil,
			},
			wantCode: errors.EConflict,
		},
		{
			name: "apply not in created state yields conflict",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPlanned,
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyPending},
			},
			wantCode: errors.EConflict,
		},
		{
			name: "run not in planned state yields conflict",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPlanning,
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
			},
			wantCode: errors.EConflict,
		},
		{
			name: "planned run with created apply transitions apply to pending",
			run: &models.Run{
				Metadata: models.ResourceMetadata{ID: "run-1"},
				Status:   models.RunPlanned,
				Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
				Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
			},
			wantApply: models.ApplyPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runStore := store.NewRunStore(&db.Client{})
			runStore.AddRun(tt.run)

			cmd := &StartApply{
				in: &StartApplyInput{
					RunID:       "run-1",
					TriggeredBy: "user@example.com",
					Comment:     "go",
				},
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
			assert.Equal(t, tt.wantApply, cmd.Updated.Apply.Status)
			assert.Equal(t, "user@example.com", cmd.Updated.Apply.TriggeredBy)
			assert.Equal(t, "go", cmd.Updated.Apply.Comment)
		})
	}
}
