package models

// TeamMember represents an association between a (human) user and a namespace
type TeamMember struct {
	UserID       string
	TeamID       string
	Metadata     ResourceMetadata
	IsMaintainer bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TeamMember) ResolveMetadata(key string) (string, error) {
	return t.Metadata.resolveFieldValue(key)
}
