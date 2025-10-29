//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for VariableSortableField
func (v VariableSortableField) getValue() string {
	return string(v)
}

func TestVariables_CreateVariable(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-namespace",
		FullPath:  "test-namespace",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		key             string
		value           string
		category        models.VariableCategory
		namespacePath   string
	}

	testCases := []testCase{
		{
			name:          "create variable",
			key:           "test-var",
			value:         "test-value",
			category:      models.TerraformVariableCategory,
			namespacePath: group.FullPath,
		},
		{
			name:            "duplicate namespace, category, and key",
			key:             "duplicate-var",
			value:           "duplicate-value",
			category:        models.TerraformVariableCategory,
			namespacePath:   group.FullPath,
			expectErrorCode: errors.EConflict,
		},
	}

	// Create first variable for duplicate test
	_, err = testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "duplicate-var",
		Value:         ptr.String("original-value"),
		Category:      models.TerraformVariableCategory,
		NamespacePath: group.FullPath,
	})
	require.Nil(t, err)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			variable, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
				Key:           test.key,
				Value:         ptr.String(test.value),
				Category:      test.category,
				NamespacePath: test.namespacePath,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, variable)

			assert.Equal(t, test.key, variable.Key)
			assert.Equal(t, test.value, *variable.Value)
			assert.Equal(t, test.category, variable.Category)
			assert.NotEmpty(t, variable.Metadata.ID)
		})
	}
}

func TestVariables_UpdateVariable(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-namespace-update",
		FullPath:  "test-namespace-update",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a variable for testing
	createdVariable, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "test-var-update",
		Value:         ptr.String("original-value"),
		Category:      models.TerraformVariableCategory,
		NamespacePath: group.FullPath,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		value           string
		id              string
	}

	testCases := []testCase{
		{
			name:    "update variable",
			id:      createdVariable.Metadata.ID,
			version: createdVariable.Metadata.Version,
			value:   "updated-value",
		},
		{
			name:            "update will fail because resource version doesn't match",
			id:              createdVariable.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			value:           "should not update",
		},
		{
			name:            "defective-ID",
			id:              invalidID,
			version:         1,
			value:           "should not update",
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			variableToUpdate := *createdVariable
			variableToUpdate.Metadata.ID = test.id
			variableToUpdate.Metadata.Version = test.version
			variableToUpdate.Value = ptr.String(test.value)

			updatedVariable, err := testClient.client.Variables.UpdateVariable(ctx, &variableToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedVariable)

			assert.Equal(t, test.value, *updatedVariable.Value)
			assert.Equal(t, createdVariable.Metadata.Version+1, updatedVariable.Metadata.Version)
		})
	}
}

func TestVariables_DeleteVariable(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-namespace-delete",
		FullPath:  "test-namespace-delete",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a variable for testing
	createdVariable, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "test-var-delete",
		Value:         ptr.String("value-to-delete"),
		Category:      models.TerraformVariableCategory,
		NamespacePath: group.FullPath,
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
			name:    "delete variable",
			id:      createdVariable.Metadata.ID,
			version: createdVariable.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdVariable.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "defective-ID",
			id:              invalidID,
			version:         1,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Variables.DeleteVariable(ctx, &models.Variable{
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

			// Verify variable was deleted
			variable, err := testClient.client.Variables.GetVariableByID(ctx, test.id)
			assert.Nil(t, variable)
			assert.Nil(t, err)
		})
	}
}

func TestVariables_GetVariableByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-variable-get-by-id",
		Description: "test group for variable get by id",
		FullPath:    "test-group-variable-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a variable for testing
	createdVariable, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "test-variable-get-by-id",
		Value:         ptr.String("test-value"),
		Category:      models.TerraformVariableCategory,
		NamespacePath: group.FullPath,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectVariable  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by id",
			id:             createdVariable.Metadata.ID,
			expectVariable: true,
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
			variable, err := testClient.client.Variables.GetVariableByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectVariable {
				require.NotNil(t, variable)
				assert.Equal(t, test.id, variable.Metadata.ID)
			} else {
				assert.Nil(t, variable)
			}
		})
	}
}

func TestVariables_GetVariables(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-variables-list",
		Description: "test group for variables list",
		FullPath:    "test-group-variables-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test variables
	variables := []models.Variable{
		{
			Key:           "test-variable-1",
			Value:         ptr.String("test-value-1"),
			Category:      models.TerraformVariableCategory,
			NamespacePath: group.FullPath,
		},
		{
			Key:           "test-variable-2",
			Value:         ptr.String("test-value-2"),
			Category:      models.EnvironmentVariableCategory,
			NamespacePath: group.FullPath,
		},
	}

	createdVariables := []models.Variable{}
	for _, variable := range variables {
		created, err := testClient.client.Variables.CreateVariable(ctx, &variable)
		require.NoError(t, err)
		createdVariables = append(createdVariables, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetVariablesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name: "get all variables",
			input: &GetVariablesInput{
				Filter: &VariableFilter{
					NamespacePaths: []string{group.FullPath},
				},
			},
			expectCount: len(createdVariables),
		},
		{
			name: "filter by variable IDs",
			input: &GetVariablesInput{
				Filter: &VariableFilter{
					VariableIDs: []string{createdVariables[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by category",
			input: &GetVariablesInput{
				Filter: &VariableFilter{
					Category: &createdVariables[0].Category,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by key",
			input: &GetVariablesInput{
				Filter: &VariableFilter{
					Key: &createdVariables[0].Key,
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Variables.GetVariables(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Variables, test.expectCount)
		})
	}
}

func TestVariables_GetVariablesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-variables-pagination",
		Description: "test group for variables pagination",
		FullPath:    "test-group-variables-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
			Key:           fmt.Sprintf("test-variable-%d", i),
			Value:         ptr.String(fmt.Sprintf("test-value-%d", i)),
			Category:      models.TerraformVariableCategory,
			NamespacePath: group.FullPath,
		})
		require.NoError(t, err)
	}

	// Only test Key and CreatedAt fields to avoid NAMESPACE_PATH complexity
	sortableFields := []sortableField{
		VariableSortableFieldKeyAsc,
		VariableSortableFieldKeyDesc,
		VariableSortableFieldCreatedAtAsc,
		VariableSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := VariableSortableField(sortByField.getValue())

		result, err := testClient.client.Variables.GetVariables(ctx, &GetVariablesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
			Filter: &VariableFilter{
				NamespacePaths: []string{group.FullPath},
			},
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Variables {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestVariables_GetVariableByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-variable-get-by-trn",
		Description: "test group for variable get by trn",
		FullPath:    "test-group-variable-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a variable for testing
	createdVariable, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "test-variable-get-by-trn",
		Value:         ptr.String("test-value"),
		Category:      models.TerraformVariableCategory,
		NamespacePath: group.FullPath,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectVariable  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by TRN",
			trn:            createdVariable.Metadata.TRN,
			expectVariable: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:variable:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			variable, err := testClient.client.Variables.GetVariableByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectVariable {
				require.NotNil(t, variable)
				assert.Equal(t, createdVariable.Metadata.ID, variable.Metadata.ID)
			} else {
				assert.Nil(t, variable)
			}
		})
	}
}

func TestVariables_CreateVariables(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-variables-bulk-create",
		Description: "test group for variables bulk create",
		FullPath:    "test-group-variables-bulk-create",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test variables for bulk creation
	variables := []*models.Variable{
		{
			Key:           "test-bulk-variable-1",
			Value:         ptr.String("test-bulk-value-1"),
			Category:      models.TerraformVariableCategory,
			NamespacePath: group.FullPath,
		},
		{
			Key:           "test-bulk-variable-2",
			Value:         ptr.String("test-bulk-value-2"),
			Category:      models.EnvironmentVariableCategory,
			NamespacePath: group.FullPath,
		},
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		namespacePath   string
		variables       []*models.Variable
	}

	testCases := []testCase{
		{
			name:          "create variables in bulk",
			namespacePath: group.FullPath,
			variables:     variables,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Variables.CreateVariables(ctx, test.namespacePath, test.variables)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
