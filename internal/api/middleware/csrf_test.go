package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestValidateCSRFToken(t *testing.T) {
	const (
		sessionID        = "session-1"
		csrfCookieName   = "tharsis_csrf_token"
		accessCookieName = "__Host-tharsis_access_token"
	)

	tests := []struct {
		name            string
		accessCookie    bool
		csrfCookie      string
		csrfHeader      string
		bearerToken     bool
		expectVerify    bool
		expectErrorCode errors.CodeType
	}{
		{
			name:         "cookie session with matching csrf cookie and header",
			accessCookie: true,
			csrfCookie:   "csrf-token",
			csrfHeader:   "csrf-token",
			expectVerify: true,
		},
		{
			// The CSRF cookie was removed or never sent: the browser still attached the session
			// cookie, so the request must not skip the check.
			name:            "cookie session without csrf cookie is rejected",
			accessCookie:    true,
			csrfHeader:      "csrf-token",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "cookie session without csrf header is rejected",
			accessCookie:    true,
			csrfCookie:      "csrf-token",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "csrf header does not match cookie",
			accessCookie:    true,
			csrfCookie:      "csrf-token",
			csrfHeader:      "other-token",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			// The CLI/SDK sends a session token in the Authorization header and has no cookies.
			name:        "bearer session token without cookies needs no csrf token",
			bearerToken: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sessionManager := auth.NewMockUserSessionManager(t)
			sessionManager.On("GetUserSessionCSRFTokenCookieName").Return(csrfCookieName).Maybe()
			sessionManager.On("GetUserSessionAccessTokenCookieName").Return(accessCookieName).Maybe()
			if test.expectVerify {
				sessionManager.On("VerifyCSRFToken", mock.Anything, sessionID, test.csrfHeader).Return(nil)
			}

			r := httptest.NewRequest(http.MethodPost, "/graphql", nil)
			if test.accessCookie {
				r.AddCookie(&http.Cookie{Name: accessCookieName, Value: "access-token"})
			}
			if test.csrfCookie != "" {
				r.AddCookie(&http.Cookie{Name: csrfCookieName, Value: test.csrfCookie})
			}
			if test.csrfHeader != "" {
				r.Header.Set(auth.CSRFTokenHeader, test.csrfHeader)
			}
			if test.bearerToken {
				r.Header.Set("Authorization", "Bearer access-token")
			}

			err := validateCSRFToken(context.Background(), r, sessionID, sessionManager)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			assert.NoError(t, err)
		})
	}
}
