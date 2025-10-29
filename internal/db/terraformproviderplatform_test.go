//go:build integration

package db

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"testing"
)

// getValue implements the sortableField interface for TerraformProviderPlatformSortableField
func (tpp TerraformProviderPlatformSortableField) getValue() string {
	return string(tpp)
}

func TestTerraformProviderPlatforms_CreateProviderPlatform(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-create",
		Description: "test group for provider platform create",
		FullPath:    "test-group-provider-platform-create",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platform
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platform-create",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platform
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		toCreate        *models.TerraformProviderPlatform
	}

	testCases := []testCase{
		{
			name: "create provider platform",
			toCreate: &models.TerraformProviderPlatform{
				OperatingSystem:   "linux",
				Architecture:      "amd64",
				ProviderVersionID: providerVersion.Metadata.ID,
				CreatedBy:         "db-integration-tests",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualProviderPlatform, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, test.toCreate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.toCreate.OperatingSystem, actualProviderPlatform.OperatingSystem)
			assert.Equal(t, test.toCreate.Architecture, actualProviderPlatform.Architecture)
		})
	}
}

func TestTerraformProviderPlatforms_UpdateProviderPlatform(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-update",
		Description: "test group for provider platform update",
		FullPath:    "test-group-provider-platform-update",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platform
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platform-update",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platform
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a provider platform to update
	createdProviderPlatform, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
		OperatingSystem:   "linux",
		Architecture:      "amd64",
		ProviderVersionID: providerVersion.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		toUpdate        *models.TerraformProviderPlatform
	}

	testCases := []testCase{
		{
			name: "update provider platform",
			toUpdate: &models.TerraformProviderPlatform{
				Metadata:          createdProviderPlatform.Metadata,
				OperatingSystem:   createdProviderPlatform.OperatingSystem,
				Architecture:      createdProviderPlatform.Architecture,
				ProviderVersionID: createdProviderPlatform.ProviderVersionID,
				CreatedBy:         createdProviderPlatform.CreatedBy,
				BinaryUploaded:    true, // This is the field that can be updated
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualProviderPlatform, err := testClient.client.TerraformProviderPlatforms.UpdateProviderPlatform(ctx, test.toUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.toUpdate.BinaryUploaded, actualProviderPlatform.BinaryUploaded)
		})
	}
}

func TestTerraformProviderPlatforms_DeleteProviderPlatform(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-delete",
		Description: "test group for provider platform delete",
		FullPath:    "test-group-provider-platform-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platform
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platform-delete",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platform
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a provider platform to delete
	createdProviderPlatform, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
		OperatingSystem:   "linux",
		Architecture:      "amd64",
		ProviderVersionID: providerVersion.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		toDelete        *models.TerraformProviderPlatform
	}

	testCases := []testCase{
		{
			name:     "delete provider platform",
			toDelete: createdProviderPlatform,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviderPlatforms.DeleteProviderPlatform(ctx, test.toDelete)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			// Verify deletion
			providerPlatform, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatformByID(ctx, test.toDelete.Metadata.ID)
			assert.Nil(t, providerPlatform)
		})
	}
}

func TestTerraformProviderPlatforms_GetProviderPlatformByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-get-by-id",
		Description: "test group for provider platform get by id",
		FullPath:    "test-group-provider-platform-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platform
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platform-get-by-id",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platform
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a provider platform for testing
	createdProviderPlatform, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
		OperatingSystem:   "linux",
		Architecture:      "amd64",
		ProviderVersionID: providerVersion.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode        errors.CodeType
		name                   string
		id                     string
		expectProviderPlatform bool
	}

	testCases := []testCase{
		{
			name:                   "get resource by id",
			id:                     createdProviderPlatform.Metadata.ID,
			expectProviderPlatform: true,
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
			providerPlatform, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatformByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProviderPlatform {
				require.NotNil(t, providerPlatform)
				assert.Equal(t, test.id, providerPlatform.Metadata.ID)
			} else {
				assert.Nil(t, providerPlatform)
			}
		})
	}
}

func TestTerraformProviderPlatforms_GetProviderPlatforms(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platforms-list",
		Description: "test group for provider platforms list",
		FullPath:    "test-group-provider-platforms-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platforms
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platforms-list",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platforms
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test provider platforms
	platforms := []models.TerraformProviderPlatform{
		{
			OperatingSystem:   "linux",
			Architecture:      "amd64",
			ProviderVersionID: providerVersion.Metadata.ID,
			CreatedBy:         "db-integration-tests",
		},
		{
			OperatingSystem:   "windows",
			Architecture:      "amd64",
			ProviderVersionID: providerVersion.Metadata.ID,
			CreatedBy:         "db-integration-tests",
		},
	}

	createdPlatforms := []models.TerraformProviderPlatform{}
	for _, platform := range platforms {
		created, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &platform)
		require.NoError(t, err)
		createdPlatforms = append(createdPlatforms, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetProviderPlatformsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all provider platforms",
			input:       &GetProviderPlatformsInput{},
			expectCount: len(createdPlatforms),
		},
		{
			name: "filter by provider version ID",
			input: &GetProviderPlatformsInput{
				Filter: &TerraformProviderPlatformFilter{
					ProviderVersionID: &providerVersion.Metadata.ID,
				},
			},
			expectCount: len(createdPlatforms),
		},
		{
			name: "filter by operating system",
			input: &GetProviderPlatformsInput{
				Filter: &TerraformProviderPlatformFilter{
					OperatingSystem: &createdPlatforms[0].OperatingSystem,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by architecture",
			input: &GetProviderPlatformsInput{
				Filter: &TerraformProviderPlatformFilter{
					Architecture: &createdPlatforms[0].Architecture,
				},
			},
			expectCount: len(createdPlatforms),
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatforms(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ProviderPlatforms, test.expectCount)
		})
	}
}

func TestTerraformProviderPlatforms_GetProviderPlatformsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platforms-pagination",
		Description: "test group for provider platforms pagination",
		FullPath:    "test-group-provider-platforms-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platforms
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platforms-pagination",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platforms
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
			OperatingSystem:   fmt.Sprintf("linux-%d", i),
			Architecture:      "amd64",
			ProviderVersionID: providerVersion.Metadata.ID,
			CreatedBy:         "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformProviderPlatformSortableFieldUpdatedAtAsc,
		TerraformProviderPlatformSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformProviderPlatformSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatforms(ctx, &GetProviderPlatformsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ProviderPlatforms {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformProviderPlatforms_GetProviderPlatformByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the terraform provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-platform-trn",
		Description: "test group for provider platform trn",
		FullPath:    "test-group-provider-platform-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider for the platform
	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-platform-trn",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform provider version for the platform
	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a provider platform for testing
	createdProviderPlatform, err := testClient.client.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
		OperatingSystem:   "linux",
		Architecture:      "amd64",
		ProviderVersionID: providerVersion.Metadata.ID,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode        errors.CodeType
		name                   string
		trn                    string
		expectProviderPlatform bool
	}

	testCases := []testCase{
		{
			name:                   "get resource by TRN",
			trn:                    createdProviderPlatform.Metadata.TRN,
			expectProviderPlatform: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform-provider-platform:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			providerPlatform, err := testClient.client.TerraformProviderPlatforms.GetProviderPlatformByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProviderPlatform {
				require.NotNil(t, providerPlatform)
				assert.Equal(t, test.trn, providerPlatform.Metadata.TRN)
			} else {
				assert.Nil(t, providerPlatform)
			}
		})
	}
}
