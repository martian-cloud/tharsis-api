package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
)

var _ Model = (*NamespaceMembership)(nil)

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
	return utils.IsDescendantOfPath(m.Path, groupPath)
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

// GetID returns the Metadata ID.
func (nm *NamespaceMembership) GetID() string {
	return nm.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (nm *NamespaceMembership) GetGlobalID() string {
	return gid.ToGlobalID(nm.GetModelType(), nm.Metadata.ID)
}

// GetModelType returns the type of the model.
func (nm *NamespaceMembership) GetModelType() types.ModelType {
	return types.NamespaceMembershipModelType
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

// Validate performs validation on the NamespaceMembership
func (nm *NamespaceMembership) Validate() error {
	return nil
}
