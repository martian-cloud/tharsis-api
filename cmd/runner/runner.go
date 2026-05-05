package main

import (
	"context"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// Version is passed in via ldflags at build time
var Version = "1.0.0"

// BuildTimestamp is passed in via ldflags at build time
var BuildTimestamp string

func main() {
	if BuildTimestamp == "" {
		BuildTimestamp = time.Now().UTC().Format(time.RFC3339)
	}

	// create root logger tagged with server version
	logger := logger.New().With("version", Version)

	logger.Infof("Starting Runner with version %s...", Version)
	logger.Infof("Build timestamp: %s", BuildTimestamp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiEndpoint := os.Getenv("THARSIS_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = os.Getenv("THARSIS_API_URL")
		if apiEndpoint != "" {
			logger.Warnf("THARSIS_API_URL is deprecated, use THARSIS_ENDPOINT instead")
		}
	}

	if apiEndpoint == "" {
		logger.Errorf("THARSIS_ENDPOINT environment variable is required")
		return
	}

	runnerID := os.Getenv("THARSIS_RUNNER_ID")
	if runnerID == "" {
		// Fall back to deprecated env var and convert path to TRN.
		runnerPath := os.Getenv("THARSIS_RUNNER_PATH")
		if runnerPath == "" {
			logger.Errorf("THARSIS_RUNNER_ID environment variable is required")
			return
		}

		logger.Warnf("THARSIS_RUNNER_PATH is deprecated, use THARSIS_RUNNER_ID instead")
		runnerID = trn.TypeRunner.Build(runnerPath)
	}

	serviceAccountID := os.Getenv("THARSIS_SERVICE_ACCOUNT_ID")
	if serviceAccountID == "" {
		// Fall back to deprecated env var and convert path to TRN.
		serviceAccountPath := os.Getenv("THARSIS_SERVICE_ACCOUNT_PATH")
		if serviceAccountPath == "" {
			logger.Errorf("THARSIS_SERVICE_ACCOUNT_ID environment variable is required")
			return
		}

		logger.Warnf("THARSIS_SERVICE_ACCOUNT_PATH is deprecated, use THARSIS_SERVICE_ACCOUNT_ID instead")
		serviceAccountID = trn.TypeServiceAccount.Build(serviceAccountPath)
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

	baseURL, err := url.Parse(apiEndpoint)
	if err != nil {
		logger.Errorf("failed to parse THARSIS_ENDPOINT %s: %v", apiEndpoint, err)
		return
	}

	var tlsSkipVerify bool
	if v := os.Getenv("TLS_SKIP_VERIFY"); v != "" {
		tlsSkipVerify, err = strconv.ParseBool(v)
		if err != nil {
			logger.Errorf("Invalid TLS_SKIP_VERIFY value: %v", err)
			return
		}
	}

	userAgent := client.BuildUserAgent("tharsis-runner", Version)
	slogger := logger.Slog()

	tokenResolver, err := NewTokenResolver(ctx, &TokenResolverInput{
		BaseURL:              baseURL,
		ServiceAccountID:     serviceAccountID,
		CredentialHelperPath: credHelperPath,
		CredentialHelperArgs: strings.Split(os.Getenv("THARSIS_CREDENTIAL_HELPER_CMD_ARGS"), " "),
		Logger:               slogger,
		UserAgent:            userAgent,
		TLSSkipVerify:        tlsSkipVerify,
	})
	if err != nil {
		logger.Errorf("failed to create token resolver: %v", err)
		return
	}

	defer func() {
		if cErr := tokenResolver.Close(); cErr != nil {
			logger.Errorf("failed to close token resolver: %v", cErr)
		}
	}()

	client, err := NewClient(ctx, &ClientConfig{
		APIEndpoint:   baseURL.String(),
		TokenResolver: tokenResolver,
		UserAgent:     userAgent,
		TLSSkipVerify: tlsSkipVerify,
		Logger:        slogger,
	})
	if err != nil {
		logger.Errorf("failed to create runner client %v", err)
		return
	}

	defer func() {
		if cErr := client.Close(); cErr != nil {
			logger.Errorf("failed to close runner client: %v", cErr)
		}
	}()

	// Resolve the runner TRN/ID to the canonical ID used by the API.
	resolvedRunnerID, err := client.GetRunnerID(ctx, runnerID)
	if err != nil {
		logger.Errorf("failed to resolve runner ID: %v", err)
		return
	}

	runner, err := runner.NewRunner(ctx, resolvedRunnerID, logger, Version, client, &runner.JobDispatcherSettings{
		DispatcherType:       dispatcherType,
		ServiceDiscoveryHost: os.Getenv("THARSIS_SERVICE_DISCOVERY_HOST"),
		PluginData:           pluginData,
		TokenGetterFunc: func(ctx context.Context) (string, error) {
			return tokenResolver.Token(ctx)
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
