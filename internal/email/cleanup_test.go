package email

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func outboxesWithIDs(ids ...string) []*models.EmailOutboxItem {
	outboxes := make([]*models.EmailOutboxItem, len(ids))
	for i, id := range ids {
		o := &models.EmailOutboxItem{}
		o.Metadata.ID = id
		outboxes[i] = o
	}
	return outboxes
}

func TestCleanerCleanBatch(t *testing.T) {
	const testRetentionDays = 7

	// The claim cutoff is time.Now()-based, so match on the fixed fields plus a cutoff at least a few
	// days in the past rather than an exact struct.
	claimInput := mock.MatchedBy(func(in *db.ClaimOutboxItemsInput) bool {
		return in.Status != nil && *in.Status == models.EmailOutboxItemStatusCompleted &&
			in.Ephemeral != nil && *in.Ephemeral &&
			in.Limit == cleanupBatchSize &&
			in.UpdatedBefore != nil &&
			time.Since(*in.UpdatedBefore) > 6*24*time.Hour
	})

	tests := []struct {
		name             string
		retentionDays    int
		setupMocks       func(*db.MockEmailOutboxItems)
		wantErr          bool
		wantConstructErr bool
	}{
		{
			name:             "retention days above the max is a construction error",
			retentionDays:    maxEphemeralRetentionDays + 1,
			setupMocks:       func(_ *db.MockEmailOutboxItems) {},
			wantConstructErr: true,
		},
		{
			name:             "negative retention days is a construction error",
			retentionDays:    -1,
			setupMocks:       func(_ *db.MockEmailOutboxItems) {},
			wantConstructErr: true,
		},
		{
			name:          "reclaims returned outboxes",
			retentionDays: testRetentionDays,
			setupMocks: func(outbox *db.MockEmailOutboxItems) {
				outbox.On("ClaimOutboxItems", mock.Anything, claimInput).Return(outboxesWithIDs("outbox-1", "outbox-2"), nil)
				outbox.On("DeleteOutboxItems", mock.Anything, []string{"outbox-1", "outbox-2"}).Return(nil)
			},
		},
		{
			name:          "no cleanable outboxes is a no-op that does not delete",
			retentionDays: testRetentionDays,
			setupMocks: func(outbox *db.MockEmailOutboxItems) {
				outbox.On("ClaimOutboxItems", mock.Anything, claimInput).Return(nil, nil)
				// No DeleteOutboxItems: the strict mock fails if it's called.
			},
		},
		{
			name:          "ignores optimistic lock error from a concurrent poller",
			retentionDays: testRetentionDays,
			setupMocks: func(outbox *db.MockEmailOutboxItems) {
				outbox.On("ClaimOutboxItems", mock.Anything, claimInput).Return(outboxesWithIDs("outbox-1"), nil)
				outbox.On("DeleteOutboxItems", mock.Anything, mock.Anything).Return(db.ErrOptimisticLockError)
			},
		},
		{
			name:          "claim error is returned",
			retentionDays: testRetentionDays,
			setupMocks: func(outbox *db.MockEmailOutboxItems) {
				outbox.On("ClaimOutboxItems", mock.Anything, claimInput).Return(nil, errors.New("claim failed"))
			},
			wantErr: true,
		},
		{
			name:          "non-lock delete error is returned",
			retentionDays: testRetentionDays,
			setupMocks: func(outbox *db.MockEmailOutboxItems) {
				outbox.On("ClaimOutboxItems", mock.Anything, claimInput).Return(outboxesWithIDs("outbox-1"), nil)
				outbox.On("DeleteOutboxItems", mock.Anything, mock.Anything).Return(errors.New("delete failed"))
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockOutboxItem := db.NewMockEmailOutboxItems(t)

			test.setupMocks(mockOutboxItem)

			cleaner, err := NewCleaner(&db.Client{
				EmailOutboxItems: mockOutboxItem,
			}, test.retentionDays)

			// Out-of-range retention fails at construction, before any batch work.
			if test.wantConstructErr {
				assert.Error(t, err)
				assert.Nil(t, cleaner)
				return
			}
			require.NoError(t, err)

			_, err = cleaner.cleanBatch(context.Background())

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewCleaner(t *testing.T) {
	t.Run("valid retention", func(t *testing.T) {
		cleaner, err := NewCleaner(&db.Client{}, 7)
		require.NoError(t, err)
		assert.NotNil(t, cleaner)
	})

	t.Run("retention above the max is an error", func(t *testing.T) {
		cleaner, err := NewCleaner(&db.Client{}, maxEphemeralRetentionDays+1)
		assert.Error(t, err)
		assert.Nil(t, cleaner)
	})
}
