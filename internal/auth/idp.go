package auth

//go:generate go tool mockery --name IdentityProvider --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
)

const privateClaimPrefix = "tharsis_"

// TokenInput provides options for creating a new service account token
type TokenInput struct {
	Expiration *time.Time
	Claims     map[string]string
	Subject    string
	JwtID      string
	Audience   string
}

// VerifyTokenOutput is the response from verifying a token
type VerifyTokenOutput struct {
	Token         jwt.Token
	PrivateClaims map[string]string
}

// IdentityProvider is an interface for generating and verifying JWT tokens
type IdentityProvider interface {
	// GenerateToken creates a new JWT token
	GenerateToken(ctx context.Context, input *TokenInput) ([]byte, error)
	// VerifyToken verifies that the token is valid
	VerifyToken(ctx context.Context, token string) (*VerifyTokenOutput, error)
}

type identityProvider struct {
	jwsPlugin jws.Provider
	issuerURL string
}

// NewIdentityProvider initializes the IdentityProvider type
func NewIdentityProvider(jwsPlugin jws.Provider, issuerURL string) IdentityProvider {
	return &identityProvider{
		jwsPlugin: jwsPlugin,
		issuerURL: issuerURL,
	}
}

// GenerateToken creates a new service account token
func (s *identityProvider) GenerateToken(ctx context.Context, input *TokenInput) ([]byte, error) {
	currentTimestamp := time.Now().Unix()

	token := jwt.New()

	if input.Expiration != nil {
		if err := token.Set(jwt.ExpirationKey, input.Expiration.Unix()); err != nil {
			return nil, err
		}
	}
	if err := token.Set(jwt.NotBeforeKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuedAtKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.IssuerKey, s.issuerURL); err != nil {
		return nil, err
	}
	if err := token.Set(jwt.SubjectKey, input.Subject); err != nil {
		return nil, err
	}

	aud := input.Audience
	if aud == "" {
		aud = "tharsis"
	}
	if err := token.Set(jwt.AudienceKey, aud); err != nil {
		return nil, err
	}
	if input.JwtID != "" {
		if err := token.Set(jwt.JwtIDKey, input.JwtID); err != nil {
			return nil, err
		}
	}

	for k, v := range input.Claims {
		if err := token.Set(fmt.Sprintf("%s%s", privateClaimPrefix, k), v); err != nil {
			return nil, nil
		}
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		return nil, err
	}

	// Create signed token
	return s.jwsPlugin.Sign(ctx, payload)
}

// VerifyToken verifies that the token is a valid service account token
func (s *identityProvider) VerifyToken(ctx context.Context, token string) (*VerifyTokenOutput, error) {
	tokenBytes := []byte(token)

	// Verify token signature
	if err := s.jwsPlugin.Verify(ctx, tokenBytes); err != nil {
		return nil, err
	}

	// Parse and validate jwt
	decodedToken, err := jwt.Parse(tokenBytes, jwt.WithVerify(false), jwt.WithValidate(true), jwt.WithIssuer(s.issuerURL))
	if err != nil {
		return nil, fmt.Errorf("failed to decode token %w", err)
	}

	return &VerifyTokenOutput{
		Token:         decodedToken,
		PrivateClaims: getPrivateClaims(decodedToken),
	}, nil
}

// GetPrivateClaims returns a map of the token's private claims
func getPrivateClaims(token jwt.Token) map[string]string {
	claimsMap := make(map[string]string)

	privClaims := token.PrivateClaims()
	for k, v := range privClaims {
		if strings.HasPrefix(k, privateClaimPrefix) {
			claimsMap[strings.TrimPrefix(k, privateClaimPrefix)] = v.(string)
		}
	}

	return claimsMap
}

func getPrivateClaim(claim string, token jwt.Token) (string, bool) {
	if claim, ok := token.Get(privateClaimPrefix + claim); ok {
		if val, ok := claim.(string); ok {
			return val, true
		}
	}
	return "", false
}
