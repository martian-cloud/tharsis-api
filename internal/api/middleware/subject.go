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

// getSourceIP returns the client IP for an unauthenticated request, which is the rate-limit subject.
//
// Only the right-most X-Forwarded-For entry is trusted: it is the address the load balancer in front
// of the API saw, and a load balancer appends to whatever X-Forwarded-For the client sent rather than
// replacing it, so every earlier entry is client-supplied. The Forwarded header is ignored for the
// same reason: load balancers such as AWS ALB pass it through untouched, so a client could set a new
// value on every request to get a fresh rate-limit bucket.
func getSourceIP(req *http.Request) (string, error) {
	// A client may send several X-Forwarded-For lines; the proxy's entry is the last one overall.
	if values := req.Header.Values("X-Forwarded-For"); len(values) > 0 {
		parts := strings.Split(values[len(values)-1], ",")
		if ip := stripPort(strings.TrimSpace(parts[len(parts)-1])); ip != "" {
			return ip, nil
		}
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

// stripPort removes a port from an X-Forwarded-For entry ("1.2.3.4:5678", "[::1]:5678"). Some
// proxies append the client's source port, which changes on every connection and would otherwise
// give each request its own rate-limit bucket. A bare IPv6 address is returned unchanged.
func stripPort(entry string) string {
	if host, _, err := net.SplitHostPort(entry); err == nil {
		return host
	}
	return strings.TrimSuffix(strings.TrimPrefix(entry, "["), "]")
}
