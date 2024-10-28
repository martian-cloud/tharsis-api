// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/collections"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"
)

func computeAttributeDiffAsTuple(change structured.Change, elementTypes []cty.Type) (computed.Diff, error) {
	var elements []computed.Diff
	current := change.GetDefaultActionForIteration()
	sliceValue := change.AsSlice()
	for ix, elementType := range elementTypes {
		childValue, err := sliceValue.GetChild(ix, ix)
		if err != nil {
			return computed.Diff{}, err
		}
		if !childValue.RelevantAttributes.MatchesPartial() {
			// Mark non-relevant attributes as unchanged.
			childValue = childValue.AsNoOp()
		}
		element, err := ComputeDiffForType(childValue, elementType)
		if err != nil {
			return computed.Diff{}, err
		}
		elements = append(elements, element)
		current = collections.CompareActions(current, element.Action)
	}
	return computed.NewDiff(renderers.List(elements), current, change.ReplacePaths.Matches()), nil
}
