// Package memory implements an in-memory ring buffer log store.
package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

const defaultCapacity = 2000

// store holds the ring buffer and active live-tail subscribers.
type store struct {
	mu          sync.Mutex
	buffer      []*logstore.LogEntry
	capacity    int
	head        int    // next write position (wraps)
	count       int    // number of valid entries (≤ capacity)
	seq         uint64 // monotonically increasing counter
	subscribers map[string]chan *logstore.LogEntry
}

// New creates an in-memory Store.
func New() (logstore.Store, error) {
	return &store{
		buffer:      make([]*logstore.LogEntry, defaultCapacity),
		capacity:    defaultCapacity,
		subscribers: make(map[string]chan *logstore.LogEntry),
	}, nil
}

// Write appends an entry to the ring buffer and broadcasts to all active subscribers.
func (s *store) Write(e *logstore.LogEntry) {
	s.mu.Lock()
	s.write(e)
	s.mu.Unlock()
}

// GetEntries returns up to limit entries in newest-first order.
// levels and search are optional filters (empty levels matches all).
func (s *store) GetEntries(levels []string, search string, limit int) ([]*logstore.LogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || s.count == 0 {
		return nil, nil
	}

	result := make([]*logstore.LogEntry, 0, limit)

	start := (s.head - 1 + s.capacity) % s.capacity
	for i := 0; i < s.count && len(result) < limit; i++ {
		idx := (start - i + s.capacity) % s.capacity
		e := s.buffer[idx]
		if e == nil {
			continue
		}
		if !e.Matches(levels, search) {
			continue
		}
		result = append(result, e)
	}

	return result, nil
}

// Subscribe registers a live-tail subscriber and returns a buffered channel that
// is closed (and deregistered) when ctx is cancelled.
func (s *store) Subscribe(ctx context.Context) (<-chan *logstore.LogEntry, error) {
	ch := make(chan *logstore.LogEntry, 256)
	id := uuid.New().String()

	s.mu.Lock()
	s.subscribers[id] = ch
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		delete(s.subscribers, id)
		close(ch)
		s.mu.Unlock()
	}()

	return ch, nil
}

// write appends an entry and broadcasts to subscribers. Slow subscribers are
// skipped (non-blocking send). Called with s.mu held.
func (s *store) write(e *logstore.LogEntry) {
	s.seq++
	e.Seq = s.seq

	s.buffer[s.head] = e
	s.head = (s.head + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}

	for _, ch := range s.subscribers {
		select {
		case ch <- e:
		default:
		}
	}
}
