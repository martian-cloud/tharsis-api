package models

import (
	"crypto/sha256"
	"encoding/hex"

	goversion "github.com/hashicorp/go-version"
	"github.com/ryanuber/go-glob"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

var _ Model = (*Policy)(nil)

// PolicyKind identifies the policy engine type.
type PolicyKind string

// PolicyKind constants.
const (
	// PolicyKindOPA is an Open Policy Agent policy.
	PolicyKindOPA PolicyKind = "opa"
)

// OPAPolicyData holds OPA-specific policy configuration.
type OPAPolicyData struct {
	PackageSource            string                 `json:"package_source"`
	PackageVersionConstraint *string                `json:"package_version_constraint"`
	PackageDigest            *string                `json:"package_digest"`
	EnforcementLevel         PolicyEnforcementLevel `json:"enforcement_level"`
	// SpeculativeRunEnforcementLevel is the level this policy is enforced at on a run that is a
	// speculative or assessment run
	SpeculativeRunEnforcementLevel PolicyEnforcementLevel `json:"speculative_run_enforcement_level"`
	Stage                          RunTaskStageName       `json:"stage"`
}

// PolicyEnforcementLevel controls how a policy failure affects a run.
type PolicyEnforcementLevel string

// PolicyEnforcementLevel constants.
const (
	// PolicyEnforcementAdvisory logs failures but never blocks a run.
	PolicyEnforcementAdvisory PolicyEnforcementLevel = "advisory"
	// PolicyEnforcementSoftMandatory blocks a run but can be overridden.
	PolicyEnforcementSoftMandatory PolicyEnforcementLevel = "soft_mandatory"
	// PolicyEnforcementHardMandatory blocks a run and cannot be overridden.
	PolicyEnforcementHardMandatory PolicyEnforcementLevel = "hard_mandatory"
)

// ValidPolicyEnforcementLevel returns an error if the enforcement level is not supported.
func ValidPolicyEnforcementLevel(level PolicyEnforcementLevel) error {
	switch level {
	case PolicyEnforcementAdvisory, PolicyEnforcementSoftMandatory, PolicyEnforcementHardMandatory:
		return nil
	default:
		return errors.New("policy enforcement level %s is not supported", level, errors.WithErrorCode(errors.EInvalid))
	}
}

// ValidSpeculativeRunEnforcementLevel returns an error if the level is not one a run without an apply
// can be enforced at. soft_mandatory is rejected rather than merely discouraged: it means "a human may
// approve past this", and there is no human waiting on one of these runs. A speculative plan finishes
// without an apply for an override to unblock, and an assessment run is created on a schedule with
// nobody watching it -- so a gate on either collects approvals that act on nothing while the run parks
// unfinished. That leaves two honest answers, and the policy has to pick one: record the failure
// (advisory) or refuse to run at all (hard_mandatory).
func ValidSpeculativeRunEnforcementLevel(level PolicyEnforcementLevel) error {
	switch level {
	case PolicyEnforcementAdvisory, PolicyEnforcementHardMandatory:
		return nil
	default:
		return errors.New("policy speculative run enforcement level %s is not supported, must be %s or %s",
			level, PolicyEnforcementAdvisory, PolicyEnforcementHardMandatory, errors.WithErrorCode(errors.EInvalid))
	}
}

// ScopeRuleType identifies which of a run's paths a scope rule is matched against.
type ScopeRuleType string

// ScopeRuleType constants.
const (
	// ScopeRuleTypeWorkspace matches the path of the run's workspace.
	ScopeRuleTypeWorkspace ScopeRuleType = "workspace"
	// ScopeRuleTypeGroup matches any workspace nested beneath a group whose path matches. It is the
	// explicit way to scope a policy to a subtree, so a path with no wildcard covers the group's whole
	// subtree rather than naming a single namespace.
	ScopeRuleTypeGroup ScopeRuleType = "group"
	// ScopeRuleTypeManagedIdentity matches the paths of the managed identities the run's workspace
	// uses. An alias is matched by its own path and by the path of the identity it aliases.
	ScopeRuleTypeManagedIdentity ScopeRuleType = "managed_identity"
)

// TRNType returns the TRN type a rule of this type accepts in its path, and whether the rule type is
// one this package knows. It does double duty: the policy service uses the bool as its "is this a
// valid rule type" check, and the TRN type as what a TRN path is allowed to name.
func (t ScopeRuleType) TRNType() (trn.Type, bool) {
	switch t {
	case ScopeRuleTypeWorkspace:
		return trn.TypeWorkspace, true
	case ScopeRuleTypeGroup:
		return trn.TypeGroup, true
	case ScopeRuleTypeManagedIdentity:
		return trn.TypeManagedIdentity, true
	}
	return "", false
}

// ScopeRuleAction controls whether the rule includes or excludes matching runs.
type ScopeRuleAction string

// ScopeRuleAction constants.
const (
	ScopeRuleActionInclude ScopeRuleAction = "include"
	ScopeRuleActionExclude ScopeRuleAction = "exclude"
)

// ScopeRule is one element of a Policy's scope array.
type ScopeRule struct {
	Type   ScopeRuleType   `json:"type"`
	Action ScopeRuleAction `json:"action"`
	// Pattern is a glob pattern matched against the path named by Type. The wildcard crosses path
	// separators, so "group/*" covers a workspace at any depth beneath that group, and a pattern
	// with no wildcard matches exactly one path. It may also be written as a TRN of the type Type
	// names ("trn:workspace:group/ws"), in which case only the resource path is matched.
	Pattern string `json:"pattern"`
}

// resolvedPattern returns the glob this rule matches with. Pattern may be given as a TRN — that is
// what the UI's copy buttons produce, so it is what gets pasted into the field — in which case only
// the resource path is matched. A malformed TRN, or one naming the wrong resource type, is returned
// unchanged and so matches nothing, since no real path starts with "trn:". The service rejects both
// when the policy is saved, so this only has to fail closed for a rule stored before that check.
func (r *ScopeRule) resolvedPattern() string {
	if !trn.IsTRN(r.Pattern) {
		return r.Pattern
	}

	parsed, err := trn.ParseAny(r.Pattern)
	if err != nil {
		return r.Pattern
	}

	if expected, ok := r.Type.TRNType(); !ok || parsed.Type() != expected {
		return r.Pattern
	}

	return parsed.Path()
}

func (r *ScopeRule) matches(workspacePath string, miPaths []string) bool {
	pattern := r.resolvedPattern()
	switch r.Type {
	case ScopeRuleTypeWorkspace:
		return glob.Glob(pattern, workspacePath)
	case ScopeRuleTypeGroup:
		// The wildcard crosses path separators, so one trailing "/*" covers the group's whole subtree
		// however deep the workspace sits — the same answer as globbing the pattern against each of
		// the workspace's ancestor group paths. No workspace can share a group's path, so matching the
		// group path itself would match nothing and is deliberately not attempted.
		return glob.Glob(pattern+"/*", workspacePath)
	case ScopeRuleTypeManagedIdentity:
		for _, p := range miPaths {
			if glob.Glob(pattern, p) {
				return true
			}
		}
	}
	return false
}

// Policy controls where a policy package is evaluated. It is always owned by a group (GroupID).
// Kind identifies the policy engine; OPAData holds OPA-specific configuration. Scope determines
// which runs it applies to; an empty Scope means the policy applies to all workspaces under the
// owning group.
type Policy struct {
	// GroupID is the owning group — always set (non-empty).
	GroupID string
	// Name is a human-readable identifier unique within the owning group.
	Name string
	// Description is an optional description of the policy.
	Description *string
	// Kind identifies the policy engine type.
	Kind                     PolicyKind
	OPAData                  *OPAPolicyData
	Scope                    []*ScopeRule
	RequiredApprovals        int
	AllowedUserIDs           []string
	AllowedServiceAccountIDs []string
	AllowedTeamIDs           []string
	CreatedBy                string
	Metadata                 ResourceMetadata
}

// GetID returns the Metadata ID.
func (p *Policy) GetID() string {
	return p.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (p *Policy) GetGlobalID() string {
	return gid.ToGlobalID(p.GetModelType(), p.Metadata.ID)
}

// GetModelType returns the model type.
func (p *Policy) GetModelType() types.ModelType {
	return types.PolicyModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (p *Policy) ResolveMetadata(key string) (*string, error) {
	return p.Metadata.resolveFieldValue(key)
}

// GetGroupPath returns the path of the owning group. A policy's TRN is the group path followed by the
// policy name, so the owner is the TRN's parent — no group lookup needed to name it.
func (p *Policy) GetGroupPath() string {
	return trn.MustParseAny(p.Metadata.TRN).ParentPath()
}

// MatchesRun reports whether the policy applies to a run with the given context.
// workspacePath is the path of the run's workspace; miPaths are the full paths of the managed
// identities that workspace uses, plus the source path of any alias among them.
func (p *Policy) MatchesRun(workspacePath string, miPaths []string) bool {
	if len(p.Scope) == 0 {
		return true
	}
	// Pass 1: excludes win — any matching exclude short-circuits the policy.
	for _, r := range p.Scope {
		if r.Action == ScopeRuleActionExclude && r.matches(workspacePath, miPaths) {
			return false
		}
	}
	// Pass 2: includes — match any one. Absence of include rules means no restriction.
	hasIncludes := false
	for _, r := range p.Scope {
		if r.Action == ScopeRuleActionInclude {
			hasIncludes = true
			if r.matches(workspacePath, miPaths) {
				return true
			}
		}
	}
	return !hasIncludes
}

// MatchesManagedIdentity reports whether the policy names one of the given managed identity paths in
// its scope. Unlike MatchesRun this asks about the identity rather than a run, so an include rule of
// type ScopeRuleTypeManagedIdentity has to match: a policy with no scope applies to every run under
// its group but singles out no identity, and the rest of the scope cannot be judged without knowing
// which workspace the run is in. An exclude rule naming the identity still wins over an include.
func (p *Policy) MatchesManagedIdentity(miPaths []string) bool {
	for _, r := range p.Scope {
		if r.Type == ScopeRuleTypeManagedIdentity && r.Action == ScopeRuleActionExclude && r.matches("", miPaths) {
			return false
		}
	}
	for _, r := range p.Scope {
		if r.Type == ScopeRuleTypeManagedIdentity && r.Action == ScopeRuleActionInclude && r.matches("", miPaths) {
			return true
		}
	}
	return false
}

// Validate returns an error if the model is not valid: GroupID and Name must be non-empty, Kind
// must be set, OPAData must be present for OPA policies, the enforcement level must be supported,
// the version constraint (if any) must parse, the digest (if any) must be a hex sha256, and
// approvers may only be set on a soft-mandatory policy and must be able to meet requiredApprovals.
func (p *Policy) Validate() error {
	if p.GroupID == "" {
		return errors.New("policy must have a group owner", errors.WithErrorCode(errors.EInvalid))
	}

	if p.Name == "" {
		return errors.New("policy name is required", errors.WithErrorCode(errors.EInvalid))
	}

	if p.Kind == "" {
		return errors.New("policy kind is required", errors.WithErrorCode(errors.EInvalid))
	}

	if p.Kind == PolicyKindOPA {
		if p.OPAData == nil {
			return errors.New("OPA policy data is required for opa kind", errors.WithErrorCode(errors.EInvalid))
		}
		if p.OPAData.PackageSource == "" {
			return errors.New("OPA policy package source is required", errors.WithErrorCode(errors.EInvalid))
		}
		if !p.OPAData.Stage.IsValid() {
			return errors.New("OPA policy stage %q is not a valid run stage", p.OPAData.Stage,
				errors.WithErrorCode(errors.EInvalid))
		}
		if err := ValidPolicyEnforcementLevel(p.OPAData.EnforcementLevel); err != nil {
			return err
		}
		if err := ValidSpeculativeRunEnforcementLevel(p.OPAData.SpeculativeRunEnforcementLevel); err != nil {
			return err
		}
		if p.OPAData.PackageVersionConstraint != nil && *p.OPAData.PackageVersionConstraint != "" {
			if _, err := goversion.NewConstraint(*p.OPAData.PackageVersionConstraint); err != nil {
				return errors.New("policy version constraint %q is invalid", *p.OPAData.PackageVersionConstraint,
					errors.WithErrorCode(errors.EInvalid))
			}
		}
		if p.OPAData.PackageDigest != nil {
			if b, err := hex.DecodeString(*p.OPAData.PackageDigest); err != nil || len(b) != sha256.Size {
				return errors.New("policy digest must be a hex-encoded sha256 checksum",
					errors.WithErrorCode(errors.EInvalid))
			}
		}
		// An approver appears once per policy: the approver tables hold one row per (policy, principal),
		// and naming a principal twice does not make it two approvers. Rejected rather than collapsed so
		// the caller finds out their list was not understood the way they wrote it -- silently dropping
		// the repeat would also silently change what requiredApprovals is measured against.
		for _, dup := range []struct {
			kind string
			ids  []string
		}{
			{"user", p.AllowedUserIDs},
			{"service account", p.AllowedServiceAccountIDs},
			{"team", p.AllowedTeamIDs},
		} {
			if id := firstDuplicateID(dup.ids); id != "" {
				return errors.New("%s %s is listed as an approver more than once", dup.kind, id,
					errors.WithErrorCode(errors.EInvalid))
			}
		}

		// Approvers only make sense for soft-mandatory policies.
		allowedUsers := len(p.AllowedUserIDs)
		allowedServiceAccounts := len(p.AllowedServiceAccountIDs)
		allowedTeams := len(p.AllowedTeamIDs)
		subjectCount := allowedUsers + allowedServiceAccounts + allowedTeams

		hasApprovers := p.RequiredApprovals > 0 || subjectCount > 0
		if hasApprovers && p.OPAData.EnforcementLevel != PolicyEnforcementSoftMandatory {
			return errors.New("approvers may only be set on a soft_mandatory policy",
				errors.WithErrorCode(errors.EInvalid))
		}
		if (p.RequiredApprovals > 0) != (subjectCount > 0) {
			return errors.New("approvers require both requiredApprovals >= 1 and at least one allowed subject",
				errors.WithErrorCode(errors.EInvalid))
		}
		// A user or service account approves at most once -- a second decision from the same principal
		// replaces the first -- so with no team subject the approver list is a ceiling on how many
		// approvals a gate can ever collect. Requiring more than that parks every run the policy soft-fails
		// with no way to reach the count, so it is rejected when the policy is written rather than
		// discovered on a stuck run. A team subject lifts the ceiling: its approvals come from live
		// membership, which is not known here and can change after the policy is written.
		if allowedTeams == 0 && p.RequiredApprovals > allowedUsers+allowedServiceAccounts {
			return errors.New(
				"requiredApprovals is %d but the policy allows only %d approver(s), so the requirement can never be met",
				p.RequiredApprovals, allowedUsers+allowedServiceAccounts,
				errors.WithErrorCode(errors.EInvalid))
		}
	}

	return nil
}

// firstDuplicateID returns the first id that appears more than once, or "" when every id is unique.
// Returning the id rather than a bool lets the caller name it in the error.
func firstDuplicateID(ids []string) string {
	if len(ids) < 2 {
		return ""
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return id
		}
		seen[id] = struct{}{}
	}
	return ""
}
