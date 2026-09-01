package models

import (
	"strings"
	"testing"

	"strconv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// ── CleanupRuleKind ──────────────────────────────────────────────────

func TestCleanupRuleKind_IsValid(t *testing.T) {
	for _, kind := range []CleanupRuleKind{
		CleanupRuleKindTerraformModules,
		CleanupRuleKindTerraformProviders,
		CleanupRuleKindRuns,
	} {
		assert.Truef(t, kind.IsValid(), "%s should be valid", kind)
	}
	assert.False(t, CleanupRuleKind("UNKNOWN").IsValid())
	assert.False(t, CleanupRuleKind("").IsValid())
}

func TestCleanupRuleKind_RequiresGroup(t *testing.T) {
	assert.True(t, CleanupRuleKindTerraformModules.RequiresGroup())
	assert.True(t, CleanupRuleKindTerraformProviders.RequiresGroup())
	assert.False(t, CleanupRuleKindRuns.RequiresGroup())
}

// ── CleanupGlob ───────────────────────────────────────────────────────────────

func TestCleanupGlob_Matches(t *testing.T) {
	type testCase struct {
		glob  CleanupGlob
		value string
		want  bool
	}
	testCases := []testCase{
		{glob: "", value: "anything", want: false}, // empty glob only matches the empty string
		{glob: "", value: "", want: true},
		{glob: "*", value: "anything", want: true},
		{glob: "*", value: "", want: true}, // * matches empty string
		{glob: "aws", value: "aws", want: true},
		{glob: "aws", value: "gcp", want: false},
		{glob: "aws", value: "", want: false}, // non-empty glob against empty value
		{glob: "aws-*", value: "aws-provider", want: true},
		{glob: "aws-*", value: "gcp-provider", want: false},
		{glob: "1.*", value: "1.2.3", want: true},
		{glob: "1.*", value: "2.0.0", want: false},
		{glob: "a?c", value: "a?c", want: true},
		{glob: "a?c", value: "abc", want: false},
		{glob: "[abc]", value: "[abc]", want: true},
		{glob: "[abc]", value: "a", want: false},
	}

	for _, test := range testCases {
		t.Run(string(test.glob)+"/"+test.value, func(t *testing.T) {
			assert.Equal(t, test.want, test.glob.Matches(test.value))
		})
	}
}

func TestCleanupGlob_Validate(t *testing.T) {
	t.Run("empty glob is rejected, it would never match anything", func(t *testing.T) {
		err := CleanupGlob("").Validate()
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("bare wildcard is valid and means match everything", func(t *testing.T) {
		require.NoError(t, CleanupGlob("*").Validate())
	})

	t.Run("valid glob pattern", func(t *testing.T) {
		require.NoError(t, CleanupGlob("aws-*").Validate())
	})

	t.Run("bracket characters are literal, not a malformed pattern", func(t *testing.T) {
		// go-glob has no pattern-syntax errors -- '*' is the only wildcard and everything else,
		// including brackets, is matched literally -- so there is no such thing as an invalid
		// glob pattern short of exceeding the length limit below.
		require.NoError(t, CleanupGlob("[invalid").Validate())
	})

	t.Run("pattern exceeding max length is rejected", func(t *testing.T) {
		long := CleanupGlob(strings.Repeat("a", cleanupMaxPatternLength+1))
		err := long.Validate()
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("pattern at exactly max length is valid", func(t *testing.T) {
		exact := CleanupGlob(strings.Repeat("a", cleanupMaxPatternLength))
		require.NoError(t, exact.Validate())
	})
}

// ── CleanupPolicy methods ────────────────────────────────────────────

func TestCleanupPolicy_GetID(t *testing.T) {
	p := &CleanupPolicy{Metadata: ResourceMetadata{ID: "policy-123"}}
	assert.Equal(t, "policy-123", p.GetID())
}

func TestCleanupPolicy_GetGlobalID(t *testing.T) {
	p := &CleanupPolicy{Metadata: ResourceMetadata{ID: "policy-123"}}
	gid := p.GetGlobalID()
	assert.NotEmpty(t, gid)
	// GlobalID encodes the model type + raw ID; it should be non-empty and different from the raw ID.
	assert.NotEqual(t, "policy-123", gid)
}

func TestCleanupPolicy_GetModelType(t *testing.T) {
	p := &CleanupPolicy{}
	assert.Equal(t, types.CleanupPolicyModelType, p.GetModelType())
}

func TestCleanupPolicy_NamespacePath(t *testing.T) {
	p := &CleanupPolicy{
		Metadata: ResourceMetadata{
			TRN: trn.TypeCleanupPolicy.Build("my-group/sub", string(CleanupRuleKindRuns)),
		},
	}
	assert.Equal(t, "my-group/sub", p.NamespacePath())
}

func TestCleanupPolicy_ResolveMetadata(t *testing.T) {
	p := &CleanupPolicy{Metadata: ResourceMetadata{ID: "p-1"}}

	v, err := p.ResolveMetadata("id")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, "p-1", *v)

	_, err = p.ResolveMetadata("unknown_field")
	assert.Error(t, err)
}

// ── CleanupPolicy.Validate ──────────────────────────────────────────

// ── validateCleanupStrategy ─────────────────────────────────────────────────────────

func validateCleanupStrategy(s CleanupStrategy, keepMin, deleteAfterDays int32) error {
	return s.validate(&keepMin, deleteAfterDays)
}

func TestValidateCleanupStrategy(t *testing.T) {
	type testCase struct {
		name            string
		strategy        CleanupStrategy
		keepMin         int32
		deleteAfterDays int32
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{name: "COUNT valid", strategy: StrategyCount, keepMin: 5},
		{name: "COUNT keepMin=0 invalid", strategy: StrategyCount, keepMin: 0, expectErrorCode: errors.EInvalid},
		{name: "COUNT keepMin=1 boundary valid", strategy: StrategyCount, keepMin: 1},
		{name: "COUNT keepMin=1000 boundary valid", strategy: StrategyCount, keepMin: cleanupMaxCountThreshold},
		{name: "COUNT keepMin>max invalid", strategy: StrategyCount, keepMin: cleanupMaxCountThreshold + 1, expectErrorCode: errors.EInvalid},
		{name: "COUNT with deleteAfterDays invalid", strategy: StrategyCount, keepMin: 1, deleteAfterDays: 10, expectErrorCode: errors.EInvalid},
		{name: "AGE valid", strategy: StrategyAge, keepMin: 1, deleteAfterDays: 30},
		{name: "AGE deleteAfterDays=7 boundary valid", strategy: StrategyAge, keepMin: 1, deleteAfterDays: cleanupMinAgeDays},
		{name: "AGE deleteAfterDays=6 below min invalid", strategy: StrategyAge, keepMin: 1, deleteAfterDays: cleanupMinAgeDays - 1, expectErrorCode: errors.EInvalid},
		{name: "AGE deleteAfterDays=3650 boundary valid", strategy: StrategyAge, keepMin: 1, deleteAfterDays: cleanupMaxAgeDays},
		{name: "AGE deleteAfterDays above max invalid", strategy: StrategyAge, keepMin: 1, deleteAfterDays: cleanupMaxAgeDays + 1, expectErrorCode: errors.EInvalid},
		{name: "AGE keepMin=0 invalid", strategy: StrategyAge, keepMin: 0, deleteAfterDays: 30, expectErrorCode: errors.EInvalid},
		{name: "AGE keepMin=1000 boundary valid", strategy: StrategyAge, keepMin: cleanupMaxCountThreshold, deleteAfterDays: 30},
		{name: "PROTECT valid", strategy: StrategyProtect},
		{name: "PROTECT with keepMin invalid", strategy: StrategyProtect, keepMin: 1, expectErrorCode: errors.EInvalid},
		{name: "PROTECT with deleteAfterDays invalid", strategy: StrategyProtect, deleteAfterDays: 10, expectErrorCode: errors.EInvalid},
		{name: "unknown strategy invalid", strategy: "UNKNOWN", expectErrorCode: errors.EInvalid},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := validateCleanupStrategy(test.strategy, test.keepMin, test.deleteAfterDays)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ── validateKindRules ─────────────────────────────────────────────────────────

func TestValidateCleanupRules(t *testing.T) {
	validRule := &TerraformModuleCleanupRule{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"}

	t.Run("matching kind with valid rules succeeds", func(t *testing.T) {
		err := validateCleanupRules(CleanupRuleKindTerraformModules,
			[]*TerraformModuleCleanupRule{validRule})
		require.NoError(t, err)
	})

	t.Run("matching kind with rules at exactly max limit succeeds", func(t *testing.T) {
		rules := make([]*TerraformModuleCleanupRule, cleanupMaxRules)
		for i := range rules {
			rules[i] = &TerraformModuleCleanupRule{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: CleanupGlob(strconv.Itoa(i)), SystemGlob: "*", VersionGlob: "*"}
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		require.NoError(t, err)
	})

	t.Run("matching kind with empty rules fails", func(t *testing.T) {
		err := validateCleanupRules(CleanupRuleKindTerraformModules,
			[]*TerraformModuleCleanupRule{})
		require.NoError(t, err)
	})

	t.Run("matching kind exceeding max rules fails", func(t *testing.T) {
		rules := make([]*TerraformModuleCleanupRule, cleanupMaxRules+1)
		for i := range rules {
			rules[i] = &TerraformModuleCleanupRule{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: CleanupGlob(strconv.Itoa(i)), SystemGlob: "*", VersionGlob: "*"}
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("matching kind with invalid rule fails", func(t *testing.T) {
		err := validateCleanupRules(CleanupRuleKindTerraformModules,
			[]*TerraformModuleCleanupRule{{Strategy: "BAD"}})
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("matching kind with a nil rule fails", func(t *testing.T) {
		err := validateCleanupRules(CleanupRuleKindTerraformModules,
			[]*TerraformModuleCleanupRule{nil})
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

}

// ── rule order / shadowing ───────────────────────────────────────────────────

func TestValidateCleanupRules_RuleOrder(t *testing.T) {
	t.Run("module: the same rules in the other order succeeds", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		require.NoError(t, err)
	})

	t.Run("module: disjoint system globs do not conflict even with an otherwise-shadowing version glob", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		require.NoError(t, err)
	})

	t.Run("module: an unscoped protect rule before another rule fails", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{
			{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("module: partial overlap that does not fully cover the later rule succeeds", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		require.NoError(t, err)
	})

	t.Run("provider: different literal name globs do not conflict", func(t *testing.T) {
		rules := []*TerraformProviderCleanupRule{}
		err := validateCleanupRules(CleanupRuleKindTerraformProviders, rules)
		require.NoError(t, err)
	})

	t.Run("run: an unconstrained rule before a constrained one fails", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1},
			{Strategy: StrategyCount, KeepMin: 1, Status: []RunStatus{RunPlannedAndFinished}},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("run: a status subset before its superset fails", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1, Status: []RunStatus{RunErrored, RunPlannedAndFinished}},
			{Strategy: StrategyCount, KeepMin: 1, Status: []RunStatus{RunErrored}},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("run: a status superset before its subset succeeds", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1, Status: []RunStatus{RunErrored}},
			{Strategy: StrategyCount, KeepMin: 1, Status: []RunStatus{RunErrored, RunPlannedAndFinished}},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		require.NoError(t, err)
	})

	t.Run("run: the same boolean condition set to different values does not conflict", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1, Speculative: new(true)},
			{Strategy: StrategyCount, KeepMin: 1, Speculative: new(false)},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		require.NoError(t, err)
	})

	t.Run("run: an unset condition on the later rule is not covered by a set condition on the earlier one", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1, Speculative: new(true)},
			{Strategy: StrategyCount, KeepMin: 1},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		require.NoError(t, err)
	})

	t.Run("run: rule 1 shadows rule 3 non-adjacently, skipping a rule 2 that conflicts with neither", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1, Speculative: new(true)},
			{Strategy: StrategyCount, KeepMin: 1, Speculative: new(false)},
			{Strategy: StrategyCount, KeepMin: 1, Speculative: new(true), Status: []RunStatus{RunErrored}},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("module: three rules with pairwise disjoint scopes never conflict", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		require.NoError(t, err)
	})

	t.Run("module: a broad PROTECT rule shadowing a later, narrower PROTECT rule still fails", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "*", VersionGlob: "*-rc.*"},
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("run: identical rules back to back are an exact-duplicate shadow", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 1, Status: []RunStatus{RunErrored}},
			{Strategy: StrategyCount, KeepMin: 5, Status: []RunStatus{RunErrored}},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("module: a PROTECT rule after a non-PROTECT rule fails even without full containment", func(t *testing.T) {
		// Neither COUNT rule fully contains the PROTECT rule (disjoint systemGlob literals vs "*"), so
		// the full-containment Shadows check alone would miss this; the ordering check must catch it.
		rules := []*TerraformModuleCleanupRule{
			{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "aws", VersionGlob: "*"},
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "*", VersionGlob: "1.*"},
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("module: multiple PROTECT rules in any relative order before all non-PROTECT rules succeeds", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "*", VersionGlob: "1.*"},
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "azure", VersionGlob: "2.*"},
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		require.NoError(t, err)
	})

	t.Run("module: a second PROTECT rule placed after a non-PROTECT rule fails", func(t *testing.T) {
		rules := []*TerraformModuleCleanupRule{
			{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: "*", SystemGlob: "*", VersionGlob: "*"},
			{Strategy: StrategyProtect, NameGlob: "*", SystemGlob: "azure", VersionGlob: "2.*"},
		}
		err := validateCleanupRules(CleanupRuleKindTerraformModules, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("provider: a PROTECT rule after a non-PROTECT rule fails even without full containment", func(t *testing.T) {
		rules := []*TerraformProviderCleanupRule{
			{Strategy: StrategyAge, DeleteAfterDays: 30, NameGlob: "*", VersionGlob: "*"},
			{Strategy: StrategyProtect, NameGlob: "*", VersionGlob: "1.*"},
		}
		err := validateCleanupRules(CleanupRuleKindTerraformProviders, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("run: a PROTECT rule after a non-PROTECT rule fails even without full containment", func(t *testing.T) {
		rules := []*RunCleanupRule{
			{Strategy: StrategyCount, KeepMin: 10, Speculative: new(true)},
			{Strategy: StrategyProtect, Status: []RunStatus{RunErrored}},
		}
		err := validateCleanupRules(CleanupRuleKindRuns, rules)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})
}

func TestGetCleanupPolicyForKind(t *testing.T) {
	runs := &CleanupPolicy{Metadata: ResourceMetadata{ID: "runs"}, Kind: CleanupRuleKindRuns}
	modules := &CleanupPolicy{Metadata: ResourceMetadata{ID: "modules"}, Kind: CleanupRuleKindTerraformModules}

	t.Run("returns the policy covering the kind", func(t *testing.T) {
		got, ok := GetCleanupPolicyForKind([]*CleanupPolicy{modules, runs}, CleanupRuleKindRuns)
		assert.True(t, ok)
		assert.Equal(t, runs, got)
	})

	t.Run("returns the first match when several cover the kind", func(t *testing.T) {
		closer := &CleanupPolicy{Metadata: ResourceMetadata{ID: "closer"}, Kind: CleanupRuleKindRuns}
		got, ok := GetCleanupPolicyForKind([]*CleanupPolicy{closer, runs}, CleanupRuleKindRuns)
		assert.True(t, ok)
		assert.Equal(t, closer, got)
	})

	t.Run("reports not found when no policy covers the kind", func(t *testing.T) {
		got, ok := GetCleanupPolicyForKind([]*CleanupPolicy{modules}, CleanupRuleKindRuns)
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("reports not found for an empty slice", func(t *testing.T) {
		got, ok := GetCleanupPolicyForKind(nil, CleanupRuleKindRuns)
		assert.False(t, ok)
		assert.Nil(t, got)
	})
}
