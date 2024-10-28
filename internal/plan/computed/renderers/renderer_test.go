// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

func TestRenderers(t *testing.T) {
	tcs := map[string]struct {
		diff     computed.Diff
		expected node.Diff
	}{
		// We're using the string "null" in these tests to demonstrate the
		// difference between rendering an actual string and rendering a null
		// value.
		"primitive_create_string": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "null", cty.String),
				Action:   action.Create,
			},
			expected: node.NewPrimitiveDiff(
				action.Create,
				false,
				nil,
				nil,
				node.NewStringValueDiff("null", action.Create, false, false),
			),
		},
		"primitive_delete_string": {
			diff: computed.Diff{
				Renderer: Primitive("null", nil, cty.String),
				Action:   action.Delete,
			},
			expected: node.NewPrimitiveDiff(
				action.Delete,
				false,
				nil,
				node.NewStringValueDiff("null", action.Delete, false, false),
				nil,
			),
		},
		"primitive_update_multiline_string_to_null": {
			diff: computed.Diff{
				Renderer: Primitive("nu\nll", nil, cty.String),
				Action:   action.Update,
			},
			expected: node.NewPrimitiveDiff(
				action.Update,
				false,
				nil,
				node.NewStringValueDiff("nu\nll", action.Update, false, true),
				node.NewStringValueDiff("null", action.Update, false, false),
			),
		},
		"primitive_update_json_string_to_null": {
			diff: computed.Diff{
				Renderer: Primitive("[null]", nil, cty.String),
				Action:   action.Update,
			},
			expected: node.NewTypeChangeDiff(
				action.Update,
				false,
				nil,
				node.NewJSONStringDiff(
					action.Delete,
					false,
					nil,
					node.NewJSONArray(action.Delete, false, nil, []node.Diff{
						node.NewPrimitiveDiff(action.Delete, false, nil, node.NewNullValueDiff(action.Delete, false), nil),
					}),
					false,
				),
				node.NewStringValueDiff("null", action.Create, false, false),
			),
		},
		"primitive_update_json_string_from_null": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "[null]", cty.String),
				Action:   action.Update,
			},
			expected: node.NewTypeChangeDiff(
				action.Update,
				false,
				nil,
				node.NewStringValueDiff("null", action.Delete, false, false),
				node.NewJSONStringDiff(
					action.Create,
					false,
					nil,
					node.NewJSONArray(action.Create, false, nil, []node.Diff{
						node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNullValueDiff(action.Create, false)),
					}),
					false,
				),
			),
		},
		"primitive_create_null_string": {
			diff: computed.Diff{
				Renderer: Primitive(nil, nil, cty.String),
				Action:   action.Create,
			},
			expected: node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewStringValueDiff("null", action.Create, false, false)),
		},
		"primitive_delete_null_string": {
			diff: computed.Diff{
				Renderer: Primitive(nil, nil, cty.String),
				Action:   action.Delete,
			},
			expected: node.NewPrimitiveDiff(action.Delete, false, nil, node.NewStringValueDiff("null", action.Delete, false, false), nil),
		},
		"primitive_create": {
			diff: computed.Diff{
				Renderer: Primitive(nil, 1.0, cty.Number),
				Action:   action.Create,
			},
			expected: node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNumberValueDiff(1.0, action.Create, false)),
		},
		"primitive_delete": {
			diff: computed.Diff{
				Renderer: Primitive(1.0, nil, cty.Number),
				Action:   action.Delete,
			},
			expected: node.NewPrimitiveDiff(action.Delete, false, nil, node.NewNumberValueDiff(1.0, action.Delete, false), nil),
		},
		"primitive_update_replace": {
			diff: computed.Diff{
				Renderer: Primitive(0.0, 1.0, cty.Number),
				Action:   action.Update,
				Replace:  true,
			},
			expected: node.NewPrimitiveDiff(action.Update, true, nil, node.NewNumberValueDiff(0.0, action.Update, true), node.NewNumberValueDiff(1.0, action.Update, true)),
		},
		"primitive_json_string_create": {
			diff: computed.Diff{
				Renderer: Primitive(nil, "{\"key_one\": \"value_one\"}", cty.String),
				Action:   action.Create,
			},
			expected: node.NewJSONStringDiff(
				action.Create,
				false,
				nil,
				node.NewJSONObjectDiff(action.Create, false, nil, []*node.KeyValueDiff{
					node.NewKeyValueDiff(
						action.Create,
						nil,
						"key_one",
						node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewStringValueDiff("value_one", action.Create, false, false)),
						false,
						7,
					),
				}),
				false,
			),
		},
		"primitive_json_string_update": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\": \"value_one\"}", "{\"key_one\": \"value_one\",\"key_two\":\"value_two\"}", cty.String),
				Action:   action.Update,
			},
			expected: node.NewJSONStringDiff(
				action.Update,
				false,
				nil,
				node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
					node.NewKeyValueDiff(
						action.NoOp,
						nil,
						"key_one",
						node.NewPrimitiveDiff(
							action.NoOp,
							false,
							nil,
							nil,
							node.NewStringValueDiff("value_one", action.NoOp, false, false),
						),
						false,
						7,
					),
					node.NewKeyValueDiff(
						action.Create,
						nil,
						"key_two",
						node.NewPrimitiveDiff(
							action.Create,
							false,
							nil,
							nil,
							node.NewStringValueDiff("value_two", action.Create, false, false),
						),
						false,
						7,
					),
				}),
				false,
			),
		},
		"primitive_json_explicit_nulls": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\":\"value_one\",\"key_two\":\"value_two\"}", "{\"key_one\":null}", cty.String),
				Action:   action.Update,
			},
			expected: node.NewJSONStringDiff(
				action.Update,
				false,
				nil,
				node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
					node.NewKeyValueDiff(
						action.Update,
						nil,
						"key_one",
						node.NewTypeChangeDiff(
							action.Update,
							false,
							nil,
							node.NewStringValueDiff("value_one", action.Delete, false, false),
							node.NewNullValueDiff(action.Create, false),
						),
						false,
						7,
					),
					node.NewKeyValueDiff(
						action.Delete,
						nil,
						"key_two",
						node.NewPrimitiveDiff(
							action.Delete,
							false,
							nil,
							node.NewStringValueDiff("value_two", action.Delete, false, false),
							nil,
						),
						false,
						7,
					),
				}),
				false,
			),
		},
		"primitive_fake_json_string_update": {
			diff: computed.Diff{
				// This isn't valid JSON, our renderer should be okay with it.
				Renderer: Primitive("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"", "{\"key_one\": \"value_one\",\"key_two\":\"value_two\",\"key_three\":\"value_three\"", cty.String),
				Action:   action.Update,
			},
			expected: node.NewPrimitiveDiff(
				action.Update,
				false,
				nil,
				node.NewStringValueDiff("{\"key_one\": \"value_one\",\"key_two\":\"value_two\"", action.Update, false, false),
				node.NewStringValueDiff("{\"key_one\": \"value_one\",\"key_two\":\"value_two\",\"key_three\":\"value_three\"", action.Update, false, false),
			),
		},
		"primitive_json_to_string_update": {
			diff: computed.Diff{
				Renderer: Primitive("{\"key_one\": \"value_one\"}", "hello world", cty.String),
				Action:   action.Update,
			},
			expected: node.NewTypeChangeDiff(
				action.Update,
				false,
				nil,
				node.NewJSONStringDiff(
					action.Delete,
					false,
					nil,
					node.NewJSONObjectDiff(action.Delete, false, nil, []*node.KeyValueDiff{
						node.NewKeyValueDiff(
							action.Delete,
							nil,
							"key_one",
							node.NewPrimitiveDiff(
								action.Delete,
								false,
								nil,
								node.NewStringValueDiff("value_one", action.Delete, false, false),
								nil,
							),
							false,
							7,
						),
					}),
					false,
				),
				node.NewStringValueDiff("hello world", action.Create, false, false),
			),
		},
		"sensitive_update": {
			diff: computed.Diff{
				Renderer: Sensitive(computed.Diff{
					Renderer: Primitive(0.0, 1.0, cty.Number),
					Action:   action.Update,
				}, true, true),
				Action: action.Update,
			},
			expected: node.NewSensitiveDiff(
				action.Update,
				false,
				[]string{},
				true,
				true,
			),
		},
		"sensitive_update_replace": {
			diff: computed.Diff{
				Renderer: Sensitive(computed.Diff{
					Renderer: Primitive(0.0, 1.0, cty.Number),
					Action:   action.Update,
					Replace:  true,
				}, true, true),
				Action:  action.Update,
				Replace: true,
			},
			expected: node.NewSensitiveDiff(
				action.Update,
				true,
				[]string{},
				true,
				true,
			),
		},
		"computed_create": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{}),
				Action:   action.Create,
			},
			expected: node.NewUnknownDiff(action.Create, false, nil, nil),
		},
		"computed_update": {
			diff: computed.Diff{
				Renderer: Unknown(computed.Diff{
					Renderer: Primitive(0.0, nil, cty.Number),
					Action:   action.Delete,
				}),
				Action: action.Update,
			},
			expected: node.NewUnknownDiff(
				action.Update,
				false,
				nil,
				node.NewNumberValueDiff(0.0, action.Delete, false),
			),
		},
		"object_created": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{}),
				Action:   action.Create,
			},
			expected: node.NewJSONObjectDiff(action.Create, false, nil, []*node.KeyValueDiff{}),
		},
		"object_created_with_attributes": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(nil, 0.0, cty.Number),
						Action:   action.Create,
					},
				}),
				Action: action.Create,
			},
			expected: node.NewJSONObjectDiff(action.Create, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Create,
					nil,
					"attribute_one",
					node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNumberValueDiff(0.0, action.Create, false)),
					false,
					13,
				),
			}),
		},
		"object_deleted": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{}),
				Action:   action.Delete,
			},
			expected: node.NewJSONObjectDiff(action.Delete, false, nil, []*node.KeyValueDiff{}),
		},
		"nested_object_deleted": {
			diff: computed.Diff{
				Renderer: NestedObject(map[string]computed.Diff{}),
				Action:   action.Delete,
			},
			expected: node.NewJSONObjectDiff(action.Delete, false, nil, []*node.KeyValueDiff{}),
		},
		"object_create_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(nil, 0.0, cty.Number),
						Action:   action.Create,
					},
				}),
				Action: action.Update,
			},
			expected: node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Create,
					nil,
					"attribute_one",
					node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNumberValueDiff(0.0, action.Create, false)),
					false,
					13,
				),
			}),
		},
		"object_update_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Primitive(0.0, 1.0, cty.Number),
						Action:   action.Update,
					},
				}),
				Action: action.Update,
			},
			expected: node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Update,
					nil,
					"attribute_one",
					node.NewPrimitiveDiff(action.Update, false, nil, node.NewNumberValueDiff(0.0, action.Update, false), node.NewNumberValueDiff(1.0, action.Update, false)),
					false,
					13,
				),
			}),
		},
		"object_create_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   action.Create,
						}, false, true),
						Action: action.Create,
					},
				}),
				Action: action.Update,
			},
			expected: node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Create,
					nil,
					"attribute_one",
					node.NewSensitiveDiff(action.Create, false, []string{}, false, true),
					false,
					13,
				),
			}),
		},
		"object_update_sensitive_attribute": {
			diff: computed.Diff{
				Renderer: Object(map[string]computed.Diff{
					"attribute_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 1.0, cty.Number),
							Action:   action.Update,
						}, true, true),
						Action: action.Update,
					},
				}),
				Action: action.Update,
			},
			expected: node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Update,
					nil,
					"attribute_one",
					node.NewSensitiveDiff(action.Update, false, []string{}, true, true),
					false,
					13,
				),
			}),
		},
		"map_create": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Primitive(nil, "new", cty.String),
						Action:   action.Create,
					},
				}),
				Action: action.Create,
			},
			expected: node.NewJSONObjectDiff(action.Create, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Create,
					nil,
					"element_one",
					node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewStringValueDiff("new", action.Create, false, false)),
					false,
					11,
				),
			}),
		},
		"map_update_sensitive_element_status": {
			diff: computed.Diff{
				Renderer: Map(map[string]computed.Diff{
					"element_one": {
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(0.0, 0.0, cty.Number),
							Action:   action.NoOp,
						}, true, false),
						Action: action.Update,
					},
				}),
				Action: action.Update,
			},
			expected: node.NewJSONObjectDiff(action.Update, false, nil, []*node.KeyValueDiff{
				node.NewKeyValueDiff(
					action.Update,
					nil,
					"element_one",
					node.NewSensitiveDiff(action.Update, false, []string{"This attribute value will no longer be marked as sensitive after applying this change (the value is unchanged)"}, true, false),
					false,
					11,
				),
			}),
		},
		"list_create": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   action.Create,
					},
				}),
				Action: action.Create,
			},
			expected: node.NewJSONArray(
				action.Create,
				false, nil,
				[]node.Diff{
					node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNumberValueDiff(1.0, action.Create, false)),
				},
			),
		},
		"list_create_sensitive_element": {
			diff: computed.Diff{
				Renderer: List([]computed.Diff{
					{
						Renderer: Sensitive(computed.Diff{
							Renderer: Primitive(nil, 1.0, cty.Number),
							Action:   action.Create,
						}, false, true),
						Action: action.Create,
					},
				}),
				Action: action.Update,
			},
			expected: node.NewJSONArray(
				action.Update,
				false,
				nil,
				[]node.Diff{
					node.NewSensitiveDiff(action.Create, false, []string{}, false, true),
				},
			),
		},
		"set_create": {
			diff: computed.Diff{
				Renderer: Set([]computed.Diff{
					{
						Renderer: Primitive(nil, 1.0, cty.Number),
						Action:   action.Create,
					},
				}),
				Action: action.Create,
			},
			expected: node.NewJSONArray(
				action.Create,
				false, nil,
				[]node.Diff{
					node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNumberValueDiff(1.0, action.Create, false)),
				},
			),
		},
		"create_empty_block": {
			diff: computed.Diff{
				Renderer: Block(nil, Blocks{}),
				Action:   action.Create,
			},
			expected: node.NewBlockDiff(
				action.Create,
				false,
				nil,
				[]*node.KeyValueDiff{},
				[]*node.NestedBlockDiff{},
			),
		},
		"create_populated_block": {
			diff: computed.Diff{
				Renderer: Block(map[string]computed.Diff{
					"boolean": {
						Renderer: Primitive(nil, true, cty.Bool),
						Action:   action.Create,
					},
				}, Blocks{
					SingleBlocks: map[string]computed.Diff{
						"nested_block": {
							Renderer: Block(map[string]computed.Diff{
								"string": {
									Renderer: Primitive(nil, "one", cty.String),
									Action:   action.Create,
								},
							}, Blocks{}),
							Action: action.Create,
						},
					},
				}),
				Action: action.Create,
			},
			expected: node.NewBlockDiff(
				action.Create,
				false,
				nil,
				[]*node.KeyValueDiff{
					node.NewKeyValueDiff(
						action.Create,
						nil,
						"boolean",
						node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewBoolValueDiff(true, action.Create, false)),
						true,
						7,
					),
				},
				[]*node.NestedBlockDiff{
					node.NewNestedBlockDiff(
						action.Create,
						false,
						nil,
						"nested_block",
						"",
						node.NewBlockDiff(
							action.Create,
							false,
							nil,
							[]*node.KeyValueDiff{
								node.NewKeyValueDiff(
									action.Create,
									nil,
									"string",
									node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewStringValueDiff("one", action.Create, false, false)),
									true,
									6,
								),
							},
							[]*node.NestedBlockDiff{},
						),
					),
				},
			),
		},
		"map_block_update": {
			diff: computed.Diff{
				Action: action.Update,
				Renderer: Block(
					nil,
					Blocks{
						MapBlocks: map[string]map[string]computed.Diff{
							"map_blocks": {
								"key_one": {
									Renderer: Block(map[string]computed.Diff{
										"number": {
											Renderer: Primitive(1.0, 2.0, cty.Number),
											Action:   action.Update,
										},
									}, Blocks{}),
									Action: action.Update,
								},
							},
						},
					}),
			},
			expected: node.NewBlockDiff(
				action.Update,
				false,
				nil,
				[]*node.KeyValueDiff{},
				[]*node.NestedBlockDiff{
					node.NewNestedBlockDiff(
						action.Update,
						false,
						nil,
						"map_blocks",
						"key_one",
						node.NewBlockDiff(
							action.Update,
							false,
							nil,
							[]*node.KeyValueDiff{
								node.NewKeyValueDiff(
									action.Update,
									nil,
									"number",
									node.NewPrimitiveDiff(action.Update, false, nil, node.NewNumberValueDiff(1.0, action.Update, false), node.NewNumberValueDiff(2.0, action.Update, false)),
									true,
									6,
								),
							},
							[]*node.NestedBlockDiff{},
						),
					),
				},
			),
		},
		"sensitive_block": {
			diff: computed.Diff{
				Renderer: SensitiveBlock(computed.Diff{
					Renderer: Block(nil, Blocks{}),
					Action:   action.NoOp,
				}, true, true),
				Action: action.Update,
			},
			expected: node.NewSensitiveBlockDiff(action.Update, false, []string{}, true, true),
		},
		"output_map_to_list": {
			diff: computed.Diff{
				Action: action.Update,
				Renderer: TypeChange(computed.Diff{
					Renderer: Map(map[string]computed.Diff{
						"element_one": {
							Renderer: Primitive(0.0, nil, cty.Number),
							Action:   action.Delete,
						},
					}),
					Action: action.Delete,
				}, computed.Diff{
					Renderer: List([]computed.Diff{
						{
							Renderer: Primitive(nil, 0.0, cty.Number),
							Action:   action.Create,
						},
					}),
					Action: action.Create,
				}),
			},
			expected: node.NewTypeChangeDiff(
				action.Update,
				false,
				nil,
				node.NewJSONObjectDiff(action.Delete, false, nil, []*node.KeyValueDiff{
					node.NewKeyValueDiff(
						action.Delete,
						nil,
						"element_one",
						node.NewPrimitiveDiff(action.Delete, false, nil, node.NewNumberValueDiff(0.0, action.Delete, false), nil),
						false,
						11),
				}),
				node.NewJSONArray(action.Create, false, nil, []node.Diff{
					node.NewPrimitiveDiff(action.Create, false, nil, nil, node.NewNumberValueDiff(0.0, action.Create, false)),
				}),
			),
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			expectedBytes, err := json.MarshalIndent(tc.expected, "", "	")
			if err != nil {
				t.Fatalf("failed to marshal expected output: %v", err)
			}

			actual, err := tc.diff.Render()
			require.NoError(t, err)

			actualBytes, err := json.MarshalIndent(actual, "", "	")
			if err != nil {
				t.Fatalf("failed to marshal actual output: %v", err)
			}

			if diff := cmp.Diff(string(expectedBytes), string(actualBytes)); len(diff) > 0 {
				t.Fatalf("\nexpected:\n%s\nactual:\n%s\ndiff:\n%s\n", string(expectedBytes), string(actualBytes), diff)
			}
		})
	}
}
