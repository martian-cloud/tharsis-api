// Package incluster provides a Kubernetes configurer that retrieves the configuration from the pod's environment.
package incluster

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes/configurer"
)

var _ configurer.Configurer = (*InCluster)(nil)

// InCluster provides a Kubernetes configurer that retrieves the configuration from the pod's environment.
type InCluster struct{}

// New creates a new InCluster configurer.
func New() *InCluster {
	return &InCluster{}
}

// GetConfig returns a Kubernetes rest.Config using the in-cluster configuration.
func (j *InCluster) GetConfig(_ context.Context) (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %v", err)
	}
	return config, nil
}
