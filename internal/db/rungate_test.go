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

// getValue implements the sortableField interface for RunGateSortableField
func (r RunGateSortableField) getValue() string {
	return string(r)
}

// createTestRunForGate creates a group, workspace, and run for use as a run gate's parent.
func createTestRunForGate(ctx context.Context, t *testing.T, testClient *testClient, groupName string) (*models.Workspace, *models.Run) {
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      groupName,
		FullPath:  groupName,
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           groupName + "-ws",
		GroupID:        group.Metadata.ID,
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

	return workspace, run
}

func TestRunGates_CreateRunGate(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, run := createTestRunForGate(ctx, t, testClient, "test-group-rungate")

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungate-create",
		Email:    "test-user-rungate-create@test.com",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		runID           string
		workspaceID     string
		policyCheckID   string
	}

	testCases := []testCase{
		{
			name:          "create run gate",
			runID:         run.Metadata.ID,
			workspaceID:   workspace.Metadata.ID,
			policyCheckID: "check-1",
		},
		{
			name:            "negative, run does not exist",
			runID:           nonExistentID,
			workspaceID:     workspace.Metadata.ID,
			policyCheckID:   "check-2",
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
				RunID:         test.runID,
				WorkspaceID:   test.workspaceID,
				PolicyCheckID: test.policyCheckID,
				Type:          models.RunGateTypeOPAPolicy,
				Status:        models.RunGatePending,
				ApprovalRules: []*models.RunGateApprovalRule{
					{
						Name:              "policy-1",
						RequiredApprovals: 1,
						AllowedSubjects: []*models.RunGateAllowedSubject{
							{ID: user.Metadata.ID, TRN: user.Metadata.TRN, Type: models.RunGateSubjectUser},
						},
					},
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, gate)

			assert.Equal(t, test.runID, gate.RunID)
			assert.Equal(t, test.workspaceID, gate.WorkspaceID)
			assert.Equal(t, models.RunGatePending, gate.Status)
			assert.NotEmpty(t, gate.Metadata.ID)
			require.Len(t, gate.ApprovalRules, 1)
			assert.Equal(t, "policy-1", gate.ApprovalRules[0].Name)
		})
	}
}

func TestRunGates_UpdateRunGate(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, run := createTestRunForGate(ctx, t, testClient, "test-group-rungate-update")

	createdGate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:       run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.RunGateTypeOPAPolicy,
		Status:      models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-1", RequiredApprovals: 1},
		},
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		status          models.RunGateStatus
		overriddenBy    *string
	}

	testCases := []testCase{
		{
			name:         "update run gate to overridden",
			version:      createdGate.Metadata.Version,
			status:       models.RunGateOverridden,
			overriddenBy: ptr.String("user@example.com"),
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			status:          models.RunGateApproved,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gateToUpdate := *createdGate
			gateToUpdate.Metadata.Version = test.version
			gateToUpdate.Status = test.status
			gateToUpdate.OverriddenBy = test.overriddenBy

			updatedGate, err := testClient.client.RunGates.UpdateRunGate(ctx, &gateToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedGate)

			assert.Equal(t, test.status, updatedGate.Status)
			assert.Equal(t, test.overriddenBy, updatedGate.OverriddenBy)
			assert.Equal(t, createdGate.Metadata.Version+1, updatedGate.Metadata.Version)
		})
	}
}

func TestRunGates_DeleteRunGate(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, run := createTestRunForGate(ctx, t, testClient, "test-group-rungate-delete")

	createdGate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:       run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.RunGateTypeOPAPolicy,
		Status:      models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-1", RequiredApprovals: 1},
		},
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
			name:    "delete run gate",
			id:      createdGate.Metadata.ID,
			version: createdGate.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdGate.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.RunGates.DeleteRunGate(ctx, &models.RunGate{
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

			gate, err := testClient.client.RunGates.GetRunGateByID(ctx, test.id)
			assert.Nil(t, gate)
			assert.Nil(t, err)
		})
	}
}

func TestRunGates_GetRunGateByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, run := createTestRunForGate(ctx, t, testClient, "test-group-rungate-get-by-id")

	createdGate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:       run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.RunGateTypeOPAPolicy,
		Status:      models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-1", RequiredApprovals: 1},
		},
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectGate      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by id",
			id:         createdGate.Metadata.ID,
			expectGate: true,
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
			gate, err := testClient.client.RunGates.GetRunGateByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGate {
				require.NotNil(t, gate)
				assert.Equal(t, test.id, gate.Metadata.ID)
			} else {
				assert.Nil(t, gate)
			}
		})
	}
}

func TestRunGates_GetRunGateByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, run := createTestRunForGate(ctx, t, testClient, "test-group-rungate-trn")

	createdGate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:       run.Metadata.ID,
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.RunGateTypeOPAPolicy,
		Status:      models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-1", RequiredApprovals: 1},
		},
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectGate      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by TRN",
			trn:        createdGate.Metadata.TRN,
			expectGate: true,
		},
		{
			name: "resource with TRN not found",
			trn:  fmt.Sprintf("trn:run_gate:%s/run-QV84MWI0NDY4OS0wNGExLTRkYTQtOTY1Mi0zYmY4OWE1ZGJkMzU/gate-QV84MWI0NDY4OS0wNGExLTRkYTQtOTY1Mi0zYmY4OWE1ZGJkMzU", workspace.FullPath),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			gate, err := testClient.client.RunGates.GetRunGateByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectGate {
				require.NotNil(t, gate)
				assert.Equal(t, test.trn, gate.Metadata.TRN)
			} else {
				assert.Nil(t, gate)
			}
		})
	}
}

func TestRunGates_GetRunGates(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, run := createTestRunForGate(ctx, t, testClient, "test-group-rungate-list")

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungate-list",
		Email:    "test-user-rungate-list@test.com",
	})
	require.NoError(t, err)

	createdGate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:         run.Metadata.ID,
		WorkspaceID:   workspace.Metadata.ID,
		PolicyCheckID: "check-1",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{
				Name:              "policy-1",
				RequiredApprovals: 1,
				AllowedSubjects: []*models.RunGateAllowedSubject{
					{ID: user.Metadata.ID, TRN: user.Metadata.TRN, Type: models.RunGateSubjectUser},
				},
			},
		},
	})
	require.NoError(t, err)

	_, otherRun := createTestRunForGate(ctx, t, testClient, "test-group-rungate-list-other")
	_, err = testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:         otherRun.Metadata.ID,
		WorkspaceID:   workspace.Metadata.ID,
		PolicyCheckID: "check-2",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGateApproved,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-2", RequiredApprovals: 1},
		},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetRunGatesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all run gates",
			input:       &GetRunGatesInput{},
			expectCount: 2,
		},
		{
			name: "get run gates filtered by run id",
			input: &GetRunGatesInput{
				Filter: &RunGateFilter{RunID: &createdGate.RunID},
			},
			expectCount: 1,
		},
		{
			name: "get run gates filtered by status",
			input: &GetRunGatesInput{
				Filter: &RunGateFilter{Statuses: []models.RunGateStatus{models.RunGatePending}},
			},
			expectCount: 1,
		},
		{
			name: "get run gates filtered by policy check ids",
			input: &GetRunGatesInput{
				Filter: &RunGateFilter{PolicyCheckIDs: []string{"nonexistent-check-id"}},
			},
			expectCount: 0,
		},
		{
			name: "get run gates filtered by eligibility",
			input: &GetRunGatesInput{
				Filter: &RunGateFilter{Eligibility: &RunGateEligibilityFilter{UserID: &user.Metadata.ID}},
			},
			expectCount: 1,
		},
		{
			name: "get run gates filtered by eligibility with no matching principal",
			input: &GetRunGatesInput{
				Filter: &RunGateFilter{Eligibility: &RunGateEligibilityFilter{UserID: ptr.String(nonExistentID)}},
			},
			expectCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.RunGates.GetRunGates(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.RunGates, test.expectCount)
		})
	}
}

func TestRunGates_GetRunGatesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	workspace, _ := createTestRunForGate(ctx, t, testClient, "test-group-rungate-pagination")

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		run, err := testClient.client.Runs.CreateRun(ctx, &models.Run{
			WorkspaceID: workspace.Metadata.ID,
			Status:      models.RunPending,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)

		_, err = testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
			RunID:         run.Metadata.ID,
			WorkspaceID:   workspace.Metadata.ID,
			PolicyCheckID: fmt.Sprintf("check-%d", i),
			Type:          models.RunGateTypeOPAPolicy,
			Status:        models.RunGatePending,
			ApprovalRules: []*models.RunGateApprovalRule{
				{Name: fmt.Sprintf("policy-%d", i), RequiredApprovals: 1},
			},
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		RunGateSortableFieldCreatedAtAsc,
		RunGateSortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := RunGateSortableField(sortByField.getValue())

		result, err := testClient.client.RunGates.GetRunGates(ctx, &GetRunGatesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.RunGates {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
