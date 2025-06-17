// Package main package
package main

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const claimJobTimeout = 60 * time.Second

// Client uses the Tharsis SDK to claim a job
type Client struct {
	tharsisClient *tharsis.Client
	tokenProvider auth.TokenProvider
}

// NewClient creates a new Client instance
func NewClient(apiURL string, tokenProvider auth.TokenProvider) (*Client, error) {
	cfg, err := config.Load(config.WithEndpoint(apiURL), config.WithTokenProvider(tokenProvider))
	if err != nil {
		return nil, err
	}

	client, err := tharsis.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{tharsisClient: client, tokenProvider: tokenProvider}, nil
}

// CreateRunnerSession creates a new runner session.
func (c *Client) CreateRunnerSession(ctx context.Context, input *runner.CreateRunnerSessionInput) (string, error) {
	runnerSession, err := c.tharsisClient.RunnerSession.CreateRunnerSession(ctx, &types.CreateRunnerSessionInput{
		RunnerPath: input.RunnerPath,
	})
	if err != nil {
		return "", err
	}

	return runnerSession.Metadata.ID, nil
}

// SendRunnerSessionHeartbeat sends a runner session heartbeat for the specified runner session.
func (c *Client) SendRunnerSessionHeartbeat(ctx context.Context, runnerSessionID string) error {
	return c.tharsisClient.RunnerSession.SendRunnerSessionHeartbeat(ctx, &types.RunnerSessionHeartbeatInput{
		RunnerSessionID: runnerSessionID,
	})
}

// CreateRunnerSessionError creates a runner session error for the specified runner session.
func (c *Client) CreateRunnerSessionError(ctx context.Context, runnerSessionID string, err error) error {
	return c.tharsisClient.RunnerSession.CreateRunnerSessionError(ctx, &types.CreateRunnerSessionErrorInput{
		RunnerSessionID: runnerSessionID,
		ErrorMessage:    err.Error(),
	})
}

// ClaimJob claims the next available job for the specified runner
func (c *Client) ClaimJob(ctx context.Context, input *runner.ClaimJobInput) (*runner.ClaimJobResponse, error) {
	for {
		// Use long polling to claim next available job
		timeoutCtx, cancel := context.WithTimeout(ctx, claimJobTimeout)
		defer cancel()

		resp, err := c.tharsisClient.Job.ClaimJob(timeoutCtx, &types.ClaimJobInput{
			RunnerPath: input.RunnerPath,
		})
		if err != nil {
			// Return if parent context has been canceled
			if ctx.Err() != nil {
				return nil, err
			}
			// Continue with polling if timeout context has timed out
			if timeoutCtx.Err() != nil {
				continue
			}
			return nil, err
		}
		return &runner.ClaimJobResponse{
			JobID: resp.JobID,
			Token: resp.Token,
		}, nil
	}
}
