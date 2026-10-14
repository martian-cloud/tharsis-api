package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const testOIDCLoginRedirectURI = "https://api.example.com/oauth/callback"

func newTestOIDCLoginClient(t *testing.T, idp *config.IdpConfig, tokenURL string) OIDCLoginClient {
	fetcher := auth.NewMockOpenIDConfigFetcher(t)
	fetcher.On("GetOpenIDConfig", mock.Anything, idp.IssuerURL).Return(&auth.OIDCConfiguration{
		Issuer:        idp.IssuerURL,
		AuthEndpoint:  "https://idp.example.com/authorize",
		TokenEndpoint: tokenURL,
	}, nil)
	fetcher.On("HTTPClient").Return(http.DefaultClient).Maybe()

	client, err := NewOIDCLoginClient(context.Background(), idp, testOIDCLoginRedirectURI, fetcher)
	require.NoError(t, err)
	return client
}

func TestNewOIDCLoginClient_ConfigFetchError(t *testing.T) {
	fetcher := auth.NewMockOpenIDConfigFetcher(t)
	fetcher.On("GetOpenIDConfig", mock.Anything, "https://idp.example.com").Return(nil, errors.New("unreachable"))

	_, err := NewOIDCLoginClient(context.Background(), &config.IdpConfig{IssuerURL: "https://idp.example.com"}, testOIDCLoginRedirectURI, fetcher)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get OIDC config")
}

func TestOIDCLoginClient_Issuer(t *testing.T) {
	for _, issParameter := range []bool{true, false} {
		fetcher := auth.NewMockOpenIDConfigFetcher(t)
		// The issuer is taken from the discovery document, which the fetcher checks against the configured one.
		fetcher.On("GetOpenIDConfig", mock.Anything, "https://idp.example.com/").Return(&auth.OIDCConfiguration{
			Issuer:        "https://idp.example.com",
			AuthEndpoint:  "https://idp.example.com/authorize",
			TokenEndpoint: "https://idp.example.com/token",
			AuthorizationResponseIssParameterSupported: issParameter,
		}, nil)
		fetcher.On("HTTPClient").Return(http.DefaultClient)

		client, err := NewOIDCLoginClient(context.Background(), &config.IdpConfig{IssuerURL: "https://idp.example.com/"}, testOIDCLoginRedirectURI, fetcher)
		require.NoError(t, err)

		assert.Equal(t, "https://idp.example.com", client.Issuer())
		assert.Equal(t, issParameter, client.IssuerParameterSupported())
	}
}

func TestOIDCLoginClient_AuthCodeURL(t *testing.T) {
	tests := []struct {
		name        string
		scope       string
		expectScope string
	}{
		{name: "openid is added when missing", scope: "profile email", expectScope: "openid profile email"},
		{name: "configured scope is kept", scope: "openid api://tharsis/user", expectScope: "openid api://tharsis/user"},
		{name: "empty scope", scope: "", expectScope: "openid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestOIDCLoginClient(t, &config.IdpConfig{
				IssuerURL: "https://idp.example.com",
				ClientID:  "idp-client",
				Scope:     test.scope,
			}, "https://idp.example.com/token")

			authURL, err := url.Parse(client.AuthCodeURL("the-state", "the-nonce", "the-challenge"))
			require.NoError(t, err)

			assert.Equal(t, "idp.example.com", authURL.Host)
			assert.Equal(t, "/authorize", authURL.Path)
			assert.Equal(t, url.Values{
				"client_id":             {"idp-client"},
				"redirect_uri":          {testOIDCLoginRedirectURI},
				"response_type":         {"code"},
				"scope":                 {test.expectScope},
				"state":                 {"the-state"},
				"nonce":                 {"the-nonce"},
				"code_challenge":        {"the-challenge"},
				"code_challenge_method": {"S256"},
			}, authURL.Query())
		})
	}
}

func TestOIDCLoginClient_ExchangeCode(t *testing.T) {
	tests := []struct {
		name              string
		clientSecret      string
		status            int
		body              map[string]string
		rawBody           string
		expectIDToken     string
		expectErrContains string
	}{
		{
			name:          "public client",
			status:        http.StatusOK,
			body:          map[string]string{"id_token": "the-id-token", "access_token": "ignored"},
			expectIDToken: "the-id-token",
		},
		{
			name:          "confidential client uses basic auth",
			clientSecret:  "s3cret/+",
			status:        http.StatusOK,
			body:          map[string]string{"id_token": "the-id-token"},
			expectIDToken: "the-id-token",
		},
		{
			name:              "oauth error response",
			status:            http.StatusBadRequest,
			body:              map[string]string{"error": "invalid_grant", "error_description": "code expired"},
			expectErrContains: "invalid_grant: code expired",
		},
		{
			name:              "non oauth error response",
			status:            http.StatusInternalServerError,
			expectErrContains: "returned 500 Internal Server Error: (empty body)",
		},
		{
			name:              "non json error response",
			status:            http.StatusBadGateway,
			rawBody:           "<html>\n  Bad Gateway\n</html>",
			expectErrContains: "returned 502 Bad Gateway: <html> Bad Gateway </html>",
		},
		{
			name:              "long non json error response is truncated",
			status:            http.StatusBadGateway,
			rawBody:           strings.Repeat("x", 1000),
			expectErrContains: strings.Repeat("x", 512) + "...",
		},
		{
			name:              "non json success response",
			status:            http.StatusOK,
			rawBody:           "<html>Sign in</html>",
			expectErrContains: "failed to parse identity provider token response",
		},
		{
			name:              "missing id token",
			status:            http.StatusOK,
			body:              map[string]string{"access_token": "only-access"},
			expectErrContains: "did not return an ID token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				require.NoError(t, r.ParseForm())
				assert.Equal(t, "authorization_code", r.PostForm.Get("grant_type"))
				assert.Equal(t, "the-code", r.PostForm.Get("code"))
				assert.Equal(t, "the-verifier", r.PostForm.Get("code_verifier"))
				assert.Equal(t, testOIDCLoginRedirectURI, r.PostForm.Get("redirect_uri"))

				user, pass, ok := r.BasicAuth()
				if test.clientSecret != "" {
					require.True(t, ok)
					assert.Equal(t, "idp-client", user)
					assert.Equal(t, url.QueryEscape(test.clientSecret), pass)
					assert.Empty(t, r.PostForm.Get("client_id"))
				} else {
					assert.False(t, ok)
					assert.Equal(t, "idp-client", r.PostForm.Get("client_id"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				if test.body != nil {
					_ = json.NewEncoder(w).Encode(test.body)
				}
				if test.rawBody != "" {
					_, _ = w.Write([]byte(test.rawBody))
				}
			}))
			defer server.Close()

			client := newTestOIDCLoginClient(t, &config.IdpConfig{
				IssuerURL:    "https://idp.example.com",
				ClientID:     "idp-client",
				ClientSecret: test.clientSecret,
			}, server.URL)

			idToken, err := client.ExchangeCode(context.Background(), "the-code", "the-verifier")
			if test.expectErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrContains)
				assert.Equal(t, errors.EUnauthorized, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expectIDToken, idToken)
		})
	}
}
