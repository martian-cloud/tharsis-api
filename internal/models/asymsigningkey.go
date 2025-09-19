package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*AsymSigningKey)(nil)

// AsymSigningKeyStatus represents the various states for a asymmetric signing key resource
type AsymSigningKeyStatus string

// AsymSigningKey status types
const (
	AsymSigningKeyStatusCreating        AsymSigningKeyStatus = "creating"
	AsymSigningKeyStatusActive          AsymSigningKeyStatus = "active"
	AsymSigningKeyStatusDecommissioning AsymSigningKeyStatus = "decommissioning"
)

// AsymSigningKey represents a GPG key used for signing
type AsymSigningKey struct {
	PublicKey  []byte
	PluginData []byte
	Metadata   ResourceMetadata
	Status     AsymSigningKeyStatus
	PubKeyID   string
	PluginType string
}

// GetID returns the Metadata ID.
func (a *AsymSigningKey) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *AsymSigningKey) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the Model's type
func (a *AsymSigningKey) GetModelType() types.ModelType {
	return types.AsymSigningKeyModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *AsymSigningKey) ResolveMetadata(key string) (*string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate validates the resource
func (a *AsymSigningKey) Validate() error {
	return nil
}
