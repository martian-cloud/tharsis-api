package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*ResourceLimit)(nil)

// ResourceLimit represents a resource limit
type ResourceLimit struct {
	Name     string
	Metadata ResourceMetadata
	Value    int
}

// GetID returns the Metadata ID.
func (r *ResourceLimit) GetID() string {
	return r.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (r *ResourceLimit) GetGlobalID() string {
	return gid.ToGlobalID(r.GetModelType(), r.Metadata.ID)
}

// GetModelType returns the ModelType.
func (r *ResourceLimit) GetModelType() types.ModelType {
	return types.ResourceLimitModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *ResourceLimit) ResolveMetadata(key string) (string, error) {
	return r.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (r *ResourceLimit) Validate() error {
	return nil
}
