package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*Team)(nil)

// Team represents a team of (human) users
type Team struct {
	Name           string
	Description    string
	SCIMExternalID string
	Metadata       ResourceMetadata
}

// GetID returns the Metadata ID.
func (t *Team) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *Team) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the model type.
func (t *Team) GetModelType() types.ModelType {
	return types.TeamModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *Team) ResolveMetadata(key string) (string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			val = t.Name
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *Team) Validate() error {
	// Verify description satisfies constraints
	return verifyValidDescription(t.Description)
}
