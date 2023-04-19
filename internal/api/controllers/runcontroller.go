package controllers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

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

	router.Put("/plans/{id}/content", c.UploadPlan)
}

func (c *runController) UploadPlan(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	defer r.Body.Close()

	err := c.runService.UploadPlan(r.Context(), planID, r.Body)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}
