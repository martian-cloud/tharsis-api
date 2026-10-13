package middleware

import (
	"io"
	"net/http"
	"sync"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// limitedBody caps how much of a request body can be read. Its limit can be changed after the root
// middleware has wrapped the body, which runs before the router knows the route, so a route can set
// its own (see LimitRequestBody).
type limitedBody struct {
	mu       sync.Mutex
	body     io.ReadCloser
	limit    int64
	read     int64
	exceeded bool
}

func (b *limitedBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// A limit lowered below what has already been read is exceeded, not a negative read size.
	if b.exceeded || b.read > b.limit {
		b.exceeded = true
		return 0, b.limitErr()
	}

	// Read at most one byte past the limit: a body of exactly the limit then ends with io.EOF, and one
	// byte more means the body is too large.
	if remaining := b.limit - b.read + 1; int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err := b.body.Read(p)
	b.read += int64(n)
	if b.read > b.limit {
		n -= int(b.read - b.limit)
		b.read = b.limit
		b.exceeded = true
		return n, b.limitErr()
	}
	return n, err
}

// limitErr is the same error type http.MaxBytesReader returns, so callers that check for it work.
func (b *limitedBody) limitErr() error {
	return &http.MaxBytesError{Limit: b.limit}
}

func (b *limitedBody) Close() error {
	return b.body.Close()
}

func (b *limitedBody) setLimit(limit int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.limit = limit
}

// MaxUnauthenticatedRequestBodySize caps the body of a request with no authenticated caller, e.g. a
// login. Those bodies are small JSON or form posts, and anyone on the internet can send them.
const MaxUnauthenticatedRequestBodySize = 1024 * 1024 // 1 MiB

// NewHTTPBodyLimitMiddleware caps the body of every request with no authenticated caller at
// MaxUnauthenticatedRequestBodySize. Without it, a single multi-GB request to an unauthenticated
// endpoint that decodes its body into memory, such as /v1/sessions, can run the API out of memory.
// Authenticated requests are not capped here: a route that needs a cap for them, or a larger one for
// unauthenticated requests, sets it with LimitRequestBody. It must run after the authentication
// middleware, which sets the caller.
func NewHTTPBodyLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth.GetCaller(r.Context()) == nil && r.Body != nil && r.Body != http.NoBody {
				r.Body = &limitedBody{body: r.Body, limit: MaxUnauthenticatedRequestBodySize}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LimitRequestBody sets a route's request body cap to limit bytes, for every caller, replacing the
// unauthenticated default. A request whose Content-Length is already over the limit is rejected
// before the handler runs.
func LimitRequestBody(limit int64, respWriter response.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				respWriter.RespondWithError(r.Context(), w, errors.New(
					"request body of %d bytes exceeds the maximum of %d bytes", r.ContentLength, limit,
					errors.WithErrorCode(errors.ETooLarge),
				))
				return
			}
			if body, ok := r.Body.(*limitedBody); ok {
				body.setLimit(limit)
			} else if r.Body != nil && r.Body != http.NoBody {
				r.Body = &limitedBody{body: r.Body, limit: limit}
			}
			next.ServeHTTP(w, r)
		})
	}
}
