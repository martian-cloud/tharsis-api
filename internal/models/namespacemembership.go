package models

import "strings"

// MembershipNamespace represents a namespace which can be a group or workspace
type MembershipNamespace struct {
	GroupID     *string
	WorkspaceID *string
	ID          string
	Path        string
}

// IsTopLevel returns true if this is a top-level namespace
func (m MembershipNamespace) IsTopLevel() bool {
	return !strings.Contains(m.Path, "/")
}

// IsDescendantOfGroup returns true if the namespace is a descendant of the specified ancestor group path.
func (m *MembershipNamespace) IsDescendantOfGroup(groupPath string) bool {
	return IsDescendantOfPath(m.Path, groupPath)
}

// NamespaceMembership represents an association between a member and a namespace
type NamespaceMembership struct {
	UserID           *string
	ServiceAccountID *string
	TeamID           *string
	Namespace        MembershipNamespace
	RoleID           string
	Metadata         ResourceMetadata
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (nm *NamespaceMembership) ResolveMetadata(key string) (string, error) {
	val, err := nm.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "namespace_path":
			val = nm.Namespace.Path
		default:
			return "", err
		}
	}

	return val, nil
}

// IsDescendantOfPath returns true if the namespace is a descendant of the specified (ancestor group) path.
func IsDescendantOfPath(descendantPath, ancestorPath string) bool {
	return strings.HasPrefix(descendantPath, ancestorPath+"/")
}
