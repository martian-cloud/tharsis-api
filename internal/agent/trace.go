package agent

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/gollem/trace"
)

// traceRepository implements trace.Repository by persisting traces to the Store.
type traceRepository struct {
	store     Store
	sessionID string
}

func newTraceRepository(store Store, sessionID string) trace.Repository {
	return &traceRepository{store: store, sessionID: sessionID}
}

func (r *traceRepository) Save(ctx context.Context, t *trace.Trace) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return r.store.UploadTrace(ctx, r.sessionID, t.TraceID, data)
}
