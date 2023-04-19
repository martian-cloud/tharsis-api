// Package tfe package
package tfe

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// BuildTFEServiceDiscoveryHandler builds a handler function which returns the TFE discovery document
func BuildTFEServiceDiscoveryHandler(
	logger logger.Logger,
	idp *auth.IdentityProviderConfig,
	loginScopes string,
	apiEndpoint string,
	tfeBasePath string,
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
		resp["login.v1"] = map[string]interface{}{
			"client":      idp.ClientID,
			"grant_types": []string{"authz_code"},
			"scopes":      []string{loginScopes},
			"authz":       idp.AuthEndpoint,
			"token":       idp.TokenEndpoint,
		}
	}

	respStr, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TFE discovery document %v", err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if _, err := w.Write([]byte(respStr)); err != nil {
			logger.Errorf("Failed to response with service discovery document %v", err)
		}
	}, nil
}
