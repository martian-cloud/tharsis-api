// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package renderers provides the rendering logic for computed diffs
package renderers

import (
	"sort"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

var _ computed.DiffRenderer = (*blockRenderer)(nil)

// Block is a renderer for a block diff
func Block(attributes map[string]computed.Diff, blocks Blocks) computed.DiffRenderer {
	return &blockRenderer{
		attributes: attributes,
		blocks:     blocks,
	}
}

type blockRenderer struct {
	NoWarningsRenderer

	attributes map[string]computed.Diff
	blocks     Blocks
}

func (renderer blockRenderer) Render(diff computed.Diff) (node.Diff, error) {
	if len(renderer.attributes) == 0 && len(renderer.blocks.GetAllKeys()) == 0 {
		return node.NewBlockDiff(diff.Action, diff.Replace, nil, []*node.KeyValueDiff{}, []*node.NestedBlockDiff{}), nil
	}

	maxKeyLength := 0
	var attributeKeys []string
	for key := range renderer.attributes {
		attributeKeys = append(attributeKeys, key)
		if len(key) > maxKeyLength {
			maxKeyLength = len(key)
		}
	}
	sort.Strings(attributeKeys)

	attributeModels := []*node.KeyValueDiff{}

	for _, key := range attributeKeys {
		attribute := renderer.attributes[key]

		renderedAttribute, err := attribute.Render()
		if err != nil {
			return nil, err
		}

		attributeModels = append(attributeModels, node.NewKeyValueDiff(
			attribute.Action,
			nil,
			key,
			renderedAttribute,
			true,
			maxKeyLength,
		))
	}

	blockModels := []*node.NestedBlockDiff{}

	blockKeys := renderer.blocks.GetAllKeys()
	for _, key := range blockKeys {

		foundChangedBlock := false
		renderBlock := func(diff computed.Diff, mapKey string) error {

			creatingSensitiveValue := diff.Action == action.Create && renderer.blocks.AfterSensitiveBlocks[key]
			deletingSensitiveValue := diff.Action == action.Delete && renderer.blocks.BeforeSensitiveBlocks[key]
			modifyingSensitiveValue := (diff.Action == action.Update || diff.Action == action.NoOp) && (renderer.blocks.AfterSensitiveBlocks[key] || renderer.blocks.BeforeSensitiveBlocks[key])

			if creatingSensitiveValue || deletingSensitiveValue || modifyingSensitiveValue {
				// Intercept the renderer here if the sensitive data was set
				// across all the blocks instead of individually.
				actionType := diff.Action
				if diff.Action == action.NoOp && renderer.blocks.BeforeSensitiveBlocks[key] != renderer.blocks.AfterSensitiveBlocks[key] {
					actionType = action.Update
				}

				diff = computed.NewDiff(SensitiveBlock(diff, renderer.blocks.BeforeSensitiveBlocks[key], renderer.blocks.AfterSensitiveBlocks[key]), actionType, diff.Replace)
			}

			if diff.Action == action.NoOp {
				return nil
			}

			if !foundChangedBlock && len(renderer.attributes) > 0 {
				foundChangedBlock = true
			}

			// If the force replacement metadata was set for every entry in the
			// block we need to override that here. Our child blocks will only
			// know about the replace function if it was set on them
			// specifically, and not if it was set for all the blocks.
			forcesReplacement := diff.Replace
			if renderer.blocks.ReplaceBlocks[key] {
				forcesReplacement = true
			}

			renderedDiff, err := diff.Render()
			if err != nil {
				return err
			}

			nestedBlockModel := node.NewNestedBlockDiff(
				diff.Action,
				forcesReplacement,
				nil,
				key,
				mapKey,
				renderedDiff,
			)

			blockModels = append(blockModels, nestedBlockModel)

			return nil
		}

		switch {
		case renderer.blocks.IsSingleBlock(key):
			if err := renderBlock(renderer.blocks.SingleBlocks[key], ""); err != nil {
				return nil, err
			}
		case renderer.blocks.IsMapBlock(key):
			var keys []string
			for key := range renderer.blocks.MapBlocks[key] {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			for _, innerKey := range keys {
				if err := renderBlock(renderer.blocks.MapBlocks[key][innerKey], innerKey); err != nil {
					return nil, err
				}
			}
		case renderer.blocks.IsSetBlock(key):
			for _, block := range renderer.blocks.SetBlocks[key] {
				if err := renderBlock(block, ""); err != nil {
					return nil, err
				}
			}
		case renderer.blocks.IsListBlock(key):
			for _, block := range renderer.blocks.ListBlocks[key] {
				if err := renderBlock(block, ""); err != nil {
					return nil, err
				}
			}
		}
	}

	return node.NewBlockDiff(
		diff.Action,
		diff.Replace,
		diff.Warnings(),
		attributeModels,
		blockModels,
	), nil
}
