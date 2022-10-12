package run

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	var duration int32 = 720

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
		WorkspaceID:            ws.Metadata.ID,
		ConfigurationVersionID: &configurationVersionID,
		Status:                 models.RunPending,
	}

	mockAuthorizer := auth.MockAuthorizer{}
	mockAuthorizer.Test(t)

	mockAuthorizer.On("RequireAccessToWorkspace", mock.Anything, ws.Metadata.ID, models.DeployerRole).Return(nil)

	// Needed to move creation of userCaller and serviceAccountCaller inside the loop.

	// Test cases
	tests := []struct {
		name              string
		caller            string
		expectErrorCode   string
		teams             []models.Team
		managedIdentities []models.ManagedIdentity
		rules             []models.ManagedIdentityAccessRule
	}{
		{
			name: "user is forbidden to create run because managed identity access rule doesn't allow it",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobPlanType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{},
				},
			},
			caller:          "user",
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "user is allowed to create run because username is in managed identity access rule",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobPlanType,
					AllowedUserIDs:           []string{"123"},
					AllowedServiceAccountIDs: []string{},
				},
			},
			caller: "user",
		},
		{
			name: "user is allowed to create run because user is team member and team is in managed identity access rule",
			teams: []models.Team{
				{
					Metadata: models.ResourceMetadata{
						ID: "42",
					},
				},
			},
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobPlanType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{},
					AllowedTeamIDs:           []string{"42"},
				},
			},
			caller: "user",
		},
		{
			name: "user is prohibited from creating a run because the managed identity access rule requires a team the user is not a member of",
			teams: []models.Team{
				{
					Metadata: models.ResourceMetadata{
						ID: "42",
					},
				},
			},
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobPlanType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{},
					AllowedTeamIDs:           []string{"789"},
				},
			},
			caller:          "user",
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "user is allowed to create run because managed identity doesn't have any access rules",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules:  []models.ManagedIdentityAccessRule{},
			caller: "user",
		},
		{
			name: "service account is forbidden to create run because managed identity access rule doesn't allow it",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobPlanType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{},
				},
			},
			caller:          "serviceAccount",
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "service account is allowed to create run because service account is in managed identity access rule",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobPlanType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{"sa1"},
				},
			},
			caller: "serviceAccount",
		},
		{
			name: "service account is allowed to create run because managed identity doesn't have any access rules",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules:  []models.ManagedIdentityAccessRule{},
			caller: "serviceAccount",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			// Select userCaller or serviceAccountCaller.
			var testCaller auth.Caller
			switch test.caller {
			case "user":
				testCaller = auth.NewUserCaller(
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

			case "serviceAccount":
				testCaller = auth.NewServiceAccountCaller(
					"sa1",
					"groupA/sa1",
					&mockAuthorizer,
				)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).Return(test.managedIdentities, nil)

			ruleMap := map[string][]models.ManagedIdentityAccessRule{}

			for _, rule := range test.rules {
				if _, ok := ruleMap[rule.ManagedIdentityID]; !ok {
					ruleMap[rule.ManagedIdentityID] = []models.ManagedIdentityAccessRule{}
				}
				ruleMap[rule.ManagedIdentityID] = append(ruleMap[rule.ManagedIdentityID], rule)
			}

			for _, managedIdentity := range test.managedIdentities {
				dbClient.MockManagedIdentities.On("GetManagedIdentityAccessRules", mock.Anything, managedIdentity.Metadata.ID).
					Return(ruleMap[managedIdentity.Metadata.ID], nil)
			}

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

			dbClient.MockTeams.On("GetTeams", mock.Anything, mock.Anything).
				Return(&db.TeamsResult{Teams: test.teams}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient.Client, &mockArtifactStore, nil, nil, nil, nil)

			_, err := service.CreateRun(auth.WithCaller(ctx, testCaller), &CreateRunInput{
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

	mockAuthorizer := auth.MockAuthorizer{}
	mockAuthorizer.Test(t)

	mockAuthorizer.On("RequireAccessToWorkspace", mock.Anything, ws.Metadata.ID, models.DeployerRole).Return(nil)

	// Needed to move creation of userCaller and serviceAccountCaller inside the loop.

	// Test cases
	tests := []struct {
		name              string
		caller            string
		expectErrorCode   string
		managedIdentities []models.ManagedIdentity
		rules             []models.ManagedIdentityAccessRule
	}{
		{
			name: "user is forbidden to apply run because managed identity access rule doesn't allow it",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobApplyType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{},
				},
			},
			caller:          "user",
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "user is allowed to apply run because username is in managed identity access rule",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobApplyType,
					AllowedUserIDs:           []string{"123"},
					AllowedServiceAccountIDs: []string{},
				},
			},
			caller: "user",
		},
		{
			name: "user is allowed to apply run because managed identity doesn't have any access rules",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules:  []models.ManagedIdentityAccessRule{},
			caller: "user",
		},
		{
			name: "service account is forbidden to apply run because managed identity access rule doesn't allow it",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobApplyType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{"sa2"},
				},
			},
			caller:          "serviceAccount",
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "service account is allowed to apply run because service account is in managed identity access rule",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules: []models.ManagedIdentityAccessRule{
				{
					ManagedIdentityID:        "1",
					RunStage:                 models.JobApplyType,
					AllowedUserIDs:           []string{},
					AllowedServiceAccountIDs: []string{"sa1"},
				},
			},
			caller: "serviceAccount",
		},
		{
			name: "service account is allowed to apply run because managed identity doesn't have any access rules",
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
					ResourcePath: "groupA/1",
				},
			},
			rules:  []models.ManagedIdentityAccessRule{},
			caller: "serviceAccount",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			dbClient := buildDBClientWithMocks(t)

			// Select userCaller or serviceAccountCaller.
			var testCaller auth.Caller
			switch test.caller {
			case "user":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: "123",
						},
						Admin:    false,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient.Client,
				)
			case "serviceAccount":
				testCaller = auth.NewServiceAccountCaller(
					"sa1",
					"groupA/sa1",
					&mockAuthorizer,
				)
			}

			ctx, cancel := context.WithCancel(auth.WithCaller(context.Background(), testCaller))
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).Return(test.managedIdentities, nil)

			ruleMap := map[string][]models.ManagedIdentityAccessRule{}

			for _, rule := range test.rules {
				if _, ok := ruleMap[rule.ManagedIdentityID]; !ok {
					ruleMap[rule.ManagedIdentityID] = []models.ManagedIdentityAccessRule{}
				}
				ruleMap[rule.ManagedIdentityID] = append(ruleMap[rule.ManagedIdentityID], rule)
			}

			for _, managedIdentity := range test.managedIdentities {
				dbClient.MockManagedIdentities.On("GetManagedIdentityAccessRules", mock.Anything, managedIdentity.Metadata.ID).
					Return(ruleMap[managedIdentity.Metadata.ID], nil)
			}

			apply.Status = models.ApplyCreated // to avoid tripping the state transition checks in UpdateApply, etc.

			dbClient.MockRuns.On("GetRun", mock.Anything, run.Metadata.ID).Return(&run, nil)
			dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(&run, nil)

			dbClient.MockApplies.On("GetApply", mock.Anything, mock.Anything).Return(&apply, nil)
			dbClient.MockApplies.On("UpdateApply", mock.Anything, mock.Anything).Return(&apply, nil)
			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(nil, nil)
			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, run.WorkspaceID).Return(ws, nil)

			dbClient.MockTeams.On("GetTeams", mock.Anything, mock.Anything).
				Return(&db.TeamsResult{Teams: []models.Team{}}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient.Client, nil, nil, nil, nil, nil)

			_, err := service.ApplyRun(ctx, run.Metadata.ID, nil)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
