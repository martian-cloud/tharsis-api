package models

import (
	"strings"
)

// TerraformModule represents a terraform module
type TerraformModule struct {
	CreatedBy     string
	Name          string // the module name
	System        string // the name of the remote system the module is intended to target
	GroupID       string
	RootGroupID   string // the module namespace is the path of the root group
	ResourcePath  string // resource path is <group-path>/<module-name>/<system>
	RepositoryURL string
	Metadata      ResourceMetadata
	Private       bool
}

// Validate returns an error if the model is not valid
func (t *TerraformModule) Validate() error {
	return verifyValidName(t.Name)
}

// GetRegistryNamespace returns the module registry namespace for the terraform module
func (t *TerraformModule) GetRegistryNamespace() string {
	return strings.Split(t.ResourcePath, "/")[0]
}

// GetGroupPath returns the group path
func (t *TerraformModule) GetGroupPath() string {
	pathParts := strings.Split(t.ResourcePath, "/")
	return strings.Join(pathParts[:len(pathParts)-2], "/")
}
