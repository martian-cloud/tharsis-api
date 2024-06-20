package workspace

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"

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

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateWorkspacePermission, mock.Anything).Return(test.authError)

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

			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient), nil, nil, mockCLIService, mockActivityEvents)

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

func TestGetWorkspaces(t *testing.T) {
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
	}

	testCases := []testCase{
		{
			name: "positive: successfully returns workspaces for a group",
			input: &GetWorkspacesInput{
				Group: &models.Group{Metadata: models.ResourceMetadata{ID: "some-group-id"}},
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
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockCaller := auth.NewMockCaller(t)

			if !test.failAuthorization {
				ctx = auth.WithCaller(ctx, mockCaller)
			}

			input := db.GetWorkspacesInput{
				Sort:              test.input.Sort,
				PaginationOptions: test.input.PaginationOptions,
				Filter: &db.WorkspaceFilter{
					Search:                    test.input.Search,
					AssignedManagedIdentityID: test.input.AssignedManagedIdentityID,
				},
			}

			if test.input.Group != nil {
				input.Filter.GroupID = &test.input.Group.Metadata.ID

				mockCaller.On("RequirePermission", mock.Anything, permissions.ViewWorkspacePermission, mock.Anything).Return(test.requireWorkspacePermissionError)
			}

			policy := auth.NamespaceAccessPolicy{AllowAll: test.accessPolicyAllowAll}
			mockCaller.On("GetNamespaceAccessPolicy", mock.Anything).Return(&policy, test.namespaceAccessPolicyError).Maybe()

			if test.userID != nil {
				input.Filter.UserMemberID = test.userID
			}

			if test.serviceAccountID != nil {
				input.Filter.ServiceAccountMemberID = test.serviceAccountID
			}

			workspacesResult := db.WorkspacesResult{Workspaces: test.expectResult}
			mockWorkspaces.On("GetWorkspaces", mock.Anything, &input).Return(&workspacesResult, test.getWorkspacesError).Maybe()

			dbClient := &db.Client{
				Workspaces: mockWorkspaces,
			}

			if test.handleCaller == nil {
				test.handleCaller = auth.HandleCaller
			}

			service := newService(nil, dbClient, nil, nil, nil, nil, nil, test.handleCaller)

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
