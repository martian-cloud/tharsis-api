package client

import (
	"fmt"
	"net/http"
	"runtime"
)

// BuildUserAgent creates a user agent string in the format "name/version (os; arch)".
func BuildUserAgent(name, version string) string {
	return fmt.Sprintf("%s/%s (%s; %s)", name, version, runtime.GOOS, runtime.GOARCH)
}

// UserAgentTransport wraps an http.RoundTripper to add the User-Agent header.
type UserAgentTransport struct {
	UserAgent string
	Base      http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t *UserAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.UserAgent)
	return t.Base.RoundTrip(req)
}
