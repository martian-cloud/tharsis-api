//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// createTestGroup is a helper that creates a group for use in cleanup policy tests.
func createTestGroupForCleanupPolicy(ctx context.Context, t *testing.T, testClient *testClient, name string) *models.Group {
	t.Helper()

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        name,
		Description: "test group for cleanup policy",
		FullPath:    name,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)
	return group
}

// createTestWorkspaceForCleanupPolicy is a helper that creates a workspace for use in cleanup policy tests.
func createTestWorkspaceForCleanupPolicy(ctx context.Context, t *testing.T, testClient *testClient, name string, groupID string) *models.Workspace {
	t.Helper()

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           name,
		GroupID:        groupID,
		Description:    "test workspace for cleanup policy",
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)
	return workspace
}

func TestCleanupPolicies_CreateCleanupPolicy(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-create-policy")
	workspace := createTestWorkspaceForCleanupPolicy(ctx, t, testClient, "test-ws-create-policy", group.Metadata.ID)

	type testCase struct {
		name            string
		input           *models.CleanupPolicy
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "create RUNS policy on workspace",
			input: &models.CleanupPolicy{
				WorkspaceID: &workspace.Metadata.ID,
				Kind:        models.CleanupRuleKindRuns,
				RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{
					{Strategy: models.StrategyAge, DeleteAfterDays: 30},
				}},
			},
		},
		{
			name: "create TERRAFORM_MODULES policy on group",
			input: &models.CleanupPolicy{
				GroupID: &group.Metadata.ID,
				Kind:    models.CleanupRuleKindTerraformModules,
				TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
					{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
				}},
			},
		},
		{
			name: "create TERRAFORM_PROVIDERS policy on group",
			input: &models.CleanupPolicy{
				GroupID: &group.Metadata.ID,
				Kind:    models.CleanupRuleKindTerraformProviders,
				TerraformProviderPolicyData: &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
					{Strategy: models.StrategyAge, DeleteAfterDays: 30},
				}},
			},
		},
		{
			name: "duplicate kind on same namespace",
			input: &models.CleanupPolicy{
				WorkspaceID: &workspace.Metadata.ID,
				Kind:        models.CleanupRuleKindRuns,
				RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{
					{Strategy: models.StrategyAge, DeleteAfterDays: 30},
				}},
			},
			expectErrorCode: errors.EConflict,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			created, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)

			assert.NotEmpty(t, created.Metadata.ID)
			assert.NotEmpty(t, created.Metadata.TRN)
			assert.Equal(t, test.input.Kind, created.Kind)
			assert.Equal(t, test.input.GroupID, created.GroupID)
			assert.Equal(t, test.input.WorkspaceID, created.WorkspaceID)
		})
	}
}

func TestCleanupPolicies_GetCleanupPolicyByID(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-get-by-id")
	policy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &group.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformModules,
		TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		expectNil       bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "found",
			id:   policy.Metadata.ID,
		},
		{
			name:      "not found",
			id:        nonExistentID,
			expectNil: true,
		},
		{
			name:            "invalid ID",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got, err := testClient.client.CleanupPolicies.GetCleanupPolicyByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, policy.Metadata.ID, got.Metadata.ID)
			assert.Equal(t, policy.Kind, got.Kind)
			assert.Equal(t, policy.Metadata.TRN, got.Metadata.TRN)
		})
	}
}

func TestCleanupPolicies_GetCleanupPolicyByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-get-by-trn")
	policy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &group.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformProviders,
		TerraformProviderPolicyData: &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		trn             string
		expectNil       bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "found",
			trn:  policy.Metadata.TRN,
		},
		{
			name:      "not found — valid TRN format, no match",
			trn:       "trn:cleanup_policy:nonexistent-group/TERRAFORM_PROVIDERS",
			expectNil: true,
		},
		{
			name:            "invalid TRN string",
			trn:             "not-a-trn",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "TRN missing parent path (no kind segment)",
			trn:             "trn:cleanup_policy:TERRAFORM_PROVIDERS",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got, err := testClient.client.CleanupPolicies.GetCleanupPolicyByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, policy.Metadata.ID, got.Metadata.ID)
			assert.Equal(t, policy.Metadata.TRN, got.Metadata.TRN)
		})
	}
}

func TestCleanupPolicies_GetCleanupPolicies(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	groupA := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-list-a")
	groupB := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-list-b")
	workspace := createTestWorkspaceForCleanupPolicy(ctx, t, testClient, "test-ws-list", groupA.Metadata.ID)

	policyModulesA, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &groupA.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformModules,
		TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	policyProvidersA, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &groupA.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformProviders,
		TerraformProviderPolicyData: &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	policyModulesB, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &groupB.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformModules,
		TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	policyRunsWS, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		WorkspaceID: &workspace.Metadata.ID,
		Kind:        models.CleanupRuleKindRuns,
		RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30},
		}},
	})
	require.NoError(t, err)

	groupAPath := "test-group-list-a"
	groupBPath := "test-group-list-b"
	workspacePath := "test-group-list-a/test-ws-list"
	kindModules := models.CleanupRuleKindTerraformModules
	nonexistentGroupPath := "nonexistent-group"

	type testCase struct {
		name      string
		input     *GetCleanupPoliciesInput
		expectIDs []string
	}

	testCases := []testCase{
		{
			name:      "filter by single namespace path — group",
			input:     &GetCleanupPoliciesInput{NamespacePaths: []string{groupAPath}},
			expectIDs: []string{policyModulesA.Metadata.ID, policyProvidersA.Metadata.ID},
		},
		{
			name:      "filter by workspace namespace path",
			input:     &GetCleanupPoliciesInput{NamespacePaths: []string{workspacePath}},
			expectIDs: []string{policyRunsWS.Metadata.ID},
		},
		{
			name:      "filter by kind",
			input:     &GetCleanupPoliciesInput{Kind: &kindModules},
			expectIDs: []string{policyModulesA.Metadata.ID, policyModulesB.Metadata.ID},
		},
		{
			name:      "filter by kind + namespace path",
			input:     &GetCleanupPoliciesInput{Kind: &kindModules, NamespacePaths: []string{groupBPath}},
			expectIDs: []string{policyModulesB.Metadata.ID},
		},
		{
			name:      "filter by specific policy IDs",
			input:     &GetCleanupPoliciesInput{CleanupPolicyIDs: []string{policyProvidersA.Metadata.ID, policyRunsWS.Metadata.ID}},
			expectIDs: []string{policyProvidersA.Metadata.ID, policyRunsWS.Metadata.ID},
		},
		{
			name:      "no matches",
			input:     &GetCleanupPoliciesInput{NamespacePaths: []string{nonexistentGroupPath}},
			expectIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			results, err := testClient.client.CleanupPolicies.GetCleanupPolicies(ctx, test.input)
			require.NoError(t, err)

			gotIDs := make([]string, len(results))
			for i, p := range results {
				gotIDs[i] = p.Metadata.ID
			}

			assert.ElementsMatch(t, test.expectIDs, gotIDs)
		})
	}
}

func TestCleanupPolicies_UpdateCleanupPolicy(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-update-policy")

	policy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &group.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformModules,
		TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	t.Run("update rules", func(t *testing.T) {
		policy.TerraformModulePolicyData = &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
			{Strategy: models.StrategyProtect, NameGlob: "stable-*"},
		}}

		updated, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, policy)
		require.NoError(t, err)
		require.NotNil(t, updated)

		assert.Equal(t, policy.Metadata.Version+1, updated.Metadata.Version)
		require.Len(t, updated.TerraformModulePolicyData.Rules, 2)
		assert.Equal(t, models.CleanupGlob("stable-*"), updated.TerraformModulePolicyData.Rules[1].NameGlob)

		policy = updated
	})

	t.Run("update sweep claimed_at", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Microsecond)
		policy.SweepClaimedAt = &now

		updated, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, policy)
		require.NoError(t, err)
		require.NotNil(t, updated.SweepClaimedAt)
		assert.WithinDuration(t, now, *updated.SweepClaimedAt, time.Second)

		policy = updated
	})

	t.Run("update sweep cursor", func(t *testing.T) {
		cursor := "some-pagination-cursor"
		policy.SweepCursor = &cursor

		updated, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, policy)
		require.NoError(t, err)
		require.NotNil(t, updated.SweepCursor)
		assert.Equal(t, cursor, *updated.SweepCursor)

		policy = updated
	})

	t.Run("clear sweep cursor", func(t *testing.T) {
		policy.SweepCursor = nil

		updated, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, policy)
		require.NoError(t, err)
		assert.Nil(t, updated.SweepCursor)

		policy = updated
	})

	t.Run("optimistic lock conflict", func(t *testing.T) {
		stale := *policy
		stale.Metadata.Version = policy.Metadata.Version - 1

		_, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, &stale)
		assert.Equal(t, ErrOptimisticLockError, err)
	})
}

func TestCleanupPolicies_DeleteCleanupPolicy(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-delete-policy")

	policy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		GroupID: &group.Metadata.ID,
		Kind:    models.CleanupRuleKindTerraformModules,
		TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}},
	})
	require.NoError(t, err)

	t.Run("stale version returns OLE", func(t *testing.T) {
		stale := *policy
		stale.Metadata.Version = policy.Metadata.Version - 1

		err := testClient.client.CleanupPolicies.DeleteCleanupPolicy(ctx, &stale)
		assert.Equal(t, ErrOptimisticLockError, err)

		// policy should still exist
		got, err := testClient.client.CleanupPolicies.GetCleanupPolicyByID(ctx, policy.Metadata.ID)
		require.NoError(t, err)
		assert.NotNil(t, got)
	})

	t.Run("delete existing policy", func(t *testing.T) {
		err := testClient.client.CleanupPolicies.DeleteCleanupPolicy(ctx, policy)
		require.NoError(t, err)

		got, err := testClient.client.CleanupPolicies.GetCleanupPolicyByID(ctx, policy.Metadata.ID)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}

func TestCleanupPolicies_ClaimCleanupPoliciesForSweep(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-claim")
	workspace := createTestWorkspaceForCleanupPolicy(ctx, t, testClient, "test-ws-claim", group.Metadata.ID)

	// Create five policies to exercise limit and ordering.
	groupB := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-claim-b")
	groupC := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-claim-c")
	groupD := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-claim-d")

	createModulesPolicy := func(groupID string) *models.CleanupPolicy {
		p, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
			GroupID: &groupID,
			Kind:    models.CleanupRuleKindTerraformModules,
			TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
				{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
			}},
		})
		require.NoError(t, err)
		return p
	}

	policyA := createModulesPolicy(group.Metadata.ID)
	policyB := createModulesPolicy(groupB.Metadata.ID)
	policyC := createModulesPolicy(groupC.Metadata.ID)
	policyD := createModulesPolicy(groupD.Metadata.ID)

	policyRuns, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
		WorkspaceID: &workspace.Metadata.ID,
		Kind:        models.CleanupRuleKindRuns,
		RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30},
		}},
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)

	t.Run("claims unclaimed policies up to limit", func(t *testing.T) {
		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         3,
		})
		require.NoError(t, err)
		assert.Len(t, claimed, 3)

		for _, p := range claimed {
			assert.NotNil(t, p.SweepClaimedAt, "claimed policy should have sweep_claimed_at set")
		}
	})

	t.Run("does not re-claim recently claimed policies", func(t *testing.T) {
		// Claim all five to stamp them.
		_, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         5,
		})
		require.NoError(t, err)

		// Now try again with ClaimedBefore in the past — nothing should be reclaimable.
		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         5,
		})
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("reclaims expired claims", func(t *testing.T) {
		// Backdate policyA's claim to simulate a stale sweeper.
		expiredTime := now.Add(-3 * time.Hour)
		policyA.SweepClaimedAt = &expiredTime
		updated, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, policyA)
		require.NoError(t, err)
		policyA = updated

		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: now.Add(-2 * time.Hour), // expiredTime is older than this
			Limit:         5,
		})
		require.NoError(t, err)

		claimedIDs := make([]string, len(claimed))
		for i, p := range claimed {
			claimedIDs[i] = p.Metadata.ID
		}
		assert.Contains(t, claimedIDs, policyA.Metadata.ID, "expired claim should have been reclaimed")
	})

	t.Run("claimed policies have all fields hydrated", func(t *testing.T) {
		// Reset all claims so we can claim fresh.
		for _, p := range []*models.CleanupPolicy{policyA, policyB, policyC, policyD, policyRuns} {
			p.SweepClaimedAt = nil
			updated, err := testClient.client.CleanupPolicies.UpdateCleanupPolicy(ctx, p)
			require.NoError(t, err)
			// re-assign to keep version current
			switch p.Metadata.ID {
			case policyA.Metadata.ID:
				policyA = updated
			case policyB.Metadata.ID:
				policyB = updated
			case policyC.Metadata.ID:
				policyC = updated
			case policyD.Metadata.ID:
				policyD = updated
			case policyRuns.Metadata.ID:
				policyRuns = updated
			}
		}

		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         1,
		})
		require.NoError(t, err)
		require.Len(t, claimed, 1)

		p := claimed[0]
		assert.NotEmpty(t, p.Metadata.ID)
		assert.NotEmpty(t, p.Metadata.TRN)
		assert.NotEmpty(t, p.Kind)
		assert.NotNil(t, p.SweepClaimedAt)
		// At least one of GroupID or WorkspaceID must be set.
		assert.True(t, p.GroupID != nil || p.WorkspaceID != nil)
	})

	t.Run("empty result when no policies are due", func(t *testing.T) {
		// Claim everything with a far-future ClaimedBefore so nothing is reclaimable.
		_, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         10,
		})
		require.NoError(t, err)

		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         10,
		})
		require.NoError(t, err)
		assert.Empty(t, claimed)
	})

	t.Run("skips a policy with disabled=true", func(t *testing.T) {
		disabledGroup := createTestGroupForCleanupPolicy(ctx, t, testClient, "test-group-claim-disabled")
		disabledGroupPolicy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
			GroupID:  &disabledGroup.Metadata.ID,
			Kind:     models.CleanupRuleKindTerraformModules,
			Disabled: true,
			TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
				{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
			}},
		})
		require.NoError(t, err)

		disabledWorkspace := createTestWorkspaceForCleanupPolicy(ctx, t, testClient, "test-ws-claim-disabled", group.Metadata.ID)
		disabledWorkspacePolicy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
			WorkspaceID: &disabledWorkspace.Metadata.ID,
			Kind:        models.CleanupRuleKindRuns,
			Disabled:    true,
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{
				{Strategy: models.StrategyAge, DeleteAfterDays: 30},
			}},
		})
		require.NoError(t, err)

		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         10,
		})
		require.NoError(t, err)

		claimedIDs := make([]string, len(claimed))
		for i, p := range claimed {
			claimedIDs[i] = p.Metadata.ID
		}
		assert.NotContains(t, claimedIDs, disabledGroupPolicy.Metadata.ID)
		assert.NotContains(t, claimedIDs, disabledWorkspacePolicy.Metadata.ID)
	})

	t.Run("claims a non-disabled policy on a workspace", func(t *testing.T) {
		enabledWorkspace := createTestWorkspaceForCleanupPolicy(ctx, t, testClient, "test-ws-claim-enabled", group.Metadata.ID)
		enabledWorkspacePolicy, err := testClient.client.CleanupPolicies.CreateCleanupPolicy(ctx, &models.CleanupPolicy{
			WorkspaceID: &enabledWorkspace.Metadata.ID,
			Kind:        models.CleanupRuleKindRuns,
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{
				{Strategy: models.StrategyAge, DeleteAfterDays: 30},
			}},
		})
		require.NoError(t, err)

		claimed, err := testClient.client.CleanupPolicies.ClaimCleanupPoliciesForSweep(ctx, &ClaimCleanupPoliciesForSweepInput{
			ClaimedBefore: past,
			Limit:         10,
		})
		require.NoError(t, err)

		claimedIDs := make([]string, len(claimed))
		for i, p := range claimed {
			claimedIDs[i] = p.Metadata.ID
		}
		assert.Contains(t, claimedIDs, enabledWorkspacePolicy.Metadata.ID)
	})
}
