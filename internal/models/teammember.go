package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TeamMember)(nil)

// TeamMember represents an association between a (human) user and a namespace
type TeamMember struct {
	UserID       string
	TeamID       string
	Metadata     ResourceMetadata
	IsMaintainer bool
}

// GetID returns the Metadata ID.
func (t *TeamMember) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TeamMember) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TeamMember) GetModelType() types.ModelType {
	return types.TeamMemberModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TeamMember) ResolveMetadata(key string) (*string, error) {
	return t.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (t *TeamMember) Validate() error {
	return nil
}
