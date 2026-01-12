// Package tools provides the mcp tool implementations
package tools

import (
	"fmt"
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// ToolContext provides context for tool execution.
type ToolContext struct {
	servicesCatalog *services.Catalog
	httpClient      *http.Client
}

// NewToolContext creates a new tool context for executing MCP tools.
func NewToolContext(catalog *services.Catalog, httpClient *http.Client) *ToolContext {
	return &ToolContext{
		servicesCatalog: catalog,
		httpClient:      httpClient,
	}
}

// MCPToolError wraps errors for MCP tools with sanitized error messages.
type MCPToolError struct {
	message string
}

// Error implements the error interface.
func (e *MCPToolError) Error() string {
	return e.message
}

// NewMCPToolError creates a new MCPToolError with the given message.
func NewMCPToolError(format string, args ...any) error {
	return &MCPToolError{
		message: fmt.Sprintf(format, args...),
	}
}

// WrapMCPToolError wraps an error with a formatted message and sanitizes it.
func WrapMCPToolError(err error, format string, args ...any) error {
	prefix := fmt.Sprintf(format, args...)
	return &MCPToolError{
		message: fmt.Sprintf("%s: %s", prefix, errors.ErrorMessage(err)),
	}
}
