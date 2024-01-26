package jobexecutor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func isCancellationError(err error) bool {
	return err != nil && (te.IsContextCanceledError(err) || errors.Is(err, context.DeadlineExceeded))
}

func sanitizedArchivePath(destination, filePath string) (string, error) {
	destPath := filepath.Join(destination, filePath)
	if !strings.HasPrefix(destPath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return "", errors.New(filePath + ": illegal file path")
	}
	return destPath, nil
}
