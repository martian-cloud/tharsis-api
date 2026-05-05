// Package main package
package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const claimJobTimeout = 60 * time.Second

// ClientConfig holds configuration for creating a new runner client.
type ClientConfig struct {
	TokenResolver client.TokenResolver
	Logger        client.LeveledLogger
	APIEndpoint   string
	UserAgent     string
	TLSSkipVerify bool
}

// Client uses gRPC to interact with the Tharsis API
type Client struct {
	grpcClient *client.GRPCClient
}

// NewClient creates a new Client instance
func NewClient(ctx context.Context, cfg *ClientConfig) (*Client, error) {
	baseURL, err := url.Parse(cfg.APIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}

	grpcClient, err := client.NewGRPCClient(ctx, &client.GRPCClientConfig{
		HTTPEndpoint:  baseURL.String(),
		TokenResolver: cfg.TokenResolver,
		TLSSkipVerify: cfg.TLSSkipVerify,
		UserAgent:     cfg.UserAgent,
		Logger:        cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{grpcClient: grpcClient}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	return c.grpcClient.Close()
}

// GetRunnerID resolves a runner ID or TRN to the runner's GID.
func (c *Client) GetRunnerID(ctx context.Context, id string) (string, error) {
	resp, err := c.grpcClient.RunnersClient.GetRunnerByID(ctx, &pb.GetRunnerByIDRequest{Id: id})
	if err != nil {
		return "", err
	}

	return resp.Metadata.Id, nil
}

// CreateRunnerSession creates a new runner session.
func (c *Client) CreateRunnerSession(ctx context.Context, input *runner.CreateRunnerSessionInput) (string, error) {
	resp, err := c.grpcClient.RunnersClient.CreateRunnerSession(ctx, &pb.CreateRunnerSessionRequest{
		RunnerId: input.RunnerID,
	})
	if err != nil {
		return "", err
	}

	return resp.Metadata.Id, nil
}

// SendRunnerSessionHeartbeat sends a runner session heartbeat for the specified runner session.
func (c *Client) SendRunnerSessionHeartbeat(ctx context.Context, runnerSessionID string) error {
	_, err := c.grpcClient.RunnersClient.SendRunnerSessionHeartbeat(ctx, &pb.SendRunnerSessionHeartbeatRequest{
		SessionId: runnerSessionID,
	})

	return err
}

// CreateRunnerSessionError creates a runner session error for the specified runner session.
func (c *Client) CreateRunnerSessionError(ctx context.Context, runnerSessionID string, err error) error {
	_, cErr := c.grpcClient.RunnersClient.CreateRunnerSessionError(ctx, &pb.CreateRunnerSessionErrorRequest{
		RunnerSessionId: runnerSessionID,
		Message:         err.Error(),
	})

	return cErr
}

// ClaimJob claims the next available job for the specified runner
func (c *Client) ClaimJob(ctx context.Context, input *runner.ClaimJobInput) (*runner.ClaimJobResponse, error) {
	for {
		resp, err := c.claimJob(ctx, input.RunnerID)
		if err == nil {
			return &runner.ClaimJobResponse{
				JobID: resp.Job.Metadata.Id,
				Token: resp.Token,
			}, nil
		}

		// Keep retrying on deadline exceeded (long-poll timeout).
		if status.Code(err) != codes.DeadlineExceeded {
			return nil, err
		}
	}
}

func (c *Client) claimJob(ctx context.Context, runnerID string) (*pb.ClaimJobResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, claimJobTimeout)
	defer cancel()

	return c.grpcClient.JobsClient.ClaimJob(timeoutCtx, &pb.ClaimJobRequest{
		RunnerId: runnerID,
	})
}
