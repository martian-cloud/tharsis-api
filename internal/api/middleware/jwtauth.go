package middleware

import (
	"fmt"
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

// NewJwtAuthMiddleware creates an instance of JwtAuthMiddleware
func NewJwtAuthMiddleware(
	authenticator *auth.Authenticator,
	logger logger.Logger,
	respWriter response.Writer,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller, err := authenticator.Authenticate(r.Context(), auth.FindToken(r), true)
			if err != nil {
				logger.Infof("Unauthorized request to %s %s: %v", r.Method, r.URL.Path, err)
				respWriter.RespondWithError(w,
					te.NewError(te.EUnauthorized, fmt.Sprintf("Unauthorized: %v", err)))
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithCaller(r.Context(), caller)))
		})
	}
}
