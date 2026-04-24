package bedrock

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBedrockTraceData_Basic(t *testing.T) {
	inputMessages := []types.Message{
		{
			Role: types.ConversationRoleUser,
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: "hello"},
			},
		},
	}

	resp := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "response text"},
				},
			},
		},
		Usage: &types.TokenUsage{
			InputTokens:  aws.Int32(100),
			OutputTokens: aws.Int32(50),
		},
	}

	data := buildBedrockTraceData(inputMessages, resp, "test-model", "system prompt")

	assert.Equal(t, "test-model", data.Model)
	assert.Equal(t, "system prompt", data.Request.SystemPrompt)
	require.Len(t, data.Request.Messages, 1)
	assert.Equal(t, "user", data.Request.Messages[0].Role)
	assert.Contains(t, data.Request.Messages[0].Content, "hello")
	require.Len(t, data.Response.Texts, 1)
	assert.Equal(t, "response text", data.Response.Texts[0])
	assert.Equal(t, 100, data.InputTokens)
	assert.Equal(t, 50, data.OutputTokens)
}

func TestBuildBedrockTraceData_WithToolUse(t *testing.T) {
	resp := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Content: []types.ContentBlock{
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call-1"),
							Name:      aws.String("get_workspace"),
						},
					},
				},
			},
		},
		Usage: &types.TokenUsage{
			InputTokens:  aws.Int32(10),
			OutputTokens: aws.Int32(20),
		},
	}

	data := buildBedrockTraceData(nil, resp, "test-model", "")

	require.Len(t, data.Response.FunctionCalls, 1)
	assert.Equal(t, "call-1", data.Response.FunctionCalls[0].ID)
	assert.Equal(t, "get_workspace", data.Response.FunctionCalls[0].Name)
}

func TestBuildBedrockTraceData_NilUsage(t *testing.T) {
	resp := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{Content: []types.ContentBlock{}},
		},
	}

	data := buildBedrockTraceData(nil, resp, "test-model", "")
	assert.Equal(t, 0, data.InputTokens)
	assert.Equal(t, 0, data.OutputTokens)
}
