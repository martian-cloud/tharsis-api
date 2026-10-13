package middleware

import (
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestHTTPBodyLimitMiddleware(t *testing.T) {
	logger, _ := logger.NewForTest()
	respWriter := response.NewWriter(logger)

	overLimit := strings.Repeat("x", MaxUnauthenticatedRequestBodySize+1)

	tests := []struct {
		name string
		body string
		// authenticated sends the request with a caller, as set by the authentication middleware.
		authenticated bool
		// routeLimit, when set, is applied with LimitRequestBody, as a route would.
		routeLimit    *int64
		contentLength int64
		expectTooBig  bool
		expectStatus  int
	}{
		{name: "unauthenticated body under the limit", body: "12345"},
		{name: "unauthenticated body exactly the limit", body: strings.Repeat("x", MaxUnauthenticatedRequestBodySize)},
		{name: "unauthenticated body over the limit", body: overLimit, expectTooBig: true},
		{name: "authenticated body is not capped", body: overLimit, authenticated: true},
		{
			// e.g. VCS webhooks, which authenticate in the handler rather than the middleware.
			name:       "route raises the limit for an unauthenticated request",
			body:       overLimit,
			routeLimit: ptrInt64(MaxUnauthenticatedRequestBodySize + 10),
		},
		{
			// e.g. TFE CreateRun, which reads the whole body into memory.
			name:          "route caps an authenticated request",
			body:          "123456",
			authenticated: true,
			routeLimit:    ptrInt64(5),
			expectTooBig:  true,
		},
		{
			name:          "route limit allows a body exactly the limit",
			body:          "12345",
			authenticated: true,
			routeLimit:    ptrInt64(5),
		},
		{
			name:          "declared length over the route limit is rejected before the handler",
			body:          "123456",
			authenticated: true,
			routeLimit:    ptrInt64(5),
			contentLength: 6,
			expectStatus:  http.StatusRequestEntityTooLarge,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var readBody string
			var readErr error
			handlerCalled := false
			var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				b, err := io.ReadAll(r.Body)
				readBody, readErr = string(b), err
				w.WriteHeader(http.StatusOK)
			})
			if test.routeLimit != nil {
				handler = LimitRequestBody(*test.routeLimit, respWriter)(handler)
			}
			handler = NewHTTPBodyLimitMiddleware()(handler)

			req := httptest.NewRequest(http.MethodPost, "/v1/sessions", strings.NewReader(test.body))
			if test.authenticated {
				req = req.WithContext(auth.WithCaller(req.Context(), auth.NewMockCaller(t)))
			}
			// Leave the length unknown unless the test sets it, as with a chunked upload.
			req.ContentLength = test.contentLength
			if test.contentLength == 0 {
				req.ContentLength = -1
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if test.expectStatus != 0 {
				assert.Equal(t, test.expectStatus, rec.Code)
				assert.False(t, handlerCalled)
				return
			}

			require.True(t, handlerCalled)
			if test.expectTooBig {
				var maxBytesErr *http.MaxBytesError
				assert.True(t, stderrors.As(readErr, &maxBytesErr), "expected an http.MaxBytesError, got %v", readErr)
				return
			}
			require.NoError(t, readErr)
			assert.Equal(t, test.body, readBody)
		})
	}
}

func ptrInt64(v int64) *int64 { return &v }

// A limit lowered below what has already been read reports the body as too large rather than panicking.
func TestLimitedBody_LimitLoweredBelowRead(t *testing.T) {
	body := &limitedBody{body: io.NopCloser(strings.NewReader("0123456789")), limit: 100}

	buf := make([]byte, 8)
	n, err := body.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 8, n)

	body.setLimit(4)

	n, err = body.Read(buf)
	assert.Equal(t, 0, n)
	var maxBytesErr *http.MaxBytesError
	assert.ErrorAs(t, err, &maxBytesErr)
}
