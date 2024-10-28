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

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/collections"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	jsonprovider "github.com/hashicorp/terraform-json"
)

func computeAttributeDiffAsMap(change structured.Change, elementType cty.Type) (computed.Diff, error) {
	mapValue := change.AsMap()
	elements, current, err := collections.TransformMap(mapValue.Before, mapValue.After, mapValue.AllKeys(), func(key string) (computed.Diff, error) {
		value := mapValue.GetChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return ComputeDiffForType(value, elementType)
	})
	if err != nil {
		return computed.Diff{}, err
	}
	return computed.NewDiff(renderers.Map(elements), current, change.ReplacePaths.Matches()), nil
}

func computeAttributeDiffAsNestedMap(change structured.Change, attributes map[string]*jsonprovider.SchemaAttribute) (computed.Diff, error) {
	mapValue := change.AsMap()
	elements, current, err := collections.TransformMap(mapValue.Before, mapValue.After, mapValue.ExplicitKeys(), func(key string) (computed.Diff, error) {
		value := mapValue.GetChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return computeDiffForNestedAttribute(value, &jsonprovider.SchemaNestedAttributeType{
			Attributes:  attributes,
			NestingMode: "single",
		})
	})
	if err != nil {
		return computed.Diff{}, err
	}
	return computed.NewDiff(renderers.NestedMap(elements), current, change.ReplacePaths.Matches()), nil
}

func computeBlockDiffsAsMap(change structured.Change, block *jsonprovider.SchemaBlock) (map[string]computed.Diff, action.Action, error) {
	mapValue := change.AsMap()
	return collections.TransformMap(mapValue.Before, mapValue.After, mapValue.ExplicitKeys(), func(key string) (computed.Diff, error) {
		value := mapValue.GetChild(key)
		if !value.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			value = value.AsNoOp()
		}
		return ComputeDiffForBlock(value, block)
	})
}
