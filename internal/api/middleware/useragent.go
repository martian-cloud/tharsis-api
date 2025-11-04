package middleware

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"net/http"
)

const (
	userAgentHeader = "User-Agent"
)

// NewUserAgentMiddleware adds user agent to the logger context if header is present
func NewUserAgentMiddleware() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get(userAgentHeader)
			if userAgent != "" {
				ctx := logger.WithUserAgent(r.Context(), userAgent)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
