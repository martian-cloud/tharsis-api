// Package email supports sending emails.
package email

//go:generate go tool mockery --name Provider --inpackage --case underscore
//go:generate go tool mockery --name FeedbackProvider --inpackage --case underscore

import (
	"context"
	"errors"
)

var (
	// ErrProviderRateLimited is the sentinel a Provider returns from SendMail when it is being throttled (rate/quota exceeded).
	ErrProviderRateLimited = errors.New("email provider rate limited")

	// ErrProviderUnavailable is the sentinel a Provider returns from SendMail on a transient, pass-wide failure (sending paused, connection failure).
	ErrProviderUnavailable = errors.New("email provider temporarily unavailable")

	// ErrProviderRejected is the sentinel a Provider returns from SendMail when it permanently refuses the
	// message (e.g. a malformed address), so retrying the same send will never succeed.
	ErrProviderRejected = errors.New("email provider rejected message")
)

// FeedbackEventType is the kind of asynchronous delivery feedback a provider reports.
type FeedbackEventType string

// FeedbackEventType constants.
const (
	// EventTypeDelivered means the provider confirmed delivery to the recipient.
	EventTypeDelivered FeedbackEventType = "delivered"
	// EventTypeSoftBounced means the message bounced transiently; delivery may still complete later.
	EventTypeSoftBounced FeedbackEventType = "soft_bounced"
	// EventTypeHardBounced means the message bounced permanently.
	EventTypeHardBounced FeedbackEventType = "hard_bounced"
	// EventTypeComplaint means the recipient reported the message as spam.
	EventTypeComplaint FeedbackEventType = "complaint"
	// EventTypeOpened means the recipient opened the message.
	EventTypeOpened FeedbackEventType = "opened"
	// EventTypeRejected means the provider refused to send the message (e.g. it failed a virus scan).
	EventTypeRejected FeedbackEventType = "rejected"
	// EventTypeDelayed means delivery is temporarily delayed; it may still succeed or bounce later.
	EventTypeDelayed FeedbackEventType = "delayed"
)

// FeedbackEvent is a single asynchronous delivery-feedback event correlated to a recipient by CorrelationID.
type FeedbackEvent struct {
	// CorrelationID is the opaque identifier the send was tagged with (the email_recipients row ID).
	CorrelationID string
	// Type is the kind of feedback.
	Type FeedbackEventType
	// FailureReason is a human-readable description of a bounce, complaint, rejection, or delay; nil for a delivery or open.
	FailureReason *string
}

// FeedbackCallback handles a single feedback event; an error signals non-durable processing so the provider redelivers rather than drops it.
type FeedbackCallback func(ctx context.Context, event FeedbackEvent) error

// Provider is an interface for sending emails.
type Provider interface {
	// SendMail sends an email to a single recipient (one per message keeps SES bounce/complaint feedback attributable); correlationID tags the send so feedback events map back to the recipient.
	SendMail(ctx context.Context, to, subject, body, correlationID string) error
}

// FeedbackProvider is a Provider that also delivers asynchronous delivery feedback; today only SES
// implements it. Callers type-assert for it rather than calling a capability-flag method, so a
// provider that doesn't support feedback needs no stub methods at all.
type FeedbackProvider interface {
	Provider

	// ConsumeFeedback blocks, invoking handle per delivery/bounce/complaint event until ctx is cancelled.
	ConsumeFeedback(ctx context.Context, handle FeedbackCallback) error
}

// NoopProvider is an email provider that doesn't send any emails.
type NoopProvider struct{}

// SendMail is a noop.
func (n *NoopProvider) SendMail(_ context.Context, _, _, _, _ string) error {
	return nil
}
