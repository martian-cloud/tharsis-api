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

// getValue implements the sortableField interface for PlanSortableField
func (p PlanSortableField) getValue() string {
	return string(p)
}

func TestPlans_CreatePlan(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-plan",
		Description: "test group for plan",
		FullPath:    "test-group-plan",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-plan",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for plan",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceID     string
		status          models.PlanStatus
	}

	testCases := []testCase{
		{
			name:        "create plan",
			workspaceID: workspace.Metadata.ID,
			status:      models.PlanPending,
		},
		{
			name:            "create plan with invalid workspace ID",
			workspaceID:     invalidID,
			status:          models.PlanPending,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			plan, err := testClient.client.Plans.CreatePlan(ctx, &models.Plan{
				WorkspaceID: test.workspaceID,
				Status:      test.status,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, plan)

			assert.Equal(t, test.workspaceID, plan.WorkspaceID)
			assert.Equal(t, test.status, plan.Status)
			assert.NotEmpty(t, plan.Metadata.ID)
		})
	}
}

func TestPlans_UpdatePlan(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-plan-update",
		Description: "test group for plan update",
		FullPath:    "test-group-plan-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-plan-update",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for plan update",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	createdPlan, err := testClient.client.Plans.CreatePlan(ctx, &models.Plan{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.PlanPending,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.PlanStatus
	}

	testCases := []testCase{
		{
			name:    "update plan",
			version: createdPlan.Metadata.Version,
			status:  models.PlanFinished,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.PlanErrored,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			planToUpdate := *createdPlan
			planToUpdate.Metadata.Version = test.version
			planToUpdate.Status = test.status

			updatedPlan, err := testClient.client.Plans.UpdatePlan(ctx, &planToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedPlan)

			assert.Equal(t, test.status, updatedPlan.Status)
			assert.Equal(t, createdPlan.Metadata.Version+1, updatedPlan.Metadata.Version)
		})
	}
}

func TestPlans_GetPlanByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-plan-get-by-id",
		Description: "test group for plan get by id",
		FullPath:    "test-group-plan-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-plan-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a plan for testing
	createdPlan, err := testClient.client.Plans.CreatePlan(ctx, &models.Plan{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.PlanPending,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectPlan      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by id",
			id:         createdPlan.Metadata.ID,
			expectPlan: true,
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
			plan, err := testClient.client.Plans.GetPlanByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPlan {
				require.NotNil(t, plan)
				assert.Equal(t, test.id, plan.Metadata.ID)
			} else {
				assert.Nil(t, plan)
			}
		})
	}
}

func TestPlans_GetPlans(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-plans-list",
		Description: "test group for plans list",
		FullPath:    "test-group-plans-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-plans-list",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test plans
	plans := []models.Plan{
		{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.PlanPending,
		},
		{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.PlanRunning,
		},
	}

	createdPlans := []models.Plan{}
	for _, plan := range plans {
		created, err := testClient.client.Plans.CreatePlan(ctx, &plan)
		require.NoError(t, err)
		createdPlans = append(createdPlans, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetPlansInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all plans",
			input:       &GetPlansInput{},
			expectCount: len(createdPlans),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Plans.GetPlans(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Plans, test.expectCount)
		})
	}
}

func TestPlans_GetPlansWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-plans-pagination",
		Description: "test group for plans pagination",
		FullPath:    "test-group-plans-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-plans-pagination",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Plans.CreatePlan(ctx, &models.Plan{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.PlanPending,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		PlanSortableFieldUpdatedAtAsc,
		PlanSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := PlanSortableField(sortByField.getValue())

		result, err := testClient.client.Plans.GetPlans(ctx, &GetPlansInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Plans {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestPlans_GetPlanByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-plan-get-by-trn",
		Description: "test group for plan get by trn",
		FullPath:    "test-group-plan-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-plan-get-by-trn",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a plan for testing
	createdPlan, err := testClient.client.Plans.CreatePlan(ctx, &models.Plan{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.PlanPending,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectPlan      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by TRN",
			trn:        createdPlan.Metadata.TRN,
			expectPlan: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:plan:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			plan, err := testClient.client.Plans.GetPlanByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPlan {
				require.NotNil(t, plan)
				assert.Equal(t, createdPlan.Metadata.ID, plan.Metadata.ID)
			} else {
				assert.Nil(t, plan)
			}
		})
	}
}
