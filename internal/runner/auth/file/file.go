// Package file provides an OIDC token source that reads the token from a file on disk.
package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/auth"
)

var _ auth.OIDCTokenSource = (*OIDCTokenSource)(nil)

// OIDCTokenSource provides an OIDC token source that reads the token from a file on disk.
//
// The file is read on every call rather than cached, because Kubernetes rewrites the projected
// service account token in place as it rotates. Caching the first read would reproduce the expiry
// behaviour of a static token and waste the kubelet's rotation.
type OIDCTokenSource struct {
	path string
}

// New creates a new file OIDC token source reading from path.
func New(path string) *OIDCTokenSource {
	return &OIDCTokenSource{path: path}
}

// GetToken returns the token currently on disk, trimmed of surrounding whitespace.
//
// Trimming matters: Kubernetes writes the projected token with no trailing newline, but a file
// created by hand for local testing usually has one, and stray whitespace inside the value would
// make the JWT unparseable on the server.
func (f *OIDCTokenSource) GetToken(_ context.Context) (string, error) {
	contents, err := os.ReadFile(filepath.Clean(f.path)) // nosemgrep: gosec.G304-1
	if err != nil {
		return "", fmt.Errorf("failed to read OIDC token file %s: %v", f.path, err)
	}

	token := strings.TrimSpace(string(contents))
	if token == "" {
		return "", fmt.Errorf("OIDC token file %s is empty", f.path)
	}

	return token, nil
}
