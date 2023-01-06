package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	// retryWaitMinimum is the minimum amount of seconds retryablehttp
	// client will wait before attempting to make another connection.
	// Default min is 2 seconds.
	retryWaitMinimum = time.Second * 5
)

// OpenIDConfigFetcher implements functions to fetch
// OpenID configuration from an issuer.
type OpenIDConfigFetcher struct {
	Client *retryablehttp.Client
}

// NewOpenIDConfigFetcher returns a new NewOpenIDConfigFetcher
func NewOpenIDConfigFetcher() *OpenIDConfigFetcher {
	// Retryablehttp client defaults to 4 retries.
	client := retryablehttp.NewClient()
	client.RetryWaitMin = retryWaitMinimum
	return &OpenIDConfigFetcher{Client: client}
}

// OIDCConfiguration contains the OIDC information for an identity provider
type OIDCConfiguration struct {
	Issuer        string `json:"issuer"`
	JwksURI       string `json:"jwks_uri"`
	TokenEndpoint string `json:"token_endpoint"`
	AuthEndpoint  string `json:"authorization_endpoint"`
}

// GetOpenIDConfig returns the IDP config from the OIDC discovery document
func (o *OpenIDConfigFetcher) GetOpenIDConfig(ctx context.Context, issuer string) (*OIDCConfiguration, error) {
	wellKnownURI := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", wellKnownURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build OIDC request: %v", err)
	}

	// Use retryablehttp client so we can retry incase request fails.
	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request OIDC discovery document: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for OIDC discovery document: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received invalid response from OIDC discovery endpoint %s: %s", resp.Status, body)
	}

	var cfg OIDCConfiguration
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("unable to parse OIDC discovery document: %v", err)
	}

	if cfg.Issuer != issuer {
		return nil, fmt.Errorf("OIDC issuer does not match the issuer returned by the OIDC discovery document, expected %q got %q", issuer, cfg.Issuer)
	}

	return &cfg, nil
}
