//go:build integration

package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for GroupSortableField
func (g GroupSortableField) getValue() string {
	return string(g)
}

func TestGroups_CreateGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		groupName       string
		description     string
		parentID        string
		fullPath        string
	}

	testCases := []testCase{
		{
			name:        "create group",
			groupName:   "test-group",
			description: "test group description",
			fullPath:    "test-group",
		},
		{
			name:            "negative, child without parent",
			groupName:       "orphan-child",
			description:     "this is a child without a parent",
			parentID:        invalidID,
			fullPath:        "missing-parent/orphan-child",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
				Name:        test.groupName,
				Description: test.description,
				ParentID:    test.parentID,
				FullPath:    test.fullPath,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, group)

			assert.Equal(t, test.groupName, group.Name)
			assert.Equal(t, test.description, group.Description)
			assert.Equal(t, test.fullPath, group.FullPath)
			assert.NotEmpty(t, group.Metadata.ID)
		})
	}
}

func TestGroups_UpdateGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-update",
		Description: "original description",
		FullPath:    "test-group-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		description     string
	}

	testCases := []testCase{
		{
			name:        "update group",
			version:     createdGroup.Metadata.Version,
			description: "updated description",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			description:     "should not update",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			groupToUpdate := *createdGroup
			groupToUpdate.Metadata.Version = test.version
			groupToUpdate.Description = test.description

			updatedGroup, err := testClient.client.Groups.UpdateGroup(ctx, &groupToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedGroup)

			assert.Equal(t, test.description, updatedGroup.Description)
			assert.Equal(t, createdGroup.Metadata.Version+1, updatedGroup.Metadata.Version)
		})
	}
}

func TestGroups_DeleteGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-delete",
		Description: "group to delete",
		FullPath:    "test-group-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:    "delete group",
			id:      createdGroup.Metadata.ID,
			version: createdGroup.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdGroup.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Groups.DeleteGroup(ctx, &models.Group{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			// Verify group was deleted
			group, err := testClient.client.Groups.GetGroupByID(ctx, test.id)
			assert.Nil(t, group)
			assert.Nil(t, err)
		})
	}
}

func TestGroups_GetGroupByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-get-by-id",
		Description: "test group for get by id",
		FullPath:    "test-group-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectGroup     bool
	}

	testCases := []testCase{
		{
			name:        "get resource by id",
			id:          createdGroup.Metadata.ID,
			expectGroup: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			group, err := testClient.client.Groups.GetGroupByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGroup {
				require.NotNil(t, group)
				assert.Equal(t, test.id, group.Metadata.ID)
			} else {
				assert.Nil(t, group)
			}
		})
	}
}

func TestGroups_GetGroups(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test groups
	groups := []models.Group{
		{
			Name:        "test-group-list-1",
			Description: "test group for list 1",
			FullPath:    "test-group-list-1",
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-group-list-2",
			Description: "test group for list 2",
			FullPath:    "test-group-list-2",
			CreatedBy:   "db-integration-tests",
		},
	}

	createdGroups := []models.Group{}
	for _, group := range groups {
		created, err := testClient.client.Groups.CreateGroup(ctx, &group)
		require.NoError(t, err)
		createdGroups = append(createdGroups, *created)
	}
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-groups",
		Email:    "test-user-groups@test.com",
	})
	require.NoError(t, err)

	_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:  user.Metadata.ID,
		GroupID: &createdGroups[0].Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetGroupsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all groups",
			input:       &GetGroupsInput{},
			expectCount: len(createdGroups),
		},
		{
			name: "get groups with favorite filter",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					FavoriteUserID: &user.Metadata.ID,
				},
			},
			expectCount: 1,
		},
		{
			name: "exclude favorited groups",
			input: &GetGroupsInput{
				Filter: &GroupFilter{
					ExcludeFavoriteUserID: &user.Metadata.ID,
				},
			},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Groups.GetGroups(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Groups, test.expectCount)
		})
	}
}

func TestGroups_GetGroupsWithMembershipFilters(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Build a hierarchy:
	//   mbr-root            top-level, the caller's "root" membership
	//   mbr-root/mbr-child  a descendant of the root membership
	//   mbr-other           top-level, NOT a membership of the caller
	//   team_a              top-level, a membership whose path contains a LIKE wildcard ('_')
	//   team_a/child        a legitimate descendant of team_a
	//   teamXa              top-level decoy: an unescaped "team_a/%" LIKE would match its tree
	//   teamXa/child        the decoy descendant that must NOT leak into team_a's results
	root, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "mbr-root", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	_, err = testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "mbr-child", ParentID: root.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	_, err = testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "mbr-other", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	teamA, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "team_a", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	_, err = testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "child", ParentID: teamA.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	teamX, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "teamXa", CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	_, err = testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "child", ParentID: teamX.Metadata.ID, CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	// The RootOnly membership filter matches by namespace ID, so resolve the root group's
	// namespace ID.
	rootNS, err := getNamespaceByPath(ctx, testClient.client.getConnection(ctx), "mbr-root")
	require.NoError(t, err)
	require.NotNil(t, rootNS)

	rootMembership := models.MembershipNamespace{ID: rootNS.id, Path: "mbr-root"}

	teamANS, err := getNamespaceByPath(ctx, testClient.client.getConnection(ctx), "team_a")
	require.NoError(t, err)
	require.NotNil(t, teamANS)

	teamAMembership := models.MembershipNamespace{ID: teamANS.id, Path: "team_a"}

	testCases := []struct {
		filter      *GroupFilter
		name        string
		expectPaths []string
	}{
		{
			name:        "membership filter returns the root and its descendants",
			filter:      &GroupFilter{RootNamespaceMemberships: []models.MembershipNamespace{rootMembership}},
			expectPaths: []string{"mbr-root", "mbr-root/mbr-child"},
		},
		{
			name:        "membership filter with root-only returns only the exact root",
			filter:      &GroupFilter{RootNamespaceMemberships: []models.MembershipNamespace{rootMembership}, RootOnly: true},
			expectPaths: []string{"mbr-root"},
		},
		{
			// Regression: an unescaped "team_a/%" LIKE prefix would also match "teamXa/child"
			// because '_' is a LIKE single-char wildcard. The escaped prefix must only return
			// team_a and its real descendants.
			name:        "membership root containing a LIKE wildcard does not leak sibling namespaces",
			filter:      &GroupFilter{RootNamespaceMemberships: []models.MembershipNamespace{teamAMembership}},
			expectPaths: []string{"team_a", "team_a/child"},
		},
		{
			name:        "root-only without a membership restriction returns top-level groups",
			filter:      &GroupFilter{RootOnly: true},
			expectPaths: []string{"mbr-root", "mbr-other", "team_a", "teamXa"},
		},
		{
			name:        "empty memberships match nothing",
			filter:      &GroupFilter{RootNamespaceMemberships: []models.MembershipNamespace{}},
			expectPaths: []string{},
		},
		{
			name:        "empty memberships with root-only match nothing",
			filter:      &GroupFilter{RootNamespaceMemberships: []models.MembershipNamespace{}, RootOnly: true},
			expectPaths: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Groups.GetGroups(ctx, &GetGroupsInput{Filter: test.filter})
			require.NoError(t, err)

			gotPaths := []string{}
			for _, g := range result.Groups {
				gotPaths = append(gotPaths, g.FullPath)
			}
			assert.ElementsMatch(t, test.expectPaths, gotPaths)
		})
	}
}

func TestGroups_GetGroupsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Name:        fmt.Sprintf("test-group-pagination-%d", i),
			Description: fmt.Sprintf("test group for pagination %d", i),
			FullPath:    fmt.Sprintf("test-group-pagination-%d", i),
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	// Only test the sortable fields that work reliably
	sortableFields := []sortableField{
		GroupSortableFieldFullPathAsc,
		GroupSortableFieldFullPathDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := GroupSortableField(sortByField.getValue())

		result, err := testClient.client.Groups.GetGroups(ctx, &GetGroupsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Groups {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestGroups_GetGroupByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	createdGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-trn",
		Description: "test group for trn",
		FullPath:    "test-group-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectGroup     bool
	}

	testCases := []testCase{
		{
			name:        "get resource by TRN",
			trn:         createdGroup.Metadata.TRN,
			expectGroup: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:group:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			group, err := testClient.client.Groups.GetGroupByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGroup {
				require.NotNil(t, group)
				assert.Equal(t, test.trn, group.Metadata.TRN)
			} else {
				assert.Nil(t, group)
			}
		})
	}
}

// TestGroups_MigrateGroup re-parents a group and checks the cleanup of the assignments the move puts
// out of scope: policy approvers, managed identities, runner service accounts, service account
// namespace memberships and workspace VCS provider links. The fixture is built and migrated once,
// because each cleanup is a single statement over the whole tree: what they have to get right is the
// several (namespace, principal) pairs each must judge differently in one pass.
func TestGroups_MigrateGroup(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// origin                     old parent, does not move
	// |-- src                    the group being migrated, to underneath dest
	// |   |-- team
	// |   |   |-- sub
	// |   |   |-- ws-a
	// |   |   `-- ws-b
	// |   |-- teamwork           name-prefixed sibling of team
	// |   |   `-- x
	// |   |-- my_team            underscore is a single-character wildcard to LIKE
	// |   `-- myzteam            which is what my_team would match
	// |       `-- p
	// `-- other
	// |   `-- ws-c
	// dest                       new parent
	groups := map[string]*models.Group{}
	for _, fullPath := range []string{
		"origin",
		"origin/src",
		"origin/src/team",
		"origin/src/team/sub",
		"origin/src/teamwork",
		"origin/src/teamwork/x",
		"origin/src/my_team",
		"origin/src/myzteam",
		"origin/src/myzteam/p",
		"origin/other",
		"dest",
	} {
		name := fullPath
		var parentID string
		if i := strings.LastIndex(fullPath, "/"); i >= 0 {
			parent, ok := groups[fullPath[:i]]
			require.True(t, ok, "parent of %s must be created first", fullPath)
			name = fullPath[i+1:]
			parentID = parent.Metadata.ID
		}

		group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
			Name:      name,
			ParentID:  parentID,
			FullPath:  fullPath,
			CreatedBy: "db-integration-tests",
		})
		require.NoError(t, err)

		groups[fullPath] = group
	}

	maxJobDuration := int32(forTestMaxJobDuration.Minutes())
	workspaces := map[string]*models.Workspace{}
	for _, fullPath := range []string{
		"origin/src/team/ws-a",
		"origin/src/team/ws-b",
		"origin/other/ws-c",
	} {
		i := strings.LastIndex(fullPath, "/")
		workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
			Name:           fullPath[i+1:],
			GroupID:        groups[fullPath[:i]].Metadata.ID,
			FullPath:       fullPath,
			MaxJobDuration: &maxJobDuration,
			CreatedBy:      "db-integration-tests",
		})
		require.NoError(t, err)

		workspaces[fullPath] = workspace
	}

	// One service account per group any of the cases below draws from, keyed by the path of the group
	// it lives in -- the only thing about it the cleanups look at.
	serviceAccounts := map[string]*models.ServiceAccount{}
	for _, groupPath := range []string{
		"origin",
		"origin/src",
		"origin/src/team",
		"origin/src/team/sub",
		"origin/src/my_team",
		"dest",
	} {
		serviceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
			Name:      "approver",
			GroupID:   groups[groupPath].Metadata.ID,
			CreatedBy: "db-integration-tests",
		})
		require.NoError(t, err)

		serviceAccounts[groupPath] = serviceAccount
	}

	// assignmentCase describes one (namespace, principal) pair the cleanup has to judge. resourcePath
	// is the namespace holding the assignment -- the group that owns the policy, membership or runner,
	// or the workspace the identity or link is attached to. principalPath is the group the principal
	// lives in, which is all the cleanup knows about it.
	type assignmentCase struct {
		name          string
		resourcePath  string
		principalPath string
		expectKept    bool
	}

	// A policy per case, carrying only the approver that case is about, so a failure names one
	// condition rather than a list difference. These cases cover the shape of the scope test itself;
	// the tables that follow cover the wiring of the same test into the other four assignments.
	policyCases := []assignmentCase{
		{
			name:          "policy approver in an ancestor group inside the migrated tree is kept",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin/src/team",
			expectKept:    true,
		},
		{
			name:          "policy approver in the migrated group itself is kept",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin/src",
			expectKept:    true,
		},
		{
			name:          "policy approver in the policy's own group is kept",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin/src/team/sub",
			expectKept:    true,
		},
		{
			name:          "policy approver in the old parent group is removed",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin",
		},
		{
			// Scope is judged against where the tree landed, not where it came from, so an approver the
			// move brings into scope is kept rather than swept up for having been out of scope before.
			name:          "policy approver in the new parent group is kept",
			resourcePath:  "origin/src/team/sub",
			principalPath: "dest",
			expectKept:    true,
		},
		{
			// team is a string prefix of teamwork, so this row only goes away if the prefix test
			// requires the path separator.
			name:          "policy approver in a name-prefixed sibling of an ancestor is removed",
			resourcePath:  "origin/src/teamwork/x",
			principalPath: "origin/src/team",
		},
		{
			// my_team matches myzteam under LIKE, where the underscore is a wildcard. This is the case
			// that makes the prefix test starts_with rather than a LIKE against a column, which has no
			// literal pattern to run through escapeLikePattern.
			name:          "policy approver whose group name contains a LIKE wildcard is removed",
			resourcePath:  "origin/src/myzteam/p",
			principalPath: "origin/src/my_team",
		},
		{
			name:          "policy approver in a group beneath the policy's group is removed",
			resourcePath:  "origin/src/team",
			principalPath: "origin/src/team/sub",
		},
		{
			// The policy is owned by the migrated group itself, which the subtree test only reaches
			// through its equals arm rather than the path prefix.
			name:          "policy approver out of scope for a policy on the migrated group is removed",
			resourcePath:  "origin/src",
			principalPath: "origin",
		},
		{
			name:          "policy approver of a policy outside the migrated tree is untouched",
			resourcePath:  "origin/other",
			principalPath: "origin",
			expectKept:    true,
		},
	}

	policyIDs := make([]string, len(policyCases))
	for i, test := range policyCases {
		policy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
			GroupID: groups[test.resourcePath].Metadata.ID,
			Name:    fmt.Sprintf("test-policy-%d", i),
			Kind:    models.PolicyKindOPA,
			OPAData: &models.OPAPolicyData{
				PackageSource:                  "trn:package_version:origin/test-package/1.0.0",
				EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
				SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				Stage:                          models.RunTaskStageNamePostPlan,
			},
			RequiredApprovals:        1,
			AllowedServiceAccountIDs: []string{serviceAccounts[test.principalPath].Metadata.ID},
			CreatedBy:                "db-integration-tests",
		})
		require.NoError(t, err)

		policyIDs[i] = policy.Metadata.ID
	}

	managedIdentityCases := []assignmentCase{
		{
			// The identity's group moved along with the workspace, so nothing about the assignment
			// changed and it has to survive.
			name:          "managed identity in an ancestor group inside the migrated tree stays assigned",
			resourcePath:  "origin/src/team/ws-a",
			principalPath: "origin/src/team",
			expectKept:    true,
		},
		{
			name:          "managed identity in the migrated group itself stays assigned",
			resourcePath:  "origin/src/team/ws-a",
			principalPath: "origin/src",
			expectKept:    true,
		},
		{
			name:          "managed identity in the old parent group is unassigned",
			resourcePath:  "origin/src/team/ws-a",
			principalPath: "origin",
		},
		{
			name:          "managed identity assigned to a workspace outside the migrated tree is untouched",
			resourcePath:  "origin/other/ws-c",
			principalPath: "origin",
			expectKept:    true,
		},
	}

	managedIdentityIDs := make([]string, len(managedIdentityCases))
	for i, test := range managedIdentityCases {
		identity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
			Name:      fmt.Sprintf("test-identity-%d", i),
			GroupID:   groups[test.principalPath].Metadata.ID,
			Type:      models.ManagedIdentityAWSFederated,
			Data:      []byte("test-identity-data"),
			CreatedBy: "db-integration-tests",
		})
		require.NoError(t, err)

		require.NoError(t, testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
			identity.Metadata.ID, workspaces[test.resourcePath].Metadata.ID))

		managedIdentityIDs[i] = identity.Metadata.ID
	}

	runnerCases := []assignmentCase{
		{
			name:          "runner service account in an ancestor group inside the migrated tree stays assigned",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin/src/team",
			expectKept:    true,
		},
		{
			name:          "runner service account in the runner's own group stays assigned",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin/src/team/sub",
			expectKept:    true,
		},
		{
			name:          "runner service account in the old parent group is unassigned",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin",
		},
	}

	// Runners are keyed by the group they belong to, so the cases above share one where they can.
	runners := map[string]*models.Runner{}
	for _, test := range runnerCases {
		runner, ok := runners[test.resourcePath]
		if !ok {
			groupID := groups[test.resourcePath].Metadata.ID
			created, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
				Name:      fmt.Sprintf("test-runner-%d", len(runners)),
				GroupID:   &groupID,
				Type:      models.GroupRunnerType,
				CreatedBy: "db-integration-tests",
			})
			require.NoError(t, err)

			runners[test.resourcePath] = created
			runner = created
		}

		require.NoError(t, testClient.client.ServiceAccounts.AssignServiceAccountToRunner(ctx,
			serviceAccounts[test.principalPath].Metadata.ID, runner.Metadata.ID))
	}

	membershipCases := []assignmentCase{
		{
			name:          "service account membership of a group, from an ancestor inside the migrated tree, is kept",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin/src/team",
			expectKept:    true,
		},
		{
			name:          "service account membership of a group, from the old parent group, is removed",
			resourcePath:  "origin/src/team/sub",
			principalPath: "origin",
		},
		{
			// A workspace namespace reaches the subtree test through its path prefix only, never the
			// equals arm, since a workspace and a group never share a path.
			name:          "service account membership of a workspace, from an ancestor inside the migrated tree, is kept",
			resourcePath:  "origin/src/team/ws-a",
			principalPath: "origin/src/team",
			expectKept:    true,
		},
		{
			name:          "service account membership of a workspace, from the old parent group, is removed",
			resourcePath:  "origin/src/team/ws-a",
			principalPath: "origin",
		},
	}

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	membershipIDs := make([]string, len(membershipCases))
	for i, test := range membershipCases {
		serviceAccountID := serviceAccounts[test.principalPath].Metadata.ID
		membership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
			NamespacePath:    test.resourcePath,
			ServiceAccountID: &serviceAccountID,
			RoleID:           role.Metadata.ID,
		})
		require.NoError(t, err)

		membershipIDs[i] = membership.Metadata.ID
	}

	// A user membership in the migrated tree has no group to be out of scope of. It is only spared
	// because the cleanup joins through service_accounts, which a membership held by a user cannot
	// match, so it is worth pinning down.
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-migrate",
		Email:    "test-user-migrate@test.com",
	})
	require.NoError(t, err)

	userMembership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: "origin/src/team/sub",
		UserID:        &user.Metadata.ID,
		RoleID:        role.Metadata.ID,
	})
	require.NoError(t, err)

	// A workspace holds at most one VCS provider link, so each case needs its own workspace.
	linkCases := []assignmentCase{
		{
			name:          "VCS provider link from an ancestor group inside the migrated tree is kept",
			resourcePath:  "origin/src/team/ws-a",
			principalPath: "origin/src/team",
			expectKept:    true,
		},
		{
			name:          "VCS provider link from the old parent group is removed",
			resourcePath:  "origin/src/team/ws-b",
			principalPath: "origin",
		},
	}

	linkIDs := make([]string, len(linkCases))
	for i, test := range linkCases {
		provider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
			Name:      fmt.Sprintf("test-provider-%d", i),
			GroupID:   groups[test.principalPath].Metadata.ID,
			Type:      models.GitLabProviderType,
			URL:       url.URL{Scheme: "https", Host: "gitlab.example.com"},
			CreatedBy: "db-integration-tests",
		})
		require.NoError(t, err)

		link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
			WorkspaceID:    workspaces[test.resourcePath].Metadata.ID,
			ProviderID:     provider.Metadata.ID,
			TokenNonce:     uuid.New().String(),
			RepositoryPath: "test-org/test-repo",
			Branch:         "main",
			CreatedBy:      "db-integration-tests",
		})
		require.NoError(t, err)

		linkIDs[i] = link.Metadata.ID
	}

	migratedGroup, err := testClient.client.Groups.MigrateGroup(ctx, groups["origin/src"], groups["dest"])
	require.NoError(t, err)
	require.NotNil(t, migratedGroup)

	assert.Equal(t, "dest/src", migratedGroup.FullPath)
	assert.Equal(t, groups["dest"].Metadata.ID, migratedGroup.ParentID)
	assert.Equal(t, groups["origin/src"].Metadata.Version+1, migratedGroup.Metadata.Version)

	// Every path comparison in the cleanups runs after the subtree has been re-pathed, so the cases
	// below mean nothing unless this holds.
	descendant, err := testClient.client.Groups.GetGroupByID(ctx, groups["origin/src/team/sub"].Metadata.ID)
	require.NoError(t, err)
	require.NotNil(t, descendant)
	assert.Equal(t, "dest/src/team/sub", descendant.FullPath)

	for i, test := range policyCases {
		t.Run(test.name, func(t *testing.T) {
			policy, err := testClient.client.Policies.GetPolicyByID(ctx, policyIDs[i])
			require.NoError(t, err)
			require.NotNil(t, policy)

			expectIDs := []string{}
			if test.expectKept {
				expectIDs = append(expectIDs, serviceAccounts[test.principalPath].Metadata.ID)
			}

			assert.ElementsMatch(t, expectIDs, policy.AllowedServiceAccountIDs)
		})
	}

	for i, test := range managedIdentityCases {
		t.Run(test.name, func(t *testing.T) {
			identities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx,
				workspaces[test.resourcePath].Metadata.ID)
			require.NoError(t, err)

			assigned := false
			for _, identity := range identities {
				if identity.Metadata.ID == managedIdentityIDs[i] {
					assigned = true
				}
			}

			assert.Equal(t, test.expectKept, assigned)
		})
	}

	for _, test := range runnerCases {
		t.Run(test.name, func(t *testing.T) {
			runnerID := runners[test.resourcePath].Metadata.ID
			result, err := testClient.client.ServiceAccounts.GetServiceAccounts(ctx, &GetServiceAccountsInput{
				Filter: &ServiceAccountFilter{RunnerID: &runnerID},
			})
			require.NoError(t, err)

			assigned := false
			for _, serviceAccount := range result.ServiceAccounts {
				if serviceAccount.Metadata.ID == serviceAccounts[test.principalPath].Metadata.ID {
					assigned = true
				}
			}

			assert.Equal(t, test.expectKept, assigned)
		})
	}

	for i, test := range membershipCases {
		t.Run(test.name, func(t *testing.T) {
			membership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, membershipIDs[i])
			require.NoError(t, err)

			assert.Equal(t, test.expectKept, membership != nil)
		})
	}

	t.Run("user membership in the migrated tree is untouched", func(t *testing.T) {
		membership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, userMembership.Metadata.ID)
		require.NoError(t, err)

		assert.NotNil(t, membership)
	})

	for i, test := range linkCases {
		t.Run(test.name, func(t *testing.T) {
			link, err := testClient.client.WorkspaceVCSProviderLinks.GetLinkByID(ctx, linkIDs[i])
			require.NoError(t, err)

			assert.Equal(t, test.expectKept, link != nil)
		})
	}
}
