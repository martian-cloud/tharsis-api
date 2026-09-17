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

	// Link a state version to run[0] and set it as the workspace's current state version.
	sv, err := testClient.client.StateVersions.CreateStateVersion(ctx, &models.StateVersion{
		WorkspaceID: workspace.Metadata.ID,
		RunID:       &createdRuns[0].Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace.CurrentStateVersionID = sv.Metadata.ID
	_, err = testClient.client.Workspaces.UpdateWorkspace(ctx, workspace)
	require.NoError(t, err)

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

func TestRunGates_GetRunGates_Eligibility(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	approver, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "gate-eligibility-approver",
		Email:    "gate-eligibility-approver@example.com",
	})
	require.NoError(t, err)

	other, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "gate-eligibility-other",
		Email:    "gate-eligibility-other@example.com",
	})
	require.NoError(t, err)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-gate-eligibility",
		FullPath:  "test-group-gate-eligibility",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-gate-eligibility",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	createRun := func() *models.Run {
		run, rErr := testClient.client.Runs.CreateRun(ctx, &models.Run{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.RunPending,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, rErr)
		return run
	}

	// Each gate needs its own check ID (unique index index_run_gates_on_policy_check_id).
	createGate := func(runID, ruleName string, status models.RunGateStatus, allowedUserIDs []string) *models.RunGate {
		subjects := make([]*models.RunGateAllowedSubject, len(allowedUserIDs))
		for i, id := range allowedUserIDs {
			subjects[i] = &models.RunGateAllowedSubject{ID: id, Type: models.RunGateSubjectUser}
		}
		gate, gErr := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
			RunID:         runID,
			WorkspaceID:   workspace.Metadata.ID,
			PolicyCheckID: "check-" + ruleName,
			Type:          models.RunGateTypeOPAPolicy,
			Status:        status,
			ApprovalRules: []*models.RunGateApprovalRule{
				{Name: ruleName, RequiredApprovals: 1, AllowedSubjects: subjects},
			},
		})
		require.NoError(t, gErr)
		return gate
	}

	createTeamGate := func(runID, ruleName string, allowedTeamIDs []string) *models.RunGate {
		subjects := make([]*models.RunGateAllowedSubject, len(allowedTeamIDs))
		for i, id := range allowedTeamIDs {
			subjects[i] = &models.RunGateAllowedSubject{ID: id, Type: models.RunGateSubjectTeam}
		}
		gate, gErr := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
			RunID:         runID,
			WorkspaceID:   workspace.Metadata.ID,
			PolicyCheckID: "check-" + ruleName,
			Type:          models.RunGateTypeOPAPolicy,
			Status:        models.RunGatePending,
			ApprovalRules: []*models.RunGateApprovalRule{
				{Name: ruleName, RequiredApprovals: 1, AllowedSubjects: subjects},
			},
		})
		require.NoError(t, gErr)
		return gate
	}

	// gateA: pending and the approver is eligible -> included.
	runA := createRun()
	gateA := createGate(runA.Metadata.ID, "policy-a1", models.RunGatePending, []string{approver.Metadata.ID})

	// gateB: pending but already decided by the approver -> excluded.
	runB := createRun()
	decidedGate := createGate(runB.Metadata.ID, "policy-b1", models.RunGatePending, []string{approver.Metadata.ID})
	_, err = testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID:    decidedGate.Metadata.ID,
		UserID:       &approver.Metadata.ID,
		CreatedBy:    approver.Email,
		Decision:     models.RunGateDecisionReject,
		CoveredRules: []string{"policy-b1"},
	})
	require.NoError(t, err)

	// gateC: allowed only to a different user -> not eligible, excluded.
	runC := createRun()
	createGate(runC.Metadata.ID, "policy-c1", models.RunGatePending, []string{other.Metadata.ID})

	// gateD: eligible but not pending -> excluded by the status filter.
	runD := createRun()
	createGate(runD.Metadata.ID, "policy-d1", models.RunGateApproved, []string{approver.Metadata.ID})

	// Team eligibility is resolved from team_members rows (a subquery), not passed in the filter.
	memberTeam, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{Name: "gate-eligibility-member-team"})
	require.NoError(t, err)
	nonMemberTeam, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{Name: "gate-eligibility-nonmember-team"})
	require.NoError(t, err)
	_, err = testClient.client.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
		UserID: approver.Metadata.ID,
		TeamID: memberTeam.Metadata.ID,
	})
	require.NoError(t, err)

	// gateE: pending and allowed via a team the approver belongs to -> included via the subquery.
	runE := createRun()
	gateE := createTeamGate(runE.Metadata.ID, "policy-e1", []string{memberTeam.Metadata.ID})

	// gateF: pending but allowed only via a team the approver is not in -> excluded.
	runF := createRun()
	createTeamGate(runF.Metadata.ID, "policy-f1", []string{nonMemberTeam.Metadata.ID})

	result, err := testClient.client.RunGates.GetRunGates(ctx, &GetRunGatesInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(100)},
		Filter: &RunGateFilter{
			Statuses:    []models.RunGateStatus{models.RunGatePending},
			Eligibility: &RunGateEligibilityFilter{UserID: &approver.Metadata.ID},
		},
	})
	require.NoError(t, err)

	ids := []string{}
	for _, gate := range result.RunGates {
		ids = append(ids, gate.Metadata.ID)
	}
	assert.ElementsMatch(t, []string{gateA.Metadata.ID, gateE.Metadata.ID}, ids)
}

func TestRunGates_GetRunGates_RootNamespaceMembershipsFilter(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-gate-membership",
		FullPath:  "test-group-gate-membership",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-gate-membership",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	gate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:         run.Metadata.ID,
		WorkspaceID:   workspace.Metadata.ID,
		PolicyCheckID: "check-gate-membership",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{{Name: "policy-1", RequiredApprovals: 1}},
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		memberships []models.MembershipNamespace
		wantGate    bool
	}{
		{
			name:        "membership at the gate's root group includes it",
			memberships: []models.MembershipNamespace{{Path: "test-group-gate-membership"}},
			wantGate:    true,
		},
		{
			name:        "membership in an unrelated namespace excludes it",
			memberships: []models.MembershipNamespace{{Path: "some-other-group"}},
			wantGate:    false,
		},
		{
			name:        "empty memberships match nothing",
			memberships: []models.MembershipNamespace{},
			wantGate:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, rErr := testClient.client.RunGates.GetRunGates(ctx, &GetRunGatesInput{
				PaginationOptions: &pagination.Options{First: ptr.Int32(100)},
				Filter: &RunGateFilter{
					RootNamespaceMemberships: test.memberships,
				},
			})
			require.NoError(t, rErr)

			ids := []string{}
			for _, g := range result.RunGates {
				ids = append(ids, g.Metadata.ID)
			}
			if test.wantGate {
				assert.Equal(t, []string{gate.Metadata.ID}, ids)
			} else {
				assert.Empty(t, ids)
			}
		})
	}
}

func TestRuns_GetWorkspaceIDForRun(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-workspace-id",
		Description: "test group for run workspace id",
		FullPath:    "test-group-run-workspace-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-workspace-id",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

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
		expectWorkspace bool
	}

	testCases := []testCase{
		{
			name:            "get workspace id for run",
			id:              createdRun.Metadata.ID,
			expectWorkspace: true,
		},
		{
			name:            "run does not exist",
			id:              nonExistentID,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "get workspace id with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspaceID, err := testClient.client.Runs.GetWorkspaceIDForRun(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectWorkspace {
				assert.Equal(t, workspace.Metadata.ID, workspaceID)
			}
		})
	}
}

func TestRuns_DeleteRunBatch(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-delete-runs",
		FullPath:  "test-group-delete-runs",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-delete-runs",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	makeRun := func(t *testing.T) *models.Run {
		t.Helper()
		run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.RunApplied,
			CreatedBy:   "db-integration-tests",
			Plan:        models.Plan{ID: newResourceID(), Status: models.PlanQueued},
		})
		require.NoError(t, err)
		return run
	}

	// Each test case owns its own runs so cases do not interfere.
	runNoOp := makeRun(t)
	runIgnore := makeRun(t)
	runPartial := makeRun(t)
	runSurvivor := makeRun(t)
	runMixed := makeRun(t)
	runMulti1 := makeRun(t)
	runMulti2 := makeRun(t)
	runStale := makeRun(t)

	type testCase struct {
		name         string
		input        DeleteRunBatchInput
		shouldBeGone []string
		shouldExist  []string
		expectOLE    bool
	}

	testCases := []testCase{
		{
			name:        "empty slice is a no-op",
			input:       DeleteRunBatchInput{Runs: []*models.Run{}},
			shouldExist: []string{runNoOp.Metadata.ID},
		},
		{
			name: "non-existent run: OLE, nothing deleted",
			input: DeleteRunBatchInput{Runs: []*models.Run{
				{Metadata: models.ResourceMetadata{ID: nonExistentID, Version: 1}},
			}},
			shouldExist: []string{runIgnore.Metadata.ID},
			expectOLE:   true,
		},
		{
			name:         "partial delete — only the specified run is removed",
			input:        DeleteRunBatchInput{Runs: []*models.Run{runPartial}},
			shouldBeGone: []string{runPartial.Metadata.ID},
			shouldExist:  []string{runSurvivor.Metadata.ID},
		},
		{
			name: "batch with non-existent run alongside valid run: OLE, valid one still deleted",
			input: DeleteRunBatchInput{Runs: []*models.Run{
				runMixed,
				{Metadata: models.ResourceMetadata{ID: nonExistentID, Version: 1}},
			}},
			shouldBeGone: []string{runMixed.Metadata.ID},
			expectOLE:    true,
		},
		{
			name:         "delete multiple runs in one call",
			input:        DeleteRunBatchInput{Runs: []*models.Run{runMulti1, runMulti2}},
			shouldBeGone: []string{runMulti1.Metadata.ID, runMulti2.Metadata.ID},
		},
		{
			name: "stale version: OLE, nothing deleted",
			input: DeleteRunBatchInput{Runs: []*models.Run{
				{Metadata: models.ResourceMetadata{ID: runStale.Metadata.ID, Version: runStale.Metadata.Version - 1}},
			}},
			shouldExist: []string{runStale.Metadata.ID},
			expectOLE:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			deletedIDs, err := testClient.client.Runs.DeleteRunBatch(ctx, &test.input)

			if test.expectOLE {
				assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
			} else {
				require.NoError(t, err)
			}
			assert.ElementsMatch(t, test.shouldBeGone, deletedIDs)

			for _, id := range test.shouldBeGone {
				got, err := testClient.client.Runs.GetRunByID(ctx, id)
				require.NoError(t, err)
				assert.Nil(t, got)
			}

			for _, id := range test.shouldExist {
				got, err := testClient.client.Runs.GetRunByID(ctx, id)
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

// TestRuns_CreateRun_Annotations verifies that run annotations round-trip through the database:
// they are persisted on create and read back by GetRunByID exactly as stored, including the optional
// link and duplicate keys. This exercises the JSONB marshal (CreateRun) and scan (scanRun) paths,
// which the model-level unit tests do not cover.
func TestRuns_CreateRun_Annotations(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-annotations",
		Description: "test group for run annotations",
		FullPath:    "test-group-run-annotations",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-annotations",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for run annotations",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	link := "https://gitlab.example.com/x/-/commit/a1b2c3d4"
	annotations := []*models.RunAnnotation{
		{Key: "commit", Value: "a1b2c3d4", Link: &link}, // with link
		{Key: "ref", Value: "main"},                     // without link
		{Key: "commit", Value: "e5f6g7h8"},              // duplicate key, preserved in order
	}

	created, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
		Plan:        models.Plan{Status: models.PlanCreated},
		Annotations: annotations,
	})
	require.Nil(t, err)
	require.NotNil(t, created)

	// Annotations are returned on the created run, in order, with the link preserved.
	require.Len(t, created.Annotations, 3)
	assert.Equal(t, "commit", created.Annotations[0].Key)
	assert.Equal(t, "a1b2c3d4", created.Annotations[0].Value)
	require.NotNil(t, created.Annotations[0].Link)
	assert.Equal(t, link, *created.Annotations[0].Link)
	assert.Equal(t, "ref", created.Annotations[1].Key)
	assert.Nil(t, created.Annotations[1].Link)
	assert.Equal(t, "commit", created.Annotations[2].Key)
	assert.Equal(t, "e5f6g7h8", created.Annotations[2].Value)

	// Re-read from the database to confirm the annotations were persisted (scan path), not just
	// echoed from the create request.
	fetched, err := testClient.client.Runs.GetRunByID(ctx, created.Metadata.ID)
	require.Nil(t, err)
	require.NotNil(t, fetched)
	require.Len(t, fetched.Annotations, 3)
	assert.Equal(t, created.Annotations[0].Key, fetched.Annotations[0].Key)
	assert.Equal(t, created.Annotations[0].Value, fetched.Annotations[0].Value)
	require.NotNil(t, fetched.Annotations[0].Link)
	assert.Equal(t, link, *fetched.Annotations[0].Link)
	assert.Equal(t, "ref", fetched.Annotations[1].Key)
	assert.Nil(t, fetched.Annotations[1].Link)
	assert.Equal(t, "commit", fetched.Annotations[2].Key)
	assert.Equal(t, "e5f6g7h8", fetched.Annotations[2].Value)
}

// TestRuns_CreateRun_NoAnnotations verifies that a run created without annotations reads back with an
// empty (non-nil) annotations list, matching the JSONB column default of '[]'.
func TestRuns_CreateRun_NoAnnotations(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-run-no-annotations",
		Description: "test group for run without annotations",
		FullPath:    "test-group-run-no-annotations",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-run-no-annotations",
		GroupID:        group.Metadata.ID,
		Description:    "test workspace for run without annotations",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	created, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
		WorkspaceID: workspace.Metadata.ID,
		Status:      models.RunPending,
		CreatedBy:   "db-integration-tests",
		Plan:        models.Plan{Status: models.PlanCreated},
	})
	require.Nil(t, err)
	require.NotNil(t, created)
	assert.Empty(t, created.Annotations)

	fetched, err := testClient.client.Runs.GetRunByID(ctx, created.Metadata.ID)
	require.Nil(t, err)
	require.NotNil(t, fetched)
	assert.Empty(t, fetched.Annotations)
}
