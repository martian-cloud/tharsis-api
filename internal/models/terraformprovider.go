package models

import (
	"strings"
)

// TerraformProvider represents a terraform provider
type TerraformProvider struct {
	CreatedBy     string
	Name          string
	GroupID       string
	RootGroupID   string
	ResourcePath  string
	RepositoryURL string
	Metadata      ResourceMetadata
	Private       bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProvider) ResolveMetadata(key string) (string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			val = t.Name
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *TerraformProvider) Validate() error {
	// Verify name satisfies constraints
	return verifyValidName(t.Name)
}

// GetRegistryNamespace returns the provider registry namespace for the terraform provider
func (t *TerraformProvider) GetRegistryNamespace() string {
	return strings.Split(t.ResourcePath, "/")[0]
}

// GetGroupPath returns the group path
func (t *TerraformProvider) GetGroupPath() string {
	return t.ResourcePath[:strings.LastIndex(t.ResourcePath, "/")]
}
