// Package sqltag injects and parses a fixed-format leading SQL comment used to tag
// queries with a stable operation name. The tag rides along to PostgreSQL (visible
// in pg_stat_statements, slow-query logs, and pg_stat_activity) and is parsed back
// out by the DB query tracer to label per-query Prometheus metrics.
//
// It lives in its own package so that both the db layer (which injects the comment
// in toSQL and parses it in the tracer) and the pagination layer (which comments
// its internally-built rows and count queries) can share one definition of the
// format without an import cycle.
package sqltag

import "strings"

const (
	prefix = "/* op='"
	suffix = "' */ "
)

// Comment returns the leading SQL comment tagging a query with op. op must be a
// static, bounded literal (e.g. "jobs.getJob") so the resulting SQL text is
// identical on every execution of a given query, keeping metric cardinality
// bounded and pgx's prepared-statement cache key stable. Any "'" or "*/" in op is
// stripped defensively to keep the comment well-formed.
func Comment(op string) string {
	op = strings.ReplaceAll(op, "'", "")
	op = strings.ReplaceAll(op, "*/", "")
	return prefix + op + suffix
}

// Inject prepends Comment(op) to sql.
func Inject(op, sql string) string {
	return Comment(op) + sql
}

// Parse returns the op tag if sql carries our comment, otherwise "".
//
// This runs on every query, so it is allocation-free (the result is a sub-slice of
// sql, not a copy) and bounded to the comment length rather than the query length:
// the leading HasPrefix check bails on untagged SQL after a few bytes, and because
// Comment strips any "'" from op, the first quote after the prefix is the tag
// terminator. IndexByte is a single-byte (SIMD-optimized) scan that stops there.
func Parse(sql string) string {
	if !strings.HasPrefix(sql, prefix) {
		return ""
	}
	rest := sql[len(prefix):]
	end := strings.IndexByte(rest, '\'')
	if end < 0 {
		return ""
	}
	return rest[:end]
}
