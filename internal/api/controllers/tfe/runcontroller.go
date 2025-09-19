package tfe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type runController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	signingKeyManager auth.SigningKeyManager
	logger            logger.Logger
	runService        run.Service
	tharsisAPIURL     string
}

// NewRunController creates an instance of runController
func NewRunController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	signingKeyManager auth.SigningKeyManager,
	runService run.Service,
	tharsisAPIURL string,
) controllers.Controller {
	return &runController{
		respWriter,
		jwtAuthMiddleware,
		signingKeyManager,
		logger,
		runService,
		tharsisAPIURL,
	}
}

// RegisterRoutes adds routes to the router
func (c *runController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/runs/{id}", c.GetRun)
	router.Get("/plans/{id}", c.GetPlan)
	router.Get("/applies/{id}", c.GetApply)

	router.Get("/plans/{id}/content", c.DownloadPlan)

	router.Post("/runs", c.CreateRun)
	router.Post("/runs/{id}/actions/apply", c.ApplyRun)
	router.Post("/runs/{id}/actions/cancel", c.CancelRun)
}

func (c *runController) DownloadPlan(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	result, err := c.runService.DownloadPlan(r.Context(), planID)
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

func (c *runController) CreateRun(w http.ResponseWriter, r *http.Request) {
	var req gotfe.RunCreateOptions

	// Read the response for re-use if variables use broken api
	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	if err = jsonapi.UnmarshalPayload(io.NopCloser(bytes.NewReader(body)), &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	variables, err := parseRunVariables(req, body)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	options := &run.CreateRunInput{
		WorkspaceID:     gid.FromGlobalID(req.Workspace.ID),
		Comment:         req.Message,
		Variables:       variables,
		TargetAddresses: req.TargetAddrs,
	}
	if req.ConfigurationVersion != nil {
		id := gid.FromGlobalID(req.ConfigurationVersion.ID)
		options.ConfigurationVersionID = &id
	}
	if req.IsDestroy != nil {
		options.IsDestroy = *req.IsDestroy
	}
	if req.Refresh != nil {
		options.Refresh = *req.Refresh
	}
	if req.RefreshOnly != nil {
		options.RefreshOnly = *req.RefreshOnly
	}

	run, err := c.runService.CreateRun(r.Context(), options)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisRunToRun(run), http.StatusCreated)
}

func parseRunVariables(req gotfe.RunCreateOptions, body []byte) ([]run.Variable, error) {
	variables := []run.Variable{}

	for _, v := range req.Variables {
		if v == nil || v.Key == "" {
			continue
		}
		val := v.Value
		variables = append(variables, run.Variable{
			Key:      v.Key,
			Value:    &val,
			Category: models.TerraformVariableCategory,
			Hcl:      true,
		})
	}

	// If variables are in the req and none were parsed, it is using a terraform version that's broken
	if len(req.Variables) > 0 && len(variables) == 0 {
		var altReq struct {
			Data struct {
				Attributes struct {
					Variables []*struct {
						Key   string `json:"Key,omitempty"`
						Value string `json:"Value,omitempty"`
					} `json:"variables,omitempty"`
				} `json:"attributes,omitempty"`
			} `json:"data,omitempty"`
		}

		if err := json.Unmarshal(body, &altReq); err != nil {
			// We should never hit this error since jsonapi already does a json decode
			return nil, fmt.Errorf("invalid create run request: %w", err)
		}

		for _, v := range altReq.Data.Attributes.Variables {
			if v == nil {
				continue
			}

			val := v.Value
			variables = append(variables, run.Variable{
				Key:      v.Key,
				Value:    &val,
				Category: models.TerraformVariableCategory,
				Hcl:      true,
			})
		}
	}

	return variables, nil
}

func (c *runController) ApplyRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	var req gotfe.RunApplyOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	run, err := c.runService.ApplyRun(r.Context(), runID, req.Comment)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisRunToRun(run), http.StatusOK)
}

func (c *runController) CancelRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	var req gotfe.RunCancelOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	run, err := c.runService.CancelRun(r.Context(), &run.CancelRunInput{
		RunID:   runID,
		Comment: req.Comment,
		// The REST API does not support the force option to cancel a run.
		// Only the GraphQL interface supports the force option.
	})
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisRunToRun(run), http.StatusOK)
}

func (c *runController) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByID(r.Context(), runID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisRunToRun(run), http.StatusOK)
}

func (c *runController) GetPlan(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	plan, err := c.runService.GetPlanByID(r.Context(), planID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	job, err := c.runService.GetLatestJobForPlan(r.Context(), planID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	resp := &gotfe.Plan{
		ID:                   plan.GetGlobalID(),
		Status:               gotfe.PlanStatus(plan.Status),
		HasChanges:           plan.HasChanges,
		ResourceAdditions:    int(plan.Summary.ResourceAdditions),
		ResourceChanges:      int(plan.Summary.ResourceChanges),
		ResourceDestructions: int(plan.Summary.ResourceDestructions),
	}

	if job != nil && c.tharsisAPIURL != "" {
		token, err := c.createJobLogToken(r.Context(), job)
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}

		resp.LogReadURL = fmt.Sprintf("%s/v1/jobs/%s/logs/%s", c.tharsisAPIURL, job.GetGlobalID(), string(token))
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, resp, http.StatusOK)
}

func (c *runController) GetApply(w http.ResponseWriter, r *http.Request) {
	applyID := gid.FromGlobalID(chi.URLParam(r, "id"))

	apply, err := c.runService.GetApplyByID(r.Context(), applyID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	job, err := c.runService.GetLatestJobForApply(r.Context(), applyID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	resp := &gotfe.Apply{
		ID:     apply.GetGlobalID(),
		Status: gotfe.ApplyStatus(apply.Status),
	}

	if job != nil && c.tharsisAPIURL != "" {
		token, err := c.createJobLogToken(r.Context(), job)
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}

		resp.LogReadURL = fmt.Sprintf("%s/v1/jobs/%s/logs/%s", c.tharsisAPIURL, job.GetGlobalID(), string(token))
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, resp, http.StatusOK)
}

func (c *runController) createJobLogToken(ctx context.Context, job *models.Job) ([]byte, error) {
	return c.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Subject:    job.Metadata.ID,
		Expiration: ptr.Time(time.Now().Add(5 * time.Minute)),
	})
}
