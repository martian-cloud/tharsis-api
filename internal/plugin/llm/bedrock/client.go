package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm"
)

const (
	defaultTemperature = 0
	defaultMaxTokens   = 2048
)

// structuredOutputModels lists Bedrock model ID prefixes that support
// the native OutputConfig structured output feature.
var structuredOutputModels = []string{
	"anthropic.claude-haiku-4",
	"anthropic.claude-sonnet-4",
	"anthropic.claude-opus-4",
}

// Client is a Bedrock LLM client.
type Client struct {
	client       *bedrockruntime.Client
	model        string
	systemPrompt string
	temperature  *float32
	maxTokens    *int32
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithModel sets the default model for the client.
func WithModel(modelName string) Option {
	return func(c *Client) {
		c.model = modelName
	}
}

// WithTemperature sets the temperature parameter.
func WithTemperature(temp float32) Option {
	return func(c *Client) {
		c.temperature = &temp
	}
}

// WithMaxTokens sets the maximum number of tokens.
func WithMaxTokens(maxTokens int32) Option {
	return func(c *Client) {
		c.maxTokens = &maxTokens
	}
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(c *Client) {
		c.systemPrompt = prompt
	}
}

// New creates a new Bedrock client.
func New(cfg aws.Config, options ...Option) (*Client, error) {
	client := &Client{
		client: bedrockruntime.NewFromConfig(cfg),
	}

	for _, option := range options {
		option(client)
	}

	return client, nil
}

// Session is a Bedrock LLM session.
type Session struct {
	apiClient       apiClient
	model           string
	tools           []types.Tool
	historyMessages []types.Message
	cfg             gollem.SessionConfig
	temperature     float32
	maxTokens       int32
}

// NewSession creates a new Bedrock session.
func (c *Client) NewSession(_ context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	cfg := gollem.NewSessionConfig(options...)

	var temperature float32
	if c.temperature != nil {
		temperature = *c.temperature
	} else {
		temperature = defaultTemperature
	}

	var maxTokens int32
	if c.maxTokens != nil {
		maxTokens = *c.maxTokens
	} else {
		maxTokens = defaultMaxTokens
	}

	bedrockTools := make([]types.Tool, len(cfg.Tools()))
	for i, tool := range cfg.Tools() {
		bedrockTools[i] = convertTool(tool)
	}

	var historyMessages []types.Message
	if cfg.History() != nil {
		var err error
		historyMessages, err = ToMessages(cfg.History())
		if err != nil {
			return nil, goerr.Wrap(err, "failed to convert history to Bedrock format")
		}
	}

	session := &Session{
		apiClient:       &realAPIClient{client: c.client},
		model:           c.model,
		tools:           bedrockTools,
		temperature:     temperature,
		maxTokens:       maxTokens,
		historyMessages: historyMessages,
		cfg:             cfg,
	}

	return session, nil
}

// History returns the session history.
func (s *Session) History() (*gollem.History, error) {
	return NewHistory(s.historyMessages)
}

// AppendHistory appends history to the session.
func (s *Session) AppendHistory(h *gollem.History) error {
	if h == nil {
		return nil
	}
	messages, err := ToMessages(h)
	if err != nil {
		return goerr.Wrap(err, "failed to convert history to Bedrock format")
	}
	s.historyMessages = append(s.historyMessages, messages...)
	return nil
}

// Generate generates a response from the model.
func (s *Session) Generate(ctx context.Context, input []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
	// Create a copy of history for middleware
	historyCopy, err := s.History()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get history")
	}

	contentReq := &gollem.ContentRequest{
		Inputs:       input,
		History:      historyCopy,
		SystemPrompt: s.cfg.SystemPrompt(),
	}

	// Create the base handler that performs the actual API call
	baseHandler := func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		// Always update history from middleware (even if same address, content may have changed)
		if req.History != nil {
			var err error
			s.historyMessages, err = ToMessages(req.History)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to convert history from middleware")
			}
		}

		// Convert inputs to messages
		newMessages, err := s.convertInputsToMessages(req.Inputs...)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to convert inputs")
		}

		// Append new messages to history
		s.historyMessages = append(s.historyMessages, newMessages...)

		// Build request inline
		request := &bedrockruntime.ConverseInput{
			ModelId:         aws.String(s.model),
			InferenceConfig: &types.InferenceConfiguration{},
			Messages:        s.historyMessages,
		}

		// Add system prompt
		if s.cfg.SystemPrompt() != "" {
			request.System = []types.SystemContentBlock{
				&types.SystemContentBlockMemberText{Value: s.cfg.SystemPrompt()},
			}
		}

		// Add tools
		if len(s.tools) > 0 {
			tools := s.tools
			request.ToolConfig = &types.ToolConfiguration{Tools: tools}
		}

		// Add output config for structured JSON output (supported on Claude 4.5+ models)
		outputConfig, err := buildOutputConfig(s.cfg, s.model)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to build output config")
		}
		if outputConfig != nil {
			request.OutputConfig = outputConfig
		}

		// Add inference config
		inferenceConfig := &types.InferenceConfiguration{
			Temperature: &s.temperature,
			MaxTokens:   &s.maxTokens,
		}

		request.InferenceConfig = inferenceConfig

		// Attach session ID to request metadata if available in context
		if sessionID, ok := llm.SessionIDFromContext(ctx); ok {
			request.RequestMetadata = map[string]string{
				"agentSessionId": sessionID,
			}
		}

		// Start LLM call trace span
		var traceData *trace.LLMCallData
		var llmErr error
		if h := trace.HandlerFrom(ctx); h != nil {
			ctx = h.StartLLMCall(ctx)
			defer func() { h.EndLLMCall(ctx, traceData, llmErr) }()
		}

		resp, err := s.apiClient.Converse(ctx, request)
		if err != nil {
			llmErr = err
			return nil, goerr.Wrap(err, "failed to generate content")
		}

		result, err := processResponse(resp)
		if err != nil {
			llmErr = err
			return nil, err
		}

		// Set trace data for defer
		traceData = buildBedrockTraceData(newMessages, resp, s.model, s.cfg.SystemPrompt())

		// Add assistant's response to history
		if resp.Output != nil {
			if msg, ok := resp.Output.(*types.ConverseOutputMemberMessage); ok {
				s.historyMessages = append(s.historyMessages, types.Message{
					Role:    types.ConversationRoleAssistant,
					Content: msg.Value.Content,
				})
			}
		}

		return &gollem.ContentResponse{
			Texts:         result.Texts,
			FunctionCalls: result.FunctionCalls,
			InputToken:    result.InputToken,
			OutputToken:   result.OutputToken,
		}, nil
	}

	// Build middleware chain
	handler := gollem.ContentBlockHandler(baseHandler)
	for i := len(s.cfg.ContentBlockMiddlewares()) - 1; i >= 0; i-- {
		handler = s.cfg.ContentBlockMiddlewares()[i](handler)
	}

	// Execute middleware chain
	contentResp, err := handler(ctx, contentReq)
	if err != nil {
		return nil, err
	}

	// Convert ContentResponse back to gollem.Response
	return &gollem.Response{
		Texts:         contentResp.Texts,
		FunctionCalls: contentResp.FunctionCalls,
		InputToken:    contentResp.InputToken,
		OutputToken:   contentResp.OutputToken,
	}, nil
}

// Stream streams a response from the model.
func (s *Session) Stream(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (<-chan *gollem.Response, error) {
	return nil, goerr.New("streaming not currently supported for Bedrock plugin")
}

// CountToken counts the number of tokens in the input.
func (s *Session) CountToken(_ context.Context, _ ...gollem.Input) (int, error) {
	return 0, goerr.New("count tokens api is not currently supported")
}

// GenerateContent is deprecated. Use Generate instead.
func (s *Session) GenerateContent(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
	return s.Generate(ctx, input)
}

// GenerateStream is deprecated. Use Stream instead.
func (s *Session) GenerateStream(ctx context.Context, input ...gollem.Input) (<-chan *gollem.Response, error) {
	return s.Stream(ctx, input)
}

// GenerateEmbedding generates embeddings for the given input.
func (c *Client) GenerateEmbedding(_ context.Context, _ int, _ []string) ([][]float64, error) {
	return nil, goerr.New("embedding not supported for Bedrock")
}

// GetCreditCount converts token counts to credits (1000 credits ≈ $1).
// Uses Amazon Nova Pro pricing: $0.80/M input tokens, $3.20/M output tokens.
func (c *Client) GetCreditCount(input llm.CreditInput) float64 {
	switch c.model {
	case "us.amazon.nova-pro-v1:0":
		// Nova Pro pricing: $0.80 per million input tokens, $3.20 per million output tokens
		dollars := (float64(input.InputTokens) * 0.80 / 1_000_000) +
			(float64(input.OutputTokens) * 3.20 / 1_000_000)
		return dollars * 1000

	default:
		return float64(input.InputTokens) + float64(input.OutputTokens) // Unknown model, return 1 credit per token as a fallback
	}
}

func (s *Session) convertInputsToMessages(input ...gollem.Input) ([]types.Message, error) {
	var newMessages []types.Message
	var userContentParts []types.ContentBlock
	var toolResults []types.ContentBlock

	for _, in := range input {
		switch v := in.(type) {
		case gollem.Text:
			userContentParts = append(userContentParts, &types.ContentBlockMemberText{
				Value: string(v),
			})

		case gollem.Image:
			userContentParts = append(userContentParts, &types.ContentBlockMemberImage{
				Value: types.ImageBlock{
					Format: types.ImageFormat(v.MimeType()),
					Source: &types.ImageSourceMemberBytes{
						Value: v.Data(),
					},
				},
			})

		case gollem.PDF:
			userContentParts = append(userContentParts, &types.ContentBlockMemberDocument{
				Value: types.DocumentBlock{
					Format: types.DocumentFormatPdf,
					Name:   aws.String("document.pdf"),
					Source: &types.DocumentSourceMemberBytes{
						Value: v.Data(),
					},
				},
			})

		case gollem.FunctionResponse:
			// Flush any accumulated user content first
			if len(userContentParts) > 0 {
				newMessages = append(newMessages, types.Message{
					Role:    types.ConversationRoleUser,
					Content: userContentParts,
				})
				userContentParts = nil
			}

			var content []types.ToolResultContentBlock
			if v.Error != nil {
				content = []types.ToolResultContentBlock{
					&types.ToolResultContentBlockMemberText{
						Value: fmt.Sprintf("Error: %v", v.Error),
					},
				}
			} else {
				// Check if Data is empty or nil
				if len(v.Data) == 0 {
					content = []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberText{
							Value: "Success",
						},
					}
				} else {
					doc := document.NewLazyDocument(v.Data)
					content = []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberJson{
							Value: doc,
						},
					}
				}
			}

			// Accumulate tool results instead of creating separate messages
			toolResults = append(toolResults, &types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(v.ID),
					Content:   content,
				},
			})

		default:
			return nil, goerr.Wrap(gollem.ErrInvalidParameter, "failed to convert input to Bedrock message: unsupported input type")
		}
	}

	// Create final user message if there's any remaining user content
	if len(userContentParts) > 0 {
		newMessages = append(newMessages, types.Message{
			Role:    types.ConversationRoleUser,
			Content: userContentParts,
		})
	}

	// Create a single message with all tool results
	if len(toolResults) > 0 {
		newMessages = append(newMessages, types.Message{
			Role:    types.ConversationRoleUser,
			Content: toolResults,
		})
	}

	return newMessages, nil
}

func processResponse(resp *bedrockruntime.ConverseOutput) (*gollem.Response, error) {
	if resp.Output == nil {
		return nil, goerr.New("empty response from Bedrock")
	}

	result := &gollem.Response{}
	if resp.Usage != nil {
		if resp.Usage.InputTokens != nil {
			result.InputToken = int(*resp.Usage.InputTokens)
		}
		if resp.Usage.OutputTokens != nil {
			result.OutputToken = int(*resp.Usage.OutputTokens)
		}
	}

	switch output := resp.Output.(type) {
	case *types.ConverseOutputMemberMessage:
		for _, content := range output.Value.Content {
			switch block := content.(type) {
			case *types.ContentBlockMemberText:
				text := block.Value
				result.Texts = append(result.Texts, text)
			case *types.ContentBlockMemberToolUse:
				// Convert document.Interface to map[string]any
				args := make(map[string]any)
				if block.Value.Input != nil {
					raw, err := block.Value.Input.MarshalSmithyDocument()
					if err != nil {
						return nil, goerr.Wrap(err, "failed to marshal tool input")
					}
					if err := json.Unmarshal(raw, &args); err != nil {
						return nil, goerr.Wrap(err, "failed to unmarshal tool input")
					}
				}
				result.FunctionCalls = append(result.FunctionCalls, &gollem.FunctionCall{
					ID:        aws.ToString(block.Value.ToolUseId),
					Name:      aws.ToString(block.Value.Name),
					Arguments: convertDocumentMap(args),
				})
			}
		}
	}

	return result, nil
}

// supportsStructuredOutput checks if the given model supports Bedrock's
// native OutputConfig structured output feature.
func supportsStructuredOutput(modelID string) bool {
	for _, prefix := range structuredOutputModels {
		if strings.HasPrefix(modelID, prefix) {
			return true
		}
	}
	return false
}

// buildOutputConfig creates a Bedrock OutputConfig from the session config
// when ContentTypeJSON and a ResponseSchema are set.
// Only supported on specific models (Claude 4.5+). Returns nil for unsupported models.
func buildOutputConfig(cfg gollem.SessionConfig, modelID string) (*types.OutputConfig, error) {
	if cfg.ContentType() != gollem.ContentTypeJSON || cfg.ResponseSchema() == nil || !supportsStructuredOutput(modelID) {
		return nil, nil
	}

	schemaMap := convertParameterToSchema(cfg.ResponseSchema())
	schemaJSON, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal response schema")
	}

	return &types.OutputConfig{
		TextFormat: &types.OutputFormat{
			Type: types.OutputFormatTypeJsonSchema,
			Structure: &types.OutputFormatStructureMemberJsonSchema{
				Value: types.JsonSchemaDefinition{
					Schema: aws.String(string(schemaJSON)),
					Name:   aws.String("response_schema"),
				},
			},
		},
	}, nil
}
