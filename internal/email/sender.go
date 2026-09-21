package email

import (
	"context"
	"errors"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	emailplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	senderAttempts    = metric.NewCounter("email_sender_attempts", "Amount of email sender attempts.")
	recipientsClaimed = metric.NewCounter("email_recipients_claimed_total", "Number of email recipients claimed by the sender.")
	emailsSent        = metric.NewCounter("email_sent_total", "Number of emails accepted by the provider.")
)

const (
	// Fallback poll interval; events wake the worker promptly, so this can stay long to spare an idle system.
	minSendInterval = 10 * time.Minute
	maxSendInterval = 15 * time.Minute
	// maxPassDuration bounds one pass so a large backlog drains over several passes instead of one instance holding the queue.
	maxSenderPassDuration = 2 * time.Minute
	// recipientClaimBatchSize is how many recipients one claim takes.
	recipientClaimBatchSize = 100
	// maxSendAttempts bounds how many send attempts a single recipient gets before it is failed, so a
	// permanently failing address (or a soft bounce that never clears) can't be retried forever.
	maxSendAttempts = 5
)

// renderedEmail is a rendered outbox payload cached for the duration of a send pass.
type renderedEmail struct {
	subject string
	body    string
}

// personalize substitutes this body's placeholders with values specific to one recipient.
func (r *renderedEmail) personalize(recipient *models.EmailRecipient) string {
	// Always embed the recipient token so links correlate back to the recipient, even without a
	// feedback provider (useful for open/click metrics on retained emails).
	return strings.NewReplacer(builder.RecipientTokenPlaceholder, recipient.GetGlobalID()).Replace(r.body)
}

// Sender is the background worker that sends queued email recipients.
type Sender struct {
	dbClient                 *db.Client
	logger                   logger.Logger
	store                    Store
	provider                 emailplugin.Provider
	recorder                 *deliveryRecorder
	templateCtx              *builder.TemplateContext
	supportsDeliveryFeedback bool
}

// NewSender creates the email sender worker.
func NewSender(
	dbClient *db.Client,
	logger logger.Logger,
	store Store,
	provider emailplugin.Provider,
	frontendURL string,
	emailFooter string,
) *Sender {
	_, supportsDeliveryFeedback := provider.(emailplugin.FeedbackProvider)

	return &Sender{
		dbClient:                 dbClient,
		logger:                   logger,
		store:                    store,
		provider:                 provider,
		recorder:                 newDeliveryRecorder(dbClient, logger),
		supportsDeliveryFeedback: supportsDeliveryFeedback,
		templateCtx:              builder.NewTemplateContext(frontendURL, emailFooter),
	}
}

// send claims and sends batches of pending recipients until the queue drains, the pass times out, or the provider becomes unavailable; it reports whether work remains so the worker re-runs immediately.
func (s *Sender) send(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, maxSenderPassDuration)
	defer cancel()

	// renderedCache holds rendered outbox payloads for the duration of the pass; see renderOutboxItem.
	renderedCache := map[string]*renderedEmail{}

	retryableStatuses := models.EmailDeliveryStatusesMatching(models.EmailDeliveryStatus.IsRetryableStatus)

	for {
		recipients, err := s.dbClient.EmailRecipients.ClaimRecipients(ctx, &db.ClaimRecipientsInput{
			Statuses: retryableStatuses,
			Limit:    recipientClaimBatchSize,
		})
		if err != nil {
			return false, te.Wrap(err, "failed to claim email recipients")
		}

		recipientsClaimed.Add(float64(len(recipients)))

		if len(recipients) > 0 {
			s.logger.WithContextFields(ctx).Debugw("email sender claimed recipients", "count", len(recipients))
		}

		for _, recipient := range recipients {
			if ctx.Err() != nil {
				// The pass hit its time cap mid-batch; more recipients remain, so ask to re-run.
				return true, ctx.Err()
			}

			switch err := s.sendMailToRecipient(ctx, recipient, renderedCache); {
			case errors.Is(err, emailplugin.ErrProviderRateLimited), errors.Is(err, emailplugin.ErrProviderUnavailable):
				// Provider-wide transient: halt the pass; claimed unsent recipients lapse back to pending.
				return false, nil
			case te.FilterContextError(err) != nil:
				// Leave it pending for another attempt (transient render errors, one-off provider errors).
				s.logger.WithContextFields(ctx).Errorw("email sender failed to send a recipient",
					"recipientID", recipient.Metadata.ID,
					"outboxItemID", recipient.EmailOutboxItemID,
					"address", recipient.Address,
					"error", err,
				)
			}
		}

		// A short batch means the queue is drained; a full batch means more may remain, so ask to re-run.
		if len(recipients) < recipientClaimBatchSize {
			return false, nil
		}
	}
}

// sendMailToRecipient renders and sends mail to one recipient, then records a successful outcome.
func (s *Sender) sendMailToRecipient(ctx context.Context, recipient *models.EmailRecipient, renderedCache map[string]*renderedEmail) error {
	// An accepted recipient is only claimable once its feedback window elapsed, so no delivery feedback
	// ever arrived; abandon it (a terminal state) rather than re-sending, so its item can complete.
	if recipient.DeliveryStatus == models.EmailDeliveryAccepted {
		return s.recordOutcome(ctx, recipient, models.EmailDeliveryAbandoned, "no delivery feedback within the feedback window")
	}

	// Once it exceeds the budget, fail the recipient instead of rendering and sending again.
	if recipient.AttemptCount > maxSendAttempts {
		return s.recordOutcome(ctx, recipient, models.EmailDeliveryFailed, "exceeded maximum send attempts")
	}

	rendered, err := s.renderOutboxItem(ctx, recipient.EmailOutboxItemID, renderedCache)
	if err != nil {
		return te.Wrap(err, "failed to render outbox")
	}

	body := rendered.personalize(recipient)

	if err := s.provider.SendMail(ctx, recipient.Address, rendered.subject, body, recipient.Metadata.ID); err != nil {
		// A permanent provider rejection (e.g. a malformed address) will never succeed on retry, so fail
		// the recipient on this attempt rather than consuming the whole budget.
		if errors.Is(err, emailplugin.ErrProviderRejected) {
			return s.recordOutcome(ctx, recipient, models.EmailDeliveryFailed, err.Error())
		}

		return err
	}

	emailsSent.Inc()

	s.logger.WithContextFields(ctx).Debugw("sent email",
		"recipientID", recipient.Metadata.ID,
		"outboxItemID", recipient.EmailOutboxItemID,
		"attemptCount", recipient.AttemptCount,
	)

	// A provider with feedback resolves the send later; one without has no better signal than acceptance.
	status := models.EmailDeliveryCompleted
	if s.supportsDeliveryFeedback {
		status = models.EmailDeliveryAccepted
	}

	return s.recordOutcome(ctx, recipient, status, "")
}

// recordOutcome applies a delivery status (with an optional failure reason) to the recipient and persists
// it; a no-op change (e.g. the recipient is already final) records nothing.
func (s *Sender) recordOutcome(ctx context.Context, recipient *models.EmailRecipient, status models.EmailDeliveryStatus, reason string) error {
	var failureReason *string
	if reason != "" {
		failureReason = &reason
	}

	if !recipient.RecordDelivery(status, failureReason) {
		return nil
	}

	return s.recorder.persistOutcome(ctx, recipient)
}

// renderOutboxItem returns the cached rendered body for an outbox, rendering once per pass since it's shared across recipients.
func (s *Sender) renderOutboxItem(ctx context.Context, outboxID string, renderedCache map[string]*renderedEmail) (*renderedEmail, error) {
	if cached, ok := renderedCache[outboxID]; ok {
		return cached, nil
	}

	outbox, err := s.dbClient.EmailOutboxItems.GetOutboxItemByID(ctx, outboxID)
	if err != nil {
		return nil, te.Wrap(err, "failed to get outbox")
	}

	if outbox == nil {
		return nil, te.New("email outbox %s not found", outboxID, te.WithErrorCode(te.ENotFound))
	}

	// A small payload is stored inline on the row; otherwise it lives in object storage.
	payload := []byte(outbox.Payload)
	if len(payload) == 0 {
		if outbox.PayloadObjectStoreKey == nil {
			return nil, te.New("email outbox %s has neither an inline payload nor an object store key", outboxID)
		}

		payload, err = s.store.GetPayload(ctx, *outbox.PayloadObjectStoreKey)
		if err != nil {
			return nil, te.Wrap(err, "failed to load email payload")
		}
	}

	emailBuilder, err := builder.EmailType(outbox.EmailType).NewBuilder()
	if err != nil {
		return nil, te.Wrap(err, "failed to construct email builder")
	}

	if err = emailBuilder.InitFromMsgpack(payload); err != nil {
		return nil, te.Wrap(err, "failed to initialize email builder from payload")
	}

	body, err := emailBuilder.Build(s.templateCtx)
	if err != nil {
		return nil, te.Wrap(err, "failed to build email body")
	}

	rendered := &renderedEmail{subject: outbox.Subject, body: body}
	renderedCache[outboxID] = rendered

	return rendered, nil
}
