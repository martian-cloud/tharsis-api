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
