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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
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
	jobService        job.Service
	tharsisAPIURL     string
}

// NewRunController creates an instance of runController
func NewRunController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	signingKeyManager auth.SigningKeyManager,
	runService run.Service,
	jobService job.Service,
	tharsisAPIURL string,
) controllers.Controller {
	return &runController{
		respWriter,
		jwtAuthMiddleware,
		signingKeyManager,
		logger,
		runService,
		jobService,
		tharsisAPIURL,
	}
}

// RegisterRoutes adds routes to the router
func (c *runController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/runs/{id}", c.GetRun)
	router.Get("/runs/{id}/policy-checks", c.ListRunPolicyChecks)
	router.Get("/runs/{id}/task-stages", c.ListRunTaskStages)
	router.Get("/runs/{id}/run-events", c.ListRunEvents)
	router.Get("/plans/{id}", c.GetPlan)
	router.Get("/applies/{id}", c.GetApply)
	router.Get("/policy-checks/{id}", c.GetPolicyCheck)
	router.Get("/policy-checks/{id}/output", c.GetPolicyCheckLogs)
	router.Get("/task-stages/{id}", c.GetTaskStage)
	router.Get("/task-stages/{id}/policy-evaluations", c.ListTaskStagePolicyEvaluations)
	router.Get("/policy-evaluations/{id}/policy-set-outcomes", c.ListPolicySetOutcomes)

	router.Get("/plans/{id}/content", c.DownloadPlan)
	router.Get("/plans/{id}/json-output", c.DownloadPlanJSON)

	router.Post("/runs", c.CreateRun)
	router.Post("/runs/{id}/actions/apply", c.ApplyRun)
	router.Post("/runs/{id}/actions/cancel", c.CancelRun)
	router.Post("/runs/{id}/actions/discard", c.DiscardRun)
	router.Post("/policy-checks/{id}/actions/override", c.OverridePolicyCheck)
	router.Post("/task-stages/{id}/actions/override", c.OverrideTaskStage)
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

func (c *runController) DownloadPlanJSON(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	result, err := c.runService.DownloadPlanJSON(r.Context(), planID)
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

	if req.AutoApply != nil {
		options.AutoApply = *req.AutoApply
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

// DiscardRun discards a planned run (e.g. when a user declines a soft-failed policy override). The
// go-tfe "cloud" backend calls this; the run service enforces that the run is in a discardable state.
func (c *runController) DiscardRun(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	discarded, err := c.runService.DiscardRun(r.Context(), &run.DiscardRunInput{RunID: runID})
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisRunToRun(discarded), http.StatusOK)
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
		Status:               toTFEPlanStatus(plan.Status),
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

func (c *runController) GetPolicyCheck(w http.ResponseWriter, r *http.Request) {
	checkID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), checkID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	check := run.PolicyCheckByID(checkID)
	if check == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("policy check with id %s not found", checkID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisPolicyCheckToPolicyCheck(check), http.StatusOK)
}

// OverrideTaskStage overrides every soft-failed (overridable) policy check in a task stage and
// returns the updated stage. The go-tfe "cloud" backend calls this when a user confirms "override"
// at the policy prompt — it overrides at the task-stage level rather than per policy check (the
// legacy OverridePolicyCheck path). Overriding each soft-failed check clears the stage's gate so the
// run can proceed.
func (c *runController) OverrideTaskStage(w http.ResponseWriter, r *http.Request) {
	stageID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), stageID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	stage := run.TaskStageByID(stageID)
	if stage == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("task stage with id %s not found", stageID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	// A soft-failed check is blocked by exactly one gate, and overriding that gate is what clears the
	// check — so collect the checks to clear and resolve their gates in one read.
	var checkIDs []string
	for _, check := range stage.PolicyChecks {
		if check.Status == models.PolicyCheckSoftFailed {
			checkIDs = append(checkIDs, check.ID)
		}
	}

	// Nothing to clear: return the stage as it stands. Guarded rather than left to the query below,
	// because an empty ID list is not a filter that matches nothing — it is no filter at all, and
	// would fetch every gate in the deployment.
	if len(checkIDs) > 0 {
		gates, gErr := c.runService.GetRunGatesByPolicyCheckIDs(r.Context(), checkIDs)
		if gErr != nil {
			c.respWriter.RespondWithError(r.Context(), w, gErr)
			return
		}

		for _, gate := range gates {
			if _, err = c.runService.OverrideRunGate(r.Context(), gate.Metadata.ID, nil); err != nil {
				c.respWriter.RespondWithError(r.Context(), w, err)
				return
			}
		}

		// Re-read rather than threading the run through the loop: each override returns its gate, and
		// the stage this responds with has to reflect all of them.
		run, err = c.runService.GetRunByID(r.Context(), run.Metadata.ID)
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, err)
			return
		}

		stage = run.TaskStageByID(stageID)
		if stage == nil {
			c.respWriter.RespondWithError(r.Context(), w,
				errors.New("task stage with id %s not found", stageID, errors.WithErrorCode(errors.ENotFound)))
			return
		}
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, tharsisTaskStageToTaskStage(stage), http.StatusOK)
}

// GetPolicyCheckLogs streams the full logs of a policy check's latest job. The go-tfe policy
// check Logs API fetches this output in a single request once the check has finished running
// (it does not paginate), so this returns the complete log rather than a single bounded chunk
// like the plan/apply log endpoint. The request is authenticated by the JWT middleware, so the
// caller's own ViewJob permission gates the read via the job service — no signed URL/token is
// needed here.
func (c *runController) GetPolicyCheckLogs(w http.ResponseWriter, r *http.Request) {
	checkID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), checkID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	check := run.PolicyCheckByID(checkID)
	if check == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("policy check with id %s not found", checkID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	if check.LatestJobID == nil {
		// The policy-eval job has not been created yet, so there are no logs to return.
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		return
	}

	c.streamJobLogs(w, r, *check.LatestJobID)
}

// policyCheckLogReadChunkSize bounds each ReadLogs call. The log stream manager rejects a read
// range larger than 1 MiB, so a job's full logs are streamed one chunk at a time.
const policyCheckLogReadChunkSize = 1024 * 1024 // 1 MiB

// streamJobLogs streams a job's entire logs to the response, reading in <=1 MiB chunks until the
// stream is exhausted. Reads run as the authenticated caller, whose ViewJob permission the job
// service enforces. Only a failure before any bytes are written can be reported as an error
// response; once streaming starts the 200 status is committed, so later failures are just logged.
func (c *runController) streamJobLogs(w http.ResponseWriter, r *http.Request, jobID string) {
	offset := 0
	wrote := false
	for {
		reader, err := c.jobService.ReadLogs(r.Context(), jobID, offset, policyCheckLogReadChunkSize)
		if err != nil {
			if !wrote {
				c.respWriter.RespondWithError(r.Context(), w, err)
				return
			}
			c.logger.WithContextFields(r.Context()).Infof("Failed to read policy check logs: %v", err)
			return
		}

		if !wrote {
			w.Header().Set("Content-Type", "text/plain")
			wrote = true
		}

		// Stream the chunk straight to the response so the full range is never buffered in memory.
		n, err := io.Copy(w, reader)
		reader.Close()
		if err != nil {
			c.logger.WithContextFields(r.Context()).Infof("Failed to write policy check logs: %v", err)
			return
		}

		if n < policyCheckLogReadChunkSize {
			// Short (or empty) final chunk: the whole stream has been written.
			return
		}
		offset += int(n)
	}
}

func (c *runController) ListRunPolicyChecks(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByID(r.Context(), runID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	allChecks := run.AllPolicyChecks()
	checks := make([]*gotfe.PolicyCheck, len(allChecks))
	for i, check := range allChecks {
		checks[i] = TharsisPolicyCheckToPolicyCheck(check)
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, checks, http.StatusOK)
}

// ListRunTaskStages returns the run's task stages (one per policy stage), each carrying an OPA
// policy evaluation per OPA check it owns and a task result per non-OPA check (module attestation
// today) it owns. This is the newer go-tfe surface for policy results; the legacy policy-checks
// endpoints remain for override and log streaming.
func (c *runController) ListRunTaskStages(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByID(r.Context(), runID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	stages := run.TaskStages
	taskStages := make([]*TaskStage, len(stages))
	for i, stage := range stages {
		taskStages[i] = tharsisTaskStageToTaskStage(stage)
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, taskStages, http.StatusOK)
}

// ListRunEvents returns the run's event timeline. The go-tfe "cloud" backend lists run events to
// render a run's activity; the older "remote" backend never calls this endpoint (which is why a run
// works under "remote" but 404s under "cloud"). Tharsis does not model TFE-style run events, so this
// returns an empty list — enough to satisfy the cloud backend, which treats the timeline as
// display-only and tracks run progress via the run status it polls separately. The run is still
// looked up so a genuinely missing run returns not-found rather than an empty list.
func (c *runController) ListRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := gid.FromGlobalID(chi.URLParam(r, "id"))

	if _, err := c.runService.GetRunByID(r.Context(), runID); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, []*gotfe.RunEvent{}, http.StatusOK)
}

// GetTaskStage returns a single task stage identified by its (task stage node) id.
func (c *runController) GetTaskStage(w http.ResponseWriter, r *http.Request) {
	stageID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), stageID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	stage := run.TaskStageByID(stageID)
	if stage == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("task stage with id %s not found", stageID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, tharsisTaskStageToTaskStage(stage), http.StatusOK)
}

// ListTaskStagePolicyEvaluations returns a task stage's OPA policy evaluations, identified by the
// task stage node's id. A non-OPA check (module attestation today) is not among them -- it is
// surfaced through the stage's task-results relation instead (see tharsisTaskStageToTaskStage).
func (c *runController) ListTaskStagePolicyEvaluations(w http.ResponseWriter, r *http.Request) {
	stageID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), stageID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	stage := run.TaskStageByID(stageID)
	if stage == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("task stage with id %s not found", stageID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, tharsisTaskStageToTaskStage(stage).PolicyEvaluations, http.StatusOK)
}

// ListPolicySetOutcomes returns the policy-set outcomes for a policy evaluation, identified by the
// policy check node's id (the ID surfaced as the go-tfe policy-evaluation ID). The "cloud" backend
// reads this to render the individual OPA policy results for a soft-failed (overridable) evaluation.
func (c *runController) ListPolicySetOutcomes(w http.ResponseWriter, r *http.Request) {
	checkID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), checkID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	check := run.PolicyCheckByID(checkID)
	if check == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("policy check with id %s not found", checkID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	// One read per policy: this endpoint is only fetched for an evaluation a caller is inspecting, and
	// it reports the failures in full rather than the check's capped summary.
	messages := make(map[string][]string, len(check.Policies))
	for _, p := range check.Policies {
		policyMessages, mErr := c.runService.GetPolicyCheckPolicyMessages(r.Context(), checkID, p.ID)
		if mErr != nil {
			c.respWriter.RespondWithError(r.Context(), w, mErr)
			return
		}
		messages[p.ID] = policyMessages
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, tharsisPolicyCheckToPolicySetOutcomes(check, messages), http.StatusOK)
}

func (c *runController) OverridePolicyCheck(w http.ResponseWriter, r *http.Request) {
	checkID := gid.FromGlobalID(chi.URLParam(r, "id"))

	run, err := c.runService.GetRunByNodeID(r.Context(), checkID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	check := run.PolicyCheckByID(checkID)
	if check == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("policy check with id %s not found", checkID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	// The check is cleared by overriding the gate blocking it; a soft-failed check always has one.
	gates, err := c.runService.GetRunGatesByPolicyCheckIDs(r.Context(), []string{checkID})
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}
	if len(gates) == 0 {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("policy check with id %s is not awaiting an override", checkID, errors.WithErrorCode(errors.EConflict)))
		return
	}

	if _, err = c.runService.OverrideRunGate(r.Context(), gates[0].Metadata.ID, nil); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	updatedRun, err := c.runService.GetRunByID(r.Context(), run.Metadata.ID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	check = updatedRun.PolicyCheckByID(checkID)
	if check == nil {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("policy check with id %s not found", checkID, errors.WithErrorCode(errors.ENotFound)))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisPolicyCheckToPolicyCheck(check), http.StatusOK)
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

// toTFEPlanStatus converts an internal plan status to its TFE equivalent. A skipped plan (one that
// never ran because the run terminated at a gate) is reported as go-tfe's "unreachable" — its
// terminal "never reached" plan status; all other statuses map one-to-one.
func toTFEPlanStatus(status models.PlanStatus) gotfe.PlanStatus {
	if status == models.PlanSkipped {
		return gotfe.PlanUnreachable
	}
	return gotfe.PlanStatus(status)
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
