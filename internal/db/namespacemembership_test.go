//go:build integration

package db

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

type holderIDs2Name struct {
	userIDs2Name           map[string]string
	serviceAccountIDs2Name map[string]string
	teamIDs2Name           map[string]string
}

type namespaceMembershipWarmupsInput struct {
	teams                []models.Team
	users                []models.User
	teamMembers          []models.TeamMember
	groups               []models.Group
	serviceAccounts      []models.ServiceAccount
	workspaces           []models.Workspace
	namespaceMemberships []CreateNamespaceMembershipInput
}

type namespaceMembershipWarmupsOutput struct {
	holderIDs2Name       holderIDs2Name
	teams                []models.Team
	teamMembers          []models.TeamMember
	groups               []models.Group
	serviceAccounts      []models.ServiceAccount
	workspaces           []models.Workspace
	namespaceMemberships []models.NamespaceMembership
	users                []models.User
}

// namespaceMembershipInfo aids convenience in accessing the information TestGetNamespaceMemberships
// needs about the warmup namespace memberships.
type namespaceMembershipInfo struct {
	updateTime            time.Time
	holder                string // user, service account, or team
	role                  string
	namespacePath         string
	namespaceMembershipID string
}

// namespaceMembershipInfoPathSlice makes a slice of workspaceInfo sortable by namespace path
type namespaceMembershipInfoPathSlice []namespaceMembershipInfo

// workspaceInfoTimeSlice makes a slice of workspaceInfo sortable by last updated time
type namespaceMembershipInfoTimeSlice []namespaceMembershipInfo

func TestGetNamespaceMemberships(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaceMemberships(ctx, testClient, namespaceMembershipWarmupsInput{
		teams:                standardWarmupTeamsForNamespaceMemberships,
		users:                standardWarmupUsersForNamespaceMemberships,
		teamMembers:          standardWarmupTeamMembersForNamespaceMemberships,
		groups:               standardWarmupGroupsForNamespaceMemberships,
		serviceAccounts:      standardWarmupServiceAccountsForNamespaceMemberships,
		workspaces:           standardWarmupWorkspacesForNamespaceMemberships,
		namespaceMemberships: standardWarmupNamespaceMemberships,
	})

	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces weren't all created.
		return
	}
	allNamespaceMembershipInfos := namespaceMembershipInfoFromNamespaceMemberships(
		createdWarmupOutput.holderIDs2Name, createdWarmupOutput.namespaceMemberships)

	// Sort by namespace paths and more.
	// A trail is more than a path.  A trail contains the path, the holder, and the role.
	sort.Sort(namespaceMembershipInfoPathSlice(allNamespaceMembershipInfos))
	allTrails := trailsFromNamespaceMembershipInfo(allNamespaceMembershipInfos, false)
	reverseTrails := reverseStringSlice(allTrails)

	// Sort by last update times.
	sort.Sort(namespaceMembershipInfoTimeSlice(allNamespaceMembershipInfos))
	allTrailsByTime := trailsFromNamespaceMembershipInfo(allNamespaceMembershipInfos, false)
	reverseTrailsByTime := reverseStringSlice(allTrailsByTime)

	/*
		These are the trails in allTrails:

		group-a--sa-1--deployer
		group-a--team-a--viewer
		group-a--user-2--owner
		group-a/group-b--sa-2--owner
		group-a/group-b--team-b--deployer
		group-a/group-b--user-0--viewer
		group-a/group-b/group-c--sa-0--viewer
		group-a/group-b/group-c--team-c--owner
		group-a/group-b/group-c--user-1--deployer
		group-a/group-b/group-c/workspace-c3--sa-2--viewer
		group-a/group-b/group-c/workspace-c3--team-b--deployer
		group-a/group-b/group-c/workspace-c3--user-0--owner
		group-a/group-b/workspace-b2--sa-1--deployer
		group-a/group-b/workspace-b2--team-a--owner
		group-a/group-b/workspace-b2--user-2--viewer
		group-a/workspace-a1--sa-0--owner
		group-a/workspace-a1--team-c--viewer
		group-a/workspace-a1--user-1--deployer

	*/

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		input                       *GetNamespaceMembershipsInput
		expectMsg                   *string
		name                        string
		expectPageInfo              PageInfo
		expectTrails                []string
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
			input: &GetNamespaceMembershipsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			sortedDescending             bool
			getBeforeCursorFromPrevious: false,
			getAfterCursorFromPrevious:  false,
			expectMsg:                   nil,
			expectTrails:                []string{},
			expectPageInfo: PageInfo{
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

		// nil input causes a nil pointer dereference in GetNamespaceMemberships, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetNamespaceMembershipsInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectTrails:         allTrails,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrails)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectTrails:         allTrails,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrails)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of namespace path",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
			},
			expectTrails:         allTrails,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrails)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of namespace path",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathDesc),
			},
			sortedDescending:     true,
			expectTrails:         reverseTrails,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrails)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order of time of last update",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldUpdatedAtAsc),
			},
			expectTrails:         allTrailsByTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrailsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order of time of last update",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldUpdatedAtDesc),
			},
			sortedDescending:     true,
			expectTrails:         reverseTrailsByTime,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrailsByTime)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectTrails:         allTrails,
			expectPageInfo:       PageInfo{TotalCount: int32(len(allTrails)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// Because of the interesting way we have to sort namespace memberships
		// within a given namespace in order to compare results, pagination must
		// keep each namespace path contained within the same page.  For these
		// cases, there are 3 namespace memberships per namespace path.
		{
			name: "pagination: first six",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(6),
				},
			},
			expectTrails: allTrails[:6],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTrails)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle six",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(6),
				},
			},
			getAfterCursorFromPrevious: true,
			expectTrails:               allTrails[6:12],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTrails)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final three",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectTrails:               allTrails[12:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTrails)),
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
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			sortedDescending: true,
			expectTrails:     reverseTrails[:3],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTrails)),
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
			input: &GetNamespaceMembershipsInput{
				Sort:              ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectTrails:                []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:    ptr.String("only first or last can be defined, not both"),
			expectTrails: allTrails[4:],
			expectPageInfo: PageInfo{
				TotalCount:      int32(len(allTrails)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "fully-populated types, nothing allowed through filters",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &NamespaceMembershipFilter{
					UserID:              ptr.String(""),
					ServiceAccountID:    ptr.String(""),
					TeamID:              ptr.String(""),
					GroupID:             ptr.String(""),
					WorkspaceID:         ptr.String(""),
					NamespacePathPrefix: ptr.String(""),
					NamespacePaths:      []string{},
				},
			},
			expectMsg:      emptyUUIDMsg2,
			expectTrails:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, user member ID, positive 0",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					UserID: ptr.String(createdWarmupOutput.users[0].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.users[0].Username),
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user member ID, positive 1",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					UserID: ptr.String(createdWarmupOutput.users[1].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.users[1].Username),
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, user member ID, exists, not a member of any namespace",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					UserID: ptr.String(findUserIDFromName(createdWarmupOutput.users, "user-99")),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, user member ID, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					UserID: ptr.String(nonExistentID),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, user member ID, invalid",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					UserID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectTrails:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, service account member ID, positive 0",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					ServiceAccountID: ptr.String(createdWarmupOutput.serviceAccounts[0].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.serviceAccounts[0].Name),
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account member ID, positive 1",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					ServiceAccountID: ptr.String(createdWarmupOutput.serviceAccounts[1].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.serviceAccounts[1].Name),
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, service account member ID, exists, not a member of any namespace",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					ServiceAccountID: ptr.String(findServiceAccountIDFromName(createdWarmupOutput.serviceAccounts, "sa-99")),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, service account member ID, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					ServiceAccountID: ptr.String(nonExistentID),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, service account member ID, invalid",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					ServiceAccountID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectTrails:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, team ID, positive 0",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID: ptr.String(createdWarmupOutput.teams[0].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.teams[0].Name),
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, team ID, positive 1",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID: ptr.String(createdWarmupOutput.teams[1].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.teams[1].Name),
			expectPageInfo:       PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, team ID, exists, not a member of any namespace",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID: ptr.String(findTeamIDFromName(createdWarmupOutput.teams, "team-99")),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, team ID, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID: ptr.String(nonExistentID),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, team ID, invalid",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectTrails:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, group ID, positive a",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID: ptr.String(createdWarmupOutput.groups[0].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.groups[0].Name+"--"),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, positive b",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID: ptr.String(createdWarmupOutput.groups[1].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.groups[1].Name+"--"),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, positive c",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID: ptr.String(createdWarmupOutput.groups[2].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.groups[2].Name+"--"),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, exists but no namespace memberships",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID: ptr.String(findGroupIDFromName(createdWarmupOutput.groups, "group-99")),
				},
			},
			expectTrails:         []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, group ID, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID: ptr.String(nonExistentID),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, group ID, invalid",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectTrails:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, workspace ID, positive 0",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					WorkspaceID: ptr.String(createdWarmupOutput.workspaces[0].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.workspaces[0].Name),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, positive 1",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					WorkspaceID: ptr.String(createdWarmupOutput.workspaces[1].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.workspaces[1].Name),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, positive 2",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					WorkspaceID: ptr.String(createdWarmupOutput.workspaces[2].Metadata.ID),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, createdWarmupOutput.workspaces[2].Name),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, workspace ID, exists, not a member of any namespace",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					WorkspaceID: ptr.String(findWorkspaceIDFromName(createdWarmupOutput.workspaces, "workspace-99")),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, workspace ID, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					WorkspaceID: ptr.String(nonExistentID),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, workspace ID, invalid",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					WorkspaceID: ptr.String(invalidID),
				},
			},
			expectMsg:      invalidUUIDMsg2,
			expectTrails:   []string{},
			expectPageInfo: PageInfo{},
		},

		{
			name: "filter, namespace path prefix, positive 0",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a"),
				},
			},
			expectTrails:         allTrails,
			expectPageInfo:       PageInfo{TotalCount: 18, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, positive 1",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a/group-b"),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, "group-a/group-b"),
			expectPageInfo:       PageInfo{TotalCount: 12, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, positive 2",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a/workspace-a1"),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, "group-a/workspace-a1"),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, positive 3",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a/group-b/group-c"),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, "group-a/group-b/group-c"),
			expectPageInfo:       PageInfo{TotalCount: 6, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, positive 4",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a/group-b/workspace-b2"),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, "group-a/group-b/workspace-b2"),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, positive 5",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a/group-b/group-c/workspace-c3"),
				},
			},
			expectTrails:         findMatchingTrails(allTrails, "group-a/group-b/group-c/workspace-c3"),
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, negative, end of path does not exist",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-a/group-b/group-c/bogus"),
				},
			},
			expectTrails:         []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace path prefix, exists, not a member of any namespace",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String("group-99"),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		{
			name: "filter, namespace path prefix, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePathPrefix: ptr.String(nonExistentID),
				},
			},
			expectTrails:   []string{},
			expectPageInfo: PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
		},

		// The namespace path prefix is not required to be a UUID, so no check for UUID format can be done.

		{
			name: "filter, empty slice of namespace paths",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{},
				},
			},
			expectTrails:         allTrails,
			expectPageInfo:       PageInfo{TotalCount: 18, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, a top-level group",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a"},
				},
			},
			expectTrails:         allTrails[:3],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, a workspace in a top-level group",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a/workspace-a1"},
				},
			},
			expectTrails:         allTrails[15:],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, a 2nd-level group",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a/group-b"},
				},
			},
			expectTrails:         allTrails[3:6],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, a workspace in a 2nd-level group",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a/group-b/workspace-b2"},
				},
			},
			expectTrails:         allTrails[12:15],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, a 3rd-level group",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a/group-b/group-c"},
				},
			},
			expectTrails:         allTrails[6:9],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, a workspace in a 3rd-level group",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a/group-b/group-c/workspace-c3"},
				},
			},
			expectTrails:         allTrails[9:12],
			expectPageInfo:       PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// Don't try to use append() to construct expectTrails.  Somehow, it corrupts allTrails.
		// append(allTrails[:3], append(allTrails[6:9], allTrails[12:15]...)...)
		{
			name: "filter, namespace paths, multiple of the above",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a", "group-a/group-b/workspace-b2", "group-a/group-b/group-c"},
				},
			},
			expectTrails: []string{
				allTrails[0],
				allTrails[1],
				allTrails[2],
				allTrails[6],
				allTrails[7],
				allTrails[8],
				allTrails[12],
				allTrails[13],
				allTrails[14],
			},
			expectPageInfo:       PageInfo{TotalCount: 9, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, namespace paths, non-existent",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					NamespacePaths: []string{"group-a/bogus"},
				},
			},
			expectTrails:         []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// A namespace path is not required to be a UUID, so no check for UUID format can be done.

		// Combining filter functions does a logical AND when deciding whether to include a result.
		// Because there are so many filter fields, do a few combinations but not all possible.

		{
			name: "filter, combination UserID and GroupID",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					UserID:  ptr.String(createdWarmupOutput.users[2].Metadata.ID),
					GroupID: ptr.String(createdWarmupOutput.groups[0].Metadata.ID),
				},
			},
			expectTrails:         []string{allTrails[2]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination ServiceAccountID and WorkspaceID",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					ServiceAccountID: ptr.String(createdWarmupOutput.serviceAccounts[1].Metadata.ID),
					WorkspaceID:      ptr.String(createdWarmupOutput.workspaces[1].Metadata.ID),
				},
			},
			expectTrails:         []string{allTrails[12]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination TeamID and GroupID",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID:  ptr.String(createdWarmupOutput.teams[1].Metadata.ID),
					GroupID: ptr.String(createdWarmupOutput.groups[1].Metadata.ID),
				},
			},
			expectTrails:         []string{allTrails[4]},
			expectPageInfo:       PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination GroupID and WorkspaceID, contradictory",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					GroupID:     ptr.String(createdWarmupOutput.groups[0].Metadata.ID),
					WorkspaceID: ptr.String(createdWarmupOutput.workspaces[0].Metadata.ID),
				},
			},
			expectTrails:         []string{},
			expectPageInfo:       PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, combination TeamID and NamespacePathPrefix",
			input: &GetNamespaceMembershipsInput{
				Sort: ptrNamespaceMembershipSortableField(NamespaceMembershipSortableFieldNamespacePathAsc),
				Filter: &NamespaceMembershipFilter{
					TeamID:              ptr.String(createdWarmupOutput.teams[2].Metadata.ID),
					NamespacePathPrefix: ptr.String("group-a/group-b/group-c"),
				},
			},
			expectTrails:         []string{allTrails[7]},
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

			// GetNamespaceMemberships(ctx context.Context, input *GetNamespaceMembershipsInput) (*NamespaceMembershipResult, error)
			namespaceMembershipsResult, err := testClient.client.NamespaceMemberships.GetNamespaceMemberships(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, namespaceMembershipsResult.PageInfo)
				assert.NotNil(t, namespaceMembershipsResult.NamespaceMemberships)
				pageInfo := namespaceMembershipsResult.PageInfo
				namespaceMemberships := namespaceMembershipsResult.NamespaceMemberships

				// Check the namespace memberships result by comparing a list of the trails.
				infos := namespaceMembershipInfoFromNamespaceMemberships(createdWarmupOutput.holderIDs2Name,
					namespaceMemberships)
				resultTrails := trailsFromNamespaceMembershipInfo(infos, test.sortedDescending)

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(resultTrails)
				}

				assert.Equal(t, len(test.expectTrails), len(resultTrails))
				assert.Equal(t, test.expectTrails, resultTrails)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one namespace membership returned.
				// If there are no namespace memberships returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(namespaceMemberships) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&namespaceMemberships[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&namespaceMemberships[len(namespaceMemberships)-1])
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

func TestGetNamespaceMembershipByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaceMemberships(ctx, testClient, namespaceMembershipWarmupsInput{
		teams:                standardWarmupTeamsForNamespaceMemberships,
		users:                standardWarmupUsersForNamespaceMemberships,
		teamMembers:          standardWarmupTeamMembersForNamespaceMemberships,
		groups:               standardWarmupGroupsForNamespaceMemberships,
		serviceAccounts:      standardWarmupServiceAccountsForNamespaceMemberships,
		workspaces:           standardWarmupWorkspacesForNamespaceMemberships,
		namespaceMemberships: standardWarmupNamespaceMemberships,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces and so forth weren't all created.
		return
	}

	type testCase struct {
		expectMsg                 *string
		name                      string
		searchID                  string
		expectNamespaceMembership bool
	}

	testCases := []testCase{}
	for _, positiveNamespaceMembership := range createdWarmupOutput.namespaceMemberships {
		testCases = append(testCases, testCase{
			name:                      positiveNamespaceMembership.Metadata.ID,
			searchID:                  positiveNamespaceMembership.Metadata.ID,
			expectNamespaceMembership: true,
		})
	}

	testCases = append(testCases,
		testCase{
			name:     "negative, does not exist",
			searchID: nonExistentID,
		},
		testCase{
			name:      "negative, invalid",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// GetNamespaceMembershipByID(ctx context.Context, id string) (*models.NamespaceMembership, error)
			namespaceMembership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectNamespaceMembership {
				// the positive case
				require.NotNil(t, namespaceMembership)
				assert.Equal(t, test.name, namespaceMembership.Metadata.ID)
			} else {
				// the negative and defective cases
				assert.Nil(t, namespaceMembership)
			}
		})
	}
}

func TestCreateNamespaceMembership(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaceMemberships(ctx, testClient, namespaceMembershipWarmupsInput{
		teams:                standardWarmupTeamsForNamespaceMemberships,
		users:                standardWarmupUsersForNamespaceMemberships,
		teamMembers:          standardWarmupTeamMembersForNamespaceMemberships,
		groups:               standardWarmupGroupsForNamespaceMemberships,
		serviceAccounts:      standardWarmupServiceAccountsForNamespaceMemberships,
		workspaces:           standardWarmupWorkspacesForNamespaceMemberships,
		namespaceMemberships: standardWarmupNamespaceMemberships,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces and so forth weren't all created.
		return
	}

	type testCase struct {
		input         *CreateNamespaceMembershipInput
		expectMsg     *string
		expectCreated *models.NamespaceMembership
		name          string
	}

	// For the positive cases, must make GroupID and WorkspaceID point to empty string rather than nil.

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, user",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-99",
				UserID:        &createdWarmupOutput.users[0].Metadata.ID,
				Role:          models.ViewerRole,
			},
			expectCreated: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Role: models.ViewerRole,
				Namespace: models.MembershipNamespace{
					Path:        "group-99",
					GroupID:     &createdWarmupOutput.groups[3].Metadata.ID,
					WorkspaceID: ptr.String(""),
				},
				UserID: &createdWarmupOutput.users[0].Metadata.ID,
			},
		},

		{
			name: "positive, service account",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    "group-a/workspace-99",
				ServiceAccountID: &createdWarmupOutput.serviceAccounts[0].Metadata.ID,
				Role:             models.DeployerRole,
			},
			expectCreated: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Role: models.DeployerRole,
				Namespace: models.MembershipNamespace{
					Path:        "group-a/workspace-99",
					GroupID:     ptr.String(""),
					WorkspaceID: &createdWarmupOutput.workspaces[3].Metadata.ID,
				},
				ServiceAccountID: &createdWarmupOutput.serviceAccounts[0].Metadata.ID,
			},
		},

		{
			name: "positive, team",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-99",
				TeamID:        &createdWarmupOutput.teamMembers[0].TeamID,
				Role:          models.OwnerRole,
			},
			expectCreated: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Role: models.OwnerRole,
				Namespace: models.MembershipNamespace{
					Path:        "group-99",
					GroupID:     &createdWarmupOutput.groups[3].Metadata.ID,
					WorkspaceID: ptr.String(""),
				},
				TeamID: &createdWarmupOutput.teamMembers[0].TeamID,
			},
		},

		// For the negative duplicate cases, repeat the same entries from the positive cases.

		{
			name: "negative, duplicate, user",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-99",
				UserID:        &createdWarmupOutput.users[0].Metadata.ID,
				Role:          models.ViewerRole,
			},
			expectMsg: ptr.String("member already exists"),
		},

		{
			name: "negative, duplicate, service account",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    "group-a/workspace-99",
				ServiceAccountID: &createdWarmupOutput.serviceAccounts[0].Metadata.ID,
				Role:             models.DeployerRole,
			},
			expectMsg: ptr.String("member already exists"),
		},

		{
			name: "negative, duplicate, team",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-99",
				TeamID:        &createdWarmupOutput.teamMembers[0].TeamID,
				Role:          models.OwnerRole,
			},
			expectMsg: ptr.String("member already exists"),
		},

		{
			name: "negative, non-existent namespace path",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-bogus",
				UserID:        &createdWarmupOutput.users[1].Metadata.ID,
				Role:          models.ViewerRole,
			},
			expectMsg: ptr.String("Namespace not found"),
		},

		{
			name: "negative, non-existent user",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-a",
				UserID:        ptr.String(nonExistentID),
				Role:          models.ViewerRole,
			},
			expectMsg: ptr.String("user does not exist"),
		},

		{
			name: "negative, non-existent service account",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    "group-a",
				ServiceAccountID: ptr.String(nonExistentID),
				Role:             models.ViewerRole,
			},
			expectMsg: ptr.String("service account does not exist"),
		},

		{
			name: "negative, non-existent team",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-a",
				TeamID:        ptr.String(nonExistentID),
				Role:          models.ViewerRole,
			},
			expectMsg: ptr.String("team does not exist"),
		},

		{
			name: "negative, invalid user",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-a",
				UserID:        ptr.String(invalidID),
				Role:          models.ViewerRole,
			},
			expectMsg: invalidUUIDMsg1,
		},

		{
			name: "negative, invalid service account",
			input: &CreateNamespaceMembershipInput{
				NamespacePath:    "group-a",
				ServiceAccountID: ptr.String(invalidID),
				Role:             models.ViewerRole,
			},
			expectMsg: invalidUUIDMsg1,
		},

		{
			name: "negative, invalid team",
			input: &CreateNamespaceMembershipInput{
				NamespacePath: "group-a",
				TeamID:        ptr.String(invalidID),
				Role:          models.ViewerRole,
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// CreateNamespaceMembership(ctx context.Context, input *CreateNamespaceMembershipInput) (*models.NamespaceMembership, error)
			actualCreated, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareNamespaceMemberships(t, test.expectCreated, actualCreated, false, timeBounds{
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

func TestUpdateNamespaceMembership(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaceMemberships(ctx, testClient, namespaceMembershipWarmupsInput{
		teams:                standardWarmupTeamsForNamespaceMemberships,
		users:                standardWarmupUsersForNamespaceMemberships,
		teamMembers:          standardWarmupTeamMembersForNamespaceMemberships,
		groups:               standardWarmupGroupsForNamespaceMemberships,
		serviceAccounts:      standardWarmupServiceAccountsForNamespaceMemberships,
		workspaces:           standardWarmupWorkspacesForNamespaceMemberships,
		namespaceMemberships: standardWarmupNamespaceMemberships,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces and so forth weren't all created.
		return
	}
	allNamespaceMembershipInfos := namespaceMembershipInfoFromNamespaceMemberships(
		createdWarmupOutput.holderIDs2Name, createdWarmupOutput.namespaceMemberships)

	type testCase struct {
		searchFor     *models.NamespaceMembership
		expectMsg     *string
		expectUpdated *models.NamespaceMembership
		name          string
	}

	// UpdateNamespaceMembership looks for ID and metadata version,
	// updates metadata version, last updated timestamp, and role.
	// The role is the only thing we control.

	testCases := []testCase{}

	for ix, preUpdate := range createdWarmupOutput.namespaceMemberships {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive-" + buildTrail(allNamespaceMembershipInfos[ix]),
			searchFor: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID:      preUpdate.Metadata.ID,
					Version: preUpdate.Metadata.Version,
				},
				Role:      rotateRole(preUpdate.Role),
				Namespace: preUpdate.Namespace,
			},
			expectUpdated: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID:                   preUpdate.Metadata.ID,
					Version:              preUpdate.Metadata.Version + 1,
					CreationTimestamp:    preUpdate.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Role: rotateRole(preUpdate.Role),
				Namespace: models.MembershipNamespace{
					ID:          preUpdate.Namespace.ID,
					Path:        preUpdate.Namespace.Path,
					GroupID:     preUpdate.Namespace.GroupID,
					WorkspaceID: preUpdate.Namespace.WorkspaceID,
				},
				UserID:           preUpdate.UserID,
				ServiceAccountID: preUpdate.ServiceAccountID,
				TeamID:           preUpdate.TeamID,
			},
		})
	}

	testCases = append(testCases, testCase{
		name: "negative, non-exist",
		searchFor: &models.NamespaceMembership{
			Metadata: models.ResourceMetadata{
				ID:      nonExistentID,
				Version: 1,
			},
		},
		expectMsg: resourceVersionMismatch,
	},
		testCase{
			name: "negative, invalid",
			searchFor: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: 1,
				},
			},
			expectMsg: invalidUUIDMsg1,
		})

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// UpdateNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) (*models.NamespaceMembership, error)
			actualUpdated, err := testClient.client.NamespaceMemberships.UpdateNamespaceMembership(ctx, test.searchFor)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// the positive case
				require.NotNil(t, actualUpdated)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				compareNamespaceMemberships(t, test.expectUpdated, actualUpdated, false, timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualUpdated)
			}
		})
	}
}

func TestDeleteNamespaceMembership(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupOutput, err := createWarmupNamespaceMemberships(ctx, testClient, namespaceMembershipWarmupsInput{
		teams:                standardWarmupTeamsForNamespaceMemberships,
		users:                standardWarmupUsersForNamespaceMemberships,
		teamMembers:          standardWarmupTeamMembersForNamespaceMemberships,
		groups:               standardWarmupGroupsForNamespaceMemberships,
		serviceAccounts:      standardWarmupServiceAccountsForNamespaceMemberships,
		workspaces:           standardWarmupWorkspacesForNamespaceMemberships,
		namespaceMemberships: standardWarmupNamespaceMemberships,
	})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup groups and workspaces and so forth weren't all created.
		return
	}
	allNamespaceMembershipInfos := namespaceMembershipInfoFromNamespaceMemberships(
		createdWarmupOutput.holderIDs2Name, createdWarmupOutput.namespaceMemberships)

	type testCase struct {
		searchFor                 *models.NamespaceMembership
		expectMsg                 *string
		name                      string
		expectNamespaceMembership bool
	}

	testCases := []testCase{}
	for ix, toDelete := range createdWarmupOutput.namespaceMemberships {
		testCases = append(testCases, testCase{
			name: "positive-" + buildTrail(allNamespaceMembershipInfos[ix]),
			searchFor: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID:      toDelete.Metadata.ID,
					Version: toDelete.Metadata.Version,
				},
			},
		})
	}

	testCases = append(testCases,
		testCase{
			name: "negative, non-exist",
			searchFor: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		testCase{
			name: "negative, defective-id",
			searchFor: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// DeleteNamespaceMembership(ctx context.Context, namespaceMembership *models.NamespaceMembership) error
			err := testClient.client.NamespaceMemberships.DeleteNamespaceMembership(ctx, test.searchFor)

			checkError(t, test.expectMsg, err)

		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup teams for tests in this module:
var standardWarmupTeamsForNamespaceMemberships = []models.Team{
	{
		Name:        "team-a",
		Description: "team a for namespace membership tests",
	},
	{
		Name:        "team-b",
		Description: "team b for namespace membership tests",
	},
	{
		Name:        "team-c",
		Description: "team c for namespace membership tests",
	},
	{
		Name:        "team-99",
		Description: "team 99 for namespace membership tests",
	},
}

// Standard warmup users for tests in this module:
// Please note: all users are _NON_-admin.
var standardWarmupUsersForNamespaceMemberships = []models.User{
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
	{
		Username: "user-team-a",
		Email:    "user-3@example.com",
	},
	{
		Username: "user-team-b",
		Email:    "user-4@example.com",
	},
	{
		Username: "user-team-c",
		Email:    "user-5@example.com",
	},
	{
		Username: "user-99",
		Email:    "user-99@example.com",
	},
}

// Standard warmup team member relationships for tests in this module:
// Please note that the ID fields contain names, not IDs.
var standardWarmupTeamMembersForNamespaceMemberships = []models.TeamMember{
	{
		UserID: "user-team-a",
		TeamID: "team-a",
	},
	{
		UserID: "user-team-b",
		TeamID: "team-b",
	},
	{
		UserID: "user-team-c",
		TeamID: "team-c",
	},
}

// Standard warmup groups for tests in this module:
// These groups are in a linear chain of descent.
// The create function will derive the parent path and name from the namespace path.
var standardWarmupGroupsForNamespaceMemberships = []models.Group{
	{
		Description: "top level group for testing namespace membership functions",
		FullPath:    "group-a",
		CreatedBy:   "someone-1",
	},
	{
		Description: "second level group for testing namespace membership functions",
		FullPath:    "group-a/group-b",
		CreatedBy:   "someone-2",
	},
	{
		Description: "third level group for testing namespace membership functions",
		FullPath:    "group-a/group-b/group-c",
		CreatedBy:   "someone-3",
	},
	{
		Description: "orphaned top-level group for testing namespace membership functions",
		FullPath:    "group-99",
		CreatedBy:   "someone-99",
	},
}

// Standard warmup service accounts for tests in this module:
// Please note: the GroupID field here contains the group _FULL_PATH_, not the _ID_.
var standardWarmupServiceAccountsForNamespaceMemberships = []models.ServiceAccount{
	{
		Name:        "sa-0",
		Description: "service account 0 for namespace membership tests",
		GroupID:     "group-a",
		CreatedBy:   "someone-0",
	},
	{
		Name:        "sa-1",
		Description: "service account 1 for namespace membership tests",
		GroupID:     "group-a/group-b",
		CreatedBy:   "someone-1",
	},
	{
		Name:        "sa-2",
		Description: "service account 2 for namespace membership tests",
		GroupID:     "group-a/group-b/group-c",
		CreatedBy:   "someone-2",
	},
	{
		Name:        "sa-99",
		Description: "service account 99 for namespace membership tests",
		GroupID:     "group-99",
		CreatedBy:   "nobody",
	},
}

// Standard warmup workspaces for tests in this module:
// The create function will derive the group ID and name from the namespace path.
var standardWarmupWorkspacesForNamespaceMemberships = []models.Workspace{
	{
		Description: "workspace a1 for testing namespace membership functions",
		FullPath:    "group-a/workspace-a1",
		CreatedBy:   "someone-1",
	},
	{
		Description: "workspace b2 for testing namespace membership functions",
		FullPath:    "group-a/group-b/workspace-b2",
		CreatedBy:   "someone-2",
	},
	{
		Description: "workspace c3 for testing namespace membership functions",
		FullPath:    "group-a/group-b/group-c/workspace-c3",
		CreatedBy:   "someone-3",
	},
	{
		Description: "workspace 99 for testing namespace membership functions",
		FullPath:    "group-a/workspace-99",
		CreatedBy:   "someone-99",
	},
}

// Standard warmup namespace memberships for tests in this module:
// In this variable, the ID field is the user, service account, and team _NAME_, NOT the ID.
var standardWarmupNamespaceMemberships = []CreateNamespaceMembershipInput{

	// Teams are given group memberships straight across: team X to group X; abc.
	{
		NamespacePath: "group-a",
		TeamID:        ptr.String("team-a"),
		Role:          models.ViewerRole,
	},
	{
		NamespacePath: "group-a/group-b",
		TeamID:        ptr.String("team-b"),
		Role:          models.DeployerRole,
	},
	{
		NamespacePath: "group-a/group-b/group-c",
		TeamID:        ptr.String("team-c"),
		Role:          models.OwnerRole,
	},

	// Users are given group memberships rotated by one slot; bca.
	{
		NamespacePath: "group-a/group-b",
		UserID:        ptr.String("user-0"),
		Role:          models.ViewerRole,
	},
	{
		NamespacePath: "group-a/group-b/group-c",
		UserID:        ptr.String("user-1"),
		Role:          models.DeployerRole,
	},
	{
		NamespacePath: "group-a",
		UserID:        ptr.String("user-2"),
		Role:          models.OwnerRole,
	},

	// Service accounts are given group memberships rotated by two slots (or back by one); cab.
	{
		NamespacePath:    "group-a/group-b/group-c",
		ServiceAccountID: ptr.String("sa-0"),
		Role:             models.ViewerRole,
	},
	{
		NamespacePath:    "group-a",
		ServiceAccountID: ptr.String("sa-1"),
		Role:             models.DeployerRole,
	},
	{
		NamespacePath:    "group-a/group-b",
		ServiceAccountID: ptr.String("sa-2"),
		Role:             models.OwnerRole,
	},

	// Teams are given workspace memberships rotated by one slot; bca.
	{
		NamespacePath: "group-a/group-b/workspace-b2",
		TeamID:        ptr.String("team-a"),
		Role:          models.OwnerRole,
	},
	{
		NamespacePath: "group-a/group-b/group-c/workspace-c3",
		TeamID:        ptr.String("team-b"),
		Role:          models.DeployerRole,
	},
	{
		NamespacePath: "group-a/workspace-a1",
		TeamID:        ptr.String("team-c"),
		Role:          models.ViewerRole,
	},

	// Users are given workspace memberships rotated by two slots (or back by one); cab.
	{
		NamespacePath: "group-a/group-b/group-c/workspace-c3",
		UserID:        ptr.String("user-0"),
		Role:          models.OwnerRole,
	},
	{
		NamespacePath: "group-a/workspace-a1",
		UserID:        ptr.String("user-1"),
		Role:          models.DeployerRole,
	},
	{
		NamespacePath: "group-a/group-b/workspace-b2",
		UserID:        ptr.String("user-2"),
		Role:          models.ViewerRole,
	},

	// Service accounts are given workspace memberships straight across; abc.
	{
		NamespacePath:    "group-a/workspace-a1",
		ServiceAccountID: ptr.String("sa-0"),
		Role:             models.OwnerRole,
	},
	{
		NamespacePath:    "group-a/group-b/workspace-b2",
		ServiceAccountID: ptr.String("sa-1"),
		Role:             models.DeployerRole,
	},
	{
		NamespacePath:    "group-a/group-b/group-c/workspace-c3",
		ServiceAccountID: ptr.String("sa-2"),
		Role:             models.ViewerRole,
	},
}

// createWarmupNamespaceMemberships creates some objects for a test
// The objects to create can be standard or otherwise.
//
// NOTE: Due to the need to supply the parent ID for non-top-level groups,
// any groups must be created in a top-down manner.
func createWarmupNamespaceMemberships(ctx context.Context, testClient *testClient,
	input namespaceMembershipWarmupsInput) (*namespaceMembershipWarmupsOutput, error) {

	resultTeams, teamName2ID, err := createInitialTeams(ctx, testClient, input.teams)
	if err != nil {
		return nil, err
	}

	resultUsers, username2ID, err := createInitialUsers(ctx, testClient, input.users)
	if err != nil {
		return nil, err
	}

	resultTeamMembers, err := createInitialTeamMembers(ctx, testClient, teamName2ID, username2ID, input.teamMembers)
	if err != nil {
		return nil, err
	}

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultServiceAccounts, serviceAccountName2ID, err := createInitialServiceAccounts(ctx, testClient,
		groupPath2ID, input.serviceAccounts)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, groupPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	resultNamespaceMemberships, err := createInitialNamespaceMemberships(ctx, testClient,
		teamName2ID, username2ID, groupPath2ID, serviceAccountName2ID, input.namespaceMemberships)
	if err != nil {
		return nil, err
	}

	return &namespaceMembershipWarmupsOutput{
		teams:                resultTeams,
		users:                resultUsers,
		teamMembers:          resultTeamMembers,
		groups:               resultGroups,
		serviceAccounts:      resultServiceAccounts,
		workspaces:           resultWorkspaces,
		namespaceMemberships: resultNamespaceMemberships,
		holderIDs2Name: holderIDs2Name{
			userIDs2Name:           reverseMap(username2ID),
			serviceAccountIDs2Name: reverseMap(serviceAccountName2ID),
			teamIDs2Name:           reverseMap(teamName2ID),
		},
	}, nil
}

// reverseMap returns a map that does the inverse mapping of the input.
// Because names and IDs are unique within a given domain, collisions are not possible.
func reverseMap(input map[string]string) map[string]string {
	result := make(map[string]string)
	for name, id := range input {
		result[id] = name
	}
	return result
}

func ptrNamespaceMembershipSortableField(arg NamespaceMembershipSortableField) *NamespaceMembershipSortableField {
	return &arg
}

func (nss namespaceMembershipInfoPathSlice) Len() int {
	return len(nss)
}

func (nss namespaceMembershipInfoPathSlice) Swap(i, j int) {
	nss[i], nss[j] = nss[j], nss[i]
}

func (nss namespaceMembershipInfoPathSlice) Less(i, j int) bool {
	return nss[i].namespacePath < nss[j].namespacePath
}

func (nss namespaceMembershipInfoTimeSlice) Len() int {
	return len(nss)
}

func (nss namespaceMembershipInfoTimeSlice) Swap(i, j int) {
	nss[i], nss[j] = nss[j], nss[i]
}

func (nss namespaceMembershipInfoTimeSlice) Less(i, j int) bool {
	return nss[i].updateTime.Before(nss[j].updateTime)
}

// namespaceMembershipInfoFromNamespaceMemberships returns a slice of namespaceMembershipInfo,
// not necessarily sorted in any order.
func namespaceMembershipInfoFromNamespaceMemberships(holderIDs2Name holderIDs2Name,
	namespaceMemberships []models.NamespaceMembership) []namespaceMembershipInfo {
	result := []namespaceMembershipInfo{}

	for _, namespaceMembership := range namespaceMemberships {

		var holder string
		switch {
		case namespaceMembership.UserID != nil:
			holder = holderIDs2Name.userIDs2Name[*namespaceMembership.UserID]
		case namespaceMembership.ServiceAccountID != nil:
			holder = holderIDs2Name.serviceAccountIDs2Name[*namespaceMembership.ServiceAccountID]
		case namespaceMembership.TeamID != nil:
			holder = holderIDs2Name.teamIDs2Name[*namespaceMembership.TeamID]
		}

		result = append(result, namespaceMembershipInfo{
			namespacePath:         namespaceMembership.Namespace.Path,
			namespaceMembershipID: namespaceMembership.Metadata.ID,
			holder:                holder,
			role:                  string(namespaceMembership.Role),
			updateTime:            *namespaceMembership.Metadata.LastUpdatedTimestamp,
		})
	}

	return result
}

/*
trailsFromNamespaceMembershipInfo preserves order to a point but not beyond.

Results from GetNamespaceMemberships are sorted by namespace path but _NOT_ by holder or role.
In order to conveniently compare lists of namespace memberships, it is necessary to sort the
namespace membership trails within the same namespace path.

If sortedDescending is true, the trails within a given namespace path are sorted in descending order.
*/
func trailsFromNamespaceMembershipInfo(namespaceMembershipInfos []namespaceMembershipInfo,
	sortedDescending bool) []string {
	result := []string{}

	sameNamespace := []string{}
	thisNamespacePath := ""
	for _, namespaceMembershipInfo := range namespaceMembershipInfos {
		thisTrail := buildTrail(namespaceMembershipInfo)

		if (len(sameNamespace) > 0) && (namespaceMembershipInfo.namespacePath != thisNamespacePath) {
			// A change of namespace: sort and flush the old sameNamespace.
			sort.Strings(sameNamespace)
			if sortedDescending {
				sameNamespace = reverseStringSlice(sameNamespace)
			}
			result = append(result, sameNamespace...)
			sameNamespace = []string{}
		}

		// Record the new entry, alone or in same namespace as those before it.
		sameNamespace = append(sameNamespace, thisTrail)
		thisNamespacePath = namespaceMembershipInfo.namespacePath

	}

	// Flush the final contents of sameNamespace to result.
	sort.Strings(sameNamespace)
	if sortedDescending {
		sameNamespace = reverseStringSlice(sameNamespace)
	}
	result = append(result, sameNamespace...)

	return result
}

// buildTrail constructs the trail for a namespaceMembership
func buildTrail(input namespaceMembershipInfo) string {
	return (input.namespacePath + "--" + input.holder + "--" + input.role)
}

// namespaceMembershipIDsFromNamespaceMembershipInfos preserves order
func namespaceMembershipIDsFromNamespaceMembershipInfos(namespaceMembershipInfos []namespaceMembershipInfo) []string {
	result := []string{}
	for _, namespaceMembershipInfo := range namespaceMembershipInfos {
		result = append(result, namespaceMembershipInfo.namespaceMembershipID)
	}
	return result
}

// Compare two namespace membership objects, including bounds for creation and updated times.
func compareNamespaceMemberships(t *testing.T, expected, actual *models.NamespaceMembership,
	checkID bool, times timeBounds) {
	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
	compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)

	assert.Equal(t, expected.Role, actual.Role)

	assert.Equal(t, expected.Namespace.ID, actual.Namespace.ID)
	assert.Equal(t, expected.Namespace.Path, actual.Namespace.Path)

	assert.Equal(t, (expected.Namespace.GroupID == nil), (actual.Namespace.GroupID == nil))
	if (expected.Namespace.GroupID != nil) && (actual.Namespace.GroupID != nil) {
		assert.Equal(t, *expected.Namespace.GroupID, *actual.Namespace.GroupID)
	}

	assert.Equal(t, (expected.Namespace.WorkspaceID == nil), (actual.Namespace.WorkspaceID == nil))
	if (expected.Namespace.WorkspaceID != nil) && (actual.Namespace.WorkspaceID != nil) {
		assert.Equal(t, *expected.Namespace.WorkspaceID, *actual.Namespace.WorkspaceID)
	}

	assert.Equal(t, (expected.UserID == nil), (actual.UserID == nil))
	if (expected.UserID != nil) && (actual.UserID != nil) {
		assert.Equal(t, *expected.UserID, *actual.UserID)
	}

	assert.Equal(t, (expected.ServiceAccountID == nil), (actual.ServiceAccountID == nil))
	if (expected.ServiceAccountID != nil) && (actual.ServiceAccountID != nil) {
		assert.Equal(t, *expected.ServiceAccountID, *actual.ServiceAccountID)
	}

	assert.Equal(t, (expected.TeamID == nil), (actual.TeamID == nil))
	if (expected.TeamID != nil) && (actual.TeamID != nil) {
		assert.Equal(t, *expected.TeamID, *actual.TeamID)
	}
}

// findMatchingTrails returns a slice of strings that contain the specified substring
func findMatchingTrails(allTrails []string, s string) []string {
	result := []string{}
	for _, candidate := range allTrails {
		if strings.Contains(candidate, s) {
			result = append(result, candidate)
		}
	}

	return result
}

// findUserIDFromName returns the matching user ID for the specified name.
func findUserIDFromName(users []models.User, nm string) string {
	for _, user := range users {
		if user.Username == nm {
			return user.Metadata.ID
		}
	}
	return ""
}

// findServiceAccountIDFromName returns the matching service account ID for the specified name.
func findServiceAccountIDFromName(serviceAccounts []models.ServiceAccount, nm string) string {
	for _, serviceAccount := range serviceAccounts {
		if serviceAccount.Name == nm {
			return serviceAccount.Metadata.ID
		}
	}
	return ""

}

// findTeamIDFromName returns the matching team ID for the specified name.
func findTeamIDFromName(teams []models.Team, nm string) string {
	for _, team := range teams {
		if team.Name == nm {
			return team.Metadata.ID
		}
	}
	return ""
}

// findGroupIDFromName returns the matching group ID for the specified name.
func findGroupIDFromName(groups []models.Group, nm string) string {
	for _, group := range groups {
		if group.Name == nm {
			return group.Metadata.ID
		}
	}
	return ""
}

// findWorkspaceIDFromName returns the matching workspace ID for the specified name.
func findWorkspaceIDFromName(workspaces []models.Workspace, nm string) string {
	for _, workspace := range workspaces {
		if workspace.Name == nm {
			return workspace.Metadata.ID
		}
	}
	return ""
}

// rotateRole returns a different role from what was passed in.
// If other modules need this function, move it to dbclient_test.
func rotateRole(input models.Role) models.Role {
	switch {
	case input == models.ViewerRole:
		return models.DeployerRole
	case input == models.DeployerRole:
		return models.OwnerRole
	case input == models.OwnerRole:
		return models.ViewerRole
	}

	// Keep the compiler happy, even if it cannot happen.
	return models.ViewerRole
}

// The End.
