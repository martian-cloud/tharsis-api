package bedrock

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm"
)

func TestGetCreditCount_NovaPro(t *testing.T) {
	c := &Client{model: "us.amazon.nova-pro-v1:0"}

	credits := c.GetCreditCount(llm.CreditInput{
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	})

	// $0.80 input + $3.20 output = $4.00 → 4000 credits
	assert.InDelta(t, 4000.0, credits, 0.01)
}

func TestGetCreditCount_NovaProZeroTokens(t *testing.T) {
	c := &Client{model: "us.amazon.nova-pro-v1:0"}

	credits := c.GetCreditCount(llm.CreditInput{})
	assert.Equal(t, float64(0), credits)
}

func TestGetCreditCount_UnknownModel(t *testing.T) {
	c := &Client{model: "unknown-model"}

	credits := c.GetCreditCount(llm.CreditInput{
		InputTokens:  100,
		OutputTokens: 50,
	})

	// Fallback: 1 credit per token
	assert.Equal(t, float64(150), credits)
}

func TestSupportsStructuredOutput(t *testing.T) {
	assert.True(t, supportsStructuredOutput("anthropic.claude-haiku-4-20250514-v1:0"))
	assert.True(t, supportsStructuredOutput("anthropic.claude-sonnet-4-20250514-v1:0"))
	assert.True(t, supportsStructuredOutput("anthropic.claude-opus-4-20250514-v1:0"))
	assert.False(t, supportsStructuredOutput("us.amazon.nova-pro-v1:0"))
	assert.False(t, supportsStructuredOutput("anthropic.claude-3-5-sonnet-20241022-v2:0"))
}

func TestProcessResponse_TextAndToolUse(t *testing.T) {
	resp := &bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "some text"},
					&types.ContentBlockMemberToolUse{
						Value: types.ToolUseBlock{
							ToolUseId: aws.String("call-1"),
							Name:      aws.String("my_tool"),
						},
					},
				},
			},
		},
		Usage: &types.TokenUsage{
			InputTokens:  aws.Int32(50),
			OutputTokens: aws.Int32(25),
		},
	}

	result, err := processResponse(resp)
	require.Nil(t, err)
	assert.Equal(t, []string{"some text"}, result.Texts)
	require.Len(t, result.FunctionCalls, 1)
	assert.Equal(t, "call-1", result.FunctionCalls[0].ID)
	assert.Equal(t, "my_tool", result.FunctionCalls[0].Name)
	assert.Equal(t, 50, result.InputToken)
	assert.Equal(t, 25, result.OutputToken)
}

func TestProcessResponse_NilOutput(t *testing.T) {
	resp := &bedrockruntime.ConverseOutput{Output: nil}
	_, err := processResponse(resp)
	assert.NotNil(t, err)
}
