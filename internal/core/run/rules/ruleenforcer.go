// Package rules package
package rules

//go:generate go tool mockery --name RuleEnforcer --inpackage --case underscore

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// RuleEnforcer is used to enforce managed identity access rules
type RuleEnforcer interface {
	EnforceRules(ctx context.Context, managedIdentity *models.ManagedIdentity, input *RunDetails) error
}

type ruleTypeHandler func(ctx context.Context, dbClient *db.Client, rule *models.ManagedIdentityAccessRule, input *RunDetails) (string, error)

// RunDetails is the input for enforcing rules
type RunDetails struct {
	ModuleSource          registry.ModuleRegistrySource
	CurrentStateVersionID *string
	RunStage              models.JobType
	ModuleDigest          []byte
	ModuleSemanticVersion *string
}

type ruleEnforcer struct {
	dbClient   *db.Client
	handlerMap map[models.ManagedIdentityAccessRuleType]ruleTypeHandler
}

// NewRuleEnforcer returns a new RuleEnforcer instance
func NewRuleEnforcer(
	dbClient *db.Client,
) RuleEnforcer {
	handlerMap := map[models.ManagedIdentityAccessRuleType]ruleTypeHandler{
		models.ManagedIdentityAccessRuleEligiblePrincipals: enforceEligiblePrincipalsRuleType,
		models.ManagedIdentityAccessRuleModuleAttestation:  enforceModuleAttestationRuleType,
	}

	return &ruleEnforcer{
		dbClient:   dbClient,
		handlerMap: handlerMap,
	}
}

// EnforceRules verifies all the managed identity rules are satisfied. An error will be returned if any rules do
// not pass.
func (r *ruleEnforcer) EnforceRules(ctx context.Context, managedIdentity *models.ManagedIdentity, input *RunDetails) error {
	results, err := r.dbClient.ManagedIdentities.GetManagedIdentityAccessRules(ctx,
		&db.GetManagedIdentityAccessRulesInput{
			Filter: &db.ManagedIdentityAccessRuleFilter{
				ManagedIdentityID: &managedIdentity.Metadata.ID,
			},
		})
	if err != nil {
		return err
	}

	ruleMap := map[models.ManagedIdentityAccessRuleType][]models.ManagedIdentityAccessRule{}

	// Filter rules by run stage and group rules by type
	for _, rule := range results.ManagedIdentityAccessRules {
		if rule.RunStage == input.RunStage {
			if _, ok := ruleMap[rule.Type]; !ok {
				ruleMap[rule.Type] = []models.ManagedIdentityAccessRule{}
			}

			ruleMap[rule.Type] = append(ruleMap[rule.Type], rule)
		}
	}

	// Rules of different types use an AND condition and must all pass
	for _, rules := range ruleMap {
		if err := r.enforceRules(ctx, managedIdentity, input, rules); err != nil {
			return err
		}
	}

	return nil
}

func (r *ruleEnforcer) enforceRules(ctx context.Context, managedIdentity *models.ManagedIdentity, input *RunDetails, rules []models.ManagedIdentityAccessRule) error {
	// Rules of the same type use an OR condition (i.e. first successful rule will pass)
	diagnostics := []string{}
	for i, rule := range rules {
		ruleCopy := rule

		handler, ok := r.handlerMap[rule.Type]
		if !ok {
			return fmt.Errorf("received unsupported managed identity rule type %s", rule.Type)
		}

		diag, err := handler(ctx, r.dbClient, &ruleCopy, input)
		if err != nil {
			return err
		}

		if diag == "" {
			// Break out of loop on first rule that passes since rules of the same type use an OR condition
			break
		}

		diagnostics = append(diagnostics, diag)

		// rule was not satisfied
		if i == (len(rules) - 1) {
			// this is the last rule
			return errors.New(
				"managed identity rule for %s not satisfied for run stage %s and managed identity %s: %s",
				rule.Type,
				rule.RunStage,
				managedIdentity.GetResourcePath(),
				strings.Join(diagnostics, ": "),
				errors.WithErrorCode(errors.EForbidden),
			)
		}
	}
	return nil
}

func enforceEligiblePrincipalsRuleType(
	ctx context.Context,
	_ *db.Client,
	rule *models.ManagedIdentityAccessRule,
	_ *RunDetails,
) (string, error) {
	// Check if subject is allowed to use this managed identity
	if err := auth.HandleCaller(
		ctx,
		func(ctx context.Context, c *auth.UserCaller) error {
			eligible, err := userIsEligiblePrincipal(ctx, c, rule.AllowedUserIDs, rule.AllowedTeamIDs)
			if err != nil {
				return err
			}
			if !eligible {
				return fmt.Errorf("user %s is not an eligible principal", c.User.Username)
			}
			return nil
		},
		func(_ context.Context, c *auth.ServiceAccountCaller) error {
			if !serviceAccountIsEligiblePrincipal(c, rule.AllowedServiceAccountIDs) {
				return fmt.Errorf("service account %s is not an eligible principal", c.ServiceAccountPath)
			}
			return nil
		},
	); err != nil {
		return err.Error(), nil
	}
	return "", nil
}

// userIsEligiblePrincipal reports whether the user caller matches one of the allowed user IDs
// directly, or has live membership in one of the allowed teams. Team membership is resolved
// live via GetTeams so a subject removed from a team loses eligibility immediately.
func userIsEligiblePrincipal(ctx context.Context, c *auth.UserCaller, allowedUserIDs, allowedTeamIDs []string) (bool, error) {
	for _, userID := range allowedUserIDs {
		if c.User.Metadata.ID == userID {
			return true, nil
		}
	}

	if len(allowedTeamIDs) == 0 {
		return false, nil
	}

	// Check whether there is an intersection between the calling user's teams and the allowed teams.
	userCallerTeams, err := c.GetTeams(ctx)
	if err != nil {
		return false, err
	}
	userCallerTeamsMap := map[string]bool{}
	for _, callerTeam := range userCallerTeams {
		userCallerTeamsMap[callerTeam.Metadata.ID] = true
	}
	for _, teamID := range allowedTeamIDs {
		if userCallerTeamsMap[teamID] {
			return true, nil
		}
	}

	return false, nil
}

// serviceAccountIsEligiblePrincipal reports whether the service account caller matches one of
// the allowed service account IDs.
func serviceAccountIsEligiblePrincipal(c *auth.ServiceAccountCaller, allowedServiceAccountIDs []string) bool {
	for _, serviceAccountID := range allowedServiceAccountIDs {
		if c.ServiceAccountID == serviceAccountID {
			return true
		}
	}
	return false
}

// EligiblePrincipal reports whether the caller (a user or service account) is an eligible
// principal against the given allowed principal ID sets: a user matches by user ID or by live
// membership in one of the allowed teams; a service account matches by service account ID.
// It reuses the same matching logic as the managed-identity eligible-principals rule so run
// gate approval authorization stays consistent with rule enforcement.
func EligiblePrincipal(ctx context.Context, allowedUserIDs, allowedServiceAccountIDs, allowedTeamIDs []string) (bool, error) {
	eligible := false
	if err := auth.HandleCaller(
		ctx,
		func(ctx context.Context, c *auth.UserCaller) error {
			ok, err := userIsEligiblePrincipal(ctx, c, allowedUserIDs, allowedTeamIDs)
			if err != nil {
				return err
			}
			eligible = ok
			return nil
		},
		func(_ context.Context, c *auth.ServiceAccountCaller) error {
			eligible = serviceAccountIsEligiblePrincipal(c, allowedServiceAccountIDs)
			return nil
		},
	); err != nil {
		return false, err
	}
	return eligible, nil
}

// enforceModuleAttestationRuleType checks a managed identity's module attestation rule
func enforceModuleAttestationRuleType(ctx context.Context, dbClient *db.Client, rule *models.ManagedIdentityAccessRule, input *RunDetails) (string, error) {
	if input.ModuleSource == nil || !input.ModuleSource.IsTharsisModule() {
		return "managed identity module attestation rule is only supported for modules in a tharsis registry", nil
	}

	if input.ModuleDigest == nil {
		return "", errors.New("module digest must be defined when checking module attestation rules for a module in the Tharsis registry")
	}

	if input.ModuleSemanticVersion == nil {
		return "", errors.New("module semantic version must be defined when checking module attestation rules for a module in the Tharsis registry")
	}

	if rule.VerifyStateLineage {
		diag, err := verifyStateLineage(ctx, dbClient, input.CurrentStateVersionID, input.ModuleSource)
		if err != nil {
			return "", err
		}
		if diag != "" {
			return diag, nil
		}
	}

	moduleDigest := hex.EncodeToString(input.ModuleDigest)

	attestations, err := input.ModuleSource.GetAttestations(ctx, *input.ModuleSemanticVersion, moduleDigest)
	if err != nil {
		return "", err
	}

	// Every attestation policy on the rule must be satisfied (AND), unlike the single public key
	// a ModuleAttestation policy carries.
	for _, policy := range rule.ModuleAttestationPolicies {
		diag, err := verifyAttestationSatisfied(ctx, attestations, moduleDigest, policy.PublicKey, policy.PredicateType)
		if err != nil {
			return "", err
		}
		if diag != "" {
			return fmt.Sprintf("no attestation is present for module matching managed identity rule: %s", diag), nil
		}
	}

	return "", nil
}
