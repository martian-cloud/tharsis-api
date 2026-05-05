// Package token provides authentication token resolvers for the Tharsis API clients.
//
// [client.TokenResolver] is the interface all authentication strategies implement.
// This package provides two built-in implementations:
//
//   - Static token resolver: wraps a function that returns a fixed or file-backed token.
//     Created via [NewStatic]. The function is called on every Token() call,
//     allowing the caller to re-read credentials from disk for long-lived processes.
//
//   - Service account token resolver: exchanges an OIDC token for a short-lived Tharsis
//     service account token, caching and auto-renewing it before expiry. Created via
//     [NewServiceAccount]. It maintains a separate unauthenticated gRPC
//     connection for token renewal to avoid circular dependencies.
//
// # Configuration
//
// [Config] selects the appropriate resolver based on environment variables.
// It supports two authentication modes with the following priority:
//
//  1. Service account: requires THARSIS_SERVICE_ACCOUNT_ID and THARSIS_SERVICE_ACCOUNT_TOKEN.
//     THARSIS_SERVICE_ACCOUNT_PATH is accepted as a deprecated alias for the ID.
//
//  2. Static token: uses THARSIS_STATIC_TOKEN, or falls back to a caller-provided
//     function (e.g. reading from a credentials file).
//
// Environment variables always override values set on the struct, allowing CI/CD
// pipelines to inject credentials without modifying configuration files.
//
// # Usage
//
//	config := &token.Config{StaticToken: savedToken}
//	resolver, err := config.Resolve(ctx, endpoint, tokenFileReader)
package token
