package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var (
	_ Model = (*Variable)(nil)
	_ Model = (*VariableVersion)(nil)
)

// VariableCategory specifies if the variable is a terraform
// or environment variable
type VariableCategory string

// Variable category Status Types
const (
	TerraformVariableCategory   VariableCategory = "terraform"
	EnvironmentVariableCategory VariableCategory = "environment"
)

// Variable resource
type Variable struct {
	Value           *string
	SecretData      []byte
	Category        VariableCategory
	NamespacePath   string
	Key             string
	Metadata        ResourceMetadata
	Hcl             bool
	Sensitive       bool
	LatestVersionID string
}

// GetID returns the Metadata ID.
func (v *Variable) GetID() string {
	return v.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (v *Variable) GetGlobalID() string {
	return gid.ToGlobalID(v.GetModelType(), v.Metadata.ID)
}

// GetModelType returns the model type
func (v *Variable) GetModelType() types.ModelType {
	return types.VariableModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (v *Variable) ResolveMetadata(key string) (*string, error) {
	val, err := v.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "namespace_path":
			return &v.NamespacePath, nil
		case "key":
			return &v.Key, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate validates the variable
func (v *Variable) Validate() error {
	return nil
}

// VariableVersion resource
type VariableVersion struct {
	VariableID string
	Value      *string
	Key        string
	Metadata   ResourceMetadata
	Hcl        bool
	// SecretData is only used for sensitive variables and it stores data
	// returned by the configured secret manager plugin
	SecretData []byte
}

// GetID returns the Metadata ID.
func (v *VariableVersion) GetID() string {
	return v.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (v *VariableVersion) GetGlobalID() string {
	return gid.ToGlobalID(types.VariableVersionModelType, v.Metadata.ID)
}

// GetModelType returns the model type
func (v *VariableVersion) GetModelType() types.ModelType {
	return types.VariableVersionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (v *VariableVersion) ResolveMetadata(key string) (*string, error) {
	return v.Metadata.resolveFieldValue(key)
}

// Validate validates the variable version
func (v *VariableVersion) Validate() error {
	return nil
}
