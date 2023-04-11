package models

import (
	"strings"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// ManagedIdentityType represents the supported managed identity types
type ManagedIdentityType string

// Managed Identity Types
const (
	ManagedIdentityAzureFederated   ManagedIdentityType = "azure_federated"
	ManagedIdentityAWSFederated     ManagedIdentityType = "aws_federated"
	ManagedIdentityTharsisFederated ManagedIdentityType = "tharsis_federated"
)

// ManagedIdentityAccessRuleType represents the supported managed identity rule types
type ManagedIdentityAccessRuleType string

// Managed Identity Rule Types
const (
	ManagedIdentityAccessRuleEligiblePrincipals ManagedIdentityAccessRuleType = "eligible_principals"
	ManagedIdentityAccessRuleModuleAttestation  ManagedIdentityAccessRuleType = "module_attestation"
)

// ManagedIdentityAccessRuleModuleAttestationPolicy is used in access rules to verify that a
// module has an in-toto attestation that is signed with the specified public key and an optional
// predicate type
type ManagedIdentityAccessRuleModuleAttestationPolicy struct {
	PredicateType *string `json:"predicateType,omitempty"`
	PublicKey     string  `json:"publicKey"`
}

// ManagedIdentityAccessRule is used to restrict access to a managed identity
type ManagedIdentityAccessRule struct {
	Metadata                  ResourceMetadata
	Type                      ManagedIdentityAccessRuleType
	RunStage                  JobType
	ManagedIdentityID         string
	ModuleAttestationPolicies []ManagedIdentityAccessRuleModuleAttestationPolicy
	AllowedUserIDs            []string
	AllowedServiceAccountIDs  []string
	AllowedTeamIDs            []string
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (m *ManagedIdentityAccessRule) ResolveMetadata(key string) (string, error) {
	return m.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (m *ManagedIdentityAccessRule) Validate() error {
	switch m.Type {
	case ManagedIdentityAccessRuleEligiblePrincipals:
		if len(m.ModuleAttestationPolicies) > 0 {
			return errors.New(errors.EInvalid, "eligible principals rule type does not support module attestation policies")
		}
	case ManagedIdentityAccessRuleModuleAttestation:
		if len(m.ModuleAttestationPolicies) == 0 {
			return errors.New(errors.EInvalid, "a minimum of one module attestation policy is required for rule type module_attestation")
		}

		for _, policy := range m.ModuleAttestationPolicies {
			if _, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(policy.PublicKey)); err != nil {
				return errors.Wrap(err, errors.EInvalid, "invalid public key")
			}
			if policy.PredicateType != nil && *policy.PredicateType == "" {
				return errors.New(errors.EInvalid, "predicate type cannot be an empty string")
			}
		}

		if len(m.AllowedServiceAccountIDs) > 0 || len(m.AllowedUserIDs) > 0 || len(m.AllowedTeamIDs) > 0 {
			return errors.New(errors.EInvalid, "module attestation rule type does not support allowed users, service accounts, or teams")
		}
	default:
		return errors.New(errors.EInvalid, "rule type %s is not supported", m.Type)
	}

	return nil
}

// ManagedIdentity is used to provide identities to terraform providers
type ManagedIdentity struct {
	Type          ManagedIdentityType
	ResourcePath  string
	Name          string
	Description   string
	GroupID       string
	CreatedBy     string
	AliasSourceID *string
	Metadata      ResourceMetadata
	Data          []byte
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (m *ManagedIdentity) ResolveMetadata(key string) (string, error) {
	return m.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (m *ManagedIdentity) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(m.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	return verifyValidDescription(m.Description)
}

// GetGroupPath returns the group path
func (m *ManagedIdentity) GetGroupPath() string {
	return m.ResourcePath[:strings.LastIndex(m.ResourcePath, "/")]
}

// IsAlias returns true is managed identity is an alias.
func (m *ManagedIdentity) IsAlias() bool {
	return m.AliasSourceID != nil
}
