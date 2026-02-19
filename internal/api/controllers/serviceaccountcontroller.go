package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hashicorp/jsonapi"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// clientCredentialsTokenResponse is the token response per RFC 6749 Section 5.1
type clientCredentialsTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"` // Required by RFC 6749
	ExpiresIn   int    `json:"expires_in"`
}

type serviceAccountController struct {
	respWriter response.Writer
	logger     logger.Logger
	saService  serviceaccount.Service
}

// NewServiceAccountController handles service account REST requests
func NewServiceAccountController(
	logger logger.Logger,
	respWriter response.Writer,
	saService serviceaccount.Service,
) Controller {
	return &serviceAccountController{respWriter, logger, saService}
}

// RegisterRoutes adds service account routes to the router
func (c *serviceAccountController) RegisterRoutes(router chi.Router) {
	// No auth header required since these are login endpoints
	// Deprecated: /serviceaccounts/login will be removed in a future release. Use the GraphQL or gRPC login methods instead.
	router.Post("/serviceaccounts/login", c.Login)
	router.Post("/serviceaccounts/token", c.ClientCredentialsToken)
}

func (c *serviceAccountController) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")

	var req ServiceAccountLoginOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	if req.ServiceAccountPath == nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.New("ServiceAccountPath field is required", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	if req.Token == nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.New("Token field is required", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	resp, err := c.saService.CreateOIDCToken(r.Context(), &serviceaccount.CreateOIDCTokenInput{
		ServiceAccountPublicID: types.ServiceAccountModelType.BuildTRN(*req.ServiceAccountPath),
		Token:                  []byte(*req.Token),
	})
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	jsonAPIResp := &ServiceAccountLoginResponse{ID: uuid.New().String(), Token: string(resp.Token)}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, jsonAPIResp, http.StatusCreated)
}

// ClientCredentialsToken handles client credentials grant for service accounts
func (c *serviceAccountController) ClientCredentialsToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.ParseForm()

	grantType := r.Form.Get("grant_type")
	if grantType != "client_credentials" {
		c.respWriter.RespondWithError(ctx, w, errors.New("unsupported grant_type, only client_credentials is supported", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")

	response, err := c.saService.CreateClientCredentialsToken(ctx, &serviceaccount.CreateClientCredentialsTokenInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	c.respWriter.RespondWithJSON(ctx, w, &clientCredentialsTokenResponse{
		AccessToken: string(response.Token),
		TokenType:   "Bearer",
		ExpiresIn:   int(response.ExpiresIn),
	}, http.StatusOK)
}
