package models

// TeamMember represents an association between a (human) user and a namespace
type TeamMember struct {
	UserID       string
	TeamID       string
	Metadata     ResourceMetadata
	IsMaintainer bool
}

// The End.
