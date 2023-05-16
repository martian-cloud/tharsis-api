package group

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type mockDBClient struct {
	*db.Client
	MockTransactions *db.MockTransactions
	MockGroups       *db.MockGroups
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)

	mockGroups := db.MockGroups{}
	mockGroups.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions: &mockTransactions,
			Groups:       &mockGroups,
		},
		MockTransactions: &mockTransactions,
		MockGroups:       &mockGroups,
	}
}

func TestCreateTopLevelGroup(t *testing.T) {
	// Test cases
	tests := []struct {
		caller          *auth.UserCaller
		name            string
		expectErrorCode string
		input           models.Group
	}{
		{
			name: "create group",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
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

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, mockNamespaceMemberships, mockActivityEvents)

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
		expectErrorCode string
		input           models.Group
	}{
		{
			name: "create group",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group0",
			},
		},
		{
			name: "caller is not authorized to create group",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group0",
			},
			authError:       errors.New(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateGroupPermission, mock.Anything).Return(test.authError)

			mockGroups := db.NewMockGroups(t)
			mockTransactions := db.NewMockTransactions(t)
			mockActivityEvents := activityevent.NewMockService(t)

			if test.authError == nil {
				mockCaller.On("GetSubject").Return("testsubject")

				mockGroups.On("CreateGroup", mock.Anything, &test.input).Return(&test.input, nil)

				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)
			}

			dbClient := db.Client{
				Groups:       mockGroups,
				Transactions: mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, nil, mockActivityEvents)

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
	parentGroupID := "this-is-a-fake-parent-group-ID"
	parentGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: parentGroupID,
		},
	}
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
				ParentGroup: parentGroup,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search absent/nil, with root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					RootOnly: true,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search empty, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search empty, with root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &emptySearch,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
					RootOnly: true,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search non-empty, no root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
				},
			},
		},
		{
			name:       "admin caller, with parent group, search non-empty, with root-only",
			callerType: "admin",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &nonEmptySearch,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
					RootOnly: true,
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
					NamespaceIDs: []string{},
					RootOnly:     true,
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
					NamespaceIDs: []string{},
					RootOnly:     true,
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
					NamespaceIDs: []string{},
					RootOnly:     true,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search absent/nil, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search absent/nil, with root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					RootOnly: true,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search empty, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search empty, with root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &emptySearch,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
					RootOnly: true,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search non-empty, no root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
				},
			},
		},
		{
			name:       "user member caller, with parent group, search non-empty, with root-only",
			callerType: "user",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &nonEmptySearch,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
					RootOnly: true,
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
					NamespaceIDs: []string{},
					RootOnly:     true,
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
					NamespaceIDs: []string{},
					RootOnly:     true,
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
					NamespaceIDs: []string{},
					RootOnly:     true,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search absent/nil, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search absent/nil, with root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					RootOnly: true,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search empty, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &emptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search empty, with root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &emptySearch,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &emptySearch,
					RootOnly: true,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search non-empty, no root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &nonEmptySearch,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
				},
			},
		},
		{
			name:       "service account member caller, with parent group, search non-empty, with root-only",
			callerType: "service-account",
			svcInput: &GetGroupsInput{
				ParentGroup: parentGroup,
				Search:      &nonEmptySearch,
				RootOnly:    true,
			},
			dbInput: &db.GetGroupsInput{
				Filter: &db.GroupFilter{
					ParentID: &parentGroupID,
					Search:   &nonEmptySearch,
					RootOnly: true,
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient := buildDBClientWithMocks(t)

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{}, nil)

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
				)
			case "service-account":
				testCaller = auth.NewServiceAccountCaller(
					serviceAccountMemberID,
					serviceAccountPath,
					&mockAuthorizer,
					dbClient.Client,
				)
			default:
				assert.Fail(t, "invalid caller type in test")
			}

			// If the service layer sends wrong/unexpected auth values, the last argument will prevent the
			// mocking of the function from taking effect.  Because the mock state is cleared between test
			// cases, an earlier .On(...) won't mask an error by the service layer.
			dbClient.MockGroups.On("GetGroups", mock.Anything, test.dbInput).
				Return(func(_ context.Context, input *db.GetGroupsInput) *db.GroupsResult {
					return &db.GroupsResult{
						Groups: []models.Group{},
					}
				}, nil,
				)

			logger, _ := logger.NewForTest()
			activityService := activityevent.NewService(dbClient.Client, logger)
			namespaceMembershipService := namespacemembership.NewService(logger, dbClient.Client, activityService)
			service := NewService(logger, dbClient.Client, namespaceMembershipService, activityService)

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
		expectErrorCode          string
		inputGroup               models.Group
		isUserAdmin              bool
		isGroupOwner             bool
		isCallerDeployerOfParent bool
	}{
		{
			name:         "successful move to root",
			inputGroup:   testGroup,
			newParentID:  nil,
			isUserAdmin:  true,
			isGroupOwner: true,
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
			expectGroup: &models.Group{
				Metadata: models.ResourceMetadata{ID: testGroupID},
				Name:     testGroupName,
				ParentID: newParentID,
				FullPath: newParentPath + "/" + testGroupName,
			},
		},
		{
			name:            "caller is not owner of group to be moved",
			inputGroup:      testGroup,
			newParentID:     nil,
			isGroupOwner:    false,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:                     "new parent group is the same as the group to be moved",
			inputGroup:               testGroup,
			newParentID:              &testGroupID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			expectErrorCode:          errors.EInvalid,
		},
		{
			name:                     "new parent group is descendant of group to be moved",
			inputGroup:               testGroup,
			newParentID:              &loopParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: true,
			expectErrorCode:          errors.EInvalid,
		},
		{
			name:                     "caller is not deployer (or better) of new parent group",
			inputGroup:               testGroup,
			newParentID:              &newParentID,
			isGroupOwner:             true,
			isCallerDeployerOfParent: false,
			expectErrorCode:          errors.EForbidden,
		},
		{
			name:            "caller is not admin but tried to move group to root",
			inputGroup:      testGroup,
			newParentID:     nil,
			isGroupOwner:    true,
			expectErrorCode: errors.EForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var groupAccessError, parentAccessError error
			if !test.isGroupOwner {
				groupAccessError = errors.New(errors.EForbidden, "test user is not owner of group being moved")
			}
			if !test.isCallerDeployerOfParent {
				parentAccessError = errors.New(errors.EForbidden, "test user is not deployer of new parent")
			}

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			perms := []permissions.Permission{permissions.DeleteGroupPermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(groupAccessError)

			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(nil)

			perms = []permissions.Permission{permissions.CreateGroupPermission}
			mockAuthorizer.On("RequireAccess", mock.Anything, perms, mock.Anything).Return(parentAccessError)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockGroups.On("GetGroupByID", mock.Anything, test.inputGroup.Metadata.ID).Return(&test.inputGroup, nil)
			mockGroups.On("GetGroupByID", mock.Anything, newParentID).Return(&testNewParent, nil)
			mockGroups.On("GetGroupByID", mock.Anything, loopParentID).Return(&loopParent, nil)

			var newParent *models.Group
			if test.newParentID != nil {
				newParent = &models.Group{
					Metadata: models.ResourceMetadata{
						ID: *test.newParentID,
					},
					FullPath: newParentPath,
					Name:     newParentName,
				}
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

			dbClient := db.Client{
				Groups:       &mockGroups,
				Transactions: &mockTransactions,
			}

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
			)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, nil, &mockActivityEvents)

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
