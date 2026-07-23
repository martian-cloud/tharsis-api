// Package objectstoregc tracks uploaded objects in object_store_refs and reclaims orphaned ones
// with a background janitor. It provides a tracked object store decorator (New) and a scheduler
// (NewJanitor) that drives the Reclaimer.
package objectstoregc

import (
	"context"
	"io"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	pkgobjectstore "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/objectstore"
)

// pendingRefGracePeriod is how long a pending ref (no FK yet) is shielded from the janitor.
// Upload sites must call LinkRef within this window to attach the owner FK.
const pendingRefGracePeriod = 5 * time.Minute

// store decorates an object store, recording a pending object_store_refs row before each upload.
type store struct {
	pkgobjectstore.ObjectStore
	refs db.ObjectStoreRefs
}

// New wraps raw so that every upload first records a pending object_store_refs row. The upload site
// must call LinkRef within pendingRefGracePeriod to attach the owner FK; until then the ref is
// shielded from the janitor.
func New(raw pkgobjectstore.ObjectStore, refs db.ObjectStoreRefs) pkgobjectstore.ObjectStore {
	return &store{ObjectStore: raw, refs: refs}
}

func (t *store) UploadObject(ctx context.Context, key string, body io.Reader) error {
	availAt := time.Now().UTC().Add(pendingRefGracePeriod)
	if err := t.refs.CreateRef(ctx, &db.CreateObjectStoreRefInput{
		ObjectKey:   key,
		AvailableAt: &availAt,
	}); err != nil {
		return err
	}

	return t.ObjectStore.UploadObject(ctx, key, body)
}
