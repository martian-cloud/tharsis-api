package models

// SCIMToken represents a SCIM token.
type SCIMToken struct {
	Nonce     string
	CreatedBy string
	Metadata  ResourceMetadata
}
