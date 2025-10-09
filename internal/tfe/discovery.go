// Package tfe package
package tfe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// BuildTFEServiceDiscoveryHandler builds a handler function which returns the TFE discovery document
func BuildTFEServiceDiscoveryHandler(
	ctx context.Context,
	logger logger.Logger,
	tfeBasePath string,
	oidcConfigFetcher auth.OpenIDConfigFetcher,
	cfg *config.Config,
) (http.HandlerFunc, error) {
	// Build response
	resp := map[string]interface{}{
		"modules.v1":   fmt.Sprintf("%s/v1/module-registry/modules/", cfg.TharsisAPIURL),
		"providers.v1": fmt.Sprintf("%s/v1/provider-registry/providers/", cfg.TharsisAPIURL),
		"state.v2":     fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		"tfe.v2":       fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		"tfe.v2.1":     fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		"tfe.v2.2":     fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
	}

	loginScopes := cfg.CLILoginOIDCScopes

	if len(cfg.OauthProviders) > 0 {
		var loginIdp *config.IdpConfig

		if cfg.CLILoginOIDCClientID != "" {
			// Find IDP that matches client ID
			for _, idp := range cfg.OauthProviders {
				if idp.ClientID == cfg.CLILoginOIDCClientID {
					idp := idp
					loginIdp = &idp
					break
				}
			}

			if loginIdp == nil {
				return nil, errors.New("OIDC Identity Provider not found for TFE login")
			}
		} else {
			loginIdp = &cfg.OauthProviders[0]
		}

		oidcConfig, err := oidcConfigFetcher.GetOpenIDConfig(ctx, loginIdp.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get OIDC config for issuer %s %v", loginIdp.IssuerURL, err)
		}

		resp["login.v1"] = map[string]interface{}{
			"client":      loginIdp.ClientID,
			"grant_types": []string{"authz_code"},
			"scopes":      []string{loginScopes},
			"authz":       oidcConfig.AuthEndpoint,
			"token":       oidcConfig.TokenEndpoint,
		}
	} else {
		resp["login.v1"] = map[string]interface{}{
			"client":      cfg.OIDCInternalIdentityProviderClientID,
			"grant_types": []string{"authz_code"},
			"scopes":      []string{loginScopes},
			"authz":       fmt.Sprintf("%s/oauth/authorize", cfg.TharsisAPIURL),
			"token":       fmt.Sprintf("%s/oauth/token", cfg.TharsisAPIURL),
		}
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logger.WithContextFields(ctx).Errorf("Failed to response with service discovery document %v", err)
		}
	}, nil
}
