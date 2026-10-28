//go:build integration

package db

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestGetNamespace(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-group",
		FullPath:  "test-get-namespace-group",
		CreatedBy: models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-get-namespace-workspace",
		FullPath:       "test-get-namespace-group/test-get-namespace-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	t.Run("group path returns NamespaceTypeGroup with the group", func(t *testing.T) {
		got, err := testClient.client.Namespaces.GetNamespace(ctx, group.FullPath)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, models.NamespaceTypeGroup, got.Type)
		require.NotNil(t, got.Group)
		assert.Nil(t, got.Workspace)
		assert.Equal(t, group.Metadata.ID, got.Group.Metadata.ID)
		assert.Equal(t, group.FullPath, got.Group.FullPath)
	})

	t.Run("workspace path returns NamespaceTypeWorkspace with the workspace", func(t *testing.T) {
		got, err := testClient.client.Namespaces.GetNamespace(ctx, workspace.FullPath)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, models.NamespaceTypeWorkspace, got.Type)
		require.NotNil(t, got.Workspace)
		assert.Nil(t, got.Group)
		assert.Equal(t, workspace.Metadata.ID, got.Workspace.Metadata.ID)
		assert.Equal(t, workspace.FullPath, got.Workspace.FullPath)
	})

	t.Run("non-existent path returns nil", func(t *testing.T) {
		got, err := testClient.client.Namespaces.GetNamespace(ctx, "does-not-exist")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestGetNamespaceByGroupID(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	groupA, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-group-id-a",
		CreatedBy: models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	groupB, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-group-id-b",
		CreatedBy: models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		expectNamespace *namespaceRow
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "positive, group a",
			id:   groupA.Metadata.ID,
			expectNamespace: &namespaceRow{
				path:    groupA.FullPath,
				groupID: groupA.Metadata.ID,
				version: initialResourceVersion,
			},
		},
		{
			name: "positive, group b",
			id:   groupB.Metadata.ID,
			expectNamespace: &namespaceRow{
				path:    groupB.FullPath,
				groupID: groupB.Metadata.ID,
				version: initialResourceVersion,
			},
		},
		{
			name: "negative, does not exist",
			id:   nonExistentID,
		},
		{
			name:            "negative, invalid",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := getNamespaceByGroupID(ctx, testClient.client.getConnection(ctx), test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}
		})
	}
}

func TestGetNamespaceByWorkspaceID(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-workspace-id",
		CreatedBy: models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-get-namespace-by-workspace-id-ws",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		expectNamespace *namespaceRow
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "positive, workspace",
			id:   workspace.Metadata.ID,
			expectNamespace: &namespaceRow{
				path:        workspace.FullPath,
				workspaceID: workspace.Metadata.ID,
				version:     initialResourceVersion,
			},
		},
		{
			name: "negative, does not exist",
			id:   nonExistentID,
		},
		{
			name:            "negative, invalid",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := getNamespaceByWorkspaceID(ctx, testClient.client.getConnection(ctx), test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}
		})
	}
}

func TestGetNamespaceByPath(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	parentGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-path-parent",
		CreatedBy: models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	childGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-path-child",
		ParentID:  parentGroup.Metadata.ID,
		CreatedBy: models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-get-namespace-by-path-ws",
		GroupID:        childGroup.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		path            string
		expectNamespace *namespaceRow
	}

	testCases := []testCase{
		{
			name: "positive, top-level group path",
			path: parentGroup.FullPath,
			expectNamespace: &namespaceRow{
				path:    parentGroup.FullPath,
				groupID: parentGroup.Metadata.ID,
				version: initialResourceVersion,
			},
		},
		{
			name: "positive, nested group path",
			path: childGroup.FullPath,
			expectNamespace: &namespaceRow{
				path:    childGroup.FullPath,
				groupID: childGroup.Metadata.ID,
				version: initialResourceVersion,
			},
		},
		{
			name: "positive, workspace path",
			path: workspace.FullPath,
			expectNamespace: &namespaceRow{
				path:        workspace.FullPath,
				workspaceID: workspace.Metadata.ID,
				version:     initialResourceVersion,
			},
		},
		{
			name: "negative, non-existent top-level path",
			path: "non-exist-top-level",
		},
		{
			name: "negative, non-existent path under an existing group",
			path: parentGroup.FullPath + "/non-exist-sub-path",
		},
		{
			name: "negative, non-existent path under an existing workspace",
			path: workspace.FullPath + "/non-exist-below-workspace",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := getNamespaceByPath(ctx, testClient.client.getConnection(ctx), test.path)
			require.NoError(t, err)

			if test.expectNamespace != nil {
				require.NotNil(t, gotNamespace)
				compareNamespaceRows(t, test.expectNamespace, gotNamespace)
			} else {
				assert.Nil(t, gotNamespace)
			}
		})
	}
}

func TestGetNamespaces(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	parent, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{Name: "list-parent", CreatedBy: models.Actor{Subject: "db-integration-tests"}})
	require.NoError(t, err)
	child, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{Name: "child-group", ParentID: parent.Metadata.ID, CreatedBy: models.Actor{Subject: "db-integration-tests"}})
	require.NoError(t, err)
	_, err = testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "child-workspace",
		GroupID:        parent.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)
	// A grandchild, which a direct-children listing must leave out
	_, err = testClient.client.Groups.CreateGroup(ctx, &models.Group{Name: "grandchild-group", ParentID: child.Metadata.ID, CreatedBy: models.Actor{Subject: "db-integration-tests"}})
	require.NoError(t, err)
	// A second top-level group, so top-level listings have more than one row
	_, err = testClient.client.Groups.CreateGroup(ctx, &models.Group{Name: "other-top", CreatedBy: models.Actor{Subject: "db-integration-tests"}})
	require.NoError(t, err)

	list := func(input *GetNamespacesInput) []string {
		got, err := testClient.client.Namespaces.GetNamespaces(ctx, input)
		require.NoError(t, err)
		return namespacePaths(got.Namespaces)
	}

	t.Run("parent group lists only its direct children, sorted by path", func(t *testing.T) {
		assert.Equal(t, []string{"list-parent/child-group", "list-parent/child-workspace"}, list(&GetNamespacesInput{
			Filter: &NamespaceFilter{ParentPath: &parent.FullPath},
		}))
	})

	t.Run("FULL_PATH_DESC reverses the order", func(t *testing.T) {
		sort := NamespaceSortableFieldFullPathDesc
		assert.Equal(t, []string{"list-parent/child-workspace", "list-parent/child-group"}, list(&GetNamespacesInput{
			Sort:   &sort,
			Filter: &NamespaceFilter{ParentPath: &parent.FullPath},
		}))
	})

	t.Run("top level without memberships lists root groups only", func(t *testing.T) {
		assert.Equal(t, []string{"list-parent", "other-top"}, list(&GetNamespacesInput{
			Filter: &NamespaceFilter{TopLevel: true},
		}))
	})

	t.Run("memberships restrict to the root namespaces and their descendants", func(t *testing.T) {
		memberships := []models.MembershipNamespace{{ID: child.Metadata.ID, GroupID: &child.Metadata.ID, Path: child.FullPath}}

		assert.Equal(t, []string{"list-parent/child-group", "list-parent/child-group/grandchild-group"}, list(&GetNamespacesInput{
			Filter: &NamespaceFilter{RootNamespaceMemberships: memberships},
		}))
		// Top level with memberships lists the membership roots themselves, even when nested
		assert.Equal(t, []string{"list-parent/child-group"}, list(&GetNamespacesInput{
			Filter: &NamespaceFilter{RootNamespaceMemberships: memberships, TopLevel: true},
		}))
	})

	t.Run("empty memberships match nothing", func(t *testing.T) {
		empty := []models.MembershipNamespace{}

		assert.Empty(t, list(&GetNamespacesInput{Filter: &NamespaceFilter{RootNamespaceMemberships: empty}}))
		assert.Empty(t, list(&GetNamespacesInput{Filter: &NamespaceFilter{RootNamespaceMemberships: empty, TopLevel: true}}))
	})

	t.Run("pagination", func(t *testing.T) {
		filter := &NamespaceFilter{ParentPath: &parent.FullPath, IncludeNested: true}
		all := []string{
			"list-parent/child-group",
			"list-parent/child-group/grandchild-group",
			"list-parent/child-workspace",
		}

		page := func(opts *pagination.Options) *NamespacesResult {
			got, err := testClient.client.Namespaces.GetNamespaces(ctx, &GetNamespacesInput{Filter: filter, PaginationOptions: opts})
			require.NoError(t, err)
			return got
		}
		cursor := func(result *NamespacesResult, i int) *string {
			c, err := result.PageInfo.Cursor(namespaceModelForTest(&result.Namespaces[i]))
			require.NoError(t, err)
			return c
		}

		first := page(&pagination.Options{First: ptr.Int32(2)})
		assert.Equal(t, all[:2], namespacePaths(first.Namespaces))
		assert.True(t, first.PageInfo.HasNextPage)
		totalCount, err := first.PageInfo.TotalCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, int32(3), totalCount)

		next := page(&pagination.Options{First: ptr.Int32(2), After: cursor(first, 1)})
		assert.Equal(t, all[2:], namespacePaths(next.Namespaces))
		assert.False(t, next.PageInfo.HasNextPage)
		assert.True(t, next.PageInfo.HasPreviousPage)

		last := page(&pagination.Options{Last: ptr.Int32(2)})
		assert.Equal(t, all[1:], namespacePaths(last.Namespaces))
		assert.True(t, last.PageInfo.HasPreviousPage)

		prev := page(&pagination.Options{Last: ptr.Int32(2), Before: cursor(last, 0)})
		assert.Equal(t, all[:1], namespacePaths(prev.Namespaces))
		assert.False(t, prev.PageInfo.HasPreviousPage)
	})

	t.Run("search matches anywhere in the path below the ancestor", func(t *testing.T) {
		search := func(term string) []string {
			return list(&GetNamespacesInput{Filter: &NamespaceFilter{ParentPath: &parent.FullPath, IncludeNested: true, Search: &term}})
		}

		// The grandchild matches through its parent's name, not its own
		assert.Equal(t, []string{
			"list-parent/child-group",
			"list-parent/child-group/grandchild-group",
			"list-parent/child-workspace",
		}, search("child"))
		// The ancestor's own path segments never match
		assert.Empty(t, search("parent"))
	})

	t.Run("search of direct children matches the child's own name only", func(t *testing.T) {
		search := func(term string) []string {
			return list(&GetNamespacesInput{Filter: &NamespaceFilter{ParentPath: &parent.FullPath, Search: &term}})
		}

		assert.Equal(t, []string{"list-parent/child-group"}, search("group"))
		assert.Equal(t, []string{"list-parent/child-workspace"}, search("WORKSPACE"))
		// The parent's own path segments don't match every child
		assert.Empty(t, search("parent"))
	})

	t.Run("unknown parent path lists nothing", func(t *testing.T) {
		assert.Empty(t, list(&GetNamespacesInput{
			Filter: &NamespaceFilter{ParentPath: ptr.String("no-such-group")},
		}))
	})
}

// namespacePaths returns the full path of each namespace, in order.
func namespacePaths(namespaces []Namespace) []string {
	paths := []string{}
	for i := range namespaces {
		if namespaces[i].Group != nil {
			paths = append(paths, namespaces[i].Group.FullPath)
		} else {
			paths = append(paths, namespaces[i].Workspace.FullPath)
		}
	}
	return paths
}

// namespaceModelForTest returns whichever of the namespace's group or workspace is set, for building cursors.
func namespaceModelForTest(ns *Namespace) models.Model {
	if ns.Group != nil {
		return ns.Group
	}
	return ns.Workspace
}

// compareNamespaceRows compares two namespace row objects.
// Because there's no way to find the expected ID, it cannot be checked.
func compareNamespaceRows(t *testing.T, expected, actual *namespaceRow) {
	t.Helper()
	assert.Equal(t, expected.path, actual.path)
	assert.Equal(t, expected.groupID, actual.groupID)
	assert.Equal(t, expected.workspaceID, actual.workspaceID)
	assert.Equal(t, expected.version, actual.version)
}
