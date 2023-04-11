// Package job package
package job

import (
	"context"
	"fmt"
	"io"
	"os"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/objectstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// LogStore interface encapsulates the logic for saving and retrieving logs
type LogStore interface {
	SaveLogs(ctx context.Context, workspaceID string, runID string, logID string, startOffset int, buffer []byte) error
	GetLogs(ctx context.Context, workspaceID string, runID string, logID string, startOffset int, limit int) ([]byte, error)
}

type logStore struct {
	objectStore objectstore.ObjectStore
	dbClient    *db.Client
}

// NewLogStore creates an instance of the LogStore interface
func NewLogStore(objectStore objectstore.ObjectStore, dbClient *db.Client) LogStore {
	return &logStore{objectStore: objectStore, dbClient: dbClient}
}

// SaveLogs saves a log buffer
func (ls *logStore) SaveLogs(ctx context.Context, workspaceID string, runID string, jobID string, startOffset int, buffer []byte) error {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "run-logs")
	if err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create temporary directory for run logs",
		)
	}
	defer os.RemoveAll(tmpDir)

	filePath := fmt.Sprintf("%s/%s", tmpDir, jobID)
	key := getObjectKey(workspaceID, runID, jobID)

	logFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create temporary file for run logs",
		)
	}

	defer logFile.Close()

	// Download logs
	if err = ls.objectStore.DownloadObject(ctx, key, logFile, nil); err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to download log file from object storage",
		)
	}

	writer, err := os.OpenFile(filePath, os.O_RDWR, 0o600) // nosemgrep: gosec.G304-1
	if err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to open log file for writing",
		)
	}
	defer writer.Close()

	fileInfo, err := writer.Stat()
	if err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get file stats for log file",
		)
	}

	if int64(startOffset) > fileInfo.Size() {
		return errors.New(
			errors.EInvalid,
			"Start offset of %d is past the end of the file", startOffset,
		)
	}

	if _, err = writer.WriteAt(buffer, int64(startOffset)); err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to append logs to log file",
		)
	}

	if err = writer.Truncate(int64(startOffset + len(buffer))); err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to truncate log file",
		)
	}

	if _, err = writer.Seek(0, io.SeekStart); err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to seek to start of log file",
		)
	}

	if err = ls.objectStore.UploadObject(ctx, key, writer); err != nil {
		return errors.Wrap(
			err,
			errors.EInternal,
			"Failed to upload log file to object storage",
		)
	}

	descriptor, err := ls.dbClient.Jobs.GetJobLogDescriptorByJobID(ctx, jobID)
	if err != nil {
		return err
	}

	size := startOffset + len(buffer)

	if descriptor == nil {
		if _, err := ls.dbClient.Jobs.CreateJobLogDescriptor(ctx, &models.JobLogDescriptor{
			JobID: jobID,
			Size:  size,
		}); err != nil {
			return err
		}
	} else {
		descriptor.Size = size
		if _, err := ls.dbClient.Jobs.UpdateJobLogDescriptor(ctx, descriptor); err != nil {
			return err
		}
	}

	return nil
}

// GetLogs gets a chunk of logs
func (ls *logStore) GetLogs(ctx context.Context, workspaceID string, runID string, jobID string, startOffset int, limit int) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "run-logs")
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create temporary directory for run logs",
		)
	}
	defer os.RemoveAll(tmpDir)

	filePath := fmt.Sprintf("%s/%s", tmpDir, jobID)
	key := getObjectKey(workspaceID, runID, jobID)

	contentRange := fmt.Sprintf("bytes=%d-%d", startOffset, startOffset+limit)

	logFile, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to create temporary file for run logs",
		)
	}

	defer logFile.Close()

	// Download logs from object store
	err = ls.objectStore.DownloadObject(
		ctx,
		key,
		logFile,
		&objectstore.DownloadOptions{
			ContentRange: &contentRange,
		},
	)

	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			// Return empty byte array
			return []byte{}, nil
		}
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to download log file from object store",
		)
	}

	return io.ReadAll(logFile)
}

func getObjectKey(workspaceID string, runID string, logID string) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/logs/%s.txt", workspaceID, runID, logID)
}
