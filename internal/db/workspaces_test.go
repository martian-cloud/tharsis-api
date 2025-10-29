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
)

// getValue implements the sortableField interface for WorkspaceSortableField
func (w WorkspaceSortableField) getValue() string {
	return string(w)
}

func TestWorkspaces_CreateWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspace",
		Description: "test group for workspace",
		FullPath:    "test-group-workspace",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceName   string
		groupID         string
		description     string
	}

	testCases := []testCase{
		{
			name:          "create workspace",
			workspaceName: "test-workspace",
			groupID:       group.Metadata.ID,
			description:   "test workspace description",
		},
		{
			name:            "create workspace with invalid group ID",
			workspaceName:   "invalid-workspace",
			groupID:         invalidID,
			description:     "invalid workspace",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
				Name:           test.workspaceName,
				GroupID:        test.groupID,
				Description:    test.description,
				CreatedBy:      "db-integration-tests",
				MaxJobDuration: ptr.Int32(1),
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, workspace)

			assert.Equal(t, test.workspaceName, workspace.Name)
			assert.Equal(t, test.groupID, workspace.GroupID)
			assert.Equal(t, test.description, workspace.Description)
			assert.NotEmpty(t, workspace.Metadata.ID)
		})
	}
}

func TestWorkspaces_UpdateWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspace-update",
		Description: "test group for workspace update",
		FullPath:    "test-group-workspace-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a workspace for testing
	createdWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-update",
		GroupID:        group.Metadata.ID,
		Description:    "original description",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		workspaceID     string
		description     string
	}

	testCases := []testCase{
		{
			name:        "update workspace",
			workspaceID: createdWorkspace.Metadata.ID,
			version:     createdWorkspace.Metadata.Version,
			description: "updated description",
		},
		{
			name:            "update will fail because resource version doesn't match",
			workspaceID:     createdWorkspace.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			description:     "should not update",
		},
		{
			name:            "defective-id",
			workspaceID:     invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
			description:     "should not update",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspaceToUpdate := *createdWorkspace
			workspaceToUpdate.Metadata.ID = test.workspaceID
			workspaceToUpdate.Metadata.Version = test.version
			workspaceToUpdate.Description = test.description

			updatedWorkspace, err := testClient.client.Workspaces.UpdateWorkspace(ctx, &workspaceToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedWorkspace)

			assert.Equal(t, test.description, updatedWorkspace.Description)
			assert.Equal(t, createdWorkspace.Metadata.Version+1, updatedWorkspace.Metadata.Version)
		})
	}
}

func TestWorkspaces_DeleteWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspace-delete",
		Description: "test group for workspace delete",
		FullPath:    "test-group-workspace-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a workspace for testing
	createdWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-delete",
		GroupID:        group.Metadata.ID,
		Description:    "workspace to delete",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
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
			name:    "delete workspace",
			id:      createdWorkspace.Metadata.ID,
			version: createdWorkspace.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdWorkspace.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "defective-id",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
			version:         1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Workspaces.DeleteWorkspace(ctx, &models.Workspace{
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

			// Verify workspace was deleted
			workspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, test.id)
			assert.Nil(t, workspace)
			assert.Nil(t, err)
		})
	}
}

func TestWorkspaces_GetWorkspaceByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspace-get-by-id",
		Description: "test group for workspace get by id",
		FullPath:    "test-group-workspace-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	createdWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectWorkspace bool
	}

	testCases := []testCase{
		{
			name:            "get resource by id",
			id:              createdWorkspace.Metadata.ID,
			expectWorkspace: true,
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
			workspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectWorkspace {
				require.NotNil(t, workspace)
				assert.Equal(t, test.id, workspace.Metadata.ID)
			} else {
				assert.Nil(t, workspace)
			}
		})
	}
}

func TestWorkspaces_GetWorkspaces(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspaces
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspaces-list",
		Description: "test group for workspaces list",
		FullPath:    "test-group-workspaces-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test workspaces
	workspaces := []models.Workspace{
		{
			Name:           "test-workspace-list-1",
			GroupID:        group.Metadata.ID,
			MaxJobDuration: ptr.Int32(1),
			CreatedBy:      "db-integration-tests",
		},
		{
			Name:           "test-workspace-list-2",
			GroupID:        group.Metadata.ID,
			MaxJobDuration: ptr.Int32(1),
			CreatedBy:      "db-integration-tests",
		},
	}

	createdWorkspaces := []models.Workspace{}
	for _, workspace := range workspaces {
		created, err := testClient.client.Workspaces.CreateWorkspace(ctx, &workspace)
		require.NoError(t, err)
		createdWorkspaces = append(createdWorkspaces, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetWorkspacesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all workspaces",
			input:       &GetWorkspacesInput{},
			expectCount: len(createdWorkspaces),
		},
		{
			name: "filter by group ID",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					GroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdWorkspaces),
		},
		{
			name: "filter by search",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					Search: ptr.String("test-workspace-list-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by workspace IDs",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					WorkspaceIDs: []string{createdWorkspaces[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by workspace path",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					WorkspacePath: &createdWorkspaces[0].FullPath,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by locked",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					Locked: ptr.Bool(false),
				},
			},
			expectCount: len(createdWorkspaces),
		},
		{
			name: "filter by dirty",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					Dirty: ptr.Bool(false),
				},
			},
			expectCount: len(createdWorkspaces),
		},
		{
			name: "filter by has state version",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					HasStateVersion: ptr.Bool(false),
				},
			},
			expectCount: len(createdWorkspaces),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Workspaces.GetWorkspaces(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Workspaces, test.expectCount)
		})
	}
}

func TestWorkspaces_GetWorkspacesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspaces
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspaces-pagination",
		Description: "test group for workspaces pagination",
		FullPath:    "test-group-workspaces-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
			Name:           fmt.Sprintf("test-workspace-pagination-%d", i),
			GroupID:        group.Metadata.ID,
			MaxJobDuration: ptr.Int32(1),
			CreatedBy:      "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		WorkspaceSortableFieldFullPathAsc,
		WorkspaceSortableFieldFullPathDesc,
		WorkspaceSortableFieldUpdatedAtAsc,
		WorkspaceSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := WorkspaceSortableField(sortByField.getValue())

		result, err := testClient.client.Workspaces.GetWorkspaces(ctx, &GetWorkspacesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Workspaces {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestWorkspaces_GetWorkspaceByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspace-trn",
		Description: "test group for workspace trn",
		FullPath:    "test-group-workspace-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	createdWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-trn",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectWorkspace bool
	}

	testCases := []testCase{
		{
			name:            "get resource by TRN",
			trn:             createdWorkspace.Metadata.TRN,
			expectWorkspace: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:workspace:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspace, err := testClient.client.Workspaces.GetWorkspaceByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectWorkspace {
				require.NotNil(t, workspace)
				assert.Equal(t, test.trn, workspace.Metadata.TRN)
			} else {
				assert.Nil(t, workspace)
			}
		})
	}
}
