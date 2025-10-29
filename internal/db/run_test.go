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

// getValue implements the sortableField interface for RunSortableField
func (r RunSortableField) getValue() string {
	return string(r)
}

func TestRuns_CreateRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run",
		Description: "test group for run",
		FullPath:    "test-group-run",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for run",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceID     string
		status          models.RunStatus
	}

	testCases := []testCase{
		{
			name:        "create run",
			workspaceID: workspace.Metadata.ID,
			status:      models.RunPending,
		},
		{
			name:            "create run with invalid workspace ID",
			workspaceID:     invalidID,
			status:          models.RunPending,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
				WorkspaceID: test.workspaceID,
				Status:      test.status,
				CreatedBy:   "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, run)

			assert.Equal(t, test.workspaceID, run.WorkspaceID)
			assert.Equal(t, test.status, run.Status)
			assert.NotEmpty(t, run.Metadata.ID)
		})
	}
}

func TestRuns_UpdateRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group, workspace, and run for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-update",
		Description: "test group for run update",
		FullPath:    "test-group-run-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-update",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for run update",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.RunStatus
	}

	testCases := []testCase{
		{
			name:    "update run",
			version: createdRun.Metadata.Version,
			status:  models.RunApplied,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.RunErrored,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			runToUpdate := *createdRun
			runToUpdate.Metadata.Version = test.version
			runToUpdate.Status = test.status

			updatedRun, err := testClient.client.Runs.UpdateRun(ctx, &runToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedRun)

			assert.Equal(t, test.status, updatedRun.Status)
			assert.Equal(t, createdRun.Metadata.Version+1, updatedRun.Metadata.Version)
		})
	}
}

func TestRuns_GetRunByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-get-by-id",
		Description: "test group for run get by id",
		FullPath:    "test-group-run-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for testing
	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectRun       bool
	}

	testCases := []testCase{
		{
			name:      "get resource by id",
			id:        createdRun.Metadata.ID,
			expectRun: true,
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
			run, err := testClient.client.Runs.GetRunByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, test.id, run.Metadata.ID)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestRuns_GetRuns(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runs-list",
		Description: "test group for runs list",
		FullPath:    "test-group-runs-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-runs-list",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test runs
	runs := []models.Run{
		{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.RunPending,
			CreatedBy:   "db-integration-tests",
		},
		{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.RunPlanning,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdRuns := []models.Run{}
	for _, run := range runs {
		created, err := testClient.client.Runs.CreateRun(ctx, &run)
		require.NoError(t, err)
		createdRuns = append(createdRuns, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetRunsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all runs",
			input:       &GetRunsInput{},
			expectCount: len(createdRuns),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Runs.GetRuns(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Runs, test.expectCount)
		})
	}
}

func TestRuns_GetRunsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runs-pagination",
		Description: "test group for runs pagination",
		FullPath:    "test-group-runs-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-runs-pagination",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.RunPending,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		RunSortableFieldCreatedAtAsc,
		RunSortableFieldCreatedAtDesc,
		RunSortableFieldUpdatedAtAsc,
		RunSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := RunSortableField(sortByField.getValue())

		result, err := testClient.client.Runs.GetRuns(ctx, &GetRunsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Runs {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestRuns_GetRunByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-get-by-trn",
		Description: "test group for run get by trn",
		FullPath:    "test-group-run-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-get-by-trn",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for testing
	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectRun       bool
	}

	testCases := []testCase{
		{
			name:      "get resource by TRN",
			trn:       createdRun.Metadata.TRN,
			expectRun: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:run:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.GetRunByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, createdRun.Metadata.ID, run.Metadata.ID)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestRuns_GetRunByPlanID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-get-by-plan-id",
		Description: "test group for run get by plan id",
		FullPath:    "test-group-run-get-by-plan-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-get-by-plan-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for testing
	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a plan for testing
	createdPlan, err := testClient.client.Plans.CreatePlan(ctx, &models.Plan{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.PlanPending,
	})
	require.NoError(t, err)

	// Update the run with the plan ID
	createdRun.PlanID = createdPlan.Metadata.ID
	_, err = testClient.client.Runs.UpdateRun(ctx, createdRun)
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		planID          string
		expectRun       bool
	}

	testCases := []testCase{
		{
			name:      "get resource by plan id",
			planID:    createdPlan.Metadata.ID,
			expectRun: true,
		},
		{
			name:   "resource with plan id not found",
			planID: nonExistentID,
		},
		{
			name:            "get resource with invalid plan id will return an error",
			planID:          invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.GetRunByPlanID(ctx, test.planID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, createdRun.Metadata.ID, run.Metadata.ID)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}

func TestRuns_GetRunByApplyID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-get-by-apply-id",
		Description: "test group for run get by apply id",
		FullPath:    "test-group-run-get-by-apply-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-get-by-apply-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run for testing
	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create an apply for testing
	createdApply, err := testClient.client.Applies.CreateApply(ctx, &models.Apply{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.ApplyPending,
	})
	require.NoError(t, err)

	// Update the run with the apply ID
	createdRun.ApplyID = createdApply.Metadata.ID
	_, err = testClient.client.Runs.UpdateRun(ctx, createdRun)
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		applyID         string
		expectRun       bool
	}

	testCases := []testCase{
		{
			name:      "get resource by apply id",
			applyID:   createdApply.Metadata.ID,
			expectRun: true,
		},
		{
			name:    "resource with apply id not found",
			applyID: nonExistentID,
		},
		{
			name:            "get resource with invalid apply id will return an error",
			applyID:         invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.GetRunByApplyID(ctx, test.applyID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, createdRun.Metadata.ID, run.Metadata.ID)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}
