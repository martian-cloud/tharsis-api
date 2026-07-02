package joblogger

import (
	"bytes"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testJobID = "job-1"

// newTestLogger builds a *jobLogger backed by a real (file) LogBuffer and the given mock client,
// without starting the periodic sender. The buffer's temp file is removed on cleanup.
func newTestLogger(t *testing.T, client jobclient.Client) *jobLogger {
	t.Helper()
	logr, _ := logger.NewForTest()
	l, err := NewLogger(testJobID, client, logr)
	require.NoError(t, err)
	jl := l.(*jobLogger)
	t.Cleanup(jl.buffer.Close)
	return jl
}

// newTestLoggerWithLogs additionally returns the observed log sink for asserting on warn/error output.
func newTestLoggerWithLogs(t *testing.T, client jobclient.Client) (*jobLogger, func(level zapcore.Level, snippet string) int) {
	t.Helper()
	logr, observed := logger.NewForTest()
	l, err := NewLogger(testJobID, client, logr)
	require.NoError(t, err)
	jl := l.(*jobLogger)
	t.Cleanup(jl.buffer.Close)

	count := func(level zapcore.Level, snippet string) int {
		n := 0
		for _, e := range observed.All() {
			if e.Level == level && bytes.Contains([]byte(e.Message), []byte(snippet)) {
				n++
			}
		}
		return n
	}
	return jl, count
}

func TestSendPatch(t *testing.T) {
	t.Run("sends buffered content at offset 0 and advances bytesSent", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, []byte("hello")).Return(nil).Once()

		jl := newTestLogger(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		require.NoError(t, jl.sendPatch())
		assert.Equal(t, 5, jl.bytesSent)
	})

	t.Run("no buffered data is a no-op that does not call the server", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		jl := newTestLogger(t, client)

		require.NoError(t, jl.sendPatch())
		assert.Equal(t, 0, jl.bytesSent)
		client.AssertNotCalled(t, "SaveJobLogs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a single send is bounded by maxBytesPerPatch", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, []byte("0123")).Return(nil).Once()

		jl := newTestLogger(t, client)
		jl.maxBytesPerPatch = 4
		_, err := jl.buffer.Write([]byte("0123456789"))
		require.NoError(t, err)

		require.NoError(t, jl.sendPatch())
		// Only the first maxBytesPerPatch bytes were sent; the rest stay buffered.
		assert.Equal(t, 4, jl.bytesSent)
	})

	t.Run("error from the server leaves bytesSent unchanged", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(status.Error(codes.Internal, "boom")).Once()

		jl := newTestLogger(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		require.Error(t, jl.sendPatch())
		assert.Equal(t, 0, jl.bytesSent)
	})
}

func TestFlush(t *testing.T) {
	t.Run("ships the whole buffer in maxBytesPerPatch-sized patches", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, []byte("0123")).Return(nil).Once()
		client.On("SaveJobLogs", mock.Anything, testJobID, 4, []byte("4567")).Return(nil).Once()
		client.On("SaveJobLogs", mock.Anything, testJobID, 8, []byte("89")).Return(nil).Once()

		jl := newTestLogger(t, client)
		jl.maxBytesPerPatch = 4
		_, err := jl.buffer.Write([]byte("0123456789"))
		require.NoError(t, err)

		jl.Flush()
		assert.Equal(t, 10, jl.bytesSent)
		assert.False(t, jl.anyLogsToSend())
	})

	t.Run("is best-effort: a non-retryable error is logged and stops without panicking", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(status.Error(codes.Internal, "boom")).Once()

		jl, countLogs := newTestLoggerWithLogs(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		jl.Flush() // must not panic
		assert.Equal(t, 0, jl.bytesSent)
		assert.Equal(t, 1, countLogs(zapcore.ErrorLevel, "Failed to flush job logs"))
	})

	t.Run("nothing buffered sends nothing", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		jl := newTestLogger(t, client)
		jl.Flush()
		client.AssertNotCalled(t, "SaveJobLogs", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestSendPatchWithRetry(t *testing.T) {
	// Shrink backoff so the retry/exhaustion paths run in milliseconds.
	restore := shrinkBackoff(t)
	defer restore()

	t.Run("retries a transient error then succeeds", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(status.Error(codes.Unavailable, "try later")).Once()
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(nil).Once()

		jl := newTestLogger(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		require.NoError(t, jl.sendPatchWithRetry())
		assert.Equal(t, 5, jl.bytesSent)
		client.AssertNumberOfCalls(t, "SaveJobLogs", 2)
	})

	t.Run("gives up after flushMaxAttempts on a persistent transient error", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(status.Error(codes.Unavailable, "down"))

		jl := newTestLogger(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		require.Error(t, jl.sendPatchWithRetry())
		assert.Equal(t, 0, jl.bytesSent)
		client.AssertNumberOfCalls(t, "SaveJobLogs", flushMaxAttempts)
	})

	t.Run("does not retry a non-retryable error", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(status.Error(codes.Internal, "boom")).Once()

		jl := newTestLogger(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		require.Error(t, jl.sendPatchWithRetry())
		client.AssertNumberOfCalls(t, "SaveJobLogs", 1)
	})

	t.Run("returns a non-retryable rejection as-is without retrying", func(t *testing.T) {
		// The size-limit cap and write conflicts are non-retryable; sendPatchWithRetry returns them
		// unchanged for the caller (Flush/run) to handle via handleTerminalSendError.
		client := jobclient.NewMockClient(t)
		client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
			Return(status.Error(codes.InvalidArgument, logSizeLimitReachedMarker+" (1024 bytes)")).Once()

		jl := newTestLogger(t, client)
		_, err := jl.buffer.Write([]byte("hello"))
		require.NoError(t, err)

		require.Error(t, jl.sendPatchWithRetry())
		assert.False(t, jl.sendingStopped) // latching happens in handleTerminalSendError, not here
		client.AssertNumberOfCalls(t, "SaveJobLogs", 1)
	})
}

func TestFlushStopsWhenLogLimitReached(t *testing.T) {
	restore := shrinkBackoff(t)
	defer restore()

	client := jobclient.NewMockClient(t)
	client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
		Return(status.Error(codes.InvalidArgument, logSizeLimitReachedMarker+" (1024 bytes)")).Once()

	jl, countLogs := newTestLoggerWithLogs(t, client)
	_, err := jl.buffer.Write([]byte("hello"))
	require.NoError(t, err)

	jl.Flush()
	// The cap latches sendingStopped, anyLogsToSend returns false, and the loop exits after one send.
	assert.True(t, jl.sendingStopped)
	client.AssertNumberOfCalls(t, "SaveJobLogs", 1)
	assert.Equal(t, 1, countLogs(zapcore.WarnLevel, "reached its log size limit"))
}

func TestFlushStopsOnConflict(t *testing.T) {
	restore := shrinkBackoff(t)
	defer restore()

	client := jobclient.NewMockClient(t)
	// A write conflict (EConflict -> gRPC AlreadyExists) is terminal: resending the same offset keeps
	// failing, so Flush must stop after one attempt rather than loop forever shipping nothing.
	client.On("SaveJobLogs", mock.Anything, testJobID, 0, mock.Anything).
		Return(status.Error(codes.AlreadyExists, "log write at offset 0 conflicts with already-stored data")).Once()

	jl, countLogs := newTestLoggerWithLogs(t, client)
	_, err := jl.buffer.Write([]byte("hello"))
	require.NoError(t, err)

	jl.Flush()
	assert.True(t, jl.sendingStopped)
	client.AssertNumberOfCalls(t, "SaveJobLogs", 1)
	assert.Equal(t, 1, countLogs(zapcore.WarnLevel, "log write conflict"))
}

func TestIsRetryableSendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"unavailable", status.Error(codes.Unavailable, ""), true},
		{"aborted", status.Error(codes.Aborted, ""), true},
		{"deadline exceeded", status.Error(codes.DeadlineExceeded, ""), true},
		{"internal", status.Error(codes.Internal, ""), false},
		{"invalid argument", status.Error(codes.InvalidArgument, ""), false},
		{"not found", status.Error(codes.NotFound, ""), false},
		{"nil error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetryableSendError(tt.err))
		})
	}
}

func TestHandleTerminalSendError(t *testing.T) {
	t.Run("a transient error is not terminal", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		jl := newTestLogger(t, client)
		assert.False(t, jl.handleTerminalSendError(status.Error(codes.Unavailable, "")))
		assert.False(t, jl.sendingStopped)
	})

	t.Run("an InvalidArgument without the cap marker is not terminal", func(t *testing.T) {
		// EInvalid offset/gap rejections map to InvalidArgument but lack the cap message marker; they
		// must not stop sending.
		client := jobclient.NewMockClient(t)
		jl := newTestLogger(t, client)
		assert.False(t, jl.handleTerminalSendError(status.Error(codes.InvalidArgument, "start offset is past the end of the stream")))
		assert.False(t, jl.sendingStopped)
	})

	t.Run("an InvalidArgument carrying the cap marker stops sending and warns exactly once", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		jl, countLogs := newTestLoggerWithLogs(t, client)

		capErr := status.Error(codes.InvalidArgument, logSizeLimitReachedMarker+" (1024 bytes)")
		assert.True(t, jl.handleTerminalSendError(capErr))
		assert.True(t, jl.sendingStopped)
		// Second call still reports terminal but does not warn again.
		assert.True(t, jl.handleTerminalSendError(capErr))
		assert.Equal(t, 1, countLogs(zapcore.WarnLevel, "reached its log size limit"))
	})

	t.Run("an AlreadyExists conflict stops sending and warns exactly once", func(t *testing.T) {
		client := jobclient.NewMockClient(t)
		jl, countLogs := newTestLoggerWithLogs(t, client)

		conflictErr := status.Error(codes.AlreadyExists, "log write at offset 0 conflicts with already-stored data")
		assert.True(t, jl.handleTerminalSendError(conflictErr))
		assert.True(t, jl.sendingStopped)
		assert.True(t, jl.handleTerminalSendError(conflictErr))
		assert.Equal(t, 1, countLogs(zapcore.WarnLevel, "log write conflict"))
	})
}

func TestAnyLogsToSend(t *testing.T) {
	client := jobclient.NewMockClient(t)
	jl := newTestLogger(t, client)

	// Empty buffer: nothing to send.
	assert.False(t, jl.anyLogsToSend())

	_, err := jl.buffer.Write([]byte("data"))
	require.NoError(t, err)
	assert.True(t, jl.anyLogsToSend())

	// Simulate everything already sent.
	jl.bytesSent = 4
	assert.False(t, jl.anyLogsToSend())

	// Even with unsent bytes, a terminal rejection stops sending.
	jl.bytesSent = 0
	jl.sendingStopped = true
	assert.False(t, jl.anyLogsToSend())
}

func TestCloseFlushesAndIsIdempotent(t *testing.T) {
	client := jobclient.NewMockClient(t)
	client.On("SaveJobLogs", mock.Anything, testJobID, 0, []byte("data")).Return(nil).Once()

	jl := newTestLogger(t, client)
	jl.Start() // launches the periodic sender so finish() can signal it
	_, err := jl.buffer.Write([]byte("data"))
	require.NoError(t, err)

	jl.Close()
	assert.Equal(t, 4, jl.bytesSent)

	// A second Close must not flush again (closeOnce) and must not panic on the closed buffer.
	jl.Close()
	client.AssertNumberOfCalls(t, "SaveJobLogs", 1)
}

// TestSendPatchSerialization runs many concurrent senders against one logger and asserts (under -race)
// that the serialized sendPatch never ships overlapping or duplicate byte ranges: the sent segments
// tile [0, total) exactly once.
func TestSendPatchSerialization(t *testing.T) {
	const total = 10000

	type segment struct{ off, n int }
	var (
		mu   sync.Mutex
		segs []segment
	)

	client := jobclient.NewMockClient(t)
	client.On("SaveJobLogs", mock.Anything, testJobID, mock.AnythingOfType("int"), mock.AnythingOfType("[]uint8")).
		Return(nil).
		Run(func(args mock.Arguments) {
			off := args.Int(2)
			buf := args.Get(3).([]byte)
			mu.Lock()
			segs = append(segs, segment{off: off, n: len(buf)})
			mu.Unlock()
		})

	jl := newTestLogger(t, client)
	jl.maxBytesPerPatch = 1000
	_, err := jl.buffer.Write(bytes.Repeat([]byte("x"), total))
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jl.anyLogsToSend() {
				require.NoError(t, jl.sendPatch())
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, total, jl.bytesSent)

	sort.Slice(segs, func(i, j int) bool { return segs[i].off < segs[j].off })
	expectedOffset := 0
	for _, s := range segs {
		assert.Equal(t, expectedOffset, s.off, "sent segments must be contiguous with no gaps/overlaps")
		expectedOffset += s.n
	}
	assert.Equal(t, total, expectedOffset, "sent segments must cover the whole buffer exactly once")
}

// shrinkBackoff temporarily reduces the retry backoff so retry tests run fast. Returns a restore func.
func shrinkBackoff(t *testing.T) func() {
	t.Helper()
	origInitial, origMax := flushInitialBackoff, flushMaxBackoff
	flushInitialBackoff = time.Millisecond
	flushMaxBackoff = 2 * time.Millisecond
	return func() {
		flushInitialBackoff = origInitial
		flushMaxBackoff = origMax
	}
}
