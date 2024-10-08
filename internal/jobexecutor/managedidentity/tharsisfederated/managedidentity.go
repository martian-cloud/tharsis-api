// Package tharsisfederated package
package tharsisfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/tharsisfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Authenticator supports Tharsis Federation
type Authenticator struct {
	// No fields.
}

// New creates a new instance of Authenticator
func New() *Authenticator {
	return &Authenticator{}
}

// Close cleans up any open resources
func (a *Authenticator) Close(context.Context) error {
	// Nothing needs to be done, but this method is required by the interface.
	return nil
}

// Authenticate configures the environment with the identity information used by the Tharsis terraform provider
func (a *Authenticator) Authenticate(
	ctx context.Context,
	managedIdentities []types.ManagedIdentity,
	credsRetriever func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error),
) (map[string]string, error) {
	if len(managedIdentities) != 1 {
		return nil, fmt.Errorf("expected exactly one tharsis federated managed identity, got %d", len(managedIdentities))
	}

	managedIdentity := managedIdentities[0]

	decodedData, err := base64.StdEncoding.DecodeString(string(managedIdentity.Data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode managed identity payload %v", err)
	}

	federatedData := tharsisfederated.Data{}
	if err = json.Unmarshal(decodedData, &federatedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed identity payload %v", err)
	}

	creds, err := credsRetriever(ctx, &managedIdentity)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials %v", err)
	}

	return map[string]string{
		"THARSIS_SERVICE_ACCOUNT_PATH":  federatedData.ServiceAccountPath,
		"THARSIS_SERVICE_ACCOUNT_TOKEN": string(creds),
	}, nil
}
