package token

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

var _ client.TokenResolver = (*serviceAccountTokenResolver)(nil)

// serviceAccountTokenResolver exchanges an OIDC token for a short-lived
// service account token, caching and renewing it transparently.
type serviceAccountTokenResolver struct {
	sync.RWMutex
	oidcTokenFunc    func() ([]byte, error)
	grpcClient       *client.GRPCClient
	logger           client.LeveledLogger
	serviceAccountID string
	token            string
	renewAt          *time.Time
}

// NewServiceAccount creates a client.TokenResolver that authenticates as a
// Tharsis service account using OIDC token exchange. It creates a separate
// unauthenticated gRPC connection for token renewal to avoid a circular
// dependency on the token being obtained.
//
// serviceAccountID is the TRN or GID of the service account.
// oidcTokenFunc is called each time a new OIDC token is needed for exchange.
func NewServiceAccount(
	ctx context.Context,
	httpEndpoint string,
	serviceAccountID string,
	oidcTokenFunc func() ([]byte, error),
	opts ...ResolverOption,
) (client.TokenResolver, error) {
	options := &resolverOptions{}
	for _, o := range opts {
		o(options)
	}

	// Separate unauthenticated gRPC client for token renewal to avoid
	// circular dependency on the token we're trying to obtain.
	c, err := client.NewGRPCClient(ctx, &client.GRPCClientConfig{
		HTTPEndpoint:  httpEndpoint,
		TLSSkipVerify: options.tlsSkipVerify,
		Logger:        options.logger,
		UserAgent:     options.userAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create unauthenticated client for token renewal: %w", err)
	}

	r := &serviceAccountTokenResolver{
		oidcTokenFunc:    oidcTokenFunc,
		grpcClient:       c,
		logger:           options.logger,
		serviceAccountID: serviceAccountID,
	}

	// Fail fast if credentials are invalid.
	if _, err := r.Token(ctx); err != nil {
		c.Close()
		return nil, err
	}

	return r, nil
}

// Token returns a valid service account token, renewing it if expired.
func (r *serviceAccountTokenResolver) Token(ctx context.Context) (string, error) {
	// Fast path: if the token is still valid, return it under a read lock.
	r.RLock()
	if r.renewAt != nil && time.Now().Before(*r.renewAt) {
		token := r.token
		r.RUnlock()
		return token, nil
	}
	r.RUnlock()

	// Slow path: acquire write lock and renew.
	r.Lock()
	defer r.Unlock()

	// Double-check after acquiring write lock.
	if r.renewAt != nil && time.Now().Before(*r.renewAt) {
		return r.token, nil
	}

	if err := r.renewToken(ctx); err != nil {
		if r.logger != nil {
			r.logger.Error("service account token renewal failed", "error", err)
		}

		return "", fmt.Errorf("service account token renewal failed: %w", err)
	}

	return r.token, nil
}

// Close closes the underlying gRPC connection used for token renewal.
func (r *serviceAccountTokenResolver) Close() error {
	return r.grpcClient.Close()
}

func (r *serviceAccountTokenResolver) renewToken(ctx context.Context) error {
	oidcToken, err := r.oidcTokenFunc()
	if err != nil {
		return fmt.Errorf("failed to get OIDC token: %w", err)
	}

	tokenResp, err := r.grpcClient.ServiceAccountsClient.CreateOIDCToken(ctx, &pb.CreateOIDCTokenRequest{
		ServiceAccountId: r.serviceAccountID,
		Token:            string(oidcToken),
	})
	if err != nil {
		return fmt.Errorf("failed to create service account token: %w", err)
	}

	// Eligible to renew one minute before expiration.
	renewAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn)*time.Second - time.Minute)
	r.token = tokenResp.Token
	r.renewAt = &renewAt

	return nil
}
