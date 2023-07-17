package models

// TerraformProviderVersionMirror represents a version of a Terraform provider
// that's mirrored using the Provider Network Mirror Protocol.
type TerraformProviderVersionMirror struct {
	CreatedBy         string
	Type              string
	SemanticVersion   string
	RegistryNamespace string
	RegistryHostname  string
	Digests           map[string][]byte
	GroupID           string
	Metadata          ResourceMetadata
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderVersionMirror) ResolveMetadata(key string) (string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "semantic_version":
			val = t.SemanticVersion
		default:
			return "", err
		}
	}

	return val, nil
}
