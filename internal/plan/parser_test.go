package plan

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestParse(t *testing.T) {
	type testCase struct {
		name               string
		expectErrorMessage string
		expectErrorCode    errors.CodeType
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
			expectErrorCode:    errors.EInvalid,
		},
		{
			name:               "nil plan",
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "the plan json format is not valid: plan is nil",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name:               "nil provider schemas",
			tfPlan:             &tfjson.Plan{FormatVersion: "1.2"},
			expectErrorMessage: "the provider schemas json is not valid: provider schema data is nil",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name:               "provider schemas with unsupported version",
			tfPlan:             &tfjson.Plan{FormatVersion: "1.2"},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "99.0"},
			expectErrorMessage: "the provider schemas json is not valid: unsupported provider schema format version: \"99.0.0\" does not satisfy \">= 0.1, < 2.0\"",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "resource change without change data",
			tfPlan: &tfjson.Plan{
				FormatVersion:   "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{Address: "test_resource.foo", Mode: "managed", Type: "test_resource", ProviderName: "test"}},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "plan contains a resource change without change data",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "null resource change",
			tfPlan: &tfjson.Plan{
				FormatVersion:   "1.2",
				ResourceChanges: []*tfjson.ResourceChange{nil},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "plan contains a resource change without change data",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "null resource drift",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceDrift: []*tfjson.ResourceChange{nil},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "plan contains a null resource drift entry",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "null output change",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				OutputChanges: map[string]*tfjson.Change{"test": nil},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "plan contains output test without change data",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "no-op resource change without a schema is not rendered",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{
					Address: "test_resource.foo", Mode: "managed", Type: "test_resource", ProviderName: "test",
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}},
				}},
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectDiff:        &Diff{Resources: []*ResourceDiff{}, Outputs: []*OutputDiff{}},
		},
		{
			name: "moved no-op resource change without a schema is rendered, so it needs one",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{
					Address: "test_resource.bar", PreviousAddress: "test_resource.foo", Mode: "managed", Type: "test_resource", ProviderName: "test",
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}},
				}},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "provider schema not found for provider test",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "imported no-op resource change without a schema is rendered, so it needs one",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{
					Address: "test_resource.foo", Mode: "managed", Type: "test_resource", ProviderName: "test",
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}, Importing: &tfjson.Importing{ID: "foo"}},
				}},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "provider schema not found for provider test",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "resource change for a provider missing from the schemas",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{
					Address: "test_resource.foo", Mode: "managed", Type: "test_resource", ProviderName: "test",
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}},
				}},
			},
			tfProviderSchemas:  &tfjson.ProviderSchemas{FormatVersion: "1.0"},
			expectErrorMessage: "provider schema not found for provider test",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "resource change for a resource type missing from the provider schema",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{
					Address: "test_resource.foo", Mode: "managed", Type: "test_resource", ProviderName: "test",
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}},
				}},
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{
				FormatVersion: "1.0",
				Schemas:       map[string]*tfjson.ProviderSchema{"test": {}},
			},
			expectErrorMessage: "schema not found for resource test_resource.foo",
			expectErrorCode:    errors.EInvalid,
		},
		{
			name: "resource change with an unrecognized mode",
			tfPlan: &tfjson.Plan{
				FormatVersion: "1.2",
				ResourceChanges: []*tfjson.ResourceChange{{
					Address: "test_resource.foo", Mode: "bogus", Type: "test_resource", ProviderName: "test",
					Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}},
				}},
			},
			tfProviderSchemas: &tfjson.ProviderSchemas{
				FormatVersion: "1.0",
				Schemas:       map[string]*tfjson.ProviderSchema{"test": {}},
			},
			expectErrorMessage: "found unrecognized resource mode: bogus",
			expectErrorCode:    errors.EInvalid,
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
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.expectDiff, actualDiff)
		})
	}
}
