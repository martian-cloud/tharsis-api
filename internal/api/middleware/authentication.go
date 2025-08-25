package middleware

import (
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// NewAuthenticationMiddleware resolves whether the caller has authenticated.
// If so, it sets the caller on the context.
// In either case, it sets the subject string on the context.
func NewAuthenticationMiddleware(
	authenticator auth.Authenticator,
	respWriter response.Writer,
	secureUserSessionCookiesEnabled bool,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			token, err := auth.FindToken(r, secureUserSessionCookiesEnabled)
			if err != nil {
				respWriter.RespondWithError(r.Context(), w, err)
				return
			}

			if token != "" {
				// Permission cache is not used for subscriptions since they are long lived
				usePermissionCache := !isGraphqlSubscriptionRequest(r)
				// Attempt to authenticate caller using token
				caller, err := authenticator.Authenticate(ctx, token, usePermissionCache)
				if err != nil {
					if errors.ErrorCode(err) != errors.EUnauthorized {
						respWriter.RespondWithError(r.Context(), w, err)
						return
					}

					// If this is an authentication error, set it on the context so that it can be handled later
					ctx = auth.WithCallerAuthenticationError(ctx, err)
				} else {
					// This request is authenticated.
					ctx = auth.WithCaller(ctx, caller)
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
