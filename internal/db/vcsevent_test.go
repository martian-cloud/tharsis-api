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

const (
	sampleRepositoryURL = "https://github.com/owner/repository"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// vcsEventInfo aids convenience in accessing the information TestVCSEvents
// needs about the warmup vcs events.
type vcsEventInfo struct {
	createTime time.Time
	updateTime time.Time
	vcsEventID string
}

// vcsEventInfoIDSlice makes a slice of vcsEventInfo sortable by ID string
type vcsEventInfoIDSlice []vcsEventInfo

// vcsEventInfoCreateSlice makes a slice of vcsEventInfo sortable by creation time
type vcsEventInfoCreateSlice []vcsEventInfo

// vcsEventInfoUpdateSlice makes a slice of vcsEventInfo sortable by last updated time
type vcsEventInfoUpdateSlice []vcsEventInfo

// warmupVCSEvents holds the inputs to and outputs from createWarmupVCSEvents.
type warmupVCSEvents struct {
	groups     []models.Group
	workspaces []models.Workspace
	events     []models.VCSEvent
}

func TestGetEventByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a vcs event with a specific ID without going into the really
	// low-level stuff, create the warmup vcs event and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupVCSEvents(ctx, testClient,
		warmupVCSEvents{
			standardWarmupGroupsForVCSEvents,
			standardWarmupWorkspacesForVCSEvents,
			standardWarmupVCSEvents,
		})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectEvent *models.VCSEvent
		expectMsg   *string
		name        string
		searchID    string
	}

	positiveEvent := warmupItems.events[0]
	testCases := []testCase{
		{
			name:        "positive",
			searchID:    positiveEvent.Metadata.ID,
			expectEvent: &positiveEvent,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect event and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualEvent, err := testClient.client.VCSEvents.GetEventByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectEvent != nil {
				require.NotNil(t, actualEvent)
				compareVCSEvents(t, test.expectEvent, actualEvent, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualEvent)
			}
		})
	}
}

func TestGetEvents(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a vcs event with a specific ID without going into the really
	// low-level stuff, create the warmup vcs event and then find the relevant ID.
	warmupItems, err := createWarmupVCSEvents(ctx, testClient,
		warmupVCSEvents{
			standardWarmupGroupsForVCSEvents,
			standardWarmupWorkspacesForVCSEvents,
			standardWarmupVCSEvents,
		})
	require.Nil(t, err)

	allVCSEventInfos := vcsEventInfoFromVCSEvents(warmupItems.events)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(vcsEventInfoIDSlice(allVCSEventInfos))
	allVCSEventIDs := vcsEventIDsFromVCSEventInfos(allVCSEventInfos)

	// Sort by creation times.
	sort.Sort(vcsEventInfoCreateSlice(allVCSEventInfos))
	allVCSEventIDsByCreateTime := vcsEventIDsFromVCSEventInfos(allVCSEventInfos)
	reverseVCSEventIDsByCreateTime := reverseStringSlice(allVCSEventIDsByCreateTime)

	// Sort by last update times.
	sort.Sort(vcsEventInfoUpdateSlice(allVCSEventInfos))
	allVCSEventIDsByUpdateTime := vcsEventIDsFromVCSEventInfos(allVCSEventInfos)
	reverseVCSEventIDsByUpdateTime := reverseStringSlice(allVCSEventIDsByUpdateTime)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetVCSEventsInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectVCSEventIDs           []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetVCSEventsInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectVCSEventIDs    		[]string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetVCSEventsInput, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetVCSEventsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectVCSEventIDs:    allVCSEventIDs,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVCSEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectVCSEventIDs:    allVCSEventIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVCSEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of creation time",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtDesc),
			},
			expectVCSEventIDs:    reverseVCSEventIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVCSEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectVCSEventIDs:    allVCSEventIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVCSEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtDesc),
			},
			expectVCSEventIDs:    reverseVCSEventIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVCSEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectVCSEventIDs:    allVCSEventIDsByUpdateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allVCSEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectVCSEventIDs: allVCSEventIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVCSEventIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectVCSEventIDs:          allVCSEventIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVCSEventIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectVCSEventIDs:          allVCSEventIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVCSEventIDs)),
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
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectVCSEventIDs: reverseVCSEventIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVCSEventIDs)),
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
			input: &GetVCSEventsInput{
				Sort:              ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectVCSEventIDs:           []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allVCSEventIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, vcs event ids, positive",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtAsc),
				Filter: &VCSEventFilter{
					VCSEventIDs: allVCSEventIDs,
				},
			},
			expectVCSEventIDs:    allVCSEventIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(5), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, vcs event ids, non existent, negative",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtAsc),
				Filter: &VCSEventFilter{
					VCSEventIDs: []string{nonExistentID},
				},
			},
			expectVCSEventIDs:    []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, vcs event ids, invalid id, negative",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtAsc),
				Filter: &VCSEventFilter{
					VCSEventIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspaceID field, warmup workspace ID",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtAsc),
				Filter: &VCSEventFilter{
					WorkspaceID: &warmupItems.workspaces[0].Metadata.ID,
				},
			},
			expectVCSEventIDs:    allVCSEventIDsByCreateTime,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(5), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspaceID field, nonExistentID",
			input: &GetVCSEventsInput{
				Sort: ptrVCSEventSortableField(VCSEventSortableFieldCreatedAtAsc),
				Filter: &VCSEventFilter{
					WorkspaceID: ptr.String(nonExistentID),
				},
			},
			expectVCSEventIDs:    []string{},
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
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

			vcsEventsActual, err := testClient.client.VCSEvents.GetEvents(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, vcsEventsActual.PageInfo)
				assert.NotNil(t, vcsEventsActual.VCSEvents)
				pageInfo := vcsEventsActual.PageInfo
				vcsEvents := vcsEventsActual.VCSEvents

				// Check the vcs events result by comparing a list of the vcs event IDs.
				actualVCSEventIDs := []string{}
				for _, vcsEvent := range vcsEvents {
					actualVCSEventIDs = append(actualVCSEventIDs, vcsEvent.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualVCSEventIDs)
				}

				assert.Equal(t, len(test.expectVCSEventIDs), len(actualVCSEventIDs))
				assert.Equal(t, test.expectVCSEventIDs, actualVCSEventIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one vcs event returned.
				// If there are no vcs event returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(vcsEvents) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&vcsEvents[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&vcsEvents[len(vcsEvents)-1])
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

func TestCreateEvent(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupVCSEvents(ctx, testClient,
		warmupVCSEvents{
			standardWarmupGroupsForVCSEvents,
			standardWarmupWorkspacesForVCSEvents,
			[]models.VCSEvent{},
		})
	require.Nil(t, err)

	warmupWorkspace := warmupItems.workspaces[0]
	warmupWorkspaceID := warmupWorkspace.Metadata.ID

	type testCase struct {
		toCreate      *models.VCSEvent
		expectCreated *models.VCSEvent
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.VCSEvent{
				SourceReferenceName: ptr.String("feature/branch"),
				WorkspaceID:         warmupWorkspaceID,
				RepositoryURL:       sampleRepositoryURL,
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending, // This will always be 'pending' for creation.
			},
			expectCreated: &models.VCSEvent{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				SourceReferenceName: ptr.String("feature/branch"),
				WorkspaceID:         warmupWorkspaceID,
				RepositoryURL:       sampleRepositoryURL,
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending,
			},
		},

		{
			name: "positive full",
			toCreate: &models.VCSEvent{
				SourceReferenceName: ptr.String("feature/branch"),
				WorkspaceID:         warmupWorkspaceID,
				RepositoryURL:       sampleRepositoryURL,
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending, // This will always be 'pending' for creation.
				CommitID:            ptr.String("a-commit-id-here"),
				ErrorMessage:        ptr.String("some-error-here"), // Arbitrary.
			},
			expectCreated: &models.VCSEvent{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				SourceReferenceName: ptr.String("feature/branch"),
				WorkspaceID:         warmupWorkspaceID,
				RepositoryURL:       sampleRepositoryURL,
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending,
				CommitID:            ptr.String("a-commit-id-here"),
				ErrorMessage:        ptr.String("some-error-here"),
			},
		},

		{
			name: "non-existent workspace ID",
			toCreate: &models.VCSEvent{
				WorkspaceID: nonExistentID,
			},
			expectMsg: ptr.String("workspace does not exist"),
		},

		{
			name: "defective workspace ID",
			toCreate: &models.VCSEvent{
				WorkspaceID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.VCSEvents.CreateEvent(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareVCSEvents(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateEvent(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()
	warmupItems, err := createWarmupVCSEvents(ctx, testClient,
		warmupVCSEvents{
			standardWarmupGroupsForVCSEvents,
			standardWarmupWorkspacesForVCSEvents,
			standardWarmupVCSEvents,
		})
	require.Nil(t, err)

	createdHigh := currentTime()
	warmupWorkspace := warmupItems.workspaces[0]

	type testCase struct {
		toUpdate       *models.VCSEvent
		expectVCSEvent *models.VCSEvent
		expectMsg      *string
		name           string
	}

	// Do only one positive test case, because the logic is theoretically the same for all vcs providers.
	now := currentTime()
	positiveVCSEvent := warmupItems.events[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.VCSEvent{
				Metadata: models.ResourceMetadata{
					ID:      positiveVCSEvent.Metadata.ID,
					Version: positiveVCSEvent.Metadata.Version,
				},
				Status:       models.VCSEventErrored,
				ErrorMessage: ptr.String("some-error-message-here"),
			},
			expectVCSEvent: &models.VCSEvent{
				Metadata: models.ResourceMetadata{
					ID:                   positiveVCSEvent.Metadata.ID,
					Version:              positiveVCSEvent.Metadata.Version + 1,
					CreationTimestamp:    positiveVCSEvent.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				SourceReferenceName: positiveVCSEvent.SourceReferenceName,
				WorkspaceID:         warmupWorkspace.Metadata.ID,
				RepositoryURL:       sampleRepositoryURL,
				Type:                positiveVCSEvent.Type,
				Status:              models.VCSEventErrored,
				ErrorMessage:        ptr.String("some-error-message-here"),
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.VCSEvent{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveVCSEvent.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.VCSEvent{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveVCSEvent.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualVCSEvent, err := testClient.client.VCSEvents.UpdateEvent(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			now := currentTime()
			if test.expectVCSEvent != nil {
				require.NotNil(t, actualVCSEvent)
				compareVCSEvents(t, test.expectVCSEvent, actualVCSEvent, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualVCSEvent)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForVCSEvents = []models.Group{
	{
		Description: "top level group 0 for testing vcs event functions",
		FullPath:    "top-level-group-0-for-vcs-events",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForVCSEvents = []models.Workspace{
	{
		Description: "workspace 0 for testing vcs event functions",
		FullPath:    "top-level-group-0-for-vcs-events/workspace-0-for-vcs-events",
		CreatedBy:   "someone-w0",
	},
}

var standardWarmupVCSEvents = []models.VCSEvent{
	{
		WorkspaceID:         "top-level-group-0-for-vcs-events/workspace-0-for-vcs-events",
		SourceReferenceName: ptr.String("feature/branch"),
		RepositoryURL:       sampleRepositoryURL,
		Type:                models.BranchEventType,
		Status:              models.VCSEventPending,
	},
	{
		WorkspaceID:         "top-level-group-0-for-vcs-events/workspace-0-for-vcs-events",
		SourceReferenceName: ptr.String("v0.1"),
		RepositoryURL:       sampleRepositoryURL,
		Type:                models.TagEventType,
		Status:              models.VCSEventPending,
	},
	{
		WorkspaceID:         "top-level-group-0-for-vcs-events/workspace-0-for-vcs-events",
		SourceReferenceName: ptr.String("feature/another-branch"),
		RepositoryURL:       sampleRepositoryURL,
		Type:                models.MergeRequestEventType,
		Status:              models.VCSEventPending,
	},
	{
		WorkspaceID:         "top-level-group-0-for-vcs-events/workspace-0-for-vcs-events",
		SourceReferenceName: ptr.String("main"),
		RepositoryURL:       sampleRepositoryURL,
		Type:                models.ManualEventType,
		Status:              models.VCSEventPending,
	},
	{
		WorkspaceID:         "top-level-group-0-for-vcs-events/workspace-0-for-vcs-events",
		SourceReferenceName: ptr.String("main"),
		RepositoryURL:       sampleRepositoryURL,
		Type:                models.ManualEventType,
		Status:              models.VCSEventPending,
	},
}

// createWarmupVCSEvents creates vcs events for testing.
func createWarmupVCSEvents(ctx context.Context, testClient *testClient, input warmupVCSEvents) (*warmupVCSEvents, error) {
	// It is necessary to create at least one group and workspace
	// in order to provide the necessary IDs for the vcs events.

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, groupPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	workspacePath2ID := make(map[string]string, len(resultWorkspaces))
	for _, workspace := range resultWorkspaces {
		workspacePath2ID[workspace.FullPath] = workspace.Metadata.ID
	}

	resultEvents, err := createInitialVCSEvents(ctx, testClient, workspacePath2ID, input.events)
	if err != nil {
		return nil, err
	}

	return &warmupVCSEvents{
		groups:     resultGroups,
		workspaces: resultWorkspaces,
		events:     resultEvents,
	}, nil
}

// createInitialVCSEvents creates some warmup vcs events for a test.
func createInitialVCSEvents(
	ctx context.Context,
	testClient *testClient,
	workspaceMap map[string]string,
	toCreate []models.VCSEvent,
) (
	[]models.VCSEvent, error,
) {
	result := []models.VCSEvent{}

	for _, input := range toCreate {
		input.WorkspaceID = workspaceMap[input.WorkspaceID]
		created, err := testClient.client.VCSEvents.CreateEvent(ctx, &input)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial vcs event: %s", err)
		}

		result = append(result, *created)
	}

	// In order to make the created-at and last-updated-at orders differ,
	// update every third object without changing any values.
	for ix, toUpdate := range result {
		if ix%3 == 0 {
			updated, err := testClient.client.VCSEvents.UpdateEvent(ctx, &toUpdate)
			if err != nil {
				return nil, fmt.Errorf("failed to update initial vcs event: %s", err)
			}
			result[ix] = *updated
		}
	}

	return result, nil
}

func ptrVCSEventSortableField(arg VCSEventSortableField) *VCSEventSortableField {
	return &arg
}

func (vp vcsEventInfoIDSlice) Len() int {
	return len(vp)
}

func (vp vcsEventInfoIDSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsEventInfoIDSlice) Less(i, j int) bool {
	return vp[i].vcsEventID < vp[j].vcsEventID
}

func (vp vcsEventInfoCreateSlice) Len() int {
	return len(vp)
}

func (vp vcsEventInfoCreateSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsEventInfoCreateSlice) Less(i, j int) bool {
	return vp[i].createTime.Before(vp[j].createTime)
}

func (vp vcsEventInfoUpdateSlice) Len() int {
	return len(vp)
}

func (vp vcsEventInfoUpdateSlice) Swap(i, j int) {
	vp[i], vp[j] = vp[j], vp[i]
}

func (vp vcsEventInfoUpdateSlice) Less(i, j int) bool {
	return vp[i].updateTime.Before(vp[j].updateTime)
}

// vcsEventInfoFromVCSEvents returns a slice of vcsEventInfo, not necessarily sorted in any order.
func vcsEventInfoFromVCSEvents(vcsEvents []models.VCSEvent) []vcsEventInfo {
	result := []vcsEventInfo{}

	for _, vcsEvent := range vcsEvents {
		result = append(result, vcsEventInfo{
			createTime: *vcsEvent.Metadata.CreationTimestamp,
			updateTime: *vcsEvent.Metadata.LastUpdatedTimestamp,
			vcsEventID: vcsEvent.Metadata.ID,
		})
	}

	return result
}

// vcsEventIDsFromVCSEventInfos preserves order
func vcsEventIDsFromVCSEventInfos(vcsEventInfos []vcsEventInfo) []string {
	result := []string{}
	for _, vcsEventInfos := range vcsEventInfos {
		result = append(result, vcsEventInfos.vcsEventID)
	}

	return result
}

// compareVCSEvents compares two vcs event objects,
// including bounds for creation and updated times. If times is nil, it compares
// the exact metadata timestamps.
func compareVCSEvents(t *testing.T, expected, actual *models.VCSEvent,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.CommitID, actual.CommitID)
	assert.Equal(t, expected.ErrorMessage, actual.ErrorMessage)
	assert.Equal(t, expected.RepositoryURL, actual.RepositoryURL)
	assert.Equal(t, expected.SourceReferenceName, actual.SourceReferenceName)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Type, actual.Type)

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
