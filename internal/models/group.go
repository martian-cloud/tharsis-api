package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
)

var _ Model = (*Group)(nil)

// Group resource
type Group struct {
	Name                 string
	Description          string
	ParentID             string
	FullPath             string
	CreatedBy            string
	Metadata             ResourceMetadata
	RunnerTags           []string
	EnableDriftDetection *bool
	EnableProviderMirror *bool
}

// GetID returns the Metadata ID.
func (g *Group) GetID() string {
	return g.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (g *Group) GetGlobalID() string {
	return gid.ToGlobalID(g.GetModelType(), g.Metadata.ID)
}

// GetModelType returns the Model's type
func (g *Group) GetModelType() types.ModelType {
	return types.GroupModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (g *Group) ResolveMetadata(key string) (*string, error) {
	val, err := g.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "full_path":
			return &g.FullPath, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (g *Group) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(g.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	if err := verifyValidDescription(g.Description); err != nil {
		return err
	}

	// Check for duplicate tags, too-long tags, and too many tags.
	return verifyValidRunnerTags(g.RunnerTags)
}

// GetRootGroupPath returns the root path for the group
func (g *Group) GetRootGroupPath() string {
	if g.ParentID == "" {
		return g.FullPath
	}
	return strings.Split(g.FullPath, "/")[0]
}

// GetPath returns the full path for this group
func (g *Group) GetPath() string {
	return g.FullPath
}

// GetParentID returns the parent ID for this group if it has a parent
func (g *Group) GetParentID() string {
	return g.ParentID
}

// GetRunnerTags returns the runner tags for this group
func (g *Group) GetRunnerTags() []string {
	return g.RunnerTags
}

// DriftDetectionEnabled returns the drift detection enabled setting
func (g *Group) DriftDetectionEnabled() *bool {
	return g.EnableDriftDetection
}

// ProviderMirrorEnabled returns the provider mirror enabled setting
func (g *Group) ProviderMirrorEnabled() *bool {
	return g.EnableProviderMirror
}

// GetParentPath returns the path for the group's immediate parent.
func (g *Group) GetParentPath() string {
	if g.ParentID == "" {
		return ""
	}
	return GetGroupParentPath(g.FullPath)
}

// ExpandPath returns the expanded path list for the group. The expanded path
// list includes the full path for the group in addition to all parent paths
func (g *Group) ExpandPath() []string {
	return utils.ExpandPath(g.FullPath)
}

// GetDepth returns the depth of the tree from root to this group.  A root group is counted as 1.
func (g *Group) GetDepth() int {
	return 1 + strings.Count(g.FullPath, "/")
}

// IsDescendantOfGroup returns true if the group is a descendant of the specified (other/ancestor group) path.
func (g *Group) IsDescendantOfGroup(otherGroupPath string) bool {
	return utils.IsDescendantOfPath(g.FullPath, otherGroupPath)
}

// GetGroupParentPath returns the path for a group's parent based only on the current path.
func GetGroupParentPath(currentPath string) string {
	pathParts := strings.Split(currentPath, "/")
	return strings.Join(pathParts[:len(pathParts)-1], "/")
}
