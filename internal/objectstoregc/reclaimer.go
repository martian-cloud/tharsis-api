package objectstoregc

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pkgobjectstore "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// maxClaimCount is the number of times a ref may be claimed and fail to delete before it is
// discarded (best-effort deleted, then the ref removed) to avoid a poison ref blocking the queue.
const maxClaimCount = 5

var (
	refsDeleted         = metric.NewCounter("janitor_refs_deleted_total", "Number of orphaned object store refs successfully deleted.")
	refsDeadLetterCount = metric.NewCounter("janitor_refs_dead_letter_total", "Number of refs discarded after exceeding max claim count.")
	objectDeleteErrors  = metric.NewCounter("janitor_object_delete_errors_total", "Number of object store delete failures.")
)

// Reclaimer deletes the objects behind orphaned object_store_refs (whose owner FK was nulled by a
// cascade delete) and removes the refs. It is the cleanup half of ref tracking; a scheduler (the
// janitor) drives it.
type Reclaimer struct {
	refs   db.ObjectStoreRefs
	store  pkgobjectstore.ObjectStore
	logger logger.Logger
}

// NewReclaimer returns a Reclaimer that deletes orphaned objects from store and their refs.
func NewReclaimer(refs db.ObjectStoreRefs, store pkgobjectstore.ObjectStore, logger logger.Logger) *Reclaimer {
	return &Reclaimer{refs: refs, store: store, logger: logger}
}

// Reclaim claims up to limit orphaned refs, batch-deletes their objects, then removes the refs. A
// batch delete failure leaves that batch's refs for a later retry (leases expire); a ref that
// exceeds maxClaimCount is discarded after a best-effort delete so a poison key can't block the queue.
func (r *Reclaimer) Reclaim(ctx context.Context, limit uint) error {
	refs, err := r.refs.ClaimOrphanedRefs(ctx, limit)
	if err != nil {
		return err
	}

	batch := make([]db.ObjectStoreRef, 0, len(refs))
	keys := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if ref.ClaimCount > maxClaimCount {
			// Best-effort final delete before we give up tracking this dead-lettered ref.
			if err := r.store.DeleteObjects(ctx, []string{ref.ObjectKey}); err != nil {
				r.logger.Errorf("reclaimer best-effort delete failed for dead-lettered ref %q (%s): %v", ref.ID, ref.ObjectKey, err)
			}
			if err := r.refs.DeleteRefs(ctx, []string{ref.ID}); err != nil {
				r.logger.Errorf("reclaimer failed to discard ref %q: %v", ref.ID, err)
			} else {
				refsDeadLetterCount.Inc()
			}
			continue
		}

		batch = append(batch, ref)
		keys = append(keys, ref.ObjectKey)
	}

	if len(keys) == 0 {
		return nil
	}

	if err := r.store.DeleteObjects(ctx, keys); err != nil {
		r.logger.Errorf("reclaimer batch delete failed for %d objects, retrying next cycle: %v", len(keys), err)
		objectDeleteErrors.Inc()
		return nil
	}

	ids := make([]string, len(batch))
	for i, ref := range batch {
		ids[i] = ref.ID
	}
	if err := r.refs.DeleteRefs(ctx, ids); err != nil {
		r.logger.Errorf("reclaimer failed to delete %d refs: %v", len(ids), err)
	} else {
		refsDeleted.Add(float64(len(ids)))
	}

	return nil
}
