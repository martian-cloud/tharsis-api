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
