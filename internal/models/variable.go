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
	Value         *string
	Category      VariableCategory
	NamespacePath string
	Key           string
	Metadata      ResourceMetadata
	Hcl           bool
}
