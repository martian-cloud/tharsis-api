package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sqltag"
)

// unknownQueryType is the op label used when a query carries no operation comment.
const unknownQueryType = "UnknownQueryType"

var (
	dbQueryCount = metric.NewCounterVec(
		"db_query_total",
		"Total number of DB queries executed, labeled by operation and status.",
		[]string{"op", "status"},
	)
	// Buckets span ~0.5ms to ~16s (0.0005 * 2^15).
	dbQueryDuration = metric.NewHistogramVec(
		"db_query_duration_seconds",
		"DB query latency in seconds, labeled by operation and status.",
		0.0005, 2, 16,
		[]string{"op", "status"},
	)
)

// normalizeOp maps an empty op (a query without an operation comment) to the
// unknownQueryType label.
func normalizeOp(op string) string {
	if op == "" {
		return unknownQueryType
	}
	return op
}

// queryStatus classifies a query result for the status label. pgx.ErrNoRows is treated as a
// successful result, not an error.
func queryStatus(err error) string {
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "error"
	}
	return "ok"
}

// recordQuery emits the count and latency metrics for a single query execution.
func recordQuery(op string, start time.Time, err error) {
	op = normalizeOp(op)
	status := queryStatus(err)
	dbQueryCount.WithLabelValues(op, status).Inc()
	dbQueryDuration.WithLabelValues(op, status).Observe(time.Since(start).Seconds())
}

// queryMetricsTracer implements pgx.QueryTracer to record per-query Prometheus
// metrics. It is installed on the pool's ConnConfig.Tracer, so it covers Query,
// QueryRow, and Exec across pooled connections and transactions. pgx calls
// TraceQueryEnd when the operation truly completes (including the deferred Scan
// of a QueryRow), so the measured latency captures the full round-trip.
//
// The operation tag is parsed from the leading SQL comment injected by sqltag
// (via toSQLWithTag / pagination.WithQueryTag).
type queryMetricsTracer struct{}

type queryMetricsCtxKey struct{}

type queryMetricsState struct {
	op    string
	start time.Time
}

func (queryMetricsTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	op := sqltag.Parse(data.SQL)
	if op == "" {
		return ctx
	}

	return context.WithValue(ctx, queryMetricsCtxKey{}, &queryMetricsState{
		op:    op,
		start: time.Now(),
	})
}

func (queryMetricsTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	state, ok := ctx.Value(queryMetricsCtxKey{}).(*queryMetricsState)
	if !ok {
		return
	}
	recordQuery(state.op, state.start, data.Err)
}
