package llm

import (
	"context"
	"fmt"

	"github.com/m-mizutani/gollem"
)

// NoopClient is an LLM client that returns an error for all operations.
type NoopClient struct{}

// NewSession returns an error indicating the LLM client is not configured.
func (n *NoopClient) NewSession(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
	return nil, fmt.Errorf("LLM client is not configured")
}

// GenerateEmbedding returns an error indicating the LLM client is not configured.
func (n *NoopClient) GenerateEmbedding(_ context.Context, _ int, _ []string) ([][]float64, error) {
	return nil, fmt.Errorf("LLM client is not configured")
}

// GetCreditCount returns 0 since no LLM client is configured.
func (n *NoopClient) GetCreditCount(_ CreditInput) float64 {
	return 0
}
