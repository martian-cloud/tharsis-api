package rules

//go:generate mockery --name RuleEnforcer --inpackage --case underscore

import (
	"context"
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	ssldsse "github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// RuleEnforcer is used to enforce managed identity access rules
type RuleEnforcer interface {
	EnforceRules(ctx context.Context, managedIdentity *models.ManagedIdentity, input *RunDetails) error
}

type ruleTypeHandler func(ctx context.Context, dbClient *db.Client, rule *models.ManagedIdentityAccessRule, input *RunDetails) (string, error)

// RunDetails is the input for enforcing rules
type RunDetails struct {
	RunStage     models.JobType
	ModuleID     *string
	ModuleDigest []byte
}

type ruleEnforcer struct {
	dbClient   *db.Client
	handlerMap map[models.ManagedIdentityAccessRuleType]ruleTypeHandler
}

// NewRuleEnforcer returns a new RuleEnforcer instance
func NewRuleEnforcer(dbClient *db.Client) RuleEnforcer {
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
			return terrors.NewError(
				terrors.EForbidden,
				fmt.Sprintf(
					"managed identity rule for %s not satisfied for run stage %s and managed identity %s: %s",
					rule.Type,
					rule.RunStage,
					managedIdentity.ResourcePath,
					strings.Join(diagnostics, ": ")),
			)
		}
	}
	return nil
}

func enforceEligiblePrincipalsRuleType(ctx context.Context, _ *db.Client, rule *models.ManagedIdentityAccessRule, _ *RunDetails) (string, error) {
	// Check if subject is allowed to use this managed identity
	if err := auth.HandleCaller(
		ctx,
		func(ctx context.Context, c *auth.UserCaller) error {
			found := false
			for _, userID := range rule.AllowedUserIDs {
				if c.User.Metadata.ID == userID {
					found = true
					break
				}
			}

			// Check whether there is an intersection between the
			// calling user's teams and this access rule's allowed teams.
			userCallerTeams, err := c.GetTeams(ctx)
			if err != nil {
				return err
			}
			// The time spent converting from slice to map is expected to be minor.
			userCallerTeamsMap := map[string]bool{}
			for _, callerTeamID := range userCallerTeams {
				userCallerTeamsMap[callerTeamID.Metadata.ID] = true
			}
			for _, teamID := range rule.AllowedTeamIDs {
				if _, ok := userCallerTeamsMap[teamID]; ok {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("user %s is not an eligible principal", c.User.Username)
			}

			return nil
		},
		func(_ context.Context, c *auth.ServiceAccountCaller) error {
			found := false
			for _, serviceAccountID := range rule.AllowedServiceAccountIDs {
				if c.ServiceAccountID == serviceAccountID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("service account %s is not an eligible principal", c.ServiceAccountPath)
			}

			return nil
		},
	); err != nil {
		return err.Error(), nil
	}
	return "", nil
}

func enforceModuleAttestationRuleType(ctx context.Context, dbClient *db.Client, rule *models.ManagedIdentityAccessRule, input *RunDetails) (string, error) {
	if input.ModuleID == nil {
		return "managed identity module attestation rule only allows modules in the Tharsis registry", nil
	}

	if input.ModuleDigest == nil {
		return "", errors.New("module digest must be defined when checking module attestation rules for a module in the Tharsis registry")
	}

	moduleDigest := hex.EncodeToString(input.ModuleDigest)

	attestations, err := dbClient.TerraformModuleAttestations.GetModuleAttestations(ctx, &db.GetModuleAttestationsInput{
		Filter: &db.TerraformModuleAttestationFilter{
			ModuleID: input.ModuleID,
			Digest:   &moduleDigest,
		},
	})
	if err != nil {
		return "", err
	}

	diagnostics := []string{}

	// Verify that all attestation policies for the rule are satisfied
	for _, policy := range rule.ModuleAttestationPolicies {
		foundMatch := false

		pub, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(policy.PublicKey))
		if err != nil {
			return "", err
		}

		verifier, err := signature.LoadVerifier(pub, crypto.SHA256)
		if err != nil {
			return "", err
		}

		for _, attestation := range attestations.ModuleAttestations {
			decodedSig, err := base64.StdEncoding.DecodeString(attestation.Data)
			if err != nil {
				return "", fmt.Errorf("failed to decode attestation signature: %v", err)
			}

			// Verify the signature on the attestation against the provided public key
			env := ssldsse.Envelope{}
			if err = json.Unmarshal(decodedSig, &env); err != nil {
				return "", fmt.Errorf("failed to unmarshal dsse envelope: %v", err)
			}

			dssev, err := ssldsse.NewEnvelopeVerifier(&dsse.VerifierAdapter{SignatureVerifier: verifier})
			if err != nil {
				return "", fmt.Errorf("failed to create new dsse envelope verifier: %v", err)
			}

			// Verify signature
			if _, err = dssev.Verify(&env); err != nil {
				diagnostics = append(diagnostics, "signature is not valid for required public key")
				continue
			}

			// Get the expected digest from the attestation
			decodedPredicate, err := base64.StdEncoding.DecodeString(env.Payload)
			if err != nil {
				return "", fmt.Errorf("failed to decode dsse payload: %v", err)
			}
			var statement in_toto.Statement
			if err := json.Unmarshal(decodedPredicate, &statement); err != nil {
				return "", fmt.Errorf("failed to decode attestation predicate: %v", err)
			}

			// Compare the actual and expected
			if statement.Subject == nil {
				diagnostics = append(diagnostics, "no subject in intoto statement")
				continue
			}

			// Verify a subject exists that matches the module digest
			foundSubject := false
			for _, subj := range statement.Subject {
				shaSum, ok := subj.Digest["sha256"]
				if ok && (shaSum == moduleDigest) {
					foundSubject = true
					break
				}
			}

			if !foundSubject {
				diagnostics = append(diagnostics, fmt.Sprintf("subject with digest %s not found in module attestation", moduleDigest))
				continue
			}

			// Verify predicate type if it's defined in the policy
			if policy.PredicateType != nil && statement.PredicateType != *policy.PredicateType {
				diagnostics = append(diagnostics, fmt.Sprintf("invalid predicate type, expected=%s actual=%s", *policy.PredicateType, statement.PredicateType))
				continue
			}

			foundMatch = true
			break
		}

		if !foundMatch {
			return fmt.Sprintf("no attestation is present for module matching managed identity rule: %s", strings.Join(diagnostics, ": ")), nil
		}
	}

	return "", nil
}
