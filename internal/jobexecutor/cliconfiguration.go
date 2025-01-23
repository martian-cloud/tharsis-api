package jobexecutor

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	filename               = "cli-configuration.tfrc"
	permissions            = os.FileMode(0o600)
	tfCliConfigFileEnvName = "TF_CLI_CONFIG_FILE"

	cliConfigurationFormat = `credentials_helper "%s" {}`
)

func (t *terraformWorkspace) setupCliConfiguration(credentialHelperName string) error {
	tempDir, err := t.createTempDir()
	if err != nil {
		return err
	}

	cliConfigurationPath, err := writeCliConfigurationFile(credentialHelperName, *tempDir)
	if err != nil {
		return err
	}

	t.fullEnv[tfCliConfigFileEnvName] = *cliConfigurationPath

	return nil
}

func (t *terraformWorkspace) createTempDir() (*string, error) {
	tempDir, err := os.MkdirTemp("", "cli-configuration-*")
	if err != nil {
		return nil, err
	}

	t.deletePathWhenJobCompletes(tempDir)

	return &tempDir, nil
}

func writeCliConfigurationFile(credHelperName string, tempDir string) (*string, error) {
	cliConfigurationPath := filepath.Join(tempDir, filename)

	contents := fmt.Sprintf(cliConfigurationFormat, credHelperName)

	err := os.WriteFile(cliConfigurationPath, []byte(contents), permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to write terraform cli configuration file %v", err)
	}

	return &cliConfigurationPath, nil
}
