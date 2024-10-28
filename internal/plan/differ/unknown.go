// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	jsonprovider "github.com/hashicorp/terraform-json"
)

func checkForUnknownType(change structured.Change, ctype cty.Type) (computed.Diff, bool, error) {
	return change.CheckForUnknown(
		false,
		processUnknown,
		createProcessUnknownWithBefore(func(value structured.Change) (computed.Diff, error) {
			return ComputeDiffForType(value, ctype)
		}))
}

func checkForUnknownNestedAttribute(change structured.Change, attribute *jsonprovider.SchemaNestedAttributeType) (computed.Diff, bool, error) {

	// We want our child attributes to show up as computed instead of deleted.
	// Let's populate that here.
	childUnknown := make(map[string]interface{})
	for key := range attribute.Attributes {
		childUnknown[key] = true
	}

	return change.CheckForUnknown(
		childUnknown,
		processUnknown,
		createProcessUnknownWithBefore(func(value structured.Change) (computed.Diff, error) {
			return computeDiffForNestedAttribute(value, attribute)
		}))
}

func checkForUnknownBlock(change structured.Change, block *jsonprovider.SchemaBlock) (computed.Diff, bool, error) {

	// We want our child attributes to show up as computed instead of deleted.
	// Let's populate that here.
	childUnknown := make(map[string]interface{})
	for key := range block.Attributes {
		childUnknown[key] = true
	}

	return change.CheckForUnknown(
		childUnknown,
		processUnknown,
		createProcessUnknownWithBefore(func(value structured.Change) (computed.Diff, error) {
			return ComputeDiffForBlock(value, block)
		}))
}

func processUnknown(current structured.Change) (computed.Diff, error) {
	return asDiff(current, renderers.Unknown(computed.Diff{})), nil
}

func createProcessUnknownWithBefore(computeDiff func(value structured.Change) (computed.Diff, error)) structured.ProcessUnknownWithBefore {
	return func(current structured.Change, before structured.Change) (computed.Diff, error) {
		diff, err := computeDiff(before)
		if err != nil {
			return computed.Diff{}, err
		}
		return asDiff(current, renderers.Unknown(diff)), nil
	}
}
