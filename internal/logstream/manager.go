// Package logstream provides functionality for saving and retrieving logs
package logstream

//go:generate go tool mockery --name Manager --inpackage --case underscore

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	logEventChunkSizeBytes = 1024 * 1024 // 1MB

	// MaxLogPatchSizeBytes bounds a single WriteLogs patch and a single ReadLogs range to 1 MiB. The
	// runner already self-limits each log patch it sends (see joblogger.defaultMaxBytesPerPatch) and
	// callers page reads in 1 MiB windows, so this rejects only non-conforming clients and keeps any
	// one request's memory/object-store cost bounded. It applies to the public WriteLogs/ReadLogs
	// entry points only; the manager's own internal reads (live-subscription windows, overlap
	// verification, whole-stream compaction) are not subject to it.
	MaxLogPatchSizeBytes = 1024 * 1024 // 1 MiB

	// DefaultMaxChunkSize is the size new chunks are filled to before sealing and rolling over.
	// Appending into the active chunk is a read-modify-write of the whole chunk object (object stores
	// have no in-place append), so this bounds how much data each append downloads and re-uploads:
	// 128 KiB keeps that cost small for the many-small-patches pattern of terraform logs. It is a
	// manager-level fill granularity, not a per-stream value: it bounds only how newly written data is
	// split into chunks, so a legacy chunk adopted whole can be larger.
	DefaultMaxChunkSize = 128 * 1024 // 128 KiB

	// LogSizeLimitReachedMsg is the stable marker embedded in the gRPC status message of the
	// log-size-cap rejection (ETooLarge) returned by WriteLogs. It is a wire-contract between this
	// server and the job-log runner, which matches on it (as a substring) to tell the cap apart from
	// other InvalidArgument rejections. The runner keeps its own copy of this string; keep them in
	// sync (see jobLogger.handleLogLimitReached).
	LogSizeLimitReachedMsg = "log size limit reached"
)

// SubscriptionOptions includes options for setting up a log event subscription
type SubscriptionOptions struct {
	LastSeenLogSize *int
	LogStreamID     string
}

// LogEventData contains the data for a log event.
type LogEventData struct {
	Offset int
	Logs   string
}

// LogEvent represents a log stream event
type LogEvent struct {
	Size      int
	Completed bool
	Truncated bool
	Data      *LogEventData
}

// Manager interface encapsulates the logic for saving and retrieving logs
type Manager interface {
	// WriteLogs appends buffer to the stream at startOffset and returns the updated stream. The caller
	// passes the already-loaded log stream (which it has from its own authorization lookup) to avoid a
	// redundant fetch.
	WriteLogs(ctx context.Context, logStream *models.LogStream, startOffset int, buffer []byte) (*models.LogStream, error)
	// ReadLogs returns a reader that streams [startOffset, startOffset+limit) of the log stream.
	// The data is streamed chunk-by-chunk; the full range is never held in memory at once. The
	// caller passes the already-loaded log stream (which it has from its own authorization lookup)
	// to avoid a redundant fetch. The caller must Close the returned reader.
	ReadLogs(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error)
	Subscribe(ctx context.Context, options *SubscriptionOptions) (<-chan *LogEvent, error)
	// CompactStream consolidates a completed stream's chunk data into the single static-key object
	// and marks the stream compacted. Safe to retry; chunk rows/objects are left in place.
	CompactStream(ctx context.Context, logStream *models.LogStream) error
}

type stream struct {
	store           Store
	dbClient        *db.Client
	eventManager    *events.EventManager
	logger          logger.Logger
	maxLogSizeBytes int
}

// New creates an instance of the Manager interface. maxLogSizeBytes is the server-authoritative
// maximum total size of a log stream; writes past it are truncated. A value <= 0 disables the cap.
// New writes are split into chunks of DefaultMaxChunkSize; that fill size is not stored per stream,
// so changing it just affects the chunks active streams write from then on.
func New(store Store, dbClient *db.Client, eventManager *events.EventManager, logger logger.Logger, maxLogSizeBytes int) Manager {
	return &stream{
		store:           store,
		dbClient:        dbClient,
		eventManager:    eventManager,
		logger:          logger,
		maxLogSizeBytes: maxLogSizeBytes,
	}
}

// objectWrite describes a single chunk-object IO operation to perform before the metadata commit.
type objectWrite struct {
	key        string
	byteOffset int
	data       []byte
}

// openChunk tracks the chunk currently being filled while planning a write.
type openChunk struct {
	key         string
	index       int
	startOffset int
	size        int
	existing    *models.LogStreamChunk // non-nil when filling an already-persisted chunk row
}

// WriteLogs appends a chunk of logs to the stream.
//
// Object storage is written FIRST, then the DB row changes are committed in a transaction. This is
// safe because writes are append-only and idempotent on resend, and reads only expose bytes up to
// LogStream.Size: any object bytes written beyond the committed size are invisible until a later DB
// update references them, and a retry re-writes the same (overwrite-safe) chunk object.
func (s *stream) WriteLogs(ctx context.Context, logStream *models.LogStream, startOffset int, buffer []byte) (*models.LogStream, error) {
	if logStream == nil {
		return nil, errors.New("log stream cannot be nil", errors.WithErrorCode(errors.EInvalid))
	}

	if startOffset < 0 {
		return nil, errors.New("offset cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}

	if len(buffer) > MaxLogPatchSizeBytes {
		return nil, errors.New("log write of %d bytes exceeds the maximum patch size of %d bytes",
			len(buffer), MaxLogPatchSizeBytes, errors.WithErrorCode(errors.EInvalid))
	}

	logStreamID := logStream.Metadata.ID

	// A completed stream is terminal: its Completed flag is set only when the job reaches a final
	// status, which the runner reports only after it has flushed all remaining logs. Any write after
	// that point is anomalous, so reject it as a conflict rather than reopening the stream (which
	// would clear Compacted and resurrect a stream compaction already finalized). EConflict maps to a
	// terminal rejection the runner stops on, instead of retrying forever. This runs before the
	// size/offset checks so a post-completion write always gets this clear reason.
	if logStream.Completed {
		return nil, errors.New("cannot write to completed log stream %s", logStreamID,
			errors.WithErrorCode(errors.EConflict))
	}

	// Enforce the server-authoritative maximum log size. Once the cap is reached we reject the write
	// with ETooLarge so the runner is told to stop streaming (it warns and ceases sending) rather
	// than silently dropping ever-higher offsets. This must run BEFORE the offset/gap validation
	// below so late or in-flight patches arriving after the cap get the clear cap error instead of a
	// confusing "offset past end of stream" gap rejection.
	if s.maxLogSizeBytes > 0 && logStream.Size >= s.maxLogSizeBytes {
		return logStream, errors.New("%s (%d bytes)", LogSizeLimitReachedMsg, s.maxLogSizeBytes,
			errors.WithErrorCode(errors.ETooLarge))
	}

	// Validate the incoming offset against the current stream size (append-only semantics).
	if startOffset > logStream.Size {
		return nil, errors.New("start offset %d is past the end of the stream (size %d)",
			startOffset, logStream.Size, errors.WithErrorCode(errors.EInvalid))
	}

	// A resend overlaps already-stored bytes. The overlapping prefix must byte-for-byte match what is
	// stored: if it matches we skip it (idempotent resend), if not the caller is sending conflicting
	// data and we reject with a conflict rather than silently corrupting the log.
	skip := logStream.Size - startOffset
	if skip > 0 {
		overlap := buffer
		if skip < len(buffer) {
			overlap = buffer[:skip]
		}
		if err := s.verifyOverlap(ctx, logStream, startOffset, overlap); err != nil {
			return nil, err
		}
	}
	if skip >= len(buffer) {
		// All incoming bytes are already stored (and verified to match); idempotent no-op.
		return logStream, nil
	}
	effectiveBuffer := buffer[skip:]

	// Truncate at the server-authoritative maximum log size. We already returned above if the stream
	// was at/over the cap, so maxLogSizeBytes-Size is strictly positive here. We persist the prefix
	// that fits and mark the stream truncated, then return ETooLarge after the commit so the runner
	// keeps the final bytes up to the cap yet is still told to stop streaming.
	reachedCap := false
	if s.maxLogSizeBytes > 0 && logStream.Size+len(effectiveBuffer) > s.maxLogSizeBytes {
		effectiveBuffer = effectiveBuffer[:s.maxLogSizeBytes-logStream.Size]
		reachedCap = true
	}

	// The fill size is a fixed default, not stored per stream: it bounds only how new data is split
	// into chunks.
	maxChunkSize := DefaultMaxChunkSize

	activeChunk, err := s.dbClient.LogStreamChunks.GetActiveChunk(ctx, logStreamID)
	if err != nil {
		return nil, err
	}

	var (
		objectWrites []objectWrite
		chunkCreates []*models.LogStreamChunk
		chunkUpdates []*models.LogStreamChunk
	)

	// Determine the chunk we begin appending into. The first byte always lands at absolute offset
	// logStream.Size.
	var cur *openChunk
	switch {
	case activeChunk != nil && !activeChunk.Sealed:
		cur = &openChunk{
			key:         activeChunk.ObjectKey,
			index:       activeChunk.ChunkIndex,
			startOffset: activeChunk.StartOffset,
			size:        activeChunk.Size,
			existing:    activeChunk,
		}
	case activeChunk != nil:
		// The tail chunk is sealed; start the next chunk after it.
		cur = &openChunk{
			key:         chunkObjectKey(logStreamID),
			index:       activeChunk.ChunkIndex + 1,
			startOffset: activeChunk.StartOffset + activeChunk.Size,
		}
	default:
		// No chunk rows yet.
		if logStream.Size > 0 {
			// A stream with bytes but no chunks is a pre-chunking (legacy) stream whose data lives in
			// the single consolidated object. It is reopened here: adopt that object as a sealed chunk 0
			// and write new data into chunk 1.
			//
			// Chunk 0 spans the whole legacy object, so its Size can exceed the fill size (maxChunkSize).
			// That is expected: the fill size bounds only how new writes are split into chunks; it is not
			// a per-chunk invariant. The legacy single-file writer always truncated the object to exactly
			// LogStream.Size on every write, so the object's byte length equals logStream.Size here, and
			// reads of chunk 0 never run past the object's end.
			chunkCreates = append(chunkCreates, &models.LogStreamChunk{
				LogStreamID: logStreamID,
				ChunkIndex:  0,
				StartOffset: 0,
				Size:        logStream.Size,
				ObjectKey:   consolidatedObjectKey(logStreamID),
				Sealed:      true,
			})
			cur = &openChunk{
				key:         chunkObjectKey(logStreamID),
				index:       1,
				startOffset: logStream.Size,
			}
		} else {
			cur = &openChunk{
				key:   chunkObjectKey(logStreamID),
				index: 0,
			}
		}
	}

	finalize := func(c *openChunk, sealed bool) {
		if c.existing != nil {
			c.existing.Size = c.size
			c.existing.Sealed = sealed
			chunkUpdates = append(chunkUpdates, c.existing)
			return
		}
		chunkCreates = append(chunkCreates, &models.LogStreamChunk{
			LogStreamID: logStreamID,
			ChunkIndex:  c.index,
			StartOffset: c.startOffset,
			Size:        c.size,
			ObjectKey:   c.key,
			Sealed:      sealed,
		})
	}

	// Fill the open chunk, sealing and rolling over to a new chunk whenever it reaches the max size.
	data := effectiveBuffer
	for len(data) > 0 {
		capacity := maxChunkSize - cur.size
		if capacity <= 0 {
			finalize(cur, true)
			cur = &openChunk{
				key:         chunkObjectKey(logStreamID),
				index:       cur.index + 1,
				startOffset: cur.startOffset + cur.size,
			}
			capacity = maxChunkSize
		}

		n := capacity
		if n > len(data) {
			n = len(data)
		}

		objectWrites = append(objectWrites, objectWrite{key: cur.key, byteOffset: cur.size, data: data[:n]})
		cur.size += n
		data = data[n:]
	}
	finalize(cur, cur.size >= maxChunkSize)

	// Write objects BEFORE touching the database. A failure here leaves the DB untouched.
	for i := range objectWrites {
		w := &objectWrites[i]
		if err = s.store.WriteChunk(ctx, w.key, w.byteOffset, w.data); err != nil {
			return nil, err
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	for _, c := range chunkCreates {
		if _, err = s.dbClient.LogStreamChunks.CreateLogStreamChunk(txContext, c); err != nil {
			return nil, err
		}
	}
	for _, c := range chunkUpdates {
		if _, err = s.dbClient.LogStreamChunks.UpdateLogStreamChunk(txContext, c); err != nil {
			return nil, err
		}
	}

	logStream.Size += len(effectiveBuffer)
	if reachedCap {
		logStream.Truncated = true
	}

	updatedStream, err := s.dbClient.LogStreams.UpdateLogStream(txContext, logStream)
	if err != nil {
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	if reachedCap {
		// The prefix that fits is now committed and the stream is flagged truncated; signal the cap so
		// the caller (runner) stops streaming further logs.
		return updatedStream, errors.New("%s (%d bytes)", LogSizeLimitReachedMsg, s.maxLogSizeBytes,
			errors.WithErrorCode(errors.ETooLarge))
	}

	return updatedStream, nil
}

// CompactStream consolidates a completed stream's chunk data into the single static-key object and
// marks the stream compacted. The object is written before the flag is set (object-before-DB), so a
// failure leaves Compacted=false and the next run retries (idempotent overwrite of the static key).
// Chunk rows/objects are intentionally left in place; reclaiming them is future work.
//
// The caller (the compaction scheduler) is responsible for claiming the stream first via
// LogStreams.ClaimLogStreamsForCompaction, which row-locks it with SELECT ... FOR UPDATE SKIP LOCKED
// so concurrent instances never process the same stream. CompactStream therefore assumes the passed
// stream is already claimed and does not re-claim it.
func (s *stream) CompactStream(ctx context.Context, logStream *models.LogStream) error {
	if logStream == nil {
		return errors.New("log stream cannot be nil", errors.WithErrorCode(errors.EInvalid))
	}
	if logStream.Compacted {
		return nil
	}

	// Only write an object if there is log data. A completed stream that logged nothing has no
	// chunks; reads of it return empty via readConsolidated, so just set the flag.
	if logStream.Size > 0 {
		// logStream.Compacted is false here, so readLogs streams from the chunk objects.
		reader, err := s.readLogs(ctx, logStream, 0, logStream.Size)
		if err != nil {
			return err
		}
		defer reader.Close()

		if err := s.store.WriteObject(ctx, consolidatedObjectKey(logStream.Metadata.ID), reader); err != nil {
			return err
		}
	}

	logStream.Compacted = true
	if _, err := s.dbClient.LogStreams.UpdateLogStream(ctx, logStream); err != nil {
		if errors.ErrorCode(err) == errors.EOptimisticLock {
			// A concurrent update raced with this mark. Completed streams reject further WriteLogs and
			// the SKIP LOCKED claim keeps other instances off this stream, so the consolidated object we
			// wrote is still valid. Abort cleanly without marking Compacted: reads keep using the chunk
			// path and a later run recompacts, harmlessly overwriting the same consolidated object.
			return nil
		}
		return err
	}

	return nil
}

// verifyOverlap reads the already-stored bytes at [startOffset, startOffset+len(expected)) and
// returns an EConflict error if they differ from expected. This makes idempotent resends safe: the
// overlapping portion of a re-sent write must match what was already persisted.
func (s *stream) verifyOverlap(ctx context.Context, logStream *models.LogStream, startOffset int, expected []byte) error {
	reader, err := s.readLogs(ctx, logStream, startOffset, len(expected))
	if err != nil {
		return err
	}
	defer reader.Close()

	stored, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	if !bytes.Equal(stored, expected) {
		return errors.New("log write at offset %d conflicts with already-stored data", startOffset,
			errors.WithErrorCode(errors.EConflict))
	}

	return nil
}

// ReadLogs returns a reader that streams a range of logs from the stream.
func (s *stream) ReadLogs(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error) {
	if logStream == nil {
		return nil, errors.New("log stream cannot be nil", errors.WithErrorCode(errors.EInvalid))
	}

	if limit > MaxLogPatchSizeBytes {
		return nil, errors.New("log read limit of %d bytes exceeds the maximum of %d bytes",
			limit, MaxLogPatchSizeBytes, errors.WithErrorCode(errors.EInvalid))
	}

	return s.readLogs(ctx, logStream, startOffset, limit)
}

// readLogs returns a reader over [startOffset, startOffset+limit) that streams the overlapping
// chunk objects one at a time, so the full range is never buffered in memory. Streams with no chunk
// rows fall back to the legacy single-object layout.
func (s *stream) readLogs(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error) {
	if startOffset < 0 || limit < 0 {
		return nil, errors.New("limit and offset cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}
	if limit == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}

	// A compacted stream's data lives in the single static-key object; read it directly and ignore
	// any (orphaned, not-yet-reclaimed) chunk rows.
	if logStream.Compacted {
		return s.readConsolidated(ctx, logStream, startOffset, limit)
	}

	end := startOffset + limit

	chunks, err := s.dbClient.LogStreamChunks.GetOverlappingChunks(ctx, logStream.Metadata.ID, startOffset, end)
	if err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		// No chunk rows: this is either a pre-chunking (legacy) stream or one that has not been written.
		return s.readConsolidated(ctx, logStream, startOffset, limit)
	}

	slices := make([]chunkSlice, 0, len(chunks))
	for i := range chunks {
		c := &chunks[i]

		sliceStart := max(startOffset, c.StartOffset) - c.StartOffset
		sliceEnd := min(end, c.StartOffset+c.Size) - c.StartOffset
		if sliceEnd <= sliceStart {
			continue
		}

		slices = append(slices, chunkSlice{key: c.ObjectKey, offset: sliceStart, length: sliceEnd - sliceStart})
	}

	return &chunkReader{ctx: ctx, store: s.store, slices: slices}, nil
}

// readConsolidated reads the single whole-stream object: the logstreams/{id}.txt key
// first, then the older per-job workspaces/... key, mirroring the historical fallback behavior.
func (s *stream) readConsolidated(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error) {
	logs, err := s.store.ReadRange(ctx, consolidatedObjectKey(logStream.Metadata.ID), startOffset, limit)
	if err != nil {
		if errors.ErrorCode(err) != errors.ENotFound {
			return nil, errors.Wrap(err, "failed to read log file from object store")
		}

		if logStream.JobID == nil {
			// Nothing has been written yet.
			return io.NopCloser(bytes.NewReader(nil)), nil
		}

		job, jErr := s.dbClient.Jobs.GetJobByID(ctx, *logStream.JobID)
		if jErr != nil {
			return nil, jErr
		}
		if job == nil {
			return nil, errors.New("job with ID %s not found", *logStream.JobID, errors.WithErrorCode(errors.ENotFound))
		}

		legacyLogs, lErr := s.store.ReadRange(ctx, legacyJobObjectKey(job), startOffset, limit)
		if lErr != nil {
			if errors.ErrorCode(lErr) == errors.ENotFound {
				return io.NopCloser(bytes.NewReader(nil)), nil
			}
			return nil, errors.Wrap(lErr, "failed to read log file from object store")
		}
		return legacyLogs, nil
	}

	return logs, nil
}

// chunkSlice identifies a byte range within a single chunk object.
type chunkSlice struct {
	key    string
	offset int
	length int
}

// chunkReader streams a sequence of chunk slices, opening each chunk object lazily and reading it to
// completion before moving to the next. At most one chunk object stream is open at a time, so the
// full log range is never held in memory.
type chunkReader struct {
	ctx    context.Context
	store  Store
	slices []chunkSlice
	idx    int
	cur    io.ReadCloser
}

func (r *chunkReader) Read(p []byte) (int, error) {
	for {
		if r.cur == nil {
			if r.idx >= len(r.slices) {
				return 0, io.EOF
			}
			sl := r.slices[r.idx]
			rc, err := r.store.ReadRange(r.ctx, sl.key, sl.offset, sl.length)
			if err != nil {
				return 0, err
			}
			r.cur = rc
		}

		n, err := r.cur.Read(p)
		if err == io.EOF {
			closeErr := r.cur.Close()
			r.cur = nil
			r.idx++
			if n > 0 {
				return n, nil
			}
			if closeErr != nil {
				return 0, closeErr
			}
			continue
		}
		return n, err
	}
}

func (r *chunkReader) Close() error {
	if r.cur != nil {
		err := r.cur.Close()
		r.cur = nil
		return err
	}
	return nil
}

func (s *stream) Subscribe(ctx context.Context, options *SubscriptionOptions) (<-chan *LogEvent, error) {
	logStream, err := s.dbClient.LogStreams.GetLogStreamByID(ctx, options.LogStreamID)
	if err != nil {
		return nil, err
	}

	if logStream == nil {
		return nil, fmt.Errorf("log stream not found with ID: %s", options.LogStreamID)
	}

	subscription := events.Subscription{
		Type: events.LogStreamSubscription,
		ID:   logStream.Metadata.ID,
		Actions: []events.SubscriptionAction{
			events.CreateAction,
			events.UpdateAction,
		},
	}
	subscriber := s.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *LogEvent)
	var completed bool

	go func() {
		var currentSize int

		// Defer close of outgoing channel
		defer close(outgoing)
		defer s.eventManager.Unsubscribe(subscriber)

		if options.LastSeenLogSize != nil {
			// Send all logs that were missed since last seen size
			currentSize, completed = s.sendLogstreamEvent(ctx, logStream, *options.LastSeenLogSize, logStream.Size, logStream.Completed, logStream.Truncated, outgoing)
			if completed {
				return
			}
		} else {
			currentSize = logStream.Size
		}

		// Wait for log stream updates
		for {
			event, err := subscriber.GetEvent(ctx)
			if err != nil {
				if !errors.IsContextCanceledError(err) && !errors.IsDeadlineExceededError(err) {
					s.logger.WithContextFields(ctx).Errorf("error occurred while waiting for log events: %v", err)
				}
				return
			}

			logStreamEventData, err := event.ToLogStreamEventData()
			if err != nil {
				s.logger.WithContextFields(ctx).Errorf("failed to get log stream event data in log stream subscription, log event %s: %v", event.ID, err)
				return
			}

			currentSize, completed = s.sendLogstreamEvent(ctx, logStream, currentSize, logStreamEventData.Size, logStreamEventData.Completed, logStreamEventData.Truncated, outgoing)
			if completed {
				return
			}
		}
	}()

	return outgoing, nil
}

func (s *stream) sendLogstreamEvent(ctx context.Context, logStream *models.LogStream, lastSeenLogSize int, actualLogSize int, completed bool, truncated bool, outgoing chan *LogEvent) (int, bool) {
	for lastSeenLogSize < actualLogSize {
		offset := lastSeenLogSize
		// Read logs in bounded windows. The window is small (logEventChunkSizeBytes), so reading it
		// fully into memory here is safe.
		reader, err := s.readLogs(ctx, logStream, lastSeenLogSize, logEventChunkSizeBytes)
		if err != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to read logs for log stream %s: %v", logStream.Metadata.ID, err)
			return 0, true
		}

		logs, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to read logs for log stream %s: %v", logStream.Metadata.ID, err)
			return 0, true
		}

		if len(logs) == 0 {
			// Guard against making no progress if the bytes are not yet readable.
			break
		}

		lastSeenLogSize += len(logs)

		select {
		case <-ctx.Done():
			return 0, true
		case outgoing <- &LogEvent{Size: lastSeenLogSize, Truncated: truncated, Data: &LogEventData{Offset: offset, Logs: string(logs)}}:
		}
	}

	// Return from loop if log stream has been completed because there are no more logs to process
	if completed {
		select {
		case <-ctx.Done():
			return 0, true
		case outgoing <- &LogEvent{Size: lastSeenLogSize, Completed: completed, Truncated: truncated}:
		}
	}

	return lastSeenLogSize, completed
}

// chunkObjectKey returns a fresh, unique object key for a new log stream chunk.
func chunkObjectKey(logStreamID string) string {
	return fmt.Sprintf("logstreams/%s/%s.txt", logStreamID, uuid.New().String())
}

// consolidatedObjectKey returns the single whole-stream object key for a log stream. It holds the
// full log for both pre-chunking (legacy) streams and streams consolidated by compaction.
func consolidatedObjectKey(logStreamID string) string {
	return fmt.Sprintf("logstreams/%s.txt", logStreamID)
}

// legacyJobObjectKey returns the oldest per-job log object key.
func legacyJobObjectKey(job *models.Job) string {
	return fmt.Sprintf("workspaces/%s/runs/%s/logs/%s.txt", job.WorkspaceID, job.RunID, job.Metadata.ID)
}
