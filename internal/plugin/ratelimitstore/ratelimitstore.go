// Package ratelimitstore package
package ratelimitstore

import "context"

// Store is an interface for rate limiter backends
type Store interface {
	TakeMany(ctx context.Context, key string, takeAmount uint64) (tokens, remaining, reset uint64, ok bool, err error)
}
