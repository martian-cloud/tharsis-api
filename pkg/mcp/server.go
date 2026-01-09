// Package mcp provides reusable infrastructure for building Model Context Protocol (MCP) servers.
//
// This package enables applications to expose functionality to LLMs via the MCP protocol.
// It provides server configuration, tool management, and flexible toolset organization
// inspired by GitHub's MCP server architecture.
package mcp

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/mcp/tools"
)

// ServerConfig holds configuration for creating an MCP server.
type ServerConfig struct {
	// Name is the server implementation name
	Name string
	// Title is the human-readable server title
	Title string
	// Version is the server version
	Version string
	// Logger is the structured logger for server operations
	Logger *slog.Logger
	// Instructions are the server-level instructions for the MCP client
	Instructions string
	// EnabledToolsets is a comma-separated list of toolset names to enable
	EnabledToolsets string
	// EnabledTools is a comma-separated list of specific tool names to enable (overrides toolsets)
	EnabledTools string
	// ReadOnly indicates whether to skip registering write tools
	ReadOnly bool
}

// Validate checks if the server configuration is valid.
func (cfg *ServerConfig) Validate() error {
	if cfg.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}
	if cfg.Title == "" {
		return fmt.Errorf("server title cannot be empty")
	}
	if cfg.Version == "" {
		return fmt.Errorf("server version cannot be empty")
	}
	if cfg.EnabledToolsets == "" && cfg.EnabledTools == "" {
		return fmt.Errorf("no toolsets or tools enabled")
	}
	return nil
}

// NewServer creates a new MCP server with the provided configuration and toolset group.
// The toolset group should be provided by the consumer with their specific tools.
func NewServer(cfg *ServerConfig, toolsetGroup *tools.ToolsetGroup) (*mcp.Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if toolsetGroup == nil || !toolsetGroup.HasToolsets() {
		return nil, fmt.Errorf("toolset group cannot be nil or empty")
	}

	enabledToolsets, invalidToolsets := tools.ParseToolsets(cfg.EnabledToolsets)

	if len(invalidToolsets) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: invalid toolsets: %v\n", invalidToolsets)
	}

	if len(enabledToolsets) > 0 {
		if err := toolsetGroup.EnableToolsets(enabledToolsets...); err != nil {
			return nil, fmt.Errorf("failed to enable toolsets: %w", err)
		}
	}

	enabledTools := tools.ParseTools(cfg.EnabledTools)

	// Check if any toolsets or tools will actually be registered
	if !toolsetGroup.HasEnabledToolsets() && len(enabledTools) == 0 {
		return nil, fmt.Errorf("no valid toolsets or tools to register")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Name,
		Title:   cfg.Title,
		Version: cfg.Version,
	}, &mcp.ServerOptions{
		Logger:       cfg.Logger,
		Instructions: cfg.Instructions,
	})

	if len(enabledToolsets) > 0 {
		toolsetGroup.RegisterAll(server)
	}

	if len(enabledTools) > 0 {
		if err := toolsetGroup.RegisterSpecificTools(server, enabledTools, cfg.ReadOnly); err != nil {
			return nil, fmt.Errorf("failed to register tools: %w", err)
		}
	}

	return server, nil
}
