package group

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
)

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

			mockNamespaceMemberships := namespacemembership.MockService{}
			mockNamespaceMemberships.Test(t)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockGroups.On("CreateGroup", mock.Anything, &test.input).Return(&test.input, nil)

			createNamespaceMembershipInput := &namespacemembership.CreateNamespaceMembershipInput{
				NamespacePath: test.input.FullPath,
				Role:          models.OwnerRole,
				User:          test.caller.User,
			}
			mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, createNamespaceMembershipInput).Return(nil, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			dbClient := db.Client{
				Groups:       &mockGroups,
				Transactions: &mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockNamespaceMemberships, &mockActivityEvents)

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
		name            string
		expectErrorCode string
		input           models.Group
		isAuthorized    bool
	}{
		{
			name: "create group",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group0",
			},
			isAuthorized: true,
		},
		{
			name: "caller is not authorized to create group",
			input: models.Group{
				Name:     "group1",
				Metadata: models.ResourceMetadata{ID: "group1"},
				ParentID: "group0",
			},
			isAuthorized:    false,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.MockCaller{}
			mockCaller.On("GetSubject").Return("testsubject")
			mockCaller.Test(t)

			retFunc := func(_ context.Context, _ string, _ models.Role) error {
				if test.isAuthorized {
					return nil
				}
				return errors.NewError(errors.EForbidden, "Forbidden")
			}
			mockCaller.On("RequireAccessToGroup", mock.Anything, test.input.ParentID, models.DeployerRole).Return(retFunc)

			mockGroups := db.MockGroups{}
			mockGroups.Test(t)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockGroups.On("CreateGroup", mock.Anything, &test.input).Return(&test.input, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			dbClient := db.Client{
				Groups:       &mockGroups,
				Transactions: &mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, nil, &mockActivityEvents)

			group, err := service.CreateGroup(auth.WithCaller(ctx, &mockCaller), &test.input)
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
				groupAccessError = errors.NewError(errors.EForbidden, "test user is not owner of group being moved")
			}
			if !test.isCallerDeployerOfParent {
				parentAccessError = errors.NewError(errors.EForbidden, "test user is not deployer of new parent")
			}

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockAuthorizer.On("RequireAccessToNamespace",
				mock.Anything, testGroupOldPath, models.OwnerRole).Return(groupAccessError)

			mockAuthorizer.On("RequireAccessToNamespace",
				mock.Anything, testGroupOldPath, models.DeployerRole).Return(nil)

			mockAuthorizer.On("RequireAccessToNamespace",
				mock.Anything, newParentPath, models.DeployerRole).Return(parentAccessError)

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
