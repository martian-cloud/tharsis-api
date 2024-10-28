// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package structured contains the structured representation of the JSON changes
// returned by the tjson package.
//
// Placing these in a dedicated package allows for greater reuse across the
// various type of renderers.
package structured
