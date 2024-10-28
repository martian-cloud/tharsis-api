// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

var _ computed.DiffRenderer = (*listRenderer)(nil)

// List renders a list of computed.Diffs.
func List(elements []computed.Diff) computed.DiffRenderer {
	return &listRenderer{
		elements: elements,
	}
}

// NestedList renders a nested list of computed.Diffs
func NestedList(elements []computed.Diff) computed.DiffRenderer {
	return &listRenderer{
		elements: elements,
		nested:   true,
	}
}

type listRenderer struct {
	NoWarningsRenderer

	nested   bool
	elements []computed.Diff
}

func (renderer listRenderer) Render(diff computed.Diff) (node.Diff, error) {
	if len(renderer.elements) == 0 {
		return node.NewJSONArray(diff.Action, diff.Replace, diff.Warnings(), nil), nil
	}

	elementModels := []node.Diff{}

	for _, element := range renderer.elements {
		elementModel, err := element.Render()
		if err != nil {
			return nil, err
		}

		elementModels = append(elementModels, elementModel)
	}

	return node.NewJSONArray(diff.Action, diff.Replace, diff.Warnings(), elementModels), nil
}
