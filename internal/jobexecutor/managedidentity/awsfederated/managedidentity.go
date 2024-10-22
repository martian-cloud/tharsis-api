// Package awsfederated package
package awsfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	// awsProfileTemplate is the template for the AWS profile
	awsProfileTemplate = `[profile %s]
role_arn=%s
web_identity_token_file=%s

`
)

// Authenticator supports AWS OIDC Federation
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
func (a *Authenticator) Close(context.Context) error {
	return os.RemoveAll(a.dir)
}

// Authenticate configures the environment with the identity information used by the AWS terraform provider
func (a *Authenticator) Authenticate(
	ctx context.Context,
	managedIdentities []types.ManagedIdentity,
	credsRetriever func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error),
) (*managedidentity.AuthenticateResponse, error) {
	envs := map[string]string{}

	configFile, tErr := os.CreateTemp(a.dir, "aws-profiles-*")
	if tErr != nil {
		return nil, fmt.Errorf("failed to create aws profiles file %v", tErr)
	}
	defer configFile.Close()

	envs["AWS_CONFIG_FILE"] = configFile.Name()

	for ix := range managedIdentities {
		managedIdentity := managedIdentities[ix]
		decodedData, err := base64.StdEncoding.DecodeString(string(managedIdentity.Data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode managed identity payload %v", err)
		}

		federatedData := awsfederated.Data{}
		if err = json.Unmarshal(decodedData, &federatedData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal managed identity payload %v", err)
		}

		creds, err := credsRetriever(ctx, &managedIdentity)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve managed identity token %v", err)
		}

		tokenFilepath := filepath.Join(a.dir, fmt.Sprintf("%s-token", managedIdentity.Metadata.ID))
		if err = os.WriteFile(tokenFilepath, creds, 0o600); err != nil {
			return nil, fmt.Errorf("failed to write managed identity token to disk %v", err)
		}

		// For backward compatibility when using a single managed identity.
		if len(managedIdentities) == 1 {
			envs["AWS_ROLE_ARN"] = federatedData.Role
			envs["AWS_WEB_IDENTITY_TOKEN_FILE"] = tokenFilepath
		}

		// Use the managed identity resource path as the profile name
		if _, err = fmt.Fprintf(configFile, awsProfileTemplate, managedIdentity.ResourcePath, federatedData.Role, tokenFilepath); err != nil {
			return nil, fmt.Errorf("failed to write AWS profile %v", err)
		}
	}

	response := managedidentity.AuthenticateResponse{Env: envs}

	return &response, nil
}
