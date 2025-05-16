package models

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformProviderVersion)(nil)

// TerraformProviderVersion represents a version of a terraform provider
type TerraformProviderVersion struct {
	GPGASCIIArmor            *string
	GPGKeyID                 *uint64
	CreatedBy                string
	ProviderID               string
	SemanticVersion          string
	Metadata                 ResourceMetadata
	Protocols                []string
	SHASumsUploaded          bool
	SHASumsSignatureUploaded bool
	ReadmeUploaded           bool
	Latest                   bool
}

// GetID returns the Metadata ID.
func (t *TerraformProviderVersion) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformProviderVersion) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformProviderVersion) GetModelType() types.ModelType {
	return types.TerraformProviderVersionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderVersion) ResolveMetadata(key string) (string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "sem_version":
			val = t.SemanticVersion
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate validates the model.
func (t *TerraformProviderVersion) Validate() error {
	return nil
}

// GetHexGPGKeyID returns the GPG key id in hex format
func (t *TerraformProviderVersion) GetHexGPGKeyID() *string {
	if t.GPGKeyID != nil {
		hex := fmt.Sprintf("%016X", *t.GPGKeyID)
		return &hex
	}
	return nil
}
