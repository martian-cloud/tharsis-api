// Package visitor provides a visitor pattern for traversing the plan and rendering the hcl diff
package visitor

import (
	"fmt"
	"math/big"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

// Warning is a warning message that occurred during the plan
type Warning struct {
	Line    int
	Message string
}

func forcesReplacement(replace bool) string {
	if replace {
		return " # forces replacement"
	}
	return ""
}

type common struct {
	builder     strings.Builder
	indentLevel int
	warnings    []Warning
}

// String returns the rendered string
func (c *common) String() string {
	return c.builder.String()
}

// Warnings returns the warnings
func (c *common) Warnings() []Warning {
	return c.warnings
}

func (c *common) addWarnings(warnings []string) {
	// Get the current line number
	lines := strings.Split(c.builder.String(), "\n")
	lineNumber := len(lines)
	// Add warnings with line number
	for _, warning := range warnings {
		c.warnings = append(c.warnings, Warning{Line: lineNumber, Message: warning})
	}
}

func (c *common) incIndent() {
	c.indentLevel++
}

func (c *common) decIndent() {
	if c.indentLevel > 0 {
		c.indentLevel--
	}
}

func (c *common) indent() {
	if c.indentLevel == 0 {
		return
	}
	c.builder.WriteString(strings.Repeat("    ", c.indentLevel))
}

func (c *common) renderEmptyObject(replace bool) {
	c.builder.WriteString(fmt.Sprintf("{}%s", forcesReplacement(replace)))
}

func (c *common) renderEmptyList(replace bool) {
	c.builder.WriteString(fmt.Sprintf("[]%s", forcesReplacement(replace)))
}

func (c *common) renderNestedBlockDiff(v node.Visitor, diff *node.NestedBlockDiff) {
	c.builder.WriteString(fmt.Sprintf("%s ", diff.Name))
	if diff.Label != "" {
		c.builder.WriteString(fmt.Sprintf("%q ", diff.Label))
	}

	diff.Block.Accept(v)
}

func (c *common) renderKeyValueDiff(v node.Visitor, diff *node.KeyValueDiff) {
	if diff.BlockAttribute {
		c.builder.WriteString(fmt.Sprintf("%-*s = ", diff.MaxKeyLength, diff.Key))
	} else {
		c.builder.WriteString(fmt.Sprintf("%-*s = ", diff.MaxKeyLength+2, fmt.Sprintf("%q", diff.Key)))
	}

	diff.Value.Accept(v)
}

func (c *common) renderJSONStringDiff(v node.Visitor, diff *node.JSONStringDiff) {
	c.builder.WriteString("jsonencode(")

	diff.JSONValue.Accept(v)

	c.builder.WriteString(")")

	c.builder.WriteString(forcesReplacement(diff.Replace))
}

func (c *common) renderStringValueDiff(diff *node.StringValueDiff) {
	if diff.Multiline {
		lines := strings.Split(diff.Value, "\n")

		c.builder.WriteString("<<-EOT\n")

		c.indentLevel++
		for _, line := range lines {
			c.indent()
			c.builder.WriteString(fmt.Sprintf("%s\n", line))
		}
		c.indentLevel--

		c.indent()

		c.builder.WriteString("EOT")
	} else {
		c.builder.WriteString(fmt.Sprintf("%q", diff.Value))
	}

	c.builder.WriteString(forcesReplacement(diff.Replace))
}

func (c *common) renderNumberValueDiff(diff *node.NumberValueDiff) {
	bf := big.NewFloat(diff.Value)
	c.builder.WriteString(bf.Text('f', -1))
	c.builder.WriteString(forcesReplacement(diff.Replace))
}

func (c *common) renderBoolValueDiff(diff *node.BoolValueDiff) {
	if diff.Value {
		c.builder.WriteString("true")
	} else {
		c.builder.WriteString("false")
	}
	c.builder.WriteString(forcesReplacement(diff.Replace))
}

func (c *common) renderNull(diff *node.NullValueDiff) {
	c.builder.WriteString("null")
	c.builder.WriteString(forcesReplacement(diff.Replace))
}
