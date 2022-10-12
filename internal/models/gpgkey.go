package models

import "fmt"

// GPGKey represents a GPG key used for signing
type GPGKey struct {
	CreatedBy   string
	GroupID     string
	ASCIIArmor  string
	Fingerprint string
	Metadata    ResourceMetadata
	GPGKeyID    uint64
}

// GetHexGPGKeyID returns the GPG key ID in hex format
func (g *GPGKey) GetHexGPGKeyID() string {
	return fmt.Sprintf("%016X", g.GPGKeyID)
}
