package agent

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/gollem"
)

// historyRepository implements gollem.HistoryRepository using the Store.
type historyRepository struct {
	store Store
}

func newHistoryRepository(store Store) gollem.HistoryRepository {
	return &historyRepository{store: store}
}

func (r *historyRepository) Load(ctx context.Context, sessionID string) (*gollem.History, error) {
	data, err := r.store.LoadHistory(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var h gollem.History
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *historyRepository) Save(ctx context.Context, sessionID string, history *gollem.History) error {
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}
	return r.store.SaveHistory(ctx, sessionID, data)
}
