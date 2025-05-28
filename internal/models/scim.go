package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*SCIMToken)(nil)

// SCIMToken represents a SCIM token.
type SCIMToken struct {
	Nonce     string
	CreatedBy string
	Metadata  ResourceMetadata
}

// GetID returns the Metadata ID.
func (s *SCIMToken) GetID() string {
	return s.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (s *SCIMToken) GetGlobalID() string {
	return gid.ToGlobalID(s.GetModelType(), s.Metadata.ID)
}

// GetModelType returns the type of the model.
func (s *SCIMToken) GetModelType() types.ModelType {
	return types.SCIMTokenModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *SCIMToken) ResolveMetadata(key string) (*string, error) {
	return s.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (s *SCIMToken) Validate() error {
	return nil
}
