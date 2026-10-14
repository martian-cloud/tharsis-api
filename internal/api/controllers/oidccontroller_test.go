package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	testAPIURL         = "https://api.example.com"
	testUIURL          = "https://ui.example.com"
	testClientID       = "tharsis"
	testCLIRedirectURI = "http://localhost:10000/login"
	testIssuer         = "https://idp.example.com"
	// testCLIChallenge has the shape of an S256 code challenge.
	testCLIChallenge = "cli-challenge-0123456789abcdefghijklmnopqrs"
	// testUpstreamState has the shape of a generated state value.
	testUpstreamState = "upstream-state-0123456789abcdefghijklmnopqr"
)

func newTestOIDCController(
	t *testing.T,
	signingKeyManager auth.SigningKeyManager,
	userSessionManager auth.UserSessionManager,
	oidcLoginClient OIDCLoginClient,
) *oidcController {
	testLogger, _ := logger.NewForTest()
	controller, err := NewOIDCController(
		context.Background(),
		testLogger,
		response.NewWriter(testLogger),
		signingKeyManager,
		userSessionManager,
		auth.NewMockOpenIDConfigFetcher(t),
		nil,
		testAPIURL,
		testUIURL,
		testClientID,
	)
	require.NoError(t, err)

	// The login client is replaced by the test's, rather than built from an identity provider's config.
	oidcController := controller.(*oidcController)
	oidcController.oidcLoginClient = oidcLoginClient
	return oidcController
}

func TestNewOIDCController_CLILoginIDP(t *testing.T) {
	idp := &config.IdpConfig{IssuerURL: testIssuer, ClientID: "idp-client"}
	testLogger, _ := logger.NewForTest()

	newController := func(fetcher auth.OpenIDConfigFetcher, cliLoginIDP *config.IdpConfig) (Controller, error) {
		return NewOIDCController(context.Background(), testLogger, response.NewWriter(testLogger),
			auth.NewMockSigningKeyManager(t), auth.NewMockUserSessionManager(t), fetcher, cliLoginIDP,
			testAPIURL, testUIURL, testClientID)
	}

	t.Run("no identity provider", func(t *testing.T) {
		controller, err := newController(auth.NewMockOpenIDConfigFetcher(t), nil)
		require.NoError(t, err)
		assert.Nil(t, controller.(*oidcController).oidcLoginClient)
	})

	t.Run("identity provider", func(t *testing.T) {
		fetcher := auth.NewMockOpenIDConfigFetcher(t)
		fetcher.On("GetOpenIDConfig", mock.Anything, testIssuer).Return(&auth.OIDCConfiguration{
			Issuer:        testIssuer,
			AuthEndpoint:  testIssuer + "/authorize",
			TokenEndpoint: testIssuer + "/token",
		}, nil)
		fetcher.On("HTTPClient").Return(http.DefaultClient)

		controller, err := newController(fetcher, idp)
		require.NoError(t, err)

		loginClient := controller.(*oidcController).oidcLoginClient
		require.NotNil(t, loginClient)
		authURL, err := url.Parse(loginClient.AuthCodeURL("state", "nonce", "challenge"))
		require.NoError(t, err)
		assert.Equal(t, testAPIURL+oauthCallbackPath, authURL.Query().Get("redirect_uri"))
	})

	t.Run("identity provider config can't be fetched", func(t *testing.T) {
		fetcher := auth.NewMockOpenIDConfigFetcher(t)
		fetcher.On("GetOpenIDConfig", mock.Anything, testIssuer).Return(nil, errors.New("unreachable"))

		_, err := newController(fetcher, idp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create OIDC login client for CLI login")
	})
}

func cliAuthorizeQuery() url.Values {
	return url.Values{
		"client_id":             {testClientID},
		"redirect_uri":          {testCLIRedirectURI},
		"response_type":         {"code"},
		"state":                 {"cli-state"},
		"scope":                 {"openid tharsis"},
		"code_challenge":        {testCLIChallenge},
		"code_challenge_method": {"S256"},
	}
}

func TestOIDCController_GetOpenIDConfig(t *testing.T) {
	controller := newTestOIDCController(t, auth.NewMockSigningKeyManager(t), auth.NewMockUserSessionManager(t), nil)

	recorder := httptest.NewRecorder()
	controller.GetOpenIDConfig(recorder, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)

	var oidcConfig OpenIDConfig
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &oidcConfig))
	assert.Equal(t, testAPIURL, oidcConfig.Issuer)
	assert.Equal(t, testAPIURL+"/oauth/authorize", oidcConfig.AuthorizationEndpoint)
	assert.Equal(t, testAPIURL+"/oauth/token", oidcConfig.TokenEndpoint)
	// The CLI joins these with a space to build the scope it sends to the authorize endpoint.
	assert.Equal(t, []string{"openid", "tharsis"}, oidcConfig.ScopesSupported)
}

func encodeTestLoginState(t *testing.T, state *loginState) string {
	encoded, err := json.Marshal(state)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func TestOIDCController_Authorize_ExternalLogin(t *testing.T) {
	userSessionManager := auth.NewMockUserSessionManager(t)
	oidcLoginClient := NewMockOIDCLoginClient(t)

	userSessionManager.On("GetCurrentSession", mock.Anything).Return(nil, nil)
	userSessionManager.On("GetUserSessionRefreshTokenCookieName").Return("tharsis_refresh_token")

	var upstreamState, nonce, challenge string
	oidcLoginClient.On("Issuer").Return(testIssuer)
	oidcLoginClient.On("AuthCodeURL", mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			upstreamState, nonce, challenge = args.String(0), args.String(1), args.String(2)
		}).
		Return("https://idp.example.com/authorize?client_id=idp")

	// The login state needs no signing key: nothing is signed.
	controller := newTestOIDCController(t, auth.NewMockSigningKeyManager(t), userSessionManager, oidcLoginClient)

	recorder := httptest.NewRecorder()
	controller.Authorize(recorder, httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+cliAuthorizeQuery().Encode(), nil))

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "https://idp.example.com/authorize?client_id=idp", recorder.Header().Get("Location"))

	assert.Len(t, upstreamState, 43)
	assert.Len(t, nonce, 43)
	assert.NotEqual(t, upstreamState, nonce)

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	assert.Equal(t, "__Host-"+oauthLoginStateCookieName, cookie.Name)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/", cookie.Path)
	assert.Empty(t, cookie.Domain)
	assert.WithinDuration(t, time.Now().Add(oauthLoginStateExpiration), cookie.Expires, time.Minute)

	// The cookie carries the CLI's request and the PKCE values for the identity provider.
	state, err := decodeLoginState(cookie.Value)
	require.NoError(t, err)
	assert.Equal(t, testIssuer, state.Issuer)
	assert.Equal(t, upstreamState, state.UpstreamState)
	assert.Equal(t, testCLIRedirectURI, state.RedirectURI)
	assert.Equal(t, "cli-state", state.State)
	assert.Equal(t, testCLIChallenge, state.CodeChallenge)
	assert.Equal(t, nonce, state.Nonce)
	assert.WithinDuration(t, time.Now().Add(oauthLoginStateExpiration), time.Unix(state.ExpiresAt, 0), time.Minute)

	verifierHash := sha256.Sum256([]byte(state.CodeVerifier))
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(verifierHash[:]), challenge)
}

func TestScopesMatch(t *testing.T) {
	tests := []struct {
		scope  string
		expect bool
	}{
		{scope: "openid tharsis", expect: true},
		{scope: "tharsis openid", expect: true},
		{scope: "openid  tharsis openid", expect: true},
		{scope: "openid", expect: false},
		{scope: "openid tharsis profile", expect: false},
		{scope: "", expect: false},
	}

	for _, test := range tests {
		t.Run(test.scope, func(t *testing.T) {
			assert.Equal(t, test.expect, scopesMatch(test.scope, scopesSupported))
		})
	}
}

func TestOIDCController_Authorize_RejectsInvalidParameters(t *testing.T) {
	tests := []struct {
		name  string
		param string
		value string
	}{
		{name: "state too long", param: "state", value: strings.Repeat("s", maxOAuthStateLength+1)},
		{name: "redirect uri too long", param: "redirect_uri", value: testCLIRedirectURI + "?" + strings.Repeat("r", maxOAuthRedirectURILength)},
		{name: "code challenge that is not an S256 hash", param: "code_challenge", value: strings.Repeat("c", 4000)},
		{name: "response type other than code", param: "response_type", value: "token"},
		{name: "scope missing tharsis", param: "scope", value: "openid"},
		{name: "scope with an unsupported scope", param: "scope", value: "openid tharsis profile"},
		{name: "code challenge method other than S256", param: "code_challenge_method", value: "plain"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Rejected before any cookie is set or the identity provider is involved.
			controller := newTestOIDCController(t, auth.NewMockSigningKeyManager(t), auth.NewMockUserSessionManager(t), NewMockOIDCLoginClient(t))

			query := cliAuthorizeQuery()
			query.Set(test.param, test.value)

			recorder := httptest.NewRecorder()
			controller.Authorize(recorder, httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+query.Encode(), nil))

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Empty(t, recorder.Result().Cookies())
		})
	}
}

func TestOIDCController_Authorize_NoExternalIDPRedirectsToUILogin(t *testing.T) {
	userSessionManager := auth.NewMockUserSessionManager(t)
	userSessionManager.On("GetCurrentSession", mock.Anything).Return(nil, nil)
	userSessionManager.On("GetUserSessionRefreshTokenCookieName").Return("tharsis_refresh_token")

	controller := newTestOIDCController(t, auth.NewMockSigningKeyManager(t), userSessionManager, nil)

	recorder := httptest.NewRecorder()
	controller.Authorize(recorder, httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+cliAuthorizeQuery().Encode(), nil))

	assert.Equal(t, http.StatusFound, recorder.Code)
	location, err := url.Parse(recorder.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "ui.example.com", location.Host)
	assert.Equal(t, "/login", location.Path)
}

func TestOIDCController_Callback_NoExternalIDP(t *testing.T) {
	controller := newTestOIDCController(t, auth.NewMockSigningKeyManager(t), auth.NewMockUserSessionManager(t), nil)

	recorder := httptest.NewRecorder()
	controller.Callback(recorder, httptest.NewRequest(http.MethodGet, oauthCallbackPath+"?state=s&code=c", nil))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestOIDCController_Callback(t *testing.T) {
	validState := func() *loginState {
		return &loginState{
			Issuer:        testIssuer,
			UpstreamState: testUpstreamState,
			RedirectURI:   testCLIRedirectURI,
			State:         "cli-state",
			CodeChallenge: testCLIChallenge,
			Nonce:         "upstream-nonce",
			CodeVerifier:  "upstream-verifier",
			ExpiresAt:     time.Now().Add(5 * time.Minute).Unix(),
		}
	}

	// A nil nonce leaves the claim out.
	idToken := func(issuer string, nonce any) string {
		token := jwt.New()
		require.NoError(t, token.Set(jwt.IssuerKey, issuer))
		if nonce != nil {
			require.NoError(t, token.Set("nonce", nonce))
		}
		signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte("test-key")))
		require.NoError(t, err)
		return string(signed)
	}
	validIDToken := idToken(testIssuer, "upstream-nonce")

	session := &models.UserSession{
		Metadata:   models.ResourceMetadata{ID: "session-1"},
		Expiration: time.Now().Add(time.Hour),
	}

	cookieName := "__Host-" + oauthLoginStateCookieName

	type testCase struct {
		name              string
		cookieName        string
		cookieValue       string
		query             url.Values
		exchangeIDToken   string
		exchangeErr       error
		expectCreate      bool
		createErr         error
		currentIssuer     string
		issParameter      bool
		expectStatus      int
		expectLocation    string
		expectErrContains string
		expectNotContains string
		expectCleared     bool
	}

	tests := []testCase{
		{
			name:           "successful login with a matching iss parameter",
			query:          url.Values{"state": {testUpstreamState}, "code": {"idp-code"}, "iss": {testIssuer}},
			issParameter:   true,
			expectCreate:   true,
			expectStatus:   http.StatusFound,
			expectLocation: testCLIRedirectURI + "?client_id=tharsis&code=tharsis-code&state=" + url.QueryEscape("cli-state"),
			expectCleared:  true,
		},
		{
			name:              "iss parameter from another issuer",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}, "iss": {"https://evil.example.com"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "not the identity provider the login was started with",
			expectCleared:     true,
		},
		{
			name:              "iss parameter is compared exactly",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}, "iss": {testIssuer + "/"}},
			issParameter:      true,
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "not the identity provider the login was started with",
			expectCleared:     true,
		},
		{
			name:              "iss parameter is checked on error responses too",
			query:             url.Values{"state": {testUpstreamState}, "error": {"access_denied"}, "iss": {"https://evil.example.com"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "not the identity provider the login was started with",
			expectCleared:     true,
		},
		{
			name:              "missing iss parameter from an identity provider that sends it",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			issParameter:      true,
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "missing the iss parameter",
			expectCleared:     true,
		},
		{
			name:              "identity provider changed since the login was started",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			currentIssuer:     "https://new-idp.example.com",
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "started with a different identity provider",
			expectCleared:     true,
		},
		{
			name:              "id token from another issuer",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			exchangeIDToken:   idToken("https://other-idp.example.com", "upstream-nonce"),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "ID token was not issued by the identity provider",
			expectCleared:     true,
		},
		{
			name: "login state without an issuer",
			cookieValue: func() string {
				s := validState()
				s.Issuer = ""
				return encodeTestLoginState(t, s)
			}(),
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state is invalid",
			expectCleared:     true,
		},
		{
			name:           "successful login returns a tharsis code to the cli",
			query:          url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectCreate:   true,
			expectStatus:   http.StatusFound,
			expectLocation: testCLIRedirectURI + "?client_id=tharsis&code=tharsis-code&state=cli-state",
			expectCleared:  true,
		},
		{
			name:              "browser that did not start the login has no login state cookie",
			cookieName:        "unrelated_cookie",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state not found",
		},
		{
			name:              "state does not match the login state",
			query:             url.Values{"state": {"other-state"}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state does not match",
			expectCleared:     true,
		},
		{
			name:              "cookie without the host prefix is not accepted",
			cookieName:        oauthLoginStateCookieName,
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state not found",
		},
		{
			name:              "malformed login state",
			cookieValue:       "not-base64-json!",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state is invalid",
			expectCleared:     true,
		},
		{
			name: "login state missing fields",
			cookieValue: func() string {
				s := validState()
				s.CodeVerifier = ""
				return encodeTestLoginState(t, s)
			}(),
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state is invalid",
			expectCleared:     true,
		},
		{
			name: "expired login state",
			cookieValue: func() string {
				s := validState()
				s.ExpiresAt = time.Now().Add(-time.Minute).Unix()
				return encodeTestLoginState(t, s)
			}(),
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "login state has expired",
			expectCleared:     true,
		},
		{
			name:              "identity provider returned an error",
			query:             url.Values{"state": {testUpstreamState}, "error": {"access_denied"}, "error_description": {"user cancelled"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "identity provider login failed with error access_denied",
			expectNotContains: "user cancelled",
			expectCleared:     true,
		},
		{
			name:              "identity provider error code that isn't an oauth error code",
			query:             url.Values{"state": {testUpstreamState}, "error": {"Call support at 555-0100"}},
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "identity provider login failed with error unknown_error",
			expectNotContains: "555-0100",
			expectCleared:     true,
		},
		{
			name:              "missing code",
			query:             url.Values{"state": {testUpstreamState}},
			expectStatus:      http.StatusBadRequest,
			expectErrContains: "did not return a code",
			expectCleared:     true,
		},
		{
			name:              "code exchange fails",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			exchangeErr:       errors.New("invalid_grant", errors.WithErrorCode(errors.EUnauthorized)),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "invalid_grant",
			expectCleared:     true,
		},
		{
			name:              "nonce does not match",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			exchangeIDToken:   idToken(testIssuer, "other-nonce"),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "nonce does not match",
			expectCleared:     true,
		},
		{
			name:              "id token without a nonce",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			exchangeIDToken:   idToken(testIssuer, nil),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "missing the nonce claim",
			expectCleared:     true,
		},
		{
			name:              "id token with a non-string nonce",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			exchangeIDToken:   idToken(testIssuer, 12345),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "missing the nonce claim",
			expectCleared:     true,
		},
		{
			name:              "id token with an empty nonce",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			exchangeIDToken:   idToken(testIssuer, ""),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "missing the nonce claim",
			expectCleared:     true,
		},
		{
			name:              "id token is rejected when creating the session",
			query:             url.Values{"state": {testUpstreamState}, "code": {"idp-code"}},
			expectCreate:      true,
			createErr:         errors.New("oidc token is invalid", errors.WithErrorCode(errors.EUnauthorized)),
			expectStatus:      http.StatusUnauthorized,
			expectErrContains: "oidc token is invalid",
			expectCleared:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userSessionManager := auth.NewMockUserSessionManager(t)
			oidcLoginClient := NewMockOIDCLoginClient(t)

			currentIssuer := test.currentIssuer
			if currentIssuer == "" {
				currentIssuer = testIssuer
			}
			oidcLoginClient.On("Issuer").Return(currentIssuer).Maybe()
			oidcLoginClient.On("IssuerParameterSupported").Return(test.issParameter).Maybe()

			if test.exchangeErr != nil {
				oidcLoginClient.On("ExchangeCode", mock.Anything, "idp-code", "upstream-verifier").Return("", test.exchangeErr)
			} else if test.exchangeIDToken != "" || test.expectCreate {
				exchangedIDToken := test.exchangeIDToken
				if exchangedIDToken == "" {
					exchangedIDToken = validIDToken
				}
				oidcLoginClient.On("ExchangeCode", mock.Anything, "idp-code", "upstream-verifier").Return(exchangedIDToken, nil)
			}

			if test.expectCreate {
				createSession := userSessionManager.On("CreateSession", mock.Anything, mock.MatchedBy(func(input *auth.CreateSessionInput) bool {
					return input.Token != nil && *input.Token == validIDToken
				}))
				if test.createErr != nil {
					createSession.Return(nil, test.createErr)
				} else {
					createSession.Return(&auth.CreateSessionResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						CSRFToken:    "csrf-token",
						Session:      session,
					}, nil)

					userSessionManager.On("SetUserSessionCookies", mock.Anything, mock.MatchedBy(func(input *auth.SetUserSessionCookiesInput) bool {
						return input.AccessToken == "access-token" && input.RefreshToken == "refresh-token" &&
							input.CsrfToken != nil && *input.CsrfToken == "csrf-token"
					})).Return()

					userSessionManager.On("InitiateSessionOauthCodeFlow", mock.Anything, &auth.InitiateSessionOauthCodeFlowInput{
						CodeChallenge:       testCLIChallenge,
						CodeChallengeMethod: "S256",
						RedirectURI:         testCLIRedirectURI,
						UserSessionID:       "session-1",
					}).Return("tharsis-code", nil)
				}
			}

			controller := newTestOIDCController(t, auth.NewMockSigningKeyManager(t), userSessionManager, oidcLoginClient)

			reqCookieName := test.cookieName
			if reqCookieName == "" {
				reqCookieName = cookieName
			}
			reqCookieValue := test.cookieValue
			if reqCookieValue == "" {
				reqCookieValue = encodeTestLoginState(t, validState())
			}

			req := httptest.NewRequest(http.MethodGet, oauthCallbackPath+"?"+test.query.Encode(), nil)
			req.AddCookie(&http.Cookie{Name: reqCookieName, Value: reqCookieValue})
			recorder := httptest.NewRecorder()

			controller.Callback(recorder, req)

			assert.Equal(t, test.expectStatus, recorder.Code)
			assert.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
			assert.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
			if test.expectLocation != "" {
				assert.Equal(t, test.expectLocation, recorder.Header().Get("Location"))
			}
			if test.expectErrContains != "" {
				assert.Contains(t, recorder.Body.String(), test.expectErrContains)
			}
			if test.expectNotContains != "" {
				assert.NotContains(t, recorder.Body.String(), test.expectNotContains)
			}

			// The login state is single use, so any callback that read it clears it.
			var cleared bool
			for _, cookie := range recorder.Result().Cookies() {
				if cookie.Name == cookieName && cookie.MaxAge < 0 {
					cleared = true
				}
			}
			assert.Equal(t, test.expectCleared, cleared)
		})
	}
}
