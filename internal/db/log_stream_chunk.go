package db

//go:generate go tool mockery --name LogStreamChunks --inpackage --case underscore

import (
	"context"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"go.opentelemetry.io/otel/trace"
)

// LogStreamChunks encapsulates the logic to access log stream chunks from the database
type LogStreamChunks interface {
	// GetOverlappingChunks returns the chunks for a stream whose byte range overlaps [startOffset, endOffset),
	// ordered by start offset ascending.
	GetOverlappingChunks(ctx context.Context, logStreamID string, startOffset int, endOffset int) ([]models.LogStreamChunk, error)
	// GetActiveChunk returns the highest-indexed (tail) chunk for a stream, or nil if the stream has no chunks.
	GetActiveChunk(ctx context.Context, logStreamID string) (*models.LogStreamChunk, error)
	CreateLogStreamChunk(ctx context.Context, chunk *models.LogStreamChunk) (*models.LogStreamChunk, error)
	// UpdateLogStreamChunk persists the chunk's size, sealed flag, checksum and lease (so a writer
	// finalizing its write releases its lease by clearing LeaseExpiresAt). It is optimistic-locked on
	// the chunk's version.
	UpdateLogStreamChunk(ctx context.Context, chunk *models.LogStreamChunk) (*models.LogStreamChunk, error)
	// ClaimLogStreamChunkLease claims the chunk's write lease for leaseDuration, measured on the database
	// clock. It succeeds only if the chunk is unchanged since the caller read it (same version) and no
	// unexpired lease is held; otherwise it returns ErrOptimisticLockError. The returned chunk carries the
	// bumped version, which fences the holder: once the lease expires and another writer claims it, the
	// holder's final UpdateLogStreamChunk fails the version check.
	ClaimLogStreamChunkLease(ctx context.Context, chunk *models.LogStreamChunk, leaseDuration time.Duration) (*models.LogStreamChunk, error)
	// ReleaseLogStreamChunkLease releases a lease claimed by ClaimLogStreamChunkLease without otherwise
	// changing the chunk. It is a no-op if the chunk has changed since the claim (e.g. the lease expired
	// and was claimed by another writer).
	ReleaseLogStreamChunkLease(ctx context.Context, chunk *models.LogStreamChunk) error
}

var logStreamChunkFieldList = append(metadataFieldList, "log_stream_id", "chunk_index", "start_offset", "size", "object_store_key", "sealed",
	"lease_expires_at", "checksum")

// dbNowUTC is the database clock as a UTC TIMESTAMP (WITHOUT TIME ZONE), matching how timestamps are
// stored. Lease expiry is measured on it rather than on API instance clocks so instances with skewed
// clocks agree on when a lease has expired.
const dbNowUTC = "(now() AT TIME ZONE 'UTC')"

type logStreamChunks struct {
	dbClient *Client
}

// NewLogStreamChunks returns an instance of the LogStreamChunks interface
func NewLogStreamChunks(dbClient *Client) LogStreamChunks {
	return &logStreamChunks{dbClient: dbClient}
}

func (l *logStreamChunks) GetOverlappingChunks(ctx context.Context, logStreamID string, startOffset int, endOffset int) ([]models.LogStreamChunk, error) {
	ctx, span := tracer.Start(ctx, "db.GetOverlappingChunks")
	defer span.End()

	// A chunk [start_offset, start_offset+size) overlaps the requested [startOffset, endOffset) when
	// it begins before the request ends and ends after the request begins (half-open intervals).
	query := dialect.From(goqu.T("log_stream_chunks")).
		Prepared(true).
		Select(l.getSelectFields()...).
		Where(
			goqu.Ex{"log_stream_chunks.log_stream_id": logStreamID},
			goqu.C("start_offset").Lt(endOffset),
			goqu.L("(start_offset + size) > ?", startOffset),
		).
		Order(goqu.C("start_offset").Asc())

	return l.queryChunks(ctx, span, "log_stream_chunk.GetOverlappingChunks", query)
}

func (l *logStreamChunks) GetActiveChunk(ctx context.Context, logStreamID string) (*models.LogStreamChunk, error) {
	ctx, span := tracer.Start(ctx, "db.GetActiveChunk")
	defer span.End()

	query := dialect.From(goqu.T("log_stream_chunks")).
		Prepared(true).
		Select(l.getSelectFields()...).
		Where(goqu.Ex{"log_stream_chunks.log_stream_id": logStreamID}).
		Order(goqu.C("chunk_index").Desc()).
		Limit(1)

	sql, args, err := toSQLWithTag("log_stream_chunk.GetActiveChunk", query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	chunk, err := scanLogStreamChunk(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return chunk, nil
}

func (l *logStreamChunks) CreateLogStreamChunk(ctx context.Context, chunk *models.LogStreamChunk) (*models.LogStreamChunk, error) {
	ctx, span := tracer.Start(ctx, "db.CreateLogStreamChunk")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("log_stream_chunk.CreateLogStreamChunk", dialect.Insert("log_stream_chunks").
		Prepared(true).
		Rows(goqu.Record{
			"id":               newResourceID(),
			"version":          initialResourceVersion,
			"created_at":       timestamp,
			"updated_at":       timestamp,
			"log_stream_id":    chunk.LogStreamID,
			"chunk_index":      chunk.ChunkIndex,
			"start_offset":     chunk.StartOffset,
			"size":             chunk.Size,
			"object_store_key": chunk.ObjectStoreKey,
			"sealed":           chunk.Sealed,
			"checksum":         nullableChecksum(chunk.Checksum),
		}).
		Returning(logStreamChunkFieldList...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	createdChunk, err := scanLogStreamChunk(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isForeignKeyViolation(pgErr) && pgErr.ConstraintName == "fk_log_stream_chunks_log_stream_id" {
				return nil, errors.New("log stream does not exist", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
			}
			if isUniqueViolation(pgErr) {
				return nil, errors.New("log stream chunk already exists for stream %s index %d", chunk.LogStreamID, chunk.ChunkIndex,
					errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return createdChunk, nil
}

func (l *logStreamChunks) UpdateLogStreamChunk(ctx context.Context, chunk *models.LogStreamChunk) (*models.LogStreamChunk, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateLogStreamChunk")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := toSQLWithTag("log_stream_chunk.UpdateLogStreamChunk", dialect.Update("log_stream_chunks").
		Prepared(true).
		Set(
			goqu.Record{
				"version":          goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":       timestamp,
				"size":             chunk.Size,
				"sealed":           chunk.Sealed,
				"checksum":         nullableChecksum(chunk.Checksum),
				"lease_expires_at": chunk.LeaseExpiresAt,
			},
		).Where(goqu.Ex{"id": chunk.Metadata.ID, "version": chunk.Metadata.Version}).
		Returning(logStreamChunkFieldList...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	updatedChunk, err := scanLogStreamChunk(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return updatedChunk, nil
}

func (l *logStreamChunks) ClaimLogStreamChunkLease(ctx context.Context, chunk *models.LogStreamChunk, leaseDuration time.Duration) (*models.LogStreamChunk, error) {
	ctx, span := tracer.Start(ctx, "db.ClaimLogStreamChunkLease")
	defer span.End()

	sql, args, err := toSQLWithTag("log_stream_chunk.ClaimLogStreamChunkLease", dialect.Update("log_stream_chunks").
		Prepared(true).
		Set(
			goqu.Record{
				"version":          goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at":       currentTime(),
				"lease_expires_at": goqu.L(dbNowUTC+" + CAST(? AS DOUBLE PRECISION) * INTERVAL '1 millisecond'", float64(leaseDuration.Milliseconds())),
			},
		).Where(
		goqu.Ex{"id": chunk.Metadata.ID, "version": chunk.Metadata.Version},
		goqu.Or(
			goqu.C("lease_expires_at").IsNull(),
			goqu.C("lease_expires_at").Lte(goqu.L(dbNowUTC)),
		),
	).
		Returning(logStreamChunkFieldList...))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	claimedChunk, err := scanLogStreamChunk(l.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			// Either the chunk changed since the caller read it or another writer holds the lease.
			return nil, ErrOptimisticLockError
		}
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return claimedChunk, nil
}

func (l *logStreamChunks) ReleaseLogStreamChunkLease(ctx context.Context, chunk *models.LogStreamChunk) error {
	ctx, span := tracer.Start(ctx, "db.ReleaseLogStreamChunkLease")
	defer span.End()

	// The version is left unchanged: releasing doesn't change the chunk's data, and matching on the
	// claimed version makes this a no-op for a lease that has since been claimed by another writer.
	sql, args, err := toSQLWithTag("log_stream_chunk.ReleaseLogStreamChunkLease", dialect.Update("log_stream_chunks").
		Prepared(true).
		Set(goqu.Record{"lease_expires_at": nil}).
		Where(goqu.Ex{"id": chunk.Metadata.ID, "version": chunk.Metadata.Version}))
	if err != nil {
		return errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	if _, err = l.dbClient.getConnection(ctx).Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}

	return nil
}

func (l *logStreamChunks) queryChunks(ctx context.Context, span trace.Span, tag string, query *goqu.SelectDataset) ([]models.LogStreamChunk, error) {
	sql, args, err := toSQLWithTag(tag, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate SQL", errors.WithSpan(span))
	}

	rows, err := l.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute query", errors.WithSpan(span))
	}
	defer rows.Close()

	results := []models.LogStreamChunk{}
	for rows.Next() {
		item, err := scanLogStreamChunk(rows)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
		}
		results = append(results, *item)
	}

	return results, nil
}

func (*logStreamChunks) getSelectFields() []interface{} {
	selectFields := []interface{}{}
	for _, field := range logStreamChunkFieldList {
		selectFields = append(selectFields, fmt.Sprintf("log_stream_chunks.%s", field))
	}

	return selectFields
}

func scanLogStreamChunk(row scanner) (*models.LogStreamChunk, error) {
	chunk := &models.LogStreamChunk{}

	var checksum *int64

	fields := []interface{}{
		&chunk.Metadata.ID,
		&chunk.Metadata.CreationTimestamp,
		&chunk.Metadata.LastUpdatedTimestamp,
		&chunk.Metadata.Version,
		&chunk.LogStreamID,
		&chunk.ChunkIndex,
		&chunk.StartOffset,
		&chunk.Size,
		&chunk.ObjectStoreKey,
		&chunk.Sealed,
		&chunk.LeaseExpiresAt,
		&checksum,
	}

	if err := row.Scan(fields...); err != nil {
		return nil, err
	}

	if checksum != nil {
		// Stored as BIGINT because Postgres has no unsigned 32-bit type; the value always fits a uint32.
		c := uint32(*checksum) // nolint:gosec
		chunk.Checksum = &c
	}

	chunk.Metadata.TRN = trn.TypeLogStreamChunk.Build(chunk.GetGlobalID())

	return chunk, nil
}

// nullableChecksum converts a chunk checksum to its BIGINT column value (NULL when unknown).
func nullableChecksum(checksum *uint32) *int64 {
	if checksum == nil {
		return nil
	}
	v := int64(*checksum)
	return &v
}
