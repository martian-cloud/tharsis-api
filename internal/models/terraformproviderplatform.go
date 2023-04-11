package models

// TerraformProviderPlatform represents a supported platform for a terraform provider version
type TerraformProviderPlatform struct {
	ProviderVersionID string
	OperatingSystem   string
	Architecture      string
	SHASum            string
	Filename          string
	CreatedBy         string
	Metadata          ResourceMetadata
	BinaryUploaded    bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderPlatform) ResolveMetadata(key string) (string, error) {
	return t.Metadata.resolveFieldValue(key)
}
