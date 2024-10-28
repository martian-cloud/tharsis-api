// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package structured

import (
	"reflect"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured/attributepath"

	tjson "github.com/hashicorp/terraform-json"
)

// Change contains the unmarshalled generic interface{} types that are output by
// the JSON functions in the various json packages (such as tjson and
// jsonprovider).
//
// A Change can be converted into a computed.Diff, ready for rendering, with the
// ComputeDiffForAttribute, ComputeDiffForOutput, and ComputeDiffForBlock
// functions.
//
// The Before and After fields are actually go-cty values, but we cannot convert
// them directly because of the Terraform Cloud redacted endpoint. The redacted
// endpoint turns sensitive values into strings regardless of their types.
// Because of this, we cannot just do a direct conversion using the ctyjson
// package. We would have to iterate through the schema first, find the
// sensitive values and their mapped types, update the types inside the schema
// to strings, and then go back and do the overall conversion. This isn't
// including any of the more complicated parts around what happens if something
// was sensitive before and isn't sensitive after or vice versa. This would mean
// the type would need to change between the before and after value. It is in
// fact just easier to iterate through the values as generic JSON interfaces.
type Change struct {

	// BeforeExplicit matches AfterExplicit except references the Before value.
	BeforeExplicit bool

	// AfterExplicit refers to whether the After value is explicit or
	// implicit. It is explicit if it has been specified by the user, and
	// implicit if it has been set as a consequence of other changes.
	//
	// For example, explicitly setting a value to null in a list should result
	// in After being null and AfterExplicit being true. In comparison,
	// removing an element from a list should also result in After being null
	// and AfterExplicit being false. Without the explicit information our
	// functions would not be able to tell the difference between these two
	// cases.
	AfterExplicit bool

	// Before contains the value before the proposed change.
	//
	// The type of the value should be informed by the schema and cast
	// appropriately when needed.
	Before interface{}

	// After contains the value after the proposed change.
	//
	// The type of the value should be informed by the schema and cast
	// appropriately when needed.
	After interface{}

	// Unknown describes whether the After value is known or unknown at the time
	// of the plan. In practice, this means the after value should be rendered
	// simply as `(known after apply)`.
	//
	// The concrete value could be a boolean describing whether the entirety of
	// the After value is unknown, or it could be a list or a map depending on
	// the schema describing whether specific elements or attributes within the
	// value are unknown.
	Unknown interface{}

	// BeforeSensitive matches Unknown, but references whether the Before value
	// is sensitive.
	BeforeSensitive interface{}

	// AfterSensitive matches Unknown, but references whether the After value is
	// sensitive.
	AfterSensitive interface{}

	// ReplacePaths contains a set of paths that point to attributes/elements
	// that are causing the overall resource to be replaced rather than simply
	// updated.
	ReplacePaths attributepath.Matcher

	// RelevantAttributes contains a set of paths that point attributes/elements
	// that we should display. Any element/attribute not matched by this Matcher
	// should be skipped.
	RelevantAttributes attributepath.Matcher
}

// FromJSONChange unmarshals the raw []byte values in the tjson.Change
// structs into generic interface{} types that can be reasoned about.
func FromJSONChange(change tjson.Change, relevantAttributes attributepath.Matcher) (Change, error) {
	replacePaths, err := attributepath.Parse(change.ReplacePaths, false)
	if err != nil {
		return Change{}, err
	}
	return Change{
		Before:             change.Before,
		After:              change.After,
		Unknown:            change.AfterUnknown,
		BeforeSensitive:    change.BeforeSensitive,
		AfterSensitive:     change.AfterSensitive,
		ReplacePaths:       replacePaths,
		RelevantAttributes: relevantAttributes,
	}, nil
}

// CalculateAction does a very simple analysis to make the best guess at the
// action this change describes. For complex types such as objects, maps, lists,
// or sets it is likely more efficient to work out the action directly instead
// of relying on this function.
func (change Change) CalculateAction() action.Action {
	if (change.Before == nil && !change.BeforeExplicit) && (change.After != nil || change.AfterExplicit) {
		return action.Create
	}
	if (change.After == nil && !change.AfterExplicit) && (change.Before != nil || change.BeforeExplicit) {
		return action.Delete
	}

	if reflect.DeepEqual(change.Before, change.After) && change.AfterExplicit == change.BeforeExplicit && change.IsAfterSensitive() == change.IsBeforeSensitive() {
		return action.NoOp
	}

	return action.Update
}

// GetDefaultActionForIteration is used to guess what the change could be for
// complex attributes (collections and objects) and blocks.
//
// You can't really tell the difference between a NoOp and an Update just by
// looking at the attribute itself as you need to inspect the children.
//
// This function returns a Delete or a Create action if the before or after
// values were null, and returns a NoOp for all other cases. It should be used
// in conjunction with compareActions to calculate the actual action based on
// the actions of the children.
func (change Change) GetDefaultActionForIteration() action.Action {
	if change.Before == nil && change.After == nil {
		return action.NoOp
	}

	if change.Before == nil {
		return action.Create
	}
	if change.After == nil {
		return action.Delete
	}
	return action.NoOp
}

// AsNoOp returns the current change as if it is a NoOp operation.
//
// Basically it replaces all the after values with the before values.
func (change Change) AsNoOp() Change {
	return Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      change.BeforeExplicit,
		Before:             change.Before,
		After:              change.Before,
		Unknown:            false,
		BeforeSensitive:    change.BeforeSensitive,
		AfterSensitive:     change.BeforeSensitive,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// AsDelete returns the current change as if it is a Delete operation.
//
// Basically it replaces all the after values with nil or false.
func (change Change) AsDelete() Change {
	return Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      false,
		Before:             change.Before,
		After:              nil,
		Unknown:            nil,
		BeforeSensitive:    change.BeforeSensitive,
		AfterSensitive:     nil,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// AsCreate returns the current change as if it is a Create operation.
//
// Basically it replaces all the before values with nil or false.
func (change Change) AsCreate() Change {
	return Change{
		BeforeExplicit:     false,
		AfterExplicit:      change.AfterExplicit,
		Before:             nil,
		After:              change.After,
		Unknown:            change.Unknown,
		BeforeSensitive:    nil,
		AfterSensitive:     change.AfterSensitive,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}
