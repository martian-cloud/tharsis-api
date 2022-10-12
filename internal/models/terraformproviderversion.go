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

// GetHexGPGKeyID returns the GPG key id in hex format
func (t *TerraformProviderVersion) GetHexGPGKeyID() *string {
	if t.GPGKeyID != nil {
		hex := fmt.Sprintf("%016X", *t.GPGKeyID)
		return &hex
	}
	return nil
}
