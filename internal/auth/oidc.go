package auth

//go:generate go tool mockery --name OpenIDConfigFetcher --inpackage --case underscore
//go:generate go tool mockery --name OIDCTokenVerifier --inpackage --case underscore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// retryWaitMinimum is the minimum amount of seconds retryablehttp
	// client will wait before attempting to make another connection.
	// Default min is 2 seconds.
	retryWaitMinimum = time.Second * 5

	defaultKeyAlgorithm         = jwa.RS256
	jwtRefreshIntervalInMinutes = 60
)

// OIDCConfiguration contains the OIDC information for an identity provider
type OIDCConfiguration struct {
	Issuer        string `json:"issuer"`
	JwksURI       string `json:"jwks_uri"`
	TokenEndpoint string `json:"token_endpoint"`
	AuthEndpoint  string `json:"authorization_endpoint"`
}

// OpenIDConfigFetcher is an interface for fetching OIDC configuration
type OpenIDConfigFetcher interface {
	// GetOpenIDConfig returns the OIDC configuration for the given issuer
	GetOpenIDConfig(ctx context.Context, issuer string) (*OIDCConfiguration, error)
}

type leveledLoggerAdapter struct {
	logger logger.Logger
}

func (l *leveledLoggerAdapter) Error(msg string, keysAndValues ...interface{}) {
	l.logger.With(keysAndValues...).Error(msg)
}

func (l *leveledLoggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	l.logger.With(keysAndValues...).Info(msg)
}

func (l *leveledLoggerAdapter) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.With(keysAndValues...).Debug(msg)
}

func (l *leveledLoggerAdapter) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.With(keysAndValues...).Info(msg)
}

// OpenIDConfigFetcher implements functions to fetch
// OpenID configuration from an issuer.
type openIDConfigFetcher struct {
	client *retryablehttp.Client
}

// NewOpenIDConfigFetcher returns a new NewOpenIDConfigFetcher
func NewOpenIDConfigFetcher(logger logger.Logger) OpenIDConfigFetcher {
	// Retryablehttp client defaults to 4 retries.
	client := retryablehttp.NewClient()
	client.RetryWaitMin = retryWaitMinimum
	client.Logger = &leveledLoggerAdapter{logger: logger}

	return &openIDConfigFetcher{client: client}
}

func (o *openIDConfigFetcher) GetOpenIDConfig(ctx context.Context, issuer string) (*OIDCConfiguration, error) {
	normalizedIssuer := strings.TrimSuffix(issuer, "/")
	wellKnownURI := normalizedIssuer + "/.well-known/openid-configuration"

	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", wellKnownURI, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build OIDC request: %v", err)
	}

	// Use retryablehttp client so we can retry incase request fails.
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request OIDC discovery document: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for OIDC discovery document: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received invalid response from OIDC discovery endpoint %s: %s", resp.Status, body)
	}

	var cfg OIDCConfiguration
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("unable to parse OIDC discovery document: %v", err)
	}

	if strings.TrimSuffix(cfg.Issuer, "/") != normalizedIssuer {
		return nil, fmt.Errorf("OIDC issuer does not match the issuer returned by the OIDC discovery document, expected %q got %q", issuer, cfg.Issuer)
	}

	return &cfg, nil
}

// OIDCTokenVerifier is an interface for verifying OIDC tokens
type OIDCTokenVerifier interface {
	// VerifyToken verifies the OIDC token and returns the decoded token
	// If the token is not valid, it returns an error
	VerifyToken(ctx context.Context, token string, validationOptions []jwt.ValidateOption) (jwt.Token, error)
}

type oidcTokenVerifier struct {
	cache             *jwk.Cache
	oidcConfigMap     map[string]*OIDCConfiguration
	oidcConfigFetcher OpenIDConfigFetcher
	issuerMap         map[string]struct{}
	mu                sync.RWMutex
}

// NewOIDCTokenVerifier creates a new OIDCTokenVerifier instance
func NewOIDCTokenVerifier(ctx context.Context, issuers []string, oidcConfigFetcher OpenIDConfigFetcher, enableCache bool) OIDCTokenVerifier {
	issuerMap := map[string]struct{}{}
	for _, issuer := range issuers {
		issuerMap[NormalizeOIDCIssuer(issuer)] = struct{}{}
	}

	var cache *jwk.Cache
	if enableCache {
		cache = jwk.NewCache(ctx)
	}

	return &oidcTokenVerifier{
		cache:             cache,
		oidcConfigMap:     map[string]*OIDCConfiguration{},
		oidcConfigFetcher: oidcConfigFetcher,
		issuerMap:         issuerMap,
	}
}

func (o *oidcTokenVerifier) VerifyToken(ctx context.Context, token string, validationOptions []jwt.ValidateOption) (jwt.Token, error) {
	tokenBytes := []byte(token)

	// Parse jwt
	decodedToken, err := jwt.Parse(tokenBytes, jwt.WithVerify(false))
	if err != nil {
		return nil, fmt.Errorf("failed to decode token %w", err)
	}

	issuer := decodedToken.Issuer()
	normalizedIssuer := NormalizeOIDCIssuer(issuer)

	if _, ok := o.issuerMap[normalizedIssuer]; !ok {
		return nil, fmt.Errorf("invalid issuer %s", issuer)
	}

	oidcCfg, err := o.loadOIDCConfig(ctx, normalizedIssuer)
	if err != nil {
		return nil, err
	}

	keySet, err := o.getKeySet(ctx, tokenBytes, oidcCfg.JwksURI)
	if err != nil {
		return nil, errors.New("failed to load key set for issuer %s", issuer)
	}

	options := []jwt.ParseOption{
		jwt.WithVerify(true),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	}
	for _, o := range validationOptions {
		options = append(options, o)
	}

	// Parse and Verify token
	if _, err = jwt.Parse(tokenBytes, options...); err != nil {
		return nil, errors.Wrap(err, "failed to verify token", errors.WithErrorCode(errors.EUnauthorized))
	}

	return decodedToken, nil
}

func (o *oidcTokenVerifier) loadOIDCConfig(ctx context.Context, issuer string) (*OIDCConfiguration, error) {
	if cfg, ok := o.getOIDCConfigFromCache(issuer); ok {
		return cfg, nil
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	cfg, err := o.oidcConfigFetcher.GetOpenIDConfig(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC config for issuer url %s: %w", issuer, err)
	}

	if o.cache != nil {
		err := o.cache.Register(cfg.JwksURI, jwk.WithRefreshInterval(jwtRefreshIntervalInMinutes*time.Minute))
		if err != nil {
			return nil, fmt.Errorf("failed to register OIDC config for issuer url %s: %w", issuer, err)
		}
	}

	o.oidcConfigMap[issuer] = cfg

	return cfg, nil
}

func (o *oidcTokenVerifier) getOIDCConfigFromCache(issuer string) (*OIDCConfiguration, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	cfg, ok := o.oidcConfigMap[issuer]
	return cfg, ok
}

func (o *oidcTokenVerifier) getKeySet(ctx context.Context, token []byte, jwksURI string) (jwk.Set, error) {
	if o.cache != nil {
		key, err := o.getKey(ctx, token, jwksURI)
		if err != nil {
			return nil, err
		}

		alg := key.Algorithm()
		if alg.String() == "" {
			alg = defaultKeyAlgorithm
		}

		if err = key.Set(jwk.AlgorithmKey, alg); err != nil {
			return nil, err
		}

		keySet := jwk.NewSet()
		if err = keySet.AddKey(key); err != nil {
			return nil, err
		}

		return keySet, nil
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// Get issuer JWK response
	keySet, err := jwk.Fetch(fetchCtx, jwksURI)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query JWK URL %s", jwksURI)
	}

	// Set default key to RS256 if it's not specified in JWK set
	iter := keySet.Keys(ctx)
	for iter.Next(ctx) {
		key := iter.Pair().Value.(jwk.Key)

		alg := key.Algorithm()
		if alg.String() == "" {
			alg = defaultKeyAlgorithm
		}
		if err = key.Set(jwk.AlgorithmKey, alg); err != nil {
			return nil, err
		}
	}

	return keySet, nil
}

func (o *oidcTokenVerifier) getKey(ctx context.Context, token []byte, jwksURI string) (jwk.Key, error) {
	// Parse token headers
	msg, err := jws.Parse(token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token headers %w", err)
	}

	signatures := msg.Signatures()
	if len(signatures) < 1 {
		return nil, errors.New("token is missing signature")
	}

	keyset, err := o.cache.Get(ctx, jwksURI)
	if err != nil {
		return nil, errors.New("failed to load key set for identity provider")
	}

	kid := signatures[0].ProtectedHeaders().KeyID()

	key, found := keyset.LookupKeyID(kid)
	if !found {
		// Attempt to refresh the keyset for the IDP because the keys may have been updated
		keyset, err := o.cache.Refresh(ctx, jwksURI)
		if err != nil {
			return nil, errors.New("failed to load key set for identity provider")
		}

		key, found = keyset.LookupKeyID(kid)
		if !found {
			return nil, errors.New("failed to load key set for identity provider: kid %s not found ", kid)
		}

		return key, nil
	}

	return key, nil
}

// NormalizeOIDCIssuer normalizes the OIDC issuer URL by adding "https://" prefix if not present and removing the trailing slash
func NormalizeOIDCIssuer(issuer string) string {
	normalizedIssuer := issuer
	if !strings.HasPrefix(issuer, "http://") && !strings.HasPrefix(issuer, "https://") {
		normalizedIssuer = fmt.Sprintf("https://%s", issuer)
	}
	return strings.TrimSuffix(normalizedIssuer, "/")
}
