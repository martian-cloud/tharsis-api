package cleanuppolicy

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

func TestNewService(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	dbClient := &db.Client{}

	svc := NewService(testLogger, dbClient)

	require.NotNil(t, svc)
	inner, ok := svc.(*service)
	require.True(t, ok)
	assert.Equal(t, testLogger, inner.logger)
	assert.Equal(t, dbClient, inner.dbClient)
}

// samplePolicy is a reusable policy fixture for service tests.
var samplePolicy = &models.CleanupPolicy{
	Metadata: models.ResourceMetadata{
		ID:      "policy-id-1",
		Version: 1,
		TRN:     trn.TypeCleanupPolicy.Build("test-group", string(models.CleanupRuleKindTerraformModules)),
	},
	GroupID: new("group-id-1"),
	Kind:    models.CleanupRuleKindTerraformModules,
	TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
	}},
}

type cleanupPolicyMocks struct {
	caller         *auth.MockCaller
	policies       *db.MockCleanupPolicies
	groups         *db.MockGroups
	namespaces     *db.MockNamespaces
	workspaces     *db.MockWorkspaces
	transactions   *db.MockTransactions
	activityEvents *db.MockActivityEvents
}

func newCleanupPolicyMocks(t *testing.T) *cleanupPolicyMocks {
	return &cleanupPolicyMocks{
		caller:         auth.NewMockCaller(t),
		policies:       db.NewMockCleanupPolicies(t),
		groups:         db.NewMockGroups(t),
		namespaces:     db.NewMockNamespaces(t),
		workspaces:     db.NewMockWorkspaces(t),
		transactions:   db.NewMockTransactions(t),
		activityEvents: db.NewMockActivityEvents(t),
	}
}

func (m *cleanupPolicyMocks) dbClient() *db.Client {
	return &db.Client{
		CleanupPolicies: m.policies,
		Groups:          m.groups,
		Namespaces:      m.namespaces,
		Workspaces:      m.workspaces,
		Transactions:    m.transactions,
		ActivityEvents:  m.activityEvents,
	}
}

func TestGetCleanupPolicyByID(t *testing.T) {
	type testCase struct {
		name            string
		setupMocks      func(context.Context, *cleanupPolicyMocks)
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "no caller returns unauthorized",
			setupMocks: func(_ context.Context, _ *cleanupPolicyMocks) {
				// no caller set on ctx — auth.AuthorizeCaller will fail
			},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "policy not found",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "permission denied",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "success",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).
					Return(nil)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			mocks := newCleanupPolicyMocks(t)
			if test.expectErrorCode != errors.EUnauthorized {
				ctx = auth.WithCaller(ctx, mocks.caller)
			}
			test.setupMocks(ctx, mocks)

			svc := &service{dbClient: mocks.dbClient()}

			got, err := svc.GetCleanupPolicyByID(ctx, samplePolicy.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, samplePolicy, got)
		})
	}
}

func TestCreateCleanupPolicy(t *testing.T) {
	groupPath := "test-group"
	groupID := "group-id-1"

	type testCase struct {
		name              string
		input             *CreateCleanupPolicyInput // defaults to a group-legal module policy
		setupMocks        func(context.Context, *cleanupPolicyMocks)
		expectGroupID     *string
		expectWorkspaceID *string
		expectErrorCode   errors.CodeType
	}

	// Only runs can be cleaned up on a workspace; the registry kinds are group-only.
	workspaceInput := &CreateCleanupPolicyInput{
		NamespacePath: groupPath,
		Kind:          models.CleanupRuleKindRuns,
		RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 5}}},
	}

	testCases := []testCase{
		{
			name:            "no caller returns unauthorized",
			setupMocks:      func(_ context.Context, _ *cleanupPolicyMocks) {},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "permission denied",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequirePermission", mock.Anything,
					models.CreateCleanupPolicyPermission, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "namespace not found",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequirePermission", mock.Anything,
					models.CreateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.namespaces.On("GetNamespace", mock.Anything, groupPath).
					Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			// Defining a policy is independent of whether the setting is turned off, only whether
			// it's inherited from an ancestor.
			name: "success even though the namespace has scheduled cleanup turned off",
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequirePermission", mock.Anything,
					models.CreateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.namespaces.On("GetNamespace", mock.Anything, groupPath).
					Return(&db.NamespaceResult{Type: models.NamespaceTypeGroup, Group: &models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}}, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("CreateCleanupPolicy", mock.Anything, mock.Anything).
					Return(samplePolicy, nil)
			},
			expectGroupID: &groupID,
		},
		{
			// A root namespace has no ancestor to inherit from, so an unset setting isn't inheritance.
			name: "success for a root namespace with an unset scheduled cleanup setting",
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequirePermission", mock.Anything,
					models.CreateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.namespaces.On("GetNamespace", mock.Anything, groupPath).
					Return(&db.NamespaceResult{Type: models.NamespaceTypeGroup, Group: &models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}}, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("CreateCleanupPolicy", mock.Anything, mock.Anything).
					Return(samplePolicy, nil)
			},
			expectGroupID: &groupID,
		},
		{
			// The policy has to hang off the workspace column, not the group one.
			name:  "success for a workspace",
			input: workspaceInput,
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequirePermission", mock.Anything,
					models.CreateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.namespaces.On("GetNamespace", mock.Anything, groupPath).
					Return(&db.NamespaceResult{Type: models.NamespaceTypeWorkspace, Workspace: &models.Workspace{Metadata: models.ResourceMetadata{ID: "workspace-id"}, FullPath: groupPath}}, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("CreateCleanupPolicy", mock.Anything, mock.Anything).
					Return(samplePolicy, nil)
			},
			expectWorkspaceID: ptr.String("workspace-id"),
		},
		{
			name: "success",
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequirePermission", mock.Anything,
					models.CreateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.namespaces.On("GetNamespace", mock.Anything, groupPath).
					Return(&db.NamespaceResult{Type: models.NamespaceTypeGroup, Group: &models.Group{Metadata: models.ResourceMetadata{ID: groupID}, FullPath: groupPath}}, nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("CreateCleanupPolicy", mock.Anything, mock.Anything).
					Return(samplePolicy, nil)
			},
			expectGroupID: &groupID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			mocks := newCleanupPolicyMocks(t)
			if test.expectErrorCode != errors.EUnauthorized {
				ctx = auth.WithCaller(ctx, mocks.caller)
			}
			test.setupMocks(ctx, mocks)

			testLogger, _ := logger.NewForTest()
			svc := &service{logger: testLogger, dbClient: mocks.dbClient()}

			input := test.input
			if input == nil {
				input = &CreateCleanupPolicyInput{
					NamespacePath: groupPath,
					Kind:          models.CleanupRuleKindTerraformModules,
					TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
						{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
					}},
				}
			}

			got, err := svc.CreateCleanupPolicy(ctx, input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)

			created := mocks.policies.Calls[0].Arguments.Get(1).(*models.CleanupPolicy)
			assert.Equal(t, test.expectGroupID, created.GroupID)
			assert.Equal(t, test.expectWorkspaceID, created.WorkspaceID)
		})
	}
}

func TestUpdateCleanupPolicy(t *testing.T) {
	type testCase struct {
		name            string
		setupMocks      func(context.Context, *cleanupPolicyMocks)
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "no caller returns unauthorized",
			setupMocks:      func(_ context.Context, _ *cleanupPolicyMocks) {},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "policy not found",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "permission denied",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequirePermission", mock.Anything,
					models.UpdateCleanupPolicyPermission, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			// Modifying a policy is independent of whether the setting is turned off, only whether
			// it's inherited from an ancestor.
			name: "success even though the namespace has scheduled cleanup turned off",
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequirePermission", mock.Anything,
					models.UpdateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(samplePolicy, nil)
			},
		},
		{
			name: "success",
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequirePermission", mock.Anything,
					models.UpdateCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(samplePolicy, nil)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			mocks := newCleanupPolicyMocks(t)
			if test.expectErrorCode != errors.EUnauthorized {
				ctx = auth.WithCaller(ctx, mocks.caller)
			}
			test.setupMocks(ctx, mocks)

			testLogger, _ := logger.NewForTest()
			svc := &service{logger: testLogger, dbClient: mocks.dbClient()}

			got, err := svc.UpdateCleanupPolicy(ctx, &UpdateCleanupPolicyInput{
				ID: samplePolicy.Metadata.ID,
				TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
					{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
				}},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestDeleteCleanupPolicy(t *testing.T) {
	type testCase struct {
		name            string
		setupMocks      func(context.Context, *cleanupPolicyMocks)
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "no caller returns unauthorized",
			setupMocks:      func(_ context.Context, _ *cleanupPolicyMocks) {},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "policy not found",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(nil, nil)
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "permission denied",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequirePermission", mock.Anything,
					models.DeleteCleanupPolicyPermission, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "success",
			setupMocks: func(ctx context.Context, m *cleanupPolicyMocks) {
				m.policies.On("GetCleanupPolicyByID", mock.Anything, samplePolicy.Metadata.ID).
					Return(samplePolicy, nil)
				m.caller.On("RequirePermission", mock.Anything,
					models.DeleteCleanupPolicyPermission, mock.Anything).
					Return(nil)
				m.transactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.transactions.On("CommitTx", mock.Anything).Return(nil)
				m.policies.On("DeleteCleanupPolicy", mock.Anything, mock.Anything).
					Return(nil)
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			mocks := newCleanupPolicyMocks(t)
			if test.expectErrorCode != errors.EUnauthorized {
				ctx = auth.WithCaller(ctx, mocks.caller)
			}
			test.setupMocks(ctx, mocks)

			testLogger, _ := logger.NewForTest()
			svc := &service{logger: testLogger, dbClient: mocks.dbClient()}

			err := svc.DeleteCleanupPolicy(ctx, &DeleteCleanupPolicyInput{
				ID: samplePolicy.Metadata.ID,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetEffectiveCleanupPolicies(t *testing.T) {
	namespacePath := "parent/child"

	policyAtChild := &models.CleanupPolicy{
		Metadata: models.ResourceMetadata{
			ID:  "policy-child",
			TRN: trn.TypeCleanupPolicy.Build("parent/child", string(models.CleanupRuleKindTerraformModules)),
		},
		GroupID: ptr.String("child-group-id"),
		Kind:    models.CleanupRuleKindTerraformModules,
	}
	policyAtParent := &models.CleanupPolicy{
		Metadata: models.ResourceMetadata{
			ID:  "policy-parent",
			TRN: trn.TypeCleanupPolicy.Build("parent", string(models.CleanupRuleKindTerraformModules)),
		},
		GroupID: ptr.String("parent-group-id"),
		Kind:    models.CleanupRuleKindTerraformModules,
	}
	policyRunsAtParent := &models.CleanupPolicy{
		Metadata: models.ResourceMetadata{
			ID:  "policy-runs-parent",
			TRN: trn.TypeCleanupPolicy.Build("parent", string(models.CleanupRuleKindRuns)),
		},
		GroupID: ptr.String("parent-group-id"),
		Kind:    models.CleanupRuleKindRuns,
	}

	type testCase struct {
		name            string
		setupMocks      func(context.Context, *cleanupPolicyMocks)
		expectCount     int
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "no caller returns unauthorized",
			setupMocks:      func(_ context.Context, _ *cleanupPolicyMocks) {},
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name: "permission denied",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "child policy takes precedence over parent for same kind",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).Return(nil)
				m.policies.On("GetCleanupPolicies", mock.Anything, mock.Anything).
					Return([]*models.CleanupPolicy{policyAtChild, policyAtParent, policyRunsAtParent}, nil)
			},
			// policyAtChild wins over policyAtParent for modules; policyRunsAtParent included
			expectCount: 2,
		},
		{
			name: "single policy returned when only one namespace has a policy",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).Return(nil)
				m.policies.On("GetCleanupPolicies", mock.Anything, mock.Anything).
					Return([]*models.CleanupPolicy{policyAtParent}, nil)
			},
			expectCount: 1,
		},
		{
			name: "empty when no policies exist",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).Return(nil)
				m.policies.On("GetCleanupPolicies", mock.Anything, mock.Anything).
					Return([]*models.CleanupPolicy{}, nil)
			},
			expectCount: 0,
		},
		{
			name: "db error propagates",
			setupMocks: func(_ context.Context, m *cleanupPolicyMocks) {
				m.caller.On("RequireAccessToInheritableResource", mock.Anything,
					types.CleanupPolicyModelType, mock.Anything).Return(nil)
				m.policies.On("GetCleanupPolicies", mock.Anything, mock.Anything).
					Return(nil, errors.New("db error", errors.WithErrorCode(errors.EInternal)))
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			mocks := newCleanupPolicyMocks(t)
			if test.expectErrorCode != errors.EUnauthorized {
				ctx = auth.WithCaller(ctx, mocks.caller)
			}
			test.setupMocks(ctx, mocks)

			testLogger, _ := logger.NewForTest()
			svc := &service{logger: testLogger, dbClient: mocks.dbClient()}

			got, err := svc.GetEffectiveCleanupPolicies(ctx, namespacePath)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, test.expectCount)
		})
	}
}
