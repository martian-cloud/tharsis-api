package models

// VariableCategory specifies if the variable is a terraform
// or environment variable
type VariableCategory string

// Variable category Status Types
const (
	TerraformVariableCategory   VariableCategory = "terraform"
	EnvironmentVariableCategory VariableCategory = "environment"
)

// Variable resource
type Variable struct {
	Value           *string
	SecretData      []byte
	Category        VariableCategory
	NamespacePath   string
	Key             string
	Metadata        ResourceMetadata
	Hcl             bool
	Sensitive       bool
	LatestVersionID string
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (v *Variable) ResolveMetadata(key string) (string, error) {
	val, err := v.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "namespace_path":
			val = v.NamespacePath
		case "key":
			val = v.Key
		default:
			return "", err
		}
	}

	return val, nil
}

// VariableVersion resource
type VariableVersion struct {
	VariableID string
	Value      *string
	Key        string
	Metadata   ResourceMetadata
	Hcl        bool
	// SecretData is only used for sensitive variables and it stores data
	// returned by the configured secret manager plugin
	SecretData []byte
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (v *VariableVersion) ResolveMetadata(key string) (string, error) {
	return v.Metadata.resolveFieldValue(key)
}
