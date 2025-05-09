// Package http package
package http

import (
	"net/http"
	"time"
)

const (
	httpClientTimeout = 5 * time.Minute
)

// NewHTTPClient creates a new HTTP client with a timeout.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpClientTimeout}
}

// The End.
