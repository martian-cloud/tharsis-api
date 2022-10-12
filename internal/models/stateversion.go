package models

// StateVersion represents a specific version of the the terraform state associated with a workspace
type StateVersion struct {
	WorkspaceID string
	RunID       *string
	CreatedBy   string
	Metadata    ResourceMetadata
}
