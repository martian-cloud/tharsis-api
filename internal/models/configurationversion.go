package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*ConfigurationVersion)(nil)

// ConfigurationStatus represents the various states for a ConfigurationVersion resource
type ConfigurationStatus string

// Configuration version status types
const (
	ConfigurationErrored  ConfigurationStatus = "errored"
	ConfigurationPending  ConfigurationStatus = "pending"
	ConfigurationUploaded ConfigurationStatus = "uploaded"
)

// ConfigurationVersion resource represents a terraform configuration that can be used by a single Run
type ConfigurationVersion struct {
	VCSEventID  *string
	Status      ConfigurationStatus
	WorkspaceID string
	CreatedBy   string
	Metadata    ResourceMetadata
	Speculative bool
}

// GetID returns the ID of the ConfigurationVersion resource
func (c *ConfigurationVersion) GetID() string {
	return c.Metadata.ID
}

// GetGlobalID returns the GID of the ConfigurationVersion resource
func (c *ConfigurationVersion) GetGlobalID() string {
	return gid.ToGlobalID(c.GetModelType(), c.Metadata.ID)
}

// GetModelType returns the Model's type
func (c *ConfigurationVersion) GetModelType() types.ModelType {
	return types.ConfigurationVersionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (c *ConfigurationVersion) ResolveMetadata(key string) (*string, error) {
	return c.Metadata.resolveFieldValue(key)
}

// Validate validates the ConfigurationVersion resource
func (c *ConfigurationVersion) Validate() error {
	return nil
}
