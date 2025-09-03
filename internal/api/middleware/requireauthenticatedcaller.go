package middleware

import (
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// NewRequireAuthenticatedCallerMiddleware requires an authenticated caller.
func NewRequireAuthenticatedCallerMiddleware(
	logger logger.Logger,
	respWriter response.Writer,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			caller, err := auth.AuthorizeCaller(ctx)
			if err != nil {
				subject := auth.GetSubject(ctx)
				if subject != nil {
					logger.WithContextFields(ctx).Infof("Unauthorized request to %s %s by %s: %v", r.Method, r.URL.Path, *subject, err)
				} else {
					logger.WithContextFields(ctx).Infof("Unauthorized request to %s %s by unknown subject: %v", r.Method, r.URL.Path, err)
				}

				respWriter.RespondWithError(r.Context(), w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithCaller(ctx, caller)))
		})
	}
}
