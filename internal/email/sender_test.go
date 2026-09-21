package email

import (
	"context"
	"errors"
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	emailplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type senderMocks struct {
	recipients   *db.MockEmailRecipients
	suppressions *db.MockEmailSuppressions
	outbox       *db.MockEmailOutboxItems
	store        *MockStore
	provider     *emailplugin.MockProvider
	transactions *db.MockTransactions
}

func newSenderMocks(t *testing.T) *senderMocks {
	return &senderMocks{
		recipients:   db.NewMockEmailRecipients(t),
		suppressions: db.NewMockEmailSuppressions(t),
		outbox:       db.NewMockEmailOutboxItems(t),
		store:        NewMockStore(t),
		provider:     emailplugin.NewMockProvider(t),
		transactions: db.NewMockTransactions(t),
	}
}

func (m *senderMocks) newSender(_ *testing.T, supportsFeedback bool) *Sender {
	testLogger, _ := logger.NewForTest()
	dbClient := &db.Client{
		EmailRecipients:   m.recipients,
		EmailSuppressions: m.suppressions,
		EmailOutboxItems:  m.outbox,
		Transactions:      m.transactions,
	}
	return &Sender{
		dbClient:                 dbClient,
		store:                    m.store,
		provider:                 m.provider,
		logger:                   testLogger,
		recorder:                 newDeliveryRecorder(dbClient, testLogger),
		templateCtx:              builder.NewTemplateContext("https://tharsis.example.com", ""),
		supportsDeliveryFeedback: supportsFeedback,
	}
}

func announcementOutboxItem(id string) *models.EmailOutboxItem {
	o := &models.EmailOutboxItem{
		EmailType:             builder.AnnouncementEmailType,
		Subject:               "subject",
		PayloadObjectStoreKey: new("emails/" + id + "/payload.json"),
	}
	o.Metadata.ID = id
	return o
}

func announcementPayload() []byte {
	payload, _ := msgpack.Marshal(&builder.AnnouncementEmail{Message: "hello", Severity: "INFO"})
	return payload
}

func TestRenderedEmailPersonalize(t *testing.T) {
	body := "click here: " + builder.RecipientTokenPlaceholder
	recipient := &models.EmailRecipient{}
	recipient.Metadata.ID = "rec-1"

	t.Run("always substitutes the recipient token", func(t *testing.T) {
		r := &renderedEmail{body: body}
		got := r.personalize(recipient)
		assert.NotContains(t, got, builder.RecipientTokenPlaceholder)
		assert.Contains(t, got, recipient.GetGlobalID())
	})
}

func TestSenderRenderOutboxItem(t *testing.T) {
	const outboxID = "outbox-1"

	t.Run("returns the cached render without querying the outbox", func(t *testing.T) {
		mocks := newSenderMocks(t)
		s := mocks.newSender(t, true)

		cached := &renderedEmail{subject: "s", body: "b"}
		cache := map[string]*renderedEmail{outboxID: cached}

		got, err := s.renderOutboxItem(t.Context(), outboxID, cache)
		require.NoError(t, err)
		assert.Same(t, cached, got)
	})

	t.Run("renders, caches, and returns the outbox body on a miss", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, "emails/"+outboxID+"/payload.json").Return(announcementPayload(), nil)

		s := mocks.newSender(t, true)
		cache := map[string]*renderedEmail{}

		got, err := s.renderOutboxItem(t.Context(), outboxID, cache)
		require.NoError(t, err)
		assert.Equal(t, "subject", got.subject)
		assert.NotEmpty(t, got.body)
		// The render is cached under the outbox ID for reuse across recipients in the pass.
		assert.Same(t, got, cache[outboxID])
	})

	t.Run("outbox not found is a not-found error", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(nil, nil)

		s := mocks.newSender(t, true)
		_, err := s.renderOutboxItem(t.Context(), outboxID, map[string]*renderedEmail{})
		assert.Error(t, err)
	})

	t.Run("get outbox error is returned", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(nil, errors.New("db down"))

		s := mocks.newSender(t, true)
		_, err := s.renderOutboxItem(t.Context(), outboxID, map[string]*renderedEmail{})
		assert.Error(t, err)
	})

	t.Run("get payload error is returned", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(nil, errors.New("gone"))

		s := mocks.newSender(t, true)
		_, err := s.renderOutboxItem(t.Context(), outboxID, map[string]*renderedEmail{})
		assert.Error(t, err)
	})

	t.Run("unknown email type fails to construct a builder", func(t *testing.T) {
		mocks := newSenderMocks(t)
		unknown := &models.EmailOutboxItem{EmailType: builder.EmailType("does_not_exist"), PayloadObjectStoreKey: new("emails/x/payload.json")}
		unknown.Metadata.ID = outboxID
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(unknown, nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return([]byte(`{}`), nil)

		s := mocks.newSender(t, true)
		_, err := s.renderOutboxItem(t.Context(), outboxID, map[string]*renderedEmail{})
		assert.Error(t, err)
	})

	t.Run("invalid payload fails to initialize the builder", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return([]byte("not json"), nil)

		s := mocks.newSender(t, true)
		_, err := s.renderOutboxItem(t.Context(), outboxID, map[string]*renderedEmail{})
		assert.Error(t, err)
	})
}

func TestSenderSendMailToRecipient(t *testing.T) {
	const outboxID = "outbox-1"

	newRecipient := func() *models.EmailRecipient {
		r := &models.EmailRecipient{EmailOutboxItemID: outboxID, Address: "user@x.com", DeliveryStatus: models.EmailDeliveryPending}
		r.Metadata.ID = "rec-1"
		return r
	}

	t.Run("feedback provider marks accepted and persists", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, "user@x.com", "subject", mock.Anything, "rec-1").Return(nil)
		// accepted is non-final, so persistOutcome only claims the recipient (no complete-item handler).
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
			return r.DeliveryStatus == models.EmailDeliveryAccepted
		})).Return(newRecipient(), nil)

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), newRecipient(), map[string]*renderedEmail{})
		require.NoError(t, err)
	})

	t.Run("non-feedback provider marks completed and persists", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		// completed is final, so the complete-outbox-item handler also runs after the recipient claim.
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
			return r.DeliveryStatus == models.EmailDeliveryCompleted
		})).Return(newRecipient(), nil)
		mocks.recipients.On("AllRecipientsFinal", mock.Anything, outboxID).Return(true, nil)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).
			Return(&models.EmailOutboxItem{Status: models.EmailOutboxItemStatusReady}, nil).Maybe()
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.Anything).Return(&models.EmailOutboxItem{}, nil)

		s := mocks.newSender(t, false)
		err := s.sendMailToRecipient(t.Context(), newRecipient(), map[string]*renderedEmail{})
		require.NoError(t, err)
	})

	t.Run("provider send error is returned and nothing is persisted", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(emailplugin.ErrProviderRateLimited)

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), newRecipient(), map[string]*renderedEmail{})
		assert.ErrorIs(t, err, emailplugin.ErrProviderRateLimited)
	})

	t.Run("render failure propagates", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(nil, errors.New("db down"))

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), newRecipient(), map[string]*renderedEmail{})
		assert.Error(t, err)
	})

	t.Run("already-final recipient sends but records no outcome", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		// No UpdateRecipient expectation: RecordDelivery on a final status returns false, so the
		// strict mock fails if persistOutcome is ever reached.

		finalRecipient := &models.EmailRecipient{EmailOutboxItemID: outboxID, Address: "user@x.com", DeliveryStatus: models.EmailDeliveryCompleted}
		finalRecipient.Metadata.ID = "rec-1"

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), finalRecipient, map[string]*renderedEmail{})
		require.NoError(t, err)
	})

	t.Run("a recipient past the attempt budget is failed without sending", func(t *testing.T) {
		mocks := newSenderMocks(t)
		// No outbox/store/provider expectations: the cap check returns before rendering or sending.
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
			return r.DeliveryStatus == models.EmailDeliveryFailed
		})).Return(newRecipient(), nil)
		// failed is final, so the complete-outbox-item handler also runs.
		mocks.recipients.On("AllRecipientsFinal", mock.Anything, outboxID).Return(true, nil)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).
			Return(&models.EmailOutboxItem{Status: models.EmailOutboxItemStatusReady}, nil).Maybe()
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.Anything).Return(&models.EmailOutboxItem{}, nil)

		over := newRecipient()
		over.AttemptCount = maxSendAttempts + 1

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), over, map[string]*renderedEmail{})
		require.NoError(t, err)
	})

	t.Run("a provider rejection fails the recipient on the first attempt", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Errorf("bad: %w", emailplugin.ErrProviderRejected))
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
			return r.DeliveryStatus == models.EmailDeliveryFailed
		})).Return(newRecipient(), nil)
		mocks.recipients.On("AllRecipientsFinal", mock.Anything, outboxID).Return(true, nil)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).
			Return(&models.EmailOutboxItem{Status: models.EmailOutboxItemStatusReady}, nil).Maybe()
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.Anything).Return(&models.EmailOutboxItem{}, nil)

		first := newRecipient()
		first.AttemptCount = 1

		s := mocks.newSender(t, true)
		// The rejection is handled internally, so send returns nil (recipient failed, not retried).
		err := s.sendMailToRecipient(t.Context(), first, map[string]*renderedEmail{})
		require.NoError(t, err)
	})

	t.Run("an accepted recipient past its feedback window is abandoned without sending", func(t *testing.T) {
		mocks := newSenderMocks(t)
		// No render/provider expectations: an accepted recipient is abandoned before rendering.
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
			return r.DeliveryStatus == models.EmailDeliveryAbandoned
		})).Return(newRecipient(), nil)
		// abandoned is final, so the complete-outbox-item handler also runs.
		mocks.recipients.On("AllRecipientsFinal", mock.Anything, outboxID).Return(true, nil)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).
			Return(&models.EmailOutboxItem{Status: models.EmailOutboxItemStatusReady}, nil).Maybe()
		mocks.outbox.On("UpdateOutboxItem", mock.Anything, mock.Anything).Return(&models.EmailOutboxItem{}, nil)

		accepted := newRecipient()
		accepted.DeliveryStatus = models.EmailDeliveryAccepted

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), accepted, map[string]*renderedEmail{})
		require.NoError(t, err)
	})

	t.Run("a recipient at the last allowed attempt still sends", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
			return r.DeliveryStatus == models.EmailDeliveryAccepted
		})).Return(newRecipient(), nil)

		last := newRecipient()
		last.AttemptCount = maxSendAttempts

		s := mocks.newSender(t, true)
		err := s.sendMailToRecipient(t.Context(), last, map[string]*renderedEmail{})
		require.NoError(t, err)
	})
}

func TestSenderSend(t *testing.T) {
	const outboxID = "outbox-1"

	claimInput := &db.ClaimRecipientsInput{
		Statuses: []models.EmailDeliveryStatus{
			models.EmailDeliveryPending,
			models.EmailDeliveryAccepted,
			models.EmailDeliverySoftBounced,
			models.EmailDeliveryDelayed,
		},
		Limit: recipientClaimBatchSize,
	}

	recipient := func(id string) *models.EmailRecipient {
		r := &models.EmailRecipient{EmailOutboxItemID: outboxID, Address: "user@x.com", DeliveryStatus: models.EmailDeliveryPending}
		r.Metadata.ID = id
		return r
	}

	t.Run("empty claim drains the pass", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).Return(nil, nil)

		s := mocks.newSender(t, true)
		moreWork, sendErr := s.send(t.Context())
		require.NoError(t, sendErr)
		assert.False(t, moreWork)
	})

	t.Run("claim error is returned", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).Return(nil, errors.New("claim failed"))

		s := mocks.newSender(t, true)
		_, sendErr := s.send(t.Context())
		assert.Error(t, sendErr)
	})

	t.Run("a provider-wide rate limit halts the pass without erroring", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).Return([]*models.EmailRecipient{recipient("rec-1")}, nil).Once()
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil)
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil)
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(emailplugin.ErrProviderUnavailable)

		s := mocks.newSender(t, true)
		// No further ClaimRecipients call: the pass halts after the provider-wide failure.
		_, sendErr := s.send(t.Context())
		require.NoError(t, sendErr)
	})

	t.Run("recipients sharing an outbox reuse one cached render", func(t *testing.T) {
		mocks := newSenderMocks(t)
		// Two recipients share the outbox; it's rendered once (GetOutboxItemByID + GetPayload once).
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).
			Return([]*models.EmailRecipient{recipient("rec-1"), recipient("rec-2")}, nil).Once()
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil).Once()
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil).Once()
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		// Feedback provider marks each recipient accepted (non-final): persist claims the recipient only.
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(recipient("rec-1"), nil)

		s := mocks.newSender(t, true)
		_, sendErr := s.send(t.Context())
		require.NoError(t, sendErr)
	})

	t.Run("a transient per-recipient error is swallowed and the pass drains", func(t *testing.T) {
		mocks := newSenderMocks(t)
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).
			Return([]*models.EmailRecipient{recipient("rec-1")}, nil).Once()
		// Render fails for this recipient (generic error): it's logged and left pending, and since the
		// batch is short the pass drains without a provider send or a persist.
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(nil, errors.New("transient db error"))

		s := mocks.newSender(t, true)
		_, sendErr := s.send(t.Context())
		require.NoError(t, sendErr)
	})

	t.Run("a full batch triggers another claim until the queue drains", func(t *testing.T) {
		mocks := newSenderMocks(t)

		// First claim returns a full batch, all sharing one outbox (rendered once); a full batch
		// (== recipientClaimBatchSize) forces a second claim.
		fullBatch := make([]*models.EmailRecipient, recipientClaimBatchSize)
		for i := range fullBatch {
			fullBatch[i] = recipient(fmt.Sprintf("rec-%d", i))
		}
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).Return(fullBatch, nil).Once()
		mocks.outbox.On("GetOutboxItemByID", mock.Anything, outboxID).Return(announcementOutboxItem(outboxID), nil).Once()
		mocks.store.On("GetPayload", mock.Anything, mock.Anything).Return(announcementPayload(), nil).Once()
		mocks.provider.On("SendMail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mocks.transactions.On("BeginTx", mock.Anything).Return(t.Context(), nil)
		mocks.transactions.On("RollbackTx", mock.Anything).Return(nil)
		mocks.transactions.On("CommitTx", mock.Anything).Return(nil)
		mocks.recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(recipient("rec-0"), nil)
		// Second claim is empty, draining the pass.
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).Return(nil, nil).Once()

		s := mocks.newSender(t, true)
		_, sendErr := s.send(t.Context())
		require.NoError(t, sendErr)
	})

	t.Run("a cancelled context mid-batch returns the context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		mocks := newSenderMocks(t)
		// Cancel as the claim returns so the per-recipient ctx.Err() guard trips before any send.
		mocks.recipients.On("ClaimRecipients", mock.Anything, claimInput).
			Run(func(_ mock.Arguments) { cancel() }).
			Return([]*models.EmailRecipient{recipient("rec-1")}, nil).Once()

		s := mocks.newSender(t, true)
		_, err := s.send(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestNewSender(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	t.Run("without feedback provider sets supportsDeliveryFeedback false", func(t *testing.T) {
		s := NewSender(&db.Client{}, testLogger, NewMockStore(t), emailplugin.NewMockProvider(t), "https://example.com", "")
		assert.NotNil(t, s)
		assert.False(t, s.supportsDeliveryFeedback)
	})

	t.Run("with feedback provider sets supportsDeliveryFeedback true", func(t *testing.T) {
		s := NewSender(&db.Client{}, testLogger, NewMockStore(t), emailplugin.NewMockFeedbackProvider(t), "https://example.com", "custom footer")
		assert.NotNil(t, s)
		assert.True(t, s.supportsDeliveryFeedback)
	})
}

func TestNewStore(t *testing.T) {
	assert.NotNil(t, NewStore(nil, nil))
}
