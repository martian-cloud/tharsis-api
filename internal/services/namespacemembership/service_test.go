package namespacemembership

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestCreateNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		expectNamespaceMembership *models.NamespaceMembership
		input                     CreateNamespaceMembershipInput
		name                      string
		expectErrorCode           string
		hasOwnerRole              bool
	}{
		{
			name: "create user namespace membership with owner role in top level namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				Role:          models.OwnerRole,
				User:          &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:   models.OwnerRole,
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "create service account namespace membership with owner role in nested namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1/ns11/ns111",
				Role:          models.OwnerRole,
				ServiceAccount: &models.ServiceAccount{
					Metadata:     models.ResourceMetadata{ID: "serviceAccount1"},
					ResourcePath: "ns1/ns11/serviceAccount",
				},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11/ns111",
				},
				Role:             models.OwnerRole,
				ServiceAccountID: ptr.String("serviceAccount1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "create service account namespace membership with owner role in top-level namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				Role:          models.OwnerRole,
				ServiceAccount: &models.ServiceAccount{
					Metadata:     models.ResourceMetadata{ID: "serviceAccount1"},
					ResourcePath: "ns1/serviceAccount",
				},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:             models.OwnerRole,
				ServiceAccountID: ptr.String("serviceAccount1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "no owner role",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				Role:          models.OwnerRole,
				User:          &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "missing user and service account",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				Role:          models.OwnerRole,
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "user and service account can't both be defined",
			input: CreateNamespaceMembershipInput{
				NamespacePath:  "ns1",
				Role:           models.OwnerRole,
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
				Role:          models.OwnerRole,
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
				Role:          models.OwnerRole,
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

			retFunc := func(_ context.Context, _ string, _ models.Role) error {
				if test.hasOwnerRole {
					return nil
				}
				return errors.NewError(errors.EForbidden, "Forbidden")
			}
			mockCaller.On("RequireAccessToNamespace", mock.Anything, test.input.NamespacePath, models.OwnerRole).Return(retFunc)

			var userID, serviceAccountID *string
			if test.input.User != nil {
				userID = &test.input.User.Metadata.ID
			} else if test.input.ServiceAccount != nil {
				serviceAccountID = &test.input.ServiceAccount.Metadata.ID
			}

			mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, &db.CreateNamespaceMembershipInput{
				NamespacePath:    test.input.NamespacePath,
				Role:             test.input.Role,
				UserID:           userID,
				ServiceAccountID: serviceAccountID,
			}).Return(test.expectNamespaceMembership, nil)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient)

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
		expectErrorCode      string
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
				Role:   models.DeployerRole,
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:   models.OwnerRole,
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, Role: models.OwnerRole},
				{Metadata: models.ResourceMetadata{ID: "2"}, Role: models.OwnerRole},
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
				Role:   models.DeployerRole,
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11",
				},
				Role:   models.OwnerRole,
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
				Role:   models.DeployerRole,
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:   models.OwnerRole,
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, Role: models.OwnerRole},
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
				Role:   models.DeployerRole,
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
				Role:   models.DeployerRole,
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

			retFunc := func(_ context.Context, _ string, _ models.Role) error {
				if test.hasOwnerRole {
					return nil
				}
				return errors.NewError(errors.EForbidden, "Forbidden")
			}
			mockCaller.On("RequireAccessToNamespace", mock.Anything, test.input.Namespace.Path, models.OwnerRole).Return(retFunc)

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

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient)

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
		expectErrorCode      string
		namespaceMemberships []models.NamespaceMembership
		hasOwnerRole         bool
	}{
		{
			name: "delete namespace membership",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:   models.OwnerRole,
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, Role: models.OwnerRole},
				{Metadata: models.ResourceMetadata{ID: "2"}, Role: models.OwnerRole},
			},
			hasOwnerRole: true,
		},
		{
			name: "delete namespace membership in nested group",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11",
				},
				Role:   models.OwnerRole,
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "should not be able to delete namespace membership because only one owner exists",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:   models.OwnerRole,
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, Role: models.OwnerRole},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to delete namespace membership because caller doesn't have owner role",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				Role:   models.DeployerRole,
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

			retFunc := func(_ context.Context, _ string, _ models.Role) error {
				if test.hasOwnerRole {
					return nil
				}
				return errors.NewError(errors.EForbidden, "Forbidden")
			}
			mockCaller.On("RequireAccessToNamespace", mock.Anything, test.input.Namespace.Path, models.OwnerRole).Return(retFunc)

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

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient)

			err := service.DeleteNamespaceMembership(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
