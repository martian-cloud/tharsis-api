package models

// StateVersion represents a specific version of the the terraform state associated with a workspace
type StateVersion struct {
	WorkspaceID string
	RunID       *string
	CreatedBy   string
	Metadata    ResourceMetadata
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *StateVersion) ResolveMetadata(key string) (string, error) {
	return s.Metadata.resolveFieldValue(key)
}
