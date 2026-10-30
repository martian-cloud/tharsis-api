// Package credentialhelper provides an OIDC token source that runs an external command which
// prints the token as JSON.
package credentialhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth"
)

var _ auth.OIDCTokenSource = (*OIDCTokenSource)(nil)

// tokenResponse is the JSON document the helper is expected to print on stdout.
type tokenResponse struct {
	Token string `json:"token"`
}

// OIDCTokenSource provides an OIDC token source that runs an external command to obtain a token.
//
// The command runs on every call rather than once, so a helper that mints short-lived tokens is
// re-invoked for each renewal.
type OIDCTokenSource struct {
	path string
	args []string
}

// New creates a new credential helper OIDC token source that runs path with args.
func New(path string, args []string) *OIDCTokenSource {
	return &OIDCTokenSource{path: path, args: args}
}

// GetToken runs the helper and returns the token from its JSON output.
//
// Behaviour is deliberately unchanged from when this ran in cmd/runner, including the message:
// the caller there wrapped every error from the helper with this same prefix, so a deployment
// matching on it still matches. A helper that exits non-zero reports its stderr, and output that
// parses as JSON without a token field yields an empty token rather than an error.
func (c *OIDCTokenSource) GetToken(_ context.Context) (string, error) {
	oidcToken, err := c.invoke()
	if err != nil {
		return "", fmt.Errorf("failed to invoke credential helper: %v", err)
	}

	return oidcToken, nil
}

func (c *OIDCTokenSource) invoke() (string, error) {
	cleanedPath := filepath.Clean(c.path)
	cmd := exec.Command(cleanedPath, c.args...) // nosemgrep: gosec.G204-1

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err = cmd.Start(); err != nil {
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
	if err = json.Unmarshal(output, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.Token, nil
}
