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

// GetRegistryNamespace returns the provider registry namespace for the terraform provider
func (t *TerraformProvider) GetRegistryNamespace() string {
	return strings.Split(t.ResourcePath, "/")[0]
}

// GetGroupPath returns the group path
func (t *TerraformProvider) GetGroupPath() string {
	return t.ResourcePath[:strings.LastIndex(t.ResourcePath, "/")]
}
