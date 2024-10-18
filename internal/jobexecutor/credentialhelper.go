package jobexecutor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

const (
	credHelperPrefix       = "terraform-credentials-"
	nameOfPluginsDirectory = "plugins"
	pluginPermissions      = os.FileMode(0750)
)

const bashCredHelper = `#!/bin/bash

# hostsTokenFileMapping is an associative array that maps hosts to a token file
%v

if [ $# != 2 ]; then
	echo "2 positional arguments required, $# provided"
	exit 1
fi

command="$1"
host="$2"

if [ "$command" != "get" ]; then
	echo "The following command is not supported: $command"
	exit 1
fi

for supportedHosts in ${!hostsTokenFileMapping[@]}
do
	if [[ $supportedHosts != *$host* ]]; then
		continue
	fi

    tokenFile=${hostsTokenFileMapping[${supportedHosts}]}

	tokenContents=$(<$tokenFile)
	if [ $? -ne 0 ]; then
		echo "Failed to read token file: $tokenFile"
		exit 1
	fi

	echo '{ "token": "'"$tokenContents"'" }'
	exit 0
done

echo "{}"
exit 0
`

type homeDirGetter func() (string, error)

type credentialHelper struct {
	homeDir   homeDirGetter
	configDir string
	filepath  string
}

func newCredentialHelper() *credentialHelper {
	// We do not support the windows terraform.d because it is not used by remote data sources.
	const tfConfigDir = ".terraform.d"

	return &credentialHelper{
		homeDir:   homedir.Dir,
		configDir: tfConfigDir,
	}
}

func (c *credentialHelper) install(hostsCredentialFileMapping map[string]string) (*string, error) {
	pluginPath, err := c.setupTerraformPluginDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to setup terraform plugin directory: %v", err)
	}

	name := uuid.New().String()

	c.filepath = filepath.Join(*pluginPath, credHelperPrefix+name)

	contents := fmt.Sprintf(bashCredHelper, serializeToBashAssociativeArray(hostsCredentialFileMapping))

	err = os.WriteFile(c.filepath, []byte(contents), pluginPermissions)
	if err != nil {
		return nil, fmt.Errorf("failed to write credential helper: %v", err)
	}

	return &name, nil
}

func (c *credentialHelper) close() {
	if c.filepath != "" {
		os.Remove(c.filepath)
	}
}

func (c *credentialHelper) setupTerraformPluginDirectory() (*string, error) {
	configHome, err := c.getTerraformConfigDirectory()
	if err != nil {
		return nil, err
	}

	pluginPath := filepath.Join(*configHome, nameOfPluginsDirectory)

	exists, err := doesDirectoryExist(pluginPath)
	if err != nil {
		return nil, err
	}

	if exists {
		return &pluginPath, nil
	}

	err = os.MkdirAll(pluginPath, pluginPermissions)
	if err != nil {
		return nil, err
	}

	return &pluginPath, nil
}

func doesDirectoryExist(path string) (bool, error) {
	stat, err := os.Stat(path)

	if err == nil {
		if !stat.IsDir() {
			return false, fmt.Errorf("The following path was expected to be a directory: %s", path)
		}

		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func (c *credentialHelper) getTerraformConfigDirectory() (*string, error) {
	dir, err := c.homeDir()
	if err != nil {
		return nil, err
	}

	var terraformConfigDirectory = filepath.Join(dir, c.configDir)

	return &terraformConfigDirectory, nil
}

func serializeToBashAssociativeArray(hostsCredentialFileMapping map[string]string) string {
	bashArray := "declare -A hostsTokenFileMapping\n"

	for host, tokenFile := range hostsCredentialFileMapping {
		bashArray += fmt.Sprintf("hostsTokenFileMapping[\"%s\"]=\"%s\"\n", host, tokenFile)
	}

	return bashArray
}
