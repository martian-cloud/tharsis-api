//go:build integration

package db

import (
	"context"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for ServiceAccountSortableField
func (sa ServiceAccountSortableField) getValue() string {
	return string(sa)
}

func TestServiceAccounts_CreateServiceAccount(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-sa",
		Description: "test group for service account",
		FullPath:    "test-group-sa",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		saName          string
		description     string
		groupID         string
	}

	testCases := []testCase{
		{
			name:        "create service account",
			saName:      "test-sa",
			description: "test service account",
			groupID:     group.Metadata.ID,
		},
		{
			name:            "create service account with invalid group ID",
			saName:          "invalid-sa",
			description:     "invalid service account",
			groupID:         invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			sa, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
				Name:        test.saName,
				Description: test.description,
				GroupID:     test.groupID,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, sa)

			assert.Equal(t, test.saName, sa.Name)
			assert.Equal(t, test.description, sa.Description)
			assert.Equal(t, test.groupID, sa.GroupID)
			assert.NotEmpty(t, sa.Metadata.ID)
		})
	}
}

func TestServiceAccounts_UpdateServiceAccount(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and service account for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-sa-update",
		Description: "test group for service account update",
		FullPath:    "test-group-sa-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdSA, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:        "test-sa-update",
		Description: "original description",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		description     string
	}

	testCases := []testCase{
		{
			name:        "update service account",
			version:     createdSA.Metadata.Version,
			description: "updated description",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			description:     "should not update",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			saToUpdate := *createdSA
			saToUpdate.Metadata.Version = test.version
			saToUpdate.Description = test.description

			updatedSA, err := testClient.client.ServiceAccounts.UpdateServiceAccount(ctx, &saToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedSA)

			assert.Equal(t, test.description, updatedSA.Description)
			assert.Equal(t, createdSA.Metadata.Version+1, updatedSA.Metadata.Version)
		})
	}
}

func TestServiceAccounts_DeleteServiceAccount(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and service account for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-sa-delete",
		Description: "test group for service account delete",
		FullPath:    "test-group-sa-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdSA, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:        "test-sa-delete",
		Description: "service account to delete",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:    "delete service account",
			id:      createdSA.Metadata.ID,
			version: createdSA.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdSA.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ServiceAccounts.DeleteServiceAccount(ctx, &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			// Verify service account was deleted
			serviceAccount, err := testClient.client.ServiceAccounts.GetServiceAccountByID(ctx, test.id)
			assert.Nil(t, serviceAccount)
			assert.Nil(t, err)
		})
	}
}

func TestServiceAccounts_GetServiceAccountByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-service-account-get-by-id",
		Description: "test group for service account get by id",
		FullPath:    "test-group-service-account-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a service account for testing
	createdServiceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:        "test-service-account-get-by-id",
		Description: "test service account for get by id",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		id                   string
		expectServiceAccount bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by id",
			id:                   createdServiceAccount.Metadata.ID,
			expectServiceAccount: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			serviceAccount, err := testClient.client.ServiceAccounts.GetServiceAccountByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectServiceAccount {
				require.NotNil(t, serviceAccount)
				assert.Equal(t, test.id, serviceAccount.Metadata.ID)
			} else {
				assert.Nil(t, serviceAccount)
			}
		})
	}
}

func TestServiceAccounts_GetServiceAccounts(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-service-accounts-list",
		Description: "test group for service accounts list",
		FullPath:    "test-group-service-accounts-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test service accounts
	serviceAccounts := []models.ServiceAccount{
		{
			Name:        "test-service-account-1",
			Description: "test service account 1",
			GroupID:     group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-service-account-2",
			Description: "test service account 2",
			GroupID:     group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdServiceAccounts := []models.ServiceAccount{}
	for _, serviceAccount := range serviceAccounts {
		created, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &serviceAccount)
		require.NoError(t, err)
		createdServiceAccounts = append(createdServiceAccounts, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetServiceAccountsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all service accounts",
			input:       &GetServiceAccountsInput{},
			expectCount: len(createdServiceAccounts),
		},
		{
			name: "filter by search",
			input: &GetServiceAccountsInput{
				Filter: &ServiceAccountFilter{
					Search: ptr.String("test-service-account-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by service account IDs",
			input: &GetServiceAccountsInput{
				Filter: &ServiceAccountFilter{
					ServiceAccountIDs: []string{createdServiceAccounts[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by namespace paths",
			input: &GetServiceAccountsInput{
				Filter: &ServiceAccountFilter{
					NamespacePaths: []string{group.FullPath},
				},
			},
			expectCount: len(createdServiceAccounts),
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.ServiceAccounts.GetServiceAccounts(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ServiceAccounts, test.expectCount)
		})
	}
}

func TestServiceAccounts_GetServiceAccountsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-service-accounts-pagination",
		Description: "test group for service accounts pagination",
		FullPath:    "test-group-service-accounts-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
			Name:        fmt.Sprintf("test-service-account-%d", i),
			Description: fmt.Sprintf("test service account %d", i),
			GroupID:     group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	// Only test CreatedAt and UpdatedAt fields to avoid GROUP_LEVEL complexity
	sortableFields := []sortableField{
		ServiceAccountSortableFieldCreatedAtAsc,
		ServiceAccountSortableFieldCreatedAtDesc,
		ServiceAccountSortableFieldUpdatedAtAsc,
		ServiceAccountSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := ServiceAccountSortableField(sortByField.getValue())

		result, err := testClient.client.ServiceAccounts.GetServiceAccounts(ctx, &GetServiceAccountsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ServiceAccounts {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestServiceAccounts_GetServiceAccountByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-service-account-get-by-trn",
		Description: "test group for service account get by trn",
		FullPath:    "test-group-service-account-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a service account for testing
	createdServiceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:        "test-service-account-get-by-trn",
		Description: "test service account for get by trn",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode      errors.CodeType
		name                 string
		trn                  string
		expectServiceAccount bool
	}

	testCases := []testCase{
		{
			name:                 "get resource by TRN",
			trn:                  createdServiceAccount.Metadata.TRN,
			expectServiceAccount: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:service_account:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			serviceAccount, err := testClient.client.ServiceAccounts.GetServiceAccountByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectServiceAccount {
				require.NotNil(t, serviceAccount)
				assert.Equal(t, createdServiceAccount.Metadata.ID, serviceAccount.Metadata.ID)
			} else {
				assert.Nil(t, serviceAccount)
			}
		})
	}
}

func TestServiceAccounts_AssignServiceAccountToRunner(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-service-account-assign",
		Description: "test group for service account assign",
		FullPath:    "test-group-service-account-assign",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a service account for testing
	createdServiceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:        "test-service-account-assign",
		Description: "test service account for assign",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a runner for testing
	createdRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:        "test-runner-assign",
		Description: "test runner for assign",
		GroupID:     &group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode  errors.CodeType
		name             string
		serviceAccountID string
		runnerID         string
	}

	testCases := []testCase{
		{
			name:             "assign service account to runner",
			serviceAccountID: createdServiceAccount.Metadata.ID,
			runnerID:         createdRunner.Metadata.ID,
		},
		{
			name:             "assign with invalid service account id",
			serviceAccountID: invalidID,
			runnerID:         createdRunner.Metadata.ID,
			expectErrorCode:  errors.EInternal,
		},
		{
			name:             "assign with invalid runner id",
			serviceAccountID: createdServiceAccount.Metadata.ID,
			runnerID:         invalidID,
			expectErrorCode:  errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ServiceAccounts.AssignServiceAccountToRunner(ctx, test.serviceAccountID, test.runnerID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestServiceAccounts_UnassignServiceAccountFromRunner(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-service-account-unassign",
		Description: "test group for service account unassign",
		FullPath:    "test-group-service-account-unassign",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a service account for testing
	createdServiceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:        "test-service-account-unassign",
		Description: "test service account for unassign",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a runner for testing
	createdRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:        "test-runner-unassign",
		Description: "test runner for unassign",
		GroupID:     &group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// First assign the service account to the runner
	err = testClient.client.ServiceAccounts.AssignServiceAccountToRunner(ctx, createdServiceAccount.Metadata.ID, createdRunner.Metadata.ID)
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode  errors.CodeType
		name             string
		serviceAccountID string
		runnerID         string
	}

	testCases := []testCase{
		{
			name:             "unassign service account from runner",
			serviceAccountID: createdServiceAccount.Metadata.ID,
			runnerID:         createdRunner.Metadata.ID,
		},
		{
			name:             "unassign with invalid service account id",
			serviceAccountID: invalidID,
			runnerID:         createdRunner.Metadata.ID,
			expectErrorCode:  errors.EInternal,
		},
		{
			name:             "unassign with invalid runner id",
			serviceAccountID: createdServiceAccount.Metadata.ID,
			runnerID:         invalidID,
			expectErrorCode:  errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ServiceAccounts.UnassignServiceAccountFromRunner(ctx, test.serviceAccountID, test.runnerID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
