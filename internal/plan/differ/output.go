// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/structured"

	"github.com/zclconf/go-cty/cty"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/renderers"
)

// ComputeDiffForOutput computes the diff for the given change.
func ComputeDiffForOutput(change structured.Change) (computed.Diff, error) {
	sensitive, ok, err := checkForSensitiveType(change, cty.DynamicPseudoType)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return sensitive, nil
	}

	unknown, ok, err := checkForUnknownType(change, cty.DynamicPseudoType)
	if err != nil {
		return computed.Diff{}, err
	}
	if ok {
		return unknown, nil
	}

	jsonOpts := renderers.RendererJSONOpts()
	return jsonOpts.Transform(change)
}
