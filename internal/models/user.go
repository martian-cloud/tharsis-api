package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*User)(nil)

// User represents a human user account
type User struct {
	Username       string
	Email          string
	SCIMExternalID string
	Metadata       ResourceMetadata
	Admin          bool
	Active         bool
}

// GetID returns the Metadata ID.
func (u *User) GetID() string {
	return u.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (u *User) GetGlobalID() string {
	return gid.ToGlobalID(u.GetModelType(), u.Metadata.ID)
}

// GetModelType returns the type of the model.
func (u *User) GetModelType() types.ModelType {
	return types.UserModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (u *User) ResolveMetadata(key string) (*string, error) {
	return u.Metadata.resolveFieldValue(key)
}

// Validate validates the user.
func (u *User) Validate() error {
	return nil
}
