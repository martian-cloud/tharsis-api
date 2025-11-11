package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// maxLabelsPerWorkspace is the maximum number of labels per workspace
	maxLabelsPerWorkspace = 10
	// maxLabelValueLength is the maximum length for a label value
	maxLabelValueLength = 255
)

var _ Model = (*Workspace)(nil)

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
	Labels                map[string]string
}

// GetID returns the Metadata ID.
func (w *Workspace) GetID() string {
	return w.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (w *Workspace) GetGlobalID() string {
	return gid.ToGlobalID(w.GetModelType(), w.Metadata.ID)
}

// GetModelType returns the model type
func (w *Workspace) GetModelType() types.ModelType {
	return types.WorkspaceModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (w *Workspace) ResolveMetadata(key string) (*string, error) {
	val, err := w.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "full_path":
			return &w.FullPath, nil
		default:
			return nil, err
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
	if err := verifyValidRunnerTags(w.RunnerTags); err != nil {
		return err
	}

	// Validate labels
	return validateLabels(w.Labels)
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
	return utils.IsDescendantOfPath(w.FullPath, groupPath)
}

// HasLabels returns true if the workspace has any labels
func (w *Workspace) HasLabels() bool {
	return len(w.Labels) > 0
}

// validateLabelKey validates a label key according to format and constraint rules
func validateLabelKey(key string) error {
	if key == "" {
		return errors.New("Label key cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	// Use the same validation as workspace names for consistency
	if !nameRegex.MatchString(key) {
		return errors.New("Invalid label key, key can only include lowercase letters and numbers with - and _ supported "+
			"in non leading or trailing positions. Max length is 64 characters.", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// validateLabelValue validates a label value according to format and constraint rules
func validateLabelValue(value string) error {
	if value == "" {
		return errors.New("Label value cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	if len(value) > maxLabelValueLength {
		return errors.New("Label value exceeds maximum length of %d characters", maxLabelValueLength, errors.WithErrorCode(errors.EInvalid))
	}

	// Label values must contain only alphanumeric characters, hyphens, underscores, and spaces
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == ' ') {
			return errors.New("Label value contains invalid characters. Only alphanumeric characters, hyphens, underscores, and spaces are allowed", errors.WithErrorCode(errors.EInvalid))
		}
	}

	return nil
}

// validateLabels validates all labels in a map according to format and constraint rules
func validateLabels(labels map[string]string) error {
	if labels == nil {
		return nil
	}

	if len(labels) > maxLabelsPerWorkspace {
		return errors.New("Maximum number of labels (%d) exceeded", maxLabelsPerWorkspace, errors.WithErrorCode(errors.EInvalid))
	}

	for key, value := range labels {
		if err := validateLabelKey(key); err != nil {
			return err
		}
		if err := validateLabelValue(value); err != nil {
			return err
		}
	}

	return nil
}
