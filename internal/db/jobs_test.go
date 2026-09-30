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

// jobWithStatus sets the initial status on a freshly constructed test job. It is
// valid from the job's zero value, so the error is intentionally ignored.
func jobWithStatus(j *models.Job, status models.JobStatus) *models.Job {
	_ = j.SetStatus(status)
	return j
}

// getValue implements the sortableField interface for JobSortableField
func (js JobSortableField) getValue() string {
	return string(js)
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
			job, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
				WorkspaceID: workspace.Metadata.ID,
				RunID:       test.runID,
				Type:        test.jobType,
			}, test.status))

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, job)

			assert.Equal(t, test.runID, job.RunID)
			assert.Equal(t, test.jobType, job.Type)
			assert.Equal(t, test.status, job.GetStatus())
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

	createdJob, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
	}, models.JobQueued))
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
			// queued -> pending is the valid next transition (queued -> running is not).
			status: models.JobPending,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.JobCanceled,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			jobToUpdate := *createdJob
			jobToUpdate.Metadata.Version = test.version
			_ = jobToUpdate.SetStatus(test.status)

			updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedJob)

			assert.Equal(t, test.status, updatedJob.GetStatus())
			assert.Equal(t, createdJob.Metadata.Version+1, updatedJob.Metadata.Version)
		})
	}
}

func TestJobs_UpdateJob_ResourceUsageMetrics(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-job-metrics",
		Description: "test group for job resource metrics",
		FullPath:    "test-group-job-metrics",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-metrics",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.Nil(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdJob, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
	}, models.JobQueued))
	require.Nil(t, err)

	t.Run("nil resource metrics round-trips as nil", func(t *testing.T) {
		jobToUpdate := *createdJob
		_ = jobToUpdate.SetStatus(models.JobPending)
		// ResourceMetrics is nil by default — verify it stays nil after update.
		updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)
		require.Nil(t, err)
		assert.Nil(t, updatedJob.ResourceUsageMetrics)
		createdJob = updatedJob
	})

	t.Run("resource metrics persist and round-trip correctly", func(t *testing.T) {
		peakMem := int64(512_000_000)
		cpuMS := float64(4_200)
		netRecvBytes := int64(1_048_576)
		netSentBytes := int64(524_288)
		netRecvPkts := int64(1_024)
		netSentPkts := int64(512)
		diskR := int64(209_715_200)
		diskW := int64(104_857_600)

		jobToUpdate := *createdJob
		_ = jobToUpdate.SetStatus(models.JobRunning)
		jobToUpdate.ResourceUsageMetrics = &models.JobResourceUsageMetrics{
			PeakMemoryBytes:             &peakMem,
			TotalCPUTimeMS:              &cpuMS,
			TotalNetworkReceivedBytes:   &netRecvBytes,
			TotalNetworkSentBytes:       &netSentBytes,
			TotalNetworkReceivedPackets: &netRecvPkts,
			TotalNetworkSentPackets:     &netSentPkts,
			TotalDiskReadBytes:          &diskR,
			TotalDiskWriteBytes:         &diskW,
		}

		updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)
		require.Nil(t, err)
		require.NotNil(t, updatedJob.ResourceUsageMetrics)

		assert.Equal(t, peakMem, *updatedJob.ResourceUsageMetrics.PeakMemoryBytes)
		assert.Equal(t, cpuMS, *updatedJob.ResourceUsageMetrics.TotalCPUTimeMS)
		assert.Equal(t, netRecvBytes, *updatedJob.ResourceUsageMetrics.TotalNetworkReceivedBytes)
		assert.Equal(t, netSentBytes, *updatedJob.ResourceUsageMetrics.TotalNetworkSentBytes)
		assert.Equal(t, netRecvPkts, *updatedJob.ResourceUsageMetrics.TotalNetworkReceivedPackets)
		assert.Equal(t, netSentPkts, *updatedJob.ResourceUsageMetrics.TotalNetworkSentPackets)
		assert.Equal(t, diskR, *updatedJob.ResourceUsageMetrics.TotalDiskReadBytes)
		assert.Equal(t, diskW, *updatedJob.ResourceUsageMetrics.TotalDiskWriteBytes)
		createdJob = updatedJob
	})

	t.Run("resource metrics can be cleared back to nil", func(t *testing.T) {
		jobToUpdate := *createdJob
		jobToUpdate.ResourceUsageMetrics = nil

		updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)
		require.Nil(t, err)
		assert.Nil(t, updatedJob.ResourceUsageMetrics)
		createdJob = updatedJob
	})

	t.Run("resource usage limits persist and round-trip correctly", func(t *testing.T) {
		memLimit := int64(1_610_612_736)
		netRecvLimit := int64(2_097_152)
		netSentLimit := int64(1_048_576)
		diskRLimit := int64(419_430_400)
		diskWLimit := int64(209_715_200)

		jobToUpdate := *createdJob
		jobToUpdate.ResourceUsageLimits = &models.JobResourceUsageLimits{
			MemoryBytes:          &memLimit,
			NetworkReceivedBytes: &netRecvLimit,
			NetworkSentBytes:     &netSentLimit,
			DiskReadBytes:        &diskRLimit,
			DiskWriteBytes:       &diskWLimit,
		}

		updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)
		require.Nil(t, err)
		require.NotNil(t, updatedJob.ResourceUsageLimits)

		assert.Equal(t, memLimit, *updatedJob.ResourceUsageLimits.MemoryBytes)
		assert.Equal(t, netRecvLimit, *updatedJob.ResourceUsageLimits.NetworkReceivedBytes)
		assert.Equal(t, netSentLimit, *updatedJob.ResourceUsageLimits.NetworkSentBytes)
		assert.Equal(t, diskRLimit, *updatedJob.ResourceUsageLimits.DiskReadBytes)
		assert.Equal(t, diskWLimit, *updatedJob.ResourceUsageLimits.DiskWriteBytes)
		createdJob = updatedJob
	})

	t.Run("resource usage limits can be cleared back to nil", func(t *testing.T) {
		jobToUpdate := *createdJob
		jobToUpdate.ResourceUsageLimits = nil

		updatedJob, err := testClient.client.Jobs.UpdateJob(ctx, &jobToUpdate)
		require.Nil(t, err)
		assert.Nil(t, updatedJob.ResourceUsageLimits)
	})
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
	createdJob, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
	}, models.JobQueued))
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
		*jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
		}, models.JobQueued),
		*jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobApplyType,
		}, models.JobRunning),
	}

	createdJobs := []models.Job{}
	for _, job := range jobs {
		created, err := testClient.client.Jobs.CreateJob(ctx, &job)
		require.NoError(t, err)
		createdJobs = append(createdJobs, *created)
	}

	firstJobStatus := createdJobs[0].GetStatus()

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
					JobStatus: &firstJobStatus,
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

func TestJobs_GetJobs_TagFilter(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-jobs-tag-filter",
		FullPath:  "test-group-jobs-tag-filter",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-jobs-tag-filter",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Two tagged jobs plus one untagged job. The TagSuperset filter matches jobs
	// whose tags are a subset of the superset, so the untagged job (empty set)
	// matches any superset unless explicitly excluded.
	jobs := []models.Job{
		*jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			Tags:        []string{"linux", "arm64"},
		}, models.JobQueued),
		*jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobApplyType,
			Tags:        []string{"linux"},
		}, models.JobQueued),
		*jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			Tags:        []string{},
		}, models.JobQueued),
	}

	for i := range jobs {
		_, err := testClient.client.Jobs.CreateJob(ctx, &jobs[i])
		require.NoError(t, err)
	}

	type testCase struct {
		name        string
		input       *GetJobsInput
		expectCount int
	}

	testCases := []testCase{
		{
			name: "superset containing all job tags returns every job",
			input: &GetJobsInput{
				Filter: &JobFilter{
					WorkspaceID: &workspace.Metadata.ID,
					TagFilter: &JobTagFilter{
						TagSuperset: []string{"linux", "arm64"},
					},
				},
			},
			expectCount: 3,
		},
		{
			name: "narrower superset only matches jobs whose tags are a subset",
			input: &GetJobsInput{
				Filter: &JobFilter{
					WorkspaceID: &workspace.Metadata.ID,
					TagFilter: &JobTagFilter{
						TagSuperset: []string{"linux"},
					},
				},
			},
			// "linux" job and the untagged job, but not the "linux"+"arm64" job.
			expectCount: 2,
		},
		{
			name: "excluding untagged jobs drops the empty-tag job",
			input: &GetJobsInput{
				Filter: &JobFilter{
					WorkspaceID: &workspace.Metadata.ID,
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(true),
						TagSuperset:         []string{"linux"},
					},
				},
			},
			expectCount: 1,
		},
		{
			// A single-quote-laden value must be bound as data, not interpolated
			// into the SQL string. The query should run cleanly and simply match
			// nothing rather than erroring or executing injected SQL.
			name: "malicious tag value is treated as a literal",
			input: &GetJobsInput{
				Filter: &JobFilter{
					WorkspaceID: &workspace.Metadata.ID,
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(true),
						TagSuperset:         []string{"x'); DROP TABLE jobs; --"},
					},
				},
			},
			expectCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Jobs.GetJobs(ctx, test.input)
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
		_, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
		}, models.JobQueued))
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
	createdJob, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
	}, models.JobQueued))
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
	_, err = testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
	}, models.JobQueued))
	require.NoError(t, err)

	job2, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
	}, models.JobRunning))
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

func TestJobs_ClaimJobsForCleanup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-job-cleanup",
		FullPath:  "test-group-job-cleanup",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-cleanup",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-cleanup",
		Type: models.SharedRunnerType,
	})
	require.NoError(t, err)

	otherRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-cleanup-other",
		Type: models.SharedRunnerType,
	})
	require.NoError(t, err)

	// advanceToFinal walks a job through valid status transitions to reach a final status, since
	// SetStatus rejects illegal jumps (e.g. queued -> finished).
	advanceToFinal := func(job *models.Job, final models.JobStatus) {
		path := []models.JobStatus{models.JobPending, models.JobRunning, final}
		if final == models.JobCanceled {
			path = []models.JobStatus{models.JobCanceled}
		}
		for _, s := range path {
			require.NoError(t, job.SetStatus(s))
		}
	}

	// createCleanableJob builds a job then advances it to a final status with dispatcher metadata set,
	// which is the state the cleanup poller claims.
	createCleanableJob := func(runnerID string, status models.JobStatus) *models.Job {
		job, cErr := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			RunnerID:    &runnerID,
		}, models.JobQueued))
		require.NoError(t, cErr)

		advanceToFinal(job, status)
		job.DispatcherData = map[string]string{"resourceName": "pod-" + job.Metadata.ID}
		updated, uErr := testClient.client.Jobs.UpdateJob(ctx, job)
		require.NoError(t, uErr)
		return updated
	}

	// Eligible: final + dispatched + owned by runner.
	eligible := createCleanableJob(runner.Metadata.ID, models.JobFinished)

	// Ineligible variants that must never be claimed for this runner.
	createCleanableJob(otherRunner.Metadata.ID, models.JobFinished) // different runner

	nonFinal, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		RunnerID:    &runner.Metadata.ID,
	}, models.JobQueued))
	require.NoError(t, err)
	require.NoError(t, nonFinal.SetStatus(models.JobPending))
	require.NoError(t, nonFinal.SetStatus(models.JobRunning))
	nonFinal.DispatcherData = map[string]string{"resourceName": "pod-running"}
	_, err = testClient.client.Jobs.UpdateJob(ctx, nonFinal)
	require.NoError(t, err)

	// Final but no dispatcher metadata (never dispatched to an external runtime).
	noMetadata, err := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
		Type:        models.JobPlanType,
		RunnerID:    &runner.Metadata.ID,
	}, models.JobQueued))
	require.NoError(t, err)
	require.NoError(t, noMetadata.SetStatus(models.JobCanceled))
	_, err = testClient.client.Jobs.UpdateJob(ctx, noMetadata)
	require.NoError(t, err)

	t.Run("claims only the eligible job for the runner and leases it", func(t *testing.T) {
		claimed, cErr := testClient.client.Jobs.ClaimJobsForCleanup(ctx, &ClaimJobsForCleanupInput{
			RunnerID: runner.Metadata.ID,
			Limit:    10,
		})
		require.NoError(t, cErr)
		require.Len(t, claimed, 1)
		assert.Equal(t, eligible.Metadata.ID, claimed[0].Metadata.ID)
		assert.Equal(t, map[string]string{"resourceName": "pod-" + eligible.Metadata.ID}, claimed[0].DispatcherData)
	})

	t.Run("a leased job is not re-claimed within the lease window", func(t *testing.T) {
		claimed, cErr := testClient.client.Jobs.ClaimJobsForCleanup(ctx, &ClaimJobsForCleanupInput{
			RunnerID: runner.Metadata.ID,
			Limit:    10,
		})
		require.NoError(t, cErr)
		assert.Empty(t, claimed)
	})

	t.Run("a limit of zero claims nothing", func(t *testing.T) {
		claimed, cErr := testClient.client.Jobs.ClaimJobsForCleanup(ctx, &ClaimJobsForCleanupInput{
			RunnerID: runner.Metadata.ID,
			Limit:    0,
		})
		require.NoError(t, cErr)
		assert.Empty(t, claimed)
	})
}

func TestJobs_MarkJobsCleanedUp(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-job-mark-clean",
		FullPath:  "test-group-job-mark-clean",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-job-mark-clean",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-mark-clean",
		Type: models.SharedRunnerType,
	})
	require.NoError(t, err)

	otherRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name: "test-runner-mark-clean-other",
		Type: models.SharedRunnerType,
	})
	require.NoError(t, err)

	createFinalJob := func(runnerID string) *models.Job {
		job, cErr := testClient.client.Jobs.CreateJob(ctx, jobWithStatus(&models.Job{
			WorkspaceID: workspace.Metadata.ID,
			RunID:       run.Metadata.ID,
			Type:        models.JobPlanType,
			RunnerID:    &runnerID,
		}, models.JobQueued))
		require.NoError(t, cErr)
		require.NoError(t, job.SetStatus(models.JobPending))
		require.NoError(t, job.SetStatus(models.JobRunning))
		require.NoError(t, job.SetStatus(models.JobFinished))
		job.DispatcherData = map[string]string{"resourceName": "pod-" + job.Metadata.ID}
		updated, uErr := testClient.client.Jobs.UpdateJob(ctx, job)
		require.NoError(t, uErr)
		return updated
	}

	owned := createFinalJob(runner.Metadata.ID)
	otherOwned := createFinalJob(otherRunner.Metadata.ID)

	t.Run("marks the runner's jobs cleaned up and leaves other runners' jobs untouched", func(t *testing.T) {
		err := testClient.client.Jobs.MarkJobsCleanedUp(ctx, runner.Metadata.ID, []string{owned.Metadata.ID, otherOwned.Metadata.ID})
		require.NoError(t, err)

		gotOwned, gErr := testClient.client.Jobs.GetJobByID(ctx, owned.Metadata.ID)
		require.NoError(t, gErr)
		assert.NotNil(t, gotOwned.CleanupCompletedAt)

		// otherOwned belongs to a different runner, so the runner_id predicate must exclude it.
		gotOther, gErr := testClient.client.Jobs.GetJobByID(ctx, otherOwned.Metadata.ID)
		require.NoError(t, gErr)
		assert.Nil(t, gotOther.CleanupCompletedAt)
	})

	t.Run("an empty job list is a no-op", func(t *testing.T) {
		err := testClient.client.Jobs.MarkJobsCleanedUp(ctx, runner.Metadata.ID, nil)
		require.NoError(t, err)
	})
}
