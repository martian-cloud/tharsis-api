// Package jwsprovider package
package jwsprovider

//go:generate mockery --name JWSProvider --inpackage --case underscore

import (
	"context"
)

// JWSProvider is used to sign and verify JWT payloads
type JWSProvider interface {
	Sign(ctx context.Context, token []byte) ([]byte, error)
	Verify(ctx context.Context, token []byte) error
	GetKeySet(ctx context.Context) ([]byte, error)
}
