package token

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
)

var _ client.TokenResolver = staticTokenResolver(nil)

// staticTokenResolver wraps a token-fetching function as a client.TokenResolver.
// The caller controls whether the token is a fixed value or re-read from
// disk on each call.
type staticTokenResolver func() (string, error)

// Token returns the static token.
func (f staticTokenResolver) Token(_ context.Context) (string, error) {
	return f()
}

// Close is a no-op for static tokens.
func (f staticTokenResolver) Close() error {
	return nil
}

// NewStatic creates a TokenResolver that calls tokenFunc on each Token() invocation.
// It validates the token is non-empty at construction time to fail fast.
func NewStatic(tokenFunc func() (string, error)) (client.TokenResolver, error) {
	token, err := tokenFunc()
	if err != nil {
		return nil, err
	}

	if token == "" {
		return nil, fmt.Errorf("authentication token is empty")
	}

	return staticTokenResolver(tokenFunc), nil
}
