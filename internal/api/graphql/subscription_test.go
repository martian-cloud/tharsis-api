package graphql

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestSubscriptionAuthenticator_BuildContext(t *testing.T) {
	const (
		uiOrigin        = "https://tharsis.example.com"
		accessCookieKey = "__Host-tharsis_access_token"
	)

	tests := []struct {
		name            string
		allowedOrigins  []string
		origin          string
		sessionCookie   bool
		authenticated   bool
		expectErrorCode errors.CodeType
		expectCaller    bool
	}{
		{
			name:           "cookie session from the UI origin",
			allowedOrigins: []string{uiOrigin},
			origin:         uiOrigin,
			sessionCookie:  true,
			authenticated:  true,
			expectCaller:   true,
		},
		{
			name:           "configured origin with a trailing slash still matches",
			allowedOrigins: []string{uiOrigin + "/"},
			origin:         uiOrigin,
			sessionCookie:  true,
			authenticated:  true,
			expectCaller:   true,
		},
		{
			// A page on another subdomain of the same site: SameSite=Strict still sends the cookie.
			name:            "cookie session from a same-site but disallowed origin",
			allowedOrigins:  []string{uiOrigin},
			origin:          "https://other.example.com",
			sessionCookie:   true,
			authenticated:   true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "cookie session with no origin",
			allowedOrigins:  []string{uiOrigin},
			sessionCookie:   true,
			authenticated:   true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:           "wildcard allowed origins accept any origin",
			allowedOrigins: []string{"*"},
			origin:         "https://other.example.com",
			sessionCookie:  true,
			authenticated:  true,
			expectCaller:   true,
		},
		{
			// A bearer token on the upgrade request cannot be supplied by a browser on the user's behalf.
			name:           "bearer token from any origin",
			allowedOrigins: []string{uiOrigin},
			origin:         "https://other.example.com",
			authenticated:  true,
			expectCaller:   true,
		},
		{
			// SDK clients authenticate later, with a token in the connection params.
			name:           "unauthenticated upgrade defers to connection params",
			allowedOrigins: []string{uiOrigin},
			origin:         "https://other.example.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := newSubscriptionAuthenticator(uiOrigin, test.allowedOrigins, accessCookieKey)

			r := httptest.NewRequest(http.MethodGet, "/graphql", nil)
			if test.origin != "" {
				r.Header.Set("Origin", test.origin)
			}
			if test.sessionCookie {
				r.AddCookie(&http.Cookie{Name: accessCookieKey, Value: "access-token"})
			}
			if test.authenticated {
				r = r.WithContext(auth.WithCaller(r.Context(), auth.NewMockCaller(t)))
			}

			ctx, err := authenticator.BuildContext(context.Background(), r)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectCaller, auth.GetCaller(ctx) != nil)
		})
	}
}
