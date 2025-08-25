package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	log "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// NewSubjectMiddleware creates a middleware that sets the subject in the context.
// If the request is authenticated, the subject is taken from the Caller.
// If not authenticated, the subject is set to "anonymous-<IP_ADDRESS>".
func NewSubjectMiddleware(
	logger log.Logger,
	respWriter response.Writer,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var subject string

			caller := auth.GetCaller(ctx)

			if caller == nil {
				// This request is NOT authenticated, so use the requester's IP address as the subject
				var ip string
				ip, err := getSourceIP(r)
				if err != nil {
					logger.WithContextFields(ctx).Errorf("Error finding client IP: %v", err)
					respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "Error finding client IP", errors.WithErrorCode(errors.EInvalid)))
					return
				}
				subject = fmt.Sprintf("anonymous-%s", ip)
			} else {
				subject = caller.GetSubject()
			}

			// Add subject to auth context
			ctx = auth.WithSubject(ctx, subject)
			// Add subject to logger context
			ctx = log.WithSubject(ctx, subject)

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
