package middleware

import (
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// NewRequireAuthenticatedCallerMiddleware requires an authenticated caller.
func NewRequireAuthenticatedCallerMiddleware(
	logger logger.Logger,
	respWriter response.Writer,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			caller := auth.GetCaller(r.Context())
			if caller == nil {
				subject := auth.GetSubject(r.Context())
				if subject != nil {
					logger.Infof("Unauthorized request to %s %s by %s", r.Method, r.URL.Path, *subject)
				} else {
					logger.Infof("Unauthorized request to %s %s by unknown subject", r.Method, r.URL.Path)
				}

				respWriter.RespondWithError(w,
					// At this point, we no longer had the original error to wrap.
					errors.New(errors.EUnauthorized, "Unauthorized"))
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithCaller(r.Context(), caller)))
		})
	}
}
