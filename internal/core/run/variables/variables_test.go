package variables

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
)

func ptrTo[T any](v T) *T {
	return &v
}

func TestModuleRegistryToken(t *testing.T) {
	// module.BuildTokenEnvVar("app.terraform.io") -> "TF_TOKEN_app_terraform_io"
	// Hyphens become double underscores, periods become single underscores.
	type testCase struct {
		name          string
		hostname      string
		envVars       []Variable
		expectedToken string
		expectErr     bool
	}

	testCases := []testCase{
		{
			name:     "matching host returns the token",
			hostname: "app.terraform.io",
			envVars: []Variable{
				{Key: "TF_TOKEN_app_terraform_io", Value: ptrTo("secret-token")},
			},
			expectedToken: "secret-token",
		},
		{
			name:     "hostname with hyphen is encoded with double underscores",
			hostname: "my-registry.example.com",
			envVars: []Variable{
				{Key: "TF_TOKEN_my__registry_example_com", Value: ptrTo("hyphen-token")},
			},
			expectedToken: "hyphen-token",
		},
		{
			name:     "non-matching host returns empty token",
			hostname: "app.terraform.io",
			envVars: []Variable{
				{Key: "TF_TOKEN_other_example_com", Value: ptrTo("other-token")},
			},
			expectedToken: "",
		},
		{
			name:          "no env vars returns empty token",
			hostname:      "app.terraform.io",
			envVars:       nil,
			expectedToken: "",
		},
		{
			name:     "first matching env var wins",
			hostname: "app.terraform.io",
			envVars: []Variable{
				{Key: "TF_TOKEN_app_terraform_io", Value: ptrTo("first")},
				{Key: "TF_TOKEN_app_terraform_io", Value: ptrTo("second")},
			},
			expectedToken: "first",
		},
		{
			name:     "invalid hostname yields empty token without error",
			hostname: "not a valid host",
			envVars: []Variable{
				{Key: "TF_TOKEN_app_terraform_io", Value: ptrTo("secret-token")},
			},
			expectedToken: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			getter := ModuleRegistryToken(tc.envVars)
			token, err := getter(context.Background(), tc.hostname)
			// BuildTokenEnvVar swallows its error inside the closure, so no
			// error should ever be returned.
			require.NoError(t, err)
			assert.Equal(t, tc.expectedToken, token)
		})
	}
}

func TestBuilder_Build(t *testing.T) {
	ctx := context.Background()
	workspaceID := "ws-1"

	type testCase struct {
		name            string
		runVariables    []Variable
		workspace       *models.Workspace
		workspaceErr    error
		variables       []models.Variable
		variablesErr    error
		secretValue     string
		secretErr       error
		expectErr       bool
		expectVarCount  int
		validate        func(t *testing.T, vars []Variable)
		expectSecretGet bool
	}

	testCases := []testCase{
		{
			name: "merges workspace and run variables and resolves secrets",
			runVariables: []Variable{
				{Key: "run_key", Value: ptrTo("run_value"), Category: models.TerraformVariableCategory},
			},
			workspace: &models.Workspace{FullPath: "group/ws"},
			variables: []models.Variable{
				{
					Key:             "ws_key",
					Value:           ptrTo("ws_value"),
					Category:        models.TerraformVariableCategory,
					NamespacePath:   "group/ws",
					LatestVersionID: "vv-1",
				},
				{
					Key:             "secret_key",
					Category:        models.TerraformVariableCategory,
					NamespacePath:   "group/ws",
					Sensitive:       true,
					SecretData:      []byte("encrypted"),
					LatestVersionID: "vv-2",
				},
			},
			secretValue:     "resolved-secret",
			expectSecretGet: true,
			expectVarCount:  3,
			validate: func(t *testing.T, vars []Variable) {
				byKey := map[string]Variable{}
				for _, v := range vars {
					byKey[v.Key] = v
				}
				require.Contains(t, byKey, "run_key")
				require.Contains(t, byKey, "ws_key")
				require.Contains(t, byKey, "secret_key")
				assert.Equal(t, "run_value", *byKey["run_key"].Value)
				assert.False(t, byKey["run_key"].Sensitive)
				assert.Equal(t, "ws_value", *byKey["ws_key"].Value)
				require.True(t, byKey["secret_key"].Sensitive)
				assert.Equal(t, "resolved-secret", *byKey["secret_key"].Value)
			},
		},
		{
			name: "run variable takes precedence over workspace variable with same key/category",
			runVariables: []Variable{
				{Key: "shared", Value: ptrTo("from_run"), Category: models.TerraformVariableCategory},
			},
			workspace: &models.Workspace{FullPath: "group/ws"},
			variables: []models.Variable{
				{
					Key:           "shared",
					Value:         ptrTo("from_ws"),
					Category:      models.TerraformVariableCategory,
					NamespacePath: "group/ws",
				},
			},
			expectVarCount: 1,
			validate: func(t *testing.T, vars []Variable) {
				require.Len(t, vars, 1)
				assert.Equal(t, "from_run", *vars[0].Value)
			},
		},
		{
			name:         "workspace not found returns error",
			runVariables: nil,
			workspace:    nil,
			expectErr:    true,
		},
		{
			name:         "GetWorkspaceByID error is propagated",
			runVariables: nil,
			workspaceErr: errors.New("db down"),
			expectErr:    true,
		},
		{
			name:         "GetVariables error is propagated",
			runVariables: nil,
			workspace:    &models.Workspace{FullPath: "group/ws"},
			variablesErr: errors.New("variables query failed"),
			expectErr:    true,
		},
		{
			name:         "secret manager error is propagated",
			runVariables: nil,
			workspace:    &models.Workspace{FullPath: "group/ws"},
			variables: []models.Variable{
				{
					Key:           "secret_key",
					Category:      models.TerraformVariableCategory,
					NamespacePath: "group/ws",
					Sensitive:     true,
					SecretData:    []byte("encrypted"),
				},
			},
			secretErr:       errors.New("kms failure"),
			expectSecretGet: true,
			expectErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockVariables := db.NewMockVariables(t)
			mockSecretManager := secret.NewMockManager(t)

			// GetWorkspaceByID is only reached after the HCL-env validation.
			if !tc.expectErr || tc.name != "HCL environment variable is rejected" {
				mockWorkspaces.On("GetWorkspaceByID", mock.Anything, workspaceID).
					Return(tc.workspace, tc.workspaceErr).Maybe()
			}

			if tc.workspace != nil && tc.workspaceErr == nil {
				var result *db.VariableResult
				if tc.variablesErr == nil {
					result = &db.VariableResult{Variables: tc.variables}
				}
				mockVariables.On("GetVariables", mock.Anything, mock.Anything).
					Return(result, tc.variablesErr).Maybe()
			}

			if tc.expectSecretGet {
				mockSecretManager.On("Get", mock.Anything, mock.Anything, mock.Anything).
					Return(tc.secretValue, tc.secretErr).Maybe()
			}

			b := &Builder{
				dbClient: &db.Client{
					Workspaces: mockWorkspaces,
					Variables:  mockVariables,
				},
				secretManager: mockSecretManager,
			}

			vars, err := b.Build(ctx, workspaceID, tc.runVariables)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, vars, tc.expectVarCount)
			if tc.validate != nil {
				tc.validate(t, vars)
			}
		})
	}
}

func TestBuilder_Get(t *testing.T) {
	ctx := context.Background()
	run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}}

	encode := func(t *testing.T, vars []Variable) io.ReadCloser {
		t.Helper()
		b, err := json.Marshal(vars)
		require.NoError(t, err)
		return io.NopCloser(strings.NewReader(string(b)))
	}

	t.Run("reads non-sensitive variables without resolving secrets", func(t *testing.T) {
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockVariableVersions := db.NewMockVariableVersions(t)
		mockSecretManager := secret.NewMockManager(t)

		stored := []Variable{
			{Key: "plain", Value: ptrTo("v1"), Category: models.TerraformVariableCategory},
		}
		mockArtifactStore.On("GetRunVariables", mock.Anything, run).
			Return(encode(t, stored), nil)

		b := &Builder{
			dbClient:      &db.Client{VariableVersions: mockVariableVersions},
			secretManager: mockSecretManager,
			artifactStore: mockArtifactStore,
		}

		vars, err := b.Get(ctx, run, false)
		require.NoError(t, err)
		require.Len(t, vars, 1)
		assert.Equal(t, "plain", vars[0].Key)
		assert.Equal(t, "v1", *vars[0].Value)
	})

	t.Run("resolves sensitive values when includeSensitiveValues is true", func(t *testing.T) {
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockVariableVersions := db.NewMockVariableVersions(t)
		mockSecretManager := secret.NewMockManager(t)

		stored := []Variable{
			{Key: "plain", Value: ptrTo("v1"), Category: models.TerraformVariableCategory},
			{Key: "secret", Sensitive: true, VersionID: ptrTo("vv-1"), Category: models.TerraformVariableCategory},
		}
		mockArtifactStore.On("GetRunVariables", mock.Anything, run).
			Return(encode(t, stored), nil)

		mockVariableVersions.On("GetVariableVersions", mock.Anything, mock.Anything).
			Return(&db.VariableVersionResult{
				VariableVersions: []models.VariableVersion{
					{
						Metadata:   models.ResourceMetadata{ID: "vv-1"},
						Key:        "secret",
						SecretData: []byte("encrypted"),
					},
				},
			}, nil)

		mockSecretManager.On("Get", mock.Anything, "secret", []byte("encrypted")).
			Return("resolved-secret", nil)

		b := &Builder{
			dbClient:      &db.Client{VariableVersions: mockVariableVersions},
			secretManager: mockSecretManager,
			artifactStore: mockArtifactStore,
		}

		vars, err := b.Get(ctx, run, true)
		require.NoError(t, err)
		require.Len(t, vars, 2)
		byKey := map[string]Variable{}
		for _, v := range vars {
			byKey[v.Key] = v
		}
		assert.Equal(t, "v1", *byKey["plain"].Value)
		require.NotNil(t, byKey["secret"].Value)
		assert.Equal(t, "resolved-secret", *byKey["secret"].Value)
	})

	t.Run("artifact store error is propagated", func(t *testing.T) {
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockSecretManager := secret.NewMockManager(t)

		mockArtifactStore.On("GetRunVariables", mock.Anything, run).
			Return(nil, errors.New("object store down"))

		b := &Builder{
			dbClient:      &db.Client{},
			secretManager: mockSecretManager,
			artifactStore: mockArtifactStore,
		}

		_, err := b.Get(ctx, run, false)
		assert.Error(t, err)
	})

	t.Run("missing version ID for sensitive variable returns error", func(t *testing.T) {
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockSecretManager := secret.NewMockManager(t)

		stored := []Variable{
			{Key: "secret", Sensitive: true, VersionID: nil, Category: models.TerraformVariableCategory},
		}
		mockArtifactStore.On("GetRunVariables", mock.Anything, run).
			Return(encode(t, stored), nil)

		b := &Builder{
			dbClient:      &db.Client{},
			secretManager: mockSecretManager,
			artifactStore: mockArtifactStore,
		}

		_, err := b.Get(ctx, run, true)
		assert.Error(t, err)
	})

	t.Run("missing variable versions returns error", func(t *testing.T) {
		mockArtifactStore := workspace.NewMockArtifactStore(t)
		mockVariableVersions := db.NewMockVariableVersions(t)
		mockSecretManager := secret.NewMockManager(t)

		stored := []Variable{
			{Key: "secret", Sensitive: true, VersionID: ptrTo("vv-1"), Category: models.TerraformVariableCategory},
		}
		mockArtifactStore.On("GetRunVariables", mock.Anything, run).
			Return(encode(t, stored), nil)

		// Return fewer versions than requested.
		mockVariableVersions.On("GetVariableVersions", mock.Anything, mock.Anything).
			Return(&db.VariableVersionResult{VariableVersions: nil}, nil)

		b := &Builder{
			dbClient:      &db.Client{VariableVersions: mockVariableVersions},
			secretManager: mockSecretManager,
			artifactStore: mockArtifactStore,
		}

		_, err := b.Get(ctx, run, true)
		assert.Error(t, err)
	})
}
