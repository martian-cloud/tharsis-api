package email

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var outboxesCleaned = metric.NewCounter("email_outbox_cleaned_total", "Number of ephemeral email outboxes reclaimed by the cleanup worker.")

const (
	// The cleanup worker polls infrequently since finished ephemeral outboxes aren't time-sensitive; the interval is jittered so instances don't all wake at once.
	minCleanupInterval = 30 * time.Minute
	maxCleanupInterval = 60 * time.Minute
	// cleanupBatchSize is how many outboxes one pass reclaims.
	cleanupBatchSize = 100
	// maxEphemeralRetentionDays caps the ephemeral retention window at one year.
	maxEphemeralRetentionDays = 365
)

// Cleaner is the background worker that reclaims finished ephemeral email outboxes.
type Cleaner struct {
	dbClient      *db.Client
	retentionDays int
}

// NewCleaner creates the ephemeral outbox cleanup worker.
func NewCleaner(dbClient *db.Client, retentionDays int) (*Cleaner, error) {
	if retentionDays < 0 || retentionDays > maxEphemeralRetentionDays {
		return nil, te.New("email ephemeral retention days must be between 0 and %d", maxEphemeralRetentionDays, te.WithErrorCode(te.EInvalid))
	}

	return &Cleaner{
		dbClient:      dbClient,
		retentionDays: retentionDays,
	}, nil
}

// cleanBatch claims and deletes one batch; the claim lease keeps other pollers off the rows, so no transaction is needed.
func (c *Cleaner) cleanBatch(ctx context.Context) (bool, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -c.retentionDays)
	outboxes, err := c.dbClient.EmailOutboxItems.ClaimOutboxItems(ctx, &db.ClaimOutboxItemsInput{
		Status:        new(models.EmailOutboxItemStatusCompleted),
		Ephemeral:     new(true),
		UpdatedBefore: &cutoff,
		Limit:         cleanupBatchSize,
	})
	if err != nil {
		return false, te.Wrap(err, "failed to claim reclaimable outboxes for cleanup")
	}

	if len(outboxes) == 0 {
		return false, nil
	}

	ids := make([]string, len(outboxes))
	for i, outbox := range outboxes {
		ids[i] = outbox.Metadata.ID
	}

	// A concurrent poller deleting some rows first surfaces as an optimistic lock error, which is the intended outcome, not a failure.
	if err = c.dbClient.EmailOutboxItems.DeleteOutboxItems(ctx, ids); err != nil && te.ErrorCode(err) != te.EOptimisticLock {
		return false, te.Wrap(err, "failed to delete ephemeral outboxes")
	}

	outboxesCleaned.Add(float64(len(ids)))

	// A full batch means more reclaimable rows may remain; ask to re-run instead of waiting for the poll.
	return len(ids) == cleanupBatchSize, nil
}
