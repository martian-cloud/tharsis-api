package models

import "strings"

// Role represents the access level a user has within a namespace
type Role string

// Default Roles
const (
	ViewerRole   Role = "viewer"
	DeployerRole Role = "deployer"
	OwnerRole    Role = "owner"
)

// IsValid returns true if this is a valid role
func (r Role) IsValid() bool {
	switch r {
	case ViewerRole, DeployerRole, OwnerRole:
		return true
	}
	return false
}

// GTE checks if this access level is greater than or equal to 'other' (i.e. al >= other)
func (r Role) GTE(other Role) bool {
	if r == other {
		return true
	}
	if r == OwnerRole {
		return true
	}

	if r == ViewerRole {
		return false
	}

	if r == DeployerRole {
		return other != OwnerRole
	}

	return false
}

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
	Role             Role
	Metadata         ResourceMetadata
}
