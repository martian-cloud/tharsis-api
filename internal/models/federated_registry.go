package models

// FederatedRegistry represents a client-side federated registry.
type FederatedRegistry struct {
	Metadata ResourceMetadata
	Hostname string
	GroupID  string
	Audience string
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (f *FederatedRegistry) ResolveMetadata(key string) (string, error) {
	return f.Metadata.resolveFieldValue(key)
}
