// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/collections"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	tjson "github.com/hashicorp/terraform-json"
)

// ComputeDiffForBlock computes the diff for a block.
func ComputeDiffForBlock(change structured.Change, block *tjson.SchemaBlock) (computed.Diff, error) {
	sensitive, ok, err := checkForSensitiveBlock(change, block)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return sensitive, nil
	}

	unknown, ok, err := checkForUnknownBlock(change, block)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return unknown, nil
	}

	current := change.GetDefaultActionForIteration()

	blockValue := change.AsMap()

	attributes := make(map[string]computed.Diff)
	for key, attr := range block.Attributes {
		childValue := blockValue.GetChild(key)

		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}

		// Empty strings in blocks should be considered null for legacy reasons.
		// The SDK doesn't support null strings yet, so we work around this now.
		if before, ok := childValue.Before.(string); ok && len(before) == 0 {
			childValue.Before = nil
		}
		if after, ok := childValue.After.(string); ok && len(after) == 0 {
			childValue.After = nil
		}

		// Always treat changes to blocks as implicit.
		childValue.BeforeExplicit = false
		childValue.AfterExplicit = false

		childChange, err := ComputeDiffForAttribute(childValue, attr)
		if err != nil {
			return computed.Diff{}, err
		}
		if childChange.Action == action.NoOp && childValue.Before == nil && childValue.After == nil {
			// Don't record nil values at all in blocks.
			continue
		}

		attributes[key] = childChange
		current = collections.CompareActions(current, childChange.Action)
	}

	blocks := renderers.Blocks{
		ReplaceBlocks:         make(map[string]bool),
		BeforeSensitiveBlocks: make(map[string]bool),
		AfterSensitiveBlocks:  make(map[string]bool),
		SingleBlocks:          make(map[string]computed.Diff),
		ListBlocks:            make(map[string][]computed.Diff),
		SetBlocks:             make(map[string][]computed.Diff),
		MapBlocks:             make(map[string]map[string]computed.Diff),
	}

	for key, blockType := range block.NestedBlocks {
		childValue := blockValue.GetChild(key)

		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}

		beforeSensitive := childValue.IsBeforeSensitive()
		afterSensitive := childValue.IsAfterSensitive()
		forcesReplacement := childValue.ReplacePaths.Matches()

		switch NestingMode(blockType.NestingMode) {
		case nestingModeSet:
			diffs, actionType := computeBlockDiffsAsSet(childValue, blockType.Block)
			if actionType == action.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllSetBlock(key, diffs, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, actionType)
		case nestingModeList:
			diffs, actionType := computeBlockDiffsAsList(childValue, blockType.Block)
			if actionType == action.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllListBlock(key, diffs, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, actionType)
		case nestingModeMap:
			diffs, actionType, err := computeBlockDiffsAsMap(childValue, blockType.Block)
			if err != nil {
				return computed.Diff{}, err
			}
			if actionType == action.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddAllMapBlocks(key, diffs, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, actionType)
		case nestingModeSingle, nestingModeGroup:
			diff, err := ComputeDiffForBlock(childValue, blockType.Block)
			if err != nil {
				return computed.Diff{}, err
			}
			if diff.Action == action.NoOp && childValue.Before == nil && childValue.After == nil {
				// Don't record nil values in blocks.
				continue
			}
			blocks.AddSingleBlock(key, diff, forcesReplacement, beforeSensitive, afterSensitive)
			current = collections.CompareActions(current, diff.Action)
		default:
			return computed.Diff{}, fmt.Errorf("unrecognized nesting mode: %s", blockType.NestingMode)
		}
	}

	return computed.NewDiff(renderers.Block(attributes, blocks), current, change.ReplacePaths.Matches()), nil
}
