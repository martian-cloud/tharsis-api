package tfe

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

type workspaceController struct {
	respWriter             response.Writer
	logger                 logger.Logger
	runService             run.Service
	workspaceService       workspace.Service
	groupService           group.Service
	managedIdentityService managedidentity.Service
	signingKeyManager      auth.SigningKeyManager
	variableService        variable.Service
	tharsisAPIURL          string
	tfeVersionedPath       string
	// stateVersionMaxUploadSize caps a state version create request, whose body carries the whole state.
	stateVersionMaxUploadSize int
}

// NewWorkspaceController creates an instance of workspaceController
func NewWorkspaceController(
	logger logger.Logger,
	respWriter response.Writer,
	runService run.Service,
	workspaceService workspace.Service,
	groupService group.Service,
	managedIdentityService managedidentity.Service,
	signingKeyManager auth.SigningKeyManager,
	variableService variable.Service,
	tharsisAPIURL string,
	tfeVersionedPath string,
	stateVersionMaxUploadSize int,
) controllers.Controller {
	return &workspaceController{
		respWriter,
		logger,
		runService,
		workspaceService,
		groupService,
		managedIdentityService,
		signingKeyManager,
		variableService,
		tharsisAPIURL,
		tfeVersionedPath,
		stateVersionMaxUploadSize,
	}
}

const (
	// defaultRunsPageSize and maxRunsPageSize are TFE's page sizes for listing a workspace's runs.
	defaultRunsPageSize = 20
	maxRunsPageSize     = 100

	// maxRunsSkipped caps how many runs a request may page past. Run pagination is cursor based, so
	// reaching a later page means walking the cursors of every page before it.
	maxRunsSkipped = 1000
)

// RegisterRoutes adds routes to the router.
func (c *workspaceController) RegisterRoutes(router chi.Router) {
	router.Get("/workspaces/{workspaceId}", c.GetWorkspaceByID)
	router.Get("/workspaces/{workspaceId}/runs", c.GetWorkspaceRuns)
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

	workspace, err := c.workspaceService.GetWorkspaceByTRN(r.Context(), trn.TypeWorkspace.Build(path))
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) GetWorkspaceByID(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) GetWorkspaceVariables(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	variables, err := c.variableService.GetVariables(r.Context(), workspace.FullPath)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	var tfeVariables []*Variable
	for _, v := range variables {
		vCopy := v

		if vCopy.Value == nil {
			c.respWriter.RespondWithError(r.Context(), w, errors.New("Subject does not have the required access level to view variable values", errors.WithErrorCode(errors.EForbidden)))
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

	c.respWriter.RespondWithPaginatedJSONAPI(r.Context(), w, variableList, http.StatusOK)
}

func (c *workspaceController) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	org := chi.URLParam(r, "organization")

	var req gotfe.WorkspaceCreateOptions
	if err := unmarshalJSONAPIRequest(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	if req.Name == nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.New("workspace name is required", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	// Get group
	group, err := c.groupService.GetGroupByTRN(r.Context(), trn.TypeGroup.Build(convertOrgToGroupPath(org)))
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	options := &models.Workspace{
		Name:    *req.Name,
		GroupID: group.Metadata.ID,
	}

	ws, err := c.workspaceService.CreateWorkspace(r.Context(), options)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisWorkspaceToWorkspace(ws), http.StatusCreated)
}

func (c *workspaceController) GetWorkspaceCurrentStateVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	sv, err := c.workspaceService.GetCurrentStateVersion(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	if sv == nil {
		c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusNotFound)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL, c.tfeVersionedPath), http.StatusOK)
}

// GetWorkspaceRuns lists a workspace's runs, newest first, a page at a time as go-tfe's Runs.List asks
// for them with page[number] and page[size]. The remote backend walks these pages to report a queued
// run's position in the workspace queue.
func (c *workspaceController) GetWorkspaceRuns(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	pageNumber, err := parsePageParam(r, "page[number]", 1)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}
	pageSize, err := parsePageParam(r, "page[size]", defaultRunsPageSize)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}
	pageSize = min(pageSize, maxRunsPageSize)

	skip := (pageNumber - 1) * pageSize // int is 64 bits on supported platforms
	if skip > maxRunsSkipped {
		c.respWriter.RespondWithError(r.Context(), w, errors.New(
			"page[number] is too large, at most %d runs can be paged past", maxRunsSkipped, errors.WithErrorCode(errors.EInvalid)))
		return
	}

	ws, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	// Run pagination is cursor based, so a numbered page is reached by walking the cursors of the
	// runs before it, a page at a time to keep memory bounded.
	sortBy := db.RunSortableFieldCreatedAtDesc
	var after *string
	var result *db.RunsResult
	pastEnd := false
	for skip > 0 && !pastEnd {
		n := min(skip, maxRunsPageSize)
		first := int32(n)
		result, err = c.runService.GetRuns(r.Context(), &run.GetRunsInput{
			Sort:              &sortBy,
			PaginationOptions: &pagination.Options{First: &first, After: after},
			Workspace:         ws,
		})
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}

		if len(result.Runs) < n {
			pastEnd = true
			break
		}

		after, err = result.PageInfo.Cursor(result.Runs[n-1])
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}
		skip -= n
	}

	var pageRuns []*models.Run
	if !pastEnd {
		first := int32(pageSize)
		result, err = c.runService.GetRuns(r.Context(), &run.GetRunsInput{
			Sort:              &sortBy,
			PaginationOptions: &pagination.Options{First: &first, After: after},
			Workspace:         ws,
		})
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}
		pageRuns = result.Runs
	}

	totalCount, err := result.PageInfo.TotalCount(r.Context())
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	tfeRuns := []*Run{}
	for _, item := range pageRuns {
		tfeRuns = append(tfeRuns, TharsisRunToRun(item))
	}

	runList := &RunList{
		Pagination: newPagination(pageNumber, pageSize, int(totalCount)),
		Items:      tfeRuns,
	}

	c.respWriter.RespondWithPaginatedJSONAPI(r.Context(), w, runList, http.StatusOK)
}

func (c *workspaceController) LockWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	workspace, err = c.workspaceService.LockWorkspace(r.Context(), workspace)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, TharsisErrorToTfeError(err))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) UnlockWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	workspace, err := c.workspaceService.GetWorkspaceByID(r.Context(), workspaceID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	workspace, err = c.workspaceService.UnlockWorkspace(r.Context(), workspace)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, TharsisErrorToTfeError(err))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisWorkspaceToWorkspace(workspace), http.StatusOK)
}

func (c *workspaceController) CreateStateVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	// The state is base64 encoded inside the JSON:API document, so the whole request is capped.
	body := http.MaxBytesReader(w, r.Body, int64(c.stateVersionMaxUploadSize))
	defer body.Close()

	var req gotfe.StateVersionCreateOptions
	if err := unmarshalJSONAPIRequest(body, &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
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
		c.respWriter.RespondWithJSON(r.Context(), w, &errorPayload, http.StatusBadRequest)
		return
	}

	if err := verifyStateVersionAttributes(&req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	options := models.StateVersion{WorkspaceID: workspaceID}

	// json-state is the "terraform show -json" rendering the CLI produces alongside the state. It is
	// passed through as-is: the service decodes it and stores it in the same transaction as the state
	// version, so a malformed or unstorable rendering leaves nothing behind and the created state
	// version already carries hosted-json-state-download-url in this response.
	sv, err := c.workspaceService.CreateStateVersion(r.Context(), &options, *req.State, req.JSONState)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL, c.tfeVersionedPath), http.StatusCreated)
}

// verifyStateVersionAttributes checks a state version create request's md5, serial and lineage against the
// state it carries: TFE requires them to describe that state, and the Terraform CLI computes all three
// from the state bytes it sends, so a mismatch means the state was not the one the client meant to send.
// Each is checked only when sent.
func verifyStateVersionAttributes(req *gotfe.StateVersionCreateOptions) error {
	if req.MD5 == nil && req.Serial == nil && req.Lineage == nil {
		return nil
	}

	state, err := base64.StdEncoding.DecodeString(*req.State)
	if err != nil {
		return errors.Wrap(err, "state is not valid base64", errors.WithErrorCode(errors.EInvalid))
	}

	if req.MD5 != nil {
		if sum := md5.Sum(state); !strings.EqualFold(hex.EncodeToString(sum[:]), *req.MD5) {
			return errors.New("md5 does not match the state", errors.WithErrorCode(errors.EInvalid))
		}
	}

	if req.Serial == nil && req.Lineage == nil {
		return nil
	}

	var stateFile struct {
		Serial  *uint64 `json:"serial"`
		Lineage *string `json:"lineage"`
	}
	if err := json.Unmarshal(state, &stateFile); err != nil {
		return errors.Wrap(err, "state is not valid JSON", errors.WithErrorCode(errors.EInvalid))
	}
	if req.Serial != nil && (stateFile.Serial == nil || *req.Serial < 0 || uint64(*req.Serial) != *stateFile.Serial) {
		return errors.New("serial does not match the state", errors.WithErrorCode(errors.EInvalid))
	}
	if req.Lineage != nil && (stateFile.Lineage == nil || *req.Lineage != *stateFile.Lineage) {
		return errors.New("lineage does not match the state", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}

// parsePageParam reads a positive 32-bit page[number] or page[size] query parameter, defaulting when
// absent. The bound keeps the page arithmetic from overflowing.
func parsePageParam(r *http.Request, name string, defaultValue int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 1 {
		return 0, errors.New("%s must be a positive 32-bit integer", name, errors.WithErrorCode(errors.EInvalid))
	}
	return int(value), nil
}

// newPagination describes page pageNumber of totalCount items split into pages of pageSize.
func newPagination(pageNumber, pageSize, totalCount int) *gotfe.Pagination {
	totalPages := (totalCount + pageSize - 1) / pageSize
	p := &gotfe.Pagination{
		CurrentPage: pageNumber,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
	if pageNumber > 1 {
		p.PreviousPage = pageNumber - 1
	}
	if pageNumber < totalPages {
		p.NextPage = pageNumber + 1
	}
	return p
}

func (c *workspaceController) DownloadConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	configVersionID := gid.FromGlobalID(chi.URLParam(r, "configurationVersionId"))

	result, err := c.workspaceService.GetConfigurationVersionContent(r.Context(), configVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	defer result.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if result.ContentLength > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(result.ContentLength, 10))
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result.Body); err != nil {
		// Status 200 and part of the body are already on the wire, so we can no
		// longer change the status or send an error document. Log it and abort the
		// connection so the client sees a failed transfer, not a clean EOF.
		c.logger.WithContextFields(r.Context()).Errorf(
			"failed to stream configuration version %s: %v", configVersionID, err)
		// Use the http abort handler to terminate the connection gracefully
		panic(http.ErrAbortHandler)
	}
}

func (c *workspaceController) CreateConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	workspaceID := gid.FromGlobalID(chi.URLParam(r, "workspaceId"))

	var req gotfe.ConfigurationVersionCreateOptions
	if err := unmarshalJSONAPIRequest(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
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
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	token, err := c.createUploadToken(r.Context(), cv.GetGlobalID())
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	uploadURL := fmt.Sprintf(
		"%s%s/workspaces/%s/configuration-versions/%s/upload",
		c.tharsisAPIURL,
		c.tfeVersionedPath,
		gid.ToGlobalID(types.WorkspaceModelType, cv.WorkspaceID),
		token,
	)

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisCVToCV(cv, uploadURL), http.StatusCreated)
}

func (c *workspaceController) GetConfigurationVersion(w http.ResponseWriter, r *http.Request) {
	configurationVersionID := gid.FromGlobalID(chi.URLParam(r, "configurationVersionId"))

	cv, err := c.workspaceService.GetConfigurationVersionByID(r.Context(), configurationVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	uploadURL := ""

	caller := auth.GetCaller(r.Context())
	// Only return upload URL if the caller has the required permission
	if caller != nil && caller.RequirePermission(r.Context(), models.UpdateConfigurationVersionPermission, auth.WithWorkspaceID(cv.WorkspaceID)) == nil {
		token, err := c.createUploadToken(r.Context(), cv.GetGlobalID())
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
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

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisCVToCV(cv, uploadURL), http.StatusOK)
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
		sub, err := c.verifyUploadToken(r.Context(), token)
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
		sub, err := c.verifyUploadToken(r.Context(), token)
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "invalid token", errors.WithErrorCode(errors.EUnauthorized)))
			return
		}

		configurationVersionID = sub

		// Use system caller to invoke service since the authentication token has already been checked
		ctx = auth.WithCaller(r.Context(), &auth.SystemCaller{})
	}

	if err := c.workspaceService.UploadConfigurationVersion(ctx, gid.FromGlobalID(configurationVersionID), r.Body); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}

func (c *workspaceController) createUploadToken(ctx context.Context, subjectClaim string) ([]byte, error) {
	return c.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Subject:    subjectClaim,
		Expiration: ptr.Time(time.Now().Add(5 * time.Minute)),
	})
}

func (c *workspaceController) verifyUploadToken(ctx context.Context, token string) (string, error) {
	output, err := c.signingKeyManager.VerifyToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("configuration version upload token is invalid: %w", err)
	}

	return output.Token.Subject(), nil
}
