package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hashicorp/jsonapi"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
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
		c.respWriter.RespondWithError(w, err)
		return
	}

	if req.ServiceAccountPath == nil {
		c.respWriter.RespondWithError(w, errors.NewError(errors.EInvalid, "ServiceAccountPath field is required"))
		return
	}

	if req.Token == nil {
		c.respWriter.RespondWithError(w, errors.NewError(errors.EInvalid, "Token field is required"))
		return
	}

	resp, err := c.saService.Login(r.Context(), &serviceaccount.LoginInput{
		ServiceAccount: *req.ServiceAccountPath,
		Token:          []byte(*req.Token),
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	jsonAPIResp := &ServiceAccountLoginResponse{ID: uuid.New().String(), Token: string(resp.Token)}

	c.respWriter.RespondWithJSONAPI(w, jsonAPIResp, http.StatusCreated)
}
