package models

import "strings"

// Workspace represents a terraform workspace
type Workspace struct {
	MaxJobDuration        *int32
	Name                  string
	FullPath              string
	GroupID               string
	Description           string
	CurrentJobID          string
	CurrentStateVersionID string
	CreatedBy             string
	TerraformVersion      string
	Metadata              ResourceMetadata
	DirtyState            bool
	Locked                bool
}

// Validate returns an error if the model is not valid
func (w *Workspace) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(w.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	return verifyValidDescription(w.Description)
}

// GetGroupPath returns the group path
func (w *Workspace) GetGroupPath() string {
	return w.FullPath[:strings.LastIndex(w.FullPath, "/")]
}
