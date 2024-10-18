package jobexecutor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const tfConfigDir = "terraform.d"

func TestInstallCredentialHelper(t *testing.T) {
	tests := []struct {
		name                     string
		hosts                    []string
		expectedArrayAssignments []string
		homeDir                  homeDirGetter
		setup                    func(*testing.T, homeDirGetter)
		expectedError            string
	}{
		{
			name:                     "should install credential helper",
			hosts:                    []string{"example.com"},
			expectedArrayAssignments: []string{`hostsTokenFileMapping["example.com"]="tokenFileContents"`},
		},
		{
			name:                     "should install credential helper with complex home directory",
			hosts:                    []string{"example.com"},
			expectedArrayAssignments: []string{`hostsTokenFileMapping["example.com"]="tokenFileContents"`},
			homeDir:                  buildTempComplexPathHomeDirGetter(),
		},
		{
			name:                     "should install credential helper when plugin directory exists",
			setup:                    createPluginDirectory,
			hosts:                    []string{"example.com"},
			expectedArrayAssignments: []string{`hostsTokenFileMapping["example.com"]="tokenFileContents"`},
		},
		{
			name: "should install credential helper when there are multiple hosts",
			hosts: []string{
				"example.com",
				"myotherdomain.com",
			},
			expectedArrayAssignments: []string{
				`hostsTokenFileMapping["example.com"]="tokenFileContents"`,
				`hostsTokenFileMapping["myotherdomain.com"]="tokenFileContents"`,
			},
		},
		{
			name:          "should fail to install credential if plugin directory is unable to get home directory",
			homeDir:       failToGetHomeDir,
			expectedError: "failed to setup terraform plugin directory: unable to get home directory",
		},
		{
			name:          "should fail to install credential if plugin directory is a file",
			setup:         createPluginFile,
			expectedError: "failed to setup terraform plugin directory: The following path was expected to be a directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			homeDir := test.homeDir
			if homeDir == nil {
				homeDir = buildTempHomeDirGetter()
			}

			if test.setup != nil {
				test.setup(t, homeDir)
			}

			fileMapping := map[string]string{}
			for _, host := range test.hosts {
				fileMapping[host] = "tokenFileContents"
			}

			credentialHelper := buildCredentialHelper(homeDir)

			name, err := credentialHelper.install(fileMapping)

			verifyFailedInstallResult(t, err, test.expectedError)
			if err != nil {
				return
			}

			verifyCredHelperFileContains(t, name, homeDir, test.expectedArrayAssignments)
			verifyFilePath(t, credentialHelper, homeDir, name)
		})
	}
}

func TestClose(t *testing.T) {
	file, err := os.CreateTemp("", "credentialhelper-test-filepath*")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	credentialHelper := &credentialHelper{filepath: file.Name()}

	credentialHelper.close()

	_, err = os.Stat(file.Name())
	assert.True(t, os.IsNotExist(err))
}

func buildTempHomeDirGetter() func() (string, error) {
	var path string
	var err error

	return func() (string, error) {
		if path == "" {
			path, err = os.MkdirTemp("", "credentialhelper-test-*")
		}

		return path, err
	}
}

func buildTempComplexPathHomeDirGetter() func() (string, error) {
	homeDir := buildTempHomeDirGetter()

	return func() (string, error) {
		path, err := homeDir()

		return filepath.Join(path, "complex", "path"), err
	}
}

func createPluginDirectory(t *testing.T, homeDir homeDirGetter) {
	pluginPath, err := buildPluginPath(homeDir)
	if err != nil {
		t.Fatal(err)
		return
	}

	createDirectory(t, pluginPath)
}

func createDirectory(t *testing.T, path string) {
	err := os.MkdirAll(path, pluginPermissions)
	if err != nil {
		t.Fatalf("Failed to create directory: %v, error: %v", path, err)
	}
}

func createPluginFile(t *testing.T, homeDir homeDirGetter) {
	configHome, err := buildTerraformConfigPath(homeDir)
	if err != nil {
		t.Fatal(err)
		return
	}

	createDirectory(t, configHome)

	pluginPath, err := buildPluginPath(homeDir)
	if err != nil {
		t.Fatal(err)
		return
	}

	err = os.WriteFile(pluginPath, []byte("Test"), pluginPermissions)
	if err != nil {
		t.Fatal(err)
	}
}

func failToGetHomeDir() (string, error) {
	return "", errors.New("unable to get home directory")
}

func buildCredentialHelper(homeDir homeDirGetter) *credentialHelper {
	credentialHelper := &credentialHelper{
		homeDir:   homeDir,
		configDir: tfConfigDir,
	}
	return credentialHelper
}

func verifyFailedInstallResult(t *testing.T, err error, expectedError string) {
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

func verifyCredHelperFileContains(t *testing.T, name *string, homeDir homeDirGetter, expectedContents []string) {
	helperPath, err := buildCredHelperFilePath(homeDir, *name)
	if err != nil {
		t.Fatal(err)
		return
	}

	contents, err := os.ReadFile(helperPath)
	if err != nil {
		t.Fatal(err)
		return
	}

	for _, expectedContent := range expectedContents {
		assert.Contains(t, string(contents), expectedContent)
	}
}

func verifyFilePath(t *testing.T, credentialHelper *credentialHelper, homeDir homeDirGetter, name *string) {
	helperPath, err := buildCredHelperFilePath(homeDir, *name)
	if err != nil {
		t.Fatal(err)
		return
	}

	assert.Equal(t, helperPath, credentialHelper.filepath)
}

func buildCredHelperFilePath(homeDir homeDirGetter, name string) (string, error) {
	pluginPath, err := buildPluginPath(homeDir)
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginPath, credHelperPrefix+name), nil
}

func buildPluginPath(homeDir homeDirGetter) (string, error) {
	configPath, err := buildTerraformConfigPath(homeDir)
	if err != nil {
		return "", nil
	}

	return filepath.Join(configPath, "plugins"), nil
}

func buildTerraformConfigPath(homeDir homeDirGetter) (string, error) {
	homePath, err := homeDir()
	if err != nil {
		return "", nil
	}

	return filepath.Join(homePath, tfConfigDir), nil
}
