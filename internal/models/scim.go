package models

import (
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// minSCIMTokenExpiration is the minimum time a SCIM token with an expiration must remain valid.
const minSCIMTokenExpiration = 24 * time.Hour

var _ Model = (*SCIMToken)(nil)

// SCIMToken represents a SCIM token.
type SCIMToken struct {
	ExpiresAt    *time.Time // Nil means the token never expires.
	TokenHash    string
	IDPIssuerURL string
	CreatedBy    Actor
	Metadata     ResourceMetadata
}

// GetID returns the Metadata ID.
func (s *SCIMToken) GetID() string {
	return s.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (s *SCIMToken) GetGlobalID() string {
	return gid.ToGlobalID(s.GetModelType(), s.Metadata.ID)
}

// GetModelType returns the type of the model.
func (s *SCIMToken) GetModelType() types.ModelType {
	return types.SCIMTokenModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *SCIMToken) ResolveMetadata(key string) (*string, error) {
	return s.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid
func (s *SCIMToken) Validate() error {
	if s.TokenHash == "" {
		return errors.New("scim token hash is required", errors.WithErrorCode(errors.EInvalid))
	}

	if s.IDPIssuerURL == "" {
		return errors.New("scim token idp issuer url is required", errors.WithErrorCode(errors.EInvalid))
	}

	if s.ExpiresAt != nil && s.ExpiresAt.Before(time.Now().UTC().Add(minSCIMTokenExpiration)) {
		return errors.New("scim token expiration must be at least 1 day from now", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
