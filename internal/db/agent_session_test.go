//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestAgentSessions_CreateAgentSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-agent-session",
		Email:    "agent-session@example.com",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		userID          string
	}

	testCases := []testCase{
		{
			name:   "create agent session",
			userID: user.Metadata.ID,
		},
		{
			name:            "create agent session with invalid user ID",
			userID:          invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
				UserID:       test.userID,
				TotalCredits: 0,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, session)
			assert.Equal(t, test.userID, session.UserID)
			assert.Equal(t, float64(0), session.TotalCredits)
			assert.NotEmpty(t, session.Metadata.ID)
		})
	}
}

func TestAgentSessions_GetAgentSessionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-get-session",
		Email:    "get-session@example.com",
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		expectSession   bool
	}

	testCases := []testCase{
		{
			name:          "get session by id",
			id:            created.Metadata.ID,
			expectSession: true,
		},
		{
			name: "session not found",
			id:   nonExistentID,
		},
		{
			name:            "invalid id",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			session, err := testClient.client.AgentSessions.GetAgentSessionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			if test.expectSession {
				require.NotNil(t, session)
				assert.Equal(t, created.Metadata.ID, session.Metadata.ID)
			} else {
				assert.Nil(t, session)
			}
		})
	}
}

func TestAgentSessions_UpdateAgentSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-update-session",
		Email:    "update-session@example.com",
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		session         *models.AgentSession
	}

	testCases := []testCase{
		{
			name: "update agent session",
			session: &models.AgentSession{
				Metadata:     created.Metadata,
				UserID:       created.UserID,
				TotalCredits: 42.5,
			},
		},
		{
			name: "update fails with wrong version",
			session: &models.AgentSession{
				Metadata: models.ResourceMetadata{
					ID:      created.Metadata.ID,
					Version: -1,
				},
			},
			expectErrorCode: errors.EOptimisticLock,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			updated, err := testClient.client.AgentSessions.UpdateAgentSession(ctx, test.session)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, 42.5, updated.TotalCredits)
		})
	}
}

func TestAgentSessions_DeleteOldestSessionsByUserID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-delete-oldest",
		Email:    "delete-oldest@example.com",
	})
	require.Nil(t, err)

	// Create 5 sessions, track their IDs
	sessionIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		s, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
			UserID: user.Metadata.ID,
		})
		require.Nil(t, err)
		sessionIDs[i] = s.Metadata.ID
	}

	// Keep only 2 (the 2 most recent)
	err = testClient.client.AgentSessions.DeleteOldestSessionsByUserID(ctx, user.Metadata.ID, 2)
	require.Nil(t, err)

	// The 3 oldest should be deleted
	for _, id := range sessionIDs[:3] {
		s, err := testClient.client.AgentSessions.GetAgentSessionByID(ctx, id)
		require.Nil(t, err)
		assert.Nil(t, s)
	}

	// The 2 newest should still exist
	for _, id := range sessionIDs[3:] {
		s, err := testClient.client.AgentSessions.GetAgentSessionByID(ctx, id)
		require.Nil(t, err)
		require.NotNil(t, s)
	}
}

func TestAgentSessions_GetAgentSessionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-session-trn",
		Email:    "session-trn@example.com",
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		trn             string
		expectSession   bool
	}

	testCases := []testCase{
		{
			name:          "get session by TRN",
			trn:           types.AgentSessionModelType.BuildTRN(created.GetGlobalID()),
			expectSession: true,
		},
		{
			name:            "invalid TRN",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			session, err := testClient.client.AgentSessions.GetAgentSessionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			if test.expectSession {
				require.NotNil(t, session)
				assert.Equal(t, created.Metadata.ID, session.Metadata.ID)
			} else {
				assert.Nil(t, session)
			}
		})
	}
}
