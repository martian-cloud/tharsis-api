//go:build integration

package db

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

type activityEventWarmups struct {
	groups          []models.Group
	workspaces      []models.Workspace
	users           []models.User
	serviceAccounts []models.ServiceAccount
	variables       []models.Variable
	activityEvents  []models.ActivityEvent
}

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// activityEventInfo aids convenience in accessing the information
// TestGetActivityEvents needs about the warmup activity events.
// A nil ID is represented here by an empty string.
type activityEventInfo struct {
	id            string
	creationTime  time.Time
	namespacePath string
	action        models.ActivityEventAction
	targetType    models.ActivityEventTargetType
	targetID      string
}

// activityEventInfoIDSlice makes a slice of activityEventInfo sortable by ID string
type activityEventInfoIDSlice []activityEventInfo

// activityEventInfoTimeSlice makes a slice of activityEventInfo sortable by creation time
type activityEventInfoTimeSlice []activityEventInfo

// activityEventInfoNamespacePathSlice makes a slice of activityEventInfo sortable by namespace path
type activityEventInfoNamespacePathSlice []activityEventInfo

// activityEventInfoActionSlice makes a slice of activityEventInfo sortable by action
type activityEventInfoActionSlice []activityEventInfo

func TestGetActivityEvents(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupActivityEvents(ctx, testClient, activityEventWarmups{
		groups:          standardWarmupGroupsForActivityEvents,
		workspaces:      standardWarmupWorkspacesForActivityEvents,
		users:           standardWarmupUsersForActivityEvents,
		serviceAccounts: standardWarmupServiceAccountsForActivityEvents,
		variables:       standardWarmupVariablesForActivityEvents,
		activityEvents:  buildStandardWarmupActivityEvents(t),
	})
	require.Nil(t, err)

	allActivityEventInfos := activityEventInfoFromActivityEvents(*warmupItems)

	// Sort by activity event IDs.
	sort.Sort(activityEventInfoIDSlice(allActivityEventInfos))
	allActivityEventIDs := activityEventIDsFromActivityEventInfos(allActivityEventInfos)

	// Sort by creation time.
	sort.Sort(activityEventInfoTimeSlice(allActivityEventInfos))
	allActivityEventIDsByCreationTime := activityEventIDsFromActivityEventInfos(allActivityEventInfos)
	reverseActivityEventIDsByCreationTime := reverseStringSlice(allActivityEventIDsByCreationTime)

	// Sort by namespace paths.
	sort.Sort(activityEventInfoNamespacePathSlice(allActivityEventInfos))
	allActivityEventIDsByNamespacePath := activityEventIDsFromActivityEventInfos(allActivityEventInfos)
	reverseActivityEventIDsByNamespacePath := reverseStringSlice(allActivityEventIDsByNamespacePath)

	// Sort by actions.
	sort.Sort(activityEventInfoActionSlice(allActivityEventInfos))
	allActivityEventIDsByAction := activityEventIDsFromActivityEventInfos(allActivityEventInfos)
	reverseActivityEventIDsByAction := reverseStringSlice(allActivityEventIDsByAction)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		input                       *GetActivityEventsInput
		expectMsg                   *string
		name                        string
		expectPageInfo              pagination.PageInfo
		expectActivityEventIDs      []string
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
			input: &GetActivityEventsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectActivityEventIDs:       []string{},
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
		// nil input likely causes a nil pointer dereference in GetActivityEvents, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetActivityEventsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectActivityEventIDs: allActivityEventIDs,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "sort in ascending order of time of creation",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "sort in descending order of time of creation",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtDesc),
			},
			sortedDescending:       true,
			expectActivityEventIDs: reverseActivityEventIDsByCreationTime,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "sort in ascending order of namespace path",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldNamespacePathAsc),
			},
			expectActivityEventIDs: allActivityEventIDsByNamespacePath,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "sort in descending order of namespace path",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldNamespacePathDesc),
			},
			sortedDescending:       true,
			expectActivityEventIDs: reverseActivityEventIDsByNamespacePath,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "sort in ascending order of action",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldActionAsc),
			},
			expectActivityEventIDs: allActivityEventIDsByAction,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "sort in descending order of action",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldActionDesc),
			},
			sortedDescending:       true,
			expectActivityEventIDs: reverseActivityEventIDsByAction,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "pagination: everything at once",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime,
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(len(allActivityEventIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "pagination: first one",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
				},
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime[:1],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allActivityEventIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectActivityEventIDs:     allActivityEventIDsByCreationTime[1:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allActivityEventIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectActivityEventIDs:     allActivityEventIDsByCreationTime[3:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allActivityEventIDs)),
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
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending:       true,
			expectActivityEventIDs: reverseActivityEventIDsByCreationTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allActivityEventIDs)),
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
			input: &GetActivityEventsInput{
				Sort:              ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectActivityEventIDs:      []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:              ptr.String("only first or last can be defined, not both"),
			expectActivityEventIDs: allActivityEventIDs[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allActivityEventIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &ActivityEventFilter{
					// Passing an empty slice likely causes an SQL syntax error ("... IN ()"), so don't try it.
					// ActivityEventIDs    []string{},
					// NamespacePaths:     []string{},
					// Actions:            []string{},
					// TargetTypes:        []string{},
					TimeRangeStart: nil,
					TimeRangeEnd:   nil,
				},
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime,
			expectPageInfo: pagination.PageInfo{
				TotalCount: int32(len(allActivityEventIDs)),
				Cursor:     dummyCursorFunc,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, activity event IDs, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					ActivityEventIDs: []string{
						allActivityEventIDsByCreationTime[1],
						allActivityEventIDsByCreationTime[2],
					},
				},
			},
			expectActivityEventIDs: []string{
				allActivityEventIDsByCreationTime[1],
				allActivityEventIDsByCreationTime[2],
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, activity event IDs, non-existent",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					ActivityEventIDs: []string{nonExistentID},
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, activity event IDs, invalid UUID",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					ActivityEventIDs: []string{invalidID},
				},
			},
			expectMsg:              invalidUUIDMsg2,
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, userIDs, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					UserID: warmupItems.activityEvents[0].UserID, // Don't try to use one that is nil.
				},
			},
			expectActivityEventIDs: []string{
				warmupItems.activityEvents[0].Metadata.ID,
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, userIDs, require membership",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					NamespaceMembershipRequirement: &ActivityEventNamespaceMembershipRequirement{
						UserID: warmupItems.activityEvents[0].UserID,
					},
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, userIDs, non-existent",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					UserID: ptr.String(nonExistentID),
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, userIDs, invalid",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					UserID: ptr.String(invalidID),
				},
			},
			expectMsg:              invalidUUIDMsg2,
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, service account IDs, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					ServiceAccountID: warmupItems.activityEvents[1].ServiceAccountID, // Don't try to use one that is nil.
				},
			},
			expectActivityEventIDs: []string{
				warmupItems.activityEvents[1].Metadata.ID,
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account IDs, require membership",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					NamespaceMembershipRequirement: &ActivityEventNamespaceMembershipRequirement{
						ServiceAccountID: warmupItems.activityEvents[1].ServiceAccountID,
					},
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, service account IDs, non-existent",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					ServiceAccountID: ptr.String(nonExistentID),
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, service account IDs, invalid",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					ServiceAccountID: ptr.String(invalidID),
				},
			},
			expectMsg:              invalidUUIDMsg2,
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					NamespacePath: warmupItems.activityEvents[0].NamespacePath,
				},
			},
			expectActivityEventIDs: []string{
				warmupItems.activityEvents[0].Metadata.ID,
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, non-existent",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					NamespacePath: ptr.String("this-path-does-not-exist"),
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, time range start, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					TimeRangeStart: ptr.Time(time.Now().Add(-5 * time.Minute)),
				},
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime,
			expectPageInfo:         pagination.PageInfo{TotalCount: 4, Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, time range start, negative",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					TimeRangeStart: ptr.Time(time.Now().Add(5 * time.Minute)),
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, time range end, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					TimeRangeEnd: ptr.Time(time.Now().Add(5 * time.Minute)),
				},
			},
			expectActivityEventIDs: allActivityEventIDsByCreationTime,
			expectPageInfo:         pagination.PageInfo{TotalCount: 4, Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, time range end, negative",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					TimeRangeEnd: ptr.Time(time.Now().Add(-5 * time.Minute)),
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, actions, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					Actions: []models.ActivityEventAction{
						models.ActionCreate,
						models.ActionApply,
					},
				},
			},
			expectActivityEventIDs: []string{
				warmupItems.activityEvents[0].Metadata.ID,
				warmupItems.activityEvents[1].Metadata.ID,
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, actions, non-existent",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					Actions: []models.ActivityEventAction{"non-existent-action"},
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
		},

		{
			name: "filter, target types, positive",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					TargetTypes: []models.ActivityEventTargetType{
						models.TargetGroup,
						models.TargetVariable,
					},
				},
			},
			expectActivityEventIDs: []string{
				warmupItems.activityEvents[0].Metadata.ID,
				warmupItems.activityEvents[2].Metadata.ID,
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, target types, non-existent",
			input: &GetActivityEventsInput{
				Sort: ptrActivityEventSortableField(ActivityEventSortableFieldCreatedAtAsc),
				Filter: &ActivityEventFilter{
					TargetTypes: []models.ActivityEventTargetType{"non-existent-target-type"},
				},
			},
			expectActivityEventIDs: []string{},
			expectPageInfo:         pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:   true,
			expectHasEndCursor:     true,
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

			activityEventsResult, err := testClient.client.ActivityEvents.GetActivityEvents(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, activityEventsResult.PageInfo)
				assert.NotNil(t, activityEventsResult.ActivityEvents)
				pageInfo := activityEventsResult.PageInfo
				activityEvents := activityEventsResult.ActivityEvents

				// Check the activity events result by comparing a list of the activity event IDs.
				actualActivityEventIDs := []string{}
				for _, activityEvent := range activityEvents {
					actualActivityEventIDs = append(actualActivityEventIDs, activityEvent.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualActivityEventIDs)
				}

				assert.Equal(t, len(test.expectActivityEventIDs), len(actualActivityEventIDs))
				assert.Equal(t, test.expectActivityEventIDs, actualActivityEventIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one activity event returned.
				// If there are no activity events returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(activityEvents) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&activityEvents[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&activityEvents[len(activityEvents)-1])
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

func TestCreateActivityEvent(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupActivityEvents(ctx, testClient, activityEventWarmups{
		groups:          standardWarmupGroupsForActivityEvents,
		workspaces:      standardWarmupWorkspacesForActivityEvents,
		users:           standardWarmupUsersForActivityEvents,
		serviceAccounts: standardWarmupServiceAccountsForActivityEvents,
		variables:       standardWarmupVariablesForActivityEvents,
		activityEvents:  []models.ActivityEvent{},
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.ActivityEvent
		expectCreated *models.ActivityEvent
		expectMsg     *string
		name          string
	}

	now := time.Now()
	positiveTargetID := warmupItems.variables[0].Metadata.ID
	positivePayload := buildPayload(t, map[string]string{"k9": "v9"})
	// Adjust to spacing difference in Postgres vs. Golang JSON serialization.
	expectedPayload := bytes.ReplaceAll(positivePayload, []byte{':'}, []byte{':', ' '})
	testCases := []testCase{
		{
			name: "positive",
			toCreate: &models.ActivityEvent{
				UserID:           ptr.String(warmupItems.users[0].Metadata.ID),
				ServiceAccountID: ptr.String(warmupItems.serviceAccounts[0].Metadata.ID),
				NamespacePath:    ptr.String("top-level-group-0-for-activity-events/workspace-0-for-activity-events"),
				Action:           models.ActionLock,
				TargetType:       models.TargetVariable,
				TargetID:         positiveTargetID,
				Payload:          positivePayload,
			},
			expectCreated: &models.ActivityEvent{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				UserID:           ptr.String(warmupItems.users[0].Metadata.ID),
				ServiceAccountID: ptr.String(warmupItems.serviceAccounts[0].Metadata.ID),
				NamespacePath:    ptr.String("top-level-group-0-for-activity-events/workspace-0-for-activity-events"),
				Action:           models.ActionLock,
				TargetType:       models.TargetVariable,
				TargetID:         positiveTargetID,
				Payload:          expectedPayload,
			},
		},

		{
			name: "negative, non-existent user ID",
			toCreate: &models.ActivityEvent{
				UserID:     ptr.String(nonExistentID),
				Payload:    positivePayload,
				TargetType: models.TargetVariable,
				TargetID:   positiveTargetID,
			},
			expectMsg: ptr.String("user does not exist"),
			// expect activity event to be nil
		},

		{
			name: "negative, invalid user ID",
			toCreate: &models.ActivityEvent{
				UserID:     ptr.String(invalidID),
				Payload:    positivePayload,
				TargetType: models.TargetVariable,
				TargetID:   positiveTargetID,
			},
			expectMsg: invalidUUIDMsg1,
			// expect activity event to be nil
		},

		{
			name: "negative, non-existent service account",
			toCreate: &models.ActivityEvent{
				ServiceAccountID: ptr.String(nonExistentID),
				TargetType:       models.TargetVariable,
				TargetID:         positiveTargetID,
				Payload:          positivePayload,
			},
			expectMsg: ptr.String("service account does not exist"),
			// expect activity event to be nil
		},

		{
			name: "negative, invalid service account",
			toCreate: &models.ActivityEvent{
				ServiceAccountID: ptr.String(invalidID),
				TargetType:       models.TargetVariable,
				TargetID:         positiveTargetID,
				Payload:          positivePayload,
			},
			expectMsg: invalidUUIDMsg1,
			// expect activity event to be nil
		},

		{
			name: "negative, non-existent namespace path",
			toCreate: &models.ActivityEvent{
				NamespacePath: ptr.String("non-existent-namespace"),
			},
			expectMsg: ptr.String("Namespace not found"),
			// expect activity event to be nil
		},

		// It is the namespace path rather than the ID that gets passed in, so can't do an invalid UUID.

	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := time.Now()

				compareActivityEvents(t, test.expectCreated, actualCreated, false, &timeBounds{
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

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForActivityEvents = []models.Group{
	{
		Description: "top level group 0 for testing activity event functions",
		FullPath:    "top-level-group-0-for-activity-events",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForActivityEvents = []models.Workspace{
	{
		Description: "workspace 0 for testing activity event functions",
		FullPath:    "top-level-group-0-for-activity-events/workspace-0-for-activity-events",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing activity event functions",
		FullPath:    "top-level-group-0-for-activity-events/workspace-1-for-activity-events",
		CreatedBy:   "someone-w1",
	},
}

// Standard warmup users for tests in this module:
// Please note: all users are _NON_-admin.
var standardWarmupUsersForActivityEvents = []models.User{
	{
		Username: "user-0",
		Email:    "user-0@example.com",
	},
	{
		Username: "user-1",
		Email:    "user-1@example.com",
	},
	{
		Username: "user-2",
		Email:    "user-2@example.com",
	},
}

// Standard warmup service accounts for tests in this module:
// Please note: the GroupID field here contains the group _FULL_PATH_, not the _ID_.
var standardWarmupServiceAccountsForActivityEvents = []models.ServiceAccount{
	{
		Name:        "sa-0",
		Description: "service account 0 for namespace membership tests",
		GroupID:     "top-level-group-0-for-activity-events",
		CreatedBy:   "someone-sa0",
	},
	{
		Name:        "sa-1",
		Description: "service account 1 for namespace membership tests",
		GroupID:     "top-level-group-0-for-activity-events",
		CreatedBy:   "someone-sa1",
	},
	{
		Name:        "sa-2",
		Description: "service account 2 for namespace membership tests",
		GroupID:     "top-level-group-0-for-activity-events",
		CreatedBy:   "someone-sa2",
	},
}

// Standard warmup variable(s) for tests in this module:
var standardWarmupVariablesForActivityEvents = []models.Variable{
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-0-for-activity-events",
		Hcl:           false,
		Key:           "variable-0-key",
		Value:         ptr.String("variable-0-value"),
	},
}

// buildStandardWarmupActivityEvents builds the standard warmup activity events for tests in this module.
// It must be a function, because it needs t to build the JSON payload.
// The create functions will replace the username and service account name with the respective IDs.
// The create function will fill in the target ID with the first item in the list for the target type.
// Note: Actions and target types must not be duplicated in this list or sorting becomes nondeterministic.
func buildStandardWarmupActivityEvents(t *testing.T) []models.ActivityEvent {
	return []models.ActivityEvent{
		{
			UserID:           ptr.String("user-1"),
			ServiceAccountID: nil,
			NamespacePath:    ptr.String("top-level-group-0-for-activity-events/workspace-0-for-activity-events"),
			Action:           models.ActionCreate,
			TargetType:       models.TargetVariable,
			TargetID:         invalidID, // will be variable 0
			Payload:          buildPayload(t, map[string]string{"k0a": "v0a", "k0b": "v0b"}),
		},
		{
			UserID:           ptr.String("user-2"),
			ServiceAccountID: ptr.String("sa-0"),
			NamespacePath:    nil,
			Action:           models.ActionApply,
			TargetType:       models.TargetServiceAccount,
			TargetID:         invalidID, // will be service account 0
			Payload:          buildPayload(t, map[string]string{"k1": "v1"}),
		},
		{
			UserID:           ptr.String("user-0"),
			ServiceAccountID: ptr.String("sa-2"),
			NamespacePath:    ptr.String("top-level-group-0-for-activity-events/workspace-1-for-activity-events"),
			Action:           models.ActionCancel,
			TargetType:       models.TargetGroup,
			TargetID:         invalidID, // will be group 0
			Payload:          buildPayload(t, map[string]string{"k2a": "v2a", "k2b": "v2b", "k2c": "v2c"}),
		},
		{
			UserID:           nil,
			ServiceAccountID: ptr.String("sa-1"),
			NamespacePath:    ptr.String("top-level-group-0-for-activity-events"),
			Action:           models.ActionLock,
			TargetType:       models.TargetWorkspace,
			TargetID:         invalidID, // will be workspace 0
			Payload:          buildPayload(t, map[string]string{"k3": "v3"}),
		},
	}
}

// createWarmupActivityEvents creates some warmup activity events for a test
// The warmup activity events to create can be standard or otherwise.
func createWarmupActivityEvents(ctx context.Context, testClient *testClient,
	input activityEventWarmups,
) (*activityEventWarmups, error) {
	// It is necessary to create at least one group, workspace, and run
	// in order to provide the necessary IDs for the activity events.

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, groupPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	namespacePath2ID := groupPath2ID
	for _, ws := range resultWorkspaces {
		namespacePath2ID[ws.FullPath] = ws.Metadata.ID
	}

	resultUsers, username2ID, err := createInitialUsers(ctx, testClient, input.users)
	if err != nil {
		return nil, err
	}

	resultServiceAccounts, serviceAccountName2ID, err := createInitialServiceAccounts(ctx, testClient,
		groupPath2ID, input.serviceAccounts)
	if err != nil {
		return nil, err
	}

	resultVariables, err := createInitialVariables(ctx, testClient, input.variables)
	if err != nil {
		return nil, err
	}

	// Make a modified copy of the input events to set the target ID.
	modifiedInputEvents := []models.ActivityEvent{}
	for _, oldEvent := range input.activityEvents {
		newEvent := oldEvent

		var newID string
		switch newEvent.TargetType {
		case models.TargetVariable:
			newID = resultVariables[0].Metadata.ID
		case models.TargetServiceAccount:
			newID = resultServiceAccounts[0].Metadata.ID
		case models.TargetGroup:
			newID = resultGroups[0].Metadata.ID
		case models.TargetWorkspace:
			newID = resultWorkspaces[0].Metadata.ID
		}
		newEvent.TargetID = newID

		modifiedInputEvents = append(modifiedInputEvents, newEvent)
	}

	resultActivityEvents, err := createInitialActivityEvents(ctx, testClient,
		modifiedInputEvents, username2ID, serviceAccountName2ID)
	if err != nil {
		return nil, err
	}

	return &activityEventWarmups{
		groups:          resultGroups,
		workspaces:      resultWorkspaces,
		users:           resultUsers,
		serviceAccounts: resultServiceAccounts,
		variables:       resultVariables,
		activityEvents:  resultActivityEvents,
	}, nil
}

func ptrActivityEventSortableField(arg ActivityEventSortableField) *ActivityEventSortableField {
	return &arg
}

func (wis activityEventInfoIDSlice) Len() int {
	return len(wis)
}

func (wis activityEventInfoIDSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis activityEventInfoIDSlice) Less(i, j int) bool {
	return wis[i].id < wis[j].id
}

func (nss activityEventInfoTimeSlice) Len() int {
	return len(nss)
}

func (nss activityEventInfoTimeSlice) Swap(i, j int) {
	nss[i], nss[j] = nss[j], nss[i]
}

func (nss activityEventInfoTimeSlice) Less(i, j int) bool {
	return nss[i].creationTime.Before(nss[j].creationTime)
}

func (wis activityEventInfoNamespacePathSlice) Len() int {
	return len(wis)
}

func (wis activityEventInfoNamespacePathSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis activityEventInfoNamespacePathSlice) Less(i, j int) bool {
	return modifiedLessThan(wis[i].namespacePath, wis[j].namespacePath)
}

func (wis activityEventInfoActionSlice) Len() int {
	return len(wis)
}

func (wis activityEventInfoActionSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis activityEventInfoActionSlice) Less(i, j int) bool {
	return modifiedLessThan(string(wis[i].action), string(wis[j].action))
}

// modifiedLessThan compares two strings with a twist:
// SQL sorts nil (seen here as empty string) values as greater than non-nil/empty values.
func modifiedLessThan(a, b string) bool {
	if (a == "") && (b != "") {
		return false
	}
	if (a != "") && (b == "") {
		return true
	}
	return a < b
}

// activityEventInfoFromActivityEvents returns a slice of activityEventInfo, not necessarily sorted in any order.
func activityEventInfoFromActivityEvents(warmups activityEventWarmups) []activityEventInfo {
	result := []activityEventInfo{}

	for _, activityEvent := range warmups.activityEvents {

		namespacePath := ""
		if activityEvent.NamespacePath != nil {
			namespacePath = *activityEvent.NamespacePath
		}

		result = append(result, activityEventInfo{
			id:            activityEvent.Metadata.ID,
			creationTime:  *activityEvent.Metadata.CreationTimestamp,
			namespacePath: namespacePath,
			action:        activityEvent.Action,
			targetType:    activityEvent.TargetType,
			targetID:      activityEvent.TargetID,
		})
	}

	return result
}

// activityEventIDsFromActivityEventInfos preserves order
func activityEventIDsFromActivityEventInfos(activityEventInfos []activityEventInfo) []string {
	result := []string{}
	for _, activityEventInfo := range activityEventInfos {
		result = append(result, activityEventInfo.id)
	}
	return result
}

// compareActivityEvents compares two activity event objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareActivityEvents(t *testing.T, expected, actual *models.ActivityEvent,
	checkID bool, times *timeBounds,
) {
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

	compareStringPointers(t, expected.UserID, actual.UserID)
	compareStringPointers(t, expected.ServiceAccountID, actual.ServiceAccountID)
	compareStringPointers(t, expected.NamespacePath, actual.NamespacePath)

	assert.Equal(t, expected.Action, actual.Action)
	assert.Equal(t, expected.TargetType, actual.TargetType)
	assert.Equal(t, expected.TargetID, actual.TargetID)
	assert.Equal(t, expected.Payload, actual.Payload)
}

// compareStringPointers compares two string pointers.
func compareStringPointers(t *testing.T, expected, actual *string) {
	assert.Equal(t, (expected == nil), (actual == nil))

	if (expected != nil) && (actual != nil) {
		assert.Equal(t, *expected, *actual)
	}
}

// buildPayload builds a JSON payload.
func buildPayload(t *testing.T, input interface{}) []byte {
	result, err := json.Marshal(input)
	require.Nil(t, err)
	return result
}
