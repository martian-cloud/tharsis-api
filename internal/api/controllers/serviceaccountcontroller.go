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
	// No auth header required since this is a login endpoint
	router.Post("/serviceaccounts/login", c.Login)
}

func (c *serviceAccountController) Login(w http.ResponseWriter, r *http.Request) {
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

	resp, err := c.saService.CreateToken(r.Context(), &serviceaccount.CreateTokenInput{
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
