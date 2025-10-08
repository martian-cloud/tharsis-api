//go:build noui

// Package ui provides handlers for serving the web UI
package ui

import (
	"net/http"
)

// NewHandler returns an error when UI is disabled
func NewHandler() (http.Handler, error) {
	// Return no error since the UI isn't enabled.
	return nil, nil
}
