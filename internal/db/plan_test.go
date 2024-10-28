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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// planInfo aids convenience in accessing the information TestGetPlans needs about the warmup plans.
type planInfo struct {
	updateTime time.Time
	planID     string
}

// planInfoIDSlice makes a slice of planInfo sortable by ID string
type planInfoIDSlice []planInfo

// planInfoUpdateSlice makes a slice of planInfo sortable by last updated time
type planInfoUpdateSlice []planInfo

func TestGetPlan(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a plan with a specific ID without going into the really
	// low-level stuff, create the warmup plan(s) then find the relevant ID.
	createdLow := currentTime()
	_, createdWarmupPlans, err := createWarmupPlans(ctx, testClient, standardWarmupGroupsForPlans,
		standardWarmupWorkspacesForPlans, standardWarmupPlans)
	require.Nil(t, err)
	createdHigh := currentTime()

	type testCase struct {
		expectPlan *models.Plan
		expectMsg  *string
		name       string
		searchID   string
	}

	// Do only one positive test case, because the logic is theoretically the same for all plans.
	positivePlan := createdWarmupPlans[0]
	testCases := []testCase{
		{
			name:       "positive",
			searchID:   positivePlan.Metadata.ID,
			expectPlan: &positivePlan,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect plan and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			plan, err := testClient.client.Plans.GetPlan(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectPlan != nil {
				require.NotNil(t, plan)
				comparePlans(t, test.expectPlan, plan, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, plan)
			}
		})
	}
}

func TestGetPlans(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, warmupPlans, err := createWarmupPlans(ctx, testClient,
		standardWarmupGroupsForPlans, standardWarmupWorkspacesForPlans, standardWarmupPlans)
	require.Nil(t, err)
	allPlanInfos := planInfoFromPlans(warmupPlans)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(planInfoIDSlice(allPlanInfos))
	allPlanIDs := planIDsFromPlanInfos(allPlanInfos)

	// Sort by last update times.
	sort.Sort(planInfoUpdateSlice(allPlanInfos))
	allPlanIDsByUpdateTime := planIDsFromPlanInfos(allPlanInfos)
	reversePlanIDsByUpdateTime := reverseStringSlice(allPlanIDsByUpdateTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetPlansInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectPlanIDs               []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetPlansInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectPlanIDs               []string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{
		// nil input causes a nil pointer dereference in GetPlans, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetPlansInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectPlanIDs:        allPlanIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPlanIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectPlanIDs:        allPlanIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPlanIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtDesc),
			},
			expectPlanIDs:        reversePlanIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPlanIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectPlanIDs:        allPlanIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPlanIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectPlanIDs: allPlanIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allPlanIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectPlanIDs:              allPlanIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allPlanIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectPlanIDs:              allPlanIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allPlanIDs)),
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
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectPlanIDs: reversePlanIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allPlanIDs)),
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
			input: &GetPlansInput{
				Sort:              ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectPlanIDs:               []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allPlanIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// The plan filter has only PlanIDs, so there's no way to allow nothing through the filters.
		// Passing an empty slice to PlanIDs causes an SQL syntax error ("... IN ()"), so don't try it.

		{
			name: "filter, plan IDs",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				Filter: &PlanFilter{
					PlanIDs: []string{allPlanIDsByUpdateTime[0], allPlanIDsByUpdateTime[2], allPlanIDsByUpdateTime[4]},
				},
			},
			expectPlanIDs:        []string{allPlanIDsByUpdateTime[0], allPlanIDsByUpdateTime[2], allPlanIDsByUpdateTime[4]},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, plan IDs, non-existent",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				Filter: &PlanFilter{
					PlanIDs: []string{nonExistentID},
				},
			},
			expectPlanIDs:        []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, plan IDs, invalid",
			input: &GetPlansInput{
				Sort: ptrPlanSortableField(PlanSortableFieldUpdatedAtAsc),
				Filter: &PlanFilter{
					PlanIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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

			plansActual, err := testClient.client.Plans.GetPlans(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, plansActual.PageInfo)
				assert.NotNil(t, plansActual.Plans)
				pageInfo := plansActual.PageInfo
				plans := plansActual.Plans

				// Check the plans result by comparing a list of the plan IDs.
				actualPlanIDs := []string{}
				for _, plan := range plans {
					actualPlanIDs = append(actualPlanIDs, plan.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualPlanIDs)
				}

				assert.Equal(t, len(test.expectPlanIDs), len(actualPlanIDs))
				assert.Equal(t, test.expectPlanIDs, actualPlanIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one plan returned.
				// If there are no plans returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(plans) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&plans[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&plans[len(plans)-1])
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

func TestCreatePlan(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, _, err := createWarmupPlans(ctx, testClient,
		standardWarmupGroupsForPlans, standardWarmupWorkspacesForPlans, standardWarmupPlans)
	require.Nil(t, err)
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toCreate      *models.Plan
		expectCreated *models.Plan
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.Plan{
				WorkspaceID: warmupWorkspaceID,
			},
			expectCreated: &models.Plan{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
			},
		},

		{
			name: "positive full",
			toCreate: &models.Plan{
				WorkspaceID: warmupWorkspaceID,
				Status:      models.PlanFinished,
				HasChanges:  true,
			},
			expectCreated: &models.Plan{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
				Status:      models.PlanFinished,
				HasChanges:  true,
			},
		},

		// It does not make sense to try to create a duplicate plan,
		// because there is no unique name field to trigger an error.

		{
			name: "non-existent workspace ID",
			toCreate: &models.Plan{
				WorkspaceID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"plans\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.Plan{
				WorkspaceID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.Plans.CreatePlan(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				comparePlans(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdatePlan(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a plan with a specific ID without going into the really
	// low-level stuff, create the warmup plan(s) and then find the relevant ID.
	createdLow := currentTime()
	warmupWorkspaces, warmupPlans, err := createWarmupPlans(ctx, testClient,
		standardWarmupGroupsForPlans, standardWarmupWorkspacesForPlans, standardWarmupPlans)
	require.Nil(t, err)
	createdHigh := currentTime()
	warmupWorkspaceID := warmupWorkspaces[0].Metadata.ID

	type testCase struct {
		toUpdate   *models.Plan
		expectPlan *models.Plan
		expectMsg  *string
		name       string
	}

	// Do only one positive test case, because the logic is theoretically the same for all plans.
	now := currentTime()
	positivePlan := warmupPlans[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID:      positivePlan.Metadata.ID,
					Version: positivePlan.Metadata.Version,
				},
				WorkspaceID: warmupWorkspaceID,
				Status:      models.PlanFinished,
				HasChanges:  true,
			},
			expectPlan: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID:                   positivePlan.Metadata.ID,
					Version:              positivePlan.Metadata.Version + 1,
					CreationTimestamp:    positivePlan.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaceID,
				Status:      models.PlanFinished,
				HasChanges:  true,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positivePlan.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.Plan{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positivePlan.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			plan, err := testClient.client.Plans.UpdatePlan(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectPlan != nil {
				require.NotNil(t, plan)
				now := currentTime()
				comparePlans(t, test.expectPlan, plan, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  test.expectPlan.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, plan)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForPlans = []models.Group{
	{
		Description: "top level group 0 for testing plan functions",
		FullPath:    "top-level-group-0-for-plans",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForPlans = []models.Workspace{
	{
		Description: "workspace 0 for testing plan functions",
		FullPath:    "top-level-group-0-for-plans/workspace-0-for-plans",
		CreatedBy:   "someone-w0",
	},
}

// Standard warmup plan(s) for tests in this module:
var standardWarmupPlans = []models.Plan{
	{
		HasChanges: true,
	},
	{
		HasChanges: true,
	},
	{
		HasChanges: true,
	},
	{
		HasChanges: true,
	},
	{
		HasChanges: true,
	},
}

// createWarmupPlans creates some warmup plans for a test
// The warmup plans to create can be standard or otherwise.
func createWarmupPlans(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newPlans []models.Plan) (
	[]models.Workspace,
	[]models.Plan,
	error,
) {
	// It is necessary to create at least one group and workspace in order to provide the necessary IDs for the plans.

	_, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, err
	}
	workspaceID := resultWorkspaces[0].Metadata.ID

	resultPlans, err := createInitialPlans(ctx, testClient, newPlans, workspaceID)
	if err != nil {
		return nil, nil, err
	}

	return resultWorkspaces, resultPlans, nil
}

func ptrPlanSortableField(arg PlanSortableField) *PlanSortableField {
	return &arg
}

func (wis planInfoIDSlice) Len() int {
	return len(wis)
}

func (wis planInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis planInfoIDSlice) Less(i, j int) bool {
	return wis[i].planID < wis[j].planID
}

func (wis planInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis planInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis planInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// planInfoFromPlans returns a slice of planInfo, not necessarily sorted in any order.
func planInfoFromPlans(plans []models.Plan) []planInfo {
	result := []planInfo{}

	for _, plan := range plans {
		result = append(result, planInfo{
			planID:     plan.Metadata.ID,
			updateTime: *plan.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// planIDsFromPlanInfos preserves order
func planIDsFromPlanInfos(planInfos []planInfo) []string {
	result := []string{}
	for _, planInfo := range planInfos {
		result = append(result, planInfo.planID)
	}
	return result
}

// comparePlans compares two plan objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func comparePlans(t *testing.T, expected, actual *models.Plan,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.HasChanges, actual.HasChanges)
	assert.Equal(t, expected.Summary.ResourceAdditions, actual.Summary.ResourceAdditions)
	assert.Equal(t, expected.Summary.ResourceChanges, actual.Summary.ResourceChanges)
	assert.Equal(t, expected.Summary.ResourceDestructions, actual.Summary.ResourceDestructions)

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
