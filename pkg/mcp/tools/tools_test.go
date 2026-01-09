package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolset(t *testing.T) {
	metadata := ToolsetMetadata{
		Name:        "test",
		Description: "Test toolset",
	}

	toolset := NewToolset(metadata)

	assert.Equal(t, "test", toolset.Name())
	assert.Equal(t, "Test toolset", toolset.Description())
	assert.False(t, toolset.Enabled())
	assert.Empty(t, toolset.readTools)
	assert.Empty(t, toolset.writeTools)
	assert.Empty(t, toolset.prompts)
	assert.Empty(t, toolset.resources)
	assert.Empty(t, toolset.resourceTemplates)
}

func TestToolsetAddTools(t *testing.T) {
	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})

	readTool := ServerTool{
		tool: mcp.Tool{
			Name: "read_tool",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: true,
			},
		},
	}
	writeTool := ServerTool{
		tool: mcp.Tool{
			Name: "write_tool",
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint: false,
			},
		},
	}

	toolset.AddReadTools(readTool)
	toolset.AddWriteTools(writeTool)

	assert.Len(t, toolset.readTools, 1)
	assert.Len(t, toolset.writeTools, 1)
	assert.Equal(t, "read_tool", toolset.readTools[0].tool.Name)
	assert.Equal(t, "write_tool", toolset.writeTools[0].tool.Name)
}

func TestToolsetAddPrompts(t *testing.T) {
	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})

	prompt := ServerPrompt{
		prompt: mcp.Prompt{Name: "test_prompt"},
	}

	toolset.AddPrompts(prompt)

	assert.Len(t, toolset.prompts, 1)
	assert.Equal(t, "test_prompt", toolset.prompts[0].prompt.Name)
}

func TestNewToolsetGroup(t *testing.T) {
	type testCase struct {
		name     string
		readOnly bool
	}

	tests := []testCase{
		{
			name:     "read-only group",
			readOnly: true,
		},
		{
			name:     "read-write group",
			readOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group := NewToolsetGroup(tt.readOnly)
			assert.Equal(t, tt.readOnly, group.readOnly)
			assert.NotNil(t, group.Toolsets())
			assert.Empty(t, group.Toolsets())
		})
	}
}

func TestToolsetGroupAddToolset(t *testing.T) {
	group := NewToolsetGroup(false)

	toolset := NewToolset(ToolsetMetadata{
		Name:        "test",
		Description: "Test toolset",
	})

	group.AddToolset(toolset)

	toolsets := group.Toolsets()
	assert.Len(t, toolsets, 1)
	assert.Contains(t, toolsets, "test")
}

func TestToolsetGroupAddToolsetInvalidMetadata(t *testing.T) {
	assert.Panics(t, func() {
		NewToolset(ToolsetMetadata{
			Name:        "x", // Invalid: too short
			Description: "Test",
		})
	})
}

func TestToolsetGroupEnableToolsets(t *testing.T) {
	group := NewToolsetGroup(false)

	toolset1 := NewToolset(ToolsetMetadata{Name: "ts", Description: "Test 1"})
	toolset2 := NewToolset(ToolsetMetadata{Name: "tt", Description: "Test 2"})

	group.AddToolset(toolset1)
	group.AddToolset(toolset2)

	err := group.EnableToolsets("ts")
	require.NoError(t, err)

	toolsets := group.Toolsets()
	assert.True(t, toolsets["ts"].Enabled())
	assert.False(t, toolsets["tt"].Enabled())
}

func TestToolsetGroupEnableToolsetsNonExistent(t *testing.T) {
	group := NewToolsetGroup(false)

	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})
	group.AddToolset(toolset)

	err := group.EnableToolsets("nonexistent")
	require.Error(t, err)

	var toolsetErr *ToolsetDoesNotExistError
	assert.ErrorAs(t, err, &toolsetErr)
	assert.Equal(t, "nonexistent", toolsetErr.Name)
}

func TestToolsetGroupEnableMultipleToolsets(t *testing.T) {
	group := NewToolsetGroup(false)

	ts1 := NewToolset(ToolsetMetadata{Name: "toolset_one", Description: "Test 1"})
	ts2 := NewToolset(ToolsetMetadata{Name: "toolset_two", Description: "Test 2"})
	ts3 := NewToolset(ToolsetMetadata{Name: "toolset_three", Description: "Test 3"})

	group.AddToolset(ts1)
	group.AddToolset(ts2)
	group.AddToolset(ts3)

	err := group.EnableToolsets("toolset_one", "toolset_three")
	require.NoError(t, err)

	toolsets := group.Toolsets()
	assert.True(t, toolsets["toolset_one"].Enabled())
	assert.False(t, toolsets["toolset_two"].Enabled())
	assert.True(t, toolsets["toolset_three"].Enabled())
}

func TestToolsetGroupFindToolByName(t *testing.T) {
	type testCase struct {
		name        string
		toolName    string
		expectFound bool
		expectError bool
	}

	group := NewToolsetGroup(false)
	ts := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})
	ts.AddReadTools(NewServerTool(mcp.Tool{
		Name:        "read_tool",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
		return nil, struct{}{}, nil
	}))
	group.AddToolset(ts)

	tests := []testCase{
		{
			name:        "existing tool",
			toolName:    "read_tool",
			expectFound: true,
		},
		{
			name:        "non-existent tool",
			toolName:    "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, toolsetName, err := group.FindToolByName(tt.toolName)
			if tt.expectError {
				require.Error(t, err)
				var toolErr *ToolDoesNotExistError
				assert.ErrorAs(t, err, &toolErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tool)
				assert.Equal(t, "test", toolsetName)
			}
		})
	}
}

func TestToolsetGroupHasEnabledToolsets(t *testing.T) {
	group := NewToolsetGroup(false)

	assert.False(t, group.HasEnabledToolsets())

	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})
	group.AddToolset(toolset)

	assert.False(t, group.HasEnabledToolsets())

	group.EnableToolsets("test")

	assert.True(t, group.HasEnabledToolsets())
}

func TestToolsetDoesNotExistError(t *testing.T) {
	err := &ToolsetDoesNotExistError{Name: "test"}
	assert.Equal(t, "toolset test does not exist", err.Error())
}

func TestToolDoesNotExistError(t *testing.T) {
	err := &ToolDoesNotExistError{Name: "test_tool"}
	assert.Equal(t, "tool test_tool does not exist", err.Error())
}

func TestToolsetAddResources(t *testing.T) {
	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})

	resource := NewServerResource(
		mcp.Resource{URI: "test://resource"},
		func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return nil, nil
		},
	)

	toolset.AddResources(resource)

	assert.Len(t, toolset.resources, 1)
	assert.Equal(t, "test://resource", toolset.resources[0].resource.URI)
}

func TestToolsetRegisterResources(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test",
		Title:   "Test",
		Version: "1.0.0",
	}, nil)

	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})

	registered := false
	resource := ServerResource{
		resource: mcp.Resource{URI: "test://resource"},
		registerFunc: func(_ *mcp.Server) {
			registered = true
		},
	}
	toolset.AddResources(resource)

	// Should not register when disabled
	toolset.RegisterResources(server)
	assert.False(t, registered, "resource should not be registered when toolset is disabled")

	// Enable and register
	toolset.enabled = true
	toolset.RegisterResources(server)
	assert.True(t, registered, "resource should be registered when toolset is enabled")
}

func TestToolsetAddResourceTemplates(t *testing.T) {
	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})

	template := NewServerResourceTemplate(
		mcp.ResourceTemplate{URITemplate: "test://resource/{id}"},
		func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return nil, nil
		},
	)

	toolset.AddResourceTemplates(template)

	assert.Len(t, toolset.resourceTemplates, 1)
	assert.Equal(t, "test://resource/{id}", toolset.resourceTemplates[0].template.URITemplate)
}

func TestToolsetRegisterResourceTemplates(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test",
		Title:   "Test",
		Version: "1.0.0",
	}, nil)

	toolset := NewToolset(ToolsetMetadata{Name: "test", Description: "Test"})

	registered := false
	template := ServerResourceTemplate{
		template: mcp.ResourceTemplate{URITemplate: "test://resource/{id}"},
		registerFunc: func(_ *mcp.Server) {
			registered = true
		},
	}
	toolset.AddResourceTemplates(template)

	// Should not register when disabled
	toolset.RegisterResourceTemplates(server)
	assert.False(t, registered, "resource template should not be registered when toolset is disabled")

	// Enable and register
	toolset.enabled = true
	toolset.RegisterResourceTemplates(server)
	assert.True(t, registered, "resource template should be registered when toolset is enabled")
}
