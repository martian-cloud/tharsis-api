// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package action provider action constants and utilities.
package action

import (
	"fmt"
	"strings"

	tjson "github.com/hashicorp/terraform-json"
)

// Action represents the action that was taken on a resource.
type Action string

// Action constants
const (
	NoOp             Action = "NOOP"
	Create           Action = "CREATE"
	Read             Action = "READ"
	Update           Action = "UPDATE"
	DeleteThenCreate Action = "DELETE_THEN_CREATE"
	CreateThenDelete Action = "CREATE_THEN_DELETE"
	Delete           Action = "DELETE"
)

// IsReplace returns true if the action is one of the two actions that
// represents replacing an existing object with a new object:
// DeleteThenCreate or CreateThenDelete.
func (a Action) IsReplace() bool {
	return a == DeleteThenCreate || a == CreateThenDelete
}

// UnmarshalActions unmarshals a slice of actions into a single Action.
func UnmarshalActions(actions tjson.Actions) (Action, error) {
	if len(actions) == 2 {
		if actions[0] == "create" && actions[1] == "delete" {
			return CreateThenDelete, nil
		}

		if actions[0] == "delete" && actions[1] == "create" {
			return DeleteThenCreate, nil
		}
	}

	if len(actions) == 1 {
		switch actions[0] {
		case "create":
			return Create, nil
		case "delete":
			return Delete, nil
		case "update":
			return Update, nil
		case "read":
			return Read, nil
		case "no-op":
			return NoOp, nil
		}
	}

	actionValues := []string{}
	for _, action := range actions {
		actionValues = append(actionValues, string(action))
	}

	return "", fmt.Errorf("unrecognized action slice: %s", strings.Join(actionValues, ", "))
}
