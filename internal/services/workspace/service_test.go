package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	namespace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"

	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestCreateWorkspace(t *testing.T) {
	groupPath := "group/path"
	groupID := "group-id"
	workspaceID := "workspace-id"
	workspaceName := "workspace-name"
	workspaceDescription := "workspace description"
	workspacePath := groupPath + "/" + workspaceName
	terraformVersion := "1.2.2"

	// Test cases
	tests := []struct {
		authError                error
		expectCreatedWorkspace   *models.Workspace
		name                     string
		expectErrCode            errors.CodeType
		input                    models.Workspace
		limit                    int
		injectWorkspacesPerGroup int32
		exceedsLimit             bool
	}{
		{
			name: "create workspace",
			input: models.Workspace{
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
				RunnerTags:         []string{"tag1"},
			},
			expectCreatedWorkspace: &models.Workspace{
				Metadata:           models.ResourceMetadata{ID: workspaceID},
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
				FullPath:           workspacePath,
			},
			limit:                    5,
			injectWorkspacesPerGroup: 5,
		},
		{
			name: "create workspace with labels",
			input: models.Workspace{
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
				RunnerTags:         []string{"tag1"},
				Labels: map[string]string{
					"environment": "production",
					"team":        "platform",
				},
			},
			expectCreatedWorkspace: &models.Workspace{
				Metadata:           models.ResourceMetadata{ID: workspaceID},
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
				FullPath:           workspacePath,
				Labels: map[string]string{
					"environment": "production",
					"team":        "platform",
				},
			},
			limit:                    5,
			injectWorkspacesPerGroup: 5,
		},
		{
			name: "subject does not have permission",
			input: models.Workspace{
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name: "exceeds limit",
			input: models.Workspace{
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
			},
			expectCreatedWorkspace: &models.Workspace{
				Metadata:           models.ResourceMetadata{ID: workspaceID},
				Name:               workspaceName,
				GroupID:            groupID,
				Description:        workspaceDescription,
				MaxJobDuration:     ptr.Int32(1234),
				PreventDestroyPlan: true,
				TerraformVersion:   terraformVersion,
				FullPath:           workspacePath,
			},
			limit:                    5,
			injectWorkspacesPerGroup: 6,
			exceedsLimit:             true,
			expectErrCode:            errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateWorkspacePermission, mock.Anything).Return(test.authError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			mockCLIStore := cli.NewMockTerraformCLIStore(t)
			// Apparently, it is not necessary to mock anything out, just have the interface instantiated.

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				if !test.exceedsLimit {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
			}

			if (test.expectCreatedWorkspace != nil) || test.exceedsLimit {
				mockWorkspaces.On("CreateWorkspace", mock.Anything, mock.Anything).
					Return(test.expectCreatedWorkspace, nil)
			}

			dbClient := db.Client{
				Transactions:   mockTransactions,
				Workspaces:     mockWorkspaces,
				ResourceLimits: mockResourceLimits,
			}

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil && !test.exceedsLimit {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			// Called inside transaction to check resource limits.
			if test.limit > 0 {
				mockWorkspaces.On("GetWorkspaces", mock.Anything, mock.Anything).Return(&db.GetWorkspacesInput{
					Filter: &db.WorkspaceFilter{
						GroupID: &groupID,
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(0),
					},
				}).Return(func(ctx context.Context, input *db.GetWorkspacesInput) *db.WorkspacesResult {
					_ = ctx
					_ = input

					return &db.WorkspacesResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.injectWorkspacesPerGroup,
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			testLogger, _ := logger.NewForTest()
			mockCLIService := cli.NewService(testLogger, nil, nil, mockCLIStore, ">= 1.0.0")

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, nil, mockCLIService, mockActivityEvents, nil)

			workspace, err := service.CreateWorkspace(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedWorkspace, workspace)
		})
	}
}

func TestUpdateWorkspace(t *testing.T) {
	terraformVersion := "1.2.2"

	originalWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
		Name:             "workspace-name",
		FullPath:         "parent-group/workspace-name",
		Description:      "This is the old description",
		MaxJobDuration:   ptr.Int32(35),
		RunnerTags:       []string{"tag1"},
		TerraformVersion: terraformVersion,
	}

	updatedWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "workspace-id",
		},
		Name:             "workspace-name",
		FullPath:         "parent-group/workspace-name",
		Description:      "This is the new description",
		MaxJobDuration:   ptr.Int32(38),
		RunnerTags:       []string{"tag2", "tag3"},
		TerraformVersion: terraformVersion,
	}

	type testCase struct {
		name               string
		foundWorkspace     *models.Workspace
		authError          error
		updateError        error
		expectWorkspace    *models.Workspace
		expectErrorCode    errors.CodeType
		expectLabelChanges *models.LabelChangePayload
	}

	testCases := []testCase{
		{
			name:            "successfully update an workspace",
			foundWorkspace:  originalWorkspace,
			expectWorkspace: updatedWorkspace,
		},
		{
			name:            "workspace does not exist",
			updateError:     errors.New("workspace not found", errors.WithErrorCode(errors.ENotFound)),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "caller does not have permission",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "successfully update workspace with label changes",
			foundWorkspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "workspace-id",
				},
				Name:             "workspace-name",
				FullPath:         "parent-group/workspace-name",
				Description:      "workspace description",
				MaxJobDuration:   ptr.Int32(35),
				TerraformVersion: terraformVersion,
				Labels: map[string]string{
					"environment": "staging",
					"team":        "platform",
					"version":     "1.0",
				},
			},
			expectWorkspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: "workspace-id",
				},
				Name:             "workspace-name",
				FullPath:         "parent-group/workspace-name",
				Description:      "workspace description",
				MaxJobDuration:   ptr.Int32(35),
				TerraformVersion: terraformVersion,
				Labels: map[string]string{
					"environment": "production",  // updated
					"team":        "platform",    // unchanged
					"project":     "new-project", // added
					// "version" removed
				},
			},
			expectLabelChanges: &models.LabelChangePayload{
				Added: map[string]string{
					"project": "new-project",
				},
				Updated: map[string]string{
					"environment": "production",
				},
				Removed: []string{"version"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateWorkspacePermission, mock.Anything).
				Return(test.authError)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			mockTransactions.On("BeginTx", mock.Anything).
				Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).
				Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).
				Return(nil).Maybe()

			// Mock GetWorkspaceByID for label change detection
			if test.foundWorkspace != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, mock.Anything).
					Return(test.foundWorkspace, nil).Maybe()
			} else if test.updateError != nil {
				// For error cases, GetWorkspaceByID might still be called
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, mock.Anything).
					Return(nil, test.updateError).Maybe()
			}

			if test.expectWorkspace != nil {
				mockWorkspaces.On("UpdateWorkspace", mock.Anything, test.expectWorkspace).
					Return(test.expectWorkspace, test.updateError).Maybe()
			} else if test.updateError != nil {
				mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
					Return(nil, test.updateError).Maybe()
			}

			// Verify activity event with label changes if expected
			if test.expectLabelChanges != nil {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything,
					mock.MatchedBy(func(input *activityevent.CreateActivityEventInput) bool {
						if input.Action != models.ActionUpdate || input.TargetType != models.TargetWorkspace {
							return false
						}

						payload, ok := input.Payload.(*models.ActivityEventUpdateWorkspacePayload)
						if !ok || payload.LabelChanges == nil {
							return false
						}

						// Check that label changes match expected
						changes := payload.LabelChanges
						if len(changes.Added) != len(test.expectLabelChanges.Added) ||
							len(changes.Updated) != len(test.expectLabelChanges.Updated) ||
							len(changes.Removed) != len(test.expectLabelChanges.Removed) {
							return false
						}

						for k, v := range test.expectLabelChanges.Added {
							if changes.Added[k] != v {
								return false
							}
						}

						for k, v := range test.expectLabelChanges.Updated {
							if changes.Updated[k] != v {
								return false
							}
						}

						for i, key := range test.expectLabelChanges.Removed {
							if changes.Removed[i] != key {
								return false
							}
						}

						return true
					}),
				).Return(&models.ActivityEvent{}, nil).Maybe()
			} else {
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
					Return(&models.ActivityEvent{}, nil).Maybe()
			}

			testLogger, _ := logger.NewForTest()
			mockCLIStore := cli.NewMockTerraformCLIStore(t)
			mockCLIService := cli.NewService(testLogger, nil, nil, mockCLIStore, ">= 1.0.0")

			dbClient := &db.Client{
				Workspaces:   mockWorkspaces,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient:        dbClient,
				logger:          logger,
				activityService: mockActivityEvents,
				cliService:      mockCLIService,
			}

			workspaceToUpdate := updatedWorkspace
			if test.expectWorkspace != nil {
				workspaceToUpdate = test.expectWorkspace
			}

			actualUpdated, err := service.UpdateWorkspace(auth.WithCaller(ctx, mockCaller), workspaceToUpdate)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectWorkspace, actualUpdated)
		})
	}
}

func TestGetWorkspaceByID(t *testing.T) {
	workspaceID := "workspace-1"
	workspaceName := "workspace-name"
	workspacePath := "group/workspace-name"

	// Test cases
	tests := []struct {
		name            string
		workspaceID     string
		workspace       *models.Workspace
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name:        "successfully get workspace by ID",
			workspaceID: workspaceID,
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: workspaceID,
				},
				Name:     workspaceName,
				FullPath: workspacePath,
			},
		},
		{
			name:            "workspace not found",
			workspaceID:     workspaceID,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:        "subject does not have permission",
			workspaceID: workspaceID,
			workspace: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: workspaceID,
				},
				Name:     workspaceName,
				FullPath: workspacePath,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.workspaceID).Return(test.workspace, nil)

			if test.workspace != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Workspaces: mockWorkspaces,
			}

			service := &service{
				dbClient: dbClient,
			}

			workspace, err := service.GetWorkspaceByID(auth.WithCaller(ctx, mockCaller), test.workspaceID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.workspace, workspace)
		})
	}
}

func TestGetWorkspaceByTRN(t *testing.T) {
	sampleWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{
			ID:  "workspace-1",
			TRN: types.WorkspaceModelType.BuildTRN("group/workspace-name"),
		},
		Name:     "workspace-name",
		FullPath: "group/workspace-name",
		GroupID:  "group-1",
	}

	type testCase struct {
		name            string
		authError       error
		workspace       *models.Workspace
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:      "successfully get workspace by trn",
			workspace: sampleWorkspace,
		},
		{
			name:            "workspace not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view workspace",
			workspace: &models.Workspace{
				Metadata: sampleWorkspace.Metadata,
				Name:     sampleWorkspace.Name,
				FullPath: sampleWorkspace.FullPath,
				GroupID:  sampleWorkspace.GroupID,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			mockWorkspaces.On("GetWorkspaceByTRN", mock.Anything, sampleWorkspace.Metadata.TRN).Return(test.workspace, nil)

			if test.workspace != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Workspaces: mockWorkspaces,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualWorkspace, err := service.GetWorkspaceByTRN(auth.WithCaller(ctx, mockCaller), sampleWorkspace.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.workspace, actualWorkspace)
		})
	}
}

func TestGetStateVersionByID(t *testing.T) {
	stateVersionID := "state-version-1"
	workspaceID := "workspace-1"

	// Test cases
	tests := []struct {
		name            string
		stateVersionID  string
		stateVersion    *models.StateVersion
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name:           "successfully get state version by ID",
			stateVersionID: stateVersionID,
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: workspaceID,
			},
		},
		{
			name:            "state version not found",
			stateVersionID:  stateVersionID,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:           "subject does not have permission",
			stateVersionID: stateVersionID,
			stateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID: stateVersionID,
				},
				WorkspaceID: workspaceID,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockStateVersions := db.NewMockStateVersions(t)

			mockStateVersions.On("GetStateVersionByID", mock.Anything, test.stateVersionID).Return(test.stateVersion, nil)

			if test.stateVersion != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewStateVersionPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				StateVersions: mockStateVersions,
			}

			service := &service{
				dbClient: dbClient,
			}

			stateVersion, err := service.GetStateVersionByID(auth.WithCaller(ctx, mockCaller), test.stateVersionID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.stateVersion, stateVersion)
		})
	}
}

func TestGetStateVersionByTRN(t *testing.T) {
	sampleStateVersion := &models.StateVersion{
		Metadata: models.ResourceMetadata{
			ID:  "state-version-1",
			TRN: types.StateVersionModelType.BuildTRN("state-version-gid-1"),
		},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		name            string
		authError       error
		stateVersion    *models.StateVersion
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:         "successfully get state version by trn",
			stateVersion: sampleStateVersion,
		},
		{
			name:            "state version not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view state version",
			stateVersion: &models.StateVersion{
				Metadata:    sampleStateVersion.Metadata,
				WorkspaceID: sampleStateVersion.WorkspaceID,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockStateVersions := db.NewMockStateVersions(t)

			mockStateVersions.On("GetStateVersionByTRN", mock.Anything, sampleStateVersion.Metadata.TRN).Return(test.stateVersion, nil)

			if test.stateVersion != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewStateVersionPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				StateVersions: mockStateVersions,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualStateVersion, err := service.GetStateVersionByTRN(auth.WithCaller(ctx, mockCaller), sampleStateVersion.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.stateVersion, actualStateVersion)
		})
	}
}

func TestGetWorkspaceAssessmentByTRN(t *testing.T) {
	sampleAssessment := &models.WorkspaceAssessment{
		Metadata: models.ResourceMetadata{
			ID:  "assessment-1",
			TRN: types.WorkspaceAssessmentModelType.BuildTRN("group/workspace-name/assessment-1"),
		},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		name            string
		authError       error
		assessment      *models.WorkspaceAssessment
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:       "successfully get assessment by trn",
			assessment: sampleAssessment,
		},
		{
			name:            "assessment not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view assessment",
			assessment: &models.WorkspaceAssessment{
				Metadata:    sampleAssessment.Metadata,
				WorkspaceID: sampleAssessment.WorkspaceID,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockAssessments := db.NewMockWorkspaceAssessments(t)

			mockAssessments.On("GetWorkspaceAssessmentByTRN", mock.Anything, sampleAssessment.Metadata.TRN).Return(test.assessment, nil)

			if test.assessment != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				WorkspaceAssessments: mockAssessments,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualAssessment, err := service.GetWorkspaceAssessmentByTRN(auth.WithCaller(ctx, mockCaller), sampleAssessment.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.assessment, actualAssessment)
		})
	}
}

func TestGetConfigurationVersionByID(t *testing.T) {
	configVersionID := "config-version-1"
	workspaceID := "workspace-1"

	// Test cases
	tests := []struct {
		name            string
		configVersionID string
		configVersion   *models.ConfigurationVersion
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name:            "successfully get configuration version by ID",
			configVersionID: configVersionID,
			configVersion: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID: configVersionID,
				},
				WorkspaceID: workspaceID,
			},
		},
		{
			name:            "configuration version not found",
			configVersionID: configVersionID,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject does not have permission",
			configVersionID: configVersionID,
			configVersion: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					ID: configVersionID,
				},
				WorkspaceID: workspaceID,
			},
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockConfigVersions := db.NewMockConfigurationVersions(t)

			mockConfigVersions.On("GetConfigurationVersionByID", mock.Anything, test.configVersionID).Return(test.configVersion, nil)

			if test.configVersion != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewConfigurationVersionPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				ConfigurationVersions: mockConfigVersions,
			}

			service := &service{
				dbClient: dbClient,
			}

			configVersion, err := service.GetConfigurationVersionByID(auth.WithCaller(ctx, mockCaller), test.configVersionID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.configVersion, configVersion)
		})
	}
}

func TestGetConfigurationVersionByTRN(t *testing.T) {
	sampleConfigVersion := &models.ConfigurationVersion{
		Metadata: models.ResourceMetadata{
			ID:  "config-version-1",
			TRN: types.ConfigurationVersionModelType.BuildTRN("config-version-gid-1"),
		},
		WorkspaceID: "workspace-1",
	}

	type testCase struct {
		name            string
		authError       error
		configVersion   *models.ConfigurationVersion
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:          "successfully get configuration version by trn",
			configVersion: sampleConfigVersion,
		},
		{
			name:            "configuration version not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "subject is not authorized to view configuration version",
			configVersion: &models.ConfigurationVersion{
				Metadata:    sampleConfigVersion.Metadata,
				WorkspaceID: sampleConfigVersion.WorkspaceID,
			},
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockConfigVersions := db.NewMockConfigurationVersions(t)

			mockConfigVersions.On("GetConfigurationVersionByTRN", mock.Anything, sampleConfigVersion.Metadata.TRN).Return(test.configVersion, nil)

			if test.configVersion != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewConfigurationVersionPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				ConfigurationVersions: mockConfigVersions,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualConfigVersion, err := service.GetConfigurationVersionByTRN(auth.WithCaller(ctx, mockCaller), sampleConfigVersion.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.configVersion, actualConfigVersion)
		})
	}
}

func TestGetWorkspaces(t *testing.T) {
	groupID := "some-group-id"
	sampleWorkspace := models.Workspace{
		Metadata: models.ResourceMetadata{
			ID: "some-id",
		},
		Name:     "a-workspace",
		FullPath: "some/full/path",
		GroupID:  "some-group-id",
	}

	type testCase struct {
		getWorkspacesError              error
		requireWorkspacePermissionError error
		namespaceAccessPolicyError      error
		input                           *GetWorkspacesInput
		handleCaller                    handleCallerFunc
		userID                          *string
		serviceAccountID                *string
		name                            string
		expectErrorCode                 errors.CodeType
		expectResult                    []models.Workspace
		failAuthorization               bool
		accessPolicyAllowAll            bool
		dbResult                        *db.WorkspacesResult
		dbError                         error
		authError                       error
	}

	testCases := []testCase{
		{
			name: "positive: successfully returns workspaces for a group",
			input: &GetWorkspacesInput{
				GroupID: &groupID,
			},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:              "negative: failed to authorize caller",
			input:             &GetWorkspacesInput{},
			failAuthorization: true,
			expectErrorCode:   errors.EUnauthorized,
		},
		{
			name:                 "positive: successfully returns workspaces",
			input:                &GetWorkspacesInput{},
			accessPolicyAllowAll: true,
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:                       "negative: failed to get namespace access policy",
			input:                      &GetWorkspacesInput{},
			namespaceAccessPolicyError: errors.New("failure", errors.WithErrorCode(errors.EInvalid)),
			expectErrorCode:            errors.EInvalid,
		},
		{
			name:                 "positive: successfully returns workspaces the user has permission to",
			input:                &GetWorkspacesInput{},
			userID:               ptr.String("user-1"),
			accessPolicyAllowAll: false,
			handleCaller: func(ctx context.Context, userHandler func(ctx context.Context, caller *auth.UserCaller) error, _ func(ctx context.Context, caller *auth.ServiceAccountCaller) error) error {
				return userHandler(ctx, &auth.UserCaller{User: &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}}})
			},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:                 "positive: successfully returns workspaces the service account has permission to",
			input:                &GetWorkspacesInput{},
			serviceAccountID:     ptr.String("sa-1"),
			accessPolicyAllowAll: false,
			handleCaller: func(ctx context.Context, _ func(ctx context.Context, caller *auth.UserCaller) error, serviceAccountHandler func(ctx context.Context, caller *auth.ServiceAccountCaller) error) error {
				return serviceAccountHandler(ctx, &auth.ServiceAccountCaller{ServiceAccountID: "sa-1"})
			},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:                 "negative: failed to set filters for non admin caller",
			input:                &GetWorkspacesInput{},
			accessPolicyAllowAll: false,
			handleCaller: func(_ context.Context, _ func(ctx context.Context, caller *auth.UserCaller) error, _ func(ctx context.Context, caller *auth.ServiceAccountCaller) error) error {
				return errors.New("failure", errors.WithErrorCode(errors.EInvalid))
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                 "negative: failed to get workspaces",
			input:                &GetWorkspacesInput{},
			accessPolicyAllowAll: true,
			getWorkspacesError:   errors.New("failure", errors.WithErrorCode(errors.EInvalid)),
			expectErrorCode:      errors.EInvalid,
		},
		{
			name: "positive: filter workspaces by single label",
			input: &GetWorkspacesInput{
				GroupID: &groupID,
				LabelFilters: []db.WorkspaceLabelFilter{
					{Key: "environment", Value: "production"},
				},
			},
			dbResult: &db.WorkspacesResult{
				Workspaces: []models.Workspace{
					{
						Metadata: models.ResourceMetadata{ID: "workspace-1"},
						Name:     "workspace-name",
						FullPath: "group/workspace-name",
						Labels: map[string]string{
							"environment": "production",
							"team":        "platform",
						},
					},
				},
				PageInfo: &pagination.PageInfo{TotalCount: 1},
			},
			expectResult: []models.Workspace{
				{
					Metadata: models.ResourceMetadata{ID: "workspace-1"},
					Name:     "workspace-name",
					FullPath: "group/workspace-name",
					Labels: map[string]string{
						"environment": "production",
						"team":        "platform",
					},
				},
			},
		},
		{
			name: "positive: filter workspaces by multiple labels",
			input: &GetWorkspacesInput{
				GroupID: &groupID,
				LabelFilters: []db.WorkspaceLabelFilter{
					{Key: "environment", Value: "production"},
					{Key: "team", Value: "platform"},
				},
			},
			dbResult: &db.WorkspacesResult{
				Workspaces: []models.Workspace{
					{
						Metadata: models.ResourceMetadata{ID: "workspace-1"},
						Name:     "workspace-name",
						FullPath: "group/workspace-name",
						Labels: map[string]string{
							"environment": "production",
							"team":        "platform",
						},
					},
				},
				PageInfo: &pagination.PageInfo{TotalCount: 1},
			},
			expectResult: []models.Workspace{
				{
					Metadata: models.ResourceMetadata{ID: "workspace-1"},
					Name:     "workspace-name",
					FullPath: "group/workspace-name",
					Labels: map[string]string{
						"environment": "production",
						"team":        "platform",
					},
				},
			},
		},
		{
			name: "positive: no matching workspaces with label filter",
			input: &GetWorkspacesInput{
				GroupID: &groupID,
				LabelFilters: []db.WorkspaceLabelFilter{
					{Key: "environment", Value: "development"},
				},
			},
			dbResult: &db.WorkspacesResult{
				Workspaces: []models.Workspace{},
				PageInfo:   &pagination.PageInfo{TotalCount: 0},
			},
			expectResult: []models.Workspace{},
		},
		{
			name: "negative: database error with label filter",
			input: &GetWorkspacesInput{
				GroupID: &groupID,
				LabelFilters: []db.WorkspaceLabelFilter{
					{Key: "environment", Value: "production"},
				},
			},
			dbError:         errors.New("database error"),
			expectErrorCode: errors.EInternal,
		},
		{
			name: "negative: auth error with label filter",
			input: &GetWorkspacesInput{
				GroupID: &groupID,
				LabelFilters: []db.WorkspaceLabelFilter{
					{Key: "environment", Value: "production"},
				},
			},
			authError:       errors.New("auth failed", errors.WithErrorCode(errors.EUnauthorized)),
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "positive: user can filter by favorites",
			input: &GetWorkspacesInput{
				Favorites: ptr.Bool(true),
			},
			userID:               ptr.String("user-1"),
			accessPolicyAllowAll: false,
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name: "negative: service account cannot filter by favorites",
			input: &GetWorkspacesInput{
				Favorites: ptr.Bool(true),
			},
			serviceAccountID:     ptr.String("sa-1"),
			accessPolicyAllowAll: false,
			expectErrorCode:      errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockCaller := auth.NewMockCaller(t)

			if !test.failAuthorization {
				if test.input.Favorites != nil && *test.input.Favorites && test.userID != nil {
					mockAuthorizer := auth.NewMockAuthorizer(t)
					mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
					mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{}, nil).Maybe()
					ctx = auth.WithCaller(ctx, auth.NewUserCaller(
						&models.User{Metadata: models.ResourceMetadata{ID: *test.userID}},
						mockAuthorizer,
						&db.Client{Workspaces: mockWorkspaces},
						mockMaintenanceMonitor,
						nil,
					))
				} else {
					ctx = auth.WithCaller(ctx, mockCaller)
				}
			}

			input := db.GetWorkspacesInput{
				Sort:              test.input.Sort,
				PaginationOptions: test.input.PaginationOptions,
				Filter: &db.WorkspaceFilter{
					Search:                    test.input.Search,
					AssignedManagedIdentityID: test.input.AssignedManagedIdentityID,
					LabelFilters:              test.input.LabelFilters,
				},
			}

			// Handle favorites filter
			if test.input.Favorites != nil && *test.input.Favorites && test.userID != nil {
				input.Filter.FavoriteUserID = test.userID
			}

			if test.input.GroupID != nil {
				input.Filter.GroupID = test.input.GroupID

				if test.authError != nil {
					mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)
				} else {
					mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.requireWorkspacePermissionError)
				}
			}

			policy := auth.NamespaceAccessPolicy{AllowAll: test.accessPolicyAllowAll}
			mockCaller.On("GetNamespaceAccessPolicy", mock.Anything).Return(&policy, test.namespaceAccessPolicyError).Maybe()

			if test.userID != nil {
				input.Filter.UserMemberID = test.userID
			}

			if test.serviceAccountID != nil {
				input.Filter.ServiceAccountMemberID = test.serviceAccountID
			}

			// Handle label filter test cases
			if len(test.input.LabelFilters) > 0 {
				if test.dbError != nil {
					mockWorkspaces.On("GetWorkspaces", mock.Anything, mock.MatchedBy(func(input *db.GetWorkspacesInput) bool {
						return input.Filter != nil && len(input.Filter.LabelFilters) > 0
					})).Return(nil, test.dbError).Maybe()
				} else if test.dbResult != nil {
					mockWorkspaces.On("GetWorkspaces", mock.Anything, mock.MatchedBy(func(input *db.GetWorkspacesInput) bool {
						return input.Filter != nil && len(input.Filter.LabelFilters) > 0
					})).Return(test.dbResult, nil).Maybe()
				}
			} else {
				workspacesResult := db.WorkspacesResult{Workspaces: test.expectResult}
				mockWorkspaces.On("GetWorkspaces", mock.Anything, &input).Return(&workspacesResult, test.getWorkspacesError).Maybe()
			}

			dbClient := &db.Client{
				Workspaces: mockWorkspaces,
			}

			if test.handleCaller == nil {
				test.handleCaller = auth.HandleCaller
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, nil, test.handleCaller)

			result, err := service.GetWorkspaces(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, result.Workspaces)
		})
	}
}

func TestCreateStateVersion(t *testing.T) {
	stateVersionID := "state-version-1"
	workspaceID := "workspace-1"
	runID := "run-1"
	subject := "subject-1"
	goodData := buildEncodedData("{\"version\": 4}")
	badData := buildEncodedData("{\"version\": 4, \"serial\": \"bad-serial\"}")
	currentTime := time.Now().UTC()

	toCreate := &models.StateVersion{
		WorkspaceID: workspaceID,
		RunID:       &runID,
	}

	type testCase struct {
		authFail                 bool
		workspacePermissionError error
		dataUnmarshalError       error
		uploadError              error
		createError              error
		dataDecodeError          error
		toCreate                 *models.StateVersion
		injectCreated            *models.StateVersion
		expectResult             *models.StateVersion
		data                     []byte
		name                     string
		expectErrorCode          errors.CodeType
		limit                    int
		injectSVsPerWorkspace    int32
	}

	/*
		Test case template:

		name                     string
		authFail                 bool
		toCreate                 *models.StateVersion
		data                     []byte
		injectCreated            *models.StateVersion
		workspacePermissionError error
		dataDecodeError          error
		createError              error
		limit                    int
		injectSVsPerWorkspace    int32
		dataUnmarshalError       error
		uploadError              error
		expectResult             *models.StateVersion
		expectErrorCode          errors.CodeType
	*/

	// Test cases
	tests := []testCase{
		{
			name:            "auth failure",
			toCreate:        toCreate,
			authFail:        true,
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:                     "permission error",
			toCreate:                 toCreate,
			workspacePermissionError: errors.New("workspace permission denied", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:          errors.EForbidden,
		},
		{
			name:            "decode failure",
			toCreate:        toCreate,
			data:            []byte("something-not-decodable"),
			dataDecodeError: errors.New("data string not decodable", errors.WithErrorCode(errors.EInvalid)),
			expectErrorCode: errors.EInternal, // Had thought it should return EInvalid, but it returns EInternal.
		},
		{
			name:     "create failed",
			toCreate: toCreate,
			data:     goodData,
			injectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
			},
			createError:     errors.New("create failed due to something not found", errors.WithErrorCode(errors.ENotFound)),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:     "exceeds limit",
			toCreate: toCreate,
			data:     goodData,
			injectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
			},
			limit:                 4,
			injectSVsPerWorkspace: 5,
			expectErrorCode:       errors.EInvalid,
		},
		{
			name:     "unmarshal error",
			toCreate: toCreate,
			data:     badData,
			injectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
			},
			limit:                 4,
			injectSVsPerWorkspace: 4,
			dataUnmarshalError:    errors.New("failed to unmarshal decoded data", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode:       errors.EInternal,
		},
		{
			name:     "upload error",
			toCreate: toCreate,
			data:     goodData,
			injectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
			},
			limit:                 4,
			injectSVsPerWorkspace: 4,
			uploadError:           errors.New("failed to upload data", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode:       errors.EInternal,
		},
		{
			name:     "successfully created",
			toCreate: toCreate,
			data:     goodData,
			injectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
					ID:                stateVersionID,
				},
				WorkspaceID: workspaceID,
				RunID:       &runID,
				CreatedBy:   subject,
			},
			limit:                 4,
			injectSVsPerWorkspace: 4,
			expectResult: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
					ID:                stateVersionID,
				},
				CreatedBy:   subject,
				WorkspaceID: workspaceID,
				RunID:       &runID,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateStateVersionPermission, mock.Anything).
				Return(test.workspacePermissionError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockStateVersions := db.NewMockStateVersions(t)
			mockStateVersions.On("CreateStateVersion", mock.Anything, test.toCreate).
				Return(test.injectCreated, test.createError).Maybe()
			mockStateVersions.On("GetStateVersions", mock.Anything, mock.Anything).
				Return(&db.StateVersionsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: test.injectSVsPerWorkspace,
					},
				}, nil).Maybe()

			mockResourceLimits := db.NewMockResourceLimits(t)
			mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: test.limit}, nil).Maybe()

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, mock.Anything).
				Return(&models.Workspace{
					Metadata: models.ResourceMetadata{
						ID: workspaceID,
					},
				}, nil).Maybe()
			mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).
				Return(&models.Workspace{
					Metadata: models.ResourceMetadata{
						ID: workspaceID,
					},
				}, nil).Maybe()

			mockArtifactStore := MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadStateVersion", mock.Anything, mock.Anything, mock.Anything).
				Return(test.uploadError)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)
			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(&models.ActivityEvent{}, nil)

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				Transactions:   mockTransactions,
				StateVersions:  mockStateVersions,
				ResourceLimits: mockResourceLimits,
				Workspaces:     mockWorkspaces,
			}

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), &mockArtifactStore, nil, nil, &mockActivityEvents, nil)

			if !test.authFail {
				ctx = auth.WithCaller(ctx, &mockCaller)
			}

			testDataString := string(test.data)
			result, err := service.CreateStateVersion(ctx, test.toCreate, testDataString)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, result)
		})
	}
}

func buildEncodedData(input string) []byte {
	output := make([]byte, base64.StdEncoding.EncodedLen(len(input)))
	base64.StdEncoding.Encode(output, []byte(input))
	return output
}

func TestCreateConfigurationVersion(t *testing.T) {
	configurationVersionID := "configuration-version-1"
	workspaceID := "workspace-1"
	subject := "mockSubject"
	isSpeculative := false
	status := models.ConfigurationPending
	currentTime := time.Now().UTC()

	toCreate := &CreateConfigurationVersionInput{
		WorkspaceID: workspaceID,
		Speculative: isSpeculative,
	}

	type testCase struct {
		name                     string
		authFail                 bool
		toCreate                 *CreateConfigurationVersionInput
		injectCreated            *models.ConfigurationVersion
		workspacePermissionError error
		createError              error
		limit                    int
		injectSVsPerWorkspace    int32
		expectResult             *models.ConfigurationVersion
		expectErrorCode          errors.CodeType
	}

	// Test cases
	tests := []testCase{
		{
			name:            "auth failure",
			toCreate:        toCreate,
			authFail:        true,
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:                     "permission error",
			toCreate:                 toCreate,
			workspacePermissionError: errors.New("workspace permission denied", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode:          errors.EForbidden,
		},
		{
			name:     "create failed",
			toCreate: toCreate,
			injectCreated: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
			},
			createError:     errors.New("create failed due to something not found", errors.WithErrorCode(errors.ENotFound)),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:     "exceeds limit",
			toCreate: toCreate,
			injectCreated: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
				},
			},
			limit:                 4,
			injectSVsPerWorkspace: 5,
			expectErrorCode:       errors.EInvalid,
		},
		{
			name:     "successfully created",
			toCreate: toCreate,
			injectCreated: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
					ID:                configurationVersionID,
				},
				WorkspaceID: workspaceID,
				CreatedBy:   subject,
			},
			limit:                 4,
			injectSVsPerWorkspace: 4,
			expectResult: &models.ConfigurationVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
					ID:                configurationVersionID,
				},
				CreatedBy:   subject,
				WorkspaceID: workspaceID,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateConfigurationVersionPermission, mock.Anything).
				Return(test.workspacePermissionError)

			mockCaller.On("GetSubject").Return("mockSubject")

			mockTransactions := db.NewMockTransactions(t)
			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockConfigurationVersions := db.NewMockConfigurationVersions(t)
			mockConfigurationVersions.On("CreateConfigurationVersion", mock.Anything, models.ConfigurationVersion{
				Status:      status,
				WorkspaceID: test.toCreate.WorkspaceID,
				CreatedBy:   subject,
				Speculative: isSpeculative,
			}).
				Return(test.injectCreated, test.createError).Maybe()
			mockConfigurationVersions.On("GetConfigurationVersions", mock.Anything, mock.Anything).
				Return(&db.ConfigurationVersionsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: test.injectSVsPerWorkspace,
					},
				}, nil).Maybe()

			mockResourceLimits := db.NewMockResourceLimits(t)
			mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: test.limit}, nil).Maybe()

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)
			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(&models.ActivityEvent{}, nil)

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				Transactions:          mockTransactions,
				ConfigurationVersions: mockConfigurationVersions,
				ResourceLimits:        mockResourceLimits,
			}

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), nil, nil, nil, &mockActivityEvents, nil)

			if !test.authFail {
				ctx = auth.WithCaller(ctx, &mockCaller)
			}

			result, err := service.CreateConfigurationVersion(ctx, test.toCreate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, result)
		})
	}
}

func TestMigrateWorkspace(t *testing.T) {
	oldParentID := "old-parent-id"
	oldParentName := "old-parent-name"

	testOldParent := models.Group{
		Metadata: models.ResourceMetadata{ID: oldParentID},
		Name:     oldParentName,
		FullPath: oldParentName,
	}

	testWorkspaceID := "test-workspace-id"
	testWorkspaceName := "test-workspace-name"
	testWorkspaceOldPath := "old-parent-path/" + testWorkspaceName

	testWorkspace := models.Workspace{
		Metadata: models.ResourceMetadata{ID: testWorkspaceID},
		Name:     testWorkspaceName,
		GroupID:  oldParentID,
		FullPath: testWorkspaceOldPath,
	}

	newParentID := "new-parent-id"
	newParentName := "new-parent-name"
	newParentPath := "new-grandparent-name/" + newParentName

	testNewParent := models.Group{
		Metadata: models.ResourceMetadata{ID: newParentID},
		Name:     newParentName,
		FullPath: newParentPath,
	}

	// Test cases
	tests := []struct {
		newParentID              string
		expectWorkspace          *models.Workspace
		name                     string
		expectErrorCode          errors.CodeType
		inputWorkspace           models.Workspace
		limit                    int
		newParentChildren        int32
		isUserAdmin              bool
		isGroupOwner             bool
		isCallerDeployerOfParent bool
	}{
		{
			name:                     "successful move",
			inputWorkspace:           testWorkspace,
			newParentID:              newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			limit:                    5,
			newParentChildren:        5,
			expectWorkspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: testWorkspaceID},
				Name:     testWorkspaceName,
				GroupID:  newParentID,
				FullPath: newParentPath + "/" + testWorkspaceName,
			},
		},
		{
			name:            "caller is not owner of workspace to be moved",
			inputWorkspace:  testWorkspace,
			newParentID:     newParentID,
			isGroupOwner:    false,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:                     "caller is not deployer (or better) of new parent group",
			inputWorkspace:           testWorkspace,
			newParentID:              newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: false,
			expectErrorCode:          errors.EForbidden,
		},
		{
			name:                     "exceeds limit on workspaces in group",
			inputWorkspace:           testWorkspace,
			newParentID:              newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			limit:                    5,
			newParentChildren:        6,
			expectWorkspace: &models.Workspace{
				Metadata: models.ResourceMetadata{ID: testWorkspaceID},
				Name:     testWorkspaceName,
				GroupID:  "",
				FullPath: testWorkspaceName,
			},
			expectErrorCode: errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var workspaceAccessError, parentAccessError error
			if !test.isGroupOwner {
				workspaceAccessError = errors.New("test user is not owner of workspace being moved", errors.WithErrorCode(errors.EForbidden))
			}
			if !test.isCallerDeployerOfParent {
				parentAccessError = errors.New("test user is not deployer of old or new parent", errors.WithErrorCode(errors.EForbidden))
			}

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockResourceLimits := db.NewMockResourceLimits(t)

			perms := []models.Permission{models.UpdateWorkspacePermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(workspaceAccessError)

			perms = []models.Permission{models.DeleteWorkspacePermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(parentAccessError)

			perms = []models.Permission{models.CreateWorkspacePermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(parentAccessError)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockWorkspaces := db.MockWorkspaces{}
			mockGroups.Test(t)

			mockGroups.On("GetGroupByID", mock.Anything, oldParentID).Return(&testOldParent, nil)
			mockGroups.On("GetGroupByID", mock.Anything, newParentID).Return(&testNewParent, nil)

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.inputWorkspace.Metadata.ID).
				Return(&test.inputWorkspace, nil)

			newParent := &models.Group{
				Metadata: models.ResourceMetadata{
					ID: test.newParentID,
				},
				FullPath: newParentPath,
				Name:     newParentName,
			}

			mockWorkspaces.On("GetWorkspaces", mock.Anything, mock.Anything).Return(
				&db.WorkspacesResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: test.newParentChildren,
					},
				}, nil)

			if test.limit > 0 {
				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			mockGroups.On("GetGroups", mock.Anything, &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &testWorkspaceID,
				},
			}).Return(&db.GroupsResult{Groups: []models.Group{}}, nil)

			mockWorkspaces.On("MigrateWorkspace", mock.Anything, &test.inputWorkspace, newParent).Return(test.expectWorkspace, nil)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()

			dbClient := db.Client{
				Groups:         &mockGroups,
				Workspaces:     &mockWorkspaces,
				Transactions:   &mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			limiter := limits.NewLimitChecker(&dbClient)

			testCaller := auth.NewUserCaller(
				&models.User{
					Metadata: models.ResourceMetadata{
						ID: "123",
					},
					Admin:    test.isUserAdmin,
					Username: "user1",
				},
				&mockAuthorizer,
				&dbClient,
				mockMaintenanceMonitor,
				nil,
			)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, limiter, nil, nil, nil, &mockActivityEvents, nil)

			migrated, err := service.MigrateWorkspace(auth.WithCaller(ctx, testCaller),
				test.inputWorkspace.Metadata.ID, test.newParentID)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectWorkspace, migrated)
			}
		})
	}
}

func TestSubscribeToWorkspaceEvents(t *testing.T) {
	userID := "user1"

	// Test cases
	tests := []struct {
		authError      error
		input          *EventSubscriptionOptions
		name           string
		expectErrCode  errors.CodeType
		workspace      *models.Workspace
		sendEvents     []*db.Event
		expectedEvents []Event
		isAdmin        bool
		useUserCaller  bool
		nilUserMember  bool
		nilWorkspaceID bool
	}{
		{
			name: "subscribe to workspace events for a workspace",
			input: &EventSubscriptionOptions{
				WorkspaceID: "workspace1",
			},
			sendEvents: []*db.Event{
				{
					ID: "workspace1",
				},
				{
					ID: "workspace2",
				},
			},
			expectedEvents: []Event{
				{
					Workspace: models.Workspace{
						Metadata: models.ResourceMetadata{
							ID: "workspace1",
						},
					},
					Action: "UPDATE",
				},
			},
		},
		{
			name: "not authorized to subscribe to workspace events for a workspace",
			input: &EventSubscriptionOptions{
				WorkspaceID: "workspace1",
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockEvents := db.NewMockEvents(t)

			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockEventChannel := make(chan db.Event, 1)
			var roEventChan <-chan db.Event = mockEventChannel
			mockEvents.On("Listen", mock.Anything).Return(roEventChan, make(<-chan error)).Maybe()

			mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).
				Return(test.authError)

			for _, d := range test.sendEvents {
				dCopy := d

				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, dCopy.ID).
					Return(&models.Workspace{
						Metadata: models.ResourceMetadata{
							ID: dCopy.ID,
						},
					}, nil).Maybe()
			}

			dbClient := db.Client{
				Workspaces: mockWorkspaces,
				Events:     mockEvents,
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
					nil,
				)
			}

			eventChannel, err := service.SubscribeToWorkspaceEvents(auth.WithCaller(ctx, useCaller), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			receivedEvents := []*Event{}

			go func() {
				for _, d := range test.sendEvents {
					encoded, err := json.Marshal(d)
					require.Nil(t, err)

					mockEventChannel <- db.Event{
						Table:  "workspaces",
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

func TestGetDriftDetectionEnabledSetting(t *testing.T) {
	workspace := models.Workspace{
		Metadata: models.ResourceMetadata{ID: "ws-1"},
	}
	// Test cases
	tests := []struct {
		expectSetting *namespace.DriftDetectionEnabledSetting
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "get setting",
			expectSetting: &namespace.DriftDetectionEnabledSetting{
				Value: true,
			},
		},
		{
			name:          "unauthorized",
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockInheritedSettingsResolver := namespace.NewMockInheritedSettingResolver(t)
			testLogger, _ := logger.NewForTest()

			mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)

			mockInheritedSettingsResolver.On("GetDriftDetectionEnabled", mock.Anything, &workspace).Return(test.expectSetting, nil).Maybe()

			svc := service{
				logger:                    testLogger,
				inheritedSettingsResolver: mockInheritedSettingsResolver,
			}

			setting, err := svc.GetDriftDetectionEnabledSetting(auth.WithCaller(ctx, mockCaller), &workspace)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectSetting, setting)
		})
	}
}

func TestGetWorkspaceAssessmentByID(t *testing.T) {
	assessmentID := "assessment-1"

	// Test cases
	tests := []struct {
		assessment    *models.WorkspaceAssessment
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "get assessment",
			assessment: &models.WorkspaceAssessment{
				Metadata:    models.ResourceMetadata{ID: assessmentID},
				WorkspaceID: "ws-1",
			},
		},
		{
			name: "unauthorized",
			assessment: &models.WorkspaceAssessment{
				Metadata:    models.ResourceMetadata{ID: assessmentID},
				WorkspaceID: "ws-1",
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			testLogger, _ := logger.NewForTest()

			mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)

			mockWorkspaceAssessments.On("GetWorkspaceAssessmentByID", mock.Anything, assessmentID).Return(test.assessment, nil).Maybe()

			dbClient := db.Client{
				WorkspaceAssessments: mockWorkspaceAssessments,
			}

			svc := service{
				logger:   testLogger,
				dbClient: &dbClient,
			}

			assessment, err := svc.GetWorkspaceAssessmentByID(auth.WithCaller(ctx, mockCaller), assessmentID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.assessment, assessment)
		})
	}
}

func TestGetWorkspaceAssessmentsByWorkspaceIDs(t *testing.T) {
	idList := []string{"ws-1", "ws-2"}

	// Test cases
	tests := []struct {
		assessments   []models.WorkspaceAssessment
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "get assessment",
			assessments: []models.WorkspaceAssessment{
				{
					Metadata:    models.ResourceMetadata{ID: "assessment-1"},
					WorkspaceID: "ws-1",
				},
				{
					Metadata:    models.ResourceMetadata{ID: "assessment-2"},
					WorkspaceID: "ws-2",
				},
			},
		},
		{
			name: "unauthorized",
			assessments: []models.WorkspaceAssessment{
				{
					Metadata:    models.ResourceMetadata{ID: "assessment-1"},
					WorkspaceID: "ws-1",
				},
			},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockWorkspaceAssessments := db.NewMockWorkspaceAssessments(t)
			testLogger, _ := logger.NewForTest()

			mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)

			mockWorkspaceAssessments.On("GetWorkspaceAssessments", mock.Anything, &db.GetWorkspaceAssessmentsInput{
				Filter: &db.WorkspaceAssessmentFilter{
					WorkspaceIDs: idList,
				},
			}).Return(&db.WorkspaceAssessmentsResult{
				WorkspaceAssessments: test.assessments,
			}, nil).Maybe()

			dbClient := db.Client{
				WorkspaceAssessments: mockWorkspaceAssessments,
			}

			svc := service{
				logger:   testLogger,
				dbClient: &dbClient,
			}

			assessments, err := svc.GetWorkspaceAssessmentsByWorkspaceIDs(auth.WithCaller(ctx, mockCaller), idList)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.assessments, assessments)
		})
	}
}

func TestGetProviderMirrorEnabledSetting(t *testing.T) {
	workspace := models.Workspace{
		FullPath: "group1/workspace1",
	}

	tests := []struct {
		name          string
		expectSetting *namespace.ProviderMirrorEnabledSetting
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "get setting",
			expectSetting: &namespace.ProviderMirrorEnabledSetting{
				Value: true,
			},
		},
		{
			name:          "unauthorized",
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockInheritedSettingsResolver := namespace.NewMockInheritedSettingResolver(t)
			testLogger, _ := logger.NewForTest()

			mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspacePermission, mock.Anything).Return(test.authError)

			mockInheritedSettingsResolver.On("GetProviderMirrorEnabled", mock.Anything, &workspace).Return(test.expectSetting, nil).Maybe()

			svc := service{
				logger:                    testLogger,
				inheritedSettingsResolver: mockInheritedSettingsResolver,
			}

			setting, err := svc.GetProviderMirrorEnabledSetting(auth.WithCaller(ctx, mockCaller), &workspace)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectSetting, setting)
		})
	}
}
