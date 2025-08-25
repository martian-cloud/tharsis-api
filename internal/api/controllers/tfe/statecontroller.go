package tfe

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type stateController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	workspaceService  workspace.Service
	tharsisAPIURL     string
	tfeVersionedPath  string
}

// NewStateController creates an instance of stateController
func NewStateController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	workspaceService workspace.Service,
	tharsisAPIURL string,
	tfeVersionedPath string,
) controllers.Controller {
	return &stateController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		workspaceService,
		tharsisAPIURL,
		tfeVersionedPath,
	}
}

// RegisterRoutes adds routes to the router.
func (c *stateController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/state-versions/{stateVersionId}", c.GetStateVersion)
	router.Get("/state-versions/{stateVersionId}/content", c.DownloadStateVersion)
}

func (c *stateController) GetStateVersion(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	sv, err := c.workspaceService.GetStateVersionByID(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL, c.tfeVersionedPath), http.StatusOK)
}

func (c *stateController) DownloadStateVersion(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	result, err := c.workspaceService.GetStateVersionContent(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	defer result.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}
}
