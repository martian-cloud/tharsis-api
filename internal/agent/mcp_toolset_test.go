package agent

import (
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertContentToResult_SingleText(t *testing.T) {
	contents := []mcp.Content{&mcp.TextContent{Text: "hello"}}
	result := convertContentToResult(contents)
	assert.Equal(t, map[string]any{"result": "hello"}, result)
}

func TestConvertContentToResult_SingleJSON(t *testing.T) {
	contents := []mcp.Content{&mcp.TextContent{Text: `{"key":"value"}`}}
	result := convertContentToResult(contents)
	assert.Equal(t, map[string]any{"key": "value"}, result)
}

func TestConvertContentToResult_Multiple(t *testing.T) {
	contents := []mcp.Content{
		&mcp.TextContent{Text: "first"},
		&mcp.TextContent{Text: "second"},
	}
	result := convertContentToResult(contents)
	assert.Equal(t, "first", result["content_1"])
	assert.Equal(t, "second", result["content_2"])
}

func TestConvertContentToResult_Empty(t *testing.T) {
	result := convertContentToResult(nil)
	assert.Nil(t, result)
}

func TestConvertInputSchemaToParams(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name",
			},
			"count": map[string]any{
				"type": "integer",
			},
		},
		"required": []any{"name"},
	}

	params, err := convertInputSchemaToParams(schema)
	require.Nil(t, err)

	require.Contains(t, params, "name")
	assert.Equal(t, gollem.TypeString, params["name"].Type)
	assert.True(t, params["name"].Required)
	assert.Equal(t, "The name", params["name"].Description)

	require.Contains(t, params, "count")
	assert.Equal(t, gollem.TypeInteger, params["count"].Type)
	assert.False(t, params["count"].Required)
}
