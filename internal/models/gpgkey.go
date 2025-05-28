package models

import (
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*GPGKey)(nil)

// GPGKey represents a GPG key used for signing
type GPGKey struct {
	CreatedBy   string
	GroupID     string
	ASCIIArmor  string
	Fingerprint string
	Metadata    ResourceMetadata
	GPGKeyID    uint64
}

// GetID returns the Metadata ID.
func (g *GPGKey) GetID() string {
	return g.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (g *GPGKey) GetGlobalID() string {
	return gid.ToGlobalID(g.GetModelType(), g.Metadata.ID)
}

// GetModelType returns the Model's type
func (g *GPGKey) GetModelType() types.ModelType {
	return types.GPGKeyModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (g *GPGKey) ResolveMetadata(key string) (*string, error) {
	val, err := g.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			path := g.GetGroupPath()
			return &path, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate validates the resource
func (g *GPGKey) Validate() error {
	return nil
}

// GetHexGPGKeyID returns the GPG key ID in hex format
func (g *GPGKey) GetHexGPGKeyID() string {
	return fmt.Sprintf("%016X", g.GPGKeyID)
}

// GetResourcePath returns the path to the GPG Key resource
func (g *GPGKey) GetResourcePath() string {
	return strings.Split(g.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetGroupPath returns the group path
func (g *GPGKey) GetGroupPath() string {
	resourcePath := g.GetResourcePath()
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}
