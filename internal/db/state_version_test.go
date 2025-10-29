//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for StateVersionSortableField
func (sv StateVersionSortableField) getValue() string {
	return string(sv)
}

func TestStateVersions_CreateStateVersion(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group, workspace, and run for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-version",
		Description: "test group for state version",
		FullPath:    "test-group-state-version",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-version",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for state version",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		runID           string
		workspaceID     string
	}

	testCases := []testCase{
		{
			name:        "create state version",
			runID:       run.Metadata.ID,
			workspaceID: workspace.Metadata.ID,
		},
		{
			name:            "create state version with invalid workspace ID",
			runID:           run.Metadata.ID,
			workspaceID:     invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			stateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
				RunID:       &test.runID,
				WorkspaceID: test.workspaceID,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, stateVersion)

			assert.Equal(t, test.runID, *stateVersion.RunID)
			assert.Equal(t, test.workspaceID, stateVersion.WorkspaceID)
			assert.NotEmpty(t, stateVersion.Metadata.ID)
		})
	}
}

func TestStateVersions_GetStateVersionByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-version-get-by-id",
		Description: "test group for state version get by id",
		FullPath:    "test-group-state-version-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-version-get-by-id",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a state version for testing
	createdStateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode    errors.CodeType
		name               string
		id                 string
		expectStateVersion bool
	}

	testCases := []testCase{
		{
			name:               "get resource by id",
			id:                 createdStateVersion.Metadata.ID,
			expectStateVersion: true,
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
			stateVersion, err := testClient.client.StateVersions.GetStateVersionByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectStateVersion {
				require.NotNil(t, stateVersion)
				assert.Equal(t, test.id, stateVersion.Metadata.ID)
			} else {
				assert.Nil(t, stateVersion)
			}
		})
	}
}

func TestStateVersions_GetStateVersionByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-version-get-by-trn",
		Description: "test group for state version get by trn",
		FullPath:    "test-group-state-version-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-version-get-by-trn",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a state version for testing
	createdStateVersion, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode    errors.CodeType
		name               string
		trn                string
		expectStateVersion bool
	}

	testCases := []testCase{
		{
			name:               "get resource by TRN",
			trn:                createdStateVersion.Metadata.TRN,
			expectStateVersion: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:state_version:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			stateVersion, err := testClient.client.StateVersions.GetStateVersionByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectStateVersion {
				require.NotNil(t, stateVersion)
				assert.Equal(t, createdStateVersion.Metadata.ID, stateVersion.Metadata.ID)
			} else {
				assert.Nil(t, stateVersion)
			}
		})
	}
}

func TestStateVersions_GetStateVersions(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-versions-list",
		Description: "test group for state versions list",
		FullPath:    "test-group-state-versions-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a workspace for testing
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-versions-list",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create test state versions
	stateVersions := []models.StateVersion{
		{
			WorkspaceID: workspace.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
		{
			WorkspaceID: workspace.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdStateVersions := []models.StateVersion{}
	for _, stateVersion := range stateVersions {
		created, err := testClient.client.StateVersions.CreateStateVersion(ctx, &stateVersion)
		require.NoError(t, err)
		createdStateVersions = append(createdStateVersions, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetStateVersionsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name: "get all state versions for workspace",
			input: &GetStateVersionsInput{
				Filter: &StateVersionFilter{
					WorkspaceID: &workspace.Metadata.ID,
				},
			},
			expectCount: len(createdStateVersions),
		},
		{
			name: "filter by state version IDs",
			input: &GetStateVersionsInput{
				Filter: &StateVersionFilter{
					StateVersionIDs: []string{createdStateVersions[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by time range start",
			input: &GetStateVersionsInput{
				Filter: &StateVersionFilter{
					WorkspaceID:    &workspace.Metadata.ID,
					TimeRangeStart: createdStateVersions[0].Metadata.CreationTimestamp,
				},
			},
			expectCount: len(createdStateVersions),
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.StateVersions.GetStateVersions(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.StateVersions, test.expectCount)
		})
	}
}

func TestStateVersions_GetStateVersionsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group, workspace, and run for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-state-versions-pagination",
		Description: "test group for state versions pagination",
		FullPath:    "test-group-state-versions-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-state-versions-pagination",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for state versions pagination",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
			RunID:       &run.Metadata.ID,
			WorkspaceID: workspace.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		StateVersionSortableFieldUpdatedAtAsc,
		StateVersionSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := StateVersionSortableField(sortByField.getValue())

		result, err := testClient.client.StateVersions.GetStateVersions(ctx, &GetStateVersionsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
			Filter: &StateVersionFilter{
				WorkspaceID: &workspace.Metadata.ID,
			},
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.StateVersions {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
