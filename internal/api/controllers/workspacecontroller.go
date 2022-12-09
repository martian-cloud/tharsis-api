package controllers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

type workspaceController struct {
	respWriter             response.Writer
	jwtAuthMiddleware      middleware.Handler
	logger                 logger.Logger
	runService             run.Service
	workspaceService       workspace.Service
	groupService           group.Service
	managedIdentityService managedidentity.Service
	variableService        variable.Service
	tharsisAPIURL          string
}

// NewWorkspaceController creates an instance of workspaceController
func NewWorkspaceController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	runService run.Service,
	workspaceService workspace.Service,
	groupService group.Service,
	managedIdentityService managedidentity.Service,
	variableService variable.Service,
	tharsisAPIURL string,
) Controller {
	return &workspaceController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		runService,
		workspaceService,
		groupService,
		managedIdentityService,
		variableService,
		tharsisAPIURL,
	}
}

// RegisterRoutes adds routes to the router.
func (c *workspaceController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/workspaces/{workspaceId}", c.GetWorkspaceByID)
	router.Get("/workspaces/{workspace}/runs", c.GetWorkspaceRuns)
	router.Get("/workspaces/{workspaceId}/vars", c.GetWorkspaceVariables)
	router.Get("/organizations/{organization}/workspaces/{workspace}", c.GetWorkspace)
	router.Get("/workspaces/{workspaceId}/current-state-version", c.GetWorkspaceCurrentStateVersion)
	router.Get("/configuration-versions/{configurationVersionId}", c.GetConfigurationVersion)
	router.Get("/state-versions/{stateVersionId}", c.GetStateVersion)
	router.Get("/state-versions/{stateVersionId}/content", c.GetStateVersionContent)

	router.Get("/configuration-versions/{configurationVersionId}/content", c.DownloadConfigurationVersion)
	router.Get("/state-versions/{stateVersionId}/content", c.DownloadStateVersion)

	router.Put("/workspaces/{workspaceId}/configuration-versions/{configurationVersionId}/upload", c.UploadConfigurationVersion)

	router.Post("/workspaces/{workspaceId}/state-versions", c.CreateStateVersion)
	router.Post("/organizations/{organization}/workspaces", c.CreateWorkspace)
	router.Post("/workspaces/{workspaceId}/actions/lock", c.LockWorkspace)
	router.Post("/workspaces/{workspaceId}/actions/unlock", c.UnlockWorkspace)
	router.Post("/workspaces/{workspaceId}/configuration-versions", c.CreateConfigurationVersion)
}

func (c *workspaceController) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	org := chi.URLParam(r, "organization")
	workspaceName := chi.URLParam(r, "workspace")

	path := fmt.Sprintf("%s/%s", convertOrgToGroupPath(org), workspaceName)

	workspace, err := c.workspaceService.GetWorkspaceByFullPath(r.Context(), path)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) GetWorkspaceByID(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) GetWorkspaceVariables(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	variables, err := c.variableService.GetVariables(r.Context(), workspace.FullPath)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	var tfeVariables []*Variable
	for _, v := range variables {
		vCopy := v

		if vCopy.Value == nil {
			c.respWriter.RespondWithError(w, errors.NewError(errors.EForbidden, "Subject does not have the required access level to view variable values"))
			return
		}

		tfeVariables = append(tfeVariables, TharsisVariableToVariable(&vCopy, workspace))
	}

	variableList := &VariableList{
		Pagination: &gotfe.Pagination{
			CurrentPage: 1,
		},
		Items: tfeVariables,
	}

	c.respWriter.RespondWithPaginatedJSONAPI(w, variableList, http.StatusOK)
}

func (c *workspaceController) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	org := chi.URLParam(r, "organization")

	var req gotfe.WorkspaceCreateOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	// Get group
	group, err := c.groupService.GetGroupByFullPath(r.Context(), convertOrgToGroupPath(org))
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	options := &models.Workspace{
		Name:    *req.Name,
		GroupID: group.Metadata.ID,
	}

	ws, err := c.workspaceService.CreateWorkspace(r.Context(), options)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisWorkspaceToWorkspace(ws), http.StatusCreated)
}

func (c *workspaceController) GetWorkspaceCurrentStateVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	sv, err := c.workspaceService.GetCurrentStateVersion(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	if sv == nil {
		c.respWriter.RespondWithJSONAPI(w, nil, http.StatusNotFound)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL), http.StatusOK)
}

func (c *workspaceController) GetWorkspaceRuns(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	ws, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	runs, err := c.runService.GetRuns(r.Context(), &run.GetRunsInput{
		Workspace: ws,
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, runs.Runs, http.StatusOK)
}

func (c *workspaceController) GetStateVersion(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	sv, err := c.workspaceService.GetStateVersion(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL), http.StatusOK)
}

func (c *workspaceController) LockWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	workspace, err = c.workspaceService.LockWorkspace(r.Context(), workspace)
	if err != nil {
		c.respWriter.RespondWithError(w, TharsisErrorToTfeError(err))
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) UnlockWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	workspace, err = c.workspaceService.UnlockWorkspace(r.Context(), workspace)
	if err != nil {
		c.respWriter.RespondWithError(w, TharsisErrorToTfeError(err))
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) CreateStateVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	var req gotfe.StateVersionCreateOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	options := models.StateVersion{WorkspaceID: workspaceID}

	sv, err := c.workspaceService.CreateStateVersion(r.Context(), &options, req.State)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL), http.StatusCreated)
}

func (c *workspaceController) GetStateVersionContent(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	result, err := c.workspaceService.GetStateVersionContent(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	defer result.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}
}

func (c *workspaceController) DownloadStateVersion(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	result, err := c.workspaceService.GetStateVersionContent(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	defer result.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}
}

func (c *workspaceController) DownloadConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	configVersionID := gid.FromGlobalID(chi.URLParam(r, "configurationVersionId"))

	result, err := c.workspaceService.GetConfigurationVersionContent(r.Context(), configVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	defer result.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}
}

func (c *workspaceController) CreateConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	var req gotfe.ConfigurationVersionCreateOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	options := &workspace.CreateConfigurationVersionInput{
		WorkspaceID: workspaceID,
	}

	if req.Speculative != nil {
		options.Speculative = *req.Speculative
	}

	cv, err := c.workspaceService.CreateConfigurationVersion(r.Context(), options)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisCVToCV(cv, c.tharsisAPIURL), http.StatusCreated)
}

func (c *workspaceController) GetConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	configurationVersionID := gid.FromGlobalID(chi.URLParam(r, "configurationVersionId"))

	cv, err := c.workspaceService.GetConfigurationVersion(r.Context(), configurationVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisCVToCV(cv, c.tharsisAPIURL), http.StatusOK)
}

func (c *workspaceController) UploadConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	configurationVersionID := gid.FromGlobalID(chi.URLParam(r, "configurationVersionId"))

	defer r.Body.Close()

	err := c.workspaceService.UploadConfigurationVersion(r.Context(), configurationVersionID, r.Body)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}
