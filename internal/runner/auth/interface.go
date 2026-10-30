// Package auth declares the OIDC token source the runner agent exchanges for a Tharsis service
// account token, with one implementation per source in a sub-package.
//
// Choosing between the implementations is the caller's job: it depends on the consuming binary's
// configuration, so it lives with that binary rather than here. This mirrors how
// internal/runner/jobdispatcher is laid out, where the implementations sit in sub-packages and the
// selection happens in the consumer.
package auth

//go:generate go tool mockery --name OIDCTokenSource --inpackage --case underscore

import (
	"context"
)

// OIDCTokenSource is an interface for returning the OIDC token to exchange for a Tharsis service
// account token.
//
// Implementations are resolved once at startup but called on every renewal, so a source is
// expected to return the current token rather than a value captured when it was constructed. That
// is what allows Kubernetes to rotate a projected token in place, and a credential helper to mint
// a fresh one.
type OIDCTokenSource interface {
	GetToken(context.Context) (string, error)
}
