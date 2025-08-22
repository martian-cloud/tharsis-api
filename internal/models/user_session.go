package models

import (
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*UserSession)(nil)

// UserSession represents a session for a user.
type UserSession struct {
	UserID         string
	RefreshTokenID string
	UserAgent      string
	Expiration     time.Time
	Metadata       ResourceMetadata
}

// GetID returns the Metadata ID.
func (u *UserSession) GetID() string {
	return u.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (u *UserSession) GetGlobalID() string {
	return gid.ToGlobalID(u.GetModelType(), u.Metadata.ID)
}

// GetModelType returns the model type.
func (u *UserSession) GetModelType() types.ModelType {
	return types.UserSessionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (u *UserSession) ResolveMetadata(key string) (*string, error) {
	val, err := u.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "expiration":
			return ptr.String(u.Expiration.Format(time.RFC3339Nano)), nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate validates the model.
func (u *UserSession) Validate() error {
	return nil
}

// IsExpired returns true if the session has expired
func (u *UserSession) IsExpired() bool {
	return time.Now().After(u.Expiration)
}
