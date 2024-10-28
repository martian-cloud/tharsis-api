// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	tjson "github.com/hashicorp/terraform-json"
)

// ComputeDiffForAttribute computes the diff for a single attribute
func ComputeDiffForAttribute(change structured.Change, attribute *tjson.SchemaAttribute) (computed.Diff, error) {
	if attribute.AttributeNestedType != nil {
		return computeDiffForNestedAttribute(change, attribute.AttributeNestedType)
	}
	return ComputeDiffForType(change, attribute.AttributeType)
}

func computeDiffForNestedAttribute(change structured.Change, nested *tjson.SchemaNestedAttributeType) (computed.Diff, error) {
	sensitive, ok, err := checkForSensitiveNestedAttribute(change, nested)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return sensitive, nil
	}

	computedDiff, ok, err := checkForUnknownNestedAttribute(change, nested)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return computedDiff, nil
	}

	switch NestingMode(nested.NestingMode) {
	case nestingModeSingle, nestingModeGroup:
		return computeAttributeDiffAsNestedObject(change, nested.Attributes)
	case nestingModeMap:
		return computeAttributeDiffAsNestedMap(change, nested.Attributes)
	case nestingModeList:
		return computeAttributeDiffAsNestedList(change, nested.Attributes), nil
	case nestingModeSet:
		return computeAttributeDiffAsNestedSet(change, nested.Attributes), nil
	default:
		return computed.Diff{}, fmt.Errorf("unrecognized nesting mode: %v", nested.NestingMode)
	}
}

// ComputeDiffForType computes the diff for a change that has a cty type
func ComputeDiffForType(change structured.Change, ctype cty.Type) (computed.Diff, error) {
	sensitive, ok, err := checkForSensitiveType(change, ctype)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return sensitive, nil
	}

	computedDiff, ok, err := checkForUnknownType(change, ctype)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return computedDiff, nil
	}

	switch {
	case ctype == cty.NilType, ctype == cty.DynamicPseudoType:
		// Forward nil or dynamic types over to be processed as outputs.
		// There is nothing particularly special about the way outputs are
		// processed that make this unsafe, we could just as easily call this
		// function computeChangeForDynamicValues(), but external callers will
		// only be in this situation when processing outputs so this function
		// is named for their benefit.
		return ComputeDiffForOutput(change)
	case ctype.IsPrimitiveType():
		return computeAttributeDiffAsPrimitive(change, ctype), nil
	case ctype.IsObjectType():
		return computeAttributeDiffAsObject(change, ctype.AttributeTypes())
	case ctype.IsMapType():
		return computeAttributeDiffAsMap(change, ctype.ElementType())
	case ctype.IsListType():
		return computeAttributeDiffAsList(change, ctype.ElementType())
	case ctype.IsTupleType():
		return computeAttributeDiffAsTuple(change, ctype.TupleElementTypes())
	case ctype.IsSetType():
		return computeAttributeDiffAsSet(change, ctype.ElementType()), nil
	default:
		return computed.Diff{}, fmt.Errorf("unrecognized type: %s", ctype.FriendlyName())
	}
}
