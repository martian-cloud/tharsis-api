package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*FederatedRegistry)(nil)

// FederatedRegistry represents a client-side federated registry.
type FederatedRegistry struct {
	Metadata ResourceMetadata
	Hostname string
	GroupID  string
	Audience string
}

// GetID returns the ID of the FederatedRegistry resource
func (f *FederatedRegistry) GetID() string {
	return f.Metadata.ID
}

// GetGlobalID returns the GID of the FederatedRegistry resource
func (f *FederatedRegistry) GetGlobalID() string {
	return gid.ToGlobalID(f.GetModelType(), f.Metadata.ID)
}

// GetModelType returns the Model's type
func (f *FederatedRegistry) GetModelType() types.ModelType {
	return types.FederatedRegistryModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (f *FederatedRegistry) ResolveMetadata(key string) (string, error) {
	return f.Metadata.resolveFieldValue(key)
}

// Validate validates the FederatedRegistry resource
func (f *FederatedRegistry) Validate() error {
	return nil
}
