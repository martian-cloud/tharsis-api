// Package logstore provides storage and live-tail streaming of log entries.
package logstore

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// ansiEscape matches ANSI escape sequences. Copied from internal/ansi (acarl005/stripansi,
// MIT) rather than imported, to keep this package product-neutral.
var ansiEscape = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// writeQueueSize bounds the formatting queue; writes drop when full (best-effort viewer).
const writeQueueSize = 1024

// Recognized log levels (canonical uppercase, matching the GraphQL enum).
const (
	logLevelDebug = "DEBUG"
	logLevelInfo  = "INFO"
	logLevelWarn  = "WARN"
	logLevelError = "ERROR"
)

// LogEntry is a single captured log line, or — on a live-tail stream — a terminal
// error event when Err is set (carrying no log data).
type LogEntry struct {
	Seq       uint64
	Timestamp time.Time
	Level     string
	Message   string
	Caller    string // source location "pkg/file.go:line"; empty if unknown
	Stack     string // stack trace; populated for error-level entries
	Fields    string // JSON-encoded zap context fields; empty if none
	Err       error  // set only on a terminal stream event (e.g. a dropped backend connection)
}

// Matches reports whether the entry passes the level and search filters. Empty filters
// match all; levels matches if the entry's level equals any of them; search is a
// case-insensitive substring of message, caller, or fields.
func (e *LogEntry) Matches(levels []string, search string) bool {
	if len(levels) > 0 && !slices.Contains(levels, strings.ToUpper(e.Level)) {
		return false
	}

	if search == "" {
		return true
	}

	needle := strings.ToLower(search)
	return strings.Contains(strings.ToLower(e.Message), needle) ||
		strings.Contains(strings.ToLower(e.Caller), needle) ||
		strings.Contains(strings.ToLower(e.Fields), needle)
}

// AreValidLevels reports whether every provided level is a recognized log level.
func AreValidLevels(levels ...string) bool {
	for _, level := range levels {
		switch level {
		case logLevelDebug, logLevelInfo, logLevelWarn, logLevelError:
		default:
			return false
		}
	}

	return true
}

//go:generate go tool mockery --name Store --inpackage --case underscore

// Store is the interface implemented by log store backends.
type Store interface {
	Write(e *LogEntry)
	GetEntries(levels []string, search string, limit int) ([]*LogEntry, error)
	Subscribe(ctx context.Context) (<-chan *LogEntry, error)
}

var _ zapcore.Core = (*Core)(nil)

// Core is a zapcore.Core that tees log entries into a Store (stdout is unchanged). Write
// only captures field values and enqueues; a single worker does the expensive formatting,
// so that cost never lands on the logging goroutine.
type Core struct {
	store       Store
	accumulated []zapcore.Field
	queue       chan *pendingEntry
}

// pendingEntry is a captured line awaiting formatting; fields are resolved but not parsed.
type pendingEntry struct {
	timestamp time.Time
	level     string
	message   string
	caller    string
	stack     string
	fields    map[string]any
}

// NewCore returns a Core backed by the given Store and starts its formatting worker.
// If the store implements IsNoop() bool and returns true, no goroutine or queue is
// allocated — Write becomes a no-op.
func NewCore(store Store) *Core {
	c := &Core{store: store}
	if n, ok := store.(interface{ IsNoop() bool }); ok && n.IsNoop() {
		return c
	}
	c.queue = make(chan *pendingEntry, writeQueueSize)
	go c.process()
	return c
}

// Enabled always returns true; the composing logger gates by level (see logger.WithCore).
func (c *Core) Enabled(_ zapcore.Level) bool {
	return true
}

// With accumulates fields and shares the queue, so derived cores don't each spawn a worker.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	merged := make([]zapcore.Field, len(c.accumulated)+len(fields))
	copy(merged, c.accumulated)
	copy(merged[len(c.accumulated):], fields)
	return &Core{store: c.store, accumulated: merged, queue: c.queue}
}

// Check adds this core to ce.
func (c *Core) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(entry, c)
}

// Write captures the entry and enqueues it for async formatting. Fields are resolved here
// because zap may reuse a field's backing memory once Write returns.
func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if c.queue == nil {
		return nil
	}

	all := make([]zapcore.Field, 0, len(c.accumulated)+len(fields))
	all = append(all, c.accumulated...)
	all = append(all, fields...)

	p := &pendingEntry{
		timestamp: entry.Time,
		level:     entry.Level.String(),
		message:   entry.Message,
		stack:     entry.Stack,
		fields:    resolveFields(all),
	}
	if entry.Caller.Defined {
		p.caller = entry.Caller.TrimmedPath()
	}

	select {
	case c.queue <- p:
	default:
		// drop when the worker is behind (same best-effort contract as the stores)
	}
	return nil
}

// Sync is a no-op (the store handles its own persistence).
func (c *Core) Sync() error {
	return nil
}

// process formats queued entries and writes them to the store, off the logging goroutine.
func (c *Core) process() {
	for p := range c.queue {
		e := &LogEntry{
			Timestamp: p.timestamp,
			Level:     strings.ToUpper(p.level),
			Message:   ansiEscape.ReplaceAllString(p.message, ""),
			Caller:    p.caller,
			Stack:     p.stack,
			Fields:    formatFields(p.fields),
		}
		c.store.Write(e)
	}
}

// resolveFields extracts each zap field to a plain Go value (cheap; parsing is deferred).
func resolveFields(fields []zapcore.Field) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = fieldValue(f)
	}

	return m
}

// formatFields strips ANSI, deep-parses nested JSON, and renders the fields as indented text.
func formatFields(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}

	for k, v := range m {
		m[k] = deepParseJSON(v)
	}

	return formatValue(m, 0)
}

// formatValue renders v as indented text, preserving real newlines in string values.
func formatValue(v any, depth int) string {
	switch val := v.(type) {
	case map[string]any:
		if len(val) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		items := make([]string, len(keys))
		for i, k := range keys {
			kb, _ := json.Marshal(k)
			items[i] = string(kb) + ": " + formatValue(val[k], depth+1)
		}
		return wrapItems("{", "}", items, depth)
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		items := make([]string, len(val))
		for i, e := range val {
			items[i] = formatValue(e, depth+1)
		}
		return wrapItems("[", "]", items, depth)
	default:
		return formatScalar(v)
	}
}

func formatScalar(v any) string {
	if s, ok := v.(string); ok && strings.Contains(s, "\n") {
		return `"` + s + `"`
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func wrapItems(open, end string, items []string, depth int) string {
	pad := strings.Repeat("  ", depth)
	inner := strings.Repeat("  ", depth+1)
	return open + "\n" + inner + strings.Join(items, ",\n"+inner) + "\n" + pad + end
}

// deepParseJSON strips ANSI from string leaves and replaces JSON-encoded strings with
// the parsed structure, so nested payloads render as readable data.
func deepParseJSON(v any) any {
	switch val := v.(type) {
	case string:
		cleaned := ansiEscape.ReplaceAllString(val, "")
		trimmed := strings.TrimSpace(cleaned)
		if !isJSONObjectOrArray(trimmed) {
			return cleaned
		}
		var parsed any
		if json.Unmarshal([]byte(trimmed), &parsed) != nil {
			return cleaned
		}
		return deepParseJSON(parsed)
	case map[string]any:
		for k, e := range val {
			val[k] = deepParseJSON(e)
		}
		return val
	case []any:
		for i, e := range val {
			val[i] = deepParseJSON(e)
		}
		return val
	default:
		return v
	}
}

func isJSONObjectOrArray(s string) bool {
	if len(s) < 2 {
		return false
	}
	return (s[0] == '{' && s[len(s)-1] == '}') || (s[0] == '[' && s[len(s)-1] == ']')
}

func fieldValue(f zapcore.Field) any {
	switch f.Type {
	case zapcore.StringType:
		return f.String
	case zapcore.StringerType:
		return fmt.Sprintf("%s", f.Interface)
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return f.Integer
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return uint64(f.Integer)
	case zapcore.Float64Type:
		return math.Float64frombits(uint64(f.Integer))
	case zapcore.Float32Type:
		return math.Float32frombits(uint32(f.Integer))
	case zapcore.BoolType:
		return f.Integer == 1
	case zapcore.DurationType:
		return time.Duration(f.Integer).String()
	case zapcore.TimeType:
		if f.Interface != nil {
			if loc, ok := f.Interface.(*time.Location); ok {
				return time.Unix(0, f.Integer).In(loc).Format(time.RFC3339)
			}
		}
		return time.Unix(0, f.Integer).UTC().Format(time.RFC3339)
	case zapcore.ErrorType:
		if err, ok := f.Interface.(error); ok {
			return err.Error()
		}
		return fmt.Sprintf("%v", f.Interface)
	case zapcore.SkipType:
		return nil
	default:
		if f.Interface != nil {
			return fmt.Sprintf("%v", f.Interface)
		}
		return f.Integer
	}
}
