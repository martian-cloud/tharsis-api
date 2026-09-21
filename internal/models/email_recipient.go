package models

import (
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sanitize"
)

var _ Model = (*EmailRecipient)(nil)

// maxFailureReasonLength caps a recipient's stored failure reason. Provider feedback (bounce/complaint)
// reasons can be arbitrarily large; this bounds what we persist to avoid excessive storage.
const maxFailureReasonLength = 250

// EmailRecipientRetryDelay is the shared retry-pacing window: both the claim lease and the backoff after a soft bounce/delay use it.
const EmailRecipientRetryDelay = 5 * time.Minute

// EmailFeedbackWindow is how long an accepted recipient waits for delivery feedback before the sender abandons it; it bounds table growth and is not a delivery SLA.
const EmailFeedbackWindow = 24 * time.Hour

// allEmailDeliveryStatuses is the source of truth for every status; callers filter it through the predicates below.
var allEmailDeliveryStatuses = []EmailDeliveryStatus{
	EmailDeliveryPending,
	EmailDeliveryAccepted,
	EmailDeliveryCompleted,
	EmailDeliverySoftBounced,
	EmailDeliveryDelayed,
	EmailDeliveryHardBounced,
	EmailDeliveryFailed,
	EmailDeliveryAbandoned,
}

// EmailDeliveryStatusesMatching returns every delivery status satisfying the predicate.
func EmailDeliveryStatusesMatching(predicate func(EmailDeliveryStatus) bool) []EmailDeliveryStatus {
	var statuses []EmailDeliveryStatus
	for _, s := range allEmailDeliveryStatuses {
		if predicate(s) {
			statuses = append(statuses, s)
		}
	}

	return statuses
}

// EmailDeliveryStatus represents the delivery status of a single recipient.
type EmailDeliveryStatus string

// EmailDeliveryStatus constants; pending, accepted, and soft_bounced are non-final, and completed/failed/bounced are final.
const (
	// EmailDeliveryPending is the initial state, awaiting a send.
	EmailDeliveryPending EmailDeliveryStatus = "pending"
	// EmailDeliveryAccepted means the provider accepted the message and asynchronous feedback is expected.
	EmailDeliveryAccepted EmailDeliveryStatus = "accepted"
	// EmailDeliveryCompleted means delivery was confirmed by feedback, or assumed after send for a provider without feedback.
	EmailDeliveryCompleted EmailDeliveryStatus = "completed"
	// EmailDeliverySoftBounced means feedback reported a transient bounce (e.g. mailbox full); delivery may still complete later.
	EmailDeliverySoftBounced EmailDeliveryStatus = "soft_bounced"
	// EmailDeliveryDelayed means feedback reported a transient delay at the recipient's mail server; delivery may still complete or bounce later.
	EmailDeliveryDelayed EmailDeliveryStatus = "delayed"
	// EmailDeliveryHardBounced means asynchronous feedback reported a permanent bounce (e.g. unknown address).
	EmailDeliveryHardBounced EmailDeliveryStatus = "hard_bounced"
	// EmailDeliveryFailed means feedback rejected the message, or the sender exhausted its retry budget.
	EmailDeliveryFailed EmailDeliveryStatus = "failed"
	// EmailDeliveryAbandoned means no delivery feedback arrived within the window, so the sweep gave up waiting.
	EmailDeliveryAbandoned EmailDeliveryStatus = "abandoned"
)

// IsFinal reports whether the status is a terminal state.
func (s EmailDeliveryStatus) IsFinal() bool {
	switch s {
	case EmailDeliveryCompleted,
		EmailDeliveryFailed,
		EmailDeliveryHardBounced,
		EmailDeliveryAbandoned:
		return true
	default:
		return false
	}
}

// IsRetryableStatus reports whether the status is eligible for a send attempt.
func (s EmailDeliveryStatus) IsRetryableStatus() bool {
	switch s {
	case EmailDeliveryPending,
		EmailDeliverySoftBounced,
		EmailDeliveryDelayed,
		EmailDeliveryAccepted:
		return true
	default:
		return false
	}
}

// IsIssueStatus reports whether the status counts toward the "issues" bucket; a complaint is tracked separately via complained_at.
func (s EmailDeliveryStatus) IsIssueStatus() bool {
	switch s {
	case EmailDeliveryFailed,
		EmailDeliveryHardBounced,
		EmailDeliverySoftBounced,
		EmailDeliveryAbandoned:
		return true
	default:
		return false
	}
}

// EmailRecipient represents a single (email, recipient) row in the send queue.
type EmailRecipient struct {
	AvailableAt       time.Time
	LastAttemptAt     *time.Time
	OpenedAt          *time.Time
	ClickedAt         *time.Time
	ComplainedAt      *time.Time
	FailureReason     *string
	EmailOutboxItemID string
	Address           string
	DeliveryStatus    EmailDeliveryStatus
	Metadata          ResourceMetadata
	AttemptCount      int
}

// GetID returns the Metadata ID.
func (e *EmailRecipient) GetID() string {
	return e.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (e *EmailRecipient) GetGlobalID() string {
	return gid.ToGlobalID(e.GetModelType(), e.Metadata.ID)
}

// GetModelType returns the type of the model.
func (e *EmailRecipient) GetModelType() types.ModelType {
	return types.EmailRecipientModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (e *EmailRecipient) ResolveMetadata(key string) (*string, error) {
	return e.Metadata.resolveFieldValue(key)
}

// Mutate runs fn and reports whether it changed any of the fields Equal compares.
func (e *EmailRecipient) Mutate(fn func()) bool {
	before := *e
	fn()
	return !e.Equal(&before)
}

// Equal reports whether the two recipients match on the fields the Record/Mark mutators touch.
func (e *EmailRecipient) Equal(other *EmailRecipient) bool {
	return e.DeliveryStatus == other.DeliveryStatus &&
		ptr.ToString(e.FailureReason) == ptr.ToString(other.FailureReason) &&
		e.AvailableAt.Equal(other.AvailableAt) &&
		ptr.ToTime(e.OpenedAt).Equal(ptr.ToTime(other.OpenedAt)) &&
		ptr.ToTime(e.ClickedAt).Equal(ptr.ToTime(other.ClickedAt)) &&
		ptr.ToTime(e.ComplainedAt).Equal(ptr.ToTime(other.ComplainedAt))
}

// RecordDelivery sets DeliveryStatus and FailureReason, re-pacing AvailableAt only when the status actually changes; returns whether anything changed.
func (e *EmailRecipient) RecordDelivery(status EmailDeliveryStatus, failureReason *string) bool {
	if e.DeliveryStatus.IsFinal() {
		return false
	}

	previousStatus := e.DeliveryStatus

	return e.Mutate(func() {
		e.DeliveryStatus = status

		if failureReason != nil {
			sanitized := sanitize.TruncateToValidUTF8(*failureReason, maxFailureReasonLength)
			if e.FailureReason == nil || *e.FailureReason != sanitized {
				e.FailureReason = &sanitized
			}
		}

		// Only re-pace on a real transition so a redelivered same-status event doesn't push the timer out.
		if status == previousStatus {
			return
		}

		switch {
		case status == EmailDeliveryAccepted:
			// Defer re-claim until the feedback window elapses; the sender abandons it if still accepted.
			e.AvailableAt = time.Now().UTC().Add(EmailFeedbackWindow)
		case status.IsRetryableStatus():
			e.AvailableAt = time.Now().UTC().Add(EmailRecipientRetryDelay)
		}
	})
}

// MarkOpened marks the message as opened, if not already. Independent of ComplainedAt, so a complainer's earlier open is preserved.
func (e *EmailRecipient) MarkOpened() bool {
	return e.Mutate(func() {
		if e.OpenedAt == nil {
			e.OpenedAt = new(time.Now().UTC())
		}
	})
}

// MarkClicked marks the message as clicked, if not already, and backfills OpenedAt if the
// provider's open-tracking pixel never fired (e.g. blocked images) — a click can't happen without an open.
func (e *EmailRecipient) MarkClicked() bool {
	return e.Mutate(func() {
		now := time.Now().UTC()

		if e.ClickedAt == nil {
			e.ClickedAt = &now
		}

		if e.OpenedAt == nil {
			e.OpenedAt = &now
		}
	})
}

// MarkComplained marks the message as complained about, if not already.
func (e *EmailRecipient) MarkComplained() bool {
	return e.Mutate(func() {
		if e.ComplainedAt == nil {
			e.ComplainedAt = new(time.Now().UTC())
		}
	})
}

// SuppressionCause derives the cause from state; the delivery arm wins for a bounced-then-complained recipient.
func (e *EmailRecipient) SuppressionCause() (EmailSuppressionCause, bool) {
	switch {
	case e.DeliveryStatus == EmailDeliveryHardBounced:
		return EmailSuppressionCauseHardBounce, true
	case e.ComplainedAt != nil:
		return EmailSuppressionCauseComplaint, true
	default:
		return "", false
	}
}

// Validate validates the email recipient fields.
func (e *EmailRecipient) Validate() error {
	return nil
}
