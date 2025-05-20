package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestOIDCTokenVerifier_VerifyToken(t *testing.T) {
	// Create a test RSA key for signing tokens
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to JWK
	key, err := jwk.FromRaw(privateKey)
	require.NoError(t, err)

	// Set key ID and algorithm
	err = key.Set(jwk.KeyIDKey, "test-key-id")
	require.NoError(t, err)
	err = key.Set(jwk.AlgorithmKey, jwa.RS256)
	require.NoError(t, err)

	// Create a JWK set with our test key
	keySet := jwk.NewSet()
	err = keySet.AddKey(key)
	require.NoError(t, err)

	// Create a public key set for verification
	pubKey, err := jwk.FromRaw(privateKey.Public())
	require.NoError(t, err)
	err = pubKey.Set(jwk.KeyIDKey, "test-key-id")
	require.NoError(t, err)
	err = pubKey.Set(jwk.AlgorithmKey, jwa.RS256)
	require.NoError(t, err)

	pubKeySet := jwk.NewSet()
	err = pubKeySet.AddKey(pubKey)
	require.NoError(t, err)

	// Create a valid token 1
	validToken1 := jwt.New()
	err = validToken1.Set(jwt.IssuerKey, "https://example.com")
	require.NoError(t, err)
	err = validToken1.Set(jwt.SubjectKey, "test-subject")
	require.NoError(t, err)
	err = validToken1.Set(jwt.AudienceKey, []string{"test-audience"})
	require.NoError(t, err)
	err = validToken1.Set(jwt.ExpirationKey, time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)

	// Sign the token 1
	signedToken1, err := jwt.Sign(validToken1, jwt.WithKey(jwa.RS256, key))
	require.NoError(t, err)

	// Create a valid token 2
	validToken2 := jwt.New()
	err = validToken2.Set(jwt.IssuerKey, "example.com")
	require.NoError(t, err)
	err = validToken2.Set(jwt.SubjectKey, "test-subject")
	require.NoError(t, err)
	err = validToken2.Set(jwt.AudienceKey, []string{"test-audience"})
	require.NoError(t, err)
	err = validToken2.Set(jwt.ExpirationKey, time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)

	// Sign the token 2
	signedToken2, err := jwt.Sign(validToken2, jwt.WithKey(jwa.RS256, key))
	require.NoError(t, err)

	// Create an expired token
	expiredToken := jwt.New()
	err = expiredToken.Set(jwt.IssuerKey, "https://example.com")
	require.NoError(t, err)
	err = expiredToken.Set(jwt.SubjectKey, "test-subject")
	require.NoError(t, err)
	err = expiredToken.Set(jwt.ExpirationKey, time.Now().Add(-time.Hour).Unix())
	require.NoError(t, err)

	// Sign the expired token
	signedExpiredToken, err := jwt.Sign(expiredToken, jwt.WithKey(jwa.RS256, key))
	require.NoError(t, err)

	// Create an invalid token (malformed)
	invalidToken := "invalid-token"

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		keySetJSON, _ := json.Marshal(pubKeySet)
		w.Header().Set("Content-Type", "application/json")
		w.Write(keySetJSON)
	}))
	defer jwksServer.Close()

	tests := []struct {
		name               string
		token              string
		issuer             string
		setupMocks         func(*MockOpenIDConfigFetcher)
		validationOptions  []jwt.ValidateOption
		expectErrorMessage string
	}{
		{
			name:   "valid token with exact issuer",
			token:  string(signedToken1),
			issuer: "https://example.com",
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					&OIDCConfiguration{
						Issuer:  "https://example.com",
						JwksURI: jwksServer.URL,
					}, nil)
			},
			validationOptions: []jwt.ValidateOption{
				jwt.WithAudience("test-audience"),
			},
		},
		{
			name:   "valid token with issuer that ends with a slash",
			token:  string(signedToken1),
			issuer: "https://example.com/",
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					&OIDCConfiguration{
						Issuer:  "https://example.com",
						JwksURI: jwksServer.URL,
					}, nil)
			},
			validationOptions: []jwt.ValidateOption{
				jwt.WithAudience("test-audience"),
			},
		},
		{
			name:   "valid token with issuer as hostname instead of URL",
			token:  string(signedToken2),
			issuer: "https://example.com",
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					&OIDCConfiguration{
						Issuer:  "https://example.com",
						JwksURI: jwksServer.URL,
					}, nil)
			},
			validationOptions: []jwt.ValidateOption{
				jwt.WithAudience("test-audience"),
			},
		},
		{
			name:   "invalid audience",
			token:  string(signedToken1),
			issuer: "https://example.com",
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					&OIDCConfiguration{
						Issuer:  "https://example.com",
						JwksURI: jwksServer.URL,
					}, nil)
			},
			validationOptions: []jwt.ValidateOption{
				jwt.WithAudience("invalid-audience"),
			},
			expectErrorMessage: "\"aud\" not satisfied",
		},
		{
			name:               "expired token",
			token:              string(signedExpiredToken),
			issuer:             "https://example.com",
			expectErrorMessage: "failed to decode token \"exp\" not satisfied",
		},
		{
			name:               "invalid token format",
			token:              invalidToken,
			issuer:             "https://example.com",
			expectErrorMessage: "failed to decode token",
		},
		{
			name:               "invalid issuer",
			token:              string(signedToken1),
			issuer:             "https://invalid.com",
			expectErrorMessage: "invalid issuer",
		},
		{
			name:               "invalid issuer for token with hostname as issuer",
			token:              string(signedToken2),
			issuer:             "invalid.com",
			expectErrorMessage: "invalid issuer",
		},
		{
			name:   "error fetching OIDC config",
			token:  string(signedToken1),
			issuer: "https://example.com",
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					nil, fmt.Errorf("failed to fetch OIDC config"))
			},
			expectErrorMessage: "failed to get OIDC config",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockFetcher := NewMockOpenIDConfigFetcher(t)

			if test.setupMocks != nil {
				test.setupMocks(mockFetcher)
			}

			// Create the token verifier with the test issuer
			verifier := NewOIDCTokenVerifier(ctx, []string{test.issuer}, mockFetcher, false)

			// Verify the token
			token, err := verifier.VerifyToken(ctx, test.token, test.validationOptions)

			if test.expectErrorMessage != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
				assert.Equal(t, NormalizeOIDCIssuer(test.issuer), NormalizeOIDCIssuer(token.Issuer()))
			}
		})
	}
}

func TestOIDCTokenVerifier_loadOIDCConfig(t *testing.T) {
	tests := []struct {
		name               string
		issuer             string
		issuers            []string
		setupMocks         func(*MockOpenIDConfigFetcher)
		enableCache        bool
		expectErrorMessage string
	}{
		{
			name:    "valid issuer with cache enabled",
			issuer:  "https://example.com",
			issuers: []string{"https://example.com"},
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					&OIDCConfiguration{
						Issuer:  "https://example.com",
						JwksURI: "https://example.com/jwks",
					}, nil)
			},
			enableCache: true,
		},
		{
			name:    "valid issuer with cache disabled",
			issuer:  "https://example.com",
			issuers: []string{"https://example.com"},
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					&OIDCConfiguration{
						Issuer:  "https://example.com",
						JwksURI: "https://example.com/jwks",
					}, nil)
			},
			enableCache: false,
		},
		{
			name:    "error fetching OIDC config",
			issuer:  "https://example.com",
			issuers: []string{"https://example.com"},
			setupMocks: func(mockFetcher *MockOpenIDConfigFetcher) {
				mockFetcher.On("GetOpenIDConfig", mock.Anything, "https://example.com").Return(
					nil, fmt.Errorf("failed to fetch OIDC config"))
			},
			expectErrorMessage: "failed to get OIDC config",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockFetcher := NewMockOpenIDConfigFetcher(t)

			if test.setupMocks != nil {
				test.setupMocks(mockFetcher)
			}

			// Create the token verifier with the test issuers
			verifier := NewOIDCTokenVerifier(ctx, test.issuers, mockFetcher, test.enableCache)
			oidcVerifier := verifier.(*oidcTokenVerifier)

			// Call loadOIDCConfig
			config, err := oidcVerifier.loadOIDCConfig(ctx, test.issuer)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				assert.Equal(t, test.issuer, config.Issuer)

				// Test cache functionality by calling again
				cachedConfig, err := oidcVerifier.loadOIDCConfig(ctx, test.issuer)
				assert.NoError(t, err)
				assert.Equal(t, config, cachedConfig)

				// If cache is enabled, the fetcher should only be called once
				if test.enableCache {
					mockFetcher.AssertNumberOfCalls(t, "GetOpenIDConfig", 1)
				}
			}
		})
	}
}

func TestOIDCTokenVerifier_getOIDCConfigFromCache(t *testing.T) {
	tests := []struct {
		name           string
		issuer         string
		setupVerifier  func(*oidcTokenVerifier)
		expectFound    bool
		expectedConfig *OIDCConfiguration
	}{
		{
			name:   "config exists in cache",
			issuer: "https://example.com",
			setupVerifier: func(verifier *oidcTokenVerifier) {
				verifier.oidcConfigMap["https://example.com"] = &OIDCConfiguration{
					Issuer:  "https://example.com",
					JwksURI: "https://example.com/jwks",
				}
			},
			expectFound: true,
			expectedConfig: &OIDCConfiguration{
				Issuer:  "https://example.com",
				JwksURI: "https://example.com/jwks",
			},
		},
		{
			name:        "config does not exist in cache",
			issuer:      "https://example.com",
			expectFound: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockFetcher := NewMockOpenIDConfigFetcher(t)

			// Create the token verifier
			verifier := NewOIDCTokenVerifier(ctx, []string{test.issuer}, mockFetcher, false)
			oidcVerifier := verifier.(*oidcTokenVerifier)

			if test.setupVerifier != nil {
				test.setupVerifier(oidcVerifier)
			}

			// Call getOIDCConfigFromCache
			config, found := oidcVerifier.getOIDCConfigFromCache(test.issuer)

			assert.Equal(t, test.expectFound, found)
			if test.expectFound {
				assert.Equal(t, test.expectedConfig, config)
			} else {
				assert.Nil(t, config)
			}
		})
	}
}

func TestOIDCTokenVerifier_getKeySet(t *testing.T) {
	// Create a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to JWK
	key, err := jwk.FromRaw(privateKey)
	require.NoError(t, err)

	// Set key ID and algorithm
	err = key.Set(jwk.KeyIDKey, "test-key-id")
	require.NoError(t, err)

	// Create a JWK set with our test key
	keySet := jwk.NewSet()
	err = keySet.AddKey(key)
	require.NoError(t, err)

	// Create a token using the key
	token := jwt.New()
	err = token.Set(jwt.IssuerKey, "https://example.com")
	require.NoError(t, err)

	// Sign the token
	signedToken, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, key))
	require.NoError(t, err)

	tests := []struct {
		name               string
		token              []byte
		setupMocks         func() *httptest.Server
		enableCache        bool
		expectErrorMessage string
	}{
		{
			name:  "successful key set fetch without cache",
			token: signedToken,
			setupMocks: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					keySetJSON, _ := json.Marshal(keySet)
					w.Header().Set("Content-Type", "application/json")
					w.Write(keySetJSON)
				}))
			},
			enableCache: false,
		},
		{
			name:  "error fetching key set",
			token: signedToken,
			setupMocks: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			enableCache:        false,
			expectErrorMessage: "Failed to query JWK URL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			mockFetcher := NewMockOpenIDConfigFetcher(t)

			var jwksServer *httptest.Server
			if test.setupMocks != nil {
				jwksServer = test.setupMocks()
				defer jwksServer.Close()
			}

			// Create the token verifier
			verifier := NewOIDCTokenVerifier(ctx, []string{"https://example.com"}, mockFetcher, test.enableCache)
			oidcVerifier := verifier.(*oidcTokenVerifier)

			// Call getKeySet
			keySet, err := oidcVerifier.getKeySet(ctx, test.token, jwksServer.URL)

			if test.expectErrorMessage != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, keySet)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, keySet)
			}
		})
	}
}
