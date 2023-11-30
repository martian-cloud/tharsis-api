package models

// MaintenanceMode represents the maintenance mode (aka read-only mode) of the system.
type MaintenanceMode struct {
	CreatedBy string
	Message   string
	Metadata  ResourceMetadata
}
