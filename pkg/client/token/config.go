package token

import (
	"context"
	"fmt"

	"github.com/qiangxue/go-env"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// ResolverOption configures token resolver behavior.
type ResolverOption func(*resolverOptions)

type resolverOptions struct {
	logger        client.LeveledLogger
	userAgent     string
	tlsSkipVerify bool
}

// WithTLSSkipVerify disables TLS certificate verification.
func WithTLSSkipVerify(skip bool) ResolverOption {
	return func(o *resolverOptions) {
		o.tlsSkipVerify = skip
	}
}

// WithLogger sets the logger.
func WithLogger(logger client.LeveledLogger) ResolverOption {
	return func(o *resolverOptions) {
		o.logger = logger
	}
}

// WithUserAgent sets the user agent.
func WithUserAgent(userAgent string) ResolverOption {
	return func(o *resolverOptions) {
		o.userAgent = userAgent
	}
}

// Config picks the right token strategy based on environment variables.
// Priority: service account > static token > error.
// Environment variables always override values set on the struct so CI/CD
// pipelines can inject credentials without modifying the settings file.
type Config struct {
	ServiceAccountToken string `env:"SERVICE_ACCOUNT_TOKEN,secret"`
	ServiceAccountID    string `env:"SERVICE_ACCOUNT_ID"`
	ServiceAccountPath  string `env:"SERVICE_ACCOUNT_PATH"`
	StaticToken         string `env:"STATIC_TOKEN,secret"`
}

// Resolve returns the appropriate client.TokenResolver. staticTokenFunc is
// used when the static token was not overridden by an environment variable,
// allowing the caller to control how the token is fetched (e.g. re-reading
// from the credentials file for long-lived processes).
func (c *Config) Resolve(
	ctx context.Context,
	httpEndpoint string,
	staticTokenFunc func() (string, error),
	opts ...ResolverOption,
) (client.TokenResolver, error) {
	options := &resolverOptions{}
	for _, o := range opts {
		o(options)
	}

	// Snapshot before env loading so we can detect if THARSIS_STATIC_TOKEN
	// overrode the default value from the credentials file.
	defaultToken := c.StaticToken

	if err := env.New("THARSIS_", nil).Load(c); err != nil {
		return nil, fmt.Errorf("failed to load env variables: %w", err)
	}

	// SERVICE_ACCOUNT_PATH is deprecated; convert to TRN for backwards compatibility.
	// If THARSIS_SERVICE_ACCOUNT_ID is already set, ignore the path — it may have been
	// set alongside the ID by an older server for backwards compatibility with older clients.
	if c.ServiceAccountPath != "" && c.ServiceAccountID == "" {
		if options.logger != nil {
			options.logger.Warn("THARSIS_SERVICE_ACCOUNT_PATH is deprecated, use THARSIS_SERVICE_ACCOUNT_ID instead")
		}

		c.ServiceAccountID = trn.TypeServiceAccount.Build(c.ServiceAccountPath)
	}

	if c.ServiceAccountID != "" && c.ServiceAccountToken != "" {
		return NewServiceAccount(
			ctx,
			httpEndpoint,
			c.ServiceAccountID,
			func() ([]byte, error) {
				return []byte(c.ServiceAccountToken), nil
			},
			opts...,
		)
	}

	if c.StaticToken != "" {
		// If the env var didn't override the default, use staticTokenFunc
		// to re-read from the credentials file on each call.
		var tokenFunc func() (string, error)
		if staticTokenFunc != nil && c.StaticToken == defaultToken {
			tokenFunc = staticTokenFunc
		} else {
			staticToken := c.StaticToken
			tokenFunc = func() (string, error) { return staticToken, nil }
		}

		return NewStatic(tokenFunc)
	}

	return nil, fmt.Errorf("missing authentication credentials: " +
		"set THARSIS_STATIC_TOKEN for static token authentication or " +
		"THARSIS_SERVICE_ACCOUNT_ID and THARSIS_SERVICE_ACCOUNT_TOKEN for service account authentication")
}
