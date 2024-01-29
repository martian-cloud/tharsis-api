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

// stateVersionInfo aids convenience in accessing the information
// TestGetStateVersions needs about the warmup state versions.
type stateVersionInfo struct {
	updateTime     time.Time
	stateVersionID string
}

// stateVersionInfoIDSlice makes a slice of stateVersionInfo sortable by ID string
type stateVersionInfoIDSlice []stateVersionInfo

// stateVersionInfoUpdateSlice makes a slice of stateVersionInfo sortable by last updated time
type stateVersionInfoUpdateSlice []stateVersionInfo

func TestGetStateVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, _, warmupStateVersions, err := createWarmupStateVersions(ctx, testClient,
		standardWarmupGroupsForStateVersions, standardWarmupWorkspacesForStateVersions,
		standardWarmupRunsForStateVersions, standardWarmupStateVersions)
	require.Nil(t, err)
	allStateVersionInfos := stateVersionInfoFromStateVersions(warmupStateVersions)

	// Sort by state version IDs.
	sort.Sort(stateVersionInfoIDSlice(allStateVersionInfos))
	allStateVersionIDs := stateVersionIDsFromStateVersionInfos(allStateVersionInfos)

	// Sort by last update times.
	sort.Sort(stateVersionInfoUpdateSlice(allStateVersionInfos))
	allStateVersionIDsByTime := stateVersionIDsFromStateVersionInfos(allStateVersionInfos)
	reverseStateVersionIDsByTime := reverseStringSlice(allStateVersionIDsByTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		input                       *GetStateVersionsInput
		expectMsg                   *string
		name                        string
		expectPageInfo              pagination.PageInfo
		expectStateVersionIDs       []string
		getBeforeCursorFromPrevious bool
		sortedDescending            bool
		expectHasStartCursor        bool
		getAfterCursorFromPrevious  bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
			name: "",
			input: &GetStateVersionsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectStateVersionIDs:       []string{},
			expectPageInfo: pagination.PageInfo{
				Cursor:          nil,
				TotalCount:      0,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectStartCursorError: nil,
			expectHasStartCursor:   false,
			expectEndCursorError:   nil,
			expectHasEndCursor:     false,
		}
	*/

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetStateVersions, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetStateVersionsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectStateVersionIDs: allStateVersionIDs,
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(len(allStateVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectStateVersionIDs: allStateVersionIDsByTime,
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(len(allStateVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
			},
			expectStateVersionIDs: allStateVersionIDsByTime,
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(len(allStateVersionIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtDesc),
			},
			sortedDescending:      true,
			expectStateVersionIDs: reverseStateVersionIDsByTime,
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(len(allStateVersionIDsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "pagination: everything at once",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectStateVersionIDs: allStateVersionIDsByTime,
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(len(allStateVersionIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "pagination: first two",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectStateVersionIDs: allStateVersionIDsByTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allStateVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectStateVersionIDs:      allStateVersionIDsByTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allStateVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectStateVersionIDs:      allStateVersionIDsByTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allStateVersionIDs)),
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
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:      true,
			expectStateVersionIDs: reverseStateVersionIDsByTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allStateVersionIDs)),
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
			input: &GetStateVersionsInput{
				Sort:              ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectStateVersionIDs:       []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:             ptr.String("only first or last can be defined, not both"),
			expectStateVersionIDs: allStateVersionIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allStateVersionIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &StateVersionFilter{
					WorkspaceID: ptr.String(""),
					// Passing an empty slice to StateVersionIDs causes an SQL syntax error ("... IN ()"), so don't try it.
					// StateVersionIDs: []string{},
				},
			},
			expectMsg:             emptyUUIDMsg2,
			expectStateVersionIDs: []string{},
			expectPageInfo:        pagination.PageInfo{},
		},

		{
			name: "filter, workspace ID, positive",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				Filter: &StateVersionFilter{
					WorkspaceID: ptr.String(warmupWorkspaces[0].Metadata.ID),
				},
			},
			expectStateVersionIDs: allStateVersionIDsByTime[:3],
			expectPageInfo:        pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "filter, workspace ID, non-existent",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				Filter: &StateVersionFilter{
					WorkspaceID: ptr.String(nonExistentID),
				},
			},
			expectStateVersionIDs: []string{},
			expectPageInfo:        pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, workspace ID, invalid",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				Filter: &StateVersionFilter{
					WorkspaceID: ptr.String(invalidID),
				},
			},
			expectMsg:             invalidUUIDMsg2,
			expectStateVersionIDs: []string{},
			expectPageInfo:        pagination.PageInfo{},
		},

		{
			name: "filter, state versionIDs, positive",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				Filter: &StateVersionFilter{
					StateVersionIDs: []string{
						allStateVersionIDsByTime[0], allStateVersionIDsByTime[1], allStateVersionIDsByTime[3],
					},
				},
			},
			expectStateVersionIDs: []string{
				allStateVersionIDsByTime[0], allStateVersionIDsByTime[1], allStateVersionIDsByTime[3],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, state versionIDs, non-existent",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				Filter: &StateVersionFilter{
					StateVersionIDs: []string{nonExistentID},
				},
			},
			expectStateVersionIDs: []string{},
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},

		{
			name: "filter, state versionIDs, invalid ID",
			input: &GetStateVersionsInput{
				Sort: ptrStateVersionSortableField(StateVersionSortableFieldUpdatedAtAsc),
				Filter: &StateVersionFilter{
					StateVersionIDs: []string{invalidID},
				},
			},
			expectMsg:             invalidUUIDMsg2,
			expectStateVersionIDs: []string{},
			expectPageInfo:        pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:  true,
			expectHasEndCursor:    true,
		},
	}

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

			stateVersionsResult, err := testClient.client.StateVersions.GetStateVersions(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, stateVersionsResult.PageInfo)
				assert.NotNil(t, stateVersionsResult.StateVersions)
				pageInfo := stateVersionsResult.PageInfo
				stateVersions := stateVersionsResult.StateVersions

				// Check the state versions result by comparing a list of the state version IDs.
				actualStateVersionIDs := []string{}
				for _, stateVersion := range stateVersions {
					actualStateVersionIDs = append(actualStateVersionIDs, stateVersion.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualStateVersionIDs)
				}

				assert.Equal(t, len(test.expectStateVersionIDs), len(actualStateVersionIDs))
				assert.Equal(t, test.expectStateVersionIDs, actualStateVersionIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one state version returned.
				// If there are no state versions returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(stateVersions) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&stateVersions[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&stateVersions[len(stateVersions)-1])
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

func TestGetStateVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := time.Now()
	_, _, warmupStateVersions, err := createWarmupStateVersions(ctx, testClient,
		standardWarmupGroupsForStateVersions, standardWarmupWorkspacesForStateVersions,
		standardWarmupRunsForStateVersions, standardWarmupStateVersions)
	require.Nil(t, err)
	createdHigh := time.Now()

	type testCase struct {
		expectMsg          *string
		expectStateVersion *models.StateVersion
		name               string
		searchID           string
	}

	positiveStateVersion := warmupStateVersions[0]
	now := time.Now()
	testCases := []testCase{
		{
			name:     "positive",
			searchID: positiveStateVersion.Metadata.ID,
			expectStateVersion: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					ID:                positiveStateVersion.Metadata.ID,
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: positiveStateVersion.WorkspaceID,
				RunID:       positiveStateVersion.RunID,
				CreatedBy:   positiveStateVersion.CreatedBy,
			},
		},

		{
			name:     "negative, non-existent state version ID",
			searchID: nonExistentID,
			// expect state version and error to be nil
		},

		{
			name:      "defective-ID",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualStateVersion, err := testClient.client.StateVersions.GetStateVersion(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectStateVersion != nil {
				require.NotNil(t, actualStateVersion)
				compareStateVersions(t, test.expectStateVersion, actualStateVersion, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualStateVersion)
			}
		})
	}
}

func TestCreateStateVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupWorkspaces, warmupRuns, _, err := createWarmupStateVersions(ctx, testClient,
		standardWarmupGroupsForStateVersions, standardWarmupWorkspacesForStateVersions,
		standardWarmupRunsForStateVersions, standardWarmupStateVersions)
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.StateVersion
		expectCreated *models.StateVersion
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.StateVersion{
				WorkspaceID: warmupWorkspaces[1].Metadata.ID,
				RunID:       ptr.String(warmupRuns[0].Metadata.ID),
				CreatedBy:   "positive-test-case",
			},
			expectCreated: &models.StateVersion{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID: warmupWorkspaces[1].Metadata.ID,
				RunID:       ptr.String(warmupRuns[0].Metadata.ID),
				CreatedBy:   "positive-test-case",
			},
		},

		// Duplicates are not prohibited by the DB, so don't do a duplicate test case.

		{
			name: "non-existent workspace ID",
			toCreate: &models.StateVersion{
				WorkspaceID: nonExistentID,
				RunID:       ptr.String(warmupRuns[0].Metadata.ID),
				CreatedBy:   "non-existent-workspace-id-test-case",
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"state_versions\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},

		{
			name: "non-existent run ID",
			toCreate: &models.StateVersion{
				WorkspaceID: warmupWorkspaces[1].Metadata.ID,
				RunID:       ptr.String(nonExistentID),
				CreatedBy:   "non-existent-run-id",
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"state_versions\" violates foreign key constraint \"fk_run_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective workspace ID",
			toCreate: &models.StateVersion{
				WorkspaceID: invalidID,
				RunID:       ptr.String(warmupRuns[0].Metadata.ID),
				CreatedBy:   "defective-workspace-id-test-case",
			},
			expectMsg: invalidUUIDMsg1,
		},

		{
			name: "defective run ID",
			toCreate: &models.StateVersion{
				WorkspaceID: warmupWorkspaces[1].Metadata.ID,
				RunID:       ptr.String(invalidID),
				CreatedBy:   "defective-run-id",
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.StateVersions.CreateStateVersion(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareStateVersions(t, test.expectCreated, actualCreated, false, &timeBounds{
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

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForStateVersions = []models.Group{
	{
		Description: "top level group 0 for testing state version functions",
		FullPath:    "top-level-group-0-for-state-versions",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForStateVersions = []models.Workspace{
	{
		Description: "workspace 0 for testing state version functions",
		FullPath:    "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing state version functions",
		FullPath:    "top-level-group-0-for-state-versions/workspace-1-for-state-versions",
		CreatedBy:   "someone-w1",
	},
}

// Standard warmup run(s) for tests in this module
// The ID fields will be replaced by the ID(s) during the create function.
// Please note: The double nesting is required to support multiple workspaces.
var standardWarmupRunsForStateVersions = [][]models.Run{
	{
		{
			WorkspaceID: "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
			Comment:     "standard warmup run 0 for testing state versions",
		},
		{
			WorkspaceID: "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
			Comment:     "standard warmup run 1 for testing state versions",
		},
		{
			WorkspaceID: "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
			Comment:     "standard warmup run 2 for testing state versions",
		},
	},
	{
		{
			WorkspaceID: "top-level-group-0-for-state-versions/workspace-1-for-state-versions",
			Comment:     "standard warmup run 3 for testing state versions",
		},
		{
			WorkspaceID: "top-level-group-0-for-state-versions/workspace-1-for-state-versions",
			Comment:     "standard warmup run 4 for testing state versions",
		},
	},
}

// Standard warmup state versions for tests in this module:
// The ID fields will be replaced by the real IDs during the create function.
// Please note: Even though RunID is a pointer, it cannot be nil due to a not-null constraint.
var standardWarmupStateVersions = []models.StateVersion{
	{
		WorkspaceID: "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
		RunID:       ptr.String("standard warmup run 0 for testing state versions"),
		CreatedBy:   "someone-sv0",
	},
	{
		WorkspaceID: "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
		RunID:       ptr.String("standard warmup run 1 for testing state versions"),
		CreatedBy:   "someone-sv1",
	},
	{
		WorkspaceID: "top-level-group-0-for-state-versions/workspace-0-for-state-versions",
		RunID:       ptr.String("standard warmup run 2 for testing state versions"),
		CreatedBy:   "someone-sv2",
	},
	{
		WorkspaceID: "top-level-group-0-for-state-versions/workspace-1-for-state-versions",
		RunID:       ptr.String("standard warmup run 3 for testing state versions"),
		CreatedBy:   "someone-sv3",
	},
	{
		WorkspaceID: "top-level-group-0-for-state-versions/workspace-1-for-state-versions",
		RunID:       ptr.String("standard warmup run 4 for testing state versions"),
		CreatedBy:   "someone-sv4",
	},
}

// createWarmupStateVersions creates some warmup state versions for a test
// The warmup state versions to create can be standard or otherwise.
func createWarmupStateVersions(ctx context.Context, testClient *testClient,
	newGroups []models.Group,
	newWorkspaces []models.Workspace,
	newRuns [][]models.Run,
	newStateVersions []models.StateVersion) (
	[]models.Workspace,
	[]models.Run,
	[]models.StateVersion,
	error,
) {
	// It is necessary to create at least one group, workspace, and run
	// in order to provide the necessary IDs for the state versions.

	_, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, nil, err
	}

	workspaceMap := map[string]string{}
	for _, ws := range resultWorkspaces {
		workspaceMap[ws.FullPath] = ws.Metadata.ID
	}

	var resultRuns []models.Run
	for ix := range resultWorkspaces {
		partialResultRuns, err2 := createInitialRuns(ctx, testClient, newRuns[ix], resultWorkspaces[ix].Metadata.ID)
		if err2 != nil {
			return nil, nil, nil, err2
		}
		resultRuns = append(resultRuns, partialResultRuns...)
	}

	runMap := map[string]string{}
	for _, run := range resultRuns {
		runMap[run.Comment] = run.Metadata.ID
	}

	resultStateVersions, err := createInitialStateVersions(ctx, testClient, workspaceMap, runMap, newStateVersions)
	if err != nil {
		return nil, nil, nil, err
	}

	return resultWorkspaces, resultRuns, resultStateVersions, nil
}

func ptrStateVersionSortableField(arg StateVersionSortableField) *StateVersionSortableField {
	return &arg
}

func (wis stateVersionInfoIDSlice) Len() int {
	return len(wis)
}

func (wis stateVersionInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis stateVersionInfoIDSlice) Less(i, j int) bool {
	return wis[i].stateVersionID < wis[j].stateVersionID
}

func (wis stateVersionInfoUpdateSlice) Len() int {
	return len(wis)
}

func (wis stateVersionInfoUpdateSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis stateVersionInfoUpdateSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// stateVersionInfoFromStateVersions returns a slice of stateVersionInfo, not necessarily sorted in any order.
func stateVersionInfoFromStateVersions(stateVersions []models.StateVersion) []stateVersionInfo {
	result := []stateVersionInfo{}

	for _, stateVersion := range stateVersions {
		result = append(result, stateVersionInfo{
			stateVersionID: stateVersion.Metadata.ID,
			updateTime:     *stateVersion.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// stateVersionIDsFromStateVersionInfos preserves order
func stateVersionIDsFromStateVersionInfos(stateVersionInfos []stateVersionInfo) []string {
	result := []string{}
	for _, stateVersionInfo := range stateVersionInfos {
		result = append(result, stateVersionInfo.stateVersionID)
	}
	return result
}

// compareStateVersions compares two state version objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareStateVersions(t *testing.T, expected, actual *models.StateVersion,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.RunID, actual.RunID)
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
