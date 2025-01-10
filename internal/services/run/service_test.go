package run

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
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
	MockVariables             *db.MockVariables
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

	mockLogStreams := db.MockLogStreams{}
	mockLogStreams.Test(t)

	mockResourceLimits := db.MockResourceLimits{}
	mockResourceLimits.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

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
			LogStreams:            &mockLogStreams,
			ResourceLimits:        &mockResourceLimits,
			Groups:                &mockGroups,
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
		MockLogStreams:            &mockLogStreams,
		MockResourceLimits:        &mockResourceLimits,
		MockGroups:                &mockGroups,
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
			expectTags: nil,
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
					return assert.ElementsMatch(t, input.Tags, test.expectTags)
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
