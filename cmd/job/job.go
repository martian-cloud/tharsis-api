// Package main package
package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Version is passed in via ldflags at build time
var Version = "1.0.0"

// BuildTimestamp is passed in via ldflags at build time
var BuildTimestamp = time.Now().UTC().Format(time.RFC3339)

func main() {
	flag.Parse()
	// create root logger tagged with server version
	logger := logger.New().With("version", Version)

	logger.Infof("Starting Job Executor with version %s...", Version)
	logger.Infof("Build timestamp: %s", BuildTimestamp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiURL := os.Getenv("API_URL")
	jobID := os.Getenv("JOB_ID")
	token := os.Getenv("JOB_TOKEN")

	if apiURL == "" || jobID == "" || token == "" {
		logger.Errorf("API_URL, JOB_ID, and JOB_TOKEN environment variables are required")
		return
	}

	discoveryProtocolHosts := []string{}
	for _, host := range strings.Split(os.Getenv("DISCOVERY_PROTOCOL_HOSTS"), ",") {
		trimmedHost := strings.TrimSpace(host)
		if trimmedHost != "" {
			discoveryProtocolHosts = append(discoveryProtocolHosts, trimmedHost)
		}
	}

	client, err := jobclient.NewClient(apiURL, token)
	if err != nil {
		logger.Errorf("Failed to create client %v", err)
		return
	}

	defer func() {
		if err := client.Close(); err != nil {
			logger.Errorf("Error closing client %v", err)
		}
	}()

	// Create job config
	cfg := jobexecutor.JobConfig{
		JobID:                  jobID,
		APIEndpoint:            apiURL,
		JobToken:               token,
		DiscoveryProtocolHosts: discoveryProtocolHosts,
	}

	// Start the run executor
	executor := jobexecutor.NewJobExecutor(&cfg, client, logger, Version)

	if err := executor.Execute(ctx); err != nil {
		logger.Infof("Failed to execute job %v", err)
		return
	}

	logger.Infof("Completed job with ID %s", jobID)
}
