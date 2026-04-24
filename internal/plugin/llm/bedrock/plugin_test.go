package bedrock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlugin_Success(t *testing.T) {
	loader := func(_ context.Context, region string) (aws.Config, error) {
		assert.Equal(t, "us-east-1", region)
		return aws.Config{Region: region}, nil
	}

	client, err := newPlugin(context.Background(), map[string]string{
		"region": "us-east-1",
		"model":  "us.amazon.nova-pro-v1:0",
	}, loader)

	require.Nil(t, err)
	assert.NotNil(t, client)
}

func TestNewPlugin_MissingRegion(t *testing.T) {
	_, err := newPlugin(context.Background(), map[string]string{
		"model": "us.amazon.nova-pro-v1:0",
	}, nil)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "region")
}

func TestNewPlugin_MissingModel(t *testing.T) {
	_, err := newPlugin(context.Background(), map[string]string{
		"region": "us-east-1",
	}, nil)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "model")
}

func TestNewPlugin_UnsupportedModel(t *testing.T) {
	loader := func(_ context.Context, _ string) (aws.Config, error) {
		return aws.Config{}, nil
	}

	_, err := newPlugin(context.Background(), map[string]string{
		"region": "us-east-1",
		"model":  "unsupported-model",
	}, loader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported model")
}

func TestNewPlugin_WithOptionalParams(t *testing.T) {
	loader := func(_ context.Context, _ string) (aws.Config, error) {
		return aws.Config{}, nil
	}

	client, err := newPlugin(context.Background(), map[string]string{
		"region":      "us-east-1",
		"model":       "us.amazon.nova-pro-v1:0",
		"temperature": "0.7",
		"max_tokens":  "4096",
	}, loader)

	require.Nil(t, err)
	assert.NotNil(t, client)
}

func TestNewPlugin_InvalidTemperature(t *testing.T) {
	loader := func(_ context.Context, _ string) (aws.Config, error) {
		return aws.Config{}, nil
	}

	_, err := newPlugin(context.Background(), map[string]string{
		"region":      "us-east-1",
		"model":       "us.amazon.nova-pro-v1:0",
		"temperature": "not-a-number",
	}, loader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "temperature")
}

func TestNewPlugin_InvalidMaxTokens(t *testing.T) {
	loader := func(_ context.Context, _ string) (aws.Config, error) {
		return aws.Config{}, nil
	}

	_, err := newPlugin(context.Background(), map[string]string{
		"region":     "us-east-1",
		"model":      "us.amazon.nova-pro-v1:0",
		"max_tokens": "not-a-number",
	}, loader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "max_tokens")
}

func TestNewPlugin_ConfigLoaderError(t *testing.T) {
	loader := func(_ context.Context, _ string) (aws.Config, error) {
		return aws.Config{}, fmt.Errorf("aws config error")
	}

	_, err := newPlugin(context.Background(), map[string]string{
		"region": "us-east-1",
		"model":  "us.amazon.nova-pro-v1:0",
	}, loader)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "AWS config")
}
