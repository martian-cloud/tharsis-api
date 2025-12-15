package middleware

import (
	"context"
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	userAgentHeader = "User-Agent"
)

type contextKey string

var (
	contextKeyUserAgent  = contextKey("user agent")
)

func (c contextKey) String() string {
	return "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/middleware " + string(c)
}

// NewUserAgentMiddleware adds user agent to the logger context if header is present
func NewUserAgentMiddleware() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get(userAgentHeader)
			if userAgent != "" {
				// First add user agent to context
				ctx := context.WithValue(r.Context(), contextKeyUserAgent, userAgent)
				// Then add user agent to logger context
				ctx = logger.WithUserAgent(ctx, userAgent)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
