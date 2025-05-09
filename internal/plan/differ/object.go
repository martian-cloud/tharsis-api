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

func computeAttributeDiffAsObject(change structured.Change, attributes map[string]cty.Type) (computed.Diff, error) {
	attributeDiffs, action, err := processObject(change, attributes, func(value structured.Change, ctype cty.Type) (computed.Diff, error) {
		return ComputeDiffForType(value, ctype)
	})
	if err != nil {
		return computed.Diff{}, err
	}
	return computed.NewDiff(renderers.Object(attributeDiffs), action, change.ReplacePaths.Matches()), nil
}

func computeAttributeDiffAsNestedObject(change structured.Change, attributes map[string]*jsonprovider.SchemaAttribute) (computed.Diff, error) {
	attributeDiffs, action, err := processObject(change, attributes, func(value structured.Change, attribute *jsonprovider.SchemaAttribute) (computed.Diff, error) {
		return ComputeDiffForAttribute(value, attribute)
	})
	if err != nil {
		return computed.Diff{}, err
	}
	return computed.NewDiff(renderers.NestedObject(attributeDiffs), action, change.ReplacePaths.Matches()), nil
}

// processObject steps through the children of value as if it is an object and
// calls out to the provided computeDiff function once it has collated the
// diffs for each child attribute.
//
// We have to make this generic as attributes and nested objects process either
// cty.Type or jsonprovider.Attribute children respectively. And we want to
// reuse as much code as possible.
//
// Also, as it generic we cannot make this function a method on Change as you
// can't create generic methods on structs. Instead, we make this a generic
// function that receives the value as an argument.
func processObject[T any](v structured.Change, attributes map[string]T, computeDiff func(structured.Change, T) (computed.Diff, error)) (map[string]computed.Diff, action.Action, error) {
	attributeDiffs := make(map[string]computed.Diff)
	mapValue := v.AsMap()

	currentAction := v.GetDefaultActionForIteration()
	for key, attribute := range attributes {
		attributeValue := mapValue.GetChild(key)

		if !attributeValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			attributeValue = attributeValue.AsNoOp()
		}

		// We always assume changes to object are implicit.
		attributeValue.BeforeExplicit = false
		attributeValue.AfterExplicit = false

		attributeDiff, err := computeDiff(attributeValue, attribute)
		if err != nil {
			return nil, currentAction, err
		}
		if attributeDiff.Action == action.NoOp && attributeValue.Before == nil && attributeValue.After == nil {
			// We skip attributes of objects that are null both before and
			// after. We don't even count these as unchanged attributes.
			continue
		}
		attributeDiffs[key] = attributeDiff
		currentAction = collections.CompareActions(currentAction, attributeDiff.Action)
	}

	return attributeDiffs, currentAction, nil
}
