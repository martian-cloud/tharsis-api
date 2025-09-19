// Package jws package
package jws

//go:generate go tool mockery --name Provider --inpackage --case underscore

import (
	"context"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

// CreateKeyResponse contains the data returned when creating a new signing key
type CreateKeyResponse struct {
	KeyData   []byte // KeyData is optional and plugin specific data about the created key
	PublicKey jwk.Key
}

// Provider is used to sign and verify JWT payloads
type Provider interface {
	Create(ctx context.Context, keyID string) (*CreateKeyResponse, error)
	Delete(ctx context.Context, keyID string, keyData []byte) error
	Sign(ctx context.Context, token []byte, keyID string, keyData []byte, publicKeyID string) ([]byte, error)
	SupportsKeyRotation() bool
}
