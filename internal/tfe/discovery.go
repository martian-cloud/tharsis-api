// Package tfe package
package tfe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// loginScopes are the only scopes the API's OAuth authorize endpoint accepts.
var loginScopes = []string{"openid", "tharsis"}

// BuildTFEServiceDiscoveryHandler builds a handler function which returns the TFE discovery document
func BuildTFEServiceDiscoveryHandler(
	ctx context.Context,
	logger logger.Logger,
	tfeBasePath string,
	cfg *config.Config,
) http.HandlerFunc {
	resp := map[string]interface{}{
		"modules.v1":   fmt.Sprintf("%s/v1/module-registry/modules/", cfg.TharsisAPIURL),
		"providers.v1": fmt.Sprintf("%s/v1/provider-registry/providers/", cfg.TharsisAPIURL),
		"state.v2":     fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		"tfe.v2":       fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		"tfe.v2.1":     fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		"tfe.v2.2":     fmt.Sprintf("%s%s/v2/", cfg.TharsisAPIURL, tfeBasePath),
		// Terraform always logs in against the API, which proxies the login to the external identity provider
		// when one is configured and issues a Tharsis user session token.
		"login.v1": map[string]interface{}{
			"client":      auth.TharsisOAuthClientID,
			"grant_types": []string{"authz_code"},
			"scopes":      loginScopes,
			"authz":       fmt.Sprintf("%s/oauth/authorize", cfg.TharsisAPIURL),
			"token":       fmt.Sprintf("%s/oauth/token", cfg.TharsisAPIURL),
		},
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logger.WithContextFields(ctx).Errorf("Failed to response with service discovery document %v", err)
		}
	}
}
