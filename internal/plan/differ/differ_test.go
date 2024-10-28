// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured/attributepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	tjson "github.com/hashicorp/terraform-json"
)

type SetDiff struct {
	Before SetDiffEntry
	After  SetDiffEntry
}

type SetDiffEntry struct {
	SingleDiff renderers.ValidateDiffFunction
	ObjectDiff map[string]renderers.ValidateDiffFunction

	Replace bool
	Action  action.Action
}

func (entry SetDiffEntry) Validate(obj func(attributes map[string]renderers.ValidateDiffFunction, action action.Action, replace bool) renderers.ValidateDiffFunction) renderers.ValidateDiffFunction {
	if entry.SingleDiff != nil {
		return entry.SingleDiff
	}

	return obj(entry.ObjectDiff, entry.Action, entry.Replace)
}

func TestValue_SimpleBlocks(t *testing.T) {
	// Most of the other test functions wrap the test cases in various
	// collections or blocks. This function just very simply lets you specify
	// individual test cases within blocks for some simple tests.

	tcs := map[string]struct {
		input    structured.Change
		block    *tjson.SchemaBlock
		validate renderers.ValidateDiffFunction
	}{
		"delete_with_null_sensitive_value": {
			input: structured.Change{
				Before: map[string]interface{}{
					"normal_attribute": "some value",
				},
				After: nil,
				BeforeSensitive: map[string]interface{}{
					"sensitive_attribute": true,
				},
				AfterSensitive: false,
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"normal_attribute": {
						AttributeType: cty.String,
					},
					"sensitive_attribute": {
						AttributeType: cty.String,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"normal_attribute": renderers.ValidatePrimitive("some value", nil, action.Delete, false),
			}, nil, nil, nil, nil, action.Delete, false),
		},
		"create_with_null_sensitive_value": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"normal_attribute": "some value",
				},
				BeforeSensitive: map[string]interface{}{
					"sensitive_attribute": true,
				},
				AfterSensitive: false,
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"normal_attribute": {
						AttributeType: cty.String,
					},
					"sensitive_attribute": {
						AttributeType: cty.String,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"normal_attribute": renderers.ValidatePrimitive(nil, "some value", action.Create, false),
			}, nil, nil, nil, nil, action.Create, false),
		},
	}
	for name, tc := range tcs {
		// Set some default values
		if tc.input.ReplacePaths == nil {
			tc.input.ReplacePaths = &attributepath.PathMatcher{}
		}

		if tc.input.RelevantAttributes == nil {
			tc.input.RelevantAttributes = attributepath.AlwaysMatcher()
		}

		t.Run(name, func(t *testing.T) {
			computedDiff, err := ComputeDiffForBlock(tc.input, tc.block)
			require.NoError(t, err)
			tc.validate(t, computedDiff)
		})
	}
}

func TestValue_ObjectAttributes(t *testing.T) {
	// This function holds a range of test cases creating, deleting and editing
	// objects. It is built in such a way that it can automatically test these
	// operations on objects both directly and nested, as well as within all
	// types of collections.

	tcs := map[string]struct {
		input                structured.Change
		attributes           map[string]cty.Type
		validateSingleDiff   renderers.ValidateDiffFunction
		validateObject       renderers.ValidateDiffFunction
		validateNestedObject renderers.ValidateDiffFunction
		validateDiffs        map[string]renderers.ValidateDiffFunction
		validateList         renderers.ValidateDiffFunction
		validateReplace      bool
		validateAction       action.Action
		// Sets break changes out differently to the other collections, so they
		// have their own entry.
		validateSetDiffs *SetDiff
	}{
		"create": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
			},
			validateAction:  action.Create,
			validateReplace: false,
		},
		"delete": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: nil,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			},
			validateAction:  action.Delete,
			validateReplace: false,
		},
		"create_sensitive": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleDiff: renderers.ValidateSensitive(renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
			}, action.Create, false),
				false,
				true,
				action.Create,
				false),
			validateNestedObject: renderers.ValidateSensitive(renderers.ValidateNestedObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
			}, action.Create, false),
				false,
				true,
				action.Create,
				false),
		},
		"delete_sensitive": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: true,
				After:           nil,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleDiff: renderers.ValidateSensitive(renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			}, action.Delete, false), true, false, action.Delete, false),
			validateNestedObject: renderers.ValidateSensitive(renderers.ValidateNestedObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			}, action.Delete, false), true, false, action.Delete, false),
		},
		"create_unknown": {
			input: structured.Change{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateSingleDiff: renderers.ValidateUnknown(nil, action.Create, false),
		},
		"update_unknown": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After:   nil,
				Unknown: true,
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateObject: renderers.ValidateUnknown(renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			}, action.Delete, false), action.Update, false),
			validateNestedObject: renderers.ValidateUnknown(renderers.ValidateNestedObject(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateUnknown(renderers.ValidatePrimitive("old", nil, action.Delete, false), action.Update, false),
			}, action.Update, false), action.Update, false),
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					SingleDiff: renderers.ValidateUnknown(nil, action.Create, false),
				},
			},
		},
		"create_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: nil,
					Action:     action.Delete,
					Replace:    false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"create_attribute_from_explicit_null": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": nil,
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: nil,
					Action:     action.Delete,
					Replace:    false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"delete_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: nil,
					Action:     action.Create,
					Replace:    false,
				},
			},
		},
		"delete_attribute_to_explicit_null": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": nil,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: nil,
					Action:     action.Create,
					Replace:    false,
				},
			},
		},
		"update_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", action.Update, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"create_sensitive_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", action.Create, false), false, true, action.Create, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: nil,
					Action:     action.Delete,
					Replace:    false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", action.Create, false), false, true, action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"delete_sensitive_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: map[string]interface{}{
					"attribute_one": true,
				},
				After: map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, action.Delete, false), true, false, action.Delete, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, action.Delete, false), true, false, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: nil,
					Action:     action.Create,
					Replace:    false,
				},
			},
		},
		"update_sensitive_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				BeforeSensitive: map[string]interface{}{
					"attribute_one": true,
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				AfterSensitive: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", "new", action.Update, false), true, true, action.Update, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, action.Delete, false), true, false, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", action.Create, false), false, true, action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"create_computed_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{},
				After:  map[string]interface{}{},
				Unknown: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateUnknown(nil, action.Create, false),
			},
			validateAction:  action.Update,
			validateReplace: false,
		},
		"update_computed_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{},
				Unknown: map[string]interface{}{
					"attribute_one": true,
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidateUnknown(
					renderers.ValidatePrimitive("old", nil, action.Delete, false),
					action.Update,
					false),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidateUnknown(nil, action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"ignores_unset_fields": {
			input: structured.Change{
				Before: map[string]interface{}{},
				After:  map[string]interface{}{},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs:   map[string]renderers.ValidateDiffFunction{},
			validateAction:  action.NoOp,
			validateReplace: false,
		},
		"update_replace_self": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				ReplacePaths: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{},
					},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", action.Update, false),
			},
			validateAction:  action.Update,
			validateReplace: true,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: true,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
					},
					Action:  action.Create,
					Replace: true,
				},
			},
		},
		"update_replace_attribute": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old",
				},
				After: map[string]interface{}{
					"attribute_one": "new",
				},
				ReplacePaths: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{"attribute_one"},
					},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", action.Update, true),
			},
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, true),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, true),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
		"update_includes_relevant_attributes": {
			input: structured.Change{
				Before: map[string]interface{}{
					"attribute_one": "old_one",
					"attribute_two": "old_two",
				},
				After: map[string]interface{}{
					"attribute_one": "new_one",
					"attribute_two": "new_two",
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{"attribute_one"},
					},
				},
			},
			attributes: map[string]cty.Type{
				"attribute_one": cty.String,
				"attribute_two": cty.String,
			},
			validateDiffs: map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old_one", "new_one", action.Update, false),
				"attribute_two": renderers.ValidatePrimitive("old_two", "old_two", action.NoOp, false),
			},
			validateList: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					// Lists are a bit special, and in this case is actually
					// going to ignore the relevant attributes. This is
					// deliberate. See the comments in list.go for an
					// explanation.
					"attribute_one": renderers.ValidatePrimitive("old_one", "new_one", action.Update, false),
					"attribute_two": renderers.ValidatePrimitive("old_two", "new_two", action.Update, false),
				}, action.Update, false),
			}, action.Update, false),
			validateAction:  action.Update,
			validateReplace: false,
			validateSetDiffs: &SetDiff{
				Before: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
						"attribute_two": renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
					},
					Action:  action.Delete,
					Replace: false,
				},
				After: SetDiffEntry{
					ObjectDiff: map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
						"attribute_two": renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
					},
					Action:  action.Create,
					Replace: false,
				},
			},
		},
	}

	for name, tmp := range tcs {
		tc := tmp

		// Let's set some default values on the input.
		if tc.input.RelevantAttributes == nil {
			tc.input.RelevantAttributes = attributepath.AlwaysMatcher()
		}
		if tc.input.ReplacePaths == nil {
			tc.input.ReplacePaths = &attributepath.PathMatcher{}
		}

		collectionDefaultAction := action.Update
		if name == "ignores_unset_fields" {
			// Special case for this test, as it is the only one that doesn't
			// have the collection types return an update.
			collectionDefaultAction = action.NoOp
		}

		t.Run(name, func(t *testing.T) {
			t.Run("object", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.Object(tc.attributes),
				}

				if tc.validateObject != nil {
					computedDiff, err := ComputeDiffForAttribute(tc.input, attribute)
					require.NoError(t, err)

					tc.validateObject(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					computedDiff, err := ComputeDiffForAttribute(tc.input, attribute)
					require.NoError(t, err)
					tc.validateSingleDiff(t, computedDiff)
					return
				}

				validate := renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace)
				computedDiff, err := ComputeDiffForAttribute(tc.input, attribute)
				require.NoError(t, err)

				validate(t, computedDiff)
			})

			t.Run("map", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.Map(cty.Object(tc.attributes)),
				}

				input := wrapChangeInMap(tc.input)

				if tc.validateObject != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateObject,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)

					validate(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateSingleDiff,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"element": renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("list", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.List(cty.Object(tc.attributes)),
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateList != nil {
					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)

					tc.validateList(t, computedDiff)
					return
				}

				if tc.validateObject != nil {
					validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
						tc.validateObject,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("set", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.Set(cty.Object(tc.attributes)),
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateSetDiffs != nil {
					validate := renderers.ValidateSet(func() []renderers.ValidateDiffFunction {
						var ret []renderers.ValidateDiffFunction
						ret = append(ret, tc.validateSetDiffs.Before.Validate(renderers.ValidateObject))
						ret = append(ret, tc.validateSetDiffs.After.Validate(renderers.ValidateObject))
						return ret
					}(), collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateObject != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateObject,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
					renderers.ValidateObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
		})

		t.Run(fmt.Sprintf("nested_%s", name), func(t *testing.T) {
			t.Run("object", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeNestedType: &tjson.SchemaNestedAttributeType{
						Attributes: func() map[string]*tjson.SchemaAttribute {
							attributes := make(map[string]*tjson.SchemaAttribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &tjson.SchemaAttribute{
									AttributeType: attribute,
								}
							}
							return attributes
						}(),
						NestingMode: "single",
					},
				}

				if tc.validateNestedObject != nil {
					computedDiff, err := ComputeDiffForAttribute(tc.input, attribute)
					require.NoError(t, err)

					tc.validateNestedObject(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					computedDiff, err := ComputeDiffForAttribute(tc.input, attribute)
					require.NoError(t, err)

					tc.validateSingleDiff(t, computedDiff)
					return
				}

				validate := renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace)

				computedDiff, err := ComputeDiffForAttribute(tc.input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("map", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeNestedType: &tjson.SchemaNestedAttributeType{
						Attributes: func() map[string]*tjson.SchemaAttribute {
							attributes := make(map[string]*tjson.SchemaAttribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &tjson.SchemaAttribute{
									AttributeType: attribute,
								}
							}
							return attributes
						}(),
						NestingMode: "map",
					},
				}

				input := wrapChangeInMap(tc.input)

				if tc.validateNestedObject != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateNestedObject,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
						"element": tc.validateSingleDiff,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"element": renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("list", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeNestedType: &tjson.SchemaNestedAttributeType{
						Attributes: func() map[string]*tjson.SchemaAttribute {
							attributes := make(map[string]*tjson.SchemaAttribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &tjson.SchemaAttribute{
									AttributeType: attribute,
								}
							}
							return attributes
						}(),
						NestingMode: "list",
					},
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateNestedObject != nil {
					validate := renderers.ValidateNestedList([]renderers.ValidateDiffFunction{
						tc.validateNestedObject,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateNestedList([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateNestedList([]renderers.ValidateDiffFunction{
					renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("set", func(t *testing.T) {
				attribute := &tjson.SchemaAttribute{
					AttributeNestedType: &tjson.SchemaNestedAttributeType{
						Attributes: func() map[string]*tjson.SchemaAttribute {
							attributes := make(map[string]*tjson.SchemaAttribute)
							for key, attribute := range tc.attributes {
								attributes[key] = &tjson.SchemaAttribute{
									AttributeType: attribute,
								}
							}
							return attributes
						}(),
						NestingMode: "set",
					},
				}

				input := wrapChangeInSlice(tc.input)

				if tc.validateSetDiffs != nil {
					validate := renderers.ValidateSet(func() []renderers.ValidateDiffFunction {
						var ret []renderers.ValidateDiffFunction
						ret = append(ret, tc.validateSetDiffs.Before.Validate(renderers.ValidateNestedObject))
						ret = append(ret, tc.validateSetDiffs.After.Validate(renderers.ValidateNestedObject))
						return ret
					}(), collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateNestedObject != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateNestedObject,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				if tc.validateSingleDiff != nil {
					validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
						tc.validateSingleDiff,
					}, collectionDefaultAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
					renderers.ValidateNestedObject(tc.validateDiffs, tc.validateAction, tc.validateReplace),
				}, collectionDefaultAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
		})
	}
}

func TestValue_BlockAttributesAndNestedBlocks(t *testing.T) {
	// This function tests manipulating simple attributes and blocks within
	// blocks. It automatically tests these operations within the contexts of
	// different block types.

	tcs := map[string]struct {
		before      interface{}
		after       interface{}
		block       *tjson.SchemaBlock
		validate    renderers.ValidateDiffFunction
		validateSet []renderers.ValidateDiffFunction
	}{
		"create_attribute": {
			before: map[string]interface{}{},
			after: map[string]interface{}{
				"attribute_one": "new",
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"attribute_one": {
						AttributeType: cty.String,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
			}, nil, nil, nil, nil, action.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, nil, nil, nil, nil, action.Delete, false),
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
				}, nil, nil, nil, nil, action.Create, false),
			},
		},
		"update_attribute": {
			before: map[string]interface{}{
				"attribute_one": "old",
			},
			after: map[string]interface{}{
				"attribute_one": "new",
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"attribute_one": {
						AttributeType: cty.String,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", "new", action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
				}, nil, nil, nil, nil, action.Delete, false),
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
				}, nil, nil, nil, nil, action.Create, false),
			},
		},
		"delete_attribute": {
			before: map[string]interface{}{
				"attribute_one": "old",
			},
			after: map[string]interface{}{},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"attribute_one": {
						AttributeType: cty.String,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
			}, nil, nil, nil, nil, action.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
				}, nil, nil, nil, nil, action.Delete, false),
				renderers.ValidateBlock(nil, nil, nil, nil, nil, action.Create, false),
			},
		},
		"create_block": {
			before: map[string]interface{}{},
			after: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "new",
				},
			},
			block: &tjson.SchemaBlock{
				NestedBlocks: map[string]*tjson.SchemaBlockType{
					"block_one": {
						Block: &tjson.SchemaBlock{
							Attributes: map[string]*tjson.SchemaAttribute{
								"attribute_one": {
									AttributeType: cty.String,
								},
							},
						},
						NestingMode: "single",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
				"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
				}, nil, nil, nil, nil, action.Create, false),
			}, nil, nil, nil, action.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, nil, nil, nil, nil, action.Delete, false),
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
					}, nil, nil, nil, nil, action.Create, false),
				}, nil, nil, nil, action.Create, false),
			},
		},
		"update_block": {
			before: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "old",
				},
			},
			after: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "new",
				},
			},
			block: &tjson.SchemaBlock{
				NestedBlocks: map[string]*tjson.SchemaBlockType{
					"block_one": {
						Block: &tjson.SchemaBlock{
							Attributes: map[string]*tjson.SchemaAttribute{
								"attribute_one": {
									AttributeType: cty.String,
								},
							},
						},
						NestingMode: "single",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
				"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", "new", action.Update, false),
				}, nil, nil, nil, nil, action.Update, false),
			}, nil, nil, nil, action.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					}, nil, nil, nil, nil, action.Delete, false),
				}, nil, nil, nil, action.Delete, false),
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive(nil, "new", action.Create, false),
					}, nil, nil, nil, nil, action.Create, false),
				}, nil, nil, nil, action.Create, false),
			},
		},
		"delete_block": {
			before: map[string]interface{}{
				"block_one": map[string]interface{}{
					"attribute_one": "old",
				},
			},
			after: map[string]interface{}{},
			block: &tjson.SchemaBlock{
				NestedBlocks: map[string]*tjson.SchemaBlockType{
					"block_one": {
						Block: &tjson.SchemaBlock{
							Attributes: map[string]*tjson.SchemaAttribute{
								"attribute_one": {
									AttributeType: cty.String,
								},
							},
						},
						NestingMode: "single",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
				"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
					"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
				}, nil, nil, nil, nil, action.Delete, false),
			}, nil, nil, nil, action.Update, false),
			validateSet: []renderers.ValidateDiffFunction{
				renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_one": renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"attribute_one": renderers.ValidatePrimitive("old", nil, action.Delete, false),
					}, nil, nil, nil, nil, action.Delete, false),
				}, nil, nil, nil, action.Delete, false),
				renderers.ValidateBlock(nil, nil, nil, nil, nil, action.Create, false),
			},
		},
	}
	for name, tmp := range tcs {
		tc := tmp

		t.Run(name, func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				input := structured.Change{
					Before: map[string]interface{}{
						"block_type": tc.before,
					},
					After: map[string]interface{}{
						"block_type": tc.after,
					},
					ReplacePaths:       &attributepath.PathMatcher{},
					RelevantAttributes: attributepath.AlwaysMatcher(),
				}

				block := &tjson.SchemaBlock{
					NestedBlocks: map[string]*tjson.SchemaBlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "single",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, map[string]renderers.ValidateDiffFunction{
					"block_type": tc.validate,
				}, nil, nil, nil, action.Update, false)

				computedDiff, err := ComputeDiffForBlock(input, block)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
			t.Run("map", func(t *testing.T) {
				input := structured.Change{
					Before: map[string]interface{}{
						"block_type": map[string]interface{}{
							"one": tc.before,
						},
					},
					After: map[string]interface{}{
						"block_type": map[string]interface{}{
							"one": tc.after,
						},
					},
					ReplacePaths:       &attributepath.PathMatcher{},
					RelevantAttributes: attributepath.AlwaysMatcher(),
				}

				block := &tjson.SchemaBlock{
					NestedBlocks: map[string]*tjson.SchemaBlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "map",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, nil, nil, map[string]map[string]renderers.ValidateDiffFunction{
					"block_type": {
						"one": tc.validate,
					},
				}, nil, action.Update, false)

				computedDiff, err := ComputeDiffForBlock(input, block)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
			t.Run("list", func(t *testing.T) {
				input := structured.Change{
					Before: map[string]interface{}{
						"block_type": []interface{}{
							tc.before,
						},
					},
					After: map[string]interface{}{
						"block_type": []interface{}{
							tc.after,
						},
					},
					ReplacePaths:       &attributepath.PathMatcher{},
					RelevantAttributes: attributepath.AlwaysMatcher(),
				}

				block := &tjson.SchemaBlock{
					NestedBlocks: map[string]*tjson.SchemaBlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "list",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, nil, map[string][]renderers.ValidateDiffFunction{
					"block_type": {
						tc.validate,
					},
				}, nil, nil, action.Update, false)

				computedDiff, err := ComputeDiffForBlock(input, block)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
			t.Run("set", func(t *testing.T) {
				input := structured.Change{
					Before: map[string]interface{}{
						"block_type": []interface{}{
							tc.before,
						},
					},
					After: map[string]interface{}{
						"block_type": []interface{}{
							tc.after,
						},
					},
					ReplacePaths:       &attributepath.PathMatcher{},
					RelevantAttributes: attributepath.AlwaysMatcher(),
				}

				block := &tjson.SchemaBlock{
					NestedBlocks: map[string]*tjson.SchemaBlockType{
						"block_type": {
							Block:       tc.block,
							NestingMode: "set",
						},
					},
				}

				validate := renderers.ValidateBlock(nil, nil, nil, nil, map[string][]renderers.ValidateDiffFunction{
					"block_type": func() []renderers.ValidateDiffFunction {
						if tc.validateSet != nil {
							return tc.validateSet
						}
						return []renderers.ValidateDiffFunction{tc.validate}
					}(),
				}, action.Update, false)

				computedDiff, err := ComputeDiffForBlock(input, block)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
		})
	}
}

func TestValue_Outputs(t *testing.T) {
	tcs := map[string]struct {
		input        structured.Change
		validateDiff renderers.ValidateDiffFunction
	}{
		"primitive_create": {
			input: structured.Change{
				Before: nil,
				After:  "new",
			},
			validateDiff: renderers.ValidatePrimitive(nil, "new", action.Create, false),
		},
		"object_create": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
				"element_two": renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
			}, action.Create, false),
		},
		"list_create": {
			input: structured.Change{
				Before: nil,
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
				renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
			}, action.Create, false),
		},
		"primitive_update": {
			input: structured.Change{
				Before: "old",
				After:  "new",
			},
			validateDiff: renderers.ValidatePrimitive("old", "new", action.Update, false),
		},
		"object_update": {
			input: structured.Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive("old_one", "new_one", action.Update, false),
				"element_two": renderers.ValidatePrimitive("old_two", "new_two", action.Update, false),
			}, action.Update, false),
		},
		"list_update": {
			input: structured.Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
				renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
				renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
			}, action.Update, false),
		},
		"primitive_delete": {
			input: structured.Change{
				Before: "old",
				After:  nil,
			},
			validateDiff: renderers.ValidatePrimitive("old", nil, action.Delete, false),
		},
		"object_delete": {
			input: structured.Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: nil,
			},
			validateDiff: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
				"element_two": renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
			}, action.Delete, false),
		},
		"list_delete": {
			input: structured.Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: nil,
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
				renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
			}, action.Delete, false),
		},
		"primitive_to_list": {
			input: structured.Change{
				Before: "old",
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
					renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
				}, action.Create, false), action.Update, false),
		},
		"primitive_to_object": {
			input: structured.Change{
				Before: "old",
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
					"element_two": renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
				}, action.Create, false), action.Update, false),
		},
		"list_to_primitive": {
			input: structured.Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: "new",
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
					renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
				}, action.Delete, false),
				renderers.ValidatePrimitive(nil, "new", action.Create, false),
				action.Update, false),
		},
		"list_to_object": {
			input: structured.Change{
				Before: []interface{}{
					"old_one",
					"old_two",
				},
				After: map[string]interface{}{
					"element_one": "new_one",
					"element_two": "new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
					renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
				}, action.Delete, false),
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
					"element_two": renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
				}, action.Create, false), action.Update, false),
		},
		"object_to_primitive": {
			input: structured.Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: "new",
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
					"element_two": renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
				}, action.Delete, false),
				renderers.ValidatePrimitive(nil, "new", action.Create, false),
				action.Update, false),
		},
		"object_to_list": {
			input: structured.Change{
				Before: map[string]interface{}{
					"element_one": "old_one",
					"element_two": "old_two",
				},
				After: []interface{}{
					"new_one",
					"new_two",
				},
			},
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"element_one": renderers.ValidatePrimitive("old_one", nil, action.Delete, false),
					"element_two": renderers.ValidatePrimitive("old_two", nil, action.Delete, false),
				}, action.Delete, false),
				renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive(nil, "new_one", action.Create, false),
					renderers.ValidatePrimitive(nil, "new_two", action.Create, false),
				}, action.Create, false), action.Update, false),
		},
	}

	for name, tc := range tcs {

		// Let's set some default values on the input.
		if tc.input.RelevantAttributes == nil {
			tc.input.RelevantAttributes = attributepath.AlwaysMatcher()
		}
		if tc.input.ReplacePaths == nil {
			tc.input.ReplacePaths = &attributepath.PathMatcher{}
		}

		t.Run(name, func(t *testing.T) {
			computedDiff, err := ComputeDiffForOutput(tc.input)
			require.NoError(t, err)

			tc.validateDiff(t, computedDiff)
		})
	}
}

func TestValue_PrimitiveAttributes(t *testing.T) {
	// This function tests manipulating primitives: creating, deleting and
	// updating. It also automatically tests these operations within the
	// contexts of collections.

	tcs := map[string]struct {
		input              structured.Change
		attribute          cty.Type
		validateDiff       renderers.ValidateDiffFunction
		validateSliceDiffs []renderers.ValidateDiffFunction // Lists are special in some cases.
	}{
		"primitive_create": {
			input: structured.Change{
				After: "new",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive(nil, "new", action.Create, false),
		},
		"primitive_delete": {
			input: structured.Change{
				Before: "old",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", nil, action.Delete, false),
		},
		"primitive_update": {
			input: structured.Change{
				Before: "old",
				After:  "new",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", "new", action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, "new", action.Create, false),
			},
		},
		"primitive_set_explicit_null": {
			input: structured.Change{
				Before:        "old",
				After:         nil,
				AfterExplicit: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", nil, action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, nil, action.Create, false),
			},
		},
		"primitive_unset_explicit_null": {
			input: structured.Change{
				BeforeExplicit: true,
				Before:         nil,
				After:          "new",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive(nil, "new", action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, "new", action.Create, false),
			},
		},
		"primitive_create_sensitive": {
			input: structured.Change{
				Before:         nil,
				After:          "new",
				AfterSensitive: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", action.Create, false), false, true, action.Create, false),
		},
		"primitive_delete_sensitive": {
			input: structured.Change{
				Before:          "old",
				BeforeSensitive: true,
				After:           nil,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, action.Delete, false), true, false, action.Delete, false),
		},
		"primitive_update_sensitive": {
			input: structured.Change{
				Before:          "old",
				BeforeSensitive: true,
				After:           "new",
				AfterSensitive:  true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateSensitive(renderers.ValidatePrimitive("old", "new", action.Update, false), true, true, action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidateSensitive(renderers.ValidatePrimitive("old", nil, action.Delete, false), true, false, action.Delete, false),
				renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "new", action.Create, false), false, true, action.Create, false),
			},
		},
		"primitive_create_computed": {
			input: structured.Change{
				Before:  nil,
				After:   nil,
				Unknown: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateUnknown(nil, action.Create, false),
		},
		"primitive_update_computed": {
			input: structured.Change{
				Before:  "old",
				After:   nil,
				Unknown: true,
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidateUnknown(renderers.ValidatePrimitive("old", nil, action.Delete, false), action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidateUnknown(nil, action.Create, false),
			},
		},
		"primitive_update_replace": {
			input: structured.Change{
				Before: "old",
				After:  "new",
				ReplacePaths: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{}, // An empty path suggests replace should be true.
					},
				},
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", "new", action.Update, true),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, action.Delete, true),
				renderers.ValidatePrimitive(nil, "new", action.Create, true),
			},
		},
		"noop": {
			input: structured.Change{
				Before: "old",
				After:  "old",
			},
			attribute:    cty.String,
			validateDiff: renderers.ValidatePrimitive("old", "old", action.NoOp, false),
		},
		"dynamic": {
			input: structured.Change{
				Before: "old",
				After:  "new",
			},
			attribute:    cty.DynamicPseudoType,
			validateDiff: renderers.ValidatePrimitive("old", "new", action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, "new", action.Create, false),
			},
		},
		"dynamic_type_change": {
			input: structured.Change{
				Before: "old",
				After:  4.0,
			},
			attribute: cty.DynamicPseudoType,
			validateDiff: renderers.ValidateTypeChange(
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, 4.0, action.Create, false),
				action.Update, false),
			validateSliceDiffs: []renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("old", nil, action.Delete, false),
				renderers.ValidatePrimitive(nil, 4.0, action.Create, false),
			},
		},
	}
	for name, tmp := range tcs {
		tc := tmp

		// Let's set some default values on the input.
		if tc.input.RelevantAttributes == nil {
			tc.input.RelevantAttributes = attributepath.AlwaysMatcher()
		}
		if tc.input.ReplacePaths == nil {
			tc.input.ReplacePaths = &attributepath.PathMatcher{}
		}

		defaultCollectionsAction := action.Update
		if name == "noop" {
			defaultCollectionsAction = action.NoOp
		}

		t.Run(name, func(t *testing.T) {
			t.Run("direct", func(t *testing.T) {
				computedDiff, err := ComputeDiffForAttribute(tc.input, &tjson.SchemaAttribute{
					AttributeType: tc.attribute,
				})
				require.NoError(t, err)

				tc.validateDiff(t, computedDiff)
			})

			t.Run("map", func(t *testing.T) {
				input := wrapChangeInMap(tc.input)
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.Map(tc.attribute),
				}

				validate := renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"element": tc.validateDiff,
				}, defaultCollectionsAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("list", func(t *testing.T) {
				input := wrapChangeInSlice(tc.input)
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.List(tc.attribute),
				}

				if tc.validateSliceDiffs != nil {
					validate := renderers.ValidateList(tc.validateSliceDiffs, defaultCollectionsAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateList([]renderers.ValidateDiffFunction{
					tc.validateDiff,
				}, defaultCollectionsAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})

			t.Run("set", func(t *testing.T) {
				input := wrapChangeInSlice(tc.input)
				attribute := &tjson.SchemaAttribute{
					AttributeType: cty.Set(tc.attribute),
				}

				if tc.validateSliceDiffs != nil {
					validate := renderers.ValidateSet(tc.validateSliceDiffs, defaultCollectionsAction, false)

					computedDiff, err := ComputeDiffForAttribute(input, attribute)
					require.NoError(t, err)
					validate(t, computedDiff)
					return
				}

				validate := renderers.ValidateSet([]renderers.ValidateDiffFunction{
					tc.validateDiff,
				}, defaultCollectionsAction, false)

				computedDiff, err := ComputeDiffForAttribute(input, attribute)
				require.NoError(t, err)
				validate(t, computedDiff)
			})
		})
	}
}

func TestValue_CollectionAttributes(t *testing.T) {
	// This function tests creating and deleting collections. Note, it does not
	// generally cover editing collections except in special cases as editing
	// collections is handled automatically by other functions.
	tcs := map[string]struct {
		input        structured.Change
		attribute    *tjson.SchemaAttribute
		validateDiff renderers.ValidateDiffFunction
	}{
		"map_create_empty": {
			input: structured.Change{
				Before: nil,
				After:  map[string]interface{}{},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateMap(nil, action.Create, false),
		},
		"map_create_populated": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"element_one": "one",
					"element_two": "two",
				},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive(nil, "one", action.Create, false),
				"element_two": renderers.ValidatePrimitive(nil, "two", action.Create, false),
			}, action.Create, false),
		},
		"map_delete_empty": {
			input: structured.Change{
				Before: map[string]interface{}{},
				After:  nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateMap(nil, action.Delete, false),
		},
		"map_delete_populated": {
			input: structured.Change{
				Before: map[string]interface{}{
					"element_one": "one",
					"element_two": "two",
				},
				After: nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
				"element_one": renderers.ValidatePrimitive("one", nil, action.Delete, false),
				"element_two": renderers.ValidatePrimitive("two", nil, action.Delete, false),
			}, action.Delete, false),
		},
		"map_create_sensitive": {
			input: structured.Change{
				Before:         nil,
				After:          map[string]interface{}{},
				AfterSensitive: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateMap(nil, action.Create, false), false, true, action.Create, false),
		},
		"map_update_sensitive": {
			input: structured.Change{
				Before: map[string]interface{}{
					"element": "one",
				},
				BeforeSensitive: true,
				After:           map[string]interface{}{},
				AfterSensitive:  true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
				"element": renderers.ValidatePrimitive("one", nil, action.Delete, false),
			}, action.Update, false), true, true, action.Update, false),
		},
		"map_delete_sensitive": {
			input: structured.Change{
				Before:          map[string]interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateMap(nil, action.Delete, false), true, false, action.Delete, false),
		},
		"map_create_unknown": {
			input: structured.Change{
				Before:  nil,
				After:   map[string]interface{}{},
				Unknown: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateUnknown(nil, action.Create, false),
		},
		"map_update_unknown": {
			input: structured.Change{
				Before: map[string]interface{}{},
				After: map[string]interface{}{
					"element": "one",
				},
				Unknown: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Map(cty.String),
			},
			validateDiff: renderers.ValidateUnknown(renderers.ValidateMap(nil, action.Delete, false), action.Update, false),
		},
		"list_create_empty": {
			input: structured.Change{
				Before: nil,
				After:  []interface{}{},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateList(nil, action.Create, false),
		},
		"list_create_populated": {
			input: structured.Change{
				Before: nil,
				After:  []interface{}{"one", "two"},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, "one", action.Create, false),
				renderers.ValidatePrimitive(nil, "two", action.Create, false),
			}, action.Create, false),
		},
		"list_delete_empty": {
			input: structured.Change{
				Before: []interface{}{},
				After:  nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateList(nil, action.Delete, false),
		},
		"list_delete_populated": {
			input: structured.Change{
				Before: []interface{}{"one", "two"},
				After:  nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, action.Delete, false),
				renderers.ValidatePrimitive("two", nil, action.Delete, false),
			}, action.Delete, false),
		},
		"list_create_sensitive": {
			input: structured.Change{
				Before:         nil,
				After:          []interface{}{},
				AfterSensitive: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateList(nil, action.Create, false), false, true, action.Create, false),
		},
		"list_update_sensitive": {
			input: structured.Change{
				Before:          []interface{}{"one"},
				BeforeSensitive: true,
				After:           []interface{}{},
				AfterSensitive:  true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, action.Delete, false),
			}, action.Update, false), true, true, action.Update, false),
		},
		"list_delete_sensitive": {
			input: structured.Change{
				Before:          []interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateList(nil, action.Delete, false), true, false, action.Delete, false),
		},
		"list_create_unknown": {
			input: structured.Change{
				Before:  nil,
				After:   []interface{}{},
				Unknown: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateUnknown(nil, action.Create, false),
		},
		"list_update_unknown": {
			input: structured.Change{
				Before:  []interface{}{},
				After:   []interface{}{"one"},
				Unknown: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.List(cty.String),
			},
			validateDiff: renderers.ValidateUnknown(renderers.ValidateList(nil, action.Delete, false), action.Update, false),
		},
		"set_create_empty": {
			input: structured.Change{
				Before: nil,
				After:  []interface{}{},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSet(nil, action.Create, false),
		},
		"set_create_populated": {
			input: structured.Change{
				Before: nil,
				After:  []interface{}{"one", "two"},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSet([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive(nil, "one", action.Create, false),
				renderers.ValidatePrimitive(nil, "two", action.Create, false),
			}, action.Create, false),
		},
		"set_delete_empty": {
			input: structured.Change{
				Before: []interface{}{},
				After:  nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSet(nil, action.Delete, false),
		},
		"set_delete_populated": {
			input: structured.Change{
				Before: []interface{}{"one", "two"},
				After:  nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSet([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, action.Delete, false),
				renderers.ValidatePrimitive("two", nil, action.Delete, false),
			}, action.Delete, false),
		},
		"set_create_sensitive": {
			input: structured.Change{
				Before:         nil,
				After:          []interface{}{},
				AfterSensitive: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateSet(nil, action.Create, false), false, true, action.Create, false),
		},
		"set_update_sensitive": {
			input: structured.Change{
				Before:          []interface{}{"one"},
				BeforeSensitive: true,
				After:           []interface{}{},
				AfterSensitive:  true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateSet([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", nil, action.Delete, false),
			}, action.Update, false), true, true, action.Update, false),
		},
		"set_delete_sensitive": {
			input: structured.Change{
				Before:          []interface{}{},
				BeforeSensitive: true,
				After:           nil,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateSensitive(renderers.ValidateSet(nil, action.Delete, false), true, false, action.Delete, false),
		},
		"set_create_unknown": {
			input: structured.Change{
				Before:  nil,
				After:   []interface{}{},
				Unknown: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateUnknown(nil, action.Create, false),
		},
		"set_update_unknown": {
			input: structured.Change{
				Before:  []interface{}{},
				After:   []interface{}{"one"},
				Unknown: true,
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Set(cty.String),
			},
			validateDiff: renderers.ValidateUnknown(renderers.ValidateSet(nil, action.Delete, false), action.Update, false),
		},
		"tuple_primitive": {
			input: structured.Change{
				Before: []interface{}{
					"one",
					2.0,
					"three",
				},
				After: []interface{}{
					"one",
					4.0,
					"three",
				},
			},
			attribute: &tjson.SchemaAttribute{
				AttributeType: cty.Tuple([]cty.Type{cty.String, cty.Number, cty.String}),
			},
			validateDiff: renderers.ValidateList([]renderers.ValidateDiffFunction{
				renderers.ValidatePrimitive("one", "one", action.NoOp, false),
				renderers.ValidatePrimitive(2.0, 4.0, action.Update, false),
				renderers.ValidatePrimitive("three", "three", action.NoOp, false),
			}, action.Update, false),
		},
	}

	for name, tc := range tcs {

		// Let's set some default values on the input.
		if tc.input.RelevantAttributes == nil {
			tc.input.RelevantAttributes = attributepath.AlwaysMatcher()
		}
		if tc.input.ReplacePaths == nil {
			tc.input.ReplacePaths = &attributepath.PathMatcher{}
		}

		t.Run(name, func(t *testing.T) {
			computedDiff, err := ComputeDiffForAttribute(tc.input, tc.attribute)
			require.NoError(t, err)

			tc.validateDiff(t, computedDiff)
		})
	}
}

func TestRelevantAttributes(t *testing.T) {
	tcs := map[string]struct {
		input    structured.Change
		block    *tjson.SchemaBlock
		validate renderers.ValidateDiffFunction
	}{
		"simple_attributes": {
			input: structured.Change{
				Before: map[string]interface{}{
					"id":     "old_id",
					"ignore": "doesn't matter",
				},
				After: map[string]interface{}{
					"id":     "new_id",
					"ignore": "doesn't matter but modified",
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{
							"id",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"id": {
						AttributeType: cty.String,
					},
					"ignore": {
						AttributeType: cty.String,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"id":     renderers.ValidatePrimitive("old_id", "new_id", action.Update, false),
				"ignore": renderers.ValidatePrimitive("doesn't matter", "doesn't matter", action.NoOp, false),
			}, nil, nil, nil, nil, action.Update, false),
		},
		"nested_attributes": {
			input: structured.Change{
				Before: map[string]interface{}{
					"list_block": []interface{}{
						map[string]interface{}{
							"id": "old_one",
						},
						map[string]interface{}{
							"id": "ignored",
						},
					},
				},
				After: map[string]interface{}{
					"list_block": []interface{}{
						map[string]interface{}{
							"id": "new_one",
						},
						map[string]interface{}{
							"id": "ignored_but_changed",
						},
					},
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{
							"list_block",
							float64(0),
							"id",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				NestedBlocks: map[string]*tjson.SchemaBlockType{
					"list_block": {
						Block: &tjson.SchemaBlock{
							Attributes: map[string]*tjson.SchemaAttribute{
								"id": {
									AttributeType: cty.String,
								},
							},
						},
						NestingMode: "list",
					},
				},
			},
			validate: renderers.ValidateBlock(nil, nil, map[string][]renderers.ValidateDiffFunction{
				"list_block": {
					renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"id": renderers.ValidatePrimitive("old_one", "new_one", action.Update, false),
					}, nil, nil, nil, nil, action.Update, false),
					renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
						"id": renderers.ValidatePrimitive("ignored", "ignored", action.NoOp, false),
					}, nil, nil, nil, nil, action.NoOp, false),
				},
			}, nil, nil, action.Update, false),
		},
		"nested_attributes_in_object": {
			input: structured.Change{
				Before: map[string]interface{}{
					"object": map[string]interface{}{
						"id": "old_id",
					},
				},
				After: map[string]interface{}{
					"object": map[string]interface{}{
						"id": "new_id",
					},
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Propagate: true,
					Paths: [][]interface{}{
						{
							"object", // Even though we just specify object, it should now include every below object as well.
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"object": {
						AttributeType: cty.Object(map[string]cty.Type{
							"id": cty.String,
						}),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"object": renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"id": renderers.ValidatePrimitive("old_id", "new_id", action.Update, false),
				}, action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
		},
		"elements_in_list": {
			input: structured.Change{
				Before: map[string]interface{}{
					"list": []interface{}{
						0, 1, 2, 3, 4,
					},
				},
				After: map[string]interface{}{
					"list": []interface{}{
						0, 5, 6, 7, 4,
					},
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Paths: [][]interface{}{ // The list is actually just going to ignore this.
						{
							"list",
							float64(0),
						},
						{
							"list",
							float64(2),
						},
						{
							"list",
							float64(4),
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"list": {
						AttributeType: cty.List(cty.Number),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				// The list validator below just ignores our relevant
				// attributes. This is deliberate.
				"list": renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive(0, 0, action.NoOp, false),
					renderers.ValidatePrimitive(1, nil, action.Delete, false),
					renderers.ValidatePrimitive(2, nil, action.Delete, false),
					renderers.ValidatePrimitive(3, nil, action.Delete, false),
					renderers.ValidatePrimitive(nil, 5, action.Create, false),
					renderers.ValidatePrimitive(nil, 6, action.Create, false),
					renderers.ValidatePrimitive(nil, 7, action.Create, false),
					renderers.ValidatePrimitive(4, 4, action.NoOp, false),
				}, action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
		},
		"elements_in_map": {
			input: structured.Change{
				Before: map[string]interface{}{
					"map": map[string]interface{}{
						"key_one":   "value_one",
						"key_two":   "value_two",
						"key_three": "value_three",
					},
				},
				After: map[string]interface{}{
					"map": map[string]interface{}{
						"key_one":  "value_three",
						"key_two":  "value_seven",
						"key_four": "value_four",
					},
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Paths: [][]interface{}{
						{
							"map",
							"key_one",
						},
						{
							"map",
							"key_three",
						},
						{
							"map",
							"key_four",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"map": {
						AttributeType: cty.Map(cty.String),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"map": renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"key_one":   renderers.ValidatePrimitive("value_one", "value_three", action.Update, false),
					"key_two":   renderers.ValidatePrimitive("value_two", "value_two", action.NoOp, false),
					"key_three": renderers.ValidatePrimitive("value_three", nil, action.Delete, false),
					"key_four":  renderers.ValidatePrimitive(nil, "value_four", action.Create, false),
				}, action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
		},
		"elements_in_set": {
			input: structured.Change{
				Before: map[string]interface{}{
					"set": []interface{}{
						0, 1, 2, 3, 4,
					},
				},
				After: map[string]interface{}{
					"set": []interface{}{
						0, 2, 4, 5, 6,
					},
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Propagate: true,
					Paths: [][]interface{}{
						{
							"set",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"set": {
						AttributeType: cty.Set(cty.Number),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"set": renderers.ValidateSet([]renderers.ValidateDiffFunction{
					renderers.ValidatePrimitive(0, 0, action.NoOp, false),
					renderers.ValidatePrimitive(1, nil, action.Delete, false),
					renderers.ValidatePrimitive(2, 2, action.NoOp, false),
					renderers.ValidatePrimitive(3, nil, action.Delete, false),
					renderers.ValidatePrimitive(4, 4, action.NoOp, false),
					renderers.ValidatePrimitive(nil, 5, action.Create, false),
					renderers.ValidatePrimitive(nil, 6, action.Create, false),
				}, action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
		},
		"dynamic_types": {
			input: structured.Change{
				Before: map[string]interface{}{
					"dynamic_nested_type": map[string]interface{}{
						"nested_id": "nomatch",
						"nested_object": map[string]interface{}{
							"nested_nested_id": "matched",
						},
					},
					"dynamic_nested_type_match": map[string]interface{}{
						"nested_id": "allmatch",
						"nested_object": map[string]interface{}{
							"nested_nested_id": "allmatch",
						},
					},
				},
				After: map[string]interface{}{
					"dynamic_nested_type": map[string]interface{}{
						"nested_id": "nomatch_changed",
						"nested_object": map[string]interface{}{
							"nested_nested_id": "matched",
						},
					},
					"dynamic_nested_type_match": map[string]interface{}{
						"nested_id": "allmatch",
						"nested_object": map[string]interface{}{
							"nested_nested_id": "allmatch",
						},
					},
				},
				RelevantAttributes: &attributepath.PathMatcher{
					Propagate: true,
					Paths: [][]interface{}{
						{
							"dynamic_nested_type",
							"nested_object",
							"nested_nested_id",
						},
						{
							"dynamic_nested_type_match",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"dynamic_nested_type": {
						AttributeType: cty.DynamicPseudoType,
					},
					"dynamic_nested_type_match": {
						AttributeType: cty.DynamicPseudoType,
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"dynamic_nested_type": renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"nested_id": renderers.ValidatePrimitive("nomatch", "nomatch", action.NoOp, false),
					"nested_object": renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
						"nested_nested_id": renderers.ValidatePrimitive("matched", "matched", action.NoOp, false),
					}, action.NoOp, false),
				}, action.NoOp, false),
				"dynamic_nested_type_match": renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
					"nested_id": renderers.ValidatePrimitive("allmatch", "allmatch", action.NoOp, false),
					"nested_object": renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
						"nested_nested_id": renderers.ValidatePrimitive("allmatch", "allmatch", action.NoOp, false),
					}, action.NoOp, false),
				}, action.NoOp, false),
			}, nil, nil, nil, nil, action.NoOp, false),
		},
	}
	for name, tc := range tcs {
		if tc.input.ReplacePaths == nil {
			tc.input.ReplacePaths = &attributepath.PathMatcher{}
		}
		t.Run(name, func(t *testing.T) {
			computedDiff, err := ComputeDiffForBlock(tc.input, tc.block)
			require.NoError(t, err)
			tc.validate(t, computedDiff)
		})
	}
}

func TestDynamicPseudoType(t *testing.T) {
	tcs := map[string]struct {
		input    structured.Change
		validate renderers.ValidateDiffFunction
	}{
		"after_sensitive_in_dynamic_type": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"key": "value",
				},
				Unknown:         false,
				BeforeSensitive: false,
				AfterSensitive: map[string]interface{}{
					"key": true,
				},
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			validate: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"key": renderers.ValidateSensitive(renderers.ValidatePrimitive(nil, "value", action.Create, false), false, true, action.Create, false),
			}, action.Create, false),
		},
		"before_sensitive_in_dynamic_type": {
			input: structured.Change{
				Before: map[string]interface{}{
					"key": "value",
				},
				After:   nil,
				Unknown: false,
				BeforeSensitive: map[string]interface{}{
					"key": true,
				},
				AfterSensitive:     false,
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			validate: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"key": renderers.ValidateSensitive(renderers.ValidatePrimitive("value", nil, action.Delete, false), true, false, action.Delete, false),
			}, action.Delete, false),
		},
		"sensitive_in_dynamic_type": {
			input: structured.Change{
				Before: map[string]interface{}{
					"key": "before",
				},
				After: map[string]interface{}{
					"key": "after",
				},
				Unknown: false,
				BeforeSensitive: map[string]interface{}{
					"key": true,
				},
				AfterSensitive: map[string]interface{}{
					"key": true,
				},
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			validate: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"key": renderers.ValidateSensitive(renderers.ValidatePrimitive("before", "after", action.Update, false), true, true, action.Update, false),
			}, action.Update, false),
		},
		"create_unknown_in_dynamic_type": {
			input: structured.Change{
				Before: nil,
				After:  map[string]interface{}{},
				Unknown: map[string]interface{}{
					"key": true,
				},
				BeforeSensitive:    false,
				AfterSensitive:     false,
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			validate: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"key": renderers.ValidateUnknown(nil, action.Create, false),
			}, action.Create, false),
		},
		"update_unknown_in_dynamic_type": {
			input: structured.Change{
				Before: map[string]interface{}{
					"key": "before",
				},
				After: map[string]interface{}{},
				Unknown: map[string]interface{}{
					"key": true,
				},
				BeforeSensitive:    false,
				AfterSensitive:     false,
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			validate: renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
				"key": renderers.ValidateUnknown(renderers.ValidatePrimitive("before", nil, action.Delete, false), action.Update, false),
			}, action.Update, false),
		},
	}
	for key, tc := range tcs {
		t.Run(key, func(t *testing.T) {
			computedDiff, err := ComputeDiffForType(tc.input, cty.DynamicPseudoType)
			require.NoError(t, err)

			tc.validate(t, computedDiff)
		})
	}
}

func TestSpecificCases(t *testing.T) {
	// This is a special test that can contain any combination of individual
	// cases and will execute against them. For testing/fixing specific issues
	// you can generally put the test case in here.
	tcs := map[string]struct {
		input    structured.Change
		block    *tjson.SchemaBlock
		validate renderers.ValidateDiffFunction
	}{
		"issues/33016/unknown": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"triggers": map[string]interface{}{},
				},
				Unknown: map[string]interface{}{
					"id": true,
					"triggers": map[string]interface{}{
						"rotation": true,
					},
				},
				BeforeSensitive: false,
				AfterSensitive: map[string]interface{}{
					"triggers": map[string]interface{}{},
				},
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"id": {
						AttributeType: cty.String,
					},
					"triggers": {
						AttributeType: cty.Map(cty.String),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"id": renderers.ValidateUnknown(nil, action.Create, false),
				"triggers": renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"rotation": renderers.ValidateUnknown(nil, action.Create, false),
				}, action.Create, false),
			}, nil, nil, nil, nil, action.Create, false),
		},
		"issues/33016/null": {
			input: structured.Change{
				Before: nil,
				After: map[string]interface{}{
					"triggers": map[string]interface{}{
						"rotation": nil,
					},
				},
				Unknown: map[string]interface{}{
					"id":       true,
					"triggers": map[string]interface{}{},
				},
				BeforeSensitive: false,
				AfterSensitive: map[string]interface{}{
					"triggers": map[string]interface{}{},
				},
				ReplacePaths:       attributepath.Empty(false),
				RelevantAttributes: attributepath.AlwaysMatcher(),
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"id": {
						AttributeType: cty.String,
					},
					"triggers": {
						AttributeType: cty.Map(cty.String),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"id": renderers.ValidateUnknown(nil, action.Create, false),
				"triggers": renderers.ValidateMap(map[string]renderers.ValidateDiffFunction{
					"rotation": renderers.ValidatePrimitive(nil, nil, action.Create, false),
				}, action.Create, false),
			}, nil, nil, nil, nil, action.Create, false),
		},

		// The following tests are from issue 33472. Basically Terraform allows
		// callers to treat numbers as strings in references and expects us
		// to coerce the strings into numbers. For example the following are
		// equivalent.
		//    - test_resource.resource.list[0].attribute
		//    - test_resource.resource.list["0"].attribute
		//
		// We need our attributepath package (used within the ReplacePaths and
		// RelevantAttributes fields) to handle coercing strings into numbers
		// when it's expected.

		"issues/33472/expected": {
			input: structured.Change{
				Before: map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{
							"number": -1,
						},
					},
				},
				After: map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{
							"number": 2,
						},
					},
				},
				Unknown:         false,
				BeforeSensitive: false,
				AfterSensitive:  false,
				ReplacePaths:    attributepath.Empty(false),
				RelevantAttributes: &attributepath.PathMatcher{
					Propagate: true,
					Paths: [][]interface{}{
						{
							"list",
							0.0, // This is normal and expected so easy case.
							"number",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"list": {
						AttributeType: cty.List(cty.Object(map[string]cty.Type{
							"number": cty.Number,
						})),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"list": renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
						"number": renderers.ValidatePrimitive(-1, 2, action.Update, false),
					}, action.Update, false),
				}, action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
		},

		"issues/33472/coerce": {
			input: structured.Change{
				Before: map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{
							"number": -1,
						},
					},
				},
				After: map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{
							"number": 2,
						},
					},
				},
				Unknown:         false,
				BeforeSensitive: false,
				AfterSensitive:  false,
				ReplacePaths:    attributepath.Empty(false),
				RelevantAttributes: &attributepath.PathMatcher{
					Propagate: true,
					Paths: [][]interface{}{
						{
							"list",
							"0", // Difficult but allowed, we need to handle this.
							"number",
						},
					},
				},
			},
			block: &tjson.SchemaBlock{
				Attributes: map[string]*tjson.SchemaAttribute{
					"list": {
						AttributeType: cty.List(cty.Object(map[string]cty.Type{
							"number": cty.Number,
						})),
					},
				},
			},
			validate: renderers.ValidateBlock(map[string]renderers.ValidateDiffFunction{
				"list": renderers.ValidateList([]renderers.ValidateDiffFunction{
					renderers.ValidateObject(map[string]renderers.ValidateDiffFunction{
						"number": renderers.ValidatePrimitive(-1, 2, action.Update, false),
					}, action.Update, false),
				}, action.Update, false),
			}, nil, nil, nil, nil, action.Update, false),
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			computedDiff, err := ComputeDiffForBlock(tc.input, tc.block)
			require.NoError(t, err)
			tc.validate(t, computedDiff)
		})
	}
}

// wrapChangeInSlice does the same as wrapChangeInMap, except it wraps it into a
// slice internally.
func wrapChangeInSlice(input structured.Change) structured.Change {
	return wrapChange(input, float64(0), func(value interface{}, unknown interface{}, explicit bool) interface{} {
		switch value.(type) {
		case nil:
			if set, ok := unknown.(bool); (set && ok) || explicit {
				return []interface{}{nil}

			}
			return []interface{}{}
		default:
			return []interface{}{value}
		}
	})
}

// wrapChangeInMap access a single structured.Change and returns a new
// structured.Change that represents a map with a single element. That single
// element is the input value.
func wrapChangeInMap(input structured.Change) structured.Change {
	return wrapChange(input, "element", func(value interface{}, unknown interface{}, explicit bool) interface{} {
		switch value.(type) {
		case nil:
			if set, ok := unknown.(bool); (set && ok) || explicit {
				return map[string]interface{}{
					"element": nil,
				}
			}
			return map[string]interface{}{}
		default:
			return map[string]interface{}{
				"element": value,
			}
		}
	})
}

func wrapChange(input structured.Change, step interface{}, wrap func(interface{}, interface{}, bool) interface{}) structured.Change {

	replacePaths := &attributepath.PathMatcher{}
	for _, path := range input.ReplacePaths.(*attributepath.PathMatcher).Paths {
		var updated []interface{}
		updated = append(updated, step)
		updated = append(updated, path...)
		replacePaths.Paths = append(replacePaths.Paths, updated)
	}

	// relevantAttributes usually default to AlwaysMatcher, which means we can
	// just ignore it. But if we have had some paths specified we need to wrap
	// those as well.
	relevantAttributes := input.RelevantAttributes
	if concrete, ok := relevantAttributes.(*attributepath.PathMatcher); ok {

		newRelevantAttributes := &attributepath.PathMatcher{}
		for _, path := range concrete.Paths {
			var updated []interface{}
			updated = append(updated, step)
			updated = append(updated, path...)
			newRelevantAttributes.Paths = append(newRelevantAttributes.Paths, updated)
		}
		relevantAttributes = newRelevantAttributes
	}

	return structured.Change{
		Before:             wrap(input.Before, nil, input.BeforeExplicit),
		After:              wrap(input.After, input.Unknown, input.AfterExplicit),
		Unknown:            wrap(input.Unknown, nil, false),
		BeforeSensitive:    wrap(input.BeforeSensitive, nil, false),
		AfterSensitive:     wrap(input.AfterSensitive, nil, false),
		ReplacePaths:       replacePaths,
		RelevantAttributes: relevantAttributes,
	}
}
