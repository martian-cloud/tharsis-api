package middleware

import (
	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"net/http"
)

// NewRequestIDMiddleware adds a request ID to the logger context
func NewRequestIDMiddleware() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := logger.WithRequestID(r.Context(), uuid.NewString())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
