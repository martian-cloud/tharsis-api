// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured/attributepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/collections"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	jsonprovider "github.com/hashicorp/terraform-json"
)

func computeAttributeDiffAsList(change structured.Change, elementType cty.Type) (computed.Diff, error) {
	sliceValue := change.AsSlice()

	processIndices := func(beforeIx, afterIx int) (computed.Diff, error) {
		value, err := sliceValue.GetChild(beforeIx, afterIx)
		if err != nil {
			return computed.Diff{}, err
		}

		// It's actually really difficult to render the diffs when some indices
		// within a slice are relevant and others aren't. To make this simpler
		// we just treat all children of a relevant list or set as also
		// relevant.
		//
		// Interestingly the terraform plan builder also agrees with this, and
		// never sets relevant attributes beneath lists or sets. We're just
		// going to enforce this logic here as well. If the collection is
		// relevant (decided elsewhere), then every element in the collection is
		// also relevant. To be clear, in practice even if we didn't do the
		// following explicitly the effect would be the same. It's just nicer
		// for us to be clear about the behaviour we expect.
		//
		// What makes this difficult is the fact that the beforeIx and afterIx
		// can be different, and it's quite difficult to work out which one is
		// the relevant one. For nested lists, block lists, and tuples it's much
		// easier because we always process the same indices in the before and
		// after.
		value.RelevantAttributes = attributepath.AlwaysMatcher()

		return ComputeDiffForType(value, elementType)
	}

	isObjType := func(_ interface{}) (bool, error) {
		return elementType.IsObjectType(), nil
	}

	elements, current, err := collections.TransformSlice(sliceValue.Before, sliceValue.After, processIndices, isObjType)
	if err != nil {
		return computed.Diff{}, err
	}
	return computed.NewDiff(renderers.List(elements), current, change.ReplacePaths.Matches()), nil
}

func computeAttributeDiffAsNestedList(change structured.Change, attributes map[string]*jsonprovider.SchemaAttribute) computed.Diff {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processNestedList(change, func(value structured.Change) error {
		element, err := computeDiffForNestedAttribute(value, &jsonprovider.SchemaNestedAttributeType{
			Attributes:  attributes,
			NestingMode: "single",
		})
		if err != nil {
			return err
		}
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
		return nil
	})
	return computed.NewDiff(renderers.NestedList(elements), current, change.ReplacePaths.Matches())
}

func computeBlockDiffsAsList(change structured.Change, block *jsonprovider.SchemaBlock) ([]computed.Diff, action.Action) {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	processNestedList(change, func(value structured.Change) error {
		element, err := ComputeDiffForBlock(value, block)
		if err != nil {
			return err
		}
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
		return nil
	})
	return elements, current
}

func processNestedList(change structured.Change, process func(value structured.Change) error) error {
	sliceValue := change.AsSlice()
	for ix := 0; ix < len(sliceValue.Before) || ix < len(sliceValue.After); ix++ {
		value, err := sliceValue.GetChild(ix, ix)
		if err != nil {
			return err
		}
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		if err := process(value); err != nil {
			return err
		}
	}
	return nil
}
