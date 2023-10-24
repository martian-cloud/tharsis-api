package namespacemembership

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestCreateNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		expectNamespaceMembership *models.NamespaceMembership
		input                     CreateNamespaceMembershipInput
		name                      string
		expectErrorCode           errors.CodeType
		hasOwnerRole              bool
	}{
		{
			name: "create user namespace membership with owner role in top level namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				User:          &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path:    "ns1",
					GroupID: ptr.String("group1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "create service account namespace membership with owner role in nested namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1/ns11/ns111",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata:     models.ResourceMetadata{ID: "serviceAccount1"},
					ResourcePath: "ns1/ns11/serviceAccount",
				},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path:    "ns1/ns11/ns111",
					GroupID: ptr.String("group1"),
				},
				RoleID:           models.OwnerRoleID.String(),
				ServiceAccountID: ptr.String("serviceAccount1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "create service account namespace membership with owner role in top-level namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata:     models.ResourceMetadata{ID: "serviceAccount1"},
					ResourcePath: "ns1/serviceAccount",
				},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path:    "ns1",
					GroupID: ptr.String("group1"),
				},
				RoleID:           models.OwnerRoleID.String(),
				ServiceAccountID: ptr.String("serviceAccount1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "no owner role",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				User:          &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "missing user and service account",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "user and service account can't both be defined",
			input: CreateNamespaceMembershipInput{
				NamespacePath:  "ns1",
				RoleID:         models.OwnerRoleID.String(),
				User:           &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
				ServiceAccount: &models.ServiceAccount{Metadata: models.ResourceMetadata{ID: "serviceAccount1"}},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to create service account namespace membership in a namespace it doesn't exist in",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata:     models.ResourceMetadata{ID: "serviceAccount1"},
					ResourcePath: "ns2/serviceAccount",
				},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to create service account namespace membership in a nested namespace it doesn't exist in",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata:     models.ResourceMetadata{ID: "serviceAccount1"},
					ResourcePath: "ns1/ns11/serviceAccount",
				},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			var authError error
			if !test.hasOwnerRole {
				authError = errors.New("not authorized", errors.WithErrorCode(errors.EForbidden))
			}
			mockCaller.On("RequirePermission", mock.Anything, permissions.CreateNamespaceMembershipPermission, mock.Anything).Return(authError)

			var userID, serviceAccountID *string
			if test.input.User != nil {
				userID = &test.input.User.Metadata.ID
			} else if test.input.ServiceAccount != nil {
				serviceAccountID = &test.input.ServiceAccount.Metadata.ID
			}

			mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, &db.CreateNamespaceMembershipInput{
				NamespacePath:    test.input.NamespacePath,
				RoleID:           test.input.RoleID,
				UserID:           userID,
				ServiceAccountID: serviceAccountID,
			}).Return(test.expectNamespaceMembership, nil)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)
			// The mocks are enabled by the above function.

			mockUsers := db.MockUsers{}
			mockTransactions.Test(t)

			mockServiceAccounts := db.MockServiceAccounts{}
			mockServiceAccounts.Test(t)

			mockRoles := db.MockRoles{}
			mockRoles.Test(t)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Transactions:         &mockTransactions,
				Users:                &mockUsers,
				ServiceAccounts:      &mockServiceAccounts,
				Roles:                &mockRoles,
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockRoles.On("GetRoleByID", mock.Anything, test.input.RoleID).Return(&models.Role{Name: "role-1"}, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockUsers.On("GetUserByID", mock.Anything, mock.Anything).Return(&models.User{
				Username: "mock-user",
				Email:    "mock-user@example.invalid",
			}, nil)

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).Return(&models.ServiceAccount{
				Name: "mock-service-account-name",
			}, nil)

			// If a new test case is added that uses a team principal, will need to mock GetTeamByID here.

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockActivityEvents)

			namespaceMembership, err := service.CreateNamespaceMembership(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectNamespaceMembership, namespaceMembership)
		})
	}
}

func TestUpdateNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		input                *models.NamespaceMembership
		current              *models.NamespaceMembership
		name                 string
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
		hasOwnerRole         bool
	}{
		{
			name: "update namespace membership by reducing role from owner to deployer",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
				{Metadata: models.ResourceMetadata{ID: "2"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole: true,
		},
		{
			name: "update namespace membership by reducing role from owner to deployer in nested group",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11",
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "should not be able to update namespace membership because only one owner exists",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "current namespace membership not found",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "should not be able to update namespace membership because caller doesn't have owner role",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockRoles := db.MockRoles{}
			mockRoles.Test(t)

			var authError error
			if !test.hasOwnerRole {
				authError = errors.New("not authorized", errors.WithErrorCode(errors.EForbidden))
			}

			mockCaller.On("RequirePermission", mock.Anything, permissions.UpdateNamespaceMembershipPermission, mock.Anything).Return(authError)

			mockNamespaceMemberships.On("GetNamespaceMembershipByID", mock.Anything, test.input.Metadata.ID).Return(test.current, nil)

			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					NamespacePaths: []string{test.input.Namespace.Path},
				},
			}
			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			mockNamespaceMemberships.On("UpdateNamespaceMembership", mock.Anything, test.input).Return(test.input, nil)

			mockRoles.On("GetRoleByID", mock.Anything, test.input.RoleID).Return(&models.Role{
				Metadata: models.ResourceMetadata{ID: test.input.RoleID},
				Name:     "role-1",
			}, nil)

			if test.current != nil {
				mockRoles.On("GetRoleByID", mock.Anything, test.current.RoleID).Return(&models.Role{
					Metadata: models.ResourceMetadata{ID: test.current.RoleID},
					Name:     "role-2",
				}, nil)
			}

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)
			// The mocks are enabled by the above function.

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Transactions:         &mockTransactions,
				Roles:                &mockRoles,
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockActivityEvents)

			namespaceMembership, err := service.UpdateNamespaceMembership(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			} else if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.input, namespaceMembership)
		})
	}
}

func TestDeleteNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		input                *models.NamespaceMembership
		name                 string
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
		hasOwnerRole         bool
	}{
		{
			name: "delete namespace membership",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:    "ns1",
					GroupID: ptr.String("group1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
				{Metadata: models.ResourceMetadata{ID: "2"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole: true,
		},
		{
			name: "delete namespace membership in nested group",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:    "ns1/ns11",
					GroupID: ptr.String("group1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "should not be able to delete namespace membership because only one owner exists",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:        "ns1",
					WorkspaceID: ptr.String("ws1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to delete namespace membership because caller doesn't have owner role",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:        "ns1",
					WorkspaceID: ptr.String("ws1"),
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			var authError error
			if !test.hasOwnerRole {
				authError = errors.New("not authorized", errors.WithErrorCode(errors.EForbidden))
			}

			mockCaller.On("RequirePermission", mock.Anything, permissions.DeleteNamespaceMembershipPermission, mock.Anything).Return(authError)

			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					NamespacePaths: []string{test.input.Namespace.Path},
				},
			}
			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			mockNamespaceMemberships.On("DeleteNamespaceMembership", mock.Anything, test.input).Return(nil)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)
			// The mocks are enabled by the above function.

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Transactions:         &mockTransactions,
			}

			mockActivityEvents := activityevent.MockService{}
			mockActivityEvents.Test(t)

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).Return(&models.ActivityEvent{}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockActivityEvents)

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			err := service.DeleteNamespaceMembership(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
