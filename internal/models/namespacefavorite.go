package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*NamespaceFavorite)(nil)

// NamespaceFavorite represents a user's favorite namespace (group or workspace)
type NamespaceFavorite struct {
	UserID      string
	GroupID     *string
	WorkspaceID *string
	Metadata    ResourceMetadata
}

// GetID returns the Metadata ID.
func (f *NamespaceFavorite) GetID() string {
	return f.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (f *NamespaceFavorite) GetGlobalID() string {
	return gid.ToGlobalID(f.GetModelType(), f.Metadata.ID)
}

// GetModelType returns the type of the model.
func (f *NamespaceFavorite) GetModelType() types.ModelType {
	return types.NamespaceFavoriteModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (f *NamespaceFavorite) ResolveMetadata(key string) (*string, error) {
	return f.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (f *NamespaceFavorite) Validate() error {
	if (f.GroupID == nil && f.WorkspaceID == nil) || (f.GroupID != nil && f.WorkspaceID != nil) {
		return errors.New("exactly one of group ID or workspace ID must be set", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}
