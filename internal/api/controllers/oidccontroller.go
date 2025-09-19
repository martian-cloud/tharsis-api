package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
)

type openIDConfig struct {
	Issuer                           string   `json:"issuer"`
	JwksURI                          string   `json:"jwks_uri"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
}

type oidcController struct {
	respWriter        response.Writer
	signingKeyManager auth.SigningKeyManager
}

// NewOIDCController creates an instance of oidcController
func NewOIDCController(respWriter response.Writer, signingKeyManager auth.SigningKeyManager) Controller {
	return &oidcController{
		respWriter:        respWriter,
		signingKeyManager: signingKeyManager,
	}
}

// RegisterRoutes adds health routes to the router
func (c *oidcController) RegisterRoutes(router chi.Router) {
	router.Get("/.well-known/openid-configuration", c.GetOpenIDConfig)
	router.Get("/oauth/discovery/keys", c.GetKeys)
}

func (c *oidcController) GetOpenIDConfig(w http.ResponseWriter, r *http.Request) {
	c.respWriter.RespondWithJSON(r.Context(), w, c.signingKeyManager.GetOpenIDConfig(), http.StatusOK)
}

func (c *oidcController) GetKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := c.signingKeyManager.GetKeys(r.Context())
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	if _, err := w.Write(keys); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
