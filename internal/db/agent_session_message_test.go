//go:build integration

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for AgentSessionMessageSortableField
func (sf AgentSessionMessageSortableField) getValue() string {
	return string(sf)
}

func TestAgentSessionMessages_CreateAgentSessionMessage(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-msg-create",
		Email:    "msg-create@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	run, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		sessionID       string
		runID           string
	}

	testCases := []testCase{
		{
			name:      "create message",
			sessionID: session.Metadata.ID,
			runID:     run.Metadata.ID,
		},
		{
			name:            "create with invalid session ID",
			sessionID:       invalidID,
			runID:           run.Metadata.ID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			msg, err := testClient.client.AgentSessionMessages.CreateAgentSessionMessage(ctx, &models.AgentSessionMessage{
				SessionID: test.sessionID,
				RunID:     test.runID,
				Role:      "user",
				Content:   json.RawMessage(`{"text":"hello"}`),
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, msg)
			assert.Equal(t, "user", msg.Role)
			assert.Equal(t, test.sessionID, msg.SessionID)
			assert.NotEmpty(t, msg.Metadata.ID)
		})
	}
}

func TestAgentSessionMessages_GetAgentSessionMessageByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-msg-get",
		Email:    "msg-get@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	run, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessionMessages.CreateAgentSessionMessage(ctx, &models.AgentSessionMessage{
		SessionID: session.Metadata.ID,
		RunID:     run.Metadata.ID,
		Role:      "assistant",
		Content:   json.RawMessage(`{"text":"response"}`),
	})
	require.Nil(t, err)

	type testCase struct {
		name      string
		id        string
		expectMsg bool
	}

	testCases := []testCase{
		{
			name:      "get message by id",
			id:        created.Metadata.ID,
			expectMsg: true,
		},
		{
			name: "message not found",
			id:   nonExistentID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			msg, err := testClient.client.AgentSessionMessages.GetAgentSessionMessageByID(ctx, session.Metadata.ID, test.id)
			require.Nil(t, err)

			if test.expectMsg {
				require.NotNil(t, msg)
				assert.Equal(t, created.Metadata.ID, msg.Metadata.ID)
			} else {
				assert.Nil(t, msg)
			}
		})
	}
}

func TestAgentSessionMessages_GetAgentSessionMessages(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-msgs-list",
		Email:    "msgs-list@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	run, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	for i := 0; i < 3; i++ {
		_, err := testClient.client.AgentSessionMessages.CreateAgentSessionMessage(ctx, &models.AgentSessionMessage{
			SessionID: session.Metadata.ID,
			RunID:     run.Metadata.ID,
			Role:      "user",
		})
		require.Nil(t, err)
	}

	type testCase struct {
		name        string
		input       *GetAgentSessionMessagesInput
		expectCount int
	}

	testCases := []testCase{
		{
			name: "filter by session ID",
			input: &GetAgentSessionMessagesInput{
				Filter: &AgentSessionMessageFilter{
					SessionID: ptr.String(session.Metadata.ID),
				},
			},
			expectCount: 3,
		},
		{
			name: "filter by run ID",
			input: &GetAgentSessionMessagesInput{
				Filter: &AgentSessionMessageFilter{
					RunID: ptr.String(run.Metadata.ID),
				},
			},
			expectCount: 3,
		},
		{
			name: "filter by non-existent session ID",
			input: &GetAgentSessionMessagesInput{
				Filter: &AgentSessionMessageFilter{
					SessionID: ptr.String(nonExistentID),
				},
			},
			expectCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.AgentSessionMessages.GetAgentSessionMessages(ctx, test.input)
			require.Nil(t, err)
			assert.Len(t, result.AgentSessionMessages, test.expectCount)
		})
	}
}

func TestAgentSessionMessages_PaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-msgs-pagination",
		Email:    "msgs-pagination@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	run, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.AgentSessionMessages.CreateAgentSessionMessage(ctx, &models.AgentSessionMessage{
			SessionID: session.Metadata.ID,
			RunID:     run.Metadata.ID,
			Role:      "user",
			Content:   json.RawMessage(fmt.Sprintf(`{"text":"msg-%d"}`, i)),
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		AgentSessionMessageSortableFieldCreatedAtAsc,
		AgentSessionMessageSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := AgentSessionMessageSortableField(sortByField.getValue())

		result, err := testClient.client.AgentSessionMessages.GetAgentSessionMessages(ctx, &GetAgentSessionMessagesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
			Filter: &AgentSessionMessageFilter{
				SessionID: ptr.String(session.Metadata.ID),
			},
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, r := range result.AgentSessionMessages {
			rCopy := r
			resources = append(resources, &rCopy)
		}

		return result.PageInfo, resources, nil
	})
}
