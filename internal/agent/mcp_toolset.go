package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/gollem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// inMemoryToolSet implements gollem.ToolSet by connecting to an MCP server
// over an in-memory transport. This avoids spawning a separate process.
type inMemoryToolSet struct {
	session       *mcp.ClientSession
	serverSession *mcp.ServerSession
}

// NewInMemoryToolSet creates a ToolSet backed by an in-memory MCP server.
func NewInMemoryToolSet(ctx context.Context, server *mcp.Server) (gollem.ToolSet, error) {
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ss, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MCP server: %w", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "tharsis-agent"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect MCP client: %w", err)
	}

	return &inMemoryToolSet{session: cs, serverSession: ss}, nil
}

// Specs implements gollem.ToolSet.
func (t *inMemoryToolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	resp, err := t.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	specs := make([]gollem.ToolSpec, len(resp.Tools))
	for i, tool := range resp.Tools {
		spec := gollem.ToolSpec{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  make(map[string]*gollem.Parameter),
		}
		if tool.InputSchema != nil {
			param, err := convertInputSchemaToParams(tool.InputSchema)
			if err != nil {
				return nil, fmt.Errorf("failed to convert schema for tool %q: %w", tool.Name, err)
			}
			spec.Parameters = param
		}
		specs[i] = spec
	}

	return specs, nil
}

// Run implements gollem.ToolSet.
func (t *inMemoryToolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	resp, err := t.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %q: %w", name, err)
	}

	return convertContentToResult(resp.Content), nil
}

// Close shuts down both sides of the in-memory connection.
func (t *inMemoryToolSet) Close() error {
	if t.session != nil {
		_ = t.session.Close()
	}
	if t.serverSession != nil {
		_ = t.serverSession.Close()
	}
	return nil
}

// convertInputSchemaToParams converts an MCP tool's InputSchema (JSON Schema)
// into gollem Parameter map for tool spec registration.
func convertInputSchemaToParams(schema any) (map[string]*gollem.Parameter, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(raw, &schemaMap); err != nil {
		return nil, err
	}

	params := make(map[string]*gollem.Parameter)

	props, _ := schemaMap["properties"].(map[string]any)
	for name, propSchema := range props {
		p, err := convertSchemaToParam(propSchema)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", name, err)
		}
		params[name] = p
	}

	if required, ok := schemaMap["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				if p, exists := params[s]; exists {
					p.Required = true
				}
			}
		}
	}

	return params, nil
}

func convertSchemaToParam(schema any) (*gollem.Parameter, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	p := &gollem.Parameter{}

	switch t := m["type"].(type) {
	case string:
		p.Type = gollem.ParameterType(t)
	case []any:
		for _, v := range t {
			if s, ok := v.(string); ok && s != "null" {
				p.Type = gollem.ParameterType(s)
				break
			}
		}
	}

	if desc, ok := m["description"].(string); ok {
		p.Description = desc
	}

	if p.Type == gollem.TypeObject {
		if props, ok := m["properties"].(map[string]any); ok {
			p.Properties = make(map[string]*gollem.Parameter)
			for name, propSchema := range props {
				nested, err := convertSchemaToParam(propSchema)
				if err != nil {
					return nil, fmt.Errorf("property %q: %w", name, err)
				}
				p.Properties[name] = nested
			}
		}
		if required, ok := m["required"].([]any); ok {
			for _, r := range required {
				if s, ok := r.(string); ok {
					if prop, exists := p.Properties[s]; exists {
						prop.Required = true
					}
				}
			}
		}
	}

	if p.Type == gollem.TypeArray {
		if items, ok := m["items"]; ok {
			itemParam, err := convertSchemaToParam(items)
			if err != nil {
				return nil, err
			}
			p.Items = itemParam
		}
	}

	if enumVal, ok := m["enum"].([]any); ok {
		for _, e := range enumVal {
			p.Enum = append(p.Enum, fmt.Sprintf("%v", e))
		}
	}

	return p, nil
}

// convertContentToResult converts MCP Content to map[string]any for gollem.
func convertContentToResult(contents []mcp.Content) map[string]any {
	if len(contents) == 0 {
		return nil
	}

	if len(contents) == 1 {
		if tc, ok := contents[0].(*mcp.TextContent); ok {
			var v any
			if err := json.Unmarshal([]byte(tc.Text), &v); err == nil {
				if m, ok := v.(map[string]any); ok {
					return m
				}
			}
			return map[string]any{"result": tc.Text}
		}
		return nil
	}

	result := map[string]any{}
	for i, c := range contents {
		if tc, ok := c.(*mcp.TextContent); ok {
			result[fmt.Sprintf("content_%d", i+1)] = tc.Text
		}
	}
	return result
}
