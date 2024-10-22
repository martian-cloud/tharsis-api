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
	cliConfigurationPath, err := writeCliConfigurationFile(credentialHelperName, t.workspaceDir)
	if err != nil {
		return err
	}

	t.fullEnv[tfCliConfigFileEnvName] = *cliConfigurationPath
	return nil
}

func writeCliConfigurationFile(credHelperName, workspaceDir string) (*string, error) {
	contents := fmt.Sprintf(cliConfigurationFormat, credHelperName)

	cliConfigurationPath := filepath.Join(workspaceDir, filename)

	err := os.WriteFile(cliConfigurationPath, []byte(contents), permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to write terraform cli configuration file %v", err)
	}

	return &cliConfigurationPath, nil
}
