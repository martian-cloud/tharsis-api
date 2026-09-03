//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for PolicySortableField
func (p PolicySortableField) getValue() string {
	return string(p)
}

func TestPolicies_CreatePolicy(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy",
		FullPath:  "test-group-policy",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-policy",
		Email:    "test-user-policy@test.com",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		groupID         string
		policyName      string
		allowedUserIDs  []string
	}

	testCases := []testCase{
		{
			name:       "create policy",
			groupID:    group.Metadata.ID,
			policyName: "test-policy",
		},
		{
			name:           "create policy with allowed approvers",
			groupID:        group.Metadata.ID,
			policyName:     "test-policy-approvers",
			allowedUserIDs: []string{user.Metadata.ID},
		},
		{
			// Policy.Validate rejects a repeated approver before it gets here, so this covers the
			// backstop: the unique index on (policy_id, user_id), reported as EInvalid rather than a
			// raw constraint error.
			name:            "negative, a repeated approver is rejected",
			groupID:         group.Metadata.ID,
			policyName:      "test-policy-dupe-approvers",
			allowedUserIDs:  []string{user.Metadata.ID, user.Metadata.ID},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "negative, group does not exist",
			groupID:         nonExistentID,
			policyName:      "orphan-policy",
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			policy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
				GroupID: test.groupID,
				Name:    test.policyName,
				Kind:    models.PolicyKindOPA,
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "trn:package_version:test-group-policy/test-package/1.0.0",
					EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
					Stage:                          models.RunTaskStageNamePostPlan,
				},
				RequiredApprovals: 1,
				AllowedUserIDs:    test.allowedUserIDs,
				CreatedBy:         "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, policy)

			assert.Equal(t, test.policyName, policy.Name)
			assert.Equal(t, test.groupID, policy.GroupID)
			assert.Equal(t, models.PolicyKindOPA, policy.Kind)
			// Both levels live in the kind_data JSONB, so a create is also the round-trip check.
			require.NotNil(t, policy.OPAData)
			assert.Equal(t, models.PolicyEnforcementSoftMandatory, policy.OPAData.EnforcementLevel)
			assert.Equal(t, models.PolicyEnforcementAdvisory, policy.OPAData.SpeculativeRunEnforcementLevel)
			assert.NotEmpty(t, policy.Metadata.ID)
			assert.ElementsMatch(t, test.allowedUserIDs, policy.AllowedUserIDs)
		})
	}
}

// TestPolicies_CreateModuleAttestationPolicy verifies the module attestation kind round-trips through
// the kind_data JSONB column and is selectable by the stage filter, which reads kind_data->>'stage'.
func TestPolicies_CreateModuleAttestationPolicy(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-attestation",
		FullPath:  "test-group-attestation",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	predicateType := "https://slsa.dev/provenance/v1"
	created, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "require-provenance",
		Kind:    models.PolicyKindModuleAttestation,
		ModuleAttestationData: &models.ModuleAttestationPolicyData{
			PublicKey:                      "-----BEGIN PUBLIC KEY-----\nkey-body\n-----END PUBLIC KEY-----",
			PredicateType:                  &predicateType,
			VerifyStateLineage:             true,
			EnforcementLevel:               models.PolicyEnforcementHardMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementHardMandatory,
			Stage:                          models.RunTaskStageNamePrePlan,
		},
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)
	require.NotNil(t, created)

	assert.Equal(t, models.PolicyKindModuleAttestation, created.Kind)
	assert.Nil(t, created.OPAData)
	require.NotNil(t, created.ModuleAttestationData)
	assert.Equal(t, predicateType, *created.ModuleAttestationData.PredicateType)
	assert.True(t, created.ModuleAttestationData.VerifyStateLineage)
	assert.Equal(t, models.RunTaskStageNamePrePlan, created.ModuleAttestationData.Stage)
	assert.Equal(t, models.PolicyEnforcementHardMandatory, created.ModuleAttestationData.EnforcementLevel)

	fetched, err := testClient.client.Policies.GetPolicyByID(ctx, created.Metadata.ID)
	require.Nil(t, err)
	require.NotNil(t, fetched)
	require.NotNil(t, fetched.ModuleAttestationData)
	assert.Equal(t, created.ModuleAttestationData.PublicKey, fetched.ModuleAttestationData.PublicKey)

	prePlan := models.RunTaskStageNamePrePlan
	result, err := testClient.client.Policies.GetPolicies(ctx, &GetPoliciesInput{
		Filter: &PolicyFilter{GroupIDs: []string{group.Metadata.ID}, Stage: &prePlan},
	})
	require.Nil(t, err)
	require.Len(t, result.Policies, 1)
	assert.Equal(t, created.Metadata.ID, result.Policies[0].Metadata.ID)

	postPlan := models.RunTaskStageNamePostPlan
	result, err = testClient.client.Policies.GetPolicies(ctx, &GetPoliciesInput{
		Filter: &PolicyFilter{GroupIDs: []string{group.Metadata.ID}, Stage: &postPlan},
	})
	require.Nil(t, err)
	assert.Empty(t, result.Policies)
}

func TestPolicies_UpdatePolicy(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-update",
		FullPath:  "test-group-policy-update",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-policy-update",
		Email:    "test-user-policy-update@test.com",
	})
	require.Nil(t, err)

	createdPolicy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "test-policy-update",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-update/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePostPlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name              string
		expectErrorCode   errors.CodeType
		version           int
		requiredApprovals int
		allowedUserIDs    []string
	}

	testCases := []testCase{
		{
			name:              "update policy",
			version:           createdPolicy.Metadata.Version,
			requiredApprovals: 2,
			allowedUserIDs:    []string{user.Metadata.ID},
		},
		{
			name:              "update will fail because resource version doesn't match",
			expectErrorCode:   errors.EOptimisticLock,
			version:           -1,
			requiredApprovals: 3,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			policyToUpdate := *createdPolicy
			policyToUpdate.Metadata.Version = test.version
			policyToUpdate.RequiredApprovals = test.requiredApprovals
			policyToUpdate.AllowedUserIDs = test.allowedUserIDs

			updatedPolicy, err := testClient.client.Policies.UpdatePolicy(ctx, &policyToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedPolicy)

			assert.Equal(t, test.requiredApprovals, updatedPolicy.RequiredApprovals)
			assert.Equal(t, createdPolicy.Metadata.Version+1, updatedPolicy.Metadata.Version)
			assert.ElementsMatch(t, test.allowedUserIDs, updatedPolicy.AllowedUserIDs)
		})
	}
}

// UpdatePolicy replaces the approver rows rather than merging them, so it is the other write path the
// unique index on (policy_id, user_id) has to guard. It gets its own fixture because the cases in
// TestPolicies_UpdatePolicy share one policy's version.
func TestPolicies_UpdatePolicy_RejectsDuplicateApprovers(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-update-dedupe",
		FullPath:  "test-group-policy-update-dedupe",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-policy-update-dedupe",
		Email:    "test-user-policy-update-dedupe@test.com",
	})
	require.Nil(t, err)

	policy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "test-policy-update-dedupe",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-update-dedupe/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePostPlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
	})
	require.Nil(t, err)

	policy.AllowedUserIDs = []string{user.Metadata.ID, user.Metadata.ID}
	updated, err := testClient.client.Policies.UpdatePolicy(ctx, policy)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	assert.Nil(t, updated)

	// The update runs in one transaction, so a rejected approver list must leave no approvers behind.
	reread, err := testClient.client.Policies.GetPolicyByID(ctx, policy.Metadata.ID)
	require.Nil(t, err)
	assert.Empty(t, reread.AllowedUserIDs)

	// The same list without the repeat is accepted, so the rejection is the duplicate and nothing else.
	policy.AllowedUserIDs = []string{user.Metadata.ID}
	updated, err = testClient.client.Policies.UpdatePolicy(ctx, policy)
	require.Nil(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, []string{user.Metadata.ID}, updated.AllowedUserIDs)
}

func TestPolicies_DeletePolicy(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-delete",
		FullPath:  "test-group-policy-delete",
		CreatedBy: "db-integration-tests",
	})
	require.Nil(t, err)

	createdPolicy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "test-policy-delete",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-delete/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePostPlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
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
			name:    "delete policy",
			id:      createdPolicy.Metadata.ID,
			version: createdPolicy.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdPolicy.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Policies.DeletePolicy(ctx, &models.Policy{
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

			policy, err := testClient.client.Policies.GetPolicyByID(ctx, test.id)
			assert.Nil(t, policy)
			assert.Nil(t, err)
		})
	}
}

func TestPolicies_GetPolicyByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-get-by-id",
		FullPath:  "test-group-policy-get-by-id",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	createdPolicy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "test-policy-get-by-id",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-get-by-id/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePostPlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectPolicy    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdPolicy.Metadata.ID,
			expectPolicy: true,
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
			policy, err := testClient.client.Policies.GetPolicyByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPolicy {
				require.NotNil(t, policy)
				assert.Equal(t, test.id, policy.Metadata.ID)
			} else {
				assert.Nil(t, policy)
			}
		})
	}
}

func TestPolicies_GetPolicyByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-trn",
		FullPath:  "test-group-policy-trn",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	createdPolicy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "test-policy-trn",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-trn/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePostPlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectPolicy    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by TRN",
			trn:          createdPolicy.Metadata.TRN,
			expectPolicy: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:policy:test-group-policy-trn/non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			policy, err := testClient.client.Policies.GetPolicyByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectPolicy {
				require.NotNil(t, policy)
				assert.Equal(t, test.trn, policy.Metadata.TRN)
			} else {
				assert.Nil(t, policy)
			}
		})
	}
}

func TestPolicies_GetPolicies(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-list",
		FullPath:  "test-group-policy-list",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	otherGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-list-other",
		FullPath:  "test-group-policy-list-other",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	createdPolicy, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: group.Metadata.ID,
		Name:    "test-policy-list-1",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-list/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePostPlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	_, err = testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
		GroupID: otherGroup.Metadata.ID,
		Name:    "test-policy-list-2",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "trn:package_version:test-group-policy-list-other/test-package/1.0.0",
			EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			Stage:                          models.RunTaskStageNamePrePlan,
		},
		RequiredApprovals: 1,
		CreatedBy:         "db-integration-tests",
	})
	require.NoError(t, err)

	postPlanStage := models.RunTaskStageNamePostPlan

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetPoliciesInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all policies",
			input:       &GetPoliciesInput{},
			expectCount: 2,
		},
		{
			name: "get policies filtered by group",
			input: &GetPoliciesInput{
				Filter: &PolicyFilter{GroupIDs: []string{group.Metadata.ID}},
			},
			expectCount: 1,
		},
		{
			name: "get policies filtered by policy ids",
			input: &GetPoliciesInput{
				Filter: &PolicyFilter{PolicyIDs: []string{createdPolicy.Metadata.ID}},
			},
			expectCount: 1,
		},
		{
			name: "get policies filtered by group path tree",
			input: &GetPoliciesInput{
				Filter: &PolicyFilter{GroupPathTree: &group.FullPath},
			},
			expectCount: 1,
		},
		{
			name: "get policies filtered by stage",
			input: &GetPoliciesInput{
				Filter: &PolicyFilter{Stage: &postPlanStage},
			},
			expectCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Policies.GetPolicies(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Policies, test.expectCount)
		})
	}
}

func TestPolicies_GetPoliciesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-policy-pagination",
		FullPath:  "test-group-policy-pagination",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Policies.CreatePolicy(ctx, &models.Policy{
			GroupID: group.Metadata.ID,
			Name:    fmt.Sprintf("test-policy-pagination-%d", i),
			Kind:    models.PolicyKindOPA,
			OPAData: &models.OPAPolicyData{
				PackageSource:                  "trn:package_version:test-group-policy-pagination/test-package/1.0.0",
				EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
				SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				Stage:                          models.RunTaskStageNamePostPlan,
			},
			RequiredApprovals: 1,
			CreatedBy:         "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		PolicySortableFieldCreatedAtAsc,
		PolicySortableFieldCreatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := PolicySortableField(sortByField.getValue())

		result, err := testClient.client.Policies.GetPolicies(ctx, &GetPoliciesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Policies {
			resources = append(resources, resource)
		}

		return result.PageInfo, resources, nil
	})
}
