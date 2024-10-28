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

var _ computed.DiffRenderer = (*setRenderer)(nil)

// Set renders a set of computed.Diffs.
func Set(elements []computed.Diff) computed.DiffRenderer {
	return &setRenderer{
		elements: elements,
	}
}

// NestedSet renders a nested set of computed.Diffs
func NestedSet(elements []computed.Diff) computed.DiffRenderer {
	return &setRenderer{
		elements: elements,
		nested:   true,
	}
}

type setRenderer struct {
	NoWarningsRenderer

	elements []computed.Diff

	nested bool
}

func (renderer setRenderer) Render(diff computed.Diff) (node.Diff, error) {
	// Sets are a bit finicky, nested sets don't render the forces replacement
	// suffix themselves, but push it onto their children. So if we are
	// overriding the forces replacement setting, we set it to true for children
	// and false for ourselves.
	displayForcesReplacementInSelf := diff.Replace && !renderer.nested
	displayForcesReplacementInChildren := diff.Replace && renderer.nested

	if len(renderer.elements) == 0 {
		return node.NewJSONArray(diff.Action, diff.Replace, diff.Warnings(), nil), nil
	}

	elementModels := []node.Diff{}

	for _, element := range renderer.elements {
		if displayForcesReplacementInChildren {
			element.Replace = true
		}

		elementModel, err := element.Render()
		if err != nil {
			return nil, err
		}
		elementModels = append(elementModels, elementModel)
	}

	return node.NewJSONArray(diff.Action, displayForcesReplacementInSelf, diff.Warnings(), elementModels), nil
}
