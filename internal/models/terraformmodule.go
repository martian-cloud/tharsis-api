package models

import (
	"strings"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformModule)(nil)

// TerraformModule represents a terraform module
type TerraformModule struct {
	CreatedBy     string
	Name          string // the module name
	System        string // the name of the remote system the module is intended to target
	GroupID       string
	RootGroupID   string // the module namespace is the path of the root group
	RepositoryURL string
	Metadata      ResourceMetadata
	Private       bool
}

// GetID returns the Metadata ID.
func (t *TerraformModule) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformModule) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the model type
func (t *TerraformModule) GetModelType() types.ModelType {
	return types.TerraformModuleModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformModule) ResolveMetadata(key string) (*string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			return &t.Name, nil
		case "group_path":
			return ptr.String(t.GetGroupPath()), nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *TerraformModule) Validate() error {
	return verifyValidName(t.Name)
}

// GetResourcePath returns the resource path for the terraform module
func (t *TerraformModule) GetResourcePath() string {
	return strings.Split(t.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetRegistryNamespace returns the module registry namespace for the terraform module
func (t *TerraformModule) GetRegistryNamespace() string {
	return strings.Split(t.GetResourcePath(), "/")[0]
}

// GetGroupPath returns the group path
func (t *TerraformModule) GetGroupPath() string {
	pathParts := strings.Split(t.GetResourcePath(), "/")
	return strings.Join(pathParts[:len(pathParts)-2], "/")
}
