// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package structured

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"

// ProcessUnknown is a function that processes an unknown value
type ProcessUnknown func(current Change) (computed.Diff, error)

// ProcessUnknownWithBefore is a function that processes an unknown value with a before value
type ProcessUnknownWithBefore func(current Change, before Change) (computed.Diff, error)

// IsUnknown returns true if the change is unknown
func (change Change) IsUnknown() bool {
	if unknown, ok := change.Unknown.(bool); ok {
		return unknown
	}
	return false
}

// CheckForUnknown is a helper function that handles all common functionality
// for processing an unknown value.
//
// It returns the computed unknown diff and true if this value was unknown and
// needs to be rendered as such, otherwise it returns the second return value as
// false and the first return value should be discarded.
//
// The actual processing of unknown values happens in the ProcessUnknown and
// ProcessUnknownWithBefore functions. If a value is unknown and is being
// created, the ProcessUnknown function is called and the caller should decide
// how to create the unknown value. If a value is being updated the
// ProcessUnknownWithBefore function is called and the function provides the
// before value as if it is being deleted for the caller to handle. Note that
// values being deleted will never be marked as unknown so this case isn't
// handled.
//
// The childUnknown argument is meant to allow callers with extra information
// about the type being processed to provide a list of known children that might
// not be present in the before or after values. These values will be propagated
// as the unknown values in the before value should it be needed.
func (change Change) CheckForUnknown(childUnknown interface{}, process ProcessUnknown, processBefore ProcessUnknownWithBefore) (computed.Diff, bool, error) {
	unknown := change.IsUnknown()

	if !unknown {
		return computed.Diff{}, false, nil
	}

	// No matter what we do here, we want to treat the after value as explicit.
	// This is because it is going to be null in the value, and we don't want
	// the functions in this package to assume this means it has been deleted.
	change.AfterExplicit = true

	if change.Before == nil {
		processedBefore, err := process(change)
		if err != nil {
			return computed.Diff{}, false, err
		}
		return processedBefore, true, nil
	}

	// If we get here, then we have a before value. We're going to model a
	// delete operation and our renderer later can render the overall change
	// accurately.
	before := change.AsDelete()

	// We also let our callers override the unknown values in any before, this
	// is the renderers can display them as being computed instead of deleted.
	before.Unknown = childUnknown

	processedBefore, err := processBefore(change, before)
	if err != nil {
		return computed.Diff{}, false, err
	}
	return processedBefore, true, nil
}
