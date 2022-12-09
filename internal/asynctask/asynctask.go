package asynctask

//go:generate mockery --name Manager --inpackage --case underscore

import "sync"

// Manager handles the lifecycle for async tasks
type Manager interface {
	StartTask(fn func())
	Shutdown()
}

type manager struct {
	wg sync.WaitGroup
}

// NewManager returns an instance of Manager interface.
func NewManager() Manager {
	return &manager{}
}

// StartTask starts a new async task
func (a *manager) StartTask(fn func()) {
	a.wg.Add(1)
	go func() {
		fn()
		a.wg.Done()
	}()
}

// Shutdown will wait for all async tasks to complete before shutting down
func (a *manager) Shutdown() {
	a.wg.Wait()
}
