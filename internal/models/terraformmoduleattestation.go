package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformModuleAttestation)(nil)

// TerraformModuleAttestation represents a terraform module attestation
type TerraformModuleAttestation struct {
	CreatedBy     string
	Description   string
	ModuleID      string
	SchemaType    string
	PredicateType string
	Data          string
	DataSHASum    []byte
	Metadata      ResourceMetadata
	Digests       []string
}

// GetID returns the Metadata ID.
func (t *TerraformModuleAttestation) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformModuleAttestation) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the model type.
func (t *TerraformModuleAttestation) GetModelType() types.ModelType {
	return types.TerraformModuleAttestationModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformModuleAttestation) ResolveMetadata(key string) (*string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "predicate":
			return &t.PredicateType, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *TerraformModuleAttestation) Validate() error {
	// Verify description satisfies constraints
	return verifyValidDescription(t.Description)
}
