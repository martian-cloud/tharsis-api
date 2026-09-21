package email

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// fixedSendAt is a stable SendAt used to assert propagation onto the outbox item.
var fixedSendAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

type enqueuerMocks struct {
	outbox       *db.MockEmailOutboxItems
	transactions *db.MockTransactions
	store        *MockStore
}

func newEnqueuerMocks(t *testing.T) *enqueuerMocks {
	return &enqueuerMocks{
		outbox:       db.NewMockEmailOutboxItems(t),
		transactions: db.NewMockTransactions(t),
		store:        NewMockStore(t),
	}
}

func (m *enqueuerMocks) newEnqueuer(_ *testing.T) *enqueuer {
	testLogger, _ := logger.NewForTest()
	return &enqueuer{
		dbClient: &db.Client{
			EmailOutboxItems: m.outbox,
			Transactions:     m.transactions,
		},
		store:  m.store,
		logger: testLogger,
	}
}

// largeBuilder returns a builder whose serialized payload exceeds maxInlineRowSize, forcing the object-store path.
func largeBuilder() builder.EmailBuilder {
	return &builder.AnnouncementEmail{Message: strings.Repeat("x", maxInlineRowSize+1)}
}

// manyRecipientIDs returns enough recipient IDs that their combined row footprint alone exceeds maxInlineRowSize.
func manyRecipientIDs() []string {
	ids := make([]string, 0, 100)
	for i := 0; i < 100; i++ {
		ids = append(ids, strings.Repeat("a", uuidStringSize))
	}

	return ids
}

func TestNewEnqueuer(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	dbClient := &db.Client{}
	store := NewMockStore(t)

	e := NewEnqueuer(dbClient, store, testLogger)
	assert.NotNil(t, e)
}

func TestEnqueuerEnqueueEmail(t *testing.T) {
	tests := []struct {
		name       string
		input      *EnqueueEmailInput
		setupMocks func(*enqueuerMocks)
		wantErr    bool
	}{
		{
			name:    "nil builder is an error",
			input:   &EnqueueEmailInput{},
			wantErr: true,
		},
		{
			name:  "upload error is returned before any transaction",
			input: &EnqueueEmailInput{Builder: largeBuilder(), Subject: "hi", UserIDs: []string{"u1"}},
			setupMocks: func(m *enqueuerMocks) {
				m.store.On("UploadPayload", mock.Anything, mock.Anything).Return(nil, "", errors.New("upload failed"))
			},
			wantErr: true,
		},
		{
			name: "small payload is stored inline without an upload and commits",
			input: &EnqueueEmailInput{
				Builder: &builder.AnnouncementEmail{},
				Subject: "hi",
				UserIDs: []string{"u1", "u2"},
				TeamIDs: []string{"t1"},
			},
			setupMocks: func(m *enqueuerMocks) {
				// No UploadPayload: a small payload goes inline, so the strict store mock fails if it's called.
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)

				created := &models.EmailOutboxItem{}
				created.Metadata.ID = "outbox-1"
				m.outbox.On("CreateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
					// Payload is inline, no object key, ephemeral (default), recipients are the spec.
					return o.Subject == "hi" &&
						len(o.Payload) > 0 &&
						o.PayloadObjectStoreKey == nil &&
						o.Ephemeral &&
						o.Status == models.EmailOutboxItemStatusPreparing &&
						len(o.RecipientUserIDs) == 2 &&
						len(o.RecipientTeamIDs) == 1 &&
						!o.SendToAllUsers
				})).Return(created, nil)

				m.transactions.On("CommitTx", mock.Anything).Return(nil)
			},
		},
		{
			name: "large payload uploads to object storage and links the ref before commit",
			input: &EnqueueEmailInput{
				Builder: largeBuilder(),
				Subject: "big",
			},
			setupMocks: func(m *enqueuerMocks) {
				retained := false
				retain := db.RetainObjectRefFunc(func(_ context.Context, _ string) error {
					retained = true
					return nil
				})
				m.store.On("UploadPayload", mock.Anything, mock.Anything).Return(retain, "emails/x/payload.json", nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)

				created := &models.EmailOutboxItem{}
				created.Metadata.ID = "outbox-1"
				m.outbox.On("CreateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
					// Payload lives in object storage: no inline payload, key is set.
					return len(o.Payload) == 0 && o.PayloadObjectStoreKey != nil && *o.PayloadObjectStoreKey == "emails/x/payload.json"
				})).Return(created, nil)

				m.transactions.On("CommitTx", mock.Anything).Return(nil).
					Run(func(_ mock.Arguments) { assert.True(t, retained, "retain func must run before commit") })
			},
		},
		{
			name: "large recipient arrays with a small payload still offload to object storage",
			input: &EnqueueEmailInput{
				Builder: &builder.AnnouncementEmail{},
				Subject: "many recipients",
				UserIDs: manyRecipientIDs(),
			},
			setupMocks: func(m *enqueuerMocks) {
				retain := db.RetainObjectRefFunc(func(_ context.Context, _ string) error { return nil })
				m.store.On("UploadPayload", mock.Anything, mock.Anything).Return(retain, "emails/x/payload.json", nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)

				created := &models.EmailOutboxItem{}
				created.Metadata.ID = "outbox-1"
				m.outbox.On("CreateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
					// Payload offloaded because the recipient arrays push the combined row size over the limit.
					return len(o.Payload) == 0 && o.PayloadObjectStoreKey != nil
				})).Return(created, nil)

				m.transactions.On("CommitTx", mock.Anything).Return(nil)
			},
		},
		{
			name: "retain keeps the outbox (non-ephemeral) and send-to-all/send-at propagate",
			input: &EnqueueEmailInput{
				Builder:        &builder.AnnouncementEmail{},
				Subject:        "retained",
				SendToAllUsers: true,
				Retain:         true,
				SendAt:         &fixedSendAt,
			},
			setupMocks: func(m *enqueuerMocks) {
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)

				created := &models.EmailOutboxItem{}
				created.Metadata.ID = "outbox-1"
				m.outbox.On("CreateOutboxItem", mock.Anything, mock.MatchedBy(func(o *models.EmailOutboxItem) bool {
					return !o.Ephemeral &&
						o.SendToAllUsers &&
						o.SendAt != nil && o.SendAt.Equal(fixedSendAt)
				})).Return(created, nil)

				m.transactions.On("CommitTx", mock.Anything).Return(nil)
			},
		},
		{
			name:  "begin transaction error is returned",
			input: &EnqueueEmailInput{Builder: &builder.AnnouncementEmail{}, Subject: "hi"},
			setupMocks: func(m *enqueuerMocks) {
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), errors.New("no tx"))
			},
			wantErr: true,
		},
		{
			name:  "create outbox error rolls back and is returned",
			input: &EnqueueEmailInput{Builder: &builder.AnnouncementEmail{}, Subject: "hi"},
			setupMocks: func(m *enqueuerMocks) {
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				m.outbox.On("CreateOutboxItem", mock.Anything, mock.Anything).Return(nil, errors.New("insert failed"))
			},
			wantErr: true,
		},
		{
			name:  "retain link error rolls back and is returned",
			input: &EnqueueEmailInput{Builder: largeBuilder(), Subject: "hi"},
			setupMocks: func(m *enqueuerMocks) {
				failingRetain := db.RetainObjectRefFunc(func(_ context.Context, _ string) error {
					return errors.New("link failed")
				})
				m.store.On("UploadPayload", mock.Anything, mock.Anything).Return(failingRetain, "emails/x/payload.json", nil)
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
				created := &models.EmailOutboxItem{}
				created.Metadata.ID = "outbox-1"
				m.outbox.On("CreateOutboxItem", mock.Anything, mock.Anything).Return(created, nil)
			},
			wantErr: true,
		},
		{
			name:  "invalid outbox item (empty subject) rolls back and is returned",
			input: &EnqueueEmailInput{Builder: &builder.AnnouncementEmail{}},
			setupMocks: func(m *enqueuerMocks) {
				m.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
				m.transactions.On("RollbackTx", mock.Anything).Return(nil)
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocks := newEnqueuerMocks(t)
			if test.setupMocks != nil {
				test.setupMocks(mocks)
			}

			e := mocks.newEnqueuer(t)

			err := e.EnqueueEmail(t.Context(), test.input)

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
