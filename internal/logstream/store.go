package logstream

//go:generate go tool mockery --name Store --inpackage --case underscore

import (
	"context"
	"fmt"
	"io"
	"os"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// Store interface encapsulates the logic for saving and retrieving logs
type Store interface {
	WriteLogs(ctx context.Context, logStreamID string, startOffset int, buffer []byte) error
	ReadLogs(ctx context.Context, logStreamID string, startOffset int, limit int) ([]byte, error)
}

type store struct {
	objectStore objectstore.ObjectStore
	dbClient    *db.Client
}

// NewLogStore creates an instance of the LogStore interface
func NewLogStore(objectStore objectstore.ObjectStore, dbClient *db.Client) Store {
	return &store{objectStore: objectStore, dbClient: dbClient}
}

// WriteLogs saves a chunk of logs to the store
func (ls *store) WriteLogs(ctx context.Context, logStreamID string, startOffset int, buffer []byte) error {
	if startOffset < 0 {
		return errors.New("offset cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "log-stream")
	if err != nil {
		return errors.Wrap(
			err,
			"Failed to create temporary directory for run logs",
		)
	}
	defer os.RemoveAll(tmpDir)

	filePath := fmt.Sprintf("%s/%s", tmpDir, logStreamID)
	key := getObjectKey(logStreamID)

	logFile, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(
			err,
			"Failed to create temporary file for run logs",
		)
	}

	defer logFile.Close()

	// Download logs
	if err = ls.objectStore.DownloadObject(ctx, key, logFile, nil); err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return errors.Wrap(
			err,
			"Failed to download log file from object storage",
		)
	}

	writer, err := os.OpenFile(filePath, os.O_RDWR, 0o600) // nosemgrep: gosec.G304-1
	if err != nil {
		return errors.Wrap(
			err,
			"Failed to open log file for writing",
		)
	}
	defer writer.Close()

	fileInfo, err := writer.Stat()
	if err != nil {
		return errors.Wrap(
			err,
			"Failed to get file stats for log file",
		)
	}

	if int64(startOffset) > fileInfo.Size() {
		return errors.New(
			"Start offset of %d is past the end of the file", startOffset, errors.WithErrorCode(errors.EInvalid),
		)
	}

	if _, err = writer.WriteAt(buffer, int64(startOffset)); err != nil {
		return errors.Wrap(
			err,
			"Failed to append logs to log file",
		)
	}

	if err = writer.Truncate(int64(startOffset + len(buffer))); err != nil {
		return errors.Wrap(
			err,
			"Failed to truncate log file",
		)
	}

	if _, err = writer.Seek(0, io.SeekStart); err != nil {
		return errors.Wrap(
			err,
			"Failed to seek to start of log file",
		)
	}

	if err = ls.objectStore.UploadObject(ctx, key, writer); err != nil {
		return errors.Wrap(
			err,
			"Failed to upload log file to object storage",
		)
	}

	return nil
}

// ReadLogs returns a chunk of logs
func (ls *store) ReadLogs(ctx context.Context, logStreamID string, startOffset int, limit int) ([]byte, error) {
	if limit < 0 || startOffset < 0 {
		return nil, errors.New("limit and offset cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}

	tmpDir, err := os.MkdirTemp("", "log-stream")
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to create temporary directory for run logs",
		)
	}
	defer os.RemoveAll(tmpDir)

	filePath := fmt.Sprintf("%s/%s", tmpDir, logStreamID)
	key := getObjectKey(logStreamID)

	logFile, err := os.Create(filePath)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Failed to create temporary file for run logs",
		)
	}

	defer logFile.Close()

	// Download logs from object store
	logs, err := ls.readLogs(ctx, key, logFile, startOffset, limit)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			// Check if this is a logstream using the legacy key format
			logStream, glsErr := ls.dbClient.LogStreams.GetLogStreamByID(ctx, logStreamID)
			if glsErr != nil {
				return nil, glsErr
			}
			if logStream == nil {
				return nil, errors.New("log stream with ID %s not found", logStreamID, errors.WithErrorCode(errors.ENotFound))
			}

			if logStream.JobID != nil {
				return ls.attemptReadForLegacyFormat(ctx, *logStream.JobID, logFile, startOffset, limit)
			}

			// Return empty byte array
			return []byte{}, nil
		}
		return nil, errors.Wrap(
			err,
			"Failed to download log file from object store",
		)
	}

	return logs, nil
}

func (ls *store) attemptReadForLegacyFormat(ctx context.Context, jobID string, logFile *os.File, startOffset int, limit int) ([]byte, error) {
	job, err := ls.dbClient.Jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, errors.New("job with ID %s not found", jobID, errors.WithErrorCode(errors.ENotFound))
	}

	legacyKey := getLegacyObjectKeyForJob(job)
	logs, rlErr := ls.readLogs(ctx, legacyKey, logFile, startOffset, limit)
	if rlErr != nil {
		if errors.ErrorCode(rlErr) == errors.ENotFound {
			// Return empty byte array
			return []byte{}, nil
		}
		return nil, errors.Wrap(
			rlErr,
			"Failed to download log file from object store",
		)
	}
	return logs, nil
}

func (ls *store) readLogs(ctx context.Context, key string, logFile *os.File, startOffset int, limit int) ([]byte, error) {
	contentRange := fmt.Sprintf("bytes=%d-%d", startOffset, startOffset+limit)

	// Download logs from object store
	err := ls.objectStore.DownloadObject(
		ctx,
		key,
		logFile,
		&objectstore.DownloadOptions{
			ContentRange: &contentRange,
		},
	)

	if err != nil {
		return nil, err
	}

	return io.ReadAll(logFile)
}

func getObjectKey(streamID string) string {
	return fmt.Sprintf("logstreams/%s.txt", streamID)
}

func getLegacyObjectKeyForJob(job *models.Job) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/logs/%s.txt", job.WorkspaceID, job.RunID, job.Metadata.ID)
}
