// Package managedidentity package
package managedidentity

//go:generate go tool mockery --name Authenticator --inpackage --case underscore

import (
	"context"

	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// Authenticator provides the cloud provider specific authentication logic
type Authenticator interface {
	Authenticate(
		ctx context.Context,
		managedIdentities []*pb.ManagedIdentity,
		credsRetriever func(ctx context.Context, managedIdentity *pb.ManagedIdentity) ([]byte, error),
	) (*AuthenticateResponse, error)
	Close(ctx context.Context) error
}

// AuthenticateResponse contains the environment variables and host credential file mappings
type AuthenticateResponse struct {
	Env                       map[string]string
	HostCredentialFileMapping map[string]string
}
