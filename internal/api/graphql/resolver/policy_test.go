package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestToModelScopeRules(t *testing.T) {
	type testCase struct {
		name        string
		input       *[]PolicyScopeRuleInput
		expectRules []*models.ScopeRule
	}

	testCases := []testCase{
		{
			name:        "nil input yields no rules",
			input:       nil,
			expectRules: nil,
		},
		{
			name: "a workspace rule keeps its glob",
			input: &[]PolicyScopeRuleInput{{
				Type:    "WORKSPACE",
				Action:  "INCLUDE",
				Pattern: "acme/*",
			}},
			expectRules: []*models.ScopeRule{{
				Type:    models.ScopeRuleTypeWorkspace,
				Action:  models.ScopeRuleActionInclude,
				Pattern: "acme/*",
			}},
		},
		{
			name: "a group rule keeps its path",
			input: &[]PolicyScopeRuleInput{{
				Type:    "GROUP",
				Action:  "INCLUDE",
				Pattern: "acme/team",
			}},
			expectRules: []*models.ScopeRule{{
				Type:    models.ScopeRuleTypeGroup,
				Action:  models.ScopeRuleActionInclude,
				Pattern: "acme/team",
			}},
		},
		{
			// A TRN path is stored verbatim; stripping it is the model's job at match time, so the
			// path the user typed is what comes back out of the API.
			name: "a TRN path is passed through unchanged",
			input: &[]PolicyScopeRuleInput{{
				Type:    "WORKSPACE",
				Action:  "INCLUDE",
				Pattern: "trn:workspace:acme/team/ws-1",
			}},
			expectRules: []*models.ScopeRule{{
				Type:    models.ScopeRuleTypeWorkspace,
				Action:  models.ScopeRuleActionInclude,
				Pattern: "trn:workspace:acme/team/ws-1",
			}},
		},
		{
			name: "a managed identity rule keeps its glob on an exclude rule",
			input: &[]PolicyScopeRuleInput{{
				Type:    "MANAGED_IDENTITY",
				Action:  "EXCLUDE",
				Pattern: "acme/prod-*",
			}},
			expectRules: []*models.ScopeRule{{
				Type:    models.ScopeRuleTypeManagedIdentity,
				Action:  models.ScopeRuleActionExclude,
				Pattern: "acme/prod-*",
			}},
		},
		{
			// Conversion stores what it is given so the required-field message lives in one place:
			// the policy service's scope validation.
			name: "an empty path is left for the service to reject",
			input: &[]PolicyScopeRuleInput{{
				Type:   "WORKSPACE",
				Action: "INCLUDE",
			}},
			expectRules: []*models.ScopeRule{{
				Type:   models.ScopeRuleTypeWorkspace,
				Action: models.ScopeRuleActionInclude,
			}},
		},
		{
			name: "every rule in a multi-rule scope is converted",
			input: &[]PolicyScopeRuleInput{
				{
					Type:    "WORKSPACE",
					Action:  "INCLUDE",
					Pattern: "acme/*",
				},
				{
					Type:    "WORKSPACE",
					Action:  "EXCLUDE",
					Pattern: "acme/prod/db",
				},
				{
					Type:    "MANAGED_IDENTITY",
					Action:  "INCLUDE",
					Pattern: "acme/deploy-mi",
				},
			},
			expectRules: []*models.ScopeRule{
				{
					Type:    models.ScopeRuleTypeWorkspace,
					Action:  models.ScopeRuleActionInclude,
					Pattern: "acme/*",
				},
				{
					Type:    models.ScopeRuleTypeWorkspace,
					Action:  models.ScopeRuleActionExclude,
					Pattern: "acme/prod/db",
				},
				{
					Type:    models.ScopeRuleTypeManagedIdentity,
					Action:  models.ScopeRuleActionInclude,
					Pattern: "acme/deploy-mi",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectRules, toModelScopeRules(tc.input))
		})
	}
}

// TestScopeRuleRoundTrips covers the read/write symmetry the policy edit flow depends on: the fields a
// client reads back from a policy must be accepted verbatim on the way back in, since the UI echoes
// them into updatePolicy unchanged.
func TestScopeRuleRoundTrips(t *testing.T) {
	rules := []*models.ScopeRule{
		{Type: models.ScopeRuleTypeWorkspace, Action: models.ScopeRuleActionInclude, Pattern: "acme/*"},
		{Type: models.ScopeRuleTypeManagedIdentity, Action: models.ScopeRuleActionExclude, Pattern: "acme/prod-mi"},
	}

	for _, rule := range rules {
		t.Run(string(rule.Type), func(t *testing.T) {
			resolved := &PolicyScopeRuleResolver{rule: rule}

			input := &PolicyScopeRuleInput{
				Type:    resolved.Type(),
				Action:  resolved.Action(),
				Pattern: resolved.Pattern(),
			}

			assert.Equal(t, rule, input.toModelScopeRule())
		})
	}
}
