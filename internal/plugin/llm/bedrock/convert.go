package bedrock

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	smithydoc "github.com/aws/smithy-go/document"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
)

func convertTool(tool gollem.Tool) types.Tool {
	spec := tool.Spec()

	inputSchema := convertParametersToSchema(spec.Parameters)

	// Pass the map directly to NewLazyDocument (not JSON bytes)
	doc := document.NewLazyDocument(inputSchema)

	return &types.ToolMemberToolSpec{
		Value: types.ToolSpecification{
			Name:        &spec.Name,
			Description: &spec.Description,
			InputSchema: &types.ToolInputSchemaMemberJson{
				Value: doc,
			},
		},
	}
}

func convertParametersToSchema(params map[string]*gollem.Parameter) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false, // Bedrock requires this
	}

	// Only add properties if there are parameters
	if len(params) > 0 {
		properties := make(map[string]any)
		var required []string

		for name, param := range params {
			properties[name] = convertParameterToSchema(param)
			if param.Required {
				required = append(required, name)
			}
		}

		schema["properties"] = properties
		if len(required) > 0 {
			schema["required"] = required
		}
	} else {
		// For tools with no parameters, provide empty properties
		schema["properties"] = map[string]any{}
	}

	return schema
}

func convertParameterToSchema(param *gollem.Parameter) map[string]any {
	schema := map[string]any{
		"type": getBedrockType(param.Type),
	}

	if param.Description != "" {
		schema["description"] = param.Description
	}

	if len(param.Enum) > 0 {
		schema["enum"] = param.Enum
	}

	if param.Type == gollem.TypeArray && param.Items != nil {
		schema["items"] = convertParameterToSchema(param.Items)
	}

	if param.Type == gollem.TypeObject && param.Properties != nil {
		properties := make(map[string]any)
		var required []string

		for name, prop := range param.Properties {
			properties[name] = convertParameterToSchema(prop)
			if prop.Required {
				required = append(required, name)
			}
		}

		schema["properties"] = properties
		schema["additionalProperties"] = false // Bedrock requires this for all object types
		if len(required) > 0 {
			schema["required"] = required
		}
	}

	return schema
}

func getBedrockType(paramType gollem.ParameterType) string {
	switch paramType {
	case gollem.TypeString:
		return "string"
	case gollem.TypeNumber:
		return "number"
	case gollem.TypeInteger:
		return "integer"
	case gollem.TypeBoolean:
		return "boolean"
	case gollem.TypeArray:
		return "array"
	case gollem.TypeObject:
		return "object"
	default:
		return "string"
	}
}

// NewHistory converts Bedrock messages to a gollem History.
func NewHistory(messages []types.Message) (*gollem.History, error) {
	gollemMessages, err := convertBedrockToMessages(messages)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to convert Bedrock messages")
	}
	return &gollem.History{Messages: gollemMessages, Version: gollem.HistoryVersion}, nil
}

// ToMessages converts a gollem History to Bedrock messages.
func ToMessages(h *gollem.History) ([]types.Message, error) {
	return convertMessagesToBedrock(h.Messages)
}

func convertMessagesToBedrock(messages []gollem.Message) ([]types.Message, error) {
	bedrockMessages := make([]types.Message, 0, len(messages))
	for _, msg := range messages {
		bedrockMsg, err := convertMessageToBedrock(msg)
		if err != nil {
			return nil, err
		}
		bedrockMessages = append(bedrockMessages, bedrockMsg)
	}
	return bedrockMessages, nil
}

func convertMessageToBedrock(msg gollem.Message) (types.Message, error) {
	var role types.ConversationRole
	switch msg.Role {
	case gollem.RoleUser:
		role = types.ConversationRoleUser
	case gollem.RoleAssistant, gollem.RoleTool:
		role = types.ConversationRoleAssistant
	default:
		return types.Message{}, goerr.New("unsupported message role")
	}

	contentBlocks := make([]types.ContentBlock, 0, len(msg.Contents))
	for _, content := range msg.Contents {
		block, err := convertContentToBedrock(content)
		if err != nil {
			return types.Message{}, err
		}
		contentBlocks = append(contentBlocks, block)
	}

	return types.Message{
		Role:    role,
		Content: contentBlocks,
	}, nil
}

func convertContentToBedrock(content gollem.MessageContent) (types.ContentBlock, error) {
	switch content.Type {
	case gollem.MessageContentTypeText:
		textContent, err := decodeContent[gollem.TextContent](gollem.MessageContentTypeText, &content)
		if err != nil {
			return nil, err
		}
		return &types.ContentBlockMemberText{Value: textContent.Text}, nil
	case gollem.MessageContentTypeToolCall:
		toolCall, err := decodeContent[gollem.ToolCallContent](gollem.MessageContentTypeToolCall, &content)
		if err != nil {
			return nil, err
		}
		// Convert arguments to document.Interface
		doc := document.NewLazyDocument(toolCall.Arguments)
		return &types.ContentBlockMemberToolUse{
			Value: types.ToolUseBlock{
				ToolUseId: &toolCall.ID,
				Name:      &toolCall.Name,
				Input:     doc,
			},
		}, nil
	case gollem.MessageContentTypeToolResponse:
		resp, err := decodeContent[gollem.ToolResponseContent](gollem.MessageContentTypeToolResponse, &content)
		if err != nil {
			return nil, err
		}
		var resultContent []types.ToolResultContentBlock
		if resp.IsError {
			resultContent = []types.ToolResultContentBlock{
				&types.ToolResultContentBlockMemberText{
					Value: fmt.Sprintf("Error: %v", resp.Response),
				},
			}
		} else if len(resp.Response) == 0 {
			resultContent = []types.ToolResultContentBlock{
				&types.ToolResultContentBlockMemberText{Value: "Success"},
			}
		} else {
			resultContent = []types.ToolResultContentBlock{
				&types.ToolResultContentBlockMemberJson{
					Value: document.NewLazyDocument(resp.Response),
				},
			}
		}
		status := types.ToolResultStatusSuccess
		if resp.IsError {
			status = types.ToolResultStatusError
		}
		return &types.ContentBlockMemberToolResult{
			Value: types.ToolResultBlock{
				ToolUseId: &resp.ToolCallID,
				Content:   resultContent,
				Status:    status,
			},
		}, nil
	case gollem.MessageContentTypeImage:
		img, err := decodeContent[gollem.ImageContent](gollem.MessageContentTypeImage, &content)
		if err != nil {
			return nil, err
		}
		return &types.ContentBlockMemberImage{
			Value: types.ImageBlock{
				Format: types.ImageFormat(img.MediaType),
				Source: &types.ImageSourceMemberBytes{
					Value: img.Data,
				},
			},
		}, nil
	case gollem.MessageContentTypePDF:
		pdf, err := decodeContent[gollem.PDFContent](gollem.MessageContentTypePDF, &content)
		if err != nil {
			return nil, err
		}
		return &types.ContentBlockMemberDocument{
			Value: types.DocumentBlock{
				Format: types.DocumentFormatPdf,
				Name:   aws.String("document.pdf"),
				Source: &types.DocumentSourceMemberBytes{
					Value: pdf.Data,
				},
			},
		}, nil
	default:
		return nil, goerr.New("unsupported content type")
	}
}

func convertBedrockToMessages(messages []types.Message) ([]gollem.Message, error) {
	gollemMessages := make([]gollem.Message, 0, len(messages))
	for _, msg := range messages {
		gollemMsg, err := convertBedrockMessage(msg)
		if err != nil {
			return nil, err
		}
		gollemMessages = append(gollemMessages, gollemMsg)
	}
	return gollemMessages, nil
}

func convertBedrockMessage(msg types.Message) (gollem.Message, error) {
	var role gollem.MessageRole
	switch msg.Role {
	case types.ConversationRoleUser:
		role = gollem.RoleUser
	case types.ConversationRoleAssistant:
		role = gollem.RoleAssistant
	default:
		return gollem.Message{}, goerr.New("unsupported conversation role")
	}

	contents := make([]gollem.MessageContent, 0, len(msg.Content))
	for _, block := range msg.Content {
		content, err := convertBedrockContentBlock(block)
		if err != nil {
			return gollem.Message{}, err
		}
		contents = append(contents, content)
	}

	return gollem.Message{
		Role:     role,
		Contents: contents,
	}, nil
}

func convertBedrockContentBlock(block types.ContentBlock) (gollem.MessageContent, error) {
	switch b := block.(type) {
	case *types.ContentBlockMemberText:
		return makeContent(gollem.MessageContentTypeText, gollem.TextContent{
			Text: b.Value,
		})
	case *types.ContentBlockMemberToolUse:
		// Convert document.Interface to map[string]any
		// Use MarshalSmithyDocument + json.Unmarshal instead of UnmarshalSmithyDocument
		// to work around an AWS SDK bug in documentMarshaler.UnmarshalSmithyDocument.
		args := make(map[string]any)
		if b.Value.Input != nil {
			raw, err := b.Value.Input.MarshalSmithyDocument()
			if err != nil {
				return gollem.MessageContent{}, goerr.Wrap(err, "failed to marshal tool input")
			}
			if err := json.Unmarshal(raw, &args); err != nil {
				return gollem.MessageContent{}, goerr.Wrap(err, "failed to unmarshal tool input")
			}
		}
		return makeContent(gollem.MessageContentTypeToolCall, gollem.ToolCallContent{
			ID:        aws.ToString(b.Value.ToolUseId),
			Name:      aws.ToString(b.Value.Name),
			Arguments: args,
		})
	case *types.ContentBlockMemberToolResult:
		// Convert tool result to gollem format
		resultData := make(map[string]any)
		if len(b.Value.Content) > 0 {
			// Extract content from tool result
			for _, content := range b.Value.Content {
				switch c := content.(type) {
				case *types.ToolResultContentBlockMemberJson:
					// Use MarshalSmithyDocument + json.Unmarshal instead of UnmarshalSmithyDocument
					// to work around an AWS SDK bug in documentMarshaler.UnmarshalSmithyDocument
					// where it decodes into &v instead of &jv, causing a type mismatch when the
					// value is later passed to DecodeJSONInterface.
					raw, err := c.Value.MarshalSmithyDocument()
					if err != nil {
						return gollem.MessageContent{}, goerr.Wrap(err, "failed to marshal tool result")
					}
					var data map[string]any
					if err := json.Unmarshal(raw, &data); err != nil {
						return gollem.MessageContent{}, goerr.Wrap(err, "failed to unmarshal tool result")
					}
					resultData = data
				case *types.ToolResultContentBlockMemberText:
					// For text results, wrap in a simple structure
					resultData = map[string]any{"result": c.Value}
				}
			}
		}
		return makeContent(gollem.MessageContentTypeToolResponse, gollem.ToolResponseContent{
			ToolCallID: aws.ToString(b.Value.ToolUseId),
			Response:   resultData,
			IsError:    b.Value.Status == types.ToolResultStatusError,
		})
	case *types.ContentBlockMemberImage:
		// Skip image blocks in history conversion for now
		return makeContent(gollem.MessageContentTypeText, gollem.TextContent{
			Text: "[Image content]",
		})
	case *types.ContentBlockMemberDocument:
		// Skip document blocks in history conversion for now
		return makeContent(gollem.MessageContentTypeText, gollem.TextContent{
			Text: "[Document content]",
		})
	default:
		return gollem.MessageContent{}, goerr.New("unsupported content block type")
	}
}

// normalizeTypes converts string numbers to actual numbers
func decodeContent[T any](t gollem.MessageContentType, mc *gollem.MessageContent) (*T, error) {
	if mc.Type != t {
		return nil, goerr.New("content type mismatch")
	}
	var content T
	if err := json.Unmarshal(mc.Data, &content); err != nil {
		return nil, err
	}
	return &content, nil
}

func makeContent[T any](t gollem.MessageContentType, v T) (gollem.MessageContent, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return gollem.MessageContent{}, err
	}
	return gollem.MessageContent{Type: t, Data: data}, nil
}

// ConvertDocumentMap converts smithy document values to plain Go primitives:
// - document.Number → float64 (or int64 if it fits without loss)
// - document bool/string/array/map → native equivalents
// - nested structures are recursed
func convertDocumentMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = convertValue(v)
	}
	return out
}

func convertValue(v interface{}) interface{} {
	switch val := v.(type) {
	case smithydoc.Number:
		// Prefer int64 if it fits exactly (common for tool args like IDs, counts)
		if i, err := val.Int64(); err == nil {
			return i
		}
		// Otherwise fall back to float64 (safe for most model outputs)
		f, err := val.Float64()
		if err != nil {
			// Very rare — malformed number from model
			return val.String() // or panic/log, your choice
		}
		return f

	case bool, string:
		return val // already primitive

	case float64, float32, int, int64, int32:
		return val // already primitive

	case []interface{}:
		out := make([]interface{}, len(val))
		for i, elem := range val {
			out[i] = convertValue(elem)
		}
		return out

	case map[string]interface{}:
		return convertDocumentMap(val) // recurse

	case nil:
		return nil

	default:
		// Should not happen with valid smithy document unmarshal
		return fmt.Sprintf("<unexpected type %T>", val)
	}
}
