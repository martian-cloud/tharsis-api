package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*LogStreamChunk)(nil)

// LogStreamChunk represents a contiguous chunk of a log stream stored as a single object-store file.
type LogStreamChunk struct {
	Metadata    ResourceMetadata
	LogStreamID string
	ObjectKey   string
	ChunkIndex  int
	StartOffset int
	Size        int
	Sealed      bool
}

// GetID returns the Metadata ID.
func (c *LogStreamChunk) GetID() string {
	return c.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (c *LogStreamChunk) GetGlobalID() string {
	return gid.ToGlobalID(c.GetModelType(), c.Metadata.ID)
}

// GetModelType returns the type of the model.
func (c *LogStreamChunk) GetModelType() types.ModelType {
	return types.LogStreamChunkModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (c *LogStreamChunk) ResolveMetadata(key string) (*string, error) {
	return c.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (c *LogStreamChunk) Validate() error {
	return nil
}
