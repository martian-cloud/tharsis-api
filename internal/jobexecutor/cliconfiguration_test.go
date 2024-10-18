package jobexecutor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupCliConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		workspaceDir  string
		expectedError string
	}{
		{
			name: "should configure environment when authenticating",
		},
		{
			name:          "should fail if unable to write cli configuration file",
			workspaceDir:  "~InvalidDirectory~",
			expectedError: "failed to write terraform cli configuration file",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := buildTerraformWorkspace(t, test.workspaceDir)

			err := workspace.setupCliConfiguration("helper-name")

			verifyFailedSetupResult(t, err, test.expectedError)
			if err != nil {
				return
			}

			verifyFileHasContents(t, workspace, `credentials_helper "helper-name" {}`)
			verifyEnvironmentVariableSet(t, workspace)
		})
	}
}

func buildTerraformWorkspace(t *testing.T, workspaceDir string) *terraformWorkspace {
	if workspaceDir != "" {
		return buildTerraformWorkspaceWith(workspaceDir)

	}

	path, err := os.MkdirTemp("", "cliconfiguration-test-workspace-*")
	if err != nil {
		t.Fatalf("failed to create temporary workspace directory: %v", err)
	}

	return buildTerraformWorkspaceWith(path)
}

func buildTerraformWorkspaceWith(path string) *terraformWorkspace {
	return &terraformWorkspace{
		workspaceDir: path,
		fullEnv:      make(map[string]string),
	}
}
func verifyFailedSetupResult(t *testing.T, err error, expectedError string) {
	if err == nil {
		if expectedError == "" {
			return
		}

		t.Fatalf("Expected error %v but got nil", expectedError)
	}

	if expectedError == "" {
		t.Fatal(err)
	}

	assert.Contains(t, err.Error(), expectedError)
}

func verifyFileHasContents(t *testing.T, workspace *terraformWorkspace, expectedContents string) {
	path := buildCliConfigurationFilePath(workspace)

	contents := readFile(t, path)

	assert.Equal(t, expectedContents, contents)
}

func verifyEnvironmentVariableSet(t *testing.T, workspace *terraformWorkspace) {
	path := buildCliConfigurationFilePath(workspace)

	assert.Equal(t, path, workspace.fullEnv[tfCliConfigFileEnvName])
}

func buildCliConfigurationFilePath(workspace *terraformWorkspace) string {
	path := filepath.Join(workspace.workspaceDir, filename)
	return path
}

func readFile(t *testing.T, path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(contents)
}
