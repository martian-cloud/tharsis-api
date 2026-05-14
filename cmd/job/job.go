// Package main package
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Version is passed in via ldflags at build time
var Version = "1.0.0"

// BuildTimestamp is passed in via ldflags at build time
var BuildTimestamp string

func main() {
	if BuildTimestamp == "" {
		BuildTimestamp = time.Now().UTC().Format(time.RFC3339)
	}

	flag.Parse()
	// create root logger tagged with server version
	logger := logger.New().With("version", Version)

	logger.Infof("Starting Job Executor with version %s...", Version)
	logger.Infof("Build timestamp: %s", BuildTimestamp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiEndpoint := os.Getenv("ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = os.Getenv("API_URL")
		if apiEndpoint != "" {
			logger.Warnf("API_URL is deprecated, use ENDPOINT instead")
		}
	}
	jobID := os.Getenv("JOB_ID")
	token := os.Getenv("JOB_TOKEN")

	if apiEndpoint == "" || jobID == "" || token == "" {
		logger.Errorf("ENDPOINT, JOB_ID, and JOB_TOKEN environment variables are required")
		return
	}

	discoveryProtocolHosts := []string{}
	for _, host := range strings.Split(os.Getenv("DISCOVERY_PROTOCOL_HOSTS"), ",") {
		trimmedHost := strings.TrimSpace(host)
		if trimmedHost != "" {
			discoveryProtocolHosts = append(discoveryProtocolHosts, trimmedHost)
		}
	}

	var tlsSkipVerify bool
	if v := os.Getenv("TLS_SKIP_VERIFY"); v != "" {
		value, err := strconv.ParseBool(v)
		if err != nil {
			logger.Errorf("Invalid TLS_SKIP_VERIFY value: %v", err)
			return
		}

		tlsSkipVerify = value
	}

	client, err := jobclient.NewClient(ctx, &jobclient.ClientConfig{
		APIEndpoint:   apiEndpoint,
		Token:         token,
		UserAgent:     client.BuildUserAgent("tharsis-job-executor", Version),
		TLSSkipVerify: tlsSkipVerify,
		Logger:        logger.Slog(),
	})
	if err != nil {
		logger.Errorf("Failed to create client %v", err)
		return
	}

	defer func() {
		if err := client.Close(); err != nil {
			logger.Errorf("Error closing client %v", err)
		}
	}()

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, syscall.SIGTERM)

		// Wait for signal
		<-sigint

		logger.Info("Job received SIGTERM signal and is attempting to gracefully cancel")

		// Cancel context to give job the ability to gracefully cancel
		cancel()
	}()

	// Create job config
	cfg := jobexecutor.JobConfig{
		JobID:                  jobID,
		APIEndpoint:            apiEndpoint,
		JobToken:               token,
		DiscoveryProtocolHosts: discoveryProtocolHosts,
	}

	// Start the run executor
	executor := jobexecutor.NewJobExecutor(ctx, &cfg, client, logger, Version)

	if err := executor.Execute(ctx); err != nil {
		logger.Infof("Failed to execute job %v", err)
		return
	}

	logger.Infof("Completed job with ID %s", jobID)
}
