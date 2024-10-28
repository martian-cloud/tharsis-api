// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/jsondiff"
)

// RendererJSONOpts creates a jsondiff.JSONOpts object that returns the correct
// embedded renderers for each JSON type.
//
// We need to define this in our renderers package in order to avoid cycles, and
// to help with reuse between the output processing in the differs package, and
// our JSON string rendering here.
func RendererJSONOpts() jsondiff.JSONOpts {
	return jsondiff.JSONOpts{
		Primitive: func(before, after interface{}, ctype cty.Type, action action.Action) computed.Diff {
			return computed.NewDiff(Primitive(before, after, ctype), action, false)
		},
		Object: func(elements map[string]computed.Diff, action action.Action) computed.Diff {
			return computed.NewDiff(Object(elements), action, false)
		},
		Array: func(elements []computed.Diff, action action.Action) computed.Diff {
			return computed.NewDiff(List(elements), action, false)
		},
		Unknown: func(diff computed.Diff, action action.Action) computed.Diff {
			return computed.NewDiff(Unknown(diff), action, false)
		},
		Sensitive: func(diff computed.Diff, beforeSensitive bool, afterSensitive bool, action action.Action) computed.Diff {
			return computed.NewDiff(Sensitive(diff, beforeSensitive, afterSensitive), action, false)
		},
		TypeChange: func(before, after computed.Diff, action action.Action) computed.Diff {
			return computed.NewDiff(TypeChange(before, after), action, false)
		},
	}
}
