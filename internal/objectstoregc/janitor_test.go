package objectstoregc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pkgobjectstore "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

func TestJanitorStart(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*maintenance.MockMonitor, *db.MockObjectStoreRefs) <-chan struct{}
	}{
		{
			name: "not in maintenance runs Reclaim",
			setupMocks: func(mon *maintenance.MockMonitor, refs *db.MockObjectStoreRefs) <-chan struct{} {
				done := make(chan struct{})
				mon.On("InMaintenanceMode", mock.Anything).Return(false, nil).Once()
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(batchSize)).
					Run(func(_ mock.Arguments) { close(done) }).
					Return(nil, nil).Once()
				return done
			},
		},
		{
			name: "in maintenance mode skips Reclaim",
			setupMocks: func(mon *maintenance.MockMonitor, _ *db.MockObjectStoreRefs) <-chan struct{} {
				done := make(chan struct{})
				mon.On("InMaintenanceMode", mock.Anything).
					Run(func(_ mock.Arguments) { close(done) }).
					Return(true, nil).Once()
				return done
			},
		},
		{
			name: "maintenance check error skips Reclaim",
			setupMocks: func(mon *maintenance.MockMonitor, _ *db.MockObjectStoreRefs) <-chan struct{} {
				done := make(chan struct{})
				mon.On("InMaintenanceMode", mock.Anything).
					Run(func(_ mock.Arguments) { close(done) }).
					Return(false, errors.New("db error")).Once()
				return done
			},
		},
		{
			name: "Reclaim error is logged and does not crash the janitor",
			setupMocks: func(mon *maintenance.MockMonitor, refs *db.MockObjectStoreRefs) <-chan struct{} {
				done := make(chan struct{})
				mon.On("InMaintenanceMode", mock.Anything).Return(false, nil).Once()
				refs.On("ClaimOrphanedRefs", mock.Anything, uint(batchSize)).
					Run(func(_ mock.Arguments) { close(done) }).
					Return(nil, errors.New("db error")).Once()
				return done
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			mockMaintenance := maintenance.NewMockMonitor(t)
			mockRefs := db.NewMockObjectStoreRefs(t)
			mockStore := pkgobjectstore.NewMockObjectStore(t)

			done := tc.setupMocks(mockMaintenance, mockRefs)

			logr, _ := logger.NewForTest()
			j := NewJanitor(logr, &db.Client{ObjectStoreRefs: mockRefs}, mockStore, mockMaintenance)
			j.Start(ctx)

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("janitor did not complete first iteration in time")
			}
			cancel()
		})
	}
}

func TestJanitorStartContextCancel(t *testing.T) {
	// Verify the goroutine exits when the context is cancelled before the first tick.
	// We use a context that's already cancelled so the goroutine's select picks ctx.Done()
	// instead of the timer.
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancelled before Start

	mockMaintenance := maintenance.NewMockMonitor(t)
	mockRefs := db.NewMockObjectStoreRefs(t)
	mockStore := pkgobjectstore.NewMockObjectStore(t)
	// No mock expectations -- if the goroutine somehow calls InMaintenanceMode the mock
	// would panic with "unexpected call", which is exactly what we want to detect.

	logr, _ := logger.NewForTest()
	j := NewJanitor(logr, &db.Client{ObjectStoreRefs: mockRefs}, mockStore, mockMaintenance)
	j.Start(ctx)

	// Give the goroutine time to exit; no assertions beyond the mock having no unexpected calls.
	time.Sleep(50 * time.Millisecond)
}
