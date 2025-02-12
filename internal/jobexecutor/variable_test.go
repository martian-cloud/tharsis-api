package jobexecutor

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/stretchr/testify/assert"
)

func TestIsHCLVariable(t *testing.T) {
	type testCase struct {
		name      string
		rawValue  *string
		varType   string // Type in the variable definition
		expectHCL bool
	}

	testCases := []testCase{
		// No type is defined
		// String type
		{
			name:     "no type, string with no quotes or spaces should be treated as string",
			rawValue: ptr.String("test"),
		},
		{
			name:     "no type, string with spaces and no quotes should be treated as string",
			rawValue: ptr.String("test value"),
		},
		{
			name:     "no type, a quoted string should be treated as string",
			rawValue: ptr.String(`"test value"`),
		},
		{
			name:     "no type, a quoted number should be treated as string",
			rawValue: ptr.String(`"123"`),
		},
		{
			name:     "no type, a quoted bool should be treated as string",
			rawValue: ptr.String(`"true"`),
		},
		{
			name:     "no type, a quoted map should be treated as a string",
			rawValue: ptr.String(`"{a = 1, b = 2}"`),
		},
		{
			name:     "no type, a quoted list should be treated as a string",
			rawValue: ptr.String(`"[1, 2]"`),
		},
		{
			name:     "no type, a quoted null should be treated as a string",
			rawValue: ptr.String(`"null"`),
		},
		// Complex types
		{
			name:      "no type, a bool should be treated as a string type",
			rawValue:  ptr.String("true"),
			expectHCL: false,
		},
		{
			name:      "no type, a boolean expression should be treated as a string type",
			rawValue:  ptr.String(`1 == 1`),
			expectHCL: false,
		},
		{
			name:      "no type, a number should be treated as a string type",
			rawValue:  ptr.String("123"),
			expectHCL: false,
		},
		{
			name:      "no type, a number expression should be treated as a string type",
			rawValue:  ptr.String(`1 + 1`),
			expectHCL: false,
		},
		{
			name:      "no type, a list should be treated as a complex type",
			rawValue:  ptr.String("[1, 2]"),
			expectHCL: true,
		},
		{
			name:      "no type, a map should be treated as a complex type",
			rawValue:  ptr.String(`{a = 1, b = 2}`),
			expectHCL: true,
		},
		{
			name:      "no type, a quoted list should be treated as a complex type",
			rawValue:  ptr.String(`["a", "b"]`),
			expectHCL: true,
		},
		{
			name:      "no type, a null should be treated as a complex type",
			rawValue:  ptr.String(`null`),
			expectHCL: true,
		},
		// Type is defined
		// This should use the type definition regardless of the raw value.
		{
			name:    "string type should be treated as string",
			varType: "string",
		},
		{
			name:      "bool type should be treated as complex type",
			varType:   "bool",
			expectHCL: true,
		},
		{
			name:      "number type should be treated as complex type",
			varType:   "number",
			expectHCL: true,
		},
		{
			name:      "list type should be treated as complex type",
			varType:   "list",
			expectHCL: true,
		},
		{
			name:      "map type should be treated as complex type",
			varType:   "map",
			expectHCL: true,
		},
		{
			name:      "object type should be treated as complex type",
			varType:   "object",
			expectHCL: true,
		},
		{
			name:      "set type should be treated as complex type",
			varType:   "set",
			expectHCL: true,
		},
		{
			name:      "tuple type should be treated as complex type",
			varType:   "tuple",
			expectHCL: true,
		},
		{
			name:      "any type should be treated as complex type",
			varType:   "any",
			expectHCL: true,
		},
		{
			name:      "unknown type should be treated as complex type",
			varType:   "unknown",
			expectHCL: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hcl := isHCLVariable(tc.rawValue, &tfconfig.Variable{
				Name: "my_var",
				Type: tc.varType,
			})

			assert.Equal(t, tc.expectHCL, hcl)
		})
	}
}
