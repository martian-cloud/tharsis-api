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
)

// InMemoryJWSProvider uses a secret key stored in memory to sign JWT tokens
// using a SHA-256 HMAC signature algorithm
type InMemoryJWSProvider struct {
	privKey *rsa.PrivateKey
	pubKey  jwk.Key
	keySet  []byte
}

// New creates an InMemoryJWSProvider
func New(pluginData map[string]string) (*InMemoryJWSProvider, error) {
	var privKey *rsa.PrivateKey
	var err error
	b64Key, ok := pluginData["signing_key_b64"]

	if ok {
		privKey, err = parseBase64Key(b64Key)
		if err != nil {
			return nil, err
		}
	} else {
		// Generate random key if one is not provided
		privKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
	}

	pubKey, err := jwk.FromRaw(privKey.PublicKey)
	if err != nil {
		return nil, err
	}

	if err = jwk.AssignKeyID(pubKey); err != nil {
		return nil, err
	}
	if err = pubKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		return nil, err
	}
	if err = pubKey.Set(jwk.KeyUsageKey, jwk.ForSignature); err != nil {
		return nil, err
	}

	keySet, err := buildKeySet(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to build JWK key set %v", err)
	}

	return &InMemoryJWSProvider{
		privKey: privKey,
		pubKey:  pubKey,
		keySet:  keySet,
	}, nil
}

// Sign signs a JWT payload
func (im *InMemoryJWSProvider) Sign(_ context.Context, token []byte) ([]byte, error) {
	key, err := jwk.FromRaw(im.privKey)
	if err != nil {
		return nil, err
	}

	if err = jwk.AssignKeyID(key); err != nil {
		return nil, err
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

// GetKeySet returns the JWK key set in JSON format
func (im *InMemoryJWSProvider) GetKeySet(_ context.Context) ([]byte, error) {
	return im.keySet, nil
}

// Verify will return an error if the JWT does not have a valid signature
func (im *InMemoryJWSProvider) Verify(_ context.Context, token []byte) error {
	keySet := jwk.NewSet()
	if err := keySet.AddKey(jwk.Key(im.pubKey)); err != nil {
		return err
	}

	if _, err := jws.Verify(token, jws.WithKeySet(keySet)); err != nil {
		return err
	}

	return nil
}

func buildKeySet(pubKey jwk.Key) ([]byte, error) {
	keySet := jwk.NewSet()
	if err := keySet.AddKey(pubKey); err != nil {
		return nil, err
	}
	buf, err := json.Marshal(keySet)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func parseBase64Key(b64Key string) (*rsa.PrivateKey, error) {
	pemBuf, err := base64.URLEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemBuf)
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
