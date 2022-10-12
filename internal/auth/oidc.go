package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OIDCConfiguration contains the OIDC information for an identity provider
type OIDCConfiguration struct {
	Issuer        string `json:"issuer"`
	JwksURI       string `json:"jwks_uri"`
	TokenEndpoint string `json:"token_endpoint"`
	AuthEndpoint  string `json:"authorization_endpoint"`
}

// GetOpenIDConfig returns the IDP config from the OIDC discovery document
func GetOpenIDConfig(ctx context.Context, issuer string) (*OIDCConfiguration, error) {
	wellKnownURI := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequest("GET", wellKnownURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build OIDC request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
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
