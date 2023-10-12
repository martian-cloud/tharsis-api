package middleware

import (
	"net/http"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/ratelimitstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	headerRateLimit     = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset     = "X-RateLimit-Reset"
	headerRetryAfter    = "Retry-After"
)

// HTTPRateLimiterMiddleware creates a handler for HTTP rate limiting.
func HTTPRateLimiterMiddleware(
	logger logger.Logger,
	respWriter response.Writer,
	store ratelimitstore.Store,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use the subject string set by the ResolveSubject middleware.
			subject := auth.GetSubject(r.Context())
			if subject == nil {
				logger.Errorf("No subject string in context")
				respWriter.RespondWithError(w, errors.New(errors.EInternal, "No subject string in context"))

				return
			}

			// Check whether rate limit has been exceeded.
			tokenLimit, remaining, _, ok, err := store.TakeMany(r.Context(), "http-"+*subject, uint64(1))
			if err != nil {
				logger.Errorf("Failed to check HTTP rate limit: %w", err)
				respWriter.RespondWithError(w, errors.Wrap(err, errors.EInternal, "Failed to check HTTP rate limit"))
				return
			}

			// Tell the requester the current rate limit status.
			w.Header().Add(headerRateLimit, strconv.Itoa(int(tokenLimit)))
			w.Header().Add(headerRateRemaining, strconv.Itoa(int(remaining)))
			w.Header().Add(headerRateReset, "1") // we always use a 1-second interval

			if !ok {
				logger.Infof("HTTP rate limit exceeded for subject: %s", *subject)
				respWriter.RespondWithError(w, errors.New(errors.ETooManyRequests, "request rate limit exceeded"))

				// Tell the requester how long to wait before trying again.
				w.Header().Add(headerRetryAfter, "1") // we always use a 1-second interval

				return
			}

			// If the limit was not exceeded, invoke the next handler.
			next.ServeHTTP(w, r)
		})
	}
}
