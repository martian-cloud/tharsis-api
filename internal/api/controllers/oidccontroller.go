package controllers

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var (
	idTokenSigningAlgValuesSupported = []string{"RS256"}
)

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
	if a.responseType == "" {
		return errors.New("response_type is required")
	}
	if a.responseType != "code" {
		return errors.New("invalid response_type, only 'code' is supported")
	}
	if a.state == "" {
		return errors.New("state is required")
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
	if a.scope == "" {
		return errors.New("scope is required")
	}
	if a.scope != "openid tharsis" {
		return errors.New("invalid scope, only 'openid tharsis' is supported")
	}

	return nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type oidcController struct {
	respWriter         response.Writer
	signingKeyManager  auth.SigningKeyManager
	frontendHost       string
	frontendScheme     string
	userSessionManager auth.UserSessionManager
	oidcConfig         []byte
	clientID           string
}

// NewOIDCController creates an instance of oidcController
func NewOIDCController(
	respWriter response.Writer,
	signingKeyManager auth.SigningKeyManager,
	userSessionManager auth.UserSessionManager,
	apiURL string,
	frontendURL string,
	clientID string,
) (Controller, error) {
	parsedURL, err := url.Parse(frontendURL)
	if err != nil {
		return nil, fmt.Errorf("invalid frontend url: %w", err)
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
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenID configuration: %w", err)
	}

	return &oidcController{
		respWriter:         respWriter,
		signingKeyManager:  signingKeyManager,
		frontendHost:       parsedURL.Host,
		frontendScheme:     parsedURL.Scheme,
		userSessionManager: userSessionManager,
		oidcConfig:         oidcConfig,
		clientID:           clientID,
	}, nil
}

// RegisterRoutes adds health routes to the router
func (c *oidcController) RegisterRoutes(router chi.Router) {
	router.Get("/.well-known/openid-configuration", c.GetOpenIDConfig)

	router.Get("/oauth/authorize", c.Authorize)
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
	redirectURI := r.Form.Get("redirect_uri")
	code := r.Form.Get("code")
	codeVerifier := r.Form.Get("code_verifier")

	if clientID != c.clientID {
		c.respWriter.RespondWithError(ctx, w, errors.New("invalid client id", errors.WithErrorCode(errors.EInvalid)))
		return
	}

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
