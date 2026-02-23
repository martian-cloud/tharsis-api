// Package controllers package
package controllers

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// ObjectStoreController handles object store download requests
type ObjectStoreController struct {
	logger      logger.Logger
	respWriter  response.Writer
	objectStore objectstore.ObjectStore
}

// NewObjectStoreController creates a new ObjectStoreController
func NewObjectStoreController(
	logger logger.Logger,
	respWriter response.Writer,
	objectStore objectstore.ObjectStore,
) *ObjectStoreController {
	return &ObjectStoreController{
		logger:      logger,
		respWriter:  respWriter,
		objectStore: objectStore,
	}
}

// RegisterRoutes adds object store routes to the router
func (c *ObjectStoreController) RegisterRoutes(router chi.Router) {
	router.MethodFunc("GET", "/objectstore/*", c.DownloadObject)
}

// DownloadObject handles presigned URL downloads for filesystem object store
func (c *ObjectStoreController) DownloadObject(w http.ResponseWriter, r *http.Request) {
	key, err := c.objectStore.VerifyPresignedURL(r.Context(), r.URL.RequestURI())
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	stream, err := c.objectStore.GetObjectStream(r.Context(), key, nil)
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			c.logger.WithContextFields(r.Context()).Errorf("failed to close object stream for key %s: %v", key, closeErr)
		}
	}()

	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(key, ".zip"):
		contentType = "application/zip"
	case strings.HasSuffix(key, ".tar.gz"), strings.HasSuffix(key, ".tgz"):
		contentType = "application/gzip"
	case strings.HasSuffix(key, ".json"):
		contentType = "application/json"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment")

	if _, err := io.Copy(w, stream); err != nil {
		c.logger.WithContextFields(r.Context()).Errorf("failed to copy object stream for key %s: %v", key, err)
	}
}
