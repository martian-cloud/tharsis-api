package run

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

type mockDBClient struct {
	*db.Client
	MockTransactions          *db.MockTransactions
	MockManagedIdentities     *db.MockManagedIdentities
	MockWorkspaces            *db.MockWorkspaces
	MockVariables             *db.MockVariables
	MockRuns                  *db.MockRuns
	MockConfigurationVersions *db.MockConfigurationVersions
	MockApplies               *db.MockApplies
	MockPlans                 *db.MockPlans
	MockJobs                  *db.MockJobs
	MockTeams                 *db.MockTeams
	MockTeamMembers           *db.MockTeamMembers
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)
	// The mocks are enabled by the above function.

	mockManagedIdentities := db.MockManagedIdentities{}
	mockManagedIdentities.Test(t)

	mockWorkspaces := db.MockWorkspaces{}
	mockWorkspaces.Test(t)

	mockVariables := db.MockVariables{}
	mockVariables.Test(t)

	mockRuns := db.MockRuns{}
	mockRuns.Test(t)

	mockConfigurationVersions := db.MockConfigurationVersions{}
	mockConfigurationVersions.Test(t)

	mockPlans := db.MockPlans{}
	mockPlans.Test(t)

	mockApplies := db.MockApplies{}
	mockApplies.Test(t)

	mockJobs := db.MockJobs{}
	mockJobs.Test(t)

	mockTeams := db.MockTeams{}
	mockTeams.Test(t)

	mockTeamMembers := db.MockTeamMembers{}
	mockTeamMembers.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:          &mockTransactions,
			ManagedIdentities:     &mockManagedIdentities,
			Workspaces:            &mockWorkspaces,
			Variables:             &mockVariables,
			Runs:                  &mockRuns,
			ConfigurationVersions: &mockConfigurationVersions,
			Applies:               &mockApplies,
			Plans:                 &mockPlans,
			Jobs:                  &mockJobs,
			Teams:                 &mockTeams,
			TeamMembers:           &mockTeamMembers,
		},
		MockTransactions:          &mockTransactions,
		MockManagedIdentities:     &mockManagedIdentities,
		MockWorkspaces:            &mockWorkspaces,
		MockVariables:             &mockVariables,
		MockRuns:                  &mockRuns,
		MockConfigurationVersions: &mockConfigurationVersions,
		MockApplies:               &mockApplies,
		MockPlans:                 &mockPlans,
		MockJobs:                  &mockJobs,
		MockTeams:                 &mockTeams,
		MockTeamMembers:           &mockTeamMembers,
	}
}

func TestCreateRunWithManagedIdentityAccessRules(t *testing.T) {
	configurationVersionID := "cv1"

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:       "groupA/ws1",
		MaxJobDuration: ptr.Int32(60),
	}

	run := models.Run{
		Metadata: models.ResourceMetadata{
			ID: "run1",
		},
		WorkspaceID:            ws.Metadata.ID,
		ConfigurationVersionID: &configurationVersionID,
		Status:                 models.RunPending,
	}

	// Test cases
	tests := []struct {
		name                 string
		expectErrorCode      string
		enforceRulesResponse error
		managedIdentities    []models.ManagedIdentity
	}{
		{
			name: "run is created because all managed identity rules are satisfied",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
				},
			},
		},
		{
			name:              "run is created because there are no managed identities",
			managedIdentities: []models.ManagedIdentity{},
		},
		{
			name: "run is not created because a managed identity rule is not satisfied",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
				},
			},
			enforceRulesResponse: errors.NewError(errors.EForbidden, "rule not satisfied"),
			expectErrorCode:      errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequireAccessToWorkspace", mock.Anything, ws.Metadata.ID, models.DeployerRole).Return(nil)
			mockCaller.On("GetSubject").Return("mock-caller").Maybe()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).Return(test.managedIdentities, nil)

			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil)

			dbClient.MockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.VariableResult{
				Variables: []models.Variable{},
			}, nil)

			dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).Return(&run, nil)
			dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(&run, nil)

			dbClient.MockConfigurationVersions.On("GetConfigurationVersion", mock.Anything, configurationVersionID).Return(&models.ConfigurationVersion{
				Speculative: false,
			}, nil)

			dbClient.MockPlans.On("CreatePlan", mock.Anything, mock.Anything).Return(&models.Plan{
				Metadata: models.ResourceMetadata{
					ID: "plan1",
				},
			}, nil)

			dbClient.MockApplies.On("CreateApply", mock.Anything, mock.Anything).Return(&models.Apply{
				Metadata: models.ResourceMetadata{
					ID: "apply1",
				},
			}, nil)
			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(nil, nil)

			mockArtifactStore := workspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			mockModuleService := moduleregistry.NewMockService(t)
			mockModuleResolver := NewMockModuleResolver(t)
			ruleEnforcer := rules.NewMockRuleEnforcer(t)

			for _, mi := range test.managedIdentities {
				miCopy := mi
				ruleEnforcer.On("EnforceRules", mock.Anything, &miCopy, mock.Anything).Return(test.enforceRulesResponse)
			}

			logger, _ := logger.NewForTest()
			service := newService(
				logger,
				dbClient.Client,
				&mockArtifactStore,
				nil,
				nil,
				nil,
				&mockActivityEvents,
				mockModuleService,
				mockModuleResolver,
				nil,
				ruleEnforcer,
			)

			_, err := service.CreateRun(auth.WithCaller(ctx, mockCaller), &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
			})
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateRunWithPreventDestroy(t *testing.T) {
	configurationVersionID := "cv1"
	var duration int32 = 720

	mockAuthorizer := auth.MockAuthorizer{}
	mockAuthorizer.Test(t)

	// Test cases
	type testCase struct {
		name            string
		workspace       *models.Workspace
		runInput        *CreateRunInput
		expectErrorCode string
	}

	/*
		Test case template.
		name            string
		workspace       *models.Workspace
		runInput        *CreateRunInput
		expectErrorCode string
	*/

	tests := []testCase{

		{
			name: "non-destroy plan is allowed independent of PreventDestroyPlan",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-1",
				},
				MaxJobDuration:     &duration,
				PreventDestroyPlan: false,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-1",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              false,
			},
		},

		{
			name: "destroy plan is allowed, because PreventDestroyPlan is falsee",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-2",
				},
				MaxJobDuration:     &duration,
				PreventDestroyPlan: false,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-2",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              true,
			},
		},

		{
			name: "non-destroy plan is allowed even when PreventDestroyPlan is true",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-3",
				},
				MaxJobDuration:     &duration,
				PreventDestroyPlan: true,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-3",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              false,
			},
		},

		{
			name: "destroy plan is NOT allowed, because PreventDestroyPlan is true",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-4",
				},
				MaxJobDuration:     &duration,
				PreventDestroyPlan: true,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-4",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              true,
			},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			testCaller := auth.NewUserCaller(
				&models.User{
					Metadata: models.ResourceMetadata{
						ID: "123",
					},
					Admin:    false,
					Username: "user1",
				},
				&mockAuthorizer,
				dbClient.Client, // was nil
			)

			run := models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:            test.workspace.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Status:                 models.RunPending,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockAuthorizer.On("RequireAccessToWorkspace",
				mock.Anything, test.workspace.Metadata.ID, models.DeployerRole).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace",
				mock.Anything, test.workspace.Metadata.ID).Return([]models.ManagedIdentity{}, nil)

			dbClient.MockWorkspaces.On("GetWorkspaceByID",
				mock.Anything, test.workspace.Metadata.ID).Return(test.workspace, nil)

			dbClient.MockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.VariableResult{
				Variables: []models.Variable{},
			}, nil)

			dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).Return(&run, nil)

			dbClient.MockConfigurationVersions.On("GetConfigurationVersion", mock.Anything, configurationVersionID).Return(&models.ConfigurationVersion{
				Speculative: false,
			}, nil)

			dbClient.MockPlans.On("CreatePlan", mock.Anything, mock.Anything).Return(&models.Plan{
				Metadata: models.ResourceMetadata{
					ID: "plan1",
				},
			}, nil)

			dbClient.MockApplies.On("CreateApply", mock.Anything, mock.Anything).Return(&models.Apply{
				Metadata: models.ResourceMetadata{
					ID: "apply1",
				},
			}, nil)
			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(nil, nil)

			mockArtifactStore := workspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			logger, _ := logger.NewForTest()

			service := NewService(logger, dbClient.Client, &mockArtifactStore, nil, nil, nil, &mockActivityEvents, nil, nil, nil)

			_, err := service.CreateRun(auth.WithCaller(ctx, testCaller), test.runInput)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestApplyRunWithManagedIdentityAccessRules(t *testing.T) {
	var duration int32 = 1
	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:       "groupA/ws1",
		MaxJobDuration: &duration,
	}

	run := models.Run{
		Metadata: models.ResourceMetadata{
			ID: "run1",
		},
		WorkspaceID: ws.Metadata.ID,
	}

	apply := models.Apply{
		Metadata: models.ResourceMetadata{
			ID: "apply1",
		},
	}

	// Test cases
	tests := []struct {
		name                 string
		expectErrorCode      string
		enforceRulesResponse error
		managedIdentities    []models.ManagedIdentity
	}{
		{
			name: "apply is created because all managed identity rules are satisfied",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
				},
			},
		},
		{
			name:              "apply is created because there are no managed identities",
			managedIdentities: []models.ManagedIdentity{},
		},
		{
			name: "apply is not created because a managed identity rule is not satisfied",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
				},
			},
			enforceRulesResponse: errors.NewError(errors.EForbidden, "rule not satisfied"),
			expectErrorCode:      errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequireAccessToWorkspace", mock.Anything, ws.Metadata.ID, models.DeployerRole).Return(nil)
			mockCaller.On("GetSubject").Return("mock-caller").Maybe()

			ctx, cancel := context.WithCancel(auth.WithCaller(context.Background(), mockCaller))
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).Return(test.managedIdentities, nil)

			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil)

			apply.Status = models.ApplyCreated // to avoid tripping the state transition checks in UpdateApply, etc.

			dbClient.MockRuns.On("GetRun", mock.Anything, run.Metadata.ID).Return(&run, nil)
			dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(&run, nil)

			dbClient.MockApplies.On("GetApply", mock.Anything, mock.Anything).Return(&apply, nil)
			dbClient.MockApplies.On("UpdateApply", mock.Anything, mock.Anything).Return(&apply, nil)
			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(nil, nil)
			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, run.WorkspaceID).Return(ws, nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			mockModuleService := moduleregistry.NewMockService(t)
			mockModuleResolver := NewMockModuleResolver(t)
			ruleEnforcer := rules.NewMockRuleEnforcer(t)

			for _, mi := range test.managedIdentities {
				miCopy := mi
				ruleEnforcer.On("EnforceRules", mock.Anything, &miCopy, mock.Anything).Return(test.enforceRulesResponse)
			}

			logger, _ := logger.NewForTest()
			service := newService(
				logger,
				dbClient.Client,
				nil,
				nil,
				nil,
				nil,
				&mockActivityEvents,
				mockModuleService,
				mockModuleResolver,
				state.NewRunStateManager(dbClient.Client, logger),
				ruleEnforcer,
			)

			_, err := service.ApplyRun(ctx, run.Metadata.ID, nil)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
