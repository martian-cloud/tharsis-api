package controllers

//go:generate go tool mockery --name OIDCLoginClient --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/units"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// maxOIDCTokenResponseBytes caps how much of an identity provider's token response is read.
	maxOIDCTokenResponseBytes = 1 * units.MiB
	// maxOIDCErrorBodyBytes caps how much of a token endpoint error that isn't an OAuth error is reported.
	maxOIDCErrorBodyBytes = 512
)

// OIDCLoginClient signs a user in with an external identity provider using the OAuth authorization
// code flow with PKCE, with the API as the OAuth client. It is used to proxy CLI logins through the API.
type OIDCLoginClient interface {
	// Issuer returns the identity provider's issuer identifier, exactly as its discovery document states it.
	Issuer() string
	// IssuerParameterSupported returns whether the identity provider returns the iss parameter in its
	// authorization responses (RFC 9207), in which case a response without it must be rejected.
	IssuerParameterSupported() bool
	// AuthCodeURL returns the identity provider URL that starts the login.
	AuthCodeURL(state, nonce, codeChallenge string) string
	// ExchangeCode redeems an authorization code and returns the identity provider's ID token.
	ExchangeCode(ctx context.Context, code, codeVerifier string) (string, error)
}

type oidcLoginClient struct {
	httpClient   *http.Client
	issuer       string
	issParameter bool
	authEndpoint string
	tokenURL     string
	clientID     string
	clientSecret string
	redirectURI  string
	scope        string
}

// NewOIDCLoginClient creates an OIDCLoginClient for the identity provider. redirectURI is where the
// identity provider returns the user to, and must be registered with it.
func NewOIDCLoginClient(
	ctx context.Context,
	idp *config.IdpConfig,
	redirectURI string,
	configFetcher auth.OpenIDConfigFetcher,
) (OIDCLoginClient, error) {
	oidcConfig, err := configFetcher.GetOpenIDConfig(ctx, idp.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC config for issuer %s: %w", idp.IssuerURL, err)
	}

	if oidcConfig.AuthEndpoint == "" || oidcConfig.TokenEndpoint == "" {
		return nil, fmt.Errorf("OIDC config for issuer %s is missing the authorization or token endpoint", idp.IssuerURL)
	}

	// The openid scope is what makes the identity provider return an ID token.
	scopes := strings.Fields(idp.Scope)
	if !slices.Contains(scopes, "openid") {
		scopes = append([]string{"openid"}, scopes...)
	}

	return &oidcLoginClient{
		httpClient:   configFetcher.HTTPClient(),
		issuer:       oidcConfig.Issuer,
		issParameter: oidcConfig.AuthorizationResponseIssParameterSupported,
		authEndpoint: oidcConfig.AuthEndpoint,
		tokenURL:     oidcConfig.TokenEndpoint,
		clientID:     idp.ClientID,
		clientSecret: idp.ClientSecret,
		redirectURI:  redirectURI,
		scope:        strings.Join(scopes, " "),
	}, nil
}

func (o *oidcLoginClient) Issuer() string {
	return o.issuer
}

func (o *oidcLoginClient) IssuerParameterSupported() bool {
	return o.issParameter
}

func (o *oidcLoginClient) AuthCodeURL(state, nonce, codeChallenge string) string {
	params := url.Values{
		"client_id":             {o.clientID},
		"redirect_uri":          {o.redirectURI},
		"response_type":         {"code"},
		"scope":                 {o.scope},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	separator := "?"
	if strings.Contains(o.authEndpoint, "?") {
		separator = "&"
	}

	return o.authEndpoint + separator + params.Encode()
}

func (o *oidcLoginClient) ExchangeCode(ctx context.Context, code, codeVerifier string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"redirect_uri":  {o.redirectURI},
	}
	if o.clientSecret == "" {
		// Public clients identify themselves in the request body.
		form.Set("client_id", o.clientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", errors.Wrap(err, "failed to build identity provider token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if o.clientSecret != "" {
		// client_secret_basic, the OIDC default: the credentials are form-encoded before being base64 encoded.
		req.SetBasicAuth(url.QueryEscape(o.clientID), url.QueryEscape(o.clientSecret))
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to request token from identity provider")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOIDCTokenResponseBytes))
	if err != nil {
		return "", errors.Wrap(err, "failed to read identity provider token response")
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != "" {
			return "", errors.New("identity provider rejected the authorization code: %s: %s",
				errorResp.Error, errorResp.ErrorDescription, errors.WithErrorCode(errors.EUnauthorized))
		}
		// Error responses aren't always OAuth errors (e.g. a proxy error page), so the body is reported as text.
		return "", errors.New("identity provider token endpoint returned %s: %s", resp.Status, errorBodyText(body),
			errors.WithErrorCode(errors.EUnauthorized))
	}

	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", errors.Wrap(err, "failed to parse identity provider token response", errors.WithErrorCode(errors.EUnauthorized))
	}

	if tokenResp.IDToken == "" {
		return "", errors.New("identity provider did not return an ID token; ensure the openid scope is allowed", errors.WithErrorCode(errors.EUnauthorized))
	}

	return tokenResp.IDToken, nil
}

// errorBodyText returns the start of a response body as a single line of text.
func errorBodyText(body []byte) string {
	text := strings.Join(strings.Fields(strings.ToValidUTF8(string(body), "")), " ")
	if len(text) > maxOIDCErrorBodyBytes {
		text = strings.ToValidUTF8(text[:maxOIDCErrorBodyBytes], "") + "..."
	}
	if text == "" {
		return "(empty body)"
	}
	return text
}
