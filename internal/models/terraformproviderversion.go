package models

import (
	"fmt"
)

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

// GetHexGPGKeyID returns the GPG key id in hex format
func (t *TerraformProviderVersion) GetHexGPGKeyID() *string {
	if t.GPGKeyID != nil {
		hex := fmt.Sprintf("%016X", *t.GPGKeyID)
		return &hex
	}
	return nil
}
