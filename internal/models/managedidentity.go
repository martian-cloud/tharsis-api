package models

import "strings"

// ManagedIdentityType represents the supported managed identity types
type ManagedIdentityType string

// Managed Identity Types
const (
	ManagedIdentityAzureFederated   ManagedIdentityType = "azure_federated"
	ManagedIdentityAWSFederated     ManagedIdentityType = "aws_federated"
	ManagedIdentityTharsisFederated ManagedIdentityType = "tharsis_federated"
)

// ManagedIdentityAccessRule is used to restrict access to a managed identity
type ManagedIdentityAccessRule struct {
	Metadata                 ResourceMetadata
	RunStage                 JobType
	ManagedIdentityID        string
	AllowedUserIDs           []string
	AllowedServiceAccountIDs []string
	AllowedTeamIDs           []string
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
