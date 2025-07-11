package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*MaintenanceMode)(nil)

// MaintenanceMode represents the maintenance mode (aka read-only mode) of the system.
type MaintenanceMode struct {
	CreatedBy string
	Metadata  ResourceMetadata
}

// GetID returns the Metadata ID.
func (m *MaintenanceMode) GetID() string {
	return m.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (m *MaintenanceMode) GetGlobalID() string {
	return gid.ToGlobalID(m.GetModelType(), m.Metadata.ID)
}

// GetModelType returns the type of the model.
func (m *MaintenanceMode) GetModelType() types.ModelType {
	return types.MaintenanceModeModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (m *MaintenanceMode) ResolveMetadata(key string) (*string, error) {
	return m.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (m *MaintenanceMode) Validate() error {
	return nil
}
