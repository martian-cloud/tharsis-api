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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for AgentSessionRunSortableField
func (sf AgentSessionRunSortableField) getValue() string {
	return string(sf)
}

func TestAgentSessionRuns_CreateAgentSessionRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-run-create",
		Email:    "run-create@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		sessionID       string
	}

	testCases := []testCase{
		{
			name:      "create agent session run",
			sessionID: session.Metadata.ID,
		},
		{
			name:            "create with invalid session ID",
			sessionID:       invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
				SessionID: test.sessionID,
				Status:    models.AgentSessionRunRunning,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, run)
			assert.Equal(t, test.sessionID, run.SessionID)
			assert.Equal(t, models.AgentSessionRunRunning, run.Status)
			assert.NotEmpty(t, run.Metadata.ID)
		})
	}
}

func TestAgentSessionRuns_GetAgentSessionRunByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-run-get",
		Email:    "run-get@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	type testCase struct {
		name      string
		id        string
		expectRun bool
	}

	testCases := []testCase{
		{
			name:      "get run by id",
			id:        created.Metadata.ID,
			expectRun: true,
		},
		{
			name: "run not found",
			id:   nonExistentID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.AgentSessionRuns.GetAgentSessionRunByID(ctx, test.id)
			require.Nil(t, err)

			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, created.Metadata.ID, run.Metadata.ID)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestAgentSessionRuns_UpdateAgentSessionRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-run-update",
		Email:    "run-update@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		run             *models.AgentSessionRun
		expectStatus    models.AgentSessionRunStatus
	}

	testCases := []testCase{
		{
			name: "update run status",
			run: &models.AgentSessionRun{
				Metadata:  created.Metadata,
				SessionID: created.SessionID,
				Status:    models.AgentSessionRunFinished,
			},
			expectStatus: models.AgentSessionRunFinished,
		},
		{
			name: "update fails with wrong version",
			run: &models.AgentSessionRun{
				Metadata: models.ResourceMetadata{
					ID:      created.Metadata.ID,
					Version: -1,
				},
				Status: models.AgentSessionRunErrored,
			},
			expectErrorCode: errors.EOptimisticLock,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			updated, err := testClient.client.AgentSessionRuns.UpdateAgentSessionRun(ctx, test.run)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, test.expectStatus, updated.Status)
		})
	}
}

func TestAgentSessionRuns_GetAgentSessionRunsBySessionID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-runs-by-session",
		Email:    "runs-by-session@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	// Create 3 runs
	for i := 0; i < 3; i++ {
		_, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
			SessionID: session.Metadata.ID,
			Status:    models.AgentSessionRunRunning,
		})
		require.Nil(t, err)
	}

	sessionID := session.Metadata.ID
	result, err := testClient.client.AgentSessionRuns.GetAgentSessionRuns(ctx, &GetAgentSessionRunsInput{
		Filter: &AgentSessionRunFilter{SessionID: &sessionID},
	})
	require.Nil(t, err)
	assert.Len(t, result.AgentSessionRuns, 3)
}

func TestAgentSessionRuns_GetAgentSessionRuns(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-runs-list",
		Email:    "runs-list@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	for i := 0; i < 3; i++ {
		_, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
			SessionID: session.Metadata.ID,
			Status:    models.AgentSessionRunRunning,
		})
		require.Nil(t, err)
	}

	type testCase struct {
		name        string
		input       *GetAgentSessionRunsInput
		expectCount int
	}

	testCases := []testCase{
		{
			name: "filter by session ID",
			input: &GetAgentSessionRunsInput{
				Filter: &AgentSessionRunFilter{
					SessionID: ptr.String(session.Metadata.ID),
				},
			},
			expectCount: 3,
		},
		{
			name: "filter by non-existent session ID",
			input: &GetAgentSessionRunsInput{
				Filter: &AgentSessionRunFilter{
					SessionID: ptr.String(nonExistentID),
				},
			},
			expectCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.AgentSessionRuns.GetAgentSessionRuns(ctx, test.input)
			require.Nil(t, err)
			assert.Len(t, result.AgentSessionRuns, test.expectCount)
		})
	}
}

func TestAgentSessionRuns_GetAgentSessionRunByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-run-trn",
		Email:    "run-trn@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	created, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
		SessionID: session.Metadata.ID,
		Status:    models.AgentSessionRunRunning,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		trn             string
		expectRun       bool
	}

	testCases := []testCase{
		{
			name:      "get run by TRN",
			trn:       types.AgentSessionRunModelType.BuildTRN(session.GetGlobalID(), created.GetGlobalID()),
			expectRun: true,
		},
		{
			name:            "invalid TRN",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.AgentSessionRuns.GetAgentSessionRunByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, created.Metadata.ID, run.Metadata.ID)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestAgentSessionRuns_PaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-runs-pagination",
		Email:    "runs-pagination@example.com",
	})
	require.Nil(t, err)

	session, err := testClient.client.AgentSessions.CreateAgentSession(ctx, &models.AgentSession{
		UserID: user.Metadata.ID,
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.AgentSessionRuns.CreateAgentSessionRun(ctx, &models.AgentSessionRun{
			SessionID:    session.Metadata.ID,
			Status:       models.AgentSessionRunRunning,
			ErrorMessage: ptr.String(fmt.Sprintf("run-%d", i)),
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		AgentSessionRunSortableFieldCreatedAtAsc,
		AgentSessionRunSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := AgentSessionRunSortableField(sortByField.getValue())

		result, err := testClient.client.AgentSessionRuns.GetAgentSessionRuns(ctx, &GetAgentSessionRunsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
			Filter: &AgentSessionRunFilter{
				SessionID: ptr.String(session.Metadata.ID),
			},
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, r := range result.AgentSessionRuns {
			rCopy := r
			resources = append(resources, &rCopy)
		}

		return result.PageInfo, resources, nil
	})
}
