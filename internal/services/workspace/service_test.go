package workspace

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"

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
		name            string
		foundWorkspace  *models.Workspace
		authError       error
		updateError     error
		expectWorkspace *models.Workspace
		expectErrorCode errors.CodeType
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
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockWorkspaces := db.NewMockWorkspaces(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateWorkspacePermission, mock.Anything).
				Return(test.authError)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			mockTransactions.On("BeginTx", mock.Anything).
				Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).
				Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).
				Return(nil).Maybe()

			mockWorkspaces.On("UpdateWorkspace", mock.Anything, updatedWorkspace).
				Return(updatedWorkspace, test.updateError).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything,
				&activityevent.CreateActivityEventInput{
					NamespacePath: &originalWorkspace.FullPath,
					Action:        models.ActionUpdate,
					TargetType:    models.TargetWorkspace,
					TargetID:      originalWorkspace.Metadata.ID,
				},
			).Return(&models.ActivityEvent{}, nil).Maybe()

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

			actualUpdated, err := service.UpdateWorkspace(auth.WithCaller(ctx, mockCaller), updatedWorkspace)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, updatedWorkspace, actualUpdated)
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

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateStateVersionPermission, mock.Anything).
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

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), &mockArtifactStore, nil, nil, &mockActivityEvents)

			if !test.authFail {
				ctx = auth.WithCaller(ctx, &mockCaller)
			}

			testDataString := string(test.data)
			result, err := service.CreateStateVersion(ctx, test.toCreate, &testDataString)

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

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateConfigurationVersionPermission, mock.Anything).
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

			service := NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient), nil, nil, nil, &mockActivityEvents)

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

			perms := []permissions.Permission{permissions.UpdateWorkspacePermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(workspaceAccessError)

			perms = []permissions.Permission{permissions.DeleteWorkspacePermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(parentAccessError)

			perms = []permissions.Permission{permissions.CreateWorkspacePermission}
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
			)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, limiter, nil, nil, nil, &mockActivityEvents)

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
