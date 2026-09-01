package cleanup

//go:generate go tool mockery --name pruner --inpackage --case underscore

import (
	"context"
	"errors"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
)

const (
	// deleteChunkSize is the number of resources deleted in a single statement.
	deleteChunkSize = 100
	// sweepPageSize is the number of resources fetched per paginated batch in a sweep pass.
	sweepPageSize = 500
)

// pageSleepInterval is the delay between paginated batches within a single pass.
var pageSleepInterval = 10 * time.Second

// onDeleteFn is called after each chunk of resources has been deleted, with their TRNs.
type onDeleteFn func(trns ...string)

// pruner deletes the resources that one kind of cleanup policy marks stale.
type pruner interface {
	prune(ctx context.Context, ns namespace.Namespace, policy *models.CleanupPolicy, onDelete onDeleteFn) error
}

// deleteInChunks deletes candidates in chunks of deleteChunkSize, calling onDelete with the TRNs actually deleted.
func deleteInChunks[T any](
	ctx context.Context,
	candidates []T,
	idOf func(T) string,
	trnOf func(T) string,
	deleteBatch func(context.Context, []T) ([]string, error),
	onDelete onDeleteFn,
) error {
	for start := 0; start < len(candidates); start += deleteChunkSize {
		end := min(start+deleteChunkSize, len(candidates))
		chunk := candidates[start:end]

		deletedIDs, err := deleteBatch(ctx, chunk)
		if err != nil && !errors.Is(err, db.ErrOptimisticLockError) {
			// An OLE is expected here since the candidate may have been updated since queried.
			return err
		}

		deleted := make(map[string]struct{}, len(deletedIDs))
		for _, id := range deletedIDs {
			deleted[id] = struct{}{}
		}

		var trns []string
		for _, c := range chunk {
			if _, ok := deleted[idOf(c)]; ok {
				trns = append(trns, trnOf(c))
			}
		}

		onDelete(trns...)
	}

	return nil
}
