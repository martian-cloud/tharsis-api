package models

import "strings"

// Group resource
type Group struct {
	Name        string
	Description string
	ParentID    string
	FullPath    string
	CreatedBy   string
	Metadata    ResourceMetadata
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
	return nil
}

// GetRootGroupPath returns the root path for the group
func (g *Group) GetRootGroupPath() string {
	if g.ParentID == "" {
		return g.FullPath
	}
	return strings.Split(g.FullPath, "/")[0]
}

// GetParentPath returns the path for the group's immediate parent.
func (g *Group) GetParentPath() string {
	if g.ParentID == "" {
		return ""
	}
	pathParts := strings.Split(g.FullPath, "/")
	return strings.Join(pathParts[:len(pathParts)-1], "/")
}

// ExpandPath returns the expanded path list for the group. The expanded path
// list includes the full path for the group in addition to all parent paths
func (g *Group) ExpandPath() []string {
	pathParts := strings.Split(g.FullPath, "/")

	paths := []string{}
	for len(pathParts) > 0 {
		paths = append(paths, strings.Join(pathParts, "/"))
		// Remove last element
		pathParts = pathParts[:len(pathParts)-1]
	}

	return paths
}
