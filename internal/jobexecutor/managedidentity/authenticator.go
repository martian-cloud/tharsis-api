package managedidentity

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Authenticator provides the cloud provider specific authentication logic
type Authenticator interface {
	Authenticate(ctx context.Context, managedIdentity *types.ManagedIdentity, creds []byte) (map[string]string, error)
	Close(ctx context.Context) error
}
