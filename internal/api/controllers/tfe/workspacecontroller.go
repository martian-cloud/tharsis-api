package tfe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type workspaceController struct {
	respWriter             response.Writer
	logger                 logger.Logger
	runService             run.Service
	workspaceService       workspace.Service
	groupService           group.Service
	managedIdentityService managedidentity.Service
	jwsProvider            jws.Provider
	variableService        variable.Service
	tharsisAPIURL          string
	tfeVersionedPath       string
}

// NewWorkspaceController creates an instance of workspaceController
func NewWorkspaceController(
	logger logger.Logger,
	respWriter response.Writer,
	runService run.Service,
	workspaceService workspace.Service,
	groupService group.Service,
	managedIdentityService managedidentity.Service,
	jwsProvider jws.Provider,
	variableService variable.Service,
	tharsisAPIURL string,
	tfeVersionedPath string,
) controllers.Controller {
	return &workspaceController{
		respWriter,
		logger,
		runService,
		workspaceService,
		groupService,
		managedIdentityService,
		jwsProvider,
		variableService,
		tharsisAPIURL,
		tfeVersionedPath,
	}
}

// RegisterRoutes adds routes to the router.
func (c *workspaceController) RegisterRoutes(router chi.Router) {
	router.Get("/workspaces/{workspaceId}", c.GetWorkspaceByID)
	router.Get("/workspaces/{workspace}/runs", c.GetWorkspaceRuns)
	router.Get("/workspaces/{workspaceId}/vars", c.GetWorkspaceVariables)
	router.Get("/organizations/{organization}/workspaces/{workspace}", c.GetWorkspace)
	router.Get("/workspaces/{workspaceId}/current-state-version", c.GetWorkspaceCurrentStateVersion)
	router.Get("/configuration-versions/{configurationVersionId}", c.GetConfigurationVersion)

	router.Get("/configuration-versions/{configurationVersionId}/content", c.DownloadConfigurationVersion)

	router.Put("/workspaces/{workspaceId}/configuration-versions/{token}/upload", c.UploadConfigurationVersion)

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

	workspace, err := c.workspaceService.GetWorkspaceByTRN(r.Context(), types.WorkspaceModelType.BuildTRN(path))
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
			c.respWriter.RespondWithError(w, errors.New("Subject does not have the required access level to view variable values", errors.WithErrorCode(errors.EForbidden)))
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
	group, err := c.groupService.GetGroupByTRN(r.Context(), types.GroupModelType.BuildTRN(convertOrgToGroupPath(org)))
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

	c.respWriter.RespondWithJSONAPI(w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL, c.tfeVersionedPath), http.StatusOK)
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

	if req.State == nil {
		// This specific error is expected by the tfe library
		errorPayload := jsonapi.ErrorsPayload{
			Errors: []*jsonapi.ErrorObject{
				{
					ID:     "1",
					Title:  "Invalid request",
					Detail: "param is missing or the value is empty: state",
				},
			},
		}
		c.respWriter.RespondWithJSON(w, &errorPayload, http.StatusBadRequest)
		return
	}

	options := models.StateVersion{WorkspaceID: workspaceID}

	sv, err := c.workspaceService.CreateStateVersion(r.Context(), &options, req.State)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL, c.tfeVersionedPath), http.StatusCreated)
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

	token, err := c.createUploadToken(r.Context(), cv.GetGlobalID())
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	uploadURL := fmt.Sprintf(
		"%s%s/workspaces/%s/configuration-versions/%s/upload",
		c.tharsisAPIURL,
		c.tfeVersionedPath,
		gid.ToGlobalID(types.WorkspaceModelType, cv.WorkspaceID),
		token,
	)

	c.respWriter.RespondWithJSONAPI(w, TharsisCVToCV(cv, uploadURL), http.StatusCreated)
}

func (c *workspaceController) GetConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	configurationVersionID := gid.FromGlobalID(chi.URLParam(r, "configurationVersionId"))

	cv, err := c.workspaceService.GetConfigurationVersionByID(r.Context(), configurationVersionID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	uploadURL := ""

	caller := auth.GetCaller(r.Context())
	// Only return upload URL if the caller has the required permission
	if caller != nil && caller.RequirePermission(r.Context(), models.UpdateConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID)) == nil {
		token, err := c.createUploadToken(r.Context(), cv.GetGlobalID())
		if err != nil {
			c.respWriter.RespondWithError(w, err)
			return
		}

		uploadURL = fmt.Sprintf(
			"%s%s/workspaces/%s/configuration-versions/%s/upload",
			c.tharsisAPIURL,
			c.tfeVersionedPath,
			gid.ToGlobalID(types.WorkspaceModelType, cv.WorkspaceID),
			token,
		)
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisCVToCV(cv, uploadURL), http.StatusOK)
}

func (c *workspaceController) UploadConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	var configurationVersionID string
	var ctx context.Context

	defer r.Body.Close()

	caller := auth.GetCaller(r.Context())
	// Check if caller is defined, this is here for backward compatibility and can be removed after the
	// Tharsis SDK has been updated
	if caller != nil {
		token := chi.URLParam(r, "token")

		// Validate token
		sub, err := c.verifyUploadToken(r.Context(), []byte(token))
		if err != nil {
			// If token validation fails then this the token is the configuration version ID
			configurationVersionID = token
		} else {
			configurationVersionID = sub
		}

		ctx = r.Context()
	} else {
		token := chi.URLParam(r, "token")

		// Validate token
		sub, err := c.verifyUploadToken(r.Context(), []byte(token))
		if err != nil {
			c.respWriter.RespondWithError(w, errors.Wrap(err, "invalid token", errors.WithErrorCode(errors.EUnauthorized)))
			return
		}

		configurationVersionID = sub

		// Use system caller to invoke service since the authentication token has already been checked
		ctx = auth.WithCaller(r.Context(), &auth.SystemCaller{})
	}

	if err := c.workspaceService.UploadConfigurationVersion(ctx, gid.FromGlobalID(configurationVersionID), r.Body); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, nil, http.StatusOK)
}

func (c *workspaceController) createUploadToken(ctx context.Context, subjectClaim string) ([]byte, error) {
	currentTimestamp := time.Now().Unix()

	token := jwt.New()

	if err := token.Set(jwt.ExpirationKey, time.Now().Add(5*time.Minute).Unix()); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.NotBeforeKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuedAtKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.SubjectKey, subjectClaim); err != nil {
		return nil, err
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		return nil, err
	}

	return c.jwsProvider.Sign(ctx, payload)
}

func (c *workspaceController) verifyUploadToken(ctx context.Context, token []byte) (string, error) {
	// Validate token
	if err := c.jwsProvider.Verify(ctx, token); err != nil {
		return "", err
	}

	// Parse and validate jwt
	parsedToken, err := jwt.Parse(token, jwt.WithVerify(false), jwt.WithValidate(true))
	if err != nil {
		return "", fmt.Errorf("failed to decode token %w", err)
	}

	return parsedToken.Subject(), nil
}
