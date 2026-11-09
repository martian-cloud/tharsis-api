//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestTerraformProviderDocItems_CreateProviderDocItems(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-doc-item-create",
		Description: "test group for provider doc item create",
		FullPath:    "test-group-provider-doc-item-create",
		CreatedBy:   models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-doc-item-create",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	// Pre-create an item to exercise the unique-constraint conflict path.
	require.Nil(t, testClient.client.TerraformProviderDocItems.CreateProviderDocItems(ctx, []*models.TerraformProviderDocItem{
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "existing", Category: models.TerraformProviderDocItemCategoryResources},
	}))

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		toCreate        []*models.TerraformProviderDocItem
	}

	testCases := []testCase{
		{
			name: "creates doc items across categories",
			toCreate: []*models.TerraformProviderDocItem{
				{ProviderVersionID: providerVersion.Metadata.ID, Filename: "instance", Category: models.TerraformProviderDocItemCategoryResources},
				{ProviderVersionID: providerVersion.Metadata.ID, Filename: "index", Category: models.TerraformProviderDocItemCategoryOverview},
				{ProviderVersionID: providerVersion.Metadata.ID, Filename: "vsphere_virtual_machine", Category: models.TerraformProviderDocItemCategoryResources, Subcategory: ptr.String("Virtual Machine")},
			},
		},
		{
			name: "negative, duplicate filename in same category",
			toCreate: []*models.TerraformProviderDocItem{
				{ProviderVersionID: providerVersion.Metadata.ID, Filename: "existing", Category: models.TerraformProviderDocItemCategoryResources},
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "negative, provider version does not exist",
			toCreate: []*models.TerraformProviderDocItem{
				{ProviderVersionID: nonExistentID, Filename: "orphan", Category: models.TerraformProviderDocItemCategoryResources},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "negative, invalid provider version ID",
			toCreate: []*models.TerraformProviderDocItem{
				{ProviderVersionID: invalidID, Filename: "orphan", Category: models.TerraformProviderDocItemCategoryResources},
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviderDocItems.CreateProviderDocItems(ctx, test.toCreate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
		})
	}
}

func TestTerraformProviderDocItems_GetProviderDocItems(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-doc-item-get",
		Description: "test group for provider doc item get",
		FullPath:    "test-group-provider-doc-item-get",
		CreatedBy:   models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-doc-item-get",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	// A second version with no doc items, to verify the empty result.
	emptyVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "2.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	// Seed doc items across categories. Insert filenames out of order to verify ordering.
	seed := []*models.TerraformProviderDocItem{
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "index", Category: models.TerraformProviderDocItemCategoryOverview},
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "vpc", Category: models.TerraformProviderDocItemCategoryResources},
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "instance", Category: models.TerraformProviderDocItemCategoryResources},
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "ami", Category: models.TerraformProviderDocItemCategoryDataSources},
	}
	require.Nil(t, testClient.client.TerraformProviderDocItems.CreateProviderDocItems(ctx, seed))

	resourcesCategory := models.TerraformProviderDocItemCategoryResources

	type testCase struct {
		name            string
		versionID       string
		category        *models.TerraformProviderDocItemCategory
		expectFilenames []string
	}

	testCases := []testCase{
		{
			name:            "empty result for version with no doc items",
			versionID:       emptyVersion.Metadata.ID,
			expectFilenames: []string{},
		},
		{
			name:            "negative, non-existent provider version returns no doc items",
			versionID:       nonExistentID,
			expectFilenames: []string{},
		},
		{
			name:      "all doc items ordered by category then filename",
			versionID: providerVersion.Metadata.ID,
			// Ordered by category asc (data-sources, overview, resources) then filename asc.
			expectFilenames: []string{"ami", "index", "instance", "vpc"},
		},
		{
			name:            "filter by category returns only that category, filename-ordered",
			versionID:       providerVersion.Metadata.ID,
			category:        &resourcesCategory,
			expectFilenames: []string{"instance", "vpc"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformProviderDocItems.GetProviderDocItems(ctx, &GetProviderDocItemsInput{
				ProviderVersionID: test.versionID,
				Category:          test.category,
			})
			require.Nil(t, err)

			actualFilenames := make([]string, len(result))
			for i, item := range result {
				actualFilenames[i] = item.Filename
			}
			assert.Equal(t, test.expectFilenames, actualFilenames)
		})
	}
}

func TestTerraformProviderDocItems_GetProviderDocItemByFilename(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-doc-item-by-filename",
		Description: "test group for provider doc item by-filename lookup",
		FullPath:    "test-group-provider-doc-item-by-filename",
		CreatedBy:   models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-doc-item-by-filename",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       models.Actor{Subject: "db-integration-tests"},
	})
	require.Nil(t, err)

	// "index" lives under overview and "instance" under resources, so a cross-category lookup must not match either.
	require.Nil(t, testClient.client.TerraformProviderDocItems.CreateProviderDocItems(ctx, []*models.TerraformProviderDocItem{
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "index", Category: models.TerraformProviderDocItemCategoryOverview},
		{ProviderVersionID: providerVersion.Metadata.ID, Filename: "instance", Category: models.TerraformProviderDocItemCategoryResources},
	}))

	resourcesCategory := models.TerraformProviderDocItemCategoryResources

	type testCase struct {
		name        string
		category    models.TerraformProviderDocItemCategory
		filename    string
		expectFound bool
	}

	testCases := []testCase{
		{
			name:        "returns the matching item",
			category:    resourcesCategory,
			filename:    "instance",
			expectFound: true,
		},
		{
			name:        "returns nil when no item matches",
			category:    resourcesCategory,
			filename:    "does-not-exist",
			expectFound: false,
		},
		{
			name:        "does not match a filename that exists only in a different category",
			category:    resourcesCategory,
			filename:    "index",
			expectFound: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			item, err := testClient.client.TerraformProviderDocItems.GetProviderDocItemByFilename(
				ctx, &GetProviderDocItemByFilenameInput{
					ProviderVersionID: providerVersion.Metadata.ID,
					Category:          test.category,
					Filename:          test.filename,
				})
			require.Nil(t, err)

			if !test.expectFound {
				assert.Nil(t, item)
				return
			}

			require.NotNil(t, item)
			assert.Equal(t, test.filename, item.Filename)
			assert.Equal(t, test.category, item.Category)
		})
	}
}
