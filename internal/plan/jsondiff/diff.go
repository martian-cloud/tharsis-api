// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package jsondiff provides utilities for working with JSON types
package jsondiff

import (
	"fmt"
	"reflect"

	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/collections"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"
)

// TransformPrimitiveJSON is a callback function that processes a primitive JSON
type TransformPrimitiveJSON func(before, after interface{}, ctype cty.Type, action action.Action) computed.Diff

// TransformObjectJSON is a callback function that processes an object JSON
type TransformObjectJSON func(map[string]computed.Diff, action.Action) computed.Diff

// TransformArrayJSON is a callback function that processes an array JSON
type TransformArrayJSON func([]computed.Diff, action.Action) computed.Diff

// TransformUnknownJSON is a callback function that processes an unknown JSON
type TransformUnknownJSON func(computed.Diff, action.Action) computed.Diff

// TransformSensitiveJSON is a callback function that processes a sensitive JSON
type TransformSensitiveJSON func(computed.Diff, bool, bool, action.Action) computed.Diff

// TransformTypeChangeJSON is a callback function that processes a type change JSON
type TransformTypeChangeJSON func(before, after computed.Diff, action action.Action) computed.Diff

// JSONOpts defines the external callback functions that callers should
// implement to process the supplied diffs.
type JSONOpts struct {
	Primitive  TransformPrimitiveJSON
	Object     TransformObjectJSON
	Array      TransformArrayJSON
	Unknown    TransformUnknownJSON
	Sensitive  TransformSensitiveJSON
	TypeChange TransformTypeChangeJSON
}

// Transform accepts a generic before and after value that is assumed to be JSON
// formatted and transforms it into a computed.Diff, using the callbacks
// supplied in the JSONOpts class.
func (opts JSONOpts) Transform(change structured.Change) (computed.Diff, error) {
	sensitive, ok, err := opts.processSensitive(change)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return sensitive, nil
	}

	unknown, ok, err := opts.processUnknown(change)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return unknown, nil
	}

	beforeType, err := GetType(change.Before)
	if err != nil {
		return computed.Diff{}, err
	}
	afterType, err := GetType(change.After)
	if err != nil {
		return computed.Diff{}, err
	}

	deleted := afterType == Null && !change.AfterExplicit
	created := beforeType == Null && !change.BeforeExplicit

	if beforeType == afterType || (created || deleted) {
		targetType := beforeType
		if targetType == Null {
			targetType = afterType
		}
		return opts.processUpdate(change, targetType)
	}

	b, err := opts.processUpdate(change.AsDelete(), beforeType)
	if err != nil {
		return computed.Diff{}, err
	}
	a, err := opts.processUpdate(change.AsCreate(), afterType)
	if err != nil {
		return computed.Diff{}, err
	}
	return opts.TypeChange(b, a, action.Update), nil
}

func (opts JSONOpts) processUpdate(change structured.Change, jtype Type) (computed.Diff, error) {
	switch jtype {
	case Null:
		return opts.processPrimitive(change, cty.NilType)
	case Bool:
		return opts.processPrimitive(change, cty.Bool)
	case String:
		return opts.processPrimitive(change, cty.String)
	case Number:
		return opts.processPrimitive(change, cty.Number)
	case Object:
		return opts.processObject(change.AsMap())
	case Array:
		return opts.processArray(change.AsSlice())
	default:
		return computed.Diff{}, fmt.Errorf("unrecognized json type: %s", jtype)
	}
}

func (opts JSONOpts) processPrimitive(change structured.Change, ctype cty.Type) (computed.Diff, error) {
	beforeMissing := change.Before == nil && !change.BeforeExplicit
	afterMissing := change.After == nil && !change.AfterExplicit

	var actionType action.Action
	switch {
	case beforeMissing && !afterMissing:
		actionType = action.Create
	case !beforeMissing && afterMissing:
		actionType = action.Delete
	case reflect.DeepEqual(change.Before, change.After):
		actionType = action.NoOp
	default:
		actionType = action.Update
	}

	return opts.Primitive(change.Before, change.After, ctype, actionType), nil
}

func (opts JSONOpts) processArray(change structured.ChangeSlice) (computed.Diff, error) {
	processIndices := func(beforeIx, afterIx int) (computed.Diff, error) {
		// It's actually really difficult to render the diffs when some indices
		// within a list are relevant and others aren't. To make this simpler
		// we just treat all children of a relevant list as also relevant, so we
		// ignore the relevant attributes field.
		//
		// Interestingly the terraform plan builder also agrees with this, and
		// never sets relevant attributes beneath lists or sets. We're just
		// going to enforce this logic here as well. If the list is relevant
		// (decided elsewhere), then every element in the list is also relevant.

		child, err := change.GetChild(beforeIx, afterIx)
		if err != nil {
			return computed.Diff{}, err
		}
		return opts.Transform(child)
	}

	isObjType := func(value interface{}) (bool, error) {
		typ, err := GetType(value)
		if err != nil {
			return false, err
		}
		return typ == Object, nil
	}

	transformedSlice, action, err := collections.TransformSlice(change.Before, change.After, processIndices, isObjType)
	if err != nil {
		return computed.Diff{}, err
	}

	return opts.Array(transformedSlice, action), nil
}

func (opts JSONOpts) processObject(change structured.ChangeMap) (computed.Diff, error) {
	transformedMap, action, err := collections.TransformMap(change.Before, change.After, change.AllKeys(), func(key string) (computed.Diff, error) {
		child := change.GetChild(key)
		if !child.RelevantAttributes.MatchesPartial() {
			child = child.AsNoOp()
		}

		return opts.Transform(child)
	})
	if err != nil {
		return computed.Diff{}, err
	}

	return opts.Object(transformedMap, action), nil
}

func (opts JSONOpts) processUnknown(change structured.Change) (computed.Diff, bool, error) {
	return change.CheckForUnknown(
		false,
		func(_ structured.Change) (computed.Diff, error) {
			return opts.Unknown(computed.Diff{}, action.Create), nil
		}, func(_ structured.Change, before structured.Change) (computed.Diff, error) {
			transformedDiff, err := opts.Transform(before)
			if err != nil {
				return computed.Diff{}, err
			}
			return opts.Unknown(transformedDiff, action.Update), nil
		},
	)
}

func (opts JSONOpts) processSensitive(change structured.Change) (computed.Diff, bool, error) {
	return change.CheckForSensitive(opts.Transform, func(inner computed.Diff, beforeSensitive, afterSensitive bool, action action.Action) computed.Diff {
		return opts.Sensitive(inner, beforeSensitive, afterSensitive, action)
	})
}
