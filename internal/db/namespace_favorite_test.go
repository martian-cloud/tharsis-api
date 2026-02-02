//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestCreateNamespaceFavorite(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		favorite        *models.NamespaceFavorite
	}

	testCases := []testCase{
		{
			name: "successfully create namespace favorite",
			favorite: &models.NamespaceFavorite{
				UserID:  user.Metadata.ID,
				GroupID: &group.Metadata.ID,
			},
		},
		{
			name: "create will fail because favorite already exists",
			favorite: &models.NamespaceFavorite{
				UserID:  user.Metadata.ID,
				GroupID: &group.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "create will fail because user does not exist",
			favorite: &models.NamespaceFavorite{
				UserID:  nonExistentID,
				GroupID: &group.Metadata.ID,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "create will fail because group does not exist",
			favorite: &models.NamespaceFavorite{
				UserID: user.Metadata.ID,
				GroupID: func() *string {
					id := nonExistentID
					return &id
				}(),
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			namespaceFavorite, err := testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, test.favorite)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, namespaceFavorite)
			assert.Equal(t, test.favorite.UserID, namespaceFavorite.UserID)
			assert.Equal(t, test.favorite.GroupID, namespaceFavorite.GroupID)
			assert.Equal(t, test.favorite.WorkspaceID, namespaceFavorite.WorkspaceID)
		})
	}
}

func TestDeleteNamespaceFavorite(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	namespaceFavorite, err := testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user.Metadata.ID,
		GroupID: &group.Metadata.ID,
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
			name:            "delete will fail because resource version doesn't match",
			id:              namespaceFavorite.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete namespace favorite",
			id:      namespaceFavorite.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.NamespaceFavorites.DeleteNamespaceFavorite(ctx, &models.NamespaceFavorite{
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
		})
	}
}

func TestGetNamespaceFavorites(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group-1",
	})
	require.Nil(t, err)

	group2, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group-2",
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-1",
		Email:    "test-user-1@test.com",
	})
	require.Nil(t, err)

	user2, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-2",
		Email:    "test-user-2@test.com",
	})
	require.Nil(t, err)

	_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user1.Metadata.ID,
		GroupID: &group1.Metadata.ID,
	})
	require.Nil(t, err)

	_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user1.Metadata.ID,
		GroupID: &group2.Metadata.ID,
	})
	require.Nil(t, err)

	_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user2.Metadata.ID,
		GroupID: &group1.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		filter            *NamespaceFavoriteFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name: "return all favorites for user 1",
			filter: &NamespaceFavoriteFilter{
				UserIDs: []string{user1.Metadata.ID},
			},
			expectResultCount: 2,
		},
		{
			name: "return all favorites for user 2",
			filter: &NamespaceFavoriteFilter{
				UserIDs: []string{user2.Metadata.ID},
			},
			expectResultCount: 1,
		},
		{
			name: "return favorites for specific namespace",
			filter: &NamespaceFavoriteFilter{
				NamespacePath: &group1.FullPath,
			},
			expectResultCount: 2,
		},
		{
			name: "return favorites for user 1 and specific namespace",
			filter: &NamespaceFavoriteFilter{
				UserIDs:       []string{user1.Metadata.ID},
				NamespacePath: &group1.FullPath,
			},
			expectResultCount: 1,
		},
		{
			name: "search favorites by partial path match",
			filter: &NamespaceFavoriteFilter{
				UserIDs: []string{user1.Metadata.ID},
				Search:  func() *string { s := "group-1"; return &s }(),
			},
			expectResultCount: 1,
		},
		{
			name: "search favorites returns no results for non-matching query",
			filter: &NamespaceFavoriteFilter{
				UserIDs: []string{user1.Metadata.ID},
				Search:  func() *string { s := "nonexistent"; return &s }(),
			},
			expectResultCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.NamespaceFavorites.GetNamespaceFavorites(ctx, &GetNamespaceFavoritesInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			assert.Equal(t, test.expectResultCount, len(result.NamespaceFavorites))
		})
	}
}

func TestGetNamespaceFavoritesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Name: fmt.Sprintf("test-group-%d", i),
		})
		require.Nil(t, err)

		_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
			UserID:  user.Metadata.ID,
			GroupID: &group.Metadata.ID,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		NamespaceFavoriteSortableFieldCreatedAtAsc,
		NamespaceFavoriteSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := NamespaceFavoriteSortableField(sortByField.getValue())

		result, err := testClient.client.NamespaceFavorites.GetNamespaceFavorites(ctx, &GetNamespaceFavoritesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
			Filter: &NamespaceFavoriteFilter{
				UserIDs: []string{user.Metadata.ID},
			},
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.NamespaceFavorites {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestGetNamespaceFavoriteByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	createdFavorite, err := testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user.Metadata.ID,
		GroupID: &group.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		id              string
		expectErrorCode errors.CodeType
		expectNil       bool
	}

	testCases := []testCase{
		{
			name: "successfully get namespace favorite by ID",
			id:   createdFavorite.Metadata.ID,
		},
		{
			name:      "return nil when favorite not found",
			id:        "12345678-1234-1234-1234-123456789abc",
			expectNil: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			favorite, err := testClient.client.NamespaceFavorites.GetNamespaceFavoriteByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			if test.expectNil {
				assert.Nil(t, favorite)
				return
			}

			require.NotNil(t, favorite)
			assert.Equal(t, createdFavorite.Metadata.ID, favorite.Metadata.ID)
			assert.Equal(t, createdFavorite.UserID, favorite.UserID)
			assert.Equal(t, createdFavorite.GroupID, favorite.GroupID)
			assert.Equal(t, createdFavorite.WorkspaceID, favorite.WorkspaceID)
		})
	}
}

func TestGetNamespaceFavoriteByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	createdFavorite, err := testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user.Metadata.ID,
		GroupID: &group.Metadata.ID,
	})
	require.Nil(t, err)
	require.NotNil(t, createdFavorite)

	type testCase struct {
		name            string
		trn             string
		expectErrorCode errors.CodeType
		expectNil       bool
	}

	testCases := []testCase{
		{
			name: "successfully get namespace favorite by TRN",
			trn: func() string {
				if createdFavorite.Metadata.TRN != "" {
					return createdFavorite.Metadata.TRN
				}
				return types.NamespaceFavoriteModelType.BuildTRN(group.FullPath, gid.ToGlobalID(types.UserModelType, user.Metadata.ID))
			}(),
		},
		{
			name:            "return error for invalid TRN format",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "return error for TRN without namespace path",
			trn:             types.NamespaceFavoriteModelType.BuildTRN("TkZfYmY0ZTJjNjUtYjc4YS00ODZjLWI4MjAtNzQ4YTkyNjI1MDE3"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:      "return nil when favorite not found",
			trn:       types.NamespaceFavoriteModelType.BuildTRN(group.FullPath, "TkZfYmY0ZTJjNjUtYjc4YS00ODZjLWI4MjAtNzQ4YTkyNjI1MDE3"),
			expectNil: true,
		},
		{
			name:      "return nil when TRN namespace path does not match",
			trn:       types.NamespaceFavoriteModelType.BuildTRN("wrong/path", gid.ToGlobalID(types.UserModelType, user.Metadata.ID)),
			expectNil: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			favorite, err := testClient.client.NamespaceFavorites.GetNamespaceFavoriteByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			if test.expectNil {
				assert.Nil(t, favorite)
				return
			}

			require.NotNil(t, favorite)
			assert.Equal(t, createdFavorite.Metadata.ID, favorite.Metadata.ID)
			assert.Equal(t, createdFavorite.UserID, favorite.UserID)
			assert.Equal(t, createdFavorite.GroupID, favorite.GroupID)
			assert.Equal(t, createdFavorite.WorkspaceID, favorite.WorkspaceID)
		})
	}
}
