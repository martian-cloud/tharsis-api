package models

import (
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestValidPolicyEnforcementLevel(t *testing.T) {
	tests := []struct {
		name            string
		level           PolicyEnforcementLevel
		expectErrorCode errors.CodeType
	}{
		{
			name:  "advisory is valid",
			level: PolicyEnforcementAdvisory,
		},
		{
			name:  "soft_mandatory is valid",
			level: PolicyEnforcementSoftMandatory,
		},
		{
			name:  "hard_mandatory is valid",
			level: PolicyEnforcementHardMandatory,
		},
		{
			name:            "unknown level is invalid",
			level:           PolicyEnforcementLevel("bogus"),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidPolicyEnforcementLevel(tt.level)
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidSpeculativeRunEnforcementLevel(t *testing.T) {
	tests := []struct {
		name            string
		level           PolicyEnforcementLevel
		expectErrorCode errors.CodeType
	}{
		{
			name:  "advisory is valid",
			level: PolicyEnforcementAdvisory,
		},
		{
			name:  "hard_mandatory is valid",
			level: PolicyEnforcementHardMandatory,
		},
		{
			// A run with no apply has nothing an override would unblock, so the level that means
			// "approvable" is not one it can be enforced at.
			name:            "soft_mandatory is invalid",
			level:           PolicyEnforcementSoftMandatory,
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "unset is invalid",
			level:           PolicyEnforcementLevel(""),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "unknown level is invalid",
			level:           PolicyEnforcementLevel("bogus"),
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidSpeculativeRunEnforcementLevel(tt.level)
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPolicy_Validate(t *testing.T) {
	validOPA := func(mutate func(*Policy)) *Policy {
		p := &Policy{
			GroupID: "group-1",
			Name:    "my-policy",
			Kind:    PolicyKindOPA,
			OPAData: &OPAPolicyData{
				PackageSource:                  "acme/policies",
				EnforcementLevel:               PolicyEnforcementAdvisory,
				SpeculativeRunEnforcementLevel: PolicyEnforcementAdvisory,
				Stage:                          RunTaskStageNamePrePlan,
			},
		}
		if mutate != nil {
			mutate(p)
		}
		return p
	}

	tests := []struct {
		name            string
		policy          *Policy
		expectErrorCode errors.CodeType
	}{
		{
			name:   "valid advisory OPA policy",
			policy: validOPA(nil),
		},
		{
			name:            "missing group owner",
			policy:          validOPA(func(p *Policy) { p.GroupID = "" }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "missing name",
			policy:          validOPA(func(p *Policy) { p.Name = "" }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "missing kind",
			policy:          validOPA(func(p *Policy) { p.Kind = "" }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "opa kind without opa data",
			policy:          validOPA(func(p *Policy) { p.OPAData = nil }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "unsupported enforcement level",
			policy:          validOPA(func(p *Policy) { p.OPAData.EnforcementLevel = PolicyEnforcementLevel("bogus") }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "hard_mandatory speculative level is allowed alongside a soft_mandatory level",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.OPAData.SpeculativeRunEnforcementLevel = PolicyEnforcementHardMandatory
			}),
		},
		{
			name: "soft_mandatory speculative level is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.SpeculativeRunEnforcementLevel = PolicyEnforcementSoftMandatory
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "unset speculative level is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.SpeculativeRunEnforcementLevel = PolicyEnforcementLevel("")
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:   "valid version constraint",
			policy: validOPA(func(p *Policy) { p.OPAData.PackageVersionConstraint = ptr.String(">= 1.0.0") }),
		},
		{
			name:            "invalid version constraint",
			policy:          validOPA(func(p *Policy) { p.OPAData.PackageVersionConstraint = ptr.String("not-a-constraint") }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:   "empty version constraint is ignored",
			policy: validOPA(func(p *Policy) { p.OPAData.PackageVersionConstraint = ptr.String("") }),
		},
		{
			name:   "valid hex sha256 digest",
			policy: validOPA(func(p *Policy) { p.OPAData.PackageDigest = ptr.String(strings.Repeat("a", 64)) }),
		},
		{
			name:            "digest wrong length",
			policy:          validOPA(func(p *Policy) { p.OPAData.PackageDigest = ptr.String("abcd") }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "digest not hex",
			policy:          validOPA(func(p *Policy) { p.OPAData.PackageDigest = ptr.String(strings.Repeat("z", 64)) }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "valid soft_mandatory policy with approvers",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
				p.AllowedUserIDs = []string{"user-1"}
			}),
		},
		{
			name: "approvers on advisory policy are rejected",
			policy: validOPA(func(p *Policy) {
				p.RequiredApprovals = 1
				p.AllowedUserIDs = []string{"user-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "required approvals without a subject is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "subject without required approvals is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.AllowedTeamIDs = []string{"team-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			// Rejected on the repeat itself, not on whether the count happens to work out -- so it is
			// rejected here even though one approval is reachable from one approver.
			name: "a repeated user approver is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
				p.AllowedUserIDs = []string{"user-1", "user-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "a repeated service account approver is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
				p.AllowedServiceAccountIDs = []string{"sa-1", "sa-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "a repeated team approver is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
				p.AllowedTeamIDs = []string{"team-1", "team-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			// The same id in two different principal lists is two different principals.
			name: "the same id as a user and as a team is not a duplicate",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
				p.AllowedUserIDs = []string{"shared-id"}
				p.AllowedTeamIDs = []string{"shared-id"}
			}),
		},
		{
			name: "more required approvals than approvers is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 3
				p.AllowedUserIDs = []string{"user-1"}
				p.AllowedServiceAccountIDs = []string{"sa-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "required approvals equal to the approver count is allowed",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 2
				p.AllowedUserIDs = []string{"user-1"}
				p.AllowedServiceAccountIDs = []string{"sa-1"}
			}),
		},
		{
			// A team's approvals come from its membership, which is not known here and can change after
			// the policy is written, so a team subject carries no ceiling to check against.
			name: "a team subject may back more required approvals than there are subjects",
			policy: validOPA(func(p *Policy) {
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 5
				p.AllowedTeamIDs = []string{"team-1"}
			}),
		},
		{
			name:            "empty package source is rejected",
			policy:          validOPA(func(p *Policy) { p.OPAData.PackageSource = "" }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "invalid stage is rejected",
			policy:          validOPA(func(p *Policy) { p.OPAData.Stage = RunTaskStageName("bogus") }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "empty stage is rejected",
			policy:          validOPA(func(p *Policy) { p.OPAData.Stage = "" }),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:   "post_apply with advisory enforcement is accepted",
			policy: validOPA(func(p *Policy) { p.OPAData.Stage = RunTaskStageNamePostApply }),
		},
		{
			name: "post_apply with soft_mandatory enforcement is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.Stage = RunTaskStageNamePostApply
				p.OPAData.EnforcementLevel = PolicyEnforcementSoftMandatory
				p.RequiredApprovals = 1
				p.AllowedUserIDs = []string{"user-1"}
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "post_apply with hard_mandatory enforcement is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.Stage = RunTaskStageNamePostApply
				p.OPAData.EnforcementLevel = PolicyEnforcementHardMandatory
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "post_apply with hard_mandatory speculative enforcement is rejected",
			policy: validOPA(func(p *Policy) {
				p.OPAData.Stage = RunTaskStageNamePostApply
				p.OPAData.SpeculativeRunEnforcementLevel = PolicyEnforcementHardMandatory
			}),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:   "pre_apply with advisory enforcement is accepted",
			policy: validOPA(func(p *Policy) { p.OPAData.Stage = RunTaskStageNamePreApply }),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.expectErrorCode != "" {
				require.Error(t, err)
				assert.Equal(t, tt.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPolicy_MatchesRun(t *testing.T) {
	tests := []struct {
		name          string
		scope         []*ScopeRule
		workspacePath string
		miPaths       []string
		want          bool
	}{
		{
			name:          "empty scope matches everything",
			scope:         nil,
			workspacePath: "acme/team/ws-1",
			want:          true,
		},
		{
			name: "an exact namespace path matches that workspace",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-1",
			want:          true,
		},
		{
			// The wildcard-free case has to be equality rather than a prefix, or a rule naming one
			// workspace would silently pick up its siblings.
			name: "an exact namespace path matches nothing else",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-10",
			want:          false,
		},
		{
			// This is what replaces scoping by group id: the wildcard crosses path separators, so one
			// rule covers the group's whole subtree however deep the workspace sits.
			name: "a group glob matches a workspace nested several levels down",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "acme/*"},
			},
			workspacePath: "acme/team/sub/deeper/ws-1",
			want:          true,
		},
		{
			name: "a group glob does not match a sibling group",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "acme/*"},
			},
			workspacePath: "other/team/ws-1",
			want:          false,
		},
		{
			name: "include present but no rule matches",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "acme/other/*"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			name: "exclude wins over include",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "acme/*"},
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionExclude, Pattern: "acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			name: "only excludes and none match allows the run",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionExclude, Pattern: "acme/other/*"},
			},
			workspacePath: "acme/team/ws-1",
			want:          true,
		},
		{
			name: "a managed identity glob matches one of the workspace's identities",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/prod-*"},
			},
			workspacePath: "acme/team/ws-1",
			miPaths:       []string{"acme/dev-mi", "acme/prod-mi"},
			want:          true,
		},
		{
			// A workspace can use an identity inherited from a group above the policy's owner, so an
			// identity path outside the owning subtree is a legitimate match.
			name: "a managed identity path above the owning group still matches",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/shared-mi"},
			},
			workspacePath: "acme/team/ws-1",
			miPaths:       []string{"acme/shared-mi"},
			want:          true,
		},
		{
			name: "a managed identity rule does not match a workspace with no identities",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/prod-*"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			// Every rule has a path, so an empty one is only reachable through a stored rule the
			// service would have rejected. It must never widen the policy's reach.
			name: "an empty path matches nothing",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: ""},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			name: "an unknown rule type matches nothing",
			scope: []*ScopeRule{
				{Type: ScopeRuleType("project"), Action: ScopeRuleActionInclude, Pattern: "acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			// The point of the group type: naming the group covers everything under it, with no
			// wildcard to remember. A workspace rule with the same path would match nothing.
			name: "a group rule matches a workspace nested beneath it",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme/team"},
			},
			workspacePath: "acme/team/sub/deeper/ws-1",
			want:          true,
		},
		{
			name: "a group rule does not match a sibling group",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme/team"},
			},
			workspacePath: "acme/other/ws-1",
			want:          false,
		},
		{
			// A group rule reaches downward only, so a workspace sitting above the named group is
			// outside it — matching the group's own path would also match nothing, since no workspace
			// can share a group's path.
			name: "a group rule does not match a workspace in an ancestor group",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme/team/sub"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			// A wildcard in a group rule names groups *below* that path, so the workspaces it covers
			// start one level deeper than the wildcard-free form.
			name: "a group rule ending in a wildcard names groups below it",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme/team/*"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			name: "a group rule ending in a wildcard matches a workspace one level deeper",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme/team/*"},
			},
			workspacePath: "acme/team/sub/ws-1",
			want:          true,
		},
		{
			// A TRN is what the UI's copy buttons produce, so it is what gets pasted into the field.
			name: "a workspace rule accepts a TRN",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "trn:workspace:acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-1",
			want:          true,
		},
		{
			name: "a workspace rule with a TRN naming another workspace does not match",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "trn:workspace:acme/team/ws-2"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			name: "a group rule accepts a TRN and still covers the subtree",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "trn:group:acme/team"},
			},
			workspacePath: "acme/team/sub/ws-1",
			want:          true,
		},
		{
			name: "a managed identity rule accepts a TRN",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "trn:managed_identity:acme/prod-mi"},
			},
			workspacePath: "acme/team/ws-1",
			miPaths:       []string{"acme/prod-mi"},
			want:          true,
		},
		{
			name: "an exclude written as a TRN wins over an include",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme"},
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionExclude, Pattern: "trn:workspace:acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			// The service rejects a wrong-type TRN on save, so this only happens for a rule stored
			// before that check. It must fail closed rather than match the stripped path.
			name: "a TRN of the wrong type matches nothing",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "trn:group:acme/team/ws-1"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
		{
			name: "a malformed TRN matches nothing",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "trn:workspace:"},
			},
			workspacePath: "acme/team/ws-1",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Scope: tt.scope}
			got := p.MatchesRun(tt.workspacePath, tt.miPaths)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPolicy_MatchesManagedIdentity(t *testing.T) {
	tests := []struct {
		name    string
		scope   []*ScopeRule
		miPaths []string
		want    bool
	}{
		{
			// The difference from MatchesRun, which treats an empty scope as applying to everything: a
			// policy that names no identity is not one of this identity's policies.
			name:    "an empty scope matches nothing",
			miPaths: []string{"acme/prod-mi"},
			want:    false,
		},
		{
			name: "an identity rule matches by exact path",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/prod-mi"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    true,
		},
		{
			name: "an identity rule matches by glob",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/*"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    true,
		},
		{
			// Aliases arrive as a second path for the same identity, so either one naming it is enough.
			name: "an identity rule matches any of the paths",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "other/prod/aws"},
			},
			miPaths: []string{"acme/aws-alias", "other/prod/aws"},
			want:    true,
		},
		{
			name: "an identity rule naming something else does not match",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/dev-*"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    false,
		},
		{
			// A workspace rule is about the run's workspace, which this question does not have. Same
			// for a group rule, which is also matched against the workspace path.
			name: "a workspace rule never matches",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeWorkspace, Action: ScopeRuleActionInclude, Pattern: "*"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    false,
		},
		{
			name: "a group rule never matches",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeGroup, Action: ScopeRuleActionInclude, Pattern: "acme"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    false,
		},
		{
			name: "an identity rule matches a TRN path",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "trn:managed_identity:acme/prod-mi"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    true,
		},
		{
			name: "an exclude naming the identity wins over an include",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/*"},
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionExclude, Pattern: "acme/prod-mi"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    false,
		},
		{
			// Without an include there is nothing to exclude from, so this is not a match either way.
			name: "an exclude alone matches nothing",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionExclude, Pattern: "acme/dev-*"},
			},
			miPaths: []string{"acme/prod-mi"},
			want:    false,
		},
		{
			name: "no identity paths matches nothing",
			scope: []*ScopeRule{
				{Type: ScopeRuleTypeManagedIdentity, Action: ScopeRuleActionInclude, Pattern: "acme/*"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Scope: tt.scope}
			assert.Equal(t, tt.want, p.MatchesManagedIdentity(tt.miPaths))
		})
	}
}

func TestPolicy_GetGroupPath(t *testing.T) {
	tests := []struct {
		name string
		trn  string
		want string
	}{
		{
			name: "nested group path",
			trn:  "trn:policy:acme/team/security/require-tags",
			want: "acme/team/security",
		},
		{
			name: "single level group path",
			trn:  "trn:policy:acme/require-tags",
			want: "acme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Metadata: ResourceMetadata{TRN: tt.trn}}
			assert.Equal(t, tt.want, p.GetGroupPath())
		})
	}
}
