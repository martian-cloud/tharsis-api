package visitor

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

// BeforeVisitor is a visitor for rendering the before state of a diff
type BeforeVisitor struct {
	common
}

// NewBeforeVisitor creates a new BeforeVisitor
func NewBeforeVisitor(initialIndent int) *BeforeVisitor {
	return &BeforeVisitor{
		common: common{
			indentLevel: initialIndent,
		},
	}
}

// VisitBlockDiff renders a block diff
func (v *BeforeVisitor) VisitBlockDiff(diff *node.BlockDiff) {
	if diff.Action != action.Create {
		if len(diff.Attributes) == 0 && len(diff.Blocks) == 0 {
			v.renderEmptyObject(diff.Replace)
			return
		}

		v.builder.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace)))

		for _, attr := range diff.Attributes {
			if attr.Action == action.Create {
				continue
			}

			v.incIndent()

			v.indent()
			attr.Accept(v)
			v.decIndent()

			v.builder.WriteString("\n")
		}

		if len(diff.Attributes) > 0 && len(diff.Blocks) > 0 {
			v.builder.WriteString("\n")
		}

		for i, block := range diff.Blocks {
			if block.Action == action.Create {
				continue
			}

			v.incIndent()

			v.indent()
			block.Accept(v)
			v.decIndent()

			v.builder.WriteString("\n")

			if i < len(diff.Blocks)-1 {
				v.builder.WriteString("\n")
			}
		}

		v.indent()
		v.builder.WriteString("}")
	}
}

// VisitNestedBlockDiff renders a nested block diff
func (v *BeforeVisitor) VisitNestedBlockDiff(diff *node.NestedBlockDiff) {
	if diff.Action != action.Create {
		v.renderNestedBlockDiff(v, diff)
	}
}

// VisitJSONObjectDiff renders a JSON diff
func (v *BeforeVisitor) VisitJSONObjectDiff(diff *node.JSONObjectDiff) {
	if diff.Action != action.Create {
		attributesToRender := 0
		for _, attr := range diff.Attributes {
			if attr.Action != action.Create {
				attributesToRender++
			}
		}

		if attributesToRender == 0 {
			v.renderEmptyObject(diff.Replace)
			return
		}

		v.builder.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace)))

		for _, attr := range diff.Attributes {
			if attr.Action == action.Create {
				continue
			}
			v.incIndent()

			v.indent()
			attr.Accept(v)
			v.decIndent()

			v.builder.WriteString("\n")
		}

		v.indent()
		v.builder.WriteString("}")
	}
}

// VisitJSONArray renders a JSON array
func (v *BeforeVisitor) VisitJSONArray(diff *node.JSONArray) {
	if diff.Action != action.Create {
		if len(diff.Elements) == 0 {
			v.renderEmptyList(diff.Replace)
			return
		}

		v.builder.WriteString(fmt.Sprintf("[%s\n", forcesReplacement(diff.Replace)))

		for _, attr := range diff.Elements {
			if attr.GetAction() == action.Create {
				continue
			}

			v.incIndent()

			v.indent()
			attr.Accept(v)
			v.decIndent()

			v.builder.WriteString(",\n")
		}

		v.indent()
		v.builder.WriteString("]")
	}
}

// VisitKeyValueDiff renders a key value diff
func (v *BeforeVisitor) VisitKeyValueDiff(diff *node.KeyValueDiff) {
	if diff.Action != action.Create {
		v.renderKeyValueDiff(v, diff)
	}
}

// VisitUnknownDiff renders an unknown diff
func (v *BeforeVisitor) VisitUnknownDiff(diff *node.UnknownDiff) {
	if diff.Before != nil {
		diff.Before.Accept(v)
	}
}

// VisitTypeChangeDiff renders a type change diff
func (v *BeforeVisitor) VisitTypeChangeDiff(diff *node.TypeChangeDiff) {
	if diff.Before != nil {
		diff.Before.Accept(v)
	}
}

// VisitPrimitiveDiff renders a primitive diff
func (v *BeforeVisitor) VisitPrimitiveDiff(diff *node.PrimitiveDiff) {
	switch diff.Action {
	case action.NoOp:
		diff.After.Accept(v)
	default:
		if diff.Before != nil {
			diff.Before.Accept(v)
		}
	}
}

// VisitJSONStringDiff renders a JSON string diff
func (v *BeforeVisitor) VisitJSONStringDiff(diff *node.JSONStringDiff) {
	if diff.Action != action.Create {
		v.renderJSONStringDiff(v, diff)
	}
}

// VisitSensitiveDiff renders a sensitive diff
func (v *BeforeVisitor) VisitSensitiveDiff(diff *node.SensitiveDiff) {
	text := "sensitive value"
	if !diff.BeforeSensitive {
		text = "value"
	}

	if diff.Action == action.Update {
		v.builder.WriteString(fmt.Sprintf("(old %s)", text))
	} else {
		v.builder.WriteString(fmt.Sprintf("(%s)", text))
	}

	v.builder.WriteString(forcesReplacement(diff.Replace))

	// Only add warnings for deleted actions since noop warnings will be added in the after visitor
	if diff.Action == action.Delete {
		v.addWarnings(diff.Warnings)
	}
}

// VisitStringValueDiff renders a string value
func (v *BeforeVisitor) VisitStringValueDiff(diff *node.StringValueDiff) {
	v.renderStringValueDiff(diff)
}

// VisitNumberValueDiff renders a number value
func (v *BeforeVisitor) VisitNumberValueDiff(diff *node.NumberValueDiff) {
	v.renderNumberValueDiff(diff)
}

// VisitBoolValueDiff renders a bool value
func (v *BeforeVisitor) VisitBoolValueDiff(diff *node.BoolValueDiff) {
	v.renderBoolValueDiff(diff)
}

// VisitNull renders a null value
func (v *BeforeVisitor) VisitNull(diff *node.NullValueDiff) {
	v.renderNull(diff)
}
