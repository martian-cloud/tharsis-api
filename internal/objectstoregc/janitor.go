package objectstoregc

import (
	"context"
	"math/rand/v2"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

const (
	batchSize        = 1000
	minSleepSeconds  = 60
	maxSleepSeconds  = 120
	maintenanceSleep = 60
)

// Janitor periodically drives the Reclaimer to delete orphaned object store objects
// (owner FK nullified by a cascade delete) and their refs.
type Janitor struct {
	reclaimer          *Reclaimer
	logger             logger.Logger
	maintenanceMonitor maintenance.Monitor
}

// NewJanitor creates a new Janitor.
func NewJanitor(
	logger logger.Logger,
	dbClient *db.Client,
	objectStore objectstore.ObjectStore,
	maintenanceMonitor maintenance.Monitor,
) *Janitor {
	return &Janitor{
		logger:             logger,
		reclaimer:          NewReclaimer(dbClient.ObjectStoreRefs, objectStore, logger),
		maintenanceMonitor: maintenanceMonitor,
	}
}

// Start starts the janitor in the background.
func (j *Janitor) Start(ctx context.Context) {
	j.logger.Info("janitor started")

	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				j.logger.Info("janitor stopped")
				return
			case <-timer.C:
				if ctx.Err() != nil {
					return
				}

				sleepSeconds := maintenanceSleep

				inMaintenance, err := j.maintenanceMonitor.InMaintenanceMode(ctx)
				if err != nil {
					j.logger.Errorf("janitor failed to check maintenance mode: %v", err)
				}

				if err == nil && !inMaintenance {
					if err := j.reclaimer.Reclaim(ctx, batchSize); err != nil {
						j.logger.Errorf("janitor failed to reclaim orphaned objects: %v", err)
					}
					sleepSeconds = rand.IntN(maxSleepSeconds-minSleepSeconds) + minSleepSeconds
				}

				timer.Reset(time.Duration(sleepSeconds) * time.Second)
			}
		}
	}()
}
