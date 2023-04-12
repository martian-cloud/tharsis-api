// Package jws package
package jws

//go:generate mockery --name Provider --inpackage --case underscore

import (
	"context"
)

// Provider is used to sign and verify JWT payloads
type Provider interface {
	Sign(ctx context.Context, token []byte) ([]byte, error)
	Verify(ctx context.Context, token []byte) error
	GetKeySet(ctx context.Context) ([]byte, error)
}
