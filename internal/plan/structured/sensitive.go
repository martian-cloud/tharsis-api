// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package structured

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
)

// ProcessSensitiveInner is a callback function that processes a sensitive change
type ProcessSensitiveInner func(change Change) (computed.Diff, error)

// CreateSensitiveDiff is a callback function that creates a sensitive diff
type CreateSensitiveDiff func(inner computed.Diff, beforeSensitive, afterSensitive bool, action action.Action) computed.Diff

// IsBeforeSensitive returns true if the before value is sensitive
func (change Change) IsBeforeSensitive() bool {
	if sensitive, ok := change.BeforeSensitive.(bool); ok {
		return sensitive
	}
	return false
}

// IsAfterSensitive returns true if the after value is sensitive
func (change Change) IsAfterSensitive() bool {
	if sensitive, ok := change.AfterSensitive.(bool); ok {
		return sensitive
	}
	return false
}

// CheckForSensitive is a helper function that handles all common functionality
// for processing a sensitive value.
//
// It returns the computed sensitive diff and true if this value was sensitive
// and needs to be rendered as such, otherwise it returns the second return
// value as false and the first value can be discarded.
//
// The actual processing of sensitive values happens within the
// ProcessSensitiveInner and CreateSensitiveDiff functions. Callers should
// implement these functions as appropriate when using this function.
//
// The ProcessSensitiveInner function should simply return a computed.Diff for
// the provided Change. The provided Change will be the same as the original
// change but with the sensitive metadata removed. The new inner diff is then
// passed into the actual CreateSensitiveDiff function which should return the
// actual sensitive diff.
//
// We include the inner change into the sensitive diff as a way to let the
// sensitive renderer have as much information as possible, while still letting
// it do the actual rendering.
func (change Change) CheckForSensitive(processInner ProcessSensitiveInner, createDiff CreateSensitiveDiff) (computed.Diff, bool, error) {
	beforeSensitive := change.IsBeforeSensitive()
	afterSensitive := change.IsAfterSensitive()

	if !beforeSensitive && !afterSensitive {
		return computed.Diff{}, false, nil
	}

	// We are still going to give the change the contents of the actual change.
	// So we create a new Change with everything matching the current value,
	// except for the sensitivity.
	//
	// The change can choose what to do with this information, in most cases
	// it will just be ignored in favour of printing `(sensitive value)`.

	value := Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      change.AfterExplicit,
		Before:             change.Before,
		After:              change.After,
		Unknown:            change.Unknown,
		BeforeSensitive:    false,
		AfterSensitive:     false,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}

	inner, err := processInner(value)
	if err != nil {
		return computed.Diff{}, false, err
	}

	actionType := inner.Action
	sensitiveStatusChanged := beforeSensitive != afterSensitive

	// nullNoOp is a stronger NoOp, where not only is there no change happening
	// but the before and after values are not explicitly set and are both
	// null. This will override even the sensitive state changing.
	nullNoOp := change.Before == nil && !change.BeforeExplicit && change.After == nil && !change.AfterExplicit

	if actionType == action.NoOp && sensitiveStatusChanged && !nullNoOp {
		// Let's override this, since it means the sensitive status has changed
		// rather than the actual content of the value.
		actionType = action.Update
	}

	return createDiff(inner, beforeSensitive, afterSensitive, actionType), true, nil
}
