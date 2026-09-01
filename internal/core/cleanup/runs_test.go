package cleanup

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	nsErrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestRunPruner_prune(t *testing.T) {
	errDB := errors.New("db error")

	newPruner := func(mockRuns *db.MockRuns, mockAssessments *db.MockWorkspaceAssessments) *runPruner {
		return &runPruner{dbClient: &db.Client{Runs: mockRuns, WorkspaceAssessments: mockAssessments}}
	}

	// noAssessment stubs the workspace as having no assessment yet.
	noAssessment := func(m *db.MockWorkspaceAssessments) {
		m.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, mock.Anything).Return((*models.WorkspaceAssessment)(nil), nil)
	}

	noOnDelete := func(t *testing.T) onDeleteFn {
		t.Helper()
		return func(trns ...string) {
			t.Errorf("onDelete called unexpectedly with %v", trns)
		}
	}

	collectOnDelete := func(out *[]string) onDeleteFn {
		return func(trns ...string) {
			*out = append(*out, trns...)
		}
	}

	t.Run("ns is not a workspace: returns an internal error", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		p := newPruner(mockRuns, mockAssessments)

		err := p.prune(t.Context(), &models.Group{}, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, noOnDelete(t))

		require.Error(t, err)
		assert.Equal(t, nsErrors.EInternal, nsErrors.ErrorCode(err))
		// Mocks assert no unexpected calls on cleanup.
	})

	t.Run("no target statuses (all PROTECT rules): returns nil without GetRuns or assessment lookup", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		// No expectations set: matcher.empty() short-circuits before the loop, so neither GetRuns nor
		// GetWorkspaceAssessmentByWorkspaceID should ever be called.
		p := newPruner(mockRuns, mockAssessments)
		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyProtect}}},
		}, noOnDelete(t))

		require.NoError(t, err)
	})

	t.Run("error from GetRuns: error propagates, assessment never queried", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		// GetRuns is queried before the assessment, so a GetRuns failure means the assessment lookup
		// never happens — no expectation set on mockAssessments.
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return((*db.RunsResult)(nil), errDB)

		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, noOnDelete(t))

		require.Error(t, err)
	})

	t.Run("error from GetWorkspaceAssessmentByWorkspaceID: propagates after a successful GetRuns", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{run1},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, mock.Anything).Return((*models.WorkspaceAssessment)(nil), errDB)

		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, noOnDelete(t))

		require.Error(t, err)
	})

	t.Run("stale runs: DeleteRunBatch called with the stale run, onDelete called with its TRN", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		noAssessment(mockAssessments)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		// Two runs; keepMin=1 means first is kept, second is stale.
		run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}
		run2 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-2", TRN: "trn-run-2", Version: 4}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{run1, run2},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockRuns.On("DeleteRunBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteRunBatchInput) bool {
			return len(input.Runs) == 1 && input.Runs[0] == run2
		})).Return([]string{"run-2"}, nil)

		var got []string
		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, collectOnDelete(&got))

		require.NoError(t, err)
		assert.Equal(t, []string{"trn-run-2"}, got)
	})

	t.Run("assessment's completed run: never matched even if otherwise stale", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}
		protectedID := "run-protected"

		run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}
		protectedRun := &models.Run{Metadata: models.ResourceMetadata{ID: protectedID, TRN: "trn-protected", Version: 9}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{run1, protectedRun},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, mock.Anything).
			Return(&models.WorkspaceAssessment{RunID: &protectedID}, nil)
		// keepMin=1 keeps run1; protectedRun would otherwise be stale but is excluded by the matcher.

		onDeleteNotCalled := true
		var got []string
		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, func(trns ...string) {
			onDeleteNotCalled = false
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.True(t, onDeleteNotCalled)
		assert.Empty(t, got)
	})

	t.Run("non-stale runs: DeleteRunBatch not called, onDelete called with no TRNs", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		noAssessment(mockAssessments)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}
		run2 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-2", TRN: "trn-run-2", Version: 1}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{run1, run2},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)
		// No DeleteRunBatch expectation — mockery fails if it is called.

		onDeleteNotCalled := true
		var got []string
		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 5}}},
		}, func(trns ...string) {
			onDeleteNotCalled = false
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.True(t, onDeleteNotCalled, "onDelete should not be called when nothing is deleted")
		assert.Empty(t, got)
	})

	t.Run("run produced current state version: never appears in GetRuns results", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		noAssessment(mockAssessments)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		// The DB-level HasStateVersion=false filter means such runs never appear here.
		runA := &models.Run{Metadata: models.ResourceMetadata{ID: "run-a", TRN: "trn-run-a", Version: 1}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{runA},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)

		var got []string
		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, func(trns ...string) {
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.Empty(t, got, "no runs should be deleted when the run is within keepMin")
	})

	t.Run("ErrOptimisticLockError from DeleteRunBatch: not fatal, onDelete gets the survivors", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		noAssessment(mockAssessments)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		// keepMin=1: run1 (first matched) is kept; run2 and run3 are the 2 stale candidates.
		run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}
		run2 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-2", TRN: "trn-run-2", Version: 1}, Status: models.RunErrored}
		run3 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-3", TRN: "trn-run-3", Version: 1}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{run1, run2, run3},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)
		// DB reports run3 lost its lock race; run2 still deleted.
		mockRuns.On("DeleteRunBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteRunBatchInput) bool {
			return len(input.Runs) == 2
		})).Return([]string{"run-2"}, db.ErrOptimisticLockError)

		var got []string
		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, collectOnDelete(&got))

		require.NoError(t, err, "ErrOptimisticLockError from DeleteRunBatch must not propagate")
		assert.Equal(t, []string{"trn-run-2"}, got)
	})

	t.Run("non-OLE error from DeleteRunBatch: error propagates, onDelete not called", func(t *testing.T) {
		mockRuns := db.NewMockRuns(t)
		mockAssessments := db.NewMockWorkspaceAssessments(t)
		noAssessment(mockAssessments)
		p := newPruner(mockRuns, mockAssessments)

		ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

		run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}
		run2 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-2", TRN: "trn-run-2", Version: 2}, Status: models.RunErrored}

		mockRuns.On("GetRuns", mock.Anything, mock.Anything).Return(&db.RunsResult{
			Runs:     []*models.Run{run1, run2},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockRuns.On("DeleteRunBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteRunBatchInput) bool {
			return len(input.Runs) == 1 && input.Runs[0] == run2
		})).Return(nil, errDB)

		err := p.prune(t.Context(), ws, &models.CleanupPolicy{
			RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
		}, noOnDelete(t))

		require.Error(t, err)
	})
}

func TestRunPruner_prune_multiPage(t *testing.T) {
	orig := pageSleepInterval
	pageSleepInterval = 0
	defer func() { pageSleepInterval = orig }()

	mockRuns := db.NewMockRuns(t)
	mockAssessments := db.NewMockWorkspaceAssessments(t)
	// Queried once per page (twice total here) since the assessment lookup moved inside the loop.
	mockAssessments.On("GetWorkspaceAssessmentByWorkspaceID", mock.Anything, mock.Anything).Return((*models.WorkspaceAssessment)(nil), nil).Twice()
	p := &runPruner{dbClient: &db.Client{Runs: mockRuns, WorkspaceAssessments: mockAssessments}}

	ws := &models.Workspace{Metadata: models.ResourceMetadata{ID: "ws-1"}}

	run1 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1", TRN: "trn-run-1", Version: 1}, Status: models.RunErrored}
	run2 := &models.Run{Metadata: models.ResourceMetadata{ID: "run-2", TRN: "trn-run-2", Version: 3}, Status: models.RunErrored}

	cursor := "page-2-cursor"

	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.PaginationOptions.After == nil
	})).Return(&db.RunsResult{
		Runs: []*models.Run{run1},
		PageInfo: &pagination.PageInfo{
			HasNextPage: true,
			Cursor:      func(_ pagination.CursorPaginatable) (*string, error) { return &cursor, nil },
		},
	}, nil).Once()

	mockRuns.On("GetRuns", mock.Anything, mock.MatchedBy(func(in *db.GetRunsInput) bool {
		return in.PaginationOptions.After != nil && *in.PaginationOptions.After == cursor
	})).Return(&db.RunsResult{
		Runs:     []*models.Run{run2},
		PageInfo: &pagination.PageInfo{HasNextPage: false},
	}, nil).Once()

	mockRuns.On("DeleteRunBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteRunBatchInput) bool {
		return len(input.Runs) == 1 && input.Runs[0] == run2
	})).Return([]string{"run-2"}, nil).Once()

	var deletedTRNs []string
	err := p.prune(t.Context(), ws, &models.CleanupPolicy{
		RunPolicyData: &models.RunCleanupPolicyData{Rules: []*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}}},
	}, func(trns ...string) {
		deletedTRNs = append(deletedTRNs, trns...)
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"trn-run-2"}, deletedTRNs)
}

// ── runMatcher ────────────────────────────────────────────────────────────────

func TestNewRunMatcher(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	t.Run("PROTECT-only rules produce empty targetStatuses", func(t *testing.T) {
		m := newRunMatcher([]*models.RunCleanupRule{{Strategy: models.StrategyProtect}})
		assert.Empty(t, m.targetStatuses)
		assert.True(t, m.empty())
	})

	t.Run("COUNT rule with empty Status includes all candidate statuses", func(t *testing.T) {
		m := newRunMatcher([]*models.RunCleanupRule{{Strategy: models.StrategyCount, KeepMin: 1}})
		assert.ElementsMatch(t, models.CleanupCandidateRunStatuses, m.targetStatuses)
		assert.False(t, m.empty())
	})

	t.Run("COUNT rule with specific Status includes only that status", func(t *testing.T) {
		m := newRunMatcher([]*models.RunCleanupRule{
			{Strategy: models.StrategyCount, KeepMin: 1, Status: []models.RunStatus{models.RunErrored}},
		})
		assert.Equal(t, []models.RunStatus{models.RunErrored}, m.targetStatuses)
	})

	t.Run("speculative filter does not affect targetStatuses", func(t *testing.T) {
		m := newRunMatcher([]*models.RunCleanupRule{
			{Strategy: models.StrategyCount, KeepMin: 1, Speculative: boolPtr(true)},
		})
		assert.ElementsMatch(t, models.CleanupCandidateRunStatuses, m.targetStatuses)
	})

}

func TestRunMatcher_match(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	old := func() *time.Time { ts := time.Now().UTC().AddDate(0, 0, -35); return &ts }
	recent := func() *time.Time { ts := time.Now().UTC().AddDate(0, 0, -5); return &ts }

	type testCase struct {
		name           string
		rule           *models.RunCleanupRule
		run            *models.Run
		protectedRunID *string
		want           bool
	}

	protectedID := "run-protected"

	tests := []testCase{
		{
			name: "no rules — false",
			rule: nil,
			run:  &models.Run{Status: models.RunErrored},
			want: false,
		},
		{
			name: "status mismatch — rule skipped, false",
			rule: &models.RunCleanupRule{Strategy: models.StrategyCount, KeepMin: 0, Status: []models.RunStatus{models.RunCanceled}},
			run:  &models.Run{Status: models.RunErrored},
			want: false,
		},
		{
			name: "speculative mismatch — rule skipped, false",
			rule: &models.RunCleanupRule{Strategy: models.StrategyCount, KeepMin: 0, Speculative: boolPtr(true)},
			run:  &models.Run{Status: models.RunErrored, Apply: &models.Apply{}},
			want: false,
		},
		{
			name: "assessment mismatch — rule skipped, false",
			rule: &models.RunCleanupRule{Strategy: models.StrategyCount, KeepMin: 0, Assessment: boolPtr(true)},
			run:  &models.Run{Status: models.RunErrored, IsAssessmentRun: false},
			want: false,
		},
		{
			name: "PROTECT rule matches — false",
			rule: &models.RunCleanupRule{Strategy: models.StrategyProtect},
			run:  &models.Run{Status: models.RunErrored},
			want: false,
		},
		{
			name: "COUNT within keepMin — false",
			rule: &models.RunCleanupRule{Strategy: models.StrategyCount, KeepMin: 2},
			run:  &models.Run{Status: models.RunErrored},
			want: false,
		},
		{
			name: "AGE strategy — recent run kept, false",
			rule: &models.RunCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30},
			run:  &models.Run{Status: models.RunErrored, Metadata: models.ResourceMetadata{CreationTimestamp: recent()}},
			want: false,
		},
		{
			name: "AGE strategy — old run deleted, true",
			rule: &models.RunCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30},
			run:  &models.Run{Status: models.RunErrored, Metadata: models.ResourceMetadata{CreationTimestamp: old()}},
			want: true,
		},
		{
			name:           "protected run ID — never deleted even if AGE-stale",
			rule:           &models.RunCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30},
			run:            &models.Run{Metadata: models.ResourceMetadata{ID: protectedID, CreationTimestamp: old()}, Status: models.RunErrored},
			protectedRunID: &protectedID,
			want:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rules []*models.RunCleanupRule
			if tc.rule != nil {
				rules = []*models.RunCleanupRule{tc.rule}
			}
			m := newRunMatcher(rules)
			assert.Equal(t, tc.want, m.match(tc.run, tc.protectedRunID))
		})
	}

	t.Run("COUNT beyond keepMin — second call returns true", func(t *testing.T) {
		rule := &models.RunCleanupRule{Strategy: models.StrategyCount, KeepMin: 1}
		m := newRunMatcher([]*models.RunCleanupRule{rule})
		run := &models.Run{Status: models.RunErrored}
		assert.False(t, m.match(run, nil), "first run: kept=1, within keepMin")
		assert.True(t, m.match(run, nil), "second run: kept=2, beyond keepMin")
	})

	t.Run("protectedRunID can change between calls — each call uses whatever is passed in", func(t *testing.T) {
		rule := &models.RunCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30}
		m := newRunMatcher([]*models.RunCleanupRule{rule})
		run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-x", CreationTimestamp: old()}, Status: models.RunErrored}

		assert.True(t, m.match(run, nil), "not protected yet — matches")

		protectedID := "run-x"
		assert.False(t, m.match(run, &protectedID), "now protected — must not match")

		assert.True(t, m.match(run, nil), "protection lifted — matches again")
	})
}
