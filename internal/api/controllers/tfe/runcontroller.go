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
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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
	// Pass the optional pointer through; run creation resolves nil to true (Terraform's
	// default), matching the GraphQL behavior.
	options.Refresh = req.Refresh
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

func parseRunVariables(req gotfe.RunCreateOptions, body []byte) ([]runvariables.Variable, error) {
	variables := []runvariables.Variable{}

	for _, v := range req.Variables {
		if v == nil || v.Key == "" {
			continue
		}
		val := v.Value
		variables = append(variables, runvariables.Variable{
			Key:      v.Key,
			Value:    &val,
			Category: models.TerraformVariableCategory,
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
			variables = append(variables, runvariables.Variable{
				Key:      v.Key,
				Value:    &val,
				Category: models.TerraformVariableCategory,
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

	run, err := c.runService.GetRunByNodeID(r.Context(), planID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	plan := &run.Plan

	resp := &gotfe.Plan{
		ID:                   plan.GetGlobalID(),
		Status:               gotfe.PlanStatus(plan.Status),
		HasChanges:           plan.HasChanges,
		ResourceAdditions:    int(plan.Summary.ResourceAdditions),
		ResourceChanges:      int(plan.Summary.ResourceChanges),
		ResourceDestructions: int(plan.Summary.ResourceDestructions),
	}

	// Always return a LogReadURL; the Terraform CLI requires one on every plan response,
	// even before the plan's job has been created. The URL is scoped to the run node so
	// the log endpoint can resolve the job lazily once it exists.
	if c.tharsisAPIURL != "" {
		token, err := c.createRunLogToken(r.Context(), run.GetGlobalID())
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}

		resp.LogReadURL = fmt.Sprintf("%s/v1/runs/%s/%s/logs/%s", c.tharsisAPIURL, run.GetGlobalID(), models.PlanNodePath, string(token))
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, resp, http.StatusOK)
}

func (c *runController) GetApply(w http.ResponseWriter, r *http.Request) {
	applyID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), applyID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	apply := run.Apply
	if apply == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("apply with id %s not found", applyID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	resp := &gotfe.Apply{
		ID:     apply.GetGlobalID(),
		Status: toTFEApplyStatus(apply.Status),
	}

	// Always return a LogReadURL; the Terraform CLI requires one on every apply response,
	// even before the apply's job has been created. The URL is scoped to the run node so
	// the log endpoint can resolve the job lazily once it exists.
	if c.tharsisAPIURL != "" {
		token, err := c.createRunLogToken(r.Context(), run.GetGlobalID())
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}

		resp.LogReadURL = fmt.Sprintf("%s/v1/runs/%s/%s/logs/%s", c.tharsisAPIURL, run.GetGlobalID(), models.ApplyNodePath, string(token))
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, resp, http.StatusOK)
}

// toTFEApplyStatus converts an internal apply status to its TFE equivalent. The
// Terraform CLI has no skipped status, so a skipped apply (one that never started
// before the run ended) is reported as created, the status it held before being
// skipped; all other statuses map one-to-one.
func toTFEApplyStatus(status models.ApplyStatus) gotfe.ApplyStatus {
	if status == models.ApplySkipped {
		return gotfe.ApplyStatus(models.ApplyCreated)
	}
	return gotfe.ApplyStatus(status)
}

// createRunLogToken mints a short-lived token authorizing reads of a run's node logs.
// The subject is the run ID so the log endpoint can verify the token before resolving the
// run, without needing the node's job, which may not exist yet when the token is issued.
func (c *runController) createRunLogToken(ctx context.Context, runID string) ([]byte, error) {
	return c.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Subject:    runID,
		Expiration: ptr.Time(time.Now().Add(5 * time.Minute)),
	})
}
