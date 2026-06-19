package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

func newTestStore(capacity int) *store {
	return &store{
		buffer:      make([]*logstore.LogEntry, capacity),
		capacity:    capacity,
		subscribers: make(map[string]chan *logstore.LogEntry),
	}
}

func messages(entries []*logstore.LogEntry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Message
	}
	return out
}

func TestStoreGetEntriesNewestFirst(t *testing.T) {
	s := newTestStore(5)
	s.Write(&logstore.LogEntry{Level: "INFO", Message: "a"})
	s.Write(&logstore.LogEntry{Level: "INFO", Message: "b"})
	s.Write(&logstore.LogEntry{Level: "INFO", Message: "c"})

	got, err := s.GetEntries(nil, "", 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b", "a"}, messages(got))
}

func TestStoreGetEntriesLimit(t *testing.T) {
	testCases := []struct {
		name  string
		limit int
		want  []string
	}{
		{name: "limit fewer than count", limit: 2, want: []string{"c", "b"}},
		{name: "limit exceeds count", limit: 10, want: []string{"c", "b", "a"}},
		{name: "limit one", limit: 1, want: []string{"c"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStore(5)
			s.Write(&logstore.LogEntry{Message: "a"})
			s.Write(&logstore.LogEntry{Message: "b"})
			s.Write(&logstore.LogEntry{Message: "c"})

			got, err := s.GetEntries(nil, "", tc.limit)
			require.NoError(t, err)
			assert.Equal(t, tc.want, messages(got))
		})
	}
}

func TestStoreGetEntriesEmptyOrZeroLimit(t *testing.T) {
	s := newTestStore(5)

	got, err := s.GetEntries(nil, "", 10)
	require.NoError(t, err)
	assert.Nil(t, got, "empty store returns nil")

	s.Write(&logstore.LogEntry{Message: "a"})
	got, err = s.GetEntries(nil, "", 0)
	require.NoError(t, err)
	assert.Nil(t, got, "non-positive limit returns nil")
}

func TestStoreRingBufferWraparound(t *testing.T) {
	s := newTestStore(3)
	for _, m := range []string{"1", "2", "3", "4", "5"} {
		s.Write(&logstore.LogEntry{Message: m})
	}

	got, err := s.GetEntries(nil, "", 10)
	require.NoError(t, err)
	// Only the last 3 survive, newest first.
	assert.Equal(t, []string{"5", "4", "3"}, messages(got))
}

func TestStoreFiltering(t *testing.T) {
	s := newTestStore(10)
	s.Write(&logstore.LogEntry{Level: "INFO", Message: "connect ok"})
	s.Write(&logstore.LogEntry{Level: "ERROR", Message: "connect failed"})
	s.Write(&logstore.LogEntry{Level: "INFO", Message: "shutdown"})

	t.Run("by level", func(t *testing.T) {
		got, err := s.GetEntries([]string{"ERROR"}, "", 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"connect failed"}, messages(got))
	})

	t.Run("by search", func(t *testing.T) {
		got, err := s.GetEntries(nil, "connect", 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"connect failed", "connect ok"}, messages(got))
	})

	t.Run("by level and search", func(t *testing.T) {
		got, err := s.GetEntries([]string{"INFO"}, "connect", 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"connect ok"}, messages(got))
	})
}

func TestStoreSeqIncrements(t *testing.T) {
	s := newTestStore(5)
	s.Write(&logstore.LogEntry{Message: "a"})
	s.Write(&logstore.LogEntry{Message: "b"})

	got, err := s.GetEntries(nil, "", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, uint64(2), got[0].Seq)
	assert.Equal(t, uint64(1), got[1].Seq)
}

func TestStoreSubscribeReceives(t *testing.T) {
	s := newTestStore(5)
	ch, err := s.Subscribe(t.Context())
	require.NoError(t, err)

	s.Write(&logstore.LogEntry{Message: "live"})

	select {
	case got := <-ch:
		assert.Equal(t, "live", got.Message)
	case <-time.After(time.Second):
		t.Fatal("expected a broadcast entry")
	}
}

func TestStoreMultipleSubscribers(t *testing.T) {
	s := newTestStore(5)
	ch1, _ := s.Subscribe(t.Context())
	ch2, _ := s.Subscribe(t.Context())

	s.Write(&logstore.LogEntry{Message: "x"})

	for _, ch := range []<-chan *logstore.LogEntry{ch1, ch2} {
		select {
		case got := <-ch:
			assert.Equal(t, "x", got.Message)
		case <-time.After(time.Second):
			t.Fatal("subscriber did not receive entry")
		}
	}
}

func TestStoreSlowSubscriberDropped(t *testing.T) {
	s := newTestStore(5)

	// A full subscriber channel: a broadcast must drop rather than block.
	full := make(chan *logstore.LogEntry, 1)
	full <- &logstore.LogEntry{Message: "prefill"}
	s.subscribers["slow"] = full

	done := make(chan struct{})
	go func() {
		s.Write(&logstore.LogEntry{Message: "new"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Write blocked on a full subscriber")
	}

	assert.Len(t, full, 1, "the new entry should have been dropped, not queued")
}

func TestStoreContextCancelClosesChannel(t *testing.T) {
	s := newTestStore(5)
	ctx, cancel := context.WithCancel(t.Context())
	ch, _ := s.Subscribe(ctx)

	cancel()

	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after context cancellation")

	// A subsequent write must not panic (subscriber was removed before close).
	assert.NotPanics(t, func() {
		s.Write(&logstore.LogEntry{Message: "after"})
	})
}
