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
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// runInfo aids convenience in accessing the information TestGetRuns needs about the warmup runs.
type runInfo struct {
	createTime time.Time
	updateTime time.Time
	runID      string
}

// runInfoIDSlice makes a slice of runInfo sortable by ID string
type runInfoIDSlice []runInfo

// runInfoCreateSlice makes a slice of runInfo sortable by creation time
type runInfoCreateSlice []runInfo

// runInfoUpdateSlice makes a slice of runInfo sortable by last updated time
type runInfoUpdateSlice []runInfo

func TestGetRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a run with a specific ID without going into the really
	// low-level stuff, create the warmup run(s) by name and then find the relevant ID.
	createdLow := currentTime()
	_, _, createdWarmupRuns, _, _, err := createWarmupRuns(ctx, testClient, standardWarmupGroupsForRuns,
		standardWarmupWorkspacesForRuns, standardWarmupRuns, nil, nil, false)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectRun *models.Run
		expectMsg *string
		name      string
		searchID  string
	}

	// Run only one positive test case, because the logic is theoretically the same for all runs.
	positiveRun := createdWarmupRuns[0]
	testCases := []testCase{
		{
			name:      "positive-" + positiveRun.Comment,
			searchID:  positiveRun.Metadata.ID,
			expectRun: &positiveRun,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect run and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.GetRun(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectRun != nil {
				require.NotNil(t, run)
				compareRuns(t, test.expectRun, run, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestGetRunByPlanID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a plan with a specific ID without going into the really
	// low-level stuff, create the warmup run(s) by name and then find the relevant ID.
	createdLow := currentTime()
	_, _, createdWarmupRuns, createdWarmupPlans, _, err := createWarmupRuns(ctx, testClient,
		standardWarmupGroupsForRuns, standardWarmupWorkspacesForRuns, standardWarmupRuns,
		standardWarmupPlansForRuns, standardWarmupAppliesForRuns, true)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectRun *models.Run
		expectMsg *string
		name      string
		searchID  string
	}

	// Run only one positive test case, because the logic is theoretically the same for all runs.
	testCases := []testCase{
		{
			name:      "positive",
			searchID:  createdWarmupPlans[0].Metadata.ID,
			expectRun: &createdWarmupRuns[1],
		},
		{
			name:      "negative, non-existent ID",
			searchID:  nonExistentID,
			expectMsg: ptr.String("Failed to get run for plan"),
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String("Failed to get run for plan: Failed to scan query count result: " + *invalidUUIDMsg1),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualRun, err := testClient.client.Runs.GetRunByPlanID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectRun != nil {
				require.NotNil(t, actualRun)
				compareRuns(t, test.expectRun, actualRun, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualRun)
			}
		})
	}
}

func TestGetRunByApplyID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a apply with a specific ID without going into the really
	// low-level stuff, create the warmup run(s) by name and then find the relevant ID.
	createdLow := currentTime()
	_, _, createdWarmupRuns, _, createdWarmupApplies, err := createWarmupRuns(ctx, testClient,
		standardWarmupGroupsForRuns, standardWarmupWorkspacesForRuns, standardWarmupRuns,
		standardWarmupPlansForRuns, standardWarmupAppliesForRuns, true)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectRun *models.Run
		expectMsg *string
		name      string
		searchID  string
	}

	// Run only one positive test case, because the logic is theoretically the same for all runs.
	testCases := []testCase{
		{
			name:      "positive",
			searchID:  createdWarmupApplies[0].Metadata.ID,
			expectRun: &createdWarmupRuns[2],
		},
		{
			name:      "negative, non-existent ID",
			searchID:  nonExistentID,
			expectMsg: ptr.String("Failed to get run for apply"),
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String("Failed to get run for apply: Failed to scan query count result: " + *invalidUUIDMsg1),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualRun, err := testClient.client.Runs.GetRunByApplyID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectRun != nil {
				require.NotNil(t, actualRun)
				compareRuns(t, test.expectRun, actualRun, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualRun)
			}
		})
	}
}

func TestCreateRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupWorkspaces, _, warmupPlans, warmupApplies, err := createWarmupRuns(ctx, testClient,
		standardWarmupGroupsForRuns, standardWarmupWorkspacesForRuns, nil,
		standardWarmupPlansForRuns, standardWarmupAppliesForRuns, false)
	require.Nil(t, err)
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toCreate      *models.Run
		expectCreated *models.Run
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.Run{
				WorkspaceID: warmupWorkspaceID,
			},
			expectCreated: &models.Run{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
			},
		},

		{
			// The PlanID and ApplyID fields are not normally set when creating a run, but do it here anyway.
			// Don't bother setting a configuration version ID.
			// Don't try to set the AutoApply field, because it's hard-set to false.
			// Don't bother setting ForceCancelAvailableAt, because comparisons don't work well.
			name: "positive full",
			toCreate: &models.Run{
				Status:          models.RunPlannedAndFinished,
				IsDestroy:       true,
				HasChanges:      true,
				WorkspaceID:     warmupWorkspaceID,
				PlanID:          warmupPlans[0].Metadata.ID,
				ApplyID:         warmupApplies[0].Metadata.ID,
				CreatedBy:       "function TestCreateRun",
				ModuleSource:    ptr.String("some module source"),
				ModuleVersion:   ptr.String("some module version"),
				ForceCanceledBy: ptr.String("some force canceller"),
				ForceCanceled:   true,
				Comment:         "the positive full run from TestCreateRun",
			},
			expectCreated: &models.Run{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Status:          models.RunPlannedAndFinished,
				IsDestroy:       true,
				HasChanges:      true,
				WorkspaceID:     warmupWorkspaceID,
				PlanID:          warmupPlans[0].Metadata.ID,
				ApplyID:         warmupApplies[0].Metadata.ID,
				CreatedBy:       "function TestCreateRun",
				ModuleSource:    ptr.String("some module source"),
				ModuleVersion:   ptr.String("some module version"),
				ForceCanceledBy: ptr.String("some force canceller"),
				ForceCanceled:   true,
				Comment:         "the positive full run from TestCreateRun",
			},
		},

		// It does not make sense to try to create a duplicate run, because there is no unique name field to trigger an error.

		// It might be possible to test creating a run with (combinations of) non-existent or invalid plan,
		// apply, and workspace IDs, but that is not (yet) done.

		{
			name: "non-existent workspace ID",
			toCreate: &models.Run{
				WorkspaceID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"runs\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.Run{
				WorkspaceID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.Runs.CreateRun(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareRuns(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a run with a specific ID without going into the really
	// low-level stuff, create the warmup run(s) by name and then find the relevant ID.
	createdLow := currentTime()
	_, warmupWorkspaces, warmupRuns, warmupPlans, warmupApplies, err := createWarmupRuns(ctx, testClient,
		standardWarmupGroupsForRuns, standardWarmupWorkspacesForRuns, standardWarmupRuns,
		standardWarmupPlansForRuns, standardWarmupAppliesForRuns, false)
	require.Nil(t, err)
	createdHigh := currentTime()
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toUpdate  *models.Run
		expectRun *models.Run
		expectMsg *string
		name      string
	}

	// Run only one positive test case, because the logic is theoretically the same for all runs.
	now := currentTime()
	positiveRun := warmupRuns[0]
	testCases := []testCase{
		{
			name: "positive-" + positiveRun.Comment,
			toUpdate: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:      positiveRun.Metadata.ID,
					Version: positiveRun.Metadata.Version,
				},
				Status:          models.RunPlannedAndFinished,
				IsDestroy:       true,
				HasChanges:      true,
				WorkspaceID:     warmupWorkspaceID,
				PlanID:          warmupPlans[0].Metadata.ID,
				ApplyID:         warmupApplies[0].Metadata.ID,
				ModuleSource:    ptr.String("updated module source"),
				ModuleVersion:   ptr.String("updated module version"),
				ForceCanceledBy: ptr.String("updated force canceller"),
				ForceCanceled:   true,
			},
			expectRun: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:                   positiveRun.Metadata.ID,
					Version:              positiveRun.Metadata.Version + 1,
					CreationTimestamp:    positiveRun.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Status:          models.RunPlannedAndFinished,
				IsDestroy:       true,
				HasChanges:      true,
				WorkspaceID:     warmupWorkspaceID,
				PlanID:          warmupPlans[0].Metadata.ID,
				ApplyID:         warmupApplies[0].Metadata.ID,
				CreatedBy:       positiveRun.CreatedBy, // cannot be updated
				ModuleSource:    ptr.String("updated module source"),
				ModuleVersion:   ptr.String("updated module version"),
				ForceCanceledBy: ptr.String("updated force canceller"),
				ForceCanceled:   true,
				Comment:         positiveRun.Comment, // cannot be updated
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveRun.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.Run{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveRun.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.UpdateRun(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectRun != nil {
				require.NotNil(t, run)
				now := currentTime()
				compareRuns(t, test.expectRun, run, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  test.expectRun.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestGetRuns(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupGroups, warmupWorkspaces, warmupRuns, warmupPlans, warmupApplies, err := createWarmupRuns(ctx, testClient,
		standardWarmupGroupsForRuns, standardWarmupWorkspacesForRuns, standardWarmupRuns,
		standardWarmupPlansForRuns, standardWarmupAppliesForRuns, true)
	require.Nil(t, err)
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID
	warmupGroupID := warmupGroups[0].Metadata.ID
	allRunInfos := runInfoFromRuns(warmupRuns)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(runInfoIDSlice(allRunInfos))
	allRunIDs := runIDsFromRunInfos(allRunInfos)

	// Sort by creation times.
	sort.Sort(runInfoCreateSlice(allRunInfos))
	allRunIDsByCreationTime := runIDsFromRunInfos(allRunInfos)
	reverseRunIDsByCreationTime := reverseStringSlice(allRunIDsByCreationTime)

	// Sort by last update times.
	sort.Sort(runInfoUpdateSlice(allRunInfos))
	allRunIDsByUpdateTime := runIDsFromRunInfos(allRunInfos)
	reverseRunIDsByUpdateTime := reverseStringSlice(allRunIDsByUpdateTime)

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetRunsInput
		name                        string
		expectPageInfo              PageInfo
		expectRunIDs                []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetRunsInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectRunIDs                []string
		expectPageInfo              PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{
		// nil input causes a nil pointer dereference in GetRuns, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetRunsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectRunIDs:         allRunIDs,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectRunIDs:         allRunIDsByCreationTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of creation time",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtDesc),
			},
			expectRunIDs:         reverseRunIDsByCreationTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldUpdatedAtAsc),
			},
			expectRunIDs:         allRunIDsByUpdateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldUpdatedAtDesc),
			},
			expectRunIDs:         reverseRunIDsByUpdateTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectRunIDs:         allRunIDsByCreationTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectRunIDs: allRunIDsByCreationTime[:2],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allRunIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle one",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
				},
			},
			getAfterCursorFromPrevious: true,
			expectRunIDs:               allRunIDsByCreationTime[2:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allRunIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final two",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectRunIDs:               allRunIDsByCreationTime[3:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allRunIDs)),
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
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectRunIDs: reverseRunIDsByCreationTime[:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allRunIDs)),
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
			input: &GetRunsInput{
				Sort:              ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectRunIDs:                []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allRunIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &RunFilter{
					// Passing an empty slice to RunIDs causes an SQL syntax error ("... IN ()"), so don't try it.
					// RunIDs: []string{},
					PlanID:      ptr.String(""),
					ApplyID:     ptr.String(""),
					WorkspaceID: ptr.String(""),
					GroupID:     ptr.String(""),
				},
			},
			expectMsg:      emptyUUIDMsg2,
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, run IDs",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					RunIDs: []string{allRunIDsByCreationTime[0], allRunIDsByCreationTime[2], allRunIDsByCreationTime[4]},
				},
			},
			expectRunIDs:         []string{allRunIDsByCreationTime[0], allRunIDsByCreationTime[2], allRunIDsByCreationTime[4]},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, run IDs, non-existent",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					RunIDs: []string{nonExistentID},
				},
			},
			expectRunIDs:         []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, run IDs, invalid",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					RunIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, plan ID, positive",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					PlanID: ptr.String(warmupPlans[0].Metadata.ID),
				},
			},
			expectRunIDs:         []string{allRunIDsByCreationTime[1]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, plan ID, non-existent",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					PlanID: ptr.String(nonExistentID),
				},
			},
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, plan ID, invalid",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					PlanID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, apply ID, positive",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					ApplyID: ptr.String(warmupApplies[0].Metadata.ID),
				},
			},
			expectRunIDs:         []string{allRunIDsByCreationTime[2]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, apply ID, non-existent",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					ApplyID: ptr.String(nonExistentID),
				},
			},
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, apply ID, invalid",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					ApplyID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, workspace ID, positive",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					WorkspaceID: ptr.String(warmupWorkspaceID),
				},
			},
			expectRunIDs:         allRunIDsByCreationTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, non-existent",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					WorkspaceID: ptr.String(nonExistentID),
				},
			},
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, workspace ID, invalid",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					WorkspaceID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, group ID, positive",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					GroupID: ptr.String(warmupGroupID),
				},
			},
			expectRunIDs:         allRunIDsByCreationTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allRunIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetRunsInput{
				Sort: ptrRunSortableField(RunSortableFieldCreatedAtAsc),
				Filter: &RunFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectRunIDs:   []string{},
			expectPageInfo: PageInfo{},
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

			runsActual, err := testClient.client.Runs.GetRuns(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, runsActual.PageInfo)
				assert.NotNil(t, runsActual.Runs)
				pageInfo := runsActual.PageInfo
				runs := runsActual.Runs

				// Check the runs result by comparing a list of the run IDs.
				actualRunIDs := []string{}
				for _, run := range runs {
					actualRunIDs = append(actualRunIDs, run.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualRunIDs)
				}

				assert.Equal(t, len(test.expectRunIDs), len(actualRunIDs))
				assert.Equal(t, test.expectRunIDs, actualRunIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one run returned.
				// If there are no runs returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(runs) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&runs[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&runs[len(runs)-1])
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

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForRuns = []models.Group{
	{
		Description: "top level group 0 for testing run functions",
		FullPath:    "top-level-group-0-for-runs",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForRuns = []models.Workspace{
	{
		Description: "workspace 0 for testing run functions",
		FullPath:    "top-level-group-0-for-runs/workspace-0-for-runs",
		CreatedBy:   "someone-w0",
	},
}

// Standard warmup runs for tests in this module:
var standardWarmupRuns = []models.Run{
	{
		CreatedBy: "someone-0",
		Comment:   "run 0 for testing run functions",
	},
	{
		CreatedBy: "someone-1",
		Comment:   "run 1 for testing run functions",
	},
	{
		CreatedBy: "someone-2",
		Comment:   "run 2 for testing run functions",
	},
	{
		CreatedBy: "someone-3",
		Comment:   "run 3 for testing run functions",
	},
	{
		CreatedBy: "someone-4",
		Comment:   "run 4 for testing run functions",
	},
}

// Standard warmup plan(s) for tests in this module:
var standardWarmupPlansForRuns = []models.Plan{
	{
		HasChanges:        true,
		ResourceAdditions: 10,
	},
	{
		HasChanges:        true,
		ResourceAdditions: 11,
	},
}

// Standard warmup apply/applies for tests in this module:
var standardWarmupAppliesForRuns = []models.Apply{
	{
		Comment: "standard warmup apply for runs: 0",
	},
	{
		Comment: "standard warmup apply for runs: 1",
	},
}

// createWarmupRuns creates some warmup runs for a test
// The warmup runs to create can be standard or otherwise.
func createWarmupRuns(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newRuns []models.Run,
	newPlans []models.Plan,
	newApplies []models.Apply,
	connectPlansApplies bool) (
	[]models.Group,
	[]models.Workspace,
	[]models.Run,
	[]models.Plan,
	[]models.Apply,
	error,
) {
	// It is necessary to create at least one group, workspace, plan, and apply
	// in order to provide the necessary IDs for the runs.

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	workspaceID := resultWorkspaces[0].Metadata.ID

	resultRuns, err := createInitialRuns(ctx, testClient, newRuns, workspaceID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	resultPlans, err := createInitialPlans(ctx, testClient, newPlans, workspaceID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	resultApplies, err := createInitialApplies(ctx, testClient, newApplies, workspaceID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// If enabled, connect 1 plans and 1 applies to 2 of 3 (standard) runs.
	if connectPlansApplies {
		resultRuns[1].PlanID = resultPlans[0].Metadata.ID
		resultRuns[2].ApplyID = resultApplies[0].Metadata.ID
		resultRuns[3].PlanID = resultPlans[1].Metadata.ID
		resultRuns[3].ApplyID = resultApplies[1].Metadata.ID

		// Make the plan and apply assignments persistent.
		for ix, runObj := range resultRuns {

			updatedRun, err := testClient.client.Runs.UpdateRun(ctx, &runObj)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}

			resultRuns[ix] = *updatedRun
		}

	}

	return resultGroups, resultWorkspaces, resultRuns, resultPlans, resultApplies, nil
}

func ptrRunSortableField(arg RunSortableField) *RunSortableField {
	return &arg
}

func (wis runInfoIDSlice) Len() int {
	return len(wis)
}

func (wis runInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis runInfoIDSlice) Less(i, j int) bool {
	return wis[i].runID < wis[j].runID
}

func (wis runInfoCreateSlice) Len() int {
	return len(wis)
}

func (wis runInfoCreateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis runInfoCreateSlice) Less(i, j int) bool {
	return wis[i].createTime.Before(wis[j].createTime)
}

func (wis runInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis runInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis runInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// runInfoFromRuns returns a slice of runInfo, not necessarily sorted in any order.
func runInfoFromRuns(runs []models.Run) []runInfo {
	result := []runInfo{}

	for _, run := range runs {
		result = append(result, runInfo{
			runID:      run.Metadata.ID,
			createTime: *run.Metadata.CreationTimestamp,
			updateTime: *run.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// runIDsFromRunInfos preserves order
func runIDsFromRunInfos(runInfos []runInfo) []string {
	result := []string{}
	for _, runInfo := range runInfos {
		result = append(result, runInfo.runID)
	}
	return result
}

// compareRuns compares two run objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareRuns(t *testing.T, expected, actual *models.Run,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.IsDestroy, actual.IsDestroy)
	assert.Equal(t, expected.HasChanges, actual.HasChanges)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.ConfigurationVersionID, actual.ConfigurationVersionID)
	assert.Equal(t, expected.PlanID, actual.PlanID)
	assert.Equal(t, expected.ApplyID, actual.ApplyID)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.ModuleSource, actual.ModuleSource)
	assert.Equal(t, expected.ModuleVersion, actual.ModuleVersion)
	assert.Equal(t, expected.ForceCanceledBy, actual.ForceCanceledBy)
	assert.Equal(t, expected.ForceCancelAvailableAt, actual.ForceCancelAvailableAt)
	assert.Equal(t, expected.ForceCanceled, actual.ForceCanceled)
	assert.Equal(t, expected.Comment, actual.Comment)
	assert.Equal(t, expected.AutoApply, actual.AutoApply)

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
