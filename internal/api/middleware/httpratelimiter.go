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
			ctx := r.Context()
			// Use the subject string set by the ResolveSubject middleware.
			subject := auth.GetSubject(ctx)
			if subject == nil {
				logger.WithContextFields(ctx).Errorf("No subject string in context")
				respWriter.RespondWithError(r.Context(), w, errors.New("No subject string in context"))

				return
			}

			// Check whether rate limit has been exceeded.
			tokenLimit, remaining, _, ok, err := store.TakeMany(ctx, "http-"+*subject, uint64(1))
			if err != nil {
				logger.WithContextFields(ctx).Errorf("Failed to check HTTP rate limit: %w", err)
				respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "Failed to check HTTP rate limit"))
				return
			}

			// Tell the requester the current rate limit status.
			w.Header().Add(headerRateLimit, strconv.Itoa(int(tokenLimit)))
			w.Header().Add(headerRateRemaining, strconv.Itoa(int(remaining)))
			w.Header().Add(headerRateReset, "1") // we always use a 1-second interval

			if !ok {
				logger.WithContextFields(ctx).Infof("HTTP rate limit exceeded for subject: %s", *subject)
				respWriter.RespondWithError(r.Context(), w, errors.New("request rate limit exceeded", errors.WithErrorCode(errors.ETooManyRequests)))

				// Tell the requester how long to wait before trying again.
				w.Header().Add(headerRetryAfter, "1") // we always use a 1-second interval

				return
			}

			// If the limit was not exceeded, invoke the next handler.
			next.ServeHTTP(w, r)
		})
	}
}
