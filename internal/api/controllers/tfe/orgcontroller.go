package tfe

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// EntitlementSettings represents the entitlements for a particular user
var EntitlementSettings = gotfe.Entitlements{
	ID:                    "1",
	Agents:                true,
	AuditLogging:          true,
	CostEstimation:        true,
	Operations:            true,
	PrivateModuleRegistry: true,
	SSO:                   true,
	Sentinel:              true,
	StateStorage:          true,
	Teams:                 true,
	VCSIntegrations:       true,
}

type orgController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	runService        run.Service
	groupService      group.Service
}

// NewOrgController creates an instance of orgController
func NewOrgController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	runService run.Service,
	groupService group.Service,
) controllers.Controller {
	return &orgController{respWriter, jwtAuthMiddleware, logger, runService, groupService}
}

// RegisterRoutes adds health routes to the router
func (c *orgController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/organizations/{organization}/entitlement-set", c.GetEntitlements)
	router.Get("/organizations/{organization}/runs/queue", c.GetRunQueue)
}

func (c *orgController) GetEntitlements(w http.ResponseWriter, r *http.Request) {
	c.respWriter.RespondWithJSONAPI(r.Context(), w, &EntitlementSettings, http.StatusOK)
}

func (c *orgController) GetRunQueue(w http.ResponseWriter, r *http.Request) {
	org := chi.URLParam(r, "organization")

	groupPath := convertOrgToGroupPath(org)

	group, err := c.groupService.GetGroupByTRN(r.Context(), types.GroupModelType.BuildTRN(groupPath))
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	sortBy := db.RunSortableFieldCreatedAtDesc
	result, err := c.runService.GetRuns(r.Context(), &run.GetRunsInput{
		Sort:  &sortBy,
		Group: group,
	})
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	var tfeRuns []*Run
	for _, run := range result.Runs {
		if run.Status == models.RunPlanQueued || run.Status == models.RunApplyQueued {
			r := run
			tfeRuns = append(tfeRuns, TharsisRunToRun(&r))
		}
	}

	runQueue := RunQueue{
		Pagination: &gotfe.Pagination{
			CurrentPage: 1,
		},
		Items: tfeRuns,
	}

	c.respWriter.RespondWithPaginatedJSONAPI(r.Context(), w, runQueue, http.StatusOK)
}
