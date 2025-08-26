package controllers

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	tfjson "github.com/hashicorp/terraform-json"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type planWithProviderSchemas struct {
	Plan            *tfjson.Plan            `json:"plan"`
	ProviderSchemas *tfjson.ProviderSchemas `json:"provider_schemas"`
}

type runController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	runService        run.Service
}

// NewRunController creates an instance of runController
func NewRunController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	runService run.Service,
) Controller {
	return &runController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		runService,
	}
}

// RegisterRoutes adds routes to the router
func (c *runController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Put("/plans/{id}/content", c.UploadPlanBinary)
	router.Put("/plans/{id}/content.json", c.UploadPlanData)
}

func (c *runController) UploadPlanBinary(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	defer r.Body.Close()

	err := c.runService.UploadPlanBinary(r.Context(), planID, r.Body)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}

func (c *runController) UploadPlanData(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	defer r.Body.Close()

	// Check that the server actually sent compressed data
	var reader io.ReadCloser
	var err error
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(r.Body)
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, fmt.Errorf("failed to create gzip reader: %w", err))
			return
		}
		defer reader.Close()
	default:
		reader = r.Body
	}

	var planData planWithProviderSchemas
	if err = json.NewDecoder(reader).Decode(&planData); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, fmt.Errorf("failed to decode plan data: %w", err))
		return
	}

	err = c.runService.ProcessPlanData(r.Context(), planID, planData.Plan, planData.ProviderSchemas)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, fmt.Errorf("failed to process plan data: %w", err))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}
