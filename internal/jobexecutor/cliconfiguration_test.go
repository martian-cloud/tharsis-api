package jobexecutor

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupCliConfiguration(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "should setup cli configuration",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := buildTerraformWorkspace()

			err := workspace.setupCliConfiguration("helper-name")

			assert.Nil(t, err, "should not have any errors")

			cliConfigurationPath := workspace.fullEnv[tfCliConfigFileEnvName]

			assert.NotEmpty(t, cliConfigurationPath)

			assert.Equal(t, `credentials_helper "helper-name" {}`, readFile(t, cliConfigurationPath))

			verifyWillCleanupTempDir(t, workspace, cliConfigurationPath)
		})
	}
}

func buildTerraformWorkspace() *terraformWorkspace {
	return &terraformWorkspace{
		fullEnv: make(map[string]string),
	}
}

func readFile(t *testing.T, path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(contents)
}

func verifyWillCleanupTempDir(t *testing.T, workspace *terraformWorkspace, cliConfigurationPath string) {
	indexOfTempDir := slices.IndexFunc(workspace.pathsToRemove, func(path string) bool {
		return strings.Contains(cliConfigurationPath, path)
	})

	assert.True(t, indexOfTempDir >= 0, "Should remove the tempDir when job finished")
}
