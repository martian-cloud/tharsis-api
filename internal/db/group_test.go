//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// groupInfo aids convenience in accessing the information TestGetGroups needs about the warmup groups.
type groupInfo struct {
	fullPath    string
	groupID     string
	namespaceID string
}

// groupInfoSlice makes a slice of groupInfo sortable
type groupInfoSlice []groupInfo

// TestGetGroupByID tests GetGroupByID
func TestGetGroupByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a group with a specific ID without going into the really
	// low-level stuff, create the warmup group(s) by name and then find the relevant ID.
	createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups weren't all created.
		return
	}

	type testCase struct {
		expectMsg   *string
		name        string
		searchID    string
		expectGroup bool
	}

	// Run only one positive test case, because the logic is theoretically the same for all groups.
	positiveGroup := createdWarmupGroups[0]
	testCases := []testCase{
		{
			name:        "positive-" + positiveGroup.FullPath,
			searchID:    positiveGroup.Metadata.ID,
			expectGroup: true,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect group and error to be nil
		},
		{
			name:        "defective-id",
			searchID:    invalidID,
			expectGroup: false,
			expectMsg:   invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			group, err := testClient.client.Groups.GetGroupByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectGroup {
				require.NotNil(t, group)
				assert.Equal(t, test.searchID, group.Metadata.ID)
			} else {
				assert.Nil(t, group)
			}

		})
	}
}

// TestGetGroupByFullPath tests GetGroupByFullPath
func TestGetGroupByFullPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups weren't all created.
		return
	}

	type testCase struct {
		expectMsg   *string
		name        string
		searchPath  string
		expectGroup bool
	}

	testCases := []testCase{}
	for _, positiveGroup := range createdWarmupGroups {
		testCases = append(testCases, testCase{
			name:        "positive-" + positiveGroup.FullPath,
			searchPath:  positiveGroup.FullPath,
			expectGroup: true,
		})
	}

	testCases = append(testCases,
		testCase{
			name:       "negative, non-existent top-level group",
			searchPath: "non-existent-top-level-group",
			// expect group and error to be nil
		},
		testCase{
			name:       "negative, non-existent second-level group",
			searchPath: "top-level-group-1/non-existent-2nd-level-group",
			// expect group and error to be nil
		},
		testCase{
			name:       "negative, non-existent third-level group",
			searchPath: "top-level-group-1/2nd-level-group-1b/non-existent-3rd-level-group",
			// expect group and error to be nil
		},
		testCase{
			name:       "negative, non-existent fourth-level",
			searchPath: "top-level-group-1/2nd-level-group-1b/3rd-level-group-1b1/non-existent-4th-level-group",
			// expect group and error to be nil
		},
		testCase{
			name:       "defective-path",
			searchPath: "this*is*a*not*a*valid*path",
			// expect group and error to be nil
			// At the DB layer, the search path is just looked up, with no group returned.
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			group, err := testClient.client.Groups.GetGroupByFullPath(ctx, test.searchPath)

			checkError(t, test.expectMsg, err)

			if test.expectGroup {
				require.NotNil(t, group)
				assert.Equal(t, test.searchPath, group.FullPath)
			} else {
				assert.Nil(t, group)
			}

		})
	}
}

// TestDeleteGroup tests DeleteGroup
func TestDeleteGroup(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create initial warmup groups in order to get their paths in sorted order, etc.
	initialWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups weren't all created.
		return
	}

	// Build slices of group paths, one ascending (but only top-level) and one descending.
	ascendingTopPaths := sort.StringSlice{}
	descendingPaths := sort.StringSlice{}
	for _, group := range initialWarmupGroups {
		if strings.HasPrefix(group.FullPath, "top") {
			ascendingTopPaths = append(ascendingTopPaths, group.FullPath)
		}
		descendingPaths = append(descendingPaths, group.FullPath)
	}
	sort.Sort(ascendingTopPaths)
	sort.Sort(sort.Reverse(descendingPaths))

	// Must close manually, because more test clients will be created inside the loop.
	testClient.close(ctx)

	type testCase struct {
		expectMsgTop     *string
		expectMsgNonTop  *string
		name             string
		paths            sort.StringSlice
		ids              []string
		expectFinalCount int
	}

	// Please note: An attempt to delete a group at the DB layer produces a message that the
	// "resource version does not match specified version".  However, that is not a bug, because
	// the GraphQL resolver does a GetGroupByFullPath before making the actual attempt to delete
	// the group, and that GetGroupByFullPath will allow the GraphQL resolver to return an error
	// indicating that the group does not exist.

	testCases := []testCase{
		{
			// When doing DeleteGroup top-down, descendant groups are automatically deleted,
			// so expect an error on non-top-level groups.
			name:             "positive, ascending top-down",
			paths:            ascendingTopPaths,
			expectMsgNonTop:  resourceVersionMismatch,
			expectFinalCount: 0,
		},
		{
			// When doing DeleteGroup bottom-up, expect no errors.
			name:             "positive, descending, bottom-up",
			paths:            descendingPaths,
			expectFinalCount: 0,
		},
		{
			name:             "negative, non-existent ID",
			ids:              []string{nonExistentID},
			expectMsgNonTop:  resourceVersionMismatch,
			expectFinalCount: 6,
		},
		{
			name:             "negative, invalid ID",
			ids:              []string{invalidID},
			expectMsgNonTop:  invalidUUIDMsg1,
			expectFinalCount: 6,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// Because deleting a group is destructive, this test must create/delete
			// the warmup groups inside the loop.
			// Creating the new client deletes the groups from the previous run.
			testClient := newTestClient(ctx, t)
			// Must close manually, because later test cases will create their own test clients.

			createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
			assert.Nil(t, err)
			if err != nil {
				// No point continuing if warmup groups weren't all created.
				return
			}

			groupsToDelete := []*models.Group{}
			switch {
			case test.paths != nil:
				for _, path := range test.paths {
					// Must find the group object in the current generation of warmup groups.
					// Using the initial warmup groups would cause metadata version mismatches.
					var foundGroup *models.Group
					for _, candidate := range createdWarmupGroups {
						if candidate.FullPath == path {
							foundGroup = &candidate
							break
						}
					}
					require.NotNil(t, foundGroup)
					groupsToDelete = append(groupsToDelete, foundGroup)
				}
			case test.ids != nil:
				for _, id := range test.ids {
					groupToDelete := models.Group{
						Metadata: models.ResourceMetadata{
							ID:      id,
							Version: 1,
						},
					}
					groupsToDelete = append(groupsToDelete, &groupToDelete)
				}
			default:
				assert.Nil(t, "each test case must have either paths or ids but not both")
				return
			}

			// Now, try to delete the groups in sequence.
			for _, groupToDelete := range groupsToDelete {
				err = testClient.client.Groups.DeleteGroup(ctx, groupToDelete)

				var expectMsg *string // which error message (or lack thereof) to check for
				if strings.HasPrefix(groupToDelete.Name, "top-") {
					expectMsg = test.expectMsgTop
				} else {
					expectMsg = test.expectMsgNonTop
				}
				checkError(t, expectMsg, err)
			}

			// Check that the final count is right.
			stillAround, err := testClient.client.Groups.GetGroups(ctx, &GetGroupsInput{})
			assert.Nil(t, err)
			assert.Equal(t, test.expectFinalCount, len(stillAround.Groups))

			// Must close manually, because later test cases will create their own test clients.
			testClient.close(ctx)

		})
	}
}

// TestGetGroups tests GetGroups
func TestGetGroups(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups weren't all created.
		return
	}

	allGroupInfos, err := groupInfoFromGroups(ctx, testClient.client.getConnection(ctx), createdWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups weren't all created.
		return
	}

	allPaths := pathsFromGroupInfo(allGroupInfos)
	reversePaths := reverseStringSlice(allPaths)
	allGroupIDs := groupIDsFromGroupInfos(allGroupInfos)
	allNamespaceIDs := namespaceIDsFromGroupInfos(allGroupInfos)

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectPageInfo              PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetGroupsInput
		name                        string
		expectGroupPaths            []string
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		getBeforeCursorFromPrevious bool
		expectHasEndCursor          bool
	}

	testCases := []testCase{

		// nil input causes a nil pointer dereference in GetGroups, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetGroupsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathDesc),
			},
			expectGroupPaths:     reversePaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectGroupPaths: allPaths[:2],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allPaths)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectGroupPaths:           allPaths[2:4],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allPaths)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final two",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectGroupPaths:           allPaths[4:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allPaths)),
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
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectGroupPaths: reversePaths[:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allPaths)),
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
			If it did work, it could be tested by something similar to the following:
			(Not yet fully debugged.)

			{
				name: "pagination: first four to set up for next test",
				input: &GetGroupsInput{
					Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
					PaginationOptions: &PaginationOptions{
						First: ptr.Int32(4),
					},
				},
				expectGroupPaths: allPaths[:4],
				expectPageInfo: PageInfo{
					TotalCount:      int32(len(allPaths)),
					Cursor:          dummyCursorFunc,
					HasNextPage:     true,
					HasPreviousPage: false,
				},
				expectHasStartCursor: true,
				expectHasEndCursor:   true,
			},
			{
				name: "pagination, before and last",
				input: &GetGroupsInput{
					Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
					PaginationOptions: &PaginationOptions{
						Last: ptr.Int32(2),
					},
				},
				getBeforeCursorFromPrevious: true,
				expectGroupPaths:            allPaths[1:3],
				expectPageInfo: PageInfo{
					TotalCount:      int32(len(allPaths)),
					Cursor:          dummyCursorFunc,
					HasNextPage:     true,
					HasPreviousPage: true,
				},
			},

		*/

		{
			name: "pagination, before and after, expect error",
			input: &GetGroupsInput{
				Sort:              ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectGroupPaths:            []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:        ptr.String("only first or last can be defined, not both"),
			expectGroupPaths: allPaths[5:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allPaths)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &GroupFilter{
					RootOnly:     false,
					GroupIDs:     []string{},
					ParentID:     ptr.String(""),
					NamespaceIDs: []string{},
				},
			},
			expectGroupPaths: []string{},
			expectPageInfo:   PageInfo{},
		},

		{
			name: "filter, root only",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					RootOnly: true,
				},
			},
			expectGroupPaths:     []string{"top-level-group-1", "top-level-group-2", "top-level-group-3"},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, empty slice of group IDs",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					GroupIDs: []string{},
				},
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       PageInfo{TotalCount: 6, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group IDs",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					GroupIDs: []string{allGroupIDs[0], allGroupIDs[2], allGroupIDs[4]},
				},
			},
			expectGroupPaths:     []string{allPaths[0], allPaths[2], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group IDs, non-existent",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					GroupIDs: []string{nonExistentID},
				},
			},
			expectGroupPaths:     []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group IDs, invalid",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					GroupIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectGroupPaths:     []string{allPaths[0], allPaths[2], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, parent ID, parent is top, has children",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					ParentID: ptr.String(allGroupIDs[0]),
				},
			},
			expectGroupPaths:     []string{allPaths[1], allPaths[2]},
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, parent ID, parent is sub, has a child",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					ParentID: ptr.String(allGroupIDs[2]),
				},
			},
			expectGroupPaths:     []string{allPaths[3]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, parent ID, parent is top, has none",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					ParentID: ptr.String(allGroupIDs[4]),
				},
			},
			expectGroupPaths:     []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, parent ID, parent is sub, has none",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					ParentID: ptr.String(allGroupIDs[1]),
				},
			},
			expectGroupPaths:     []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, empty slice of namespace IDs",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					NamespaceIDs: []string{},
				},
			},
			expectGroupPaths:     []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: nil},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace IDs",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					NamespaceIDs: []string{allNamespaceIDs[0], allNamespaceIDs[2], allNamespaceIDs[4]},
				},
			},
			expectGroupPaths:     []string{allPaths[0], allPaths[2], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace IDs, non-existent",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					NamespaceIDs: []string{nonExistentID},
				},
			},
			expectGroupPaths:     []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace IDs, invalid",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					NamespaceIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectGroupPaths:     []string{allPaths[0], allPaths[2], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// Combining filter functions appears to do a logical AND when deciding whether to include a result.

		{
			name: "filter, combination, root only and group IDs",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					RootOnly: true,
					GroupIDs: []string{allGroupIDs[3], allGroupIDs[4]},
				},
			},
			expectGroupPaths:     []string{allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// Combining root only and parent ID don't appear to make sense,
		// because a root group won't have a parent ID.

		{
			name: "filter, combination, root only and namespace ids",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					RootOnly:     true,
					NamespaceIDs: allNamespaceIDs[0:5],
				},
			},
			expectGroupPaths:     []string{allPaths[0], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination group ids and parent id",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					GroupIDs: allGroupIDs[2:],
					ParentID: ptr.String(allGroupIDs[0]),
				},
			},
			expectGroupPaths:     []string{allPaths[2]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination, group ids and namespace ids",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					GroupIDs:     allGroupIDs[:4],
					NamespaceIDs: allNamespaceIDs[2:],
				},
			},
			expectGroupPaths:     allPaths[2:4],
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination, parent id and namespace ids",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				Filter: &GroupFilter{
					ParentID:     ptr.String(allGroupIDs[2]),
					NamespaceIDs: allNamespaceIDs[1:5],
				},
			},
			expectGroupPaths:     []string{allPaths[3]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
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

			groupsResult, err := testClient.client.Groups.GetGroups(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, groupsResult.PageInfo)
				assert.NotNil(t, groupsResult.Groups)
				pageInfo := groupsResult.PageInfo
				groups := groupsResult.Groups

				// Check the groups result by comparing a list of the full paths.
				resultPaths := []string{}
				for _, group := range groups {
					resultPaths = append(resultPaths, group.FullPath)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(resultPaths)
				}

				assert.Equal(t, len(test.expectGroupPaths), len(resultPaths))
				assert.Equal(t, test.expectGroupPaths, resultPaths)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one group returned.
				// If there are no groups returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(groups) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&groups[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&groups[len(groups)-1])
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

// TestCreateGroup tests CreateGroup
func TestCreateGroup(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		expectMsg      *string
		name           string
		toCreate       []models.Group
		isPositiveCase bool
		findParentID   bool
	}

	// Test cases are additive in that each test case (tries to) create(s) one or more groups.
	// After each case, the cumulative number of groups and the contents of the groups are verified.
	testCases := []testCase{
		{
			name:           "positive, standard warmup groups",
			isPositiveCase: true,
			toCreate:       standardWarmupGroups,
			// expect message to be nil
		},
		{
			name: "negative, child without parent",
			toCreate: []models.Group{
				{
					Description: "this is a child without a parent",
					ParentID:    invalidID,
					FullPath:    "missing-parent/orphan-child",
					CreatedBy:   "db-integration-tests-test-create-group-case-1",
				},
			},
			expectMsg: ptr.String("invalid id: the id must be a valid uuid"),
		},
		{
			name:         "negative, duplicate top-level",
			findParentID: true,
			toCreate:     []models.Group{standardWarmupGroups[2]},
			expectMsg:    ptr.String("namespace " + standardWarmupGroups[2].FullPath + " already exists"),
		},
		{
			name:         "negative, duplicate sub-group",
			findParentID: true,
			toCreate:     []models.Group{standardWarmupGroups[4]},
			expectMsg:    ptr.String("namespace " + standardWarmupGroups[4].FullPath + " already exists"),
		},
	}

	cumulativeToCreate := []models.Group{}
	cumulativeClaimedCreated := []models.Group{}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			if test.isPositiveCase {

				// Derive the name fields and parent ID fields from the full path.
				test.toCreate = deriveGroupNames(test.toCreate)

				// Capture any that should be created.
				cumulativeToCreate = append(cumulativeToCreate, test.toCreate...)

				// For a positive case, just use the utility function.
				claimedCreated, _, err := createInitialGroups(ctx, testClient, test.toCreate)
				assert.Nil(t, err)
				if err != nil {
					// No point if warmup groups weren't all created.
					return
				}

				// Capture any new claimed to be created.
				cumulativeClaimedCreated = append(cumulativeClaimedCreated, claimedCreated...)
			} else {

				// For a negative case, do each group one at a time.
				for _, input := range test.toCreate {

					if test.findParentID {

						// Must find the parent's ID based on the full path.
						parentPath := fullPath2ParentPath(input.FullPath)
						if parentPath == "" {

							// input must be top-level
							input.ParentID = ""
						} else {

							// input must be not top-level, so look up the parent's ID.
							parent, err := testClient.client.Groups.GetGroupByFullPath(ctx, parentPath)
							assert.Nil(t, err)
							if err != nil {
								// No point continuing if things are falling apart.
								return
							}

							input.ParentID = parent.Metadata.ID
						}
					}

					// Must also derive the name from the full path.
					input.Name = fullPath2Name(input.FullPath)

					_, err := testClient.client.Groups.CreateGroup(ctx, &input)
					require.NotNil(t, err)
					assert.Equal(t, *test.expectMsg, err.Error())

				}
			}

			// Get the groups for comparison.
			gotResult, err := testClient.client.Groups.GetGroups(ctx, &GetGroupsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			})
			assert.Nil(t, err)
			if err != nil {
				// No point if we couldn't retrieve the groups.
				return
			}
			retrievedGroups := gotResult.Groups

			// Compare lengths.
			assert.Equal(t, len(cumulativeToCreate), len(cumulativeClaimedCreated))
			assert.Equal(t, len(cumulativeToCreate), len(retrievedGroups))

			// Compare the contents of the cumulative groups to create, the groups reportedly created,
			// and the groups retrieved.  The three slices aren't guaranteed to be in the same order,
			// so it is necessary to look them up.  FullPath is used for that purpose.
			copyCumulativeClaimedCreated := cumulativeClaimedCreated
			for _, toldToCreate := range cumulativeToCreate {
				var claimedCreated, retrieved models.Group

				claimedCreated, copyCumulativeClaimedCreated, err = removeMatching(toldToCreate.FullPath, copyCumulativeClaimedCreated)
				assert.Nil(t, err)
				if err != nil {
					break
				}

				retrieved, retrievedGroups, err = removeMatching(toldToCreate.FullPath, retrievedGroups)
				assert.Nil(t, err)
				if err != nil {
					break
				}

				compareGroupsCreate(t, toldToCreate, claimedCreated)
				compareGroupsCreate(t, toldToCreate, retrieved)
			}

			// Must not have any leftovers.
			assert.Equal(t, 0, len(copyCumulativeClaimedCreated))
			assert.Equal(t, 0, len(retrievedGroups))
		})
	}
}

// TestUpdateGroup tests UpdateGroup
func TestUpdateGroup(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups weren't all created.
		return
	}

	// In a given test case, use exactly one of findFullPath and findGroup.
	type testCase struct {
		findFullPath   *string
		findGroup      *models.Group
		newDescription *string
		expectMsg      *string
		name           string
		isPositive     bool
	}

	testCases := []testCase{}
	for _, positiveGroup := range createdWarmupGroups {
		testCases = append(testCases, testCase{
			name:           "positive-" + positiveGroup.FullPath,
			findFullPath:   &positiveGroup.FullPath,
			newDescription: updateDescription(positiveGroup.Description),
			isPositive:     true,
		})
	}

	testCases = append(testCases,
		testCase{
			name: "negative, not exist, top-level",
			findGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: 1,
				},
				Name:        "top-level-group-that-does-not-exist",
				Description: "This is the top-level group that does not exist.",
				FullPath:    "top-level-group-that-does-not-exist",
				CreatedBy:   "someone",
			},
			newDescription: ptr.String("Update description for a group that does not exist."),
			expectMsg:      resourceVersionMismatch,
		},
		testCase{
			name: "negative, not exist, sub-group",
			findGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: 1,
				},
				Name:        "group-that-does-not-exist",
				Description: "This is the sub-group that does not exist.",
				ParentID:    createdWarmupGroups[0].Metadata.ID,
				FullPath:    fmt.Sprintf("%s/group-that-does-not-exist", createdWarmupGroups[0].FullPath),
				CreatedBy:   "someone else",
			},
			newDescription: ptr.String("Update description for a group that does not exist."),
			expectMsg:      resourceVersionMismatch,
		},
		testCase{
			name: "negative, invalid uuid, top-level",
			findGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: 1,
				},
				Name:        "top-level-group-that-does-not-exist",
				Description: "This is the top-level group that does not exist.",
				FullPath:    "top-level-group-that-does-not-exist",
				CreatedBy:   "someone",
			},
			newDescription: ptr.String("Update description for a group that does not exist."),
			expectMsg:      invalidUUIDMsg1,
		},
		testCase{
			name: "negative, invalid uuid, sub-group",
			findGroup: &models.Group{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: 1,
				},
				Name:        "group-that-does-not-exist",
				Description: "This is the sub-group that does not exist.",
				ParentID:    createdWarmupGroups[0].Metadata.ID,
				FullPath:    fmt.Sprintf("%s/group-that-does-not-exist", createdWarmupGroups[0].FullPath),
				CreatedBy:   "someone else",
			},
			newDescription: ptr.String("Update description for a group that does not exist."),
			expectMsg:      invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			var originalGroup *models.Group
			switch {

			case (test.findFullPath != nil) && (test.findGroup == nil):
				originalGroup, err = testClient.client.Groups.GetGroupByFullPath(ctx, *test.findFullPath)
				require.Nil(t, err)
				require.NotNil(t, originalGroup)

			case (test.findFullPath == nil) && (test.findGroup != nil):
				originalGroup = test.findGroup

			default:
				assert.Equal(t, "test case should use exactly one of findFullPath, findGroup",
					"test case violated that rule")
				// No point in going forward with this test case.
				return
			}

			expectUpdatedDescription := updateDescription(originalGroup.Description)
			copyOriginalGroup := originalGroup
			copyOriginalGroup.Description = *expectUpdatedDescription

			claimedUpdatedGroup, err := testClient.client.Groups.UpdateGroup(ctx, copyOriginalGroup)

			checkError(t, test.expectMsg, err)

			if test.isPositive {

				require.NotNil(t, claimedUpdatedGroup)

				compareGroupsUpdate(t, *expectUpdatedDescription, *originalGroup, *claimedUpdatedGroup)

				retrieved, err := testClient.client.Groups.GetGroupByFullPath(ctx, originalGroup.FullPath)
				require.Nil(t, err)

				require.NotNil(t, retrieved)

				compareGroupsUpdate(t, *expectUpdatedDescription, *originalGroup, *retrieved)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup groups for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroups = []models.Group{
	{
		Description: "top level group 1 for testing group functions",
		FullPath:    "top-level-group-1",
		CreatedBy:   "someone",
	},
	{
		Description: "top level group 2 for testing group functions",
		FullPath:    "top-level-group-2",
		CreatedBy:   "someone-else",
	},
	{
		Description: "top level group 3 for testing group functions",
		FullPath:    "top-level-group-3",
		CreatedBy:   "yet-another-someone",
	},
	{
		Description: "second level group 1a for testing group functions",
		FullPath:    "top-level-group-1/2nd-level-group-1a",
		CreatedBy:   "someone-lower",
	},
	{
		Description: "second level group 1b for testing group functions",
		FullPath:    "top-level-group-1/2nd-level-group-1b",
		CreatedBy:   "someone-else-lower",
	},
	{
		Description: "third level group 1b1 for testing group functions",
		FullPath:    "top-level-group-1/2nd-level-group-1b/3rd-level-group-1b1",
		CreatedBy:   "someone-third",
	},
}

// Fix up the list: derive each name field from the full path field.
func deriveGroupNames(input []models.Group) []models.Group {
	output := []models.Group{}

	for _, inp := range input {
		inp.Name = fullPath2Name(inp.FullPath)
		output = append(output, inp)
	}

	return output
}

func ptrGroupSortableField(arg GroupSortableField) *GroupSortableField {
	return &arg
}

func (gis groupInfoSlice) Len() int {
	return len(gis)
}

func (gis groupInfoSlice) Swap(i, j int) {
	gis[i], gis[j] = gis[j], gis[i]
}

func (gis groupInfoSlice) Less(i, j int) bool {
	return gis[i].fullPath < gis[j].fullPath
}

// groupInfoFromGroups returns a slice of groupInfo, sorted by full path.
func groupInfoFromGroups(ctx context.Context, conn connection, groups []models.Group) ([]groupInfo, error) {
	result := []groupInfo{}
	for _, group := range groups {

		namespaceRow, err := getNamespaceByGroupID(ctx, conn, group.Metadata.ID)
		if err != nil {
			return nil, err
		}

		result = append(result, groupInfo{
			fullPath:    group.FullPath,
			groupID:     group.Metadata.ID,
			namespaceID: namespaceRow.id,
		})
	}

	sort.Sort(groupInfoSlice(result))

	return result, nil
}

// pathsFromGroupInfo preserves order
func pathsFromGroupInfo(groupInfos []groupInfo) []string {
	result := []string{}
	for _, groupInfo := range groupInfos {
		result = append(result, groupInfo.fullPath)
	}
	return result
}

// groupIDsFromGroupInfos preserves order
func groupIDsFromGroupInfos(groupInfos []groupInfo) []string {
	result := []string{}
	for _, groupInfo := range groupInfos {
		result = append(result, groupInfo.groupID)
	}
	return result
}

// namespaceIDsFromGroupInfos preserves order
func namespaceIDsFromGroupInfos(groupInfos []groupInfo) []string {
	result := []string{}
	for _, groupInfo := range groupInfos {
		result = append(result, groupInfo.namespaceID)
	}
	return result
}

// removeMatching finds a group if a slice with a specified full path.
// It returns the found group, a shortened slice, and an error.
func removeMatching(lookFor string, oldSlice []models.Group) (models.Group, []models.Group, error) {
	found := models.Group{}
	foundOne := false
	newSlice := []models.Group{}

	for _, candidate := range oldSlice {
		if candidate.FullPath == lookFor {
			found = candidate
			foundOne = true
		} else {
			newSlice = append(newSlice, candidate)
		}
	}

	if !foundOne {
		return models.Group{}, nil, fmt.Errorf("Failed to find group with full path: %s", lookFor)
	}

	return found, newSlice, nil
}

// compareGroupsCreate compares two groups for TestCreateGroup
// There are some fields that cannot be compared, or DeepEqual would work.
func compareGroupsCreate(t *testing.T, expected, actual models.Group) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.FullPath, actual.FullPath)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
}

// updateDescription takes an original description and returns a modified version for TestUpdateGroup
func updateDescription(input string) *string {
	return ptr.String(fmt.Sprintf("Updated description: %s", input))
}

// compareGroupsUpdate compares two groups for TestUpdateGroup
func compareGroupsUpdate(t *testing.T, expectedDescription string, expected, actual models.Group) {

	assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
	assert.NotEqual(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEqual(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)

	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expectedDescription, actual.Description)
	assert.Equal(t, expected.FullPath, actual.FullPath)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
}

// The End.
