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
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestPersistDeliveryOutcome(t *testing.T) {
	const address = "user@example.com"

	now := time.Now()

	tests := []struct {
		name       string
		recipient  *models.EmailRecipient
		setupMocks func(*db.MockEmailRecipients, *db.MockEmailSuppressions)
		wantErr    bool
	}{
		{
			name:      "completed does not suppress",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryCompleted},
			setupMocks: func(recipients *db.MockEmailRecipients, _ *db.MockEmailSuppressions) {
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(&models.EmailRecipient{}, nil)
			},
		},
		{
			name:      "hard bounce suppresses as bounce",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryHardBounced},
			setupMocks: func(recipients *db.MockEmailRecipients, suppressions *db.MockEmailSuppressions) {
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(&models.EmailRecipient{}, nil)
				suppressions.On("CreateSuppression", mock.Anything, mock.MatchedBy(func(s *models.EmailSuppression) bool {
					return s.Address == address && s.Cause == models.EmailSuppressionCauseHardBounce
				})).Return(&models.EmailSuppression{Cause: models.EmailSuppressionCauseHardBounce}, nil)
			},
		},
		{
			name:      "complaint suppresses as complaint",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryCompleted, ComplainedAt: &now},
			setupMocks: func(recipients *db.MockEmailRecipients, suppressions *db.MockEmailSuppressions) {
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(&models.EmailRecipient{}, nil)
				suppressions.On("CreateSuppression", mock.Anything, mock.MatchedBy(func(s *models.EmailSuppression) bool {
					return s.Cause == models.EmailSuppressionCauseComplaint
				})).Return(&models.EmailSuppression{Cause: models.EmailSuppressionCauseComplaint}, nil)
			},
		},
		{
			name:      "optimistic lock on update returns nil without suppressing",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryHardBounced},
			setupMocks: func(recipients *db.MockEmailRecipients, _ *db.MockEmailSuppressions) {
				// A concurrent claimer won the update and owns the full outcome; the loser must not
				// suppress on its stale in-memory state, so CreateSuppression is never called.
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(nil, db.ErrOptimisticLockError)
			},
		},
		{
			name:      "non-lock update error is returned",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryCompleted},
			setupMocks: func(recipients *db.MockEmailRecipients, _ *db.MockEmailSuppressions) {
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name:      "conflict on suppression is swallowed (already suppressed)",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryHardBounced},
			setupMocks: func(recipients *db.MockEmailRecipients, suppressions *db.MockEmailSuppressions) {
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(&models.EmailRecipient{}, nil)
				suppressions.On("CreateSuppression", mock.Anything, mock.Anything).
					Return(nil, te.New("already suppressed", te.WithErrorCode(te.EConflict)))
			},
		},
		{
			name:      "non-conflict suppression error is returned",
			recipient: &models.EmailRecipient{Address: address, DeliveryStatus: models.EmailDeliveryHardBounced},
			setupMocks: func(recipients *db.MockEmailRecipients, suppressions *db.MockEmailSuppressions) {
				recipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(&models.EmailRecipient{}, nil)
				suppressions.On("CreateSuppression", mock.Anything, mock.Anything).Return(nil, errors.New("suppress failed"))
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRecipients := db.NewMockEmailRecipients(t)
			mockSuppressions := db.NewMockEmailSuppressions(t)
			mockTransactions := db.NewMockTransactions(t)

			mockTransactions.On("BeginTx", mock.Anything).Return(func(ctx context.Context) context.Context { return ctx }, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			// A final-status recipient runs the complete-item effect; report a non-final recipient still
			// present so the item is not completed here, keeping these cases focused on update/suppress.
			mockRecipients.On("AllRecipientsFinal", mock.Anything, mock.Anything).Return(false, nil).Maybe()

			test.setupMocks(mockRecipients, mockSuppressions)

			dbClient := &db.Client{
				EmailRecipients:   mockRecipients,
				EmailSuppressions: mockSuppressions,
				Transactions:      mockTransactions,
			}

			testLogger, _ := logger.NewForTest()

			err := newDeliveryRecorder(dbClient, testLogger).persistOutcome(t.Context(), test.recipient)

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
