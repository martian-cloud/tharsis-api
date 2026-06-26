package db

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sqltag"
)

func TestNormalizeOp(t *testing.T) {
	if got := normalizeOp(""); got != unknownQueryType {
		t.Errorf("normalizeOp(%q) = %q, want %q", "", got, unknownQueryType)
	}
	if got := normalizeOp("jobs.GetJobs"); got != "jobs.GetJobs" {
		t.Errorf("normalizeOp(%q) = %q, want it unchanged", "jobs.GetJobs", got)
	}
}

func TestQueryStatus(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil is ok", nil, "ok"},
		{"ErrNoRows is ok", pgx.ErrNoRows, "ok"},
		{"wrapped ErrNoRows is ok", fmt.Errorf("scan failed: %w", pgx.ErrNoRows), "ok"},
		{"other error is error", errors.New("boom"), "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := queryStatus(tc.err); got != tc.want {
				t.Errorf("queryStatus(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

// TestQueryMetricsTracerStart verifies the tracer measures only tagged queries: a query whose
// SQL carries an operation comment is tracked with that op, while untagged queries (including
// the events module's raw "listen events" command) are skipped and carry no metrics state.
func TestQueryMetricsTracerStart(t *testing.T) {
	tr := queryMetricsTracer{}

	t.Run("tagged query carries its op", func(t *testing.T) {
		ctx := tr.TraceQueryStart(context.Background(), nil,
			pgx.TraceQueryStartData{SQL: sqltag.Inject("jobs.GetJobs", "SELECT 1")})
		state, ok := ctx.Value(queryMetricsCtxKey{}).(*queryMetricsState)
		if !ok {
			t.Fatal("expected tagged query to carry metrics state")
		}
		if state.op != "jobs.GetJobs" {
			t.Fatalf("op = %q, want %q", state.op, "jobs.GetJobs")
		}
	})

	// Untagged queries are not measured: TraceQueryStart returns the context unchanged so
	// TraceQueryEnd has no state to record.
	for _, sql := range []string{"SELECT 1 /* no leading op comment */", "listen events"} {
		t.Run("untagged query is skipped: "+sql, func(t *testing.T) {
			ctx := tr.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: sql})
			if ctx.Value(queryMetricsCtxKey{}) != nil {
				t.Fatalf("expected no metrics state for untagged query %q", sql)
			}
			// TraceQueryEnd on a skipped context must be a no-op (no panic).
			tr.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
		})
	}
}
