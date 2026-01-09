package tfe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestBuildTFEServiceDiscoveryHandler(t *testing.T) {
	tests := []struct {
		name               string
		tfeBasePath        string
		cfg                *config.Config
		mockSetup          func(*auth.MockOpenIDConfigFetcher)
		expectErrorMessage string
		expectLoginClient  string
		expectLoginAuthz   string
		expectLoginToken   string
		expectModulesV1    string
		expectProvidersV1  string
		expectStateV2      string
		expectTfeV2        string
	}{
		{
			name:        "successful handler with oauth providers",
			tfeBasePath: "/tfe",
			cfg: &config.Config{
				TharsisAPIURL:        "https://api.example.com",
				CLILoginOIDCScopes:   "openid profile",
				CLILoginOIDCClientID: "test-client-id",
				OauthProviders: []config.IdpConfig{
					{
						ClientID:  "test-client-id",
						IssuerURL: "https://issuer.example.com",
					},
				},
			},
			mockSetup: func(mockFetcher *auth.MockOpenIDConfigFetcher) {
				oidcConfig := &auth.OIDCConfiguration{
					AuthEndpoint:  "https://issuer.example.com/auth",
					TokenEndpoint: "https://issuer.example.com/token",
				}
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://issuer.example.com").Return(oidcConfig, nil)
			},
			expectLoginClient: "test-client-id",
			expectLoginAuthz:  "https://issuer.example.com/auth",
			expectLoginToken:  "https://issuer.example.com/token",
			expectModulesV1:   "https://api.example.com/v1/module-registry/modules/",
			expectProvidersV1: "https://api.example.com/v1/provider-registry/providers/",
			expectStateV2:     "https://api.example.com/tfe/v2/",
			expectTfeV2:       "https://api.example.com/tfe/v2/",
		},
		{
			name:        "successful handler with first oauth provider when no client ID specified",
			tfeBasePath: "/tfe",
			cfg: &config.Config{
				TharsisAPIURL:      "https://api.example.com",
				CLILoginOIDCScopes: "openid profile",
				OauthProviders: []config.IdpConfig{
					{
						ClientID:  "first-client-id",
						IssuerURL: "https://first-issuer.example.com",
					},
					{
						ClientID:  "second-client-id",
						IssuerURL: "https://second-issuer.example.com",
					},
				},
			},
			mockSetup: func(mockFetcher *auth.MockOpenIDConfigFetcher) {
				oidcConfig := &auth.OIDCConfiguration{
					AuthEndpoint:  "https://first-issuer.example.com/auth",
					TokenEndpoint: "https://first-issuer.example.com/token",
				}
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://first-issuer.example.com").Return(oidcConfig, nil)
			},
			expectLoginClient: "first-client-id",
			expectLoginAuthz:  "https://first-issuer.example.com/auth",
			expectLoginToken:  "https://first-issuer.example.com/token",
		},
		{
			name:        "successful handler with internal identity provider",
			tfeBasePath: "/tfe",
			cfg: &config.Config{
				TharsisAPIURL:                        "https://api.example.com",
				CLILoginOIDCScopes:                   "openid profile",
				OIDCInternalIdentityProviderClientID: "internal-client-id",
				OauthProviders:                       []config.IdpConfig{},
			},
			expectLoginClient: "internal-client-id",
			expectLoginAuthz:  "https://api.example.com/oauth/authorize",
			expectLoginToken:  "https://api.example.com/oauth/token",
		},
		{
			name:        "client ID not found in oauth providers",
			tfeBasePath: "/tfe",
			cfg: &config.Config{
				TharsisAPIURL:        "https://api.example.com",
				CLILoginOIDCScopes:   "openid profile",
				CLILoginOIDCClientID: "nonexistent-client-id",
				OauthProviders: []config.IdpConfig{
					{
						ClientID:  "test-client-id",
						IssuerURL: "https://issuer.example.com",
					},
				},
			},
			expectErrorMessage: "OIDC Identity Provider not found for TFE login",
		},
		{
			name:        "OIDC config fetch error",
			tfeBasePath: "/tfe",
			cfg: &config.Config{
				TharsisAPIURL:        "https://api.example.com",
				CLILoginOIDCScopes:   "openid profile",
				CLILoginOIDCClientID: "test-client-id",
				OauthProviders: []config.IdpConfig{
					{
						ClientID:  "test-client-id",
						IssuerURL: "https://issuer.example.com",
					},
				},
			},
			mockSetup: func(mockFetcher *auth.MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://issuer.example.com").Return(nil, assert.AnError)
			},
			expectErrorMessage: "failed to get OIDC config for issuer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockFetcher := &auth.MockOpenIDConfigFetcher{}
			logger, _ := logger.NewForTest()

			if test.mockSetup != nil {
				test.mockSetup(mockFetcher)
			}

			handler, err := BuildTFEServiceDiscoveryHandler(
				context.Background(),
				logger,
				test.tfeBasePath,
				mockFetcher,
				test.cfg,
			)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, handler)
			} else {
				require.NoError(t, err)
				require.NotNil(t, handler)

				// Test the handler by making a request
				req := httptest.NewRequest("GET", "/", nil)
				recorder := httptest.NewRecorder()

				handler(recorder, req)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify expected fields
				if test.expectModulesV1 != "" {
					assert.Equal(t, test.expectModulesV1, response["modules.v1"])
				}
				if test.expectProvidersV1 != "" {
					assert.Equal(t, test.expectProvidersV1, response["providers.v1"])
				}
				if test.expectStateV2 != "" {
					assert.Equal(t, test.expectStateV2, response["state.v2"])
				}
				if test.expectTfeV2 != "" {
					assert.Equal(t, test.expectTfeV2, response["tfe.v2"])
					assert.Equal(t, test.expectTfeV2, response["tfe.v2.1"])
					assert.Equal(t, test.expectTfeV2, response["tfe.v2.2"])
				}

				// Verify login.v1 section
				loginV1, ok := response["login.v1"].(map[string]interface{})
				require.True(t, ok)

				if test.expectLoginClient != "" {
					assert.Equal(t, test.expectLoginClient, loginV1["client"])
				}
				if test.expectLoginAuthz != "" {
					assert.Equal(t, test.expectLoginAuthz, loginV1["authz"])
				}
				if test.expectLoginToken != "" {
					assert.Equal(t, test.expectLoginToken, loginV1["token"])
				}

				// Verify grant_types and scopes
				grantTypes, ok := loginV1["grant_types"].([]interface{})
				require.True(t, ok)
				assert.Contains(t, grantTypes, "authz_code")

				scopes, ok := loginV1["scopes"].([]interface{})
				require.True(t, ok)
				assert.Contains(t, scopes, test.cfg.CLILoginOIDCScopes)
			}
		})
	}
}
