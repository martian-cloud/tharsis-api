package models

import (
	"strings"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var (
	_ Model = (*ManagedIdentity)(nil)
	_ Model = (*ManagedIdentityAccessRule)(nil)
)

// ManagedIdentityType represents the supported managed identity types
type ManagedIdentityType string

// Managed Identity Types
const (
	ManagedIdentityAzureFederated      ManagedIdentityType = "azure_federated"
	ManagedIdentityAWSFederated        ManagedIdentityType = "aws_federated"
	ManagedIdentityTharsisFederated    ManagedIdentityType = "tharsis_federated"
	ManagedIdentityKubernetesFederated ManagedIdentityType = "kubernetes_federated"
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
	VerifyStateLineage        bool
}

// GetID returns the Metadata ID.
func (m *ManagedIdentityAccessRule) GetID() string {
	return m.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (m *ManagedIdentityAccessRule) GetGlobalID() string {
	return gid.ToGlobalID(m.GetModelType(), m.Metadata.ID)
}

// GetModelType returns the model type.
func (m *ManagedIdentityAccessRule) GetModelType() types.ModelType {
	return types.ManagedIdentityAccessRuleModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (m *ManagedIdentityAccessRule) ResolveMetadata(key string) (*string, error) {
	return m.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (m *ManagedIdentityAccessRule) Validate() error {
	switch m.Type {
	case ManagedIdentityAccessRuleEligiblePrincipals:
		if len(m.ModuleAttestationPolicies) > 0 {
			return errors.New("eligible principals rule type does not support module attestation policies", errors.WithErrorCode(errors.EInvalid))
		}
	case ManagedIdentityAccessRuleModuleAttestation:
		if len(m.ModuleAttestationPolicies) == 0 {
			return errors.New("a minimum of one module attestation policy is required for rule type module_attestation", errors.WithErrorCode(errors.EInvalid))
		}

		for _, policy := range m.ModuleAttestationPolicies {
			if _, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(policy.PublicKey)); err != nil {
				return errors.Wrap(err, "invalid public key", errors.WithErrorCode(errors.EInvalid))
			}
			if policy.PredicateType != nil && *policy.PredicateType == "" {
				return errors.New("predicate type cannot be an empty string", errors.WithErrorCode(errors.EInvalid))
			}
		}

		if len(m.AllowedServiceAccountIDs) > 0 || len(m.AllowedUserIDs) > 0 || len(m.AllowedTeamIDs) > 0 {
			return errors.New("module attestation rule type does not support allowed users, service accounts, or teams", errors.WithErrorCode(errors.EInvalid))
		}
	default:
		return errors.New("rule type %s is not supported", m.Type, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// ManagedIdentity is used to provide identities to terraform providers
type ManagedIdentity struct {
	Type          ManagedIdentityType
	Name          string
	Description   string
	GroupID       string
	CreatedBy     string
	AliasSourceID *string
	Metadata      ResourceMetadata
	Data          []byte
}

// GetID returns the Metadata ID.
func (m *ManagedIdentity) GetID() string {
	return m.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (m *ManagedIdentity) GetGlobalID() string {
	return gid.ToGlobalID(m.GetModelType(), m.Metadata.ID)
}

// GetModelType returns the model type.
func (m *ManagedIdentity) GetModelType() types.ModelType {
	return types.ManagedIdentityModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (m *ManagedIdentity) ResolveMetadata(key string) (*string, error) {
	val, err := m.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			path := m.GetGroupPath()
			return &path, nil
		default:
			return nil, err
		}
	}

	return val, nil
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

// GetResourcePath returns the resource path
func (m *ManagedIdentity) GetResourcePath() string {
	return strings.Split(m.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetGroupPath returns the group path
func (m *ManagedIdentity) GetGroupPath() string {
	resourcePath := m.GetResourcePath()
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}

// IsAlias returns true is managed identity is an alias.
func (m *ManagedIdentity) IsAlias() bool {
	return m.AliasSourceID != nil
}
