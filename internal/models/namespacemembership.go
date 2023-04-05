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

// NamespaceMembership represents an association between a member and a namespace
type NamespaceMembership struct {
	UserID           *string
	ServiceAccountID *string
	TeamID           *string
	Namespace        MembershipNamespace
	RoleID           string
	Metadata         ResourceMetadata
}
