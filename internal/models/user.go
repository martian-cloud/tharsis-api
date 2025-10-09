package models

import (
	"net/mail"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

var _ Model = (*User)(nil)

// User represents a human user account
type User struct {
	Username       string
	Email          string
	PasswordHash   []byte
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
	if u.Username == "" {
		return errors.New("username is required", errors.WithErrorCode(errors.EInvalid))
	}
	if u.Email == "" {
		return errors.New("email is required", errors.WithErrorCode(errors.EInvalid))
	}

	if err := verifyValidName(u.Username); err != nil {
		return errors.Wrap(err, "username is invalid")
	}

	if _, err := mail.ParseAddress(u.Email); err != nil {
		return errors.New("email is invalid: %v", err, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// VerifyPassword verifies the given password against the stored password hash.
func (u *User) VerifyPassword(password string) bool {
	if u.PasswordHash == nil || password == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password))
	return err == nil
}

// SetPassword hashes the given password and sets it as the user's password hash.
func (u *User) SetPassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}
