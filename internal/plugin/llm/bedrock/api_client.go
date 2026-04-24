// Package bedrock implements the Bedrock LLM plugin.
package bedrock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

type apiClient interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
	ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error)
	CountTokens(ctx context.Context, params *bedrockruntime.CountTokensInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.CountTokensOutput, error)
}

type realAPIClient struct {
	client *bedrockruntime.Client
}

func (r *realAPIClient) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	return r.client.Converse(ctx, params, optFns...)
}

func (r *realAPIClient) ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
	return r.client.ConverseStream(ctx, params, optFns...)
}

func (r *realAPIClient) CountTokens(ctx context.Context, params *bedrockruntime.CountTokensInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.CountTokensOutput, error) {
	return r.client.CountTokens(ctx, params, optFns...)
}
