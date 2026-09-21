package email

import (
	"context"
	"errors"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	emailplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// feedbackEvents tracks applied delivery-feedback events by type (delivered, opened, complaint, bounces, etc.),
// so open and complaint rates are visible without querying the DB directly.
var feedbackEvents = metric.NewCounterVec("email_feedback_events_total", "Number of applied email delivery-feedback events, by type.", []string{"type"})

// feedbackRecipientNotFound counts feedback events whose recipient was not found (aged out, or a foreign event on a shared queue), so a spike is visible rather than silent.
var feedbackRecipientNotFound = metric.NewCounter("email_feedback_recipient_not_found_total", "Number of feedback events whose recipient could not be found.")

const (
	// feedbackRestartDelay is how long Start waits before restarting a consumer that exited unexpectedly.
	feedbackRestartDelay = 5 * time.Second
	// feedbackMaintenancePollInterval is how long Start waits before re-checking maintenance mode while paused or consuming.
	feedbackMaintenancePollInterval = 30 * time.Second
)

// errPausedForMaintenance signals that consume stopped the provider because maintenance mode turned on.
var errPausedForMaintenance = errors.New("email feedback consumer paused for maintenance")

// FeedbackManager consumes asynchronous delivery-feedback events and applies them to recipient state and the suppression list.
type FeedbackManager struct {
	dbClient           *db.Client
	logger             logger.Logger
	maintenanceMonitor maintenance.Monitor
	provider           emailplugin.FeedbackProvider
	recorder           *deliveryRecorder
}

// NewFeedbackManager creates a FeedbackManager.
func NewFeedbackManager(
	dbClient *db.Client,
	logger logger.Logger,
	maintenanceMonitor maintenance.Monitor,
	provider emailplugin.FeedbackProvider,
) *FeedbackManager {
	return &FeedbackManager{
		dbClient:           dbClient,
		logger:             logger,
		maintenanceMonitor: maintenanceMonitor,
		provider:           provider,
		recorder:           newDeliveryRecorder(dbClient, logger),
	}
}

// Start launches a background consumer, pausing while in maintenance mode and restarting after feedbackRestartDelay on an unexpected exit, until ctx is canceled.
func (m *FeedbackManager) Start(ctx context.Context) {
	go func() {
		m.logger.Info("email feedback manager started")
		defer m.logger.Info("email feedback manager stopped")

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if m.inMaintenance(ctx) {
				// Paused: wait a poll interval and re-check rather than consuming.
				m.wait(ctx, feedbackMaintenancePollInterval)
				continue
			}

			err := m.consume(ctx)

			switch {
			case ctx.Err() != nil:
				return
			case errors.Is(err, errPausedForMaintenance):
				// Stopped for maintenance; loop back to the pause check.
				continue
			}

			// The consumer isn't expected to return while ctx is live (nil or error), so restart it rather than silently stopping.
			m.logger.Errorf("email feedback consumer exited, restarting in %s: %v", feedbackRestartDelay, err)
			m.wait(ctx, feedbackRestartDelay)
		}
	}()
}

// Handle applies one feedback event to its recipient (by CorrelationID); an error signals non-durable processing so the provider redelivers.
func (m *FeedbackManager) Handle(ctx context.Context, event emailplugin.FeedbackEvent) error {
	recipient, err := m.dbClient.EmailRecipients.GetRecipientByID(ctx, event.CorrelationID)
	if err != nil {
		return te.Wrap(err, "failed to get recipient for feedback event")
	}

	if recipient == nil {
		// The recipient no longer exists (e.g. aged out, or an event from another environment
		// sharing the feedback topic), so there is nothing to apply.
		feedbackRecipientNotFound.Inc()
		return nil
	}

	m.logger.WithContextFields(ctx).Debugw("applying email feedback event",
		"recipientID", event.CorrelationID,
		"eventType", event.Type,
		"failureReason", event.FailureReason,
	)

	switch event.Type {
	case emailplugin.EventTypeDelivered:
		if !recipient.RecordDelivery(models.EmailDeliveryCompleted, nil) {
			return nil
		}
	case emailplugin.EventTypeSoftBounced:
		if !recipient.RecordDelivery(models.EmailDeliverySoftBounced, event.FailureReason) {
			return nil
		}
	case emailplugin.EventTypeHardBounced:
		if !recipient.RecordDelivery(models.EmailDeliveryHardBounced, event.FailureReason) {
			return nil
		}
	case emailplugin.EventTypeComplaint:
		if !recipient.MarkComplained() {
			return nil
		}
	case emailplugin.EventTypeOpened:
		if !recipient.MarkOpened() {
			return nil
		}
	case emailplugin.EventTypeRejected:
		if !recipient.RecordDelivery(models.EmailDeliveryFailed, event.FailureReason) {
			return nil
		}
	case emailplugin.EventTypeDelayed:
		if !recipient.RecordDelivery(models.EmailDeliveryDelayed, event.FailureReason) {
			return nil
		}
	default:
		m.logger.WithContextFields(ctx).Errorf("unknown email feedback event type %q for recipient %s", event.Type, event.CorrelationID)
		return nil
	}

	// RecordDelivery/Mark return false for a redelivered already-applied event, so the no-op arms return above and a duplicate isn't counted here.
	feedbackEvents.WithLabelValues(string(event.Type)).Inc()

	if err := m.recorder.persistOutcome(ctx, recipient); err != nil {
		return err
	}

	m.logger.WithContextFields(ctx).Debugw("email feedback event applied",
		"recipientID", recipient.Metadata.ID,
		"deliveryStatus", recipient.DeliveryStatus,
	)

	return nil
}

// consume runs the provider's consumer until ctx is canceled, it exits on its own, or maintenance mode turns on (returning errPausedForMaintenance).
func (m *FeedbackManager) consume(ctx context.Context) error {
	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Polls maintenance mode and cancels consumeCtx once it turns on, or exits once the consumer returns on its own.
	go func() {
		ticker := time.NewTicker(feedbackMaintenancePollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-consumeCtx.Done():
				return
			case <-ticker.C:
				if m.inMaintenance(consumeCtx) {
					cancel()
					return
				}
			}
		}
	}()

	err := m.provider.ConsumeFeedback(consumeCtx, m.Handle)
	if consumeCtx.Err() != nil && ctx.Err() == nil {
		return errPausedForMaintenance
	}

	return err
}

// inMaintenance reports whether the system is in maintenance mode, treating a check failure as not-in-maintenance (after logging) so it doesn't wedge the consumer.
func (m *FeedbackManager) inMaintenance(ctx context.Context) bool {
	inMaintenance, err := m.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		m.logger.Errorf("email feedback manager failed to check maintenance mode: %v", err)
		return false
	}

	return inMaintenance
}

// wait blocks for d or until ctx is canceled.
func (m *FeedbackManager) wait(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}
