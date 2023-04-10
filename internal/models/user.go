package models

// User represents a human user account
type User struct {
	Username       string
	Email          string
	SCIMExternalID string
	Metadata       ResourceMetadata
	Admin          bool
	Active         bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (u *User) ResolveMetadata(key string) (string, error) {
	return u.Metadata.resolveFieldValue(key)
}
