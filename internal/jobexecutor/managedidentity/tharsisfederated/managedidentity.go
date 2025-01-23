// Package tharsisfederated package
package tharsisfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/tharsisfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	serviceAccountTokenFilename       = "service-account-token"
	tharsisServiceAccountPathEnvName  = "THARSIS_SERVICE_ACCOUNT_PATH"
	tharsisServiceAccountTokenEnvName = "THARSIS_SERVICE_ACCOUNT_TOKEN"
	tokenFilePermissions              = os.FileMode(0640)
)

// Authenticator supports Tharsis Federation
type Authenticator struct {
	client                    jobclient.Client
	jobLogger                 joblogger.Logger
	refreshTokenEarlyDuration time.Duration
	tokenDir                  string
	apiEndpoint               string
	discoveryProtocolHost     *string
}

// New creates a new instance of Authenticator
func New(client jobclient.Client, jobLogger joblogger.Logger, apiEndpoint string, discoveryProtocolHost *string) (*Authenticator, error) {
	tokenDir, err := os.MkdirTemp("", "tharsis-federated-token-*")
	if err != nil {
		return nil, err
	}

	return newAuthenticator(client, jobLogger, 1*time.Minute, tokenDir, apiEndpoint, discoveryProtocolHost)
}

func newAuthenticator(
	client jobclient.Client,
	jobLogger joblogger.Logger,
	refreshTokenEarlyDuration time.Duration,
	tokenDir string,
	apiEndpoint string,
	discoveryProtocolHost *string,
) (*Authenticator, error) {
	return &Authenticator{
		client:                    client,
		jobLogger:                 jobLogger,
		refreshTokenEarlyDuration: refreshTokenEarlyDuration,
		tokenDir:                  tokenDir,
		apiEndpoint:               apiEndpoint,
		discoveryProtocolHost:     discoveryProtocolHost,
	}, nil
}

// Close cleans up any open resources
func (a *Authenticator) Close(_ context.Context) error {
	return os.RemoveAll(a.tokenDir)
}

// Authenticate configures the environment with the identity information used by the Tharsis terraform provider
func (a *Authenticator) Authenticate(
	ctx context.Context,
	managedIdentities []types.ManagedIdentity,
	credsRetriever func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error),
) (*managedidentity.AuthenticateResponse, error) {
	if len(managedIdentities) != 1 {
		return nil, fmt.Errorf("expected exactly one tharsis federated managed identity, got %d", len(managedIdentities))
	}

	managedIdentity := managedIdentities[0]

	federatedData, err := retrieveManagedIdentityPayload(managedIdentity.Data)
	if err != nil {
		return nil, err
	}

	creds, err := credsRetriever(ctx, &managedIdentity)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials %v", err)
	}

	convertedCreds := string(creds)

	response := managedidentity.AuthenticateResponse{
		Env: map[string]string{
			tharsisServiceAccountPathEnvName:  federatedData.ServiceAccountPath,
			tharsisServiceAccountTokenEnvName: convertedCreds,
		},
		HostCredentialFileMapping: map[string]string{},
	}

	if !federatedData.UseServiceAccountForTerraformCLI {
		return &response, nil
	}

	err = a.setupServiceAccountTokenWithRefresh(ctx, federatedData.ServiceAccountPath, convertedCreds)
	if err != nil {
		return nil, err
	}

	err = a.setupHostCredentialFileMapping(response.HostCredentialFileMapping)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func buildServiceAccountTokenFilepath(tokenDir string) string {
	return filepath.Join(tokenDir, serviceAccountTokenFilename)
}

func retrieveManagedIdentityPayload(data string) (*tharsisfederated.Data, error) {
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode managed identity payload %v", err)
	}

	federatedData := tharsisfederated.Data{}
	if err = json.Unmarshal(decodedData, &federatedData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal managed identity payload %v", err)
	}

	return &federatedData, nil
}

func (a *Authenticator) setupServiceAccountTokenWithRefresh(ctx context.Context, serviceAccountPath string, creds string) error {
	expiresIn, err := updateServiceAccountTokenFile(ctx, serviceAccountPath, creds, a)
	if err != nil {
		return err
	}

	go refreshTokenBeforeExpiration(ctx, serviceAccountPath, creds, expiresIn, a)

	return nil
}

func (a *Authenticator) setupHostCredentialFileMapping(hostCredFileMap map[string]string) error {
	apiURL, err := url.Parse(a.apiEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse apiEndpoint: %v", err)
	}

	tokenFilePath := buildServiceAccountTokenFilepath(a.tokenDir)

	hostCredFileMap[apiURL.Host] = tokenFilePath

	if a.discoveryProtocolHost != nil && *a.discoveryProtocolHost != "" {
		hostCredFileMap[*a.discoveryProtocolHost] = tokenFilePath
	}

	return nil
}

func updateServiceAccountTokenFile(ctx context.Context, serviceAccountPath string, creds string, a *Authenticator) (*time.Duration, error) {
	serviceAccountToken, expiresIn, err := a.client.CreateServiceAccountToken(ctx, serviceAccountPath, creds)
	if err != nil {
		return nil, err
	}

	err = writeServiceAccountTokenFile(a.tokenDir, serviceAccountToken)
	if err != nil {
		return nil, err
	}

	return expiresIn, nil
}

func refreshTokenBeforeExpiration(ctx context.Context, serviceAccountPath string, creds string, expiresIn *time.Duration, a *Authenticator) {
	for {
		if expiresIn == nil {
			return
		}

		if *expiresIn <= a.refreshTokenEarlyDuration {
			a.jobLogger.Errorf("Warning: Service account token expiration is less than or equal to estimated time to refresh, token will not be refreshed")
			return
		}

		refreshAt := *expiresIn - a.refreshTokenEarlyDuration

		select {
		case <-ctx.Done():
			return
		case <-time.After(refreshAt):
			var err error
			expiresIn, err = updateServiceAccountTokenFile(ctx, serviceAccountPath, creds, a)
			if err != nil {
				a.jobLogger.Errorf("Failed to refresh service account token file: %v", err)
				return
			}
		}
	}
}

func writeServiceAccountTokenFile(tokenDir string, serviceAccountToken string) error {
	filepath := buildServiceAccountTokenFilepath(tokenDir)
	if err := os.WriteFile(filepath, []byte(serviceAccountToken), tokenFilePermissions); err != nil {
		return fmt.Errorf("failed to write managed identity service account token to disk %v", err)
	}

	return nil
}
