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
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

var (
	// returned for some other invalid UUID cases
	invalidUUIDMsg3 = ptr.String("invalid id: the id must be a valid uuid")

	// returned for some non-existent parent group cases
	nonExistParentGroupMsg1 = ptr.String("invalid group parent: the specified parent group does not exist")
)

// workspaceInfo aids convenience in accessing the information TestGetWorkspaces needs about the warmup workspaces.
type workspaceInfo struct {
	updateTime  time.Time
	fullPath    string
	workspaceID string
}

// workspaceInfoPathSlice makes a slice of workspaceInfo sortable by full path
type workspaceInfoPathSlice []workspaceInfo

// workspaceInfoTimeSlice makes a slice of workspaceInfo sortable by last updated time
type workspaceInfoTimeSlice []workspaceInfo

func TestGetWorkspaceByFullPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, createdWarmupWorkspaces, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces, standardWarmupWorkspaces)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}

	type testCase struct {
		expectMsg       *string
		name            string
		searchPath      string
		expectWorkspace bool
	}

	testCases := []testCase{}
	for _, positiveWorkspace := range createdWarmupWorkspaces {
		testCases = append(testCases, testCase{
			name:            "positive-" + positiveWorkspace.FullPath,
			searchPath:      positiveWorkspace.FullPath,
			expectWorkspace: true,
		})
	}

	testCases = append(testCases,
		testCase{
			name:       "negative, non-existent first-level workspace",
			searchPath: "non-existent-first-level-workspace",
			// expect workspace and error to be nil
		},
		testCase{
			name:       "negative, non-existent second-level workspace",
			searchPath: "top-level-group-1-for-workspaces/non-existent-2nd-level-group",
			// expect workspace and error to be nil
		},
		testCase{
			name:       "defective-path",
			searchPath: "this*is*a*not*a*valid*path",
			// expect workspace and error to be nil
			// At the DB layer, the search path is just looked up, with no workspace returned.
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			workspace, err := testClient.client.Workspaces.GetWorkspaceByFullPath(ctx, test.searchPath)

			checkError(t, test.expectMsg, err)

			if test.expectWorkspace {
				// the positive case
				require.NotNil(t, workspace)
				assert.Equal(t, test.searchPath, workspace.FullPath)
			} else {
				// the negative and defective cases
				assert.Nil(t, workspace)
			}
		})
	}
}

func TestGetWorkspaceByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a workspace with a specific ID without going into the really
	// low-level stuff, create the warmup workspace(s) by name and then find the relevant ID.
	_, createdWarmupWorkspaces, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces, standardWarmupWorkspaces)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}

	type testCase struct {
		expectMsg       *string
		name            string
		searchID        string
		expectWorkspace bool
	}

	testCases := []testCase{}
	for _, positiveWorkspace := range createdWarmupWorkspaces {
		testCases = append(testCases, testCase{
			name:            "positive-" + positiveWorkspace.FullPath,
			searchID:        positiveWorkspace.Metadata.ID,
			expectWorkspace: true,
		})
	}

	testCases = append(testCases,
		testCase{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect workspace and error to be nil
		},
		testCase{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			workspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectWorkspace {
				// the positive case
				require.NotNil(t, workspace)
				assert.Equal(t, test.searchID, workspace.Metadata.ID)
			} else {
				// the negative and defective cases
				assert.Nil(t, workspace)
			}
		})
	}
}

func TestGetWorkspaces(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, createdWarmupWorkspaces, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces, standardWarmupWorkspaces)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}

	allGroupInfos, err := groupInfoFromGroups(ctx, testClient.client.getConnection(ctx), createdWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}
	allGroupIDs := groupIDsFromGroupInfos(allGroupInfos)
	allWorkspaceInfos := workspaceInfoFromWorkspaces(createdWarmupWorkspaces)

	// Sort by full paths.
	sort.Sort(workspaceInfoPathSlice(allWorkspaceInfos))
	allPaths := pathsFromWorkspaceInfo(allWorkspaceInfos)
	reversePaths := reverseStringSlice(allPaths)
	allWorkspaceIDs := workspaceIDsFromWorkspaceInfos(allWorkspaceInfos)

	// Sort by last update times.
	sort.Sort(workspaceInfoTimeSlice(allWorkspaceInfos))
	allPathsByTime := pathsFromWorkspaceInfo(allWorkspaceInfos)
	reversePathsByTime := reverseStringSlice(allPathsByTime)

	// Some teams for namespace memberships based on team member relationships.
	// Please note the difference in numbering between the teams for groups vs. those for workspaces.
	teamIDs := []*string{}
	for _, toCreateTeam := range []models.Team{
		{
			Name:        "teamG0",
			Description: "team for group 0 for testing the workspace DB layer",
		},
		{
			Name:        "teamG1",
			Description: "team for group 1 for testing the workspace DB layer",
		},
		{
			Name:        "teamG2",
			Description: "team for group 2 for testing the workspace DB layer",
		},
		{
			Name:        "teamW1",
			Description: "team for workspace-1 for testing the workspace DB layer",
		},
		{
			Name:        "teamW2",
			Description: "team for workspace-2 for testing the workspace DB layer",
		},
		{
			Name:        "teamW3",
			Description: "team for workspace-3 for testing the workspace DB layer",
		},
		{
			Name:        "teamW4",
			Description: "team for workspace-4 for testing the workspace DB layer",
		},
		{
			Name:        "teamW5",
			Description: "team for workspace-5 for testing the workspace DB layer",
		},
	} {
		newTeam, err := testClient.client.Teams.CreateTeam(ctx, &toCreateTeam)
		assert.Nil(t, err)
		if err != nil {
			// No point in continuing if teams weren't created as designed.
			return
		}
		teamIDs = append(teamIDs, &newTeam.Metadata.ID)
	}

	// Some users for namespace memberships.
	userMemberIDs := []*string{}
	for _, toCreateUser := range []models.User{
		{
			Username: "userMember0",
			Email:    "user-member-0@example.com",
		},
		{
			Username: "userMember1",
			Email:    "user-member-1@example.com",
		},
		{
			Username: "userNotMember",
			Email:    "user-not-member@example.com",
		},
		{
			Username: "userMember3",
			Email:    "user-member-3@example.com",
		},
		{
			Username: "userMember4",
			Email:    "user-member-4@example.com",
		},
	} {
		newUser, err := testClient.client.Users.CreateUser(ctx, &toCreateUser)
		assert.Nil(t, err)
		if err != nil {
			// No point in continuing if users weren't created as designed.
			return
		}
		userMemberIDs = append(userMemberIDs, &newUser.Metadata.ID)
	}

	// Some team member relationships.
	// Apparently, the team member ID is not needed later.
	for _, toCreateTeamMember := range []models.TeamMember{
		{
			UserID: *userMemberIDs[3],
			TeamID: *teamIDs[0],
		},
		{
			UserID: *userMemberIDs[3],
			TeamID: *teamIDs[2],
		},
		{
			UserID: *userMemberIDs[3],
			TeamID: *teamIDs[4],
		},
		{
			UserID: *userMemberIDs[4],
			TeamID: *teamIDs[1],
		},
		{
			UserID: *userMemberIDs[4],
			TeamID: *teamIDs[3],
		},
		{
			UserID: *userMemberIDs[4],
			TeamID: *teamIDs[5],
		},
	} {
		_, err := testClient.client.TeamMembers.AddUserToTeam(ctx, &toCreateTeamMember)
		assert.Nil(t, err)
		if err != nil {
			// No point in continuing if team member relationships weren't created as designed.
			return
		}
	}

	// Some service accounts for namespace memberships.
	serviceAccountMemberIDs := []*string{}
	for _, toCreateServiceAccount := range []models.ServiceAccount{
		{
			Name:    "serviceAccount0",
			GroupID: createdWarmupGroups[0].Metadata.ID,
		},
		{
			Name:    "serviceAccount1",
			GroupID: createdWarmupGroups[1].Metadata.ID,
		},
		{
			Name:    "serviceAccountNotMember",
			GroupID: createdWarmupGroups[2].Metadata.ID,
		},
	} {
		newServiceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &toCreateServiceAccount)
		assert.Nil(t, err)
		if err != nil {
			// No point in continuing if service accounts weren't created.
			return
		}
		serviceAccountMemberIDs = append(serviceAccountMemberIDs, &newServiceAccount.Metadata.ID)
	}

	// Add namespace memberships.

	// add user member 0 to allPaths 1, 2, 3
	// add user member 1 to allPaths 2, 3, 4
	// user member 2 to nothing
	for ix, userMemberID := range userMemberIDs[0:2] {
		for ix2 := ix + 1; ix2 <= ix+3; ix2++ {
			_, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx,
				&CreateNamespaceMembershipInput{
					NamespacePath: allPaths[ix2],
					UserID:        userMemberID,
				})
			assert.Nil(t, err)
			if err != nil {
				// No point in continuing if namespace memberships weren't created.
				return
			}
		}
	}

	// add service account member 0 (group 0) to allPaths 0, 1
	// add service account member 1 (group 1) to allPaths 2, 3
	// service account member 2 to nothing
	for ix, serviceAccountMemberID := range serviceAccountMemberIDs[0:2] {
		for ix2 := 2 * ix; ix2 <= (2*ix)+1; ix2++ {
			_, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx,
				&CreateNamespaceMembershipInput{
					NamespacePath:    allPaths[ix2],
					ServiceAccountID: serviceAccountMemberID,
				})
			assert.Nil(t, err)
			if err != nil {
				// No point in continuing if namespace memberships weren't created.
				return
			}
		}
	}

	// add namespace memberships for teams.
	for _, nsm := range []CreateNamespaceMembershipInput{
		{
			NamespacePath: "top-level-group-0-for-workspaces",
			TeamID:        teamIDs[0], // g0
			Role:          models.ViewerRole,
		},
		{
			NamespacePath: "top-level-group-1-for-workspaces",
			TeamID:        teamIDs[1], // g1
			Role:          models.DeployerRole,
		},
		{
			NamespacePath: "top-level-group-2-for-workspaces",
			TeamID:        teamIDs[2], // g2
			Role:          models.OwnerRole,
		},
		{
			NamespacePath: "top-level-group-0-for-workspaces/workspace-1",
			TeamID:        teamIDs[3], // w1
			Role:          models.OwnerRole,
		},
		{
			NamespacePath: "top-level-group-1-for-workspaces/workspace-5",
			TeamID:        teamIDs[4], // w5
			Role:          models.DeployerRole,
		},
		{
			NamespacePath: "top-level-group-2-for-workspaces/workspace-3",
			TeamID:        teamIDs[5], // w3
			Role:          models.ViewerRole,
		},
		{
			NamespacePath: "top-level-group-0-for-workspaces/workspace-4",
			TeamID:        teamIDs[6], // w4
			Role:          models.OwnerRole,
		},
		{
			NamespacePath: "top-level-group-1-for-workspaces/workspace-2",
			TeamID:        teamIDs[7], // w2
			Role:          models.DeployerRole,
		},
	} {
		_, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &nsm)
		assert.Nil(t, err)
		if err != nil {
			// No point in continuing if namespace memberships weren't created.
			return
		}
	}

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectPageInfo              PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetWorkspacesInput
		name                        string
		expectWorkspacePaths        []string
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		getBeforeCursorFromPrevious bool
		expectHasEndCursor          bool
	}

	testCases := []testCase{

		// nil input causes a nil pointer dereference in GetWorkspaces, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetWorkspacesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectWorkspacePaths: allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectWorkspacePaths: allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of full path",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
			},
			expectWorkspacePaths: allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of full path",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathDesc),
			},
			expectWorkspacePaths: reversePaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldUpdatedAtAsc),
			},
			expectWorkspacePaths: allPathsByTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPathsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldUpdatedAtDesc),
			},
			expectWorkspacePaths: reversePathsByTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPathsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectWorkspacePaths: allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectWorkspacePaths: allPaths[:2],
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
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectWorkspacePaths:       allPaths[2:4],
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
			name: "pagination: final one",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectWorkspacePaths:       allPaths[4:],
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
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectWorkspacePaths: reversePaths[:3],
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
			If it did work, it could be tested by adapting the test cases corresponding to the
			next few cases after a similar block of text from group_test.go

		*/

		{
			name: "pagination, before and after, expect error",
			input: &GetWorkspacesInput{
				Sort:              ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectWorkspacePaths:        []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:            ptr.String("only first or last can be defined, not both"),
			expectWorkspacePaths: allPaths[4:],
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
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &WorkspaceFilter{
					WorkspaceIDs:           []string{},
					GroupID:                ptr.String(""),
					UserMemberID:           ptr.String(""),
					ServiceAccountMemberID: ptr.String(""),
					Search:                 ptr.String(""),
				},
			},
			expectMsg:            emptyUUIDMsg2,
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{},
		},

		{
			name: "filter, empty slice of workspace IDs",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs: []string{},
				},
			},
			expectWorkspacePaths: allPaths,
			expectPageInfo:       PageInfo{TotalCount: 5, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace IDs",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs: []string{allWorkspaceIDs[0], allWorkspaceIDs[2], allWorkspaceIDs[4]},
				},
			},
			expectWorkspacePaths: []string{allPaths[0], allPaths[2], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace IDs, non-existent",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs: []string{nonExistentID},
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace IDs, invalid",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs: []string{invalidID},
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, positive",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					GroupID: ptr.String(allGroupIDs[0]),
				},
			},
			expectWorkspacePaths: []string{allPaths[0], allPaths[1]},
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{},
		},

		{
			name: "filter, user member ID, positive 0",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: userMemberIDs[0],
				},
			},
			expectWorkspacePaths: allPaths[1:4],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user member ID, positive 1",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: userMemberIDs[1],
				},
			},
			expectWorkspacePaths: allPaths[2:5],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user member ID, exists, not a member of any namespace",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: userMemberIDs[2],
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, user member ID, non-existent",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: ptr.String(nonExistentID),
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, user member ID, invalid",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: ptr.String(invalidID),
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{},
		},

		/*

		   Explanation of expected results for users 3 and 4 for team member effects:

		   userMember3 is in teams 0, 2, 4
		   userMember4 is in teams 1, 3, 5

		   team 0 is viewer of g0
		   team 1 is deployer of g1
		   team 2 is owner of g2
		   team 3 is owner of w1
		   team 4 is deployer of w5
		   team 5 is viewer of w3
		   team 6 is owner of w4
		   team 7 is deployer of w2

		   g0 has w1, w4
		   g1 has w5, w2
		   g2 has w3

		   u3 should be in names w1, w4, w3, w5
		   u4 should be in names w5, w2, w1, w3

		   "top-level-group-0-for-workspaces/workspace-1"
		   "top-level-group-0-for-workspaces/workspace-4"
		   "top-level-group-1-for-workspaces/workspace-2"
		   "top-level-group-1-for-workspaces/workspace-5"
		   "top-level-group-2-for-workspaces/workspace-3"

		   u3 sorted should be in names w1, w4, w5, w3
		   u4 sorted should be in names w1, w2, w5, w3

		*/

		{
			name: "filter, user member ID, positive 3 to catch team member effects",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: userMemberIDs[3],
				},
			},
			expectWorkspacePaths: []string{
				"top-level-group-0-for-workspaces/workspace-1",
				"top-level-group-0-for-workspaces/workspace-4",
				"top-level-group-1-for-workspaces/workspace-5",
				"top-level-group-2-for-workspaces/workspace-3",
			},
			expectPageInfo:       PageInfo{TotalCount: 4, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user member ID, positive 4 to catch team member effects",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID: userMemberIDs[4],
				},
			},
			expectWorkspacePaths: []string{
				"top-level-group-0-for-workspaces/workspace-1",
				"top-level-group-1-for-workspaces/workspace-2",
				"top-level-group-1-for-workspaces/workspace-5",
				"top-level-group-2-for-workspaces/workspace-3",
			},
			expectPageInfo:       PageInfo{TotalCount: 4, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account member ID, positive 0",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					ServiceAccountMemberID: serviceAccountMemberIDs[0],
				},
			},
			expectWorkspacePaths: allPaths[0:2],
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account member ID, positive 1",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					ServiceAccountMemberID: serviceAccountMemberIDs[1],
				},
			},
			expectWorkspacePaths: allPaths[2:4],
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account member ID, exists, not a member of any namespace",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					ServiceAccountMemberID: serviceAccountMemberIDs[2],
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, service account member ID, non-existent",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					ServiceAccountMemberID: ptr.String(nonExistentID),
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, service account member ID, invalid",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					ServiceAccountMemberID: ptr.String(invalidID),
				},
			},
			expectMsg:            invalidUUIDMsg2,
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{},
		},

		{
			name: "filter, search field, empty string",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					Search: ptr.String(""),
				},
			},
			expectWorkspacePaths: allPaths,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 1",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					Search: ptr.String("1"),
				},
			},
			expectWorkspacePaths: []string{allPaths[0], allPaths[2], allPaths[3]},
			expectPageInfo:       PageInfo{TotalCount: int32(3), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 2",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					Search: ptr.String("2"),
				},
			},
			expectWorkspacePaths: []string{allPaths[2], allPaths[4]},
			expectPageInfo:       PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, 5",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					Search: ptr.String("5"),
				},
			},
			expectWorkspacePaths: allPaths[3:4],
			expectPageInfo:       PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectWorkspacePaths: []string{},
			expectPageInfo:       PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// Combining filter functions does a logical AND when deciding whether to include a result.
		// Because there are so many filter fields, do a few combinations but not all possible.

		{
			name: "filter, combination workspace IDs and group ID",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs: allWorkspaceIDs[0:3],
					GroupID:      &allGroupIDs[1],
				},
			},
			expectWorkspacePaths: []string{allPaths[2]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination workspace IDs and user member ID",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs: allWorkspaceIDs[:4],
					UserMemberID: userMemberIDs[1],
				},
			},
			expectWorkspacePaths: allPaths[2:4],
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination workspace IDs and service account member ID",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					WorkspaceIDs:           allWorkspaceIDs[1:5],
					ServiceAccountMemberID: serviceAccountMemberIDs[0],
				},
			},
			expectWorkspacePaths: []string{allPaths[1]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination group ID and search",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					GroupID: &allGroupIDs[0],
					Search:  ptr.String("4"),
				},
			},
			expectWorkspacePaths: []string{allPaths[1]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination user member ID and service account member ID",
			input: &GetWorkspacesInput{
				Sort: ptrWorkspaceSortableField(WorkspaceSortableFieldFullPathAsc),
				Filter: &WorkspaceFilter{
					UserMemberID:           userMemberIDs[0],
					ServiceAccountMemberID: serviceAccountMemberIDs[0],
				},
			},
			expectWorkspacePaths: []string{allPaths[1]},
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

			workspacesResult, err := testClient.client.Workspaces.GetWorkspaces(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, workspacesResult.PageInfo)
				assert.NotNil(t, workspacesResult.Workspaces)
				pageInfo := workspacesResult.PageInfo
				workspaces := workspacesResult.Workspaces

				// Check the workspaces result by comparing a list of the full paths.
				resultPaths := []string{}
				for _, workspace := range workspaces {
					resultPaths = append(resultPaths, workspace.FullPath)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(resultPaths)
				}

				assert.Equal(t, len(test.expectWorkspacePaths), len(resultPaths))
				assert.Equal(t, test.expectWorkspacePaths, resultPaths)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one workspace returned.
				// If there are no workspaces returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(workspaces) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&workspaces[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&workspaces[len(workspaces)-1])
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

func TestUpdateWorkspace(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, createdWarmupWorkspaces, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces, standardWarmupWorkspaces)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}

	type testCase struct {
		toUpdate      *models.Workspace
		expectUpdated *models.Workspace
		expectMsg     *string
		name          string
	}

	testCases := []testCase{}
	for ix, positiveWorkspace := range createdWarmupWorkspaces {
		now := currentTime()
		newDescription := fmt.Sprintf("updated description: %s", positiveWorkspace.Description)

		newJobID, newStateVersionID, err := createJobStateVersion(ctx, testClient.client, positiveWorkspace.Metadata.ID)
		assert.Nil(t, err)
		if err != nil {
			// No point continuing.
			return
		}

		newDirtyState := true
		newMaxJobDuration := ptr.Int32(int32(400 + (100 * ix)))

		// Test all combinations of old vs. new PreventDestroyPlan values.
		// The old value is true for ix values of 1 and 3.
		newPreventDestroyPlan := (ix == 2) || (ix == 3)

		testCases = append(testCases, testCase{
			name: "positive-" + positiveWorkspace.FullPath,
			toUpdate: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID:      positiveWorkspace.Metadata.ID,
					Version: positiveWorkspace.Metadata.Version,
				},
				Description:           newDescription,
				CurrentJobID:          newJobID,
				CurrentStateVersionID: newStateVersionID,
				DirtyState:            newDirtyState,
				MaxJobDuration:        newMaxJobDuration,
				PreventDestroyPlan:    newPreventDestroyPlan,
			},
			expectUpdated: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID:                   positiveWorkspace.Metadata.ID,
					Version:              positiveWorkspace.Metadata.Version + 1,
					CreationTimestamp:    positiveWorkspace.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Name:                  positiveWorkspace.Name,
				FullPath:              positiveWorkspace.FullPath,
				GroupID:               positiveWorkspace.GroupID,
				Description:           newDescription,
				CurrentJobID:          newJobID,
				CurrentStateVersionID: newStateVersionID,
				DirtyState:            newDirtyState,
				MaxJobDuration:        newMaxJobDuration,
				CreatedBy:             positiveWorkspace.CreatedBy,
				PreventDestroyPlan:    newPreventDestroyPlan,
			},
		})
	}

	testCases = append(testCases,
		testCase{
			name: "negative, non-existent ID",
			toUpdate: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				Description: "looking for a non-existent ID",
			},
			expectMsg: resourceVersionMismatch,
			// expect workspace to be nil
		},
		testCase{
			name: "defective-id",
			toUpdate: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
				Description: "looking for a defective ID",
			},
			expectMsg: invalidUUIDMsg1,
			// expect workspace to be nil
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualUpdated, err := testClient.client.Workspaces.UpdateWorkspace(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// the positive case
				require.NotNil(t, actualUpdated)

				// The update process must leave creation timestamp alone
				// and must make the last updated timestamp between when
				// the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()
				compareWorkspaces(t, test.expectUpdated, actualUpdated, true, timeBounds{
					createLow:  whenCreated,
					createHigh: whenCreated,
					updateLow:  test.expectUpdated.Metadata.LastUpdatedTimestamp,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualUpdated)
			}
		})
	}
}

func TestCreateWorkspace(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create only one warmup group and _NOT_ any warmup workspaces.
	createdWarmupGroups, _, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces[:1], []models.Workspace{})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup group wasn't created.
		return
	}

	require.Equal(t, 1, len(createdWarmupGroups))

	warmupGroup0 := createdWarmupGroups[0]
	warmupGroupName := warmupGroup0.Name
	warmupGroupID := warmupGroup0.Metadata.ID
	defaultJobDuration := int32((time.Hour * 12).Minutes()) // defined in service layer, so not readily available

	type testCase struct {
		toCreate      *models.Workspace
		expectCreated *models.Workspace
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{

		{
			name: "positive empty",
			toCreate: &models.Workspace{
				Name:           "empty-workspace",
				GroupID:        warmupGroupID,
				Description:    "this is an almost empty workspace",
				MaxJobDuration: &defaultJobDuration,
				CreatedBy:      "empty-workspace-creator",
			},
			expectCreated: &models.Workspace{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:           "empty-workspace",
				FullPath:       warmupGroupName + "/empty-workspace",
				GroupID:        warmupGroupID,
				Description:    "this is an almost empty workspace",
				MaxJobDuration: &defaultJobDuration,
				CreatedBy:      "empty-workspace-creator",
			},
		},

		{
			// It's not possible to directly create a workspace with a job or state version,
			// because the workspace ID is required before the job and state version can be created.
			name: "positive full",
			toCreate: &models.Workspace{
				Name:           "full-workspace",
				GroupID:        warmupGroupID,
				Description:    "this is a full workspace",
				DirtyState:     true,
				MaxJobDuration: ptr.Int32(954),
				CreatedBy:      "full-workspace-creator",
			},
			expectCreated: &models.Workspace{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:                  "full-workspace",
				FullPath:              warmupGroupName + "/full-workspace",
				GroupID:               warmupGroupID,
				Description:           "this is a full workspace",
				CurrentJobID:          "",
				CurrentStateVersionID: "",
				DirtyState:            true,
				MaxJobDuration:        ptr.Int32(954),
				CreatedBy:             "full-workspace-creator",
			},
		},

		{
			name: "duplicate will fail",
			toCreate: &models.Workspace{
				Name:           "full-workspace",
				GroupID:        warmupGroupID,
				Description:    "this would be a full workspace, but it's a duplicate",
				DirtyState:     true,
				MaxJobDuration: ptr.Int32(954),
				CreatedBy:      "full-workspace-creator",
			},
			expectMsg: ptr.String("namespace top-level-group-0-for-workspaces/full-workspace already exists"),
		},

		{
			name: "non-existent group ID",
			toCreate: &models.Workspace{
				Name:           "non-existent-group-workspace",
				GroupID:        nonExistentID,
				Description:    "this would be a workspace, except the group does not exist",
				MaxJobDuration: &defaultJobDuration,
				CreatedBy:      "non-existent-group-workspace-creator",
			},
			expectMsg: nonExistParentGroupMsg1,
		},

		{
			name: "defective group ID",
			toCreate: &models.Workspace{
				Name:           "invalid-group-id-workspace",
				GroupID:        invalidID,
				Description:    "this would be a workspace, except the group's ID is defective",
				MaxJobDuration: &defaultJobDuration,
				CreatedBy:      "defective-group-id-workspace-creator",
			},
			expectMsg: invalidUUIDMsg3,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.Workspaces.CreateWorkspace(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareWorkspaces(t, test.expectCreated, actualCreated, false, timeBounds{
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

func TestDeleteWorkspace(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	_, createdWarmupWorkspaces, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces, standardWarmupWorkspaces)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}

	type testCase struct {
		toDelete  *models.Workspace
		expectMsg *string
		name      string
	}

	testCases := []testCase{}
	for _, positiveWorkspace := range createdWarmupWorkspaces {
		testCases = append(testCases, testCase{
			name: "positive-" + positiveWorkspace.FullPath,
			toDelete: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID:      positiveWorkspace.Metadata.ID,
					Version: positiveWorkspace.Metadata.Version,
				},
			},
		})
	}

	testCases = append(testCases,
		testCase{
			name: "negative, non-existent ID",
			toDelete: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				Description: "looking for a non-existent ID",
			},
			expectMsg: resourceVersionMismatch,
		},
		testCase{
			name: "defective-id",
			toDelete: &models.Workspace{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
				Description: "looking for a defective ID",
			},
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.Workspaces.DeleteWorkspace(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

func TestGetWorkspacesForManagedIdentity(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, createdWarmupWorkspaces, err := createWarmupWorkspaces(ctx, testClient,
		standardWarmupGroupsForWorkspaces, standardWarmupWorkspaces)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}

	allGroupInfos, err := groupInfoFromGroups(ctx, testClient.client.getConnection(ctx), createdWarmupGroups)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}
	allGroupIDs := groupIDsFromGroupInfos(allGroupInfos)

	// Some managed identities and their connections to workspaces.
	managedIdentityIDs := []string{}
	managedIdentityID2WorkspacePaths := make(map[string][]string)
	for ix := 0; ix < len(allGroupIDs); ix++ {

		// Create one managed identity per group.
		newManagedIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
			Type: models.ManagedIdentityAWSFederated,
			//			ResourcePath: x,
			Name:        fmt.Sprintf("managed-identity-%d", ix),
			Description: fmt.Sprintf("This is managed identity %d.", ix),
			GroupID:     allGroupIDs[ix],
			Data:        []byte(fmt.Sprintf("managed identity %d data", ix)),
			CreatedBy:   fmt.Sprintf("someone %d", ix),
		})
		assert.Nil(t, err)
		if err != nil {
			// No point in continuing if users weren't created as designed.
			return
		}
		thisManagedIdentityID := newManagedIdentity.Metadata.ID
		managedIdentityIDs = append(managedIdentityIDs, thisManagedIdentityID)
		managedIdentityID2WorkspacePaths[thisManagedIdentityID] = []string{}

		// Except for the first one, connect the managed identity to the workspaces in the group.
		if ix > 0 {
			for ix2 := 0; ix2 < len(createdWarmupWorkspaces); ix2++ {
				if createdWarmupWorkspaces[ix2].GroupID == allGroupIDs[ix] {

					err := testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
						thisManagedIdentityID, createdWarmupWorkspaces[ix2].Metadata.ID)
					if err != nil {
						// No point in continuing.
						return
					}

					managedIdentityID2WorkspacePaths[thisManagedIdentityID] = append(managedIdentityID2WorkspacePaths[thisManagedIdentityID],
						createdWarmupWorkspaces[ix2].FullPath)

				}
			}
		}
	}

	// Sort the slices of workspace paths
	for k, v := range managedIdentityID2WorkspacePaths {
		sort.Strings(v)
		managedIdentityID2WorkspacePaths[k] = v
	}

	type testCase struct {
		name                 string
		managedIdentityID    string
		expectMsg            *string
		expectWorkspacePaths []string
	}

	// Positive cases.
	testCases := []testCase{}
	for ix := 1; ix < len(allGroupIDs); ix++ {
		managedIdentityID := managedIdentityIDs[ix]
		testCases = append(testCases, testCase{
			name:                 "positive-for-" + allGroupInfos[ix].fullPath,
			managedIdentityID:    managedIdentityID,
			expectWorkspacePaths: managedIdentityID2WorkspacePaths[managedIdentityID],
		})
	}

	// Negative cases:
	testCases = append(testCases,
		testCase{
			name:                 "negative, exists but no workspaces",
			managedIdentityID:    managedIdentityIDs[0],
			expectWorkspacePaths: []string{},
		},
		testCase{
			name:                 "negative, non-existent",
			managedIdentityID:    nonExistentID,
			expectWorkspacePaths: []string{},
		},
		testCase{
			name:                 "negative, invalid",
			managedIdentityID:    invalidID,
			expectMsg:            invalidUUIDMsg1,
			expectWorkspacePaths: []string{},
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			workspacesResult, err := testClient.client.Workspaces.GetWorkspacesForManagedIdentity(ctx, test.managedIdentityID)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {
				actualPaths := []string{}
				for _, ws := range workspacesResult {
					actualPaths = append(actualPaths, ws.FullPath)
				}

				// Order is not significant for this call, so sort the paths here to avoid false negatives.
				sort.Strings(test.expectWorkspacePaths)
				sort.Strings(actualPaths)
				assert.Equal(t, test.expectWorkspacePaths, actualPaths)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup groups for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForWorkspaces = []models.Group{
	{
		Description: "top level group 0 for testing workspace functions",
		FullPath:    "top-level-group-0-for-workspaces",
		CreatedBy:   "someone-1",
	},
	{
		Description: "top level group 1 for testing workspace functions",
		FullPath:    "top-level-group-1-for-workspaces",
		CreatedBy:   "someone-2",
	},
	{
		Description: "top level group 2 for testing workspace functions",
		FullPath:    "top-level-group-2-for-workspaces",
		CreatedBy:   "someone-3",
	},
	{
		Description: "top level group 3 for nothing",
		FullPath:    "top-level-group-3-for-nothing",
		CreatedBy:   "someone-4",
	},
}

// Standard warmup workspaces for tests in this module:
// Make sure the order in this slice is _NOT_ exactly full path or name order.
// The create function will derive the group ID and name from the full path.
var standardWarmupWorkspaces = []models.Workspace{
	{
		Description:        "workspace 1 for testing workspace functions",
		FullPath:           "top-level-group-0-for-workspaces/workspace-1",
		CreatedBy:          "someone-1",
		PreventDestroyPlan: false,
	},
	{
		Description:        "workspace 5 for testing workspace functions",
		FullPath:           "top-level-group-1-for-workspaces/workspace-5",
		CreatedBy:          "someone-6",
		PreventDestroyPlan: true,
	},
	{
		Description:        "workspace 3 for testing workspace functions",
		FullPath:           "top-level-group-2-for-workspaces/workspace-3",
		CreatedBy:          "someone-5",
		PreventDestroyPlan: false,
	},
	{
		Description:        "workspace 4 for testing workspace functions",
		FullPath:           "top-level-group-0-for-workspaces/workspace-4",
		CreatedBy:          "someone-3",
		PreventDestroyPlan: true,
	},
	{
		Description:        "workspace 2 for testing workspace functions",
		FullPath:           "top-level-group-1-for-workspaces/workspace-2",
		CreatedBy:          "someone-2",
		PreventDestroyPlan: false,
	},
}

// createWarmupWorkspaces creates some warmup groups and workspaces for a test
// The warmup groups and workspaces to create can be standard or otherwise.
//
// NOTE: Due to the need to supply the parent ID for non-top-level groups,
// the groups must be created in a top-down manner.
func createWarmupWorkspaces(ctx context.Context, testClient *testClient,
	newGroups []models.Group, newWorkspaces []models.Workspace) ([]models.Group, []models.Workspace, error) {

	resultGroups, parentPath2ID, err := createInitialGroups(ctx, testClient, newGroups)
	if err != nil {
		return nil, nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, parentPath2ID, newWorkspaces)
	if err != nil {
		return nil, nil, err
	}

	return resultGroups, resultWorkspaces, nil
}

func ptrWorkspaceSortableField(arg WorkspaceSortableField) *WorkspaceSortableField {
	return &arg
}

func (wis workspaceInfoPathSlice) Len() int {
	return len(wis)
}

func (wis workspaceInfoPathSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis workspaceInfoPathSlice) Less(i, j int) bool {
	return wis[i].fullPath < wis[j].fullPath
}

func (wis workspaceInfoTimeSlice) Len() int {
	return len(wis)
}

func (wis workspaceInfoTimeSlice) Swap(i, j int) {
	wis[i], wis[j] = wis[j], wis[i]
}

func (wis workspaceInfoTimeSlice) Less(i, j int) bool {
	return wis[i].updateTime.Before(wis[j].updateTime)
}

// workspaceInfoFromWorkspaces returns a slice of workspaceInfo, not necessarily sorted in any order.
func workspaceInfoFromWorkspaces(workspaces []models.Workspace) []workspaceInfo {
	result := []workspaceInfo{}

	for _, workspace := range workspaces {
		result = append(result, workspaceInfo{
			fullPath:    workspace.FullPath,
			workspaceID: workspace.Metadata.ID,
			updateTime:  *workspace.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

// pathsFromWorkspaceInfo preserves order
func pathsFromWorkspaceInfo(workspaceInfos []workspaceInfo) []string {
	result := []string{}
	for _, workspaceInfo := range workspaceInfos {
		result = append(result, workspaceInfo.fullPath)
	}
	return result
}

// workspaceIDsFromWorkspaceInfos preserves order
func workspaceIDsFromWorkspaceInfos(workspaceInfos []workspaceInfo) []string {
	result := []string{}
	for _, workspaceInfo := range workspaceInfos {
		result = append(result, workspaceInfo.workspaceID)
	}
	return result
}

// createJobStateVersion creates the records necessary to avoid violating
// foreign key constraints when updating and creating workspaces.
func createJobStateVersion(ctx context.Context, client *Client, workspaceID string) (string, string, error) {

	newPlan, err := client.Plans.CreatePlan(ctx, &models.Plan{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return "", "", err
	}

	newRun, err := client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspaceID,
		PlanID:      newPlan.Metadata.ID,
	})
	if err != nil {
		return "", "", err
	}

	newJob, err := client.Jobs.CreateJob(ctx, &models.Job{
		RunID:       newRun.Metadata.ID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return "", "", err
	}

	newStateVersion, err := client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspaceID,
		RunID:       ptr.String(newRun.Metadata.ID),
	})
	if err != nil {
		return "", "", err
	}

	return newJob.Metadata.ID, newStateVersion.Metadata.ID, nil
}

// Compare two workspace objects, including bounds for creation and updated times.
func compareWorkspaces(t *testing.T, expected, actual *models.Workspace, checkID bool, times timeBounds) {
	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
	compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)

	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.FullPath, actual.FullPath)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.CurrentJobID, actual.CurrentJobID)
	assert.Equal(t, expected.CurrentStateVersionID, actual.CurrentStateVersionID)
	assert.Equal(t, expected.DirtyState, actual.DirtyState)
	assert.Equal(t, expected.MaxJobDuration, actual.MaxJobDuration)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.PreventDestroyPlan, actual.PreventDestroyPlan)
}

// The End.
