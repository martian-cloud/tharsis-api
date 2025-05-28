package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformProviderPlatform)(nil)

// TerraformProviderPlatform represents a supported platform for a terraform provider version
type TerraformProviderPlatform struct {
	ProviderVersionID string
	OperatingSystem   string
	Architecture      string
	SHASum            string
	Filename          string
	CreatedBy         string
	Metadata          ResourceMetadata
	BinaryUploaded    bool
}

// GetID returns the Metadata ID.
func (t *TerraformProviderPlatform) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformProviderPlatform) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformProviderPlatform) GetModelType() types.ModelType {
	return types.TerraformProviderPlatformModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderPlatform) ResolveMetadata(key string) (*string, error) {
	return t.Metadata.resolveFieldValue(key)
}

// Validate validates the TerraformProviderPlatform model.
func (t *TerraformProviderPlatform) Validate() error {
	return nil
}
