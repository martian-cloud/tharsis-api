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

// Validate returns an error if the model is not valid
func (t *TerraformModuleAttestation) Validate() error {
	// Verify description satisfies constraints
	return verifyValidDescription(t.Description)
}
