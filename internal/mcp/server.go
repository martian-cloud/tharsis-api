// Package mcp provides the MCP functionality for the API
package mcp

import (
	"fmt"
	"net/http"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/mcp/tools"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/mcp"
)

// ServerOptions holds configuration for creating an MCP server.
type ServerOptions struct {
	ServicesCatalog *services.Catalog
	Config          *config.MCPServerConfig
	HTTPClient      *http.Client
	Version         string
}

// NewSSEHandler creates a new MCP server with the provided options and returns an SSE HTTP handler.
func NewSSEHandler(opts *ServerOptions) (http.Handler, error) {
	server, err := newServer(opts)
	if err != nil {
		return nil, err
	}

	return mcpsdk.NewSSEHandler(func(_ *http.Request) *mcpsdk.Server {
		return server
	}, nil), nil
}

func newServer(opts *ServerOptions) (*mcpsdk.Server, error) {
	toolContext := tools.NewToolContext(opts.ServicesCatalog, opts.HTTPClient)
	toolsetGroup, err := tools.BuildToolsetGroup(opts.Config.ReadOnly, toolContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build toolset group: %w", err)
	}

	// Enable all toolsets by default if none are specified.
	enabledToolsets := opts.Config.EnabledToolsets
	if enabledToolsets == "" {
		enabledToolsets = strings.Join(tools.AllToolsets(), ",")
	}

	server, err := mcp.NewServer(&mcp.ServerConfig{
		Name:            "tharsis-api",
		Title:           "Tharsis API MCP Server",
		Version:         opts.Version,
		Instructions:    mcp.DefaultInstructions(),
		EnabledToolsets: enabledToolsets,
		EnabledTools:    opts.Config.EnabledTools,
		ReadOnly:        opts.Config.ReadOnly,
	}, toolsetGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return server, nil
}
