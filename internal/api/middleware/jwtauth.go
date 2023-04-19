package middleware

import (
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
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
					te.Wrap(err, te.EUnauthorized, "Unauthorized"))
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithCaller(r.Context(), caller)))
		})
	}
}
