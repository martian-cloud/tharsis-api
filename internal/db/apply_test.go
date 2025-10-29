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

// getValue implements the sortableField interface for ApplySortableField
func (a ApplySortableField) getValue() string {
	return string(a)
}

func TestApplies_CreateApply(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-apply",
		Description: "test group for apply",
		FullPath:    "test-group-apply",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-apply",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for apply",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceID     string
		status          models.ApplyStatus
	}

	testCases := []testCase{
		{
			name:        "create apply",
			workspaceID: workspace.Metadata.ID,
			status:      models.ApplyPending,
		},
		{
			name:            "create apply with invalid workspace ID",
			workspaceID:     invalidID,
			status:          models.ApplyPending,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			apply, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
				WorkspaceID: test.workspaceID,
				Status:      test.status,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, apply)

			assert.Equal(t, test.workspaceID, apply.WorkspaceID)
			assert.Equal(t, test.status, apply.Status)
			assert.NotEmpty(t, apply.Metadata.ID)
		})
	}
}

func TestApplies_UpdateApply(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-apply-update",
		Description: "test group for apply update",
		FullPath:    "test-group-apply-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-apply-update",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for apply update",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	createdApply, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.ApplyPending,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.ApplyStatus
	}

	testCases := []testCase{
		{
			name:    "update apply",
			version: createdApply.Metadata.Version,
			status:  models.ApplyFinished,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.ApplyErrored,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			applyToUpdate := *createdApply
			applyToUpdate.Metadata.Version = test.version
			applyToUpdate.Status = test.status

			updatedApply, err := testClient.client.Applies.UpdateApply(ctx, &applyToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedApply)

			assert.Equal(t, test.status, updatedApply.Status)
			assert.Equal(t, createdApply.Metadata.Version+1, updatedApply.Metadata.Version)
		})
	}
}

func TestApplies_GetApplyByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-apply-get-by-id",
		Description: "test group for apply get by id",
		FullPath:    "test-group-apply-get-by-id",
	})
	require.NoError(t, err)

	// Create a workspace for the apply
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-apply-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create an apply for testing
	createdApply, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
		Status:      models.ApplyPending,
		WorkspaceID: workspace.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
	}

	testCases := []testCase{
		{
			name: "get resource by id",
			id:   createdApply.Metadata.ID,
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
			apply, err := testClient.client.Applies.GetApplyByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.id == createdApply.Metadata.ID {
				require.NotNil(t, apply)
				assert.Equal(t, test.id, apply.Metadata.ID)
			} else {
				assert.Nil(t, apply)
			}
		})
	}
}

func TestApplies_GetApplies(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-applies-list",
		Description: "test group for applies list",
		FullPath:    "test-group-applies-list",
	})
	require.NoError(t, err)

	// Create a workspace for the applies
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-applies-list",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create test applies
	applies := []models.Apply{
		{
			Status:      models.ApplyPending,
			WorkspaceID: workspace.Metadata.ID,
		},
		{
			Status:      models.ApplyRunning,
			WorkspaceID: workspace.Metadata.ID,
		},
	}

	createdApplies := []models.Apply{}
	for _, apply := range applies {
		created, err := testClient.client.Applies.CreateApply(ctx, &apply)
		require.NoError(t, err)
		createdApplies = append(createdApplies, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetAppliesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all applies",
			input:       &GetAppliesInput{},
			expectCount: len(createdApplies),
		},
		{
			name: "filter by apply IDs",
			input: &GetAppliesInput{
				Filter: &ApplyFilter{
					ApplyIDs: []string{createdApplies[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Applies.GetApplies(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Applies, test.expectCount)
		})
	}
}

func TestApplies_GetAppliesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-applies-pagination",
		Description: "test group for applies pagination",
		FullPath:    "test-group-applies-pagination",
	})
	require.NoError(t, err)

	// Create a workspace for the applies
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-applies-pagination",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
			Status:      models.ApplyPending,
			WorkspaceID: workspace.Metadata.ID,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		ApplySortableFieldUpdatedAtAsc,
		ApplySortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := ApplySortableField(sortByField.getValue())

		result, err := testClient.client.Applies.GetApplies(ctx, &GetAppliesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Applies {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestApplies_GetApplyByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the workspace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-apply-trn",
		Description: "test group for apply trn",
		FullPath:    "test-group-apply-trn",
	})
	require.NoError(t, err)

	// Create a workspace for the apply
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-apply-trn",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create an apply for testing
	createdApply, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
		Status:      models.ApplyPending,
		WorkspaceID: workspace.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
	}

	testCases := []testCase{
		{
			name: "get resource by TRN",
			trn:  createdApply.Metadata.TRN,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:apply:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			apply, err := testClient.client.Applies.GetApplyByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.trn == createdApply.Metadata.TRN {
				require.NotNil(t, apply)
				assert.Equal(t, test.trn, apply.Metadata.TRN)
			} else {
				assert.Nil(t, apply)
			}
		})
	}
}
