// Package azurefederated package
package azurefederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Authenticator supports Azure OIDC Federation
type Authenticator struct {
	dir string
}

// New creates a new instance of Authenticator
func New() (*Authenticator, error) {
	// Create a temporary directory to avoid a problem with Terraform choking
	// if there is a pre-existing file in the workspace directory.
	tempDir, err := os.MkdirTemp("", "authenticator-temp-*")
	if err != nil {
		return nil, err
	}
	return &Authenticator{
		dir: tempDir,
	}, nil
}

// Close cleans up any open resources
func (a *Authenticator) Close(_ context.Context) error {
	return os.RemoveAll(a.dir)
}

// Authenticate configures the environment with the identity information used by the Azure terraform provider
func (a *Authenticator) Authenticate(
	ctx context.Context,
	managedIdentities []types.ManagedIdentity,
	credsRetriever func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error),
) (*managedidentity.AuthenticateResponse, error) {
	if len(managedIdentities) != 1 {
		return nil, fmt.Errorf("expected exactly one azure federated managed identity, got %d", len(managedIdentities))
	}

	managedIdentity := managedIdentities[0]

	decodedData, err := base64.StdEncoding.DecodeString(string(managedIdentity.Data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode managed identity payload %v", err)
	}

	federatedData := azurefederated.Data{}
	if err = json.Unmarshal(decodedData, &federatedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed identity payload %v", err)
	}

	creds, err := credsRetriever(ctx, &managedIdentity)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve managed identity token %v", err)
	}

	filePath := filepath.Join(a.dir, fmt.Sprintf("%s-token", managedIdentity.Metadata.ID))
	if err := os.WriteFile(filePath, creds, 0o600); err != nil {
		return nil, fmt.Errorf("failed to write managed identity token to disk %v", err)
	}

	response := managedidentity.AuthenticateResponse{
		Env: map[string]string{
			"ARM_TENANT_ID":              federatedData.TenantID,
			"ARM_CLIENT_ID":              federatedData.ClientID,
			"ARM_USE_OIDC":               "true",
			"ARM_OIDC_TOKEN":             string(creds),
			"AZURE_CLIENT_ID":            federatedData.ClientID,
			"AZURE_TENANT_ID":            federatedData.TenantID,
			"AZURE_FEDERATED_TOKEN_FILE": filePath,
		},
	}

	return &response, nil
}
