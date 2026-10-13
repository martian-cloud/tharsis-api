//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for UserSortableField
func (u UserSortableField) getValue() string {
	return string(u)
}

func TestGetUserByExternalID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a test user
	createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-external-user",
		Email:    "test-external@example.com",
		Active:   true,
	})
	require.Nil(t, err)

	// Link external ID
	issuer := "https://test-idp.com"
	externalID := "external-123"
	err = testClient.client.Users.LinkUserWithExternalID(ctx, issuer, externalID, createdUser.Metadata.ID)
	require.Nil(t, err)

	// Test positive case
	gotUser, err := testClient.client.Users.GetUserByExternalID(ctx, issuer, externalID)
	require.Nil(t, err)
	require.NotNil(t, gotUser)
	assert.Equal(t, createdUser.Metadata.ID, gotUser.Metadata.ID)

	// Test negative case
	gotUser, err = testClient.client.Users.GetUserByExternalID(ctx, issuer, "non-existent")
	require.Nil(t, err)
	assert.Nil(t, gotUser)
}
func TestUsers_CreateUser(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		username        string
		email           string
	}

	testCases := []testCase{
		{
			name:     "create user",
			username: "test-user",
			email:    "test@example.com",
		},
		{
			name:     "create user with invalid email",
			username: "invalid-user",
			email:    "invalid-email",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			user, err := testClient.client.Users.CreateUser(ctx, &models.User{
				Username: test.username,
				Email:    test.email,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, user)

			assert.Equal(t, test.username, user.Username)
			assert.Equal(t, test.email, user.Email)
			assert.NotEmpty(t, user.Metadata.ID)
		})
	}
}

func TestUsers_UpdateUser(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-update",
		Email:    "original@example.com",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		email           string
	}

	testCases := []testCase{
		{
			name:    "update user",
			version: createdUser.Metadata.Version,
			email:   "updated@example.com",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			email:           "should-not-update@example.com",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			userToUpdate := *createdUser
			userToUpdate.Metadata.Version = test.version
			userToUpdate.Email = test.email

			updatedUser, err := testClient.client.Users.UpdateUser(ctx, &userToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedUser)

			assert.Equal(t, test.email, updatedUser.Email)
			assert.Equal(t, createdUser.Metadata.Version+1, updatedUser.Metadata.Version)
		})
	}
}

func TestUsers_DeleteUser(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-delete",
		Email:    "delete@example.com",
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
			name:    "delete user",
			id:      createdUser.Metadata.ID,
			version: createdUser.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdUser.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Users.DeleteUser(ctx, &models.User{
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

			// Verify user was deleted
			user, err := testClient.client.Users.GetUserByID(ctx, test.id)
			assert.Nil(t, user)
			assert.Nil(t, err)
		})
	}
}

func TestUsers_GetUserByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-get-by-id",
		Email:    "test-get-by-id@example.com",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectUser      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by id",
			id:         createdUser.Metadata.ID,
			expectUser: true,
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
			user, err := testClient.client.Users.GetUserByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectUser {
				require.NotNil(t, user)
				assert.Equal(t, test.id, user.Metadata.ID)
			} else {
				assert.Nil(t, user)
			}
		})
	}
}

func TestUsers_GetUsers(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test users
	users := []models.User{
		{
			Username: "test-user-1",
			Email:    "test-user-1@example.com",
		},
		{
			Username: "test-user-2",
			Email:    "test-user-2@example.com",
		},
	}

	createdUsers := []models.User{}
	for _, user := range users {
		created, err := testClient.client.Users.CreateUser(ctx, &user)
		require.NoError(t, err)
		createdUsers = append(createdUsers, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetUsersInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all users",
			input:       &GetUsersInput{},
			expectCount: len(createdUsers),
		},
		{
			name: "filter by username",
			input: &GetUsersInput{
				Filter: &UserFilter{
					Search: ptr.String("test-user-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by username prefix",
			input: &GetUsersInput{
				Filter: &UserFilter{
					UsernamePrefix: ptr.String("test-user"),
				},
			},
			expectCount: len(createdUsers),
		},
		{
			name: "filter by user IDs",
			input: &GetUsersInput{
				Filter: &UserFilter{
					UserIDs: []string{createdUsers[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Users.GetUsers(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Users, test.expectCount)
		})
	}
}

func TestUsers_GetUsersWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Users.CreateUser(ctx, &models.User{
			Username: fmt.Sprintf("user-%d", i),
			Email:    fmt.Sprintf("user-%d@example.com", i),
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		UserSortableFieldUpdatedAtAsc,
		UserSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := UserSortableField(sortByField.getValue())

		result, err := testClient.client.Users.GetUsers(ctx, &GetUsersInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Users {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestUsers_GetUserByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for testing
	createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-trn",
		Email:    "test-user-trn@example.com",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectUser      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by TRN",
			trn:        createdUser.Metadata.TRN,
			expectUser: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:user:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			user, err := testClient.client.Users.GetUserByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectUser {
				require.NotNil(t, user)
				assert.Equal(t, test.trn, user.Metadata.TRN)
			} else {
				assert.Nil(t, user)
			}
		})
	}
}

func TestUsers_LinkUserWithExternalID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user first
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test@example.com",
		Active:   true,
	})
	require.Nil(t, err)

	// Test successful linking
	err = testClient.client.Users.LinkUserWithExternalID(ctx, "https://example.com", "ext-123", user.Metadata.ID)
	assert.Nil(t, err)

	// Verify link was created
	linkedUser, err := testClient.client.Users.GetUserByExternalID(ctx, "https://example.com", "ext-123")
	assert.Nil(t, err)
	assert.Equal(t, user.Metadata.ID, linkedUser.Metadata.ID)

	// Test duplicate external ID returns conflict error
	err = testClient.client.Users.LinkUserWithExternalID(ctx, "https://example.com", "ext-123", user.Metadata.ID)
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
}

func TestUsers_HasExternalIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	linkedUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "linked-user",
		Email:    "linked@example.com",
		Active:   true,
	})
	require.Nil(t, err)

	unlinkedUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "unlinked-user",
		Email:    "unlinked@example.com",
		Active:   true,
	})
	require.Nil(t, err)

	issuer := "https://example.com"
	require.Nil(t, testClient.client.Users.LinkUserWithExternalID(ctx, issuer, "ext-123", linkedUser.Metadata.ID))

	testCases := []struct {
		name   string
		userID string
		issuer *string
		expect bool
	}{
		{name: "linked user, any issuer", userID: linkedUser.Metadata.ID, expect: true},
		{name: "linked user, matching issuer", userID: linkedUser.Metadata.ID, issuer: &issuer, expect: true},
		{name: "linked user, other issuer", userID: linkedUser.Metadata.ID, issuer: ptr.String("https://other.com")},
		{name: "unlinked user, any issuer", userID: unlinkedUser.Metadata.ID},
		{name: "unlinked user, matching issuer", userID: unlinkedUser.Metadata.ID, issuer: &issuer},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			exists, err := testClient.client.Users.HasExternalIdentity(ctx, test.userID, test.issuer)
			require.Nil(t, err)
			assert.Equal(t, test.expect, exists)
		})
	}
}

// TestUsers_SCIMIDPIssuerURL checks that a user's SCIM IDP issuer URL is stored, kept across updates and
// usable as a filter, which is how a SCIM client is limited to its own IDP's users.
func TestUsers_SCIMIDPIssuerURL(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	const (
		issuerA = "https://idp-a.example.com"
		issuerB = "https://idp-b.example.com"
	)

	users := map[string]*models.User{}
	for _, user := range []*models.User{
		{Username: "user-a", Email: "user-a@example.com", SCIMExternalID: "11111111-1111-1111-1111-111111111111", SCIMIDPIssuerURL: issuerA},
		{Username: "user-b", Email: "user-b@example.com", SCIMExternalID: "22222222-2222-2222-2222-222222222222", SCIMIDPIssuerURL: issuerB},
		{Username: "user-manual", Email: "user-manual@example.com"},
	} {
		created, err := testClient.client.Users.CreateUser(ctx, user)
		require.NoError(t, err)
		assert.Equal(t, user.SCIMIDPIssuerURL, created.SCIMIDPIssuerURL)
		users[user.Username] = created
	}

	t.Run("kept across an update", func(t *testing.T) {
		user := users["user-a"]
		user.Active = !user.Active
		updated, err := testClient.client.Users.UpdateUser(ctx, user)
		require.NoError(t, err)
		assert.Equal(t, issuerA, updated.SCIMIDPIssuerURL)

		fetched, err := testClient.client.Users.GetUserByID(ctx, user.Metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, issuerA, fetched.SCIMIDPIssuerURL)
	})

	for _, test := range []struct {
		issuer        string
		expectedUsers []string
	}{
		{issuer: issuerA, expectedUsers: []string{"user-a"}},
		{issuer: issuerB, expectedUsers: []string{"user-b"}},
		{issuer: "https://unknown.example.com", expectedUsers: []string{}},
	} {
		t.Run("filter by "+test.issuer, func(t *testing.T) {
			issuer := test.issuer
			result, err := testClient.client.Users.GetUsers(ctx, &GetUsersInput{
				Filter: &UserFilter{SCIMExternalID: true, SCIMIDPIssuerURL: &issuer},
			})
			require.NoError(t, err)

			got := []string{}
			for _, user := range result.Users {
				got = append(got, user.Username)
			}
			assert.ElementsMatch(t, test.expectedUsers, got)
		})
	}
}

func TestUsers_GetUserBySCIMExternalID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	issuer := "https://idp.example.com"
	externalID := uuid.New().String()

	createdUser, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username:         "scim-user",
		Email:            "scim-user@example.com",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuer,
		Active:           true,
	})
	require.Nil(t, err)

	// The same external ID is allowed for a user of another IDP.
	_, err = testClient.client.Users.CreateUser(ctx, &models.User{
		Username:         "other-idp-user",
		Email:            "other-idp-user@example.com",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: "https://other-idp.example.com",
		Active:           true,
	})
	require.Nil(t, err)

	// But not twice for the same IDP.
	_, err = testClient.client.Users.CreateUser(ctx, &models.User{
		Username:         "duplicate-user",
		Email:            "duplicate-user@example.com",
		SCIMExternalID:   externalID,
		SCIMIDPIssuerURL: issuer,
		Active:           true,
	})
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))

	gotUser, err := testClient.client.Users.GetUserBySCIMExternalID(ctx, issuer, externalID)
	require.Nil(t, err)
	require.NotNil(t, gotUser)
	assert.Equal(t, createdUser.Metadata.ID, gotUser.Metadata.ID)

	gotUser, err = testClient.client.Users.GetUserBySCIMExternalID(ctx, "https://unknown-idp.example.com", externalID)
	require.Nil(t, err)
	assert.Nil(t, gotUser)
}

func TestUsers_UpdateUserConflicts(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, err := testClient.client.Users.CreateUser(ctx, &models.User{Username: "existing-user", Email: "existing@example.com", Active: true})
	require.Nil(t, err)

	tests := []struct {
		name          string
		update        func(u *models.User)
		expectMessage string
	}{
		{
			name:          "duplicate username",
			update:        func(u *models.User) { u.Username = "existing-user" },
			expectMessage: "user with username existing-user already exists",
		},
		{
			name:          "duplicate email",
			update:        func(u *models.User) { u.Email = "existing@example.com" },
			expectMessage: "user with email existing@example.com already exists",
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user, err := testClient.client.Users.CreateUser(ctx, &models.User{
				Username: fmt.Sprintf("other-user-%d", i),
				Email:    fmt.Sprintf("other-%d@example.com", i),
				Active:   true,
			})
			require.Nil(t, err)

			test.update(user)
			_, err = testClient.client.Users.UpdateUser(ctx, user)
			require.NotNil(t, err)
			assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
			assert.Equal(t, test.expectMessage, errors.ErrorMessage(err))
		})
	}
}
