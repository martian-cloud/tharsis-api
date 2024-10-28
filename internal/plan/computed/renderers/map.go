// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"sort"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

var _ computed.DiffRenderer = (*mapRenderer)(nil)

// Map renders a map of computed.Diffs.
func Map(elements map[string]computed.Diff) computed.DiffRenderer {
	return &mapRenderer{
		elements: elements,
	}
}

// NestedMap renders a map of computed.Diffs
func NestedMap(elements map[string]computed.Diff) computed.DiffRenderer {
	return &mapRenderer{
		elements: elements,
		nested:   true,
	}
}

type mapRenderer struct {
	NoWarningsRenderer

	elements map[string]computed.Diff

	nested bool
}

func (renderer mapRenderer) Render(diff computed.Diff) (node.Diff, error) {
	forcesReplacementSelf := diff.Replace && !renderer.nested
	forcesReplacementChildren := diff.Replace && renderer.nested

	if len(renderer.elements) == 0 {
		return node.NewJSONObjectDiff(diff.Action, diff.Replace, diff.Warnings(), nil), nil
	}

	// Sort the map elements by key, so we have a deterministic ordering in
	// the output.
	var keys []string
	maxKeyLength := 0
	for key := range renderer.elements {
		keys = append(keys, key)
		if len(key) > maxKeyLength {
			maxKeyLength = len(key)
		}
	}
	sort.Strings(keys)

	attributeModels := []*node.KeyValueDiff{}

	for _, key := range keys {
		element := renderer.elements[key]

		if forcesReplacementChildren {
			element.Replace = true
		}

		elementModel, err := element.Render()
		if err != nil {
			return nil, err
		}

		attributeModels = append(attributeModels, node.NewKeyValueDiff(
			element.Action,
			nil,
			key,
			elementModel,
			false,
			maxKeyLength,
		))
	}

	return node.NewJSONObjectDiff(diff.Action, forcesReplacementSelf, diff.Warnings(), attributeModels), nil
}
