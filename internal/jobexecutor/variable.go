package jobexecutor

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/zclconf/go-cty/cty"
)

// isHCLVariable determines if a variable is a complex type or a simple string.
func isHCLVariable(rawValue *string, v *tfconfig.Variable) bool {
	// Firstly refer to the variable type if it is defined.
	if v.Type != "" {
		switch v.Type {
		case "string":
			// Type is a simple string.
			return false
		default:
			// Type is a complex or unknown type.
			// Terraform will automatically handle unknown types.
			return true
		}
	}

	// Secondly refer to the raw value if it is defined.
	if rawValue != nil {
		fakeFileName := fmt.Sprintf("<value for var.%s>", v.Name)
		expression, diags := hclsyntax.ParseExpression([]byte(*rawValue), fakeFileName, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			// Since we allow strings to be passed as raw values, we can safely ignore this error.
			// This is likely a string with spaces and no quotes, which is treated as an expression
			// but in our case it is a valid string raw value.
			return false
		}

		ctyVal, diags := expression.Value(nil)
		if diags.HasErrors() {
			// Since we allow strings to be passed as raw values, we can safely ignore this error.
			// This is likely a string without spaces or quotes, which is treated as a variable reference
			// but in our case it is a valid string raw value.
			return false
		}

		if ctyVal.Type().Equals(cty.String) {
			// The raw value is quoted string.
			return false
		}

		// The raw value is a complex type (e.g. bool, number, list, map, etc).
		return true
	}

	// We cannot determine the type of the variable. Assume it is a complex type.
	return true
}
