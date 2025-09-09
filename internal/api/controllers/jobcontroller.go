package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const defaultLogReadLimit = 1024 * 1024 // 1 MiB

type jobController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	idp               auth.IdentityProvider
	logger            logger.Logger
	jobService        job.Service
}

// NewJobController creates an instance of jobController
func NewJobController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	idp auth.IdentityProvider,
	jobService job.Service,
) Controller {
	return &jobController{respWriter, jwtAuthMiddleware, idp, logger, jobService}
}

// RegisterRoutes adds health routes to the router
func (c *jobController) RegisterRoutes(router chi.Router) {
	// TODO: Non header based authentication needs to be used for logs
	router.Get("/jobs/{jobId}/logs/{token}", c.GetJobLogs)
}

func (c *jobController) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	jobID := gid.FromGlobalID(chi.URLParam(r, "jobId"))

	token := chi.URLParam(r, "token")
	if token == "" {
		c.respWriter.RespondWithError(r.Context(), w, errors.New("Missing token query parameter in log URL", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	// Validate token
	if err := c.verifyJobLogToken(r.Context(), token, jobID); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "invalid token", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = defaultLogReadLimit
	}
	// offset defaults to 0 if not provided
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	// TODO: Remove when log endpoint token authentication is added
	logs, err := c.jobService.ReadLogs(auth.WithCaller(r.Context(), &auth.SystemCaller{}), jobID, offset, limit)
	if err != nil {
		c.logger.WithContextFields(r.Context()).Infof("Failed to get logs: %v", err)
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	if _, err := w.Write(logs); err != nil {
		c.logger.WithContextFields(r.Context()).Infof("Failed to respond with log data: %v", err)
		c.respWriter.RespondWithError(r.Context(), w, err)
	}
}

func (c *jobController) verifyJobLogToken(ctx context.Context, token string, jobID string) error {
	if _, err := c.idp.VerifyToken(ctx, token, jwt.WithSubject(jobID)); err != nil {
		return fmt.Errorf("job log token is invalid: %w", err)
	}

	return nil
}
