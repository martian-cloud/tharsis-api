package jobexecutor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	coreworkspace "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func isCancellationError(err error) bool {
	return err != nil && (te.IsContextCanceledError(err) || errors.Is(err, context.DeadlineExceeded))
}

// isOutputLimitViolation reports whether err is the gRPC error returned by
// WorkspaceService.CreateStateVersion when the state version was created successfully
// but one or more outputs were rejected for exceeding configured size/count limits.
// This is the only CreateStateVersion error path that occurs after the state version
// has already been committed, so callers must not treat it as a creation failure
// (e.g., it should not trigger raw-state recovery logging).
func isOutputLimitViolation(err error) bool {
	return err != nil &&
		status.Code(err) == codes.InvalidArgument &&
		strings.Contains(status.Convert(err).Message(), coreworkspace.StateVersionOutputLimitViolationMsg)
}

func sanitizedArchivePath(destination, filePath string) (string, error) {
	destPath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destPath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return "", errors.New(filePath + ": illegal file path")
	}
	return destPath, nil
}
