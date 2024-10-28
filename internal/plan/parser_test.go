package plan

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
)

func TestParse(t *testing.T) {
	type testCase struct {
		name               string
		expectErrorMessage string
		tfPlan             *tfjson.Plan
		tfProviderSchemas  *tfjson.ProviderSchemas
		expectDiff         *Diff
	}

	testCases := []testCase{
		{
			name: "parse plan with invalid version",
			tfPlan: &tfjson.Plan{
				FormatVersion: "2.0",
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{
				FormatVersion: "2.0",
			},
			expectErrorMessage: "the plan json format is not valid: unsupported plan format version: \"2.0.0\" does not satisfy \">= 0.1, < 2.0\"",
		},
		{
			name: "parse plan with valid version",
			tfPlan: &tfjson.Plan{
				FormatVersion: "0.1",
				ResourceChanges: []*tfjson.ResourceChange{
					{
						Address:      "test_resource.foo[0]",
						Mode:         "managed",
						Type:         "test_resource",
						Name:         "foo",
						ProviderName: "test",
						Change: &tfjson.Change{
							Actions: tfjson.Actions{tfjson.ActionDelete},
							Before: map[string]interface{}{
								"normal_attribute": "some value",
							},
						},
					},
				},
				OutputChanges: map[string]*tfjson.Change{
					"test": {
						After: "test parser output",
					},
				},
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{
				FormatVersion: "0.1",
				Schemas: map[string]*tfjson.ProviderSchema{
					"test": {
						ResourceSchemas: map[string]*tfjson.Schema{
							"test_resource": {
								Block: &tfjson.SchemaBlock{
									Attributes: map[string]*tfjson.SchemaAttribute{
										"normal_attribute": {
											AttributeType: cty.String,
										},
									},
								},
							},
						},
					},
				},
			},
			expectDiff: &Diff{
				Outputs: []*OutputDiff{
					{
						OutputName:  "test",
						Action:      action.Create,
						UnifiedDiff: "--- before\n+++ after\n@@ -1 +1,3 @@\n+output \"test\" {\n+   value = \"test parser output\"\n+}\n\\ No newline at end of file\n",
						Warnings:    []*ChangeWarning{},
					},
				},
				Resources: []*ResourceDiff{
					{
						Address:        "test_resource.foo[0]",
						Mode:           "managed",
						ResourceType:   "test_resource",
						ResourceName:   "foo",
						ProviderName:   "test",
						Action:         action.Delete,
						Warnings:       []*ChangeWarning{},
						OriginalSource: "resource \"test_resource\" \"foo\" {\n    normal_attribute = \"some value\"\n}",
						UnifiedDiff:    "--- before\n+++ after\n@@ -1,3 +1 @@\n-resource \"test_resource\" \"foo\" {\n-    normal_attribute = \"some value\"\n-}\n\\ No newline at end of file\n",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			parser := &parser{}
			actualDiff, err := parser.Parse(test.tfPlan, test.tfProviderSchemas)

			if test.expectErrorMessage != "" {
				assert.EqualError(t, err, test.expectErrorMessage)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectDiff, actualDiff)
		})
	}
}
