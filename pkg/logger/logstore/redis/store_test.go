package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawFields builds a Redis stream field slice ([]byte key/value pairs) from strings.
func rawFields(kv ...string) []any {
	out := make([]any, 0, len(kv))
	for _, s := range kv {
		out = append(out, []byte(s))
	}
	return out
}

func TestNewValidation(t *testing.T) {
	testCases := []struct {
		name       string
		pluginData map[string]string
		expectErr  bool
		wantPrefix string
		wantMax    int
	}{
		{
			name:       "missing endpoint",
			pluginData: map[string]string{},
			expectErr:  true,
		},
		{
			name:       "non-numeric max_entries",
			pluginData: map[string]string{"redis_endpoint": "redis://127.0.0.1:6390", "max_entries": "abc"},
			expectErr:  true,
		},
		{
			name:       "zero max_entries",
			pluginData: map[string]string{"redis_endpoint": "redis://127.0.0.1:6390", "max_entries": "0"},
			expectErr:  true,
		},
		{
			name:       "defaults applied",
			pluginData: map[string]string{"redis_endpoint": "redis://127.0.0.1:6390"},
			wantPrefix: defaultKeyPrefix,
			wantMax:    defaultMaxEntries,
		},
		{
			name:       "overrides applied",
			pluginData: map[string]string{"redis_endpoint": "redis://127.0.0.1:6390", "key_prefix": "myapp:logs", "max_entries": "500"},
			wantPrefix: "myapp:logs",
			wantMax:    500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// t.Context() is cancelled when the test ends, stopping the background writer.
			s, err := New(t.Context(), tc.pluginData)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, s)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, s)

			st := s.(*store)
			assert.Equal(t, tc.wantPrefix, st.streamKey)
			assert.Equal(t, tc.wantMax, st.maxEntries)
			assert.Equal(t, tc.pluginData["redis_endpoint"], st.endpoint)
		})
	}
}

func TestParseStreamID(t *testing.T) {
	testCases := []struct {
		id     string
		ms, n  uint64
		wantOk bool
	}{
		{id: "1700000000000-0", ms: 1700000000000, n: 0, wantOk: true},
		{id: "5-3", ms: 5, n: 3, wantOk: true},
		{id: "noseparator", wantOk: false},
		{id: "abc-0", wantOk: false},
		{id: "5-xyz", wantOk: false},
		{id: "", wantOk: false},
	}

	for _, tc := range testCases {
		t.Run(tc.id, func(t *testing.T) {
			ms, n, ok := parseStreamID(tc.id)
			assert.Equal(t, tc.wantOk, ok)
			if tc.wantOk {
				assert.Equal(t, tc.ms, ms)
				assert.Equal(t, tc.n, n)
			}
		})
	}
}

func TestStreamIDToSeq(t *testing.T) {
	assert.Equal(t, uint64(5<<20|3), streamIDToSeq("5-3"))
	assert.Equal(t, uint64(1700000000000<<20), streamIDToSeq("1700000000000-0"))
	assert.Equal(t, uint64(0), streamIDToSeq("invalid"))
}

func TestStreamEntryFields(t *testing.T) {
	t.Run("valid shape", func(t *testing.T) {
		id, fields, ok := streamEntryFields([]any{[]byte("5-0"), []any{[]byte("k"), []byte("v")}})
		require.True(t, ok)
		assert.Equal(t, "5-0", id)
		assert.Len(t, fields, 2)
	})

	testCases := []struct {
		name string
		in   any
	}{
		{name: "not a slice", in: "x"},
		{name: "wrong outer length", in: []any{[]byte("5-0")}},
		{name: "id not bytes", in: []any{"5-0", []any{}}},
		{name: "fields not slice", in: []any{[]byte("5-0"), "x"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, ok := streamEntryFields(tc.in)
			assert.False(t, ok)
		})
	}
}

func TestParseStreamEntry(t *testing.T) {
	t.Run("full entry", func(t *testing.T) {
		fields := rawFields(
			"level", "INFO",
			"message", "hi",
			"caller", "f.go:1",
			"stack", "st",
			"fields", "{}",
			"timestamp", "2026-06-11T01:02:03Z",
		)

		e := parseStreamEntry("5-3", fields)
		require.NotNil(t, e)
		assert.Equal(t, uint64(5<<20|3), e.Seq)
		assert.Equal(t, "INFO", e.Level)
		assert.Equal(t, "hi", e.Message)
		assert.Equal(t, "f.go:1", e.Caller)
		assert.Equal(t, "st", e.Stack)
		assert.Equal(t, "{}", e.Fields)
		assert.Equal(t, time.Date(2026, 6, 11, 1, 2, 3, 0, time.UTC), e.Timestamp.UTC())
	})

	t.Run("level and message both empty returns nil", func(t *testing.T) {
		assert.Nil(t, parseStreamEntry("5-0", rawFields("caller", "f.go:1")))
	})

	t.Run("only level present is kept", func(t *testing.T) {
		e := parseStreamEntry("5-0", rawFields("level", "ERROR"))
		require.NotNil(t, e)
		assert.Equal(t, "ERROR", e.Level)
		assert.True(t, e.Timestamp.IsZero(), "missing timestamp leaves zero value")
	})
}
