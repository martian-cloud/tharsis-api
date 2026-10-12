package models

import (
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*LogStreamChunk)(nil)

// LogStreamChunk represents a contiguous chunk of a log stream stored as a single object-store file.
type LogStreamChunk struct {
	Metadata ResourceMetadata
	// LeaseExpiresAt is set while a writer holds the chunk's write lease (see
	// LogStreamChunks.ClaimLogStreamChunkLease). Nil when the chunk is not leased.
	LeaseExpiresAt *time.Time
	// Checksum is the CRC-32 (IEEE) of the chunk's committed bytes [0, Size). Nil when unknown (a chunk
	// written before checksums were tracked), in which case it is not verified.
	Checksum       *uint32
	LogStreamID    string
	ObjectStoreKey string
	ChunkIndex     int
	StartOffset    int
	Size           int
	Sealed         bool
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
