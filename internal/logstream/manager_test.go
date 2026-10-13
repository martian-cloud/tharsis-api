package logstream

import (
	"bytes"
	"context"
	"encoding/json"
	"hash/crc32"
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

// claimChunk mimics a successful ClaimLogStreamChunkLease: the returned chunk is a copy with its
// version bumped and the lease set.
func claimChunk(_ context.Context, c *models.LogStreamChunk, leaseDuration time.Duration) (*models.LogStreamChunk, error) {
	claimed := *c
	claimed.Metadata.Version++
	expires := time.Now().Add(leaseDuration)
	claimed.LeaseExpiresAt = &expires
	return &claimed, nil
}

func echoStream(_ context.Context, ls *models.LogStream) (*models.LogStream, error) {
	return ls, nil
}

// assertSavedCompacted asserts the stream was saved as compacted while the caller's copy was left
// unchanged.
func assertSavedCompacted(t *testing.T, mockLS *db.MockLogStreams, logStream *models.LogStream) {
	t.Helper()
	mockLS.AssertCalled(t, "UpdateLogStream", mock.Anything, mock.MatchedBy(func(ls *models.LogStream) bool { return ls.Compacted }))
	assert.False(t, logStream.Compacted, "the caller's stream must not be modified")
}

// mockStreamContent backs a log stream's reads with content, held in one chunk object "k".
func mockStreamContent(mockChunks *db.MockLogStreamChunks, mockStore *MockStore, streamID string, content string) {
	mockChunks.On("GetOverlappingChunks", mock.Anything, streamID, mock.Anything, mock.Anything).Return(
		[]models.LogStreamChunk{{LogStreamID: streamID, StartOffset: 0, Size: len(content), ObjectStoreKey: "k"}}, nil).Maybe()
	mockStore.On("ReadRange", mock.Anything, "k", mock.Anything, mock.Anything).Return(
		func(_ context.Context, _ string, offset int, length int) (io.ReadCloser, error) {
			return rc(content[offset:min(offset+length, len(content))]), nil
		}).Maybe()
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
		mockStore.On("WriteChunk", mock.Anything, mock.AnythingOfType("string"), 0, logs).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
		assert.Equal(t, 0, logStream.Size, "the caller's stream must not be modified")
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
			ChunkIndex: 0, StartOffset: 0, Size: existing, ObjectStoreKey: "logstreams/stream1/c0.txt",
		}, nil)
		mockLC.On("ClaimLogStreamChunkLease", mock.Anything, mock.Anything, chunkLeaseDuration).Return(claimChunk)
		mockStore.On("WriteChunk", mock.Anything, "logstreams/stream1/c0.txt", existing, logs).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, logs).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
		assert.Equal(t, consolidatedObjectKey(streamID), created[0].ObjectStoreKey)
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
			{StartOffset: 0, Size: 10, ObjectStoreKey: "k"},
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
			{StartOffset: 0, Size: 10, ObjectStoreKey: "k"},
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
		mockStore.On("WriteChunk", mock.Anything, mock.AnythingOfType("string"), 0, []byte("0123456789")).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
			{StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectStoreKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectStoreKey: "k2"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 4).Return(rc("abcd"), nil)
		mockStore.On("ReadRange", mock.Anything, "k1", 0, 4).Return(rc("efgh"), nil)
		mockStore.On("ReadRange", mock.Anything, "k2", 0, 2).Return(rc("ij"), nil)

		// WriteConsolidated captures the reader bytes and returns a no-op retainFn.
		var written string
		mockStore.On("WriteConsolidated", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
			Run(func(args mock.Arguments) {
				b, _ := io.ReadAll(args.Get(2).(io.Reader))
				written = string(b)
			}).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

		mockTx := db.NewMockTransactions(t)
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
		assertSavedCompacted(t, mockLS, logStream)
		assert.Equal(t, "abcdefghij", written)
	})

	t.Run("zero-size completed stream just sets the flag", func(t *testing.T) {
		ctx := context.Background()

		mockLS := db.NewMockLogStreams(t)
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 0, Completed: true}
		// No WriteObject / ReadRange / chunk calls expected.
		mockTx := db.NewMockTransactions(t)
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS}, nil, logr, 0)

		err := manager.CompactStream(ctx, logStream)
		require.NoError(t, err)
		assertSavedCompacted(t, mockLS, logStream)
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
		mockTx := db.NewMockTransactions(t)
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(nil, db.ErrOptimisticLockError).Once()

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS}, nil, logr, 0)

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
			{StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectStoreKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectStoreKey: "k2"},
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
			{StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectStoreKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectStoreKey: "k2"},
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
		content            string // the stream's full content, including bytes the events add
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
			content:         "0123456789new",
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
			content:         "01234chunkdata2",
			expectedEvents: []struct {
				size   int
				offset int
				logs   string
			}{
				{size: 15, offset: 5, logs: "chunkdata2"},
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
				mockStreamContent(mockChunks, mockStore, streamID, test.content)
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
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, logs).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
			{StartOffset: 0, Size: 5, ObjectStoreKey: "k"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k", 0, 5).Return(rc("hello"), nil)
		// Only the new "world" bytes are written, at offset 5 into the active chunk.
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(&models.LogStreamChunk{
			Metadata: models.ResourceMetadata{ID: "chunk0"}, LogStreamID: streamID,
			ChunkIndex: 0, StartOffset: 0, Size: 5, ObjectStoreKey: "k",
		}, nil)
		mockLC.On("ClaimLogStreamChunkLease", mock.Anything, mock.Anything, chunkLeaseDuration).Return(claimChunk)
		mockStore.On("WriteChunk", mock.Anything, "k", 5, []byte("world")).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, mock.Anything).
			Return(db.RetainObjectRefFunc(nil), errors.New("object store down"))

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
			ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0", Sealed: true,
		}, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, []byte("xyz")).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)

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
			{StartOffset: 0, Size: 10, ObjectStoreKey: "k"},
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
			{StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
			{StartOffset: 4, Size: 4, ObjectStoreKey: "k1"},
			{StartOffset: 8, Size: 2, ObjectStoreKey: "k2"},
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
		mockStore := NewMockStore(t)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 10, Completed: true}

		// The stream is already claimed by the caller. Chunks read back, but the consolidated object
		// write fails before any DB write, so the compacted flag is never set and a later run retries.
		mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 10).Return([]models.LogStreamChunk{
			{StartOffset: 0, Size: 10, ObjectStoreKey: "k0"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 10).Return(rc("abcdefghij"), nil)
		mockStore.On("WriteConsolidated", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
			Run(func(args mock.Arguments) {
				_, _ = io.ReadAll(args.Get(2).(io.Reader))
			}).Return(db.RetainObjectRefFunc(nil), errors.New("object store down"))

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

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
			{StartOffset: 0, Size: 10, ObjectStoreKey: "k0"},
		}, nil)
		mockStore.On("ReadRange", mock.Anything, "k0", 0, 10).Return(rc("abcdefghij"), nil)
		mockStore.On("WriteConsolidated", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
			Run(func(args mock.Arguments) {
				_, _ = io.ReadAll(args.Get(2).(io.Reader))
			}).Return(db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil }), nil)
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).
			Return(nil, errors.New("optimistic lock", errors.WithErrorCode(errors.EOptimisticLock))).Once()

		mockTx := db.NewMockTransactions(t)
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

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
	// The tail [10,13) is read from the chunk during the drain.
	mockStreamContent(mockChunks, mockStore, streamID, "0123456789abc")

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

func TestStream_Subscribe_Failure(t *testing.T) {
	streamID := "stream1"

	type testCase struct {
		name string
		// readErr is returned when reading the logs missed since the last seen size; nil means the
		// read is not reached and the failure comes from the event listener instead.
		readErr     error
		listenerErr error
		// event is sent to the subscriber when set.
		event           *db.Event
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "committed bytes missing from the object store is a permanent failure",
			readErr:         errors.New("range not satisfiable", errors.WithErrorCode(errors.ENotFound)),
			expectErrorCode: errors.EDataLoss,
		},
		{
			name:            "object store outage is a transient failure",
			readErr:         errors.New("object store unavailable"),
			expectErrorCode: errors.EServiceUnavailable,
		},
		{
			name:            "event listener failure is a transient failure",
			listenerErr:     errors.New("db listener connection lost"),
			expectErrorCode: errors.EServiceUnavailable,
		},
		{
			// Every later event would fail the same way, so a reconnect can't recover.
			name:            "a malformed event is a permanent failure",
			event:           &db.Event{Table: "log_streams", Action: "UPDATE", ID: streamID, Data: []byte("not json")},
			expectErrorCode: errors.EInternal,
		},
		{
			// Reconnecting with the same request would fail the same way, so it must not look transient.
			name:            "an invalid read keeps its code",
			readErr:         errors.New("offset cannot be negative", errors.WithErrorCode(errors.EInvalid)),
			expectErrorCode: errors.EInvalid,
		},
		{
			// A timeout inside the store while the subscriber is still connected must not end the stream
			// cleanly, which the client would read as completion.
			name:            "a store timeout while the subscriber is connected is a transient failure",
			readErr:         context.DeadlineExceeded,
			expectErrorCode: errors.EServiceUnavailable,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			mockLogStreams := db.NewMockLogStreams(t)
			mockChunks := db.NewMockLogStreamChunks(t)
			mockStore := NewMockStore(t)
			mockEvents := db.NewMockEvents(t)

			eventChan := make(chan db.Event, 1)
			errorChan := make(chan error, 1)
			// Optional: a read failure can end the test before the event manager goroutine starts listening.
			mockEvents.On("Listen", mock.Anything).Return((<-chan db.Event)(eventChan), (<-chan error)(errorChan)).Maybe()

			mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(&models.LogStream{
				Metadata: models.ResourceMetadata{ID: streamID}, Size: 10,
			}, nil)

			var lastSeenLogSize *int
			if test.readErr != nil {
				// The subscriber has seen nothing of the 10 byte stream, so it reads the backlog first.
				lastSeenLogSize = ptr.Int(0)
				mockChunks.On("GetOverlappingChunks", mock.Anything, streamID, 0, logEventChunkSizeBytes).Return(
					[]models.LogStreamChunk{{StartOffset: 0, Size: 10, ObjectStoreKey: "k"}}, nil)
				mockStore.On("ReadRange", mock.Anything, "k", 0, 10).Return(nil, test.readErr)
			}

			dbClient := &db.Client{LogStreams: mockLogStreams, LogStreamChunks: mockChunks, Events: mockEvents}

			logr, _ := logger.NewForTest()
			eventManager := events.NewEventManager(dbClient, logr)
			eventManager.Start(ctx)

			manager := New(mockStore, dbClient, eventManager, logr, 0)

			channel, err := manager.Subscribe(ctx, &SubscriptionOptions{LogStreamID: streamID, LastSeenLogSize: lastSeenLogSize})
			require.NoError(t, err)

			if test.listenerErr != nil {
				go func() {
					time.Sleep(10 * time.Millisecond)
					errorChan <- test.listenerErr
				}()
			}
			if test.event != nil {
				go func() {
					time.Sleep(10 * time.Millisecond)
					eventChan <- *test.event
				}()
			}

			// The failure arrives as a terminal error event, not as a clean close or a completion.
			select {
			case event, ok := <-channel:
				require.True(t, ok, "expected an error event before the channel closed")
				require.Error(t, event.Err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(event.Err))
				// Only a client error's cause reaches the client; a server fault's is only logged.
				cause := test.readErr
				if cause == nil {
					cause = test.listenerErr
				}
				if cause != nil {
					if test.expectErrorCode == errors.EInvalid {
						assert.Contains(t, errors.ErrorMessage(event.Err), cause.Error())
					} else {
						assert.NotContains(t, errors.ErrorMessage(event.Err), cause.Error())
					}
				}
				assert.False(t, event.Completed)
				assert.Nil(t, event.Data)
			case <-time.After(500 * time.Millisecond):
				t.Fatal("expected an error event")
			}

			select {
			case _, ok := <-channel:
				assert.False(t, ok, "channel should be closed after the error event")
			case <-time.After(200 * time.Millisecond):
				t.Fatal("expected the channel to close after the error event")
			}
		})
	}
}

func TestStream_Subscribe_ContextCanceledClosesWithoutError(t *testing.T) {
	streamID := "stream1"

	ctx, cancel := context.WithCancel(context.Background())

	mockLogStreams := db.NewMockLogStreams(t)
	mockEvents := db.NewMockEvents(t)

	eventChan := make(chan db.Event, 1)
	errorChan := make(chan error, 1)
	mockEvents.On("Listen", mock.Anything).Return((<-chan db.Event)(eventChan), (<-chan error)(errorChan)).Maybe()
	mockLogStreams.On("GetLogStreamByID", mock.Anything, streamID).Return(&models.LogStream{
		Metadata: models.ResourceMetadata{ID: streamID}, Size: 10,
	}, nil)

	dbClient := &db.Client{LogStreams: mockLogStreams, Events: mockEvents}

	logr, _ := logger.NewForTest()
	eventManager := events.NewEventManager(dbClient, logr)
	eventManager.Start(t.Context())

	manager := New(nil, dbClient, eventManager, logr, 0)

	channel, err := manager.Subscribe(ctx, &SubscriptionOptions{LogStreamID: streamID})
	require.NoError(t, err)

	cancel()

	// The subscriber going away is a normal end: the channel closes with no error event.
	select {
	case event, ok := <-channel:
		assert.False(t, ok, "expected the channel to close without an event, got %+v", event)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected the channel to close")
	}
}

func crc(s string) *uint32 {
	c := crc32.ChecksumIEEE([]byte(s))
	return &c
}

func noopRetain() db.RetainObjectRefFunc {
	return func(_ context.Context, _ string) error { return nil }
}

func TestStream_WriteLogs_ChunkLease(t *testing.T) {
	streamID := "stream1"

	activeChunk := func(checksum *uint32) *models.LogStreamChunk {
		return &models.LogStreamChunk{
			Metadata: models.ResourceMetadata{ID: "chunk0", Version: 3}, LogStreamID: streamID,
			ChunkIndex: 0, StartOffset: 0, Size: 5, ObjectStoreKey: "k0", Checksum: checksum,
		}
	}

	t.Run("a held lease rejects the write before touching the object store", func(t *testing.T) {
		ctx := context.Background()
		mockLC := db.NewMockLogStreamChunks(t)
		// No store or transaction expectations: any object write or DB commit fails the test.
		mockStore := NewMockStore(t)

		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(activeChunk(crc("hello")), nil)
		mockLC.On("ClaimLogStreamChunkLease", mock.Anything, mock.Anything, chunkLeaseDuration).Return(nil, db.ErrOptimisticLockError)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 5}
		_, err := manager.WriteLogs(ctx, logStream, 5, []byte("world"))
		require.Error(t, err)
		assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
	})

	t.Run("a stale stream view is rejected without claiming the lease", func(t *testing.T) {
		ctx := context.Background()
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		// The stream view says 5 bytes but the tail chunk already ends at 10: another write committed
		// since the view was loaded (the situation that let a stale writer clobber the chunk).
		chunk := activeChunk(nil)
		chunk.Size = 10
		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(chunk, nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 5}
		_, err := manager.WriteLogs(ctx, logStream, 5, []byte("world"))
		require.Error(t, err)
		assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
	})

	t.Run("a failed object write releases the claimed lease", func(t *testing.T) {
		ctx := context.Background()
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(activeChunk(crc("hello")), nil)
		mockLC.On("ClaimLogStreamChunkLease", mock.Anything, mock.Anything, chunkLeaseDuration).Return(claimChunk)
		mockStore.On("WriteChunk", mock.Anything, "k0", 5, []byte("world")).Return(db.RetainObjectRefFunc(nil), errors.New("object store down"))

		var released *models.LogStreamChunk
		mockLC.On("ReleaseLogStreamChunkLease", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			released = args.Get(1).(*models.LogStreamChunk)
		}).Return(nil)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 5}
		_, err := manager.WriteLogs(ctx, logStream, 5, []byte("world"))
		require.Error(t, err)

		// The release targets the claimed version, so it can't clear a lease someone else claimed later.
		require.NotNil(t, released)
		assert.Equal(t, "chunk0", released.Metadata.ID)
		assert.Equal(t, 4, released.Metadata.Version)
	})

	t.Run("a committed write clears the lease and extends the chunk checksum", func(t *testing.T) {
		ctx := context.Background()
		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(activeChunk(crc("hello")), nil)
		mockLC.On("ClaimLogStreamChunkLease", mock.Anything, mock.Anything, chunkLeaseDuration).Return(claimChunk)
		mockStore.On("WriteChunk", mock.Anything, "k0", 5, []byte("world")).Return(noopRetain(), nil)

		var updated *models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("UpdateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				updated = c
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		// No ReleaseLogStreamChunkLease expectation: a committed write must not issue a separate release.
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 5}
		result, err := manager.WriteLogs(ctx, logStream, 5, []byte("world"))
		require.NoError(t, err)
		assert.Equal(t, 10, result.Size)

		require.NotNil(t, updated)
		assert.Equal(t, 4, updated.Metadata.Version, "the update must be fenced by the claimed version")
		assert.Nil(t, updated.LeaseExpiresAt, "committing the write releases the lease")
		assert.Equal(t, 10, updated.Size)
		require.NotNil(t, updated.Checksum)
		assert.Equal(t, *crc("helloworld"), *updated.Checksum)
	})

	t.Run("a chunk without a known checksum keeps it unknown", func(t *testing.T) {
		ctx := context.Background()
		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(activeChunk(nil), nil)
		mockLC.On("ClaimLogStreamChunkLease", mock.Anything, mock.Anything, chunkLeaseDuration).Return(claimChunk)
		mockStore.On("WriteChunk", mock.Anything, "k0", 5, []byte("world")).Return(noopRetain(), nil)

		var updated *models.LogStreamChunk
		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockTx.On("CommitTx", mock.Anything).Return(nil)
		mockLC.On("UpdateLogStreamChunk", mock.Anything, mock.Anything).Return(
			func(_ context.Context, c *models.LogStreamChunk) (*models.LogStreamChunk, error) {
				updated = c
				return c, nil
			})
		mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 5}
		_, err := manager.WriteLogs(ctx, logStream, 5, []byte("world"))
		require.NoError(t, err)

		require.NotNil(t, updated)
		assert.Nil(t, updated.Checksum)
	})

	t.Run("new chunks record the checksum of their bytes", func(t *testing.T) {
		ctx := context.Background()
		mockTx := db.NewMockTransactions(t)
		mockLS := db.NewMockLogStreams(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		// Fill the first chunk and spill 3 bytes into a second one.
		first := strings.Repeat("a", DefaultMaxChunkSize)
		logs := []byte(first + "bcd")

		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, mock.Anything).Return(noopRetain(), nil)

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

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 0}
		_, err := manager.WriteLogs(ctx, logStream, 0, logs)
		require.NoError(t, err)

		require.Len(t, created, 2)
		require.NotNil(t, created[0].Checksum)
		require.NotNil(t, created[1].Checksum)
		assert.Equal(t, *crc(first), *created[0].Checksum)
		assert.Equal(t, *crc("bcd"), *created[1].Checksum)
	})

	t.Run("a chunk index created concurrently is reported as retryable", func(t *testing.T) {
		ctx := context.Background()
		mockTx := db.NewMockTransactions(t)
		mockLC := db.NewMockLogStreamChunks(t)
		mockStore := NewMockStore(t)

		mockLC.On("GetActiveChunk", mock.Anything, streamID).Return(nil, nil)
		mockStore.On("WriteChunk", mock.Anything, mock.Anything, 0, []byte("data")).Return(noopRetain(), nil)

		mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
		mockTx.On("RollbackTx", mock.Anything).Return(nil)
		mockLC.On("CreateLogStreamChunk", mock.Anything, mock.Anything).
			Return(nil, errors.New("log stream chunk already exists", errors.WithErrorCode(errors.EConflict)))

		logr, _ := logger.NewForTest()
		manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreamChunks: mockLC}, nil, logr, 0)

		logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 0}
		_, err := manager.WriteLogs(ctx, logStream, 0, []byte("data"))
		require.Error(t, err)
		// EConflict would map to AlreadyExists, which the runner treats as terminal and stops logging.
		assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
	})
}

func TestStream_CompactStream_DamagedChunks(t *testing.T) {
	streamID := "stream1"

	type testCase struct {
		name          string
		chunks        []models.LogStreamChunk
		reads         map[string]string // object key -> bytes the object store returns
		expectWritten string
		expectDamaged []string // object keys expected to be logged as damaged
	}

	testCases := []testCase{
		{
			name: "chunks matching their checksums are compacted without errors",
			chunks: []models.LogStreamChunk{
				{ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0", Checksum: crc("abcd")},
				{ChunkIndex: 1, StartOffset: 4, Size: 2, ObjectStoreKey: "k1", Checksum: crc("ef")},
			},
			reads:         map[string]string{"k0": "abcd", "k1": "ef"},
			expectWritten: "abcdef",
		},
		{
			name: "a chunk object missing committed bytes is logged and compaction still completes",
			chunks: []models.LogStreamChunk{
				{ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0", Checksum: crc("abcd")},
				// Committed as "efgh", but the object was overwritten with a shorter one (the incident).
				{ChunkIndex: 1, StartOffset: 4, Size: 4, ObjectStoreKey: "k1", Checksum: crc("efgh")},
			},
			reads:         map[string]string{"k0": "abcd", "k1": "ef"},
			expectWritten: "abcdef",
			expectDamaged: []string{"k1"},
		},
		{
			name: "a chunk object with different bytes is logged",
			chunks: []models.LogStreamChunk{
				{ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0", Checksum: crc("abcd")},
			},
			reads:         map[string]string{"k0": "abXd"},
			expectWritten: "abXd",
			expectDamaged: []string{"k0"},
		},
		{
			name: "a chunk without a known checksum is not checksum-verified",
			chunks: []models.LogStreamChunk{
				{ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
			},
			reads:         map[string]string{"k0": "abXd"},
			expectWritten: "abXd",
		},
		{
			name: "a chunk without a known checksum is still checked for missing bytes",
			chunks: []models.LogStreamChunk{
				{ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
				{ChunkIndex: 1, StartOffset: 4, Size: 2, ObjectStoreKey: "k1"},
			},
			reads:         map[string]string{"k0": "ab", "k1": "ef"},
			expectWritten: "abef",
			expectDamaged: []string{"k0"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			mockLC := db.NewMockLogStreamChunks(t)
			mockLS := db.NewMockLogStreams(t)
			mockStore := NewMockStore(t)
			mockTx := db.NewMockTransactions(t)

			size := 0
			for _, c := range test.chunks {
				size += c.Size
			}
			logStream := &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: size, Completed: true}

			mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, size).Return(test.chunks, nil)
			for _, c := range test.chunks {
				mockStore.On("ReadRange", mock.Anything, c.ObjectStoreKey, 0, c.Size).Return(rc(test.reads[c.ObjectStoreKey]), nil)
			}

			var written string
			mockStore.On("WriteConsolidated", mock.Anything, consolidatedObjectKey(streamID), mock.Anything).
				Run(func(args mock.Arguments) {
					b, _ := io.ReadAll(args.Get(2).(io.Reader))
					written = string(b)
				}).Return(noopRetain(), nil)

			mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
			mockTx.On("RollbackTx", mock.Anything).Return(nil)
			mockTx.On("CommitTx", mock.Anything).Return(nil)
			mockLS.On("UpdateLogStream", mock.Anything, mock.Anything).Return(echoStream)

			logr, logs := logger.NewForTest()
			manager := New(mockStore, &db.Client{Transactions: mockTx, LogStreams: mockLS, LogStreamChunks: mockLC}, nil, logr, 0)

			err := manager.CompactStream(ctx, logStream)
			require.NoError(t, err)
			assertSavedCompacted(t, mockLS, logStream) // a damaged chunk must not stop compaction
			assert.Equal(t, test.expectWritten, written)

			damaged := logs.FilterMessageSnippet("is damaged").All()
			require.Len(t, damaged, len(test.expectDamaged))
			for i, key := range test.expectDamaged {
				assert.Contains(t, damaged[i].Message, key)
			}
		})
	}
}

func TestReadLogs_ShortChunkObject(t *testing.T) {
	streamID := "stream1"
	ctx := context.Background()

	// Chunk 0 is committed with 4 bytes, but its object holds only 2.
	chunks := []models.LogStreamChunk{
		{LogStreamID: streamID, ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
		{LogStreamID: streamID, ChunkIndex: 1, StartOffset: 4, Size: 4, ObjectStoreKey: "k1"},
	}

	mockLC := db.NewMockLogStreamChunks(t)
	mockStore := NewMockStore(t)
	mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, 8).Return(chunks, nil)
	mockStore.On("ReadRange", mock.Anything, "k0", 0, 4).Return(rc("ab"), nil)

	logr, _ := logger.NewForTest()
	manager := New(mockStore, &db.Client{LogStreamChunks: mockLC}, nil, logr, 0)

	reader, err := manager.ReadLogs(ctx, &models.LogStream{Metadata: models.ResourceMetadata{ID: streamID}, Size: 8}, 0, 8)
	require.NoError(t, err)
	defer reader.Close()

	got, err := io.ReadAll(reader)
	require.Error(t, err)
	assert.Equal(t, errors.EDataLoss, errors.ErrorCode(err))
	// Chunk 1's bytes must not be returned at chunk 0's missing offsets.
	assert.Equal(t, "ab", string(got))
	mockStore.AssertNotCalled(t, "ReadRange", mock.Anything, "k1", mock.Anything, mock.Anything)
}

func TestReadLogs_ShortChunkObjectFailsLiveSubscription(t *testing.T) {
	streamID := "stream1"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mockLS := db.NewMockLogStreams(t)
	mockLC := db.NewMockLogStreamChunks(t)
	mockStore := NewMockStore(t)

	mockLS.On("GetLogStreamByID", mock.Anything, streamID).Return(&models.LogStream{
		Metadata: models.ResourceMetadata{ID: streamID}, Size: 4,
	}, nil)
	mockLC.On("GetOverlappingChunks", mock.Anything, streamID, 0, mock.Anything).Return([]models.LogStreamChunk{
		{LogStreamID: streamID, ChunkIndex: 0, StartOffset: 0, Size: 4, ObjectStoreKey: "k0"},
	}, nil)
	mockStore.On("ReadRange", mock.Anything, "k0", 0, 4).Return(rc("ab"), nil)

	logr, _ := logger.NewForTest()
	manager := New(mockStore, &db.Client{LogStreams: mockLS, LogStreamChunks: mockLC}, events.NewEventManager(nil, logr), logr, 0)

	channel, err := manager.Subscribe(ctx, &SubscriptionOptions{LogStreamID: streamID, LastSeenLogSize: ptr.Int(0)})
	require.NoError(t, err)

	select {
	case event := <-channel:
		require.Error(t, event.Err)
		// Lost data is permanent, so the client is told not to reconnect.
		assert.Equal(t, errors.EDataLoss, errors.ErrorCode(event.Err))
	case <-ctx.Done():
		t.Fatal("expected a failure event")
	}
}

func TestSubscribe_StreamNotFound(t *testing.T) {
	mockLS := db.NewMockLogStreams(t)
	mockLS.On("GetLogStreamByID", mock.Anything, "missing").Return(nil, nil)

	logr, _ := logger.NewForTest()
	manager := New(nil, &db.Client{LogStreams: mockLS}, nil, logr, 0)

	_, err := manager.Subscribe(context.Background(), &SubscriptionOptions{LogStreamID: "missing"})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
}

func TestPlanChunkWrites(t *testing.T) {
	streamID := "stream1"
	const maxChunkSize = 4

	type chunkRow struct {
		index       int
		startOffset int
		size        int
		sealed      bool
		data        string // the chunk's full content, for its checksum; "" when the checksum is unknown
	}
	type write struct {
		chunkIndex int
		byteOffset int
		data       string
		newChunk   bool
	}

	tests := []struct {
		name          string
		activeChunk   *models.LogStreamChunk
		streamSize    int
		data          string
		expectWrites  []write
		expectCreates []chunkRow
		expectUpdates []chunkRow
		legacyChunk0  bool
	}{
		{
			name:          "first write to an empty stream",
			data:          "ab",
			expectWrites:  []write{{chunkIndex: 0, byteOffset: 0, data: "ab", newChunk: true}},
			expectCreates: []chunkRow{{index: 0, size: 2, data: "ab"}},
		},
		{
			name:          "a write that fills a chunk seals it",
			data:          "abcd",
			expectWrites:  []write{{chunkIndex: 0, byteOffset: 0, data: "abcd", newChunk: true}},
			expectCreates: []chunkRow{{index: 0, size: 4, sealed: true, data: "abcd"}},
		},
		{
			name: "a write spanning chunks seals each full one",
			data: "abcdefghij",
			expectWrites: []write{
				{chunkIndex: 0, byteOffset: 0, data: "abcd", newChunk: true},
				{chunkIndex: 1, byteOffset: 0, data: "efgh", newChunk: true},
				{chunkIndex: 2, byteOffset: 0, data: "ij", newChunk: true},
			},
			expectCreates: []chunkRow{
				{index: 0, size: 4, sealed: true, data: "abcd"},
				{index: 1, startOffset: 4, size: 4, sealed: true, data: "efgh"},
				{index: 2, startOffset: 8, size: 2, data: "ij"},
			},
		},
		{
			name: "an unsealed active chunk is filled first",
			activeChunk: &models.LogStreamChunk{
				Metadata: models.ResourceMetadata{ID: "c1"}, ChunkIndex: 1, StartOffset: 4, Size: 2,
				ObjectStoreKey: "active", Checksum: crc("ef"),
			},
			streamSize: 6,
			data:       "ghij",
			expectWrites: []write{
				{chunkIndex: 1, byteOffset: 2, data: "gh"},
				{chunkIndex: 2, byteOffset: 0, data: "ij", newChunk: true},
			},
			expectUpdates: []chunkRow{{index: 1, startOffset: 4, size: 4, sealed: true, data: "efgh"}},
			expectCreates: []chunkRow{{index: 2, startOffset: 8, size: 2, data: "ij"}},
		},
		{
			name: "an active chunk with an unknown checksum keeps it unknown",
			activeChunk: &models.LogStreamChunk{
				Metadata: models.ResourceMetadata{ID: "c0"}, ChunkIndex: 0, StartOffset: 0, Size: 1, ObjectStoreKey: "active",
			},
			streamSize:    1,
			data:          "b",
			expectWrites:  []write{{chunkIndex: 0, byteOffset: 1, data: "b"}},
			expectUpdates: []chunkRow{{index: 0, size: 2}},
		},
		{
			name: "a sealed active chunk starts the next chunk",
			activeChunk: &models.LogStreamChunk{
				Metadata: models.ResourceMetadata{ID: "c0"}, ChunkIndex: 0, StartOffset: 0, Size: 4,
				ObjectStoreKey: "sealed", Sealed: true,
			},
			streamSize:    4,
			data:          "ef",
			expectWrites:  []write{{chunkIndex: 1, byteOffset: 0, data: "ef", newChunk: true}},
			expectCreates: []chunkRow{{index: 1, startOffset: 4, size: 2, data: "ef"}},
		},
		{
			name:          "a legacy stream adopts its object as sealed chunk 0",
			streamSize:    10,
			data:          "kl",
			legacyChunk0:  true,
			expectWrites:  []write{{chunkIndex: 1, byteOffset: 0, data: "kl", newChunk: true}},
			expectCreates: []chunkRow{{index: 1, startOffset: 10, size: 2, data: "kl"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var activeCopy models.LogStreamChunk
			if test.activeChunk != nil {
				activeCopy = *test.activeChunk
			}

			plan := planChunkWrites(streamID, test.activeChunk, test.streamSize, []byte(test.data), maxChunkSize)

			if test.activeChunk != nil {
				assert.Equal(t, activeCopy, *test.activeChunk, "the active chunk must not be modified")
			}

			creates := plan.creates
			if test.legacyChunk0 {
				require.NotEmpty(t, creates)
				assert.Equal(t, &models.LogStreamChunk{
					LogStreamID: streamID, ChunkIndex: 0, StartOffset: 0, Size: test.streamSize,
					ObjectStoreKey: consolidatedObjectKey(streamID), Sealed: true,
				}, creates[0])
				creates = creates[1:]
			}

			keysByIndex := map[int]string{}
			checkRows := func(expected []chunkRow, actual []*models.LogStreamChunk) {
				require.Len(t, actual, len(expected))
				for i, e := range expected {
					a := actual[i]
					assert.Equal(t, e.index, a.ChunkIndex)
					assert.Equal(t, e.startOffset, a.StartOffset)
					assert.Equal(t, e.size, a.Size)
					assert.Equal(t, e.sealed, a.Sealed)
					assert.Nil(t, a.LeaseExpiresAt)
					if e.data == "" {
						assert.Nil(t, a.Checksum)
					} else {
						assert.Equal(t, crc(e.data), a.Checksum)
					}
					keysByIndex[a.ChunkIndex] = a.ObjectStoreKey
				}
			}
			checkRows(test.expectCreates, creates)
			checkRows(test.expectUpdates, plan.updates)

			require.Len(t, plan.objectWrites, len(test.expectWrites))
			for i, e := range test.expectWrites {
				w := plan.objectWrites[i]
				assert.Equal(t, keysByIndex[e.chunkIndex], w.key)
				assert.Equal(t, e.byteOffset, w.byteOffset)
				assert.Equal(t, e.data, string(w.data))
				assert.Equal(t, e.newChunk, w.newChunk)
			}
		})
	}
}

func TestStream_Subscribe_NegativeLastSeenLogSize(t *testing.T) {
	logr, _ := logger.NewForTest()
	// No DB or event expectations: the request is rejected before anything is read or subscribed.
	manager := New(NewMockStore(t), &db.Client{}, nil, logr, 0)

	channel, err := manager.Subscribe(context.Background(), &SubscriptionOptions{LogStreamID: "stream1", LastSeenLogSize: ptr.Int(-1)})
	require.Error(t, err)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	assert.Nil(t, channel)
}
