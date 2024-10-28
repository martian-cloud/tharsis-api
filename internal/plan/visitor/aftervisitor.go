package visitor

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

// AfterVisitor is a visitor for rendering the after state of a diff
type AfterVisitor struct {
	common
}

// NewAfterVisitor creates a new AfterVisitor
func NewAfterVisitor(initialIndent int) *AfterVisitor {
	return &AfterVisitor{
		common: common{
			indentLevel: initialIndent,
		},
	}
}

// VisitBlockDiff renders a block diff
func (v *AfterVisitor) VisitBlockDiff(diff *node.BlockDiff) {
	if diff.Action != action.Delete {
		if len(diff.Attributes) == 0 && len(diff.Blocks) == 0 {
			v.renderEmptyObject(diff.Replace)
			return
		}

		v.builder.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace)))

		for _, attr := range diff.Attributes {
			if attr.Action == action.Delete {
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
			if block.Action == action.Delete {
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
func (v *AfterVisitor) VisitNestedBlockDiff(diff *node.NestedBlockDiff) {
	if diff.Action != action.Delete {
		v.renderNestedBlockDiff(v, diff)
	}
}

// VisitJSONObjectDiff renders a JSON diff
func (v *AfterVisitor) VisitJSONObjectDiff(diff *node.JSONObjectDiff) {
	if diff.Action != action.Delete {
		attributesToRender := 0
		for _, attr := range diff.Attributes {
			if attr.Action != action.Delete {
				attributesToRender++
			}
		}

		if attributesToRender == 0 {
			v.renderEmptyObject(diff.Replace)
			return
		}

		v.builder.WriteString(fmt.Sprintf("{%s\n", forcesReplacement(diff.Replace)))

		for _, attr := range diff.Attributes {
			if attr.Action == action.Delete {
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
func (v *AfterVisitor) VisitJSONArray(diff *node.JSONArray) {
	if diff.Action != action.Delete {
		if len(diff.Elements) == 0 {
			v.renderEmptyList(diff.Replace)
			return
		}

		v.builder.WriteString(fmt.Sprintf("[%s\n", forcesReplacement(diff.Replace)))

		for _, attr := range diff.Elements {
			if attr.GetAction() == action.Delete {
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
func (v *AfterVisitor) VisitKeyValueDiff(diff *node.KeyValueDiff) {
	if diff.Action != action.Delete {
		v.renderKeyValueDiff(v, diff)
	}
}

// VisitUnknownDiff renders an unknown diff
func (v *AfterVisitor) VisitUnknownDiff(diff *node.UnknownDiff) {
	v.builder.WriteString("(known after apply)")
	v.builder.WriteString(forcesReplacement(diff.Replace))
}

// VisitTypeChangeDiff renders a type change diff
func (v *AfterVisitor) VisitTypeChangeDiff(diff *node.TypeChangeDiff) {
	if diff.After != nil {
		diff.After.Accept(v)
	}
}

// VisitPrimitiveDiff renders a primitive diff
func (v *AfterVisitor) VisitPrimitiveDiff(diff *node.PrimitiveDiff) {
	switch diff.Action {
	case action.NoOp:
		diff.After.Accept(v)
	default:
		if diff.After != nil {
			diff.After.Accept(v)
		}
	}
}

// VisitJSONStringDiff renders a JSON string diff
func (v *AfterVisitor) VisitJSONStringDiff(diff *node.JSONStringDiff) {
	if diff.Action != action.Delete {
		v.renderJSONStringDiff(v, diff)
	}
}

// VisitSensitiveDiff renders a sensitive diff
func (v *AfterVisitor) VisitSensitiveDiff(diff *node.SensitiveDiff) {
	text := "sensitive value"
	if !diff.AfterSensitive {
		text = "value"
	}

	if diff.Action == action.Update {
		v.builder.WriteString(fmt.Sprintf("(new %s)", text))
	} else {
		v.builder.WriteString(fmt.Sprintf("(%s)", text))
	}

	v.builder.WriteString(forcesReplacement(diff.Replace))

	// Dont add warnings for deleted actions since they will be added in the before visitor
	if diff.Action != action.Delete {
		v.addWarnings(diff.Warnings)
	}
}

// VisitStringValueDiff renders a string value
func (v *AfterVisitor) VisitStringValueDiff(diff *node.StringValueDiff) {
	v.renderStringValueDiff(diff)
}

// VisitNumberValueDiff renders a number value
func (v *AfterVisitor) VisitNumberValueDiff(diff *node.NumberValueDiff) {
	v.renderNumberValueDiff(diff)
}

// VisitBoolValueDiff renders a bool value
func (v *AfterVisitor) VisitBoolValueDiff(diff *node.BoolValueDiff) {
	v.renderBoolValueDiff(diff)
}

// VisitNull renders a null value
func (v *AfterVisitor) VisitNull(diff *node.NullValueDiff) {
	v.renderNull(diff)
}
