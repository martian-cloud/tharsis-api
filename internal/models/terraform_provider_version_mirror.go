package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformProviderVersionMirror)(nil)

// TerraformProviderVersionMirror represents a version of a Terraform provider
// that's mirrored using the Provider Network Mirror Protocol.
type TerraformProviderVersionMirror struct {
	CreatedBy         string
	Type              string
	SemanticVersion   string
	RegistryNamespace string
	RegistryHostname  string
	Digests           map[string][]byte
	GroupID           string
	Metadata          ResourceMetadata
}

// GetID returns the Metadata ID.
func (t *TerraformProviderVersionMirror) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformProviderVersionMirror) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformProviderVersionMirror) GetModelType() types.ModelType {
	return types.TerraformProviderVersionMirrorModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderVersionMirror) ResolveMetadata(key string) (*string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "semantic_version":
			return &t.SemanticVersion, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate performs validation on the model.
func (t *TerraformProviderVersionMirror) Validate() error {
	return nil
}
