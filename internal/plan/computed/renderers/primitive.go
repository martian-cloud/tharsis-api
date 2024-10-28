// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured/attributepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"
)

var _ computed.DiffRenderer = (*primitiveRenderer)(nil)

// Primitive is a renderer for a primitive diff
func Primitive(before, after interface{}, ctype cty.Type) computed.DiffRenderer {
	return &primitiveRenderer{
		before: before,
		after:  after,
		ctype:  ctype,
	}
}

type primitiveRenderer struct {
	NoWarningsRenderer

	before interface{}
	after  interface{}
	ctype  cty.Type
}

func (renderer primitiveRenderer) Render(diff computed.Diff) (node.Diff, error) {
	var err error
	if renderer.ctype == cty.String {
		return renderer.renderStringDiffModel(diff)
	}

	var beforeValue, afterValue node.Diff

	switch diff.Action {
	case action.Create:
		afterValue, err = renderPrimitiveModelValue(renderer.after, renderer.ctype, diff)
		if err != nil {
			return nil, err
		}
	case action.Delete:
		beforeValue, err = renderPrimitiveModelValue(renderer.before, renderer.ctype, diff)
		if err != nil {
			return nil, err
		}
	default:
		beforeValue, err = renderPrimitiveModelValue(renderer.before, renderer.ctype, diff)
		if err != nil {
			return nil, err
		}
		afterValue, err = renderPrimitiveModelValue(renderer.after, renderer.ctype, diff)
		if err != nil {
			return nil, err
		}
	}

	return node.NewPrimitiveDiff(diff.Action, diff.Replace, diff.Warnings(), beforeValue, afterValue), nil
}

func renderPrimitiveModelValue(value interface{}, t cty.Type, diff computed.Diff) (node.Diff, error) {
	if value == nil {
		return node.NewNullValueDiff(diff.Action, diff.Replace), nil
	}

	switch {
	case t == cty.Bool:
		return node.NewBoolValueDiff(value.(bool), diff.Action, diff.Replace), nil
	case t == cty.Number:
		return node.NewNumberValueDiff(value.(float64), diff.Action, diff.Replace), nil
	default:
		return nil, fmt.Errorf("unrecognized primitive type: %s", t.FriendlyName())
	}
}

func (renderer primitiveRenderer) renderStringDiffModel(diff computed.Diff) (node.Diff, error) {
	switch diff.Action {
	case action.Create, action.NoOp:
		str := evaluatePrimitiveString(renderer.after)

		if str.JSON != nil {
			if diff.Action == action.NoOp {
				return renderer.renderStringDiffAsJSONModel(diff, str, str)
			}
			return renderer.renderStringDiffAsJSONModel(diff, evaluatedString{}, str)
		}

		return node.NewPrimitiveDiff(diff.Action, diff.Replace, diff.Warnings(), nil, node.NewStringValueDiff(str.String, diff.Action, diff.Replace, str.IsMultiline)), nil
	case action.Delete:
		str := evaluatePrimitiveString(renderer.before)
		if str.IsNull {
			return node.NewPrimitiveDiff(diff.Action, diff.Replace, diff.Warnings(), node.NewStringValueDiff(str.String, diff.Action, diff.Replace, str.IsMultiline), nil), nil
		}

		if str.JSON != nil {
			return renderer.renderStringDiffAsJSONModel(diff, str, evaluatedString{})
		}

		return node.NewPrimitiveDiff(diff.Action, diff.Replace, diff.Warnings(), node.NewStringValueDiff(str.String, diff.Action, diff.Replace, str.IsMultiline), nil), nil
	default:
		beforeString := evaluatePrimitiveString(renderer.before)
		afterString := evaluatePrimitiveString(renderer.after)

		if beforeString.JSON != nil && afterString.JSON != nil {
			return renderer.renderStringDiffAsJSONModel(diff, beforeString, afterString)
		}

		if beforeString.JSON != nil || afterString.JSON != nil {
			// This means one of the strings is JSON and one isn't. We're going
			// to be a little inefficient here, but we can just reuse another
			// renderer for this so let's keep it simple.
			return computed.NewDiff(
				TypeChange(
					computed.NewDiff(Primitive(renderer.before, nil, cty.String), action.Delete, false),
					computed.NewDiff(Primitive(nil, renderer.after, cty.String), action.Create, false)),
				diff.Action,
				diff.Replace).Render()
		}

		return node.NewPrimitiveDiff(
			diff.Action,
			diff.Replace,
			diff.Warnings(),
			node.NewStringValueDiff(beforeString.String, diff.Action, diff.Replace, beforeString.IsMultiline),
			node.NewStringValueDiff(afterString.String, diff.Action, diff.Replace, afterString.IsMultiline),
		), nil
	}
}

func (renderer primitiveRenderer) renderStringDiffAsJSONModel(diff computed.Diff, before evaluatedString, after evaluatedString) (node.Diff, error) {
	jsonDiff, err := RendererJSONOpts().Transform(structured.Change{
		BeforeExplicit:     diff.Action != action.Create,
		AfterExplicit:      diff.Action != action.Delete,
		Before:             before.JSON,
		After:              after.JSON,
		Unknown:            false,
		BeforeSensitive:    false,
		AfterSensitive:     false,
		ReplacePaths:       attributepath.Empty(false),
		RelevantAttributes: attributepath.AlwaysMatcher(),
	})
	if err != nil {
		return nil, err
	}

	jsonDiffModel, err := jsonDiff.Render()
	if err != nil {
		return nil, err
	}

	return node.NewJSONStringDiff(diff.Action, diff.Replace, diff.Warnings(), jsonDiffModel, jsonDiff.Action == action.NoOp && diff.Action == action.Update), nil
}
