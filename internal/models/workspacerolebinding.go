package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*WorkspaceRoleBinding)(nil)

// WorkspaceRoleBinding confers a Role on a workspace. The role's permissions become available to the
// workspace's job caller at the workspace's DIRECT PARENT namespace, and — because permissions are
// inherited downward — at every namespace beneath it. This is what lets a workspace manage Tharsis
// resources with the Tharsis provider instead of requiring a service account and managed identity.
//
// Creating a binding is gated on the caller holding both the WorkspaceRoleBinding permission and every
// permission in the bound role, at the parent namespace. See the workspace service for that check.
type WorkspaceRoleBinding struct {
	Metadata    ResourceMetadata
	WorkspaceID string
	RoleID      string
	CreatedBy   string
}

// GetID returns the Metadata ID.
func (w *WorkspaceRoleBinding) GetID() string {
	return w.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (w *WorkspaceRoleBinding) GetGlobalID() string {
	return gid.ToGlobalID(w.GetModelType(), w.Metadata.ID)
}

// GetModelType returns the Model's type.
func (w *WorkspaceRoleBinding) GetModelType() types.ModelType {
	return types.WorkspaceRoleBindingModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (w *WorkspaceRoleBinding) ResolveMetadata(key string) (*string, error) {
	return w.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid.
func (w *WorkspaceRoleBinding) Validate() error {
	if w.WorkspaceID == "" {
		return errors.New("workspace ID is required", errors.WithErrorCode(errors.EInvalid))
	}

	if w.RoleID == "" {
		return errors.New("role ID is required", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
