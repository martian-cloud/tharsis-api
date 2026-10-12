//go:build !noui

// Package ui provides handlers for serving the web UI
package ui

import (
	"io/fs"
	"net/http"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/frontend"
)

// securityHeaders are set on every UI response. They must be real headers because browsers ignore
// X-Content-Type-Options and the frame-ancestors CSP directive when set via <meta> tags.
var securityHeaders = map[string]string{
	// Combined with the CSP <meta> tag in index.html; this header only restricts framing.
	"Content-Security-Policy": "frame-ancestors 'none'",
	// Legacy equivalent of frame-ancestors for older browsers.
	"X-Frame-Options":        "DENY",
	"X-Content-Type-Options": "nosniff",
}

// Handler serves the web UI with SPA fallback support
type Handler struct {
	uiFS fs.FS
}

// NewHandler returns an HTTP handler that serves static files with SPA fallback
func NewHandler() (http.Handler, error) {
	uiFS, err := frontend.DistFS()
	if err != nil {
		return nil, err
	}

	return &Handler{
		uiFS: uiFS,
	}, nil
}

// ServeHTTP implements http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for k, v := range securityHeaders {
		w.Header().Set(k, v)
	}

	fileServer := http.FileServer(http.FS(h.uiFS))

	// Check if file exists
	if _, err := fs.Stat(h.uiFS, strings.TrimPrefix(r.URL.Path, "/")); err == nil {
		fileServer.ServeHTTP(w, r)
		return
	}

	// File doesn't exist, serve index.html for SPA routing
	r.URL.Path = "/"
	fileServer.ServeHTTP(w, r)
}
