package federatedregistry

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type mockDBClient struct {
	*db.Client
	MockTransactions        *db.MockTransactions
	MockResourceLimits      *db.MockResourceLimits
	MockGroups              *db.MockGroups
	MockFederatedRegistries *db.MockFederatedRegistries
	MockWorkspaces          *db.MockWorkspaces
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)
	// The mocks are enabled by the above function.

	mockResourceLimits := db.MockResourceLimits{}
	mockResourceLimits.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

	mockFederatedRegistries := db.MockFederatedRegistries{}
	mockFederatedRegistries.Test(t)

	mockWorkspaces := db.MockWorkspaces{}
	mockWorkspaces.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:        &mockTransactions,
			ResourceLimits:      &mockResourceLimits,
			Groups:              &mockGroups,
			FederatedRegistries: &mockFederatedRegistries,
			Workspaces:          &mockWorkspaces,
		},
		MockTransactions:        &mockTransactions,
		MockResourceLimits:      &mockResourceLimits,
		MockGroups:              &mockGroups,
		MockFederatedRegistries: &mockFederatedRegistries,
		MockWorkspaces:          &mockWorkspaces,
	}
}

func TestGetFederatedRegistriesByIDs(t *testing.T) {
	groupID := "group-1"
	groupPath := "group/path"
	otherGroupPath := "other/group/path"
	registryID := "federated-registry-1"
	otherRegistryID := "registry-that-does-not-exist"
	hostname := "test.example.invalid"

	testRegistry := &models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID: registryID,
		},
		Hostname:  hostname,
		GroupID:   groupID,
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name                      string
		ids                       []string
		authError                 error
		injectGroupAccessError    error
		injectFederatedRegistries []*models.FederatedRegistry
		expectFederatedRegistries []*models.FederatedRegistry
		expectErrCode             errors.CodeType
	}

	tests := []testCase{
		{
			name:          "subject is not authorized",
			ids:           []string{},
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EUnauthorized,
		},
		{
			name:                      "caller lacks permission to access the group",
			ids:                       []string{registryID},
			injectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
			injectGroupAccessError:    errors.New("no permission to access group", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:             errors.EForbidden,
		},
		{
			name: "by registry IDs, neg",
			ids:  []string{otherRegistryID},
		},
		{
			name:                      "by registry IDs, pos",
			ids:                       []string{registryID},
			injectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockCaller := auth.NewMockCaller(t)

			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			mockDBClient := buildDBClientWithMocks(t)

			mockDBClient.MockGroups.On("GetGroups", mock.Anything, &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					GroupPaths: []string{groupPath},
				},
			}).
				Return(&db.GroupsResult{
					Groups: []models.Group{
						{
							Metadata: models.ResourceMetadata{
								ID: groupID,
							},
						},
					},
				}, nil).Maybe()

			mockDBClient.MockGroups.On("GetGroups", mock.Anything, &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					GroupPaths: []string{otherGroupPath},
				},
			}).
				Return(&db.GroupsResult{
					Groups: []models.Group{},
				}, nil).Maybe()

			mockCaller.On("RequireAccessToInheritableResource", mock.Anything,
				types.FederatedRegistryModelType, mock.Anything).
				Return(test.injectGroupAccessError).Maybe()

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:          testLogger,
				dbClient:        mockDBClient.Client,
				limitChecker:    mockLimits,
				activityService: mockActivity,
			}

			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistries", mock.Anything,
				&db.GetFederatedRegistriesInput{
					Filter: &db.FederatedRegistryFilter{
						FederatedRegistryIDs: test.ids,
					},
				}).
				Return(&db.FederatedRegistriesResult{
					FederatedRegistries: test.injectFederatedRegistries,
				}, nil).Maybe()

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()

			testCtx := ctx
			if test.authError == nil {
				testCtx = auth.WithCaller(ctx, mockCaller)
			}
			actualRegistry, err := service.GetFederatedRegistriesByIDs(testCtx, test.ids)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectFederatedRegistries, actualRegistry)
		})
	}
}

func TestGetFederatedRegistryByID(t *testing.T) {
	groupID := "group-1"
	registryID := "federated-registry-1"
	otherRegistryID := "registry-that-does-not-exist"
	hostname := "test.example.invalid"

	testRegistry := &models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID: registryID,
		},
		Hostname:  hostname,
		GroupID:   groupID,
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name                          string
		requestID                     string
		authError                     error
		injectGroupAccessError        error
		injectRegistryPermissionError error
		expectFederatedRegistry       *models.FederatedRegistry
		expectErrCode                 errors.CodeType
	}

	tests := []testCase{
		{
			name:          "subject is not authorized",
			requestID:     registryID,
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EUnauthorized,
		},
		{
			name:                   "caller lacks permission to access the group",
			requestID:              registryID,
			injectGroupAccessError: errors.New("no permission to access the group", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:          errors.EForbidden,
		},
		{
			name:          "ID not found",
			requestID:     otherRegistryID,
			expectErrCode: errors.ENotFound,
		},
		{
			name:                    "success",
			requestID:               registryID,
			expectFederatedRegistry: testRegistry,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			mockDBClient := buildDBClientWithMocks(t)

			mockCaller.On("RequireAccessToInheritableResource", mock.Anything,
				types.FederatedRegistryModelType, mock.Anything).
				Return(test.injectGroupAccessError).Maybe()

			mockCaller.On("RequirePermission", mock.Anything,
				models.ViewFederatedRegistryPermission, mock.Anything).
				Return(test.injectRegistryPermissionError).Maybe()

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:          testLogger,
				dbClient:        mockDBClient.Client,
				limitChecker:    mockLimits,
				activityService: mockActivity,
			}

			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistryByID", mock.Anything, registryID).
				Return(testRegistry, nil).Maybe()
			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistryByID", mock.Anything, otherRegistryID).
				Return(nil, errors.New("registry not found by ID", errors.WithErrorCode(errors.ENotFound))).Maybe()

			testCtx := ctx
			if test.authError == nil {
				testCtx = auth.WithCaller(ctx, mockCaller)
			}

			actualRegistry, err := service.GetFederatedRegistryByID(testCtx, test.requestID)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectFederatedRegistry, actualRegistry)
		})
	}
}

func TestFederatedRegistryByTRN(t *testing.T) {
	sampleFederatedRegistry := &models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID:  "federated-registry-id-1",
			TRN: types.FederatedRegistryModelType.BuildTRN("my-group/123341"),
		},
		GroupID:   "group-1",
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name            string
		authError       error
		registry        *models.FederatedRegistry
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:     "successfully get federated registry by trn",
			registry: sampleFederatedRegistry,
		},
		{
			name:            "federated registry not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view federated registry",
			registry:        sampleFederatedRegistry,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockFederatedRegistries := db.NewMockFederatedRegistries(t)

			mockFederatedRegistries.On("GetFederatedRegistryByTRN", mock.Anything, sampleFederatedRegistry.Metadata.TRN).Return(test.registry, nil)

			if test.registry != nil {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.FederatedRegistryModelType, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				FederatedRegistries: mockFederatedRegistries,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualRegistry, err := service.GetFederatedRegistryByTRN(auth.WithCaller(ctx, mockCaller), sampleFederatedRegistry.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.registry, actualRegistry)
		})
	}
}

func TestGetFederatedRegistries(t *testing.T) {
	groupID := "group-1"
	groupPath := "group/path"
	otherGroupPath := "other/group/path"
	registryID := "federated-registry-1"
	hostname := "test.example.invalid"
	otherHostname := "other.test.example.invalid"
	sortAsc := db.FederatedRegistrySortableFieldUpdatedAtAsc
	sortDesc := db.FederatedRegistrySortableFieldUpdatedAtDesc

	testRegistry := &models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID: registryID,
		},
		Hostname:  hostname,
		GroupID:   groupID,
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name                          string
		input                         *GetFederatedRegistriesInput
		userCaller                    bool
		isAdmin                       bool
		authError                     error
		injectGroupAccessError        error
		injectRegistryPermissionError error
		expectFederatedRegistries     []*models.FederatedRegistry
		expectErrCode                 errors.CodeType
	}

	tests := []testCase{
		{
			name:          "subject is not authorized",
			input:         &GetFederatedRegistriesInput{},
			userCaller:    true,
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EUnauthorized,
		},
		{
			name:                   "caller lacks permission to access the group",
			input:                  &GetFederatedRegistriesInput{},
			userCaller:             true,
			injectGroupAccessError: errors.New("no permission to access group", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:          errors.EForbidden,
		},
		{
			name:                          "caller lacks permission to view the registry",
			input:                         &GetFederatedRegistriesInput{},
			userCaller:                    true,
			injectRegistryPermissionError: errors.New("no permission to view registry", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:                 errors.EForbidden,
		},
		{
			name:          "no filtering, non-admin, neg",
			input:         &GetFederatedRegistriesInput{},
			userCaller:    true,
			expectErrCode: errors.EForbidden,
		},
		{
			name:                      "no filtering, admin, pos",
			input:                     &GetFederatedRegistriesInput{},
			userCaller:                true,
			isAdmin:                   true,
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
		{
			name: "by registry endpoint, neg",
			input: &GetFederatedRegistriesInput{
				Hostname: &otherHostname,
			},
			userCaller: true,
			isAdmin:    true,
		},
		{
			name: "by registry endpoint, pos",
			input: &GetFederatedRegistriesInput{
				Hostname: &hostname,
			},
			userCaller:                true,
			isAdmin:                   true,
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
		{
			name: "by group ID, neg",
			input: &GetFederatedRegistriesInput{
				GroupPath: &otherGroupPath,
			},
		},
		{
			name: "by group ID, pos",
			input: &GetFederatedRegistriesInput{
				GroupPath: &groupPath,
			},
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
		{
			name: "sort by asc",
			input: &GetFederatedRegistriesInput{
				Sort: &sortAsc,
			},
			userCaller:                true,
			isAdmin:                   true,
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
		{
			name: "sort by desc",
			input: &GetFederatedRegistriesInput{
				Sort: &sortDesc,
			},
			userCaller:                true,
			isAdmin:                   true,
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
		{
			name: "paginate",
			input: &GetFederatedRegistriesInput{
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(5),
				},
			},
			userCaller:                true,
			isAdmin:                   true,
			expectFederatedRegistries: []*models.FederatedRegistry{testRegistry},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)
			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
			mockCaller := auth.NewMockCaller(t)

			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			mockDBClient := buildDBClientWithMocks(t)

			mockDBClient.MockGroups.On("GetGroups", mock.Anything, &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					GroupPaths: []string{groupPath},
				},
			}).
				Return(&db.GroupsResult{
					Groups: []models.Group{
						{
							Metadata: models.ResourceMetadata{
								ID: groupID,
							},
						},
					},
				}, nil).Maybe()

			mockDBClient.MockGroups.On("GetGroups", mock.Anything, &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					GroupPaths: []string{otherGroupPath},
				},
			}).
				Return(&db.GroupsResult{
					Groups: []models.Group{},
				}, nil).Maybe()

			mockCaller.On("RequireAccessToInheritableResource", mock.Anything,
				types.FederatedRegistryModelType, mock.Anything).
				Return(test.injectGroupAccessError).Maybe()

			mockCaller.On("RequirePermission", mock.Anything,
				models.ViewFederatedRegistryPermission, mock.Anything).
				Return(test.injectRegistryPermissionError).Maybe()

			testCaller := auth.NewUserCaller(
				&models.User{
					Metadata: models.ResourceMetadata{
						ID: "user-1-id",
					},
					Admin:    test.isAdmin,
					Username: "user1",
				},
				&mockAuthorizer,
				mockDBClient.Client,
				mockMaintenanceMonitor,
				nil,
			)

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:          testLogger,
				dbClient:        mockDBClient.Client,
				limitChecker:    mockLimits,
				activityService: mockActivity,
			}

			var wantGroupPaths []string
			if test.input.GroupPath != nil {
				wantGroupPaths = []string{*test.input.GroupPath}
			}
			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistries", mock.Anything,
				&db.GetFederatedRegistriesInput{
					Sort:              test.input.Sort,
					PaginationOptions: test.input.PaginationOptions,
					Filter: &db.FederatedRegistryFilter{
						Hostname:   test.input.Hostname,
						GroupPaths: wantGroupPaths,
					},
				}).
				Return(&db.FederatedRegistriesResult{
					FederatedRegistries: test.expectFederatedRegistries,
				}, nil).Maybe()

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()

			testCtx := ctx
			if test.authError == nil {
				if test.userCaller {
					// User caller for testing admin vs. non-admin.
					testCtx = auth.WithCaller(ctx, testCaller)
				} else {
					// Mock non-user caller for testing permissions.
					testCtx = auth.WithCaller(ctx, mockCaller)
				}
			}

			actualRegistry, err := service.GetFederatedRegistries(testCtx, test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectFederatedRegistries, actualRegistry.FederatedRegistries)
		})
	}
}

func TestCreateFederatedRegistry(t *testing.T) {
	groupID := "12345678-1234-1234-1234-123456789abc"
	registryID := "federated-registry-1"
	hostname := "test.example.invalid"

	testRegistry := models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID: registryID,
		},
		Hostname:  hostname,
		GroupID:   groupID,
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name                          string
		input                         models.FederatedRegistry
		authError                     error
		injectGroupAccessError        error
		injectRegistryPermissionError error
		exceedsLimitError             error
		expectFederatedRegistry       models.FederatedRegistry
		expectErrCode                 errors.CodeType
	}

	tests := []testCase{
		{
			name:          "subject is not authorized",
			input:         testRegistry,
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EUnauthorized,
		},
		{
			name:                          "caller lacks permission to create the registry",
			input:                         testRegistry,
			injectRegistryPermissionError: errors.New("no permission to create registry", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:                 errors.EForbidden,
		},
		{
			name:              "exceeds limit",
			input:             testRegistry,
			exceedsLimitError: errors.New("limit exceeded", errors.WithErrorCode(errors.EInvalid)),
			expectErrCode:     errors.EInvalid,
		},
		{
			name:                    "success",
			input:                   testRegistry,
			expectFederatedRegistry: testRegistry,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			mockDBClient := buildDBClientWithMocks(t)

			mockCaller.On("GetSubject").Return("test-subject").Maybe()

			mockCaller.On("RequireAccessToInheritableResource", mock.Anything,
				types.FederatedRegistryModelType, mock.Anything).
				Return(test.injectGroupAccessError).Maybe()

			mockCaller.On("RequirePermission", mock.Anything,
				models.CreateFederatedRegistryPermission, mock.Anything).
				Return(test.injectRegistryPermissionError).Maybe()

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:          testLogger,
				dbClient:        mockDBClient.Client,
				limitChecker:    mockLimits,
				activityService: mockActivity,
			}

			mockDBClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockDBClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockDBClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockDBClient.MockFederatedRegistries.On("CreateFederatedRegistry", mock.Anything, mock.MatchedBy(func(input *models.FederatedRegistry) bool {
				// Only match on the fields we care about, ignoring CreatedBy which is set by the service
				return input.Hostname == test.input.Hostname &&
					input.GroupID == test.input.GroupID &&
					input.Audience == test.input.Audience
			})).
				Return(&test.expectFederatedRegistry, nil).Maybe()

			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistries", mock.Anything,
				&db.GetFederatedRegistriesInput{
					Filter: &db.FederatedRegistryFilter{
						GroupID: &test.input.GroupID,
					},
				}).
				Return(&db.FederatedRegistriesResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: 1,
					},
					FederatedRegistries: []*models.FederatedRegistry{&testRegistry},
				}, nil).Maybe()

			mockLimits.On("CheckLimit", mock.Anything, mock.Anything, int32(1)).
				Return(test.exceedsLimitError).Maybe()

			mockDBClient.MockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{
					Metadata: models.ResourceMetadata{
						ID: groupID,
					},
				}, nil).Maybe()

			mockActivity.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(&models.ActivityEvent{
					// The event is ignored, so don't need to assign anything.
				}, nil).Maybe()

			testCtx := ctx
			if test.authError == nil {
				testCtx = auth.WithCaller(ctx, mockCaller)
			}

			inputCopy := test.input
			actualRegistry, err := service.CreateFederatedRegistry(testCtx, &inputCopy)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, actualRegistry)
			assert.Equal(t, test.expectFederatedRegistry, *actualRegistry)
		})
	}
}

func TestUpdateFederatedRegistry(t *testing.T) {
	groupID := "12345678-1234-1234-1234-123456789abc"
	registryID := "federated-registry"
	otherRegistryID := "other-federated-registry"
	hostname := "test.example.invalid"

	testRegistry := models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID: registryID,
		},
		Hostname:  hostname,
		GroupID:   groupID,
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name                          string
		input                         models.FederatedRegistry
		authError                     error
		injectRegistryPermissionError error
		expectFederatedRegistry       models.FederatedRegistry
		expectErrCode                 errors.CodeType
	}

	tests := []testCase{
		{
			name:          "subject is not authorized",
			input:         testRegistry,
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EUnauthorized,
		},
		{
			name:                          "caller lacks permission to update the registry",
			input:                         testRegistry,
			injectRegistryPermissionError: errors.New("no permission to update registry", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:                 errors.EForbidden,
		},
		{
			name: "registry not found",
			input: models.FederatedRegistry{
				Metadata: models.ResourceMetadata{
					ID: otherRegistryID,
				},
				Hostname:  hostname,
				GroupID:   groupID,
				Audience:  "test-audience",
				CreatedBy: "test-user",
			},
			expectErrCode: errors.ENotFound,
		},
		{
			name:                    "success",
			input:                   testRegistry,
			expectFederatedRegistry: testRegistry,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			mockDBClient := buildDBClientWithMocks(t)

			mockCaller.On("GetSubject").Return("test-subject").Maybe()

			mockCaller.On("RequirePermission", mock.Anything,
				models.UpdateFederatedRegistryPermission, mock.Anything).
				Return(test.injectRegistryPermissionError).Maybe()

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:          testLogger,
				dbClient:        mockDBClient.Client,
				limitChecker:    mockLimits,
				activityService: mockActivity,
			}

			mockDBClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockDBClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockDBClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockDBClient.MockFederatedRegistries.On("UpdateFederatedRegistry", mock.Anything, mock.MatchedBy(func(input *models.FederatedRegistry) bool {
				return input.Metadata.ID == registryID
			})).
				Return(&test.expectFederatedRegistry, nil).Maybe()
			mockDBClient.MockFederatedRegistries.On("UpdateFederatedRegistry", mock.Anything, mock.MatchedBy(func(input *models.FederatedRegistry) bool {
				return input.Metadata.ID == otherRegistryID
			})).
				Return(nil, errors.New("test registry not found", errors.WithErrorCode(errors.ENotFound))).Maybe()

			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistries", mock.Anything,
				&db.GetFederatedRegistriesInput{
					Filter: &db.FederatedRegistryFilter{
						GroupID: &test.input.GroupID,
					},
				}).
				Return(&db.FederatedRegistriesResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: 1,
					},
					FederatedRegistries: []*models.FederatedRegistry{&testRegistry},
				}, nil).Maybe()

			mockDBClient.MockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{
					Metadata: models.ResourceMetadata{
						ID: groupID,
					},
				}, nil).Maybe()

			mockActivity.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(&models.ActivityEvent{
					// The event is ignored, so don't need to assign anything.
				}, nil).Maybe()

			testCtx := ctx
			if test.authError == nil {
				testCtx = auth.WithCaller(ctx, mockCaller)
			}

			inputCopy := test.input
			actualRegistry, err := service.UpdateFederatedRegistry(testCtx, &inputCopy)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			require.NotNil(t, actualRegistry)
			assert.Equal(t, test.expectFederatedRegistry, *actualRegistry)
		})
	}
}

func TestDeleteFederatedRegistry(t *testing.T) {
	groupID := "12345678-1234-1234-1234-123456789abc"
	registryID := "federated-registry"
	otherRegistryID := "other-federated-registry"
	hostname := "test.example.invalid"

	testRegistry := models.FederatedRegistry{
		Metadata: models.ResourceMetadata{
			ID: registryID,
		},
		Hostname:  hostname,
		GroupID:   groupID,
		Audience:  "test-audience",
		CreatedBy: "test-user",
	}

	type testCase struct {
		name                          string
		input                         models.FederatedRegistry
		authError                     error
		injectRegistryPermissionError error
		expectErrCode                 errors.CodeType
	}

	tests := []testCase{
		{
			name:          "subject is not authorized",
			input:         testRegistry,
			authError:     errors.New("Unauthorized", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EUnauthorized,
		},
		{
			name:                          "caller lacks permission to delete the registry",
			input:                         testRegistry,
			injectRegistryPermissionError: errors.New("no permission to delete registry", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode:                 errors.EForbidden,
		},
		{
			name:  "success",
			input: testRegistry,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			mockDBClient := buildDBClientWithMocks(t)

			mockCaller.On("GetSubject").Return("test-subject").Maybe()

			mockCaller.On("RequirePermission", mock.Anything,
				models.DeleteFederatedRegistryPermission, mock.Anything).
				Return(test.injectRegistryPermissionError).Maybe()

			testLogger, _ := logger.NewForTest()
			service := &service{
				logger:          testLogger,
				dbClient:        mockDBClient.Client,
				limitChecker:    mockLimits,
				activityService: mockActivity,
			}

			mockDBClient.MockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockDBClient.MockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockDBClient.MockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockDBClient.MockFederatedRegistries.On("DeleteFederatedRegistry", mock.Anything, &testRegistry).
				Return(nil).Maybe()
			mockDBClient.MockFederatedRegistries.On("DeleteFederatedRegistry", mock.Anything,
				&models.FederatedRegistry{
					Metadata: models.ResourceMetadata{
						ID: otherRegistryID,
					},
					Hostname:  hostname,
					GroupID:   groupID,
					Audience:  "test-audience",
					CreatedBy: "test-user",
				}).
				Return(nil, errors.New("test registry not found", errors.WithErrorCode(errors.ENotFound))).Maybe()

			mockDBClient.MockFederatedRegistries.On("GetFederatedRegistries", mock.Anything,
				&db.GetFederatedRegistriesInput{
					Filter: &db.FederatedRegistryFilter{
						GroupID: &test.input.GroupID,
					},
				}).
				Return(&db.FederatedRegistriesResult{
					PageInfo: &pagination.PageInfo{
						TotalCount: 1,
					},
					FederatedRegistries: []*models.FederatedRegistry{&testRegistry},
				}, nil).Maybe()

			mockDBClient.MockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{
					Metadata: models.ResourceMetadata{
						ID: groupID,
					},
				}, nil).Maybe()

			mockActivity.On("CreateActivityEvent", mock.Anything, mock.Anything).
				Return(&models.ActivityEvent{
					// The event is ignored, so don't need to assign anything.
				}, nil).Maybe()

			testCtx := ctx
			if test.authError == nil {
				testCtx = auth.WithCaller(ctx, mockCaller)
			}

			inputCopy := test.input
			err := service.DeleteFederatedRegistry(testCtx, &inputCopy)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateFederatedRegistryTokensForJob(t *testing.T) {
	jobID := "job-1"
	workspaceID := "workspace-1"
	groupPath := "group/path"
	hostname1 := "registry1.example.com"

	testCases := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockJobs, *db.MockWorkspaces, *db.MockFederatedRegistries, *auth.MockSigningKeyManager)
		expectTokens    []*Token
		expectErrorCode errors.CodeType
	}{
		{
			name: "authorization fails",
			setupMocks: func(mockCaller *auth.MockCaller, _ *db.MockJobs, _ *db.MockWorkspaces, _ *db.MockFederatedRegistries, _ *auth.MockSigningKeyManager) {
				mockCaller.On("RequirePermission", mock.Anything, models.CreateFederatedRegistryTokenPermission, mock.Anything).
					Return(errors.New("caller lacks permission", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "successful token creation",
			setupMocks: func(mockCaller *auth.MockCaller, mockJobs *db.MockJobs, mockWorkspaces *db.MockWorkspaces, mockFederatedRegistries *db.MockFederatedRegistries, mockIDP *auth.MockSigningKeyManager) {
				mockCaller.On("RequirePermission", mock.Anything, models.CreateFederatedRegistryTokenPermission, mock.Anything).
					Return(nil)
				mockJobs.On("GetJobByID", mock.Anything, jobID).Return(&models.Job{
					WorkspaceID: workspaceID,
				}, nil)
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).Return(&models.Workspace{
					FullPath: groupPath + "/workspace-name",
				}, nil)

				mockFederatedRegistries.On("GetFederatedRegistries", mock.Anything, &db.GetFederatedRegistriesInput{
					Filter: &db.FederatedRegistryFilter{
						GroupPaths: utils.ExpandPath(groupPath),
					},
				}).Return(&db.FederatedRegistriesResult{
					FederatedRegistries: []*models.FederatedRegistry{
						{
							Metadata: models.ResourceMetadata{
								ID: "registry1",
							},
							Hostname:  hostname1,
							GroupID:   "group-1",
							Audience:  "test-audience",
							CreatedBy: "test-user",
						},
					},
				}, nil)

				// Mock the token creation for two registries
				mockIDP.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("token1"), nil).Once()
			},
			expectTokens: []*Token{
				{Token: "token1", Hostname: hostname1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			mockCaller := auth.NewMockCaller(t)
			mockJobs := db.NewMockJobs(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockFederatedRegistries := db.NewMockFederatedRegistries(t)
			mockIDP := auth.NewMockSigningKeyManager(t)

			mockDBClient := &db.Client{
				Jobs:                mockJobs,
				Workspaces:          mockWorkspaces,
				FederatedRegistries: mockFederatedRegistries,
			}

			logger, _ := logger.NewForTest()
			mockLimits := limits.NewMockLimitChecker(t)
			mockActivity := activityevent.NewMockService(t)

			service := &service{
				logger:           logger,
				dbClient:         mockDBClient,
				limitChecker:     mockLimits,
				activityService:  mockActivity,
				identityProvider: mockIDP,
			}

			// Setup mocks
			tc.setupMocks(mockCaller, mockJobs, mockWorkspaces, mockFederatedRegistries, mockIDP)

			// Call the function with the mocked caller
			testCtx := auth.WithCaller(ctx, mockCaller)
			tokens, err := service.CreateFederatedRegistryTokensForJob(testCtx, jobID)

			// Verify results
			if tc.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, errors.ErrorCode(err), tc.expectErrorCode)
			} else {
				require.NoError(t, err)
				require.Equal(t, len(tc.expectTokens), len(tokens))

				// Create maps for easier comparison
				actualMap := make(map[string]string)
				expectedMap := make(map[string]string)

				for _, token := range tokens {
					actualMap[token.Hostname] = token.Token
				}

				for _, token := range tc.expectTokens {
					expectedMap[token.Hostname] = token.Token
				}

				assert.Equal(t, expectedMap, actualMap)
			}
		})
	}
}
