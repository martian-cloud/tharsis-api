//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestGetRunnerSessionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner",
	})
	require.Nil(t, err)

	runnerSession, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
		RunnerID: runner.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode     errors.CodeType
		name                string
		id                  string
		expectRunnerSession bool
	}

	testCases := []testCase{
		{
			name:                "get resource by id",
			id:                  runnerSession.Metadata.ID,
			expectRunnerSession: true,
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
			runnerSession, err := testClient.client.RunnerSessions.GetRunnerSessionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRunnerSession {
				require.NotNil(t, runnerSession)
				assert.Equal(t, test.id, runnerSession.Metadata.ID)
			} else {
				assert.Nil(t, runnerSession)
			}
		})
	}
}

func TestGetRunnerSessionByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	groupRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:    "test-group-runner",
		GroupID: &group.Metadata.ID,
		Type:    models.GroupRunnerType,
	})
	require.NoError(t, err)

	groupRunnerSession, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
		RunnerID: groupRunner.Metadata.ID,
	})
	require.NoError(t, err)

	sharedRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner",
		Type: models.SharedRunnerType,
	})
	require.NoError(t, err)

	sharedRunnerSession, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
		RunnerID: sharedRunner.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectSession   bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:          "get shared runner session by TRN",
			trn:           sharedRunnerSession.Metadata.TRN,
			expectSession: true,
		},
		{
			name:          "get group runner session by TRN",
			trn:           groupRunnerSession.Metadata.TRN,
			expectSession: true,
		},
		{
			name: "shared runner session with TRN not found",
			trn:  types.RunnerSessionModelType.BuildTRN(sharedRunner.Name, nonExistentGlobalID),
		},
		{
			name: "group runner session with TRN not found",
			trn:  types.RunnerSessionModelType.BuildTRN(group.FullPath, groupRunner.Name, nonExistentGlobalID),
		},
		{
			name:            "runner session TRN cannot have less than two parts",
			trn:             types.RunnerSessionModelType.BuildTRN(nonExistentGlobalID),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualSession, err := testClient.client.RunnerSessions.GetRunnerSessionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectSession {
				require.NotNil(t, actualSession)
				if test.trn == sharedRunnerSession.Metadata.TRN {
					assert.Equal(t, types.RunnerSessionModelType.BuildTRN(sharedRunner.Name, sharedRunnerSession.GetGlobalID()), actualSession.Metadata.TRN)
				} else {
					assert.Equal(t, types.RunnerSessionModelType.BuildTRN(group.FullPath, groupRunner.Name, groupRunnerSession.GetGlobalID()), actualSession.Metadata.TRN)
				}
			} else {
				assert.Nil(t, actualSession)
			}
		})
	}
}

func TestCreateRunnerSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		runnerID        string
	}

	testCases := []testCase{
		{
			name:     "successfully create resource",
			runnerID: runner.Metadata.ID,
		},
		{
			name:            "create will fail because runner does not exist",
			runnerID:        nonExistentID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			runnerSession, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
				RunnerID: test.runnerID,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, runnerSession)
		})
	}
}

func TestUpdateRunnerSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner",
	})
	require.Nil(t, err)

	runnerSession, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
		RunnerID: runner.Metadata.ID,
	})
	require.Nil(t, err)

	currentTime := time.Now().UTC()

	type testCase struct {
		name                 string
		expectErrorCode      errors.CodeType
		lastContactTimestamp time.Time
		version              int
	}

	testCases := []testCase{
		{
			name:                 "successfully update resource",
			lastContactTimestamp: currentTime,
			version:              1,
		},
		{
			name:                 "update will fail because resource version doesn't match",
			lastContactTimestamp: currentTime,
			expectErrorCode:      errors.EOptimisticLock,
			version:              -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualRunnerSession, err := testClient.client.RunnerSessions.UpdateRunnerSession(ctx, &models.RunnerSession{
				Metadata: models.ResourceMetadata{
					ID:      runnerSession.Metadata.ID,
					Version: test.version,
				},
				LastContactTimestamp: test.lastContactTimestamp,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, actualRunnerSession)
			assert.Equal(t, test.lastContactTimestamp.Format(time.RFC3339), actualRunnerSession.LastContactTimestamp.Format(time.RFC3339))
		})
	}
}

func TestDeleteRunnerSession(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner",
	})
	require.Nil(t, err)

	runnerSession, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
		RunnerID: runner.Metadata.ID,
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
			id:              runnerSession.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      runnerSession.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.RunnerSessions.DeleteRunnerSession(ctx, &models.RunnerSession{
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

func TestGetRunnerSessions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-1",
	})
	require.Nil(t, err)

	runner2, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-2",
	})
	require.Nil(t, err)

	sessions := make([]*models.RunnerSession, 10)
	for i := 0; i < len(sessions); i++ {
		session, aErr := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
			RunnerID: runner1.Metadata.ID,
		})
		require.Nil(t, aErr)

		sessions[i] = session
	}

	session, err := testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
		RunnerID: runner2.Metadata.ID,
	})
	require.Nil(t, err)

	sessions = append(sessions, session)

	type testCase struct {
		filter            *RunnerSessionFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name: "return all sessions for runner 1",
			filter: &RunnerSessionFilter{
				RunnerID: &runner1.Metadata.ID,
			},
			expectResultCount: len(sessions) - 1,
		},
		{
			name: "return all sessions for runner 2",
			filter: &RunnerSessionFilter{
				RunnerID: &runner2.Metadata.ID,
			},
			expectResultCount: 1,
		},
		{
			name: "return all sessions matching specific IDs",
			filter: &RunnerSessionFilter{
				RunnerSessionIDs: []string{sessions[0].Metadata.ID, sessions[1].Metadata.ID},
			},
			expectResultCount: 2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.RunnerSessions.GetRunnerSessions(ctx, &GetRunnerSessionsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			assert.Equal(t, test.expectResultCount, len(result.RunnerSessions))
		})
	}
}

func TestGetRunnerSessionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-1",
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err = testClient.client.RunnerSessions.CreateRunnerSession(ctx, &models.RunnerSession{
			RunnerID: runner1.Metadata.ID,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		RunnerSessionSortableFieldCreatedAtAsc,
		RunnerSessionSortableFieldCreatedAtDesc,
		RunnerSessionSortableFieldLastContactedAtAsc,
		RunnerSessionSortableFieldLastContactedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := RunnerSessionSortableField(sortByField.getValue())

		result, err := testClient.client.RunnerSessions.GetRunnerSessions(ctx, &GetRunnerSessionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.RunnerSessions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
