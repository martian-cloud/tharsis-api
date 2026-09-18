package workspace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
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

func TestGetWorkspaceRoleBindingByID(t *testing.T) {
	bindingID := "binding-1"
	workspaceID := "workspace-1"

	sampleBinding := &models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: bindingID},
		WorkspaceID: workspaceID,
		RoleID:      "role-1",
	}

	type testCase struct {
		name            string
		binding         *models.WorkspaceRoleBinding
		authError       error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "successfully get workspace role binding by ID",
			binding: sampleBinding,
		},
		{
			name:            "workspace role binding not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view workspace role binding",
			binding:         sampleBinding,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)

			mockBindings.On("GetWorkspaceRoleBindingByID", mock.Anything, bindingID).Return(test.binding, nil)

			if test.binding != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspaceRoleBindingPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				WorkspaceRoleBindings: mockBindings,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualBinding, err := service.GetWorkspaceRoleBindingByID(auth.WithCaller(ctx, mockCaller), bindingID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.binding, actualBinding)
		})
	}
}

func TestGetWorkspaceRoleBindingByTRN(t *testing.T) {
	workspaceID := "workspace-1"
	bindingTRN := trn.TypeWorkspaceRoleBinding.Build("group/workspace-name")

	sampleBinding := &models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: "binding-1", TRN: bindingTRN},
		WorkspaceID: workspaceID,
		RoleID:      "role-1",
	}

	type testCase struct {
		name            string
		binding         *models.WorkspaceRoleBinding
		authError       error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "successfully get workspace role binding by TRN",
			binding: sampleBinding,
		},
		{
			name:            "workspace role binding not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view workspace role binding",
			binding:         sampleBinding,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)

			mockBindings.On("GetWorkspaceRoleBindingByTRN", mock.Anything, bindingTRN).Return(test.binding, nil)

			if test.binding != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspaceRoleBindingPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				WorkspaceRoleBindings: mockBindings,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualBinding, err := service.GetWorkspaceRoleBindingByTRN(auth.WithCaller(ctx, mockCaller), bindingTRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.binding, actualBinding)
		})
	}
}

func TestGetWorkspaceRoleBindingByWorkspaceID(t *testing.T) {
	workspaceID := "workspace-1"

	sampleBinding := &models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: "binding-1"},
		WorkspaceID: workspaceID,
		RoleID:      "role-1",
	}

	type testCase struct {
		name            string
		binding         *models.WorkspaceRoleBinding
		authError       error
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "successfully get workspace role binding by workspace ID",
			binding: sampleBinding,
		},
		{
			name: "workspace has no role binding is not an error",
		},
		{
			name:            "subject is not authorized to view workspace role binding",
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)

			mockCaller.On("RequirePermission", mock.Anything, models.ViewWorkspaceRoleBindingPermission, mock.Anything).Return(test.authError)

			if test.authError == nil {
				mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, workspaceID).Return(test.binding, nil)
			}

			dbClient := &db.Client{
				WorkspaceRoleBindings: mockBindings,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualBinding, err := service.GetWorkspaceRoleBindingByWorkspaceID(auth.WithCaller(ctx, mockCaller), workspaceID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.binding, actualBinding)
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
		linkRefErr      error
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
		{
			name:            "retainFn error is propagated",
			status:          models.ConfigurationPending,
			linkRefErr:      errors.New("link failed", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
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

			mockObjectStoreRefs := db.NewMockObjectStoreRefs(t)
			mockTransactions := db.NewMockTransactions(t)

			if test.authError == nil && test.status == models.ConfigurationPending {
				mockObjectStoreRefs.On("LinkRef", mock.Anything, mock.Anything, db.ObjectStoreRefOwnerConfigurationVersion, cv.Metadata.ID).Return(test.linkRefErr)
				mockArtifactStore.On("UploadConfigurationVersion", mock.Anything, cv, mock.Anything).
					Return(db.RetainObjectRefFunc(func(ctx context.Context, ownerID string) error {
						return mockObjectStoreRefs.LinkRef(ctx, "workspaces/workspace-1/configuration_versions/some-uuid", db.ObjectStoreRefOwnerConfigurationVersion, ownerID)
					}), "workspaces/workspace-1/configuration_versions/some-uuid", nil)
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				if test.linkRefErr == nil {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				}
				mockConfigVersions.On("UpdateConfigurationVersion", mock.Anything, mock.Anything).Return(cv, nil).Maybe()
			}

			dbClient := &db.Client{
				ConfigurationVersions: mockConfigVersions,
				Transactions:          mockTransactions,
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
		{
			name: "positive: user caller with exclude favorites filter",
			input: &GetWorkspacesInput{
				ExcludeFavorites: ptr.Bool(true),
			},
			userID:            ptr.String("user-1"),
			adminMode:         false,
			expectMemberships: []models.MembershipNamespace{},
			expectResult: []models.Workspace{
				sampleWorkspace,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockCaller := auth.NewMockCaller(t)

			if !test.failAuthorization {
				if (test.input.Favorites != nil && *test.input.Favorites && test.userID != nil) ||
					(test.input.ExcludeFavorites != nil && *test.input.ExcludeFavorites && test.userID != nil) {
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

			// Handle exclude favorites filter
			if test.input.ExcludeFavorites != nil && *test.input.ExcludeFavorites && test.userID != nil {
				input.Filter.ExcludeFavoriteUserID = test.userID
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
			usesMockCaller := !((test.input.Favorites != nil && *test.input.Favorites && test.userID != nil) ||
				(test.input.ExcludeFavorites != nil && *test.input.ExcludeFavorites && test.userID != nil))
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

func TestGetStateVersionInventory(t *testing.T) {
	workspaceID := "workspace-1"

	setupCaller := func(ctx context.Context, t *testing.T, permErr error) context.Context {
		mockCaller := auth.MockCaller{}
		mockCaller.Test(t)
		mockCaller.On("RequirePermission", mock.Anything, models.ViewStateVersionPermission, mock.Anything).
			Return(permErr)
		return auth.WithCaller(ctx, &mockCaller)
	}

	setupService := func(t *testing.T, reader io.ReadCloser, readErr error) *service {
		mockArtifactStore := coreworkspace.NewMockArtifactStore(t)
		mockArtifactStore.On("GetStateVersion", mock.Anything, mock.Anything).
			Return(reader, readErr)
		return &service{dbClient: &db.Client{}, artifactStore: mockArtifactStore}
	}

	t.Run("auth failure", func(t *testing.T) {
		ctx := t.Context()
		svc := &service{dbClient: &db.Client{}}

		_, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
		assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
	})

	t.Run("permission denied", func(t *testing.T) {
		ctx := setupCaller(t.Context(), t, errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
		svc := &service{dbClient: &db.Client{}}

		_, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})

	t.Run("artifact store error", func(t *testing.T) {
		ctx := setupCaller(t.Context(), t, nil)
		svc := setupService(t, nil, errors.New("store error", errors.WithErrorCode(errors.EInternal)))

		_, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("invalid state JSON", func(t *testing.T) {
		ctx := setupCaller(t.Context(), t, nil)
		svc := setupService(t, io.NopCloser(strings.NewReader("not-json")), nil)

		_, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("wrong state version", func(t *testing.T) {
		ctx := setupCaller(t.Context(), t, nil)
		state, _ := json.Marshal(stateV4{Version: 3})
		svc := setupService(t, io.NopCloser(strings.NewReader(string(state))), nil)

		_, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("resources", func(t *testing.T) {
		buildStateJSON := func(resources []resourceStateV4) string {
			state := stateV4{
				Version:   version4,
				Resources: resources,
			}
			b, _ := json.Marshal(state)
			return string(b)
		}

		tests := []struct {
			name         string
			resources    []resourceStateV4
			expectResult []*StateVersionResource
		}{
			{
				name: "valid resources",
				resources: []resourceStateV4{
					{
						Mode:           "managed",
						Type:           "aws_instance",
						Name:           "web",
						ProviderConfig: `provider["registry.terraform.io/hashicorp/aws"]`,
						Instances:      []instanceObjectStateV4{{}},
					},
				},
				expectResult: []*StateVersionResource{
					{
						Mode:     "managed",
						Type:     "aws_instance",
						Name:     "web",
						Provider: "registry.terraform.io/hashicorp/aws",
						Module:   "root",
					},
				},
			},
			{
				name: "resource with module",
				resources: []resourceStateV4{
					{
						Module:         "module.vpc",
						Mode:           "managed",
						Type:           "aws_subnet",
						Name:           "private",
						ProviderConfig: `provider["registry.terraform.io/hashicorp/aws"]`,
						Instances:      []instanceObjectStateV4{{}},
					},
				},
				expectResult: []*StateVersionResource{
					{
						Mode:     "managed",
						Type:     "aws_subnet",
						Name:     "private",
						Provider: "registry.terraform.io/hashicorp/aws",
						Module:   "module.vpc",
					},
				},
			},
			{
				name:         "no resources in state",
				resources:    []resourceStateV4{},
				expectResult: []*StateVersionResource{},
			},
			{
				name: "multiple resources",
				resources: []resourceStateV4{
					{
						Mode:           "managed",
						Type:           "aws_instance",
						Name:           "web",
						ProviderConfig: `provider["registry.terraform.io/hashicorp/aws"]`,
						Instances:      []instanceObjectStateV4{{}},
					},
					{
						Mode:           "data",
						Type:           "aws_ami",
						Name:           "ubuntu",
						ProviderConfig: `provider["registry.terraform.io/hashicorp/aws"]`,
						Instances:      []instanceObjectStateV4{{}},
					},
				},
				expectResult: []*StateVersionResource{
					{
						Mode:     "managed",
						Type:     "aws_instance",
						Name:     "web",
						Provider: "registry.terraform.io/hashicorp/aws",
						Module:   "root",
					},
					{
						Mode:     "data",
						Type:     "aws_ami",
						Name:     "ubuntu",
						Provider: "registry.terraform.io/hashicorp/aws",
						Module:   "root",
					},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				ctx := setupCaller(t.Context(), t, nil)
				stateJSON := buildStateJSON(test.resources)
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)

				result, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
				require.NoError(t, err)
				assert.Equal(t, test.expectResult, result.Resources)
			})
		}
	})

	t.Run("dependencies", func(t *testing.T) {
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

		tests := []struct {
			name            string
			mockSetup       func(ctx context.Context, t *testing.T) (context.Context, *service)
			expectResult    []*StateVersionDependency
			expectErrorCode errors.CodeType
		}{
			{
				name: "valid dependencies",
				mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
					ctx = setupCaller(ctx, t, nil)
					stateJSON := buildStateJSON(json.RawMessage(`{"full_path":"group/ws","state_version_id":"SV_abc","workspace_id":"WS_123"}`))
					svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
					return ctx, svc
				},
				expectResult: []*StateVersionDependency{
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
				expectResult: []*StateVersionDependency{},
			},
			{
				name: "partially nil attribute values are skipped",
				mockSetup: func(ctx context.Context, t *testing.T) (context.Context, *service) {
					ctx = setupCaller(ctx, t, nil)
					stateJSON := buildStateJSON(json.RawMessage(`{"full_path":"group/ws","state_version_id":null,"workspace_id":"WS_123"}`))
					svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)
					return ctx, svc
				},
				expectResult: []*StateVersionDependency{},
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
							{ProviderConfig: `provider["registry.terraform.io/hashicorp/aws"]`, Type: "aws_instance", Name: "test"},
						},
					})
					svc := setupService(t, io.NopCloser(strings.NewReader(string(state))), nil)
					return ctx, svc
				},
				expectResult: []*StateVersionDependency{},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				ctx, svc := test.mockSetup(t.Context(), t)

				result, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})

				if test.expectErrorCode != "" {
					assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
					return
				}

				require.NoError(t, err)
				assert.Equal(t, test.expectResult, result.Dependencies)
			})
		}
	})

	t.Run("check results", func(t *testing.T) {
		buildStateJSON := func(checkResults []checkResultV4) string {
			state := stateV4{
				Version:      version4,
				CheckResults: checkResults,
			}
			b, _ := json.Marshal(state)
			return string(b)
		}

		tests := []struct {
			name         string
			checkResults []checkResultV4
			expectResult []*corerun.CheckResult
		}{
			{
				name:         "all checks passing",
				checkResults: []checkResultV4{{ConfigAddr: "check.certificate", Status: "pass", Objects: []checkResultObjectV4{{ObjectAddr: "check.certificate", Status: "pass"}}}},
				expectResult: []*corerun.CheckResult{{Name: "check.certificate", Status: "pass", Objects: []corerun.CheckResultObject{{Address: "check.certificate", Status: "pass", FailureMessages: nil}}}},
			},
			{
				name:         "failed check with failure messages",
				checkResults: []checkResultV4{{ConfigAddr: "check.health_endpoint", Status: "fail", Objects: []checkResultObjectV4{{ObjectAddr: "check.health_endpoint", Status: "fail", FailureMessages: []string{"App returned 503"}}}}},
				expectResult: []*corerun.CheckResult{{Name: "check.health_endpoint", Status: "fail", Objects: []corerun.CheckResultObject{{Address: "check.health_endpoint", Status: "fail", FailureMessages: []string{"App returned 503"}}}}},
			},
			{
				name: "mixed statuses",
				checkResults: []checkResultV4{
					{ConfigAddr: "check.certificate", Status: "pass", Objects: []checkResultObjectV4{{ObjectAddr: "check.certificate", Status: "pass"}}},
					{ConfigAddr: "check.health", Status: "fail", Objects: []checkResultObjectV4{{ObjectAddr: "check.health", Status: "fail", FailureMessages: []string{"Service down"}}}},
					{ConfigAddr: "check.dns", Status: "error", Objects: []checkResultObjectV4{{ObjectAddr: "check.dns", Status: "error", FailureMessages: []string{"Could not resolve"}}}},
					{ConfigAddr: "check.pending", Status: "unknown", Objects: []checkResultObjectV4{{ObjectAddr: "check.pending", Status: "unknown"}}},
				},
				expectResult: []*corerun.CheckResult{
					{Name: "check.certificate", Status: "pass", Objects: []corerun.CheckResultObject{{Address: "check.certificate", Status: "pass", FailureMessages: nil}}},
					{Name: "check.health", Status: "fail", Objects: []corerun.CheckResultObject{{Address: "check.health", Status: "fail", FailureMessages: []string{"Service down"}}}},
					{Name: "check.dns", Status: "error", Objects: []corerun.CheckResultObject{{Address: "check.dns", Status: "error", FailureMessages: []string{"Could not resolve"}}}},
					{Name: "check.pending", Status: "unknown", Objects: []corerun.CheckResultObject{{Address: "check.pending", Status: "unknown", FailureMessages: nil}}},
				},
			},
			{
				name:         "no check results in state (null)",
				checkResults: nil,
				expectResult: []*corerun.CheckResult{},
			},
			{
				name:         "empty check results array",
				checkResults: []checkResultV4{},
				expectResult: []*corerun.CheckResult{},
			},
			{
				name: "check with multiple objects preserves per-object detail",
				checkResults: []checkResultV4{
					{ConfigAddr: "check.multi", Status: "fail", Objects: []checkResultObjectV4{
						{ObjectAddr: "check.multi[0]", Status: "fail", FailureMessages: []string{"First failed"}},
						{ObjectAddr: "check.multi[1]", Status: "fail", FailureMessages: []string{"Second failed"}},
					}},
				},
				expectResult: []*corerun.CheckResult{
					{Name: "check.multi", Status: "fail", Objects: []corerun.CheckResultObject{
						{Address: "check.multi[0]", Status: "fail", FailureMessages: []string{"First failed"}},
						{Address: "check.multi[1]", Status: "fail", FailureMessages: []string{"Second failed"}},
					}},
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				ctx := setupCaller(t.Context(), t, nil)
				stateJSON := buildStateJSON(test.checkResults)
				svc := setupService(t, io.NopCloser(strings.NewReader(stateJSON)), nil)

				result, err := svc.GetStateVersionInventory(ctx, &models.StateVersion{WorkspaceID: workspaceID})
				require.NoError(t, err)
				assert.Equal(t, test.expectResult, result.CheckResults)
			})
		}
	})
}

// TestUploadStateVersionJSON covers the second-request upload of a state version's JSON rendering:
// it is gated on the same permission as creating state, the object is written before the row is
// updated so a failed update leaves only a collectable orphan, and the recorded key is the one the
// artifact store minted.
func TestUploadStateVersionJSON(t *testing.T) {
	stateVersionID := "state-version-1"
	workspaceID := "workspace-1"
	jsonKey := "workspaces/workspace-1/state_versions/abc.json"
	testLogger, _ := logger.NewForTest()

	stored := func() *models.StateVersion {
		return &models.StateVersion{
			Metadata:    models.ResourceMetadata{ID: stateVersionID, Version: 1},
			WorkspaceID: workspaceID,
		}
	}

	setupCaller := func(ctx context.Context, t *testing.T, permErr error) context.Context {
		mockCaller := auth.MockCaller{}
		mockCaller.Test(t)
		mockCaller.On("RequirePermission", mock.Anything, models.CreateStateVersionPermission, mock.Anything).
			Return(permErr)
		return auth.WithCaller(ctx, &mockCaller)
	}

	t.Run("auth failure", func(t *testing.T) {
		svc := &service{dbClient: &db.Client{}}

		err := svc.UploadStateVersionJSON(t.Context(), stateVersionID, strings.NewReader("{}"))
		assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
	})

	t.Run("state version not found", func(t *testing.T) {
		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(nil, nil)

		// The caller is authenticated but no permission check should be reached: the workspace to check
		// against is not known until the state version is loaded.
		mockCaller := auth.MockCaller{}
		mockCaller.Test(t)
		ctx := auth.WithCaller(t.Context(), &mockCaller)

		svc := &service{dbClient: &db.Client{StateVersions: mockStateVersions}}

		err := svc.UploadStateVersionJSON(ctx, stateVersionID, strings.NewReader("{}"))
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("permission denied", func(t *testing.T) {
		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(stored(), nil)

		ctx := setupCaller(t.Context(), t, errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
		svc := &service{dbClient: &db.Client{StateVersions: mockStateVersions}}

		err := svc.UploadStateVersionJSON(ctx, stateVersionID, strings.NewReader("{}"))
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})

	t.Run("upload error", func(t *testing.T) {
		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(stored(), nil)

		mockArtifactStore := coreworkspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadStateVersionJSON", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, "", errors.New("store error", errors.WithErrorCode(errors.EInternal)))

		ctx := setupCaller(t.Context(), t, nil)
		svc := &service{dbClient: &db.Client{StateVersions: mockStateVersions}, artifactStore: mockArtifactStore}

		err := svc.UploadStateVersionJSON(ctx, stateVersionID, strings.NewReader("{}"))
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("success records the key and links the ref", func(t *testing.T) {
		var updated *models.StateVersion
		var linkedOwnerID string

		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(stored(), nil)
		mockStateVersions.On("UpdateStateVersion", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				updated = args.Get(1).(*models.StateVersion)
			}).Return(stored(), nil)

		mockArtifactStore := coreworkspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadStateVersionJSON", mock.Anything, mock.Anything, mock.Anything).
			Return(db.RetainObjectRefFunc(func(_ context.Context, ownerID string) error {
				linkedOwnerID = ownerID
				return nil
			}), jsonKey, nil)

		// The update and the ref link must land in one transaction that commits.
		mockTransactions := db.NewMockTransactions(t)
		mockTransactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
		mockTransactions.On("CommitTx", mock.Anything).Return(nil)

		ctx := setupCaller(t.Context(), t, nil)
		svc := &service{
			logger:        testLogger,
			dbClient:      &db.Client{StateVersions: mockStateVersions, Transactions: mockTransactions},
			artifactStore: mockArtifactStore,
		}

		require.NoError(t, svc.UploadStateVersionJSON(ctx, stateVersionID, strings.NewReader("{}")))
		require.NotNil(t, updated)
		require.NotNil(t, updated.JSONObjectStoreKey)
		assert.Equal(t, jsonKey, *updated.JSONObjectStoreKey)
		assert.Equal(t, stateVersionID, linkedOwnerID)
	})

	// A failure to link the ref must abort the whole thing rather than leaving a row pointing at an
	// object the janitor is free to collect, so the transaction is never committed.
	t.Run("retain ref failure rolls back", func(t *testing.T) {
		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(stored(), nil)
		mockStateVersions.On("UpdateStateVersion", mock.Anything, mock.Anything).Return(stored(), nil)

		mockArtifactStore := coreworkspace.NewMockArtifactStore(t)
		mockArtifactStore.On("UploadStateVersionJSON", mock.Anything, mock.Anything, mock.Anything).
			Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error {
				return errors.New("link failed", errors.WithErrorCode(errors.EInternal))
			}), jsonKey, nil)

		// CommitTx is deliberately not expected: the mock fails the test if it is called.
		mockTransactions := db.NewMockTransactions(t)
		mockTransactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

		ctx := setupCaller(t.Context(), t, nil)
		svc := &service{
			logger:        testLogger,
			dbClient:      &db.Client{StateVersions: mockStateVersions, Transactions: mockTransactions},
			artifactStore: mockArtifactStore,
		}

		err := svc.UploadStateVersionJSON(ctx, stateVersionID, strings.NewReader("{}"))
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}

// TestGetStateVersionJSONContent verifies that a state version with no rendering stored reports
// ENotFound rather than reaching the artifact store with an empty key — that is what lets the job
// executor distinguish "no rendering" from a storage failure and evaluate policies without it.
func TestGetStateVersionJSONContent(t *testing.T) {
	stateVersionID := "state-version-1"
	workspaceID := "workspace-1"
	jsonKey := "workspaces/workspace-1/state_versions/abc.json"

	setupCaller := func(ctx context.Context, t *testing.T, permErr error) context.Context {
		mockCaller := auth.MockCaller{}
		mockCaller.Test(t)
		mockCaller.On("RequirePermission", mock.Anything, models.ViewStateVersionDataPermission, mock.Anything).
			Return(permErr)
		return auth.WithCaller(ctx, &mockCaller)
	}

	t.Run("auth failure", func(t *testing.T) {
		svc := &service{dbClient: &db.Client{}}

		_, err := svc.GetStateVersionJSONContent(t.Context(), stateVersionID)
		assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
	})

	t.Run("no rendering stored", func(t *testing.T) {
		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(&models.StateVersion{
			Metadata:    models.ResourceMetadata{ID: stateVersionID},
			WorkspaceID: workspaceID,
		}, nil)

		ctx := setupCaller(t.Context(), t, nil)
		// No artifact store is wired: reaching it would be the bug this case guards against.
		svc := &service{dbClient: &db.Client{StateVersions: mockStateVersions}}

		_, err := svc.GetStateVersionJSONContent(ctx, stateVersionID)
		assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	})

	t.Run("success", func(t *testing.T) {
		mockStateVersions := db.NewMockStateVersions(t)
		mockStateVersions.On("GetStateVersionByID", mock.Anything, stateVersionID).Return(&models.StateVersion{
			Metadata:           models.ResourceMetadata{ID: stateVersionID},
			WorkspaceID:        workspaceID,
			JSONObjectStoreKey: &jsonKey,
		}, nil)

		mockArtifactStore := coreworkspace.NewMockArtifactStore(t)
		mockArtifactStore.On("GetStateVersionJSON", mock.Anything, mock.Anything).
			Return(io.NopCloser(strings.NewReader(`{"format_version":"1.0"}`)), nil)

		ctx := setupCaller(t.Context(), t, nil)
		svc := &service{dbClient: &db.Client{StateVersions: mockStateVersions}, artifactStore: mockArtifactStore}

		reader, err := svc.GetStateVersionJSONContent(ctx, stateVersionID)
		require.NoError(t, err)
		defer reader.Close()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.JSONEq(t, `{"format_version":"1.0"}`, string(content))
	})
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

	// Helpers for the output-limit test cases.
	//
	// smallValue is a small, valid JSON output value; oversizedValue is a JSON
	// string whose raw form exceeds the per-output size limit. buildState encodes
	// a stateV4 document containing the given outputs (name -> raw JSON value).
	smallValue := `"small"`
	oversizedValue := `"` + strings.Repeat("a", maxStateVersionOutputSizeBytes+1) + `"`
	buildState := func(outputs map[string]string) []byte {
		parts := make([]string, 0, len(outputs))
		for name, val := range outputs {
			parts = append(parts, fmt.Sprintf(`%q: {"value": %s, "type": "string"}`, name, val))
		}
		return buildEncodedData(fmt.Sprintf(`{"version": 4, "outputs": {%s}}`, strings.Join(parts, ",")))
	}

	// outputStateVersion is the created state version returned for the
	// output-limit cases (they all need a non-nil created version so outputs can
	// be attached and the state can be uploaded).
	outputStateVersion := &models.StateVersion{
		Metadata: models.ResourceMetadata{
			CreationTimestamp: &currentTime,
			ID:                stateVersionID,
		},
		WorkspaceID: workspaceID,
		RunID:       &runID,
	}

	// The rendering cases each need their own model: the service records the rendering's object store
	// key on the instance it is handed, so sharing one would leak that mutation between subtests.
	renderingToCreate := &models.StateVersion{WorkspaceID: workspaceID, RunID: &runID}
	renderingUploadFailToCreate := &models.StateVersion{WorkspaceID: workspaceID, RunID: &runID}
	goodJSONData := buildEncodedData(`{"format_version": "1.0"}`)

	type testCase struct {
		authFail                 bool
		workspacePermissionError error
		dataUnmarshalError       error
		uploadError              error
		jsonUploadError          error
		linkRefErr               error
		createError              error
		dataDecodeError          error
		outputLimitLookupErr     error
		toCreate                 *models.StateVersion
		injectCreated            *models.StateVersion
		expectResult             *models.StateVersion
		data                     []byte
		// jsonData is the optional base64 "terraform show -json" rendering sent with the state.
		jsonData *string
		name     string
		// expectJSONKey is whether the row handed to the database carries the rendering's object store
		// key, which is what makes the rendering land in the same INSERT as the state version.
		expectJSONKey bool
		// expectNoWrites asserts the request was rejected before anything was uploaded or a transaction
		// was opened.
		expectNoWrites        bool
		expectErrorCode       errors.CodeType
		limit                 int
		injectSVsPerWorkspace int32
		outputLimit           int
		expectStoredOutputs   []string
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
			name:     "retainFn error is propagated",
			toCreate: toCreate,
			data:     goodData,
			injectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					CreationTimestamp: &currentTime,
					ID:                stateVersionID,
				},
			},
			limit:                 4,
			injectSVsPerWorkspace: 4,
			linkRefErr:            errors.New("link failed", errors.WithErrorCode(errors.EInternal)),
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
		{
			name:     "output exceeding size limit is skipped while smaller outputs are still stored",
			toCreate: toCreate,
			data: buildState(map[string]string{
				"big":   oversizedValue,
				"small": smallValue,
			}),
			injectCreated:         outputStateVersion,
			limit:                 1000,
			injectSVsPerWorkspace: 0,
			outputLimit:           400,
			expectStoredOutputs:   []string{"small"},
			expectErrorCode:       errors.EInvalid,
		},
		{
			name:     "outputs exceeding count limit are truncated deterministically by sorted name",
			toCreate: toCreate,
			data: buildState(map[string]string{
				"a": smallValue,
				"b": smallValue,
				"c": smallValue,
				"d": smallValue,
			}),
			injectCreated:         outputStateVersion,
			limit:                 1000,
			injectSVsPerWorkspace: 0,
			outputLimit:           2,
			expectStoredOutputs:   []string{"a", "b"},
			expectErrorCode:       errors.EInvalid,
		},
		{
			name:     "fail-safe: all outputs stored when the output limit lookup fails",
			toCreate: toCreate,
			data: buildState(map[string]string{
				"a": smallValue,
				"b": smallValue,
				"c": smallValue,
			}),
			injectCreated:         outputStateVersion,
			limit:                 1000,
			injectSVsPerWorkspace: 0,
			outputLimitLookupErr:  errors.New("db unavailable", errors.WithErrorCode(errors.EInternal)),
			expectStoredOutputs:   []string{"a", "b", "c"},
			expectResult:          outputStateVersion,
		},
		{
			// A rendering supplied with the state is uploaded and its key set on the row before the
			// insert, so the state version is created with the rendering already attached.
			name:                  "json rendering is stored with the state version",
			toCreate:              renderingToCreate,
			data:                  goodData,
			jsonData:              ptr.String(string(goodJSONData)),
			injectCreated:         outputStateVersion,
			limit:                 1000,
			injectSVsPerWorkspace: 0,
			expectJSONKey:         true,
			expectResult:          outputStateVersion,
		},
		{
			// Decoding happens before any write, so a malformed rendering costs nothing: no object is
			// uploaded and no transaction is opened.
			name:            "malformed json rendering is rejected before anything is written",
			toCreate:        toCreate,
			data:            goodData,
			jsonData:        ptr.String("not-base64!!"),
			expectNoWrites:  true,
			expectErrorCode: errors.EInvalid,
		},
		{
			// The rendering is uploaded before the transaction, so a storage failure leaves the state
			// version uncreated rather than committing it without the rendering the caller asked for.
			name:            "a failed json rendering upload fails the create",
			toCreate:        renderingUploadFailToCreate,
			data:            goodData,
			jsonData:        ptr.String(string(goodJSONData)),
			jsonUploadError: errors.New("object store unavailable", errors.WithErrorCode(errors.EInternal)),
			expectNoWrites:  true,
			expectErrorCode: errors.EInternal,
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
			// The state-versions-per-workspace-per-time-period check and the
			// output-count check both call GetResourceLimit; key the mock by name
			// so the fail-safe case can fail only the output lookup.
			mockResourceLimits.On("GetResourceLimit", mock.Anything, string(limits.ResourceLimitStateVersionsPerWorkspacePerTimePeriod)).
				Return(&models.ResourceLimit{Value: test.limit}, nil).Maybe()
			mockResourceLimits.On("GetResourceLimit", mock.Anything, string(limits.ResourceLimitOutputsPerStateVersion)).
				Return(&models.ResourceLimit{Value: test.outputLimit}, test.outputLimitLookupErr).Maybe()

			var storedOutputNames []string
			mockStateVersionOutputs := db.NewMockStateVersionOutputs(t)
			mockStateVersionOutputs.On("CreateStateVersionOutput", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					o := args.Get(1).(*models.StateVersionOutput)
					storedOutputNames = append(storedOutputNames, o.Name)
				}).
				Return(&models.StateVersionOutput{}, nil).Maybe()

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

			mockObjectStoreRefs := db.NewMockObjectStoreRefs(t)
			mockObjectStoreRefs.On("LinkRef", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(test.linkRefErr).Maybe()

			mockArtifactStore.On("UploadStateVersion", mock.Anything, mock.Anything, mock.Anything).
				Return(db.RetainObjectRefFunc(func(ctx context.Context, ownerID string) error {
					return mockObjectStoreRefs.LinkRef(ctx, "workspaces/ws/state_versions/uuid", db.ObjectStoreRefOwnerStateVersion, ownerID)
				}), "workspaces/ws/state_versions/uuid", test.uploadError).Maybe()

			mockArtifactStore.On("UploadStateVersionJSON", mock.Anything, mock.Anything, mock.Anything).
				Return(db.RetainObjectRefFunc(func(ctx context.Context, ownerID string) error {
					return mockObjectStoreRefs.LinkRef(ctx, "workspaces/ws/state_versions/uuid.json", db.ObjectStoreRefOwnerStateVersion, ownerID)
				}), "workspaces/ws/state_versions/uuid.json", test.jsonUploadError).Maybe()

			testLogger, _ := logger.NewForTest()
			dbClient := &db.Client{
				Transactions:        mockTransactions,
				StateVersions:       mockStateVersions,
				StateVersionOutputs: mockStateVersionOutputs,
				ResourceLimits:      mockResourceLimits,
				Workspaces:          mockWorkspaces,
			}

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), &mockArtifactStore, nil, "", nil)

			if !test.authFail {
				ctx = auth.WithCaller(ctx, &mockCaller)
			}

			testDataString := string(test.data)
			result, err := service.CreateStateVersion(ctx, test.toCreate, testDataString, test.jsonData)

			if test.expectJSONKey {
				require.NotNil(t, test.toCreate.JSONObjectStoreKey)
				assert.Equal(t, "workspaces/ws/state_versions/uuid.json", *test.toCreate.JSONObjectStoreKey)
				// The key and the object store ref have to be written together: a key with no ref would
				// let the janitor collect an object the row still points at.
				mockObjectStoreRefs.AssertCalled(t, "LinkRef", mock.Anything,
					"workspaces/ws/state_versions/uuid.json", db.ObjectStoreRefOwnerStateVersion, mock.Anything)
			}

			if test.expectNoWrites {
				mockArtifactStore.AssertNotCalled(t, "UploadStateVersion", mock.Anything, mock.Anything, mock.Anything)
				mockTransactions.AssertNotCalled(t, "BeginTx", mock.Anything)
			}

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else {
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, test.expectResult, result)
			}

			// For output-limit cases, verify exactly which outputs were persisted.
			// This runs even when an error is expected, because size/count
			// violations are partial failures: the state is still saved and the
			// non-violating outputs are still stored.
			if test.expectStoredOutputs != nil {
				sort.Strings(storedOutputNames)
				assert.Equal(t, test.expectStoredOutputs, storedOutputNames)

				// This is the core safety property the partial-failure design
				// depends on: the full state blob must still be uploaded even
				// when some outputs are rejected, since the infrastructure
				// changes it describes have already been applied. Assert this
				// explicitly rather than relying on the mock's .Maybe() default,
				// so a regression that skips or short-circuits the upload on a
				// limit violation is caught here instead of only in production.
				mockArtifactStore.AssertCalled(t, "UploadStateVersion", mock.Anything, mock.Anything, mock.Anything)
			}
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
		isCallerOwnerOfParent    bool
	}{
		{
			name:                     "successful move",
			inputWorkspace:           testWorkspace,
			newParentID:              newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			isCallerOwnerOfParent:    true,
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
			// Migrating a workspace in introduces principals the destination's administrator never
			// approved and changes path-derived output visibility, so it requires the permission that
			// governs conferring access at the destination.
			name:                     "caller cannot create namespace memberships in new parent",
			inputWorkspace:           testWorkspace,
			newParentID:              newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			isCallerOwnerOfParent:    false,
			expectErrorCode:          errors.EForbidden,
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
			isCallerOwnerOfParent:    true,
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

			var workspaceAccessError, parentAccessError, parentMembershipAccessError error
			if !test.isGroupOwner {
				workspaceAccessError = errors.New("test user is not owner of workspace being moved", errors.WithErrorCode(errors.EForbidden))
			}
			if !test.isCallerDeployerOfParent {
				parentAccessError = errors.New("test user is not deployer of old or new parent", errors.WithErrorCode(errors.EForbidden))
			}
			if !test.isCallerOwnerOfParent {
				parentMembershipAccessError = errors.New("test user cannot create namespace memberships in new parent", errors.WithErrorCode(errors.EForbidden))
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

			perms = []models.Permission{models.CreateNamespaceMembershipPermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(parentMembershipAccessError)

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

func TestGetOutputVisibilitySetting(t *testing.T) {
	workspace := models.Workspace{
		Metadata: models.ResourceMetadata{ID: "ws-1"},
	}
	// Test cases
	tests := []struct {
		expectSetting *namespace.OutputVisibilitySetting
		name          string
		authError     error
		expectErrCode errors.CodeType
	}{
		{
			name: "get setting",
			expectSetting: &namespace.OutputVisibilitySetting{
				Value: models.OutputVisibilityDirectGroupOnly,
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

			mockInheritedSettingsResolver.On("GetOutputVisibility", mock.Anything, &workspace).Return(test.expectSetting, nil).Maybe()

			svc := service{
				logger:                    testLogger,
				inheritedSettingsResolver: mockInheritedSettingsResolver,
			}

			setting, err := svc.GetOutputVisibilitySetting(auth.WithCaller(ctx, mockCaller), &workspace)

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

// TestSetWorkspaceRoleBinding_CreateEscalation covers the escalation gate: setting a binding must
// check BOTH the WorkspaceRoleBinding permission AND the subset of the bound role's permissions, and
// BOTH checks must be evaluated against the workspace's PARENT namespace, never the workspace itself.
func TestSetWorkspaceRoleBinding_CreateEscalation(t *testing.T) {
	workspaceID := "ws-1"
	groupID := "group-1"
	groupPath := "group-1-path"
	roleID := "role-1"

	testWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: workspaceID},
		GroupID:  groupID,
		FullPath: groupPath + "/ws-1",
	}

	// requireEffectivePermissionSuperset resolves the group by ID to get its FullPath for the
	// GetEffectivePermissions check below -- unrelated to (and not removed by) eliminating the
	// separate, now-unnecessary GetGroupByID lookup that createOrUpdateWorkspaceRoleBinding used to
	// make solely to build the activity event's namespace path.
	testGroup := &models.Group{
		Metadata: models.ResourceMetadata{ID: groupID},
		FullPath: groupPath,
	}

	deployerPerms, ok := models.DeployerRoleID.Permissions()
	require.True(t, ok)

	tests := []struct {
		name string
		// effectivePermissions simulates what GetEffectivePermissions returns for the caller at the
		// group's namespace path.
		effectivePermissions map[string]models.Permission
		hasBindingPermission bool
		isAdmin              bool
		expectErrorCode      errors.CodeType
	}{
		{
			name: "owner with the full deployer permission set can bind deployer",
			effectivePermissions: func() map[string]models.Permission {
				m := map[string]models.Permission{}
				for _, p := range deployerPerms {
					m[p.String()] = p
				}
				return m
			}(),
			hasBindingPermission: true,
		},
		{
			name:                 "caller without the binding permission is denied even if they hold the role's permissions",
			hasBindingPermission: false,
			expectErrorCode:      errors.EForbidden,
		},
		{
			name: "caller holding the binding permission but missing a permission in the role is denied",
			effectivePermissions: map[string]models.Permission{
				models.ViewWorkspacePermission.String(): models.ViewWorkspacePermission,
				// Missing the rest of deployer's permission set.
			},
			hasBindingPermission: true,
			expectErrorCode:      errors.EForbidden,
		},
		{
			// requireEffectivePermissionSuperset must use GTE, not an exact Permission.String()
			// match: a caller holding variable:update (which GTEs variable:view, see
			// Permission.GTE) already implies variable:view even though it is not literally in
			// their held set. Deployer's permission set includes variable:view, so binding it must
			// be allowed by variable:update alone -- an exact-match check would wrongly deny this.
			name: "caller holding a superseding permission (variable:update) satisfies a required lesser permission (variable:view) via GTE",
			effectivePermissions: func() map[string]models.Permission {
				m := map[string]models.Permission{}
				for _, p := range deployerPerms {
					if p.String() == models.ViewVariablePermission.String() {
						// Deliberately omit the literal variable:view permission, replacing its
						// coverage with variable:update, which GTEs it.
						continue
					}
					m[p.String()] = p
				}
				m[models.UpdateVariablePermission.String()] = models.UpdateVariablePermission
				return m
			}(),
			hasBindingPermission: true,
		},
		{
			// An admin in admin mode is granted the binding permission outright (see
			// UserCaller.RequirePermission), and requireEffectivePermissionSuperset skips the
			// subset check entirely for such a caller -- it must not deny an admin with zero
			// namespace memberships, since GetNamespacePermissions would otherwise (correctly)
			// report that they hold nothing.
			name:                 "admin with admin mode active can bind deployer despite holding no memberships",
			hasBindingPermission: true,
			isAdmin:              true,
			effectivePermissions: map[string]models.Permission{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockGroups := db.NewMockGroups(t)
			mockRoles := db.NewMockRoles(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)
			mockUsers := db.NewMockUsers(t)
			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := db.NewMockActivityEvents(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(testWorkspace, nil)
			mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, workspaceID).Return(nil, nil)

			var adminUser *models.User
			if test.isAdmin {
				future := time.Now().Add(time.Hour)
				adminUser = &models.User{
					Metadata:            models.ResourceMetadata{ID: "user-1"},
					Admin:               true,
					AdminModeExpiration: &future,
				}
				// IsAdminModeActivated re-fetches the user to get the latest admin mode expiration.
				mockUsers.On("GetUserByID", mock.Anything, "user-1").Return(adminUser, nil)
			}

			// Admin mode grants the binding permission outright, so the authorizer is never
			// consulted for RequireAccess in that case.
			if !test.isAdmin {
				if test.hasBindingPermission {
					mockAuthorizer.On("RequireAccess", mock.Anything,
						[]models.Permission{models.CreateWorkspaceRoleBindingPermission}, mock.Anything).Return(nil)
				} else {
					mockAuthorizer.On("RequireAccess", mock.Anything,
						[]models.Permission{models.CreateWorkspaceRoleBindingPermission}, mock.Anything).
						Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
				}
			}

			if test.hasBindingPermission {
				role := &models.Role{}
				role.SetPermissions(deployerPerms)
				mockRoles.On("GetRoleByID", mock.Anything, roleID).Return(role, nil)
				if !test.isAdmin {
					mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(testGroup, nil)
					mockAuthorizer.On("GetEffectivePermissions", mock.Anything, groupPath).
						Return(test.effectivePermissions, nil)
				}
			}

			if test.expectErrorCode == "" {
				mockTransactions.On("BeginTx", mock.Anything).Return(func(c context.Context) context.Context { return c }, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockBindings.On("CreateWorkspaceRoleBinding", mock.Anything, mock.Anything).
					Return(&models.WorkspaceRoleBinding{
						Metadata:    models.ResourceMetadata{ID: "binding-1"},
						WorkspaceID: workspaceID,
						RoleID:      roleID,
					}, nil)
				// The CREATE event must target the new binding's own ID (WORKSPACE_ROLE_BINDING),
				// not the workspace.
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(event *models.ActivityEvent) bool {
					return event.TargetType == models.TargetWorkspaceRoleBinding &&
						event.TargetID == "binding-1" &&
						event.Action == models.ActionCreate
				})).Return(&models.ActivityEvent{}, nil)
			}

			dbClient := &db.Client{
				Workspaces:            mockWorkspaces,
				Groups:                mockGroups,
				Roles:                 mockRoles,
				WorkspaceRoleBindings: mockBindings,
				Users:                 mockUsers,
				Transactions:          mockTransactions,
				ActivityEvents:        mockActivityEvents,
			}

			testLogger, _ := logger.NewForTest()

			svc := &service{
				dbClient: dbClient,
				logger:   testLogger,
			}

			user := adminUser
			if user == nil {
				user = &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}}
			}

			caller := auth.NewUserCaller(
				user,
				mockAuthorizer,
				dbClient,
				mockMaintenanceMonitor,
				nil,
			)

			_, err := svc.SetWorkspaceRoleBinding(auth.WithCaller(ctx, caller), &SetWorkspaceRoleBindingInput{
				WorkspaceID: workspaceID,
				RoleID:      &roleID,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestSetWorkspaceRoleBinding_ChecksParentNotWorkspace guards the bug the design depends on not
// shipping: both authorization checks (the binding permission, and the subset check via
// GetEffectivePermissions) must be scoped to the workspace's PARENT group, not the workspace itself.
// If either checked the workspace instead, a caller whose membership is scoped to only the workspace
// could confer authority over the parent group — a namespace they may have no access to at all.
//
// This is proven by mockAuthorizer.GetEffectivePermissions being expected with groupPath and NOT
// with the workspace's own FullPath: if the implementation passed the workspace's path instead, this
// expectation would go unmet and testify would fail the test for an unexpected call.
func TestSetWorkspaceRoleBinding_ChecksParentNotWorkspace(t *testing.T) {
	workspaceID := "ws-1"
	groupID := "group-1"
	groupPath := "group-1-path"
	roleID := "role-1"

	testWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: workspaceID},
		GroupID:  groupID,
		FullPath: groupPath + "/ws-1",
	}

	// requireEffectivePermissionSuperset resolves the group by ID to get its FullPath for the
	// GetEffectivePermissions check below.
	testGroup := &models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}

	viewerPerms, ok := models.ViewerRoleID.Permissions()
	require.True(t, ok)

	ctx := context.Background()

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockGroups := db.NewMockGroups(t)
	mockRoles := db.NewMockRoles(t)
	mockBindings := db.NewMockWorkspaceRoleBindings(t)
	mockAuthorizer := auth.NewMockAuthorizer(t)
	mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
	mockTransactions := db.NewMockTransactions(t)
	mockActivityEvents := db.NewMockActivityEvents(t)

	mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(testWorkspace, nil)
	mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, workspaceID).Return(nil, nil)
	mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(testGroup, nil)

	// The permission check is scoped via WithGroupID, but a mocked Authorizer cannot distinguish
	// which functional option produced the constraint, so this expectation alone is not proof of
	// scope. The proof is the GetEffectivePermissions call below.
	mockAuthorizer.On("RequireAccess", mock.Anything,
		[]models.Permission{models.CreateWorkspaceRoleBindingPermission}, mock.Anything).Return(nil)

	role := &models.Role{}
	role.SetPermissions(viewerPerms)
	mockRoles.On("GetRoleByID", mock.Anything, roleID).Return(role, nil)

	effective := map[string]models.Permission{}
	for _, p := range viewerPerms {
		effective[p.String()] = p
	}
	// Expect the call with the GROUP's path. Any other argument (in particular the workspace's own
	// FullPath) does not match this expectation and testify fails the test.
	mockAuthorizer.On("GetEffectivePermissions", mock.Anything, groupPath).Return(effective, nil)

	mockTransactions.On("BeginTx", mock.Anything).Return(func(c context.Context) context.Context { return c }, nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)
	mockBindings.On("CreateWorkspaceRoleBinding", mock.Anything, mock.Anything).
		Return(&models.WorkspaceRoleBinding{
			Metadata:    models.ResourceMetadata{ID: "binding-1"},
			WorkspaceID: workspaceID,
			RoleID:      roleID,
		}, nil)
	mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
		Return(&models.ActivityEvent{}, nil)

	dbClient := &db.Client{
		Workspaces:            mockWorkspaces,
		Groups:                mockGroups,
		Roles:                 mockRoles,
		WorkspaceRoleBindings: mockBindings,
		Transactions:          mockTransactions,
		ActivityEvents:        mockActivityEvents,
	}

	testLogger, _ := logger.NewForTest()

	_, err := (&service{dbClient: dbClient, logger: testLogger}).SetWorkspaceRoleBinding(
		auth.WithCaller(ctx, auth.NewUserCaller(
			&models.User{Metadata: models.ResourceMetadata{ID: "user-1"}},
			mockAuthorizer,
			dbClient,
			mockMaintenanceMonitor,
			nil,
		)),
		&SetWorkspaceRoleBindingInput{WorkspaceID: workspaceID, RoleID: &roleID},
	)

	require.NoError(t, err)
}

// TestSetWorkspaceRoleBinding_Remove covers removal: it is gated only on
// DeleteWorkspaceRoleBindingPermission at the parent namespace, since removing a binding never
// grants anything and so needs no subset check.
func TestSetWorkspaceRoleBinding_Remove(t *testing.T) {
	workspaceID := "ws-1"
	groupID := "group-1"
	groupPath := "group-1-path"

	testWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: workspaceID},
		GroupID:  groupID,
		FullPath: groupPath + "/ws-1",
	}

	existingBinding := &models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: "binding-1"},
		WorkspaceID: workspaceID,
		RoleID:      "role-1",
	}

	existingRole := &models.Role{
		Metadata: models.ResourceMetadata{ID: "role-1"},
		Name:     "deployer",
	}

	tests := []struct {
		name            string
		hasPermission   bool
		hasExisting     bool
		expectErrorCode errors.CodeType
	}{
		{name: "caller with permission removes an existing binding", hasPermission: true, hasExisting: true},
		{name: "caller without permission is denied", hasPermission: false, hasExisting: true, expectErrorCode: errors.EForbidden},
		{name: "removing when no binding exists is a not-found", hasPermission: true, hasExisting: false, expectErrorCode: errors.ENotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockRoles := db.NewMockRoles(t)
			mockBindings := db.NewMockWorkspaceRoleBindings(t)
			mockAuthorizer := auth.NewMockAuthorizer(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := db.NewMockActivityEvents(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(testWorkspace, nil)

			var existing *models.WorkspaceRoleBinding
			if test.hasExisting {
				existing = existingBinding
			}
			mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, workspaceID).Return(existing, nil)

			if test.hasExisting {
				if test.hasPermission {
					mockAuthorizer.On("RequireAccess", mock.Anything,
						[]models.Permission{models.DeleteWorkspaceRoleBindingPermission}, mock.Anything).Return(nil)
				} else {
					mockAuthorizer.On("RequireAccess", mock.Anything,
						[]models.Permission{models.DeleteWorkspaceRoleBindingPermission}, mock.Anything).
						Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
				}
			}

			if test.expectErrorCode == "" {
				mockRoles.On("GetRoleByID", mock.Anything, existingBinding.RoleID).Return(existingRole, nil)
				mockTransactions.On("BeginTx", mock.Anything).Return(func(c context.Context) context.Context { return c }, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockBindings.On("DeleteWorkspaceRoleBinding", mock.Anything, existingBinding).Return(nil)
				// Removing a binding is recorded as a DeleteChildResource event against the
				// WORKSPACE, not the binding itself — the binding no longer exists once deleted.
				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(event *models.ActivityEvent) bool {
					if event.TargetType != models.TargetWorkspace ||
						event.TargetID != workspaceID ||
						event.Action != models.ActionDeleteChildResource {
						return false
					}
					var payload models.ActivityEventDeleteChildResourcePayload
					if err := json.Unmarshal(event.Payload, &payload); err != nil {
						return false
					}
					return payload.Name == existingRole.Name &&
						payload.ID == existingBinding.Metadata.ID &&
						payload.Type == string(models.TargetWorkspaceRoleBinding)
				})).Return(&models.ActivityEvent{}, nil)
			}

			dbClient := &db.Client{
				Workspaces:            mockWorkspaces,
				Roles:                 mockRoles,
				WorkspaceRoleBindings: mockBindings,
				Transactions:          mockTransactions,
				ActivityEvents:        mockActivityEvents,
			}

			testLogger, _ := logger.NewForTest()

			svc := &service{
				dbClient: dbClient,
				logger:   testLogger,
			}

			caller := auth.NewUserCaller(
				&models.User{Metadata: models.ResourceMetadata{ID: "user-1"}},
				mockAuthorizer,
				dbClient,
				mockMaintenanceMonitor,
				nil,
			)

			_, err := svc.SetWorkspaceRoleBinding(auth.WithCaller(ctx, caller), &SetWorkspaceRoleBindingInput{
				WorkspaceID: workspaceID,
				RoleID:      nil,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestSetWorkspaceRoleBinding_UpdateRole covers changing an existing binding's role: the activity
// event must still target the binding's own ID (WORKSPACE_ROLE_BINDING), and must use ActionUpdate
// (not ActionCreate) with both the previous and new role IDs recorded in the payload, since this is
// a change to an existing binding, not the binding coming into existence for the first time.
func TestSetWorkspaceRoleBinding_UpdateRole(t *testing.T) {
	workspaceID := "ws-1"
	groupID := "group-1"
	groupPath := "group-1-path"
	newRoleID := "role-2"

	testWorkspace := &models.Workspace{
		Metadata: models.ResourceMetadata{ID: workspaceID},
		GroupID:  groupID,
		FullPath: groupPath + "/ws-1",
	}

	// requireEffectivePermissionSuperset resolves the group by ID to get its FullPath for the
	// GetEffectivePermissions check below.
	testGroup := &models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}

	existingBinding := &models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: "binding-1"},
		WorkspaceID: workspaceID,
		RoleID:      "role-1",
	}

	deployerPerms, ok := models.DeployerRoleID.Permissions()
	require.True(t, ok)
	effective := map[string]models.Permission{}
	for _, p := range deployerPerms {
		effective[p.String()] = p
	}

	ctx := context.Background()

	mockWorkspaces := db.NewMockWorkspaces(t)
	mockGroups := db.NewMockGroups(t)
	mockRoles := db.NewMockRoles(t)
	mockBindings := db.NewMockWorkspaceRoleBindings(t)
	mockAuthorizer := auth.NewMockAuthorizer(t)
	mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
	mockTransactions := db.NewMockTransactions(t)
	mockActivityEvents := db.NewMockActivityEvents(t)

	mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
	mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(testWorkspace, nil)
	mockBindings.On("GetWorkspaceRoleBindingByWorkspaceID", mock.Anything, workspaceID).Return(existingBinding, nil)
	mockGroups.On("GetGroupByID", mock.Anything, groupID).Return(testGroup, nil)

	mockAuthorizer.On("RequireAccess", mock.Anything,
		[]models.Permission{models.UpdateWorkspaceRoleBindingPermission}, mock.Anything).Return(nil)

	role := &models.Role{}
	role.SetPermissions(deployerPerms)
	mockRoles.On("GetRoleByID", mock.Anything, newRoleID).Return(role, nil)
	mockAuthorizer.On("GetEffectivePermissions", mock.Anything, groupPath).Return(effective, nil)

	mockTransactions.On("BeginTx", mock.Anything).Return(func(c context.Context) context.Context { return c }, nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)

	updatedBinding := &models.WorkspaceRoleBinding{
		Metadata:    existingBinding.Metadata,
		WorkspaceID: workspaceID,
		RoleID:      newRoleID,
	}
	mockBindings.On("UpdateWorkspaceRoleBinding", mock.Anything, mock.Anything).Return(updatedBinding, nil)

	mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(event *models.ActivityEvent) bool {
		if event.TargetType != models.TargetWorkspaceRoleBinding ||
			event.TargetID != existingBinding.Metadata.ID ||
			event.Action != models.ActionUpdate {
			return false
		}
		var payload models.ActivityEventSetWorkspaceRoleBindingPayload
		require.NoError(t, json.Unmarshal(event.Payload, &payload))
		return payload.PreviousRoleID == "role-1" && payload.NewRoleID == newRoleID
	})).Return(&models.ActivityEvent{}, nil)

	dbClient := &db.Client{
		Workspaces:            mockWorkspaces,
		Groups:                mockGroups,
		Roles:                 mockRoles,
		WorkspaceRoleBindings: mockBindings,
		Transactions:          mockTransactions,
		ActivityEvents:        mockActivityEvents,
	}

	testLogger, _ := logger.NewForTest()

	_, err := (&service{dbClient: dbClient, logger: testLogger}).SetWorkspaceRoleBinding(
		auth.WithCaller(ctx, auth.NewUserCaller(
			&models.User{Metadata: models.ResourceMetadata{ID: "user-1"}},
			mockAuthorizer,
			dbClient,
			mockMaintenanceMonitor,
			nil,
		)),
		&SetWorkspaceRoleBindingInput{WorkspaceID: workspaceID, RoleID: &newRoleID},
	)

	require.NoError(t, err)
}

// TestGetWorkspaceRoleBindingsByWorkspaceIDs covers the batch loader's backing service method: it
// must return only the bindings that exist (workspaces with none are simply absent, not an error),
// and it must still enforce ViewWorkspaceRoleBindingPermission per result.
func TestGetWorkspaceRoleBindingsByWorkspaceIDs(t *testing.T) {
	bindingA := models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: "binding-a"},
		WorkspaceID: "ws-a",
		RoleID:      "role-1",
	}
	bindingB := models.WorkspaceRoleBinding{
		Metadata:    models.ResourceMetadata{ID: "binding-b"},
		WorkspaceID: "ws-b",
		RoleID:      "role-1",
	}

	t.Run("returns only workspaces that have a binding", func(t *testing.T) {
		ctx := context.Background()

		mockBindings := db.NewMockWorkspaceRoleBindings(t)
		mockWorkspaces := db.NewMockWorkspaces(t)
		mockAuthorizer := auth.NewMockAuthorizer(t)
		mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

		mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
		mockBindings.On("GetWorkspaceRoleBindings", mock.Anything, &db.GetWorkspaceRoleBindingsInput{
			Filter: &db.WorkspaceRoleBindingFilter{WorkspaceIDs: []string{"ws-a", "ws-b", "ws-c"}},
		}).Return(&db.WorkspaceRoleBindingsResult{
			WorkspaceRoleBindings: []models.WorkspaceRoleBinding{bindingA, bindingB},
		}, nil)
		mockWorkspaces.On("GetWorkspaces", mock.Anything, &db.GetWorkspacesInput{
			Filter: &db.WorkspaceFilter{WorkspaceIDs: []string{"ws-a", "ws-b"}},
		}).Return(&db.WorkspacesResult{
			Workspaces: []models.Workspace{
				{Metadata: models.ResourceMetadata{ID: "ws-a"}, FullPath: "group-a/ws-a"},
				{Metadata: models.ResourceMetadata{ID: "ws-b"}, FullPath: "group-b/ws-b"},
			},
		}, nil)

		mockAuthorizer.On("RequireAccess", mock.Anything,
			[]models.Permission{models.ViewWorkspaceRoleBindingPermission}, mock.Anything).Return(nil)

		dbClient := &db.Client{WorkspaceRoleBindings: mockBindings, Workspaces: mockWorkspaces}
		testLogger, _ := logger.NewForTest()
		svc := &service{dbClient: dbClient, logger: testLogger}

		caller := auth.NewUserCaller(
			&models.User{Metadata: models.ResourceMetadata{ID: "user-1"}},
			mockAuthorizer,
			dbClient,
			mockMaintenanceMonitor,
			nil,
		)

		result, err := svc.GetWorkspaceRoleBindingsByWorkspaceIDs(auth.WithCaller(ctx, caller), []string{"ws-a", "ws-b", "ws-c"})
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("denies if the caller lacks permission on any returned binding", func(t *testing.T) {
		ctx := context.Background()

		mockBindings := db.NewMockWorkspaceRoleBindings(t)
		mockWorkspaces := db.NewMockWorkspaces(t)
		mockAuthorizer := auth.NewMockAuthorizer(t)
		mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

		mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()
		mockBindings.On("GetWorkspaceRoleBindings", mock.Anything, mock.Anything).Return(&db.WorkspaceRoleBindingsResult{
			WorkspaceRoleBindings: []models.WorkspaceRoleBinding{bindingA},
		}, nil)
		mockWorkspaces.On("GetWorkspaces", mock.Anything, &db.GetWorkspacesInput{
			Filter: &db.WorkspaceFilter{WorkspaceIDs: []string{"ws-a"}},
		}).Return(&db.WorkspacesResult{
			Workspaces: []models.Workspace{
				{Metadata: models.ResourceMetadata{ID: "ws-a"}, FullPath: "group-a/ws-a"},
			},
		}, nil)

		mockAuthorizer.On("RequireAccess", mock.Anything,
			[]models.Permission{models.ViewWorkspaceRoleBindingPermission}, mock.Anything).
			Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))

		dbClient := &db.Client{WorkspaceRoleBindings: mockBindings, Workspaces: mockWorkspaces}
		testLogger, _ := logger.NewForTest()
		svc := &service{dbClient: dbClient, logger: testLogger}

		caller := auth.NewUserCaller(
			&models.User{Metadata: models.ResourceMetadata{ID: "user-1"}},
			mockAuthorizer,
			dbClient,
			mockMaintenanceMonitor,
			nil,
		)

		_, err := svc.GetWorkspaceRoleBindingsByWorkspaceIDs(auth.WithCaller(ctx, caller), []string{"ws-a"})
		require.Error(t, err)
		assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	})
}
