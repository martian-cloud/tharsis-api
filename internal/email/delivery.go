package email

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// persistDeliveryTimeout bounds the post-send bookkeeping write, detached from the pass deadline.
const persistDeliveryTimeout = 30 * time.Second

var (
	// recipientOutcomes tracks delivery status transitions so bounce/failure/complaint rates are visible without querying the DB directly.
	recipientOutcomes = metric.NewCounterVec("email_recipient_outcome_total", "Number of email recipient delivery status updates, by resulting status.", []string{"status"})
	// suppressionsCreated tracks new suppression-list entries, by cause.
	suppressionsCreated = metric.NewCounterVec("email_suppressions_created_total", "Number of addresses added to the suppression list, by cause.", []string{"cause"})
)

// deliveryOutcome accumulates what a recorded outcome did, so post-commit reactions (metrics today,
// DB-backed metrics or other effects later) can act on the committed result.
type deliveryOutcome struct {
	suppressedCause models.EmailSuppressionCause // "" when nothing was newly suppressed
}

// deliveryOutcomeHandler is one transactional side effect of recording a delivery outcome, run in order inside the claiming transaction so the first error rolls back the whole outcome.
type deliveryOutcomeHandler interface {
	// handles reports whether this handler runs for the given recipient outcome.
	handles(r *models.EmailRecipient) bool
	// handle performs the side effect within the transaction, recording anything post-commit reactions need onto outcome.
	handle(ctx context.Context, r *models.EmailRecipient, outcome *deliveryOutcome) error
}

// deliveryRecorder records a recipient's delivery outcome and its side effects atomically.
type deliveryRecorder struct {
	dbClient *db.Client
	logger   logger.Logger
	handlers []deliveryOutcomeHandler
}

// newDeliveryRecorder builds the recorder with its transactional side-effect chain.
func newDeliveryRecorder(dbClient *db.Client, logger logger.Logger) *deliveryRecorder {
	return &deliveryRecorder{
		dbClient: dbClient,
		logger:   logger,
		handlers: []deliveryOutcomeHandler{
			completeOutboxItemHandler{dbClient: dbClient, logger: logger},
			suppressRecipientHandler{dbClient: dbClient, logger: logger},
		},
	}
}

// persistOutcome claims the recipient and runs its side effects in one transaction, then records
// post-commit metrics. The recipient update is the ownership-claiming step: an optimistic-lock conflict
// means another process owns this outcome, so it aborts without error and commits nothing.
func (d *deliveryRecorder) persistOutcome(ctx context.Context, r *models.EmailRecipient) error {
	// Use a detached ctx here so changes can be persisted even if parent ctx is canceled.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistDeliveryTimeout)
	defer cancel()

	txContext, err := d.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return te.Wrap(err, "failed to begin delivery outcome transaction")
	}

	defer func() {
		if txErr := d.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			d.logger.WithContextFields(ctx).Errorf("failed to roll back email delivery outcome transaction: %v", txErr)
		}
	}()

	if _, err := d.dbClient.EmailRecipients.UpdateRecipient(txContext, r); err != nil {
		if te.ErrorCode(err) == te.EOptimisticLock {
			return nil
		}

		return te.Wrap(err, "failed to update email recipient")
	}

	outcome := &deliveryOutcome{}
	for _, handler := range d.handlers {
		if !handler.handles(r) {
			continue
		}

		if err := handler.handle(txContext, r, outcome); err != nil {
			return err
		}
	}

	if err := d.dbClient.Transactions.CommitTx(txContext); err != nil {
		return te.Wrap(err, "failed to commit delivery outcome transaction")
	}

	// Record metrics only after the outcome is durable so a rolled-back write doesn't inflate counters.
	recipientOutcomes.WithLabelValues(string(r.DeliveryStatus)).Inc()

	if outcome.suppressedCause != "" {
		suppressionsCreated.WithLabelValues(string(outcome.suppressedCause)).Inc()
	}

	return nil
}

// completeOutboxItemHandler flips the recipient's outbox item to completed once every recipient is final.
type completeOutboxItemHandler struct {
	dbClient *db.Client
	logger   logger.Logger
}

// handles gates on a final status since only a recipient reaching a final status can be the last one to complete an item.
func (completeOutboxItemHandler) handles(r *models.EmailRecipient) bool {
	return r.DeliveryStatus.IsFinal()
}

func (h completeOutboxItemHandler) handle(ctx context.Context, r *models.EmailRecipient, _ *deliveryOutcome) error {
	allFinal, err := h.dbClient.EmailRecipients.AllRecipientsFinal(ctx, r.EmailOutboxItemID)
	if err != nil {
		return te.Wrap(err, "failed to check for non-final recipients")
	}

	if !allFinal {
		return nil
	}

	item, err := h.dbClient.EmailOutboxItems.GetOutboxItemByID(ctx, r.EmailOutboxItemID)
	if err != nil {
		return te.Wrap(err, "failed to get outbox item")
	}

	if item == nil || item.Status == models.EmailOutboxItemStatusCompleted {
		return nil
	}

	item.Status = models.EmailOutboxItemStatusCompleted

	// An OLE means another finalizing recipient already completed the item, which is the intended outcome.
	if _, err := h.dbClient.EmailOutboxItems.UpdateOutboxItem(ctx, item); err != nil && te.ErrorCode(err) != te.EOptimisticLock {
		return te.Wrap(err, "failed to mark outbox item completed")
	}

	return nil
}

// suppressRecipientHandler adds the recipient's address to the suppression list when its outcome warrants it.
type suppressRecipientHandler struct {
	dbClient *db.Client
	logger   logger.Logger
}

func (suppressRecipientHandler) handles(r *models.EmailRecipient) bool {
	_, ok := r.SuppressionCause()
	return ok
}

func (h suppressRecipientHandler) handle(ctx context.Context, r *models.EmailRecipient, outcome *deliveryOutcome) error {
	cause, _ := r.SuppressionCause()

	suppression, err := h.dbClient.EmailSuppressions.CreateSuppression(ctx, &models.EmailSuppression{
		Address: r.Address,
		Cause:   cause,
	})
	if err != nil && te.ErrorCode(err) != te.EConflict {
		return te.Wrap(err, "failed to suppress recipient")
	}

	// Only a fresh insert returns a non-nil suppression, so a swallowed conflict is not counted.
	if suppression != nil {
		outcome.suppressedCause = suppression.Cause
	}

	return nil
}
