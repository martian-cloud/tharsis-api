package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const defaultLogReadLimit = 1024 * 1024 // 1 MiB

type jobController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	signingKeyManager auth.SigningKeyManager
	logger            logger.Logger
	jobService        job.Service
	runService        run.Service
}

// NewJobController creates an instance of jobController
func NewJobController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	signingKeyManager auth.SigningKeyManager,
	jobService job.Service,
	runService run.Service,
) Controller {
	return &jobController{respWriter, jwtAuthMiddleware, signingKeyManager, logger, jobService, runService}
}

// RegisterRoutes adds health routes to the router
func (c *jobController) RegisterRoutes(router chi.Router) {
	// Run-node log endpoint. The Terraform CLI requires a LogReadURL before a job
	// exists, so the TFE controller issues a URL scoped to the run node (plan/apply)
	// rather than the job. This endpoint resolves the node's job at request time and
	// streams its logs, returning an empty 200 while the job has not been created yet.
	router.Get("/runs/{runId}/{nodePath}/logs/{token}", c.GetRunNodeLogs)
}

// GetRunNodeLogs streams the logs for a run node's (plan or apply) latest job. The
// Terraform CLI expects a LogReadURL on every plan/apply response, including before the
// node's job has been created, so this endpoint is scoped to the run node rather than the
// job. While the node has no job yet it returns an empty 200 so the CLI keeps polling
// (the go-tfe LogReader aborts the stream on any non-2xx response).
func (c *jobController) GetRunNodeLogs(w http.ResponseWriter, r *http.Request) {
	runGID := chi.URLParam(r, "runId")
	runID := gid.FromGlobalID(runGID)
	nodePath := chi.URLParam(r, "nodePath")

	token := chi.URLParam(r, "token")
	if token == "" {
		c.respWriter.RespondWithError(r.Context(), w, errors.New("Missing token in log URL", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	if nodePath != models.PlanNodePath && nodePath != models.ApplyNodePath {
		c.respWriter.RespondWithError(r.Context(), w,
			errors.New("invalid run node path %q", nodePath, errors.WithErrorCode(errors.EInvalid)))
		return
	}

	// Validate the run-scoped token before querying for the run.
	if err := c.verifyLogToken(r.Context(), token, runGID); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "invalid token", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	// The path token is the trust anchor (mirroring GetJobLogs), so resolve the run with a
	// system caller rather than depending on header-based authentication.
	ctx := auth.WithCaller(r.Context(), &auth.SystemCaller{})

	run, err := c.runService.GetRunByID(ctx, runID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	var latestJobID *string
	switch nodePath {
	case models.PlanNodePath:
		latestJobID = run.Plan.LatestJobID
	case models.ApplyNodePath:
		if run.Apply == nil {
			// Speculative runs have no apply node; nothing to stream yet.
			w.WriteHeader(http.StatusOK)
			return
		}
		latestJobID = run.Apply.LatestJobID
	}

	if latestJobID == nil {
		// The node's job has not been created yet; return an empty 200 so the CLI polls again.
		w.WriteHeader(http.StatusOK)
		return
	}

	c.writeJobLogs(w, r, *latestJobID)
}

// writeJobLogs streams a chunk of the given job's logs honoring the request's limit and
// offset query parameters. It reads with a system caller since the calling handler has
// already authorized the request via its path token.
func (c *jobController) writeJobLogs(w http.ResponseWriter, r *http.Request, jobID string) {
	w.Header().Set("Content-Type", "text/plain")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = defaultLogReadLimit
	}
	// offset defaults to 0 if not provided
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	reader, err := c.jobService.ReadLogs(auth.WithCaller(r.Context(), &auth.SystemCaller{}), jobID, offset, limit)
	if err != nil {
		c.logger.WithContextFields(r.Context()).Infof("Failed to get logs: %v", err)
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}
	defer reader.Close()

	// Stream the logs straight to the response so the full range is never buffered in memory.
	if _, err := io.Copy(w, reader); err != nil {
		c.logger.WithContextFields(r.Context()).Infof("Failed to respond with log data: %v", err)
		c.respWriter.RespondWithError(r.Context(), w, err)
	}
}

func (c *jobController) verifyLogToken(ctx context.Context, token string, subject string) error {
	if _, err := c.signingKeyManager.VerifyToken(ctx, token, jwt.WithSubject(subject)); err != nil {
		return fmt.Errorf("log token is invalid: %w", err)
	}

	return nil
}
