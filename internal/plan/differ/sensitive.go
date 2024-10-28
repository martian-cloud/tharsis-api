// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	tjson "github.com/hashicorp/terraform-json"
)

// CreateSensitiveRenderer is a function that creates a sensitive renderer
type CreateSensitiveRenderer func(computed.Diff, bool, bool) computed.DiffRenderer

func checkForSensitiveType(change structured.Change, ctype cty.Type) (computed.Diff, bool, error) {
	return change.CheckForSensitive(
		func(value structured.Change) (computed.Diff, error) {
			return ComputeDiffForType(value, ctype)
		}, func(inner computed.Diff, beforeSensitive, afterSensitive bool, action action.Action) computed.Diff {
			return computed.NewDiff(renderers.Sensitive(inner, beforeSensitive, afterSensitive), action, change.ReplacePaths.Matches())
		},
	)
}

func checkForSensitiveNestedAttribute(change structured.Change, attribute *tjson.SchemaNestedAttributeType) (computed.Diff, bool, error) {
	return change.CheckForSensitive(
		func(value structured.Change) (computed.Diff, error) {
			return computeDiffForNestedAttribute(value, attribute)
		}, func(inner computed.Diff, beforeSensitive, afterSensitive bool, action action.Action) computed.Diff {
			return computed.NewDiff(renderers.Sensitive(inner, beforeSensitive, afterSensitive), action, change.ReplacePaths.Matches())
		},
	)
}

func checkForSensitiveBlock(change structured.Change, block *tjson.SchemaBlock) (computed.Diff, bool, error) {
	return change.CheckForSensitive(
		func(value structured.Change) (computed.Diff, error) {
			return ComputeDiffForBlock(value, block)
		}, func(inner computed.Diff, beforeSensitive, afterSensitive bool, action action.Action) computed.Diff {
			return computed.NewDiff(renderers.SensitiveBlock(inner, beforeSensitive, afterSensitive), action, change.ReplacePaths.Matches())
		},
	)
}
