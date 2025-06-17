package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/auth"
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

	baseURL, err := url.Parse(apiURL)
	if err != nil {
		logger.Errorf("failed to parse THARSIS_API_URL %s: %v", apiURL, err)
		return
	}

	tokenProvider, err := createTokenProvider(baseURL.String(), serviceAccountPath, credHelperPath, strings.Split(os.Getenv("THARSIS_CREDENTIAL_HELPER_CMD_ARGS"), " "))
	if err != nil {
		logger.Errorf("failed to create token provider: %v", err)
		return
	}

	client, err := NewClient(baseURL.String(), tokenProvider)
	if err != nil {
		logger.Errorf("failed to create runner client %v", err)
		return
	}

	runner, err := runner.NewRunner(ctx, runnerPath, logger, Version, client, &runner.JobDispatcherSettings{
		DispatcherType:       dispatcherType,
		ServiceDiscoveryHost: os.Getenv("THARSIS_SERVICE_DISCOVERY_HOST"),
		PluginData:           pluginData,
		TokenGetterFunc: func(_ context.Context) (string, error) {
			return tokenProvider.GetToken()
		},
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

func createTokenProvider(apiURL string, serviceAccountPath string, credentialHelperPath string, credentialHelperArgs []string) (auth.TokenProvider, error) {
	// Setup service account token provider
	tokenProvider, err := auth.NewServiceAccountTokenProvider(apiURL, serviceAccountPath, func() (string, error) {
		token, chErr := invokeCredentialHelper(credentialHelperPath, credentialHelperArgs)
		if chErr != nil {
			return "", fmt.Errorf("failed to invoke credential helper: %v", chErr)
		}
		return token, nil
	})
	if err != nil {
		return nil, err
	}
	return tokenProvider, nil
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
