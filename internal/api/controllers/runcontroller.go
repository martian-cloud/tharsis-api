package controllers

import (
	"compress/gzip"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	tfjson "github.com/hashicorp/terraform-json"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type planWithProviderSchemas struct {
	Plan            *tfjson.Plan            `json:"plan"`
	ProviderSchemas *tfjson.ProviderSchemas `json:"provider_schemas"`
}

type runController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	runService        run.Service
	maxUploadSize     int
}

// NewRunController creates an instance of runController
func NewRunController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	runService run.Service,
	maxUploadSize int,
) Controller {
	return &runController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		runService,
		maxUploadSize,
	}
}

// RegisterRoutes adds routes to the router
func (c *runController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Put("/plans/{id}/content", c.UploadPlanBinary)
	router.Put("/plans/{id}/content.json", c.UploadPlanData)
}

func (c *runController) UploadPlanBinary(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	// Limit size of request body
	body := http.MaxBytesReader(w, r.Body, int64(c.maxUploadSize))
	defer body.Close()

	err := c.runService.UploadPlanBinary(r.Context(), planID, body)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, c.checkUploadSize(err))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}

func (c *runController) UploadPlanData(w http.ResponseWriter, r *http.Request) {
	planID := gid.FromGlobalID(chi.URLParam(r, "id"))

	// Limit size of request body (the compressed size when gzip encoded)
	body := http.MaxBytesReader(w, r.Body, int64(c.maxUploadSize))
	defer body.Close()

	// Check that the server actually sent compressed data
	var reader io.ReadCloser
	var err error
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		gzipReader, gzErr := gzip.NewReader(body)
		if gzErr != nil {
			c.respWriter.RespondWithError(r.Context(), w, c.checkUploadSize(errors.Wrap(gzErr, "invalid gzip request body", errors.WithErrorCode(errors.EInvalid))))
			return
		}
		defer gzipReader.Close()

		// Also limit the decompressed size to guard against gzip bombs
		reader = http.MaxBytesReader(w, gzipReader, int64(c.maxUploadSize))
	default:
		reader = body
	}

	var planData planWithProviderSchemas
	if err = json.NewDecoder(reader).Decode(&planData); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, c.checkUploadSize(errors.Wrap(err, "failed to decode plan data", errors.WithErrorCode(errors.EInvalid))))
		return
	}

	err = c.runService.ProcessPlanData(r.Context(), planID, planData.Plan, planData.ProviderSchemas)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, fmt.Errorf("failed to process plan data: %w", err))
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}

// checkUploadSize returns an ETooLarge error if err was caused by the request body exceeding the max upload size.
func (c *runController) checkUploadSize(err error) error {
	var maxBytesErr *http.MaxBytesError
	if stderrors.As(err, &maxBytesErr) {
		return errors.New("upload failed, plan size exceeds maximum size of %d bytes", c.maxUploadSize, errors.WithErrorCode(errors.ETooLarge))
	}
	return err
}
