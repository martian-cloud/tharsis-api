package logstore

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// recordingStore returns a MockStore that records written entries plus a getter for them.
func recordingStore(t *testing.T) (*MockStore, func() []*LogEntry) {
	m := NewMockStore(t)

	var mu sync.Mutex
	var entries []*LogEntry

	m.On("Write", mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		entries = append(entries, args.Get(0).(*LogEntry))
		mu.Unlock()
	}).Return().Maybe()

	return m, func() []*LogEntry {
		mu.Lock()
		defer mu.Unlock()
		out := make([]*LogEntry, len(entries))
		copy(out, entries)
		return out
	}
}

func TestLogEntryMatches(t *testing.T) {
	entry := &LogEntry{
		Level:   "INFO",
		Message: "Failed to connect to DB",
		Caller:  "db/conn.go:88",
		Fields:  `{"jobId":"abc123","method":"SaveJobLogs"}`,
	}

	testCases := []struct {
		name   string
		levels []string
		search string
		want   bool
	}{
		{name: "no filters matches", want: true},
		{name: "level match exact", levels: []string{"INFO"}, want: true},
		{name: "level mismatch", levels: []string{"ERROR"}, want: false},
		{name: "multiple levels matches one", levels: []string{"WARN", "INFO"}, want: true},
		{name: "multiple levels none match", levels: []string{"WARN", "ERROR"}, want: false},
		{name: "search substring match", search: "connect", want: true},
		{name: "search case-insensitive", search: "FAILED", want: true},
		{name: "search no match", search: "timeout", want: false},
		{name: "both match", levels: []string{"INFO"}, search: "db", want: true},
		{name: "level ok search bad", levels: []string{"INFO"}, search: "nope", want: false},
		{name: "level bad search ok", levels: []string{"WARN"}, search: "db", want: false},
		{name: "search matches caller", search: "conn.go", want: true},
		{name: "search matches fields jobId", search: "abc123", want: true},
		{name: "search matches fields method", search: "savejoblogs", want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, entry.Matches(tc.levels, tc.search))
		})
	}
}

type namedStringer struct{ name string }

func (n namedStringer) String() string { return n.name }

func TestFieldValue(t *testing.T) {
	utc := time.Date(2026, 6, 11, 1, 2, 3, 0, time.UTC)

	testCases := []struct {
		name  string
		field zapcore.Field
		want  any
	}{
		{name: "string", field: zap.String("k", "v"), want: "v"},
		{name: "int", field: zap.Int("k", 5), want: int64(5)},
		{name: "int32", field: zap.Int32("k", 3), want: int64(3)},
		{name: "uint", field: zap.Uint("k", 7), want: uint64(7)},
		{name: "float64", field: zap.Float64("k", 1.5), want: 1.5},
		{name: "float32", field: zap.Float32("k", 2.5), want: float32(2.5)},
		{name: "bool true", field: zap.Bool("k", true), want: true},
		{name: "bool false", field: zap.Bool("k", false), want: false},
		{name: "duration", field: zap.Duration("k", 2*time.Second), want: "2s"},
		{name: "time utc", field: zap.Time("k", utc), want: "2026-06-11T01:02:03Z"},
		{name: "error", field: zap.Error(errors.New("boom")), want: "boom"},
		{name: "stringer", field: zap.Stringer("k", namedStringer{name: "abc"}), want: "abc"},
		{name: "skip", field: zap.Skip(), want: nil},
		{name: "reflect falls through to %v", field: zap.Reflect("k", 42), want: "42"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fieldValue(tc.field))
		})
	}
}

func TestIsJSONObjectOrArray(t *testing.T) {
	testCases := []struct {
		in   string
		want bool
	}{
		{in: "", want: false},
		{in: "{", want: false},
		{in: "}", want: false},
		{in: "{}", want: true},
		{in: "[]", want: true},
		{in: `{"a":1}`, want: true},
		{in: `[1,2]`, want: true},
		{in: "{abc", want: false},
		{in: "abc}", want: false},
		{in: "[}", want: false},
		{in: "plain text", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, isJSONObjectOrArray(tc.in))
		})
	}
}

func TestDeepParseJSON(t *testing.T) {
	t.Run("plain string passes through", func(t *testing.T) {
		assert.Equal(t, "hello", deepParseJSON("hello"))
	})

	t.Run("strips ANSI from string leaf", func(t *testing.T) {
		assert.Equal(t, "red", deepParseJSON("\x1b[1mred\x1b[0m"))
	})

	t.Run("parses JSON object string", func(t *testing.T) {
		assert.Equal(t, map[string]any{"a": float64(1)}, deepParseJSON(`{"a":1}`))
	})

	t.Run("parses JSON array string", func(t *testing.T) {
		assert.Equal(t, []any{float64(1), float64(2)}, deepParseJSON(`[1,2]`))
	})

	t.Run("recursively parses nested JSON-in-string", func(t *testing.T) {
		got := deepParseJSON(`{"x":"{\"y\":1}"}`)
		assert.Equal(t, map[string]any{"x": map[string]any{"y": float64(1)}}, got)
	})

	t.Run("invalid JSON returns cleaned string", func(t *testing.T) {
		assert.Equal(t, `{"a":}`, deepParseJSON(`{"a":}`))
	})

	t.Run("strips ANSI inside nested values after parse", func(t *testing.T) {
		got := deepParseJSON(`{"msg":"` + "\x1b[31mboom\x1b[0m" + `"}`)
		assert.Equal(t, map[string]any{"msg": "boom"}, got)
	})

	t.Run("non-string scalar passes through", func(t *testing.T) {
		assert.Equal(t, float64(3), deepParseJSON(float64(3)))
		assert.Equal(t, true, deepParseJSON(true))
		assert.Nil(t, deepParseJSON(nil))
	})

	t.Run("recurses into map and slice values", func(t *testing.T) {
		in := map[string]any{"a": []any{"\x1b[1mx\x1b[0m"}}
		assert.Equal(t, map[string]any{"a": []any{"x"}}, deepParseJSON(in))
	})
}

func TestFormatScalar(t *testing.T) {
	testCases := []struct {
		name string
		in   any
		want string
	}{
		{name: "string", in: "hi", want: `"hi"`},
		{name: "string with newline keeps real breaks", in: "a\nb", want: "\"a\nb\""},
		{name: "number", in: float64(1026), want: "1026"},
		{name: "bool", in: true, want: "true"},
		{name: "nil", in: nil, want: "null"},
		{name: "string needing escape", in: `a"b`, want: `"a\"b"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatScalar(tc.in))
		})
	}
}

func TestFormatValue(t *testing.T) {
	testCases := []struct {
		name string
		in   any
		want string
	}{
		{name: "empty object", in: map[string]any{}, want: "{}"},
		{name: "empty array", in: []any{}, want: "[]"},
		{name: "scalar", in: float64(5), want: "5"},
		{
			name: "object sorts keys",
			in:   map[string]any{"b": float64(1), "a": "x"},
			want: "{\n  \"a\": \"x\",\n  \"b\": 1\n}",
		},
		{
			name: "array",
			in:   []any{"x", float64(2)},
			want: "[\n  \"x\",\n  2\n]",
		},
		{
			name: "nested object indents",
			in:   map[string]any{"o": map[string]any{"k": "v"}},
			want: "{\n  \"o\": {\n    \"k\": \"v\"\n  }\n}",
		},
		{
			name: "multiline string value honored",
			in:   map[string]any{"logs": "line1\nline2"},
			want: "{\n  \"logs\": \"line1\nline2\"\n}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatValue(tc.in, 0))
		})
	}
}

func TestResolveFields(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, resolveFields(nil))
	})

	t.Run("resolves field values without parsing", func(t *testing.T) {
		got := resolveFields([]zapcore.Field{
			zap.String("a", `{"x":1}`),
			zap.Int("b", 2),
		})
		// resolveFields does NOT parse nested JSON — that's deferred to formatFields.
		assert.Equal(t, map[string]any{"a": `{"x":1}`, "b": int64(2)}, got)
	})
}

func TestFormatFields(t *testing.T) {
	t.Run("empty map returns empty string", func(t *testing.T) {
		assert.Equal(t, "", formatFields(nil))
		assert.Equal(t, "", formatFields(map[string]any{}))
	})

	t.Run("deep-parses and formats", func(t *testing.T) {
		got := formatFields(map[string]any{"input": `{"id":"x","n":1}`})
		assert.Equal(t, "{\n  \"input\": {\n    \"id\": \"x\",\n    \"n\": 1\n  }\n}", got)
	})
}

func TestCoreWriteAsync(t *testing.T) {
	store, snapshot := recordingStore(t)
	core := NewCore(store)

	entry := zapcore.Entry{
		Time:    time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC),
		Level:   zapcore.InfoLevel,
		Message: "\x1b[31mfailed\x1b[0m to connect",
		Stack:   "goroutine 1 [running]",
		Caller:  zapcore.EntryCaller{Defined: true, File: "pkg/db/conn.go", Line: 88},
	}

	require.NoError(t, core.Write(entry, []zapcore.Field{
		zap.String("input", `{"jobId":"abc","n":1}`),
	}))

	require.Eventually(t, func() bool {
		return len(snapshot()) == 1
	}, time.Second, time.Millisecond)

	got := snapshot()[0]
	assert.Equal(t, "failed to connect", got.Message, "ANSI should be stripped")
	assert.Equal(t, "INFO", got.Level)
	assert.Equal(t, "db/conn.go:88", got.Caller, "caller is the trimmed (dir/file) path")
	assert.Equal(t, "goroutine 1 [running]", got.Stack)
	assert.Equal(t, "{\n  \"input\": {\n    \"jobId\": \"abc\",\n    \"n\": 1\n  }\n}", got.Fields)
	assert.Equal(t, entry.Time, got.Timestamp)
}

func TestCoreWriteNoFields(t *testing.T) {
	store, snapshot := recordingStore(t)
	core := NewCore(store)

	require.NoError(t, core.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "hi"}, nil))

	require.Eventually(t, func() bool {
		return len(snapshot()) == 1
	}, time.Second, time.Millisecond)

	got := snapshot()[0]
	assert.Empty(t, got.Fields)
	assert.Empty(t, got.Caller)
}

func TestCoreWithAccumulatesFields(t *testing.T) {
	store, snapshot := recordingStore(t)
	core := NewCore(store)

	child := core.With([]zapcore.Field{zap.String("service", "api")})
	require.NoError(t, child.Write(zapcore.Entry{Level: zapcore.InfoLevel, Message: "x"}, []zapcore.Field{
		zap.String("request", "123"),
	}))

	require.Eventually(t, func() bool {
		return len(snapshot()) == 1
	}, time.Second, time.Millisecond)

	got := snapshot()[0].Fields
	assert.Contains(t, got, `"service": "api"`)
	assert.Contains(t, got, `"request": "123"`)
}
