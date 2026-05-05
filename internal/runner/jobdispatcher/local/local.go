// Package local package
package local

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var pluginDataRequiredFields = []string{"endpoint"}

// JobDispatcher is used for debugging jobs and should not be used in production
type JobDispatcher struct {
	logger                logger.Logger
	apiURL                string
	discoveryProtocolHost string
	version               string
	tlsSkipVerify         bool
}

// New creates a JobDispatcher
func New(pluginData map[string]string, discoveryProtocolHost string, logger logger.Logger, version string) (*JobDispatcher, error) {
	if err := types.MigrateDeprecatedPluginDataFields(pluginData, logger); err != nil {
		return nil, err
	}

	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("docker job dispatcher requires plugin data '%s' field", field)
		}
	}

	var tlsSkipVerify bool
	if v := pluginData["tls_skip_verify"]; v != "" {
		var parseErr error
		tlsSkipVerify, parseErr = strconv.ParseBool(v)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid tls_skip_verify value: %w", parseErr)
		}
	}

	return &JobDispatcher{
		logger:                logger,
		apiURL:                pluginData["endpoint"],
		discoveryProtocolHost: discoveryProtocolHost,
		version:               version,
		tlsSkipVerify:         tlsSkipVerify,
	}, nil
}

// DispatchJob will launch a local job executor that can be used to facilitate debugging
func (l *JobDispatcher) DispatchJob(ctx context.Context, jobID string, token string) (string, error) {
	client, err := jobclient.NewClient(ctx, &jobclient.ClientConfig{
		APIEndpoint:   l.apiURL,
		Token:         token,
		UserAgent:     client.BuildUserAgent("tharsis-job-executor", l.version),
		TLSSkipVerify: l.tlsSkipVerify,
		Logger:        l.logger.Slog(),
	})
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
