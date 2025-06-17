// Package configfile provides a Configurer implementation that reads Kubernetes configuration from a specified file.
package configfile

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
)

var _ configurer.Configurer = (*ConfigFile)(nil)

// ConfigFile provides a Kubernetes configurer that reads configuration from a specified kubeconfig file.
type ConfigFile struct {
	filePath string
}

// New creates a new ConfigFile configurer with the provided kubeconfig file path.
func New(configFilePath string) (*ConfigFile, error) {
	// Check if kubeconfig file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig file does not exist: %s", configFilePath)
	}

	return &ConfigFile{filePath: configFilePath}, nil
}

// GetConfig returns a Kubernetes rest.Config using the kubeconfig file specified in ConfigFile.
func (c *ConfigFile) GetConfig(_ context.Context) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", c.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config from kubeconfig file: %v", err)
	}

	return config, nil
}
