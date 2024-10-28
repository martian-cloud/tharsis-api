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

var _ computed.DiffRenderer = (*unknownRenderer)(nil)

// Unknown is a renderer for an unknown diff
func Unknown(before computed.Diff) computed.DiffRenderer {
	return &unknownRenderer{
		before: before,
	}
}

type unknownRenderer struct {
	NoWarningsRenderer

	before computed.Diff
}

func (renderer unknownRenderer) Render(diff computed.Diff) (node.Diff, error) {
	warnings := diff.Warnings()

	if diff.Action == action.Create {
		return node.NewUnknownDiff(diff.Action, diff.Replace, warnings, nil), nil
	}

	m, err := renderer.before.Render()
	if err != nil {
		return nil, err
	}
	if m != nil && m.GetType() == node.DiffTypePrimitive {
		primitiveModel, ok := m.(*node.PrimitiveDiff)
		if !ok {
			return nil, fmt.Errorf("unexpected primitive model: %T", primitiveModel)
		}
		return node.NewUnknownDiff(diff.Action, diff.Replace, warnings, primitiveModel.Before), nil
	}

	renderedBefore, err := renderer.before.Render()
	if err != nil {
		return nil, err
	}

	return node.NewUnknownDiff(diff.Action, diff.Replace, warnings, renderedBefore), nil
}
