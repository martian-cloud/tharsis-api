package bedrock

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm"
)

var supportedModels = []string{"us.amazon.nova-pro-v1:0"}

var pluginDataRequiredFields = []string{"region", "model"}

type awsConfigLoader func(ctx context.Context, region string) (aws.Config, error)

// NewPlugin creates a Bedrock LLM client plugin from plugin data configuration.
func NewPlugin(ctx context.Context, pluginData map[string]string) (llm.Client, error) {
	return newPlugin(ctx, pluginData, defaultConfigLoader)
}

func newPlugin(ctx context.Context, pluginData map[string]string, loader awsConfigLoader) (llm.Client, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("bedrock LLM plugin requires %q field", field)
		}
	}

	cfg, err := loader(ctx, pluginData["region"])
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	model, ok := pluginData["model"]
	if !ok {
		return nil, fmt.Errorf("model field is required in plugin data")
	}

	if !slices.Contains(supportedModels, model) {
		return nil, fmt.Errorf("unsupported model: %q", model)
	}

	opts := []Option{
		WithModel(model),
	}

	if v, ok := pluginData["temperature"]; ok {
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid temperature value: %v", err)
		}
		opts = append(opts, WithTemperature(float32(f)))
	}

	if v, ok := pluginData["max_tokens"]; ok {
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid max_tokens value: %v", err)
		}
		opts = append(opts, WithMaxTokens(int32(i)))
	}

	return New(cfg, opts...)
}

func defaultConfigLoader(ctx context.Context, region string) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
}
