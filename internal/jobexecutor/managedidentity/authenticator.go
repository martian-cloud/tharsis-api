// Package managedidentity package
package managedidentity

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Authenticator provides the cloud provider specific authentication logic
type Authenticator interface {
	Authenticate(
		ctx context.Context,
		managedIdentities []types.ManagedIdentity,
		credsRetriever func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error),
	) (map[string]string, error)
	Close(ctx context.Context) error
}
