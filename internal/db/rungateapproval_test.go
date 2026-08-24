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
)

// createTestRunGateForApproval creates a group, workspace, run, and run gate for use as a run gate
// approval's parent.
func createTestRunGateForApproval(ctx context.Context, t *testing.T, testClient *testClient, groupName string) *models.RunGate {
	workspace, run := createTestRunForGate(ctx, t, testClient, groupName)

	gate, err := testClient.client.RunGates.CreateRunGate(ctx, &models.RunGate{
		RunID:         run.Metadata.ID,
		WorkspaceID:   workspace.Metadata.ID,
		PolicyCheckID: groupName + "-check",
		Type:          models.RunGateTypeOPAPolicy,
		Status:        models.RunGatePending,
		ApprovalRules: []*models.RunGateApprovalRule{
			{Name: "policy-1", RequiredApprovals: 1},
		},
	})
	require.NoError(t, err)

	return gate
}

func TestRunGateApprovals_CreateRunGateApproval(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	gate := createTestRunGateForApproval(ctx, t, testClient, "test-group-rungateapproval")

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungateapproval",
		Email:    "test-user-rungateapproval@test.com",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		runGateID       string
		userID          *string
	}

	testCases := []testCase{
		{
			name:      "create run gate approval",
			runGateID: gate.Metadata.ID,
			userID:    &user.Metadata.ID,
		},
		{
			name:            "negative, run gate does not exist",
			runGateID:       nonExistentID,
			userID:          &user.Metadata.ID,
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			approval, err := testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
				RunGateID:    test.runGateID,
				UserID:       test.userID,
				CreatedBy:    "db-integration-tests",
				Decision:     models.RunGateDecisionApprove,
				CoveredRules: []string{"policy-1"},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, approval)

			assert.Equal(t, test.runGateID, approval.RunGateID)
			assert.Equal(t, test.userID, approval.UserID)
			assert.Equal(t, models.RunGateDecisionApprove, approval.Decision)
			assert.Equal(t, []string{"policy-1"}, approval.CoveredRules)
			assert.NotEmpty(t, approval.Metadata.ID)
		})
	}
}

func TestRunGateApprovals_UpdateRunGateApproval(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	gate := createTestRunGateForApproval(ctx, t, testClient, "test-group-rungateapproval-update")

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungateapproval-update",
		Email:    "test-user-rungateapproval-update@test.com",
	})
	require.NoError(t, err)

	createdApproval, err := testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID: gate.Metadata.ID,
		UserID:    &user.Metadata.ID,
		CreatedBy: "db-integration-tests",
		Decision:  models.RunGateDecisionApprove,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		comment         *string
	}

	testCases := []testCase{
		{
			name:    "update run gate approval",
			version: createdApproval.Metadata.Version,
			comment: ptr.String("looks good"),
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			comment:         ptr.String("should not update"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			approvalToUpdate := *createdApproval
			approvalToUpdate.Metadata.Version = test.version
			approvalToUpdate.Comment = test.comment

			updatedApproval, err := testClient.client.RunGateApprovals.UpdateRunGateApproval(ctx, &approvalToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedApproval)

			assert.Equal(t, test.comment, updatedApproval.Comment)
			assert.Equal(t, createdApproval.Metadata.Version+1, updatedApproval.Metadata.Version)
		})
	}
}

func TestRunGateApprovals_GetRunGateApprovalByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	gate := createTestRunGateForApproval(ctx, t, testClient, "test-group-rungateapproval-get-by-id")

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungateapproval-get-by-id",
		Email:    "test-user-rungateapproval-get-by-id@test.com",
	})
	require.NoError(t, err)

	createdApproval, err := testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID: gate.Metadata.ID,
		UserID:    &user.Metadata.ID,
		CreatedBy: "db-integration-tests",
		Decision:  models.RunGateDecisionApprove,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectApproval  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by id",
			id:             createdApproval.Metadata.ID,
			expectApproval: true,
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
			approval, err := testClient.client.RunGateApprovals.GetRunGateApprovalByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectApproval {
				require.NotNil(t, approval)
				assert.Equal(t, test.id, approval.Metadata.ID)
			} else {
				assert.Nil(t, approval)
			}
		})
	}
}

func TestRunGateApprovals_GetRunGateApprovalByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	gate := createTestRunGateForApproval(ctx, t, testClient, "test-group-rungateapproval-trn")

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungateapproval-trn",
		Email:    "test-user-rungateapproval-trn@test.com",
	})
	require.NoError(t, err)

	createdApproval, err := testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID: gate.Metadata.ID,
		UserID:    &user.Metadata.ID,
		CreatedBy: "db-integration-tests",
		Decision:  models.RunGateDecisionApprove,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectApproval  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by TRN",
			trn:            createdApproval.Metadata.TRN,
			expectApproval: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:run_gate_approval:non-existent/non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			approval, err := testClient.client.RunGateApprovals.GetRunGateApprovalByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectApproval {
				require.NotNil(t, approval)
				assert.Equal(t, test.trn, approval.Metadata.TRN)
			} else {
				assert.Nil(t, approval)
			}
		})
	}
}

func TestRunGateApprovals_GetRunGateApprovalsByGateID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	gate := createTestRunGateForApproval(ctx, t, testClient, "test-group-rungateapproval-list")
	otherGate := createTestRunGateForApproval(ctx, t, testClient, "test-group-rungateapproval-list-other")

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungateapproval-list-1",
		Email:    "test-user-rungateapproval-list-1@test.com",
	})
	require.NoError(t, err)

	user2, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-rungateapproval-list-2",
		Email:    "test-user-rungateapproval-list-2@test.com",
	})
	require.NoError(t, err)

	_, err = testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID: gate.Metadata.ID,
		UserID:    &user1.Metadata.ID,
		CreatedBy: "db-integration-tests",
		Decision:  models.RunGateDecisionApprove,
	})
	require.NoError(t, err)

	_, err = testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID: gate.Metadata.ID,
		UserID:    &user2.Metadata.ID,
		CreatedBy: "db-integration-tests",
		Decision:  models.RunGateDecisionReject,
	})
	require.NoError(t, err)

	_, err = testClient.client.RunGateApprovals.CreateRunGateApproval(ctx, &models.RunGateApproval{
		RunGateID: otherGate.Metadata.ID,
		UserID:    &user1.Metadata.ID,
		CreatedBy: "db-integration-tests",
		Decision:  models.RunGateDecisionApprove,
	})
	require.NoError(t, err)

	type testCase struct {
		name        string
		gateID      string
		expectCount int
	}

	testCases := []testCase{
		{
			name:        "get approvals for a gate with two decisions",
			gateID:      gate.Metadata.ID,
			expectCount: 2,
		},
		{
			name:        "get approvals for a gate with one decision",
			gateID:      otherGate.Metadata.ID,
			expectCount: 1,
		},
		{
			name:        "get approvals for a gate with no decisions",
			gateID:      nonExistentID,
			expectCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.RunGateApprovals.GetRunGateApprovalsByGateID(ctx, test.gateID)
			require.NoError(t, err)
			assert.Len(t, result, test.expectCount)
		})
	}
}
