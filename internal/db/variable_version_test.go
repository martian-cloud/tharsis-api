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

func (sf VariableVersionSortableField) getValue() string {
	return string(sf)
}

func TestGetVariableVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	variable, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "test-version",
		Value:         ptr.String("test-value"),
		NamespacePath: group.FullPath,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode       errors.CodeType
		name                  string
		id                    string
		expectVariableVersion bool
	}

	testCases := []testCase{
		{
			name:                  "get resource by id",
			id:                    variable.LatestVersionID,
			expectVariableVersion: true,
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
			variableVersion, err := testClient.client.VariableVersions.GetVariableVersionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectVariableVersion {
				require.NotNil(t, variableVersion)
				assert.Equal(t, test.id, variableVersion.Metadata.ID)
			} else {
				assert.Nil(t, variableVersion)
			}
		})
	}
}

func TestGetVariableVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group1",
	})
	require.Nil(t, err)

	variableVersions := []*models.VariableVersion{}

	// Create variable1
	variable1, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "a",
		Value:         ptr.String("val-a"),
		NamespacePath: group.FullPath,
	})
	require.Nil(t, err)

	variableVersion, err := testClient.client.VariableVersions.GetVariableVersionByID(ctx, variable1.LatestVersionID)
	require.Nil(t, err)

	variableVersions = append(variableVersions, variableVersion)

	// Update variable1 to create a new version
	variable1.Value = ptr.String("updated-val-a")
	updatedVariable1, err := testClient.client.Variables.UpdateVariable(ctx, variable1)
	require.Nil(t, err)

	variableVersion, err = testClient.client.VariableVersions.GetVariableVersionByID(ctx, updatedVariable1.LatestVersionID)
	require.Nil(t, err)

	variableVersions = append(variableVersions, variableVersion)

	// Create variable2
	variable2, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
		Key:           "b",
		Value:         ptr.String("val-b"),
		NamespacePath: group.FullPath,
	})
	require.Nil(t, err)

	variableVersion, err = testClient.client.VariableVersions.GetVariableVersionByID(ctx, variable2.LatestVersionID)
	require.Nil(t, err)

	variableVersions = append(variableVersions, variableVersion)

	type testCase struct {
		filter            *VariableVersionFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name: "return all variables versions for variable 1",
			filter: &VariableVersionFilter{
				VariableID: &variable1.Metadata.ID,
			},
			expectResultCount: 2,
		},
		{
			name: "return all variable versions for variable 2",
			filter: &VariableVersionFilter{
				VariableID: &variable2.Metadata.ID,
			},
			expectResultCount: 1,
		},
		{
			name: "return all variable versions matching specific IDs",
			filter: &VariableVersionFilter{
				VariableVersionIDs: []string{variableVersions[0].Metadata.ID, variableVersions[1].Metadata.ID},
			},
			expectResultCount: 2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.VariableVersions.GetVariableVersions(ctx, &GetVariableVersionsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			assert.Equal(t, test.expectResultCount, len(result.VariableVersions))
		})
	}
}

func TestGetVariableVersionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group1",
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Variables.CreateVariable(ctx, &models.Variable{
			Key:           fmt.Sprintf("var-%d", i),
			Value:         ptr.String(fmt.Sprintf("val-%d", i)),
			NamespacePath: group.FullPath,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		VariableVersionSortableFieldCreatedAtAsc,
		VariableVersionSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := VariableVersionSortableField(sortByField.getValue())

		result, err := testClient.client.VariableVersions.GetVariableVersions(ctx, &GetVariableVersionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.VariableVersions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
