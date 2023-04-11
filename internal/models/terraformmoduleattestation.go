package models

// TerraformModuleAttestation represents a terraform module attestation
type TerraformModuleAttestation struct {
	CreatedBy     string
	Description   string
	ModuleID      string
	SchemaType    string
	PredicateType string
	Data          string
	DataSHASum    []byte
	Metadata      ResourceMetadata
	Digests       []string
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformModuleAttestation) ResolveMetadata(key string) (string, error) {
	val, err := t.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "predicate":
			val = t.PredicateType
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (t *TerraformModuleAttestation) Validate() error {
	// Verify description satisfies constraints
	return verifyValidDescription(t.Description)
}
