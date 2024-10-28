// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"sort"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
)

// Blocks is a helper struct for collating the different kinds of blocks in a
// simple way for rendering.
type Blocks struct {
	SingleBlocks map[string]computed.Diff
	ListBlocks   map[string][]computed.Diff
	SetBlocks    map[string][]computed.Diff
	MapBlocks    map[string]map[string]computed.Diff

	// ReplaceBlocks and Before/AfterSensitiveBlocks carry forward the
	// information about an entire group of blocks (eg. if all the blocks for a
	// given list block are sensitive that isn't captured in the individual
	// blocks as they are processed independently). These maps allow the
	// renderer to check the metadata on the overall groups and respond
	// accordingly.

	ReplaceBlocks         map[string]bool
	BeforeSensitiveBlocks map[string]bool
	AfterSensitiveBlocks  map[string]bool
}

// GetAllKeys returns a list of keys for the blocks
func (blocks *Blocks) GetAllKeys() []string {
	var keys []string
	for key := range blocks.SingleBlocks {
		keys = append(keys, key)
	}
	for key := range blocks.ListBlocks {
		keys = append(keys, key)
	}
	for key := range blocks.SetBlocks {
		keys = append(keys, key)
	}
	for key := range blocks.MapBlocks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// IsSingleBlock returns true if the key is for a single block
func (blocks *Blocks) IsSingleBlock(key string) bool {
	_, ok := blocks.SingleBlocks[key]
	return ok
}

// IsListBlock returns true if the key is for a list block
func (blocks *Blocks) IsListBlock(key string) bool {
	_, ok := blocks.ListBlocks[key]
	return ok
}

// IsMapBlock returns true if the key is for a map block
func (blocks *Blocks) IsMapBlock(key string) bool {
	_, ok := blocks.MapBlocks[key]
	return ok
}

// IsSetBlock returns true if the key is for a set block
func (blocks *Blocks) IsSetBlock(key string) bool {
	_, ok := blocks.SetBlocks[key]
	return ok
}

// AddSingleBlock adds a single block
func (blocks *Blocks) AddSingleBlock(key string, diff computed.Diff, replace, beforeSensitive, afterSensitive bool) {
	blocks.SingleBlocks[key] = diff
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
}

// AddAllListBlock adds a list of block diffs
func (blocks *Blocks) AddAllListBlock(key string, diffs []computed.Diff, replace, beforeSensitive, afterSensitive bool) {
	blocks.ListBlocks[key] = diffs
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
}

// AddAllSetBlock adds a set of block diffs
func (blocks *Blocks) AddAllSetBlock(key string, diffs []computed.Diff, replace, beforeSensitive, afterSensitive bool) {
	blocks.SetBlocks[key] = diffs
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
}

// AddAllMapBlocks adds a map of block diffs
func (blocks *Blocks) AddAllMapBlocks(key string, diffs map[string]computed.Diff, replace, beforeSensitive, afterSensitive bool) {
	blocks.MapBlocks[key] = diffs
	blocks.ReplaceBlocks[key] = replace
	blocks.BeforeSensitiveBlocks[key] = beforeSensitive
	blocks.AfterSensitiveBlocks[key] = afterSensitive
}
