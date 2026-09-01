//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for TerraformProviderVersionSortableField
func (ts TerraformProviderVersionSortableField) getValue() string {
	return string(ts)
}

func TestTerraformProviderVersions_CreateProviderVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version",
		Description: "test group for provider version",
		FullPath:    "test-group-provider-version",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-version",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         string
		providerID      string
	}

	testCases := []testCase{
		{
			name:       "create provider version",
			version:    "1.0.0",
			providerID: provider.Metadata.ID,
		},
		{
			name:            "create provider version with invalid provider ID",
			version:         "1.0.1",
			providerID:      invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			providerVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
				SemanticVersion: test.version,
				ProviderID:      test.providerID,
				CreatedBy:       "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, providerVersion)

			assert.Equal(t, test.version, providerVersion.SemanticVersion)
			assert.Equal(t, test.providerID, providerVersion.ProviderID)
			assert.NotEmpty(t, providerVersion.Metadata.ID)
		})
	}
}

func TestTerraformProviderVersions_UpdateProviderVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version-update",
		Description: "test group for provider version update",
		FullPath:    "test-group-provider-version-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-version-update",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdProviderVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name              string
		expectErrorCode   errors.CodeType
		version           int
		providerVersionID string
		protocols         []string
	}

	testCases := []testCase{
		{
			name:              "update provider version",
			providerVersionID: createdProviderVersion.Metadata.ID,
			version:           createdProviderVersion.Metadata.Version,
			protocols:         []string{"5.0"},
		},
		{
			name:              "would-be_duplicate_provider_ID_and_semantic_version",
			providerVersionID: createdProviderVersion.Metadata.ID,
			expectErrorCode:   errors.EOptimisticLock,
			version:           -1,
			protocols:         []string{"4.0"},
		},
		{
			name:              "negative, non-existent Terraform provider version ID",
			providerVersionID: invalidID,
			expectErrorCode:   errors.EInternal,
			version:           1,
			protocols:         []string{"4.0"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			providerVersionToUpdate := *createdProviderVersion
			providerVersionToUpdate.Metadata.ID = test.providerVersionID
			providerVersionToUpdate.Metadata.Version = test.version
			providerVersionToUpdate.Protocols = test.protocols

			updatedProviderVersion, err := testClient.client.TerraformProviderVersions.UpdateProviderVersion(ctx, &providerVersionToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedProviderVersion)

			assert.Equal(t, test.protocols, updatedProviderVersion.Protocols)
			assert.Equal(t, createdProviderVersion.Metadata.Version+1, updatedProviderVersion.Metadata.Version)
		})
	}
}

func TestTerraformProviderVersions_DeleteProviderVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version-delete",
		Description: "test group for provider version delete",
		FullPath:    "test-group-provider-version-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-version-delete",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdProviderVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
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
			name:    "delete provider version",
			id:      createdProviderVersion.Metadata.ID,
			version: createdProviderVersion.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdProviderVersion.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "negative, non-existent Terraform provider version ID",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformProviderVersions.DeleteProviderVersion(ctx, &models.TerraformProviderVersion{
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

			providerVersion, err := testClient.client.TerraformProviderVersions.GetProviderVersionByID(ctx, test.id)
			assert.Nil(t, providerVersion)
			assert.Nil(t, err)
		})
	}
}

func TestTerraformProviderVersions_GetProviderVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version-get-by-id",
		Description: "test group for provider version get by id",
		FullPath:    "test-group-provider-version-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-version-get-by-id",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	createdProviderVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode       errors.CodeType
		name                  string
		id                    string
		expectProviderVersion bool
	}

	testCases := []testCase{
		{
			name:                  "get resource by id",
			id:                    createdProviderVersion.Metadata.ID,
			expectProviderVersion: true,
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
			providerVersion, err := testClient.client.TerraformProviderVersions.GetProviderVersionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProviderVersion {
				require.NotNil(t, providerVersion)
				assert.Equal(t, test.id, providerVersion.Metadata.ID)
			} else {
				assert.Nil(t, providerVersion)
			}
		})
	}
}

func TestTerraformProviderVersions_GetProviderVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-versions-list",
		Description: "test group for provider versions list",
		FullPath:    "test-group-provider-versions-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-versions-list",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	versions := []models.TerraformProviderVersion{
		{
			SemanticVersion: "1.0.0",
			ProviderID:      provider.Metadata.ID,
			CreatedBy:       "db-integration-tests",
		},
		{
			SemanticVersion: "1.1.0",
			ProviderID:      provider.Metadata.ID,
			CreatedBy:       "db-integration-tests",
		},
	}

	createdVersions := []models.TerraformProviderVersion{}
	for _, version := range versions {
		created, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &version)
		require.NoError(t, err)
		createdVersions = append(createdVersions, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetProviderVersionsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all provider versions",
			input:       &GetProviderVersionsInput{},
			expectCount: len(createdVersions),
		},
		{
			name: "filter by provider ID",
			input: &GetProviderVersionsInput{
				Filter: &TerraformProviderVersionFilter{
					ProviderID: &provider.Metadata.ID,
				},
			},
			expectCount: len(createdVersions),
		},
		{
			name: "filter by semantic version",
			input: &GetProviderVersionsInput{
				Filter: &TerraformProviderVersionFilter{
					SemanticVersion: &createdVersions[0].SemanticVersion,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by provider version IDs",
			input: &GetProviderVersionsInput{
				Filter: &TerraformProviderVersionFilter{
					ProviderVersionIDs: []string{createdVersions[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformProviderVersions.GetProviderVersions(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ProviderVersions, test.expectCount)
		})
	}
}

func TestTerraformProviderVersions_GetProviderVersionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-versions-pagination",
		Description: "test group for provider versions pagination",
		FullPath:    "test-group-provider-versions-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-versions-pagination",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
			SemanticVersion: fmt.Sprintf("1.%d.0", i),
			ProviderID:      provider.Metadata.ID,
			CreatedBy:       "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformProviderVersionSortableFieldUpdatedAtAsc,
		TerraformProviderVersionSortableFieldUpdatedAtDesc,
		TerraformProviderVersionSortableFieldCreatedAtAsc,
		TerraformProviderVersionSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformProviderVersionSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformProviderVersions.GetProviderVersions(ctx, &GetProviderVersionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ProviderVersions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformProviderVersions_GetProviderVersionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-provider-version-trn",
		Description: "test group for provider version trn",
		FullPath:    "test-group-provider-version-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-version-trn",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		Private:     false,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	createdProviderVersion, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
		SemanticVersion: "1.0.0",
		ProviderID:      provider.Metadata.ID,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode       errors.CodeType
		name                  string
		trn                   string
		expectProviderVersion bool
	}

	testCases := []testCase{
		{
			name:                  "get resource by TRN",
			trn:                   createdProviderVersion.Metadata.TRN,
			expectProviderVersion: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform-provider-version:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			providerVersion, err := testClient.client.TerraformProviderVersions.GetProviderVersionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProviderVersion {
				require.NotNil(t, providerVersion)
				assert.Equal(t, test.trn, providerVersion.Metadata.TRN)
			} else {
				assert.Nil(t, providerVersion)
			}
		})
	}
}

func TestTerraformProviderVersions_DeleteProviderVersionBatch(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-delete-provider-versions",
		FullPath:  "test-group-delete-provider-versions",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	provider, err := testClient.client.TerraformProviders.CreateProvider(ctx, &models.TerraformProvider{
		Name:        "test-provider-delete-versions",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	versionSeq := 0
	makeVersion := func(t *testing.T) *models.TerraformProviderVersion {
		t.Helper()
		versionSeq++
		pv, err := testClient.client.TerraformProviderVersions.CreateProviderVersion(ctx, &models.TerraformProviderVersion{
			SemanticVersion: fmt.Sprintf("%d.0.0", versionSeq),
			ProviderID:      provider.Metadata.ID,
			CreatedBy:       "db-integration-tests",
		})
		require.NoError(t, err)
		return pv
	}

	// Each test case owns its own versions so cases do not interfere.
	pvNoOp := makeVersion(t)
	pvIgnore := makeVersion(t)
	pvPartial := makeVersion(t)
	pvSurvivor := makeVersion(t)
	pvMixed := makeVersion(t)
	pvMulti1 := makeVersion(t)
	pvMulti2 := makeVersion(t)
	pvStale := makeVersion(t)

	type testCase struct {
		name         string
		input        DeleteProviderVersionBatchInput
		shouldBeGone []string
		shouldExist  []string
		expectOLE    bool
	}

	testCases := []testCase{
		{
			name:        "empty slice is a no-op",
			input:       DeleteProviderVersionBatchInput{ProviderVersions: []*models.TerraformProviderVersion{}},
			shouldExist: []string{pvNoOp.Metadata.ID},
		},
		{
			name: "non-existent version: OLE, nothing deleted",
			input: DeleteProviderVersionBatchInput{ProviderVersions: []*models.TerraformProviderVersion{
				{Metadata: models.ResourceMetadata{ID: nonExistentID, Version: 1}},
			}},
			shouldExist: []string{pvIgnore.Metadata.ID},
			expectOLE:   true,
		},
		{
			name:         "partial delete — only the specified version is removed",
			input:        DeleteProviderVersionBatchInput{ProviderVersions: []*models.TerraformProviderVersion{pvPartial}},
			shouldBeGone: []string{pvPartial.Metadata.ID},
			shouldExist:  []string{pvSurvivor.Metadata.ID},
		},
		{
			name: "batch with non-existent version alongside valid version: OLE, valid one still deleted",
			input: DeleteProviderVersionBatchInput{ProviderVersions: []*models.TerraformProviderVersion{
				pvMixed,
				{Metadata: models.ResourceMetadata{ID: nonExistentID, Version: 1}},
			}},
			shouldBeGone: []string{pvMixed.Metadata.ID},
			expectOLE:    true,
		},
		{
			name:         "delete multiple versions in one call",
			input:        DeleteProviderVersionBatchInput{ProviderVersions: []*models.TerraformProviderVersion{pvMulti1, pvMulti2}},
			shouldBeGone: []string{pvMulti1.Metadata.ID, pvMulti2.Metadata.ID},
		},
		{
			name: "stale version: OLE, nothing deleted",
			input: DeleteProviderVersionBatchInput{ProviderVersions: []*models.TerraformProviderVersion{
				{Metadata: models.ResourceMetadata{ID: pvStale.Metadata.ID, Version: pvStale.Metadata.Version - 1}},
			}},
			shouldExist: []string{pvStale.Metadata.ID},
			expectOLE:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			deletedIDs, err := testClient.client.TerraformProviderVersions.DeleteProviderVersionBatch(ctx, &test.input)

			if test.expectOLE {
				assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
			} else {
				require.NoError(t, err)
			}
			assert.ElementsMatch(t, test.shouldBeGone, deletedIDs)

			for _, id := range test.shouldBeGone {
				got, err := testClient.client.TerraformProviderVersions.GetProviderVersionByID(ctx, id)
				require.NoError(t, err)
				assert.Nil(t, got)
			}

			for _, id := range test.shouldExist {
				got, err := testClient.client.TerraformProviderVersions.GetProviderVersionByID(ctx, id)
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}
