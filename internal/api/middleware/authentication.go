package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// NewAuthenticationMiddleware resolves whether the caller has authenticated.
// If so, it sets the caller on the context.
// In either case, it sets the subject string on the context.
func NewAuthenticationMiddleware(
	authenticator *auth.Authenticator,
	logger logger.Logger,
	respWriter response.Writer,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var subject string

			caller, err := authenticator.Authenticate(ctx, auth.FindToken(r), true)
			if err == nil {

				// This request is authenticated.
				ctx = auth.WithCaller(ctx, caller)
				subject = caller.GetSubject()
			} else if errors.ErrorCode(err) != errors.EUnauthorized {

				// Do not return an unauthorized error here since the service layer is responsible for determining
				// if a request requires an authenticated user
				respWriter.RespondWithError(w, errors.Wrap(err, errors.EInternal, "Error finding authentication token"))
				return
			} else {

				// This request is NOT authenticated, so use the requester's IP address
				var ip string
				ip, err = getSourceIP(r)
				if err != nil {
					logger.Errorf("Error finding client IP: %v", err)
					respWriter.RespondWithError(w, errors.Wrap(err, errors.EInvalid, "Error finding client IP"))
					return
				}
				subject = fmt.Sprintf("anonymous-%s", ip)
			}

			ctx = auth.WithSubject(ctx, subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getSourceIP(req *http.Request) (string, error) {
	// Check the Forward header
	forwardedHeader := req.Header.Get("Forwarded")
	if forwardedHeader != "" {
		parts := strings.Split(forwardedHeader, ",")
		firstPart := strings.TrimSpace(parts[0])
		subParts := strings.Split(firstPart, ";")
		for _, part := range subParts {
			normalisedPart := strings.ToLower(strings.TrimSpace(part))
			if strings.HasPrefix(normalisedPart, "for=") {
				return normalisedPart[4:], nil
			}
		}
	}

	// Check the X-Forwarded-For header
	xForwardedForHeader := req.Header.Get("X-Forwarded-For")
	if xForwardedForHeader != "" {
		parts := strings.Split(xForwardedForHeader, ",")
		firstPart := strings.TrimSpace(parts[0])
		return firstPart, nil
	}

	// Check on the request
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return "", err
	}

	if host == "::1" {
		return "127.0.0.1", nil
	}

	return host, nil
}
