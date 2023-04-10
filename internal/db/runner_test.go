//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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

	warmupItems, err := createWarmupRunners(ctx, testClient, warmupRunners{
		groups:  standardWarmupGroupsForRunners,
		runners: standardWarmupRunners,
	})
	require.Nil(t, err)

	allRunnerInfos := runnerInfoFromRunners(warmupItems.runners)

	// Sort by runner IDs.
	sort.Sort(runnerInfoIDSlice(allRunnerInfos))
	allRunnerIDs := runnerIDsFromRunnerInfos(allRunnerInfos)

	// Sort by last update times.
	sort.Sort(runnerInfoUpdateSlice(allRunnerInfos))
	allRunnerIDsByTime := runnerIDsFromRunnerInfos(allRunnerInfos)
	reverseRunnerIDsByTime := reverseStringSlice(allRunnerIDsByTime)

	type testCase struct {
		input           *GetRunnersInput
		expectMsg       *string
		name            string
		expectRunnerIDs []string
	}

	testCases := []testCase{
		{
			name: "non-nil but mostly empty input",
			input: &GetRunnersInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectRunnerIDs: allRunnerIDs,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectRunnerIDs: allRunnerIDsByTime,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
			},
			expectRunnerIDs: allRunnerIDsByTime,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtDesc),
			},
			expectRunnerIDs: reverseRunnerIDsByTime,
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:       ptr.String("only first or last can be defined, not both"),
			expectRunnerIDs: allRunnerIDs[4:],
		},

		{
			name: "filter, group ID, positive",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				Filter: &RunnerFilter{
					GroupID: warmupItems.runners[0].GroupID,
				},
			},
			expectRunnerIDs: allRunnerIDsByTime[0:1],
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				Filter: &RunnerFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectRunnerIDs: []string{},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				Filter: &RunnerFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:       invalidUUIDMsg2,
			expectRunnerIDs: []string{},
		},

		{
			name: "filter, runner IDs, positive",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				Filter: &RunnerFilter{
					RunnerIDs: []string{
						allRunnerIDsByTime[0], allRunnerIDsByTime[1], allRunnerIDsByTime[3]},
				},
			},
			expectRunnerIDs: []string{
				allRunnerIDsByTime[0], allRunnerIDsByTime[1], allRunnerIDsByTime[3],
			},
		},

		{
			name: "filter, runner IDs, non-existent",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				Filter: &RunnerFilter{
					RunnerIDs: []string{nonExistentID},
				},
			},
			expectRunnerIDs: []string{},
		},

		{
			name: "filter, runner IDs, invalid ID",
			input: &GetRunnersInput{
				Sort: ptrRunnerSortableField(RunnerSortableFieldUpdatedAtAsc),
				Filter: &RunnerFilter{
					RunnerIDs: []string{invalidID},
				},
			},
			expectMsg:       invalidUUIDMsg2,
			expectRunnerIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			runnersResult, err := testClient.client.Runners.GetRunners(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if err == nil {
				// Never returns nil if error is nil.
				require.NotNil(t, runnersResult.PageInfo)

				runners := runnersResult.Runners

				// Check the runners result by comparing a list of the runner IDs.
				actualRunnerIDs := []string{}
				for _, runner := range runners {
					actualRunnerIDs = append(actualRunnerIDs, runner.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualRunnerIDs)
				}

				assert.Equal(t, len(test.expectRunnerIDs), len(actualRunnerIDs))
				assert.Equal(t, test.expectRunnerIDs, actualRunnerIDs)
			}
		})
	}
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
			name: "positive",
			toCreate: &models.Runner{
				Type:      models.GroupRunnerType,
				Name:      "runner-create-test",
				GroupID:   &warmupItems.groups[0].Metadata.ID,
				CreatedBy: "TestCreateRunner",
			},
			expectCreated: &models.Runner{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Type:         models.GroupRunnerType,
				Name:         "runner-create-test",
				ResourcePath: warmupItems.groups[0].FullPath + "/runner-create-test",
				GroupID:      &warmupItems.groups[0].Metadata.ID,
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
				Type:        models.GroupRunnerType,
				Name:        positiveRunner.Name,
				Description: "Updated description",
				GroupID:     &positiveGroup.Metadata.ID,
			},
			expectUpdated: &models.Runner{
				Metadata: models.ResourceMetadata{
					ID:                   positiveRunner.Metadata.ID,
					Version:              initialResourceVersion + 1,
					CreationTimestamp:    positiveRunner.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Type:         models.GroupRunnerType,
				Name:         positiveRunner.Name,
				Description:  "Updated description",
				ResourcePath: positiveRunner.ResourcePath,
				GroupID:      positiveRunner.GroupID,
				CreatedBy:    positiveRunner.CreatedBy,
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

// The End.
