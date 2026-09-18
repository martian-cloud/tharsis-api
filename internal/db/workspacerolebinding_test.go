//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// getValue implements the sortableField interface for WorkspaceRoleBindingSortableField
func (sf WorkspaceRoleBindingSortableField) getValue() string {
	return string(sf)
}

func createTestGroupForBinding(ctx context.Context, t *testing.T, testClient *testClient, name string) *models.Group {
	t.Helper()

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        name,
		Description: "test group for workspace role binding",
		FullPath:    name,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)
	return group
}

func createTestWorkspaceForBinding(ctx context.Context, t *testing.T, testClient *testClient, name, groupID string) *models.Workspace {
	t.Helper()

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           name,
		GroupID:        groupID,
		Description:    "test workspace for role binding",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)
	return workspace
}

func createTestRoleForBinding(ctx context.Context, t *testing.T, testClient *testClient, name string, perms []models.Permission) *models.Role {
	t.Helper()

	role := &models.Role{
		Name:        name,
		Description: "test role for workspace role binding",
		CreatedBy:   "db-integration-tests",
	}
	role.SetPermissions(perms)

	created, err := testClient.client.Roles.CreateRole(ctx, role)
	require.NoError(t, err)
	return created
}

func TestWorkspaceRoleBindings_CreateWorkspaceRoleBinding(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-create-group")
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-create-role", []models.Permission{models.CreateWorkspacePermission})

	boundWorkspace := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-create-bound-ws", group.Metadata.ID)
	_, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: boundWorkspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceID     string
		roleID          string
	}

	testCases := []testCase{
		{
			name:        "create workspace role binding",
			workspaceID: createTestWorkspaceForBinding(ctx, t, testClient, "wrb-create-ws", group.Metadata.ID).Metadata.ID,
			roleID:      role.Metadata.ID,
		},
		{
			name:            "negative, workspace does not exist",
			workspaceID:     nonExistentID,
			roleID:          role.Metadata.ID,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "negative, role does not exist",
			workspaceID:     createTestWorkspaceForBinding(ctx, t, testClient, "wrb-create-ws-no-role", group.Metadata.ID).Metadata.ID,
			roleID:          nonExistentID,
			expectErrorCode: errors.ENotFound,
		},
		{
			// One binding per workspace: a binding always applies at the parent namespace, so a
			// second row would have no distinct meaning.
			name:            "negative, workspace already has a binding",
			workspaceID:     boundWorkspace.Metadata.ID,
			roleID:          role.Metadata.ID,
			expectErrorCode: errors.EConflict,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			binding, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
				WorkspaceID: test.workspaceID,
				RoleID:      test.roleID,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, binding)

			assert.Equal(t, test.workspaceID, binding.WorkspaceID)
			assert.Equal(t, test.roleID, binding.RoleID)
			assert.Equal(t, "db-integration-tests", binding.CreatedBy)
			assert.NotEmpty(t, binding.Metadata.ID)
		})
	}
}

func TestWorkspaceRoleBindings_GetWorkspaceRoleBindingByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-get-id-group")
	workspace := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-get-id-ws", group.Metadata.ID)
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-get-id-role", []models.Permission{models.ViewWorkspacePermission})

	created, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: workspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		expectFound     bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "get resource by id",
			id:          created.Metadata.ID,
			expectFound: true,
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
			binding, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindingByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectFound {
				require.NotNil(t, binding)
				assert.Equal(t, created.Metadata.ID, binding.Metadata.ID)
			} else {
				assert.Nil(t, binding)
			}
		})
	}
}

func TestWorkspaceRoleBindings_GetWorkspaceRoleBindingByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-get-trn-group")
	workspace := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-get-trn-ws", group.Metadata.ID)
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-get-trn-role", []models.Permission{models.ViewWorkspacePermission})

	created, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: workspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// The TRN is derived from the workspace's namespace path, since a workspace has at most one binding.
	assert.Equal(t, trn.TypeWorkspaceRoleBinding.Build(workspace.FullPath), created.Metadata.TRN)

	type testCase struct {
		name            string
		trn             string
		expectFound     bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:        "get resource by TRN",
			trn:         created.Metadata.TRN,
			expectFound: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeWorkspaceRoleBinding.Build("non-existent-workspace-path"),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			binding, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindingByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectFound {
				require.NotNil(t, binding)
				assert.Equal(t, created.Metadata.ID, binding.Metadata.ID)
			} else {
				assert.Nil(t, binding)
			}
		})
	}
}

func TestWorkspaceRoleBindings_GetWorkspaceRoleBindingByWorkspaceID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-get-ws-group")
	workspace := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-get-ws-ws", group.Metadata.ID)
	unbound := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-get-ws-unbound", group.Metadata.ID)
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-get-ws-role", []models.Permission{models.ViewWorkspacePermission})

	created, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: workspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name        string
		workspaceID string
		expectFound bool
	}

	testCases := []testCase{
		{
			name:        "get by workspace ID",
			workspaceID: workspace.Metadata.ID,
			expectFound: true,
		},
		{
			name:        "get by workspace ID returns nil for an unbound workspace",
			workspaceID: unbound.Metadata.ID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			binding, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindingByWorkspaceID(ctx, test.workspaceID)
			require.NoError(t, err)

			if test.expectFound {
				require.NotNil(t, binding)
				assert.Equal(t, created.Metadata.ID, binding.Metadata.ID)
			} else {
				assert.Nil(t, binding)
			}
		})
	}
}

func TestWorkspaceRoleBindings_UpdateWorkspaceRoleBinding(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-update-group")
	workspace := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-update-ws", group.Metadata.ID)
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-update-role", []models.Permission{models.ViewWorkspacePermission})
	newRole := createTestRoleForBinding(ctx, t, testClient, "wrb-update-role-new", []models.Permission{models.CreateWorkspacePermission})

	created, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: workspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		roleID          string
	}

	testCases := []testCase{
		{
			name:    "update the role",
			version: created.Metadata.Version,
			roleID:  newRole.Metadata.ID,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			roleID:          newRole.Metadata.ID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			toUpdate := *created
			toUpdate.Metadata.Version = test.version
			toUpdate.RoleID = test.roleID

			updated, err := testClient.client.WorkspaceRoleBindings.UpdateWorkspaceRoleBinding(ctx, &toUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, updated)

			assert.Equal(t, test.roleID, updated.RoleID)
			assert.Equal(t, created.Metadata.Version+1, updated.Metadata.Version)
		})
	}
}

func TestWorkspaceRoleBindings_DeleteWorkspaceRoleBinding(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-delete-group")
	workspace := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-delete-ws", group.Metadata.ID)
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-delete-role", []models.Permission{models.ViewWorkspacePermission})

	created, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: workspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:    "delete binding",
			id:      created.Metadata.ID,
			version: created.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              created.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.WorkspaceRoleBindings.DeleteWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			binding, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindingByID(ctx, test.id)
			require.NoError(t, err)
			assert.Nil(t, binding)
		})
	}
}

func TestWorkspaceRoleBindings_ForeignKeys(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-fk-group")
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-fk-role", []models.Permission{models.ViewWorkspacePermission})

	// Deleting the workspace must take its binding with it, so no binding is left conferring
	// authority for a workspace that no longer exists.
	t.Run("deleting the workspace cascades to the binding", func(t *testing.T) {
		ws := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-cascade-ws", group.Metadata.ID)

		binding, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
			WorkspaceID: ws.Metadata.ID,
			RoleID:      role.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)

		require.NoError(t, testClient.client.Workspaces.DeleteWorkspace(ctx, ws))

		got, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindingByID(ctx, binding.Metadata.ID)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	// RESTRICT rather than CASCADE: silently dropping a binding when its role is deleted would
	// quietly de-privilege a workspace. The role deletion must fail so the dependency is visible.
	t.Run("deleting a bound role is refused", func(t *testing.T) {
		boundRole := createTestRoleForBinding(ctx, t, testClient, "wrb-bound-role", []models.Permission{models.ViewWorkspacePermission})
		ws := createTestWorkspaceForBinding(ctx, t, testClient, "wrb-bound-role-ws", group.Metadata.ID)

		_, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
			WorkspaceID: ws.Metadata.ID,
			RoleID:      boundRole.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)

		err = testClient.client.Roles.DeleteRole(ctx, boundRole)
		assert.Error(t, err, "expected deleting a bound role to be refused")
	})
}

func TestWorkspaceRoleBindings_GetWorkspaceRoleBindings(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	parent := createTestGroupForBinding(ctx, t, testClient, "wrb-list-parent")
	other := createTestGroupForBinding(ctx, t, testClient, "wrb-list-other")

	role := createTestRoleForBinding(ctx, t, testClient, "wrb-list-role", []models.Permission{models.ViewWorkspacePermission})
	otherRole := createTestRoleForBinding(ctx, t, testClient, "wrb-list-role-other", []models.Permission{models.CreateWorkspacePermission})

	inParent := createTestWorkspaceForBinding(ctx, t, testClient, "ws-in-parent", parent.Metadata.ID)
	inOther := createTestWorkspaceForBinding(ctx, t, testClient, "ws-in-other", other.Metadata.ID)

	var bindingInParent *models.WorkspaceRoleBinding
	for ws, r := range map[string]string{
		inParent.Metadata.ID: role.Metadata.ID,
		inOther.Metadata.ID:  otherRole.Metadata.ID,
	} {
		created, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
			WorkspaceID: ws,
			RoleID:      r,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)

		if ws == inParent.Metadata.ID {
			bindingInParent = created
		}
	}

	type testCase struct {
		name        string
		filter      *WorkspaceRoleBindingFilter
		expectCount int
		expectIDs   []string
	}

	testCases := []testCase{
		{
			name:        "no filter returns all bindings",
			filter:      &WorkspaceRoleBindingFilter{},
			expectCount: 2,
		},
		{
			name:        "filter by IDs",
			filter:      &WorkspaceRoleBindingFilter{IDs: []string{bindingInParent.Metadata.ID}},
			expectCount: 1,
			expectIDs:   []string{bindingInParent.Metadata.ID},
		},
		{
			name:        "filter by workspace IDs",
			filter:      &WorkspaceRoleBindingFilter{WorkspaceIDs: []string{inParent.Metadata.ID}},
			expectCount: 1,
		},
		{
			name:        "filter by role ID",
			filter:      &WorkspaceRoleBindingFilter{RoleID: &otherRole.Metadata.ID},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindings(ctx, &GetWorkspaceRoleBindingsInput{
				Filter: test.filter,
			})
			require.NoError(t, err)
			require.Len(t, result.WorkspaceRoleBindings, test.expectCount)

			if test.expectIDs != nil {
				gotIDs := make([]string, len(result.WorkspaceRoleBindings))
				for i, b := range result.WorkspaceRoleBindings {
					gotIDs[i] = b.Metadata.ID
				}
				assert.ElementsMatch(t, test.expectIDs, gotIDs)
			}
		})
	}
}

func TestWorkspaceRoleBindings_GetWorkspaceRoleBindingsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForBinding(ctx, t, testClient, "wrb-paginate-group")
	role := createTestRoleForBinding(ctx, t, testClient, "wrb-paginate-role", []models.Permission{models.ViewWorkspacePermission})

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		ws := createTestWorkspaceForBinding(ctx, t, testClient, fmt.Sprintf("wrb-paginate-ws-%d", i), group.Metadata.ID)

		_, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
			WorkspaceID: ws.Metadata.ID,
			RoleID:      role.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		WorkspaceRoleBindingSortableFieldCreatedAtAsc,
		WorkspaceRoleBindingSortableFieldCreatedAtDesc,
		WorkspaceRoleBindingSortableFieldUpdatedAtAsc,
		WorkspaceRoleBindingSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := WorkspaceRoleBindingSortableField(sortByField.getValue())

		result, err := testClient.client.WorkspaceRoleBindings.GetWorkspaceRoleBindings(ctx, &GetWorkspaceRoleBindingsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.WorkspaceRoleBindings {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
