// Package node provides the rendered node types
package node

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
)

// DiffType is the type of the diff
type DiffType string

// DiffType constants
const (
	DiffTypeBlock       DiffType = "block"
	DiffTypeNestedBlock DiffType = "nested_block"
	DiffTypeJSONObject  DiffType = "json_object"
	DiffTypeJSONArray   DiffType = "json_array"
	DiffTypeKeyValue    DiffType = "key_value"
	DiffTypeUnknown     DiffType = "unknown_value"
	DiffTypeTypeChange  DiffType = "type_change"
	DiffTypePrimitive   DiffType = "primitive"
	DiffTypeJSONString  DiffType = "json_string"
	DiffTypeSensitive   DiffType = "sensitive"
	DiffTypeString      DiffType = "string"
	DiffTypeNumber      DiffType = "number"
	DiffTypeBool        DiffType = "bool"
	DiffTypeNull        DiffType = "null"
)

// Visitor is the interface for visiting diffs
type Visitor interface {
	VisitBlockDiff(diff *BlockDiff)
	VisitNestedBlockDiff(diff *NestedBlockDiff)
	VisitJSONObjectDiff(diff *JSONObjectDiff)
	VisitJSONArray(diff *JSONArray)
	VisitKeyValueDiff(diff *KeyValueDiff)
	VisitUnknownDiff(diff *UnknownDiff)
	VisitTypeChangeDiff(diff *TypeChangeDiff)
	VisitPrimitiveDiff(diff *PrimitiveDiff)
	VisitJSONStringDiff(diff *JSONStringDiff)
	VisitSensitiveDiff(diff *SensitiveDiff)
	VisitStringValueDiff(diff *StringValueDiff)
	VisitNumberValueDiff(diff *NumberValueDiff)
	VisitBoolValueDiff(diff *BoolValueDiff)
	VisitNull(diff *NullValueDiff)
}

// Diff is the interface for all diff types
type Diff interface {
	// GetType returns the type of the diff
	GetType() DiffType
	// GetWarnings returns the warnings for the diff
	GetWarnings() []string
	// GetAction returns the action for the diff
	GetAction() action.Action
	// Accept accepts a visitor
	Accept(visitor Visitor)
}

type diff struct {
	Type     DiffType
	Warnings []string
	Action   action.Action
	Replace  bool
}

func (d diff) GetType() DiffType {
	return d.Type
}

func (d diff) GetAction() action.Action {
	return d.Action
}

func (d diff) GetWarnings() []string {
	return d.Warnings
}

func newDiff(modelType DiffType, action action.Action, replace bool, warnings []string) diff {
	return diff{
		Type:     modelType,
		Warnings: warnings,
		Action:   action,
		Replace:  replace,
	}
}

// BlockDiff represents a block diff
type BlockDiff struct {
	diff
	Attributes []*KeyValueDiff
	Blocks     []*NestedBlockDiff
}

// NewBlockDiff creates a new block diff
func NewBlockDiff(
	action action.Action,
	replace bool,
	warnings []string,
	attributes []*KeyValueDiff,
	blocks []*NestedBlockDiff,
) *BlockDiff {
	return &BlockDiff{
		diff:       newDiff(DiffTypeBlock, action, replace, warnings),
		Attributes: attributes,
		Blocks:     blocks,
	}
}

// Accept accepts a visitor
func (d *BlockDiff) Accept(visitor Visitor) {
	visitor.VisitBlockDiff(d)
}

// NestedBlockDiff represents a nested block diff
type NestedBlockDiff struct {
	diff
	Name  string
	Label string
	Block Diff
}

// NewNestedBlockDiff creates a new nested block diff
func NewNestedBlockDiff(action action.Action, replace bool, warnings []string, name string, label string, block Diff) *NestedBlockDiff {
	return &NestedBlockDiff{
		diff:  newDiff(DiffTypeNestedBlock, action, replace, warnings),
		Name:  name,
		Label: label,
		Block: block,
	}
}

// Accept accepts a visitor
func (d *NestedBlockDiff) Accept(visitor Visitor) {
	visitor.VisitNestedBlockDiff(d)
}

// JSONObjectDiff represents a JSON diff
type JSONObjectDiff struct {
	diff
	Attributes []*KeyValueDiff
}

// NewJSONObjectDiff creates a new JSON diff
func NewJSONObjectDiff(action action.Action, replace bool, warnings []string, Attributes []*KeyValueDiff) *JSONObjectDiff {
	return &JSONObjectDiff{
		diff:       newDiff(DiffTypeJSONObject, action, replace, warnings),
		Attributes: Attributes,
	}
}

// Accept accepts a visitor
func (d *JSONObjectDiff) Accept(visitor Visitor) {
	visitor.VisitJSONObjectDiff(d)
}

// JSONArray represents a JSON array diff
type JSONArray struct {
	diff
	Elements []Diff
}

// NewJSONArray creates a new JSON array diff
func NewJSONArray(action action.Action, replace bool, warnings []string, elements []Diff) *JSONArray {
	return &JSONArray{
		diff:     newDiff(DiffTypeJSONArray, action, replace, warnings),
		Elements: elements,
	}
}

// Accept accepts a visitor
func (d *JSONArray) Accept(visitor Visitor) {
	visitor.VisitJSONArray(d)
}

// KeyValueDiff represents a key value diff
type KeyValueDiff struct {
	diff
	Key            string
	Value          Diff
	BlockAttribute bool
	MaxKeyLength   int
}

// NewKeyValueDiff creates a new key value diff
func NewKeyValueDiff(action action.Action, warnings []string, key string, value Diff, blockAttribute bool, maxKeyLength int) *KeyValueDiff {
	return &KeyValueDiff{
		diff:           newDiff(DiffTypeKeyValue, action, false, warnings),
		Key:            key,
		Value:          value,
		BlockAttribute: blockAttribute,
		MaxKeyLength:   maxKeyLength,
	}
}

// Accept accepts a visitor
func (d *KeyValueDiff) Accept(visitor Visitor) {
	visitor.VisitKeyValueDiff(d)
}

// SensitiveDiff represents a sensitive diff
type SensitiveDiff struct {
	diff
	Block           bool
	BeforeSensitive bool
	AfterSensitive  bool
}

// NewSensitiveDiff creates a new sensitive diff
func NewSensitiveDiff(action action.Action, replace bool, warnings []string, beforeSensitive bool, afterSensitive bool) *SensitiveDiff {
	return &SensitiveDiff{
		diff:            newDiff(DiffTypeSensitive, action, replace, warnings),
		BeforeSensitive: beforeSensitive,
		AfterSensitive:  afterSensitive,
	}
}

// Accept accepts a visitor
func (d *SensitiveDiff) Accept(visitor Visitor) {
	visitor.VisitSensitiveDiff(d)
}

// NewSensitiveBlockDiff creates a new sensitive block diff
func NewSensitiveBlockDiff(action action.Action, replace bool, warnings []string, beforeSensitive bool, afterSensitive bool) *SensitiveDiff {
	return &SensitiveDiff{
		diff:            newDiff(DiffTypeSensitive, action, replace, warnings),
		Block:           true,
		BeforeSensitive: beforeSensitive,
		AfterSensitive:  afterSensitive,
	}
}

// UnknownDiff represents an unknown diff
type UnknownDiff struct {
	diff
	Before Diff
}

// NewUnknownDiff creates a new unknown diff
func NewUnknownDiff(action action.Action, replace bool, warnings []string, before Diff) *UnknownDiff {
	return &UnknownDiff{
		diff:   newDiff(DiffTypeUnknown, action, replace, warnings),
		Before: before,
	}
}

// Accept accepts a visitor
func (d *UnknownDiff) Accept(visitor Visitor) {
	visitor.VisitUnknownDiff(d)
}

// TypeChangeDiff represents a type change diff
type TypeChangeDiff struct {
	diff
	Before Diff
	After  Diff
}

// NewTypeChangeDiff creates a new type change diff
func NewTypeChangeDiff(action action.Action, replace bool, warnings []string, before Diff, after Diff) *TypeChangeDiff {
	return &TypeChangeDiff{
		diff:   newDiff(DiffTypeTypeChange, action, replace, warnings),
		Before: before,
		After:  after,
	}
}

// Accept accepts a visitor
func (d *TypeChangeDiff) Accept(visitor Visitor) {
	visitor.VisitTypeChangeDiff(d)
}

// PrimitiveDiff represents a primitive diff
type PrimitiveDiff struct {
	diff
	Before Diff
	After  Diff
}

// NewPrimitiveDiff creates a new primitive diff
func NewPrimitiveDiff(action action.Action, replace bool, warnings []string, before Diff, after Diff) *PrimitiveDiff {
	return &PrimitiveDiff{
		diff:   newDiff(DiffTypePrimitive, action, replace, warnings),
		Before: before,
		After:  after,
	}
}

// Accept accepts a visitor
func (d *PrimitiveDiff) Accept(visitor Visitor) {
	visitor.VisitPrimitiveDiff(d)
}

// JSONStringDiff represents a JSON string diff
type JSONStringDiff struct {
	diff
	WhitespaceOnlyChange bool
	JSONValue            Diff
}

// NewJSONStringDiff creates a new JSON string diff
func NewJSONStringDiff(action action.Action, replace bool, warnings []string, value Diff, whitespaceOnlyChange bool) *JSONStringDiff {
	return &JSONStringDiff{
		diff:                 newDiff(DiffTypeJSONString, action, replace, warnings),
		WhitespaceOnlyChange: whitespaceOnlyChange,
		JSONValue:            value,
	}
}

// Accept accepts a visitor
func (d *JSONStringDiff) Accept(visitor Visitor) {
	visitor.VisitJSONStringDiff(d)
}

// StringValueDiff represents a string value
type StringValueDiff struct {
	diff
	Value     string
	Multiline bool
}

// NewStringValueDiff creates a new string value
func NewStringValueDiff(value string, action action.Action, replace bool, multiline bool) *StringValueDiff {
	return &StringValueDiff{
		diff:      newDiff(DiffTypeString, action, replace, nil),
		Value:     value,
		Multiline: multiline,
	}
}

// Accept accepts a visitor
func (d *StringValueDiff) Accept(visitor Visitor) {
	visitor.VisitStringValueDiff(d)
}

// NumberValueDiff represents a number value
type NumberValueDiff struct {
	diff
	Value float64
}

// NewNumberValueDiff creates a new number value
func NewNumberValueDiff(value float64, action action.Action, replace bool) *NumberValueDiff {
	return &NumberValueDiff{
		diff:  newDiff(DiffTypeNumber, action, replace, nil),
		Value: value,
	}
}

// Accept accepts a visitor
func (d *NumberValueDiff) Accept(visitor Visitor) {
	visitor.VisitNumberValueDiff(d)
}

// BoolValueDiff represents a bool value
type BoolValueDiff struct {
	diff
	Value bool
}

// NewBoolValueDiff creates a new bool value
func NewBoolValueDiff(value bool, action action.Action, replace bool) *BoolValueDiff {
	return &BoolValueDiff{
		diff:  newDiff(DiffTypeBool, action, replace, nil),
		Value: value,
	}
}

// Accept accepts a visitor
func (d *BoolValueDiff) Accept(visitor Visitor) {
	visitor.VisitBoolValueDiff(d)
}

// NullValueDiff represents a null value
type NullValueDiff struct {
	diff
}

// NewNullValueDiff creates a new null value
func NewNullValueDiff(action action.Action, replace bool) *NullValueDiff {
	return &NullValueDiff{
		diff: newDiff(DiffTypeNull, action, replace, nil),
	}
}

// Accept accepts a visitor
func (d *NullValueDiff) Accept(visitor Visitor) {
	visitor.VisitNull(d)
}
