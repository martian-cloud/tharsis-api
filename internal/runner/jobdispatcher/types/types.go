// Package types defines types used by the job dispatchers
package types

import "context"

// TokenGetterFunc is a function type that retrieves a runner ID token for authentication.
type TokenGetterFunc func(ctx context.Context) (string, error)
