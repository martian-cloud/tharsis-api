package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformProviderPlatformMirror)(nil)

// TerraformProviderPlatformMirror represents the platforms a
// Terraform provider version mirror supports.
type TerraformProviderPlatformMirror struct {
	OS              string
	Architecture    string
	VersionMirrorID string
	Metadata        ResourceMetadata
}

// GetID returns the Metadata ID.
func (t *TerraformProviderPlatformMirror) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformProviderPlatformMirror) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformProviderPlatformMirror) GetModelType() types.ModelType {
	return types.TerraformProviderPlatformMirrorModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderPlatformMirror) ResolveMetadata(key string) (*string, error) {
	return t.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (t *TerraformProviderPlatformMirror) Validate() error {
	return nil
}
