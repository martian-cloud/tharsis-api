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

var _ computed.DiffRenderer = (*objectRenderer)(nil)

// Object renders a map of computed.Diffs.
func Object(attributes map[string]computed.Diff) computed.DiffRenderer {
	return &objectRenderer{
		attributes: attributes,
	}
}

// NestedObject renders a nested map of computed.Diffs
func NestedObject(attributes map[string]computed.Diff) computed.DiffRenderer {
	return &objectRenderer{
		attributes: attributes,
		nested:     true,
	}
}

type objectRenderer struct {
	NoWarningsRenderer

	attributes map[string]computed.Diff
	nested     bool
}

func (renderer objectRenderer) Render(diff computed.Diff) (node.Diff, error) {
	if len(renderer.attributes) == 0 {
		return node.NewJSONObjectDiff(diff.Action, diff.Replace, diff.Warnings(), []*node.KeyValueDiff{}), nil
	}

	// Sort the map elements by key, so we have a deterministic ordering in
	// the output.
	var keys []string
	maxKeyLength := 0
	for key := range renderer.attributes {
		keys = append(keys, key)
		if len(key) > maxKeyLength {
			maxKeyLength = len(key)
		}
	}
	sort.Strings(keys)

	attributeModels := []*node.KeyValueDiff{}

	for _, key := range keys {
		element := renderer.attributes[key]

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

	return node.NewJSONObjectDiff(diff.Action, diff.Replace, diff.Warnings(), attributeModels), nil
}
