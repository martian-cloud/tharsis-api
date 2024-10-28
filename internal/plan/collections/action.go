// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package collections provides utilities for working with collections inside of plan
package collections

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
)

// CompareActions will compare current and next, and return plans.Update if they
// are different, and current if they are the same.
func CompareActions(current, next action.Action) action.Action {
	if next == action.NoOp {
		return current
	}

	if current != next {
		return action.Update
	}
	return current
}
