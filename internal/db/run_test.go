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
		withApply       bool
	}

	testCases := []testCase{
		{
			name:        "create run with plan and apply nodes",
			workspaceID: workspace.Metadata.ID,
			status:      models.RunPending,
			withApply:   true,
		},
		{
			name:        "create speculative run with only a plan node",
			workspaceID: workspace.Metadata.ID,
			status:      models.RunPending,
			withApply:   false,
		},
		{
			name:            "create run with invalid workspace ID",
			workspaceID:     invalidID,
			status:          models.RunPending,
			expectErrorCode: errors.EInternal,
		},
	}

	// Distinct, non-zero summary values so a mismatched column mapping would be caught.
	planSummary := models.PlanSummary{
		ResourceAdditions:    1,
		ResourceChanges:      2,
		ResourceDestructions: 3,
		ResourceImports:      4,
		ResourceDrift:        5,
		OutputAdditions:      6,
		OutputChanges:        7,
		OutputDestructions:   8,
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			input := &models.Run{
				WorkspaceID: test.workspaceID,
				Status:      test.status,
				CreatedBy:   "db-integration-tests",
				Plan: models.Plan{
					Status:     models.PlanRunning,
					HasChanges: true,
					DiffSize:   42,
					Summary:    planSummary,
				},
			}
			if test.withApply {
				input.Apply = &models.Apply{
					Status:      models.ApplyCreated,
					TriggeredBy: "db-integration-tests",
					Comment:     "apply node comment",
				}
			}

			run, err := testClient.client.Runs.CreateRun(ctx, input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, run)

			assert.Equal(t, test.workspaceID, run.WorkspaceID)
			assert.Equal(t, test.status, run.Status)
			assert.NotEmpty(t, run.Metadata.ID)

			// The plan node is always stored and hydrated back on the returned run.
			assert.NotEmpty(t, run.Plan.ID)
			assert.Equal(t, models.PlanRunning, run.Plan.Status)
			assert.True(t, run.Plan.HasChanges)
			assert.Equal(t, 42, run.Plan.DiffSize)
			assert.Equal(t, planSummary, run.Plan.Summary)

			if test.withApply {
				require.NotNil(t, run.Apply)
				assert.NotEmpty(t, run.Apply.ID)
				assert.Equal(t, models.ApplyCreated, run.Apply.Status)
				assert.Equal(t, "db-integration-tests", run.Apply.TriggeredBy)
				assert.Equal(t, "apply node comment", run.Apply.Comment)
				assert.False(t, run.Speculative())
			} else {
				assert.Nil(t, run.Apply)
				assert.True(t, run.Speculative())
			}
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
		Plan:        models.Plan{Status: models.PlanQueued},
		Apply:       &models.Apply{Status: models.ApplyCreated, TriggeredBy: "db-integration-tests"},
	})
	require.Nil(t, err)
	require.NotEmpty(t, createdRun.Plan.ID)
	require.NotNil(t, createdRun.Apply)

	t.Run("update fails when resource version doesn't match", func(t *testing.T) {
		stale := createdRun.Copy()
		stale.Metadata.Version = -1
		stale.Status = models.RunErrored

		_, err := testClient.client.Runs.UpdateRun(ctx, stale)
		assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
	})

	t.Run("updates run-level and node fields (all nodes)", func(t *testing.T) {
		upd := createdRun.Copy()
		upd.Status = models.RunPlanning
		upd.Plan.Status = models.PlanFinished
		upd.Plan.HasChanges = true
		upd.Plan.DiffSize = 7
		upd.Plan.Summary.ResourceAdditions = 9
		upd.Apply.Status = models.ApplyQueued
		upd.Apply.Comment = "updated comment"

		updatedRun, err := testClient.client.Runs.UpdateRun(ctx, upd)
		require.Nil(t, err)
		require.NotNil(t, updatedRun)

		assert.Equal(t, models.RunPlanning, updatedRun.Status)
		assert.Equal(t, createdRun.Metadata.Version+1, updatedRun.Metadata.Version)
		assert.Equal(t, models.PlanFinished, updatedRun.Plan.Status)
		assert.True(t, updatedRun.Plan.HasChanges)
		assert.Equal(t, 7, updatedRun.Plan.DiffSize)
		assert.Equal(t, int32(9), updatedRun.Plan.Summary.ResourceAdditions)
		assert.Equal(t, models.ApplyQueued, updatedRun.Apply.Status)
		assert.Equal(t, "updated comment", updatedRun.Apply.Comment)

		// Re-query to confirm the node changes were persisted, not just returned.
		fetched, err := testClient.client.Runs.GetRunByID(ctx, createdRun.Metadata.ID)
		require.Nil(t, err)
		require.NotNil(t, fetched)
		assert.Equal(t, models.PlanFinished, fetched.Plan.Status)
		assert.Equal(t, int32(9), fetched.Plan.Summary.ResourceAdditions)
		require.NotNil(t, fetched.Apply)
		assert.Equal(t, models.ApplyQueued, fetched.Apply.Status)
		assert.Equal(t, "updated comment", fetched.Apply.Comment)
	})

	t.Run("selective update only touches the named node", func(t *testing.T) {
		current, err := testClient.client.Runs.GetRunByID(ctx, createdRun.Metadata.ID)
		require.Nil(t, err)

		upd := current.Copy()
		upd.Plan.Status = models.PlanErrored
		upd.Apply.Status = models.ApplyErrored // changed in memory but NOT in the nodeIDs filter

		// Only the plan node ID is passed, so only the plan row should be written.
		updatedRun, err := testClient.client.Runs.UpdateRun(ctx, upd, upd.Plan.ID)
		require.Nil(t, err)
		require.NotNil(t, updatedRun)

		fetched, err := testClient.client.Runs.GetRunByID(ctx, createdRun.Metadata.ID)
		require.Nil(t, err)
		assert.Equal(t, models.PlanErrored, fetched.Plan.Status)
		require.NotNil(t, fetched.Apply)
		assert.Equal(t, models.ApplyQueued, fetched.Apply.Status, "apply node must be unchanged when not in the nodeIDs filter")
	})
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

	// Create a run with both nodes so the query path's node hydration is exercised.
	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
		Plan:        models.Plan{Status: models.PlanRunning, HasChanges: true, DiffSize: 3},
		Apply:       &models.Apply{Status: models.ApplyCreated, Comment: "c"},
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
				// Nodes are hydrated on the queried run.
				assert.Equal(t, createdRun.Plan.ID, run.Plan.ID)
				assert.Equal(t, models.PlanRunning, run.Plan.Status)
				assert.True(t, run.Plan.HasChanges)
				assert.Equal(t, 3, run.Plan.DiffSize)
				require.NotNil(t, run.Apply)
				assert.Equal(t, createdRun.Apply.ID, run.Apply.ID)
				assert.Equal(t, models.ApplyCreated, run.Apply.Status)
				assert.Equal(t, "c", run.Apply.Comment)
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
			resources = append(resources, resource)
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

func TestRuns_GetRunByNodeID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-get-by-node-id",
		Description: "test group for run get by node id",
		FullPath:    "test-group-run-get-by-node-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-get-by-node-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a run with stages for testing
	planNodeID := newResourceID()
	applyNodeID := newResourceID()
	createdRun, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
		Plan:        models.Plan{ID: planNodeID, Status: models.PlanQueued},
		Apply:       &models.Apply{ID: applyNodeID, Status: models.ApplyCreated},
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		nodeID          string
		expectRun       bool
	}

	testCases := []testCase{
		{
			name:      "get run by plan node id",
			nodeID:    planNodeID,
			expectRun: true,
		},
		{
			name:      "get run by apply node id",
			nodeID:    applyNodeID,
			expectRun: true,
		},
		{
			name:   "run with node id not found",
			nodeID: nonExistentID,
		},
		{
			name:            "get run with invalid node id will return an error",
			nodeID:          invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			run, err := testClient.client.Runs.GetRunByNodeID(ctx, test.nodeID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRun {
				require.NotNil(t, run)
				assert.Equal(t, createdRun.Metadata.ID, run.Metadata.ID)
				// Looking up by either node ID returns the run with both nodes hydrated.
				assert.Equal(t, planNodeID, run.Plan.ID)
				assert.Equal(t, models.PlanQueued, run.Plan.Status)
				require.NotNil(t, run.Apply)
				assert.Equal(t, applyNodeID, run.Apply.ID)
				assert.Equal(t, models.ApplyCreated, run.Apply.Status)
			} else {
				assert.Nil(t, run)
			}
		})
	}
}
