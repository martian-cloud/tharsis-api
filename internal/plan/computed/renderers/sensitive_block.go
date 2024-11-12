// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

// SensitiveBlock is a renderer for a sensitive block diff
func SensitiveBlock(diff computed.Diff, beforeSensitive, afterSensitive bool) computed.DiffRenderer {
	return &sensitiveBlockRenderer{
		inner:           diff,
		beforeSensitive: beforeSensitive,
		afterSensitive:  afterSensitive,
	}
}

type sensitiveBlockRenderer struct {
	inner computed.Diff

	afterSensitive  bool
	beforeSensitive bool
}

func (renderer sensitiveBlockRenderer) Render(diff computed.Diff) (node.Diff, error) {
	sensitiveModel := node.NewSensitiveBlockDiff(diff.Action, diff.Replace, diff.Warnings(), renderer.beforeSensitive, renderer.afterSensitive)
	sensitiveModel.Block = true
	return sensitiveModel, nil
}

func (renderer sensitiveBlockRenderer) Warnings(_ computed.Diff) []string {
	if (renderer.beforeSensitive == renderer.afterSensitive) || renderer.inner.Action == action.Create || renderer.inner.Action == action.Delete {
		// Only display warnings for sensitive values if they are changing from
		// being sensitive or to being sensitive and if they are not being
		// destroyed or created.
		return []string{}
	}

	if renderer.beforeSensitive {
		return []string{"This block will no longer be marked as sensitive after applying this change"}
	}

	return []string{"This block will be marked as sensitive after applying this change"}
}
