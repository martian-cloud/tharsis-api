package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
)

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
