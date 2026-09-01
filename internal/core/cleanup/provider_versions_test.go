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

func TestProviderVersionPruner_pruneVersions(t *testing.T) {
	errDB := errors.New("db error")

	newPruner := func(mockPVs *db.MockTerraformProviderVersions) *providerVersionPruner {
		return &providerVersionPruner{dbClient: &db.Client{TerraformProviderVersions: mockPVs}}
	}

	noOnDelete := func(t *testing.T) onDeleteFn {
		t.Helper()
		return func(trns ...string) {
			t.Errorf("onDelete called unexpectedly with %v", trns)
		}
	}

	provider := &models.TerraformProvider{Metadata: models.ResourceMetadata{ID: "prov-1"}, Name: "my-provider"}
	ageData := &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", VersionGlob: "*"},
	}}

	t.Run("NameGlob mismatch: no DB calls, onDelete not called", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		mismatchRules := &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "other-*"},
		}}

		err := p.pruneVersions(t.Context(), provider, mismatchRules, noOnDelete(t))
		require.NoError(t, err)
	})

	t.Run("stale versions: DeleteProviderVersionBatch called, onDelete with TRNs", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		pv1 := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-1", TRN: "trn-pv-1", Version: 1, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
		}
		pv2 := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-2", TRN: "trn-pv-2", Version: 3, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()},
			SemanticVersion: "0.9.0",
		}

		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.ProviderVersionsResult{
			ProviderVersions: []models.TerraformProviderVersion{pv1, pv2},
			PageInfo:         &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockPVs.On("DeleteProviderVersionBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteProviderVersionBatchInput) bool {
			return len(input.ProviderVersions) == 1 && input.ProviderVersions[0].Metadata.ID == "pv-2"
		})).Return([]string{"pv-2"}, nil)

		var got []string
		err := p.pruneVersions(t.Context(), provider, ageData, func(trns ...string) {
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"trn-pv-2"}, got)
	})

	t.Run("ErrOptimisticLockError from DeleteProviderVersionBatch: not fatal, onDelete gets survivors", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		stale := func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()
		pv1 := models.TerraformProviderVersion{Metadata: models.ResourceMetadata{ID: "pv-1", TRN: "trn-pv-1", Version: 2, CreationTimestamp: stale}, SemanticVersion: "1.0.0"}
		pv2 := models.TerraformProviderVersion{Metadata: models.ResourceMetadata{ID: "pv-2", TRN: "trn-pv-2", Version: 6, CreationTimestamp: stale}, SemanticVersion: "0.9.0"}

		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.ProviderVersionsResult{
			ProviderVersions: []models.TerraformProviderVersion{pv1, pv2},
			PageInfo:         &pagination.PageInfo{HasNextPage: false},
		}, nil)
		// pv-1 loses its optimistic lock race.
		mockPVs.On("DeleteProviderVersionBatch", mock.Anything, mock.MatchedBy(func(input *db.DeleteProviderVersionBatchInput) bool {
			return len(input.ProviderVersions) == 2
		})).Return([]string{"pv-2"}, db.ErrOptimisticLockError)

		var got []string
		err := p.pruneVersions(t.Context(), provider, ageData, func(trns ...string) {
			got = append(got, trns...)
		})

		require.NoError(t, err, "ErrOptimisticLockError must not propagate")
		assert.Equal(t, []string{"trn-pv-2"}, got)
	})

	t.Run("Latest=true version is never deleted, even when otherwise stale by AGE", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		pv1 := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-1", TRN: "trn-pv-1", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
			Latest:          false,
		}
		pv2Latest := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-2", TRN: "trn-pv-2", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()},
			SemanticVersion: "0.9.0",
			Latest:          true,
		}

		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.ProviderVersionsResult{
			ProviderVersions: []models.TerraformProviderVersion{pv1, pv2Latest},
			PageInfo:         &pagination.PageInfo{HasNextPage: false},
		}, nil)
		// No DeleteProviderVersionBatch call at all.

		onDeleteNotCalled := true
		var got []string
		err := p.pruneVersions(t.Context(), provider, ageData, func(trns ...string) {
			onDeleteNotCalled = false
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.True(t, onDeleteNotCalled)
		assert.Empty(t, got)
	})

	t.Run("no stale versions: DeleteProviderVersionBatch not called", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		pv1 := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-1", TRN: "trn-pv-1", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
		}
		pv2 := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-2", TRN: "trn-pv-2", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()},
			SemanticVersion: "0.9.0",
		}

		bigKeepRules := &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 90, NameGlob: "*", VersionGlob: "*"},
		}}

		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.ProviderVersionsResult{
			ProviderVersions: []models.TerraformProviderVersion{pv1, pv2},
			PageInfo:         &pagination.PageInfo{HasNextPage: false},
		}, nil)

		onDeleteNotCalled := true
		var got []string
		err := p.pruneVersions(t.Context(), provider, bigKeepRules, func(trns ...string) {
			onDeleteNotCalled = false
			got = append(got, trns...)
		})

		require.NoError(t, err)
		assert.True(t, onDeleteNotCalled)
		assert.Empty(t, got)
	})

	t.Run("error from GetProviderVersions: propagates", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return((*db.ProviderVersionsResult)(nil), errDB)

		err := p.pruneVersions(t.Context(), provider, ageData, noOnDelete(t))
		require.Error(t, err)
	})

	t.Run("non-OLE error from DeleteProviderVersionBatch: propagates, onDelete not called", func(t *testing.T) {
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := newPruner(mockPVs)

		pv1 := models.TerraformProviderVersion{Metadata: models.ResourceMetadata{ID: "pv-1", TRN: "trn-pv-1", Version: 1, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()}, SemanticVersion: "1.0.0"}
		pv2 := models.TerraformProviderVersion{Metadata: models.ResourceMetadata{ID: "pv-2", TRN: "trn-pv-2", Version: 2, CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -35); return &t }()}, SemanticVersion: "0.9.0"}

		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.ProviderVersionsResult{
			ProviderVersions: []models.TerraformProviderVersion{pv1, pv2},
			PageInfo:         &pagination.PageInfo{HasNextPage: false},
		}, nil)
		mockPVs.On("DeleteProviderVersionBatch", mock.Anything, mock.Anything).Return(nil, errDB)

		err := p.pruneVersions(t.Context(), provider, ageData, noOnDelete(t))
		require.Error(t, err)
	})
}

func TestProviderVersionPruner_prune(t *testing.T) {
	policy := &models.CleanupPolicy{
		TerraformProviderPolicyData: &models.TerraformProviderCleanupPolicyData{Rules: []*models.TerraformProviderCleanupRule{
			{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "*", VersionGlob: "*"},
		}},
	}

	t.Run("ns is not a group — returns an internal error", func(t *testing.T) {
		mockProviders := db.NewMockTerraformProviders(t)
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := &providerVersionPruner{dbClient: &db.Client{TerraformProviders: mockProviders, TerraformProviderVersions: mockPVs}}

		err := p.prune(t.Context(), &models.Workspace{}, policy, func(...string) {
			t.Error("onDelete called unexpectedly")
		})

		require.Error(t, err)
		assert.Equal(t, nsErrors.EInternal, nsErrors.ErrorCode(err))
	})

	t.Run("error from GetProviders propagates", func(t *testing.T) {
		mockProviders := db.NewMockTerraformProviders(t)
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := &providerVersionPruner{dbClient: &db.Client{TerraformProviders: mockProviders, TerraformProviderVersions: mockPVs}}

		mockProviders.On("GetProviders", mock.Anything, mock.Anything).
			Return((*db.ProvidersResult)(nil), errors.New("db error"))

		group := &models.Group{Metadata: models.ResourceMetadata{ID: "g-1"}}
		err := p.prune(t.Context(), group, policy, func(...string) {})

		require.Error(t, err)
	})

	t.Run("calls pruneVersions for each provider in result", func(t *testing.T) {
		mockProviders := db.NewMockTerraformProviders(t)
		mockPVs := db.NewMockTerraformProviderVersions(t)
		p := &providerVersionPruner{dbClient: &db.Client{TerraformProviders: mockProviders, TerraformProviderVersions: mockPVs}}

		group := &models.Group{Metadata: models.ResourceMetadata{ID: "g-1"}}
		provider := models.TerraformProvider{Metadata: models.ResourceMetadata{ID: "prov-1"}, Name: "aws"}

		mockProviders.On("GetProviders", mock.Anything, mock.Anything).Return(&db.ProvidersResult{
			Providers: []models.TerraformProvider{provider},
			PageInfo:  &pagination.PageInfo{HasNextPage: false},
		}, nil)

		pv := models.TerraformProviderVersion{
			Metadata:        models.ResourceMetadata{ID: "pv-1", TRN: "trn-pv-1", CreationTimestamp: func() *time.Time { t := time.Now().AddDate(0, 0, -5); return &t }()},
			SemanticVersion: "1.0.0",
			ProviderID:      provider.Metadata.ID,
		}
		mockPVs.On("GetProviderVersions", mock.Anything, mock.Anything).Return(&db.ProviderVersionsResult{
			ProviderVersions: []models.TerraformProviderVersion{pv},
			PageInfo:         &pagination.PageInfo{HasNextPage: false},
		}, nil)

		err := p.prune(t.Context(), group, policy, func(...string) {})
		require.NoError(t, err)
	})
}

// ── providerVersionMatcher ────────────────────────────────────────────────────

func TestNewProviderVersionMatcher(t *testing.T) {
	rules := []*models.TerraformProviderCleanupRule{
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "my-*", VersionGlob: "*"},
		{Strategy: models.StrategyAge, DeleteAfterDays: 30, NameGlob: "other", VersionGlob: "*"},
	}

	t.Run("name matches — rule included", func(t *testing.T) {
		m := newProviderVersionMatcher(&models.TerraformProvider{Name: "my-provider"}, rules)
		assert.Len(t, m.rules, 1)
		assert.False(t, m.empty())
	})

	t.Run("name mismatch — no rules, empty", func(t *testing.T) {
		m := newProviderVersionMatcher(&models.TerraformProvider{Name: "unrelated"}, rules)
		assert.Empty(t, m.rules)
		assert.True(t, m.empty())
	})
}

func TestProviderVersionMatcher_match(t *testing.T) {
	old := func() *time.Time { ts := time.Now().UTC().AddDate(0, 0, -35); return &ts }
	recent := func() *time.Time { ts := time.Now().UTC().AddDate(0, 0, -5); return &ts }

	type testCase struct {
		name    string
		rule    *models.TerraformProviderCleanupRule
		version *models.TerraformProviderVersion
		want    bool
	}

	tests := []testCase{
		{
			name: "version glob mismatch — false",
			rule: &models.TerraformProviderCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "0.*"},
			version: &models.TerraformProviderVersion{
				SemanticVersion: "1.0.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: false,
		},
		{
			name: "PROTECT rule — false",
			rule: &models.TerraformProviderCleanupRule{Strategy: models.StrategyProtect, VersionGlob: "*"},
			version: &models.TerraformProviderVersion{
				SemanticVersion: "1.0.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: false,
		},
		{
			name: "AGE — recent version kept, false",
			rule: &models.TerraformProviderCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "*"},
			version: &models.TerraformProviderVersion{
				SemanticVersion: "1.0.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: recent()},
			},
			want: false,
		},
		{
			name: "AGE — old version deleted, true",
			rule: &models.TerraformProviderCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "*"},
			version: &models.TerraformProviderVersion{
				SemanticVersion: "0.9.0",
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: true,
		},
		{
			name: "Latest=true — never deleted regardless of strategy or age",
			rule: &models.TerraformProviderCleanupRule{Strategy: models.StrategyAge, DeleteAfterDays: 30, VersionGlob: "*"},
			version: &models.TerraformProviderVersion{
				SemanticVersion: "0.9.0",
				Latest:          true,
				Metadata:        models.ResourceMetadata{CreationTimestamp: old()},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &providerVersionMatcher{rules: []*models.TerraformProviderCleanupRule{tc.rule}, now: time.Now().UTC()}
			assert.Equal(t, tc.want, m.match(tc.version))
		})
	}
}
