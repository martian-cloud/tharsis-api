package logstream

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// compactionClaimTTL bounds how long a started-but-unfinished compaction blocks other instances from
// retrying a stream. A compaction abandoned by a crashed instance becomes reclaimable after this
// window. It is generously larger than any expected compaction so a still-running compaction is never
// reclaimed from underneath the instance doing the work.
const compactionClaimTTL = 10 * time.Minute

// compactionInterval is the base interval between compaction scheduler runs. Each run sleeps a random
// duration in [interval, 2*interval) to spread load across horizontally-scaled instances.
const compactionInterval = 1 * time.Minute

// compactionBatchSize is the number of completed streams compacted per scheduler run.
const compactionBatchSize = 100

// CompactionScheduler periodically consolidates completed-but-uncompacted log streams into a single
// static-key object so future reads are a single GET with no chunk stitching.
type CompactionScheduler struct {
	dbClient           *db.Client
	logger             logger.Logger
	manager            Manager
	maintenanceMonitor maintenance.Monitor
}

// NewCompactionScheduler creates a new CompactionScheduler.
func NewCompactionScheduler(
	dbClient *db.Client,
	logger logger.Logger,
	manager Manager,
	maintenanceMonitor maintenance.Monitor,
) *CompactionScheduler {
	return &CompactionScheduler{
		dbClient:           dbClient,
		logger:             logger,
		manager:            manager,
		maintenanceMonitor: maintenanceMonitor,
	}
}

// Start starts the compaction scheduler.
func (c *CompactionScheduler) Start(ctx context.Context) {
	c.logger.Info("log stream compaction scheduler started")

	go func() {
		for {
			// Randomize the sleep within [interval, 2*interval) to spread load across instances.
			sleep := compactionInterval + time.Duration(rand.Int64N(int64(compactionInterval)))

			select {
			case <-time.After(sleep):
				if err := c.execute(ctx); err != nil {
					c.logger.Error(err)
				}
			case <-ctx.Done():
				c.logger.Info("log stream compaction scheduler stopped")
				return
			}
		}
	}()
}

func (c *CompactionScheduler) execute(ctx context.Context) error {
	inMaintenance, err := c.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for maintenance mode in compaction scheduler: %w", err)
	}
	if inMaintenance {
		return nil
	}

	// Atomically claim a batch of completed, not-yet-compacted streams. The claim stamps each stream
	// (SELECT ... FOR UPDATE SKIP LOCKED) so concurrent instances grab disjoint batches and stale
	// claims left by a crashed instance become reclaimable after the TTL. Compacting a stream marks it
	// compacted and removes it from the claimable set, so the next run naturally picks up the next
	// batch (no cursor needed).
	claimed, err := c.dbClient.LogStreams.ClaimLogStreamsForCompaction(ctx, compactionBatchSize, time.Now().Add(-compactionClaimTTL))
	if err != nil {
		return fmt.Errorf("failed to claim log streams in compaction scheduler: %w", err)
	}

	for i := range claimed {
		ls := &claimed[i]
		if err := c.manager.CompactStream(ctx, ls); err != nil {
			// Isolate per-stream failures; a still-uncompacted stream is retried next run.
			c.logger.Errorf("failed to compact log stream %s: %v", ls.Metadata.ID, err)
		}
	}

	return nil
}
