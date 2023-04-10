package models

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

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (c *ConfigurationVersion) ResolveMetadata(key string) (string, error) {
	return c.Metadata.resolveFieldValue(key)
}
