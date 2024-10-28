// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package collections

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
)

// ProcessKey is a callback function that processes a key
type ProcessKey func(key string) (computed.Diff, error)

// TransformMap transforms an input map into a computed.Diff and a action.Action
func TransformMap[Input any](before, after map[string]Input, keys []string, process ProcessKey) (map[string]computed.Diff, action.Action, error) {
	var err error

	current := action.NoOp
	if before != nil && after == nil {
		current = action.Delete
	}
	if before == nil && after != nil {
		current = action.Create
	}

	elements := make(map[string]computed.Diff)
	for _, key := range keys {
		elements[key], err = process(key)
		if err != nil {
			return nil, action.NoOp, err
		}
		current = CompareActions(current, elements[key].Action)
	}

	return elements, current, nil
}
