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

// BuildTFEServiceDiscoveryHandler builds a handler function which returns the TFE discovery document
func BuildTFEServiceDiscoveryHandler(
	ctx context.Context,
	logger logger.Logger,
	idp *config.IdpConfig,
	loginScopes string,
	apiEndpoint string,
	tfeBasePath string,
	oidcConfigFetcher auth.OpenIDConfigFetcher,
) (http.HandlerFunc, error) {
	// Build response
	resp := map[string]interface{}{
		"modules.v1":   fmt.Sprintf("%s/v1/module-registry/modules/", apiEndpoint),
		"providers.v1": fmt.Sprintf("%s/v1/provider-registry/providers/", apiEndpoint),
		"state.v2":     fmt.Sprintf("%s%s/v2/", apiEndpoint, tfeBasePath),
		"tfe.v2":       fmt.Sprintf("%s%s/v2/", apiEndpoint, tfeBasePath),
		"tfe.v2.1":     fmt.Sprintf("%s%s/v2/", apiEndpoint, tfeBasePath),
		"tfe.v2.2":     fmt.Sprintf("%s%s/v2/", apiEndpoint, tfeBasePath),
	}

	if idp != nil {
		cfg, err := oidcConfigFetcher.GetOpenIDConfig(ctx, idp.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get OIDC config for issuer %s %v", idp.IssuerURL, err)
		}

		resp["login.v1"] = map[string]interface{}{
			"client":      idp.ClientID,
			"grant_types": []string{"authz_code"},
			"scopes":      []string{loginScopes},
			"authz":       cfg.AuthEndpoint,
			"token":       cfg.TokenEndpoint,
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
