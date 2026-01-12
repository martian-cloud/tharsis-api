// Package tools provides infrastructure for organizing and managing MCP tools.
//
// Tools are organized into toolsets (logical groups) that can be enabled/disabled
// as a unit. This allows flexible configuration of which functionality is exposed
// via the MCP server.
//
// This code has been derived from the github-mcp-server project licensed under the MIT license.
// https://github.com/github/github-mcp-server/blob/82c493056edfd49a4e15d9cd0ce5908bd9b59e1a/pkg/github/tools.go
package tools

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolsetDoesNotExistError is returned when a toolset is not found.
type ToolsetDoesNotExistError struct {
	Name string
}

func (e *ToolsetDoesNotExistError) Error() string {
	return fmt.Sprintf("toolset %s does not exist", e.Name)
}

// ToolDoesNotExistError is returned when a tool is not found.
type ToolDoesNotExistError struct {
	Name string
}

func (e *ToolDoesNotExistError) Error() string {
	return fmt.Sprintf("tool %s does not exist", e.Name)
}

// ServerTool wraps an MCP tool with its registration function.
type ServerTool struct {
	tool         mcp.Tool
	registerFunc func(*mcp.Server)
}

// NewServerTool creates a ServerTool with type-safe handler.
func NewServerTool[In any, Out any](tool mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) ServerTool {
	return ServerTool{
		tool: tool,
		registerFunc: func(s *mcp.Server) {
			// use mcp.AddTool instead of s.AddTool() since mcp.AddTool does a lot
			// automatically, and forces tools to conform to the MCP spec.
			mcp.AddTool(s, &tool, handler)
		},
	}
}

// ServerResource wraps an MCP resource with its registration function.
//
// Resources expose read-only data that LLMs can access as context. Use resources when:
//   - The LLM needs to read/inspect data to answer questions or make decisions
//   - You have a fixed, enumerable set of resources to expose
//   - The resource URI is static (e.g., "config://settings", "docs://readme")
//
// For dynamic or parameterized resources, use ServerResourceTemplate instead.
type ServerResource struct {
	resource     mcp.Resource
	registerFunc func(*mcp.Server)
}

// NewServerResource creates a ServerResource with type-safe handler.
func NewServerResource(resource mcp.Resource, handler mcp.ResourceHandler) ServerResource {
	return ServerResource{
		resource: resource,
		registerFunc: func(s *mcp.Server) {
			s.AddResource(&resource, handler)
		},
	}
}

// ServerResourceTemplate wraps an MCP resource template with its registration function.
//
// Resource templates expose parameterized resources using URI templates (RFC 6570).
// Use resource templates when:
//   - You have a large or dynamic set of resources following a pattern
//   - Resources are identified by parameters (e.g., workspace name, file path)
//   - The LLM should construct resource URIs based on context
//
// Example: "tharsis://workspace/{workspace}/file/{path}" allows the LLM to read
// any file by filling in the parameters, rather than listing every possible file.
//
// Resources vs Tools:
//   - Resources: Read-only data retrieval for context (e.g., reading config files, logs)
//   - Tools: Actions that perform operations or have side effects (e.g., creating resources, triggering runs)
type ServerResourceTemplate struct {
	template     mcp.ResourceTemplate
	registerFunc func(*mcp.Server)
}

// NewServerResourceTemplate creates a ServerResourceTemplate with type-safe handler.
func NewServerResourceTemplate(template mcp.ResourceTemplate, handler mcp.ResourceHandler) ServerResourceTemplate {
	return ServerResourceTemplate{
		template: template,
		registerFunc: func(s *mcp.Server) {
			s.AddResourceTemplate(&template, handler)
		},
	}
}

// ServerPrompt wraps an MCP prompt with its handler.
type ServerPrompt struct {
	prompt  mcp.Prompt
	handler mcp.PromptHandler
}

// NewServerPrompt creates a ServerPrompt.
func NewServerPrompt(prompt mcp.Prompt, handler mcp.PromptHandler) ServerPrompt {
	return ServerPrompt{
		prompt:  prompt,
		handler: handler,
	}
}

// Toolset represents a collection of MCP functionality.
type Toolset struct {
	name              string
	description       string
	enabled           bool
	readOnly          bool
	readTools         []ServerTool
	writeTools        []ServerTool
	prompts           []ServerPrompt
	resources         []ServerResource
	resourceTemplates []ServerResourceTemplate
}

// Name returns the toolset name.
func (t *Toolset) Name() string {
	return t.name
}

// Description returns the toolset description.
func (t *Toolset) Description() string {
	return t.description
}

// Enabled returns whether the toolset is enabled.
func (t *Toolset) Enabled() bool {
	return t.enabled
}

// NewToolset creates a new toolset from metadata.
// Panics if metadata is invalid.
func NewToolset(metadata ToolsetMetadata) *Toolset {
	if err := metadata.validate(); err != nil {
		panic(fmt.Sprintf("invalid toolset metadata: %v", err))
	}
	return &Toolset{
		name:        metadata.Name,
		description: metadata.Description,
		enabled:     false,
		readOnly:    false,
	}
}

// SetReadOnly marks the toolset as read-only.
func (t *Toolset) SetReadOnly() {
	t.readOnly = true
}

// HasReadTools returns true if the toolset has any read-only tools.
func (t *Toolset) HasReadTools() bool {
	return len(t.readTools) > 0
}

// RegisterTools registers all enabled tools in the toolset.
func (t *Toolset) RegisterTools(s *mcp.Server) {
	if !t.enabled {
		return
	}
	for _, tool := range t.readTools {
		tool.registerFunc(s)
	}
	if !t.readOnly {
		for _, tool := range t.writeTools {
			tool.registerFunc(s)
		}
	}
}

// AddPrompts adds prompts to the toolset.
func (t *Toolset) AddPrompts(prompts ...ServerPrompt) *Toolset {
	t.prompts = append(t.prompts, prompts...)
	return t
}

// RegisterPrompts registers all enabled prompts in the toolset.
func (t *Toolset) RegisterPrompts(s *mcp.Server) {
	if !t.enabled {
		return
	}
	for _, prompt := range t.prompts {
		s.AddPrompt(&prompt.prompt, prompt.handler)
	}
}

// AddResources adds resources to the toolset.
func (t *Toolset) AddResources(resources ...ServerResource) *Toolset {
	t.resources = append(t.resources, resources...)
	return t
}

// RegisterResources registers all enabled resources in the toolset.
func (t *Toolset) RegisterResources(s *mcp.Server) {
	if !t.enabled {
		return
	}
	for _, resource := range t.resources {
		resource.registerFunc(s)
	}
}

// AddResourceTemplates adds resource templates to the toolset.
func (t *Toolset) AddResourceTemplates(templates ...ServerResourceTemplate) *Toolset {
	t.resourceTemplates = append(t.resourceTemplates, templates...)
	return t
}

// RegisterResourceTemplates registers all enabled resource templates in the toolset.
func (t *Toolset) RegisterResourceTemplates(s *mcp.Server) {
	if !t.enabled {
		return
	}
	for _, template := range t.resourceTemplates {
		template.registerFunc(s)
	}
}

// AddWriteTools adds write tools to the toolset.
func (t *Toolset) AddWriteTools(tools ...ServerTool) *Toolset {
	for _, tool := range tools {
		if tool.tool.Annotations.ReadOnlyHint {
			panic(fmt.Sprintf("tool (%s) is incorrectly annotated as read-only", tool.tool.Name))
		}
	}
	if !t.readOnly {
		t.writeTools = append(t.writeTools, tools...)
	}
	return t
}

// AddReadTools adds read-only tools to the toolset.
func (t *Toolset) AddReadTools(tools ...ServerTool) *Toolset {
	for _, tool := range tools {
		if !tool.tool.Annotations.ReadOnlyHint {
			panic(fmt.Sprintf("tool (%s) must be annotated as read-only", tool.tool.Name))
		}
	}
	t.readTools = append(t.readTools, tools...)
	return t
}

// ToolsetGroup manages toolsets and tool registration.
type ToolsetGroup struct {
	toolsets map[string]*Toolset
	readOnly bool
}

// Toolsets returns a copy of the toolsets map.
func (tg *ToolsetGroup) Toolsets() map[string]*Toolset {
	result := make(map[string]*Toolset, len(tg.toolsets))
	for k, v := range tg.toolsets {
		result[k] = v
	}
	return result
}

// NewToolsetGroup creates an empty toolset group.
func NewToolsetGroup(readOnly bool) *ToolsetGroup {
	return &ToolsetGroup{
		toolsets: make(map[string]*Toolset),
		readOnly: readOnly,
	}
}

// AddToolset adds a toolset to the group.
func (tg *ToolsetGroup) AddToolset(ts *Toolset) {
	if tg.readOnly {
		ts.SetReadOnly()
	}
	tg.toolsets[ts.name] = ts
}

// EnableToolsets enables the specified toolsets by name.
func (tg *ToolsetGroup) EnableToolsets(names ...string) error {
	for _, name := range names {
		toolset, exists := tg.toolsets[name]
		if !exists {
			return &ToolsetDoesNotExistError{Name: name}
		}

		// Skip toolsets with no read tools in read-only mode
		if tg.readOnly && !toolset.HasReadTools() {
			continue
		}

		toolset.enabled = true
	}
	return nil
}

// RegisterAll registers all enabled toolsets with the MCP server.
func (tg *ToolsetGroup) RegisterAll(s *mcp.Server) {
	for _, toolset := range tg.toolsets {
		toolset.RegisterTools(s)
		toolset.RegisterPrompts(s)
		toolset.RegisterResources(s)
		toolset.RegisterResourceTemplates(s)
	}
}

// HasEnabledToolsets returns true if any toolsets are enabled.
func (tg *ToolsetGroup) HasEnabledToolsets() bool {
	for _, toolset := range tg.toolsets {
		if toolset.enabled {
			return true
		}
	}
	return false
}

// HasToolsets returns true if the group has any toolsets.
func (tg *ToolsetGroup) HasToolsets() bool {
	return len(tg.toolsets) > 0
}

// FindToolByName searches all toolsets (enabled or disabled) for a tool by name.
// Returns the tool, its parent toolset name, and an error if not found.
func (tg *ToolsetGroup) FindToolByName(toolName string) (*ServerTool, string, error) {
	for toolsetName, toolset := range tg.toolsets {
		for _, tool := range toolset.readTools {
			if tool.tool.Name == toolName {
				return &tool, toolsetName, nil
			}
		}
		for _, tool := range toolset.writeTools {
			if tool.tool.Name == toolName {
				return &tool, toolsetName, nil
			}
		}
	}

	return nil, "", &ToolDoesNotExistError{Name: toolName}
}

// RegisterSpecificTools registers only the specified tools.
// Respects read-only mode (skips write tools if readOnly=true).
// Returns error if any tool is not found.
func (tg *ToolsetGroup) RegisterSpecificTools(s *mcp.Server, toolNames []string, readOnly bool) error {
	var skippedTools []string

	for _, toolName := range toolNames {
		tool, _, err := tg.FindToolByName(toolName)
		if err != nil {
			return err
		}

		if !tool.tool.Annotations.ReadOnlyHint && readOnly {
			skippedTools = append(skippedTools, toolName)
			continue
		}

		tool.registerFunc(s)
	}

	if len(skippedTools) > 0 {
		fmt.Fprintf(os.Stderr, "Write tools skipped due to read-only mode: %s\n", strings.Join(skippedTools, ", "))
	}

	return nil
}

// ParseTools parses a comma-separated string of tool names, trims whitespace, and removes duplicates.
// Validation of tool existence is done during registration.
func ParseTools(toolsStr string) []string {
	if toolsStr == "" {
		return nil
	}

	toolNames := strings.Split(toolsStr, ",")
	result := make([]string, 0, len(toolNames))

	for _, tool := range toolNames {
		trimmed := strings.TrimSpace(tool)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	// Remove duplicates while preserving order
	slices.Sort(result)
	return slices.Compact(result)
}
