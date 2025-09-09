//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestUserSessions_GetUserSessionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test@example.com",
	})
	require.NoError(t, err)

	userSession, err := testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
		UserID:         user.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "test-user-agent",
		Expiration:     time.Now().Add(time.Hour).UTC(),
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode   errors.CodeType
		name              string
		id                string
		expectUserSession bool
	}

	testCases := []testCase{
		{
			name:              "get resource by id",
			id:                userSession.Metadata.ID,
			expectUserSession: true,
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
			userSession, err := testClient.client.UserSessions.GetUserSessionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectUserSession {
				require.NotNil(t, userSession)
				assert.Equal(t, test.id, userSession.Metadata.ID)
			} else {
				assert.Nil(t, userSession)
			}
		})
	}
}

func TestUserSessions_CreateUserSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test@example.com",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		userID          string
		refreshTokenID  string
		userAgent       string
	}

	testCases := []testCase{
		{
			name:           "successfully create resource",
			userID:         user.Metadata.ID,
			refreshTokenID: uuid.New().String(),
			userAgent:      "test-user-agent",
		},
		{
			name:            "create will fail because user does not exist",
			userID:          nonExistentID,
			refreshTokenID:  uuid.New().String(),
			userAgent:       "test-user-agent",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			userSession, err := testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
				UserID:         test.userID,
				RefreshTokenID: test.refreshTokenID,
				UserAgent:      test.userAgent,
				Expiration:     time.Now().Add(time.Hour).UTC(),
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, userSession)
			assert.Equal(t, test.userID, userSession.UserID)
			assert.Equal(t, test.refreshTokenID, userSession.RefreshTokenID)
			assert.Equal(t, test.userAgent, userSession.UserAgent)
		})
	}
}

func TestUserSessions_UpdateUserSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test@example.com",
	})
	require.NoError(t, err)

	userSession, err := testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
		UserID:         user.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "test-user-agent",
		Expiration:     time.Now().Add(time.Hour).UTC(),
	})
	require.NoError(t, err)

	newExpiration := time.Now().Add(2 * time.Hour).UTC()

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		refreshTokenID  string
	}

	testCases := []testCase{
		{
			name:           "successfully update resource",
			version:        1,
			refreshTokenID: uuid.New().String(),
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			refreshTokenID:  uuid.New().String(),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualUserSession, err := testClient.client.UserSessions.UpdateUserSession(ctx, &models.UserSession{
				Metadata: models.ResourceMetadata{
					ID:      userSession.Metadata.ID,
					Version: test.version,
				},
				RefreshTokenID: test.refreshTokenID,
				Expiration:     newExpiration,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualUserSession)
			assert.Equal(t, test.refreshTokenID, actualUserSession.RefreshTokenID)
			assert.Equal(t, newExpiration.Format(time.RFC3339), actualUserSession.Expiration.Format(time.RFC3339))
		})
	}
}

func TestUserSessions_DeleteUserSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test@example.com",
	})
	require.NoError(t, err)

	userSession, err := testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
		UserID:         user.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "test-user-agent",
		Expiration:     time.Now().Add(time.Hour).UTC(),
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:            "delete will fail because resource version doesn't match",
			id:              userSession.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      userSession.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.UserSessions.DeleteUserSession(ctx, &models.UserSession{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestUserSessions_GetUserSessions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-1",
		Email:    "test1@example.com",
	})
	require.NoError(t, err)

	user2, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-2",
		Email:    "test2@example.com",
	})
	require.NoError(t, err)

	sessions := []*models.UserSession{}

	session, err := testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
		UserID:         user1.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "user-agent-1",
		Expiration:     time.Now().Add(time.Hour).UTC(),
	})
	require.NoError(t, err)
	sessions = append(sessions, session)

	session, err = testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
		UserID:         user1.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "user-agent-2",
		Expiration:     time.Now().Add(2 * time.Hour).UTC(),
	})
	require.NoError(t, err)
	sessions = append(sessions, session)

	session, err = testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
		UserID:         user2.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "user-agent-3",
		Expiration:     time.Now().Add(time.Hour).UTC(),
	})
	require.NoError(t, err)
	sessions = append(sessions, session)

	type testCase struct {
		filter            *UserSessionFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name: "return all sessions for user 1",
			filter: &UserSessionFilter{
				UserID: ptr.String(user1.Metadata.ID),
			},
			expectResultCount: 2,
		},
		{
			name: "return all sessions for user 2",
			filter: &UserSessionFilter{
				UserID: ptr.String(user2.Metadata.ID),
			},
			expectResultCount: 1,
		},
		{
			name: "return session by refresh token ID",
			filter: &UserSessionFilter{
				RefreshTokenID: &sessions[1].RefreshTokenID,
			},
			expectResultCount: 1,
		},
		{
			name: "return sessions by session IDs",
			filter: &UserSessionFilter{
				UserSessionIDs: []string{sessions[0].Metadata.ID, sessions[2].Metadata.ID},
			},
			expectResultCount: 2,
		},
		{
			name:              "return all sessions when no filter is provided",
			filter:            nil,
			expectResultCount: 3,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.UserSessions.GetUserSessions(ctx, &GetUserSessionsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.UserSessions))
		})
	}
}

func TestUserSessions_GetUserSessionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user",
		Email:    "test@example.com",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err = testClient.client.UserSessions.CreateUserSession(ctx, &models.UserSession{
			UserID:         user.Metadata.ID,
			RefreshTokenID: uuid.New().String(),
			UserAgent:      fmt.Sprintf("user-agent-%d", i),
			Expiration:     time.Now().Add(time.Duration(i+1) * time.Hour).UTC(),
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		UserSessionSortableFieldCreatedAtAsc,
		UserSessionSortableFieldCreatedAtDesc,
		UserSessionSortableFieldExpirationAsc,
		UserSessionSortableFieldExpirationDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := UserSessionSortableField(sortByField.getValue())

		result, err := testClient.client.UserSessions.GetUserSessions(ctx, &GetUserSessionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.UserSessions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestGetUserSessionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user first
	user := &models.User{
		Username: "test-user",
		Email:    "test@example.com",
	}

	createdUser, err := testClient.client.Users.CreateUser(ctx, user)
	require.NoError(t, err)

	// Create a user session
	userSession := &models.UserSession{
		UserID:         createdUser.Metadata.ID,
		RefreshTokenID: uuid.New().String(),
		UserAgent:      "test-agent",
		Expiration:     time.Now().Add(24 * time.Hour),
	}

	createdSession, err := testClient.client.UserSessions.CreateUserSession(ctx, userSession)
	require.NoError(t, err)
	require.NotNil(t, createdSession)

	// Test getting the session by TRN
	retrievedSession, err := testClient.client.UserSessions.GetUserSessionByTRN(ctx, createdSession.Metadata.TRN)
	require.NoError(t, err)
	require.NotNil(t, retrievedSession)

	// Verify the retrieved session matches the created one
	assert.Equal(t, createdSession.Metadata.ID, retrievedSession.Metadata.ID)
	assert.Equal(t, createdSession.UserID, retrievedSession.UserID)
	assert.Equal(t, createdSession.RefreshTokenID, retrievedSession.RefreshTokenID)
	assert.Equal(t, createdSession.UserAgent, retrievedSession.UserAgent)
	assert.Equal(t, createdSession.Metadata.TRN, retrievedSession.Metadata.TRN)

	// Test with invalid TRN
	_, err = testClient.client.UserSessions.GetUserSessionByTRN(ctx, "invalid-trn")
	assert.Error(t, err)
}
