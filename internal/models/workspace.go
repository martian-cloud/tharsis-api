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
	PreventDestroyPlan    bool
	RunnerTags            []string
	EnableDriftDetection  *bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (w *Workspace) ResolveMetadata(key string) (string, error) {
	val, err := w.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "full_path":
			val = w.FullPath
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (w *Workspace) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(w.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	if err := verifyValidDescription(w.Description); err != nil {
		return err
	}

	// Check for duplicate tags, too-long tags, and too many tags.
	return verifyValidRunnerTags(w.RunnerTags)
}

// GetPath returns the full path for this workspace
func (w *Workspace) GetPath() string {
	return w.FullPath
}

// GetParentID returns the parent group ID
func (w *Workspace) GetParentID() string {
	return w.GroupID
}

// GetRunnerTags returns the runner tags for this workspace
func (w *Workspace) GetRunnerTags() []string {
	return w.RunnerTags
}

// DriftDetectionEnabled returns the drift detection enabled setting
func (w *Workspace) DriftDetectionEnabled() *bool {
	return w.EnableDriftDetection
}

// GetGroupPath returns the group path
func (w *Workspace) GetGroupPath() string {
	return w.FullPath[:strings.LastIndex(w.FullPath, "/")]
}

// ExpandPath returns the expanded path list for the workspace. The expanded path
// list includes the full path for the workspace in addition to all parent paths
func (w *Workspace) ExpandPath() []string {
	pathParts := strings.Split(w.FullPath, "/")

	paths := []string{}
	for len(pathParts) > 0 {
		paths = append(paths, strings.Join(pathParts, "/"))
		// Remove last element
		pathParts = pathParts[:len(pathParts)-1]
	}

	return paths
}

// IsDescendantOfGroup returns true if the workspace is a descendant of the specified ancestor group path.
func (w *Workspace) IsDescendantOfGroup(groupPath string) bool {
	return IsDescendantOfPath(w.FullPath, groupPath)
}
