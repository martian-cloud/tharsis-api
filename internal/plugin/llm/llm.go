// Package llm package
package llm

//go:generate go tool mockery --name Client --inpackage --case underscore

import (
	"context"

	"github.com/m-mizutani/gollem"
)

type contextKey string

const sessionIDKey contextKey = "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm session id"

// WithSessionID returns a context with the given session ID attached.
func WithSessionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionIDKey, id)
}

// SessionIDFromContext extracts the session ID from the context, if present.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(sessionIDKey).(string)
	return id, ok
}

// CreditInput contains token counts for credit calculation.
type CreditInput struct {
	InputTokens  int
	OutputTokens int
}

// Client is the interface for LLM client plugins.
type Client interface {
	gollem.LLMClient
	GetCreditCount(input CreditInput) float64
}
