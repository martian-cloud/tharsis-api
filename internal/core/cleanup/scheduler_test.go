package cleanup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	nsErrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// ── constructors ─────────────────────────────────────────────────────────────

func TestNewScheduler(t *testing.T) {
	dbClient := &db.Client{}
	testLogger, _ := logger.NewForTest()
	mockMaintenance := maintenance.NewMockMonitor(t)

	s := NewScheduler(dbClient, testLogger, mockMaintenance)

	require.NotNil(t, s)
	assert.Equal(t, dbClient, s.dbClient)
	assert.Equal(t, testLogger, s.logger)
	assert.Equal(t, mockMaintenance, s.maintenanceMonitor)

	runsP, ok := s.pruners[models.CleanupRuleKindRuns].(*runPruner)
	require.True(t, ok, "runs pruner should be *runPruner")
	assert.Equal(t, dbClient, runsP.dbClient)

	modulesP, ok := s.pruners[models.CleanupRuleKindTerraformModules].(*moduleVersionPruner)
	require.True(t, ok, "modules pruner should be *moduleVersionPruner")
	assert.Equal(t, dbClient, modulesP.dbClient)

	providersP, ok := s.pruners[models.CleanupRuleKindTerraformProviders].(*providerVersionPruner)
	require.True(t, ok, "providers pruner should be *providerVersionPruner")
	assert.Equal(t, dbClient, providersP.dbClient)
}

// ── fetchGroups ───────────────────────────────────────────────────────────────

func TestFetchGroups(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*db.MockGroups)
		wantCount      int
		wantLastCursor bool
		wantErr        bool
	}{
		{
			name: "single page — no next cursor",
			setupMocks: func(m *db.MockGroups) {
				m.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
					Groups:   []models.Group{{Metadata: models.ResourceMetadata{ID: "g1"}}, {Metadata: models.ResourceMetadata{ID: "g2"}}},
					PageInfo: &pagination.PageInfo{HasNextPage: false},
				}, nil)
			},
			wantCount: 2,
		},
		{
			name: "has next page — cursor returned",
			setupMocks: func(m *db.MockGroups) {
				m.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
					Groups: []models.Group{{Metadata: models.ResourceMetadata{ID: "g1"}}},
					PageInfo: &pagination.PageInfo{
						HasNextPage: true,
						Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
							c := "cursor-abc"
							return &c, nil
						},
					},
				}, nil)
			},
			wantCount:      1,
			wantLastCursor: true,
		},
		{
			name: "empty result — no cursor",
			setupMocks: func(m *db.MockGroups) {
				m.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
					Groups:   []models.Group{},
					PageInfo: &pagination.PageInfo{HasNextPage: false},
				}, nil)
			},
		},
		{
			name: "DB error propagates",
			setupMocks: func(m *db.MockGroups) {
				m.On("GetGroups", mock.Anything, mock.Anything).Return((*db.GroupsResult)(nil), nsErrors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockGroups := db.NewMockGroups(t)
			test.setupMocks(mockGroups)

			testLogger, _ := logger.NewForTest()
			s := &Scheduler{
				dbClient: &db.Client{Groups: mockGroups},
				logger:   testLogger,
			}
			result, err := s.fetchGroups(t.Context(), "some/group", nil)

			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.namespaces, test.wantCount)
			assert.Equal(t, test.wantLastCursor, result.pageInfo.HasNextPage)
		})
	}
}

// ── fetchWorkspaces ───────────────────────────────────────────────────────────

func TestFetchWorkspaces(t *testing.T) {
	testCases := []struct {
		name           string
		setupMocks     func(*db.MockWorkspaces)
		wantCount      int
		wantLastCursor bool
		wantErr        bool
	}{
		{
			name: "single page — no next cursor",
			setupMocks: func(m *db.MockWorkspaces) {
				m.On("GetWorkspaces", mock.Anything, mock.Anything).Return(&db.WorkspacesResult{
					Workspaces: []models.Workspace{{Metadata: models.ResourceMetadata{ID: "ws1"}}, {Metadata: models.ResourceMetadata{ID: "ws2"}}},
					PageInfo:   &pagination.PageInfo{HasNextPage: false},
				}, nil)
			},
			wantCount: 2,
		},
		{
			name: "has next page — cursor returned",
			setupMocks: func(m *db.MockWorkspaces) {
				m.On("GetWorkspaces", mock.Anything, mock.Anything).Return(&db.WorkspacesResult{
					Workspaces: []models.Workspace{{Metadata: models.ResourceMetadata{ID: "ws1"}}},
					PageInfo: &pagination.PageInfo{
						HasNextPage: true,
						Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
							c := "cursor-xyz"
							return &c, nil
						},
					},
				}, nil)
			},
			wantCount:      1,
			wantLastCursor: true,
		},
		{
			name: "empty result — no cursor",
			setupMocks: func(m *db.MockWorkspaces) {
				m.On("GetWorkspaces", mock.Anything, mock.Anything).Return(&db.WorkspacesResult{
					Workspaces: []models.Workspace{},
					PageInfo:   &pagination.PageInfo{HasNextPage: false},
				}, nil)
			},
		},
		{
			name: "DB error propagates",
			setupMocks: func(m *db.MockWorkspaces) {
				m.On("GetWorkspaces", mock.Anything, mock.Anything).Return((*db.WorkspacesResult)(nil), nsErrors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			test.setupMocks(mockWorkspaces)

			testLogger, _ := logger.NewForTest()
			s := &Scheduler{
				dbClient: &db.Client{Workspaces: mockWorkspaces},
				logger:   testLogger,
			}

			result, err := s.fetchWorkspaces(t.Context(), "some/ws", nil)

			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.namespaces, test.wantCount)
			assert.Equal(t, test.wantLastCursor, result.pageInfo.HasNextPage)
		})
	}
}

// ── finishSweep ───────────────────────────────────────────────────────────────

func TestFinishSweep(t *testing.T) {
	basePolicy := &models.CleanupPolicy{
		Metadata: models.ResourceMetadata{ID: "policy-id", Version: 1},
	}
	cursor := "page-cursor"

	// freshPolicy returns a new copy of basePolicy's fields so a subtest that mutates the policy
	// finishSweep is given does not leak that mutation into a later subtest sharing the same ID.
	freshPolicy := func() *models.CleanupPolicy {
		return &models.CleanupPolicy{Metadata: basePolicy.Metadata}
	}

	type testCase struct {
		name       string
		cursor     *string
		setupMocks func(*db.MockCleanupPolicies)
		wantErr    bool
	}

	testCases := []testCase{
		{
			name:   "nil cursor — SweepClaimedAt unchanged, LastSweepCompletedAt set",
			cursor: nil,
			setupMocks: func(m *db.MockCleanupPolicies) {
				m.On("GetCleanupPolicyByID", mock.Anything, basePolicy.Metadata.ID).Return(freshPolicy(), nil)
				m.On("UpdateCleanupPolicy", mock.Anything, mock.MatchedBy(func(p *models.CleanupPolicy) bool {
					return p.SweepCursor == nil && p.SweepClaimedAt == nil && p.LastSweepCompletedAt != nil
				})).Return(basePolicy, nil)
			},
		},
		{
			name:   "non-nil cursor — SweepClaimedAt cleared so policy is due again",
			cursor: &cursor,
			setupMocks: func(m *db.MockCleanupPolicies) {
				m.On("GetCleanupPolicyByID", mock.Anything, basePolicy.Metadata.ID).Return(freshPolicy(), nil)
				m.On("UpdateCleanupPolicy", mock.Anything, mock.MatchedBy(func(p *models.CleanupPolicy) bool {
					return p.SweepCursor != nil && *p.SweepCursor == cursor && p.SweepClaimedAt == nil && p.LastSweepCompletedAt == nil
				})).Return(basePolicy, nil)
			},
		},
		{
			name:   "policy deleted while sweeping — returns nil",
			cursor: nil,
			setupMocks: func(m *db.MockCleanupPolicies) {
				m.On("GetCleanupPolicyByID", mock.Anything, basePolicy.Metadata.ID).Return(nil, nil)
			},
		},
		{
			name:   "DB error propagates",
			cursor: nil,
			setupMocks: func(m *db.MockCleanupPolicies) {
				m.On("GetCleanupPolicyByID", mock.Anything, basePolicy.Metadata.ID).
					Return(nil, nsErrors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockPolicies := db.NewMockCleanupPolicies(t)
			test.setupMocks(mockPolicies)

			testLogger, _ := logger.NewForTest()
			s := &Scheduler{
				dbClient: &db.Client{CleanupPolicies: mockPolicies},
				logger:   testLogger,
			}

			err := s.finishSweep(t.Context(), basePolicy, test.cursor)

			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// ── sweepNamespaces ───────────────────────────────────────────────────────────

// sweepMocks bundles the dependencies used by TestSweepNamespaces. It keeps
// setupMocks signatures short and makes it clear which mocks are exercised by
// each case (unused ones are passed but never called — the strict mock catches
// any unexpected invocations on cleanup).
type sweepMocks struct {
	cleanupPolicies *db.MockCleanupPolicies
	workspaces      *db.MockWorkspaces
	groups          *db.MockGroups
	pruner          *mockPruner
}

func newSweepMocks(t *testing.T) *sweepMocks {
	t.Helper()
	return &sweepMocks{
		cleanupPolicies: db.NewMockCleanupPolicies(t),
		workspaces:      db.NewMockWorkspaces(t),
		groups:          db.NewMockGroups(t),
		pruner:          newMockPruner(t),
	}
}

func (m *sweepMocks) dbClient() *db.Client {
	return &db.Client{
		CleanupPolicies: m.cleanupPolicies,
		Workspaces:      m.workspaces,
		Groups:          m.groups,
	}
}

func TestSweepNamespaces(t *testing.T) {
	const (
		policyID = "policy-id"
		wsID     = "ws-id"
		groupID  = "g-id"

		// TRN format: trn:cleanup_policy:<namespace-path>/<kind>.
		// ParentPath() strips the last "/"-separated segment to give the namespace path.
		wsScopedRunsTRN    = "trn:cleanup_policy:g/ws/runs"           // NamespacePath() == "g/ws"
		groupScopedRunsTRN = "trn:cleanup_policy:g/runs"              // NamespacePath() == "g"
		groupScopedModsTRN = "trn:cleanup_policy:g/terraform_modules" // NamespacePath() == "g"
	)

	wsPtr := func(s string) *string { return &s }

	// Policy constructors return fresh copies so table cases cannot share mutable state.
	wsRunsPolicy := func() *models.CleanupPolicy {
		return &models.CleanupPolicy{
			Metadata:    models.ResourceMetadata{ID: policyID, TRN: wsScopedRunsTRN},
			Kind:        models.CleanupRuleKindRuns,
			WorkspaceID: wsPtr(wsID),
		}
	}
	groupRunsPolicy := func() *models.CleanupPolicy {
		return &models.CleanupPolicy{
			Metadata: models.ResourceMetadata{ID: policyID, TRN: groupScopedRunsTRN},
			Kind:     models.CleanupRuleKindRuns,
			GroupID:  wsPtr(groupID),
		}
	}
	groupModsPolicy := func() *models.CleanupPolicy {
		return &models.CleanupPolicy{
			Metadata: models.ResourceMetadata{ID: policyID, TRN: groupScopedModsTRN},
			Kind:     models.CleanupRuleKindTerraformModules,
			GroupID:  wsPtr(groupID),
		}
	}

	// noCursorPage builds a single-page PageInfo with a working cursor function.
	noCursorPage := func() *pagination.PageInfo {
		return &pagination.PageInfo{
			HasNextPage: false,
			Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
				c := "ns-cursor"
				return &c, nil
			},
		}
	}

	type testCase struct {
		name       string
		policy     *models.CleanupPolicy
		ctx        func() context.Context // nil → use t.Context()
		skipPruner bool                   // true for the "no registered pruner" case
		setupMocks func(*sweepMocks)
		wantErr    bool
	}

	testCases := []testCase{
		{
			name: "unregistered pruner kind — returns error, finishSweep not deferred",
			policy: &models.CleanupPolicy{
				// Returns before the defer is registered, so no DB calls are made.
				Kind: models.CleanupRuleKind("unsupported"),
			},
			skipPruner: true,
			setupMocks: func(_ *sweepMocks) {},
			wantErr:    true,
		},
		{
			name:   "policy deleted mid-sweep — re-read returns nil, exits cleanly",
			policy: wsRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				// nil returned on every call (main loop + finishSweep).
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return((*models.CleanupPolicy)(nil), nil)
			},
		},
		{
			name:   "policy disabled mid-sweep — exits cleanly",
			policy: wsRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				disabled := wsRunsPolicy()
				disabled.Disabled = true
				// First call (main loop): disabled policy → early return.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(disabled, nil).Once()
				// Second call (finishSweep): policy gone, nothing to update.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return((*models.CleanupPolicy)(nil), nil)
			},
		},
		{
			name:   "workspace-scoped runs policy: single page, prune succeeds",
			policy: wsRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				// FullPath matches wsScopedRunsTRN NamespacePath() → override check is skipped.
				ws := models.Workspace{Metadata: models.ResourceMetadata{ID: wsID}, FullPath: "g/ws"}
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(wsRunsPolicy(), nil)
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.Anything).
					Return(&db.WorkspacesResult{Workspaces: []models.Workspace{ws}, PageInfo: noCursorPage()}, nil)
				m.pruner.On("prune", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				// finishSweep (sweep completed, cursor = nil → LastSweepCompletedAt set).
				m.cleanupPolicies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(wsRunsPolicy(), nil)
			},
		},
		{
			name:   "group-scoped modules policy: single page, uses fetchGroups",
			policy: groupModsPolicy(),
			setupMocks: func(m *sweepMocks) {
				// FullPath "g/sub" differs from policyNamespacePath "g" → override check performed.
				child := models.Group{Metadata: models.ResourceMetadata{ID: "child-id"}, FullPath: "g/sub"}
				kind := models.CleanupRuleKindTerraformModules
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(groupModsPolicy(), nil)
				m.groups.On("GetGroups", mock.Anything, mock.Anything).
					Return(&db.GroupsResult{Groups: []models.Group{child}, PageInfo: noCursorPage()}, nil)
				// No child policies at "g/sub".
				m.cleanupPolicies.On("GetCleanupPolicies", mock.Anything, mock.MatchedBy(func(inp *db.GetCleanupPoliciesInput) bool {
					return inp.Kind != nil && *inp.Kind == kind
				})).Return([]*models.CleanupPolicy{}, nil)
				m.pruner.On("prune", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				// finishSweep.
				m.cleanupPolicies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(groupModsPolicy(), nil)
			},
		},
		{
			name:   "namespace overridden by child policy — prune skipped for overridden namespace",
			policy: groupRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				ws1 := models.Workspace{Metadata: models.ResourceMetadata{ID: "ws1"}, FullPath: "g/ws1"}
				ws2 := models.Workspace{Metadata: models.ResourceMetadata{ID: "ws2"}, FullPath: "g/ws2"}
				kind := models.CleanupRuleKindRuns
				// Child policy for ws2; TRN encodes NamespacePath() = "g/ws2".
				childPolicy := &models.CleanupPolicy{
					Metadata: models.ResourceMetadata{ID: "child-pol", TRN: "trn:cleanup_policy:g/ws2/runs"},
					Kind:     models.CleanupRuleKindRuns,
				}
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(groupRunsPolicy(), nil)
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.Anything).
					Return(&db.WorkspacesResult{Workspaces: []models.Workspace{ws1, ws2}, PageInfo: noCursorPage()}, nil)
				m.cleanupPolicies.On("GetCleanupPolicies", mock.Anything, mock.MatchedBy(func(inp *db.GetCleanupPoliciesInput) bool {
					return inp.Kind != nil && *inp.Kind == kind
				})).Return([]*models.CleanupPolicy{childPolicy}, nil)
				// prune called once for ws1 only; strict mock catches any spurious ws2 call.
				m.pruner.On("prune", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				// finishSweep.
				m.cleanupPolicies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(groupRunsPolicy(), nil)
			},
		},
		{
			name: "stale cursor (EInvalid) — cursor reset to nil and loop retried",
			policy: func() *models.CleanupPolicy {
				p := wsRunsPolicy()
				stale := "stale-cursor"
				p.SweepCursor = &stale
				return p
			}(),
			setupMocks: func(m *sweepMocks) {
				ws := models.Workspace{Metadata: models.ResourceMetadata{ID: wsID}, FullPath: "g/ws"}
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(wsRunsPolicy(), nil)
				// First call: stale cursor → EInvalid.
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.MatchedBy(func(inp *db.GetWorkspacesInput) bool {
					return inp.PaginationOptions.After != nil
				})).Return((*db.WorkspacesResult)(nil), nsErrors.New("stale", nsErrors.WithErrorCode(nsErrors.EInvalid))).Once()
				// Retry with nil cursor succeeds.
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.MatchedBy(func(inp *db.GetWorkspacesInput) bool {
					return inp.PaginationOptions.After == nil
				})).Return(&db.WorkspacesResult{Workspaces: []models.Workspace{ws}, PageInfo: noCursorPage()}, nil).Once()
				m.pruner.On("prune", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
				// finishSweep.
				m.cleanupPolicies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(wsRunsPolicy(), nil)
			},
		},
		{
			name:   "fetch error propagates",
			policy: wsRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				// Main loop: policy found, then fetch fails.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(wsRunsPolicy(), nil).Once()
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.Anything).
					Return((*db.WorkspacesResult)(nil), nsErrors.New("db error"))
				// finishSweep: policy gone (no UpdateCleanupPolicy).
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return((*models.CleanupPolicy)(nil), nil)
			},
			wantErr: true,
		},
		{
			name:   "override cache update error propagates",
			policy: groupRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				ws := models.Workspace{Metadata: models.ResourceMetadata{ID: wsID}, FullPath: "g/ws1"}
				// Main loop: policy found, fetch succeeds, GetCleanupPolicies fails.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(groupRunsPolicy(), nil).Once()
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.Anything).
					Return(&db.WorkspacesResult{Workspaces: []models.Workspace{ws}, PageInfo: noCursorPage()}, nil)
				m.cleanupPolicies.On("GetCleanupPolicies", mock.Anything, mock.Anything).
					Return(([]*models.CleanupPolicy)(nil), nsErrors.New("db error"))
				// finishSweep: policy gone.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return((*models.CleanupPolicy)(nil), nil)
			},
			wantErr: true,
		},
		{
			name:   "context canceled before namespace processing — returns nil",
			policy: wsRunsPolicy(),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			setupMocks: func(m *sweepMocks) {
				ws := models.Workspace{Metadata: models.ResourceMetadata{ID: wsID}, FullPath: "g/ws"}
				// Mocks return configured values regardless of context state.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(wsRunsPolicy(), nil)
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.Anything).
					Return(&db.WorkspacesResult{Workspaces: []models.Workspace{ws}, PageInfo: noCursorPage()}, nil)
				// prune NOT called: ctx.Err() != nil triggers early return in the namespace loop.
				// finishSweep uses context.WithoutCancel so it writes despite the cancelled ctx.
				m.cleanupPolicies.On("UpdateCleanupPolicy", mock.Anything, mock.Anything).
					Return(wsRunsPolicy(), nil)
			},
		},
		{
			name:   "cursor advance fails — error propagates",
			policy: wsRunsPolicy(),
			setupMocks: func(m *sweepMocks) {
				ws := models.Workspace{Metadata: models.ResourceMetadata{ID: wsID}, FullPath: "g/ws"}
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return(wsRunsPolicy(), nil).Once()
				m.workspaces.On("GetWorkspaces", mock.Anything, mock.Anything).
					Return(&db.WorkspacesResult{
						Workspaces: []models.Workspace{ws},
						PageInfo: &pagination.PageInfo{
							HasNextPage: false,
							Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
								return nil, nsErrors.New("cursor error")
							},
						},
					}, nil)
				// finishSweep: policy gone.
				m.cleanupPolicies.On("GetCleanupPolicyByID", mock.Anything, policyID).
					Return((*models.CleanupPolicy)(nil), nil)
			},
			wantErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			m := newSweepMocks(t)
			test.setupMocks(m)

			pruners := map[models.CleanupRuleKind]pruner{}
			if !test.skipPruner {
				pruners[test.policy.Kind] = m.pruner
			}

			testLogger, _ := logger.NewForTest()
			s := &Scheduler{
				dbClient: m.dbClient(),
				logger:   testLogger,
				pruners:  pruners,
			}

			ctx := t.Context()
			if test.ctx != nil {
				ctx = test.ctx()
			}
			err := s.sweepNamespaces(ctx, test.policy)

			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

// ── overrideCache ─────────────────────────────────────────────────────────────

func TestNewOverrideCache(t *testing.T) {
	mockPolicies := db.NewMockCleanupPolicies(t)
	policy := &models.CleanupPolicy{Metadata: models.ResourceMetadata{ID: "p-1"}}

	c := newOverrideCache(&db.Client{CleanupPolicies: mockPolicies}, policy)

	require.NotNil(t, c)
	assert.Empty(t, c.overriddenPrefixes)
	assert.Equal(t, policy, c.policy)
}

func TestOverrideCache_isOverridden(t *testing.T) {
	c := newOverrideCache(&db.Client{}, &models.CleanupPolicy{})
	c.addOverride("a/b")

	assert.True(t, c.isOverridden("a/b"), "exact match")
	assert.True(t, c.isOverridden("a/b/c"), "descendant")
	assert.False(t, c.isOverridden("a/bc"), "sibling prefix — not a path descendant")
	assert.False(t, c.isOverridden("a/c"), "unrelated sibling")
	assert.False(t, c.isOverridden("a"), "ancestor — not overridden")
}

func TestOverrideCache_updateFromPage(t *testing.T) {
	kind := models.CleanupRuleKindRuns
	// TRN encodes NamespacePath() = "g" (everything before last "/").
	policy := &models.CleanupPolicy{
		Metadata: models.ResourceMetadata{ID: "p-1", TRN: "trn:cleanup_policy:g/runs"},
		Kind:     kind,
	}

	t.Run("namespace path equals policyNamespacePath — skipped, GetCleanupPolicies not called", func(t *testing.T) {
		mockPolicies := db.NewMockCleanupPolicies(t)
		c := newOverrideCache(&db.Client{CleanupPolicies: mockPolicies}, policy)

		ws := &models.Workspace{FullPath: "g"} // path == "g" == policyNamespacePath
		require.NoError(t, c.updateFromPage(t.Context(), []namespace.Namespace{ws}))
	})

	t.Run("already overridden namespace — skipped, not re-queried", func(t *testing.T) {
		mockPolicies := db.NewMockCleanupPolicies(t)
		c := newOverrideCache(&db.Client{CleanupPolicies: mockPolicies}, policy)
		c.addOverride("g/ws1")

		ws := &models.Workspace{FullPath: "g/ws1"}
		require.NoError(t, c.updateFromPage(t.Context(), []namespace.Namespace{ws}))
		// strict mock: GetCleanupPolicies not expected, so any call would fail.
	})

	t.Run("unoverridden namespace — GetCleanupPolicies called, result stored", func(t *testing.T) {
		mockPolicies := db.NewMockCleanupPolicies(t)
		childPolicy := &models.CleanupPolicy{
			Metadata: models.ResourceMetadata{ID: "child-p", TRN: "trn:cleanup_policy:g/ws1/runs"},
			Kind:     kind,
		}
		mockPolicies.On("GetCleanupPolicies", mock.Anything, mock.MatchedBy(func(inp *db.GetCleanupPoliciesInput) bool {
			return len(inp.NamespacePaths) == 1 && inp.NamespacePaths[0] == "g/ws1"
		})).Return([]*models.CleanupPolicy{childPolicy}, nil)

		c := newOverrideCache(&db.Client{CleanupPolicies: mockPolicies}, policy)
		ws := &models.Workspace{FullPath: "g/ws1"}
		require.NoError(t, c.updateFromPage(t.Context(), []namespace.Namespace{ws}))

		assert.True(t, c.isOverridden("g/ws1"))
	})

	t.Run("GetCleanupPolicies error propagates", func(t *testing.T) {
		mockPolicies := db.NewMockCleanupPolicies(t)
		mockPolicies.On("GetCleanupPolicies", mock.Anything, mock.Anything).
			Return(([]*models.CleanupPolicy)(nil), nsErrors.New("db error"))

		c := newOverrideCache(&db.Client{CleanupPolicies: mockPolicies}, policy)
		ws := &models.Workspace{FullPath: "g/ws1"}
		require.Error(t, c.updateFromPage(t.Context(), []namespace.Namespace{ws}))
	})
}
