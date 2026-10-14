package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	idTokenSigningAlgValuesSupported = []string{"RS256"}
	// scopesSupported are the scopes the authorize endpoint requires, all of them and no others. The CLI
	// requests the scopes listed here, so they must match what oauthCodeFlowParams.validate requires.
	scopesSupported = []string{"openid", "tharsis"}
	// oauthErrorCodePattern matches the error codes RFC 6749 defines, so that an identity provider error
	// echoed to the user can't carry arbitrary text: the callback's query string can be crafted by anyone.
	oauthErrorCodePattern = regexp.MustCompile(`^[a-z_]{1,64}$`)
)

const (
	// oauthLoginStateCookieName is the cookie that carries the state of a CLI login proxied to an external
	// identity provider, from the authorize request to the identity provider's callback. There is one per
	// browser, so a new login replaces any login in progress: a cookie per login would let a page that
	// sends a browser to the authorize endpoint repeatedly fill it with cookies until its requests fail.
	oauthLoginStateCookieName = "tharsis_oauth_login_state"
	// oauthLoginStateExpiration is how long a user has to sign in with the identity provider.
	oauthLoginStateExpiration = 10 * time.Minute
	// oauthCallbackPath is the redirect URI path registered with the external identity provider.
	oauthCallbackPath = "/oauth/callback"
	// randomValueBytes is the size of the random state, nonce and PKCE code verifier sent to the identity provider.
	randomValueBytes = 32
	// maxOAuthStateLength and maxOAuthRedirectURILength bound the client's values, which are carried in the
	// login state cookie and must keep it well within cookie and request header size limits.
	maxOAuthStateLength       = 512
	maxOAuthRedirectURILength = 512
)

// s256CodeChallengePattern matches an S256 PKCE code challenge: an unpadded base64url SHA-256 hash (RFC 7636).
var s256CodeChallengePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// loginState is the state of a CLI login proxied to an external identity provider. It is kept in a cookie
// that only the API host can set, which binds it to the browser that started the login. It isn't signed:
// nothing in it is trusted beyond what the user could already request from the authorize endpoint, and
// the user's identity comes only from the ID token, which is verified when the session is created.
type loginState struct {
	// Issuer is the identity provider the login was started with. The response and the ID token must come
	// from it, which guards against mix-up attacks (RFC 9700 section 4.4).
	Issuer        string `json:"issuer"`
	UpstreamState string `json:"upstream_state"`
	RedirectURI   string `json:"redirect_uri"`
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge"`
	Nonce         string `json:"nonce"`
	CodeVerifier  string `json:"code_verifier"`
	ExpiresAt     int64  `json:"expires_at"`
}

// OpenIDConfig represents the OpenID Connect configuration
type OpenIDConfig struct {
	Issuer                           string   `json:"issuer"`
	JwksURI                          string   `json:"jwks_uri"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
}

type oauthCodeFlowParams struct {
	clientID            string
	redirectURI         string
	responseType        string
	state               string
	scope               string
	codeChallenge       string
	codeChallengeMethod string
}

func newOAuthCodeFlowParams(values url.Values) *oauthCodeFlowParams {
	return &oauthCodeFlowParams{
		clientID:            values.Get("client_id"),
		redirectURI:         values.Get("redirect_uri"),
		responseType:        values.Get("response_type"),
		state:               values.Get("state"),
		scope:               values.Get("scope"),
		codeChallenge:       values.Get("code_challenge"),
		codeChallengeMethod: values.Get("code_challenge_method"),
	}
}

// scopesMatch returns whether a space-delimited scope requests exactly the expected scopes. Scopes are a set
// (RFC 6749 section 3.3), so order and repeats don't matter.
func scopesMatch(scope string, expected []string) bool {
	requested := strings.Fields(scope)
	slices.Sort(requested)
	return slices.Equal(slices.Compact(requested), slices.Sorted(slices.Values(expected)))
}

func (a *oauthCodeFlowParams) validate(validClientID string) error {
	if a.clientID == "" {
		return errors.New("client_id is required")
	}
	if a.clientID != validClientID {
		return errors.New("invalid client_id, only %q is supported", validClientID)
	}
	if a.redirectURI == "" {
		return errors.New("redirect_uri is required")
	}
	if len(a.redirectURI) > maxOAuthRedirectURILength {
		return errors.New("redirect_uri must be at most %d characters", maxOAuthRedirectURILength)
	}
	// Checked here as well as when the code is issued, so an invalid redirect_uri is rejected before
	// the user is sent to the login page rather than after they have signed in.
	if err := auth.ValidateOAuthRedirectURI(a.redirectURI); err != nil {
		return err
	}
	if a.responseType == "" {
		return errors.New("response_type is required")
	}
	if a.responseType != "code" {
		return errors.New("invalid response_type, only 'code' is supported")
	}
	if a.state == "" {
		return errors.New("state is required")
	}
	if len(a.state) > maxOAuthStateLength {
		return errors.New("state must be at most %d characters", maxOAuthStateLength)
	}
	if a.codeChallenge == "" {
		return errors.New("code_challenge is required")
	}
	if a.codeChallengeMethod == "" {
		return errors.New("code_challenge_method is required")
	}
	if a.codeChallengeMethod != "S256" {
		return errors.New("invalid code_challenge_method, only 'S256' is supported")
	}
	if !s256CodeChallengePattern.MatchString(a.codeChallenge) {
		return errors.New("invalid code_challenge, it must be an unpadded base64url encoded SHA-256 hash")
	}
	if a.scope == "" {
		return errors.New("scope is required")
	}
	if !scopesMatch(a.scope, scopesSupported) {
		return errors.New("invalid scope, only %q is supported", strings.Join(scopesSupported, " "))
	}

	return nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type oidcController struct {
	logger             logger.Logger
	respWriter         response.Writer
	signingKeyManager  auth.SigningKeyManager
	frontendHost       string
	frontendScheme     string
	userSessionManager auth.UserSessionManager
	oidcLoginClient    OIDCLoginClient
	oidcConfig         []byte
	clientID           string
	secureCookies      bool
}

// NewOIDCController creates an instance of oidcController. cliLoginIDP is the external identity provider that
// CLI logins without a session are proxied to, or nil when none is configured.
func NewOIDCController(
	ctx context.Context,
	logger logger.Logger,
	respWriter response.Writer,
	signingKeyManager auth.SigningKeyManager,
	userSessionManager auth.UserSessionManager,
	openIDConfigFetcher auth.OpenIDConfigFetcher,
	cliLoginIDP *config.IdpConfig,
	apiURL string,
	frontendURL string,
	clientID string,
) (Controller, error) {
	parsedURL, err := url.Parse(frontendURL)
	if err != nil {
		return nil, fmt.Errorf("invalid frontend url: %w", err)
	}

	var oidcLoginClient OIDCLoginClient
	if cliLoginIDP != nil {
		// The callback is the redirect URI to register with the identity provider.
		oidcLoginClient, err = NewOIDCLoginClient(ctx, cliLoginIDP, apiURL+oauthCallbackPath, openIDConfigFetcher)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC login client for CLI login: %w", err)
		}
	}

	oidcConfig, err := json.Marshal(&OpenIDConfig{
		Issuer:                           apiURL,
		JwksURI:                          fmt.Sprintf("%s/%s", apiURL, "oauth/discovery/keys"),
		AuthorizationEndpoint:            fmt.Sprintf("%s/%s", apiURL, "oauth/authorize"),
		TokenEndpoint:                    fmt.Sprintf("%s/%s", apiURL, "oauth/token"),
		ResponseTypesSupported:           []string{"code"},
		GrantTypesSupported:              []string{"authorization_code"},
		SubjectTypesSupported:            []string{}, // Explicitly set to empty list
		IDTokenSigningAlgValuesSupported: idTokenSigningAlgValuesSupported,
		ScopesSupported:                  scopesSupported,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenID configuration: %w", err)
	}

	return &oidcController{
		logger:             logger,
		respWriter:         respWriter,
		signingKeyManager:  signingKeyManager,
		frontendHost:       parsedURL.Host,
		frontendScheme:     parsedURL.Scheme,
		userSessionManager: userSessionManager,
		oidcLoginClient:    oidcLoginClient,
		oidcConfig:         oidcConfig,
		clientID:           clientID,
		secureCookies:      auth.SecureCookiesEnabled(apiURL),
	}, nil
}

// RegisterRoutes adds health routes to the router
func (c *oidcController) RegisterRoutes(router chi.Router) {
	router.Get("/.well-known/openid-configuration", c.GetOpenIDConfig)

	router.Get("/oauth/authorize", c.Authorize)
	router.Get(oauthCallbackPath, c.Callback)
	router.Get("/oauth/discovery/keys", c.GetKeys)

	router.Post("/oauth/login", c.UserCredentialsLogin)
	router.Post("/oauth/token", c.Token)
}

func (c *oidcController) Authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	oauthParams := newOAuthCodeFlowParams(r.URL.Query())
	if err := oauthParams.validate(c.clientID); err != nil {
		c.respWriter.RespondWithError(ctx, w, errors.Wrap(err, "invalid request parameters", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	currentSession, err := c.userSessionManager.GetCurrentSession(ctx)
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	if currentSession != nil {
		targetURL, err := c.initiateOAuthCodeFlow(ctx, currentSession.Metadata.ID, oauthParams)
		if err != nil {
			c.respWriter.RespondWithError(ctx, w, err)
			return
		}

		http.Redirect(w, r, targetURL.String(), http.StatusFound)
		return
	}

	refreshTokenCookie, _ := r.Cookie(c.userSessionManager.GetUserSessionRefreshTokenCookieName())
	if refreshTokenCookie != nil {
		refreshResponse, err := c.userSessionManager.RefreshSession(ctx, refreshTokenCookie.Value)
		if err != nil && errors.ErrorCode(err) != errors.EUnauthorized && errors.ErrorCode(err) != errors.EOptimisticLock {
			c.respWriter.RespondWithError(ctx, w, errors.Wrap(err, "failed to refresh session"))
			return
		}

		if refreshResponse != nil {
			targetURL, err := c.initiateOAuthCodeFlow(ctx, refreshResponse.Session.Metadata.ID, oauthParams)
			if err != nil {
				c.respWriter.RespondWithError(ctx, w, err)
				return
			}

			c.userSessionManager.SetUserSessionCookies(w, &auth.SetUserSessionCookiesInput{
				AccessToken:       refreshResponse.AccessToken,
				RefreshToken:      refreshResponse.RefreshToken,
				SessionExpiration: refreshResponse.Session.Expiration,
			})

			http.Redirect(w, r, targetURL.String(), http.StatusFound)
			return
		}
	}

	if c.oidcLoginClient != nil {
		// Sign the user in with the external identity provider, which returns them to Callback.
		loginURL, err := c.startExternalLogin(w, oauthParams)
		if err != nil {
			c.respWriter.RespondWithError(ctx, w, err)
			return
		}

		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	targetURL := &url.URL{
		Scheme: c.frontendScheme,
		Host:   c.frontendHost,
		Path:   "/login",
		RawQuery: url.Values{
			"client_id":             {oauthParams.clientID},
			"redirect_uri":          {oauthParams.redirectURI},
			"response_type":         {oauthParams.responseType},
			"state":                 {oauthParams.state},
			"scope":                 {oauthParams.scope},
			"code_challenge":        {oauthParams.codeChallenge},
			"code_challenge_method": {oauthParams.codeChallengeMethod},
		}.Encode(),
	}

	http.Redirect(w, r, targetURL.String(), http.StatusFound)
}

// Callback completes a CLI login proxied to the external identity provider: it redeems the identity
// provider's code, signs the user in with the ID token, and returns a Tharsis code to the CLI.
func (c *oidcController) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// The URL carries the identity provider's code: keep it out of caches and Referer headers.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")

	if c.oidcLoginClient == nil {
		c.respWriter.RespondWithError(ctx, w, errors.New("no external identity provider is configured", errors.WithErrorCode(errors.ENotFound)))
		return
	}

	query := r.URL.Query()

	stateCookie, _ := r.Cookie(c.loginStateCookieName())
	if stateCookie == nil {
		c.respWriter.RespondWithError(ctx, w, errors.New("login state not found, the login may have expired; restart the login", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	// The login state is single use.
	c.clearLoginStateCookie(w)

	loginState, err := decodeLoginState(stateCookie.Value)
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	// The state binds the identity provider's response to the browser that started the login. A mismatch
	// is also what a browser sees if it started a newer login, which replaced this one's cookie.
	if subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(loginState.UpstreamState)) != 1 {
		c.respWriter.RespondWithError(ctx, w, errors.New("login state does not match; if another login was started from this browser, use that one or restart the login", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	// Checked before the response is otherwise used, error responses included (RFC 9207 section 2.4).
	if err := c.checkAuthorizationResponseIssuer(loginState, query.Get("iss")); err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	if idpErr := query.Get("error"); idpErr != "" {
		// The description is logged rather than shown: it is free text, so showing it would let a crafted
		// link put any message on a Tharsis page.
		c.logger.WithContextFields(ctx).Infow("Identity provider returned an error for a CLI login",
			"error", idpErr,
			"errorDescription", query.Get("error_description"),
		)
		if !oauthErrorCodePattern.MatchString(idpErr) {
			idpErr = "unknown_error"
		}
		c.respWriter.RespondWithError(ctx, w, errors.New("identity provider login failed with error %s", idpErr, errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	code := query.Get("code")
	if code == "" {
		c.respWriter.RespondWithError(ctx, w, errors.New("identity provider did not return a code", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	idToken, err := c.oidcLoginClient.ExchangeCode(ctx, code, loginState.CodeVerifier)
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	// The token's signature is verified when the session is created below.
	parsedIDToken, err := jwt.ParseString(idToken, jwt.WithVerify(false))
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, errors.Wrap(err, "identity provider returned an invalid ID token", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}
	// The ID token must be issued by the identity provider the login was started with. CreateSession accepts
	// a token from any configured identity provider, so this is checked here.
	if parsedIDToken.Issuer() != loginState.Issuer {
		c.respWriter.RespondWithError(ctx, w, errors.New("ID token was not issued by the identity provider the login was started with", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	nonce, ok := parsedIDToken.Get("nonce")
	nonceStr, isString := nonce.(string)
	if !ok || !isString || nonceStr == "" {
		c.respWriter.RespondWithError(ctx, w, errors.New("ID token is missing the nonce claim", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}
	if subtle.ConstantTimeCompare([]byte(nonceStr), []byte(loginState.Nonce)) != 1 {
		c.respWriter.RespondWithError(ctx, w, errors.New("ID token nonce does not match", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	createSessionResponse, err := c.userSessionManager.CreateSession(ctx, &auth.CreateSessionInput{
		Token:     &idToken,
		UserAgent: r.UserAgent(),
	})
	if err != nil && !goerrors.Is(err, auth.ErrSessionAlreadyExists) {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	var sessionID string

	// Response will be nil if the create session failed because a session already exists
	if createSessionResponse != nil {
		// The browser gets the session too, so later logins from it skip the identity provider.
		c.userSessionManager.SetUserSessionCookies(w, &auth.SetUserSessionCookiesInput{
			AccessToken:       createSessionResponse.AccessToken,
			RefreshToken:      createSessionResponse.RefreshToken,
			CsrfToken:         &createSessionResponse.CSRFToken,
			SessionExpiration: createSessionResponse.Session.Expiration,
		})
		sessionID = createSessionResponse.Session.Metadata.ID
	} else {
		id, ok := auth.GetRequestUserSessionID(ctx)
		if !ok {
			c.respWriter.RespondWithError(ctx, w, errors.New("unable to retrieve existing session", errors.WithErrorCode(errors.EUnauthorized)))
			return
		}
		sessionID = id
	}

	targetURL, err := c.initiateOAuthCodeFlow(ctx, sessionID, &oauthCodeFlowParams{
		clientID:            c.clientID,
		redirectURI:         loginState.RedirectURI,
		state:               loginState.State,
		codeChallenge:       loginState.CodeChallenge,
		codeChallengeMethod: "S256",
	})
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	http.Redirect(w, r, targetURL.String(), http.StatusFound)
}

func (c *oidcController) UserCredentialsLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.ParseForm()
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	oauthParams := newOAuthCodeFlowParams(r.Form)

	if err := oauthParams.validate(c.clientID); err != nil {
		c.respWriter.RespondWithError(ctx, w, errors.Wrap(err, "invalid request parameters", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	createSessionResponse, err := c.userSessionManager.CreateSession(ctx, &auth.CreateSessionInput{
		Username:  &username,
		Password:  &password,
		UserAgent: r.UserAgent(),
	})
	if err != nil && !goerrors.Is(err, auth.ErrSessionAlreadyExists) {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	var sessionID string

	// Response will be nil if the create session failed because a session already exists
	if createSessionResponse != nil {
		c.userSessionManager.SetUserSessionCookies(w, &auth.SetUserSessionCookiesInput{
			AccessToken:       createSessionResponse.AccessToken,
			RefreshToken:      createSessionResponse.RefreshToken,
			CsrfToken:         &createSessionResponse.CSRFToken,
			SessionExpiration: createSessionResponse.Session.Expiration,
		})
		sessionID = createSessionResponse.Session.Metadata.ID
	} else {
		id, ok := auth.GetRequestUserSessionID(ctx)
		if !ok {
			c.respWriter.RespondWithError(ctx, w, errors.New("unable to retrieve existing session", errors.WithErrorCode(errors.EUnauthorized)))
			return
		}
		sessionID = id
	}

	targetURL, err := c.initiateOAuthCodeFlow(ctx, sessionID, oauthParams)
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	http.Redirect(w, r, targetURL.String(), http.StatusFound)
}

func (c *oidcController) Token(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.ParseForm()

	clientID := r.Form.Get("client_id")
	grantType := r.Form.Get("grant_type")

	if clientID != c.clientID {
		c.respWriter.RespondWithError(ctx, w, errors.New("invalid client id", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	switch grantType {
	case "password":
		username := r.Form.Get("username")
		password := r.Form.Get("password")

		createSessionResponse, err := c.userSessionManager.CreateSession(ctx, &auth.CreateSessionInput{
			Username:  &username,
			Password:  &password,
			UserAgent: r.UserAgent(),
		})
		if err != nil {
			c.respWriter.RespondWithError(ctx, w, err)
			return
		}

		c.respWriter.RespondWithJSON(ctx, w, &tokenResponse{
			AccessToken: createSessionResponse.AccessToken,
			ExpiresIn:   createSessionResponse.ExpiresIn,
		}, http.StatusOK)
	case "authorization_code":
		redirectURI := r.Form.Get("redirect_uri")
		code := r.Form.Get("code")
		codeVerifier := r.Form.Get("code_verifier")

		response, err := c.userSessionManager.ExchangeOAuthCodeForSessionToken(ctx, &auth.ExchangeOAuthCodeForSessionTokenInput{
			OAuthCode:         code,
			OAuthCodeVerifier: codeVerifier,
			RedirectURI:       redirectURI,
		})
		if err != nil {
			c.respWriter.RespondWithError(ctx, w, err)
			return
		}

		c.respWriter.RespondWithJSON(ctx, w, &tokenResponse{
			AccessToken: response.AccessToken,
			ExpiresIn:   response.ExpiresIn,
		}, http.StatusOK)
	default:
		// Reject unsupported grant types explicitly rather than silently falling
		// through to the authorization_code flow.
		c.respWriter.RespondWithError(ctx, w, errors.New("unsupported grant_type %q", grantType, errors.WithErrorCode(errors.EInvalid)))
	}
}

func (c *oidcController) GetOpenIDConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(c.oidcConfig); err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}
}

func (c *oidcController) GetKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	keys, err := c.signingKeyManager.GetKeys(ctx)
	if err != nil {
		c.respWriter.RespondWithError(ctx, w, err)
		return
	}

	if _, err := w.Write(keys); err != nil {
		c.respWriter.RespondWithError(ctx, w, errors.Wrap(err, "failed to marshal jwk response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (c *oidcController) initiateOAuthCodeFlow(ctx context.Context, sessionID string, oauthParams *oauthCodeFlowParams) (*url.URL, error) {
	targetURL, err := url.Parse(oauthParams.redirectURI)
	if err != nil {
		return nil, errors.Wrap(err, "invalid redirect url", errors.WithErrorCode(errors.EInvalid))
	}

	authCode, err := c.userSessionManager.InitiateSessionOauthCodeFlow(
		ctx,
		&auth.InitiateSessionOauthCodeFlowInput{
			CodeChallenge:       oauthParams.codeChallenge,
			CodeChallengeMethod: oauthParams.codeChallengeMethod,
			RedirectURI:         oauthParams.redirectURI,
			UserSessionID:       sessionID,
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initiate oauth code flow")
	}

	targetURL.RawQuery = url.Values{
		"code":      {authCode},
		"state":     {oauthParams.state},
		"client_id": {oauthParams.clientID},
	}.Encode()

	return targetURL, nil
}

// startExternalLogin stores the CLI's authorize request in a login state cookie and returns the external
// identity provider URL to send the user to.
func (c *oidcController) startExternalLogin(w http.ResponseWriter, oauthParams *oauthCodeFlowParams) (string, error) {
	upstreamState, err := randomURLSafeString()
	if err != nil {
		return "", err
	}
	nonce, err := randomURLSafeString()
	if err != nil {
		return "", err
	}
	codeVerifier, err := randomURLSafeString()
	if err != nil {
		return "", err
	}

	expiration := time.Now().Add(oauthLoginStateExpiration)
	encodedState, err := json.Marshal(&loginState{
		Issuer:        c.oidcLoginClient.Issuer(),
		UpstreamState: upstreamState,
		RedirectURI:   oauthParams.redirectURI,
		State:         oauthParams.state,
		CodeChallenge: oauthParams.codeChallenge,
		Nonce:         nonce,
		CodeVerifier:  codeVerifier,
		ExpiresAt:     expiration.Unix(),
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to encode login state")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     c.loginStateCookieName(),
		Value:    base64.RawURLEncoding.EncodeToString(encodedState),
		HttpOnly: true,
		Secure:   c.secureCookies,
		// Lax, not Strict: the cookie must be sent on the top-level redirect back from the identity provider.
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  expiration,
	})

	challenge := sha256.Sum256([]byte(codeVerifier))
	return c.oidcLoginClient.AuthCodeURL(upstreamState, nonce, base64.RawURLEncoding.EncodeToString(challenge[:])), nil
}

// checkAuthorizationResponseIssuer checks that an authorization response comes from the identity provider the
// login was started with (RFC 9207). iss is the response's iss parameter, which is empty if it has none.
func (c *oidcController) checkAuthorizationResponseIssuer(state *loginState, iss string) error {
	// The identity provider can change between the start of a login and its callback if the API's
	// configuration changes, so the login state's issuer is checked against the current one.
	if state.Issuer != c.oidcLoginClient.Issuer() {
		return errors.New("the login was started with a different identity provider; restart the login", errors.WithErrorCode(errors.EUnauthorized))
	}

	if iss == "" {
		if c.oidcLoginClient.IssuerParameterSupported() {
			return errors.New("identity provider response is missing the iss parameter", errors.WithErrorCode(errors.EUnauthorized))
		}
		return nil
	}

	// Compared exactly, as RFC 9207 requires: no normalization such as removing a trailing slash.
	if iss != state.Issuer {
		return errors.New("identity provider response was issued by %q, not the identity provider the login was started with", iss, errors.WithErrorCode(errors.EUnauthorized))
	}

	return nil
}

// decodeLoginState decodes and checks a login state cookie value.
func decodeLoginState(value string) (*loginState, error) {
	invalid := errors.New("login state is invalid; restart the login", errors.WithErrorCode(errors.EUnauthorized))

	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, invalid
	}

	var state loginState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return nil, invalid
	}

	if state.Issuer == "" || state.UpstreamState == "" || state.RedirectURI == "" || state.State == "" || state.CodeChallenge == "" || state.Nonce == "" || state.CodeVerifier == "" {
		return nil, invalid
	}

	// The cookie expires on its own; this also covers a browser that kept it.
	if time.Now().After(time.Unix(state.ExpiresAt, 0)) {
		return nil, errors.New("login state has expired; restart the login", errors.WithErrorCode(errors.EUnauthorized))
	}

	return &state, nil
}

func (c *oidcController) clearLoginStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.loginStateCookieName(),
		Value:    "",
		HttpOnly: true,
		Secure:   c.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1,
	})
}

func (c *oidcController) loginStateCookieName() string {
	if c.secureCookies {
		// The __Host- prefix means only this host can set the cookie, over https, so it can't be planted by
		// another subdomain or a network attacker.
		return "__Host-" + oauthLoginStateCookieName
	}
	return oauthLoginStateCookieName
}

// randomURLSafeString returns random bytes encoded as unpadded base64url, which is also a valid PKCE code verifier.
func randomURLSafeString() (string, error) {
	b := make([]byte, randomValueBytes)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Wrap(err, "failed to generate random value")
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
