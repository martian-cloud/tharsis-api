package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Version is passed in via ldflags at build time
var Version = "1.0.0"

// BuildTimestamp is passed in via ldflags at build time
var BuildTimestamp = time.Now().UTC().Format(time.RFC3339)

func main() {
	// create root logger tagged with server version
	logger := logger.New().With("version", Version)

	logger.Infof("Starting Runner with version %s...", Version)
	logger.Infof("Build timestamp: %s", BuildTimestamp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiURL := os.Getenv("THARSIS_API_URL")
	if apiURL == "" {
		logger.Errorf("THARSIS_API_URL environment variable is required")
		return
	}

	runnerPath := os.Getenv("THARSIS_RUNNER_PATH")
	if runnerPath == "" {
		logger.Errorf("THARSIS_RUNNER_PATH environment variable is required")
		return
	}

	serviceAccountPath := os.Getenv("THARSIS_SERVICE_ACCOUNT_PATH")
	if serviceAccountPath == "" {
		logger.Errorf("THARSIS_SERVICE_ACCOUNT_PATH environment variable is required")
		return
	}

	dispatcherType := os.Getenv("THARSIS_JOB_DISPATCHER_TYPE")
	if dispatcherType == "" {
		logger.Errorf("THARSIS_JOB_DISPATCHER_TYPE environment variable is required")
		return
	}

	credHelperPath := os.Getenv("THARSIS_CREDENTIAL_HELPER_CMD_PATH")
	if credHelperPath == "" {
		logger.Errorf("THARSIS_CREDENTIAL_HELPER_CMD_PATH environment variable is required")
		return
	}

	pluginData := map[string]string{}
	// Load Job Dispatcher plugin data
	for k, v := range loadDispatcherData("THARSIS_JOB_DISPATCHER_DATA_") {
		pluginData[k] = v
	}

	client, err := NewClient(apiURL, serviceAccountPath, credHelperPath, strings.Split(os.Getenv("THARSIS_CREDENTIAL_HELPER_CMD_ARGS"), " "))
	if err != nil {
		logger.Errorf("failed to create runner client %v", err)
		return
	}

	runner, err := runner.NewRunner(ctx, runnerPath, logger, Version, client, &runner.JobDispatcherSettings{
		DispatcherType:       dispatcherType,
		ServiceDiscoveryHost: os.Getenv("THARSIS_SERVICE_DISCOVERY_HOST"),
		PluginData:           pluginData,
	})
	if err != nil {
		logger.Errorf("Failed to create runner %v", err)
		return
	}

	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

		// Wait for signal
		<-sigint

		logger.Info("Shutting down runner...")

		// Gracefully shutdown server
		cancel()
	}()

	runner.Start(ctx)

	logger.Info("Runner has gracefully shutdown")
}

func loadDispatcherData(envPrefix string) map[string]string {
	pluginData := make(map[string]string)

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		key := pair[0]
		val := pair[1]

		if strings.HasPrefix(key, envPrefix) {
			pluginDataKey := strings.ToLower(key[len(envPrefix):])
			pluginData[pluginDataKey] = val
		}
	}

	return pluginData
}
