// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package structured

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured/attributepath"
)

// ChangeSlice is a Change that represents a Tuple, Set, or List type, and has
// converted the relevant interfaces into slices for easier access.
type ChangeSlice struct {
	// Before contains the value before the proposed change.
	Before []interface{}

	// After contains the value after the proposed change.
	After []interface{}

	// Unknown contains the unknown status of any elements of this list/set.
	Unknown []interface{}

	// BeforeSensitive contains the before sensitive status of any elements of
	//this list/set.
	BeforeSensitive []interface{}

	// AfterSensitive contains the after sensitive status of any elements of
	//this list/set.
	AfterSensitive []interface{}

	// ReplacePaths matches the same attributes in Change exactly.
	ReplacePaths attributepath.Matcher

	// RelevantAttributes matches the same attributes in Change exactly.
	RelevantAttributes attributepath.Matcher
}

// AsSlice converts the Change into a slice representation by converting the
// internal Before, After, Unknown, BeforeSensitive, and AfterSensitive data
// structures into generic slices.
func (change Change) AsSlice() ChangeSlice {
	return ChangeSlice{
		Before:             genericToSlice(change.Before),
		After:              genericToSlice(change.After),
		Unknown:            genericToSlice(change.Unknown),
		BeforeSensitive:    genericToSlice(change.BeforeSensitive),
		AfterSensitive:     genericToSlice(change.AfterSensitive),
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}
}

// GetChild safely packages up a Change object for the given child, handling
// all the cases where the data might be null or a static boolean.
func (s ChangeSlice) GetChild(beforeIx, afterIx int) (Change, error) {
	before, beforeExplicit := getFromGenericSlice(s.Before, beforeIx)
	after, afterExplicit := getFromGenericSlice(s.After, afterIx)
	unknown, _ := getFromGenericSlice(s.Unknown, afterIx)
	beforeSensitive, _ := getFromGenericSlice(s.BeforeSensitive, beforeIx)
	afterSensitive, _ := getFromGenericSlice(s.AfterSensitive, afterIx)

	mostRelevantIx := beforeIx
	if beforeIx < 0 || beforeIx >= len(s.Before) {
		mostRelevantIx = afterIx
	}

	replacePaths, err := s.ReplacePaths.GetChildWithIndex(mostRelevantIx)
	if err != nil {
		return Change{}, err
	}

	relevantAttributes, err := s.RelevantAttributes.GetChildWithIndex(mostRelevantIx)
	if err != nil {
		return Change{}, err
	}

	return Change{
		BeforeExplicit:     beforeExplicit,
		AfterExplicit:      afterExplicit,
		Before:             before,
		After:              after,
		Unknown:            unknown,
		BeforeSensitive:    beforeSensitive,
		AfterSensitive:     afterSensitive,
		ReplacePaths:       replacePaths,
		RelevantAttributes: relevantAttributes,
	}, nil
}

func getFromGenericSlice(generic []interface{}, ix int) (interface{}, bool) {
	if generic == nil {
		return nil, false
	}
	if ix < 0 || ix >= len(generic) {
		return nil, false
	}
	return generic[ix], true
}

func genericToSlice(generic interface{}) []interface{} {
	if concrete, ok := generic.([]interface{}); ok {
		return concrete
	}
	return nil
}
