// Package managedidentity package
package managedidentity

//go:generate go tool mockery --name Authenticator --inpackage --case underscore

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
	) (*AuthenticateResponse, error)
	Close(ctx context.Context) error
}

// AuthenticateResponse contains the environment variables and host credential file mappings
type AuthenticateResponse struct {
	Env                       map[string]string
	HostCredentialFileMapping map[string]string
}
