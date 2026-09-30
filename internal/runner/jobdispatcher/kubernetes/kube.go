// Package kubernetes package
package kubernetes

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var _ jobdispatcher.JobDispatcher = (*JobDispatcher)(nil)

// podNameKey is the dispatcher-data key under which the pod name is stored for cleanup.
const podNameKey = "podName"

// JobDispatcher uses a kubernetes client to dispatch jobs.
type JobDispatcher struct {
	config *config
	client client
	logger logger.Logger
}

// New creates a JobDispatcher.
func New(ctx context.Context, pluginData map[string]string, discoveryProtocolHost string, tokenGetter types.TokenGetterFunc, logger logger.Logger) (*JobDispatcher, error) {
	cfg, err := parseConfig(pluginData, discoveryProtocolHost, logger)
	if err != nil {
		return nil, err
	}

	configurer, err := parseConfigurer(ctx, pluginData, tokenGetter)
	if err != nil {
		return nil, err
	}

	logger.Infof("kubernetes job dispatcher will create job pods in namespace %q", cfg.namespace)

	return &JobDispatcher{
		config: cfg,
		client: &k8sRunner{
			namespace:  cfg.namespace,
			configurer: configurer,
		},
		logger: logger,
	}, nil
}

// DispatchJob starts a bare Kubernetes pod to execute the job.
func (j *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (map[string]string, error) {
	pod, err := j.config.buildPod(jobID, token)
	if err != nil {
		return nil, fmt.Errorf("kubernetes job dispatcher failed to build pod for job %s: %v", jobID, err)
	}

	result, err := j.client.CreatePod(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("kubernetes job dispatcher failed to run for job %s: %v", jobID, err)
	}

	return map[string]string{podNameKey: result.Name}, nil
}

// CleanupJob deletes the job's pod, whose name DispatchJob stored in the dispatcher data.
func (j *JobDispatcher) CleanupJob(ctx context.Context, jobID string, dispatcherData map[string]string) error {
	podName := dispatcherData[podNameKey]
	if podName == "" {
		// Nothing to delete without a pod name; skip the API call rather than delete an empty name.
		j.logger.Warnf("kubernetes job dispatcher skipping cleanup for job %s: dispatcher data has no pod name", jobID)
		return nil
	}

	if err := j.client.DeletePod(ctx, podName); err != nil {
		// The pod is already gone; nothing to clean up.
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("kubernetes job dispatcher failed to delete pod %s for job %s: %v", podName, jobID, err)
	}

	return nil
}

// Limits returns the resource limits jobs run under.
func (j *JobDispatcher) Limits() *types.ResourceLimits {
	return j.config.limits
}
