package tfe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestBuildTFEServiceDiscoveryHandler(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "with oauth providers the login is still served by the api",
			cfg: &config.Config{
				TharsisAPIURL: "https://api.example.com",
				OauthProviders: []config.IdpConfig{
					{
						ClientID:  "test-client-id",
						IssuerURL: "https://issuer.example.com",
					},
				},
			},
		},
		{
			name: "without oauth providers",
			cfg: &config.Config{
				TharsisAPIURL: "https://api.example.com",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := logger.NewForTest()

			handler := BuildTFEServiceDiscoveryHandler(context.Background(), logger, "/tfe", test.cfg)

			req := httptest.NewRequest("GET", "/", nil)
			recorder := httptest.NewRecorder()

			handler(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

			var response map[string]interface{}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))

			assert.Equal(t, "https://api.example.com/v1/module-registry/modules/", response["modules.v1"])
			assert.Equal(t, "https://api.example.com/v1/provider-registry/providers/", response["providers.v1"])
			assert.Equal(t, "https://api.example.com/tfe/v2/", response["state.v2"])
			assert.Equal(t, "https://api.example.com/tfe/v2/", response["tfe.v2"])
			assert.Equal(t, "https://api.example.com/tfe/v2/", response["tfe.v2.1"])
			assert.Equal(t, "https://api.example.com/tfe/v2/", response["tfe.v2.2"])

			loginV1, ok := response["login.v1"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "tharsis", loginV1["client"])
			assert.Equal(t, "https://api.example.com/oauth/authorize", loginV1["authz"])
			assert.Equal(t, "https://api.example.com/oauth/token", loginV1["token"])
			assert.Equal(t, []interface{}{"authz_code"}, loginV1["grant_types"])
			// Terraform joins the scopes with a space, which must match what the authorize endpoint accepts.
			assert.Equal(t, []interface{}{"openid", "tharsis"}, loginV1["scopes"])
		})
	}
}
