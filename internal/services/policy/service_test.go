package policy

import (
	"context"
	"slices"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

func TestCreatePolicy(t *testing.T) {
	groupID := "group-id"
	packageSource := "my-group/my-package"
	policyID := "policy-id"
	groupPath := "my-group"

	tests := []struct {
		name              string
		limit             int
		injectPolicyCount int32
		expectLimitCheck  bool
		expectErrCode     errors.CodeType
	}{
		{
			name:              "group owner under limit",
			limit:             5,
			injectPolicyCount: 5,
			expectLimitCheck:  true,
		},
		{
			name:              "group owner exceeds limit",
			limit:             5,
			injectPolicyCount: 6,
			expectLimitCheck:  true,
			expectErrCode:     errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test := test

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockGroups := db.NewMockGroups(t)
			mockPolicies := db.NewMockPolicies(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			mockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}, nil)

			input := &CreatePolicyInput{
				GroupID: groupID,
				Name:    "test-policy",
				Kind:    models.PolicyKindOPA,
				OPAData: &models.OPAPolicyData{
					PackageSource:                  packageSource,
					Stage:                          models.RunTaskStageNamePostPlan,
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			}

			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

			mockPolicies.On("CreatePolicy", mock.Anything, mock.Anything).
				Return(&models.Policy{Metadata: models.ResourceMetadata{ID: policyID}}, nil)

			if test.expectLimitCheck {
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
					Return(&db.PoliciesResult{
						PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(test.injectPolicyCount)},
					}, nil)
				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: test.limit}, nil)
			}

			if test.expectErrCode == "" {
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
			}

			dbClient := db.Client{
				Groups:         mockGroups,
				Policies:       mockPolicies,
				Transactions:   mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			testLogger, _ := logger.NewForTest()
			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient))

			policy, err := service.CreatePolicy(auth.WithCaller(ctx, mockCaller), input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, policyID, policy.Metadata.ID)
		})
	}
}

// TestCreatePolicy_StageGate verifies the OPA stage gate: pre_plan and post_plan are accepted,
// while post_apply is rejected as not-yet-supported.
func TestCreatePolicy_StageGate(t *testing.T) {
	const groupID = "group-id"

	tests := []struct {
		name          string
		stage         models.RunTaskStageName
		expectErrCode errors.CodeType
	}{
		{name: "pre_plan accepted", stage: models.RunTaskStageNamePrePlan},
		{name: "post_plan accepted", stage: models.RunTaskStageNamePostPlan},
		{name: "post_apply rejected", stage: models.RunTaskStageNamePostApply, expectErrCode: errors.EInvalid},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockGroups := db.NewMockGroups(t)
			mockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: "my-group"}, nil)

			mockPolicies := db.NewMockPolicies(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			// An accepted stage proceeds through creation; a rejected stage returns before the tx.
			if test.expectErrCode == "" {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("CreatePolicy", mock.Anything, mock.Anything).
					Return(&models.Policy{Metadata: models.ResourceMetadata{ID: "policy-id"}}, nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
					Return(&db.PoliciesResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(0)}}, nil)
				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: 100}, nil)
			}

			dbClient := db.Client{
				Groups:         mockGroups,
				Policies:       mockPolicies,
				Transactions:   mockTransactions,
				ResourceLimits: mockResourceLimits,
			}

			testLogger, _ := logger.NewForTest()
			service := NewService(testLogger, &dbClient, limits.NewLimitChecker(&dbClient))

			input := &CreatePolicyInput{
				GroupID: groupID,
				Name:    "test-policy",
				Kind:    models.PolicyKindOPA,
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "my-group/my-package",
					Stage:                          test.stage,
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			}

			_, err := service.CreatePolicy(auth.WithCaller(ctx, mockCaller), input)
			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestPolicy_ServiceAccountAccessForGroup_MultipleApprovers covers what a per-ID lookup could not get
// wrong: the whole approver list resolves through one query, so the result rows arrive in the DB's order
// rather than the caller's, and the check has to pair them back up itself. The mock accepts exactly one
// call carrying every requested ID, which fails the test if the batching regresses to a loop.
func TestPolicy_ServiceAccountAccessForGroup_MultipleApprovers(t *testing.T) {
	const (
		groupID   = "group-id"
		groupPath = "root/mid/team"
	)

	// inGroup and outside are both real; missing resolves to no row at all.
	inGroup := models.ServiceAccount{Metadata: models.ResourceMetadata{
		ID: "sa-in-group", TRN: trn.TypeServiceAccount.Build(groupPath + "/deployer"),
	}}
	outside := models.ServiceAccount{Metadata: models.ResourceMetadata{
		ID: "sa-outside", TRN: trn.TypeServiceAccount.Build("root/other/deployer"),
	}}

	tests := []struct {
		name              string
		requestedIDs      []string
		expectErrCode     errors.CodeType
		expectErrContains string
	}{
		{
			// Rows come back reversed relative to the request, so pairing a requested ID with a row
			// by position rather than by ID would blame the wrong service account here.
			name:              "out of scope approver is reported by its own path",
			requestedIDs:      []string{inGroup.Metadata.ID, outside.Metadata.ID},
			expectErrCode:     errors.EInvalid,
			expectErrContains: "service account root/other/deployer is outside the scope of group " + groupPath,
		},
		{
			// Two bad IDs at once: the earlier one in the caller's list wins, so neither failure mode
			// unconditionally shadows the other.
			name:              "earliest problem in the request wins",
			requestedIDs:      []string{"sa-missing", outside.Metadata.ID},
			expectErrCode:     errors.ENotFound,
			expectErrContains: "service account with ID sa-missing not found",
		},
		{
			name:         "every approver in scope",
			requestedIDs: []string{inGroup.Metadata.ID},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockGroups := db.NewMockGroups(t)
			mockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}, nil)

			// Reversed on purpose: the DB orders by ID, not by the order the IDs were supplied in.
			returned := []models.ServiceAccount{}
			for i := len(test.requestedIDs) - 1; i >= 0; i-- {
				switch test.requestedIDs[i] {
				case inGroup.Metadata.ID:
					returned = append(returned, inGroup)
				case outside.Metadata.ID:
					returned = append(returned, outside)
				}
			}

			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockServiceAccounts.On("GetServiceAccounts", mock.Anything,
				mock.MatchedBy(func(input *db.GetServiceAccountsInput) bool {
					return input.Filter != nil && slices.Equal(input.Filter.ServiceAccountIDs, test.requestedIDs)
				})).
				Return(&db.ServiceAccountsResult{ServiceAccounts: returned}, nil).
				Once()

			mockPolicies := db.NewMockPolicies(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			// A rejected approver returns before the transaction is ever opened.
			if test.expectErrCode == "" {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("CreatePolicy", mock.Anything, mock.Anything).
					Return(&models.Policy{Metadata: models.ResourceMetadata{ID: "policy-id"}}, nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
					Return(&db.PoliciesResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(0)}}, nil)
				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: 100}, nil)
			}

			dbClient := db.Client{
				Groups:          mockGroups,
				Policies:        mockPolicies,
				ServiceAccounts: mockServiceAccounts,
				Transactions:    mockTransactions,
				ResourceLimits:  mockResourceLimits,
			}

			_, err := newTestPolicyService(&dbClient).CreatePolicy(auth.WithCaller(ctx, mockCaller), &CreatePolicyInput{
				GroupID: groupID,
				Name:    "test-policy",
				Kind:    models.PolicyKindOPA,
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "my-group/my-package",
					Stage:                          models.RunTaskStageNamePostPlan,
					EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
				RequiredApprovals:        1,
				AllowedServiceAccountIDs: test.requestedIDs,
			})

			if test.expectErrCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				assert.Contains(t, err.Error(), test.expectErrContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestPolicy_ServiceAccountAccessForGroup verifies the approver scope check on both mutations: an
// approver service account must live in the owning group or one of its ancestors, the direction
// inheritance runs. Both entry points are exercised because each calls the check at a different point
// in its own validation order.
func TestPolicy_ServiceAccountAccessForGroup(t *testing.T) {
	const (
		groupID   = "group-id"
		policyID  = "policy-id"
		groupPath = "root/mid/team"
		saID      = "sa-id"
	)

	tests := []struct {
		name string
		// saTRNPath is the service account's "<groupPath>/<name>", which is what its group path is
		// derived from. Empty means the lookup returns no service account at all.
		saTRNPath string
		// expectErrCode and expectErrContains are asserted together: the code alone would also match an
		// EInvalid raised by any other validation in these mutations, so the message pins the source.
		expectErrCode     errors.CodeType
		expectErrContains string
	}{
		{
			name:      "service account in the owning group",
			saTRNPath: groupPath + "/deployer",
		},
		{
			name:      "service account in a direct ancestor",
			saTRNPath: "root/mid/deployer",
		},
		{
			name:      "service account in the root ancestor",
			saTRNPath: "root/deployer",
		},
		{
			// A sibling subtree: visible to neither the owning group nor its members.
			name:              "service account in a sibling subtree",
			saTRNPath:         "root/other/deployer",
			expectErrCode:     errors.EInvalid,
			expectErrContains: "service account root/other/deployer is outside the scope of group " + groupPath,
		},
		{
			// A descendant of the owning group. Private visibility runs downwards, so the owning group
			// cannot reach into its own child.
			name:              "service account in a descendant",
			saTRNPath:         groupPath + "/sub/deployer",
			expectErrCode:     errors.EInvalid,
			expectErrContains: "is outside the scope of group " + groupPath,
		},
		{
			name:              "service account in an unrelated root group",
			saTRNPath:         "elsewhere/deployer",
			expectErrCode:     errors.EInvalid,
			expectErrContains: "is outside the scope of group " + groupPath,
		},
		{
			name:              "service account not found",
			saTRNPath:         "",
			expectErrCode:     errors.ENotFound,
			expectErrContains: "service account with ID " + saID + " not found",
		},
	}

	for _, test := range tests {
		test := test

		// setupServiceAccounts returns the lookup this test's service account resolves through. The
		// whole approver list is fetched in a single query, so the expectation matches on the ID filter
		// rather than on one ID at a time -- that also pins the batching, since a per-ID lookup would
		// not satisfy it.
		setupServiceAccounts := func(t *testing.T) *db.MockServiceAccounts {
			mockServiceAccounts := db.NewMockServiceAccounts(t)
			matchesIDFilter := mock.MatchedBy(func(input *db.GetServiceAccountsInput) bool {
				return input.Filter != nil && slices.Equal(input.Filter.ServiceAccountIDs, []string{saID})
			})

			// A missing service account is an ID the query matches nothing for, not an error.
			if test.saTRNPath == "" {
				mockServiceAccounts.On("GetServiceAccounts", mock.Anything, matchesIDFilter).
					Return(&db.ServiceAccountsResult{}, nil)
				return mockServiceAccounts
			}
			mockServiceAccounts.On("GetServiceAccounts", mock.Anything, matchesIDFilter).
				Return(&db.ServiceAccountsResult{
					ServiceAccounts: []models.ServiceAccount{{
						Metadata: models.ResourceMetadata{
							ID:  saID,
							TRN: trn.TypeServiceAccount.Build(test.saTRNPath),
						},
					}},
				}, nil)
			return mockServiceAccounts
		}

		t.Run("create: "+test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockGroups := db.NewMockGroups(t)
			mockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}, nil)

			mockPolicies := db.NewMockPolicies(t)
			mockTransactions := db.NewMockTransactions(t)
			mockResourceLimits := db.NewMockResourceLimits(t)

			// A rejected approver returns before the transaction is ever opened.
			if test.expectErrCode == "" {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("CreatePolicy", mock.Anything, mock.Anything).
					Return(&models.Policy{Metadata: models.ResourceMetadata{ID: policyID}}, nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
					Return(&db.PoliciesResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(0)}}, nil)
				mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
					Return(&models.ResourceLimit{Value: 100}, nil)
			}

			dbClient := db.Client{
				Groups:          mockGroups,
				Policies:        mockPolicies,
				ServiceAccounts: setupServiceAccounts(t),
				Transactions:    mockTransactions,
				ResourceLimits:  mockResourceLimits,
			}

			policy, err := newTestPolicyService(&dbClient).CreatePolicy(auth.WithCaller(ctx, mockCaller), &CreatePolicyInput{
				GroupID: groupID,
				Name:    "test-policy",
				Kind:    models.PolicyKindOPA,
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "my-group/my-package",
					Stage:                          models.RunTaskStageNamePostPlan,
					EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
				RequiredApprovals:        1,
				AllowedServiceAccountIDs: []string{saID},
			})

			if test.expectErrCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				assert.Contains(t, err.Error(), test.expectErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, policyID, policy.Metadata.ID)
		})

		t.Run("update: "+test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			existing := &models.Policy{
				Metadata: models.ResourceMetadata{ID: policyID},
				GroupID:  groupID,
				Name:     "test-policy",
				Kind:     models.PolicyKindOPA,
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "my-group/my-package",
					Stage:                          models.RunTaskStageNamePostPlan,
					EnforcementLevel:               models.PolicyEnforcementSoftMandatory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			}

			mockPolicies := db.NewMockPolicies(t)
			mockPolicies.On("GetPolicyByID", mock.Anything, policyID).Return(existing, nil)

			mockGroups := db.NewMockGroups(t)
			mockGroups.On("GetGroupByID", mock.Anything, groupID).
				Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}, nil)

			mockTransactions := db.NewMockTransactions(t)

			if test.expectErrCode == "" {
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("UpdatePolicy", mock.Anything, mock.Anything).
					Return(&models.Policy{Metadata: models.ResourceMetadata{ID: policyID}}, nil)
			}

			dbClient := db.Client{
				Groups:          mockGroups,
				Policies:        mockPolicies,
				ServiceAccounts: setupServiceAccounts(t),
				Transactions:    mockTransactions,
			}

			policy, err := newTestPolicyService(&dbClient).UpdatePolicy(auth.WithCaller(ctx, mockCaller), &UpdatePolicyInput{
				ID:                       policyID,
				RequiredApprovals:        ptr.Int(1),
				AllowedServiceAccountIDs: &[]string{saID},
			})

			if test.expectErrCode != "" {
				require.Error(t, err)
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				assert.Contains(t, err.Error(), test.expectErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, policyID, policy.Metadata.ID)
		})
	}
}

// TestPolicy_NoApproverServiceAccountsSkipsLookup pins that the check costs nothing when a policy has no
// service account approvers: a strict mock with no expectations fails if any lookup is attempted.
func TestPolicy_NoApproverServiceAccountsSkipsLookup(t *testing.T) {
	const groupID = "group-id"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockCaller := auth.NewMockCaller(t)
	mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockCaller.On("GetSubject").Return("mockSubject").Maybe()

	mockGroups := db.NewMockGroups(t)
	mockGroups.On("GetGroupByID", mock.Anything, groupID).
		Return(&models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: "root/team"}, nil)

	mockPolicies := db.NewMockPolicies(t)
	mockPolicies.On("CreatePolicy", mock.Anything, mock.Anything).
		Return(&models.Policy{Metadata: models.ResourceMetadata{ID: "policy-id"}}, nil)
	mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
		Return(&db.PoliciesResult{PageInfo: &pagination.PageInfo{TotalCount: pagination.StaticCount(0)}}, nil)

	mockTransactions := db.NewMockTransactions(t)
	mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, mockCaller), nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)

	mockResourceLimits := db.NewMockResourceLimits(t)
	mockResourceLimits.On("GetResourceLimit", mock.Anything, mock.Anything).
		Return(&models.ResourceLimit{Value: 100}, nil)

	dbClient := db.Client{
		Groups:   mockGroups,
		Policies: mockPolicies,
		// No expectations registered: an empty approver list must not reach the DB at all.
		ServiceAccounts: db.NewMockServiceAccounts(t),
		Transactions:    mockTransactions,
		ResourceLimits:  mockResourceLimits,
	}

	_, err := newTestPolicyService(&dbClient).CreatePolicy(auth.WithCaller(ctx, mockCaller), &CreatePolicyInput{
		GroupID: groupID,
		Name:    "test-policy",
		Kind:    models.PolicyKindOPA,
		OPAData: &models.OPAPolicyData{
			PackageSource:                  "my-group/my-package",
			Stage:                          models.RunTaskStageNamePostPlan,
			EnforcementLevel:               models.PolicyEnforcementAdvisory,
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
		},
	})
	require.NoError(t, err)
}

func newTestPolicyService(dbClient *db.Client) Service {
	testLogger, _ := logger.NewForTest()
	return NewService(testLogger, dbClient, limits.NewLimitChecker(dbClient))
}

func TestGetPolicies(t *testing.T) {
	group := &models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}

	tests := []struct {
		name            string
		input           *GetPoliciesInput
		setupMocks      func(*auth.MockCaller, *db.MockGroups, *db.MockPolicies)
		wantPolicyIDs   []string
		expectErrorCode errors.CodeType
	}{
		{
			name:            "nil group is invalid",
			input:           &GetPoliciesInput{},
			setupMocks:      func(_ *auth.MockCaller, _ *db.MockGroups, _ *db.MockPolicies) {},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:  "lists policies for the group",
			input: &GetPoliciesInput{Group: group},
			setupMocks: func(mockCaller *auth.MockCaller, _ *db.MockGroups, mockPolicies *db.MockPolicies) {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).Return(nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).Return(&db.PoliciesResult{
					Policies: []*models.Policy{
						{Metadata: models.ResourceMetadata{ID: "policy-1"}},
						{Metadata: models.ResourceMetadata{ID: "policy-2"}},
					},
				}, nil)
			},
			wantPolicyIDs: []string{"policy-1", "policy-2"},
		},
		{
			name:  "expands ancestor groups when inherited is requested",
			input: &GetPoliciesInput{Group: group, IncludeInherited: true},
			setupMocks: func(mockCaller *auth.MockCaller, mockGroups *db.MockGroups, mockPolicies *db.MockPolicies) {
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).Return(nil)
				mockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
					Groups: []models.Group{
						{Metadata: models.ResourceMetadata{ID: "group-1"}},
						{Metadata: models.ResourceMetadata{ID: "group-root"}},
					},
				}, nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).Return(&db.PoliciesResult{
					Policies: []*models.Policy{{Metadata: models.ResourceMetadata{ID: "policy-1"}}},
				}, nil)
			},
			wantPolicyIDs: []string{"policy-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockGroups := db.NewMockGroups(t)
			mockPolicies := db.NewMockPolicies(t)
			tt.setupMocks(mockCaller, mockGroups, mockPolicies)

			dbClient := &db.Client{Groups: mockGroups, Policies: mockPolicies}
			result, err := newTestPolicyService(dbClient).GetPolicies(auth.WithCaller(ctx, mockCaller), tt.input)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			gotIDs := make([]string, 0, len(result.Policies))
			for i := range result.Policies {
				gotIDs = append(gotIDs, result.Policies[i].Metadata.ID)
			}
			assert.Equal(t, tt.wantPolicyIDs, gotIDs)
		})
	}
}

func TestGetPolicyByID(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockPolicies)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the policy when found",
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(&models.Policy{
					GroupID:  "group-1",
					Metadata: models.ResourceMetadata{ID: "policy-1"},
				}, nil)
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).Return(nil)
			},
			wantID: "policy-1",
		},
		{
			name: "returns not found when the policy does not exist",
			setupMocks: func(_ *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			// The policy is gated as an inheritable resource, so this is a caller holding no policy view
			// rights anywhere in the owning group's subtree — not merely one outside the owning group.
			name: "returns the access error when the caller cannot reach the owning group's subtree",
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(&models.Policy{
					GroupID:  "group-1",
					Metadata: models.ResourceMetadata{ID: "policy-1"},
				}, nil)
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPolicies := db.NewMockPolicies(t)
			tt.setupMocks(mockCaller, mockPolicies)

			dbClient := &db.Client{Policies: mockPolicies}
			got, err := newTestPolicyService(dbClient).GetPolicyByID(auth.WithCaller(ctx, mockCaller), "policy-1")

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestGetPolicyByTRN(t *testing.T) {
	const policyTRN = "trn:policy:root/team/my-policy"

	tests := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockPolicies)
		wantID          string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the policy when found",
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicyByTRN", mock.Anything, policyTRN).Return(&models.Policy{
					GroupID:  "group-1",
					Metadata: models.ResourceMetadata{ID: "policy-1", TRN: policyTRN},
				}, nil)
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).Return(nil)
			},
			wantID: "policy-1",
		},
		{
			name: "returns not found when the policy does not exist",
			setupMocks: func(_ *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicyByTRN", mock.Anything, policyTRN).Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPolicies := db.NewMockPolicies(t)
			tt.setupMocks(mockCaller, mockPolicies)

			dbClient := &db.Client{Policies: mockPolicies}
			got, err := newTestPolicyService(dbClient).GetPolicyByTRN(auth.WithCaller(ctx, mockCaller), policyTRN)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)
		})
	}
}

func TestGetPoliciesByIDs(t *testing.T) {
	policyInGroup := func(name, group string) *models.Policy {
		return &models.Policy{
			Name:     name,
			Metadata: models.ResourceMetadata{ID: name, TRN: trn.TypePolicy.Build(group, name)},
		}
	}

	tests := []struct {
		name            string
		ids             []string
		setupMocks      func(*auth.MockCaller, *db.MockPolicies)
		wantIDs         []string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns the policies for the batch",
			ids:  []string{"policy-1", "policy-2"},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).Return(&db.PoliciesResult{
					Policies: []*models.Policy{
						policyInGroup("policy-1", "root/team"),
						// Same group as policy-1, so the access check sees one path, not two.
						policyInGroup("policy-2", "root/team"),
						policyInGroup("policy-3", "root"),
					},
				}, nil)
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).Return(nil)
			},
			wantIDs: []string{"policy-1", "policy-2", "policy-3"},
		},
		{
			// All-or-nothing: an unreadable policy fails the batch rather than being dropped from it.
			name: "returns the access error when a policy in the batch is unreadable",
			ids:  []string{"policy-1"},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).Return(&db.PoliciesResult{
					Policies: []*models.Policy{policyInGroup("policy-1", "root/team")},
				}, nil)
				mockCaller.On("RequireAccessToInheritableResource", mock.Anything, types.PolicyModelType, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			// No policies means no group paths, and the inheritable check rejects an empty constraint set —
			// so it must be skipped rather than called with nothing. Missing IDs are the loader's problem.
			name: "skips the access check when no policy matched",
			ids:  []string{"gone"},
			setupMocks: func(_ *auth.MockCaller, mockPolicies *db.MockPolicies) {
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).
					Return(&db.PoliciesResult{Policies: []*models.Policy{}}, nil)
			},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockPolicies := db.NewMockPolicies(t)
			tt.setupMocks(mockCaller, mockPolicies)

			dbClient := &db.Client{Policies: mockPolicies}
			got, err := newTestPolicyService(dbClient).GetPoliciesByIDs(auth.WithCaller(ctx, mockCaller), tt.ids)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			gotIDs := make([]string, 0, len(got))
			for i := range got {
				gotIDs = append(gotIDs, got[i].Metadata.ID)
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestGetWorkspaceAssignedPolicies(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockWorkspaces, *db.MockManagedIdentities, *db.MockGroups, *db.MockPolicies)
		wantNames       []string
		expectErrorCode errors.CodeType
	}{
		{
			name: "returns policies matching the workspace",
			setupMocks: func(
				mockCaller *auth.MockCaller,
				mockWorkspaces *db.MockWorkspaces,
				mockManagedIdentities *db.MockManagedIdentities,
				mockGroups *db.MockGroups,
				mockPolicies *db.MockPolicies,
			) {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewPolicyPermission, mock.Anything).Return(nil)
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(&models.Workspace{
					FullPath: "root/team/ws",
					Metadata: models.ResourceMetadata{ID: "ws-1"},
				}, nil)
				mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, "ws-1").
					Return([]models.ManagedIdentity{}, nil)
				mockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
					Groups: []models.Group{{Metadata: models.ResourceMetadata{ID: "group-1"}}},
				}, nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).Return(&db.PoliciesResult{
					Policies: []*models.Policy{{Name: "applies-to-all"}},
				}, nil)
			},
			wantNames: []string{"applies-to-all"},
		},
		{
			name: "returns not found when workspace is missing",
			setupMocks: func(
				mockCaller *auth.MockCaller,
				mockWorkspaces *db.MockWorkspaces,
				_ *db.MockManagedIdentities,
				_ *db.MockGroups,
				_ *db.MockPolicies,
			) {
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, "ws-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockGroups := db.NewMockGroups(t)
			mockPolicies := db.NewMockPolicies(t)
			tt.setupMocks(mockCaller, mockWorkspaces, mockManagedIdentities, mockGroups, mockPolicies)

			dbClient := &db.Client{
				Workspaces:        mockWorkspaces,
				ManagedIdentities: mockManagedIdentities,
				Groups:            mockGroups,
				Policies:          mockPolicies,
			}
			got, err := newTestPolicyService(dbClient).GetWorkspaceAssignedPolicies(
				auth.WithCaller(ctx, mockCaller),
				"ws-1",
			)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			gotNames := make([]string, 0, len(got))
			for i := range got {
				gotNames = append(gotNames, got[i].Name)
			}
			assert.Equal(t, tt.wantNames, gotNames)
		})
	}
}

func TestGetPoliciesReferencingManagedIdentity(t *testing.T) {
	tests := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockManagedIdentities, *db.MockPolicies)
		wantNames       []string
		expectErrorCode errors.CodeType
	}{
		{
			// Gated on policy view rights in the identity's own group, which a member of that group or
			// of an ancestor holds — and which therefore covers every subgroup the candidates come from.
			name: "returns the policies naming the identity",
			setupMocks: func(
				mockCaller *auth.MockCaller,
				mockManagedIdentities *db.MockManagedIdentities,
				mockPolicies *db.MockPolicies,
			) {
				mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, "mi-1").Return(&models.ManagedIdentity{
					Name:    "aws",
					GroupID: "group-1",
					Metadata: models.ResourceMetadata{
						ID:  "mi-1",
						TRN: trn.TypeManagedIdentity.Build("root/team", "aws"),
					},
				}, nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewPolicyPermission, mock.Anything).Return(nil)
				mockPolicies.On("GetPolicies", mock.Anything, mock.Anything).Return(&db.PoliciesResult{
					Policies: []*models.Policy{
						{
							Name: "names-the-identity",
							Scope: []*models.ScopeRule{
								{
									Type:    models.ScopeRuleTypeManagedIdentity,
									Action:  models.ScopeRuleActionInclude,
									Pattern: "root/team/aws",
								},
							},
						},
						{
							Name: "names-a-namespace",
							Scope: []*models.ScopeRule{
								{
									Type:    models.ScopeRuleTypeWorkspace,
									Action:  models.ScopeRuleActionInclude,
									Pattern: "root/team/*",
								},
							},
						},
						// Fires on every run under its group, this identity's included, but names no
						// identity — so it is not one of the identity's policies.
						{Name: "applies-to-all"},
					},
				}, nil)
			},
			wantNames: []string{"names-the-identity"},
		},
		{
			// The permission check needs the identity's group, so the load comes first and a missing
			// identity is reported before any access decision.
			name: "returns not found when the managed identity is missing",
			setupMocks: func(
				_ *auth.MockCaller,
				mockManagedIdentities *db.MockManagedIdentities,
				_ *db.MockPolicies,
			) {
				mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, "mi-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "returns the access error when the caller cannot view policies in the identity's group",
			setupMocks: func(
				mockCaller *auth.MockCaller,
				mockManagedIdentities *db.MockManagedIdentities,
				_ *db.MockPolicies,
			) {
				mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, "mi-1").Return(&models.ManagedIdentity{
					Name:    "aws",
					GroupID: "group-1",
					Metadata: models.ResourceMetadata{
						ID:  "mi-1",
						TRN: trn.TypeManagedIdentity.Build("root/team", "aws"),
					},
				}, nil)
				mockCaller.On("RequirePermission", mock.Anything, models.ViewPolicyPermission, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockPolicies := db.NewMockPolicies(t)
			tt.setupMocks(mockCaller, mockManagedIdentities, mockPolicies)

			// No Groups mock: the identity's policies come from one policy query over its group tree, so
			// a group lookup here would be a regression.
			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Policies:          mockPolicies,
			}
			got, err := newTestPolicyService(dbClient).GetPoliciesReferencingManagedIdentity(
				auth.WithCaller(ctx, mockCaller),
				"mi-1",
			)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			gotNames := make([]string, 0, len(got))
			for i := range got {
				gotNames = append(gotNames, got[i].Name)
			}
			assert.Equal(t, tt.wantNames, gotNames)
		})
	}
}

func TestUpdatePolicy(t *testing.T) {
	existingPolicy := func() *models.Policy {
		return &models.Policy{
			GroupID:  "group-1",
			Name:     "my-policy",
			Kind:     models.PolicyKindOPA,
			Metadata: models.ResourceMetadata{ID: "policy-1"},
			OPAData: &models.OPAPolicyData{
				PackageSource:                  "root/team/pkg",
				Stage:                          models.RunTaskStageNamePostPlan,
				EnforcementLevel:               models.PolicyEnforcementAdvisory,
				SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
			},
		}
	}

	// happyPathMocks sets up the full successful update path.
	happyPathMocks := func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies, mockGroups *db.MockGroups, mockTransactions *db.MockTransactions) {
		mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(existingPolicy(), nil)
		mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockGroups.On("GetGroupByID", mock.Anything, "group-1").
			Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
		mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(context.Background(), mockCaller), nil)
		mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
		mockTransactions.On("CommitTx", mock.Anything).Return(nil)
		mockPolicies.On("UpdatePolicy", mock.Anything, mock.Anything).Return(&models.Policy{
			Metadata: models.ResourceMetadata{ID: "policy-1"},
		}, nil)
	}

	tests := []struct {
		name                  string
		input                 *UpdatePolicyInput
		setupMocks            func(*auth.MockCaller, *db.MockPolicies, *db.MockGroups, *db.MockTransactions)
		wantID                string
		wantPackageSource     string
		wantStage             models.RunTaskStageName
		wantSpeculativeLevel  models.PolicyEnforcementLevel
		wantRequiredApprovals *int
		wantAllowedUserIDs    []string
		wantScopeLen          *int
		expectErrorCode       errors.CodeType
	}{
		{
			name:  "updates the policy",
			input: &UpdatePolicyInput{ID: "policy-1", Description: ptr.String("updated")},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies, mockGroups *db.MockGroups, mockTransactions *db.MockTransactions) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(existingPolicy(), nil)
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(context.Background(), mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("UpdatePolicy", mock.Anything, mock.Anything).Return(&models.Policy{
					Metadata: models.ResourceMetadata{ID: "policy-1"},
				}, nil)
			},
			wantID: "policy-1",
		},
		{
			name: "updates the package source",
			input: &UpdatePolicyInput{
				ID: "policy-1",
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "root/team/other-pkg",
					Stage:                          models.RunTaskStageNamePostPlan,
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			},
			setupMocks:        happyPathMocks,
			wantID:            "policy-1",
			wantPackageSource: "root/team/other-pkg",
		},
		{
			// The stage is mutable, so a supported value replaces the stored one.
			name: "updates the stage",
			input: &UpdatePolicyInput{
				ID: "policy-1",
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "root/team/pkg",
					Stage:                          models.RunTaskStageNamePrePlan,
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			},
			setupMocks: happyPathMocks,
			wantID:     "policy-1",
			wantStage:  models.RunTaskStageNamePrePlan,
		},
		{
			// The speculative-run enforcement level is mutable; previously the update path dropped it,
			// so it could never be changed after creation.
			name: "updates the speculative run enforcement level",
			input: &UpdatePolicyInput{
				ID: "policy-1",
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "root/team/pkg",
					Stage:                          models.RunTaskStageNamePostPlan,
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementHardMandatory,
				},
			},
			setupMocks:           happyPathMocks,
			wantID:               "policy-1",
			wantSpeculativeLevel: models.PolicyEnforcementHardMandatory,
		},
		{
			// A partial update that supplies only the description must leave the policy's scope and
			// approval configuration untouched, not overwrite them with zero values.
			name:  "partial update preserves scope and approval config",
			input: &UpdatePolicyInput{ID: "policy-1", Description: ptr.String("updated")},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies, mockGroups *db.MockGroups, mockTransactions *db.MockTransactions) {
				withApprovers := existingPolicy()
				withApprovers.OPAData.EnforcementLevel = models.PolicyEnforcementSoftMandatory
				withApprovers.RequiredApprovals = 2
				withApprovers.AllowedUserIDs = []string{"user-1", "user-2"}
				withApprovers.Scope = []*models.ScopeRule{
					{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "root/team/*"},
				}
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(withApprovers, nil)
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(context.Background(), mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("UpdatePolicy", mock.Anything, mock.Anything).
					Return(&models.Policy{Metadata: models.ResourceMetadata{ID: "policy-1"}}, nil)
			},
			wantID:                "policy-1",
			wantRequiredApprovals: ptr.Int(2),
			wantAllowedUserIDs:    []string{"user-1", "user-2"},
			wantScopeLen:          ptr.Int(1),
		},
		{
			// Nothing evaluates policies after apply, so storing that stage would leave a policy that
			// never fires.
			name: "rejects an unsupported stage",
			input: &UpdatePolicyInput{
				ID: "policy-1",
				OPAData: &models.OPAPolicyData{
					PackageSource:                  "root/team/pkg",
					Stage:                          models.RunTaskStageNamePostApply,
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies, mockGroups *db.MockGroups, _ *db.MockTransactions) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(existingPolicy(), nil)
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "rejects opaData without a package source",
			input: &UpdatePolicyInput{
				ID: "policy-1",
				OPAData: &models.OPAPolicyData{
					EnforcementLevel:               models.PolicyEnforcementAdvisory,
					SpeculativeRunEnforcementLevel: models.PolicyEnforcementAdvisory,
				},
			},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies, mockGroups *db.MockGroups, _ *db.MockTransactions) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(existingPolicy(), nil)
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:  "returns not found when the policy does not exist",
			input: &UpdatePolicyInput{ID: "policy-1"},
			setupMocks: func(_ *auth.MockCaller, mockPolicies *db.MockPolicies, _ *db.MockGroups, _ *db.MockTransactions) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "rejects an invalid scope rule",
			input: &UpdatePolicyInput{
				ID: "policy-1",
				Scope: &[]*models.ScopeRule{
					{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude},
				},
			},
			setupMocks: func(mockCaller *auth.MockCaller, mockPolicies *db.MockPolicies, mockGroups *db.MockGroups, _ *db.MockTransactions) {
				mockPolicies.On("GetPolicyByID", mock.Anything, "policy-1").Return(existingPolicy(), nil)
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()
			mockPolicies := db.NewMockPolicies(t)
			mockGroups := db.NewMockGroups(t)
			mockTransactions := db.NewMockTransactions(t)
			tt.setupMocks(mockCaller, mockPolicies, mockGroups, mockTransactions)

			dbClient := &db.Client{Policies: mockPolicies, Groups: mockGroups, Transactions: mockTransactions}
			got, err := newTestPolicyService(dbClient).UpdatePolicy(auth.WithCaller(ctx, mockCaller), tt.input)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantID, got.Metadata.ID)

			// The mock returns a bare policy, so what was actually written is read off the call itself.
			var updated *models.Policy
			for _, call := range mockPolicies.Calls {
				if call.Method == "UpdatePolicy" {
					updated = call.Arguments.Get(1).(*models.Policy)
				}
			}

			if tt.wantPackageSource != "" {
				require.NotNil(t, updated)
				assert.Equal(t, tt.wantPackageSource, updated.OPAData.PackageSource)
			}

			if tt.wantStage != "" {
				require.NotNil(t, updated)
				assert.Equal(t, tt.wantStage, updated.OPAData.Stage)
			}

			if tt.wantSpeculativeLevel != "" {
				require.NotNil(t, updated)
				assert.Equal(t, tt.wantSpeculativeLevel, updated.OPAData.SpeculativeRunEnforcementLevel)
			}

			if tt.wantRequiredApprovals != nil {
				require.NotNil(t, updated)
				assert.Equal(t, *tt.wantRequiredApprovals, updated.RequiredApprovals)
			}

			if tt.wantAllowedUserIDs != nil {
				require.NotNil(t, updated)
				assert.Equal(t, tt.wantAllowedUserIDs, updated.AllowedUserIDs)
			}

			if tt.wantScopeLen != nil {
				require.NotNil(t, updated)
				assert.Len(t, updated.Scope, *tt.wantScopeLen)
			}
		})
	}
}

func TestDeletePolicy(t *testing.T) {
	policy := &models.Policy{
		GroupID:  "group-1",
		Name:     "my-policy",
		Metadata: models.ResourceMetadata{ID: "policy-1"},
	}

	tests := []struct {
		name            string
		setupMocks      func(*auth.MockCaller, *db.MockGroups, *db.MockPolicies, *db.MockTransactions)
		expectErrorCode errors.CodeType
	}{
		{
			name: "deletes the policy",
			setupMocks: func(mockCaller *auth.MockCaller, mockGroups *db.MockGroups, mockPolicies *db.MockPolicies, mockTransactions *db.MockTransactions) {
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").
					Return(&models.Group{Metadata: models.ResourceMetadata{ID: "group-1"}, FullPath: "root/team"}, nil)
				mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(context.Background(), mockCaller), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockPolicies.On("DeletePolicy", mock.Anything, mock.Anything).Return(nil)
			},
		},
		{
			name: "returns not found when the owner group is missing",
			setupMocks: func(mockCaller *auth.MockCaller, mockGroups *db.MockGroups, _ *db.MockPolicies, _ *db.MockTransactions) {
				mockCaller.On("RequirePermission", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockGroups.On("GetGroupByID", mock.Anything, "group-1").Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()
			mockGroups := db.NewMockGroups(t)
			mockPolicies := db.NewMockPolicies(t)
			mockTransactions := db.NewMockTransactions(t)
			tt.setupMocks(mockCaller, mockGroups, mockPolicies, mockTransactions)

			dbClient := &db.Client{Groups: mockGroups, Policies: mockPolicies, Transactions: mockTransactions}
			err := newTestPolicyService(dbClient).DeletePolicy(auth.WithCaller(ctx, mockCaller), policy)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateScopeRules(t *testing.T) {
	tests := []struct {
		name            string
		rules           []*models.ScopeRule
		expectErrorCode errors.CodeType
	}{
		{
			name:  "no rules is valid",
			rules: nil,
		},
		{
			name: "a rule of each type with a glob path",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "acme/team/ws-1"},
				{Type: models.ScopeRuleTypeGroup, Action: models.ScopeRuleActionExclude, Pattern: "acme/other"},
				{Type: models.ScopeRuleTypeManagedIdentity, Action: models.ScopeRuleActionInclude, Pattern: "acme/prod-*"},
			},
		},
		{
			// The TRN each type accepts is the one naming the resource that type matches against.
			name: "a rule of each type with a TRN path of its own type",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "trn:workspace:acme/team/ws-1"},
				{Type: models.ScopeRuleTypeGroup, Action: models.ScopeRuleActionInclude, Pattern: "trn:group:acme/team"},
				{Type: models.ScopeRuleTypeManagedIdentity, Action: models.ScopeRuleActionInclude, Pattern: "trn:managed_identity:acme/prod-mi"},
			},
		},
		{
			name: "a TRN path may still carry a glob",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "trn:workspace:acme/*"},
			},
		},
		{
			name: "rejects an invalid action",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleAction("maybe"), Pattern: "acme/*"},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "rejects an invalid type",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleType("project"), Action: models.ScopeRuleActionInclude, Pattern: "acme/*"},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "rejects an empty path",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeGroup, Action: models.ScopeRuleActionInclude},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			// Stripping a group TRN would leave a path no workspace can have, so the rule would store
			// and never fire. The error is what tells the author to pick the Group type instead.
			name: "rejects a TRN naming the wrong resource type",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "trn:group:acme/team"},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "rejects a managed identity rule given a workspace TRN",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeManagedIdentity, Action: models.ScopeRuleActionInclude, Pattern: "trn:workspace:acme/team/ws-1"},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "rejects a TRN with no resource path",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "trn:workspace:"},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "rejects a malformed TRN",
			rules: []*models.ScopeRule{
				{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "trn:acme/team/ws-1"},
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScopeRules(tt.rules)

			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
