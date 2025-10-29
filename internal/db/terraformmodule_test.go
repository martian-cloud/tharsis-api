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

// getValue implements the sortableField interface for TerraformModuleSortableField
func (tm TerraformModuleSortableField) getValue() string {
	return string(tm)
}

func TestTerraformModules_CreateModule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module",
		Description: "test group for module",
		FullPath:    "test-group-module",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		moduleName      string
		groupID         string
		system          string
	}

	testCases := []testCase{
		{
			name:       "create module",
			moduleName: "test-module",
			groupID:    group.Metadata.ID,
			system:     "aws",
		},
		{
			name:            "create module with invalid group ID",
			moduleName:      "invalid-module",
			groupID:         invalidID,
			system:          "aws",
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			module, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
				Name:        test.moduleName,
				GroupID:     test.groupID,
				RootGroupID: group.Metadata.ID,
				System:      test.system,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, module)

			assert.Equal(t, test.moduleName, module.Name)
			assert.Equal(t, test.groupID, module.GroupID)
			assert.Equal(t, test.system, module.System)
			assert.NotEmpty(t, module.Metadata.ID)
		})
	}
}

func TestTerraformModules_UpdateModule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-update",
		Description: "test group for module update",
		FullPath:    "test-group-module-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a module for testing
	createdModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-update",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		moduleID        string
		system          string
	}

	testCases := []testCase{
		{
			name:     "update module",
			moduleID: createdModule.Metadata.ID,
			version:  createdModule.Metadata.Version,
			system:   "aws", // Keep the original system value
		},
		{
			name:            "would-be-duplicate-group-id-and-module-name",
			moduleID:        createdModule.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			system:          "azure",
		},
		{
			name:            "negative, non-existent Terraform module ID",
			moduleID:        invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
			system:          "aws",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			moduleToUpdate := *createdModule
			moduleToUpdate.Metadata.ID = test.moduleID
			moduleToUpdate.Metadata.Version = test.version
			moduleToUpdate.System = test.system

			updatedModule, err := testClient.client.TerraformModules.UpdateModule(ctx, &moduleToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedModule)

			assert.Equal(t, test.system, updatedModule.System)
			assert.Equal(t, createdModule.Metadata.Version+1, updatedModule.Metadata.Version)
		})
	}
}

func TestTerraformModules_DeleteModule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-delete",
		Description: "test group for module delete",
		FullPath:    "test-group-module-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a module for testing
	createdModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-delete",
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
			name:    "delete module",
			id:      createdModule.Metadata.ID,
			version: createdModule.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdModule.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "negative, non-existent Terraform module ID",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.TerraformModules.DeleteModule(ctx, &models.TerraformModule{
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

			// Verify module was deleted
			module, err := testClient.client.TerraformModules.GetModuleByID(ctx, test.id)
			assert.Nil(t, module)
			assert.Nil(t, err)
		})
	}
}

func TestTerraformModules_GetModuleByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-get-by-id",
		Description: "test group for module get by id",
		FullPath:    "test-group-module-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform module for testing
	createdModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-get-by-id",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectModule    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdModule.Metadata.ID,
			expectModule: true,
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
			module, err := testClient.client.TerraformModules.GetModuleByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModule {
				require.NotNil(t, module)
				assert.Equal(t, test.id, module.Metadata.ID)
			} else {
				assert.Nil(t, module)
			}
		})
	}
}

func TestTerraformModules_GetModules(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-modules-list",
		Description: "test group for modules list",
		FullPath:    "test-group-modules-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test terraform modules
	modules := []models.TerraformModule{
		{
			Name:        "test-module-1",
			System:      "aws",
			GroupID:     group.Metadata.ID,
			RootGroupID: group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-module-2",
			System:      "gcp",
			GroupID:     group.Metadata.ID,
			RootGroupID: group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdModules := []models.TerraformModule{}
	for _, module := range modules {
		created, err := testClient.client.TerraformModules.CreateModule(ctx, &module)
		require.NoError(t, err)
		createdModules = append(createdModules, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetModulesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name: "get all modules",
			input: &GetModulesInput{
				Filter: &TerraformModuleFilter{
					GroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdModules),
		},
		{
			name: "filter by search",
			input: &GetModulesInput{
				Filter: &TerraformModuleFilter{
					Search: ptr.String("test-module-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by name",
			input: &GetModulesInput{
				Filter: &TerraformModuleFilter{
					Name: &createdModules[0].Name,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by system",
			input: &GetModulesInput{
				Filter: &TerraformModuleFilter{
					System: &createdModules[0].System,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by root group ID",
			input: &GetModulesInput{
				Filter: &TerraformModuleFilter{
					RootGroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdModules),
		},
		{
			name: "filter by terraform module IDs",
			input: &GetModulesInput{
				Filter: &TerraformModuleFilter{
					TerraformModuleIDs: []string{createdModules[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.TerraformModules.GetModules(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Modules, test.expectCount)
		})
	}
}

func TestTerraformModules_GetModulesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-modules-pagination",
		Description: "test group for modules pagination",
		FullPath:    "test-group-modules-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
			Name:        fmt.Sprintf("test-module-%d", i),
			System:      "aws",
			GroupID:     group.Metadata.ID,
			RootGroupID: group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		TerraformModuleSortableFieldNameAsc,
		TerraformModuleSortableFieldNameDesc,
		TerraformModuleSortableFieldUpdatedAtAsc,
		TerraformModuleSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := TerraformModuleSortableField(sortByField.getValue())

		result, err := testClient.client.TerraformModules.GetModules(ctx, &GetModulesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
			Filter: &TerraformModuleFilter{
				GroupID: &group.Metadata.ID,
			},
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Modules {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestTerraformModules_GetModuleByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-module-get-by-trn",
		Description: "test group for module get by trn",
		FullPath:    "test-group-module-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a terraform module for testing
	createdModule, err := testClient.client.TerraformModules.CreateModule(ctx, &models.TerraformModule{
		Name:        "test-module-get-by-trn",
		System:      "aws",
		GroupID:     group.Metadata.ID,
		RootGroupID: group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectModule    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by TRN",
			trn:          createdModule.Metadata.TRN,
			expectModule: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:terraform_module:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			module, err := testClient.client.TerraformModules.GetModuleByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModule {
				require.NotNil(t, module)
				assert.Equal(t, createdModule.Metadata.ID, module.Metadata.ID)
			} else {
				assert.Nil(t, module)
			}
		})
	}
}
