//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

type warmupLogStreams struct {
	groups     []models.Group
	workspaces []models.Workspace
	runs       []models.Run
	jobs       []models.Job
	logStreams []models.LogStream
	runners    []models.Runner
}

func TestGetLogStreamByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupLogStreams(ctx, testClient, warmupLogStreams{
		groups:     standardWarmupGroupsForLogStreams,
		workspaces: standardWarmupWorkspacesForLogStreams,
		runs:       standardWarmupRunsForLogStreams,
		jobs:       standardWarmupJobsForLogStreams,
		logStreams: standardWarmupLogStreams,
		runners:    standardWarmupRunnersForLogStreams,
	})
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectErrorCode errors.CodeType
		expectLogStream *models.LogStream
		name            string
		searchID        string
	}

	positiveLogStream := warmupItems.logStreams[0]

	/*
		test case template

		{
			name            string
			searchID        string
			expectErrorCode errors.CodeType
			expectLogStream *models.LogStream
		}
	*/

	testCases := []testCase{
		{
			name:            "get log stream by id",
			searchID:        positiveLogStream.Metadata.ID,
			expectLogStream: &positiveLogStream,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
		},
		{
			name:            "defective-id",
			searchID:        invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStream, err := testClient.client.LogStreams.GetLogStreamByID(ctx, test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLogStream != nil {
				require.NotNil(t, logStream)
				compareLogStreams(t, test.expectLogStream, logStream, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, logStream)
			}
		})
	}
}

func TestGetLogStreamByJobID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupLogStreams(ctx, testClient, warmupLogStreams{
		groups:     standardWarmupGroupsForLogStreams,
		workspaces: standardWarmupWorkspacesForLogStreams,
		runs:       standardWarmupRunsForLogStreams,
		jobs:       standardWarmupJobsForLogStreams,
		runners:    standardWarmupRunnersForLogStreams,
		logStreams: standardWarmupLogStreams,
	})
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectErrorCode errors.CodeType
		expectLogStream *models.LogStream
		name            string
		jobID           *string
	}

	positiveLogStream := warmupItems.logStreams[0]

	/*
		test case template

		{
			name            string
			jobID           string
			expectErrorCode errors.CodeType
			expectLogStream *models.LogStream
		}
	*/

	testCases := []testCase{
		{
			name:            "get log stream by job id",
			jobID:           positiveLogStream.JobID,
			expectLogStream: &positiveLogStream,
		},
		{
			name:  "negative, non-existent ID",
			jobID: ptr.String(nonExistentID),
		},
		{
			name:            "defective-id",
			jobID:           ptr.String(invalidID),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStream, err := testClient.client.LogStreams.GetLogStreamByJobID(ctx, *test.jobID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLogStream != nil {
				require.NotNil(t, logStream)
				compareLogStreams(t, test.expectLogStream, logStream, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, logStream)
			}
		})
	}
}

func TestCreateLogStream(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupLogStreams(ctx, testClient, warmupLogStreams{
		groups:     standardWarmupGroupsForLogStreams,
		workspaces: standardWarmupWorkspacesForLogStreams,
		runs:       standardWarmupRunsForLogStreams,
		jobs:       standardWarmupJobsForLogStreams,
		runners:    standardWarmupRunnersForLogStreams,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		toCreate        *models.LogStream
		expectCreated   *models.LogStream
		name            string
	}

	now := currentTime()

	warmupJobID := warmupItems.jobs[0].Metadata.ID

	/*
		test case template

		{
			name            string
			toCreate        *models.LogStream
			expectErrorCode errors.CodeType
			expectCreated   *models.LogStream
		}
	*/

	testCases := []testCase{
		{
			name: "positive, standard warmup logStreams",
			toCreate: &models.LogStream{
				JobID:     &warmupJobID,
				Size:      2,
				Completed: false,
			},
			expectCreated: &models.LogStream{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				JobID:     &warmupJobID,
				Size:      2,
				Completed: false,
			},
		},
		{
			name: "negative, non-existent jobID",
			toCreate: &models.LogStream{
				JobID: ptr.String(nonExistentID),
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "negative, defective jobID",
			toCreate: &models.LogStream{
				JobID: ptr.String(invalidID),
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "negative, duplicate logStream",
			toCreate: &models.LogStream{
				JobID: &warmupJobID,
			},
			expectErrorCode: errors.EConflict,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.LogStreams.CreateLogStream(ctx, test.toCreate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareLogStreams(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateLogStream(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupLogStreams(ctx, testClient, warmupLogStreams{
		groups:     standardWarmupGroupsForLogStreams,
		workspaces: standardWarmupWorkspacesForLogStreams,
		runs:       standardWarmupRunsForLogStreams,
		jobs:       standardWarmupJobsForLogStreams,
		logStreams: standardWarmupLogStreams,
		runners:    standardWarmupRunnersForLogStreams,
	})
	require.Nil(t, err)
	createdHigh := currentTime()

	positiveLogStream := warmupItems.logStreams[0]
	updatedSize := 10

	type testCase struct {
		expectErrorCode errors.CodeType
		expectLogStream *models.LogStream
		toUpdate        *models.LogStream
		name            string
	}

	now := currentTime()

	/*
		test case template

		{
			name            string
			toUpdate        *models.LogStream
			expectErrorCode errors.CodeType
			expectLogStream *models.LogStream
		}
	*/

	testCases := []testCase{
		{
			name: "update size",
			toUpdate: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID:      positiveLogStream.Metadata.ID,
					Version: positiveLogStream.Metadata.Version,
				},
				JobID:     positiveLogStream.JobID,
				Size:      updatedSize,
				Completed: true,
			},
			expectLogStream: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID:                   positiveLogStream.Metadata.ID,
					Version:              positiveLogStream.Metadata.Version + 1,
					CreationTimestamp:    positiveLogStream.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				JobID:     positiveLogStream.JobID,
				Size:      updatedSize,
				Completed: true,
			},
		},
		{
			name: "negative, not exist",
			toUpdate: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectErrorCode: errors.EOptimisticLock,
		},
		{
			name: "negative, invalid uuid",
			toUpdate: &models.LogStream{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			logStream, err := testClient.client.LogStreams.UpdateLogStream(ctx, test.toUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLogStream != nil {
				require.NotNil(t, logStream)
				now := currentTime()
				compareLogStreams(t, test.expectLogStream, logStream, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  test.expectLogStream.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, logStream)
			}
		})
	}
}

// standardWarmupGroupsForLogStreams is a list of groups that are created for testing in this module
var standardWarmupGroupsForLogStreams = []models.Group{
	{
		Name:        "group-1",
		Description: "standard warmup group-1",
		CreatedBy:   "someone-1",
	},
}

// standardWarmupWorkspacesForLogStreams is a list of workspaces that are created for testing in this module
var standardWarmupWorkspacesForLogStreams = []models.Workspace{
	{
		Name:        "workspace-1",
		Description: "standard warmup workspace-1",
		CreatedBy:   "someone-1",
		FullPath:    "group-1/workspace-1",
	},
}

// standardWarmupRunnersForLogStreams is a list of runners that are created for testing in this module
var standardWarmupRunnersForLogStreams = []models.Runner{
	{
		GroupID:     ptr.String("group-1"),
		Name:        "runner-1",
		Type:        models.GroupRunnerType,
		Description: "standard warmup runner-1",
		CreatedBy:   "someone-1",
	},
}

// standardWarmupRunsForLogStreams is a list of runs that are created for testing in this module
var standardWarmupRunsForLogStreams = []models.Run{
	{
		WorkspaceID: "workspace-1",
	},
}

// standardWarmupJobsForLogStreams is a list of jobs that are created for testing in this module
var standardWarmupJobsForLogStreams = []models.Job{
	{
		WorkspaceID:              "workspace-1",
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
		RunnerID:       ptr.String("runner-1"),
		RunnerPath:     ptr.String("runner-1"),
	},
}

// standardWarmupLogStreams is a list of logStreams that are created for testing in this module
var standardWarmupLogStreams = []models.LogStream{
	{
		JobID: ptr.String("job-1"),
		Size:  2,
	},
}

// createWarmupLogStreams creates a set of logStreams and their dependencies for testing
func createWarmupLogStreams(
	ctx context.Context,
	testClient *testClient,
	input warmupLogStreams,
) (*warmupLogStreams, error) {

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultRunners, _, err := createInitialRunners(ctx, testClient, input.runners, groupPath2ID)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, groupPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	resultRuns, err := createInitialRuns(ctx, testClient, input.runs, resultWorkspaces[0].Metadata.ID)
	if err != nil {
		return nil, err
	}

	resultJobs, err := createInitialJobs(ctx, testClient, input.jobs,
		resultWorkspaces[0].Metadata.ID, resultRuns[0].Metadata.ID, resultRunners[0].Metadata.ID)
	if err != nil {
		return nil, err
	}

	resultLogStreams, err := createInitialLogStreams(ctx, testClient, resultJobs, input.logStreams)
	if err != nil {
		return nil, err
	}

	return &warmupLogStreams{
		groups:     resultGroups,
		runners:    resultRunners,
		workspaces: resultWorkspaces,
		runs:       resultRuns,
		jobs:       resultJobs,
		logStreams: resultLogStreams,
	}, nil
}

// createInitialLogStreams creates a set of logStreams for testing
func createInitialLogStreams(
	ctx context.Context,
	testClient *testClient,
	jobs []models.Job,
	toCreate []models.LogStream,
) ([]models.LogStream, error) {
	if len(jobs) == 0 {
		return nil, fmt.Errorf("no jobs available in createInitialLogStreams")
	}

	result := []models.LogStream{}

	for _, l := range toCreate {
		l.JobID = &jobs[0].Metadata.ID

		created, err := testClient.client.LogStreams.CreateLogStream(ctx, &l)
		if err != nil {
			return nil, err
		}

		result = append(result, *created)
	}

	return result, nil
}

// compareLogStreams compares two logStreams
// If times is nil, it compares the exact metadata timestamps.
func compareLogStreams(t *testing.T,
	expected,
	actual *models.LogStream,
	checkID bool,
	times *timeBounds,
) {
	assert.Equal(t, expected.JobID, actual.JobID)
	assert.Equal(t, expected.Size, actual.Size)
	assert.Equal(t, expected.Completed, actual.Completed)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
