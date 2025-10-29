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

// getValue implements the sortableField interface for JobSortableField
func (j JobSortableField) getValue() string {
	return string(j)
}

func TestJobs_CreateJob(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-job",
		Description: "test group for job",
		FullPath:    "test-group-job",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for job",
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

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		runID           string
		jobType         models.JobType
		status          models.JobStatus
	}

	testCases := []testCase{
		{
			name:    "create job",
			runID:   run.Metadata.ID,
			jobType: models.JobPlanType,
			status:  models.JobQueued,
		},
		{
			name:            "create job with invalid run ID",
			runID:           invalidID,
			jobType:         models.JobPlanType,
			status:          models.JobQueued,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
				WorkspaceID: workspace.Metadata.ID,
				RunID:       test.runID,
				Type:        test.jobType,
				Status:      test.status,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, job)

			assert.Equal(t, test.runID, job.RunID)
			assert.Equal(t, test.jobType, job.Type)
			assert.Equal(t, test.status, job.Status)
			assert.NotEmpty(t, job.Metadata.ID)
		})
	}
}

func TestJobs_UpdateJob(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create dependencies for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-job-update",
		Description: "test group for job update",
		FullPath:    "test-group-job-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-update",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for job update",
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

	createdJob, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.JobStatus
	}

	testCases := []testCase{
		{
			name:    "update job",
			version: createdJob.Metadata.Version,
			status:  models.JobRunning,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.JobFinished,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			jobToUpdate := *createdJob
			jobToUpdate.Metadata.Version = test.version
			jobToUpdate.Status = test.status

			updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedJob)

			assert.Equal(t, test.status, updatedJob.Status)
			assert.Equal(t, createdJob.Metadata.Version+1, updatedJob.Metadata.Version)
		})
	}
}

func TestJobs_GetJobByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-job-get-by-id",
		Description: "test group for job get by id",
		FullPath:    "test-group-job-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the run
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-get-by-id",
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

	// Create a job for testing
	createdJob, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectJob       bool
	}

	testCases := []testCase{
		{
			name:      "get resource by id",
			id:        createdJob.Metadata.ID,
			expectJob: true,
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
			job, err := testClient.client.Jobs.GetJobByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectJob {
				require.NotNil(t, job)
				assert.Equal(t, test.id, job.Metadata.ID)
			} else {
				assert.Nil(t, job)
			}
		})
	}
}

func TestJobs_GetJobs(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-jobs-list",
		Description: "test group for jobs list",
		FullPath:    "test-group-jobs-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the runs
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-jobs-list",
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

	// Create test jobs
	jobs := []models.Job{
		{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			Status:      models.JobQueued,
		},
		{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobApplyType,
			Status:      models.JobRunning,
		},
	}

	createdJobs := []models.Job{}
	for _, job := range jobs {
		created, err := testClient.client.Jobs.CreateJob(ctx, &job)
		require.NoError(t, err)
		createdJobs = append(createdJobs, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetJobsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all jobs",
			input:       &GetJobsInput{},
			expectCount: len(createdJobs),
		},
		{
			name: "filter by run ID",
			input: &GetJobsInput{
				Filter: &JobFilter{
					RunID: &run.Metadata.ID,
				},
			},
			expectCount: len(createdJobs),
		},
		{
			name: "filter by workspace ID",
			input: &GetJobsInput{
				Filter: &JobFilter{
					WorkspaceID: &workspace.Metadata.ID,
				},
			},
			expectCount: len(createdJobs),
		},
		{
			name: "filter by job type",
			input: &GetJobsInput{
				Filter: &JobFilter{
					JobType: &createdJobs[0].Type,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by job status",
			input: &GetJobsInput{
				Filter: &JobFilter{
					JobStatus: &createdJobs[0].Status,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by job IDs",
			input: &GetJobsInput{
				Filter: &JobFilter{
					JobIDs: []string{createdJobs[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Jobs.GetJobs(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Jobs, test.expectCount)
		})
	}
}

func TestJobs_GetJobsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-jobs-pagination",
		Description: "test group for jobs pagination",
		FullPath:    "test-group-jobs-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the runs
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-jobs-pagination",
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
		_, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			Status:      models.JobQueued,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		JobSortableFieldCreatedAtAsc,
		JobSortableFieldCreatedAtDesc,
		JobSortableFieldUpdatedAtAsc,
		JobSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := JobSortableField(sortByField.getValue())

		result, err := testClient.client.Jobs.GetJobs(ctx, &GetJobsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Jobs {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestJobs_GetJobByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-job-trn",
		Description: "test group for job trn",
		FullPath:    "test-group-job-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the run
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-trn",
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

	// Create a job for testing
	createdJob, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectJob       bool
	}

	testCases := []testCase{
		{
			name:      "get resource by TRN",
			trn:       createdJob.Metadata.TRN,
			expectJob: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:job:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			job, err := testClient.client.Jobs.GetJobByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectJob {
				require.NotNil(t, job)
				assert.Equal(t, test.trn, job.Metadata.TRN)
			} else {
				assert.Nil(t, job)
			}
		})
	}
}

func TestJobs_GetLatestJobByType(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-job-latest",
		Description: "test group for job latest",
		FullPath:    "test-group-job-latest",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for the run
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-latest",
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

	// Create multiple jobs of the same type
	_, err = testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobQueued,
	})
	require.NoError(t, err)

	job2, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		Status:      models.JobRunning,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		runID           string
		jobType         models.JobType
		expectJob       bool
		expectedJobID   string
	}

	testCases := []testCase{
		{
			name:          "get latest job by type",
			runID:         run.Metadata.ID,
			jobType:       models.JobPlanType,
			expectJob:     true,
			expectedJobID: job2.Metadata.ID, // job2 was created later
		},
		{
			name:    "no job found for type",
			runID:   run.Metadata.ID,
			jobType: models.JobApplyType,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			job, err := testClient.client.Jobs.GetLatestJobByType(ctx, test.runID, test.jobType)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectJob {
				require.NotNil(t, job)
				assert.Equal(t, test.expectedJobID, job.Metadata.ID)
				assert.Equal(t, test.jobType, job.Type)
			} else {
				assert.Nil(t, job)
			}
		})
	}
}
