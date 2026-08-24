package controllers

import (
	stderrors "errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/packageregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// packageRegistryMaxUploadSize is the maximum allowed size for a package version package upload.
const packageRegistryMaxUploadSize = 1024 * 1024 * 32 // 32 MiB

type packageRegistryController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	packageService    packageregistry.Service
}

// NewPackageRegistryController creates an instance of packageRegistryController
func NewPackageRegistryController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	packageService packageregistry.Service,
) Controller {
	return &packageRegistryController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		packageService,
	}
}

// RegisterRoutes adds the package registry routes to the router
func (c *packageRegistryController) RegisterRoutes(router chi.Router) {
	// Require JWT authentication
	router.Use(c.jwtAuthMiddleware)

	router.Put("/package-registry/versions/{packageVersionId}/upload", c.UploadPackageVersion)
	router.Get("/package-registry/versions/{packageVersionId}/download", c.DownloadPackageVersion)
}

func (c *packageRegistryController) UploadPackageVersion(w http.ResponseWriter, r *http.Request) {
	packageVersionID := gid.FromGlobalID(chi.URLParam(r, "packageVersionId"))

	packageVersion, err := c.packageService.GetPackageVersionByID(r.Context(), packageVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	// Limit size of request body
	limitReader := http.MaxBytesReader(w, r.Body, int64(packageRegistryMaxUploadSize))
	defer limitReader.Close()

	if err := c.packageService.UploadPackageVersion(r.Context(), packageVersion, limitReader); err != nil {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			c.respWriter.RespondWithError(r.Context(), w, terrors.New("upload failed, package size exceeds maximum size of %d bytes", packageRegistryMaxUploadSize, terrors.WithErrorCode(errors.ETooLarge)))
		} else {
			c.respWriter.RespondWithError(r.Context(), w, err)
		}
		return
	}

	c.respWriter.RespondWithJSONAPI(r.Context(), w, nil, http.StatusOK)
}

func (c *packageRegistryController) DownloadPackageVersion(w http.ResponseWriter, r *http.Request) {
	packageVersionID := gid.FromGlobalID(chi.URLParam(r, "packageVersionId"))

	packageVersion, err := c.packageService.GetPackageVersionByID(r.Context(), packageVersionID)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	downloadURL, err := c.packageService.DownloadPackageVersion(r.Context(), packageVersion)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	// Return the presigned URL in a response header rather than issuing a 302 redirect.
	// A browser fetch that follows the redirect carries the request's Authorization header
	// through to the object store, which turns the object-store request into a non-simple
	// cross-origin request (triggering a CORS preflight the object store rejects). Instead the
	// client reads the URL from this header and fetches it directly (unauthenticated, a simple
	// GET), mirroring the module registry download flow.
	w.Header().Set("Access-Control-Expose-Headers", "X-Download-Url")
	w.Header().Set("X-Download-Url", downloadURL)
	w.WriteHeader(http.StatusNoContent)
}
