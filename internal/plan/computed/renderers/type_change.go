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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
)

var _ computed.DiffRenderer = (*typeChangeRenderer)(nil)

// TypeChange is a renderer for a type change diff
func TypeChange(before, after computed.Diff) computed.DiffRenderer {
	return &typeChangeRenderer{
		before: before,
		after:  after,
	}
}

type typeChangeRenderer struct {
	NoWarningsRenderer

	before computed.Diff
	after  computed.Diff
}

func (renderer typeChangeRenderer) Render(diff computed.Diff) (node.Diff, error) {
	var before, after node.Diff

	beforeModel, err := renderer.before.Render()
	if err != nil {
		return nil, err
	}
	if beforeModel != nil && beforeModel.GetType() == node.DiffTypePrimitive {
		primitiveModel, ok := beforeModel.(*node.PrimitiveDiff)
		if !ok {
			return nil, fmt.Errorf("unexpected primitive model: %T", primitiveModel)
		}
		if primitiveModel.Before != nil {
			before = primitiveModel.Before
		}
	}

	afterModel, err := renderer.after.Render()
	if err != nil {
		return nil, err
	}
	if afterModel != nil && afterModel.GetType() == node.DiffTypePrimitive {
		primitiveModel, ok := afterModel.(*node.PrimitiveDiff)
		if !ok {
			return nil, fmt.Errorf("unexpected primitive model: %T", primitiveModel)
		}
		if primitiveModel.After != nil {
			after = primitiveModel.After
		}
	}

	if before == nil {
		before = beforeModel
	}

	if after == nil {
		after = afterModel
	}

	return node.NewTypeChangeDiff(diff.Action, diff.Replace, diff.Warnings(), before, after), nil
}
