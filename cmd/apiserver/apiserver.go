// Package main package
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
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

	shutdownDone := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)

		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

		// Wait for signal
		<-sigint

		// Gracefully shutdown server
		apiServer.Shutdown(ctx)

		close(shutdownDone)
	}()

	// Start server
	apiServer.Start()

	// Wait for shutdown to finish
	<-shutdownDone
}
