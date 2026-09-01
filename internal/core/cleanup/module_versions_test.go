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

func TestModuleVersionPruner_pruneVersions(t *testing.T) {
	errDB := errors.New("db error")

	newPruner := func(mockMVs *db.MockTerraformModuleVersions) *moduleVersionPruner {
		return &moduleVersionPruner{dbClient: &db.Client{TerraformModuleVersions: mockMVs}}
	}

	noOnDelete := func(t *testing.T) onDeleteFn {
		t.Helper()
		return func(trns ...string) {
			t.Errorf("onDelete called unexpectedly with %v", trns)
		}
	}

	module := &models.TerraformModule{Metadata: models.ResourceMetadata{ID: "mod-1"}, Name: "my-module", System: "aws"}
	ageData := &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
	}}

	t.Run("NameGlob mismatch: no DB calls, onDelete not called", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		mismatchRules := &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "other-*"},
		}}

		err := p.pruneVersions(t.Context(), module, mismatchRules, noOnDelete(t))
		require.NoError(t, err)
	})

	t.Run("stale versions: DeleteModuleVersionBatch called, onDelete with TRNs", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		mv1 := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-1", TRN: "trn-mv-1", Version: 1, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
		}
		mv2 := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-2", TRN: "trn-mv-2", Version: 3, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()},
			SemanticVersion: "0.9.0",
		}

		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{mv1, mv2},
			PageInfo:       &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockMVs.On("DeleteModuleVersionBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteModuleVersionBatchInput) bool {
			return len(input.ModuleVersions) == 1 && input.ModuleVersions[0].Metadata.ID == "mv-2"
		})).Return([]string{"mv-2"}, nil)

		var got []string
		err := p.pruneVersions(t.Context(), module, ageData, func(trns ...string) {
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"trn-mv-2"}, got)
	})

	t.Run("ErrOptimisticLockError from DeleteModuleVersionBatch: not fatal, onDelete gets survivors", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		stale := func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()
		mv1 := models.TerraformModuleVersion{Metadata: models.ResourceMetadata{ID: "mv-1", TRN: "trn-mv-1", Version: 2, CreationTimestamp: stale}, SemanticVersion: "1.0.0"}
		mv2 := models.TerraformModuleVersion{Metadata: models.ResourceMetadata{ID: "mv-2", TRN: "trn-mv-2", Version: 5, CreationTimestamp: stale}, SemanticVersion: "0.9.0"}

		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{mv1, mv2},
			PageInfo:       &pagination.PageInfo{HasNextPage: false},
		}, nil)
		// mv-1 loses its optimistic lock race (e.g. re-uploaded between match and delete).
		mockMVs.On("DeleteModuleVersionBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteModuleVersionBatchInput) bool {
			return len(input.ModuleVersions) == 2
		})).Return([]string{"mv-2"}, db.ErrOptimisticLockError)

		var got []string
		err := p.pruneVersions(t.Context(), module, ageData, func(trns ...string) {
			got = append(got, trns...)
		})

		require.NoError(t, err, "ErrOptimisticLockError must not propagate")
		assert.Equal(t, []string{"trn-mv-2"}, got)
	})

	t.Run("Latest=true version is never deleted, even when otherwise stale by AGE", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		mv1 := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-1", TRN: "trn-mv-1", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
			Latest:          false,
		}
		mv2Latest := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-2", TRN: "trn-mv-2", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()},
			SemanticVersion: "0.9.0",
			Latest:          true,
		}

		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{mv1, mv2Latest},
			PageInfo:       &pagination.PageInfo{HasNextPage: false},
		}, nil)
		// No DeleteModuleVersionBatch call at all.

		onDeleteNotCalled := true
		var got []string
		err := p.pruneVersions(t.Context(), module, ageData, func(trns ...string) {
			onDeleteNotCalled = false
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.True(t, onDeleteNotCalled)
		assert.Empty(t, got)
	})

	t.Run("no stale versions: DeleteModuleVersionBatch not called", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		mv1 := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-1", TRN: "trn-mv-1", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
		}
		mv2 := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-2", TRN: "trn-mv-2", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()},
			SemanticVersion: "0.9.0",
		}

		bigKeepRules := &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 90, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}}

		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{mv1, mv2},
			PageInfo:       &pagination.PageInfo{HasNextPage: false},
		}, nil)

		onDeleteNotCalled := true
		var got []string
		err := p.pruneVersions(t.Context(), module, bigKeepRules, func(trns ...string) {
			onDeleteNotCalled = false
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.True(t, onDeleteNotCalled)
		assert.Empty(t, got)
	})

	t.Run("error from GetModuleVersions: propagates", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return((*db.ModuleVersionsResult)(nil), errDB)

		err := p.pruneVersions(t.Context(), module, ageData, noOnDelete(t))
		require.Error(t, err)
	})

	t.Run("non-OLE error from DeleteModuleVersionBatch: propagates, onDelete not called", func(t *testing.T) {
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := newPruner(mockMVs)

		mv1 := models.TerraformModuleVersion{Metadata: models.ResourceMetadata{ID: "mv-1", TRN: "trn-mv-1", Version: 1, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()}, SemanticVersion: "1.0.0"}
		mv2 := models.TerraformModuleVersion{Metadata: models.ResourceMetadata{ID: "mv-2", TRN: "trn-mv-2", Version: 2, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()}, SemanticVersion: "0.9.0"}

		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{mv1, mv2},
			PageInfo:       &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockMVs.On("DeleteModuleVersionBatch", mock.Anything, mock.Anything).Return(nil, errDB)

		err := p.pruneVersions(t.Context(), module, ageData, noOnDelete(t))
		require.Error(t, err)
	})
}

func TestModuleVersionPruner_prune(t *testing.T) {
	policy := &models.CleanupPolicy{
		TerraformModulePolicyData: &models.TerraformModuleCleanupPolicyData{Rules: []*models.TerraformModuleCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}},
	}

	t.Run("ns is not a group — returns an internal error", func(t *testing.T) {
		mockModules := db.NewMockTerraformModules(t)
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := &moduleVersionPruner{dbClient: &db.Client{TerraformModules: mockModules, TerraformModuleVersions: mockMVs}}

		err := p.prune(t.Context(), &models.Workspace{}, policy, func(...string) {
			t.Error("onDelete called unexpectedly")
		})

		require.Error(t, err)
		assert.Equal(t, nsErrors.EInternal, nsErrors.ErrorCode(err))
	})

	t.Run("error from GetModules propagates", func(t *testing.T) {
		mockModules := db.NewMockTerraformModules(t)
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := &moduleVersionPruner{dbClient: &db.Client{TerraformModules: mockModules, TerraformModuleVersions: mockMVs}}

		mockModules.On("GetModules", mock.Anything, mock.Anything).Return((*db.ModulesResult)(nil), errors.New("db error"))

		group := &models.Group{Metadata: models.ResourceMetadata{ID: "g-1"}}
		err := p.prune(t.Context(), group, policy, func(...string) {})

		require.Error(t, err)
	})

	t.Run("calls pruneVersions for each module in result", func(t *testing.T) {
		mockModules := db.NewMockTerraformModules(t)
		mockMVs := db.NewMockTerraformModuleVersions(t)
		p := &moduleVersionPruner{dbClient: &db.Client{TerraformModules: mockModules, TerraformModuleVersions: mockMVs}}

		group := &models.Group{Metadata: models.ResourceMetadata{ID: "g-1"}}
		mod := models.TerraformModule{Metadata: models.ResourceMetadata{ID: "m-1"}, Name: "my-module", System: "aws"}

		mockModules.On("GetModules", mock.Anything, mock.Anything).Return(&db.ModulesResult{
			Modules:  []models.TerraformModule{mod},
			PageInfo: &pagination.PageInfo{HasNextPage: false},
		}, nil)

		mv := models.TerraformModuleVersion{
			Metadata:        models.ResourceMetadata{ID: "mv-1", TRN: "trn-mv-1", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
			ModuleID:        mod.Metadata.ID,
		}
		mockMVs.On("GetModuleVersions", mock.Anything, mock.Anything).Return(&db.ModuleVersionsResult{
			ModuleVersions: []models.TerraformModuleVersion{mv},
			PageInfo:       &pagination.PageInfo{HasNextPage: false},
		}, nil)

		err := p.prune(t.Context(), group, policy, func(...string) {})
		require.NoError(t, err)
	})
}

// ── moduleVersionMatcher ──────────────────────────────────────────────────────

func TestNewModuleVersionMatcher(t *testing.T) {
	rules := []*models.TerraformModuleCleanupRule{
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "my-*", SystemGlob: "aws", VersionGlob: "*"},
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "gcp", VersionGlob: "*"},
	}

	t.Run("name and system match — rule included", func(t *testing.T) {
		m := newModuleVersionMatcher(&models.TerraformModule{Name: "my-mod", System: "aws"}, rules)
		assert.Len(t, m.rules, 1)
		assert.False(t, m.empty())
	})

	t.Run("name mismatch — no rules, empty", func(t *testing.T) {
		m := newModuleVersionMatcher(&models.TerraformModule{Name: "other", System: "aws"}, rules)
		assert.Empty(t, m.rules)
		assert.True(t, m.empty())
	})

	t.Run("system mismatch — no rules, empty", func(t *testing.T) {
		m := newModuleVersionMatcher(&models.TerraformModule{Name: "my-mod", System: "azure"}, rules)
		assert.Empty(t, m.rules)
		assert.True(t, m.empty())
	})
}

func TestModuleVersionMatcher_match(t *testing.T) {
	old := func() *time.Time { ts := time.Now().UTC().AddDate(0, 0, -35); return &ts }
	recent := func() *time.Time { ts := time.Now().UTC().AddDate(0, 0, -5); return &ts }

	type testCase struct {
		name    string
		rule    *models.TerraformModuleCleanupRule
		version *models.TerraformModuleVersion
		want    bool
	}

	tests := []testCase{
		{
			name: "version glob mismatch — false",
			rule: &models.TerraformModuleCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "0.*"},
			version: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: false,
		},
		{
			name: "PROTECT rule — false",
			rule: &models.TerraformModuleCleanupRule{Strategy: models.StrategyProtect, VersionGlob: "*"},
			version: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: false,
		},
		{
			name: "AGE — recent version kept, false",
			rule: &models.TerraformModuleCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "*"},
			version: &models.TerraformModuleVersion{
				SemanticVersion: "1.0.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: recent()},
			},
			want: false,
		},
		{
			name: "AGE — old version deleted, true",
			rule: &models.TerraformModuleCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "*"},
			version: &models.TerraformModuleVersion{
				SemanticVersion: "0.9.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: true,
		},
		{
			name: "Latest=true — never deleted regardless of strategy or age",
			rule: &models.TerraformModuleCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "*"},
			version: &models.TerraformModuleVersion{
				SemanticVersion: "0.9.0",
				Latest:          true,
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &moduleVersionMatcher{rules: []*models.TerraformModuleCleanupRule{tc.rule}, now: time.Now().UTC()}
			assert.Equal(t, tc.want, m.match(tc.version))
		})
	}
}
