package mcp

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/mcp/tools"
)

func TestServerConfigValidate(t *testing.T) {
	type testCase struct {
		name        string
		config      *ServerConfig
		expectError string
	}

	tests := []testCase{
		{
			name: "valid config",
			config: &ServerConfig{
				Name:            "test-server",
				Title:           "Test Server",
				Version:         "1.0.0",
				EnabledToolsets: "test",
			},
			expectError: "",
		},
		{
			name: "empty name",
			config: &ServerConfig{
				Title:           "Test Server",
				Version:         "1.0.0",
				EnabledToolsets: "test",
			},
			expectError: "server name cannot be empty",
		},
		{
			name: "empty title",
			config: &ServerConfig{
				Name:            "test-server",
				Version:         "1.0.0",
				EnabledToolsets: "test",
			},
			expectError: "server title cannot be empty",
		},
		{
			name: "empty version",
			config: &ServerConfig{
				Name:            "test-server",
				Title:           "Test Server",
				EnabledToolsets: "test",
			},
			expectError: "server version cannot be empty",
		},
		{
			name: "no toolsets or tools enabled",
			config: &ServerConfig{
				Name:    "test-server",
				Title:   "Test Server",
				Version: "1.0.0",
			},
			expectError: "no toolsets or tools enabled",
		},
		{
			name: "enabled tools only",
			config: &ServerConfig{
				Name:         "test-server",
				Title:        "Test Server",
				Version:      "1.0.0",
				EnabledTools: "tool1,tool2",
			},
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	type testCase struct {
		name         string
		config       *ServerConfig
		toolsetGroup *tools.ToolsetGroup
		expectError  string
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tests := []testCase{
		{
			name: "nil toolset group",
			config: &ServerConfig{
				Name:            "test-server",
				Title:           "Test Server",
				Version:         "1.0.0",
				Logger:          logger,
				EnabledToolsets: "test",
			},
			toolsetGroup: nil,
			expectError:  "toolset group cannot be nil or empty",
		},
		{
			name: "empty toolset group",
			config: &ServerConfig{
				Name:            "test-server",
				Title:           "Test Server",
				Version:         "1.0.0",
				Logger:          logger,
				EnabledToolsets: "test",
			},
			toolsetGroup: tools.NewToolsetGroup(true),
			expectError:  "toolset group cannot be nil or empty",
		},
		{
			name: "invalid config",
			config: &ServerConfig{
				Title:           "Test Server",
				Version:         "1.0.0",
				Logger:          logger,
				EnabledToolsets: "test",
			},
			toolsetGroup: func() *tools.ToolsetGroup {
				group := tools.NewToolsetGroup(false)
				group.AddToolset(tools.NewToolset(tools.ToolsetMetadata{Name: "test", Description: "Test"}))
				return group
			}(),
			expectError: "server name cannot be empty",
		},
		{
			name: "no valid toolsets",
			config: &ServerConfig{
				Name:            "test-server",
				Title:           "Test Server",
				Version:         "1.0.0",
				Logger:          logger,
				EnabledToolsets: "nonexistent",
			},
			toolsetGroup: func() *tools.ToolsetGroup {
				group := tools.NewToolsetGroup(false)
				group.AddToolset(tools.NewToolset(tools.ToolsetMetadata{Name: "test", Description: "Test"}))
				return group
			}(),
			expectError: "failed to enable toolsets",
		},
		{
			name: "valid with enabled tools only",
			config: &ServerConfig{
				Name:         "test-server",
				Title:        "Test Server",
				Version:      "1.0.0",
				Logger:       logger,
				EnabledTools: "test_tool",
			},
			toolsetGroup: func() *tools.ToolsetGroup {
				group := tools.NewToolsetGroup(false)
				ts := tools.NewToolset(tools.ToolsetMetadata{Name: "test", Description: "Test"})
				ts.AddReadTools(tools.NewServerTool(mcp.Tool{
					Name:        "test_tool",
					Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
				}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
					return nil, struct{}{}, nil
				}))
				group.AddToolset(ts)
				return group
			}(),
			expectError: "",
		},
		{
			name: "no valid toolsets or tools",
			config: &ServerConfig{
				Name:            "test-server",
				Title:           "Test Server",
				Version:         "1.0.0",
				Logger:          logger,
				EnabledToolsets: "nonexistent",
				EnabledTools:    "nonexistent_tool",
			},
			toolsetGroup: func() *tools.ToolsetGroup {
				group := tools.NewToolsetGroup(false)
				group.AddToolset(tools.NewToolset(tools.ToolsetMetadata{Name: "test", Description: "Test"}))
				return group
			}(),
			expectError: "failed to enable toolsets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewServer(tt.config, tt.toolsetGroup)
			if tt.expectError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, server)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
				assert.Nil(t, server)
			}
		})
	}
}
