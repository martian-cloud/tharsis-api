//go:build integration

package db

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetNamespace(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-group",
		FullPath:  "test-get-namespace-group",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-get-namespace-workspace",
		FullPath:       "test-get-namespace-group/test-get-namespace-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
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
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	groupB, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-group-id-b",
		CreatedBy: "db-integration-tests",
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
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-get-namespace-by-workspace-id-ws",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
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
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	childGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-get-namespace-by-path-child",
		ParentID:  parentGroup.Metadata.ID,
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-get-namespace-by-path-ws",
		GroupID:        childGroup.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
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

func TestCreateNamespace(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// It is not feasible to make a direct positive test case here, because the group_id/workspace_id
	// foreign keys must point to an existing group or workspace, and creating a group or workspace
	// already creates its namespace row (see TestGetNamespace and friends for indirect coverage).
	// These fixtures exist only to give the duplicate-conflict cases a real group/workspace ID to
	// collide with.
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-create-namespace-group",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-create-namespace-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		input           *namespaceRow
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "negative, duplicate group",
			input: &namespaceRow{
				path:    "would/duplicate/a/group",
				groupID: group.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "negative, duplicate workspace",
			input: &namespaceRow{
				path:        "would/duplicate/a/workspace",
				workspaceID: workspace.Metadata.ID,
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "negative, non-existent group id",
			input: &namespaceRow{
				path:    "group/id/does/not/exist",
				groupID: nonExistentID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "negative, non-existent workspace id",
			input: &namespaceRow{
				path:        "workspace/id/does/not/exist",
				workspaceID: nonExistentID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "negative, invalid group id",
			input: &namespaceRow{
				path:    "group/id/invalid",
				groupID: invalidID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "negative, invalid workspace id",
			input: &namespaceRow{
				path:        "workspace/id/invalid",
				workspaceID: invalidID,
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gotNamespace, err := createNamespace(ctx, testClient.client.getConnection(ctx), test.input)
			assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			assert.Nil(t, gotNamespace)
		})
	}
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
