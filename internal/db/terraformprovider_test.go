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

// getValue implements the sortableField interface for TerraformProviderSortableField
func (tp TerraformProviderSortableField) getValue() string {
	return string(tp)
}

func TestTerraformProviders_CreateProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider",
		Description: "test group for provider",
		FullPath:    "test-group-provider",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		providerName    string
		groupID         string
	}

	testCases := []testCase{
		{
			name:         "create provider",
			providerName: "test-provider",
			groupID:      group.Metadata.ID,
		},
		{
			name:            "create provider with invalid group ID",
			providerName:    "invalid-provider",
			groupID:         invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
				Name:        test.providerName,
				GroupID:     test.groupID,
				RootGroupID: group.Metadata.ID,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, provider)

			assert.Equal(t, test.providerName, provider.Name)
			assert.Equal(t, test.groupID, provider.GroupID)
			assert.NotEmpty(t, provider.Metadata.ID)
		})
	}
}

func TestTerraformProviders_UpdateProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-update",
		Description: "test group for provider update",
		FullPath:    "test-group-provider-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a provider for testing
	createdProvider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-update",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		providerID      string
		private         bool
	}

	testCases := []testCase{
		{
			name:       "update provider",
			providerID: createdProvider.Metadata.ID,
			version:    createdProvider.Metadata.Version,
			private:    true,
		},
		{
			name:            "would-be-duplicate-group-id-and-provider-name",
			providerID:      createdProvider.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			private:         false,
		},
		{
			name:            "negative, non-existent Terraform provider ID",
			providerID:      invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
			private:         false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			providerToUpdate := *createdProvider
			providerToUpdate.Metadata.ID = test.providerID
			providerToUpdate.Metadata.Version = test.version
			providerToUpdate.Private = test.private

			updatedProvider, err := testClient.client.TerraformProviders.UpdateProvider(ctx, &providerToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedProvider)

			assert.Equal(t, test.private, updatedProvider.Private)
			assert.Equal(t, createdProvider.Metadata.Version+1, updatedProvider.Metadata.Version)
		})
	}
}

func TestTerraformProviders_DeleteProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-delete",
		Description: "test group for provider delete",
		FullPath:    "test-group-provider-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a provider for testing
	createdProvider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-delete",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
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
			name:    "delete provider",
			id:      createdProvider.Metadata.ID,
			version: createdProvider.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdProvider.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "negative, non-existent Terraform provider ID",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviders.DeleteProvider(ctx, &models.TerraformProvider{
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

			// Verify provider was deleted
			provider, err := testClient.client.TerraformProviders.GetProviderByID(ctx, test.id)
			assert.Nil(t, provider)
			assert.Nil(t, err)
		})
	}
}

func TestTerraformProviders_GetProviderByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-get-by-id",
		Description: "test group for provider get by id",
		FullPath:    "test-group-provider-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for testing
	createdProvider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-get-by-id",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectProvider  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by id",
			id:             createdProvider.Metadata.ID,
			expectProvider: true,
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
			provider, err := testClient.client.TerraformProviders.GetProviderByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProvider {
				require.NotNil(t, provider)
				assert.Equal(t, test.id, provider.Metadata.ID)
			} else {
				assert.Nil(t, provider)
			}
		})
	}
}

func TestTerraformProviders_GetProviders(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform providers
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-providers-list",
		Description: "test group for providers list",
		FullPath:    "test-group-providers-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test terraform providers
	providers := []models.TerraformProvider{
		{
			Name:        "test-provider-list-1",
			GroupID:     group.Metadata.ID,
			RootGroupID: group.Metadata.ID,
			Private:     false,
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-provider-list-2",
			GroupID:     group.Metadata.ID,
			RootGroupID: group.Metadata.ID,
			Private:     true,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdProviders := []models.TerraformProvider{}
	for _, provider := range providers {
		created, err := testClient.client.TerraformProviders.CreateProvider(ctx, &provider)
		require.NoError(t, err)
		createdProviders = append(createdProviders, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetProvidersInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all providers",
			input:       &GetProvidersInput{},
			expectCount: len(createdProviders),
		},
		{
			name: "filter by search",
			input: &GetProvidersInput{
				Filter: &TerraformProviderFilter{
					Search: ptr.String("test-provider-list-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by name",
			input: &GetProvidersInput{
				Filter: &TerraformProviderFilter{
					Name: &createdProviders[0].Name,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by group ID",
			input: &GetProvidersInput{
				Filter: &TerraformProviderFilter{
					GroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdProviders),
		},
		{
			name: "filter by root group ID",
			input: &GetProvidersInput{
				Filter: &TerraformProviderFilter{
					RootGroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdProviders),
		},
		{
			name: "filter by terraform provider IDs",
			input: &GetProvidersInput{
				Filter: &TerraformProviderFilter{
					TerraformProviderIDs: []string{createdProviders[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformProviders.GetProviders(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Providers, test.expectCount)
		})
	}
}

func TestTerraformProviders_GetProvidersWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform providers
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-providers-pagination",
		Description: "test group for providers pagination",
		FullPath:    "test-group-providers-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
			Name:        fmt.Sprintf("test-provider-pagination-%d", i),
			GroupID:     group.Metadata.ID,
			RootGroupID: group.Metadata.ID,
			Private:     false,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformProviderSortableFieldNameAsc,
		TerraformProviderSortableFieldNameDesc,
		TerraformProviderSortableFieldUpdatedAtAsc,
		TerraformProviderSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformProviderSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformProviders.GetProviders(ctx, &GetProvidersInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Providers {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformProviders_GetProviderByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-trn",
		Description: "test group for provider trn",
		FullPath:    "test-group-provider-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for testing
	createdProvider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-trn",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectProvider  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by TRN",
			trn:            createdProvider.Metadata.TRN,
			expectProvider: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform-provider:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := testClient.client.TerraformProviders.GetProviderByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProvider {
				require.NotNil(t, provider)
				assert.Equal(t, test.trn, provider.Metadata.TRN)
			} else {
				assert.Nil(t, provider)
			}
		})
	}
}
