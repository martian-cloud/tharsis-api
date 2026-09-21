// Package ses defines the ses email plugin
package ses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// referenceTagName is the SES message-tag name carrying the recipient reference, echoed in feedback notifications to correlate events.
	referenceTagName = "tharsis_email_recipient_id"

	// feedbackQueueFilterTagName is an optional SES message-tag set only when multiple Tharsis instances
	// share a single SES feedback SQS queue; echoed in feedback notifications so each instance's
	// consumer can filter out events that aren't its own.
	feedbackQueueFilterTagName = "tharsis_email_feedback_queue_filter"

	// feedbackReceiveMaxMessages is how many SQS messages one receive call fetches (SQS max is 10).
	feedbackReceiveMaxMessages = 10
	// feedbackReceiveWaitSeconds enables long polling so idle receives don't spin.
	feedbackReceiveWaitSeconds = 20
	// feedbackVisibilityTimeoutSeconds hides a received message while handle runs.
	feedbackVisibilityTimeoutSeconds = 60
	// feedbackHandlerConcurrency caps how many received batches are handled concurrently so slow handling can't accumulate unbounded goroutines.
	feedbackHandlerConcurrency = 10
	// feedbackReceiveErrorBackoff is how long the consumer pauses after a receive failure so a persistent error doesn't spin.
	feedbackReceiveErrorBackoff = 5 * time.Second
)

// SES notification eventType values we act on.
const (
	eventTypeDelivery      = "Delivery"
	eventTypeBounce        = "Bounce"
	eventTypeComplaint     = "Complaint"
	eventTypeOpen          = "Open"
	eventTypeReject        = "Reject"
	eventTypeDeliveryDelay = "DeliveryDelay"
	bounceTypeTransient    = "Transient" // SES bounceType for a soft bounce; anything else (Permanent or Undetermined) is treated as hard.
)

// RFC 3463 enhanced status code class digits.
const (
	enhancedStatusClassTransient byte = '4'
	enhancedStatusClassPermanent byte = '5'
)

// botEventLikely is the SES open/click isBotEvent value for an automated scanner/prefetcher, not a real recipient.
const botEventLikely = "Likely"

// errNotOurEvent signals a feedback notification tagged for another instance on a shared queue; the message is left for its owner rather than deleted.
var errNotOurEvent = errors.New("feedback event belongs to another instance")

// snsTypeNotification is the SNS envelope Type for a delivery notification; other types (e.g. SubscriptionConfirmation) carry no SES event.
const snsTypeNotification = "Notification"

// snsEnvelope is the SNS wrapper around an SES event notification delivered to SQS.
type snsEnvelope struct {
	Type    string `json:"Type"`
	Message string `json:"Message"`
}

// sesNotification is the SES event-publishing notification (the SNS Message payload).
type sesNotification struct {
	EventType string `json:"eventType"`
	Mail      struct {
		Tags map[string][]string `json:"tags"`
	} `json:"mail"`
	Bounce *struct {
		BounceType        string `json:"bounceType"`
		BouncedRecipients []struct {
			Status         string `json:"status"`
			DiagnosticCode string `json:"diagnosticCode"`
		} `json:"bouncedRecipients"`
	} `json:"bounce"`
	Complaint *struct {
		ComplaintFeedbackType string `json:"complaintFeedbackType"`
	} `json:"complaint"`
	Reject *struct {
		Reason string `json:"reason"`
	} `json:"reject"`
	DeliveryDelay *struct {
		DelayedRecipients []struct {
			DiagnosticCode string `json:"diagnosticCode"`
		} `json:"delayedRecipients"`
	} `json:"deliveryDelay"`
	Open *struct {
		IsBotEvent string `json:"isBotEvent"`
	} `json:"open"`
}

type sesProvider struct {
	fromAddress             string
	awsConfigurationSetName string
	feedbackQueueFilter     string
	logger                  logger.Logger
	awsClient               *ses.Client
}

// sesFeedbackProvider adds SES's SQS-based asynchronous delivery feedback on top of sesProvider;
// SES is the only email provider that supports feedback, and only when a feedback queue is configured.
type sesFeedbackProvider struct {
	*sesProvider
	sqsQueueURL string
	sqsClient   *sqs.Client
}

// NewProvider returns a new provider instance. sqsQueueURL is the SES feedback queue; when empty,
// the returned provider only implements email.Provider, so callers that type-assert for
// email.FeedbackProvider correctly see no feedback support.
func NewProvider(
	ctx context.Context,
	logger logger.Logger,
	fromAddress string,
	awsConfigurationSetName string,
	region string,
	sqsQueueURL string,
	feedbackQueueFilter string,
) (email.Provider, error) {

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	provider := &sesProvider{
		fromAddress:             fromAddress,
		awsConfigurationSetName: awsConfigurationSetName,
		feedbackQueueFilter:     feedbackQueueFilter,
		logger:                  logger,
		awsClient:               ses.NewFromConfig(awsCfg),
	}

	if sqsQueueURL == "" {
		return provider, nil
	}

	return &sesFeedbackProvider{
		sesProvider: provider,
		sqsQueueURL: sqsQueueURL,
		sqsClient:   sqs.NewFromConfig(awsCfg),
	}, nil
}

func (s *sesProvider) SendMail(ctx context.Context, to, subject, body, correlationID string) error {
	_, err := s.awsClient.SendEmail(ctx, &ses.SendEmailInput{
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &types.Message{
			Body: &types.Body{
				Html: &types.Content{
					Data: aws.String(body),
				},
			},
			Subject: &types.Content{
				Data: aws.String(subject),
			},
		},
		Source:               aws.String(fmt.Sprintf("Tharsis <%s>", s.fromAddress)),
		ConfigurationSetName: aws.String(s.awsConfigurationSetName),
		Tags:                 s.buildMessageTags(correlationID),
	})
	if err != nil {
		return normalizeSendMailError(err)
	}

	return nil
}

// ConsumeFeedback long-polls the SQS feedback queue, dispatching each received batch to a bounded pool of
// goroutines so the next receive can start immediately without letting slow handling accumulate unbounded
// goroutines, until ctx is cancelled; it waits for in-flight batches before returning. A message is deleted
// only after handle returns nil so failures redeliver.
func (s *sesFeedbackProvider) ConsumeFeedback(ctx context.Context, handle email.FeedbackCallback) error {
	s.logger.Info("ses feedback consumer started")

	var wg sync.WaitGroup
	// A buffered token channel caps how many batches are handled concurrently.
	sem := make(chan struct{}, feedbackHandlerConcurrency)

	for {
		if err := ctx.Err(); err != nil {
			s.logger.Info("ses feedback consumer stopped")
			wg.Wait()
			return err
		}

		out, err := s.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.sqsQueueURL),
			MaxNumberOfMessages: feedbackReceiveMaxMessages,
			WaitTimeSeconds:     feedbackReceiveWaitSeconds,
			VisibilityTimeout:   feedbackVisibilityTimeoutSeconds,
		})
		if err != nil {
			// Any error (including context cancellation on shutdown) leaves out nil, so skip handling.
			// Log only genuine failures; a cancelled context is an expected shutdown, and the
			// top-of-loop ctx check exits the loop on the next iteration.
			if te.FilterContextError(err) != nil {
				s.logger.WithContextFields(ctx).Errorf("ses feedback consumer failed to receive messages: %v", err)

				// Back off so a persistent receive failure doesn't spin a hot loop.
				select {
				case <-time.After(feedbackReceiveErrorBackoff):
				case <-ctx.Done():
				}
			}

			continue
		}

		if len(out.Messages) == 0 {
			continue
		}

		// Acquire a token before dispatching; block (honoring ctx) when the pool is full so goroutines stay bounded.
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			continue
		}

		wg.Add(1)
		go func(messages []sqstypes.Message) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := s.handleFeedbackMessages(ctx, messages, handle); err != nil {
				s.logger.WithContextFields(ctx).Errorf("ses feedback consumer failed to handle messages: %v", err)
			}
		}(out.Messages)
	}
}

// handleFeedbackMessages delivers each message's events and acks it (deletes it from SQS); a message is not acked when handle fails, so SQS redelivers it after the visibility timeout for retry. Per-message errors are collected and returned so the caller logs them without aborting the batch.
func (s *sesFeedbackProvider) handleFeedbackMessages(ctx context.Context, messages []sqstypes.Message, handle email.FeedbackCallback) error {
	var errs []error

	for _, message := range messages {
		body := aws.ToString(message.Body)
		s.logger.WithContextFields(ctx).Debugw("ses feedback notification", "body", body)

		event, err := parseFeedbackMessage(body, s.feedbackQueueFilter)
		if errors.Is(err, errNotOurEvent) {
			// Another instance owns this event on the shared queue; leave it visible for that instance.
			continue
		} else if err != nil {
			// A malformed message can't be parsed, so redelivery won't help; fall through to ack and drop it.
			errs = append(errs, te.Wrap(err, "dropping unparseable message"))
		} else if event != nil {
			if err := handle(ctx, *event); err != nil {
				// Leave the message so SQS redelivers it after the visibility timeout.
				errs = append(errs, te.Wrap(err, "handler failed for reference %s", event.CorrelationID))
				continue
			}
		}

		// Ack the message so SQS removes it; without this at-least-once delivery would keep redelivering it.
		if _, err := s.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(s.sqsQueueURL),
			ReceiptHandle: message.ReceiptHandle,
		}); err != nil && te.FilterContextError(err) != nil {
			// A delete failure just redelivers the message; handle is idempotent per reference.
			errs = append(errs, te.Wrap(err, "failed to delete message"))
		}
	}

	return errors.Join(errs...)
}

// buildMessageTags returns the SES message tags for a send: always the recipient reference (to
// correlate feedback), plus the feedback queue filter tag when configured (so a feedback queue shared
// across instances can be filtered to just this one).
func (s *sesProvider) buildMessageTags(correlationID string) []types.MessageTag {
	tags := []types.MessageTag{
		{
			Name:  aws.String(referenceTagName),
			Value: aws.String(correlationID),
		},
	}

	if s.feedbackQueueFilter != "" {
		tags = append(tags, types.MessageTag{
			Name:  aws.String(feedbackQueueFilterTagName),
			Value: aws.String(s.feedbackQueueFilter),
		})
	}

	return tags
}

// normalizeSendMailError maps SES SendMail errors to the provider-agnostic sentinels.
func normalizeSendMailError(err error) error {
	var (
		accountPaused *types.AccountSendingPausedException
		configPaused  *types.ConfigurationSetSendingPausedException
		limitExceeded *types.LimitExceededException
	)

	if errors.As(err, &accountPaused) || errors.As(err, &configPaused) {
		return fmt.Errorf("%w: %v", email.ErrProviderUnavailable, err)
	}

	// The SDK's own classifier catches the generic AWS throttle codes, so we don't hardcode the wire codes.
	throttleClassifier := retry.RetryableErrorCode{Codes: retry.DefaultThrottleErrorCodes}

	if errors.As(err, &limitExceeded) || throttleClassifier.IsErrorRetryable(err).Bool() {
		return fmt.Errorf("%w: %v", email.ErrProviderRateLimited, err)
	}

	// A rejected message or invalid parameter (e.g. a malformed address) is permanent, so classify it rather than letting the sender retry forever.
	if apiErr, ok := errors.AsType[smithy.APIError](err); ok {
		switch apiErr.ErrorCode() {
		case "MessageRejected", "InvalidParameterValue":
			return fmt.Errorf("%w: %v", email.ErrProviderRejected, err)
		}
	}

	return err
}

// parseFeedbackMessage unwraps the SNS envelope and SES notification into a feedback event, returning nil for events we don't act on and errNotOurEvent when a configured filter doesn't match this instance's tag.
func parseFeedbackMessage(body, filter string) (*email.FeedbackEvent, error) {
	var envelope snsEnvelope
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return nil, te.Wrap(err, "failed to unmarshal SNS envelope")
	}

	// Only delivery notifications carry an SES event; anything else (e.g. a SubscriptionConfirmation) has no event to apply.
	if envelope.Type != snsTypeNotification {
		return nil, nil
	}

	// The SES notification is a JSON string inside the SNS envelope's Message field.
	var notification sesNotification
	if err := json.Unmarshal([]byte(envelope.Message), &notification); err != nil {
		return nil, te.Wrap(err, "failed to unmarshal SES notification")
	}

	// On a shared queue, drop events whose filter tag isn't ours by leaving them for the owning instance.
	if filter != "" {
		var tag string
		if values := notification.Mail.Tags[feedbackQueueFilterTagName]; len(values) > 0 {
			tag = values[0]
		}

		if tag != filter {
			return nil, errNotOurEvent
		}
	}

	var correlationID string
	if values := notification.Mail.Tags[referenceTagName]; len(values) > 0 {
		correlationID = values[0]
	}

	if correlationID == "" {
		// Without our reference tag we can't correlate the event to a recipient; nothing to do.
		return nil, nil
	}

	switch notification.EventType {
	case eventTypeDelivery:
		return &email.FeedbackEvent{CorrelationID: correlationID, Type: email.EventTypeDelivered}, nil
	case eventTypeBounce:
		// SES treats Undetermined the same as Permanent for reputation purposes, so default to hard and
		// only downgrade to soft on an explicit Transient, rather than defaulting to soft.
		eventType := email.EventTypeHardBounced
		var reason *string
		if notification.Bounce != nil {
			if notification.Bounce.BounceType == bounceTypeTransient {
				eventType = email.EventTypeSoftBounced
			}

			if len(notification.Bounce.BouncedRecipients) > 0 {
				recipient := notification.Bounce.BouncedRecipients[0]
				reason = &recipient.DiagnosticCode

				// The recipient's own RFC 3463 enhanced status code (e.g. "5.4.4") is more precise than
				// SES's own bounceType, which can report Transient for a diagnostic that's unambiguously
				// permanent; its leading digit (5=permanent, 4=transient) overrides bounceType when present.
				var statusClass byte
				if len(recipient.Status) > 0 {
					statusClass = recipient.Status[0]
				}

				switch statusClass {
				case enhancedStatusClassPermanent:
					eventType = email.EventTypeHardBounced
				case enhancedStatusClassTransient:
					eventType = email.EventTypeSoftBounced
				}
			}
		}

		return &email.FeedbackEvent{CorrelationID: correlationID, Type: eventType, FailureReason: reason}, nil
	case eventTypeComplaint:
		event := email.FeedbackEvent{CorrelationID: correlationID, Type: email.EventTypeComplaint}
		if notification.Complaint != nil {
			event.FailureReason = &notification.Complaint.ComplaintFeedbackType
		}

		return &event, nil
	case eventTypeOpen:
		if notification.Open != nil && notification.Open.IsBotEvent == botEventLikely {
			// SES flagged this pixel fetch as an automated scanner/prefetcher, not a real recipient.
			return nil, nil
		}

		return &email.FeedbackEvent{CorrelationID: correlationID, Type: email.EventTypeOpened}, nil
	case eventTypeReject:
		event := email.FeedbackEvent{CorrelationID: correlationID, Type: email.EventTypeRejected}
		if notification.Reject != nil {
			event.FailureReason = &notification.Reject.Reason
		}

		return &event, nil
	case eventTypeDeliveryDelay:
		event := email.FeedbackEvent{CorrelationID: correlationID, Type: email.EventTypeDelayed}
		if notification.DeliveryDelay != nil && len(notification.DeliveryDelay.DelayedRecipients) > 0 {
			event.FailureReason = &notification.DeliveryDelay.DelayedRecipients[0].DiagnosticCode
		}

		return &event, nil
	default:
		return nil, nil
	}
}
