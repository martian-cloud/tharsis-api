package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type mockDBClient struct {
	*db.Client
	MockTransactions          *db.MockTransactions
	MockManagedIdentities     *db.MockManagedIdentities
	MockWorkspaces            *db.MockWorkspaces
	MockWorkspaceAssessments  *db.MockWorkspaceAssessments
	MockVariables             *db.MockVariables
	MockVariableVersions      *db.MockVariableVersions
	MockRuns                  *db.MockRuns
	MockConfigurationVersions *db.MockConfigurationVersions
	MockApplies               *db.MockApplies
	MockPlans                 *db.MockPlans
	MockJobs                  *db.MockJobs
	MockTeams                 *db.MockTeams
	MockTeamMembers           *db.MockTeamMembers
	MockLogStreams            *db.MockLogStreams
	MockResourceLimits        *db.MockResourceLimits
	MockGroups                *db.MockGroups
	MockStateVersions         *db.MockStateVersions
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)
	// The mocks are enabled by the above function.

	mockManagedIdentities := db.MockManagedIdentities{}
	mockManagedIdentities.Test(t)

	mockWorkspaces := db.MockWorkspaces{}
	mockWorkspaces.Test(t)

	mockWorkspaceAssessments := db.MockWorkspaceAssessments{}
	mockWorkspaceAssessments.Test(t)

	mockVariables := db.MockVariables{}
	mockVariables.Test(t)

	mockVariableVersions := db.MockVariableVersions{}
	mockVariableVersions.Test(t)

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

	mockLogStreams := db.MockLogStreams{}
	mockLogStreams.Test(t)

	mockResourceLimits := db.MockResourceLimits{}
	mockResourceLimits.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

	mockStateVersions := db.MockStateVersions{}
	mockStateVersions.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:          &mockTransactions,
			ManagedIdentities:     &mockManagedIdentities,
			Workspaces:            &mockWorkspaces,
			WorkspaceAssessments:  &mockWorkspaceAssessments,
			Variables:             &mockVariables,
			VariableVersions:      &mockVariableVersions,
			Runs:                  &mockRuns,
			ConfigurationVersions: &mockConfigurationVersions,
			Applies:               &mockApplies,
			Plans:                 &mockPlans,
			Jobs:                  &mockJobs,
			Teams:                 &mockTeams,
			TeamMembers:           &mockTeamMembers,
			LogStreams:            &mockLogStreams,
			ResourceLimits:        &mockResourceLimits,
			Groups:                &mockGroups,
			StateVersions:         &mockStateVersions,
		},
		MockTransactions:          &mockTransactions,
		MockManagedIdentities:     &mockManagedIdentities,
		MockWorkspaces:            &mockWorkspaces,
		MockWorkspaceAssessments:  &mockWorkspaceAssessments,
		MockVariables:             &mockVariables,
		MockVariableVersions:      &mockVariableVersions,
		MockRuns:                  &mockRuns,
		MockConfigurationVersions: &mockConfigurationVersions,
		MockApplies:               &mockApplies,
		MockPlans:                 &mockPlans,
		MockJobs:                  &mockJobs,
		MockTeams:                 &mockTeams,
		MockTeamMembers:           &mockTeamMembers,
		MockLogStreams:            &mockLogStreams,
		MockResourceLimits:        &mockResourceLimits,
		MockGroups:                &mockGroups,
		MockStateVersions:         &mockStateVersions,
	}
}

func TestGetRunVariables(t *testing.T) {
	ctx := context.Background()

	runID := "run1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: "ws1",
	}

	runVariables := []Variable{
		{
			Key:       "var1",
			Value:     ptr.String("value1"),
			Category:  models.TerraformVariableCategory,
			Sensitive: false,
		},
		{
			Key:       "var2",
			Category:  models.EnvironmentVariableCategory,
			Sensitive: true,
			VersionID: ptr.String("1"),
		},
	}

	marshaledRunVariables, err := json.Marshal(runVariables)
	require.NoError(t, err)

	variableVersions := []models.VariableVersion{}
	variableVersionIDs := []string{}
	// Add variable version for each sensitive variable
	for i, v := range runVariables {
		if v.Sensitive {
			id := strconv.Itoa(i)
			variableVersionIDs = append(variableVersionIDs, id)
			variableVersions = append(variableVersions, models.VariableVersion{
				Metadata:   models.ResourceMetadata{ID: id},
				Key:        v.Key,
				SecretData: []byte(fmt.Sprintf("%s-encrypted", v.Key)),
			})
		}
	}

	tests := []struct {
		name                            string
		includeSensitiveValues          bool
		expectedVariables               []Variable
		hasViewVariableValuePermissions bool
		authError                       error
		expectedErrorCode               errors.CodeType
	}{
		{
			name:                            "include sensitive values for caller with view variable value permission",
			includeSensitiveValues:          true,
			hasViewVariableValuePermissions: true,
			expectedVariables: []Variable{
				{
					Key:       "var1",
					Value:     ptr.String("value1"),
					Category:  models.TerraformVariableCategory,
					Sensitive: false,
				},
				{
					Key:       "var2",
					Value:     ptr.String("var2-plaintext"),
					Category:  models.EnvironmentVariableCategory,
					Sensitive: true,
					VersionID: ptr.String("1"),
				},
			},
		},
		{
			name:                            "don't include sensitive values for caller with view variable value permission",
			includeSensitiveValues:          false,
			hasViewVariableValuePermissions: true,
			expectedVariables: []Variable{
				{
					Key:       "var1",
					Value:     ptr.String("value1"),
					Category:  models.TerraformVariableCategory,
					Sensitive: false,
				},
				{
					Key:       "var2",
					Category:  models.EnvironmentVariableCategory,
					Sensitive: true,
					VersionID: ptr.String("1"),
				},
			},
		},
		{
			name:                            "don't include any values for caller without view variable value permission",
			includeSensitiveValues:          false,
			hasViewVariableValuePermissions: false,
			expectedVariables: []Variable{
				{
					Key:       "var1",
					Category:  models.TerraformVariableCategory,
					Sensitive: false,
				},
				{
					Key:       "var2",
					Category:  models.EnvironmentVariableCategory,
					Sensitive: true,
					VersionID: ptr.String("1"),
				},
			},
		},
		{
			name:                            "return error if caller without view variable value permission requests sensitive values",
			includeSensitiveValues:          true,
			hasViewVariableValuePermissions: false,
			expectedErrorCode:               errors.EForbidden,
		},
		{
			name:                            "return error if caller doesn't have view variable permission",
			hasViewVariableValuePermissions: false,
			authError:                       errors.New("no permission", errors.WithErrorCode(errors.EForbidden)),
			expectedErrorCode:               errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockDBClient := buildDBClientWithMocks(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)
			mockSecretManager := secret.NewMockManager(t)
			mockCaller := auth.NewMockCaller(t)

			service := &service{
				dbClient:      mockDBClient.Client,
				artifactStore: mockArtifactStore,
				secretManager: mockSecretManager,
			}

			if test.hasViewVariableValuePermissions {
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewVariableValuePermission, mock.Anything).Return(nil)
			} else {
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewVariableValuePermission, mock.Anything).Return(errors.New("no permission", errors.WithErrorCode(errors.EForbidden)))
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewVariablePermission, mock.Anything).Return(test.authError)
			}

			mockDBClient.MockRuns.On("GetRun", mock.Anything, runID).Return(run, nil)

			mockArtifactStore.On("GetRunVariables", mock.Anything, run).Return(io.NopCloser(bytes.NewReader(marshaledRunVariables)), nil).Maybe()

			if test.includeSensitiveValues && test.expectedErrorCode == "" {
				mockDBClient.MockVariableVersions.On("GetVariableVersions", mock.Anything, &db.GetVariableVersionsInput{
					Filter: &db.VariableVersionFilter{
						VariableVersionIDs: variableVersionIDs,
					},
				}).Return(&db.VariableVersionResult{
					VariableVersions: variableVersions,
				}, nil)

				for _, v := range variableVersions {
					mockSecretManager.On("Get", mock.Anything, v.Key, v.SecretData).Return(fmt.Sprintf("%s-plaintext", v.Key), nil)
				}
			}

			vars, err := service.GetRunVariables(auth.WithCaller(ctx, mockCaller), runID, test.includeSensitiveValues)
			if test.expectedErrorCode != "" {
				require.Error(t, err)
				require.Equal(t, test.expectedErrorCode, errors.ErrorCode(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedVariables, vars)
			}
		})
	}
}

func TestCreateRunWithSensitiveVariables(t *testing.T) {
	configurationVersionID := "cv1"

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:       "groupA/ws1",
		MaxJobDuration: ptr.Int32(60),
	}
	groupName := "groupA"

	run := models.Run{
		Metadata: models.ResourceMetadata{
			ID: "run1",
		},
		WorkspaceID:            ws.Metadata.ID,
		ConfigurationVersionID: &configurationVersionID,
		Status:                 models.RunPending,
	}

	dbClient := buildDBClientWithMocks(t)

	mockSecretManager := secret.NewMockManager(t)

	mockCaller := auth.NewMockCaller(t)
	mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(nil)
	mockCaller.On("GetSubject").Return("mock-caller").Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
	dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

	dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).Return([]models.ManagedIdentity{}, nil)

	dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil)

	sortBy := db.VariableSortableFieldNamespacePathDesc
	dbClient.MockVariables.On("GetVariables", mock.Anything, &db.GetVariablesInput{
		Filter: &db.VariableFilter{
			NamespacePaths: ws.ExpandPath(),
		},
		Sort: &sortBy,
	}).Return(&db.VariableResult{
		Variables: []models.Variable{
			{
				Key:             "v1",
				Value:           ptr.String("v1-value"),
				Category:        models.TerraformVariableCategory,
				NamespacePath:   ws.FullPath,
				Sensitive:       true,
				SecretData:      []byte("v1-encrypted"),
				LatestVersionID: "1",
			},
			{
				Key:             "v2",
				Value:           ptr.String("v2-value"),
				Category:        models.TerraformVariableCategory,
				NamespacePath:   ws.FullPath,
				LatestVersionID: "2",
			},
			{
				Key:             "v3",
				Value:           ptr.String("v3-value"),
				Category:        models.EnvironmentVariableCategory,
				NamespacePath:   ws.FullPath,
				LatestVersionID: "3",
			},
		},
	}, nil)

	// Mock for secret manager plugin since the v1 variable is sensitive
	mockSecretManager.On("Get", mock.Anything, "v1", []byte("v1-encrypted")).Return("v1-value", nil)

	dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).
		Return(func(_ context.Context, run *models.Run) (*models.Run, error) {
			run.Metadata.CreationTimestamp = ptr.Time(time.Now().UTC())
			return run, nil
		})
	dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
		Return(&db.RunsResult{
			PageInfo: &pagination.PageInfo{
				TotalCount: 1,
			},
		}, nil)
	dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(&run, nil)

	dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
		Return(&models.ResourceLimit{Value: 100}, nil)

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
	dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(&models.Job{
		Metadata: models.ResourceMetadata{
			ID: "job1",
		},
		WorkspaceID: ws.Metadata.ID,
		RunID:       run.Metadata.ID,
	}, nil)

	dbClient.MockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
		Groups: []models.Group{
			{
				Metadata: models.ResourceMetadata{
					ID: groupName,
				},
				Name: groupName,
			},
		},
	}, nil)

	dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

	mockArtifactStore := workspace.MockArtifactStore{}
	mockArtifactStore.Test(t)

	readerMatcher := mock.MatchedBy(func(r io.Reader) bool {
		body, _ := io.ReadAll(r)

		var decodedVariables []Variable
		if err := json.Unmarshal(body, &decodedVariables); err != nil {
			t.Fatal(err)
		}

		// Verify that value is nill for sensitive variables
		for _, v := range decodedVariables {
			if v.Sensitive && v.Value != nil {
				return false
			}
			if !v.Sensitive && v.Value == nil {
				return false
			}
		}

		return true
	})
	mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, readerMatcher).Return(nil)

	mockActivityEvents := activityevent.MockService{}
	mockActivityEvents.Test(t)

	mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

	mockModuleService := moduleregistry.NewMockService(t)
	mockModuleResolver := NewMockModuleResolver(t)
	ruleEnforcer := rules.NewMockRuleEnforcer(t)

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
		limits.NewLimitChecker(dbClient.Client),
		nil,
		mockSecretManager,
	)

	_, err := service.CreateRun(auth.WithCaller(ctx, mockCaller), &CreateRunInput{
		WorkspaceID:            ws.Metadata.ID,
		ConfigurationVersionID: &configurationVersionID,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateRunWithManagedIdentityAccessRules(t *testing.T) {
	configurationVersionID := "cv1"
	currentTime := time.Now().UTC()

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:       "groupA/ws1",
		MaxJobDuration: ptr.Int32(60),
	}
	groupName := "groupA"

	run := models.Run{
		Metadata: models.ResourceMetadata{
			ID: "run1",
		},
		WorkspaceID:            ws.Metadata.ID,
		ConfigurationVersionID: &configurationVersionID,
		Status:                 models.RunPending,
	}

	injectJob := models.Job{
		Metadata: models.ResourceMetadata{
			ID: "job1",
		},
		WorkspaceID: ws.Metadata.ID,
		RunID:       run.Metadata.ID,
	}

	// Test cases
	tests := []struct {
		name                   string
		injectJob              *models.Job
		expectErrorCode        errors.CodeType
		enforceRulesResponse   error
		managedIdentities      []models.ManagedIdentity
		limit                  int
		injectRunsPerWorkspace int32
	}{
		{
			name:      "run is created because all managed identity rules are satisfied",
			injectJob: &injectJob,
			managedIdentities: []models.ManagedIdentity{
				{
					Metadata: models.ResourceMetadata{
						ID: "1",
					},
				},
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name:                   "run is created because there are no managed identities",
			injectJob:              &injectJob,
			managedIdentities:      []models.ManagedIdentity{},
			limit:                  4,
			injectRunsPerWorkspace: 4,
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
			enforceRulesResponse: errors.New("rule not satisfied", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:      errors.EForbidden,
		},
		{
			name:                   "resource limit exceeded",
			injectJob:              &injectJob,
			managedIdentities:      []models.ManagedIdentity{},
			limit:                  4,
			injectRunsPerWorkspace: 5,
			expectErrorCode:        errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(nil)
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

			dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).
				Return(func(ctx context.Context, run *models.Run) (*models.Run, error) {
					_ = ctx

					if run != nil {
						// Must inject creation timestamp so limit check won't hit a nil pointer.
						runWithTimestamp := *run
						runWithTimestamp.Metadata.CreationTimestamp = &currentTime
						return &runWithTimestamp, nil
					}
					return nil, nil
				})
			dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
				Return(&db.RunsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: test.injectRunsPerWorkspace,
					},
				}, nil)
			dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(&run, nil)

			dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: test.limit}, nil)

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
			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(test.injectJob, nil)

			dbClient.MockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
				Groups: []models.Group{
					{
						Metadata: models.ResourceMetadata{
							ID: groupName,
						},
						Name: groupName,
					},
				},
			}, nil)

			dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

			mockArtifactStore := workspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockSecretManager := secret.NewMockManager(t)

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
				limits.NewLimitChecker(dbClient.Client),
				nil,
				mockSecretManager,
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
	currentTime := time.Now().UTC()

	groupName := "groupA"

	injectJob := models.Job{
		Metadata: models.ResourceMetadata{
			ID: "job1",
		},
	}

	// Test cases
	type testCase struct {
		workspace              *models.Workspace
		runInput               *CreateRunInput
		injectJob              *models.Job
		name                   string
		expectErrorCode        errors.CodeType
		limit                  int
		injectRunsPerWorkspace int32
	}

	/*
		Test case template.
		name            string
		workspace       *models.Workspace
		runInput        *CreateRunInput
		expectErrorCode errors.CodeType
		injectJob       *models.Job
	*/

	tests := []testCase{

		{
			name: "non-destroy plan is allowed independent of PreventDestroyPlan",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-1",
				},
				FullPath:           groupName + "/ws1",
				MaxJobDuration:     &duration,
				PreventDestroyPlan: false,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-1",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              false,
			},
			injectJob:              &injectJob,
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},

		{
			name: "destroy plan is allowed, because PreventDestroyPlan is false",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-2",
				},
				FullPath:           groupName + "/ws1",
				MaxJobDuration:     &duration,
				PreventDestroyPlan: false,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-2",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              true,
			},
			injectJob:              &injectJob,
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},

		{
			name: "non-destroy plan is allowed even when PreventDestroyPlan is true",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-3",
				},
				FullPath:           groupName + "/ws1",
				MaxJobDuration:     &duration,
				PreventDestroyPlan: true,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-3",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              false,
			},
			injectJob:              &injectJob,
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},

		{
			name: "destroy plan is NOT allowed, because PreventDestroyPlan is true",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-4",
				},
				FullPath:           groupName + "/ws1",
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

		{
			name: "exceeds resource limit",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "test-workspace-metadata-id-1",
				},
				FullPath:           groupName + "/ws1",
				MaxJobDuration:     &duration,
				PreventDestroyPlan: false,
			},
			runInput: &CreateRunInput{
				WorkspaceID:            "test-workspace-metadata-id-1",
				ConfigurationVersionID: &configurationVersionID,
				IsDestroy:              false,
			},
			injectJob:              &injectJob,
			limit:                  4,
			injectRunsPerWorkspace: 5,
			expectErrorCode:        errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace",
				mock.Anything, test.workspace.Metadata.ID).Return([]models.ManagedIdentity{}, nil)

			dbClient.MockWorkspaces.On("GetWorkspaceByID",
				mock.Anything, test.workspace.Metadata.ID).Return(test.workspace, nil)

			dbClient.MockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.VariableResult{
				Variables: []models.Variable{},
			}, nil)

			dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).
				Return(func(ctx context.Context, run *models.Run) (*models.Run, error) {
					_ = ctx

					if run != nil {
						// Must inject creation timestamp so limit check won't hit a nil pointer.
						runWithTimestamp := *run
						runWithTimestamp.Metadata.CreationTimestamp = &currentTime
						return &runWithTimestamp, nil
					}
					return nil, nil
				})
			dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
				Return(&db.RunsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: test.injectRunsPerWorkspace,
					},
				}, nil)

			dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: test.limit}, nil)

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

			dbClient.MockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
				Groups: []models.Group{
					{
						Metadata: models.ResourceMetadata{
							ID: groupName,
						},
						Name: groupName,
					},
				},
			}, nil)

			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(test.injectJob, nil)

			dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

			mockArtifactStore := workspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockSecretManager := secret.NewMockManager(t)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			logger, _ := logger.NewForTest()

			service := NewService(
				logger,
				dbClient.Client,
				&mockArtifactStore,
				nil,
				nil,
				nil,
				&mockActivityEvents,
				nil,
				nil,
				nil,
				limits.NewLimitChecker(dbClient.Client),
				mockSecretManager,
			)

			_, err := service.CreateRun(auth.WithCaller(ctx, mockCaller), test.runInput)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateRunWithSpeculativeOption(t *testing.T) {
	configurationVersionID := "configuration-version-id-1"
	moduleSource := "module-source-1"
	moduleVersion := "1.2.3"
	createdBySubject := "mock-caller"
	groupName := "groupA"
	planID := "plan1"
	applyID := "apply1"
	isTrue := true
	isFalse := false
	currentTime := time.Now().UTC()

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:       "groupA/ws1",
		MaxJobDuration: ptr.Int32(60),
	}

	// Test cases
	tests := []struct {
		input                   *CreateRunInput
		expectCreateRun         *models.Run
		name                    string
		expectErrorCode         errors.CodeType
		injectConfigVersionSpec bool
		limit                   int
		injectRunsPerWorkspace  int32
	}{
		{
			name: "module source, speculative not specified; expect false",
			input: &CreateRunInput{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				Speculative:   nil,
			},
			expectCreateRun: &models.Run{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				CreatedBy:     createdBySubject,
				PlanID:        planID,
				ApplyID:       applyID,
				Status:        models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "module source, speculative specified true; expect true",
			input: &CreateRunInput{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				Speculative:   &isTrue,
			},
			expectCreateRun: &models.Run{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				CreatedBy:     createdBySubject,
				PlanID:        planID,
				ApplyID:       "",
				Status:        models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "module source, speculative specified false; expect false",
			input: &CreateRunInput{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				Speculative:   &isFalse,
			},
			expectCreateRun: &models.Run{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				CreatedBy:     createdBySubject,
				PlanID:        planID,
				ApplyID:       applyID,
				Status:        models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "configuration version spec=false, options spec=nil; expect false",
			input: &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Speculative:            nil,
			},
			injectConfigVersionSpec: false,
			expectCreateRun: &models.Run{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				CreatedBy:              createdBySubject,
				PlanID:                 planID,
				ApplyID:                applyID,
				Status:                 models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "configuration version spec=false, options spec=true; expect true",
			input: &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Speculative:            &isTrue,
			},
			injectConfigVersionSpec: false,
			expectCreateRun: &models.Run{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				CreatedBy:              createdBySubject,
				PlanID:                 planID,
				ApplyID:                "",
				Status:                 models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "configuration version spec=false, options spec=false; expect false",
			input: &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Speculative:            &isFalse,
			},
			injectConfigVersionSpec: false,
			expectCreateRun: &models.Run{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				CreatedBy:              createdBySubject,
				PlanID:                 planID,
				ApplyID:                applyID,
				Status:                 models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "configuration version spec=true, options spec=nil; expect true",
			input: &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Speculative:            nil,
			},
			injectConfigVersionSpec: true,
			expectCreateRun: &models.Run{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				CreatedBy:              createdBySubject,
				PlanID:                 planID,
				ApplyID:                "",
				Status:                 models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "configuration version spec=true, options spec=true; expect true",
			input: &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Speculative:            &isTrue,
			},
			injectConfigVersionSpec: true,
			expectCreateRun: &models.Run{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				CreatedBy:              createdBySubject,
				PlanID:                 planID,
				ApplyID:                "",
				Status:                 models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 4,
		},
		{
			name: "configuration version spec=true, options spec=false; expect error",
			input: &CreateRunInput{
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
				Speculative:            &isFalse,
			},
			injectConfigVersionSpec: true,
			expectErrorCode:         errors.EInvalid,
		},
		{
			name: "exceeds limit",
			input: &CreateRunInput{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				Speculative:   nil,
			},
			expectCreateRun: &models.Run{
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				CreatedBy:     createdBySubject,
				PlanID:        planID,
				ApplyID:       applyID,
				Status:        models.RunPlanQueued,
			},
			limit:                  4,
			injectRunsPerWorkspace: 5,
			expectErrorCode:        errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return(createdBySubject).Maybe()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).
				Return([]models.ManagedIdentity{}, nil)

			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil)

			dbClient.MockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.VariableResult{
				Variables: []models.Variable{},
			}, nil)

			dbClient.MockRuns.On("CreateRun", mock.Anything, test.expectCreateRun).
				Return(func(ctx context.Context, run *models.Run) (*models.Run, error) {
					_ = ctx

					if run != nil {
						// Must inject creation timestamp so limit check won't hit a nil pointer.
						runWithTimestamp := *run
						runWithTimestamp.Metadata.CreationTimestamp = &currentTime
						return &runWithTimestamp, nil
					}
					return nil, nil
				})
			dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
				Return(&db.RunsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: test.injectRunsPerWorkspace,
					},
				}, nil)
			dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(test.expectCreateRun, nil)

			dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: test.limit}, nil)

			dbClient.MockConfigurationVersions.On("GetConfigurationVersion", mock.Anything, configurationVersionID).
				Return(&models.ConfigurationVersion{
					Speculative: test.injectConfigVersionSpec,
				}, nil)

			dbClient.MockPlans.On("CreatePlan", mock.Anything, mock.Anything).Return(&models.Plan{
				Metadata: models.ResourceMetadata{
					ID: planID,
				},
			}, nil)

			dbClient.MockApplies.On("CreateApply", mock.Anything, mock.Anything).Return(&models.Apply{
				Metadata: models.ResourceMetadata{
					ID: applyID,
				},
			}, nil)

			dbClient.MockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
				Groups: []models.Group{
					{
						Metadata: models.ResourceMetadata{
							ID: groupName,
						},
						Name: groupName,
					},
				},
			}, nil)

			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).
				Return(func(_ context.Context, _ *models.Job) (*models.Job, error) {
					return &models.Job{
						Metadata: models.ResourceMetadata{
							ID: "job1",
						},
						WorkspaceID: ws.Metadata.ID,
					}, nil
				}, nil)

			dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

			mockArtifactStore := workspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			mockModuleService := moduleregistry.NewMockService(t)
			mockModuleResolver := NewMockModuleResolver(t)

			mockModuleResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything).
				Return(&ModuleRegistrySource{}, nil).Maybe()

			mockModuleResolver.On("ResolveModuleVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(moduleVersion, nil).Maybe()

			mockSecretManager := secret.NewMockManager(t)

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
				nil,
				limits.NewLimitChecker(dbClient.Client),
				nil,
				mockSecretManager,
			)

			_, err := service.CreateRun(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateRunWithJobTags(t *testing.T) {
	configVersionID := "cv1"
	createdBySubject := "mock-caller"
	planID := "plan1"
	applyID := "apply1"
	currentTime := time.Now().UTC()
	runnerTags := []string{
		"tag1",
		"tag2",
	}

	tests := []struct {
		name             string
		workspace        *models.Workspace
		parentGroup      *models.Group
		grandparentGroup *models.Group
		createRunInputs  *CreateRunInput
		expectTags       []string
		// No errors expected for this test function.
	}{
		{
			name: "tags set by workspace",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "ws1",
				},
				FullPath:       "groupB/ws1",
				MaxJobDuration: ptr.Int32(60),
				RunnerTags:     runnerTags,
			},
			parentGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: "groupB",
				},
				Name:     "groupB",
				FullPath: "groupB",
			},
			createRunInputs: &CreateRunInput{
				WorkspaceID:            "ws1",
				ConfigurationVersionID: &configVersionID,
			},
			expectTags: runnerTags,
		},
		{
			name: "tags set by parent group",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "ws1",
				},
				FullPath:       "groupB/ws1",
				MaxJobDuration: ptr.Int32(60),
				GroupID:        "groupB",
			},
			parentGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: "groupB",
				},
				Name:       "groupB",
				FullPath:   "groupB",
				RunnerTags: runnerTags,
			},
			createRunInputs: &CreateRunInput{
				WorkspaceID:            "ws1",
				ConfigurationVersionID: &configVersionID,
			},
			expectTags: runnerTags,
		},
		{
			name: "tags set by grandparent group",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "ws1",
				},
				FullPath:       "groupA/groupB/ws1",
				MaxJobDuration: ptr.Int32(60),
				GroupID:        "groupB",
			},
			parentGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: "groupB",
				},
				Name:     "groupB",
				FullPath: "groupA/groupB",
				ParentID: "groupA",
			},
			grandparentGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: "groupA",
				},
				Name:       "groupA",
				FullPath:   "groupA",
				RunnerTags: runnerTags,
			},
			createRunInputs: &CreateRunInput{
				WorkspaceID:            "ws1",
				ConfigurationVersionID: &configVersionID,
			},
			expectTags: runnerTags,
		},
		{
			name: "tags not set by any group",
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "ws1",
				},
				FullPath:       "groupA/groupB/ws1",
				MaxJobDuration: ptr.Int32(60),
				GroupID:        "groupB",
			},
			parentGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: "groupB",
				},
				Name:     "groupB",
				FullPath: "groupA/groupB",
				ParentID: "groupA",
			},
			grandparentGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID: "groupA",
				},
				Name:     "groupA",
				FullPath: "groupA",
			},
			createRunInputs: &CreateRunInput{
				WorkspaceID:            "ws1",
				ConfigurationVersionID: &configVersionID,
			},
			expectTags: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return(createdBySubject).Maybe()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.workspace.Metadata.ID).
				Return(test.workspace, nil)

			dbClient.MockVariables.On("GetVariables", mock.Anything, mock.Anything).Return(&db.VariableResult{
				Variables: []models.Variable{},
			}, nil)

			dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, test.workspace.Metadata.ID).
				Return([]models.ManagedIdentity{}, nil)

			dbClient.MockPlans.On("CreatePlan", mock.Anything, mock.Anything).Return(&models.Plan{
				Metadata: models.ResourceMetadata{
					ID: planID,
				},
			}, nil)

			dbClient.MockConfigurationVersions.On("GetConfigurationVersion", mock.Anything, configVersionID).
				Return(&models.ConfigurationVersion{}, nil)

			dbClient.MockApplies.On("CreateApply", mock.Anything, mock.Anything).Return(&models.Apply{
				Metadata: models.ResourceMetadata{
					ID: applyID,
				},
			}, nil)

			dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).
				Return(func(ctx context.Context, run *models.Run) (*models.Run, error) {
					_ = ctx

					if run != nil {
						// Must inject creation timestamp so limit check won't hit a nil pointer.
						runWithTimestamp := *run
						runWithTimestamp.Metadata.CreationTimestamp = &currentTime
						return &runWithTimestamp, nil
					}
					return nil, nil
				})
			dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
				Return(&db.RunsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: 1,
					},
				}, nil)

			dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: 1}, nil)

			dbClient.MockJobs.On("CreateJob", mock.Anything,
				mock.MatchedBy(func(input *models.Job) bool {
					return assert.ElementsMatch(t, input.Tags, test.expectTags) && assert.NotNil(t, input.Tags, "Should never be nil")
				})).
				Return(&models.Job{
					Metadata: models.ResourceMetadata{
						ID: "job1",
					},
					WorkspaceID: test.workspace.Metadata.ID,
				}, nil)

			dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

			dbClient.MockGroups.On("GetGroups", mock.Anything, mock.Anything).
				Return(func(_ context.Context, input *db.GetGroupsInput) (*db.GroupsResult, error) {
					groups := []models.Group{}
					if test.parentGroup != nil {
						groups = append(groups, *test.parentGroup)
					}
					if test.grandparentGroup != nil {
						groups = append(groups, *test.grandparentGroup)
					}

					// Make sure we're returning the correct number of groups.
					assert.Equal(t, len(input.Filter.GroupPaths), len(groups))

					return &db.GroupsResult{
						Groups: groups,
					}, nil
				})

			mockArtifactStore := workspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadRunVariables", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(&models.ActivityEvent{}, nil)

			mockModuleService := moduleregistry.NewMockService(t)
			mockModuleResolver := NewMockModuleResolver(t)

			mockModuleResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything).
				Return(&ModuleRegistrySource{}, nil).Maybe()

			mockSecretManager := secret.NewMockManager(t)

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
				nil,
				limits.NewLimitChecker(dbClient.Client),
				nil,
				mockSecretManager,
			)

			_, err := service.CreateRun(auth.WithCaller(ctx, mockCaller), test.createRunInputs)
			assert.Nil(t, err)

			// The expected job tags are checked by the inputs to CreateJob.
		})
	}
}

func TestApplyRunWithManagedIdentityAccessRules(t *testing.T) {
	var duration int32 = 1
	groupName := "groupA"
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

	injectJob := models.Job{
		Metadata: models.ResourceMetadata{
			ID: "job1",
		},
		WorkspaceID: ws.Metadata.ID,
		RunID:       run.Metadata.ID,
	}

	// Test cases
	tests := []struct {
		enforceRulesResponse error
		injectJob            *models.Job
		name                 string
		expectErrorCode      errors.CodeType
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
			injectJob: &injectJob,
		},
		{
			name:              "apply is created because there are no managed identities",
			managedIdentities: []models.ManagedIdentity{},
			injectJob:         &injectJob,
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
			enforceRulesResponse: errors.New("rule not satisfied", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:      errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dbClient := buildDBClientWithMocks(t)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(nil)
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

			dbClient.MockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
				Groups: []models.Group{
					{
						Metadata: models.ResourceMetadata{
							ID: groupName,
						},
						Name: groupName,
					},
				},
			}, nil)

			dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).Return(test.injectJob, nil)
			dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)
			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, run.WorkspaceID).Return(ws, nil)

			mockSecretManager := secret.NewMockManager(t)

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
				limits.NewLimitChecker(dbClient.Client),
				nil,
				mockSecretManager,
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

func TestCreateDestroyRunForWorkspace(t *testing.T) {
	testSubject := "tester"
	currentTime := time.Now()

	stateVersionID := "sv1"
	planID := "plan1"
	applyID := "apply1"

	moduleSource := "mymodule"
	moduleVersion := "1.0.0"
	configurationVersionID := "cv1"

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:              "groupA/ws1",
		CurrentStateVersionID: stateVersionID,
		MaxJobDuration:        ptr.Int32(60),
		RunnerTags:            []string{},
	}

	// Test cases
	tests := []struct {
		input           *CreateDestroyRunForWorkspaceInput
		stateVersion    *models.StateVersion
		currentRun      *models.Run
		expectCreateRun *models.Run
		name            string
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name: "create workspace destroy run for workspace with module applied",
			input: &CreateDestroyRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
				RunID:       ptr.String("run1"),
			},
			currentRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
			},
			expectCreateRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:                "run2",
					CreationTimestamp: &currentTime,
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				IsDestroy:     true,
			},
		},
		{
			name: "create workspace destroy run for workspace with configuration version",
			input: &CreateDestroyRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
				RunID:       ptr.String("run1"),
			},
			currentRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:            ws.Metadata.ID,
				ConfigurationVersionID: &configurationVersionID,
			},
			expectCreateRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:                "run2",
					CreationTimestamp: &currentTime,
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
				IsDestroy:     true,
			},
		},
		{
			name: "cannot create destroy run if state version was created manually",
			input: &CreateDestroyRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "expect authorization error",
			input: &CreateDestroyRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			authError:       errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup mocks
			dbClient := buildDBClientWithMocks(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)
			mockModuleService := moduleregistry.NewMockService(t)
			mockModuleResolver := NewMockModuleResolver(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return(testSubject).Maybe()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil).Maybe()
			dbClient.MockStateVersions.On("GetStateVersion", mock.Anything, stateVersionID).Return(test.stateVersion, nil).Maybe()

			if test.expectErrorCode == "" {

				dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

				dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).
					Return([]models.ManagedIdentity{}, nil)

				if test.stateVersion.RunID != nil {
					dbClient.MockRuns.On("GetRun", mock.Anything, *test.stateVersion.RunID).Return(test.currentRun, nil)
				}

				dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).
					Return(test.expectCreateRun, nil)
				dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
					Return(&db.RunsResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: 1,
						},
					}, nil)
				dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(test.expectCreateRun, nil)

				dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: 10}, nil)

				if test.currentRun.ConfigurationVersionID != nil {
					dbClient.MockConfigurationVersions.On("GetConfigurationVersion", mock.Anything, configurationVersionID).
						Return(&models.ConfigurationVersion{
							Speculative: false,
						}, nil)
				}

				dbClient.MockPlans.On("CreatePlan", mock.Anything, mock.Anything).Return(&models.Plan{
					Metadata: models.ResourceMetadata{
						ID: planID,
					},
				}, nil)

				dbClient.MockApplies.On("CreateApply", mock.Anything, mock.Anything).Return(&models.Apply{
					Metadata: models.ResourceMetadata{
						ID: applyID,
					},
				}, nil)

				dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).
					Return(func(_ context.Context, _ *models.Job) (*models.Job, error) {
						return &models.Job{
							Metadata: models.ResourceMetadata{
								ID: "job1",
							},
							WorkspaceID: ws.Metadata.ID,
						}, nil
					}, nil)

				dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

				data, err := json.Marshal([]Variable{
					{Key: "k1", Value: ptr.String("v1"), Category: models.TerraformVariableCategory},
				})
				require.NoError(t, err)

				mockArtifactStore.On("GetRunVariables", mock.Anything, test.currentRun).Return(io.NopCloser(bytes.NewReader(data)), nil)

				matcher := mock.MatchedBy(func(run *models.Run) bool {
					return run.Metadata.ID == test.expectCreateRun.Metadata.ID
				})
				mockArtifactStore.On("UploadRunVariables", mock.Anything, matcher, bytes.NewReader(data)).Return(nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockModuleResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything).
					Return(&ModuleRegistrySource{}, nil).Maybe()

				mockModuleResolver.On("ResolveModuleVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(moduleVersion, nil).Maybe()
			}

			// Create test fixture
			logger, _ := logger.NewForTest()
			testService := service{
				logger:          logger,
				dbClient:        dbClient.Client,
				artifactStore:   mockArtifactStore,
				activityService: mockActivityEvents,
				moduleService:   mockModuleService,
				moduleResolver:  mockModuleResolver,
				limitChecker:    limits.NewLimitChecker(dbClient.Client),
			}

			// Invoke test method
			run, err := testService.CreateDestroyRunForWorkspace(auth.WithCaller(ctx, mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			// Assertions
			require.NoError(t, err)

			assert.True(t, run.IsDestroy)
			assert.Equal(t, test.expectCreateRun.WorkspaceID, run.WorkspaceID)
			assert.Equal(t, test.expectCreateRun.ModuleSource, run.ModuleSource)
			assert.Equal(t, test.expectCreateRun.ModuleVersion, run.ModuleVersion)
			assert.Equal(t, test.expectCreateRun.ConfigurationVersionID, run.ConfigurationVersionID)
		})
	}
}

func TestGetStateVersionsByRunIDs(t *testing.T) {
	workspaceID := "ws1"

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		runIDs          []string
	}

	testCases := []testCase{
		{
			name:   "get state versions by run ids",
			runIDs: []string{"run1", "run2"},
		},
		{
			name: "no run ids",
		},
		{
			name:            "subject does not have permission to view run",
			runIDs:          []string{"run1"},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockStateVersions := db.NewMockStateVersions(t)

			runsCount := len(test.runIDs)

			mockRuns.On("GetRuns", mock.Anything, &db.GetRunsInput{
				Filter: &db.RunFilter{
					RunIDs: test.runIDs,
				},
			}).Return(func(_ context.Context, _ *db.GetRunsInput) (*db.RunsResult, error) {
				// Create runs
				runs := make([]models.Run, runsCount)
				for i := 0; i < runsCount; i++ {
					runs[i] = models.Run{
						Metadata: models.ResourceMetadata{
							ID: test.runIDs[i],
						},
						WorkspaceID: workspaceID,
					}
				}

				return &db.RunsResult{
					Runs: runs,
					PageInfo: &pagination.PageInfo{
						TotalCount: int32(runsCount),
					},
				}, nil
			})

			if runsCount > 0 {
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError).Times(runsCount)

				if test.authError == nil {
					mockStateVersions.On("GetStateVersions", mock.Anything, &db.GetStateVersionsInput{
						Filter: &db.StateVersionFilter{
							RunIDs: test.runIDs,
						},
					}).Return(func(_ context.Context, _ *db.GetStateVersionsInput) (*db.StateVersionsResult, error) {
						// Create state versions
						stateVersions := make([]models.StateVersion, runsCount)
						for i := 0; i < runsCount; i++ {
							stateVersions[i] = models.StateVersion{
								Metadata: models.ResourceMetadata{
									ID: fmt.Sprintf("sv%d", i),
								},
								WorkspaceID: workspaceID,
								RunID:       &test.runIDs[i],
							}
						}

						return &db.StateVersionsResult{
							StateVersions: stateVersions,
							PageInfo: &pagination.PageInfo{
								TotalCount: int32(runsCount),
							},
						}, nil
					})
				}
			}

			dbClient := &db.Client{
				Runs:          mockRuns,
				StateVersions: mockStateVersions,
			}

			service := &service{
				dbClient: dbClient,
			}

			result, err := service.GetStateVersionsByRunIDs(auth.WithCaller(ctx, mockCaller), test.runIDs)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, runsCount)
		})
	}
}

func TestGetRuns(t *testing.T) {
	workspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-1",
		},
	}

	group := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-1",
		},
	}

	userID := "userID"

	type testCase struct {
		authError       error
		input           *GetRunsInput
		name            string
		expectErrorCode errors.CodeType
		isAdmin         bool
	}

	testCases := []testCase{
		{
			name: "filter by workspace",
			input: &GetRunsInput{
				Workspace: workspace,
			},
		},
		{
			name: "filter by group",
			input: &GetRunsInput{
				Group: group,
			},
		},
		{
			name:    "admin user queries for all runs",
			input:   &GetRunsInput{},
			isAdmin: true,
		},
		{
			name: "caller does not have access to workspace",
			input: &GetRunsInput{
				Workspace: workspace,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "caller does not have access to group",
			input: &GetRunsInput{
				Group: group,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRuns := db.NewMockRuns(t)
			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			filter := &db.RunFilter{}

			switch {
			case test.input.Workspace != nil:
				filter.WorkspaceID = ptr.String(test.input.Workspace.Metadata.ID)
			case test.input.Group != nil:
				filter.GroupID = ptr.String(test.input.Group.Metadata.ID)
			default:
				if !test.isAdmin {
					filter.UserMemberID = &userID
				}
			}

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
			mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewRunPermission}, mock.Anything).Return(test.authError).Maybe()

			if test.expectErrorCode == "" {
				mockRuns.On("GetRuns", mock.Anything, &db.GetRunsInput{
					Sort:              test.input.Sort,
					PaginationOptions: test.input.PaginationOptions,
					Filter:            filter,
				}).Return(&db.RunsResult{}, nil)
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient: dbClient,
			}

			userCaller := auth.NewUserCaller(
				&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
					Admin: test.isAdmin,
				},
				mockAuthorizer,
				dbClient,
				mockMaintenanceMonitor,
			)

			runsResult, err := service.GetRuns(auth.WithCaller(ctx, userCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, errors.ErrorCode(err), test.expectErrorCode)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, runsResult)
		})
	}
}

func TestGetPlanDiff(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: workspaceID,
		PlanID:      "plan-1",
	}

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		expectedDiff    *plan.Diff
	}

	testCases := []testCase{
		{
			name:            "subject does not have permission to view run",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:         "get plan diff",
			expectedDiff: &plan.Diff{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRuns := db.NewMockRuns(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)

			mockRuns.On("GetRunByPlanID", mock.Anything, run.PlanID).Return(run, nil)

			mockCaller.On("RequirePermission", mock.Anything, permissions.ViewRunPermission, mock.Anything, mock.Anything).Return(test.authError)

			planDiffBuf, err := json.Marshal(test.expectedDiff)
			require.NoError(t, err)

			mockArtifactStore.On("GetPlanDiff", mock.Anything, run).Return(io.NopCloser(bytes.NewReader(planDiffBuf)), nil).Maybe()

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			actualDiff, err := service.GetPlanDiff(auth.WithCaller(ctx, mockCaller), run.PlanID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectedDiff, actualDiff)
		})
	}
}

func TestUploadPlanBinary(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"
	planID := "plan-1"

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: workspaceID,
		PlanID:      planID,
	}

	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		expectData      string
	}

	testCases := []testCase{
		{
			name:            "subject does not have permission to upload plan binary",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:       "upload plan binary",
			expectData: "test data",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockRuns := db.NewMockRuns(t)

			mockArtifactStore := workspace.NewMockArtifactStore(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdatePlanPermission, mock.Anything).Return(test.authError)

			mockRuns.On("GetRunByPlanID", mock.Anything, run.PlanID).Return(run, nil).Maybe()

			if test.authError == nil {
				matcher := mock.MatchedBy(func(reader io.Reader) bool {
					actual, err := io.ReadAll(reader)
					require.NoError(t, err)

					return string(actual) == test.expectData
				})
				mockArtifactStore.On("UploadPlanCache", mock.Anything, run, matcher).Return(nil)
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			err := service.UploadPlanBinary(auth.WithCaller(ctx, mockCaller), run.PlanID, strings.NewReader(test.expectData))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestProcessPlanData(t *testing.T) {
	workspaceID := "ws1"
	runID := "run1"
	planID := "plan-1"

	run := &models.Run{
		Metadata: models.ResourceMetadata{
			ID: runID,
		},
		WorkspaceID: workspaceID,
		PlanID:      planID,
	}

	type testCase struct {
		authError         error
		name              string
		expectErrorCode   errors.CodeType
		tfPlan            *tfjson.Plan
		tfProviderSchemas *tfjson.ProviderSchemas
		expectedPlan      *models.Plan
		expectDiff        *plan.Diff
	}

	testCases := []testCase{
		{
			name:            "subject does not have permission to update plan",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "process plan data",
			tfPlan: &tfjson.Plan{
				FormatVersion: "0.1",
				OutputChanges: map[string]*tfjson.Change{
					"test": {
						Actions: tfjson.Actions{tfjson.ActionCreate},
					},
				},
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{
				FormatVersion: "0.1",
			},
			expectedPlan: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID: planID,
				},
				WorkspaceID: workspaceID,
				Summary: models.PlanSummary{
					OutputAdditions: 1,
				},
				PlanDiffSize: 126,
			},
			expectDiff: &plan.Diff{
				Outputs: []*plan.OutputDiff{
					{
						OutputName: "test",
						Action:     action.Create,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockRuns := db.NewMockRuns(t)
			mockPlans := db.NewMockPlans(t)
			mockTransactions := db.NewMockTransactions(t)

			mockArtifactStore := workspace.NewMockArtifactStore(t)

			mockParser := plan.NewMockParser(t)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdatePlanPermission, mock.Anything).Return(test.authError)

			mockRuns.On("GetRunByPlanID", mock.Anything, run.PlanID).Return(run, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			if test.authError == nil {
				mockParser.On("Parse", test.tfPlan, test.tfProviderSchemas).Return(test.expectDiff, nil)

				mockPlans.On("GetPlan", mock.Anything, run.PlanID).Return(&models.Plan{
					Metadata: models.ResourceMetadata{
						ID: planID,
					},
					WorkspaceID: workspaceID,
				}, nil)
				mockPlans.On("UpdatePlan", mock.Anything, test.expectedPlan).Return(test.expectedPlan, nil)

				planDiffMatcher := mock.MatchedBy(func(reader io.Reader) bool {
					actual, err := io.ReadAll(reader)
					require.NoError(t, err)

					expected, err := json.Marshal(test.expectDiff)
					require.NoError(t, err)

					return string(actual) == string(expected)
				})
				mockArtifactStore.On("UploadPlanDiff", mock.Anything, run, planDiffMatcher).Return(nil)

				planJSONMatcher := mock.MatchedBy(func(reader io.Reader) bool {
					actual, err := io.ReadAll(reader)
					require.NoError(t, err)

					expected, err := json.Marshal(test.tfPlan)
					require.NoError(t, err)

					return string(actual) == string(expected)
				})
				mockArtifactStore.On("UploadPlanJSON", mock.Anything, run, planJSONMatcher).Return(nil)

				mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()
			}

			dbClient := &db.Client{
				Runs:         mockRuns,
				Plans:        mockPlans,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()

			service := &service{
				dbClient:        dbClient,
				artifactStore:   mockArtifactStore,
				runStateManager: state.NewRunStateManager(dbClient, logger),
				planParser:      mockParser,
			}

			err := service.ProcessPlanData(auth.WithCaller(ctx, mockCaller), run.PlanID, test.tfPlan, test.tfProviderSchemas)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSubscribeToRunEvents(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		authError      error
		input          *EventSubscriptionOptions
		name           string
		expectErrCode  errors.CodeType
		runner         *models.Runner
		sendEventData  []*db.RunEventData
		expectedEvents []Event
		isAdmin        bool
		useUserCaller  bool
		nilUserMember  bool
		nilWorkspaceID bool
	}{
		{
			name: "subscribe to run events for a workspace",
			input: &EventSubscriptionOptions{
				WorkspaceID: ptr.String("workspace1"),
			},
			sendEventData: []*db.RunEventData{
				{
					ID:          "run1",
					WorkspaceID: "workspace1",
				},
				{
					ID:          "run2",
					WorkspaceID: "workspace1",
				},
			},
			expectedEvents: []Event{
				{
					Run: models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
				{
					Run: models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run2",
						},
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name: "not authorized to subscribe to run events for a workspace",
			input: &EventSubscriptionOptions{
				WorkspaceID: ptr.String("workspace1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "subscribe to run events for a run",
			input: &EventSubscriptionOptions{
				RunID: ptr.String("run1"),
			},
			useUserCaller:  true,
			nilWorkspaceID: true,
			sendEventData: []*db.RunEventData{
				{
					ID: "run1",
				},
				{
					ID: "run2",
				},
			},
			expectedEvents: []Event{
				{
					Run: models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group1"),
			},
		},
		{
			name: "not authorized to subscribe to run events for a run",
			input: &EventSubscriptionOptions{
				RunID: ptr.String("run1"),
			},
			runner: &models.Runner{
				Metadata: models.ResourceMetadata{ID: "runner1"},
				Type:     models.GroupRunnerType,
				GroupID:  ptr.String("group1"),
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:    "subscribe to all run events",
			input:   &EventSubscriptionOptions{},
			isAdmin: true,
			sendEventData: []*db.RunEventData{
				{
					ID: "run1",
				},
				{
					ID: "run2",
				},
			},
			useUserCaller:  true,
			nilUserMember:  true,
			nilWorkspaceID: true,
			expectedEvents: []Event{
				{
					Run: models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run1",
						},
					},
					Action: "UPDATE",
				},
				{
					Run: models.Run{
						Metadata: models.ResourceMetadata{
							ID: "run2",
						},
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name:          "not authorized to subscribe to all run events",
			input:         &EventSubscriptionOptions{},
			expectErrCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockRunners := db.NewMockRunners(t)
			mockRuns := db.NewMockRuns(t)
			mockEvents := db.NewMockEvents(t)

			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			if test.input.WorkspaceID != nil {
				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewRunPermission, mock.Anything).
					Return(test.authError)
			}

			for _, d := range test.sendEventData {
				dCopy := d

				getRunsFilter := &db.RunFilter{
					WorkspaceID: &dCopy.WorkspaceID,
					RunIDs:      []string{dCopy.ID},
				}
				if test.nilWorkspaceID {
					getRunsFilter.WorkspaceID = nil
				}
				if test.useUserCaller && !test.nilUserMember {
					getRunsFilter.UserMemberID = &userID
				}
				mockRuns.On("GetRuns", mock.Anything, &db.GetRunsInput{
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
					Filter: getRunsFilter,
				}).
					Return(&db.RunsResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: 1,
						},
						Runs: []models.Run{
							{
								Metadata: models.ResourceMetadata{
									ID: dCopy.ID,
								},
							}},
					}, nil).Maybe()
			}

			dbClient := db.Client{
				Runners: mockRunners,
				Runs:    mockRuns,
				Events:  mockEvents,
			}

			logger, _ := logger.NewForTest()
			eventManager := events.NewEventManager(&dbClient, logger)
			eventManager.Start(ctx)

			service := &service{
				dbClient:     &dbClient,
				eventManager: eventManager,
				logger:       logger,
			}

			var useCaller auth.Caller = mockCaller
			if test.useUserCaller {
				useCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: userID,
						},
						Admin: test.isAdmin,
					},
					mockAuthorizer,
					&dbClient,
					mockMaintenanceMonitor,
				)
			}

			eventChannel, err := service.SubscribeToRunEvents(auth.WithCaller(ctx, useCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			receivedEvents := []*Event{}

			go func() {
				for _, d := range test.sendEventData {
					encoded, err := json.Marshal(d)
					require.Nil(t, err)

					mockEventChannel <- db.Event{
						Table:  "runs",
						Action: "UPDATE",
						ID:     d.ID,
						Data:   encoded,
					}
				}
			}()

			if len(test.expectedEvents) > 0 {
				for e := range eventChannel {
					eCopy := e

					receivedEvents = append(receivedEvents, eCopy)

					if len(receivedEvents) == len(test.expectedEvents) {
						break
					}
				}
			}

			require.Equal(t, len(test.expectedEvents), len(receivedEvents))
			for i, e := range test.expectedEvents {
				assert.Equal(t, e, *receivedEvents[i])
			}
		})
	}
}

func TestSetVariablesIncludedInTFConfig(t *testing.T) {
	runID := "run-1"
	planID := "plan-1"

	type testCase struct {
		name            string
		run             *models.Run
		authError       error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "set run variables",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				PlanID: planID,
			},
		},
		{
			name: "not authorized to set run variables",
			run: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: runID,
				},
				PlanID: planID,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "run not found",
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockRuns := db.NewMockRuns(t)
			mockCaller := auth.NewMockCaller(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)

			sampleVariables := []Variable{
				{
					Key:           "my_var",
					Value:         ptr.String("my value"),
					NamespacePath: ptr.String("group/workspace"),
					Category:      models.TerraformVariableCategory,
				},
				{
					Key:      "my_var2",
					Value:    ptr.String("my value2"),
					Category: models.TerraformVariableCategory,
				},
				{
					Key:      "my_var",
					Value:    ptr.String("my value"),
					Category: models.EnvironmentVariableCategory,
				},
			}

			mockRuns.On("GetRun", mock.Anything, runID).Return(tc.run, nil)

			if tc.run != nil {
				mockCaller.On("RequirePermission", mock.Anything, permissions.UpdatePlanPermission, mock.Anything).Return(tc.authError)

				if tc.authError == nil {
					data, err := json.Marshal(sampleVariables)
					require.NoError(t, err)

					mockArtifactStore.On("GetRunVariables", mock.Anything, tc.run).Return(io.NopCloser(bytes.NewReader(data)), nil)

					// Should only mark first variable as used.
					sampleVariables[0].IncludedInTFConfig = ptr.Bool(true)
					sampleVariables[1].IncludedInTFConfig = ptr.Bool(false)

					data, err = json.Marshal(sampleVariables)
					require.NoError(t, err)

					mockArtifactStore.On("UploadRunVariables", mock.Anything, tc.run, bytes.NewReader(data)).Return(nil)
				}
			}

			dbClient := &db.Client{
				Runs: mockRuns,
			}

			service := &service{
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			err := service.SetVariablesIncludedInTFConfig(auth.WithCaller(ctx, mockCaller), &SetVariablesIncludedInTFConfigInput{
				RunID:        runID,
				VariableKeys: []string{"my_var"},
			})

			if tc.expectErrorCode != "" {
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCreateWorkspaceAssessmentRunForWorkspace(t *testing.T) {
	testSubject := "tester"
	currentTime := time.Now()

	stateVersionID := "sv1"
	planID := "plan1"

	moduleSource := "mymodule"
	moduleVersion := "1.0.0"
	configurationVersionID := "cv1"

	ws := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "ws1",
		},
		FullPath:              "groupA/ws1",
		CurrentStateVersionID: stateVersionID,
		MaxJobDuration:        ptr.Int32(60),
		RunnerTags:            []string{},
	}

	// Test cases
	tests := []struct {
		input              *CreateAssessmentRunForWorkspaceInput
		stateVersion       *models.StateVersion
		currentRun         *models.Run
		existingAssessment *models.WorkspaceAssessment
		expectCreateRun    *models.Run
		name               string
		authError          error
		expectErrorCode    errors.CodeType
	}{
		{
			name: "create assessment run for workspace with no existing assessment",
			input: &CreateAssessmentRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
				RunID:       ptr.String("run1"),
			},
			currentRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
			},
			expectCreateRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:                "run2",
					CreationTimestamp: &currentTime,
				},
				WorkspaceID:     ws.Metadata.ID,
				ModuleSource:    &moduleSource,
				ModuleVersion:   &moduleVersion,
				IsAssessmentRun: true,
			},
		},
		{
			name: "create assessment run for workspace with existing assessment",
			input: &CreateAssessmentRunForWorkspaceInput{
				WorkspaceID:             ws.Metadata.ID,
				LatestAssessmentVersion: ptr.Int(1),
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
				RunID:       ptr.String("run1"),
			},
			currentRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
			},
			existingAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID:      "assessment1",
					Version: 1,
				},
				WorkspaceID:          ws.Metadata.ID,
				StartedAtTimestamp:   currentTime,
				CompletedAtTimestamp: &currentTime,
			},
			expectCreateRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:                "run2",
					CreationTimestamp: &currentTime,
				},
				WorkspaceID:     ws.Metadata.ID,
				ModuleSource:    &moduleSource,
				ModuleVersion:   &moduleVersion,
				IsAssessmentRun: true,
			},
		},
		{
			name: "assessment version does not match version specified in the input",
			input: &CreateAssessmentRunForWorkspaceInput{
				WorkspaceID:             ws.Metadata.ID,
				LatestAssessmentVersion: ptr.Int(1),
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
				RunID:       ptr.String("run1"),
			},
			currentRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
			},
			existingAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID:      "assessment1",
					Version: 2,
				},
				WorkspaceID:          ws.Metadata.ID,
				StartedAtTimestamp:   currentTime,
				CompletedAtTimestamp: &currentTime,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "assessment run is already in progress",
			input: &CreateAssessmentRunForWorkspaceInput{
				WorkspaceID:             ws.Metadata.ID,
				LatestAssessmentVersion: ptr.Int(1),
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
				RunID:       ptr.String("run1"),
			},
			currentRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID: "run1",
				},
				WorkspaceID:   ws.Metadata.ID,
				ModuleSource:  &moduleSource,
				ModuleVersion: &moduleVersion,
			},
			existingAssessment: &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID:      "assessment1",
					Version: 1,
				},
				WorkspaceID:        ws.Metadata.ID,
				StartedAtTimestamp: currentTime,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "cannot create assessment run if state version was created manually",
			input: &CreateAssessmentRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: ws.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "expect authorization error",
			input: &CreateAssessmentRunForWorkspaceInput{
				WorkspaceID: ws.Metadata.ID,
			},
			authError:       errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup mocks
			dbClient := buildDBClientWithMocks(t)
			mockArtifactStore := workspace.NewMockArtifactStore(t)
			mockModuleService := moduleregistry.NewMockService(t)
			mockModuleResolver := NewMockModuleResolver(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockCaller := auth.NewMockCaller(t)

			ctx = auth.WithCaller(ctx, mockCaller)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateRunPermission, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return(testSubject).Maybe()

			dbClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			dbClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()

			dbClient.MockWorkspaces.On("GetWorkspaceByID", mock.Anything, ws.Metadata.ID).Return(ws, nil).Maybe()
			dbClient.MockStateVersions.On("GetStateVersion", mock.Anything, stateVersionID).Return(test.stateVersion, nil).Maybe()

			if test.stateVersion != nil && test.stateVersion.RunID != nil {
				dbClient.MockRuns.On("GetRun", mock.Anything, *test.stateVersion.RunID).Return(test.currentRun, nil).Maybe()
			}

			runVariablesJSON, err := json.Marshal([]Variable{
				{Key: "k1", Value: ptr.String("v1"), Category: models.TerraformVariableCategory},
			})

			if test.currentRun != nil {
				require.NoError(t, err)
				mockArtifactStore.On("GetRunVariables", mock.Anything, test.currentRun).Return(io.NopCloser(bytes.NewReader(runVariablesJSON)), nil).Maybe()
			}

			dbClient.MockWorkspaceAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, test.input.WorkspaceID).Return(test.existingAssessment, nil).Maybe()

			if test.existingAssessment == nil {
				dbClient.MockWorkspaceAssessments.On("CreateWorkspaceAssessment", mock.Anything, mock.Anything).Return(nil, nil)
			} else {
				matcher := mock.MatchedBy(func(assessment *models.WorkspaceAssessment) bool {
					return assessment.CompletedAtTimestamp == nil
				})
				dbClient.MockWorkspaceAssessments.On("UpdateWorkspaceAssessment", mock.Anything, matcher).Return(nil, nil).Maybe()
			}

			if test.expectErrorCode == "" {
				dbClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil)

				dbClient.MockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, ws.Metadata.ID).
					Return([]models.ManagedIdentity{}, nil)

				dbClient.MockRuns.On("CreateRun", mock.Anything, mock.Anything).
					Return(func(_ context.Context, run *models.Run) (*models.Run, error) {
						run.Metadata.CreationTimestamp = &currentTime
						run.Metadata.ID = test.expectCreateRun.Metadata.ID
						return run, nil
					}, nil)
				dbClient.MockRuns.On("GetRuns", mock.Anything, mock.Anything).
					Return(&db.RunsResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: 1,
						},
					}, nil)
				dbClient.MockRuns.On("UpdateRun", mock.Anything, mock.Anything).Return(test.expectCreateRun, nil)

				dbClient.MockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: 10}, nil)

				if test.currentRun.ConfigurationVersionID != nil {
					dbClient.MockConfigurationVersions.On("GetConfigurationVersion", mock.Anything, configurationVersionID).
						Return(&models.ConfigurationVersion{
							Speculative: false,
						}, nil)
				}

				dbClient.MockPlans.On("CreatePlan", mock.Anything, mock.Anything).Return(&models.Plan{
					Metadata: models.ResourceMetadata{
						ID: planID,
					},
				}, nil)

				dbClient.MockJobs.On("CreateJob", mock.Anything, mock.Anything).
					Return(func(_ context.Context, _ *models.Job) (*models.Job, error) {
						return &models.Job{
							Metadata: models.ResourceMetadata{
								ID: "job1",
							},
							WorkspaceID: ws.Metadata.ID,
						}, nil
					}, nil)

				dbClient.MockLogStreams.On("CreateLogStream", mock.Anything, mock.Anything).Return(&models.LogStream{}, nil)

				matcher := mock.MatchedBy(func(run *models.Run) bool {
					return run.Metadata.ID == test.expectCreateRun.Metadata.ID
				})
				mockArtifactStore.On("UploadRunVariables", mock.Anything, matcher, bytes.NewReader(runVariablesJSON)).Return(nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

				mockModuleResolver.On("ParseModuleRegistrySource", mock.Anything, mock.Anything).
					Return(&ModuleRegistrySource{}, nil).Maybe()

				mockModuleResolver.On("ResolveModuleVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(moduleVersion, nil).Maybe()
			}

			// Create test fixture
			logger, _ := logger.NewForTest()
			testService := service{
				logger:          logger,
				dbClient:        dbClient.Client,
				artifactStore:   mockArtifactStore,
				activityService: mockActivityEvents,
				moduleService:   mockModuleService,
				moduleResolver:  mockModuleResolver,
				limitChecker:    limits.NewLimitChecker(dbClient.Client),
			}

			// Invoke test method
			run, err := testService.CreateAssessmentRunForWorkspace(ctx, test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			// Assertions
			require.NoError(t, err)

			assert.True(t, run.IsAssessmentRun)
			assert.True(t, run.Speculative())
			assert.Equal(t, test.expectCreateRun.WorkspaceID, run.WorkspaceID)
			assert.Equal(t, test.expectCreateRun.ModuleSource, run.ModuleSource)
			assert.Equal(t, test.expectCreateRun.ModuleVersion, run.ModuleVersion)
			assert.Equal(t, test.expectCreateRun.ConfigurationVersionID, run.ConfigurationVersionID)
		})
	}
}
