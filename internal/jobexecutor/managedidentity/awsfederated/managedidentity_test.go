package awsfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestAuthenticate(t *testing.T) {
	token := []byte("tokendata")

	type testCase struct {
		identitiesCount int
	}

	testCases := []testCase{
		{identitiesCount: 1},
		{identitiesCount: 2},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			authenticator, _ := New()
			defer authenticator.Close(ctx)

			identities := make([]types.ManagedIdentity, tc.identitiesCount)
			for i := 0; i < tc.identitiesCount; i++ {
				dataBuffer, err := json.Marshal(&awsfederated.Data{Role: fmt.Sprintf("testrole-%d", i)})
				require.NoError(t, err)

				identities[i] = types.ManagedIdentity{
					Metadata: types.ResourceMetadata{
						ID: fmt.Sprintf("identity-%d", i),
					},
					Name:         fmt.Sprintf("managedIdentity-%d", i),
					ResourcePath: fmt.Sprintf("test-group/managedIdentity-%d", i),
					Data:         base64.StdEncoding.EncodeToString(dataBuffer),
				}
			}

			response, err := authenticator.Authenticate(
				ctx,
				identities,
				func(_ context.Context, _ *types.ManagedIdentity) ([]byte, error) {
					return token, nil
				},
			)

			require.NoError(t, err)

			assert.Nil(t, response.HostCredentialFileMapping)

			envs := response.Env

			if tc.identitiesCount == 1 {
				// Should set AWS_CONFIG_FILE, AWS_ROLE_ARN, and AWS_WEB_IDENTITY_TOKEN_FILE
				require.Len(t, envs, 3)
				require.Contains(t, envs, "AWS_CONFIG_FILE")
				require.Contains(t, envs, "AWS_ROLE_ARN")
				require.Contains(t, envs, "AWS_WEB_IDENTITY_TOKEN_FILE")

				// Compare the token file content
				data, rErr := os.ReadFile(envs["AWS_WEB_IDENTITY_TOKEN_FILE"])
				require.NoError(t, rErr)
				require.Equal(t, token, data)
			} else {
				// Should set AWS_CONFIG_FILE
				require.Len(t, envs, 1)
				require.Contains(t, envs, "AWS_CONFIG_FILE")
			}

			// Should always write the AWS profile to the AWS_CONFIG_FILE
			configFile, err := os.ReadFile(envs["AWS_CONFIG_FILE"])
			require.NoError(t, err)

			for i := 0; i < tc.identitiesCount; i++ {
				role := fmt.Sprintf("testrole-%d", i)
				nameOfFile := fmt.Sprintf("%s-token", identities[i].Metadata.ID)
				tokenFilepath := filepath.Join(authenticator.dir, nameOfFile)

				profile := fmt.Sprintf(awsProfileTemplate, identities[i].ResourcePath, role, tokenFilepath)
				assert.Contains(t, string(configFile), profile)
			}
		})
	}
}
