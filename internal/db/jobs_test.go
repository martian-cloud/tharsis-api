//go:build integration

package db

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// jobInfo aids convenience in accessing the information TestGetJobs needs about the warmup jobs.
type jobInfo struct {
	createTime time.Time
	updateTime time.Time
	jobID      string
}

// jobInfoIDSlice makes a slice of jobInfo sortable by ID string
type jobInfoIDSlice []jobInfo

// jobInfoCreateSlice makes a slice of jobInfo sortable by creation time
type jobInfoCreateSlice []jobInfo

// jobInfoUpdateSlice makes a slice of jobInfo sortable by last updated time
type jobInfoUpdateSlice []jobInfo

func TestGetJobByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a job with a specific ID without going into the really
	// low-level stuff, create the warmup job(s) then find the relevant ID.
	createdLow := currentTime()
	_, _, _, createdWarmupJobs, err := createWarmupJobs(ctx, testClient,
		standardWarmupGroupsForJobs, standardWarmupWorkspacesForJobs,
		standardWarmupRunsForJobs, standardWarmupRunnersForJobs,
		standardWarmupJobs)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectJob *models.Job
		expectMsg *string
		name      string
		searchID  string
	}

	// Do only one positive test case, because the logic is theoretically the same for all jobs.
	positiveJob := createdWarmupJobs[0]
	testCases := []testCase{
		{
			name:      "positive",
			searchID:  positiveJob.Metadata.ID,
			expectJob: &positiveJob,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect job and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			job, err := testClient.client.Jobs.GetJobByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectJob != nil {
				require.NotNil(t, job)
				compareJobs(t, test.expectJob, job, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, job)
			}
		})
	}
}

func TestGetJobByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(20),
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.NoError(t, err)

	job, err := testClient.client.Jobs.CreateJob(ctx, &models.Job{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       run.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectJob       bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:      "get resource by TRN",
			trn:       job.Metadata.TRN,
			expectJob: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.JobModelType.BuildTRN(workspace.FullPath, nonExistentGlobalID),
		},
		{
			name:            "job trn has less than two parts",
			trn:             "trn:job:invalid",
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
			actualJob, err := testClient.client.Jobs.GetJobByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectJob {
				require.NotNil(t, actualJob)
				assert.Equal(t, types.JobModelType.BuildTRN(workspace.FullPath, job.GetGlobalID()), actualJob.Metadata.TRN)
			} else {
				assert.Nil(t, actualJob)
			}
		})
	}
}

func TestGetJobs(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, warmupRuns, _, warmupJobs, err := createWarmupJobs(ctx, testClient,
		standardWarmupGroupsForJobs, standardWarmupWorkspacesForJobs,
		standardWarmupRunsForJobs, standardWarmupRunnersForJobs,
		standardWarmupJobs)
	require.Nil(t, err)
	allJobInfos := jobInfoFromJobs(warmupJobs)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(jobInfoIDSlice(allJobInfos))
	allJobIDs := jobIDsFromJobInfos(allJobInfos)

	// Sort by creation times.
	sort.Sort(jobInfoCreateSlice(allJobInfos))
	allJobIDsByCreateTime := jobIDsFromJobInfos(allJobInfos)

	// Sort by last update times.
	sort.Sort(jobInfoUpdateSlice(allJobInfos))
	allJobIDsByUpdateTime := jobIDsFromJobInfos(allJobInfos)
	reverseJobIDsByUpdateTime := reverseStringSlice(allJobIDsByUpdateTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetJobsInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectJobIDs                []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetJobsInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectJobIDs                []string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	JobTypeEmpty := models.JobType("")
	JobTypePlan := models.JobPlanType
	JobTypeApply := models.JobApplyType

	JobStatusEmpty := models.JobStatus("")
	JobStatusQueued := models.JobQueued
	JobStatusPending := models.JobPending
	JobStatusRunning := models.JobRunning
	JobStatusFinished := models.JobFinished

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetJobs, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetJobsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectJobIDs:         allJobIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectJobIDs:         allJobIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectJobIDs:         allJobIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtDesc),
			},
			expectJobIDs:         reverseJobIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectJobIDs:         allJobIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectJobIDs: allJobIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allJobIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectJobIDs:               allJobIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allJobIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectJobIDs:               allJobIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allJobIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// When Last is supplied, the sort order is intended to be reversed.
		{
			name: "pagination: last three",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectJobIDs: reverseJobIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allJobIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		/*

			The input.PaginationOptions.After field is tested earlier via getAfterCursorFromPrevious.

			The input.PaginationOptions.Before field is not really supported and does not work.
			If it did work, it could be tested by adapting the test cases corresponding to the
			next few cases after a similar block of text from group_test.go

		*/

		{
			name: "pagination, before and after, expect error",
			input: &GetJobsInput{
				Sort:              ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectJobIDs:                []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allJobIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &JobFilter{
					RunID:       ptr.String(""),
					WorkspaceID: ptr.String(""),
					JobType:     &JobTypeEmpty,
					JobStatus:   &JobStatusEmpty,
					// Passing an empty slice to JobIDs likely causes an SQL syntax error ("... IN ()"), so don't try it.
					// JobIDs: []string{},
				},
			},
			expectMsg:      emptyUUIDMsg2,
			expectJobIDs:   []string{},
			expectPageInfo: pagination.PageInfo{},
		},

		{
			name: "filter, run ID",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					RunID: ptr.String(warmupRuns[0].Metadata.ID),
				},
			},
			expectJobIDs:         allJobIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, run ID, non-existent",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					RunID: ptr.String(nonExistentID),
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, run ID, invalid",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					RunID: ptr.String(invalidID),
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					WorkspaceID: ptr.String(warmupWorkspaces[0].Metadata.ID),
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[2], allJobIDsByCreateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(3), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, non-existent",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					WorkspaceID: ptr.String(nonExistentID),
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, invalid",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					WorkspaceID: ptr.String(invalidID),
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job type plan",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobType: &JobTypePlan,
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[2], allJobIDsByCreateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job type apply",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobType: &JobTypeApply,
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[1], allJobIDsByCreateTime[3]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job type, empty",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					JobType: &JobTypeEmpty,
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job status queued",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobStatus: &JobStatusQueued,
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job status pending",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobStatus: &JobStatusPending,
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[1]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job status running",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobStatus: &JobStatusRunning,
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[2]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job status finished",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobStatus: &JobStatusFinished,
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[3]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job status, empty",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldUpdatedAtAsc),
				Filter: &JobFilter{
					JobStatus: &JobStatusEmpty,
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job IDs",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobIDs: []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[1], allJobIDsByCreateTime[4]},
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[1], allJobIDsByCreateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job IDs, non-existent",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobIDs: []string{nonExistentID},
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, job IDs, invalid",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, tag filter, run untagged, don't filter by tags, return all",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(false),
						TagSuperset:         nil,
					},
				},
			},
			expectJobIDs:         allJobIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allJobIDsByCreateTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, tag filter, run untagged, require empty tags, return only jobs with no tags",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(false),
						TagSuperset:         []string{},
					},
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[2], allJobIDsByCreateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(3), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, tag filter, don't run untagged, return none",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(true),
						TagSuperset:         []string{},
					},
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, tag filter, require tag 1, return empty",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(true),
						TagSuperset:         []string{"tag1"},
					},
				},
			},
			expectJobIDs:         []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, tag filter, require tag1 and tag2, return workspace 1",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					TagFilter: &JobTagFilter{
						ExcludeUntaggedJobs: ptr.Bool(true),
						TagSuperset:         []string{"tag1", "tag2"},
					},
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[1], allJobIDsByCreateTime[3]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "duplicate the exact query the service layer uses to claim a job",
			input: &GetJobsInput{
				Sort: ptrJobSortableField(JobSortableFieldCreatedAtAsc),
				Filter: &JobFilter{
					JobStatus: &JobStatusQueued,
					TagFilter: &JobTagFilter{
						TagSuperset: []string{},
					},
				},
			},
			expectJobIDs:         []string{allJobIDsByCreateTime[0], allJobIDsByCreateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},
	}

	// Combinations of filter conditions are not (yet) tested.

	var (
		previousEndCursorValue   *string
		previousStartCursorValue *string
	)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// For some pagination tests, a previous case's cursor value gets piped into the next case.
			if test.getAfterCursorFromPrevious || test.getBeforeCursorFromPrevious {

				// Make sure there's a place to put it.
				require.NotNil(t, test.input.PaginationOptions)

				if test.getAfterCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousEndCursorValue)
					test.input.PaginationOptions.After = previousEndCursorValue
				}

				if test.getBeforeCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousStartCursorValue)
					test.input.PaginationOptions.Before = previousStartCursorValue
				}

				// Clear the values so they won't be used twice.
				previousEndCursorValue = nil
				previousStartCursorValue = nil
			}

			jobsActual, err := testClient.client.Jobs.GetJobs(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, jobsActual.PageInfo)
				assert.NotNil(t, jobsActual.Jobs)
				pageInfo := jobsActual.PageInfo
				jobs := jobsActual.Jobs

				// Check the jobs result by comparing a list of the job IDs.
				actualJobIDs := []string{}
				for _, job := range jobs {
					actualJobIDs = append(actualJobIDs, job.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualJobIDs)
				}

				assert.Equal(t, len(test.expectJobIDs), len(actualJobIDs))
				assert.Equal(t, test.expectJobIDs, actualJobIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one job returned.
				// If there are no jobs returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(jobs) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&jobs[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&jobs[len(jobs)-1])
					assert.Equal(t, test.expectStartCursorError, resultStartCursorError)
					assert.Equal(t, test.expectHasStartCursor, resultStartCursor != nil)
					assert.Equal(t, test.expectEndCursorError, resultEndCursorError)
					assert.Equal(t, test.expectHasEndCursor, resultEndCursor != nil)

					// Capture the ending cursor values for the next case.
					previousEndCursorValue = resultEndCursor
					previousStartCursorValue = resultStartCursor
				}
			}
		})
	}
}

func TestCreateJob(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, warmupRuns, warmupRunners, _, err := createWarmupJobs(ctx, testClient,
		standardWarmupGroupsForJobs, standardWarmupWorkspacesForJobs,
		standardWarmupRunsForJobs, standardWarmupRunnersForJobs,
		standardWarmupJobs)
	require.Nil(t, err)
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toCreate      *models.Job
		expectCreated *models.Job
		expectMsg     *string
		name          string
	}

	now := currentTime()
	nowMinusA := now.Add(-5 * time.Minute)
	nowMinusB := now.Add(-11 * time.Minute)
	nowMinusC := now.Add(-9 * time.Minute)
	nowMinusD := now.Add(-7 * time.Minute)
	nowMinusE := now.Add(-3 * time.Minute)
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.Job{
				WorkspaceID: warmupWorkspaceID,
				RunID:       warmupRuns[0].Metadata.ID,
				RunnerID:    &warmupRunners[0].Metadata.ID,
			},
			expectCreated: &models.Job{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
				RunID:       warmupRuns[0].Metadata.ID,
				RunnerID:    &warmupRunners[0].Metadata.ID,
			},
		},

		{
			name: "positive full",
			toCreate: &models.Job{
				Status:                   models.JobFinished,
				Type:                     models.JobApplyType,
				WorkspaceID:              warmupWorkspaceID,
				RunID:                    warmupRuns[0].Metadata.ID,
				RunnerID:                 &warmupRunners[0].Metadata.ID,
				CancelRequested:          true,
				CancelRequestedTimestamp: ptr.Time(nowMinusA),
				Timestamps: models.JobTimestamps{
					QueuedTimestamp:   ptr.Time(nowMinusB),
					PendingTimestamp:  ptr.Time(nowMinusC),
					RunningTimestamp:  ptr.Time(nowMinusD),
					FinishedTimestamp: ptr.Time(nowMinusE),
				},
				MaxJobDuration: 42,
			},
			expectCreated: &models.Job{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Status:                   models.JobFinished,
				Type:                     models.JobApplyType,
				WorkspaceID:              warmupWorkspaceID,
				RunID:                    warmupRuns[0].Metadata.ID,
				RunnerID:                 &warmupRunners[0].Metadata.ID,
				CancelRequested:          true,
				CancelRequestedTimestamp: ptr.Time(nowMinusA),
				Timestamps: models.JobTimestamps{
					QueuedTimestamp:   ptr.Time(nowMinusB),
					PendingTimestamp:  ptr.Time(nowMinusC),
					RunningTimestamp:  ptr.Time(nowMinusD),
					FinishedTimestamp: ptr.Time(nowMinusE),
				},
				MaxJobDuration: 42,
			},
		},

		// It does not make sense to try to create a duplicate job,
		// because there is no unique name field to trigger an error.

		{
			name: "non-existent workspace ID",
			toCreate: &models.Job{
				WorkspaceID: nonExistentID,
				RunID:       warmupRuns[0].Metadata.ID,
				RunnerID:    &warmupRunners[0].Metadata.ID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"jobs\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective workspace ID",
			toCreate: &models.Job{
				WorkspaceID: invalidID,
				RunID:       warmupRuns[0].Metadata.ID,
				RunnerID:    &warmupRunners[0].Metadata.ID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.Jobs.CreateJob(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareJobs(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestUpdateJob(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a job with a specific ID without going into the really
	// low-level stuff, create the warmup job(s) and then find the relevant ID.
	createdLow := currentTime()
	warmupWorkspaces, warmupRuns, warmupRunners, warmupJobs, err := createWarmupJobs(ctx, testClient,
		standardWarmupGroupsForJobs, standardWarmupWorkspacesForJobs,
		standardWarmupRunsForJobs, standardWarmupRunnersForJobs,
		standardWarmupJobs)
	require.Nil(t, err)
	createdHigh := currentTime()
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toUpdate  *models.Job
		expectJob *models.Job
		expectMsg *string
		name      string
	}

	// Do only one positive test case, because the logic is theoretically the same for all jobs.
	now := currentTime()
	nowMinusA := now.Add(-25 * time.Minute)
	nowMinusB := now.Add(-31 * time.Minute)
	nowMinusC := now.Add(-29 * time.Minute)
	nowMinusD := now.Add(-27 * time.Minute)
	nowMinusE := now.Add(-23 * time.Minute)
	positiveJob := warmupJobs[0]
	newRunID := warmupRuns[1].Metadata.ID
	mainRunnerID := warmupRunners[0].Metadata.ID
	otherRunnerID := warmupRunners[1].Metadata.ID
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.Job{
				Metadata: models.ResourceMetadata{
					ID:      positiveJob.Metadata.ID,
					Version: positiveJob.Metadata.Version,
				},
				Status:                   models.JobFinished,
				Type:                     models.JobApplyType,
				WorkspaceID:              warmupWorkspaceID,
				RunID:                    newRunID,
				RunnerID:                 &otherRunnerID,
				CancelRequested:          true,
				CancelRequestedTimestamp: ptr.Time(nowMinusA),
				Timestamps: models.JobTimestamps{
					QueuedTimestamp:   ptr.Time(nowMinusB),
					PendingTimestamp:  ptr.Time(nowMinusC),
					RunningTimestamp:  ptr.Time(nowMinusD),
					FinishedTimestamp: ptr.Time(nowMinusE),
				},
				// MaxJobDuration cannot be updated.
			},
			expectJob: &models.Job{
				Metadata: models.ResourceMetadata{
					ID:                   positiveJob.Metadata.ID,
					Version:              positiveJob.Metadata.Version + 1,
					CreationTimestamp:    positiveJob.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Status:                   models.JobFinished,
				Type:                     models.JobApplyType,
				WorkspaceID:              warmupWorkspaceID,
				RunID:                    newRunID,
				RunnerID:                 &otherRunnerID,
				CancelRequested:          true,
				CancelRequestedTimestamp: ptr.Time(nowMinusA),
				Timestamps: models.JobTimestamps{
					QueuedTimestamp:   ptr.Time(nowMinusB),
					PendingTimestamp:  ptr.Time(nowMinusC),
					RunningTimestamp:  ptr.Time(nowMinusD),
					FinishedTimestamp: ptr.Time(nowMinusE),
				},
				MaxJobDuration: positiveJob.MaxJobDuration,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.Job{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveJob.Metadata.Version,
				},
				WorkspaceID: warmupWorkspaceID,
				RunID:       warmupRuns[0].Metadata.ID,
				RunnerID:    &mainRunnerID,
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.Job{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveJob.Metadata.Version,
				},
				RunID:    warmupRuns[0].Metadata.ID,
				RunnerID: &mainRunnerID,
			},
			expectMsg: invalidUUIDMsg4,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			job, err := testClient.client.Jobs.UpdateJob(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectJob != nil {
				require.NotNil(t, job)
				now := currentTime()
				compareJobs(t, test.expectJob, job, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  test.expectJob.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, job)
			}
		})
	}
}

func TestGetLatestJobByType(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a job with a specific ID without going into the really
	// low-level stuff, create the warmup job(s) then find the relevant ID.
	createdLow := currentTime()
	_, warmupRuns, _, createdWarmupJobs, err := createWarmupJobs(ctx, testClient,
		standardWarmupGroupsForJobs, standardWarmupWorkspacesForJobs,
		standardWarmupRunsForJobs, standardWarmupRunnersForJobs,
		standardWarmupJobs)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectJob     *models.Job
		expectMsg     *string
		name          string
		searchRunID   string
		searchJobType models.JobType
	}

	createdRunID := warmupRuns[0].Metadata.ID
	JobTypeEmpty := models.JobType("")

	// The positive test cases return the latest according to the late updated timestamp, not creation time.
	testCases := []testCase{
		{
			name:          "job type plan",
			searchRunID:   createdRunID,
			searchJobType: models.JobPlanType,
			expectJob:     &createdWarmupJobs[0],
			// Candidates are elements 0, 2, and 4.  0 is updated last, because its index is a multiple of 3.
		},
		{
			name:          "job type apply",
			searchRunID:   createdRunID,
			searchJobType: models.JobApplyType,
			expectJob:     &createdWarmupJobs[3],
			// Candidates are elements 1, and 3.  3 is updated late, because its index is a multiple of 3.
		},
		{
			name:          "negative, empty job type",
			searchRunID:   createdRunID,
			searchJobType: JobTypeEmpty,
			// expect job and error to be nil
		},
		{
			name:          "negative, non-existent run ID",
			searchRunID:   nonExistentID,
			searchJobType: models.JobPlanType,
			// expect job and error to be nil
		},
		{
			name:          "defective run id",
			searchRunID:   invalidID,
			searchJobType: models.JobPlanType,
			expectMsg:     ptr.String("failed to get job: " + *invalidUUIDMsg2),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			job, err := testClient.client.Jobs.GetLatestJobByType(ctx, test.searchRunID, test.searchJobType)

			checkError(t, test.expectMsg, err)

			if test.expectJob != nil {
				require.NotNil(t, job)
				compareJobs(t, test.expectJob, job, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, job)
			}
		})
	}
}

func TestGetJobCountForRunner(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a job with a specific ID without going into the really
	// low-level stuff, create the warmup job(s) and then find the relevant ID.
	_, _, warmupRunners, _, err := createWarmupJobs(ctx, testClient,
		standardWarmupGroupsForJobs, standardWarmupWorkspacesForJobs,
		standardWarmupRunsForJobs, standardWarmupRunnersForJobs,
		standardWarmupJobs)
	require.Nil(t, err)

	type testCase struct {
		expectMsg   *string
		runnerID    string
		name        string
		expectCount int
	}

	// Do only one positive test case, because the logic is theoretically the same for all log streams.
	testCases := []testCase{
		{
			name:        "positive",
			runnerID:    warmupRunners[0].Metadata.ID,
			expectCount: 2, // jobs are counted only if in pending or running state
		},
		{
			name:        "negative, non-existent ID",
			runnerID:    nonExistentID,
			expectCount: 0,
		},
		{
			name:      "defective-id",
			runnerID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCount, err := testClient.client.Jobs.GetJobCountForRunner(ctx, test.runnerID)

			checkError(t, test.expectMsg, err)

			assert.Equal(t, test.expectCount, actualCount)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForJobs = []models.Group{
	{
		Description: "top level group 0 for testing job functions",
		FullPath:    "top-level-group-0-for-jobs",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForJobs = []models.Workspace{
	{
		Description: "workspace 0 for testing job functions",
		FullPath:    "top-level-group-0-for-jobs/workspace-0-for-jobs",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing job functions",
		FullPath:    "top-level-group-0-for-jobs/workspace-1-for-jobs",
		CreatedBy:   "someone-w1",
		RunnerTags:  []string{"tag1", "tag2"},
		// The tags must also be set in the job objects,
		// because the test setup does not set the jobs tags field from the workspace and groups.
	},
}

// Standard warmup run(s) for tests in this module:
// In the job object, the RunID field must not be nil and must match a valid Run object.
var standardWarmupRunsForJobs = []models.Run{
	{
		Comment: "standard run object 0 for testing job functions",
	},
	{
		Comment: "standard run object 1 for testing job functions",
	},
}

// Standard warmup runner(s) for tests in this module:
// In the job object, the RunnerID field can be null, but if not null it has to match a runner row.
var standardWarmupRunnersForJobs = []models.Runner{
	{
		Name:        "runner-1",
		Description: "runner 1",
	},
	{
		Name:        "runner-other",
		Description: "the other runner",
	},
}

// Standard warmup job(s) for tests in this module:
var standardWarmupJobs = []models.Job{
	{
		Status:                   models.JobQueued,
		Type:                     models.JobPlanType,
		CancelRequested:          true,
		CancelRequestedTimestamp: ptr.Time(currentTime().Add(-3 * time.Minute)),
		Timestamps: models.JobTimestamps{
			QueuedTimestamp:   ptr.Time(currentTime().Add(-9 * time.Minute)),
			PendingTimestamp:  ptr.Time(currentTime().Add(-7 * time.Minute)),
			RunningTimestamp:  ptr.Time(currentTime().Add(-5 * time.Minute)),
			FinishedTimestamp: ptr.Time(currentTime().Add(-1 * time.Minute)),
		},
		MaxJobDuration: 39,
	},
	{
		Status:                   models.JobPending,
		Type:                     models.JobApplyType,
		CancelRequested:          true,
		CancelRequestedTimestamp: ptr.Time(currentTime().Add(-13 * time.Minute)),
		Timestamps: models.JobTimestamps{
			QueuedTimestamp:   ptr.Time(currentTime().Add(-19 * time.Minute)),
			PendingTimestamp:  ptr.Time(currentTime().Add(-17 * time.Minute)),
			RunningTimestamp:  ptr.Time(currentTime().Add(-15 * time.Minute)),
			FinishedTimestamp: ptr.Time(currentTime().Add(-11 * time.Minute)),
		},
		MaxJobDuration: 139,
		Tags:           []string{"tag1", "tag2"}, // corresponds to workspace 1
	},
	{
		Status:                   models.JobRunning,
		Type:                     models.JobPlanType,
		CancelRequested:          true,
		CancelRequestedTimestamp: ptr.Time(currentTime().Add(-23 * time.Minute)),
		Timestamps: models.JobTimestamps{
			QueuedTimestamp:   ptr.Time(currentTime().Add(-29 * time.Minute)),
			PendingTimestamp:  ptr.Time(currentTime().Add(-27 * time.Minute)),
			RunningTimestamp:  ptr.Time(currentTime().Add(-25 * time.Minute)),
			FinishedTimestamp: ptr.Time(currentTime().Add(-21 * time.Minute)),
		},
		MaxJobDuration: 239,
	},
	{
		Status:                   models.JobFinished,
		Type:                     models.JobApplyType,
		CancelRequested:          true,
		CancelRequestedTimestamp: ptr.Time(currentTime().Add(-33 * time.Minute)),
		Timestamps: models.JobTimestamps{
			QueuedTimestamp:   ptr.Time(currentTime().Add(-39 * time.Minute)),
			PendingTimestamp:  ptr.Time(currentTime().Add(-37 * time.Minute)),
			RunningTimestamp:  ptr.Time(currentTime().Add(-35 * time.Minute)),
			FinishedTimestamp: ptr.Time(currentTime().Add(-31 * time.Minute)),
		},
		MaxJobDuration: 339,
		Tags:           []string{"tag1", "tag2"}, // corresponds to workspace 1
	},
	{
		Status:                   models.JobQueued,
		Type:                     models.JobPlanType,
		CancelRequested:          true,
		CancelRequestedTimestamp: ptr.Time(currentTime().Add(-43 * time.Minute)),
		Timestamps: models.JobTimestamps{
			QueuedTimestamp:   ptr.Time(currentTime().Add(-49 * time.Minute)),
			PendingTimestamp:  ptr.Time(currentTime().Add(-47 * time.Minute)),
			RunningTimestamp:  ptr.Time(currentTime().Add(-45 * time.Minute)),
			FinishedTimestamp: ptr.Time(currentTime().Add(-41 * time.Minute)),
		},
		MaxJobDuration: 439,
	},
}

// createWarmupJobs creates some warmup jobs for a test
// The warmup jobs to create can be standard or otherwise.
func createWarmupJobs(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newRuns []models.Run,
	newRunners []models.Runner,
	newJobs []models.Job) (
	[]models.Workspace,
	[]models.Run,
	[]models.Runner,
	[]models.Job,
	error,
) {
	// It is necessary to create at least one group, workspace, and run
	// in order to provide the necessary IDs for the jobs.

	_, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	workspaceIDs := []string{}
	for _, workspace := range resultWorkspaces {
		workspaceIDs = append(workspaceIDs, workspace.Metadata.ID)
	}

	resultRuns, err := createInitialRuns(ctx, testClient, newRuns, workspaceIDs[0])
	if err != nil {
		return nil, nil, nil, nil, err
	}

	resultRunners, _, err := createInitialRunners(ctx, testClient, newRunners, parentPath2ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	resultJobs, err := createInitialJobs(ctx, testClient, newJobs,
		workspaceIDs, resultRuns[0].Metadata.ID, resultRunners[0].Metadata.ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return resultWorkspaces, resultRuns, resultRunners, resultJobs, nil
}

func ptrJobSortableField(arg JobSortableField) *JobSortableField {
	return &arg
}

func (wis jobInfoIDSlice) Len() int {
	return len(wis)
}

func (wis jobInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis jobInfoIDSlice) Less(i, j int) bool {
	return wis[i].jobID < wis[j].jobID
}

func (wis jobInfoCreateSlice) Len() int {
	return len(wis)
}

func (wis jobInfoCreateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis jobInfoCreateSlice) Less(i, j int) bool {
	return wis[i].createTime.Before(wis[j].createTime)
}

func (wis jobInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis jobInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis jobInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// jobInfoFromJobs returns a slice of jobInfo, not necessarily sorted in any order.
func jobInfoFromJobs(jobs []models.Job) []jobInfo {
	result := []jobInfo{}

	for _, job := range jobs {
		result = append(result, jobInfo{
			jobID:      job.Metadata.ID,
			createTime: *job.Metadata.CreationTimestamp,
			updateTime: *job.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// jobIDsFromJobInfos preserves order
func jobIDsFromJobInfos(jobInfos []jobInfo) []string {
	result := []string{}
	for _, jobInfo := range jobInfos {
		result = append(result, jobInfo.jobID)
	}
	return result
}

// compareJobs compares two job objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareJobs(t *testing.T, expected, actual *models.Job,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.RunID, actual.RunID)
	assert.Equal(t, expected.RunnerID, actual.RunnerID)
	assert.Equal(t, expected.CancelRequested, actual.CancelRequested)
	assert.Equal(t, expected.CancelRequestedTimestamp, actual.CancelRequestedTimestamp)
	assert.Equal(t, expected.Timestamps.QueuedTimestamp, actual.Timestamps.QueuedTimestamp)
	assert.Equal(t, expected.Timestamps.PendingTimestamp, actual.Timestamps.PendingTimestamp)
	assert.Equal(t, expected.Timestamps.RunningTimestamp, actual.Timestamps.RunningTimestamp)
	assert.Equal(t, expected.Timestamps.FinishedTimestamp, actual.Timestamps.FinishedTimestamp)
	assert.Equal(t, expected.MaxJobDuration, actual.MaxJobDuration)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEmpty(t, actual.Metadata.TRN)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
