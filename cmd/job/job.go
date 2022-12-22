package main

import (
	"context"
	"flag"
	"os"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

// Version is passed in via ldflags at build time
var Version = "1.0.0"

func main() {
	flag.Parse()
	// create root logger tagged with server version
	logger := logger.New().With("version", Version)

	logger.Infof("Starting Job Executor with version %s...", Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiURL := os.Getenv("API_URL")
	jobID := os.Getenv("JOB_ID")
	token := os.Getenv("JOB_TOKEN")
	discoveryProtocolHost := os.Getenv("DISCOVERY_PROTOCOL_HOST")

	if apiURL == "" || jobID == "" || token == "" {
		logger.Errorf("API_URL, JOB_ID, and JOB_TOKEN environment variables are required")
		return
	}

	client, err := jobexecutor.NewClient(apiURL, token)
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
		JobID:                 jobID,
		APIEndpoint:           apiURL,
		JobToken:              token,
		DiscoveryProtocolHost: discoveryProtocolHost,
	}

	// Start the run executor
	executor := jobexecutor.NewJobExecutor(&cfg, client, logger)

	if err := executor.Execute(ctx); err != nil {
		logger.Infof("Failed to execute job %v", err)
		return
	}

	logger.Infof("Completed job with ID %s", jobID)
}
