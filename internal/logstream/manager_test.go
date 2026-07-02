package logstream

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// rc wraps a string as an io.ReadCloser for mocking the streaming store/manager reads.
func rc(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

// readAllString fully reads and closes an io.ReadCloser, returning its contents as a string.
func readAllString(t *testing.T, r io.ReadCloser) string {
	t.Helper()
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(b)
}

func echoChunk(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
	return c, nil
}

func echoStream(_ context.Context, ls *models.LogStream) (*models.LogStream, error) {
	return ls, nil
}

func TestWriteLogs(t *testing.T) {
	streamID := "stream1"

	t.Run("first write creates a new chunk", func(t *testing.T) {
		ctx := context.Background()
		logs := []byte("this is a test log")

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 0,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.AnythingOfType("string"), 0, logs).Return(nil)

		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).Return(echoChunk)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		updated, err := manager.WriteLogs(ctx, logStream, 0, logs)
		require.NoError(t, err)
		assert.Equal(t, len(logs), updated.Size)
	})

	t.Run("append into existing active chunk", func(t *testing.T) {
		ctx := context.Background()
		existing := 4
		logs := []byte(" is a test")

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: existing,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(&models.LogStreamChunk{
			Metadata: models.ResourceMetadata{ID: "chunk0"}, LogStreamID: streamID,
			ChunkIndex: 0, StartOffset: 0, Size: existing, ObjectKey: "logstreams/stream1/c0.txt",
		}, nil)
		mockStore.On("WriteChunk", mock.Anything, "logstreams/stream1/c0.txt", existing, logs).Return(nil)

		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("UpdateLogStreamChunk", mock.Anything, mock.Anything).Return(echoChunk)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		updated, err := manager.WriteLogs(ctx, logStream, existing, logs)
		require.NoError(t, err)
		assert.Equal(t, existing+len(logs), updated.Size)
	})

	t.Run("write spanning multiple chunks seals filled chunks", func(t *testing.T) {
		ctx := context.Background()
		// 2.5x the fill size, so it splits into two full sealed chunks plus a half-full active one.
		half := DefaultMaxChunkSize / 2
		logs := bytes.Repeat([]byte("x"), 2*DefaultMaxChunkSize+half)

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 0,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		var created []*models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				created = append(created, c)
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		updated, err := manager.WriteLogs(ctx, logStream, 0, logs)
		require.NoError(t, err)
		assert.Equal(t, 2*DefaultMaxChunkSize+half, updated.Size)

		require.Len(t, created, 3)
		assert.Equal(t, []int{0, DefaultMaxChunkSize, 2 * DefaultMaxChunkSize},
			[]int{created[0].StartOffset, created[1].StartOffset, created[2].StartOffset})
		assert.Equal(t, []int{DefaultMaxChunkSize, DefaultMaxChunkSize, half},
			[]int{created[0].Size, created[1].Size, created[2].Size})
		assert.Equal(t, []bool{true, true, false}, []bool{created[0].Sealed, created[1].Sealed, created[2].Sealed})
		assert.Equal(t, []int{0, 1, 2}, []int{created[0].ChunkIndex, created[1].ChunkIndex, created[2].ChunkIndex})
	})

	t.Run("adopts legacy single-file object as sealed chunk 0", func(t *testing.T) {
		ctx := context.Background()
		logs := []byte("new")

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		// Stream has 100 bytes already written under the legacy single-file layout, but no chunk rows.
		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 100,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, logs).Return(nil)

		var created []*models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				created = append(created, c)
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		updated, err := manager.WriteLogs(ctx, logStream, 100, logs)
		require.NoError(t, err)
		assert.Equal(t, 103, updated.Size)

		require.Len(t, created, 2)
		// chunk 0 is the adopted legacy object (sealed, no object write)
		assert.Equal(t, 0, created[0].ChunkIndex)
		assert.Equal(t, 0, created[0].StartOffset)
		assert.Equal(t, 100, created[0].Size)
		assert.True(t, created[0].Sealed)
		assert.Equal(t, consolidatedObjectKey(streamID), created[0].ObjectKey)
		// chunk 1 holds the new bytes
		assert.Equal(t, 1, created[1].ChunkIndex)
		assert.Equal(t, 100, created[1].StartOffset)
		assert.Equal(t, 3, created[1].Size)
	})

	t.Run("duplicate resend whose overlap matches is an idempotent no-op", func(t *testing.T) {
		ctx := context.Background()
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 10,
		}
		// Overlap verification reads [0,5) and finds it matches the re-sent bytes.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 5).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 10, ObjectKey: "k"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k", 0, 5).Return(rc("hello"), nil)

		logr, _ := logger.NewForTest()
		// No transaction or write expected — just the read-back verification.
		manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		updated, err := manager.WriteLogs(ctx, logStream, 0, []byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 10, updated.Size)
	})

	t.Run("resend whose overlap differs is rejected with a conflict", func(t *testing.T) {
		ctx := context.Background()
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 10,
		}
		// Stored bytes at [0,5) are "hello", but the caller re-sends "world" at offset 0.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 5).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 10, ObjectKey: "k"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k", 0, 5).Return(rc("hello"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		_, err := manager.WriteLogs(ctx, logStream, 0, []byte("world"))
		require.Error(t, err)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
	})

	t.Run("offset past end of stream is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 5,
		}

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS}, nil, logr, 0)

		_, err := manager.WriteLogs(ctx, logStream, 100, []byte("x"))
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("write to a completed stream is rejected with a conflict", func(t *testing.T) {
		ctx := context.Background()
		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		// A completed stream is terminal; no chunk/store work should happen (only the lookup mock is
		// set, so any WriteChunk/tx call would fail the test).
		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 5, Completed: true,
		}

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS}, nil, logr, 0)

		_, err := manager.WriteLogs(ctx, logStream, 5, []byte("more"))
		require.Error(t, err)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
	})

}

func TestWriteLogsMaxSize(t *testing.T) {
	streamID := "stream1"

	t.Run("truncates a write that crosses the server max size and returns ETooLarge", func(t *testing.T) {
		ctx := context.Background()
		const maxSize = 10
		logs := []byte("0123456789ABCDEFGHIJKLMNO") // 25 bytes; only the first 10 should be stored

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 0,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.AnythingOfType("string"), 0, []byte("0123456789")).Return(nil)

		var created []*models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				created = append(created, c)
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, maxSize)

		// The prefix that fits is persisted and the stream is flagged truncated, but the call returns
		// ETooLarge so the runner knows to stop streaming.
		updated, err := manager.WriteLogs(ctx, logStream, 0, logs)
		require.Error(t, err)
		assert.Equal(t, errors.ETooLarge, errors.ErrorCode(err))
		// The cap message must carry the shared marker the runner matches on to stop streaming.
		assert.Contains(t, errors.ErrorMessage(err), LogSizeLimitReachedMsg)
		assert.Equal(t, maxSize, updated.Size)
		assert.True(t, updated.Truncated)
		require.Len(t, created, 1)
		assert.Equal(t, 10, created[0].Size)
	})

	t.Run("write to an already-capped stream is rejected with ETooLarge", func(t *testing.T) {
		ctx := context.Background()
		const maxSize = 10

		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		// Stream already at the cap; runner keeps sending at higher offsets.
		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: maxSize, Truncated: true,
		}

		logr, _ := logger.NewForTest()
		// No tx, chunk, or store calls expected: the cap check returns before the gap validation so the
		// runner gets a clear ETooLarge rather than an EInvalid gap error.
		manager := New(mockStore, &db.Client{LogStreams: mockLS}, nil, logr, maxSize)

		updated, err := manager.WriteLogs(ctx, logStream, maxSize+50, []byte("more"))
		require.Error(t, err)
		assert.Equal(t, errors.ETooLarge, errors.ErrorCode(err))
		assert.Contains(t, errors.ErrorMessage(err), LogSizeLimitReachedMsg)
		assert.Equal(t, maxSize, updated.Size)
	})
}

func TestReadLogsCompacted(t *testing.T) {
	streamID := "stream1"
	ctx := context.Background()

	mockStore := NewMockStore(t)

	// A compacted stream reads from the static legacy key, never querying chunks (no LogStreamChunks
	// in the db client — a chunk query would nil-panic).
	logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 14, Compacted: true}
	mockStore.On("ReadRange", mock.Anything, consolidatedObjectKey(streamID), 0, 100).Return(rc("compacted logs"), nil)

	logr, _ := logger.NewForTest()
	manager := New(mockStore, &db.Client{}, nil, logr, 0)

	logs, err := manager.ReadLogs(ctx, logStream, 0, 100)
	require.NoError(t, err)
	assert.Equal(t, "compacted logs", readAllString(t, logs))
}

func TestCompactStream(t *testing.T) {
	streamID := "stream1"

	t.Run("consolidates chunks into the static-key object and sets compacted", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10, Completed: true}

		// Reading all chunks [0,10) for the consolidation.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 10).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 4, ObjectKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectKey: "k2"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 4).Return(rc("abcd"), nil)
		mockStore.On("ReadRange", mock.Anything, "k1", 0, 4).Return(rc("efgh"), nil)
		mockStore.On("ReadRange", mock.Anything, "k2", 0, 2).Return(rc("ij"), nil)

		// The consolidated object is written to the static key with the full stitched content.
		// Capture the reader's bytes via Run (reading it once, as the real WriteObject would) instead
		// of a MatchedBy, which testify may invoke multiple times and would drain the reader.
		var written string
		mockStore.On("WriteObject", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
			Run(func(args mock.Arguments) {
				b, _ := io.ReadAll(args.Get(2).(io.Reader))
				written = string(b)
			}).Return(nil)

		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
		assert.True(t, logStream.Compacted)
		assert.Equal(t, "abcdefghij", written)
	})

	t.Run("zero-size completed stream just sets the flag", func(t *testing.T) {
		ctx := context.Background()

		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 0, Completed: true}
		// No WriteObject / ReadRange / chunk calls expected.
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
		assert.True(t, logStream.Compacted)
	})

	t.Run("already compacted is a no-op", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10, Completed: true, Compacted: true}

		logr, _ := logger.NewForTest()
		// No db or store calls expected.
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
	})

	t.Run("a concurrent update racing the compacted-mark aborts cleanly", func(t *testing.T) {
		ctx := context.Background()
		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		// Zero-size stream: no object to write, so the compacted-mark is the only DB write. The caller
		// (scheduler) has already claimed the stream via SKIP LOCKED, so a lost optimistic lock here is
		// a rare concurrent update; compaction aborts cleanly without error.
		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 0, Completed: true}
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(nil, db.ErrOptimisticLockError).Once()

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
	})
}

func TestReadLogs(t *testing.T) {
	streamID := "stream1"

	t.Run("stitches overlapping chunks", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 100).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 4, ObjectKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectKey: "k2"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 4).Return(rc("abcd"), nil)
		mockStore.On("ReadRange", mock.Anything, "k1", 0, 4).Return(rc("efgh"), nil)
		mockStore.On("ReadRange", mock.Anything, "k2", 0, 2).Return(rc("ij"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logs, err := manager.ReadLogs(ctx, logStream, 0, 100)
		require.NoError(t, err)
		assert.Equal(t, "abcdefghij", readAllString(t, logs))
	})

	t.Run("reads a sub-range across chunk boundaries", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}
		// Read [3, 9): touches chunk0 [0,4) -> bytes [3,4), chunk1 [4,8) -> [4,8), chunk2 [8,10) -> [8,9)
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 3, 9).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 4, ObjectKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectKey: "k2"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 3, 1).Return(rc("d"), nil)
		mockStore.On("ReadRange", mock.Anything, "k1", 0, 4).Return(rc("efgh"), nil)
		mockStore.On("ReadRange", mock.Anything, "k2", 0, 1).Return(rc("i"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logs, err := manager.ReadLogs(ctx, logStream, 3, 6)
		require.NoError(t, err)
		assert.Equal(t, "defghi", readAllString(t, logs))
	})

	t.Run("falls back to legacy single-file object when no chunks exist", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 11}
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 100).Return([]models.LogStreamChunk{}, nil)
		mockStore.On("ReadRange", mock.Anything, consolidatedObjectKey(streamID), 0, 100).Return(rc("legacy logs"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logs, err := manager.ReadLogs(ctx, logStream, 0, 100)
		require.NoError(t, err)
		assert.Equal(t, "legacy logs", readAllString(t, logs))
	})

	t.Run("falls back to legacy per-job object", func(t *testing.T) {
		ctx := context.Background()
		jobID := "job1"

		mockLC := db.NewMockLogStreamChunks(t)
		mockJobs := db.NewMockJobs(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 7, JobID: &jobID}
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 100).Return([]models.LogStreamChunk{}, nil)
		mockStore.On("ReadRange", mock.Anything, consolidatedObjectKey(streamID), 0, 100).Return(
			nil, errors.New("Not Found", errors.WithErrorCode(errors.ENotFound)))

		job := &models.Job{Metadata: models.ResourceMetadata{ID: jobID}, WorkspaceID: "ws1", RunID: "run1"}
		mockJobs.On("GetJobByID", mock.Anything, jobID).Return(job, nil)
		mockStore.On("ReadRange", mock.Anything, legacyJobObjectKey(job), 0, 100).Return(rc("old logs"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC, Jobs: mockJobs}, nil, logr, 0)

		logs, err := manager.ReadLogs(ctx, logStream, 0, 100)
		require.NoError(t, err)
		assert.Equal(t, "old logs", readAllString(t, logs))
	})
}

func TestSubscribe(t *testing.T) {
	streamID := "stream1"

	tests := []struct {
		name               string
		lastSeenLogSize    *int
		logStreamSize      int
		logStreamExists    bool
		expectErrorMessage string
		expectedEvents     []struct {
			size   int
			offset int
			logs   string
		}
		dbEvents []db.LogStreamEventData
	}{
		{
			name:            "successful subscription with no last seen size",
			logStreamExists: true,
			logStreamSize:   10,
			dbEvents: []db.LogStreamEventData{
				{Size: 13, Completed: false},
			},
			expectedEvents: []struct {
				size   int
				offset int
				logs   string
			}{
				{size: 13, offset: 10, logs: "new"},
			},
		},
		{
			name:            "successful subscription with last seen size",
			lastSeenLogSize: ptr.Int(5),
			logStreamExists: true,
			logStreamSize:   15,
			expectedEvents: []struct {
				size   int
				offset int
				logs   string
			}{
				{size: 10, offset: 5, logs: "chunk"},
				{size: 15, offset: 10, logs: "data2"},
			},
		},
		{
			name:               "log stream not found",
			logStreamExists:    false,
			expectErrorMessage: "log stream not found with ID",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			mockLogStreams := db.NewMockLogStreams(t)
			mockChunks := db.NewMockLogStreamChunks(t)
			mockStore := NewMockStore(t)

			var mockEvents *db.MockEvents
			var eventChan chan db.Event
			var errorChan chan error

			if len(test.dbEvents) > 0 {
				mockEvents = db.NewMockEvents(t)
				eventChan = make(chan db.Event, 10)
				errorChan = make(chan error, 1)
				mockEvents.On("Listen", mock.Anything).Return((<-chan db.Event)(eventChan), (<-chan error)(errorChan))
			}

			var logStream *models.LogStream
			if test.logStreamExists {
				logStream = &models.LogStream{
					Metadata: models.ResourceMetadata{ID: streamID},
					Size:     test.logStreamSize,
				}
			}

			mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(logStream, nil)

			if test.logStreamExists {
				// A single chunk covering the whole stream; reads are keyed by offset below.
				mockChunks.On("GetOverlappingChunks", mock.Anything, streamID, mock.Anything, mock.Anything).Return(
					[]models.LogStreamChunk{{StartOffset: 0, Size: 1 << 20, ObjectKey: "k"}}, nil).Maybe()
				for _, event := range test.expectedEvents {
					logs := event.logs
					mockStore.On("ReadRange", mock.Anything, "k", event.offset, mock.AnythingOfType("int")).Return(rc(logs), nil).Maybe()
				}
			}

			dbClient := &db.Client{
				LogStreams:      mockLogStreams,
				LogStreamChunks: mockChunks,
				Events:          mockEvents,
			}

			logr, _ := logger.NewForTest()
			var eventManager *events.EventManager
			if mockEvents != nil {
				eventManager = events.NewEventManager(dbClient, logr)
				eventManager.Start(ctx)
			} else {
				eventManager = events.NewEventManager(nil, logr)
			}

			manager := New(mockStore, dbClient, eventManager, logr, 0)

			options := &SubscriptionOptions{
				LogStreamID:     streamID,
				LastSeenLogSize: test.lastSeenLogSize,
			}

			channel, err := manager.Subscribe(ctx, options)

			if test.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectErrorMessage)
				assert.Nil(t, channel)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, channel)

				go func() {
					time.Sleep(10 * time.Millisecond)
					for _, dbEvent := range test.dbEvents {
						eventData, _ := json.Marshal(dbEvent)
						eventChan <- db.Event{
							Table:  "log_streams",
							Action: "UPDATE",
							ID:     streamID,
							Data:   eventData,
						}
					}
				}()

				for i, expectedEvent := range test.expectedEvents {
					select {
					case event := <-channel:
						assert.Equal(t, expectedEvent.size, event.Size)
						require.NotNil(t, event.Data)
						assert.Equal(t, expectedEvent.offset, event.Data.Offset)
						assert.Equal(t, expectedEvent.logs, event.Data.Logs)
					case <-time.After(100 * time.Millisecond):
						t.Fatalf("expected to receive event %d", i+1)
					}
				}
			}
		})
	}
}

func TestWriteLogsEdgeCases(t *testing.T) {
	streamID := "stream1"

	t.Run("a write that exactly fills a chunk seals it", func(t *testing.T) {
		ctx := context.Background()
		logs := bytes.Repeat([]byte("x"), DefaultMaxChunkSize) // exactly the fill size

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 0,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, logs).Return(nil)

		var created []*models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				created = append(created, c)
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		_, err := manager.WriteLogs(ctx, logStream, 0, logs)
		require.NoError(t, err)
		require.Len(t, created, 1)
		assert.Equal(t, DefaultMaxChunkSize, created[0].Size)
		assert.True(t, created[0].Sealed, "a chunk filled to exactly the fill size must be sealed")
	})

	t.Run("a negative offset is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)
		logr, _ := logger.NewForTest()
		// No db access expected: the offset is validated up front.
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}}
		_, err := manager.WriteLogs(ctx, logStream, -1, []byte("x"))
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("a write larger than the max patch size is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)
		logr, _ := logger.NewForTest()
		// No db access expected: the patch size is validated up front.
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}}
		_, err := manager.WriteLogs(ctx, logStream, 0, make([]byte, MaxLogPatchSizeBytes+1))
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("a partial-overlap resend skips the matching prefix and appends only the new bytes", func(t *testing.T) {
		ctx := context.Background()
		// Stream holds "hello" (5 bytes). The runner re-sends "helloworld" at offset 0: the first 5 bytes
		// overlap and match, so only "world" is written, appended at offset 5.
		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 5,
		}
		// Overlap verification reads the stored [0,5) and finds it matches.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 5).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 5, ObjectKey: "k"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k", 0, 5).Return(rc("hello"), nil)
		// Only the new "world" bytes are written, at offset 5 into the active chunk.
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(&models.LogStreamChunk{
			Metadata: models.ResourceMetadata{ID: "chunk0"}, LogStreamID: streamID,
			ChunkIndex: 0, StartOffset: 0, Size: 5, ObjectKey: "k",
		}, nil)
		mockStore.On("WriteChunk", mock.Anything, "k", 5, []byte("world")).Return(nil)

		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("UpdateLogStreamChunk", mock.Anything, mock.Anything).Return(echoChunk)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		updated, err := manager.WriteLogs(ctx, logStream, 0, []byte("helloworld"))
		require.NoError(t, err)
		assert.Equal(t, 10, updated.Size)
	})

	t.Run("a failed object write leaves the database untouched", func(t *testing.T) {
		ctx := context.Background()

		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 0,
		}
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		// Objects are written before the transaction, so a write failure must abort before any DB change.
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, mock.Anything).
			Return(errors.New("object store down"))

		logr, _ := logger.NewForTest()
		// No Transactions / Create / Update mocks: if any DB call happened the nil db fields would panic,
		// proving the failure short-circuits before the transaction.
		manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		_, err := manager.WriteLogs(ctx, logStream, 0, []byte("data"))
		require.Error(t, err)
	})

	t.Run("a sealed active chunk rolls over to a new chunk", func(t *testing.T) {
		ctx := context.Background()

		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{
			Metadata: models.ResourceMetadata{ID: streamID}, Size: 4,
		}
		// The tail chunk is sealed, so new data must start a fresh chunk at index+1.
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(&models.LogStreamChunk{
			Metadata: models.ResourceMetadata{ID: "chunk0"}, LogStreamID: streamID,
			ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectKey: "k0", Sealed: true,
		}, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, []byte("xyz")).Return(nil)

		var created []*models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				created = append(created, c)
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		_, err := manager.WriteLogs(ctx, logStream, 4, []byte("xyz"))
		require.NoError(t, err)
		require.Len(t, created, 1)
		assert.Equal(t, 1, created[0].ChunkIndex)
		assert.Equal(t, 4, created[0].StartOffset)
		assert.Equal(t, 3, created[0].Size)
	})
}

func TestReadLogsEdgeCases(t *testing.T) {
	streamID := "stream1"

	t.Run("a zero-length read returns empty without querying chunks", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)
		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}

		logr, _ := logger.NewForTest()
		// db.Client{} has no LogStreamChunks: a chunk query would nil-panic.
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		logs, err := manager.ReadLogs(ctx, logStream, 5, 0)
		require.NoError(t, err)
		assert.Empty(t, readAllString(t, logs))
	})

	t.Run("negative offset or limit is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)
		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		_, err := manager.ReadLogs(ctx, logStream, -1, 10)
		require.Error(t, err)

		_, err = manager.ReadLogs(ctx, logStream, 0, -1)
		require.Error(t, err)
	})

	t.Run("a read limit larger than the max patch size is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)
		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}

		logr, _ := logger.NewForTest()
		// No store/db access expected: the limit is validated before any read.
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		_, err := manager.ReadLogs(ctx, logStream, 0, MaxLogPatchSizeBytes+1)
		require.Error(t, err)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	})

	t.Run("a range entirely within one chunk", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 2, 5).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 10, ObjectKey: "k"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k", 2, 3).Return(rc("cde"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logs, err := manager.ReadLogs(ctx, logStream, 2, 3)
		require.NoError(t, err)
		assert.Equal(t, "cde", readAllString(t, logs))
	})

	t.Run("the stitched reader serves correct bytes when read one byte at a time", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10}
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 100).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 4, ObjectKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectKey: "k2"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 4).Return(rc("abcd"), nil)
		mockStore.On("ReadRange", mock.Anything, "k1", 0, 4).Return(rc("efgh"), nil)
		mockStore.On("ReadRange", mock.Anything, "k2", 0, 2).Return(rc("ij"), nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		reader, err := manager.ReadLogs(ctx, logStream, 0, 100)
		require.NoError(t, err)
		defer reader.Close()

		// Drive Read() with a 1-byte buffer so the reader must transition across all three chunk slices.
		var out []byte
		buf := make([]byte, 1)
		for {
			n, rerr := reader.Read(buf)
			out = append(out, buf[:n]...)
			if rerr == io.EOF {
				break
			}
			require.NoError(t, rerr)
		}
		assert.Equal(t, "abcdefghij", string(out))
	})
}

func TestCompactStreamEdgeCases(t *testing.T) {
	streamID := "stream1"

	t.Run("a nil log stream is rejected", func(t *testing.T) {
		ctx := context.Background()
		mockStore := NewMockStore(t)
		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{}, nil, logr, 0)

		err := manager.CompactStream(ctx, nil)
		require.Error(t, err)
	})

	t.Run("a failed consolidated write does not set the compacted flag", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10, Completed: true}

		// The stream is already claimed by the caller. Chunks read back, but the consolidated object
		// write fails before any DB write, so the compacted flag is never set and a later run retries.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 10).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 10, ObjectKey: "k0"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 10).Return(rc("abcdefghij"), nil)
		// WriteObject consumes the (lazy) stitched reader as a real upload would, then fails.
		mockStore.On("WriteObject", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
			Run(func(args mock.Arguments) {
				_, _ = io.ReadAll(args.Get(2).(io.Reader))
			}).Return(errors.New("object store down"))

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.Error(t, err)
		assert.False(t, logStream.Compacted)
	})

	t.Run("a concurrent update racing the compacted-mark aborts cleanly without error", func(t *testing.T) {
		ctx := context.Background()

		mockLC := db.NewMockLogStreamChunks(t)
		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10, Completed: true}

		// The consolidated object is written, but a concurrent update bumps the row version before the
		// mark-compacted update, so it loses the optimistic-lock check. Compaction must abort without
		// error (the scheduler shouldn't log a failure) and without marking compacted, so reads keep
		// using the chunk path and a later run recompacts.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 10).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 10, ObjectKey: "k0"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 10).Return(rc("abcdefghij"), nil)
		mockStore.On("WriteObject", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
			Run(func(args mock.Arguments) {
				_, _ = io.ReadAll(args.Get(2).(io.Reader))
			}).Return(nil)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).
			Return(nil, errors.New("optimistic lock", errors.WithErrorCode(errors.EOptimisticLock))).Once() // mark

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
	})
}

// TestSubscribeCompletionDrainsTail locks in the fix for the "log tail dropped at completion" bug: a
// single completed event whose Size is larger than what the subscriber has seen must first deliver the
// remaining bytes and only then emit the terminal completed event (carrying the full size), and the
// Truncated flag must propagate on both.
func TestSubscribeCompletionDrainsTail(t *testing.T) {
	streamID := "stream1"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mockLogStreams := db.NewMockLogStreams(t)
	mockChunks := db.NewMockLogStreamChunks(t)
	mockStore := NewMockStore(t)
	mockEvents := db.NewMockEvents(t)

	eventChan := make(chan db.Event, 10)
	errorChan := make(chan error, 1)
	mockEvents.On("Listen", mock.Anything).Return((<-chan db.Event)(eventChan), (<-chan error)(errorChan))

	// Subscriber starts already having seen nothing; the stream is at 10 when subscribed.
	mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(&models.LogStream{
		Metadata: models.ResourceMetadata{ID: streamID}, Size: 10,
	}, nil)
	mockChunks.On("GetOverlappingChunks", mock.Anything, streamID, mock.Anything, mock.Anything).Return(
		[]models.LogStreamChunk{{StartOffset: 0, Size: 1 << 20, ObjectKey: "k"}}, nil).Maybe()
	// The tail [10,13) is read from the chunk during the drain.
	mockStore.On("ReadRange", mock.Anything, "k", 10, mock.AnythingOfType("int")).Return(rc("abc"), nil).Maybe()

	dbClient := &db.Client{LogStreams: mockLogStreams, LogStreamChunks: mockChunks, Events: mockEvents}

	logr, _ := logger.NewForTest()
	eventManager := events.NewEventManager(dbClient, logr)
	eventManager.Start(ctx)

	manager := New(mockStore, dbClient, eventManager, logr, 0)

	channel, err := manager.Subscribe(ctx, &SubscriptionOptions{LogStreamID: streamID})
	require.NoError(t, err)
	require.NotNil(t, channel)

	// Emit one completion event that also grows the stream from 10 to 13 and flags truncation.
	go func() {
		time.Sleep(10 * time.Millisecond)
		eventData, _ := json.Marshal(db.LogStreamEventData{Size: 13, Completed: true, Truncated: true})
		eventChan <- db.Event{Table: "log_streams", Action: "UPDATE", ID: streamID, Data: eventData}
	}()

	// First: the drained tail data event.
	select {
	case event := <-channel:
		assert.Equal(t, 13, event.Size)
		assert.False(t, event.Completed)
		assert.True(t, event.Truncated)
		require.NotNil(t, event.Data)
		assert.Equal(t, 10, event.Data.Offset)
		assert.Equal(t, "abc", event.Data.Logs)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected the tail data event before completion")
	}

	// Second: the terminal completed event with the full size and no data.
	select {
	case event := <-channel:
		assert.Equal(t, 13, event.Size)
		assert.True(t, event.Completed)
		assert.True(t, event.Truncated)
		assert.Nil(t, event.Data)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected the terminal completed event")
	}

	// The channel closes once the subscription completes.
	select {
	case _, ok := <-channel:
		assert.False(t, ok, "channel should be closed after completion")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected the channel to close after completion")
	}
}
