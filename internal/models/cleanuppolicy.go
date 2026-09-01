package models

import (
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/glob"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

var _ Model = (*CleanupPolicy)(nil)

const (
	cleanupMaxCountThreshold = 1000
	cleanupMinAgeDays        = 7
	cleanupMaxAgeDays        = 3650 // ~ 10 years.
	cleanupMaxRules          = 20
	cleanupMaxPatternLength  = 50
)

// CleanupCandidateRunStatuses are the only terminal run statuses that are valid cleanup candidates.
var CleanupCandidateRunStatuses = []RunStatus{
	RunPlannedAndFinished,
	RunErrored,
	RunCanceled,
	RunDiscarded,
}

// CleanupStrategy selects the cleanup strategy applied to a rule's candidates.
type CleanupStrategy string

const (
	// StrategyCount keeps the newest KeepMin candidates and deletes the rest, so KeepMin is an exact
	// ceiling on what a COUNT rule keeps.
	StrategyCount CleanupStrategy = "count"
	// StrategyAge deletes candidates older than DeleteAfterDays days, always keeping the newest KeepMin, so
	// KeepMin is a floor under what an AGE rule keeps.
	StrategyAge CleanupStrategy = "age"
	// StrategyProtect never deletes the candidates it matches, shadowing every rule below it.
	StrategyProtect CleanupStrategy = "protect"
)

// validate returns an error if the strategy/keepMin/deleteAfterDays combination is invalid.
// keepMin is a pointer: nil means the caller has no floor requirement (e.g. version rules), a non-nil
// value is validated against the strategy's constraints.
func (s CleanupStrategy) validate(keepMin *int32, deleteAfterDays int32) error {
	switch s {
	case StrategyCount:
		if keepMin == nil {
			return errors.New("keepMin is required for a COUNT strategy", errors.WithErrorCode(errors.EInvalid))
		}

		if *keepMin < 1 || *keepMin > cleanupMaxCountThreshold {
			return errors.New("keepMin must be between 1 and %d for a COUNT strategy, got %d", cleanupMaxCountThreshold, *keepMin, errors.WithErrorCode(errors.EInvalid))
		}

		if deleteAfterDays != 0 {
			return errors.New("deleteAfterDays only applies to an AGE strategy, but this rule uses COUNT", errors.WithErrorCode(errors.EInvalid))
		}
	case StrategyProtect:
		if (keepMin != nil && *keepMin != 0) || deleteAfterDays != 0 {
			return errors.New("keepMin and deleteAfterDays do not apply to a PROTECT strategy, which keeps everything it matches", errors.WithErrorCode(errors.EInvalid))
		}
	case StrategyAge:
		if deleteAfterDays < cleanupMinAgeDays || deleteAfterDays > cleanupMaxAgeDays {
			return errors.New("deleteAfterDays must be between %d and %d for an AGE strategy, got %d", cleanupMinAgeDays, cleanupMaxAgeDays, deleteAfterDays, errors.WithErrorCode(errors.EInvalid))
		}

		if keepMin != nil && (*keepMin < 1 || *keepMin > cleanupMaxCountThreshold) {
			return errors.New("keepMin must be between 1 and %d for an AGE strategy, got %d", cleanupMaxCountThreshold, *keepMin, errors.WithErrorCode(errors.EInvalid))
		}
	default:
		return errors.New("unsupported strategy %q, expected one of %s, %s or %s", s, StrategyCount, StrategyAge, StrategyProtect, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// CleanupRuleKind identifies which resource type a cleanup policy applies to.
type CleanupRuleKind string

const (
	// CleanupRuleKindTerraformModules targets Terraform module versions.
	CleanupRuleKindTerraformModules CleanupRuleKind = "terraform_modules"
	// CleanupRuleKindTerraformProviders targets Terraform provider versions.
	CleanupRuleKindTerraformProviders CleanupRuleKind = "terraform_providers"
	// CleanupRuleKindRuns targets workspace runs.
	CleanupRuleKindRuns CleanupRuleKind = "runs"
)

// IsValid reports whether the kind is one this version supports.
func (k CleanupRuleKind) IsValid() bool {
	switch k {
	case CleanupRuleKindTerraformModules,
		CleanupRuleKindTerraformProviders,
		CleanupRuleKindRuns:
		return true
	default:
		return false
	}
}

// RequiresGroup reports whether this kind may only be configured on a group.
func (k CleanupRuleKind) RequiresGroup() bool {
	switch k {
	case CleanupRuleKindTerraformModules,
		CleanupRuleKindTerraformProviders:
		return true
	default:
		return false
	}
}

// CleanupGlob is a glob pattern matched against a resource attribute (name, system, version, ...),
// following go-glob syntax directly: "*" matches everything and "" matches nothing, since no real
// resource attribute is ever the empty string.
type CleanupGlob string

// Matches reports whether value matches the glob.
func (g CleanupGlob) Matches(value string) bool {
	return glob.Glob(g.String(), value)
}

// String implements the fmt.Stringer interface.
func (g CleanupGlob) String() string {
	return string(g)
}

// Validate returns an error if the pattern is set but invalid. Callers wrap the error to say which
// glob on a rule is at fault.
func (g CleanupGlob) Validate() error {
	if g == "" {
		return errors.New("pattern must not be empty, it would never match anything; use %q to match everything", "*", errors.WithErrorCode(errors.EInvalid))
	}

	if len(g) > cleanupMaxPatternLength {
		return errors.New("pattern %s exceeds maximum length of %d",
			g[:cleanupMaxPatternLength],
			cleanupMaxPatternLength,
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	return nil
}

// CleanupPolicy is the cleanup policy for one resource Kind within a namespace.
type CleanupPolicy struct {
	Metadata                    ResourceMetadata
	GroupID                     *string
	WorkspaceID                 *string
	SweepCursor                 *string
	Disabled                    bool
	SweepClaimedAt              *time.Time
	LastSweepCompletedAt        *time.Time
	TerraformModulePolicyData   *TerraformModuleCleanupPolicyData
	TerraformProviderPolicyData *TerraformProviderCleanupPolicyData
	RunPolicyData               *RunCleanupPolicyData
	Kind                        CleanupRuleKind
}

// GetID returns the Metadata ID.
func (p *CleanupPolicy) GetID() string {
	return p.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (p *CleanupPolicy) GetGlobalID() string {
	return gid.ToGlobalID(p.GetModelType(), p.Metadata.ID)
}

// GetModelType returns the Model's type.
func (p *CleanupPolicy) GetModelType() types.ModelType {
	return types.CleanupPolicyModelType
}

// NamespacePath returns the path of the associated namespace.
func (p *CleanupPolicy) NamespacePath() string {
	return trn.MustParseAny(p.Metadata.TRN).ParentPath()
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (p *CleanupPolicy) ResolveMetadata(key string) (*string, error) {
	return p.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid.
func (p *CleanupPolicy) Validate() error {
	if (p.GroupID == nil) == (p.WorkspaceID == nil) {
		if p.GroupID != nil {
			return errors.New("exactly one of groupID or workspaceID must be set, but both were set", errors.WithErrorCode(errors.EInvalid))
		}

		return errors.New("exactly one of groupID or workspaceID must be set, but neither was set", errors.WithErrorCode(errors.EInvalid))
	}

	if p.GroupID == nil && p.Kind.RequiresGroup() {
		return errors.New("a %s cleanup policy can only be set on a group, not on a workspace", p.Kind, errors.WithErrorCode(errors.EInvalid))
	}

	// For the matching kind, delegate to the data struct's own Validate; for others, the data
	// must be absent.
	switch p.Kind {
	case CleanupRuleKindTerraformModules:
		if p.TerraformProviderPolicyData != nil || p.RunPolicyData != nil {
			return errors.New("only terraform_module_policy_data may be set on a %s cleanup policy", p.Kind, errors.WithErrorCode(errors.EInvalid))
		}

		if p.TerraformModulePolicyData == nil {
			return errors.New("terraform module policy data is required when using %s kind", p.Kind, errors.WithErrorCode(errors.EInvalid))
		}

		return p.TerraformModulePolicyData.validate()
	case CleanupRuleKindTerraformProviders:
		if p.TerraformModulePolicyData != nil || p.RunPolicyData != nil {
			return errors.New("only terraform_provider_policy_data may be set on a %s cleanup policy", p.Kind, errors.WithErrorCode(errors.EInvalid))
		}

		if p.TerraformProviderPolicyData == nil {
			return errors.New("terraform provider policy data is required when using %s kind", p.Kind, errors.WithErrorCode(errors.EInvalid))
		}

		return p.TerraformProviderPolicyData.validate()
	case CleanupRuleKindRuns:
		if p.TerraformModulePolicyData != nil || p.TerraformProviderPolicyData != nil {
			return errors.New("only run_policy_data may be set on a %s cleanup policy", p.Kind, errors.WithErrorCode(errors.EInvalid))
		}

		if p.RunPolicyData == nil {
			return errors.New("run policy data is required when using %s kind", p.Kind, errors.WithErrorCode(errors.EInvalid))
		}

		return p.RunPolicyData.validate()
	default:
		return errors.New("unsupported cleanup policy kind %q", p.Kind)
	}
}

// TerraformModuleCleanupPolicyData holds the rules for a terraform_modules cleanup policy.
type TerraformModuleCleanupPolicyData struct {
	Rules []*TerraformModuleCleanupRule `json:"rules"`
}

// Validate returns an error if the policy data is invalid.
func (d *TerraformModuleCleanupPolicyData) validate() error {
	return validateCleanupRules(CleanupRuleKindTerraformModules, d.Rules)
}

// TerraformModuleCleanupRule defines a cleanup rule for a group's Terraform module versions.
type TerraformModuleCleanupRule struct {
	Strategy        CleanupStrategy `json:"strategy"`
	Description     string          `json:"description,omitempty"`
	NameGlob        CleanupGlob     `json:"name_glob,omitempty"`
	SystemGlob      CleanupGlob     `json:"system_glob,omitempty"`
	VersionGlob     CleanupGlob     `json:"version_glob,omitempty"`
	DeleteAfterDays int32           `json:"delete_after_days,omitempty"`
}

// validate returns an error if the rule is invalid.
func (r TerraformModuleCleanupRule) validate() error {
	if len(r.Description) > maxDescriptionLength {
		return errors.New("description exceeds the maximum length (%d)", maxDescriptionLength, errors.WithErrorCode(errors.EInvalid))
	}

	if err := r.NameGlob.Validate(); err != nil {
		return errors.Wrap(err, "invalid nameGlob")
	}

	if err := r.SystemGlob.Validate(); err != nil {
		return errors.Wrap(err, "invalid systemGlob")
	}

	if err := r.VersionGlob.Validate(); err != nil {
		return errors.Wrap(err, "invalid versionGlob")
	}

	return r.Strategy.validate(nil, r.DeleteAfterDays)
}

// Shadows reports whether every module version other would match also matches r, so that r placed
// before other would make other unreachable.
func (r *TerraformModuleCleanupRule) shadows(other *TerraformModuleCleanupRule) bool {
	return glob.Shadows(r.NameGlob.String(), other.NameGlob.String()) &&
		glob.Shadows(r.SystemGlob.String(), other.SystemGlob.String()) &&
		glob.Shadows(r.VersionGlob.String(), other.VersionGlob.String())
}

// getStrategy returns the rule's strategy.
func (r *TerraformModuleCleanupRule) getStrategy() CleanupStrategy {
	return r.Strategy
}

// TerraformProviderCleanupPolicyData holds the rules for a terraform_providers cleanup policy.
type TerraformProviderCleanupPolicyData struct {
	Rules []*TerraformProviderCleanupRule `json:"rules"`
}

// Validate returns an error if the policy data is invalid.
func (d *TerraformProviderCleanupPolicyData) validate() error {
	return validateCleanupRules(CleanupRuleKindTerraformProviders, d.Rules)
}

// TerraformProviderCleanupRule defines a cleanup rule for a group's Terraform provider versions.
type TerraformProviderCleanupRule struct {
	Strategy        CleanupStrategy `json:"strategy"`
	Description     string          `json:"description,omitempty"`
	NameGlob        CleanupGlob     `json:"name_glob,omitempty"`
	VersionGlob     CleanupGlob     `json:"version_glob,omitempty"`
	DeleteAfterDays int32           `json:"delete_after_days,omitempty"`
}

// Validate returns an error if the rule is invalid.
func (r TerraformProviderCleanupRule) validate() error {
	if len(r.Description) > maxDescriptionLength {
		return errors.New("description exceeds the maximum length (%d)", maxDescriptionLength, errors.WithErrorCode(errors.EInvalid))
	}

	if err := r.NameGlob.Validate(); err != nil {
		return errors.Wrap(err, "invalid nameGlob")
	}

	if err := r.VersionGlob.Validate(); err != nil {
		return errors.Wrap(err, "invalid versionGlob")
	}

	return r.Strategy.validate(nil, r.DeleteAfterDays)
}

// Shadows reports whether every provider version other would match also matches r, so that r placed
// before other would make other unreachable.
func (r *TerraformProviderCleanupRule) shadows(other *TerraformProviderCleanupRule) bool {
	return glob.Shadows(r.NameGlob.String(), other.NameGlob.String()) &&
		glob.Shadows(r.VersionGlob.String(), other.VersionGlob.String())
}

// getStrategy returns the rule's strategy.
func (r *TerraformProviderCleanupRule) getStrategy() CleanupStrategy {
	return r.Strategy
}

// RunCleanupPolicyData holds the rules for a runs cleanup policy.
type RunCleanupPolicyData struct {
	Rules []*RunCleanupRule `json:"rules"`
}

// Validate returns an error if the policy data is invalid.
func (d *RunCleanupPolicyData) validate() error {
	return validateCleanupRules(CleanupRuleKindRuns, d.Rules)
}

// RunCleanupRule defines a cleanup rule for a workspace's (or an inheriting group's) runs.
type RunCleanupRule struct {
	Strategy        CleanupStrategy `json:"strategy"`
	Description     string          `json:"description,omitempty"`
	Speculative     *bool           `json:"speculative,omitempty"`
	Assessment      *bool           `json:"assessment,omitempty"`
	Status          []RunStatus     `json:"status,omitempty"`
	KeepMin         int32           `json:"keep_min,omitempty"`
	DeleteAfterDays int32           `json:"delete_after_days,omitempty"`
}

// Validate returns an error if the rule is invalid.
func (r RunCleanupRule) validate() error {
	if len(r.Description) > maxDescriptionLength {
		return errors.New("description exceeds the maximum length (%d)", maxDescriptionLength, errors.WithErrorCode(errors.EInvalid))
	}

	// Only statuses that can be cleanup candidates are allowed; applied is excluded because
	// applied runs always produce a state version.
	seen := make(map[RunStatus]struct{}, len(r.Status))
	for _, status := range r.Status {
		if !status.In(CleanupCandidateRunStatuses) {
			return errors.New("status %q is not a valid run cleanup status; expected one of %v", status, CleanupCandidateRunStatuses, errors.WithErrorCode(errors.EInvalid))
		}

		if _, ok := seen[status]; ok {
			return errors.New("status %q appears more than once in the status list", status, errors.WithErrorCode(errors.EInvalid))
		}

		seen[status] = struct{}{}
	}

	// An assessment run is always speculative; require the rule to state this explicitly so
	// that filter comparisons (e.g. shadow detection) do not need to reason about the implication.
	if (r.Assessment != nil && *r.Assessment) && (r.Speculative == nil || !*r.Speculative) {
		return errors.New("assessment is true but speculative is not set to true; an assessment run is always speculative, so speculative must be explicitly set to true", errors.WithErrorCode(errors.EInvalid))
	}

	return r.Strategy.validate(&r.KeepMin, r.DeleteAfterDays)
}

// Shadows reports whether every run other would match also matches r, so that r placed before other
// would make other unreachable.
func (r *RunCleanupRule) shadows(other *RunCleanupRule) bool {
	// A nil field means "no constraint," so it never blocks shadowing; a set field only shadows
	// another set field with the same value, since other's could otherwise vary.
	if r.Speculative != nil && (other.Speculative == nil || *r.Speculative != *other.Speculative) {
		return false
	}

	if r.Assessment != nil && (other.Assessment == nil || *r.Assessment != *other.Assessment) {
		return false
	}

	if len(r.Status) == 0 {
		return true
	}

	if len(other.Status) == 0 {
		return false
	}

	for _, status := range other.Status {
		if !status.In(r.Status) {
			return false
		}
	}

	return true
}

// getStrategy returns the rule's strategy.
func (r *RunCleanupRule) getStrategy() CleanupStrategy {
	return r.Strategy
}

// GetCleanupPolicyForKind returns the policy covering kind and whether one was found.
func GetCleanupPolicyForKind(policies []*CleanupPolicy, kind CleanupRuleKind) (*CleanupPolicy, bool) {
	for _, policy := range policies {
		if policy.Kind == kind {
			return policy, true
		}
	}

	return nil, false
}

// cleanupRule is the contract validateKindRules requires from a rule type, generic over the
// concrete rule type since shadows compares two rules of the same type.
type cleanupRule[T any] interface {
	validate() error
	shadows(T) bool
	getStrategy() CleanupStrategy
	comparable
}

// validateCleanupRules returns an error unless rules holds between one and cleanupMaxRules valid, non-nil,
// unshadowed rules.
func validateCleanupRules[T cleanupRule[T]](kind CleanupRuleKind, rules []T) error {
	if len(rules) > cleanupMaxRules {
		return errors.New("a %s cleanup policy specifies %d rules, exceeding the maximum of %d", kind, len(rules), cleanupMaxRules, errors.WithErrorCode(errors.EInvalid))
	}

	// A PROTECT rule only has any effect on resources no earlier rule already claims, since the first
	// matching rule wins, so every PROTECT rule must come before every non-PROTECT rule.
	firstNonProtect := -1
	for i, rule := range rules {
		var zero T
		if rule == zero {
			return errors.New("%s rule %d is nil", kind, i+1, errors.WithErrorCode(errors.EInvalid))
		}

		if err := rule.validate(); err != nil {
			return errors.Wrap(err, "%s rule %d is invalid", kind, i+1, errors.WithErrorCode(errors.EInvalid))
		}

		if rule.getStrategy() == StrategyProtect {
			if firstNonProtect >= 0 {
				return errors.New(
					"rule %d uses PROTECT but comes after rule %d, which does not; a PROTECT rule never applies to a resource an earlier rule already claims, so move rule %d before rule %d",
					i+1, firstNonProtect+1, i+1, firstNonProtect+1,
					errors.WithErrorCode(errors.EInvalid),
				)
			}
			continue
		}

		if firstNonProtect < 0 {
			firstNonProtect = i
		}
	}

	for i, earlier := range rules {
		for j := i + 1; j < len(rules); j++ {
			if earlier.shadows(rules[j]) {
				return errors.New(
					"rule %d always matches everything rule %d would match, so rule %d can never apply; move rule %d before rule %d, or narrow rule %d so it no longer covers everything rule %d matches",
					i+1, j+1, j+1, j+1, i+1, i+1, j+1,
					errors.WithErrorCode(errors.EInvalid),
				)
			}
		}
	}

	return nil
}
