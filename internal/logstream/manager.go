// Package logstream provides functionality for saving and retrieving logs
package logstream

//go:generate go tool mockery --name Manager --inpackage --case underscore

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"time"

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

	// chunkLeaseDuration is how long a writer holds an active chunk's write lease. It bounds how long
	// writes to a chunk stay blocked if the holder dies without releasing it, so it must stay well
	// within the runner's log-send retry budget (joblogger's flush retries Aborted for ~32s) or the
	// runner gives up and drops logs. Keep the two in sync.
	chunkLeaseDuration = 20 * time.Second

	// chunkWriteTimeout bounds the object store IO for one write. It must be comfortably shorter than
	// chunkLeaseDuration so an upload can never complete after the lease has expired.
	chunkWriteTimeout = 10 * time.Second

	// leaseReleaseTimeout bounds the best-effort lease release after a failed write.
	leaseReleaseTimeout = 5 * time.Second
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
	// Err is set only on the final event of a subscription that ended because of a failure rather
	// than because the stream completed or the subscriber's context ended.
	Err error
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

type manager struct {
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
	return &manager{
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
	newChunk   bool // the write is to a chunk being created, whose object ref must be linked
}

// openChunk tracks the chunk currently being filled while planning a write.
type openChunk struct {
	key         string
	index       int
	startOffset int
	size        int
	checksum    *uint32                // CRC-32 of [0, size); nil when unknown. Never written through.
	existing    *models.LogStreamChunk // non-nil when filling an already-persisted chunk row
}

// WriteLogs appends a chunk of logs to the stream.
//
// Object storage is written FIRST, then the DB row changes are committed in a transaction. This is
// safe because writes are append-only and idempotent on resend, and reads only expose bytes up to
// LogStream.Size: any object bytes written beyond the committed size are invisible until a later DB
// update references them, and a retry re-writes the same (overwrite-safe) chunk object.
//
// Appending to the active chunk rewrites its whole object (object stores have no append), so two
// concurrent writers could otherwise both read the same committed state and the slower upload would
// replace the faster one's object with a shorter or different one, losing committed bytes. To prevent
// that, the writer claims the chunk's write lease before touching the object: the claim fails unless
// the chunk is unchanged since it was read and no other writer holds an unexpired lease. Object writes
// run under a deadline well inside the lease, and the lease is released when the write commits (or,
// best effort, when it fails). New chunks are written to fresh unique keys, so they need no lease.
func (m *manager) WriteLogs(ctx context.Context, logStream *models.LogStream, startOffset int, buffer []byte) (*models.LogStream, error) {
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
	if m.maxLogSizeBytes > 0 && logStream.Size >= m.maxLogSizeBytes {
		return logStream, m.logSizeLimitError()
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
		if err := m.verifyOverlap(ctx, logStream, startOffset, overlap); err != nil {
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
	if m.maxLogSizeBytes > 0 && logStream.Size+len(effectiveBuffer) > m.maxLogSizeBytes {
		effectiveBuffer = effectiveBuffer[:m.maxLogSizeBytes-logStream.Size]
		reachedCap = true
	}

	activeChunk, err := m.dbClient.LogStreamChunks.GetActiveChunk(ctx, logStreamID)
	if err != nil {
		return nil, err
	}

	// The chunks always cover exactly [0, LogStream.Size), so if the tail chunk doesn't end where the
	// caller's stream view does, another write has committed since that view was loaded. Reject it
	// early as a conflict; the client retries with fresh state (where the resend is usually a no-op).
	if activeChunk != nil && activeChunk.StartOffset+activeChunk.Size != logStream.Size {
		return nil, errors.New("log stream %s was updated by a concurrent write", logStreamID,
			errors.WithErrorCode(errors.EOptimisticLock))
	}

	// committed is set once the DB transaction commits; until then a claimed lease is released on return.
	committed := false

	if activeChunk != nil && !activeChunk.Sealed {
		// This write rewrites the active chunk's object, so claim its write lease first.
		claimedChunk, cErr := m.dbClient.LogStreamChunks.ClaimLogStreamChunkLease(ctx, activeChunk, chunkLeaseDuration)
		if cErr != nil {
			if errors.ErrorCode(cErr) == errors.EOptimisticLock {
				return nil, errors.New("log stream %s chunk %d is being written by another request", logStreamID, activeChunk.ChunkIndex,
					errors.WithErrorCode(errors.EOptimisticLock))
			}
			return nil, cErr
		}
		activeChunk = claimedChunk

		defer func() {
			if !committed {
				m.releaseChunkLease(ctx, claimedChunk)
			}
		}()
	}

	plan := planChunkWrites(logStreamID, activeChunk, logStream.Size, effectiveBuffer, DefaultMaxChunkSize)

	// Write objects BEFORE touching the database. A failure here leaves the DB untouched.
	retainFns, err := m.writeChunkObjects(ctx, plan.objectWrites)
	if err != nil {
		return nil, err
	}

	// The caller's stream is left unchanged until the commit succeeds.
	updated := *logStream
	updated.Size += len(effectiveBuffer)
	if reachedCap {
		updated.Truncated = true
	}

	updatedStream, err := m.commitChunkWrites(ctx, plan, retainFns, &updated)
	if err != nil {
		return nil, err
	}
	committed = true

	if reachedCap {
		// The prefix that fits is now committed and the stream is flagged truncated; signal the cap so
		// the caller (runner) stops streaming further logs.
		return updatedStream, m.logSizeLimitError()
	}

	return updatedStream, nil
}

// logSizeLimitError is the ETooLarge rejection for a write past the maximum log size.
func (m *manager) logSizeLimitError() error {
	return errors.New("%s (%d bytes)", LogSizeLimitReachedMsg, m.maxLogSizeBytes, errors.WithErrorCode(errors.ETooLarge))
}

// chunkWritePlan is the object writes and chunk row changes that append data to a log stream.
type chunkWritePlan struct {
	objectWrites []objectWrite
	creates      []*models.LogStreamChunk
	updates      []*models.LogStreamChunk
}

// planChunkWrites plans appending data to a stream of streamSize bytes whose tail chunk is
// activeChunk (nil when the stream has no chunk rows). Data fills the active chunk, if it is unsealed,
// then new chunks of up to maxChunkSize bytes, each sealed once full. activeChunk is not modified.
func planChunkWrites(logStreamID string, activeChunk *models.LogStreamChunk, streamSize int, data []byte, maxChunkSize int) *chunkWritePlan {
	plan := &chunkWritePlan{}

	newChunk := func(index, startOffset int) *openChunk {
		return &openChunk{key: chunkObjectKey(logStreamID), index: index, startOffset: startOffset, checksum: new(uint32)}
	}

	// Determine the chunk we begin appending into. The first byte always lands at absolute offset
	// streamSize.
	var cur *openChunk
	switch {
	case activeChunk != nil && !activeChunk.Sealed:
		cur = &openChunk{
			key:         activeChunk.ObjectStoreKey,
			index:       activeChunk.ChunkIndex,
			startOffset: activeChunk.StartOffset,
			size:        activeChunk.Size,
			checksum:    activeChunk.Checksum,
			existing:    activeChunk,
		}
	case activeChunk != nil:
		// The tail chunk is sealed; start the next chunk after it.
		cur = newChunk(activeChunk.ChunkIndex+1, activeChunk.StartOffset+activeChunk.Size)
	case streamSize > 0:
		// A stream with bytes but no chunks is a pre-chunking (legacy) stream whose data lives in
		// the single consolidated object. It is reopened here: adopt that object as a sealed chunk 0
		// and write new data into chunk 1.
		//
		// Chunk 0 spans the whole legacy object, so its Size can exceed the fill size (maxChunkSize).
		// That is expected: the fill size bounds only how new writes are split into chunks; it is not
		// a per-chunk invariant. The legacy single-file writer always truncated the object to exactly
		// LogStream.Size on every write, so the object's byte length equals streamSize here, and
		// reads of chunk 0 never run past the object's end.
		// The adopted chunk 0 has no known checksum, so it is left nil (and is never verified).
		plan.creates = append(plan.creates, &models.LogStreamChunk{
			LogStreamID:    logStreamID,
			ChunkIndex:     0,
			StartOffset:    0,
			Size:           streamSize,
			ObjectStoreKey: consolidatedObjectKey(logStreamID),
			Sealed:         true,
		})
		cur = newChunk(1, streamSize)
	default:
		// No chunk rows yet.
		cur = newChunk(0, 0)
	}

	finalize := func(c *openChunk, sealed bool) {
		if c.existing != nil {
			updated := *c.existing
			updated.Size = c.size
			updated.Sealed = sealed
			updated.Checksum = c.checksum
			// Committing this update releases the write lease claimed on the chunk.
			updated.LeaseExpiresAt = nil
			plan.updates = append(plan.updates, &updated)
			return
		}
		plan.creates = append(plan.creates, &models.LogStreamChunk{
			LogStreamID:    logStreamID,
			ChunkIndex:     c.index,
			StartOffset:    c.startOffset,
			Size:           c.size,
			ObjectStoreKey: c.key,
			Sealed:         sealed,
			Checksum:       c.checksum,
		})
	}

	// Fill the open chunk, sealing and rolling over to a new chunk whenever it reaches the max size.
	for len(data) > 0 {
		if cur.size >= maxChunkSize {
			finalize(cur, true)
			cur = newChunk(cur.index+1, cur.startOffset+cur.size)
		}

		n := min(maxChunkSize-cur.size, len(data))
		plan.objectWrites = append(plan.objectWrites, objectWrite{
			key:        cur.key,
			byteOffset: cur.size,
			data:       data[:n],
			newChunk:   cur.existing == nil,
		})
		cur.size += n
		cur.checksum = extendChecksum(cur.checksum, data[:n])
		data = data[n:]
	}
	finalize(cur, cur.size >= maxChunkSize)

	return plan
}

// extendChecksum returns the CRC-32 of a chunk's bytes followed by data, given the checksum of the
// chunk's bytes; it returns nil when that checksum is unknown. CRC-32 extends incrementally, so the
// prefix isn't rehashed. checksum is not modified.
func extendChecksum(checksum *uint32, data []byte) *uint32 {
	if checksum == nil {
		return nil
	}
	c := crc32.Update(*checksum, crc32.IEEETable, data)
	return &c
}

// writeChunkObjects performs the planned object writes and returns, by object key, the callbacks
// that link new chunks' objects to their chunk rows. The writes run under a deadline well inside the
// chunk lease, so an upload can't still be in flight (and overwrite the object) after the lease has
// expired and another writer has taken it.
func (m *manager) writeChunkObjects(ctx context.Context, writes []objectWrite) (map[string]db.RetainObjectRefFunc, error) {
	writeCtx, cancel := context.WithTimeout(ctx, chunkWriteTimeout)
	defer cancel()

	retainFns := map[string]db.RetainObjectRefFunc{}
	for _, w := range writes {
		linkFunc, err := m.store.WriteChunk(writeCtx, w.key, w.byteOffset, w.data)
		if err != nil {
			return nil, err
		}
		if w.newChunk {
			retainFns[w.key] = linkFunc
		}
	}

	return retainFns, nil
}

// commitChunkWrites saves the planned chunk rows, links the new chunks' objects and saves
// logStream, in one transaction.
func (m *manager) commitChunkWrites(ctx context.Context, plan *chunkWritePlan, retainFns map[string]db.RetainObjectRefFunc,
	logStream *models.LogStream) (*models.LogStream, error) {
	txContext, err := m.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := m.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			m.logger.WithContextFields(ctx).Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	for _, c := range plan.creates {
		createdChunk, err := m.dbClient.LogStreamChunks.CreateLogStreamChunk(txContext, c)
		if err != nil {
			if errors.ErrorCode(err) == errors.EConflict {
				// A concurrent write created this chunk index first. That is a write race, not a client
				// error, so report it as retryable rather than as a terminal conflict.
				return nil, errors.Wrap(err, "log stream %s chunk was created by a concurrent write", logStream.Metadata.ID,
					errors.WithErrorCode(errors.EOptimisticLock))
			}
			return nil, err
		}

		// The adopted legacy chunk 0 has no entry: its object was written, and linked, before chunking.
		if retainFn := retainFns[c.ObjectStoreKey]; retainFn != nil {
			if err = retainFn(txContext, createdChunk.Metadata.ID); err != nil {
				return nil, err
			}
		}
	}
	for _, c := range plan.updates {
		if _, err = m.dbClient.LogStreamChunks.UpdateLogStreamChunk(txContext, c); err != nil {
			return nil, err
		}
	}

	updatedStream, err := m.dbClient.LogStreams.UpdateLogStream(txContext, logStream)
	if err != nil {
		return nil, err
	}

	if err = m.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedStream, nil
}

// releaseChunkLease releases a chunk write lease after a failed write so the next writer doesn't have
// to wait for it to expire. It is best effort: if it fails the lease simply expires.
func (m *manager) releaseChunkLease(ctx context.Context, chunk *models.LogStreamChunk) {
	// Release even if the request context has ended (a canceled request is a common way to get here).
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), leaseReleaseTimeout)
	defer cancel()

	if err := m.dbClient.LogStreamChunks.ReleaseLogStreamChunkLease(releaseCtx, chunk); err != nil {
		m.logger.WithContextFields(ctx).Errorf("failed to release write lease on log stream %s chunk %d: %v",
			chunk.LogStreamID, chunk.ChunkIndex, err)
	}
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
func (m *manager) CompactStream(ctx context.Context, logStream *models.LogStream) error {
	if logStream == nil {
		return errors.New("log stream cannot be nil", errors.WithErrorCode(errors.EInvalid))
	}
	if logStream.Compacted {
		return nil
	}

	// The caller's stream is left unchanged; only the saved copy is marked compacted.
	compacted := *logStream

	// Only write an object if there is log data. A completed stream that logged nothing has no
	// chunks; reads of it return empty via readConsolidated, so just set the flag.
	var consolidatedRetainFn db.RetainObjectRefFunc
	if logStream.Size > 0 {
		// logStream.Compacted is false here, so this streams from the chunk objects. Each chunk is
		// checked as it streams: its object must hold all its committed bytes and, when the chunk's
		// checksum is known, match it. A damaged chunk's data is already lost, so it is logged rather
		// than failing compaction (which would leave the stream uncompacted and retried forever).
		reader, err := m.openLogsReader(ctx, logStream, 0, logStream.Size, func(d *damagedChunk) {
			m.logger.WithContextFields(ctx).Errorf("log stream %s chunk %d (object %s) is damaged, compacting the bytes it holds: %s",
				logStream.Metadata.ID, d.chunk.ChunkIndex, d.chunk.ObjectStoreKey, d)
		})
		if err != nil {
			return err
		}
		defer reader.Close()

		consolidatedKey := consolidatedObjectKey(logStream.Metadata.ID)
		consolidatedRetainFn, err = m.store.WriteConsolidated(ctx, consolidatedKey, reader)
		if err != nil {
			return err
		}

		compacted.ObjectStoreKey = consolidatedKey
	}

	compacted.Compacted = true

	txContext, err := m.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction for CompactStream")
	}

	defer func() {
		if txErr := m.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			m.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CompactStream: %v", txErr)
		}
	}()

	if _, err := m.dbClient.LogStreams.UpdateLogStream(txContext, &compacted); err != nil {
		if errors.ErrorCode(err) == errors.EOptimisticLock {
			// A concurrent update raced with this mark. Completed streams reject further WriteLogs and
			// the SKIP LOCKED claim keeps other instances off this stream, so the consolidated object we
			// wrote is still valid. Abort cleanly without marking Compacted: reads keep using the chunk
			// path and a later run recompacts, harmlessly overwriting the same consolidated object.
			return nil
		}
		return errors.Wrap(err, "failed to update log stream in CompactStream")
	}

	// Link the consolidated object's ref to the stream (skipped for empty streams, which write no object).
	if consolidatedRetainFn != nil {
		if err := consolidatedRetainFn(txContext, logStream.Metadata.ID); err != nil {
			return errors.Wrap(err, "failed to link consolidated object store ref in CompactStream")
		}
	}

	if err := m.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit transaction for CompactStream")
	}

	return nil
}

// verifyOverlap reads the already-stored bytes at [startOffset, startOffset+len(expected)) and
// returns an EConflict error if they differ from expected. This makes idempotent resends safe: the
// overlapping portion of a re-sent write must match what was already persisted.
func (m *manager) verifyOverlap(ctx context.Context, logStream *models.LogStream, startOffset int, expected []byte) error {
	reader, err := m.readLogs(ctx, logStream, startOffset, len(expected))
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
func (m *manager) ReadLogs(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error) {
	if logStream == nil {
		return nil, errors.New("log stream cannot be nil", errors.WithErrorCode(errors.EInvalid))
	}

	if limit > MaxLogPatchSizeBytes {
		return nil, errors.New("log read limit of %d bytes exceeds the maximum of %d bytes",
			limit, MaxLogPatchSizeBytes, errors.WithErrorCode(errors.EInvalid))
	}

	return m.readLogs(ctx, logStream, startOffset, limit)
}

// readLogs returns a reader over [startOffset, startOffset+limit) that streams the overlapping
// chunk objects one at a time, so the full range is never buffered in memory. Streams with no chunk
// rows fall back to the legacy single-object layout.
func (m *manager) readLogs(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error) {
	return m.openLogsReader(ctx, logStream, startOffset, limit, nil)
}

// openLogsReader is readLogs with an optional damaged-chunk callback. A chunk object that holds fewer
// bytes than were committed fails the read with ENotFound, since returning the rest of the range
// would put later bytes at the wrong offsets. When onDamagedChunk is non-nil it is called instead, for
// that and for every chunk that is read in full and doesn't match its known checksum, and the read
// continues.
func (m *manager) openLogsReader(ctx context.Context, logStream *models.LogStream, startOffset int, limit int,
	onDamagedChunk func(damaged *damagedChunk)) (io.ReadCloser, error) {
	if startOffset < 0 || limit < 0 {
		return nil, errors.New("limit and offset cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}
	if limit == 0 {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}

	// A compacted stream's data lives in the single static-key object; read it directly and ignore
	// any (orphaned, not-yet-reclaimed) chunk rows.
	if logStream.Compacted {
		return m.readConsolidated(ctx, logStream, startOffset, limit)
	}

	end := startOffset + limit

	chunks, err := m.dbClient.LogStreamChunks.GetOverlappingChunks(ctx, logStream.Metadata.ID, startOffset, end)
	if err != nil {
		return nil, err
	}

	if len(chunks) == 0 {
		// No chunk rows: this is either a pre-chunking (legacy) stream or one that has not been written.
		return m.readConsolidated(ctx, logStream, startOffset, limit)
	}

	slices := make([]chunkSlice, 0, len(chunks))
	for i := range chunks {
		c := &chunks[i]

		sliceStart := max(startOffset, c.StartOffset) - c.StartOffset
		sliceEnd := min(end, c.StartOffset+c.Size) - c.StartOffset
		if sliceEnd <= sliceStart {
			continue
		}

		slices = append(slices, chunkSlice{chunk: c, offset: sliceStart, length: sliceEnd - sliceStart})
	}

	return &chunkReader{ctx: ctx, store: m.store, slices: slices, onDamagedChunk: onDamagedChunk}, nil
}

// readConsolidated reads the single whole-stream object: the logstreams/{id}.txt key
// first, then the older per-job workspaces/... key, mirroring the historical fallback behavior.
func (m *manager) readConsolidated(ctx context.Context, logStream *models.LogStream, startOffset int, limit int) (io.ReadCloser, error) {
	logs, err := m.store.ReadRange(ctx, consolidatedObjectKey(logStream.Metadata.ID), startOffset, limit)
	if err != nil {
		if errors.ErrorCode(err) != errors.ENotFound {
			return nil, errors.Wrap(err, "failed to read log file from object store")
		}

		if logStream.JobID == nil {
			// Nothing has been written yet.
			return io.NopCloser(bytes.NewReader(nil)), nil
		}

		job, jErr := m.dbClient.Jobs.GetJobByID(ctx, *logStream.JobID)
		if jErr != nil {
			return nil, jErr
		}
		if job == nil {
			return nil, errors.New("job with ID %s not found", *logStream.JobID, errors.WithErrorCode(errors.ENotFound))
		}

		legacyLogs, lErr := m.store.ReadRange(ctx, legacyJobObjectKey(job), startOffset, limit)
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
	chunk  *models.LogStreamChunk
	offset int
	length int
}

// verifiable reports whether the slice covers the whole chunk and the chunk's checksum is known.
func (sl *chunkSlice) verifiable() bool {
	return sl.chunk.Checksum != nil && sl.offset == 0 && sl.length == sl.chunk.Size
}

// damagedChunk describes a chunk whose object doesn't hold the chunk's committed bytes.
type damagedChunk struct {
	chunk     *models.LogStreamChunk
	bytesRead int
	// checksum is the CRC-32 of the bytes read; nil when the chunk wasn't verified.
	checksum *uint32
}

func (d *damagedChunk) String() string {
	if d.bytesRead < d.chunk.Size {
		return fmt.Sprintf("its object holds %d of %d committed bytes", d.bytesRead, d.chunk.Size)
	}
	return fmt.Sprintf("expected crc32 %08x, got %08x", *d.chunk.Checksum, *d.checksum)
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

	// onDamagedChunk is the damaged-chunk callback; see openLogsReader.
	onDamagedChunk func(damaged *damagedChunk)
	// hasher computes the current slice's checksum; nil when the slice isn't verified.
	hasher       hash.Hash32
	curBytesRead int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	for {
		if r.cur == nil {
			if r.idx >= len(r.slices) {
				return 0, io.EOF
			}
			sl := &r.slices[r.idx]
			rc, err := r.store.ReadRange(r.ctx, sl.chunk.ObjectStoreKey, sl.offset, sl.length)
			if err != nil {
				if errors.ErrorCode(err) == errors.ENotFound {
					// The range is within the chunk's committed size and objects are written before the
					// size is committed, so the bytes are missing from the object store. The message is
					// our own, so it carries no object store error details.
					return 0, errors.New("log stream %s chunk %d object %s is missing the %d bytes at offset %d",
						sl.chunk.LogStreamID, sl.chunk.ChunkIndex, sl.chunk.ObjectStoreKey, sl.length, sl.offset,
						errors.WithErrorCode(errors.EDataLoss))
				}
				return 0, err
			}
			r.cur = rc
			r.curBytesRead = 0
			r.hasher = nil
			if r.onDamagedChunk != nil && sl.verifiable() {
				r.hasher = crc32.NewIEEE()
			}
		}

		n, err := r.cur.Read(p)
		if n > 0 {
			r.curBytesRead += n
			if r.hasher != nil {
				_, _ = r.hasher.Write(p[:n])
			}
		}
		if err == io.EOF {
			closeErr := r.cur.Close()
			r.cur = nil
			sliceErr := r.finishSlice()
			r.idx++
			if sliceErr == nil {
				sliceErr = closeErr
			}
			if sliceErr != nil {
				return n, sliceErr
			}
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

// finishSlice checks the slice that was just read to completion. An object holding fewer bytes than
// the slice covers is an EDataLoss error (the same as a range that starts past an object's end),
// unless onDamagedChunk is set; then it and a checksum mismatch are reported to onDamagedChunk.
func (r *chunkReader) finishSlice() error {
	sl := &r.slices[r.idx]
	short := r.curBytesRead < sl.length

	if r.onDamagedChunk == nil {
		if short {
			return errors.New("log stream %s chunk %d object %s holds %d of the %d bytes at offset %d",
				sl.chunk.LogStreamID, sl.chunk.ChunkIndex, sl.chunk.ObjectStoreKey, r.curBytesRead, sl.length, sl.offset,
				errors.WithErrorCode(errors.EDataLoss))
		}
		return nil
	}

	var checksum *uint32
	if r.hasher != nil {
		c := r.hasher.Sum32()
		checksum = &c
		r.hasher = nil
	}
	if short || (checksum != nil && *checksum != *sl.chunk.Checksum) {
		r.onDamagedChunk(&damagedChunk{chunk: sl.chunk, bytesRead: r.curBytesRead, checksum: checksum})
	}
	return nil
}

func (r *chunkReader) Close() error {
	if r.cur != nil {
		err := r.cur.Close()
		r.cur = nil
		return err
	}
	return nil
}

func (m *manager) Subscribe(ctx context.Context, options *SubscriptionOptions) (<-chan *LogEvent, error) {
	// Reject a bad offset up front: failing it mid-stream would be indistinguishable from a transient
	// read failure to a client that reconnects with the same request.
	if options.LastSeenLogSize != nil && *options.LastSeenLogSize < 0 {
		return nil, errors.New("last seen log size cannot be negative", errors.WithErrorCode(errors.EInvalid))
	}

	logStream, err := m.dbClient.LogStreams.GetLogStreamByID(ctx, options.LogStreamID)
	if err != nil {
		return nil, err
	}

	if logStream == nil {
		return nil, errors.New("log stream not found with ID: %s", options.LogStreamID, errors.WithErrorCode(errors.ENotFound))
	}

	subscription := events.Subscription{
		Type: events.LogStreamSubscription,
		ID:   logStream.Metadata.ID,
		Actions: []events.SubscriptionAction{
			events.CreateAction,
			events.UpdateAction,
		},
	}
	subscriber := m.eventManager.Subscribe([]events.Subscription{subscription})

	outgoing := make(chan *LogEvent)

	go func() {
		// Defer close of outgoing channel
		defer close(outgoing)
		defer m.eventManager.Unsubscribe(subscriber)

		if err := m.streamLogEvents(ctx, logStream, options.LastSeenLogSize, subscriber, outgoing); err != nil {
			// Only the subscriber's own context ending (client disconnect, server drain) is a normal end:
			// just close the channel. A context error from anything else (e.g. an object store request
			// timing out) is a failure, and closing silently would read as completion to the client.
			if ctx.Err() != nil {
				return
			}
			if errors.FilterContextError(err) == nil {
				err = errors.Wrap(err, "log stream subscription interrupted", errors.WithErrorCode(errors.EServiceUnavailable))
			}

			m.logger.WithContextFields(ctx).Errorf("log stream subscription for log stream %s failed: %v", logStream.Metadata.ID, err)

			// A server fault can wrap database or object store errors whose details the client must not
			// see, so the client gets a new error with only the code; the full error is logged above.
			switch errors.ErrorCode(err) {
			case errors.EServiceUnavailable, errors.EInternal, errors.ENotImplemented, errors.EDataLoss:
				err = errors.New("log stream subscription for log stream %s failed", logStream.Metadata.ID,
					errors.WithErrorCode(errors.ErrorCode(err)))
			}

			// Deliver the failure as a terminal event so the consumer returns an error to its client
			// instead of closing the stream cleanly, which a client can't tell apart from completion.
			select {
			case <-ctx.Done():
			case outgoing <- &LogEvent{Err: err}:
			}
		}
	}()

	return outgoing, nil
}

// streamLogEvents sends any logs missed since lastSeenLogSize, then forwards live log stream updates
// until the stream completes (returns nil) or the subscription fails (returns the error).
func (m *manager) streamLogEvents(ctx context.Context, logStream *models.LogStream, lastSeenLogSize *int,
	subscriber *events.Subscriber, outgoing chan *LogEvent) error {
	currentSize := logStream.Size

	if lastSeenLogSize != nil {
		// Send all logs that were missed since last seen size
		size, completed, err := m.sendLogstreamEvent(ctx, logStream, *lastSeenLogSize, logStream.Size, logStream.Completed, logStream.Truncated, outgoing)
		if err != nil || completed {
			return err
		}
		currentSize = size
	}

	// Wait for log stream updates
	for {
		event, err := subscriber.GetEvent(ctx)
		if err != nil {
			if errors.FilterContextError(err) == nil {
				return err
			}
			// The event listener failed or this subscriber was dropped. Nothing is lost on the
			// stream itself, so the client can reconnect and resume from its last seen size.
			return errors.Wrap(err, "failed waiting for log stream events", errors.WithErrorCode(errors.EServiceUnavailable))
		}

		logStreamEventData, err := event.ToLogStreamEventData()
		if err != nil {
			// A malformed event payload is a bug, and every later event would fail the same way, so a
			// reconnect can't recover from it.
			return errors.Wrap(err, "failed to get log stream event data for event %s", event.ID, errors.WithErrorCode(errors.EInternal))
		}

		size, completed, err := m.sendLogstreamEvent(ctx, logStream, currentSize, logStreamEventData.Size, logStreamEventData.Completed, logStreamEventData.Truncated, outgoing)
		if err != nil || completed {
			return err
		}
		currentSize = size
	}
}

// sendLogstreamEvent sends the logs in [lastSeenLogSize, actualLogSize) followed, if the stream is
// completed, by the terminal completed event. It returns the new last seen size and whether the
// subscription is done (completed, or the context ended). A non-nil error means the logs could not be
// read and the subscription must fail rather than be reported as completed.
func (m *manager) sendLogstreamEvent(ctx context.Context, logStream *models.LogStream, lastSeenLogSize int, actualLogSize int, completed bool, truncated bool, outgoing chan *LogEvent) (int, bool, error) {
	for lastSeenLogSize < actualLogSize {
		offset := lastSeenLogSize
		// Read logs in bounded windows. The window is small (logEventChunkSizeBytes), so reading it
		// fully into memory here is safe.
		reader, err := m.readLogs(ctx, logStream, lastSeenLogSize, logEventChunkSizeBytes)
		if err != nil {
			return 0, true, readLogsError(logStream, offset, err)
		}

		logs, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			return 0, true, readLogsError(logStream, offset, err)
		}

		if len(logs) == 0 {
			// Guard against making no progress if the bytes are not yet readable.
			break
		}

		lastSeenLogSize += len(logs)

		select {
		case <-ctx.Done():
			return 0, true, ctx.Err()
		case outgoing <- &LogEvent{Size: lastSeenLogSize, Truncated: truncated, Data: &LogEventData{Offset: offset, Logs: string(logs)}}:
		}
	}

	// Return from loop if log stream has been completed because there are no more logs to process
	if completed {
		select {
		case <-ctx.Done():
			return 0, true, ctx.Err()
		case outgoing <- &LogEvent{Size: lastSeenLogSize, Completed: completed, Truncated: truncated}:
		}
	}

	return lastSeenLogSize, completed, nil
}

// readLogsError classifies a failure to read committed log bytes so the client knows whether to
// reconnect:
//   - Committed bytes that are missing (the chunk reader reports EDataLoss; a bare ENotFound from the
//     object store means the same) are permanent: reconnecting would fail the same way forever, so
//     the failure is EDataLoss.
//   - Errors caused by the request itself (e.g. an invalid offset or a denied permission) keep their
//     code, since a reconnect with the same request would fail the same way.
//   - Anything else, such as an object store outage or timeout, is transient: EServiceUnavailable, so
//     the client reconnects and resumes.
func readLogsError(logStream *models.LogStream, offset int, err error) error {
	code := errors.EServiceUnavailable
	switch errors.ErrorCode(err) {
	case errors.ENotFound, errors.EDataLoss:
		code = errors.EDataLoss
	case errors.EInvalid, errors.EForbidden, errors.EUnauthorized, errors.ETooLarge, errors.ENotImplemented:
		code = errors.ErrorCode(err)
	}

	return errors.Wrap(err, "failed to read logs for log stream %s at offset %d", logStream.Metadata.ID, offset, errors.WithErrorCode(code))
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
