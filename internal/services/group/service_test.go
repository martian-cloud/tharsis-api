package group

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type mockDBClient struct {
	*db.Client
	MockTransactions   *db.MockTransactions
	MockResourceLimits *db.MockResourceLimits
	MockGroups         *db.MockGroups
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)

	mockResourceLimits := db.MockResourceLimits{}
	mockResourceLimits.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:   &mockTransactions,
			ResourceLimits: &mockResourceLimits,
			Groups:         &mockGroups,
		},
		MockTransactions:   &mockTransactions,
		MockResourceLimits: &mockResourceLimits,
		MockGroups:         &mockGroups,
	}
}

func TestGetGroupByID(t *testing.T) {
	sampleGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-id-1",
		},
		Name:     "group-1",
		FullPath: "my-group/group-1",
	}

	type testCase struct {
		name            string
		authError       error
		group           *models.Group
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:  "successfully get group by ID",
			group: sampleGroup,
		},
		{
			name:            "group not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view group",
			group:           sampleGroup,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockGroups := db.NewMockGroups(t)

			mockGroups.On("GetGroupByID", mock.Anything, sampleGroup.Metadata.ID).Return(test.group, nil)

			if test.group != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewGroupPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Groups: mockGroups,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualGroup, err := service.GetGroupByID(auth.WithCaller(ctx, mockCaller), sampleGroup.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualGroup)
			assert.Equal(t, test.group, actualGroup)
		})
	}
}

func TestGroupByTRN(t *testing.T) {
	sampleGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID:  "group-id-1",
			TRN: types.GroupModelType.BuildTRN("my-group/group-1"),
		},
		Name:     "group-1",
		FullPath: "my-group/group-1",
	}

	type testCase struct {
		name            string
		authError       error
		group           *models.Group
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:  "successfully get group by trn",
			group: sampleGroup,
		},
		{
			name:            "group not found",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "subject is not authorized to view group",
			group:           sampleGroup,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockGroups := db.NewMockGroups(t)

			mockGroups.On("GetGroupByTRN", mock.Anything, sampleGroup.Metadata.TRN).Return(test.group, nil)

			if test.group != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewGroupPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				Groups: mockGroups,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualGroup, err := service.GetGroupByTRN(auth.WithCaller(ctx, mockCaller), sampleGroup.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.group, actualGroup)
		})
	}
}

func TestCreateTopLevelGroup(t *testing.T) {
	// Test cases
	tests := []struct {
		caller          *auth.UserCaller
		name            string
		expectErrorCode errors.CodeType
		input           models.Group
	}{
		{
			name: "create group",
			input: models.Group{
				Name:       "group1",
				Metadata:   models.ResourceMetadata{ID: "group1"},
				RunnerTags: []string{"tag1"},
			},
			caller: &auth.UserCaller{
				User: &models.User{Metadata: models.ResourceMetadata{ID: "user1"}, Admin: true},
			},
		},
		{
			name: "cannot create top-level group because caller is not an admin",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
			},
			caller: &auth.UserCaller{
				User: &models.User{Metadata: models.ResourceMetadata{ID: "user1"}, Admin: false},
			},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := namespacemembership.NewMockService(t)
			mockGroups := db.NewMockGroups(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			createNamespaceMembershipInput := &namespacemembership.CreateNamespaceMembershipInput{
				NamespacePath: test.input.FullPath,
				RoleID:        models.OwnerRoleID.String(),
				User:          test.caller.User,
			}

			if test.expectErrorCode == "" {
				mockGroups.On("CreateGroup", mock.Anything, &test.input).Return(&test.input, nil)

				mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, createNamespaceMembershipInput).Return(nil, nil)

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			dbClient := &db.Client{
				Groups:       mockGroups,
				Transactions: mockTransactions,
			}

			limiter := limits.NewLimitChecker(dbClient)

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, limiter, mockNamespaceMemberships, mockActivityEvents, nil)

			group, err := service.CreateGroup(auth.WithCaller(ctx, test.caller), &test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, &test.input, group)
				// Verify namespace membership is created
				mockNamespaceMemberships.AssertCalled(t, "CreateNamespaceMembership", mock.Anything, createNamespaceMembershipInput)
			}
		})
	}
}

func TestCreateNestedGroup(t *testing.T) {
	// Test cases
	tests := []struct {
		authError       error
		name            string
		expectErrorCode errors.CodeType
		input           models.Group
		limit           int // same for both siblings and depth
		parentChildren  int32
		exceedsDepth    bool
	}{
		{
			name: "create group",
			input: models.Group{
				Name:       "group1",
				Metadata:   models.ResourceMetadata{ID: "group1"},
				ParentID:   "group0",
				FullPath:   "a/b/c/group0/group1",
				RunnerTags: []string{"tag2", "tag3"},
			},
			limit:          5,
			parentChildren: 5,
		},
		{
			name: "caller is not authorized to create group",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group0",
				FullPath: "a/b/c/group0/group1",
			},
			limit:           5,
			parentChildren:  5,
			authError:       errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "exceeds sibling limit",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group0-sibling-limit",
				FullPath: "a/b/c/group0/group1",
			},
			limit:           5,
			parentChildren:  6,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "exceeds depth limit",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group-too-deep",
				FullPath: "a/b/c/d/group-too-deep/group1",
			},
			limit:           5,
			parentChildren:  4,
			exceedsDepth:    true,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockGroups := db.NewMockGroups(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			dbClient := db.Client{
				Groups:         mockGroups,
				Transactions:   mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			limiter := limits.NewLimitChecker(&dbClient)

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, models.CreateGroupPermission, mock.Anything).Return(test.authError)

			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockCaller.On("GetSubject").Return("testsubject")

				mockGroups.On("CreateGroup", mock.Anything, &test.input).Return(&test.input, nil)

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

				if test.expectErrorCode == "" {
					mockTransactions.On("CommitTx", mock.Anything).Return(nil)
					mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
				}

				// called from inside checkParentSubgroupLimit
				mockGroups.On("GetGroups", mock.Anything, mock.Anything).
					Return(func(ctx context.Context, input *db.GetGroupsInput) *db.GroupsResult {
						_ = ctx
						_ = input

						return &db.GroupsResult{
							PageInfo: &pagination.PageInfo{
								TotalCount: test.parentChildren,
							},
						}
					}, nil)

				// called from inside checkParentSubgroupLimit
				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, limiter, nil, mockActivityEvents, nil)

			group, err := service.CreateGroup(auth.WithCaller(ctx, mockCaller), &test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, &test.input, group)
			}
		})
	}
}

// TestGetGroups verifies that the auth filters are correctly passed to the DB layer for various conditions.
// This test currently mainly exercises the search feature.
func TestGetGroups(t *testing.T) {
	rootNamespaceID := "root-namespace-1"
	parentGroupID := "this-is-a-fake-parent-group-ID"
	emptySearch := ""
	nonEmptySearch := "non-empty-search-string"
	userMemberID := "this-is-a-fake-user-member-ID"
	serviceAccountMemberID := "this is a fake-service-account-member-ID"
	serviceAccountPath := "this/is/a/fake/service/account/path"

	// Because this test focuses only on the filters passed to the DB layer, don't worry about end-to-end errors and such.
	type testCase struct {
		svcInput   *GetGroupsInput
		dbInput    *db.GetGroupsInput
		name       string
		callerType string // "admin", "user", "service-account"
	}

	// Test cases
	testCases := []testCase{
		{
			name:       "admin caller, no parent group, search absent/nil, no root-only",
			callerType: "admin",
			svcInput:   &GetGroupsInput{
				// everything nil/false
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					// everything nil/false
				},
			},
		},
		{
			name:       "admin caller, no parent group, search absent/nil, with root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					RootOnly: true,
				},
			},
		},
		{
			name:       "admin caller, no parent group, search empty, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				Search: &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search: &emptySearch,
				},
			},
		},
		{
			name:       "admin caller, no parent group, search empty, with root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				Search:   &emptySearch,
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search:   &emptySearch,
					RootOnly: true,
				},
			},
		},
		{
			name:       "admin caller, no parent group, search non-empty, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				Search: &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search: &nonEmptySearch,
				},
			},
		},
		{
			name:       "admin caller, no parent group, search non-empty, with root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				Search:   &nonEmptySearch,
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search:   &nonEmptySearch,
					RootOnly: true,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search absent/nil, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search empty, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
				Search:        &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search non-empty, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
				Search:        &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
				},
			},
		},
		{
			name:       "user member caller, no parent group, search absent/nil, no root-only",
			callerType: "user",
			svcInput:   &GetGroupsInput{
				// everything nil/false
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					UserMemberID: &userMemberID,
				},
			},
		},
		{
			name:       "user member caller, no parent group, search absent/nil, with root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					NamespaceIDs: []string{rootNamespaceID},
					RootOnly:     false,
				},
			},
		},
		{
			name:       "user member caller, no parent group, search empty, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				Search: &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					UserMemberID: &userMemberID,
					Search:       &emptySearch,
				},
			},
		},
		{
			name:       "user member caller, no parent group, search empty, with root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				Search:   &emptySearch,
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search:       &emptySearch,
					NamespaceIDs: []string{rootNamespaceID},
					RootOnly:     false,
				},
			},
		},
		{
			name:       "user member caller, no parent group, search non-empty, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				Search: &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					UserMemberID: &userMemberID,
					Search:       &nonEmptySearch,
				},
			},
		},
		{
			name:       "user member caller, no parent group, search non-empty, with root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				Search:   &nonEmptySearch,
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search:       &nonEmptySearch,
					NamespaceIDs: []string{rootNamespaceID},
					RootOnly:     false,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search absent/nil, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search empty, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
				Search:        &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search non-empty, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
				Search:        &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
				},
			},
		},
		{
			name:       "service account member caller, no parent group, search absent/nil, no root-only",
			callerType: "service-account",
			svcInput:   &GetGroupsInput{
				// everything nil/false
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ServiceAccountMemberID: &serviceAccountMemberID,
				},
			},
		},
		{
			name:       "service account member caller, no parent group, search absent/nil, with root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					NamespaceIDs: []string{rootNamespaceID},
					RootOnly:     false,
				},
			},
		},
		{
			name:       "service account member caller, no parent group, search empty, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				Search: &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ServiceAccountMemberID: &serviceAccountMemberID,
					Search:                 &emptySearch,
				},
			},
		},
		{
			name:       "service account member caller, no parent group, search empty, with root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				Search:   &emptySearch,
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search:       &emptySearch,
					NamespaceIDs: []string{rootNamespaceID},
					RootOnly:     false,
				},
			},
		},
		{
			name:       "service account member caller, no parent group, search non-empty, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				Search: &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ServiceAccountMemberID: &serviceAccountMemberID,
					Search:                 &nonEmptySearch,
				},
			},
		},
		{
			name:       "service account member caller, no parent group, search non-empty, with root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				Search:   &nonEmptySearch,
				RootOnly: true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					Search:       &nonEmptySearch,
					NamespaceIDs: []string{rootNamespaceID},
					RootOnly:     false,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search absent/nil, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search empty, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
				Search:        &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search non-empty, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroupID: &parentGroupID,
				Search:        &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient := buildDBClientWithMocks(t)

			limiter := limits.NewLimitChecker(dbClient.Client)

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockMaintenanceMonitor := maintenance.NewMockMonitor(t)

			mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil).Maybe()

			mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{
				{ID: rootNamespaceID},
			}, nil)

			mockAuthorizer.On("RequireAccess", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			var testCaller auth.Caller
			switch test.callerType {
			case "admin":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: userMemberID,
						},
						Admin:    true,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient.Client,
					mockMaintenanceMonitor,
				)
			case "user":
				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: userMemberID,
						},
						Admin:    false,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient.Client,
					mockMaintenanceMonitor,
				)
			case "service-account":
				testCaller = auth.NewServiceAccountCaller(
					serviceAccountMemberID,
					serviceAccountPath,
					&mockAuthorizer,
					dbClient.Client,
					mockMaintenanceMonitor,
				)
			default:
				assert.Fail(t, "invalid caller type in test")
			}

			// If the service layer sends wrong/unexpected auth values, the last argument will prevent the
			// mocking of the function from taking effect.  Because the mock state is cleared between test
			// cases, an earlier .On(...) won't mask an error by the service layer.
			dbClient.MockGroups.On("GetGroups", mock.Anything, test.dbInput).
				Return(func(_ context.Context, _ *db.GetGroupsInput) *db.GroupsResult {
					return &db.GroupsResult{
						Groups: []models.Group{},
					}
				}, nil,
				)

			logger, _ := logger.NewForTest()
			activityService := activityevent.NewService(dbClient.Client, logger)
			namespaceMembershipService := namespacemembership.NewService(logger, dbClient.Client, activityService)
			service := NewService(logger, dbClient.Client, limiter, namespaceMembershipService, activityService, nil)

			// Call the service function.
			actualOutput, actualError := service.GetGroups(auth.WithCaller(ctx, testCaller), test.svcInput)
			if actualError != nil {
				t.Fatal(actualError)
			}

			assert.Equal(t, &db.GroupsResult{
				Groups: []models.Group{},
			}, actualOutput)
		})
	}
}

func TestUpdateGroup(t *testing.T) {

	originalGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-id",
		},
		Name:        "group-name",
		FullPath:    "root-group/group-name",
		Description: "This is the old description",
		RunnerTags:  []string{"tag1"},
	}

	updatedGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "group-id",
		},
		Name:        "group-name",
		FullPath:    "root-group/group-name",
		Description: "This is the new description",
		RunnerTags:  []string{"tag2", "tag3"},
	}

	type testCase struct {
		name            string
		foundGroup      *models.Group
		authError       error
		updateError     error
		expectGroup     *models.Group
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "successfully update a group",
			foundGroup:  originalGroup,
			expectGroup: updatedGroup,
		},
		{
			name:            "group does not exist",
			updateError:     errors.New("group not found", errors.WithErrorCode(errors.ENotFound)),
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

			mockGroups := db.NewMockGroups(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateGroupPermission, mock.Anything).
				Return(test.authError)

			mockCaller.On("GetSubject").Return("testsubject").Maybe()

			mockTransactions.On("BeginTx", mock.Anything).
				Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).
				Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).
				Return(nil).Maybe()

			mockGroups.On("UpdateGroup", mock.Anything, updatedGroup).
				Return(updatedGroup, test.updateError).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything,
				&activityevent.CreateActivityEventInput{
					NamespacePath: &originalGroup.FullPath,
					Action:        models.ActionUpdate,
					TargetType:    models.TargetGroup,
					TargetID:      originalGroup.Metadata.ID,
				},
			).Return(&models.ActivityEvent{}, nil).Maybe()

			dbClient := &db.Client{
				Groups:       mockGroups,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := &service{
				dbClient:        dbClient,
				logger:          logger,
				activityService: mockActivityEvents,
			}

			actualUpdated, err := service.UpdateGroup(auth.WithCaller(ctx, mockCaller), updatedGroup)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, updatedGroup, actualUpdated)
		})
	}
}

func TestMigrateGroup(t *testing.T) {
	testGroupID := "test-group-id"
	testGroupName := "test-group-name"
	oldParentID := "old-parent-id"
	testGroupOldPath := "old-parent-path/" + testGroupName

	testGroup := models.Group{
		Metadata: models.ResourceMetadata{ID: testGroupID},
		Name:     testGroupName,
		ParentID: oldParentID,
		FullPath: testGroupOldPath,
	}

	newParentID := "new-parent-id"
	newParentName := "new-parent-name"
	newParentPath := "new-grandparent-name/" + newParentName

	testNewParent := models.Group{
		Metadata: models.ResourceMetadata{ID: newParentID},
		Name:     newParentName,
		FullPath: newParentPath,
	}

	loopParentID := "loop-parent-id"
	loopParentName := "loop-parent-name"
	loopParentPath := testGroupOldPath + "/something/" + loopParentName

	loopParent := models.Group{
		Metadata: models.ResourceMetadata{ID: loopParentID},
		Name:     loopParentName,
		FullPath: loopParentPath,
	}

	// Test cases
	tests := []struct {
		newParentID              *string
		expectGroup              *models.Group
		name                     string
		expectErrorCode          errors.CodeType
		inputGroup               models.Group
		limit                    int // same for both siblings and depth
		newParentChildren        int32
		injectChildDepth         int // set to -1 to NOT do a mock.On GetChildDepth
		exceedsDepthFromRoot     bool
		exceedsChildDepth        bool
		isUserAdmin              bool
		isGroupOwner             bool
		isCallerDeployerOfParent bool
	}{
		{
			name:             "successful move to root",
			inputGroup:       testGroup,
			newParentID:      nil,
			isUserAdmin:      true,
			isGroupOwner:     true,
			injectChildDepth: 4,
			expectGroup: &models.Group{
				Metadata: models.ResourceMetadata{ID: testGroupID},
				Name:     testGroupName,
				ParentID: "",
				FullPath: testGroupName,
			},
		},
		{
			name:                     "successful move to non-root",
			inputGroup:               testGroup,
			newParentID:              &newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			limit:                    5,
			newParentChildren:        5,
			injectChildDepth:         2, // new grandparent, new parent, nomad, two levels of descendants
			expectGroup: &models.Group{
				Metadata: models.ResourceMetadata{ID: testGroupID},
				Name:     testGroupName,
				ParentID: newParentID,
				FullPath: newParentPath + "/" + testGroupName,
			},
		},
		{
			name:             "caller is not owner of group to be moved",
			inputGroup:       testGroup,
			newParentID:      nil,
			isGroupOwner:     false,
			injectChildDepth: -1,
			expectErrorCode:  errors.EForbidden,
		},
		{
			name:                     "new parent group is the same as the group to be moved",
			inputGroup:               testGroup,
			newParentID:              &testGroupID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			injectChildDepth:         -1,
			expectErrorCode:          errors.EInvalid,
		},
		{
			name:                     "new parent group is descendant of group to be moved",
			inputGroup:               testGroup,
			newParentID:              &loopParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			injectChildDepth:         -1,
			expectErrorCode:          errors.EInvalid,
		},
		{
			name:                     "caller is not deployer (or better) of new parent group",
			inputGroup:               testGroup,
			newParentID:              &newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: false,
			injectChildDepth:         -1,
			expectErrorCode:          errors.EForbidden,
		},
		{
			name:             "caller is not admin but tried to move group to root",
			inputGroup:       testGroup,
			newParentID:      nil,
			isGroupOwner:     true,
			injectChildDepth: -1,
			expectErrorCode:  errors.EForbidden,
		},
		{
			name:                     "exceeds limit on subgroups within direct parent",
			inputGroup:               testGroup,
			newParentID:              &newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			limit:                    5,
			newParentChildren:        6,
			injectChildDepth:         1,
			expectGroup: &models.Group{ // to avoid GetDepth seeing a nil group
				Metadata: models.ResourceMetadata{ID: testGroupID},
				Name:     testGroupName,
				ParentID: "",
				FullPath: testGroupName,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                     "exceeds limit on depth due to ancestors",
			inputGroup:               testGroup,
			newParentID:              &newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			limit:                    5,
			newParentChildren:        5,
			injectChildDepth:         1, // the group and one child level
			exceedsDepthFromRoot:     true,
			expectGroup: &models.Group{ // to avoid GetDepth seeing a nil group
				Metadata: models.ResourceMetadata{ID: testGroupID},
				Name:     testGroupName,
				FullPath: "a/b/c/d/e", // just exceeds depth limit: 5 + 2 - 1
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                     "exceeds limit on depth due to descendants",
			inputGroup:               testGroup,
			newParentID:              &newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			limit:                    5,
			newParentChildren:        5,
			injectChildDepth:         4, // just exceeds limit: 2 + 4
			exceedsChildDepth:        true,
			expectGroup: &models.Group{ // to avoid GetDepth seeing a nil group
				Metadata: models.ResourceMetadata{ID: testGroupID},
				Name:     testGroupName,
				FullPath: "a/b",
			},
			expectErrorCode: errors.EInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var groupAccessError, parentAccessError error
			if !test.isGroupOwner {
				groupAccessError = errors.New("test user is not owner of group being moved", errors.WithErrorCode(errors.EForbidden))
			}
			if !test.isCallerDeployerOfParent {
				parentAccessError = errors.New("test user is not deployer of new parent", errors.WithErrorCode(errors.EForbidden))
			}

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockResourceLimits := db.NewMockResourceLimits(t)

			perms := []models.Permission{models.DeleteGroupPermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(groupAccessError)

			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(nil)

			perms = []models.Permission{models.CreateGroupPermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(parentAccessError)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockGroups.On("GetGroupByID", mock.Anything, test.inputGroup.Metadata.ID).Return(&test.inputGroup, nil)
			mockGroups.On("GetGroupByID", mock.Anything, newParentID).Return(&testNewParent, nil)
			mockGroups.On("GetGroupByID", mock.Anything, loopParentID).Return(&loopParent, nil)

			if test.injectChildDepth >= 0 {
				mockGroups.On("GetChildDepth", mock.Anything, mock.Anything).Return(test.injectChildDepth, nil)
			}

			var newParent *models.Group
			if test.newParentID != nil {
				newParent = &models.Group{
					Metadata: models.ResourceMetadata{
						ID: *test.newParentID,
					},
					FullPath: newParentPath,
					Name:     newParentName,
				}

				// called from inside checkParentSubgroupLimit
				mockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(
					&db.GroupsResult{
						PageInfo: &pagination.PageInfo{
							TotalCount: test.newParentChildren,
						},
					}, nil)

				if test.limit > 0 {
					mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
						Return(&models.ResourceLimit{Value: test.limit}, nil)
				}

				// called from inside getChildDepth
				mockGroups.On("GetGroups", mock.Anything, &db.GetGroupsInput{
					Filter: &db.GroupFilter{
						ParentID: &testGroupID,
					},
				}).Return(&db.GroupsResult{Groups: []models.Group{}}, nil)
			}

			mockGroups.On("MigrateGroup", mock.Anything, &test.inputGroup, newParent).Return(test.expectGroup, nil)

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
			service := NewService(logger, &dbClient, limiter, nil, &mockActivityEvents, nil)

			migrated, err := service.MigrateGroup(auth.WithCaller(ctx, testCaller),
				test.inputGroup.Metadata.ID, test.newParentID)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.Equal(t, test.expectGroup, migrated)
			}
		})
	}
}

func TestGetDriftDetectionEnabledSetting(t *testing.T) {
	group := models.Group{
		Metadata: models.ResourceMetadata{ID: "group-1"},
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

			mockCaller.On("RequirePermission", mock.Anything, models.ViewGroupPermission, mock.Anything).Return(test.authError)

			mockInheritedSettingsResolver.On("GetDriftDetectionEnabled", mock.Anything, &group).Return(test.expectSetting, nil).Maybe()

			service := NewService(testLogger, nil, nil, nil, nil, mockInheritedSettingsResolver)

			setting, err := service.GetDriftDetectionEnabledSetting(auth.WithCaller(ctx, mockCaller), &group)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectSetting, setting)
		})
	}
}
