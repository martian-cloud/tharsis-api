// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
)

var _ computed.DiffRenderer = (*sensitiveRenderer)(nil)

// Sensitive renders a sensitive computed.Diff.
func Sensitive(change computed.Diff, beforeSensitive, afterSensitive bool) computed.DiffRenderer {
	return &sensitiveRenderer{
		inner:           change,
		beforeSensitive: beforeSensitive,
		afterSensitive:  afterSensitive,
	}
}

type sensitiveRenderer struct {
	inner computed.Diff

	beforeSensitive bool
	afterSensitive  bool
}

func (renderer sensitiveRenderer) Render(diff computed.Diff) (node.Diff, error) {
	sensitiveModel := node.NewSensitiveDiff(diff.Action, diff.Replace, diff.Warnings(), renderer.beforeSensitive, renderer.afterSensitive)
	return sensitiveModel, nil
}

func (renderer sensitiveRenderer) Warnings(_ computed.Diff) []string {
	if (renderer.beforeSensitive == renderer.afterSensitive) || renderer.inner.Action == action.Create || renderer.inner.Action == action.Delete {
		// Only display warnings for sensitive values if they are changing from
		// being sensitive or to being sensitive and if they are not being
		// destroyed or created.
		return []string{}
	}

	var warning string
	if renderer.beforeSensitive {
		warning = "This attribute value will no longer be marked as sensitive after applying this change"
	} else {
		warning = "This attribute value will be marked as sensitive after applying this change"
	}

	if renderer.inner.Action == action.NoOp {
		return []string{fmt.Sprintf("%s (the value is unchanged)", warning)}
	}
	return []string{warning}
}
