//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for LogStreamSortableField
func (ls LogStreamSortableField) getValue() string {
	return string(ls)
}

func TestLogStreams_CreateLogStream(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-log-stream",
		Description: "test group for log stream",
		FullPath:    "test-group-log-stream",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-log-stream",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for log stream",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		RunID:       run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		jobID           string
		size            int
	}

	testCases := []testCase{
		{
			name:  "create log stream",
			jobID: job.Metadata.ID,
			size:  1024,
		},
		{
			name:            "create log stream with invalid job ID",
			jobID:           invalidID,
			size:            1024,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStream, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{
				JobID: &test.jobID,
				Size:  test.size,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, logStream)

			assert.Equal(t, test.jobID, *logStream.JobID)
			assert.Equal(t, test.size, logStream.Size)
			assert.NotEmpty(t, logStream.Metadata.ID)
		})
	}
}

func TestLogStreams_UpdateLogStream(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-log-stream-update",
		Description: "test group for log stream update",
		FullPath:    "test-group-log-stream-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-log-stream-update",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for log stream update",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		RunID:       run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.Nil(t, err)

	createdLogStream, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{
		JobID: &job.Metadata.ID,
		Size:  1024,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		size            int
	}

	testCases := []testCase{
		{
			name:    "update log stream",
			version: createdLogStream.Metadata.Version,
			size:    2048,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			size:            4096,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStreamToUpdate := *createdLogStream
			logStreamToUpdate.Metadata.Version = test.version
			logStreamToUpdate.Size = test.size

			updatedLogStream, err := testClient.client.LogStreams.UpdateLogStream(ctx, &logStreamToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedLogStream)

			assert.Equal(t, test.size, updatedLogStream.Size)
			assert.Equal(t, createdLogStream.Metadata.Version+1, updatedLogStream.Metadata.Version)
		})
	}
}

func TestLogStreams_GetLogStreamByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-log-stream-get-by-id",
		Description: "test group for log stream get by id",
		FullPath:    "test-group-log-stream-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the run
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-log-stream-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for the job
	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a job for the log stream
	job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	// Create a log stream for testing
	createdLogStream, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{
		JobID: &job.Metadata.ID,
		Size:  0,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectLogStream bool
	}

	testCases := []testCase{
		{
			name:            "get resource by id",
			id:              createdLogStream.Metadata.ID,
			expectLogStream: true,
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
			logStream, err := testClient.client.LogStreams.GetLogStreamByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLogStream {
				require.NotNil(t, logStream)
				assert.Equal(t, test.id, logStream.Metadata.ID)
			} else {
				assert.Nil(t, logStream)
			}
		})
	}
}

func TestLogStreams_GetLogStreams(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-log-streams-list",
		Description: "test group for log streams list",
		FullPath:    "test-group-log-streams-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the runs
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-log-streams-list",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for the jobs
	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create jobs for the log streams
	job1, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	job2, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobApplyType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	// Create test log streams
	logStreams := []models.LogStream{
		{
			JobID: &job1.Metadata.ID,
			Size:  100,
		},
		{
			JobID: &job2.Metadata.ID,
			Size:  200,
		},
	}

	createdLogStreams := []models.LogStream{}
	for _, logStream := range logStreams {
		created, err := testClient.client.LogStreams.CreateLogStream(ctx, &logStream)
		require.NoError(t, err)
		createdLogStreams = append(createdLogStreams, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetLogStreamsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all log streams",
			input:       &GetLogStreamsInput{},
			expectCount: len(createdLogStreams),
		},
		{
			name: "filter by job IDs",
			input: &GetLogStreamsInput{
				Filter: &LogStreamFilter{
					JobIDs: []string{job1.Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.LogStreams.GetLogStreams(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.LogStreams, test.expectCount)
		})
	}
}

func TestLogStreams_GetLogStreamsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-log-streams-pagination",
		Description: "test group for log streams pagination",
		FullPath:    "test-group-log-streams-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the runs
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-log-streams-pagination",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for the jobs
	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			Status:      models.JobQueued,
		})
		require.NoError(t, err)

		_, err = testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{
			JobID: &job.Metadata.ID,
			Size:  100,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		LogStreamSortableFieldUpdatedAtAsc,
		LogStreamSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := LogStreamSortableField(sortByField.getValue())

		result, err := testClient.client.LogStreams.GetLogStreams(ctx, &GetLogStreamsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.LogStreams {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestLogStreams_GetLogStreamByJobID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-log-stream-job-id",
		Description: "test group for log stream job id",
		FullPath:    "test-group-log-stream-job-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the run
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-log-stream-job-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for the job
	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a job for the log stream
	job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	// Create a log stream for testing
	createdLogStream, err := testClient.client.LogStreams.CreateLogStream(ctx, &models.LogStream{
		JobID: &job.Metadata.ID,
		Size:  100,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		jobID           string
		expectLogStream bool
	}

	testCases := []testCase{
		{
			name:            "get resource by job ID",
			jobID:           job.Metadata.ID,
			expectLogStream: true,
		},
		{
			name:  "resource with job ID not found",
			jobID: nonExistentID,
		},
		{
			name:            "get resource with invalid job ID will return an error",
			jobID:           invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStream, err := testClient.client.LogStreams.GetLogStreamByJobID(ctx, test.jobID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLogStream {
				require.NotNil(t, logStream)
				assert.Equal(t, createdLogStream.Metadata.ID, logStream.Metadata.ID)
				assert.Equal(t, &test.jobID, logStream.JobID)
			} else {
				assert.Nil(t, logStream)
			}
		})
	}
}

func TestLogStreams_GetLogStreamByRunnerSessionID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name        string
		sessionID   string
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "resource with session ID not found - returns nil",
			sessionID:   "11111111-2222-3333-4444-555555555555",
			expectError: false,
		},
		{
			name:        "get resource with invalid session ID will return an error",
			sessionID:   invalidID,
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStream, err := testClient.client.LogStreams.GetLogStreamByRunnerSessionID(ctx, test.sessionID)

			if test.expectError {
				require.Error(t, err)
				assert.Nil(t, logStream)
			} else {
				require.NoError(t, err)
				assert.Nil(t, logStream) // Should be nil for non-existent session
			}
		})
	}
}
