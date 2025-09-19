// Package memory package
package memory

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	jwsplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
)

const (
	keyBits = 2048
)

// InMemoryJWSProvider uses a secret key stored in memory to sign JWT tokens
// using a SHA-256 HMAC signature algorithm
type InMemoryJWSProvider struct{}

// New creates an InMemoryJWSProvider
func New(_ map[string]string) (*InMemoryJWSProvider, error) {
	return &InMemoryJWSProvider{}, nil
}

// SupportsKeyRotation indicates if the plugin supports key rotation
func (im *InMemoryJWSProvider) SupportsKeyRotation() bool {
	return true
}

// Create creates a new signing key
func (im *InMemoryJWSProvider) Create(_ context.Context, _ string) (*jwsplugin.CreateKeyResponse, error) {
	// Generate random key if one is not provided
	privKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, err
	}

	privJWK, err := jwk.FromRaw(privKey)
	if err != nil {
		return nil, err
	}

	if err = jwk.AssignKeyID(privJWK); err != nil {
		return nil, err
	}

	pubJWK, err := jwk.FromRaw(privKey.PublicKey)
	if err != nil {
		return nil, err
	}

	if err = jwk.AssignKeyID(pubJWK); err != nil {
		return nil, err
	}
	if err = pubJWK.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		return nil, err
	}
	if err = pubJWK.Set(jwk.KeyUsageKey, jwk.ForSignature); err != nil {
		return nil, err
	}

	privKeyBytes, err := json.Marshal(privJWK)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWK key %v", err)
	}

	return &jwsplugin.CreateKeyResponse{
		KeyData:   privKeyBytes,
		PublicKey: pubJWK,
	}, nil
}

// Delete deletes a signing key (no-op for in-memory plugin)
func (im *InMemoryJWSProvider) Delete(_ context.Context, _ string, _ []byte) error {
	// Nothing to do since keys are not stored externally
	return nil
}

// Sign signs a JWT payload
func (im *InMemoryJWSProvider) Sign(_ context.Context, token []byte, _ string, keyData []byte, _ string) ([]byte, error) {
	key, err := jwk.ParseKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWK key %v", err)
	}

	hdrs := jws.NewHeaders()
	if err = hdrs.Set(jws.TypeKey, "JWT"); err != nil {
		return nil, err
	}

	signed, err := jws.Sign(token, jws.WithKey(jwa.RS256, key, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		return nil, err
	}

	return signed, nil
}

func parseBase64Key(b64Key string) (*rsa.PrivateKey, error) {
	pemBuf, err := base64.URLEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemBuf)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
