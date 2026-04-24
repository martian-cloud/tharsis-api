package bedrock

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	smithydoc "github.com/aws/smithy-go/document"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertParametersToSchema_WithParams(t *testing.T) {
	params := map[string]*gollem.Parameter{
		"name":  {Type: gollem.TypeString, Description: "The name", Required: true},
		"count": {Type: gollem.TypeInteger, Description: "A count"},
	}

	schema := convertParametersToSchema(params)
	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, false, schema["additionalProperties"])

	props := schema["properties"].(map[string]any)
	require.Contains(t, props, "name")
	require.Contains(t, props, "count")

	nameSchema := props["name"].(map[string]any)
	assert.Equal(t, "string", nameSchema["type"])
	assert.Equal(t, "The name", nameSchema["description"])

	required := schema["required"].([]string)
	assert.Contains(t, required, "name")
}

func TestConvertParametersToSchema_Empty(t *testing.T) {
	schema := convertParametersToSchema(nil)
	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, map[string]any{}, schema["properties"])
}

func TestConvertParameterToSchema_Array(t *testing.T) {
	param := &gollem.Parameter{
		Type:  gollem.TypeArray,
		Items: &gollem.Parameter{Type: gollem.TypeString},
	}

	schema := convertParameterToSchema(param)
	assert.Equal(t, "array", schema["type"])
	items := schema["items"].(map[string]any)
	assert.Equal(t, "string", items["type"])
}

func TestConvertParameterToSchema_Object(t *testing.T) {
	param := &gollem.Parameter{
		Type: gollem.TypeObject,
		Properties: map[string]*gollem.Parameter{
			"key": {Type: gollem.TypeString, Required: true},
		},
	}

	schema := convertParameterToSchema(param)
	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, false, schema["additionalProperties"])

	props := schema["properties"].(map[string]any)
	require.Contains(t, props, "key")

	required := schema["required"].([]string)
	assert.Contains(t, required, "key")
}

func TestConvertParameterToSchema_Enum(t *testing.T) {
	param := &gollem.Parameter{
		Type: gollem.TypeString,
		Enum: []string{"a", "b", "c"},
	}

	schema := convertParameterToSchema(param)
	assert.Equal(t, []string{"a", "b", "c"}, schema["enum"])
}

func TestGetBedrockType(t *testing.T) {
	testCases := []struct {
		input  gollem.ParameterType
		expect string
	}{
		{gollem.TypeString, "string"},
		{gollem.TypeNumber, "number"},
		{gollem.TypeInteger, "integer"},
		{gollem.TypeBoolean, "boolean"},
		{gollem.TypeArray, "array"},
		{gollem.TypeObject, "object"},
		{gollem.ParameterType("unknown"), "string"},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expect, getBedrockType(tc.input))
	}
}

func TestConvertDocumentMap(t *testing.T) {
	input := map[string]interface{}{
		"str":    "hello",
		"bool":   true,
		"float":  3.14,
		"nil":    nil,
		"nested": map[string]interface{}{"key": "val"},
		"list":   []interface{}{"a", "b"},
	}

	result := convertDocumentMap(input)
	assert.Equal(t, "hello", result["str"])
	assert.Equal(t, true, result["bool"])
	assert.Equal(t, 3.14, result["float"])
	assert.Nil(t, result["nil"])
	assert.Equal(t, map[string]interface{}{"key": "val"}, result["nested"])
	assert.Equal(t, []interface{}{"a", "b"}, result["list"])
}

func TestConvertParameterToSchema_NoDescription(t *testing.T) {
	param := &gollem.Parameter{Type: gollem.TypeBoolean}
	schema := convertParameterToSchema(param)
	assert.Equal(t, "boolean", schema["type"])
	_, hasDesc := schema["description"]
	assert.False(t, hasDesc)
}

func TestConvertValue_SmithyNumber(t *testing.T) {
	// Int-compatible number
	result := convertValue(smithydoc.Number("42"))
	assert.Equal(t, int64(42), result)

	// Float number
	result = convertValue(smithydoc.Number("3.14"))
	assert.InDelta(t, 3.14, result, 0.001)
}

func TestConvertValue_UnexpectedType(t *testing.T) {
	result := convertValue(struct{}{})
	assert.IsType(t, "", result) // should return string representation
}

func TestNewHistory_Roundtrip(t *testing.T) {
	messages := []types.Message{
		{
			Role: types.ConversationRoleUser,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: "hello"},
			},
		},
		{
			Role: types.ConversationRoleAssistant,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: "hi there"},
			},
		},
	}

	history, err := NewHistory(messages)
	require.Nil(t, err)
	assert.Len(t, history.Messages, 2)
	assert.Equal(t, gollem.RoleUser, history.Messages[0].Role)
	assert.Equal(t, gollem.RoleAssistant, history.Messages[1].Role)

	// Roundtrip back to Bedrock
	bedrockMsgs, err := ToMessages(history)
	require.Nil(t, err)
	assert.Len(t, bedrockMsgs, 2)
	assert.Equal(t, types.ConversationRoleUser, bedrockMsgs[0].Role)
	assert.Equal(t, types.ConversationRoleAssistant, bedrockMsgs[1].Role)
}

func TestConvertBedrockMessage_UnsupportedRole(t *testing.T) {
	msg := types.Message{Role: "unknown"}
	_, err := convertBedrockMessage(msg)
	assert.NotNil(t, err)
}

func TestConvertMessageToBedrock_ToolRoleMapsToAssistant(t *testing.T) {
	msg := gollem.Message{
		Role:     gollem.RoleTool,
		Contents: []gollem.MessageContent{},
	}
	bedrockMsg, err := convertMessageToBedrock(msg)
	require.Nil(t, err)
	assert.Equal(t, types.ConversationRoleAssistant, bedrockMsg.Role)
}

func TestConvertMessageToBedrock_UnsupportedRole(t *testing.T) {
	msg := gollem.Message{Role: "unknown"}
	_, err := convertMessageToBedrock(msg)
	assert.NotNil(t, err)
}
