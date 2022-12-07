package middleware

import (
	"fmt"
	"net/http"
	"strings"

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
			caller, err := authenticator.Authenticate(r.Context(), findToken(r), true)
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

func findToken(r *http.Request) string {
	// Get token from authorization header.
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}

	return ""
}
