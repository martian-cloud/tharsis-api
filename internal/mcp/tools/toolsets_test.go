package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
)

func TestBuildToolsetGroup(t *testing.T) {
	type testCase struct {
		name     string
		readOnly bool
	}

	tests := []testCase{
		{
			name:     "read-only mode",
			readOnly: true,
		},
		{
			name:     "read-write mode",
			readOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &ToolContext{servicesCatalog: &services.Catalog{}}
			group, err := BuildToolsetGroup(tt.readOnly, tc)

			require.NoError(t, err)
			assert.NotNil(t, group)
			assert.True(t, group.HasToolsets())

			for _, toolsetName := range AllToolsets() {
				err := group.EnableToolsets(toolsetName)
				assert.NoError(t, err, "toolset %s should exist", toolsetName)
			}
		})
	}
}

func TestToolsetMetadata(t *testing.T) {
	type testCase struct {
		name         string
		metadataName string
		description  string
	}

	tests := []testCase{
		{"applies", ToolsetMetadataApplies.Name, ToolsetMetadataApplies.Description},
		{"documentation", ToolsetMetadataDocumentation.Name, ToolsetMetadataDocumentation.Description},
		{"jobs", ToolsetMetadataJobs.Name, ToolsetMetadataJobs.Description},
		{"plans", ToolsetMetadataPlans.Name, ToolsetMetadataPlans.Description},
		{"runs", ToolsetMetadataRuns.Name, ToolsetMetadataRuns.Description},
		{"workspaces", ToolsetMetadataWorkspaces.Name, ToolsetMetadataWorkspaces.Description},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.metadataName)
			assert.NotEmpty(t, tt.description)
		})
	}
}

func TestAllToolsets(t *testing.T) {
	toolsets := AllToolsets()

	assert.Len(t, toolsets, 6)
	assert.Contains(t, toolsets, "apply")
	assert.Contains(t, toolsets, "documentation")
	assert.Contains(t, toolsets, "job")
	assert.Contains(t, toolsets, "plan")
	assert.Contains(t, toolsets, "run")
	assert.Contains(t, toolsets, "workspace")
}
