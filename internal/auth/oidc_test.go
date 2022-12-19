package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOpenIDConfig(t *testing.T) {
	tests := []struct {
		name          string
		data          string
		authEndpoint  string
		tokenEndpoint string
		jwksURI       string
		expectErr     bool
	}{
		{
			name: "basic case",
			data: `{
				"issuer": "ISSUER",
				"authorization_endpoint": "https://example.com/auth",
				"token_endpoint": "https://example.com/token",
				"jwks_uri": "https://example.com/keys"
			}`,
			authEndpoint:  "https://example.com/auth",
			tokenEndpoint: "https://example.com/token",
			jwksURI:       "https://example.com/keys",
		},
		{
			name:      "invalid json response",
			data:      "{}",
			expectErr: true,
		},
		{
			name: "invalid issuer",
			data: `{
				"issuer": "http://invalidurl",
				"authorization_endpoint": "https://example.com/auth",
				"token_endpoint": "https://example.com/token",
				"jwks_uri": "https://example.com/keys"
			}`,
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var issuer string
			hf := func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/.well-known/openid-configuration" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, strings.ReplaceAll(test.data, "ISSUER", issuer))
			}
			s := httptest.NewServer(http.HandlerFunc(hf))
			defer s.Close()

			issuer = s.URL

			resp, err := NewOpenIDConfigFetcher().GetOpenIDConfig(ctx, issuer)
			if err != nil {
				assert.True(t, test.expectErr)
				return
			}

			assert.False(t, test.expectErr)

			assert.Equal(t, resp.AuthEndpoint, test.authEndpoint)
			assert.Equal(t, resp.TokenEndpoint, test.tokenEndpoint)
			assert.Equal(t, resp.JwksURI, test.jwksURI)
		})
	}
}
