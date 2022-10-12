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

			dbClient := db.Client{
				Groups:       &mockGroups,
				Transactions: &mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockNamespaceMemberships)

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

			mockGroups.On("CreateGroup", mock.Anything, &test.input).Return(&test.input, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			dbClient := db.Client{
				Groups:       &mockGroups,
				Transactions: &mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, nil)

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
