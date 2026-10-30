package main

import (
	"context"
	"net/url"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client/token"
)

// TokenResolverInput is the input to create a token resolver.
type TokenResolverInput struct {
	BaseURL          *url.URL
	Logger           client.LeveledLogger
	OIDCTokenSource  auth.OIDCTokenSource
	ServiceAccountID string
	UserAgent        string
	TLSSkipVerify    bool
}

// NewTokenResolver creates a token resolver that obtains an OIDC token from the given source and
// exchanges it for a service account token.
//
// The source is called on every renewal rather than once, so a source backed by a file sees the
// file as it is at that moment.
//
// The context is captured rather than passed per renewal because token.NewServiceAccount takes a
// func() ([]byte, error) with no context parameter. The Phobos agent passes the renewal's own
// context to GetToken; here the only context available is the one this resolver was built with,
// which is the runner agent's root context and so lives as long as the resolver does.
func NewTokenResolver(ctx context.Context, input *TokenResolverInput) (client.TokenResolver, error) {
	return token.NewServiceAccount(
		ctx,
		input.BaseURL.String(),
		input.ServiceAccountID,
		func() ([]byte, error) {
			oidcToken, err := input.OIDCTokenSource.GetToken(ctx)
			if err != nil {
				return nil, err
			}

			return []byte(oidcToken), nil
		},
		token.WithTLSSkipVerify(input.TLSSkipVerify),
		token.WithLogger(input.Logger),
		token.WithUserAgent(input.UserAgent),
	)
}
