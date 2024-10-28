// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package computed contains types that represent the computed diffs for
// Terraform blocks, attributes, and outputs.
//
// Each Diff struct is made up of a renderer, an action, and a boolean
// describing the diff. The renderer internally holds child diffs or concrete
// values that allow it to know how to render the diff appropriately.
package computed
