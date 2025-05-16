package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*StateVersionOutput)(nil)

// StateVersionOutput represents a terraform state version output
type StateVersionOutput struct {
	Name           string
	StateVersionID string
	Metadata       ResourceMetadata
	Value          []byte
	Type           []byte
	Sensitive      bool
}

// GetID returns the Metadata ID.
func (s *StateVersionOutput) GetID() string {
	return s.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (s *StateVersionOutput) GetGlobalID() string {
	return gid.ToGlobalID(s.GetModelType(), s.Metadata.ID)
}

// GetModelType returns the model type.
func (s *StateVersionOutput) GetModelType() types.ModelType {
	return types.StateVersionOutputModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *StateVersionOutput) ResolveMetadata(key string) (string, error) {
	return s.Metadata.resolveFieldValue(key)
}

// Validate performs validation on the model.
func (s *StateVersionOutput) Validate() error {
	return nil
}
