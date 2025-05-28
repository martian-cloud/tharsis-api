package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*LogStream)(nil)

// LogStream represents a stream of logs
type LogStream struct {
	JobID           *string
	RunnerSessionID *string
	Metadata        ResourceMetadata
	Size            int
	Completed       bool
}

// GetID returns the Metadata ID.
func (ls *LogStream) GetID() string {
	return ls.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (ls *LogStream) GetGlobalID() string {
	return gid.ToGlobalID(ls.GetModelType(), ls.Metadata.ID)
}

// GetModelType returns the type of the model.
func (ls *LogStream) GetModelType() types.ModelType {
	return types.LogStreamModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (ls *LogStream) ResolveMetadata(key string) (*string, error) {
	return ls.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (ls *LogStream) Validate() error {
	return nil
}
