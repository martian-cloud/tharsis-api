package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"path/filepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client/token"
)

type tokenResponse struct {
	Token string `json:"token"`
}

// TokenResolverInput is the input to create a token resolver.
type TokenResolverInput struct {
	BaseURL              *url.URL
	CredentialHelperPath string
	CredentialHelperArgs []string
	Logger               client.LeveledLogger
	ServiceAccountID     string
	UserAgent            string
	TLSSkipVerify        bool
}

// NewTokenResolver creates a token resolver that uses a credential helper
// to obtain an OIDC token and exchanges it for a service account token.
func NewTokenResolver(ctx context.Context, input *TokenResolverInput) (client.TokenResolver, error) {
	return token.NewServiceAccount(
		ctx,
		input.BaseURL.String(),
		input.ServiceAccountID,
		func() ([]byte, error) {
			t, err := invokeCredentialHelper(input.CredentialHelperPath, input.CredentialHelperArgs)
			if err != nil {
				return nil, fmt.Errorf("failed to invoke credential helper: %v", err)
			}

			return []byte(t), nil
		},
		token.WithTLSSkipVerify(input.TLSSkipVerify),
		token.WithLogger(input.Logger),
		token.WithUserAgent(input.UserAgent),
	)
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
