package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*StateVersion)(nil)

// StateVersion represents a specific version of the the terraform state associated with a workspace
type StateVersion struct {
	WorkspaceID string
	RunID       *string
	CreatedBy   string
	Metadata    ResourceMetadata
}

// GetID returns the Metadata ID.
func (s *StateVersion) GetID() string {
	return s.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (s *StateVersion) GetGlobalID() string {
	return gid.ToGlobalID(s.GetModelType(), s.Metadata.ID)
}

// GetModelType returns the type of model.
func (s *StateVersion) GetModelType() types.ModelType {
	return types.StateVersionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *StateVersion) ResolveMetadata(key string) (*string, error) {
	return s.Metadata.resolveFieldValue(key)
}

// Validate performs validation on the model.
func (s *StateVersion) Validate() error {
	return nil
}
