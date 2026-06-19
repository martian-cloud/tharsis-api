// Package noop implements a no-op log store that discards all entries.
package noop

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

type store struct{}

// New returns a Store that discards all writes and returns empty results.
func New() logstore.Store {
	return &store{}
}

func (s *store) IsNoop() bool               { return true }
func (s *store) Write(_ *logstore.LogEntry) {}

func (s *store) GetEntries(_ []string, _ string, _ int) ([]*logstore.LogEntry, error) {
	return nil, nil
}

func (s *store) Subscribe(_ context.Context) (<-chan *logstore.LogEntry, error) {
	ch := make(chan *logstore.LogEntry)
	close(ch)
	return ch, nil
}
