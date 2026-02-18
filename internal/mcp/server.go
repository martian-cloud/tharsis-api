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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/mcp"
)

// ServerOptions holds configuration for creating an MCP server.
type ServerOptions struct {
	ServicesCatalog *services.Catalog
	Config          *config.MCPServerConfig
	HTTPClient      *http.Client
	Logger          logger.Logger
	Version         string
}

// NewStreamableHTTPHandler creates a new MCP server with Streamable HTTP transport.
func NewStreamableHTTPHandler(opts *ServerOptions) (http.Handler, error) {
	server, err := newServer(opts)
	if err != nil {
		return nil, err
	}

	return mcpsdk.NewStreamableHTTPHandler(func(_ *http.Request) *mcpsdk.Server {
		return server
	}, &mcpsdk.StreamableHTTPOptions{
		// Stateless mode allows the server to run across multiple instances without shared session storage.
		Stateless: true,
	}), nil
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
		Logger:          opts.Logger.Slog(),
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
