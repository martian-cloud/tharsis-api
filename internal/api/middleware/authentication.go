package middleware

import (
	"net/http"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// NewAuthenticationMiddleware resolves whether the caller has authenticated.
// If so, it sets the caller on the context.
// In either case, it sets the subject string on the context.
func NewAuthenticationMiddleware(
	logger logger.Logger,
	authenticator auth.Authenticator,
	respWriter response.Writer,
	userSessionManager auth.UserSessionManager,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			token, err := auth.FindToken(r, userSessionManager)
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

					// A user caller without a session authenticated with an identity provider token directly
					// rather than with a Tharsis session token.
					if userCaller, ok := caller.(*auth.UserCaller); ok && userCaller.UserSessionID == nil {
						logger.WithContextFields(ctx).With(
							"subject", userCaller.GetSubject(),
							"userAgent", r.UserAgent(),
						).Warnf("Authenticating with an identity provider token is deprecated; " +
							"log in to obtain a Tharsis session token instead")
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
