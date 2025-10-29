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

// getValue implements the sortableField interface for TerraformModuleVersionSortableField
func (tmv TerraformModuleVersionSortableField) getValue() string {
	return string(tmv)
}

func TestTerraformModuleVersions_CreateModuleVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-version",
		Description: "test group for module version",
		FullPath:    "test-group-module-version",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	module, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-version",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		System:      "aws",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         string
		moduleID        string
	}

	testCases := []testCase{
		{
			name:     "create module version",
			version:  "1.0.0",
			moduleID: module.Metadata.ID,
		},
		{
			name:            "duplicate_version",
			version:         "1.0.0",
			moduleID:        module.Metadata.ID,
			expectErrorCode: errors.EConflict,
		},
		{
			name:     "duplicate_latest",
			version:  "latest",
			moduleID: module.Metadata.ID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			moduleVersion, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &models.TerraformModuleVersion{
				SemanticVersion: test.version,
				ModuleID:        test.moduleID,
				SHASum:          []byte("test-sha-sum"),
				CreatedBy:       "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, moduleVersion)

			assert.Equal(t, test.version, moduleVersion.SemanticVersion)
			assert.Equal(t, test.moduleID, moduleVersion.ModuleID)
			assert.NotEmpty(t, moduleVersion.Metadata.ID)
		})
	}
}

func TestTerraformModuleVersions_UpdateModuleVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group, module, and module version for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-version-update",
		Description: "test group for module version update",
		FullPath:    "test-group-module-version-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	module, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-version-update",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		System:      "aws",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdModuleVersion, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &models.TerraformModuleVersion{
		SemanticVersion: "1.0.0",
		ModuleID:        module.Metadata.ID,
		SHASum:          []byte("test-sha-sum"),
		CreatedBy:       "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		moduleVersionID string
		status          models.TerraformModuleVersionStatus
	}

	testCases := []testCase{
		{
			name:            "update module version",
			moduleVersionID: createdModuleVersion.Metadata.ID,
			version:         createdModuleVersion.Metadata.Version,
			status:          models.TerraformModuleVersionStatusUploaded,
		},
		{
			name:            "update will fail because resource version doesn't match",
			moduleVersionID: createdModuleVersion.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.TerraformModuleVersionStatusErrored,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			moduleVersionToUpdate := *createdModuleVersion
			moduleVersionToUpdate.Metadata.ID = test.moduleVersionID
			moduleVersionToUpdate.Metadata.Version = test.version
			moduleVersionToUpdate.Status = test.status

			updatedModuleVersion, err := testClient.client.TerraformModuleVersions.UpdateModuleVersion(ctx, &moduleVersionToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedModuleVersion)

			assert.Equal(t, test.status, updatedModuleVersion.Status)
			assert.Equal(t, createdModuleVersion.Metadata.Version+1, updatedModuleVersion.Metadata.Version)
		})
	}
}

func TestTerraformModuleVersions_DeleteModuleVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group, module, and module version for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-version-delete",
		Description: "test group for module version delete",
		FullPath:    "test-group-module-version-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	module, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-version-delete",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		System:      "aws",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdModuleVersion, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &models.TerraformModuleVersion{
		SemanticVersion: "1.0.0",
		ModuleID:        module.Metadata.ID,
		SHASum:          []byte("test-sha-sum"),
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
			name:    "delete module version",
			id:      createdModuleVersion.Metadata.ID,
			version: createdModuleVersion.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdModuleVersion.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformModuleVersions.DeleteModuleVersion(ctx, &models.TerraformModuleVersion{
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

			// Verify module version was deleted
			moduleVersion, err := testClient.client.TerraformModuleVersions.GetModuleVersionByID(ctx, test.id)
			assert.Nil(t, moduleVersion)
			assert.Nil(t, err)
		})
	}
}

func TestTerraformModuleVersions_GetModuleVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-version-get-by-id",
		Description: "test group for module version get by id",
		FullPath:    "test-group-module-version-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-version-get-by-id",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a module version for testing
	createdModuleVersion, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &models.TerraformModuleVersion{
		ModuleID:        terraformModule.Metadata.ID,
		SemanticVersion: "1.0.0",
		SHASum:          []byte("test-sha-sum"),
		Status:          models.TerraformModuleVersionStatusUploaded,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode     errors.CodeType
		name                string
		id                  string
		expectModuleVersion bool
	}

	testCases := []testCase{
		{
			name:                "get resource by id",
			id:                  createdModuleVersion.Metadata.ID,
			expectModuleVersion: true,
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
			moduleVersion, err := testClient.client.TerraformModuleVersions.GetModuleVersionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModuleVersion {
				require.NotNil(t, moduleVersion)
				assert.Equal(t, test.id, moduleVersion.Metadata.ID)
			} else {
				assert.Nil(t, moduleVersion)
			}
		})
	}
}

func TestTerraformModuleVersions_GetModuleVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-versions-list",
		Description: "test group for module versions list",
		FullPath:    "test-group-module-versions-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-versions-list",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test module versions
	moduleVersions := []models.TerraformModuleVersion{
		{
			ModuleID:        terraformModule.Metadata.ID,
			SemanticVersion: "1.0.0",
			SHASum:          []byte("test-sha-sum"),
			Status:          models.TerraformModuleVersionStatusUploaded,
			CreatedBy:       "db-integration-tests",
		},
		{
			ModuleID:        terraformModule.Metadata.ID,
			SemanticVersion: "1.1.0",
			SHASum:          []byte("test-sha-sum"),
			Status:          models.TerraformModuleVersionStatusPending,
			CreatedBy:       "db-integration-tests",
		},
	}

	createdModuleVersions := []models.TerraformModuleVersion{}
	for _, moduleVersion := range moduleVersions {
		created, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &moduleVersion)
		require.NoError(t, err)
		createdModuleVersions = append(createdModuleVersions, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetModuleVersionsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all module versions",
			input:       &GetModuleVersionsInput{},
			expectCount: len(createdModuleVersions),
		},
		{
			name: "filter by module ID",
			input: &GetModuleVersionsInput{
				Filter: &TerraformModuleVersionFilter{
					ModuleID: &terraformModule.Metadata.ID,
				},
			},
			expectCount: len(createdModuleVersions),
		},
		{
			name: "filter by semantic version",
			input: &GetModuleVersionsInput{
				Filter: &TerraformModuleVersionFilter{
					SemanticVersion: &createdModuleVersions[0].SemanticVersion,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by status",
			input: &GetModuleVersionsInput{
				Filter: &TerraformModuleVersionFilter{
					Status: &createdModuleVersions[0].Status,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by module version IDs",
			input: &GetModuleVersionsInput{
				Filter: &TerraformModuleVersionFilter{
					ModuleVersionIDs: []string{createdModuleVersions[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformModuleVersions.GetModuleVersions(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ModuleVersions, test.expectCount)
		})
	}
}

func TestTerraformModuleVersions_GetModuleVersionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-versions-pagination",
		Description: "test group for module versions pagination",
		FullPath:    "test-group-module-versions-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-versions-pagination",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &models.TerraformModuleVersion{
			ModuleID:        terraformModule.Metadata.ID,
			SemanticVersion: fmt.Sprintf("1.%d.0", i),
			SHASum:          []byte(fmt.Sprintf("test-sha-sum-%d", i)),
			Status:          models.TerraformModuleVersionStatusUploaded,
			CreatedBy:       "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformModuleVersionSortableFieldUpdatedAtAsc,
		TerraformModuleVersionSortableFieldUpdatedAtDesc,
		TerraformModuleVersionSortableFieldCreatedAtAsc,
		TerraformModuleVersionSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformModuleVersionSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformModuleVersions.GetModuleVersions(ctx, &GetModuleVersionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ModuleVersions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformModuleVersions_GetModuleVersionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and module for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-version-get-by-trn",
		Description: "test group for module version get by trn",
		FullPath:    "test-group-module-version-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	terraformModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-version-get-by-trn",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a module version for testing
	createdModuleVersion, err := testClient.client.TerraformModuleVersions.CreateModuleVersion(ctx, &models.TerraformModuleVersion{
		ModuleID:        terraformModule.Metadata.ID,
		SemanticVersion: "1.0.0",
		SHASum:          []byte("test-sha-sum"),
		Status:          models.TerraformModuleVersionStatusUploaded,
		CreatedBy:       "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode     errors.CodeType
		name                string
		trn                 string
		expectModuleVersion bool
	}

	testCases := []testCase{
		{
			name:                "get resource by TRN",
			trn:                 createdModuleVersion.Metadata.TRN,
			expectModuleVersion: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform_module_version:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			moduleVersion, err := testClient.client.TerraformModuleVersions.GetModuleVersionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModuleVersion {
				require.NotNil(t, moduleVersion)
				assert.Equal(t, createdModuleVersion.Metadata.ID, moduleVersion.Metadata.ID)
			} else {
				assert.Nil(t, moduleVersion)
			}
		})
	}
}
