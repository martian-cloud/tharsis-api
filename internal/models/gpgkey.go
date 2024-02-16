package models

import (
	"fmt"
	"strings"
)

// GPGKey represents a GPG key used for signing
type GPGKey struct {
	CreatedBy    string
	GroupID      string
	ASCIIArmor   string
	Fingerprint  string
	ResourcePath string
	Metadata     ResourceMetadata
	GPGKeyID     uint64
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (g *GPGKey) ResolveMetadata(key string) (string, error) {
	val, err := g.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			val = g.GetGroupPath()
		default:
			return "", err
		}
	}

	return val, nil
}

// GetHexGPGKeyID returns the GPG key ID in hex format
func (g *GPGKey) GetHexGPGKeyID() string {
	return fmt.Sprintf("%016X", g.GPGKeyID)
}

// GetGroupPath returns the group path
func (g *GPGKey) GetGroupPath() string {
	return g.ResourcePath[:strings.LastIndex(g.ResourcePath, "/")]
}
