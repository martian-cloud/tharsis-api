// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package differ

// NestingMode is a wrapper around a string type to describe the various
// different kinds of nesting modes that can be applied to nested blocks and
// objects.
type NestingMode string

const (
	nestingModeSet    NestingMode = "set"
	nestingModeList   NestingMode = "list"
	nestingModeMap    NestingMode = "map"
	nestingModeSingle NestingMode = "single"
	nestingModeGroup  NestingMode = "group"
)
