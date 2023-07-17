package models

// TerraformProviderPlatformMirror represents the platforms a
// Terraform provider version mirror supports.
type TerraformProviderPlatformMirror struct {
	OS              string
	Architecture    string
	VersionMirrorID string
	Metadata        ResourceMetadata
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderPlatformMirror) ResolveMetadata(key string) (string, error) {
	return t.Metadata.resolveFieldValue(key)
}
