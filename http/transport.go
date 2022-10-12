package http

import (
	"net/http"
	"time"
)

const (
	httpClientTimeout = 30 * time.Second
)

// NewHTTPClient creates a new HTTP client with a timeout.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpClientTimeout}
}

// The End.
