// Package asynctask package
package asynctask

//go:generate go tool mockery --name Manager --inpackage --case underscore

import (
	"context"
	"sync"
	"time"
)

// Manager handles the lifecycle for async tasks
type Manager interface {
	StartTask(fn func(ctx context.Context))
	Shutdown()
}

// Manager handles the lifecycle for async tasks
type manager struct {
	wg          sync.WaitGroup
	taskTimeout time.Duration
}

// NewManager creates a new Manager instance
func NewManager(taskTimeout time.Duration) Manager {
	return &manager{taskTimeout: taskTimeout}
}

// StartTask starts a new async task
func (m *manager) StartTask(fn func(ctx context.Context)) {
	m.wg.Add(1)
	go func() {
		taskCtx, cancel := context.WithTimeout(context.Background(), m.taskTimeout)
		defer cancel()

		fn(taskCtx)
		m.wg.Done()
	}()
}

// Shutdown will wait for all async tasks to complete before shutting down
func (m *manager) Shutdown() {
	m.wg.Wait()
}
