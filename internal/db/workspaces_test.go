//go:build integration

package db

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
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
		{
			name:            "negative, group does not exist",
			workspaceName:   "missing-group-workspace",
			groupID:         nonExistentID,
			expectErrorCode: errors.EConflict,
		},
		{
			// Runs after "create workspace", so its namespace path is already taken
			name:            "negative, duplicate name",
			workspaceName:   "test-workspace",
			groupID:         group.Metadata.ID,
			expectErrorCode: errors.EConflict,
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

func TestWorkspaces_UpdateWorkspaceWithLabels(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspace-update-labels",
		Description: "test group for workspace update with labels",
		FullPath:    "test-group-workspace-update-labels",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create a workspace with initial labels for testing
	createdWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-update-labels",
		GroupID:        group.Metadata.ID,
		Description:    "workspace for label update testing",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
		Labels: map[string]string{
			"environment": "development",
			"team":        "backend",
		},
	})
	require.Nil(t, err)

	type testCase struct {
		name           string
		labels         map[string]string
		expectedLabels map[string]string
	}

	testCases := []testCase{
		{
			name: "update workspace with new labels",
			labels: map[string]string{
				"environment": "production",
				"team":        "platform",
				"project":     "infrastructure",
			},
			expectedLabels: map[string]string{
				"environment": "production",
				"team":        "platform",
				"project":     "infrastructure",
			},
		},
		{
			name:           "update workspace with empty labels",
			labels:         map[string]string{},
			expectedLabels: map[string]string{},
		},
		{
			name:           "update workspace with nil labels",
			labels:         nil,
			expectedLabels: map[string]string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspaceToUpdate := *createdWorkspace
			workspaceToUpdate.Labels = test.labels

			updatedWorkspace, err := testClient.client.Workspaces.UpdateWorkspace(ctx, &workspaceToUpdate)

			require.Nil(t, err)
			require.NotNil(t, updatedWorkspace)

			if len(test.expectedLabels) == 0 {
				assert.Empty(t, updatedWorkspace.Labels)
			} else {
				assert.Equal(t, test.expectedLabels, updatedWorkspace.Labels)
			}

			// Verify labels persist when retrieving workspace
			retrievedWorkspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, updatedWorkspace.Metadata.ID)
			require.Nil(t, err)
			require.NotNil(t, retrievedWorkspace)

			if len(test.expectedLabels) == 0 {
				assert.Empty(t, retrievedWorkspace.Labels)
			} else {
				assert.Equal(t, test.expectedLabels, retrievedWorkspace.Labels)
			}

			// Update createdWorkspace for next iteration
			createdWorkspace = updatedWorkspace
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

func TestWorkspaces_GetWorkspacesMinDurationSinceLastAssessment(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:     "test-group-min-duration",
		FullPath: "test-group-min-duration",
	})
	require.NoError(t, err)

	createWorkspace := func(name string) *models.Workspace {
		ws, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
			Name:           name,
			GroupID:        group.Metadata.ID,
			MaxJobDuration: ptr.Int32(1),
		})
		require.NoError(t, err)
		return ws
	}

	now := time.Now().UTC()

	// Never assessed: matched via the "no assessment" branch.
	createWorkspace("ws-no-assessment")

	// Fresh in-progress assessment: should NOT be matched (not stale, not completed).
	freshWorkspace := createWorkspace("ws-fresh-in-progress")
	_, err = testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:        freshWorkspace.Metadata.ID,
		StartedAtTimestamp: now,
	})
	require.NoError(t, err)

	// Stale in-progress assessment: matched via the stale branch once updated_at is
	// backdated beyond the stale timeout.
	staleWorkspace := createWorkspace("ws-stale-in-progress")
	staleAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:        staleWorkspace.Metadata.ID,
		StartedAtTimestamp: now,
	})
	require.NoError(t, err)
	_, err = testClient.client.getConnection(ctx).Exec(ctx,
		"UPDATE workspace_assessments SET updated_at = $1 WHERE id = $2",
		now.Add(-2*models.AssessmentStaleTimeout), staleAssessment.Metadata.ID)
	require.NoError(t, err)

	minDuration := time.Hour
	result, err := testClient.client.Workspaces.GetWorkspaces(ctx, &GetWorkspacesInput{
		Filter: &WorkspaceFilter{MinDurationSinceLastAssessment: &minDuration},
	})
	require.NoError(t, err)

	matched := map[string]bool{}
	for _, ws := range result.Workspaces {
		matched[ws.Name] = true
	}
	assert.True(t, matched["ws-no-assessment"], "workspace with no assessment should be eligible")
	assert.True(t, matched["ws-stale-in-progress"], "workspace with a stale in-progress assessment should be eligible")
	assert.False(t, matched["ws-fresh-in-progress"], "workspace with a fresh in-progress assessment should not be eligible")
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

	// second group + workspace for NamespacePathPrefix tests
	groupB, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-workspaces-list-b",
		Description: "second group for workspace list tests",
		FullPath:    "test-group-workspaces-list-b",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	wsInGroupB, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-list-b",
		GroupID:        groupB.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)
	_ = wsInGroupB

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-workspaces",
		Email:    "test-user-workspaces@test.com",
	})
	require.NoError(t, err)

	_, err = testClient.client.NamespaceFavorites.CreateNamespaceFavorite(ctx, &models.NamespaceFavorite{
		UserID:      user.Metadata.ID,
		WorkspaceID: &createdWorkspaces[0].Metadata.ID,
	})
	require.NoError(t, err)

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
			expectCount: len(createdWorkspaces) + 1, // +1 for wsInGroupB
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
			expectCount: len(createdWorkspaces) + 1, // +1 for wsInGroupB
		},
		{
			name: "filter by dirty",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					Dirty: ptr.Bool(false),
				},
			},
			expectCount: len(createdWorkspaces) + 1, // +1 for wsInGroupB
		},
		{
			name: "filter by has state version",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					HasStateVersion: ptr.Bool(false),
				},
			},
			expectCount: len(createdWorkspaces) + 1, // +1 for wsInGroupB
		},
		{
			name: "filter by favorite user",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					FavoriteUserID: &user.Metadata.ID,
				},
			},
			expectCount: 1,
		},
		{
			name: "exclude favorited workspaces",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					ExcludeFavoriteUserID: &user.Metadata.ID,
				},
			},
			expectCount: 2, // createdWorkspaces[1] + wsInGroupB
		},
		{
			name: "filter by namespace path prefix — matches all workspaces under group",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					NamespacePathPrefix: &group.FullPath,
				},
			},
			expectCount: len(createdWorkspaces),
		},
		{
			name: "filter by namespace path prefix — single workspace in other group",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					NamespacePathPrefix: &groupB.FullPath,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by namespace path prefix — no match",
			input: &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					NamespacePathPrefix: ptr.String("nonexistent-prefix"),
				},
			},
			expectCount: 0,
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

func TestWorkspaces_GetWorkspacesWithLabelFiltering(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-label-filtering",
		Description: "test group for label filtering",
		FullPath:    "test-group-label-filtering",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	// Create workspaces with different labels
	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "workspace-prod",
		GroupID:        group.Metadata.ID,
		Description:    "production workspace",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
		Labels: map[string]string{
			"environment": "production",
			"team":        "platform",
		},
	})
	require.Nil(t, err)

	workspace2, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "workspace-dev",
		GroupID:        group.Metadata.ID,
		Description:    "development workspace",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
		Labels: map[string]string{
			"environment": "development",
			"team":        "platform",
		},
	})
	require.Nil(t, err)

	workspace3, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "workspace-no-labels",
		GroupID:        group.Metadata.ID,
		Description:    "workspace without labels",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	type testCase struct {
		name          string
		labelFilters  []WorkspaceLabelFilter
		expectedCount int
		expectedIDs   []string
	}

	testCases := []testCase{
		{
			name: "filter by environment=production",
			labelFilters: []WorkspaceLabelFilter{
				{Key: "environment", Value: "production"},
			},
			expectedCount: 1,
			expectedIDs:   []string{workspace1.Metadata.ID},
		},
		{
			name: "filter by team=platform",
			labelFilters: []WorkspaceLabelFilter{
				{Key: "team", Value: "platform"},
			},
			expectedCount: 2,
			expectedIDs:   []string{workspace1.Metadata.ID, workspace2.Metadata.ID},
		},
		{
			name: "filter by multiple labels (AND logic)",
			labelFilters: []WorkspaceLabelFilter{
				{Key: "environment", Value: "development"},
				{Key: "team", Value: "platform"},
			},
			expectedCount: 1,
			expectedIDs:   []string{workspace2.Metadata.ID},
		},
		{
			name: "filter by non-existent label",
			labelFilters: []WorkspaceLabelFilter{
				{Key: "nonexistent", Value: "value"},
			},
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Workspaces.GetWorkspaces(ctx, &GetWorkspacesInput{
				Filter: &WorkspaceFilter{
					GroupID:      &group.Metadata.ID,
					LabelFilters: test.labelFilters,
				},
			})

			require.Nil(t, err)
			require.NotNil(t, result)

			assert.Equal(t, test.expectedCount, len(result.Workspaces))

			if test.expectedCount > 0 {
				actualIDs := make([]string, len(result.Workspaces))
				for i, ws := range result.Workspaces {
					actualIDs[i] = ws.Metadata.ID
				}
				assert.ElementsMatch(t, test.expectedIDs, actualIDs)
			}
		})
	}

	// Clean up test workspaces
	_ = testClient.client.Workspaces.DeleteWorkspace(ctx, workspace1)
	_ = testClient.client.Workspaces.DeleteWorkspace(ctx, workspace2)
	_ = testClient.client.Workspaces.DeleteWorkspace(ctx, workspace3)
}

func TestWorkspaces_CreateWorkspaceWithLabels(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-create-labels",
		Description: "test group for creating workspace with labels",
		FullPath:    "test-group-create-labels",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name           string
		workspaceName  string
		labels         map[string]string
		expectedLabels map[string]string
	}

	testCases := []testCase{
		{
			name:          "create workspace with labels",
			workspaceName: "workspace-with-labels",
			labels: map[string]string{
				"environment": "production",
				"team":        "platform",
				"project":     "infrastructure",
			},
			expectedLabels: map[string]string{
				"environment": "production",
				"team":        "platform",
				"project":     "infrastructure",
			},
		},
		{
			name:           "create workspace with empty labels",
			workspaceName:  "workspace-empty-labels",
			labels:         map[string]string{},
			expectedLabels: map[string]string{},
		},
		{
			name:           "create workspace with nil labels",
			workspaceName:  "workspace-nil-labels",
			labels:         nil,
			expectedLabels: map[string]string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
				Name:           test.workspaceName,
				GroupID:        group.Metadata.ID,
				Description:    "test workspace with labels",
				CreatedBy:      "db-integration-tests",
				MaxJobDuration: ptr.Int32(1),
				Labels:         test.labels,
			})

			require.Nil(t, err)
			require.NotNil(t, workspace)

			assert.Equal(t, test.workspaceName, workspace.Name)

			if len(test.expectedLabels) == 0 {
				assert.Empty(t, workspace.Labels)
			} else {
				assert.Equal(t, test.expectedLabels, workspace.Labels)
			}

			// Verify labels persist when retrieving workspace
			retrievedWorkspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, workspace.Metadata.ID)
			require.Nil(t, err)
			require.NotNil(t, retrievedWorkspace)

			if len(test.expectedLabels) == 0 {
				assert.Empty(t, retrievedWorkspace.Labels)
			} else {
				assert.Equal(t, test.expectedLabels, retrievedWorkspace.Labels)
			}
		})
	}
}

// TestWorkspaces_MigrateWorkspace moves workspaces to a new group and checks the re-pathing and the
// cleanup of the assignments the move puts out of scope: managed identities, service account
// namespace memberships and VCS provider links. Each is kept only if the principal's group is an
// ancestor of where the workspace landed.
func TestWorkspaces_MigrateWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// origin                      old grandparent
	// `-- team                    old parent
	//     |-- ws-a                migrated to dest/inner
	//     |-- ws-b                migrated to dest/inner
	//     |-- w_1                 migrated to dest/inner; '_' is a LIKE wildcard
	//     |-- other-ws            does not move
	//     `-- wx1                 what w_1 matches under an unescaped LIKE
	//         `-- origin
	//             `-- team
	//                 `-- w_1-x   path contains "origin/team/w_1" after the start
	// dest                        new grandparent
	// `-- inner                   new parent
	groups := map[string]*models.Group{}
	for _, fullPath := range []string{
		"origin",
		"origin/team",
		"origin/team/wx1",
		"origin/team/wx1/origin",
		"origin/team/wx1/origin/team",
		"origin/team/wx1/origin/team/w_1-x",
		"dest",
		"dest/inner",
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
		"origin/team/ws-a",
		"origin/team/ws-b",
		"origin/team/w_1",
		"origin/team/other-ws",
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

	// assignmentCase describes one (workspace, principal) pair the cleanup has to judge. principalPath
	// is the group the identity, service account or VCS provider lives in, which is all the cleanup
	// knows about it.
	type assignmentCase struct {
		name          string
		workspacePath string
		principalPath string
		expectKept    bool
	}

	managedIdentityCases := []assignmentCase{
		{
			name:          "managed identity in the new parent group stays assigned",
			workspacePath: "origin/team/ws-a",
			principalPath: "dest/inner",
			expectKept:    true,
		},
		{
			name:          "managed identity in the new grandparent group stays assigned",
			workspacePath: "origin/team/ws-a",
			principalPath: "dest",
			expectKept:    true,
		},
		{
			name:          "managed identity in the old parent group is unassigned",
			workspacePath: "origin/team/ws-a",
			principalPath: "origin/team",
		},
		{
			name:          "managed identity in the old grandparent group is unassigned",
			workspacePath: "origin/team/ws-a",
			principalPath: "origin",
		},
		{
			name:          "managed identity assigned to a workspace that doesn't move is untouched",
			workspacePath: "origin/team/other-ws",
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
			identity.Metadata.ID, workspaces[test.workspacePath].Metadata.ID))

		managedIdentityIDs[i] = identity.Metadata.ID
	}

	membershipCases := []assignmentCase{
		{
			name:          "service account membership from the new grandparent group is kept",
			workspacePath: "origin/team/ws-a",
			principalPath: "dest",
			expectKept:    true,
		},
		{
			name:          "service account membership from the old parent group is removed",
			workspacePath: "origin/team/ws-a",
			principalPath: "origin/team",
		},
		{
			name:          "service account membership of a workspace that doesn't move is untouched",
			workspacePath: "origin/team/other-ws",
			principalPath: "origin/team",
			expectKept:    true,
		},
	}

	role, err := testClient.client.Roles.CreateRole(ctx, &models.Role{
		Name:      "test-role",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	membershipIDs := make([]string, len(membershipCases))
	for i, test := range membershipCases {
		serviceAccount, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
			Name:      fmt.Sprintf("test-sa-%d", i),
			GroupID:   groups[test.principalPath].Metadata.ID,
			CreatedBy: "db-integration-tests",
		})
		require.NoError(t, err)

		membership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
			NamespacePath:    test.workspacePath,
			ServiceAccountID: &serviceAccount.Metadata.ID,
			RoleID:           role.Metadata.ID,
		})
		require.NoError(t, err)

		membershipIDs[i] = membership.Metadata.ID
	}

	// A user membership has no group to be out of scope of, so the move must leave it alone.
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-migrate-workspace",
		Email:    "test-user-migrate-workspace@test.com",
	})
	require.NoError(t, err)

	userMembership, err := testClient.client.NamespaceMemberships.CreateNamespaceMembership(ctx, &CreateNamespaceMembershipInput{
		NamespacePath: "origin/team/ws-a",
		UserID:        &user.Metadata.ID,
		RoleID:        role.Metadata.ID,
	})
	require.NoError(t, err)

	// A workspace holds at most one VCS provider link, so each case needs its own workspace.
	linkCases := []assignmentCase{
		{
			name:          "VCS provider link from the new grandparent group is kept",
			workspacePath: "origin/team/ws-a",
			principalPath: "dest",
			expectKept:    true,
		},
		{
			name:          "VCS provider link from the old parent group is removed",
			workspacePath: "origin/team/ws-b",
			principalPath: "origin/team",
		},
		{
			name:          "VCS provider link of a workspace that doesn't move is untouched",
			workspacePath: "origin/team/other-ws",
			principalPath: "origin/team",
			expectKept:    true,
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
			WorkspaceID:    workspaces[test.workspacePath].Metadata.ID,
			ProviderID:     provider.Metadata.ID,
			TokenNonce:     uuid.New().String(),
			RepositoryPath: "test-org/test-repo",
			Branch:         "main",
			CreatedBy:      "db-integration-tests",
		})
		require.NoError(t, err)

		linkIDs[i] = link.Metadata.ID
	}

	t.Run("a stale version fails without moving the workspace", func(t *testing.T) {
		stale := *workspaces["origin/team/ws-a"]
		stale.Metadata.Version = -1

		_, err := testClient.client.Workspaces.MigrateWorkspace(ctx, &stale, groups["dest/inner"])
		assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))

		// The namespace is re-pathed in the same transaction, so it must have been rolled back.
		workspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, stale.Metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, "origin/team/ws-a", workspace.FullPath)
		assert.Equal(t, groups["origin/team"].Metadata.ID, workspace.GroupID)
	})

	for _, path := range []string{"origin/team/ws-a", "origin/team/ws-b", "origin/team/w_1"} {
		original := workspaces[path]
		migrated, err := testClient.client.Workspaces.MigrateWorkspace(ctx, original, groups["dest/inner"])
		require.NoError(t, err)
		require.NotNil(t, migrated)

		assert.Equal(t, "dest/inner/"+original.Name, migrated.FullPath)
		assert.Equal(t, groups["dest/inner"].Metadata.ID, migrated.GroupID)
		assert.Equal(t, original.Metadata.Version+1, migrated.Metadata.Version)

		// Re-query to confirm the new path was persisted, not just returned.
		fetched, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, original.Metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, "dest/inner/"+original.Name, fetched.FullPath)
	}

	t.Run("migrating w_1 leaves namespaces it matches only as a LIKE pattern untouched", func(t *testing.T) {
		// An unescaped "origin/team/w_1/%" matches this group's path, and an unanchored regex replace
		// of "origin/team/w_1" would then rewrite its later occurrence in it.
		group, err := testClient.client.Groups.GetGroupByID(ctx, groups["origin/team/wx1/origin/team/w_1-x"].Metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, "origin/team/wx1/origin/team/w_1-x", group.FullPath)
	})

	t.Run("a workspace that doesn't move keeps its path", func(t *testing.T) {
		workspace, err := testClient.client.Workspaces.GetWorkspaceByID(ctx, workspaces["origin/team/other-ws"].Metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, "origin/team/other-ws", workspace.FullPath)
	})

	for i, test := range managedIdentityCases {
		t.Run(test.name, func(t *testing.T) {
			identities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx,
				workspaces[test.workspacePath].Metadata.ID)
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

	for i, test := range membershipCases {
		t.Run(test.name, func(t *testing.T) {
			membership, err := testClient.client.NamespaceMemberships.GetNamespaceMembershipByID(ctx, membershipIDs[i])
			require.NoError(t, err)

			assert.Equal(t, test.expectKept, membership != nil)
		})
	}

	t.Run("user membership of the migrated workspace is untouched", func(t *testing.T) {
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
