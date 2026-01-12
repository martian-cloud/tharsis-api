// Package tools provides infrastructure for organizing and managing MCP tools.
//
// This file contains toolset metadata validation and parsing utilities.
package tools

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	// toolsetNamePattern enforces lowercase letters and underscores only, no leading or trailing underscores, 2-32 chars
	toolsetNamePattern = regexp.MustCompile(`^[a-z][a-z_]{0,30}[a-z]$`)
	// maxToolsetDescriptionLength is the maximum allowed length for a toolset description
	maxToolsetDescriptionLength = 200
)

// ToolsetMetadata holds metadata for a toolset.
type ToolsetMetadata struct {
	Name        string
	Description string
}

// validate checks if the toolset metadata is valid.
func (tm ToolsetMetadata) validate() error {
	if !toolsetNamePattern.MatchString(tm.Name) {
		return fmt.Errorf("toolset name %q must be 2-32 lowercase letters and underscores, no leading or trailing underscores", tm.Name)
	}
	if tm.Description == "" {
		return fmt.Errorf("toolset description cannot be empty for %q", tm.Name)
	}
	if len(tm.Description) > maxToolsetDescriptionLength {
		return fmt.Errorf("toolset description for %q exceeds maximum length of %d characters", tm.Name, maxToolsetDescriptionLength)
	}
	return nil
}

// ParseToolsets parses a comma-separated string of toolset names, trims whitespace, and removes duplicates.
// Returns cleaned toolsets and invalid toolsets that don't match the required format.
func ParseToolsets(toolsetsStr string) ([]string, []string) {
	if toolsetsStr == "" {
		return nil, nil
	}

	enabledToolsets := strings.Split(toolsetsStr, ",")
	result := make([]string, 0, len(enabledToolsets))
	invalid := make([]string, 0)

	for _, toolset := range enabledToolsets {
		trimmed := strings.TrimSpace(toolset)
		if trimmed == "" {
			continue
		}

		// Check format
		if !toolsetNamePattern.MatchString(trimmed) {
			invalid = append(invalid, trimmed)
			continue
		}

		result = append(result, trimmed)
	}

	// Remove duplicates while preserving order
	slices.Sort(result)
	result = slices.Compact(result)

	return result, invalid
}
