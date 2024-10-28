// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package computed

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed/node"
)

// Diff captures the computed diff for a single block, element or attribute.
//
// It essentially merges common functionality across all types of changes,
// namely the replace logic and the action / change type. Any remaining
// behaviour can be offloaded to the renderer which will be unique for the
// various change types (eg. maps, objects, lists, blocks, primitives, etc.).
type Diff struct {
	// Renderer captures the uncommon functionality across the different kinds
	// of changes. Each type of change (lists, blocks, sets, etc.) will have a
	// unique renderer.
	Renderer DiffRenderer

	// Action is the action described by this change (such as create, delete,
	// update, etc.).
	Action action.Action

	// Replace tells the Change that it should add the `# forces replacement`
	// suffix.
	//
	// Every single change could potentially add this suffix, so we embed it in
	// the change as common functionality instead of in the specific renderers.
	Replace bool
}

// NewDiff creates a new Diff object with the provided renderer, action and
// replace context.
func NewDiff(renderer DiffRenderer, action action.Action, replace bool) Diff {
	return Diff{
		Renderer: renderer,
		Action:   action,
		Replace:  replace,
	}
}

// Render returns the rendered diff for the given change.
func (diff Diff) Render() (node.Diff, error) {
	return diff.Renderer.Render(diff)
}

// Warnings returns a list of strings that should be rendered as warnings
// before a given change is node.
func (diff Diff) Warnings() []string {
	return diff.Renderer.Warnings(diff)
}

// DiffRenderer is an interface that must be implemented by all diff renderers.
type DiffRenderer interface {
	Render(diff Diff) (node.Diff, error)
	Warnings(diff Diff) []string
}
