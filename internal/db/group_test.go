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

// getValue implements the sortableField interface for GroupSortableField
func (g GroupSortableField) getValue() string {
	return string(g)
}

func TestGroups_CreateGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		groupName       string
		description     string
		parentID        string
		fullPath        string
	}

	testCases := []testCase{
		{
			name:        "create group",
			groupName:   "test-group",
			description: "test group description",
			fullPath:    "test-group",
		},
		{
			name:            "negative, child without parent",
			groupName:       "orphan-child",
			description:     "this is a child without a parent",
			parentID:        invalidID,
			fullPath:        "missing-parent/orphan-child",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
				Name:        test.groupName,
				Description: test.description,
				ParentID:    test.parentID,
				FullPath:    test.fullPath,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, group)

			assert.Equal(t, test.groupName, group.Name)
			assert.Equal(t, test.description, group.Description)
			assert.Equal(t, test.fullPath, group.FullPath)
			assert.NotEmpty(t, group.Metadata.ID)
		})
	}
}

func TestGroups_UpdateGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-update",
		Description: "original description",
		FullPath:    "test-group-update",
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
			name:        "update group",
			version:     createdGroup.Metadata.Version,
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
			groupToUpdate := *createdGroup
			groupToUpdate.Metadata.Version = test.version
			groupToUpdate.Description = test.description

			updatedGroup, err := testClient.client.Groups.UpdateGroup(ctx, &groupToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedGroup)

			assert.Equal(t, test.description, updatedGroup.Description)
			assert.Equal(t, createdGroup.Metadata.Version+1, updatedGroup.Metadata.Version)
		})
	}
}

func TestGroups_DeleteGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-delete",
		Description: "group to delete",
		FullPath:    "test-group-delete",
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
			name:    "delete group",
			id:      createdGroup.Metadata.ID,
			version: createdGroup.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdGroup.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Groups.DeleteGroup(ctx, &models.Group{
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

			// Verify group was deleted
			group, err := testClient.client.Groups.GetGroupByID(ctx, test.id)
			assert.Nil(t, group)
			assert.Nil(t, err)
		})
	}
}

func TestGroups_GetGroupByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-get-by-id",
		Description: "test group for get by id",
		FullPath:    "test-group-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectGroup     bool
	}

	testCases := []testCase{
		{
			name:        "get resource by id",
			id:          createdGroup.Metadata.ID,
			expectGroup: true,
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
			group, err := testClient.client.Groups.GetGroupByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGroup {
				require.NotNil(t, group)
				assert.Equal(t, test.id, group.Metadata.ID)
			} else {
				assert.Nil(t, group)
			}
		})
	}
}

func TestGroups_GetGroups(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test groups
	groups := []models.Group{
		{
			Name:        "test-group-list-1",
			Description: "test group for list 1",
			FullPath:    "test-group-list-1",
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-group-list-2",
			Description: "test group for list 2",
			FullPath:    "test-group-list-2",
			CreatedBy:   "db-integration-tests",
		},
	}

	createdGroups := []models.Group{}
	for _, group := range groups {
		created, err := testClient.client.Groups.CreateGroup(ctx, &group)
		require.NoError(t, err)
		createdGroups = append(createdGroups, *created)
	}
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-groups",
		Email:    "test-user-groups@test.com",
	})
	require.NoError(t, err)

	_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user.Metadata.ID,
		GroupID: &createdGroups[0].Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetGroupsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all groups",
			input:       &GetGroupsInput{},
			expectCount: len(createdGroups),
		},
		{
			name: "get groups with favorite filter",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					FavoriteUserID: &user.Metadata.ID,
				},
			},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Groups.GetGroups(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Groups, test.expectCount)
		})
	}
}

func TestGroups_GetGroupsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Name:        fmt.Sprintf("test-group-pagination-%d", i),
			Description: fmt.Sprintf("test group for pagination %d", i),
			FullPath:    fmt.Sprintf("test-group-pagination-%d", i),
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	// Only test the sortable fields that work reliably
	sortableFields := []sortableField{
		GroupSortableFieldFullPathAsc,
		GroupSortableFieldFullPathDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := GroupSortableField(sortByField.getValue())

		result, err := testClient.client.Groups.GetGroups(ctx, &GetGroupsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Groups {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestGroups_GetGroupByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-trn",
		Description: "test group for trn",
		FullPath:    "test-group-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectGroup     bool
	}

	testCases := []testCase{
		{
			name:        "get resource by TRN",
			trn:         createdGroup.Metadata.TRN,
			expectGroup: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:group:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			group, err := testClient.client.Groups.GetGroupByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGroup {
				require.NotNil(t, group)
				assert.Equal(t, test.trn, group.Metadata.TRN)
			} else {
				assert.Nil(t, group)
			}
		})
	}
}
