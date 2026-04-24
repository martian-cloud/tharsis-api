package bedrock

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/m-mizutani/gollem/trace"
)

func buildBedrockTraceData(inputMessages []types.Message, resp *bedrockruntime.ConverseOutput, model string, systemPrompt string) *trace.LLMCallData {
	data := &trace.LLMCallData{
		Model: model,
		Request: &trace.LLMRequest{
			SystemPrompt: systemPrompt,
			Messages:     []trace.Message{},
			Tools:        []trace.ToolSpec{},
		},
		Response: &trace.LLMResponse{
			Texts:         []string{},
			FunctionCalls: []*trace.FunctionCall{},
		},
	}

	for _, msg := range inputMessages {
		var contentBuilder strings.Builder
		for _, content := range msg.Content {
			switch block := content.(type) {
			case *types.ContentBlockMemberText:
				contentBuilder.WriteString(block.Value + "\n")
			case *types.ContentBlockMemberImage:
				size := 0
				if src, ok := block.Value.Source.(*types.ImageSourceMemberBytes); ok {
					size = len(src.Value)
				}
				contentBuilder.WriteString(fmt.Sprintf("[Image: format=%s, size=%d bytes]", block.Value.Format, size))
			case *types.ContentBlockMemberDocument:
				size := 0
				if src, ok := block.Value.Source.(*types.DocumentSourceMemberBytes); ok {
					size = len(src.Value)
				}
				contentBuilder.WriteString(fmt.Sprintf("[Document: format=%s, name=%s, size=%d bytes]", block.Value.Format, aws.ToString(block.Value.Name), size))
			}
		}
		data.Request.Messages = append(data.Request.Messages, trace.Message{
			Role:    string(msg.Role),
			Content: contentBuilder.String(),
		})
	}

	if resp.Output != nil {
		switch output := resp.Output.(type) {
		case *types.ConverseOutputMemberMessage:
			for _, content := range output.Value.Content {
				switch block := content.(type) {
				case *types.ContentBlockMemberText:
					data.Response.Texts = append(data.Response.Texts, block.Value)
				case *types.ContentBlockMemberToolUse:
					data.Response.FunctionCalls = append(data.Response.FunctionCalls, &trace.FunctionCall{
						ID:        aws.ToString(block.Value.ToolUseId),
						Name:      aws.ToString(block.Value.Name),
						Arguments: map[string]any{
							// Arguments are not included in trace for now due to complexity of converting from document.Interface
							// Can be added in the future if needed
						},
					})
				}
			}
		}
	}

	if resp.Usage != nil {
		if resp.Usage.InputTokens != nil {
			data.InputTokens = int(*resp.Usage.InputTokens)
		}
		if resp.Usage.OutputTokens != nil {
			data.OutputTokens = int(*resp.Usage.OutputTokens)
		}
	}

	return data
}
