//go:build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
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

type migrateGroupWarmupsInput struct {
	groups             []models.Group
	groupPath2ID       map[string]string
	workspaces         []models.Workspace
	serviceAccounts    []models.ServiceAccount
	managedIdentities  []models.ManagedIdentity
	gpgKeys            []models.GPGKey
	terraformProviders []models.TerraformProvider
	terraformModules   []models.TerraformModule
	teams              []models.Team
	membershipInputs   []CreateNamespaceMembershipInput
	variables          []models.Variable
	users              []models.User
	activityEvents     []models.ActivityEvent
	vcsProviders       []models.VCSProvider
	roles              []models.Role
	runners            []models.Runner
}

type migrateGroupWarmupsOutput struct {
	groupID2Path       map[string]string
	workspaceID2Path   map[string]string // for use only as an output from creating resources
	groups             []models.Group
	workspaces         []models.Workspace
	serviceAccounts    []models.ServiceAccount
	managedIdentities  []models.ManagedIdentity
	gpgKeys            []models.GPGKey
	terraformProviders []models.TerraformProvider
	terraformModules   []models.TerraformModule
	teams              []models.Team
	memberships        []models.NamespaceMembership
	variables          []models.Variable
	users              []models.User
	activityEvents     []models.ActivityEvent
	vcsProviders       []models.VCSProvider
	roles              []models.Role
	runners            []models.Runner
}

type migrateGroupWarmupsOthers struct {
	workspaces         []models.Workspace
	serviceAccounts    []models.ServiceAccount
	managedIdentities  []models.ManagedIdentity
	gpgKeys            []models.GPGKey
	terraformProviders []models.TerraformProvider
	teams              []models.Team
	memberships        []models.NamespaceMembership
	variables          []models.Variable
	users              []models.User
	activityEvents     []models.ActivityEvent
	vcsProviders       []models.VCSProvider
	runners            []models.Runner
}

type associateManagedIdentityAssignment struct {
	workspace           *models.Workspace
	managedIdentity     *models.ManagedIdentity
	filterBase          string
	workspacePath       string
	managedIdentityPath string
}

type associateServiceAccountNamespaceMembership struct {
	namespaceMembership *models.NamespaceMembership
	filterBase          string
	serviceAccountPath  string
	namespacePath       string
	roleName            string
}

type associateServiceAccountRunnerAssignment struct {
	runner             *models.Runner
	serviceAccount     *models.ServiceAccount
	filterBase         string
	serviceAccountPath string
	runnerPath         string
}

type associateWorkspaceVCSProviderLink struct {
	workspaceVCSProviderLink *models.WorkspaceVCSProviderLink
	filterBase               string
	workspacePath            string
	providerPath             string
}

// associations wraps the input to and results of creating an association.
// It is used for testing MigrateGroup association handling.
type associations struct {
	managedIdentityAssignments         []associateManagedIdentityAssignment
	serviceAccountNamespaceMemberships []associateServiceAccountNamespaceMembership
	serviceAccountRunnerAssignments    []associateServiceAccountRunnerAssignment
	workspaceVCSProviderLinks          []associateWorkspaceVCSProviderLink
}

// TestGetGroupByID tests GetGroupByID
func TestGetGroupByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a group with a specific ID without going into the really
	// low-level stuff, create the warmup group(s) by name and then find the relevant ID.
	createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	require.Nil(t, err)

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
			expectMsg:   ptr.String(ErrInvalidID.Error()),
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

func TestGetGroupByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	// Testing edge-cases with namespace path.
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(123),
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectApply     bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "get resource by TRN",
			trn:         group.Metadata.TRN,
			expectApply: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.GroupModelType.BuildTRN("unknown"),
		},
		{
			name: "passing in a workspace TRN should not return any results",
			trn:  types.GroupModelType.BuildTRN(workspace.FullPath),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualGroup, err := testClient.client.Groups.GetGroupByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectApply {
				require.NotNil(t, actualGroup)
				assert.Equal(t, types.GroupModelType.BuildTRN(group.FullPath), actualGroup.Metadata.TRN)
			} else {
				assert.Nil(t, actualGroup)
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
	require.Nil(t, err)

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
			require.Nil(t, err)

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

	createdWarmupGroups, groupMap, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	require.Nil(t, err)

	// Users for testing search.
	createdWarmupUsers, userMap, err := createInitialUsers(ctx, testClient, warmupUsersForSearch)
	require.Nil(t, err)

	// Service account(s) for testing search.
	createdWarmupServiceAccounts, serviceAccountMap, err := createInitialServiceAccounts(ctx, testClient,
		groupMap, warmupServiceAccountsForSearch)
	require.Nil(t, err)

	// Must create a role before creating namespace memberships.
	_, rolesMap, err := createInitialRoles(ctx, testClient, warmupRolesForSearch)
	require.Nil(t, err)

	// Namespace memberships for testing search.
	emptyMap := map[string]string{}
	_, err = createInitialNamespaceMemberships(ctx, testClient,
		emptyMap, userMap, groupMap, serviceAccountMap, rolesMap, warmupNamespaceMembershipsForSearch)
	require.Nil(t, err)

	allGroupInfos, err := groupInfoFromGroups(ctx, testClient.client.getConnection(ctx), createdWarmupGroups)
	require.Nil(t, err)

	allPaths := pathsFromGroupInfo(allGroupInfos)
	reversePaths := reverseStringSlice(allPaths)
	allGroupIDs := groupIDsFromGroupInfos(allGroupInfos)
	allNamespaceIDs := namespaceIDsFromGroupInfos(allGroupInfos)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectPageInfo              pagination.PageInfo
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
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "populated sort and pagination, nil filter",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in ascending order",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "sort in descending order",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathDesc),
			},
			expectGroupPaths:     reversePaths,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: everything at once",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectGroupPaths:     allPaths,
			expectPageInfo:       pagination.PageInfo{TotalCount: int32(len(allPaths)), Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: first two",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectGroupPaths: allPaths[:2],
			expectPageInfo: pagination.PageInfo{
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
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectGroupPaths:           allPaths[2:4],
			expectPageInfo: pagination.PageInfo{
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
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectGroupPaths:           allPaths[4:],
			expectPageInfo: pagination.PageInfo{
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
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectGroupPaths: reversePaths[:3],
			expectPageInfo: pagination.PageInfo{
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
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(4),
					},
				},
				expectGroupPaths: allPaths[:4],
				expectPageInfo: pagination.PageInfo{
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
					PaginationOptions: &pagination.Options{
						Last: ptr.Int32(2),
					},
				},
				getBeforeCursorFromPrevious: true,
				expectGroupPaths:            allPaths[1:3],
				expectPageInfo: pagination.PageInfo{
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
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectGroupPaths:            []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetGroupsInput{
				Sort: ptrGroupSortableField(GroupSortableFieldFullPathAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg:        ptr.String("only first or last can be defined, not both"),
			expectGroupPaths: allPaths[5:],
			expectPageInfo: pagination.PageInfo{
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
				PaginationOptions: &pagination.Options{
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
			expectPageInfo:   pagination.PageInfo{},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 6, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: nil},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 0, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
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
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, empty string, no other restrictions",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String(""),
				},
			},
			expectGroupPaths:     allPaths, // should find all 6 of them
			expectPageInfo:       pagination.PageInfo{TotalCount: 6, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, no other restrictions",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("group"),
				},
			},
			expectGroupPaths:     allPaths, // should find all 6 of them
			expectPageInfo:       pagination.PageInfo{TotalCount: 6, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, 1, no other restrictions",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("1"),
				},
			},
			expectGroupPaths:     allPaths[0:4],
			expectPageInfo:       pagination.PageInfo{TotalCount: 4, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, a, no other restrictions",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("a"),
				},
			},
			expectGroupPaths:     allPaths[1:2],
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, 2, no other restrictions",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("2"),
				},
			},
			expectGroupPaths:     allPaths[1:5],
			expectPageInfo:       pagination.PageInfo{TotalCount: 4, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, parent ID non-empty",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search:   ptr.String("group"),
					ParentID: &allGroupIDs[0], // top-level-group-1
				},
			},
			expectGroupPaths:     allPaths[1:3], // 1, 2
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, with UserMemberID", // verifies auth checks for non-root-only
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search:       ptr.String("group"),
					UserMemberID: &createdWarmupUsers[0].Metadata.ID, // top-level-group-1/2nd-level-group-1a
				},
			},
			expectGroupPaths:     allPaths[1:2], // top-level-group-1/2nd-level-group-1a
			expectPageInfo:       pagination.PageInfo{TotalCount: 1, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, with ServiceAccountMemberID", // verifies auth checks for non-root-only
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search:                 ptr.String("group"),
					ServiceAccountMemberID: &createdWarmupServiceAccounts[0].Metadata.ID, // top-level-group-1/2nd-level-group-1b...
				},
			},
			expectGroupPaths:     allPaths[2:4], // top-level-group-1/2nd-level-group-1b...
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, GroupIDs",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("group"),
					GroupIDs: []string{
						allGroupIDs[1], // top-level-group-1/2nd-level-group-1a
						allGroupIDs[4], // top-level-group-2
					},
				},
			},
			expectGroupPaths:     []string{allPaths[1], allPaths[4]}, // top-level-group-1/2nd-level-group-1a, top-level-group-2
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, NamespaceIDs",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("group"),
					NamespaceIDs: []string{
						allNamespaceIDs[5], // top-level-group-3
						allNamespaceIDs[3], // top-level-group-1/2nd-level-group-1b/3rd-level-group-1b1
					},
				},
			},
			expectGroupPaths: []string{
				allPaths[3], // top-level-group-1/2nd-level-group-1b/3rd-level-group-1b1
				allPaths[5], // top-level-group-3
			},
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, root only",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search:   ptr.String("group"),
					RootOnly: true,
				},
			},
			expectGroupPaths:     []string{allPaths[0], allPaths[4], allPaths[5]}, // top-level-group-{1,2,3}
			expectPageInfo:       pagination.PageInfo{TotalCount: 3, Cursor: dummyCursorFunc},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "search, plain search, group, NamespaceIDs and root only", // verifies auth checks for root-only
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					Search: ptr.String("group"),
					NamespaceIDs: []string{
						allNamespaceIDs[5], // top-level-group-3
						allNamespaceIDs[0], // top-level-group-1
					},
					RootOnly: true,
				},
			},
			expectGroupPaths:     []string{allPaths[0], allPaths[5]}, // top-level-group-{1,3}
			expectPageInfo:       pagination.PageInfo{TotalCount: 2, Cursor: dummyCursorFunc},
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
				require.Nil(t, err)

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
							parent, err := testClient.client.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(parentPath))
							require.Nil(t, err)

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
			require.Nil(t, err)
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
	require.Nil(t, err)

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
				originalGroup, err = testClient.client.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(*test.findFullPath))
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

				retrieved, err := testClient.client.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(originalGroup.FullPath))
				require.Nil(t, err)

				require.NotNil(t, retrieved)

				compareGroupsUpdate(t, *expectUpdatedDescription, *originalGroup, *retrieved)
			}
		})
	}
}

// TestMigrateGroupBasics tests MigrateGroup's basic function of setting the parent ID and updating namespace paths.
func TestMigrateGroupBasics(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	require.Nil(t, err)

	allResources, err := createMigrateResources(ctx, testClient, &migrateGroupWarmupsInput{
		groups:             deriveGroupNames(createdWarmupGroups),
		groupPath2ID:       groupPath2ID,
		workspaces:         warmupWorkspacesForGroupMigration,
		serviceAccounts:    warmupServiceAccountsForGroupMigration,
		managedIdentities:  warmupManagedIdentitiesForGroupMigration,
		gpgKeys:            warmupGPGKeysForGroupMigration,
		terraformProviders: warmupTerraformProvidersForGroupMigration,
		teams:              warmupTeamsForGroupMigration,
		membershipInputs:   warmupMembershipInputsForGroupMigration,
		variables:          warmupVariablesForGroupMigration,
		users:              warmupUsersForGroupMigration,
		activityEvents:     warmupActivityEventsForGroupMigration,
		vcsProviders:       warmupVCSProvidersForGroupMigration,
		roles:              warmupRolesForGroupMigration,
		runners:            warmupRunnersForGroupMigration,
	})
	require.Nil(t, err)

	type testCase struct {
		newParent  *models.Group
		expectMsg  *string
		name       string
		others     migrateGroupWarmupsOthers
		group      models.Group
		isPositive bool
	}

	/*
				template test case:

		{
			name          string
			group         models.Group
			newParent     *models.Group
			expectMsg     *string
			isPositive    bool
			others        migrateGroupWarmupsOthers
		}
	*/

	testCases := []testCase{}

	testCases = append(testCases,
		testCase{
			name:       "positive to root",
			group:      allResources.groups[4], // move top-level-group-1/2nd-level-group-1b
			newParent:  nil,                    // to root
			isPositive: true,
			others:     migrateGroupWarmupsOthers{}, // no other attached resources
		},
		testCase{
			name:       "positive to non-root",
			group:      allResources.groups[3],  // move top-level-group-1/2nd-level-group-1a
			newParent:  &allResources.groups[1], // under top-level-group-2
			isPositive: true,
			others: migrateGroupWarmupsOthers{
				workspaces:         allResources.workspaces,
				serviceAccounts:    allResources.serviceAccounts,
				managedIdentities:  allResources.managedIdentities,
				gpgKeys:            allResources.gpgKeys,
				terraformProviders: allResources.terraformProviders,
				memberships:        allResources.memberships,
				variables:          allResources.variables,
				activityEvents:     allResources.activityEvents,
				vcsProviders:       allResources.vcsProviders,
				runners:            allResources.runners,
			},
		},

		testCase{
			name: "negative, group to move does not exist",
			group: models.Group{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				FullPath: "the-group-that-does-not-exist",
			},
			newParent: &allResources.groups[1], // under top-level-group-2
			expectMsg: resourceVersionMismatch,
		},
	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			oldGroup := test.group
			oldPath := oldGroup.FullPath

			var newPath string
			if test.newParent != nil {
				// Migrating to non-root.
				newPath = test.newParent.FullPath + "/" + oldGroup.Name
			} else {
				// Migrating to root.
				newPath = oldGroup.Name
			}

			newGroup, err := testClient.client.Groups.MigrateGroup(ctx, &oldGroup, test.newParent)

			checkError(t, test.expectMsg, err)

			if test.isPositive {

				require.NotNil(t, newGroup)

				// Claimed new group fields must match, except full path and parent ID.
				assert.Equal(t, oldGroup.Name, newGroup.Name)
				assert.Equal(t, oldGroup.Description, newGroup.Description)
				assert.Equal(t, oldGroup.CreatedBy, newGroup.CreatedBy)

				// Claimed new group full path and parent ID must be correct.
				var fetchPath string
				var newParentID string
				if test.newParent == nil {
					// move was to root
					newParentID = ""
					fetchPath = oldGroup.Name
				} else {
					// move was under another parent
					newParentID = test.newParent.Metadata.ID
					fetchPath = test.newParent.FullPath + "/" + oldGroup.Name
				}
				assert.Equal(t, newParentID, newGroup.ParentID)
				assert.Equal(t, fetchPath, newGroup.FullPath)

				// Group can be fetched from new path.
				fetched, err := testClient.client.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(fetchPath))
				require.Nil(t, err)
				require.NotNil(t, fetched)

				// Fetched group fields match claimed new group.
				compareGroupsMigrate(t, oldGroup.Description, *newGroup, *fetched)

				// No group at old path.
				oldFetchedGroup, err := testClient.client.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(oldGroup.FullPath))
				assert.Nil(t, err)
				assert.Nil(t, oldFetchedGroup)

				// Just in case something went badly awry, no workspace at old path.
				oldFetchedWorkspace, err := testClient.client.Workspaces.GetWorkspaceByTRN(ctx, types.WorkspaceModelType.BuildTRN(oldGroup.FullPath))
				assert.Nil(t, err)
				assert.Nil(t, oldFetchedWorkspace)

				// All other relevant resources point to the new group path.
				for _, oldOther := range test.others.workspaces {
					newOther, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.FullPath, oldPath, newPath, 1), newOther.FullPath)
				}
				for _, oldOther := range test.others.serviceAccounts {
					newOther, err := testClient.client.ServiceAccounts.GetServiceAccountByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.GetResourcePath(), oldPath, newPath, 1), newOther.GetResourcePath())
				}
				for _, oldOther := range test.others.managedIdentities {
					newOther, err := testClient.client.ManagedIdentities.GetManagedIdentityByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.GetResourcePath(), oldPath, newPath, 1), newOther.GetResourcePath())
				}
				for _, oldOther := range test.others.gpgKeys {
					newOther, err := testClient.client.GPGKeys.GetGPGKeyByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.GetResourcePath(), oldPath, newPath, 1), newOther.GetResourcePath())
				}
				for _, oldOther := range test.others.terraformProviders {
					newOther, err := testClient.client.TerraformProviders.GetProviderByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.GetResourcePath(), oldPath, newPath, 1), newOther.GetResourcePath())
				}
				for _, oldOther := range test.others.memberships {
					newOther, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.Namespace.Path, oldPath, newPath, 1), newOther.Namespace.Path)
				}
				for _, oldOther := range test.others.variables {
					newOther, err := testClient.client.Variables.GetVariableByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.NamespacePath, oldPath, newPath, 1), newOther.NamespacePath)
				}
				for _, oldOther := range test.others.runners {
					newOther, err := testClient.client.Runners.GetRunnerByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.GetGroupPath(), oldPath, newPath, 1), newOther.GetGroupPath())
				}
				for _, oldOther := range test.others.activityEvents {
					result, err := testClient.client.ActivityEvents.GetActivityEvents(ctx, &GetActivityEventsInput{
						Filter: &ActivityEventFilter{
							ActivityEventIDs: []string{oldOther.Metadata.ID},
						},
					})
					assert.Nil(t, err)
					assert.NotNil(t, result)
					newOther := result.ActivityEvents[0]
					assert.Equal(t, strings.Replace(*oldOther.NamespacePath, oldPath, newPath, 1),
						*newOther.NamespacePath)
				}
				for _, oldOther := range test.others.vcsProviders {
					newOther, err := testClient.client.VCSProviders.GetProviderByID(ctx, oldOther.Metadata.ID)
					assert.Nil(t, err)
					assert.NotNil(t, newOther)
					assert.Equal(t, strings.Replace(oldOther.GetResourcePath(), oldPath, newPath, 1),
						newOther.GetResourcePath())
				}
			}
		})
	}
}

/*
	This is the plan for testing MigrateGroup removal of associations and updating of RootGroupID:

	Verify correct handling of the following:
		Remove assigned managed identities to a workspace
		Remove assigned service account namespace memberships
		Remove service accounts assigned to runners
		Remove workspace VCS provider links
		Update RootGroupID of Terraform providers
		Update RootGroupID of Terraform modules

	For each case, create this group hierarchy:
		A/B/C
		A/D
		E
		A/B/X/Y

	Workspaces:
		.../X/WX
		.../Y/WY

	Managed identities in A and B; assign them to WX and WY.
	Service accounts in A and B; make one of each a member of X, Y, WX, and WY.
	VCS providers in A and B; link them to WX and WY, respectively (only one provider per workspace allowed).
	A Terraform provider in each of X and Y.
	A Terraform module in each of X and Y.

	Case 1: Migrate X to A/B/C/X:
		Verify no associations got removed.
		Verify RootGroupID is still A.

	Case 2: Migrate X to A/X:
		Verify the associations from B got removed.
		Verify RootGroupID is still A.

	Case 3: Migrate X to A/D/X:
		Verify the associations from B got removed.
		Verify RootGroupID is still A.

	Case 4: Migrate X to E/X:
		Verify all associations got removed.
		Verify RootGroupID is now E.

	Case 5: Migrate X to top level:
		Verify that all associations got removed.
		Verify RootGroupID is now X.

*/

// TestMigrateGroupOther tests MigrateGroup's removal of assigned/inherited resources and updating of RootGroupID.
func TestMigrateGroupOther(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// The test cases in this function are all positive, because the negative cases are in the ...Basics function.
	type testCase struct {
		name                   string
		oldPath                string
		newPath                string
		expectRootGroupPath    string
		expectKeepAssociations []string
	}

	/*
		template test case:

		{
			name                   string
			oldPath                string
			newPath                string
			expectKeepAssociations []string
			expectRootGroupPath    string
		}
	*/

	testCases := []testCase{
		{
			name:                   "migrate X to A/B/C/X",
			oldPath:                "A/B/X",
			newPath:                "A/B/C/X",
			expectKeepAssociations: []string{"A", "B"}, // everything should remain
			expectRootGroupPath:    "A",
		},
		{
			name:                   "migrate X to A/X",
			oldPath:                "A/B/X",
			newPath:                "A/X",
			expectKeepAssociations: []string{"A"}, // from B should get removed, from A should remain
			expectRootGroupPath:    "A",
		},
		{
			name:                   "migrate X to A/D/X",
			oldPath:                "A/B/X",
			newPath:                "A/D/X",
			expectKeepAssociations: []string{"A"}, // from B should get removed, from A should remain
			expectRootGroupPath:    "A",
		},
		{
			name:                   "migrate X to E/X",
			oldPath:                "A/B/X",
			newPath:                "E/X",
			expectKeepAssociations: []string{}, // everything should get removed
			expectRootGroupPath:    "E",
		},
		{
			name:                   "migrate X to top level",
			oldPath:                "A/B/X",
			newPath:                "X",
			expectKeepAssociations: []string{}, // everything should get removed
			expectRootGroupPath:    "X",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			createdWarmupGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, warmupGroupsForMigrateOther)
			require.Nil(t, err)

			mainResources, err := createMigrateResources(ctx, testClient, &migrateGroupWarmupsInput{
				groups:             deriveGroupNames(createdWarmupGroups),
				groupPath2ID:       groupPath2ID,
				workspaces:         warmupWorkspacesForMigrateOther,
				managedIdentities:  warmupManagedIdentitiesForMigrateOther,
				serviceAccounts:    warmupServiceAccountsForMigrateOther,
				vcsProviders:       warmupVCSProvidersForMigrateOther,
				terraformProviders: warmupTerraformProvidersForMigrateOther,
				terraformModules:   warmupTerraformModulesForMigrateOther,
				roles:              warmupRolesForMigrateOther,
				runners:            warmupRunnersForGroupMigrationOther,
			})
			require.Nil(t, err)

			allAssociations, err := createAssociations(ctx, testClient.client,
				mainResources, &warmupAssociationsForMigrateOther)
			require.Nil(t, err)

			// Find the group object that will be migrated.
			var oldGroup *models.Group
			for _, grp := range mainResources.groups {
				if grp.FullPath == test.oldPath {
					oldGroup = &grp
					break
				}
			}
			require.NotNil(t, oldGroup)

			// Filter the associations to get those that are expected after migration.
			expectedAssociations := filterAssociations(allAssociations, test.expectKeepAssociations)

			// Look up the expected root group ID from its name.
			lookForPath := test.expectRootGroupPath
			if test.expectRootGroupPath == test.newPath {
				// The group being migrated is going to top level, so search for the group being migrated.
				lookForPath = test.oldPath
			}
			var expectRootGroup *models.Group
			for _, grp := range mainResources.groups {
				if grp.FullPath == lookForPath {
					expectRootGroup = &grp
					break
				}
			}
			require.NotNil(t, expectRootGroup)
			expectRootGroupID := expectRootGroup.Metadata.ID

			// Find the new parent (if any).
			var newParentGroup *models.Group
			if test.newPath != test.expectRootGroupPath {
				// The group is being moved to a new parent (not to top level).
				// Take off the last segment of test.newPath to get the new parent group's path.
				dummyGroup := &models.Group{
					ParentID: "dummy", // Simply so GetParentPath won't return empty string.
					FullPath: test.newPath,
				}
				newParentGroup, err = testClient.client.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(dummyGroup.GetParentPath()))
				assert.Nil(t, err)
			}

			// Do the migration.  An error is never expected, because this function does only positive cases.
			_, err = testClient.client.Groups.MigrateGroup(ctx, oldGroup, newParentGroup)
			assert.Nil(t, err)

			// Check the associations.
			actualAssociations, err := gatherActualAssociations(ctx, testClient.client, allAssociations)
			assert.Nil(t, err)
			assert.Equal(t, expectedAssociations, actualAssociations)

			// Check the root group IDs.
			for _, tp := range mainResources.terraformProviders {
				// Must fetch the TF provider to verify it got updated.
				found, err := testClient.client.TerraformProviders.GetProviderByID(ctx, tp.Metadata.ID)
				assert.Nil(t, err)
				assert.Equal(t, expectRootGroupID, found.RootGroupID)
			}
			for _, tm := range mainResources.terraformModules {
				// Must fetch the TF module to verify it got updated.
				found, err := testClient.client.TerraformModules.GetModuleByID(ctx, tm.Metadata.ID)
				assert.Nil(t, err)
				assert.Equal(t, expectRootGroupID, found.RootGroupID)
			}

			// Delete all resources so the next test case has a fresh start.
			for _, r := range mainResources.roles {
				rCopy := r
				err := testClient.client.Roles.DeleteRole(ctx, &rCopy)
				assert.Nil(t, err)
			}

			// Delete the groups in reverse order.
			toDelete := mainResources.groups
			for len(toDelete) > 0 {
				g1 := toDelete[len(toDelete)-1]

				// Must get the group to have its updated metadata version.
				g2, err := testClient.client.Groups.GetGroupByID(ctx, g1.Metadata.ID)
				require.Nil(t, err)

				// Now, we can delete the group.
				err = testClient.client.Groups.DeleteGroup(ctx, g2)
				assert.Nil(t, err)
				toDelete = toDelete[:len(toDelete)-1]
			}
		})
	}
}

func TestGetChildDepth(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupGroups, _, err := createInitialGroups(ctx, testClient, standardWarmupGroups)
	require.Nil(t, err)

	type testCase struct {
		group  *models.Group
		name   string
		expect int
	}

	testCases := []testCase{
		{
			name:   "top-level",
			group:  &createdWarmupGroups[0],
			expect: 2,
		},
		{
			name:   "second-level",
			group:  &createdWarmupGroups[4],
			expect: 1,
		},
		{
			name:   "third-and-leaf-level",
			group:  &createdWarmupGroups[5],
			expect: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualDepth, err := testClient.client.Groups.GetChildDepth(ctx, test.group)
			assert.Nil(t, err)
			assert.Equal(t, test.expect, actualDepth)
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
		RunnerTags:  []string{"tag1"},
	},
	{
		Description: "top level group 2 for testing group functions",
		FullPath:    "top-level-group-2",
		CreatedBy:   "someone-else",
		RunnerTags:  []string{"tag2", "tag3"},
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

// Warmup users for GetGroups search.
var warmupUsersForSearch = []models.User{
	{
		Username: "plain-user",
		Email:    "plain-user@invalid.example",
		Admin:    false,
		Active:   true,
	},
}

// Warmup service account(s) for GetGroups search.
var warmupServiceAccountsForSearch = []models.ServiceAccount{
	{
		Name:              "service-account-for-search",
		Description:       "service account for search",
		GroupID:           "top-level-group-1", // will be fixed later
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

// A role is a prerequisite for the namespace memberships.
var warmupRolesForSearch = []models.Role{
	{
		Name:        "role-a",
		Description: "role a for namespace membership tests",
		CreatedBy:   "someone-a",
	},
}

// Namespace memberships for GetGroups search.
var warmupNamespaceMembershipsForSearch = []CreateNamespaceMembershipInput{
	{
		UserID:           ptr.String("plain-user"),
		ServiceAccountID: nil,
		NamespacePath:    "top-level-group-1/2nd-level-group-1a",
		RoleID:           "role-a",
	},
	{
		UserID:           nil,
		ServiceAccountID: ptr.String("service-account-for-search"),
		NamespacePath:    "top-level-group-1/2nd-level-group-1b",
		RoleID:           "role-a",
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

// removeMatching finds a group in a slice with a specified full path.
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

// compareGroupsMigrate compares two groups for TestMigrateGroup
func compareGroupsMigrate(t *testing.T, expectedDescription string, expected, actual models.Group) {
	assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)

	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expectedDescription, actual.Description)
	assert.Equal(t, expected.FullPath, actual.FullPath)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
}

// More resources to test MigrateGroup basic functionality.
var warmupWorkspacesForGroupMigration = []models.Workspace{
	{
		FullPath:    "top-level-group-1/2nd-level-group-1a/workspace-20",
		Description: "workspace to help test group migration",
		CreatedBy:   "someone-w2",
	},
}

var warmupServiceAccountsForGroupMigration = []models.ServiceAccount{
	{
		Name:              "1-service-account-0",
		Description:       "service account 0",
		GroupID:           "top-level-group-1/2nd-level-group-1a", // will be fixed later
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

var warmupManagedIdentitiesForGroupMigration = []models.ManagedIdentity{
	{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     "top-level-group-1/2nd-level-group-1a", // will be fixed later
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
		// Resource path is not used when creating the object, but it is returned.
	},
}

var warmupGPGKeysForGroupMigration = []models.GPGKey{
	{
		GroupID:     "top-level-group-1/2nd-level-group-1a", // will be fixed later
		CreatedBy:   "someone-k0",
		ASCIIArmor:  "armor-0",
		Fingerprint: "fingerprint-0",
		GPGKeyID:    111222333444555,
	},
}

var warmupTerraformProvidersForGroupMigration = []models.TerraformProvider{
	{
		Name:        "1-terraform-provider-0",
		RootGroupID: "top-level-group-1",                    // will be fixed later
		GroupID:     "top-level-group-1/2nd-level-group-1a", // will be fixed later
		Private:     false,
		CreatedBy:   "someone-sv0",
	},
}

var warmupTeamsForGroupMigration = []models.Team{
	{
		Name:        "team-a",
		Description: "team a for namespace membership tests",
	},
}

var warmupRolesForGroupMigration = []models.Role{
	{
		Name:        "role-1",
		Description: "role 1 for testing group migration",
		CreatedBy:   "creator-of-roles",
	},
}

var warmupMembershipInputsForGroupMigration = []CreateNamespaceMembershipInput{
	{
		NamespacePath: "top-level-group-1/2nd-level-group-1a",
		TeamID:        ptr.String("team-a"), // will be fixed later
		RoleID:        "role-1",
	},
}

var warmupVariablesForGroupMigration = []models.Variable{
	{
		Category:      models.EnvironmentVariableCategory,
		NamespacePath: "top-level-group-1/2nd-level-group-1a/workspace-20",
		Key:           "key-0",
		Value:         ptr.String("value-0"),
	},
}

var warmupUsersForGroupMigration = []models.User{
	{
		Username: "user-1",
		Email:    "user-1@example.com",
		Active:   true,
	},
}

var warmupRunnersForGroupMigration = []models.Runner{
	{
		Type:    models.GroupRunnerType,
		Name:    "group-runner-a",
		GroupID: ptr.String("top-level-group-1/2nd-level-group-1a"),
	},
}

var warmupActivityEventsForGroupMigration = []models.ActivityEvent{
	{
		UserID:           ptr.String("user-1"),
		ServiceAccountID: nil,
		NamespacePath:    ptr.String("top-level-group-1/2nd-level-group-1a/workspace-20"),
		Action:           models.ActionCreate,
		TargetType:       models.TargetVariable,
		TargetID:         invalidID, // will be variable 0
	},
}

var warmupVCSProvidersForGroupMigration = []models.VCSProvider{
	{
		Name:              "1-vcs-provider-0",
		Description:       "vcs provider 0 for testing vcs providers",
		GroupID:           "top-level-group-1/2nd-level-group-1a", // will be fixed later
		CreatedBy:         "someone-vp0",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
}

// More resources to test MigrateGroup association and root group ID handling.
var warmupGroupsForMigrateOther = []models.Group{
	{
		Description: "warmup group A for testing MigrateGroup association and root group ID handling",
		FullPath:    "A",
		CreatedBy:   "someone-A",
	},
	{
		Description: "warmup group B for testing MigrateGroup association and root group ID handling",
		FullPath:    "A/B",
		CreatedBy:   "someone-B",
	},
	{
		Description: "warmup group C for testing MigrateGroup association and root group ID handling",
		FullPath:    "A/B/C",
		CreatedBy:   "someone-C",
	},
	{
		Description: "warmup group D for testing MigrateGroup association and root group ID handling",
		FullPath:    "A/D",
		CreatedBy:   "someone-D",
	},
	{
		Description: "warmup group E for testing MigrateGroup association and root group ID handling",
		FullPath:    "E",
		CreatedBy:   "someone-E",
	},
	{
		Description: "warmup group X for testing MigrateGroup association and root group ID handling",
		FullPath:    "A/B/X",
		CreatedBy:   "someone-B",
	},
	{
		Description: "warmup group Y for testing MigrateGroup association and root group ID handling",
		FullPath:    "A/B/X/Y",
		CreatedBy:   "someone-Y",
	},
}

var warmupWorkspacesForMigrateOther = []models.Workspace{
	{
		FullPath:    "A/B/X/WX",
		Description: "workspace to help test group migration other functionality",
		CreatedBy:   "someone-WX",
	},
	{
		FullPath:    "A/B/X/Y/WY",
		Description: "workspace to help test group migration other functionality",
		CreatedBy:   "someone-WY",
	},
}

var warmupManagedIdentitiesForMigrateOther = []models.ManagedIdentity{
	{
		Name:        "MI-A1",
		Description: "managed identity in group A 1",
		GroupID:     "A", // will be fixed later
		CreatedBy:   "someone-MI-A1",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-a1-data"),
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "MI-A2",
		Description: "managed identity in group A 2",
		GroupID:     "A", // will be fixed later
		CreatedBy:   "someone-MI-A2",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-a2-data"),
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "MI-B1",
		Description: "managed identity in group B 1",
		GroupID:     "A/B", // will be fixed later
		CreatedBy:   "someone-MI-B1",
		Type:        models.ManagedIdentityAzureFederated,
		Data:        []byte("managed-identity-b1-data"),
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "MI-B2",
		Description: "managed identity in group B2",
		GroupID:     "A/B", // will be fixed later
		CreatedBy:   "someone-MI-B2",
		Type:        models.ManagedIdentityAzureFederated,
		Data:        []byte("managed-identity-b2-data"),
		// Resource path is not used when creating the object, but it is returned.
	},
}

var warmupRunnersForGroupMigrationOther = []models.Runner{
	{
		Type:    models.GroupRunnerType,
		Name:    "group-runner-a",
		GroupID: ptr.String("A/B/X"),
	},
	{
		Type:    models.GroupRunnerType,
		Name:    "group-runner-b",
		GroupID: ptr.String("A/B/X"),
	},
}

var warmupServiceAccountsForMigrateOther = []models.ServiceAccount{
	{
		Name:              "SA-A-X",
		Description:       "service account in A for X",
		GroupID:           "A", // will be fixed later
		CreatedBy:         "someone-SA-A-X",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-A-Y",
		Description:       "service account in A for Y",
		GroupID:           "A", // will be fixed later
		CreatedBy:         "someone-SA-A-Y",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-A-WX",
		Description:       "service account in A for WX",
		GroupID:           "A", // will be fixed later
		CreatedBy:         "someone-SA-A-WX",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-A-WY",
		Description:       "service account in A for WY",
		GroupID:           "A", // will be fixed later
		CreatedBy:         "someone-SA-A-WY",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-B-X",
		Description:       "service account in B for X",
		GroupID:           "A/B", // will be fixed later
		CreatedBy:         "someone-SA-B-X",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-B-Y",
		Description:       "service account in B for Y",
		GroupID:           "A/B", // will be fixed later
		CreatedBy:         "someone-SA-B-Y",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-B-WX",
		Description:       "service account in B for WX",
		GroupID:           "A/B", // will be fixed later
		CreatedBy:         "someone-SA-B-WX",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		Name:              "SA-B-WY",
		Description:       "service account in B for WY",
		GroupID:           "A/B", // will be fixed later
		CreatedBy:         "someone-SA-B-WY",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

var warmupVCSProvidersForMigrateOther = []models.VCSProvider{
	{
		Name:              "VP-A",
		Description:       "vcs provider A for testing",
		GroupID:           "A",
		CreatedBy:         "someone-vp-A",
		OAuthClientID:     "client-id-A",
		OAuthClientSecret: "client-secret-A",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:              "VP-B",
		Description:       "vcs provider B for testing",
		GroupID:           "A/B",
		CreatedBy:         "someone-vp-B",
		OAuthClientID:     "client-id-B",
		OAuthClientSecret: "client-secret-B",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitLabProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
}

var warmupTerraformProvidersForMigrateOther = []models.TerraformProvider{
	{
		Name:        "TP-X",
		RootGroupID: "A",
		GroupID:     "A/B/X",
		Private:     false,
		CreatedBy:   "someone-TP-X",
	},
	{
		Name:        "TP-Y",
		RootGroupID: "A",
		GroupID:     "A/B/X/Y",
		Private:     true,
		CreatedBy:   "someone-TP-Y",
	},
}

var warmupTerraformModulesForMigrateOther = []models.TerraformModule{
	{
		Name:        "TM-X",
		System:      "aws",
		RootGroupID: "A",
		GroupID:     "A/B/X",
		Private:     false,
		CreatedBy:   "someone-TM-X",
	},
	{
		Name:        "TM-Y",
		System:      "azure",
		RootGroupID: "A",
		GroupID:     "A/B/X/Y",
		Private:     true,
		CreatedBy:   "someone-TM-Y",
	},
}

var warmupRolesForMigrateOther = []models.Role{
	{
		Name:      "role-1",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-2",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-3",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-4",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-5",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-6",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-7",
		CreatedBy: "role-creator",
	},
	{
		Name:      "role-8",
		CreatedBy: "role-creator",
	},
}

var warmupAssociationsForMigrateOther = associations{
	managedIdentityAssignments: []associateManagedIdentityAssignment{
		{
			filterBase:          "A",
			managedIdentityPath: "A/MI-A1",
			workspacePath:       "A/B/X/WX",
		},
		{
			filterBase:          "A",
			managedIdentityPath: "A/MI-A2",
			workspacePath:       "A/B/X/Y/WY",
		},
		{
			filterBase:          "B",
			managedIdentityPath: "A/B/MI-B1",
			workspacePath:       "A/B/X/WX",
		},
		{
			filterBase:          "B",
			managedIdentityPath: "A/B/MI-B2",
			workspacePath:       "A/B/X/Y/WY",
		},
	},
	serviceAccountRunnerAssignments: []associateServiceAccountRunnerAssignment{
		{
			filterBase:         "A",
			serviceAccountPath: "A/SA-A-X",
			runnerPath:         "A/B/X/group-runner-a",
		},
		{
			filterBase:         "B",
			serviceAccountPath: "A/B/SA-B-Y",
			runnerPath:         "A/B/X/group-runner-b",
		},
	},
	serviceAccountNamespaceMemberships: []associateServiceAccountNamespaceMembership{
		{
			filterBase:         "A",
			serviceAccountPath: "A/SA-A-X",
			namespacePath:      "A/B/X",
			roleName:           "role-1",
		},
		{
			filterBase:         "A",
			serviceAccountPath: "A/SA-A-Y",
			namespacePath:      "A/B/X/Y",
			roleName:           "role-2",
		},
		{
			filterBase:         "A",
			serviceAccountPath: "A/SA-A-WX",
			namespacePath:      "A/B/X/WX",
			roleName:           "role-3",
		},
		{
			filterBase:         "A",
			serviceAccountPath: "A/SA-A-WY",
			namespacePath:      "A/B/X/Y/WY",
			roleName:           "role-4",
		},
		{
			filterBase:         "B",
			serviceAccountPath: "A/B/SA-B-X",
			namespacePath:      "A/B/X",
			roleName:           "role-5",
		},
		{
			filterBase:         "B",
			serviceAccountPath: "A/B/SA-B-Y",
			namespacePath:      "A/B/X/Y",
			roleName:           "role-6",
		},
		{
			filterBase:         "B",
			serviceAccountPath: "A/B/SA-B-WX",
			namespacePath:      "A/B/X/WX",
			roleName:           "role-7",
		},
		{
			filterBase:         "B",
			serviceAccountPath: "A/B/SA-B-WY",
			namespacePath:      "A/B/X/Y/WY",
			roleName:           "role-8",
		},
	},
	workspaceVCSProviderLinks: []associateWorkspaceVCSProviderLink{
		{
			filterBase:    "A",
			providerPath:  "A/VP-A",
			workspacePath: "A/B/X/WX",
		},
		// Only one VCS provider allowed per workspace, so cannot do cross-product.
		{
			filterBase:    "B",
			providerPath:  "A/B/VP-B",
			workspacePath: "A/B/X/Y/WY",
		},
	},
}

// createMigrateResources creates other resources connected to the standard warmup groups for group migration testing.
func createMigrateResources(ctx context.Context, testClient *testClient,
	input *migrateGroupWarmupsInput,
) (*migrateGroupWarmupsOutput, error) {
	result := migrateGroupWarmupsOutput{}
	var err error

	// The groups are already created.
	result.groups = input.groups

	result.groupID2Path = map[string]string{}
	for _, g := range result.groups {
		result.groupID2Path[g.Metadata.ID] = g.FullPath
	}

	result.workspaces, err = createInitialWorkspaces(ctx, testClient, input.groupPath2ID, input.workspaces)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialWorkspaces: %w", err)
	}

	result.workspaceID2Path = map[string]string{}
	for _, w := range result.workspaces {
		result.workspaceID2Path[w.Metadata.ID] = w.FullPath
	}

	var serviceAccountName2ID map[string]string
	result.serviceAccounts, serviceAccountName2ID, err = createInitialServiceAccounts(ctx, testClient,
		input.groupPath2ID, input.serviceAccounts)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialServiceAccounts: %w", err)
	}

	result.managedIdentities, err = createInitialManagedIdentities(ctx, testClient,
		input.groupPath2ID, input.managedIdentities)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialManagedIdentities: %w", err)
	}

	result.gpgKeys, err = createInitialGPGKeys(ctx, testClient, input.gpgKeys, input.groupPath2ID)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialManagedIdentities: %w", err)
	}

	result.terraformProviders, _, err = createInitialTerraformProviders(ctx, testClient,
		input.terraformProviders, input.groupPath2ID)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialTerraformProviders: %w", err)
	}

	result.terraformModules, _, err = createInitialTerraformModules(ctx, testClient,
		input.terraformModules, input.groupPath2ID)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialTerraformModules: %w", err)
	}

	var teamName2ID map[string]string
	result.teams, teamName2ID, err = createInitialTeams(ctx, testClient, input.teams)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialTeams: %w", err)
	}

	var roleName2ID map[string]string
	result.roles, roleName2ID, err = createInitialRoles(ctx, testClient, input.roles)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialRoles: %w", err)
	}

	// do only a service account namespace membership
	result.memberships, err = createInitialNamespaceMemberships(ctx, testClient,
		teamName2ID, map[string]string{}, input.groupPath2ID, serviceAccountName2ID, roleName2ID,
		input.membershipInputs)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialNamespaceMemberships: %w", err)
	}

	result.variables, err = createInitialVariables(ctx, testClient, input.variables)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialVariables: %w", err)
	}

	// Make a modified copy of the input events to set the target ID.
	modifiedInputEvents := []models.ActivityEvent{}
	for _, oldEvent := range input.activityEvents {
		newEvent := oldEvent

		var newID string
		switch newEvent.TargetType {
		case models.TargetVariable:
			newID = result.variables[0].Metadata.ID
		case models.TargetServiceAccount:
			newID = result.serviceAccounts[0].Metadata.ID
		}
		newEvent.TargetID = newID

		modifiedInputEvents = append(modifiedInputEvents, newEvent)
	}

	var username2ID map[string]string
	result.users, username2ID, err = createInitialUsers(ctx, testClient, input.users)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialUsers: %s", err)
	}

	result.activityEvents, err = createInitialActivityEvents(ctx, testClient,
		modifiedInputEvents, username2ID, serviceAccountName2ID)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialActivityEvents: %s", err)
	}

	result.vcsProviders, err = createInitialVCSProviders(ctx, testClient,
		input.groupPath2ID, input.vcsProviders)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialVCSProviders: %s", err)
	}

	result.runners, _, err = createInitialRunners(ctx, testClient, input.runners, input.groupPath2ID)
	if err != nil {
		return nil, fmt.Errorf("error reported by createInitialRunners: %s", err)
	}

	return &result, nil
}

// createAssociations creates the associations used in testing MigrateGroup association removal
// and root group ID update.  The returned struct will have more fields filled in than the input.
// With very small numbers of resources, it should be fast enough without maps.
func createAssociations(ctx context.Context, dbClient *Client,
	resources *migrateGroupWarmupsOutput, inputs *associations,
) (*associations, error) {
	result := associations{
		managedIdentityAssignments:         []associateManagedIdentityAssignment{},
		serviceAccountNamespaceMemberships: []associateServiceAccountNamespaceMembership{},
		workspaceVCSProviderLinks:          []associateWorkspaceVCSProviderLink{},
		serviceAccountRunnerAssignments:    []associateServiceAccountRunnerAssignment{},
	}

	// managed identity assignments
	for _, input := range inputs.managedIdentityAssignments {

		// Find the workspace resource.
		var workspace *models.Workspace
		for _, ws := range resources.workspaces {
			if ws.FullPath == input.workspacePath {
				workspace = &ws
				break
			}
		}
		if workspace == nil {
			return nil, fmt.Errorf("failed to find workspace %s to associate with managed identity %s",
				input.workspacePath, input.managedIdentityPath)
		}

		// Find the managed identity resource.
		var managedIdentity *models.ManagedIdentity
		for _, mi := range resources.managedIdentities {
			if mi.GetResourcePath() == input.managedIdentityPath {
				managedIdentity = &mi
				break
			}
		}
		if managedIdentity == nil {
			return nil, fmt.Errorf("failed to find managed identity %s to associate with workspace %s",
				input.managedIdentityPath, input.workspacePath)
		}

		// Create the assignment.
		if err := dbClient.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
			managedIdentity.Metadata.ID, workspace.Metadata.ID); err != nil {
			return nil, fmt.Errorf("failed to associate managed identity %s with workspace %s: %v",
				input.managedIdentityPath, input.workspacePath, err)
		}

		// Record the assignment for the result.
		result.managedIdentityAssignments = append(result.managedIdentityAssignments,
			associateManagedIdentityAssignment{
				filterBase:          input.filterBase,
				workspacePath:       input.workspacePath,
				managedIdentityPath: input.managedIdentityPath,
				workspace:           workspace,
				managedIdentity:     managedIdentity,
			})
	}

	// service account namespace memberships
	for _, input := range inputs.serviceAccountNamespaceMemberships {

		// Find the service account.
		var serviceAccount *models.ServiceAccount
		for _, sa := range resources.serviceAccounts {
			if sa.GetResourcePath() == input.serviceAccountPath {
				serviceAccount = &sa
				break
			}
		}
		if serviceAccount == nil {
			return nil, fmt.Errorf("failed to find service account %s to associate with namespace %s",
				input.serviceAccountPath, input.namespacePath)
		}

		// Find the namespace.  The group or workspace found here is not actually used except for consistency checking.
		var grp *models.Group
		for _, ns := range resources.groups {
			if ns.FullPath == input.namespacePath {
				grp = &ns
				break
			}
		}
		var ws *models.Workspace
		for _, ns := range resources.workspaces {
			if ns.FullPath == input.namespacePath {
				ws = &ns
				break
			}
		}
		if (grp == nil) && (ws == nil) {
			return nil, fmt.Errorf("failed to find namespace %s to associate with service account %s",
				input.namespacePath, input.serviceAccountPath)
		}

		var role *models.Role
		for _, r := range resources.roles {
			if r.Name == input.roleName {
				role = &r
				break
			}
		}

		// Create the assignment.
		createdNamespaceMembership, err := dbClient.NamespaceMemberships.CreateNamespaceMembership(ctx,
			&CreateNamespaceMembershipInput{
				NamespacePath:    input.namespacePath,
				ServiceAccountID: &serviceAccount.Metadata.ID,
				RoleID:           role.Metadata.ID,
			})
		if err != nil {
			return nil, fmt.Errorf("failed to associate service account %s with namespace %s: %v",
				input.serviceAccountPath, input.namespacePath, err)
		}

		// Record the assignment for the result.
		result.serviceAccountNamespaceMemberships = append(result.serviceAccountNamespaceMemberships,
			associateServiceAccountNamespaceMembership{
				filterBase:          input.filterBase,
				serviceAccountPath:  input.serviceAccountPath,
				namespacePath:       input.namespacePath,
				namespaceMembership: createdNamespaceMembership,
			})
	}

	for _, input := range inputs.serviceAccountRunnerAssignments {
		// Find the service account.
		var serviceAccount *models.ServiceAccount
		for _, sa := range resources.serviceAccounts {
			if sa.GetResourcePath() == input.serviceAccountPath {
				serviceAccount = &sa
				break
			}
		}
		if serviceAccount == nil {
			return nil, fmt.Errorf("failed to find service account %s to associate with runner %s",
				input.serviceAccountPath, input.runnerPath)
		}

		// Find the namespace.  The group or workspace found here is not actually used except for consistency checking.
		var runner *models.Runner
		for _, r := range resources.runners {
			if r.GetResourcePath() == input.runnerPath {
				runner = &r
				break
			}
		}

		if runner == nil {
			return nil, fmt.Errorf("failed to find runner %s to associate with service account %s",
				input.runnerPath, input.serviceAccountPath)
		}

		// Create the assignment.
		err := dbClient.ServiceAccounts.AssignServiceAccountToRunner(ctx,
			serviceAccount.Metadata.ID, runner.Metadata.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to assign service account %s with runner %s: %v",
				input.serviceAccountPath, input.runnerPath, err)
		}

		// Record the assignment for the result.
		result.serviceAccountRunnerAssignments = append(result.serviceAccountRunnerAssignments,
			associateServiceAccountRunnerAssignment{
				filterBase:         input.filterBase,
				serviceAccountPath: input.serviceAccountPath,
				runnerPath:         input.runnerPath,
				runner:             runner,
				serviceAccount:     serviceAccount,
			})
	}

	// workspace VCS provider links
	for _, input := range inputs.workspaceVCSProviderLinks {

		// Find the workspace.
		var workspace *models.Workspace
		for _, ws := range resources.workspaces {
			if ws.FullPath == input.workspacePath {
				workspace = &ws
				break
			}
		}
		if workspace == nil {
			return nil, fmt.Errorf("failed to find workspace %s to associate with VCS provider %s",
				input.workspacePath, input.providerPath)
		}

		// Find the VCS provider.
		var provider *models.VCSProvider
		for _, vp := range resources.vcsProviders {
			if vp.GetResourcePath() == input.providerPath {
				provider = &vp
				break
			}
		}
		if provider == nil {
			return nil, fmt.Errorf("failed to find VCS provider %s to associate with workspace %s",
				input.providerPath, input.workspacePath)
		}

		// Create the assignment.
		createdLink, err := dbClient.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
			WorkspaceID: workspace.Metadata.ID,
			ProviderID:  provider.Metadata.ID,
			TokenNonce:  newResourceID(), // needs only to have valid UUID syntax
		})
		if err != nil {
			return nil, fmt.Errorf("failed to associate workspace %s with VCS provider %s: %v",
				input.workspacePath, input.providerPath, err)
		}

		// Record the assignment for the result.
		result.workspaceVCSProviderLinks = append(result.workspaceVCSProviderLinks,
			associateWorkspaceVCSProviderLink{
				filterBase:               input.filterBase,
				workspacePath:            input.workspacePath,
				providerPath:             input.providerPath,
				workspaceVCSProviderLink: createdLink,
			})
	}

	return &result, nil
}

// filterAssociations filters the associations to keep those specified.
func filterAssociations(input *associations, toKeep []string) *associations {
	keepMap := map[string]interface{}{}
	for _, s := range toKeep {
		keepMap[s] = interface{}(nil)
	}

	result := associations{
		managedIdentityAssignments:         []associateManagedIdentityAssignment{},
		serviceAccountNamespaceMemberships: []associateServiceAccountNamespaceMembership{},
		workspaceVCSProviderLinks:          []associateWorkspaceVCSProviderLink{},
		serviceAccountRunnerAssignments:    []associateServiceAccountRunnerAssignment{},
	}

	for _, mia := range input.managedIdentityAssignments {
		if _, ok := keepMap[mia.filterBase]; ok {
			result.managedIdentityAssignments = append(result.managedIdentityAssignments, mia)
		}
	}

	for _, sanm := range input.serviceAccountNamespaceMemberships {
		if _, ok := keepMap[sanm.filterBase]; ok {
			result.serviceAccountNamespaceMemberships = append(result.serviceAccountNamespaceMemberships, sanm)
		}
	}

	for _, wvpl := range input.workspaceVCSProviderLinks {
		if _, ok := keepMap[wvpl.filterBase]; ok {
			result.workspaceVCSProviderLinks = append(result.workspaceVCSProviderLinks, wvpl)
		}
	}

	for _, r := range input.serviceAccountRunnerAssignments {
		if _, ok := keepMap[r.filterBase]; ok {
			result.serviceAccountRunnerAssignments = append(result.serviceAccountRunnerAssignments, r)
		}
	}

	return &result
}

// gatherActualAssociations returns the associations that still exist.
func gatherActualAssociations(ctx context.Context, dbClient *Client, inputs *associations) (*associations, error) {
	result := associations{
		managedIdentityAssignments:         []associateManagedIdentityAssignment{},
		serviceAccountNamespaceMemberships: []associateServiceAccountNamespaceMembership{},
		workspaceVCSProviderLinks:          []associateWorkspaceVCSProviderLink{},
		serviceAccountRunnerAssignments:    []associateServiceAccountRunnerAssignment{},
	}

	for _, mia := range inputs.managedIdentityAssignments {
		// This gets the managed identity assignments still tied to the workspace in question so we can check them.
		candidates, err := dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, mia.workspace.Metadata.ID)
		if err != nil {
			if !tharsis.IsNotFoundError(err) {
				return nil, err
			}
		} else {
			for _, candidate := range candidates {
				if candidate.Metadata.ID == mia.managedIdentity.Metadata.ID {
					result.managedIdentityAssignments = append(result.managedIdentityAssignments, mia)
				}
			}
		}
	}

	for _, sra := range inputs.serviceAccountRunnerAssignments {
		sraCopy := sra
		// This gets the service account assignments still tied to the runner in question so we can check them.
		candidates, err := dbClient.ServiceAccounts.GetServiceAccounts(ctx, &GetServiceAccountsInput{
			Filter: &ServiceAccountFilter{
				RunnerID: &sraCopy.runner.Metadata.ID,
			},
		})
		if err != nil {
			if !tharsis.IsNotFoundError(err) {
				return nil, err
			}
		} else {
			for _, candidate := range candidates.ServiceAccounts {
				if candidate.Metadata.ID == sra.serviceAccount.Metadata.ID {
					result.serviceAccountRunnerAssignments = append(result.serviceAccountRunnerAssignments, sra)
				}
			}
		}
	}

	for _, sanm := range inputs.serviceAccountNamespaceMemberships {
		found, err := dbClient.NamespaceMemberships.GetNamespaceMembershipByID(ctx, sanm.namespaceMembership.Metadata.ID)
		if err != nil {
			if !tharsis.IsNotFoundError(err) {
				return nil, err
			}
		} else if found != nil {
			result.serviceAccountNamespaceMemberships = append(result.serviceAccountNamespaceMemberships, sanm)
		}
	}

	for _, wvpl := range inputs.workspaceVCSProviderLinks {
		found, err := dbClient.WorkspaceVCSProviderLinks.GetLinkByID(ctx, wvpl.workspaceVCSProviderLink.Metadata.ID)
		if err != nil {
			if !tharsis.IsNotFoundError(err) {
				return nil, err
			}
		} else if found != nil {
			result.workspaceVCSProviderLinks = append(result.workspaceVCSProviderLinks, wvpl)
		}
	}

	return &result, nil
}
