// Package main package
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"path/filepath"
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
}

// NewClient creates a new Client instance
func NewClient(apiURL string, serviceAccountPath string, credentialHelperPath string, credentialHelperArgs []string) (*Client, error) {
	baseURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid api endpoint: %v", err)
	}

	// Setup service account token provider
	tokenProvider, err := auth.NewServiceAccountTokenProvider(baseURL.String(), serviceAccountPath, func() (string, error) {
		token, chErr := invokeCredentialHelper(credentialHelperPath, credentialHelperArgs)
		if chErr != nil {
			return "", fmt.Errorf("failed to invoke credential helper: %v", chErr)
		}
		return token, nil
	})
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(config.WithEndpoint(baseURL.String()), config.WithTokenProvider(tokenProvider))
	if err != nil {
		return nil, err
	}

	client, err := tharsis.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{tharsisClient: client}, nil
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

type tokenResponse struct {
	Token string `json:"token"`
}

func invokeCredentialHelper(cmdPath string, args []string) (string, error) {
	cleanedPath := filepath.Clean(cmdPath)
	cmd := exec.Command(cleanedPath, args...) // nosemgrep: gosec.G204-1

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	output, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	errOutput, err := io.ReadAll(stderr)
	if err != nil {
		return "", err
	}

	if err = cmd.Wait(); err != nil {
		return "", fmt.Errorf("credential helper returned an error %s", string(errOutput))
	}

	var tokenResp tokenResponse
	err = json.Unmarshal(output, &tokenResp)
	if err != nil {
		return "", err
	}

	return tokenResp.Token, nil
}
