package main

import (
	"context"
	"flag"
	"os"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

// Version is passed in via ldflags at build time
var Version = "1.0.0"

var flagConfig = flag.String("config", "", "path to the config file")

func main() {
	flag.Parse()
	// create root logger tagged with server version
	logger := logger.New().With("version", Version)

	logger.Info("Starting API...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// load application configurations
	cfg, err := config.Load(*flagConfig, logger)
	if err != nil {
		logger.Errorf("Application config is invalid: %v", err)
		os.Exit(1)
	}

	apiServer, err := apiserver.New(ctx, cfg, logger)
	if err != nil {
		logger.Errorf("Failed to start API server: %v", err)
		os.Exit(1)
	}

	defer apiServer.Shutdown(ctx)

	// Start server
	apiServer.Start()
}
