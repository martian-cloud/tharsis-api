package middleware

import (
	"context"
	"net/http"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func validateCSRFToken(ctx context.Context, r *http.Request, requestSessionID string, sessionManager auth.UserSessionManager) error {
	csrfTokenCookie, err := r.Cookie(auth.GetUserSessionCSRFTokenCookieName())
	if err != nil {
		return errors.New("csrf token not found in cookie", errors.WithErrorCode(errors.EUnauthorized))
	}

	csrfTokenHeader := r.Header.Get(auth.CSRFTokenHeader)
	if csrfTokenHeader == "" {
		return errors.New("csrf token header is missing in request", errors.WithErrorCode(errors.EUnauthorized))
	}

	if csrfTokenCookie.Value != csrfTokenHeader {
		return errors.New("csrf token header does not match cookie value", errors.WithErrorCode(errors.EUnauthorized))
	}

	return sessionManager.VerifyCSRFToken(ctx, requestSessionID, csrfTokenHeader)
}

// NewCSRFMiddleware checks if this is a user session and verifies the CSRF token
func NewCSRFMiddleware(
	respWriter response.Writer,
	sessionManager auth.UserSessionManager,
	logger logger.Logger,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Only check CSRF token if this is a user session authenticated request and if it's not a graphql subscription websocket request
			if requestSessionID, ok := auth.GetRequestUserSessionID(r); ok && !isGraphqlSubscriptionRequest(r) {
				if err := validateCSRFToken(ctx, r, requestSessionID, sessionManager); err != nil {
					logger.WithContextFields(ctx).Infow("Request has invalid CSRF token",
						"error", err,
						"subject", ptr.ToString(auth.GetSubject(r.Context())),
					)
					respWriter.RespondWithError(r.Context(), w, err)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
