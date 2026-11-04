package tfe

import (
	"compress/gzip"
	stderrors "errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type stateController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	workspaceService  workspace.Service
	tharsisAPIURL     string
	tfeVersionedPath  string
	// stateVersionMaxUploadSize caps a state rendering upload, both as sent and once decompressed.
	stateVersionMaxUploadSize int
}

// NewStateController creates an instance of stateController
func NewStateController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	workspaceService workspace.Service,
	tharsisAPIURL string,
	tfeVersionedPath string,
	stateVersionMaxUploadSize int,
) controllers.Controller {
	return &stateController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		workspaceService,
		tharsisAPIURL,
		tfeVersionedPath,
		stateVersionMaxUploadSize,
	}
}

// RegisterRoutes adds routes to the router.
func (c *stateController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Get("/state-versions/{stateVersionId}", c.GetStateVersion)
	router.Get("/state-versions/{stateVersionId}/content", c.DownloadStateVersion)
	router.Put("/state-versions/{stateVersionId}/content.json", c.UploadStateVersionJSON)
	router.Get("/state-versions/{stateVersionId}/content.json", c.DownloadStateVersionJSON)
}

func (c *stateController) GetStateVersion(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	sv, err := c.workspaceService.GetStateVersionByID(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, TharsisStateVersionToStateVersion(sv, c.tharsisAPIURL, c.tfeVersionedPath), http.StatusOK)
}

func (c *stateController) DownloadStateVersion(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	result, err := c.workspaceService.GetStateVersionContent(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	defer result.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		c.logger.WithContextFields(r.Context()).Errorf(
			"failed to stream state version %s: %v", stateVersionID, err)
		// Use the http abort handler to terminate the connection gracefully
		panic(http.ErrAbortHandler)
	}
}

// UploadStateVersionJSON stores the "terraform show -json" rendering of a state version, the endpoint
// TFE advertises as hosted-json-state-upload-url. The job executor calls it after creating the state
// version over gRPC: that call carries the raw state base64-encoded inside a single size-limited
// message, and the rendering is typically larger than the state it describes, so it streams here
// instead.
//
// Re-uploading replaces the stored rendering, which is what makes the executor's retry safe.
func (c *stateController) UploadStateVersionJSON(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	limit := int64(c.stateVersionMaxUploadSize)
	body := http.MaxBytesReader(w, r.Body, limit)
	defer body.Close()

	// The service reads the body itself, so read errors are recorded here: the error the service returns
	// may not carry them.
	reader := &readErrRecorder{reader: body}

	// Accept an optionally gzipped body, as the plan JSON upload does: a state rendering is highly
	// compressible and can be large. The decompressed rendering is capped too, or a small body could
	// expand without bound.
	gzipped := strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip")
	if gzipped {
		gzipReader, err := gzip.NewReader(body)
		if err != nil {
			c.respWriter.RespondWithError(r.Context(), w, requestBodyError(err))
			return
		}
		defer gzipReader.Close()
		reader.reader = http.MaxBytesReader(w, gzipReader, limit)
	}

	if err := c.workspaceService.UploadStateVersionJSON(r.Context(), stateVersionID, reader); err != nil {
		var maxBytesErr *http.MaxBytesError
		switch {
		case stderrors.As(err, &maxBytesErr), stderrors.As(reader.err, &maxBytesErr):
			err = requestBodyError(maxBytesErr)
		case reader.err != nil:
			// A body that fails mid-read (a corrupt gzip stream past a valid header, or a truncated or
			// reset connection) only fails once the service reads it.
			err = requestBodyError(reader.err)
		}
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}

// readErrRecorder records the first error other than io.EOF that reading from reader returns.
type readErrRecorder struct {
	reader io.Reader
	err    error
}

func (r *readErrRecorder) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err != nil && !stderrors.Is(err, io.EOF) && r.err == nil {
		r.err = err
	}
	return n, err
}

// DownloadStateVersionJSON streams the "terraform show -json" rendering of a state version, which TFE
// advertises as hosted-json-state-download-url.
//
// A state version predating the rendering, or one whose executor was too old to send it, has none — the
// service returns a not-found error, matching the empty hosted-json-state-download-url an HCP client
// sees in the same situation.
func (c *stateController) DownloadStateVersionJSON(w http.ResponseWriter, r *http.Request) {
	stateVersionID := gid.FromGlobalID(chi.URLParam(r, "stateVersionId"))

	result, err := c.workspaceService.GetStateVersionJSONContent(r.Context(), stateVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	defer result.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, result); err != nil {
		c.logger.WithContextFields(r.Context()).Errorf(
			"failed to stream state version JSON %s: %v", stateVersionID, err)
		// Use the http abort handler to terminate the connection gracefully
		panic(http.ErrAbortHandler)
	}
}
