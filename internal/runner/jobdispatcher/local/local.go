// Package local package
package local

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var pluginDataRequiredFields = []string{"api_url"}

// JobDispatcher is used for debugging jobs and should not be used in production
type JobDispatcher struct {
	logger                logger.Logger
	apiURL                string
	discoveryProtocolHost string
	version               string
}

// New creates a JobDispatcher
func New(pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger, version string) (*JobDispatcher, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("docker job dispatcher requires plugin data '%s' field", field)
		}
	}

	return &JobDispatcher{
		logger:                logger,
		apiURL:                pluginData["api_url"],
		discoveryProtocolHost: discoveryProtocolHost,
		version:               version,
	}, nil
}

// DispatchJob will launch a local job executor that can be used to facilitate debugging
func (l *JobDispatcher) DispatchJob(_ context.Context, jobID string, token string) (string, error) {
	client, err := jobclient.NewClient(l.apiURL, token)
	if err != nil {
		return "", err
	}

	go func() {
		jobCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		discoveryProtocolHosts := []string{}
		if l.discoveryProtocolHost != "" {
			discoveryProtocolHosts = append(discoveryProtocolHosts, l.discoveryProtocolHost)
		}

		// Create job config
		cfg := jobexecutor.JobConfig{
			JobID:                  jobID,
			APIEndpoint:            l.apiURL,
			JobToken:               token,
			DiscoveryProtocolHosts: discoveryProtocolHosts,
		}

		// Start the job executor
		executor := jobexecutor.NewJobExecutor(&cfg, client, l.logger, l.version)

		if err := executor.Execute(jobCtx); err != nil {
			l.logger.Errorf("Error running job %v", err)
		}

		if err := client.Close(); err != nil {
			l.logger.Errorf("Error closing client %v", err)
		}
	}()

	return "local", nil
}
