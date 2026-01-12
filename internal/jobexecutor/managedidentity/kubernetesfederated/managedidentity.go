// Package kubernetesfederated provides Kubernetes federated managed identity authentication.
package kubernetesfederated

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Authenticator supports Kubernetes OIDC authentication
type Authenticator struct{}

// New creates a new instance of Authenticator
func New() (*Authenticator, error) {
	return &Authenticator{}, nil
}

// Close cleans up any open resources (no-op for Kubernetes authenticator)
func (a *Authenticator) Close(context.Context) error {
	// Note: Nothing to close any resources, but we need to  do a dummy implementation
	return nil
}

// Authenticate configures the environment with Kubernetes authentication information
func (a *Authenticator) Authenticate(
	ctx context.Context,
	managedIdentities []types.ManagedIdentity,
	credsRetriever func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error),
) (*managedidentity.AuthenticateResponse, error) {
	if len(managedIdentities) != 1 {
		return nil, fmt.Errorf("expected exactly one kubernetes managed identity, got %d", len(managedIdentities))
	}

	managedIdentity := managedIdentities[0]
	// Get JWT token from credentials retriever (service layer)
	token, err := credsRetriever(ctx, &managedIdentity)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve managed identity token: %w", err)
	}

	return &managedidentity.AuthenticateResponse{
		Env: map[string]string{
			"KUBE_TOKEN": string(token),
		},
	}, nil
}
