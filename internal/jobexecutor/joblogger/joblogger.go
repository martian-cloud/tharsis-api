// Package joblogger is used to handle job logs
package joblogger

//go:generate go tool mockery --name Logger --inpackage --case underscore

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ansi"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	defaultMaxBytesPerPatch = 1024 * 1024 // in bytes
	defaultUpdateInterval   = 3 * time.Second

	// Flush retries a transient send error up to flushMaxAttempts times with exponential backoff
	// (capped at flushMaxBackoff). Bounded so a persistently unreachable server can't hang the
	// executor indefinitely.
	flushMaxAttempts = 5

	// logSizeLimitReachedMarker is the substring the server embeds in the SaveJobLogs cap-rejection
	// message (ETooLarge → gRPC InvalidArgument). handleTerminalSendError matches it to tell the cap
	// apart from other InvalidArgument rejections. It is a wire-contract with the server; keep it in
	// sync with logstream.LogSizeLimitReachedMsg.
	logSizeLimitReachedMarker = "log size limit reached"
)

// Backoff bounds for sendPatchWithRetry. Vars rather than consts only so tests can shrink them to
// keep the retry-exhaustion path fast; they are never reassigned at runtime.
var (
	flushInitialBackoff = 500 * time.Millisecond
	flushMaxBackoff     = 5 * time.Second
)

// Logger is an interface for logging job output
type Logger interface {
	// Close flushes the logger
	Close()
	// Infof writes an info log to the job's log output
	Infof(format string, a ...any)
	// Errorf writes an error log to the job's log output
	Errorf(format string, a ...any)
	// Warningf writes a warning log to the job's log output
	Warningf(format string, a ...any)
	// Printf writes a plain log to the job's log output
	Printf(format string, a ...any)
	// Write will append the data to the log buffer
	Write(data []byte) (n int, err error)
	// Start starts the logger
	Start()
	// Flush sends all buffered logs to the server, retrying transient errors and logging if they
	// ultimately can't be sent.
	Flush()
}

var _ Logger = (*jobLogger)(nil)

type jobLogger struct {
	sentTime         time.Time
	client           jobclient.Client
	logger           logger.Logger
	buffer           *LogBuffer
	finished         chan bool
	jobID            string
	bytesSent        int
	updateInterval   time.Duration
	maxBytesPerPatch int
	sendingStopped   bool
	lock             sync.RWMutex
	sendLock         sync.Mutex
	closeOnce        sync.Once
}

// NewLogger creates a new Logger
func NewLogger(jobID string, client jobclient.Client, logger logger.Logger) (Logger, error) {
	buffer, err := NewLogBuffer()
	if err != nil {
		return nil, err
	}

	return &jobLogger{
		jobID:            jobID,
		buffer:           buffer,
		maxBytesPerPatch: defaultMaxBytesPerPatch,
		updateInterval:   defaultUpdateInterval,
		client:           client,
		logger:           logger,
	}, nil
}

// Close flushes the logger
// Close stops the periodic sender, flushes all remaining logs, and closes the buffer. It is
// idempotent (safe to call explicitly before reporting a terminal job status and again via defer).
func (j *jobLogger) Close() {
	j.closeOnce.Do(j.finish)
}

// Infof writes an info log to the job's log output
func (j *jobLogger) Infof(format string, a ...any) {
	j.Write(fmt.Appendf(nil, ansi.Colorize(format, ansi.BoldCyan)+"\n", a...))
}

// Errorf writes an error log to the job's log output
func (j *jobLogger) Errorf(format string, a ...any) {
	j.Write(fmt.Appendf(nil, ansi.Colorize(format, ansi.BoldRed)+"\n", a...))
}

// Warningf writes a warning log to the job's log output
func (j *jobLogger) Warningf(format string, a ...any) {
	j.Write(fmt.Appendf(nil, ansi.Colorize(format, ansi.BoldYellow)+"\n", a...))
}

// Printf writes a plain log to the job's log output
func (j *jobLogger) Printf(format string, a ...any) {
	j.Write(fmt.Appendf(nil, format+"\n", a...))
}

// Write will append the data to the log buffer
func (j *jobLogger) Write(data []byte) (n int, err error) {
	j.logger.Infof("JOB OUTPUT: %s", string(data))
	return j.buffer.Write(data)
}

// nolint:unused
func (j *jobLogger) checksum() string {
	return j.buffer.Checksum()
}

// nolint:unused
func (j *jobLogger) bytesize() int {
	return j.buffer.Size()
}

func (j *jobLogger) Start() {
	// Buffered so finish() never blocks signaling completion even if run() already returned early
	// (e.g. after the server reported the log size limit was reached).
	j.finished = make(chan bool, 1)
	go j.run()
}

// Flush sends all buffered logs to the server, retrying transient send errors with bounded backoff.
// It is best-effort: if a patch ultimately can't be sent it logs and stops rather than failing the
// caller (a job shouldn't fail just because its logs couldn't be shipped). It returns once the
// buffer is fully flushed or the server-side size limit was reached. Concurrent sends from the
// periodic sender are safe: sendPatch ships each chunk at the offset it read it from, which the
// server treats as an idempotent overlap.
func (j *jobLogger) Flush() {
	for j.anyLogsToSend() {
		if err := j.sendPatchWithRetry(); err != nil {
			if j.handleTerminalSendError(err) {
				// Size limit or conflict: stop trying (already warned), no further logs will ship.
				return
			}
			j.logger.Errorf("Failed to flush job logs: %v", err)
			return
		}
	}
}

// sendPatchWithRetry sends one patch, retrying only transient errors with bounded exponential
// backoff. Non-retryable errors (including the size-limit cap and write conflicts) are returned
// to the caller as-is for terminal handling.
func (j *jobLogger) sendPatchWithRetry() error {
	return retry.Do(
		j.sendPatch,
		retry.Attempts(flushMaxAttempts),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(flushInitialBackoff),
		retry.MaxDelay(flushMaxBackoff),
		retry.LastErrorOnly(true),
		retry.RetryIf(isRetryableSendError),
		retry.OnRetry(func(n uint, err error) {
			j.logger.Warnf("Retrying log send after transient error (attempt %d/%d): %v", n+1, flushMaxAttempts, err)
		}),
	)
}

// isRetryableSendError reports whether a SaveJobLogs error is a transient server-availability
// condition worth retrying, rather than a permanent rejection or a local error.
func isRetryableSendError(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.Aborted, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}

func (j *jobLogger) finish() {
	j.finished <- true
	j.Flush()
	j.buffer.Close()
}

func (j *jobLogger) anyLogsToSend() bool {
	j.lock.RLock()
	defer j.lock.RUnlock()

	if j.sendingStopped {
		// A terminal rejection (size limit or conflict) stopped sending; don't attempt further sends.
		return false
	}

	return j.buffer.Size() != j.bytesSent
}

// handleTerminalSendError reports whether err is a permanent rejection that resending cannot fix and,
// if so, latches a flag that stops all further sends, warning once. Transient and one-off errors
// return false so sending continues.
//
//   - Size limit (ETooLarge → gRPC InvalidArgument): matched by the size-limit message marker so it is
//     not confused with a transient offset/gap rejection (EInvalid), which also maps to InvalidArgument.
//   - Write conflict (EConflict → gRPC AlreadyExists): the resent bytes disagree with what the server
//     already stored, so the runner's and server's views have diverged. Every subsequent append at this
//     offset will be rejected the same way, so we stop cleanly instead of looping forever shipping
//     nothing (and silently dropping the rest of the job's logs).
func (j *jobLogger) handleTerminalSendError(err error) bool {
	var reason string
	switch {
	case status.Code(err) == codes.InvalidArgument && strings.Contains(status.Convert(err).Message(), logSizeLimitReachedMarker):
		reason = "reached its log size limit"
	case status.Code(err) == codes.AlreadyExists:
		reason = "hit an unrecoverable log write conflict"
	default:
		return false
	}

	j.lock.Lock()
	already := j.sendingStopped
	j.sendingStopped = true
	j.lock.Unlock()

	if !already {
		j.logger.Warnf("Job %s %s; no further logs will be sent for this job", j.jobID, reason)
	}

	return true
}

func (j *jobLogger) sendPatch() error {
	// Serialize sends so the periodic run() sender and an explicit Flush() never ship a patch at the
	// same time. Without this they could both read the same offset and send the same chunk (duplicate
	// work), or interleave and advance the offset out from under each other. sendLock is separate from
	// the field lock (held only briefly below) so the in-flight SaveJobLogs call doesn't block readers
	// like anyLogsToSend.
	j.sendLock.Lock()
	defer j.sendLock.Unlock()

	j.lock.RLock()
	bytesSent := j.bytesSent
	content, err := j.buffer.Bytes(bytesSent, j.maxBytesPerPatch)
	j.lock.RUnlock()

	if err != nil {
		return err
	}

	if len(content) == 0 {
		return nil
	}

	// Use the offset captured under the lock — not a fresh read of j.bytesSent — so the upload offset
	// always matches the content read for it. Otherwise a concurrent sender advancing j.bytesSent
	// between the read and this call would ship this content at the wrong offset.
	if err := j.client.SaveJobLogs(context.Background(), j.jobID, bytesSent, content); err != nil {
		return err
	}

	j.lock.Lock()
	j.sentTime = time.Now()
	j.bytesSent = bytesSent + len(content)
	j.lock.Unlock()

	return nil
}

func (j *jobLogger) run() {
	for {
		select {
		case <-time.After(j.updateInterval):
			if err := j.sendPatch(); err != nil {
				if j.handleTerminalSendError(err) {
					// Size limit or conflict: resending is futile, so stop the periodic sender.
					return
				}
				j.logger.Errorf("Failed to send log patch: %v", err)
			}
		case <-j.finished:
			return
		}
	}
}
