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

func (sf NotificationPreferenceSortableField) getValue() string {
	return string(sf)
}

func TestGetNotificationPreferenceByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a test user
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	notificationPreference, err := testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, &models.NotificationPreference{
		UserID: user.Metadata.ID,
		Scope:  models.NotificationPreferenceScopeAll,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode              errors.CodeType
		name                         string
		id                           string
		expectNotificationPreference bool
	}

	testCases := []testCase{
		{
			name:                         "get notification preference by id",
			id:                           notificationPreference.Metadata.ID,
			expectNotificationPreference: true,
		},
		{
			name: "notification preference with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get notification preference with invalid id returns error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			notificationPreference, err := testClient.client.NotificationPreferences.GetNotificationPreferenceByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectNotificationPreference {
				require.NotNil(t, notificationPreference)
				assert.Equal(t, test.id, notificationPreference.Metadata.ID)
			} else {
				assert.Nil(t, notificationPreference)
			}
		})
	}
}

func TestCreateNotificationPreference(t *testing.T) {
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
		preference      *models.NotificationPreference
	}

	testCases := []testCase{
		{
			name: "successfully create global notification preference",
			preference: &models.NotificationPreference{
				UserID:       user.Metadata.ID,
				Scope:        models.NotificationPreferenceScopeCustom,
				CustomEvents: &models.NotificationPreferenceCustomEvents{FailedRun: true},
			},
		},
		{
			name: "successfully create namespace notification preference",
			preference: &models.NotificationPreference{
				UserID:        user.Metadata.ID,
				Scope:         models.NotificationPreferenceScopeCustom,
				CustomEvents:  &models.NotificationPreferenceCustomEvents{FailedRun: true},
				NamespacePath: &group.FullPath,
			},
		},
		{
			name: "create will fail because preference already exists for this namespace",
			preference: &models.NotificationPreference{
				UserID:        user.Metadata.ID,
				Scope:         models.NotificationPreferenceScopeAll,
				NamespacePath: &group.FullPath,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "create will fail because user does not exist",
			preference: &models.NotificationPreference{
				UserID: nonExistentID,
				Scope:  models.NotificationPreferenceScopeAll,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "create will fail because namespace does not exist",
			preference: &models.NotificationPreference{
				UserID:        user.Metadata.ID,
				Scope:         models.NotificationPreferenceScopeAll,
				NamespacePath: ptr.String("invalid-namespace"),
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			notificationPreference, err := testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, test.preference)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, notificationPreference)
		})
	}
}

func TestUpdateNotificationPreference(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	notificationPreference, err := testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, &models.NotificationPreference{
		UserID: user.Metadata.ID,
		Scope:  models.NotificationPreferenceScopeAll,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
	}

	testCases := []testCase{
		{
			name:    "successfully update notification preference",
			version: 1,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			updatedPreference, err := testClient.client.NotificationPreferences.UpdateNotificationPreference(ctx, &models.NotificationPreference{
				Metadata: models.ResourceMetadata{
					ID:      notificationPreference.Metadata.ID,
					Version: test.version,
				},
				UserID: user.Metadata.ID,
				Scope:  models.NotificationPreferenceScopeAll,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedPreference)
		})
	}
}

func TestDeleteNotificationPreference(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test-user@test.com",
	})
	require.Nil(t, err)

	notificationPreference, err := testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, &models.NotificationPreference{
		UserID: user.Metadata.ID,
		Scope:  models.NotificationPreferenceScopeAll,
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
			id:              notificationPreference.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete notification preference",
			id:      notificationPreference.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.NotificationPreferences.DeleteNotificationPreference(ctx, &models.NotificationPreference{
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

func TestGetNotificationPreferences(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

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

	_, err = testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, &models.NotificationPreference{
		UserID: user1.Metadata.ID,
		Scope:  models.NotificationPreferenceScopeAll,
	})
	require.Nil(t, err)

	_, err = testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, &models.NotificationPreference{
		UserID: user2.Metadata.ID,
		Scope:  models.NotificationPreferenceScopeAll,
	})
	require.Nil(t, err)

	type testCase struct {
		filter            *NotificationPreferenceFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name: "return all preferences for user 1",
			filter: &NotificationPreferenceFilter{
				UserIDs: []string{user1.Metadata.ID},
			},
			expectResultCount: 1,
		},
		{
			name: "return all email type preferences",
			filter: &NotificationPreferenceFilter{
				Global: ptr.Bool(true),
			},
			expectResultCount: 2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.NotificationPreferences.GetNotificationPreferences(ctx, &GetNotificationPreferencesInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			assert.Equal(t, test.expectResultCount, len(result.NotificationPreferences))
		})
	}
}

func TestGetNotificationPreferencesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		user, err := testClient.client.Users.CreateUser(ctx, &models.User{
			Username: fmt.Sprintf("test-user-%d", i),
			Email:    fmt.Sprintf("test-user-%d@test.com", i),
		})
		require.Nil(t, err)

		_, err = testClient.client.NotificationPreferences.CreateNotificationPreference(ctx, &models.NotificationPreference{
			UserID: user.Metadata.ID,
			Scope:  models.NotificationPreferenceScopeAll,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		NotificationPreferenceSortableFieldUpdatedAtAsc,
		NotificationPreferenceSortableFieldUpdatedAtDesc,
		NotificationPreferenceSortableFieldCreatedAtAsc,
		NotificationPreferenceSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := NotificationPreferenceSortableField(sortByField.getValue())

		result, err := testClient.client.NotificationPreferences.GetNotificationPreferences(ctx, &GetNotificationPreferencesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.NotificationPreferences {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
