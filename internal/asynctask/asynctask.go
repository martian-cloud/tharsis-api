package asynctask

import "sync"

// Manager handles the lifecycle for async tasks
type Manager struct {
	wg sync.WaitGroup
}

// StartTask starts a new async task
func (a *Manager) StartTask(fn func()) {
	a.wg.Add(1)
	go func() {
		fn()
		a.wg.Done()
	}()
}

// Shutdown will wait for all async tasks to complete before shutting down
func (a *Manager) Shutdown() {
	a.wg.Wait()
}
