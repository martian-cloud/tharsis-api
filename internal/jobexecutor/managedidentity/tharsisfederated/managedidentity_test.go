package tharsisfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
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

const refreshTokenEarlyDuration = 1 * time.Second

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name                           string
		workspaceDir                   string
		expectError                    string
		expectLogWarning               string
		createServiceAccountTokenError string
		managedIdentityData            string
		expiresIn                      *time.Duration
		tokens                         []string
		additionalManagedIdentityID    string
	}{
		{
			name:   "should configure environment when authenticating",
			tokens: []string{"expected-token1"},
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
			name:                           "should fail if unable to create service account token",
			createServiceAccountTokenError: "failed to create service account token",
			expectError:                    "failed to create service account token",
			tokens:                         []string{"expected-token1"},
		},
		{
			name:         "should fail if unable to write service account token file",
			workspaceDir: "::~NotValid~",
			expectError:  "failed to write managed identity service account token to disk open",
			tokens:       []string{"expected-token1"},
		},
		{
			name:             "should not refresh token if expiration less than the refresh token early duration",
			expiresIn:        ptr.Duration(500 * time.Millisecond),
			expectLogWarning: "Warning: Service account token expiration is less than or equal to estimated time to refresh, token will not be refreshed",
			tokens:           []string{"expected-token1"},
		},
		{
			name:             "should not refresh token if expiration equal to the refresh token early duration",
			expiresIn:        ptr.Duration(refreshTokenEarlyDuration),
			expectLogWarning: "Warning: Service account token expiration is less than or equal to estimated time to refresh, token will not be refreshed",
			tokens:           []string{"expected-token1"},
		},
		{
			name:      "should refresh token if expiration greater than the refresh token early duration",
			expiresIn: ptr.Duration(2 * time.Second),
			tokens:    []string{"expected-token1"},
		},
		{
			name:      "should update service account token file with new token before expiration",
			expiresIn: ptr.Duration(2 * time.Second),
			tokens:    []string{"expected-token1", "expected-token2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := jobclient.NewMockClient(t)

			workspaceDir := ensureWorkspaceDirSetup(t, test.workspaceDir)

			jobLogger := buildJobLoggerWithStubs(t)

			authenticator := buildAuthenticator(t, client, workspaceDir, jobLogger)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			const serviceAccountPath = "service/account/path"
			hosts := []string{"alpha.com", "beta.com"}

			identities := setupManagedIdentities(t, serviceAccountPath, hosts, test.additionalManagedIdentityID, test.managedIdentityData)

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

			verifyHostCredentialFileMapping(workspaceDir, t, hosts, response)

			lastToken := test.tokens[len(test.tokens)-1]
			verifyServiceAccountTokenFileEventuallyContains(t, workspaceDir, lastToken)

			if test.expectLogWarning != "" {
				jobLogger.AssertCalled(t, "Errorf", test.expectLogWarning)
			}
		})
	}
}

func ensureWorkspaceDirSetup(t *testing.T, workspaceDir string) string {
	if workspaceDir != "" {
		return workspaceDir
	}

	return buildWorkspaceDir(t)
}

func buildWorkspaceDir(t *testing.T) string {
	workspaceDir, err := os.MkdirTemp("", "managedidentity-test-workspace-*")
	if err != nil {
		t.Fatal(err)
	}

	return workspaceDir
}

func buildJobLoggerWithStubs(t *testing.T) *joblogger.MockJobLogger {
	jobLogger := joblogger.NewMockJobLogger(t)

	jobLogger.On("Errorf", mock.Anything).Maybe().Return()

	return jobLogger
}

func buildAuthenticator(t *testing.T, client jobclient.Client, workspaceDir string, jobLogger *joblogger.MockJobLogger) *Authenticator {
	authenticator, err := newAuthenticator(client, workspaceDir, jobLogger, refreshTokenEarlyDuration)
	if err != nil {
		t.Fatal(err)
	}

	return authenticator
}

func setupManagedIdentities(
	t *testing.T,
	serviceAccountPath string,
	hosts []string,
	additionalManagedIdentityID string,
	managedIdentityData string,
) []types.ManagedIdentity {
	identities := []types.ManagedIdentity{}

	firstIdentity := buildManagedIdentity(t, serviceAccountPath, hosts, "managedIdentity-1", managedIdentityData)
	identities = append(identities, *firstIdentity)

	if additionalManagedIdentityID != "" {
		secondIdentity := buildManagedIdentity(t, serviceAccountPath, hosts, additionalManagedIdentityID, "")
		identities = append(identities, *secondIdentity)
	}

	return identities
}

func buildManagedIdentity(t *testing.T, serviceAccountPath string, hosts []string, id string, managedIdentityData string) *types.ManagedIdentity {
	if managedIdentityData == "" {
		managedIdentityData = buildManagedIdentityData(t, serviceAccountPath, hosts)
	}

	return &types.ManagedIdentity{
		Metadata: types.ResourceMetadata{
			ID: id,
		},
		Data: managedIdentityData,
	}
}

func buildManagedIdentityData(t *testing.T, serviceAccountPath string, hosts []string) string {
	data := &tharsisfederated.Data{
		ServiceAccountPath: serviceAccountPath,
		Hosts:              hosts,
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
		err = fmt.Errorf(errorMessage)
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

	assert.Eventually(t, verify, 3*time.Second, 500*time.Millisecond)
}

func readFile(t *testing.T, path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(contents)
}
