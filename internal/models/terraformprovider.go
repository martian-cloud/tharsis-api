package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformProvider)(nil)

// TerraformProvider represents a terraform provider
type TerraformProvider struct {
	CreatedBy     string
	Name          string
	GroupID       string
	RootGroupID   string
	RepositoryURL string
	Metadata      ResourceMetadata
	Private       bool
}

// GetID returns the Metadata ID.
func (t *TerraformProvider) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformProvider) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformProvider) GetModelType() types.ModelType {
	return types.TerraformProviderModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProvider) ResolveMetadata(key string) (*string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			return &t.Name, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *TerraformProvider) Validate() error {
	// Verify name satisfies constraints
	return verifyValidName(t.Name)
}

// GetResourcePath returns the resource path for the terraform provider
func (t *TerraformProvider) GetResourcePath() string {
	return strings.Split(t.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetRegistryNamespace returns the provider registry namespace for the terraform provider
func (t *TerraformProvider) GetRegistryNamespace() string {
	return strings.Split(t.GetResourcePath(), "/")[0]
}

// GetGroupPath returns the group path
func (t *TerraformProvider) GetGroupPath() string {
	resourcePath := t.GetResourcePath()
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}
