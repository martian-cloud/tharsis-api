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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	emailplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestFeedbackManagerHandle(t *testing.T) {
	const (
		recipientID = "018f0000-0000-7000-8000-000000000000"
		address     = "user@example.com"
	)

	fixedTime := time.Now()

	tests := []struct {
		name               string
		event              emailplugin.FeedbackEvent
		recipient          *models.EmailRecipient
		wantUpdate         bool
		wantDeliveryStatus models.EmailDeliveryStatus
		wantOpened         bool
		wantComplained     bool
		wantSuppress       bool
		wantSuppressCause  models.EmailSuppressionCause
	}{
		{
			name:               "delivered marks completed",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeDelivered},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryAccepted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliveryCompleted,
		},
		{
			name:               "hard bounce marks bounced and suppresses",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeHardBounced, FailureReason: new("550 user unknown")},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryAccepted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliveryHardBounced,
			wantSuppress:       true,
			wantSuppressCause:  models.EmailSuppressionCauseHardBounce,
		},
		{
			name:               "soft bounce marks soft bounced without suppressing",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeSoftBounced, FailureReason: new("450 mailbox busy")},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryAccepted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliverySoftBounced,
		},
		{
			name:               "complaint suppresses while delivery stays completed",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeComplaint, FailureReason: new("abuse")},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryCompleted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliveryCompleted,
			wantComplained:     true,
			wantSuppress:       true,
			wantSuppressCause:  models.EmailSuppressionCauseComplaint,
		},
		{
			name:               "delivered after bounced is rejected",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeDelivered},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryHardBounced},
			wantDeliveryStatus: models.EmailDeliveryHardBounced,
		},
		{
			name:               "opened after complained does not re-record",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeOpened},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryCompleted, OpenedAt: &fixedTime},
			wantDeliveryStatus: models.EmailDeliveryCompleted,
			wantOpened:         true,
		},
		{
			name:      "unknown recipient is ignored",
			event:     emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeDelivered},
			recipient: nil,
		},
		{
			name:               "unknown event type is logged and does not update",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.FeedbackEventType("bogus")},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryAccepted},
			wantDeliveryStatus: models.EmailDeliveryAccepted,
		},
		{
			name:               "rejected marks failed",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeRejected, FailureReason: new("Bad content")},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryAccepted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliveryFailed,
		},
		{
			name:               "delayed marks delayed without suppressing",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeDelayed, FailureReason: new("smtp; 450 mailbox busy")},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryAccepted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliveryDelayed,
		},
		{
			name:               "open records engagement",
			event:              emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeOpened},
			recipient:          &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryCompleted},
			wantUpdate:         true,
			wantDeliveryStatus: models.EmailDeliveryCompleted,
			wantOpened:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRecipients := db.NewMockEmailRecipients(t)
			mockSuppressions := db.NewMockEmailSuppressions(t)
			mockOutbox := db.NewMockEmailOutboxItems(t)
			mockTransactions := db.NewMockTransactions(t)

			mockRecipients.On("GetRecipientByID", mock.Anything, recipientID).Return(test.recipient, nil)

			if test.wantUpdate {
				// persistOutcome runs in a transaction: claim the recipient, run handlers, commit.
				mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockRecipients.On("UpdateRecipient", mock.Anything, mock.MatchedBy(func(r *models.EmailRecipient) bool {
					return r.DeliveryStatus == test.wantDeliveryStatus
				})).Return(test.recipient, nil)

				// The complete-outbox-item handler only runs for a final status; it counts non-final
				// recipients and, finding none, marks the item completed.
				if test.wantDeliveryStatus.IsFinal() {
					mockRecipients.On("AllRecipientsFinal", mock.Anything, mock.Anything).Return(true, nil)
					mockOutbox.On("GetOutboxItemByID", mock.Anything, mock.Anything).
						Return(&models.EmailOutboxItem{Status: models.EmailOutboxItemStatusReady}, nil)
					mockOutbox.On("UpdateOutboxItem", mock.Anything, mock.Anything).Return(&models.EmailOutboxItem{}, nil)
				}

				if test.wantSuppress {
					mockSuppressions.On("CreateSuppression", mock.Anything, mock.MatchedBy(func(s *models.EmailSuppression) bool {
						return s.Address == address && s.Cause == test.wantSuppressCause
					})).Return(&models.EmailSuppression{Cause: test.wantSuppressCause}, nil)
				}
			}

			testLogger, _ := logger.NewForTest()
			manager := NewFeedbackManager(&db.Client{
				EmailRecipients:   mockRecipients,
				EmailSuppressions: mockSuppressions,
				EmailOutboxItems:  mockOutbox,
				Transactions:      mockTransactions,
			}, testLogger, maintenance.NewMockMonitor(t), nil)

			err := manager.Handle(context.Background(), test.event)
			require.NoError(t, err)

			if test.recipient != nil {
				require.Equal(t, test.wantDeliveryStatus, test.recipient.DeliveryStatus)
				require.Equal(t, test.wantOpened, test.recipient.OpenedAt != nil)
				require.Equal(t, test.wantComplained, test.recipient.ComplainedAt != nil)
			}
		})
	}
}

func TestFeedbackManagerHandleEdgeCases(t *testing.T) {
	const recipientID = "018f0000-0000-7000-8000-000000000000"

	newManager := func(recipients *db.MockEmailRecipients, suppressions *db.MockEmailSuppressions) *FeedbackManager {
		testLogger, _ := logger.NewForTest()
		return NewFeedbackManager(&db.Client{
			EmailRecipients:   recipients,
			EmailSuppressions: suppressions,
		}, testLogger, maintenance.NewMockMonitor(t), nil)
	}

	t.Run("get recipient error is returned so the provider redelivers", func(t *testing.T) {
		mockRecipients := db.NewMockEmailRecipients(t)
		mockRecipients.On("GetRecipientByID", mock.Anything, recipientID).Return(nil, errors.New("db down"))

		manager := newManager(mockRecipients, db.NewMockEmailSuppressions(t))
		err := manager.Handle(context.Background(), emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeDelivered})
		require.Error(t, err)
	})

	t.Run("unknown event type is logged and ignored without persisting", func(t *testing.T) {
		mockRecipients := db.NewMockEmailRecipients(t)
		// No UpdateRecipient expectation: the default branch returns before persisting.
		mockRecipients.On("GetRecipientByID", mock.Anything, recipientID).
			Return(&models.EmailRecipient{Address: "user@example.com", DeliveryStatus: models.EmailDeliveryAccepted}, nil)

		manager := newManager(mockRecipients, db.NewMockEmailSuppressions(t))
		err := manager.Handle(context.Background(), emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.FeedbackEventType("nonsense")})
		require.NoError(t, err)
	})

	t.Run("persist error propagates to the caller", func(t *testing.T) {
		recipient := &models.EmailRecipient{Address: "user@example.com", DeliveryStatus: models.EmailDeliveryAccepted}

		mockRecipients := db.NewMockEmailRecipients(t)
		mockRecipients.On("GetRecipientByID", mock.Anything, recipientID).Return(recipient, nil)
		// A non-lock update error surfaces from persistOutcome.
		mockRecipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(nil, errors.New("update failed"))

		mockTransactions := db.NewMockTransactions(t)
		mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
		mockTransactions.On("RollbackTx", mock.Anything).Return(nil)

		testLogger, _ := logger.NewForTest()
		manager := NewFeedbackManager(&db.Client{
			EmailRecipients:   mockRecipients,
			EmailSuppressions: db.NewMockEmailSuppressions(t),
			Transactions:      mockTransactions,
		}, testLogger, maintenance.NewMockMonitor(t), nil)

		err := manager.Handle(context.Background(), emailplugin.FeedbackEvent{CorrelationID: recipientID, Type: emailplugin.EventTypeDelivered})
		require.Error(t, err)
	})
}

func TestNewFeedbackManager(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	m := NewFeedbackManager(&db.Client{}, testLogger, maintenance.NewMockMonitor(t), emailplugin.NewMockFeedbackProvider(t))
	assert.NotNil(t, m)
}

func TestFeedbackManagerInMaintenance(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	tests := []struct {
		name       string
		mmReturn   bool
		mmErr      error
		wantResult bool
	}{
		{
			name:       "in maintenance",
			mmReturn:   true,
			wantResult: true,
		},
		{
			name:       "not in maintenance",
			mmReturn:   false,
			wantResult: false,
		},
		{
			name:       "check error is treated as not-in-maintenance",
			mmErr:      errors.New("monitor down"),
			wantResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mm := maintenance.NewMockMonitor(t)
			mm.On("InMaintenanceMode", mock.Anything).Return(test.mmReturn, test.mmErr)

			m := NewFeedbackManager(&db.Client{}, testLogger, mm, emailplugin.NewMockFeedbackProvider(t))
			assert.Equal(t, test.wantResult, m.inMaintenance(context.Background()))
		})
	}
}

func TestFeedbackManagerWait(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	m := NewFeedbackManager(&db.Client{}, testLogger, maintenance.NewMockMonitor(t), emailplugin.NewMockFeedbackProvider(t))

	// A cancelled context returns immediately via the ctx.Done() arm rather than waiting the full delay.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		m.wait(ctx, time.Hour)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not return promptly on context cancellation")
	}
}

func TestFeedbackManagerConsume(t *testing.T) {
	testLogger, _ := logger.NewForTest()

	t.Run("returns nil when the provider consumer exits cleanly", func(t *testing.T) {
		provider := emailplugin.NewMockFeedbackProvider(t)
		provider.On("ConsumeFeedback", mock.Anything, mock.Anything).Return(nil)

		m := NewFeedbackManager(&db.Client{}, testLogger, maintenance.NewMockMonitor(t), provider)
		require.NoError(t, m.consume(context.Background()))
	})

	t.Run("surfaces a provider error", func(t *testing.T) {
		provider := emailplugin.NewMockFeedbackProvider(t)
		provider.On("ConsumeFeedback", mock.Anything, mock.Anything).Return(errors.New("stream broke"))

		m := NewFeedbackManager(&db.Client{}, testLogger, maintenance.NewMockMonitor(t), provider)
		require.Error(t, m.consume(context.Background()))
	})

	t.Run("parent cancellation returns the provider error, not a maintenance pause", func(t *testing.T) {
		provider := emailplugin.NewMockFeedbackProvider(t)
		// The provider observes the cancelled context and returns its context error; since the parent
		// ctx is what was cancelled (not the maintenance poller), consume returns that error unchanged.
		provider.On("ConsumeFeedback", mock.Anything, mock.Anything).Return(context.Canceled)

		m := NewFeedbackManager(&db.Client{}, testLogger, maintenance.NewMockMonitor(t), provider)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := m.consume(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	})
}
