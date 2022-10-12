package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gotfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
	"github.com/lestrrat-go/jwx/jwt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jwsprovider"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
)

//const defaultLogReadLimit = 1024 * 1024 // 1 MiB

type runController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	jwsProvider       jwsprovider.JWSProvider
	logger            logger.Logger
	runService        run.Service
	tharsisAPIURL     string
}

// NewRunController creates an instance of runController
func NewRunController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	jwsProvider jwsprovider.JWSProvider,
	runService run.Service,
	tharsisAPIURL string,
) Controller {
	return &runController{
		respWriter,
		jwtAuthMiddleware,
		jwsProvider,
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
	router.Put("/plans/{id}/content", c.UploadPlan)

	router.Post("/runs", c.CreateRun)
	router.Post("/runs/{id}/actions/apply", c.ApplyRun)
	router.Post("/runs/{id}/actions/cancel", c.CancelRun)
}

func (c *runController) DownloadPlan(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	result, err := c.runService.DownloadPlan(r.Context(), planID)
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

func (c *runController) CreateRun(w http.ResponseWriter, r *http.Request) {
	var req gotfe.RunCreateOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	options := &run.CreateRunInput{
		WorkspaceID: gid.FromGlobalID(req.Workspace.ID),
		Comment:     req.Message,
	}
	if req.ConfigurationVersion != nil {
		id := gid.FromGlobalID(req.ConfigurationVersion.ID)
		options.ConfigurationVersionID = &id
	}
	if req.IsDestroy != nil {
		options.IsDestroy = *req.IsDestroy
	}

	run, err := c.runService.CreateRun(r.Context(), options)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisRunToRun(run), http.StatusCreated)
}

func (c *runController) ApplyRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	var req gotfe.RunApplyOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	run, err := c.runService.ApplyRun(r.Context(), runID, req.Comment)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisRunToRun(run), http.StatusOK)
}

func (c *runController) CancelRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	var req gotfe.RunCancelOptions
	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	run, err := c.runService.CancelRun(r.Context(), &run.CancelRunInput{
		RunID:   runID,
		Comment: req.Comment,
		// The REST API does not support the force option to cancel a run.
		// Only the GraphQL interface supports the force option.
	})
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisRunToRun(run), http.StatusOK)
}

func (c *runController) GetRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRun(r.Context(), runID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(w, TharsisRunToRun(run), http.StatusOK)
}

func (c *runController) GetPlan(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	plan, err := c.runService.GetPlan(r.Context(), planID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	job, err := c.runService.GetLatestJobForPlan(r.Context(), planID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	resp := &gotfe.Plan{
		ID:                   gid.ToGlobalID(gid.PlanType, plan.Metadata.ID),
		Status:               gotfe.PlanStatus(plan.Status),
		HasChanges:           plan.HasChanges,
		ResourceAdditions:    plan.ResourceAdditions,
		ResourceChanges:      plan.ResourceChanges,
		ResourceDestructions: plan.ResourceDestructions,
	}

	if job != nil && c.tharsisAPIURL != "" {
		token, err := c.createJobLogToken(r.Context(), job)
		if err != nil {
			c.respWriter.RespondWithError(w, err)
			return
		}

		resp.LogReadURL = fmt.Sprintf("%s/v1/jobs/%s/logs/%s", c.tharsisAPIURL, gid.ToGlobalID(gid.JobType, job.Metadata.ID), string(token))
	}

	c.respWriter.RespondWithJSONAPI(w, resp, http.StatusOK)
}

func (c *runController) GetApply(w http.ResponseWriter, r *http.Request) {
	applyID := gid.FromGlobalID(chi.URLParam(r, "id"))

	apply, err := c.runService.GetApply(r.Context(), applyID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	job, err := c.runService.GetLatestJobForApply(r.Context(), applyID)
	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	resp := &gotfe.Apply{
		ID:     gid.ToGlobalID(gid.ApplyType, apply.Metadata.ID),
		Status: gotfe.ApplyStatus(apply.Status),
	}

	if job != nil && c.tharsisAPIURL != "" {
		token, err := c.createJobLogToken(r.Context(), job)
		if err != nil {
			c.respWriter.RespondWithError(w, err)
			return
		}

		resp.LogReadURL = fmt.Sprintf("%s/v1/jobs/%s/logs/%s", c.tharsisAPIURL, gid.ToGlobalID(gid.JobType, job.Metadata.ID), string(token))
	}

	c.respWriter.RespondWithJSONAPI(w, resp, http.StatusOK)
}

func (c *runController) createJobLogToken(ctx context.Context, job *models.Job) ([]byte, error) {
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
	if err := token.Set(jwt.SubjectKey, job.Metadata.ID); err != nil {
		return nil, err
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		return nil, err
	}

	return c.jwsProvider.Sign(ctx, payload)
}
