package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	coreworkspace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	namespace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"

	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
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

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil)
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
							TotalCount: pagination.StaticCount(test.injectWorkspacesPerGroup),
						},
					}
				}, nil)

				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			testLogger, _ := logger.NewForTest()

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, nil, ">= 1.0.0", nil)

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

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateWorkspacePermission, mock.Anything).
				Return(test.authError)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			mockTransactions.On("BeginTx", mock.Anything).
				Return(auth.WithCaller(ctx, mockCaller), nil).Maybe()
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

			dbClient := &db.Client{
				Workspaces:   mockWorkspaces,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient:                      dbClient,
				logger:                        logger,
				terraformCLIVersionConstraint: ">= 1.0.0",
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
			TRN: trn.TypeWorkspace.Build("group/workspace-name"),
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
			TRN: trn.TypeStateVersion.Build("state-version-gid-1"),
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
			TRN: trn.TypeWorkspaceAssessment.Build("group/workspace-name/assessment-1"),
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

func TestCV_UploadConfigurationVersion(t *testing.T) {
	configVersionID := "config-version-1"
	workspaceID := "workspace-1"

	tests := []struct {
		name            string
		status          models.ConfigurationStatus
		authError       error
		expectErrorCode errors.CodeType
	}{
		{
			name:   "successfully upload configuration version in pending status",
			status: models.ConfigurationPending,
		},
		{
			name:            "reject upload when configuration version is already uploaded",
			status:          models.ConfigurationUploaded,
			expectErrorCode: errors.EConflict,
		},
		{
			name:            "reject upload when configuration version is errored",
			status:          models.ConfigurationErrored,
			expectErrorCode: errors.EConflict,
		},
		{
			name:            "subject does not have permission",
			status:          models.ConfigurationPending,
			authError:       errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockConfigVersions := db.NewMockConfigurationVersions(t)
			mockArtifactStore := coreworkspace.NewMockArtifactStore(t)

			cv := &models.ConfigurationVersion{
				Metadata:    models.ResourceMetadata{ID: configVersionID},
				WorkspaceID: workspaceID,
				Status:      test.status,
			}

			mockConfigVersions.On("GetConfigurationVersionByID", mock.Anything, configVersionID).Return(cv, nil)
			mockCaller.On("RequirePermission", mock.Anything, models.ViewConfigurationVersionPermission, mock.Anything).Return(nil)
			mockCaller.On("RequirePermission", mock.Anything, models.UpdateConfigurationVersionPermission, mock.Anything).Return(test.authError)

			if test.authError == nil && test.status == models.ConfigurationPending {
				mockArtifactStore.On("UploadConfigurationVersion", mock.Anything, cv, mock.Anything).Return(nil)
				mockConfigVersions.On("UpdateConfigurationVersion", mock.Anything, mock.Anything).Return(cv, nil)
			}

			dbClient := &db.Client{
				ConfigurationVersions: mockConfigVersions,
			}

			testLogger, _ := logger.NewForTest()

			service := &service{
				logger:        testLogger,
				dbClient:      dbClient,
				artifactStore: mockArtifactStore,
			}

			err := service.UploadConfigurationVersion(auth.WithCaller(ctx, mockCaller), configVersionID, strings.NewReader("test"))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetConfigurationVersionByTRN(t *testing.T) {
	sampleConfigVersion := &models.ConfigurationVersion{
		Metadata: models.ResourceMetadata{
			ID:  "config-version-1",
			TRN: trn.TypeConfigurationVersion.Build("config-version-gid-1"),
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
		getRootNamespacesError          error
		input                           *GetWorkspacesInput
		handleCaller                    handleCallerFunc
		userID                          *string
		serviceAccountID                *string
		name                            string
		expectErrorCode                 errors.CodeType
		expectResult                    []models.Workspace
		expectMemberships               []models.MembershipNamespace
		rootNamespaces                  []models.MembershipNamespace
		failAuthorization               bool
		adminMode                       bool
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
			name:      "positive: successfully returns workspaces",
			input:     &GetWorkspacesInput{},
			adminMode: true,
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:                   "negative: failed to get root namespaces",
			input:                  &GetWorkspacesInput{},
			adminMode:              false,
			getRootNamespacesError: errors.New("failure", errors.WithErrorCode(errors.EInvalid)),
			expectErrorCode:        errors.EInvalid,
		},
		{
			name:      "positive: successfully returns workspaces the user has permission to",
			input:     &GetWorkspacesInput{},
			adminMode: false,
			rootNamespaces: []models.MembershipNamespace{
				{ID: "ns-1", Path: "group-a"},
				{ID: "ns-2", Path: "group-b/sub"},
			},
			expectMemberships: []models.MembershipNamespace{
				{ID: "ns-1", Path: "group-a"},
				{ID: "ns-2", Path: "group-b/sub"},
			},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:      "positive: successfully returns workspaces the service account has permission to",
			input:     &GetWorkspacesInput{},
			adminMode: false,
			rootNamespaces: []models.MembershipNamespace{
				{ID: "ns-3", Path: "group-c"},
			},
			expectMemberships: []models.MembershipNamespace{
				{ID: "ns-3", Path: "group-c"},
			},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:              "positive: non admin caller with no root namespaces yields empty membership filter",
			input:             &GetWorkspacesInput{},
			adminMode:         false,
			rootNamespaces:    []models.MembershipNamespace{},
			expectMemberships: []models.MembershipNamespace{},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name:               "negative: failed to get workspaces",
			input:              &GetWorkspacesInput{},
			adminMode:          true,
			getWorkspacesError: errors.New("failure", errors.WithErrorCode(errors.EInvalid)),
			expectErrorCode:    errors.EInvalid,
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
				PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)},
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
				PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(1)},
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
				PageInfo:   &pagination.PageInfo{TotalCount: pagination.StaticCount(0)},
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
			userID:            ptr.String("user-1"),
			adminMode:         false,
			expectMemberships: []models.MembershipNamespace{},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
		{
			name: "negative: service account cannot filter by favorites",
			input: &GetWorkspacesInput{
				Favorites: ptr.Bool(true),
			},
			serviceAccountID: ptr.String("sa-1"),
			adminMode:        false,
			expectErrorCode:  errors.EInvalid,
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

			// The membership filter branch only runs for non-group queries on the mockCaller.
			usesMockCaller := !(test.input.Favorites != nil && *test.input.Favorites && test.userID != nil)
			if test.input.GroupID == nil && usesMockCaller {
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.adminMode).Maybe()
				if !test.adminMode {
					mockCaller.On("GetRootNamespaceMemberships", mock.Anything).Return(test.rootNamespaces, test.getRootNamespacesError).Maybe()
				}
			}

			if test.expectMemberships != nil {
				input.Filter.RootNamespaceMemberships = test.expectMemberships
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

			service := newService(nil, dbClient, nil, nil, nil, "", nil, test.handleCaller)

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

func TestGetStateVersionDependencies(t *testing.T) {
	workspaceID := "workspace-1"

	buildStateJSON := func(attributes json.RawMessage) string {
		state := stateV4{
			Version: version4,
			Resources: []resourceStateV4{
				{
					ProviderConfig: tharsisTerraformProviderConfig,
					Type:           tharsisWorkspaceOutputsDatasourceName,
					Name:           "test",
					Instances: []instanceObjectStateV4{
						{AttributesRaw: attributes},
					},
				},
			},
		}
		b, _ := json.Marshal(state)
		return string(b)
	}

	// setupCaller creates a mock caller with the given permission error and adds it to the context.
	setupCaller := func(ctx context.Context, t *testing.T, permErr error) context.Context {
		mockCaller := auth.MockCaller{}
		mockCaller.Test(t)
		mockCaller.On("RequirePermission", mock.Anything, models.ViewStateVersionPermission, mock.Anything).
			Return(permErr)
		return auth.WithCaller(ctx, &mockCaller)
	}

	// setupService creates a service with the given artifact store reader.
	setupService := func(t *testing.T, reader io.ReadCloser, readErr error) *service {
		mockArtifactStore := coreworkspace.NewMockArtifactStore(t)
		mockArtifactStore.On("GetStateVersion", mock.Anything, mock.Anything).
			Return(reader, readErr)
		return &service{dbClient: &db.Client{}, artifactStore: mockArtifactStore}
	}

	tests := []struct {
		mockSetup       func(ctx context.Context, t *testing.T) (context.Context, *service)
		name            string
		expectResult    []StateVersionDependency
		expectErrorCode errors.CodeType
	}{
		{
			name: "auth failure",
			mockSetup: func(ctx context.Context, _ *testing.T) (context.Context, *service) {
				return ctx, &service{dbClient: &db.Client{}}
			},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "permission error",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
				return ctx, &service{dbClient: &db.Client{}}
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "artifact store error",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				svc := setupService(t, nil, errors.New("store error", errors.WithErrorCode(errors.EInternal)))
				return ctx, svc
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid state JSON",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				svc := setupService(t, io.NopCloser(strings.NewReader("not-json")), nil)
				return ctx, svc
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "wrong state version",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				state, _ := json.Marshal(stateV4{Version: 3})
				svc := setupService(t, io.NopCloser(strings.NewReader(string(state))), nil)
				return ctx, svc
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "valid dependencies",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				stateJSON := buildStateJSON(json.RawMessage(`{"full_path":"group/ws","state_version_id":"SV_abc","workspace_id":"WS_123"}`))
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
				return ctx, svc
			},
			expectResult: []StateVersionDependency{
				{
					WorkspacePath:  "group/ws",
					WorkspaceID:    gid.FromGlobalID("WS_123"),
					StateVersionID: gid.FromGlobalID("SV_abc"),
				},
			},
		},
		{
			name: "nil attribute values are skipped",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				stateJSON := buildStateJSON(json.RawMessage(`{"full_path":null,"state_version_id":null,"workspace_id":null}`))
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
				return ctx, svc
			},
			expectResult: []StateVersionDependency{},
		},
		{
			name: "partially nil attribute values are skipped",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				stateJSON := buildStateJSON(json.RawMessage(`{"full_path":"group/ws","state_version_id":null,"workspace_id":"WS_123"}`))
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
				return ctx, svc
			},
			expectResult: []StateVersionDependency{},
		},
		{
			name: "missing full_path attribute",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				stateJSON := buildStateJSON(json.RawMessage(`{"state_version_id":"SV_abc","workspace_id":"WS_123"}`))
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
				return ctx, svc
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "non-string full_path attribute",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				stateJSON := buildStateJSON(json.RawMessage(`{"full_path":123,"state_version_id":"SV_abc","workspace_id":"WS_123"}`))
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
				return ctx, svc
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "no tharsis resources in state",
			mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
				ctx = setupCaller(ctx, t, nil)
				state, _ := json.Marshal(stateV4{
					Version: version4,
					Resources: []resourceStateV4{
						{ProviderConfig: "provider[\"registry.terraform.io/hashicorp/aws\"]", Type: "aws_instance", Name: "test"},
					},
				})
				svc := setupService(t, io.NopCloser(strings.NewReader(string(state))), nil)
				return ctx, svc
			},
			expectResult: []StateVersionDependency{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, svc := test.mockSetup(t.Context(), t)

			result, err := svc.GetStateVersionDependencies(ctx, &models.StateVersion{WorkspaceID: workspaceID})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResult, result)
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
			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockStateVersions := db.NewMockStateVersions(t)
			mockStateVersions.On("CreateStateVersion", mock.Anything, test.toCreate).
				Return(test.injectCreated, test.createError).Maybe()
			mockStateVersions.On("GetStateVersions", mock.Anything, mock.Anything).
				Return(&db.StateVersionsResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: pagination.StaticCount(test.injectSVsPerWorkspace),
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

			mockArtifactStore := coreworkspace.MockArtifactStore{}
			mockArtifactStore.Test(t)

			mockArtifactStore.On("UploadStateVersion", mock.Anything, mock.Anything, mock.Anything).
				Return(test.uploadError)

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				Transactions:   mockTransactions,
				StateVersions:  mockStateVersions,
				ResourceLimits: mockResourceLimits,
				Workspaces:     mockWorkspaces,
			}

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), &mockArtifactStore, nil, "", nil)

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
			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil).Maybe()
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
						TotalCount: pagination.StaticCount(test.injectSVsPerWorkspace),
					},
				}, nil).Maybe()

			mockResourceLimits := db.NewMockResourceLimits(t)
			mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
				Return(&models.ResourceLimit{Value: test.limit}, nil).Maybe()

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				Transactions:          mockTransactions,
				ConfigurationVersions: mockConfigurationVersions,
				ResourceLimits:        mockResourceLimits,
			}

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), nil, nil, "", nil)

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
						TotalCount: pagination.StaticCount(test.newParentChildren),
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

			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEventsDB := db.NewMockActivityEvents(t)
			mockActivityEventsDB.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil).Maybe()

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()

			dbClient := db.Client{
				Groups:         &mockGroups,
				Workspaces:     &mockWorkspaces,
				Transactions:   &mockTransactions,
				ResourceLimits: mockResourceLimits,
				ActivityEvents: mockActivityEventsDB,
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

			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, testCaller), nil)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, limiter, nil, nil, "", nil)

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
					Type: WorkspaceEventUpdated,
				},
			},
		},
		{
			name: "subscribe also fires on workspace assessment create/update events",
			input: &EventSubscriptionOptions{
				WorkspaceID: "workspace1",
			},
			sendEvents: []*db.Event{
				{
					// Assessment for this workspace fires (keyed on the assessment ID, scoped
					// by workspace_id in the event data).
					Table:  "workspace_assessments",
					Action: "INSERT",
					ID:     "assessment1",
					Data:   json.RawMessage(`{"workspace_id":"workspace1"}`),
				},
				{
					// Assessment for a different workspace is filtered out.
					Table:  "workspace_assessments",
					Action: "UPDATE",
					ID:     "assessment2",
					Data:   json.RawMessage(`{"workspace_id":"workspace2"}`),
				},
			},
			expectedEvents: []Event{
				{
					Workspace: models.Workspace{
						Metadata: models.ResourceMetadata{
							ID: "workspace1",
						},
					},
					Type: WorkspaceEventAssessmentCreated,
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

			// Every delivered event resolves the workspace by the subscription's workspace ID.
			if test.input != nil {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.input.WorkspaceID).
					Return(&models.Workspace{
						Metadata: models.ResourceMetadata{
							ID: test.input.WorkspaceID,
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
					// Default to a workspace UPDATE event; cases that exercise other tables
					// (e.g. workspace_assessments) set Table/Action/Data explicitly.
					table := d.Table
					if table == "" {
						table = "workspaces"
					}
					action := d.Action
					if action == "" {
						action = "UPDATE"
					}
					data := d.Data
					if data == nil {
						encoded, err := json.Marshal(d)
						require.Nil(t, err)
						data = encoded
					}

					mockEventChannel <- db.Event{
						Table:  table,
						Action: action,
						ID:     d.ID,
						Data:   data,
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

func TestDeleteWorkspace(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject is not authorized",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "subject is authorized",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			workspace := &models.Workspace{
				Metadata: models.ResourceMetadata{ID: "workspace-1"},
				FullPath: "group-1/workspace-1",
				GroupID:  "group-1",
			}

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.DeleteWorkspacePermission, mock.Anything).Return(test.authError)

			mockTransactions := db.NewMockTransactions(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockWorkspaces.On("DeleteWorkspace", mock.Anything, workspace).Return(nil)
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient: &db.Client{
					Transactions: mockTransactions,
					Workspaces:   mockWorkspaces,
				},
				logger: logger,
			}

			err := service.DeleteWorkspace(auth.WithCaller(ctx, mockCaller), workspace, false)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestLockWorkspace(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject is not authorized",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "subject is authorized",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			workspace := &models.Workspace{
				Metadata: models.ResourceMetadata{ID: "workspace-1"},
			}

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.UpdateWorkspacePermission, mock.Anything).Return(test.authError)

			mockTransactions := db.NewMockTransactions(t)
			mockWorkspaces := db.NewMockWorkspaces(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).Return(workspace, nil)
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient: &db.Client{
					Transactions: mockTransactions,
					Workspaces:   mockWorkspaces,
				},
				logger: logger,
			}

			_, err := service.LockWorkspace(auth.WithCaller(ctx, mockCaller), workspace)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUnlockWorkspace(t *testing.T) {
	type testCase struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "subject is not authorized",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "subject is authorized",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Must be locked so UnlockWorkspace proceeds past the already-unlocked check.
			workspace := &models.Workspace{
				Metadata: models.ResourceMetadata{ID: "workspace-1"},
				Locked:   true,
			}

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, models.UpdateWorkspacePermission, mock.Anything).Return(test.authError)

			mockTransactions := db.NewMockTransactions(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockWorkItemsQueue := db.NewMockWorkItemsQueue(t)

			if test.authError == nil {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockWorkspaces.On("UpdateWorkspace", mock.Anything, mock.Anything).Return(workspace, nil)
				mockWorkItemsQueue.On("AddWorkItemToQueue", mock.Anything, mock.Anything).Return(&db.WorkItem{}, nil)
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient: &db.Client{
					Transactions:   mockTransactions,
					Workspaces:     mockWorkspaces,
					WorkItemsQueue: mockWorkItemsQueue,
				},
				logger: logger,
			}

			_, err := service.UnlockWorkspace(auth.WithCaller(ctx, mockCaller), workspace)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
