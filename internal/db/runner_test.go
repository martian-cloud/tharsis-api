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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// runnerInfo aids convenience in accessing the information
// TestGetRunners needs about the warmup objects.
type runnerInfo struct {
	updateTime time.Time
	id         string
	name       string
}

// runnerInfoIDSlice makes a slice of runnerInfo sortable by ID string
type runnerInfoIDSlice []runnerInfo

// runnerInfoUpdateSlice makes a slice of runnerInfo sortable by last updated time
type runnerInfoUpdateSlice []runnerInfo

// warmupRunners holds the inputs to and outputs from createWarmupRunners.
type warmupRunners struct {
	groups  []models.Group
	runners []models.Runner
}

func TestGetRunnerByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups:  standardWarmupGroupsForRunners,
		runners: standardWarmupRunners,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg    *string
		expectRunner *models.Runner
		name         string
		searchID     string
	}

	testCases := []testCase{
		{
			name:         "get runner by ID",
			searchID:     warmupItems.runners[0].Metadata.ID,
			expectRunner: &warmupItems.runners[0],
		},

		{
			name:     "returns nil because runner does not exist",
			searchID: nonExistentID,
		},

		{
			name:      "returns an error because the runner ID is invalid",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualRunner, err := testClient.client.Runners.GetRunnerByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectRunner != nil {
				require.NotNil(t, actualRunner)
				assert.Equal(t, test.expectRunner, actualRunner)
			} else {
				assert.Nil(t, actualRunner)
			}
		})
	}
}

func TestGetRunnerByPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups:  standardWarmupGroupsForRunners,
		runners: standardWarmupRunners,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg    *string
		expectRunner *models.Runner
		name         string
		searchPath   string
	}

	testCases := []testCase{
		{
			name:         "get group runner by path",
			searchPath:   warmupItems.runners[0].ResourcePath,
			expectRunner: &warmupItems.runners[0],
		},

		{
			name:         "get shared runner by path",
			searchPath:   "6-runner-shared",
			expectRunner: &warmupItems.runners[5],
		},

		{
			name:       "negative, non-existent runner ID",
			searchPath: "this/path/does/not/exist",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualRunner, err := testClient.client.Runners.GetRunnerByPath(ctx, test.searchPath)

			checkError(t, test.expectMsg, err)

			if test.expectRunner != nil {
				require.NotNil(t, actualRunner)
				assert.Equal(t, test.expectRunner, actualRunner)
			} else {
				assert.Nil(t, actualRunner)
			}
		})
	}
}

func TestGetRunnersWithPagination(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups:  standardWarmupGroupsForRunners,
		runners: standardWarmupRunners,
	})
	require.Nil(t, err)

	// Query for first page
	middleIndex := len(warmupItems.runners) / 2
	page1, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(int32(middleIndex)),
		},
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(page1.Runners))
	assert.True(t, page1.PageInfo.HasNextPage)
	assert.False(t, page1.PageInfo.HasPreviousPage)

	cursor, err := page1.PageInfo.Cursor(&page1.Runners[len(page1.Runners)-1])
	require.Nil(t, err)

	remaining := len(warmupItems.runners) - middleIndex
	page2, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(int32(remaining)),
			After: cursor,
		},
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(page2.Runners))
	assert.True(t, page2.PageInfo.HasPreviousPage)
	assert.False(t, page2.PageInfo.HasNextPage)
}

func TestGetRunners(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	t.Run("subtest: non-nil but mostly empty input", func(t *testing.T) {

		t.Log("Setting up subtest non-nil but mostly empty input")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest non-nil but mostly empty input")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest non-nil but mostly empty input")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort:              nil,
			PaginationOptions: nil,
			Filter:            nil,
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(runnersResult.Runners))
		assert.Equal(t, runner.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
	})

	t.Run("subtest: populated sort and pagination, nil filter", func(t *testing.T) {

		t.Log("Setting up subtest populated sort and pagination, nil filter")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest populated sort and pagination, nil filter")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest populated sort and pagination, nil filter")

		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(100),
			},
			Filter: nil,
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(runnersResult.Runners))
		assert.Equal(t, runner.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
	})

	t.Run("subtest: sort in ascending order of time of last update", func(t *testing.T) {

		t.Log("Setting up subtest sort in ascending order of time of last update")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-1",
			ResourcePath: "top-level-group-0-for-runners/1-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest sort in ascending order of time of last update")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest sort in ascending order of time of last update")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(runnersResult.Runners))
		assert.Equal(t, runner0.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, runner1.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
	})

	t.Run("subtest: sort in descending order of time of last update", func(t *testing.T) {

		t.Log("Setting up subtest sort in descending order of time of last update")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-1",
			ResourcePath: "top-level-group-0-for-runners/1-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest sort in descending order of time of last update")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest sort in descending order of time of last update")

		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtDesc),
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(runnersResult.Runners))
		assert.Equal(t, runner1.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, runner0.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
	})

	t.Run("subtest: pagination, first one and last two, expect error", func(t *testing.T) {

		t.Log("Setting up subtest pagination, first one and last two, expect error")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest pagination, first one and last two, expect error")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest pagination, first one and last two, expect error")

		_, err = testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(1),
				Last:  ptr.Int32(2),
			},
		})

		require.NotNil(t, err)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("subtest: filter, group ID, positive", func(t *testing.T) {

		t.Log("Setting up subtest filter, group ID, positive")
		group0, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 1 for testing runner functions",
			Name:        "top-level-group-1-for-runners",
			FullPath:    "top-level-group-1-for-runners",
			CreatedBy:   "someone-g1",
		})
		assert.Nil(t, err)

		runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group0.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter, group ID, positive")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group0))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group1))
		})

		t.Log("Running subtest filter, group ID, positive")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				GroupID: &group0.Metadata.ID,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(runnersResult.Runners))
		assert.Equal(t, runner.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
	})

	t.Run("subtest: filter, group ID, non-existent", func(t *testing.T) {
		t.Log("Setting up subtest filter, group ID, non-existent")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				GroupID: ptr.String(nonExistentID),
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 0, len(runnersResult.Runners))
	})

	t.Run("subtest: filter, group ID, invalid", func(t *testing.T) {
		t.Log("Running subtest filter, group ID, invalid")
		_, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				GroupID: ptr.String(invalidID),
			},
		})
		require.NotNil(t, err)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("subtest: filter, runner IDs, positive", func(t *testing.T) {

		t.Log("Setting up subtest filter, runner IDs, positive")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-1",
			ResourcePath: "top-level-group-0-for-runners/1-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv1",
		})
		assert.Nil(t, err)

		runner2, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-2",
			ResourcePath: "top-level-group-0-for-runners/1-runner-2",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv2",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter, runner IDs, positive")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner2))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter, runner IDs, positive")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				RunnerIDs: []string{runner0.Metadata.ID, runner2.Metadata.ID},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(runnersResult.Runners))
		assert.Equal(t, runner0.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, runner2.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
	})

	t.Run("subtest: filter, runner IDs, non-existent", func(t *testing.T) {
		t.Log("Running subtest filter, runner IDs, non-existent")
		_, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				RunnerIDs: []string{nonExistentID},
			},
		})
		assert.Nil(t, err)
	})

	t.Run("subtest: filter, runner IDs, invalid ID", func(t *testing.T) {
		t.Log("Running subtest filter, runner IDs, invalid ID")
		_, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				RunnerIDs: []string{invalidID},
			},
		})
		require.NotNil(t, err)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("subtest: filter, get shared runners", func(t *testing.T) {

		t.Log("Setting up subtest filter, get shared runners")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		groupRunner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "group-runner-0",
			ResourcePath: "top-level-group-0-for-runners/group-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		groupRunner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "group-runner-1",
			ResourcePath: "top-level-group-0-for-runners/group-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv1",
		})
		assert.Nil(t, err)

		sharedRunner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.SharedRunnerType,
			Name:         "shared-runner-0",
			ResourcePath: "shared-runner-0",
			CreatedBy:    "someone-sv2",
		})
		assert.Nil(t, err)

		sharedRunner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.SharedRunnerType,
			Name:         "shared-runner-1",
			ResourcePath: "shared-runner-1",
			CreatedBy:    "someone-sv3",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter, get shared runners")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, groupRunner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, groupRunner1))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, sharedRunner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, sharedRunner1))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter, get shared runners")
		localSharedRunnerType := models.SharedRunnerType
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				RunnerType: &localSharedRunnerType,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(runnersResult.Runners))
		assert.Equal(t, sharedRunner0.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, sharedRunner1.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
	})

	t.Run("subtest: filter, get group runners", func(t *testing.T) {

		t.Log("Setting up subtest filter, get group runners")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		groupRunner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "group-runner-0",
			ResourcePath: "top-level-group-0-for-runners/group-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
		})
		assert.Nil(t, err)

		groupRunner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "group-runner-1",
			ResourcePath: "top-level-group-0-for-runners/group-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv1",
		})
		assert.Nil(t, err)

		sharedRunner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.SharedRunnerType,
			Name:         "shared-runner-0",
			ResourcePath: "shared-runner-0",
			CreatedBy:    "someone-sv2",
		})
		assert.Nil(t, err)

		sharedRunner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.SharedRunnerType,
			Name:         "shared-runner-1",
			ResourcePath: "shared-runner-1",
			CreatedBy:    "someone-sv3",
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter, get group runners")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, groupRunner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, groupRunner1))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, sharedRunner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, sharedRunner1))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter, get group runners")
		localGroupRunnerType := models.GroupRunnerType
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				RunnerType: &localGroupRunnerType,
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(runnersResult.Runners))
		assert.Equal(t, groupRunner0.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, groupRunner1.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
	})

	t.Run("subtest: filter by run-untagged-jobs, true", func(t *testing.T) {

		t.Log("Setting up subtest filter by run-untagged-jobs, true")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:            models.GroupRunnerType,
			Name:            "1-runner-0",
			ResourcePath:    "top-level-group-0-for-runners/1-runner-0",
			GroupID:         &group.Metadata.ID,
			CreatedBy:       "someone-sv0",
			RunUntaggedJobs: true,
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:            models.GroupRunnerType,
			Name:            "1-runner-1",
			ResourcePath:    "top-level-group-0-for-runners/1-runner-1",
			GroupID:         &group.Metadata.ID,
			CreatedBy:       "someone-sv1",
			RunUntaggedJobs: false,
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter by run-untagged-jobs, true")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter by run-untagged-jobs, true")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				TagFilter: &RunnerTagFilter{
					RunUntaggedJobs: ptr.Bool(true),
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(runnersResult.Runners))
		assert.Equal(t, runner0.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
	})

	t.Run("subtest: filter by run-untagged-jobs, false", func(t *testing.T) {

		t.Log("Setting up subtest filter by run-untagged-jobs, false")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:            models.GroupRunnerType,
			Name:            "1-runner-0",
			ResourcePath:    "top-level-group-0-for-runners/1-runner-0",
			GroupID:         &group.Metadata.ID,
			CreatedBy:       "someone-sv0",
			RunUntaggedJobs: true,
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:            models.GroupRunnerType,
			Name:            "1-runner-1",
			ResourcePath:    "top-level-group-0-for-runners/1-runner-1",
			GroupID:         &group.Metadata.ID,
			CreatedBy:       "someone-sv1",
			RunUntaggedJobs: false,
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter by run-untagged-jobs, false")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter by run-untagged-jobs, false")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				TagFilter: &RunnerTagFilter{
					RunUntaggedJobs: ptr.Bool(false),
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(runnersResult.Runners))
		assert.Equal(t, runner1.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
	})

	t.Run("subtest: filter by tags, zero tags", func(t *testing.T) {

		t.Log("Setting up subtest filter by tags, zero tags")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
			Tags:         []string{},
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-1",
			ResourcePath: "top-level-group-0-for-runners/1-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv1",
			Tags:         []string{"tag1"},
		})
		assert.Nil(t, err)

		runner2, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-2",
			ResourcePath: "top-level-group-0-for-runners/1-runner-2",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv2",
			Tags:         []string{"tag1", "tag2"},
		})
		assert.Nil(t, err)

		runner3, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-3",
			ResourcePath: "top-level-group-0-for-runners/1-runner-3",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv3",
			Tags:         []string{"tag1", "tag2", "tag3"},
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter by tags, zero tags")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner2))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner3))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter by tags, zero tags")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				TagFilter: &RunnerTagFilter{
					TagSubset: []string{},
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 4, len(runnersResult.Runners))
		assert.Equal(t, runner0.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, runner1.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
		assert.Equal(t, runner2.Metadata.ID, runnersResult.Runners[2].Metadata.ID)
		assert.Equal(t, runner3.Metadata.ID, runnersResult.Runners[3].Metadata.ID)
	})

	t.Run("subtest: filter by tags, one tag", func(t *testing.T) {

		t.Log("Setting up subtest filter by tags, one tag")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
			Tags:         []string{},
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-1",
			ResourcePath: "top-level-group-0-for-runners/1-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv1",
			Tags:         []string{"tag1"},
		})
		assert.Nil(t, err)

		runner2, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-2",
			ResourcePath: "top-level-group-0-for-runners/1-runner-2",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv2",
			Tags:         []string{"tag1", "tag2"},
		})
		assert.Nil(t, err)

		runner3, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-3",
			ResourcePath: "top-level-group-0-for-runners/1-runner-3",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv3",
			Tags:         []string{"tag1", "tag2", "tag3"},
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter by tags, one tag")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner2))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner3))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter by tags, one tag")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				TagFilter: &RunnerTagFilter{
					TagSubset: []string{"tag1"},
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 3, len(runnersResult.Runners))
		assert.Equal(t, runner1.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, runner2.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
		assert.Equal(t, runner3.Metadata.ID, runnersResult.Runners[2].Metadata.ID)
	})

	t.Run("subtest: filter by tags, two tags", func(t *testing.T) {

		t.Log("Setting up subtest filter by tags, two tags")
		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Description: "top level group 0 for testing runner functions",
			Name:        "top-level-group-0-for-runners",
			FullPath:    "top-level-group-0-for-runners",
			CreatedBy:   "someone-g0",
		})
		assert.Nil(t, err)

		runner0, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-0",
			ResourcePath: "top-level-group-0-for-runners/1-runner-0",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv0",
			Tags:         []string{},
		})
		assert.Nil(t, err)

		runner1, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-1",
			ResourcePath: "top-level-group-0-for-runners/1-runner-1",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv1",
			Tags:         []string{"tag1"},
		})
		assert.Nil(t, err)

		runner2, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-2",
			ResourcePath: "top-level-group-0-for-runners/1-runner-2",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv2",
			Tags:         []string{"tag1", "tag2"},
		})
		assert.Nil(t, err)

		runner3, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Type:         models.GroupRunnerType,
			Name:         "1-runner-3",
			ResourcePath: "top-level-group-0-for-runners/1-runner-3",
			GroupID:      &group.Metadata.ID,
			CreatedBy:    "someone-sv3",
			Tags:         []string{"tag1", "tag2", "tag3"},
		})
		assert.Nil(t, err)

		t.Cleanup(func() {
			t.Log("Cleaning up subtest filter by tags, two tags")
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner0))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner1))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner2))
			assert.Nil(t, testClient.client.Runners.DeleteRunner(ctx, runner3))
			assert.Nil(t, testClient.client.Groups.DeleteGroup(ctx, group))
		})

		t.Log("Running subtest filter by tags, two tags")
		runnersResult, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			Filter: &RunnerFilter{
				TagFilter: &RunnerTagFilter{
					TagSubset: []string{"tag1", "tag2"},
				},
			},
		})
		assert.Nil(t, err)
		assert.Equal(t, 2, len(runnersResult.Runners))
		assert.Equal(t, runner2.Metadata.ID, runnersResult.Runners[0].Metadata.ID)
		assert.Equal(t, runner3.Metadata.ID, runnersResult.Runners[1].Metadata.ID)
	})
}

func TestCreateRunner(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups: standardWarmupGroupsForRunners,
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.Runner
		expectCreated *models.Runner
		expectMsg     *string
		name          string
	}

	now := time.Now()
	testCases := []testCase{
		{
			name: "positive, group runner",
			toCreate: &models.Runner{
				Type:            models.GroupRunnerType,
				Name:            "runner-create-test",
				GroupID:         &warmupItems.groups[0].Metadata.ID,
				CreatedBy:       "TestCreateRunner",
				Tags:            []string{"tag1", "tag2"},
				RunUntaggedJobs: false,
			},
			expectCreated: &models.Runner{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Type:            models.GroupRunnerType,
				Name:            "runner-create-test",
				ResourcePath:    warmupItems.groups[0].FullPath + "/runner-create-test",
				GroupID:         &warmupItems.groups[0].Metadata.ID,
				CreatedBy:       "TestCreateRunner",
				Tags:            []string{"tag1", "tag2"},
				RunUntaggedJobs: false,
			},
		},

		{
			name: "positive, shared runner",
			toCreate: &models.Runner{
				Type:      models.SharedRunnerType,
				Name:      "runner-create-test",
				CreatedBy: "TestCreateRunner",
			},
			expectCreated: &models.Runner{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Type:         models.SharedRunnerType,
				Name:         "runner-create-test",
				ResourcePath: "runner-create-test",
				CreatedBy:    "TestCreateRunner",
			},
		},

		{
			name: "duplicate group ID and runner name",
			toCreate: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "runner-create-test",
				ResourcePath: warmupItems.groups[0].FullPath + "/runner-create-test",
				GroupID:      &warmupItems.groups[0].Metadata.ID,
			},
			expectMsg: ptr.String("runner with name runner-create-test already exists in group"),
		},

		{
			name: "negative, non-existent group ID",
			toCreate: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "runner-create-test-non-existent-group-id",
				ResourcePath: warmupItems.groups[0].FullPath + "/runner-create-test-non-existent-group-id",
				GroupID:      ptr.String(nonExistentID),
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"runners\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "negative, invalid group ID",
			toCreate: &models.Runner{
				Type:         models.GroupRunnerType,
				Name:         "runner-create-test-invalid-group-id",
				ResourcePath: warmupItems.groups[0].FullPath + "/runner-create-test-invalid-group-id",
				GroupID:      ptr.String(invalidID),
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.Runners.CreateRunner(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareRunners(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestUpdateRunner(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups:  standardWarmupGroupsForRunners,
		runners: standardWarmupRunners,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg     *string
		toUpdate      *models.Runner
		expectUpdated *models.Runner
		name          string
	}

	// Looks up by ID and version.  Also requires group ID.
	// Updates private.
	positiveRunner := warmupItems.runners[0]
	positiveGroup := warmupItems.groups[9]

	now := time.Now()
	testCases := []testCase{

		{
			name: "positive",
			toUpdate: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:      positiveRunner.Metadata.ID,
					Version: initialResourceVersion,
				},
				Type:            models.GroupRunnerType,
				Name:            positiveRunner.Name,
				Description:     "Updated description",
				GroupID:         &positiveGroup.Metadata.ID,
				Tags:            []string{"tag1", "tag2"},
				RunUntaggedJobs: false,
			},
			expectUpdated: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:                   positiveRunner.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveRunner.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Type:            models.GroupRunnerType,
				Name:            positiveRunner.Name,
				Description:     "Updated description",
				ResourcePath:    positiveRunner.ResourcePath,
				GroupID:         positiveRunner.GroupID,
				CreatedBy:       positiveRunner.CreatedBy,
				Tags:            []string{"tag1", "tag2"},
				RunUntaggedJobs: false,
			},
		},

		{
			name: "negative, non-existent runner ID",
			toUpdate: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toUpdate: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualRunner, err := testClient.client.Runners.UpdateRunner(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				require.NotNil(t, actualRunner)
				compareRunners(t, test.expectUpdated, actualRunner, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualRunner)
			}
		})
	}
}

func TestDeleteRunner(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups:  standardWarmupGroupsForRunners,
		runners: standardWarmupRunners,
	})
	require.Nil(t, err)

	type testCase struct {
		expectMsg *string
		toDelete  *models.Runner
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.runners[0].Metadata.ID,
					Version: initialResourceVersion,
				},
			},
		},

		{
			name: "negative, non-existent runner ID",
			toDelete: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-ID",
			toDelete: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: initialResourceVersion,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.Runners.DeleteRunner(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this runner:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForRunners = []models.Group{
	// Top-level groups:
	{
		Description: "top level group 0 for testing runner functions",
		FullPath:    "top-level-group-0-for-runners",
		CreatedBy:   "someone-g0",
	},
	{
		Description: "top level group 1 for testing runner functions",
		FullPath:    "top-level-group-1-for-runners",
		CreatedBy:   "someone-g1",
	},
	{
		Description: "top level group 2 for testing runner functions",
		FullPath:    "top-level-group-2-for-runners",
		CreatedBy:   "someone-g2",
	},
	{
		Description: "top level group 3 for testing runner functions",
		FullPath:    "top-level-group-3-for-runners",
		CreatedBy:   "someone-g3",
	},
	{
		Description: "top level group 4 for testing runner functions",
		FullPath:    "top-level-group-4-for-runners",
		CreatedBy:   "someone-g4",
	},
	// Nested groups:
	{
		Description: "nested group 5 for testing runner functions",
		FullPath:    "top-level-group-4-for-runners/nested-group-5-for-runners",
		CreatedBy:   "someone-g5",
	},
	{
		Description: "nested group 6 for testing runner functions",
		FullPath:    "top-level-group-3-for-runners/nested-group-6-for-runners",
		CreatedBy:   "someone-g6",
	},
	{
		Description: "nested group 7 for testing runner functions",
		FullPath:    "top-level-group-2-for-runners/nested-group-7-for-runners",
		CreatedBy:   "someone-g7",
	},
	{
		Description: "nested group 8 for testing runner functions",
		FullPath:    "top-level-group-1-for-runners/nested-group-8-for-runners",
		CreatedBy:   "someone-g8",
	},
	{
		Description: "nested group 9 for testing runner functions",
		FullPath:    "top-level-group-0-for-runners/nested-group-9-for-runners",
		CreatedBy:   "someone-g9",
	},
}

// Standard warmup runners for tests in this runner:
// The ID fields will be replaced by the real IDs during the create function.
var standardWarmupRunners = []models.Runner{
	{
		// This one is public.
		Type:         models.GroupRunnerType,
		Name:         "1-runner-0",
		ResourcePath: "top-level-group-0-for-runners/1-runner-0",
		GroupID:      ptr.String("top-level-group-0-for-runners/nested-group-9-for-runners"),
		CreatedBy:    "someone-sv0",
	},
	{
		Type:         models.GroupRunnerType,
		Name:         "1-runner-1",
		ResourcePath: "top-level-group-1-for-runners/1-runner-1",
		GroupID:      ptr.String("top-level-group-1-for-runners"),
		CreatedBy:    "someone-sv1",
	},
	{
		Type:         models.GroupRunnerType,
		Name:         "2-runner-2",
		ResourcePath: "top-level-group-2-for-runners/2-runner-2",
		GroupID:      ptr.String("top-level-group-2-for-runners/nested-group-7-for-runners"),
		CreatedBy:    "someone-sv2",
	},
	{
		Type:         models.GroupRunnerType,
		Name:         "2-runner-3",
		ResourcePath: "top-level-group-3-for-runners/2-runner-3",
		GroupID:      ptr.String("top-level-group-3-for-runners"),
		CreatedBy:    "someone-sv3",
	},
	{
		Type:         models.GroupRunnerType,
		Name:         "5-runner-4",
		ResourcePath: "top-level-group-4-for-runners/5-runner-4",
		GroupID:      ptr.String("top-level-group-4-for-runners/nested-group-5-for-runners"),
		CreatedBy:    "someone-sv4",
	},
	{
		Type:         models.SharedRunnerType,
		Name:         "6-runner-shared",
		ResourcePath: "6-runner-shared",
		CreatedBy:    "someone-sv4",
	},
}

// createWarmupRunners creates some warmup runners for a test
// The warmup runners to create can be standard or otherwise.
func createWarmupRunners(ctx context.Context, testClient *testClient,
	input warmupRunners) (*warmupRunners, error) {

	// It is necessary to create several groups in order to provide the necessary IDs for the runners.

	// If doing get operations based on user ID or service account ID, it is necessary to create a bunch of other things.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultRunners, _, err := createInitialRunners(ctx, testClient,
		input.runners, parentPath2ID)
	if err != nil {
		return nil, err
	}

	return &warmupRunners{
		groups:  resultGroups,
		runners: resultRunners,
	}, nil
}

func ptrRunnerSortableField(arg RunnerSortableField) *RunnerSortableField {
	return &arg
}

func (wis runnerInfoIDSlice) Len() int {
	return len(wis)
}

func (wis runnerInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis runnerInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (wis runnerInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis runnerInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis runnerInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// runnerInfoFromRunners returns a slice of runnerInfo, not necessarily sorted in any order.
func runnerInfoFromRunners(runners []models.Runner) []runnerInfo {
	result := []runnerInfo{}

	for _, tp := range runners {
		result = append(result, runnerInfo{
			id:         tp.Metadata.ID,
			name:       tp.Name,
			updateTime: *tp.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// runnerIDsFromRunnerInfos preserves order
func runnerIDsFromRunnerInfos(runnerInfos []runnerInfo) []string {
	result := []string{}
	for _, runnerInfo := range runnerInfos {
		result = append(result, runnerInfo.id)
	}
	return result
}

// compareRunners compares two runner objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareRunners(t *testing.T, expected, actual *models.Runner,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.ResourcePath, actual.ResourcePath)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.Disabled, actual.Disabled)

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

// createInitialRunners creates some warmup runners for a test.
func createInitialRunners(ctx context.Context, testClient *testClient,
	toCreate []models.Runner, groupPath2ID map[string]string) (
	[]models.Runner, map[string]string, error) {
	result := []models.Runner{}
	resourcePath2ID := make(map[string]string)

	for _, input := range toCreate {
		if input.GroupID != nil {
			groupPath := input.GroupID
			groupID, ok := groupPath2ID[*input.GroupID]
			if !ok {
				return nil, nil,
					fmt.Errorf("createInitialRunners failed to look up group path: %s", *groupPath)
			}
			input.GroupID = &groupID
		}

		created, err := testClient.client.Runners.CreateRunner(ctx, &input)
		if err != nil {
			return nil, nil, err
		}

		result = append(result, *created)
		resourcePath2ID[created.ResourcePath] = created.Metadata.ID
	}

	return result, resourcePath2ID, nil
}
