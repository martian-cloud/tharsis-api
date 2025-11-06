package tharsisfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/tharsisfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const refreshTokenEarlyDuration = 5 * time.Second

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name                             string
		tokenDir                         string
		apiEndpoint                      string
		discoveryProtocolHosts           []string
		expectError                      string
		expectLogWarning                 string
		createServiceAccountTokenError   string
		managedIdentityData              string
		expiresIn                        *time.Duration
		tokens                           []string
		additionalManagedIdentityID      string
		useServiceAccountForTerraformCLI bool
	}{
		{
			name:                             "should configure environment when authenticating and not using service account for terraform cli",
			tokens:                           []string{},
			useServiceAccountForTerraformCLI: false,
			discoveryProtocolHosts:           []string{"tharsis.dev.com"},
		},
		{
			name:                             "should configure environment when authenticating",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
			discoveryProtocolHosts:           []string{"tharsis.dev.com"},
		},
		{
			name:                             "should configure environment when authenticating without a discovery protocol host",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
			discoveryProtocolHosts:           []string{},
		},
		{
			name:                        "should fail if more than one managed identity is provided",
			additionalManagedIdentityID: "managedIdentity-2",
			expectError:                 "expected exactly one tharsis federated managed identity, got 2",
		},
		{
			name:                "should fail if unable to decode managed identity payload",
			managedIdentityData: "invalid-base64",
			expectError:         "failed to decode managed identity payload illegal base64 data at input byte 7",
		},
		{
			name:                "should fail if unable to unmarshal managed identity payload",
			managedIdentityData: base64.StdEncoding.EncodeToString([]byte("invalid-json")),
			expectError:         "failed to unmarshal managed identity payload invalid character 'i' looking for beginning of value",
		},
		{
			name:                             "should fail if unable to create service account token",
			createServiceAccountTokenError:   "failed to create service account token",
			expectError:                      "failed to create service account token",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
		},
		{
			name:                             "should fail if unable to write service account token file",
			tokenDir:                         "::~NotValid~",
			expectError:                      "failed to write managed identity service account token to disk open",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
		},
		{
			name:                             "should fail if unable to setup host credential file mapping",
			apiEndpoint:                      ".comhttps://",
			expectError:                      "parse \".comhttps://\": first path segment in URL cannot contain colon",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
		},
		{
			name:                             "should not refresh token if expiration less than the refresh token early duration",
			expiresIn:                        ptr.Duration(refreshTokenEarlyDuration / 2),
			expectLogWarning:                 "Warning: Service account token expiration is less than or equal to estimated time to refresh, token will not be refreshed",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
		},
		{
			name:                             "should not refresh token if expiration equal to the refresh token early duration",
			expiresIn:                        ptr.Duration(refreshTokenEarlyDuration),
			expectLogWarning:                 "Warning: Service account token expiration is less than or equal to estimated time to refresh, token will not be refreshed",
			tokens:                           []string{"expected-token1"},
			useServiceAccountForTerraformCLI: true,
		},
		{
			name:                             "should update service account token file with new token before expiration",
			expiresIn:                        ptr.Duration(2 * refreshTokenEarlyDuration),
			tokens:                           []string{"expected-token1", "expected-token2"},
			useServiceAccountForTerraformCLI: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := jobclient.NewMockClient(t)

			tokenDir := ensureTokenDirSetup(t, test.tokenDir)

			jobLogger := buildJobLoggerWithStubs(t)

			apiEndpoint := test.apiEndpoint
			if apiEndpoint == "" {
				apiEndpoint = "https://api.tharsis.dev.com"
			}

			authenticator := buildAuthenticator(t, client, tokenDir, jobLogger, apiEndpoint, test.discoveryProtocolHosts)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			const serviceAccountPath = "service/account/path"

			identities := setupManagedIdentities(t, serviceAccountPath, test.additionalManagedIdentityID, test.managedIdentityData, test.useServiceAccountForTerraformCLI)

			creds := []byte("tokendata")

			for _, token := range test.tokens {
				stubCreateServiceAccountToken(
					ctx,
					client,
					serviceAccountPath,
					creds,
					test.expiresIn,
					token,
					test.createServiceAccountTokenError)
			}

			response, err := authenticator.Authenticate(
				ctx,
				identities,
				func(_ context.Context, _ *types.ManagedIdentity) ([]byte, error) {
					return creds, nil
				})

			verifyFailedResult(t, response, err, test.expectError)
			if err != nil {
				return
			}

			expectedEnv := map[string]string{
				"THARSIS_SERVICE_ACCOUNT_PATH":  serviceAccountPath,
				"THARSIS_SERVICE_ACCOUNT_TOKEN": string(creds),
			}
			assert.Equal(t, expectedEnv, response.Env)

			expectedHosts := []string{}
			if test.useServiceAccountForTerraformCLI {
				expectedHosts = append(expectedHosts, "api.tharsis.dev.com")

				expectedHosts = append(expectedHosts, test.discoveryProtocolHosts...)
			}
			verifyHostCredentialFileMapping(tokenDir, t, expectedHosts, response)

			if len(test.tokens) >= 1 {
				lastToken := test.tokens[len(test.tokens)-1]
				verifyServiceAccountTokenFileEventuallyContains(t, tokenDir, lastToken)
			}

			if test.expectLogWarning != "" {
				time.Sleep(time.Second)
				jobLogger.AssertCalled(t, "Errorf", test.expectLogWarning)
			}
		})
	}
}

func TestClose(t *testing.T) {
	client := jobclient.NewMockClient(t)

	tokenDir := buildTokenDir(t)

	err := os.WriteFile(filepath.Join(tokenDir, "test.log"), []byte("testing"), os.ModeAppend)
	assert.Nil(t, err, "should create test file in token directory")

	jobLogger := buildJobLoggerWithStubs(t)

	const apiEndpoint = "https://api.tharsis.dev.com"
	discoveryProtocolHosts := []string{"tharsis.dev.com"}

	authenticator := buildAuthenticator(t, client, tokenDir, jobLogger, apiEndpoint, discoveryProtocolHosts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = authenticator.Close(ctx)

	assert.Nil(t, err, "should have no errors")

	_, err = os.Stat(tokenDir)

	assert.True(t, os.IsNotExist(err), "The token directory was not deleted: %v", err)
}

func ensureTokenDirSetup(t *testing.T, tokenDir string) string {
	if tokenDir != "" {
		return tokenDir
	}

	return buildTokenDir(t)
}

func buildTokenDir(t *testing.T) string {
	tokenDir, err := os.MkdirTemp("", "managedidentity-test-token-*")
	if err != nil {
		t.Fatal(err)
	}

	return tokenDir
}

func buildJobLoggerWithStubs(t *testing.T) *joblogger.MockLogger {
	jobLogger := joblogger.NewMockLogger(t)

	jobLogger.On("Infof", mock.Anything, mock.Anything).Maybe().Return()
	jobLogger.On("Errorf", mock.Anything).Maybe().Return()

	return jobLogger
}

func buildAuthenticator(
	t *testing.T,
	client jobclient.Client,
	tokenDir string,
	jobLogger *joblogger.MockLogger,
	apiEndpoint string,
	discoveryProtocolHosts []string,
) *Authenticator {
	authenticator, err := newAuthenticator(client, jobLogger, refreshTokenEarlyDuration, tokenDir, apiEndpoint, discoveryProtocolHosts)
	if err != nil {
		t.Fatal(err)
	}

	return authenticator
}

func setupManagedIdentities(
	t *testing.T,
	serviceAccountPath string,
	additionalManagedIdentityID string,
	managedIdentityData string,
	useServiceAccountForTerraformCLI bool,
) []types.ManagedIdentity {
	identities := []types.ManagedIdentity{}

	firstIdentity := buildManagedIdentity(t, serviceAccountPath, "managedIdentity-1", managedIdentityData, useServiceAccountForTerraformCLI)
	identities = append(identities, *firstIdentity)

	if additionalManagedIdentityID != "" {
		secondIdentity := buildManagedIdentity(t, serviceAccountPath, additionalManagedIdentityID, "", useServiceAccountForTerraformCLI)
		identities = append(identities, *secondIdentity)
	}

	return identities
}

func buildManagedIdentity(t *testing.T, serviceAccountPath string, id string, managedIdentityData string, useServiceAccountForTerraformCLI bool) *types.ManagedIdentity {
	if managedIdentityData == "" {
		managedIdentityData = buildManagedIdentityData(t, serviceAccountPath, useServiceAccountForTerraformCLI)
	}

	return &types.ManagedIdentity{
		Metadata: types.ResourceMetadata{
			ID: id,
		},
		Data: managedIdentityData,
	}
}

func buildManagedIdentityData(t *testing.T, serviceAccountPath string, useServiceAccountForTerraformCLI bool) string {
	data := &tharsisfederated.Data{
		ServiceAccountPath:               serviceAccountPath,
		UseServiceAccountForTerraformCLI: useServiceAccountForTerraformCLI,
	}

	dataBuffer, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	return base64.StdEncoding.EncodeToString(dataBuffer)
}

func stubCreateServiceAccountToken(
	ctx context.Context,
	client *jobclient.MockClient,
	serviceAccountPath string,
	creds []byte,
	expiresIn *time.Duration,
	createdToken string,
	errorMessage string,
) {
	var err error

	if errorMessage != "" {
		err = errors.New(errorMessage)
		expiresIn = nil
		createdToken = ""
	}

	client.
		On("CreateServiceAccountToken", ctx, serviceAccountPath, string(creds)).
		Return(createdToken, expiresIn, err).
		Once()
}

func verifyFailedResult(t *testing.T, response *managedidentity.AuthenticateResponse, err error, expectedError string) {
	if err == nil {
		if expectedError == "" {
			return
		}

		t.Fatalf("Expected error %v but got nil", expectedError)
	}

	if expectedError == "" {
		t.Fatal(err)
	}

	assert.Nil(t, response)
	assert.Contains(t, err.Error(), expectedError)
}

func verifyHostCredentialFileMapping(workspaceDir string, t *testing.T, hosts []string, response *managedidentity.AuthenticateResponse) {
	tokenFilePath := buildServiceAccountTokenFilepath(workspaceDir)

	assert.Equal(t, len(hosts), len(response.HostCredentialFileMapping))

	for _, host := range hosts {
		assert.Equal(t, tokenFilePath, response.HostCredentialFileMapping[host])
	}
}

func verifyServiceAccountTokenFileEventuallyContains(t *testing.T, workspaceDir string, token string) {
	filepath := buildServiceAccountTokenFilepath(workspaceDir)

	verify := func() bool {
		contents := readFile(t, filepath)

		return token == string(contents)
	}

	assert.Eventually(t, verify, refreshTokenEarlyDuration * 3, 500*time.Millisecond)
}

func readFile(t *testing.T, path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(contents)
}
