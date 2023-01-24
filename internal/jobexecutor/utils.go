package jobexecutor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func isCancellationError(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

func sanitizedArchivePath(destination, filePath string) (string, error) {
	destPath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destPath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return "", errors.New(filePath + ": illegal file path")
	}
	return destPath, nil
}
