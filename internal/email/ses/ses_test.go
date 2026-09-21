package ses

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
)

func TestNormalizeSendMailError(t *testing.T) {
	genericErr := errors.New("boom")

	tests := []struct {
		input        error
		wantSentinel error
		name         string
	}{
		{
			name:         "account sending paused maps to unavailable",
			input:        &types.AccountSendingPausedException{},
			wantSentinel: email.ErrProviderUnavailable,
		},
		{
			name:         "configuration set sending paused maps to unavailable",
			input:        &types.ConfigurationSetSendingPausedException{},
			wantSentinel: email.ErrProviderUnavailable,
		},
		{
			name:         "limit exceeded maps to rate limited",
			input:        &types.LimitExceededException{},
			wantSentinel: email.ErrProviderRateLimited,
		},
		{
			name:         "throttling API error maps to rate limited",
			input:        &smithy.GenericAPIError{Code: "Throttling", Message: "rate exceeded"},
			wantSentinel: email.ErrProviderRateLimited,
		},
		{
			name:         "message rejected maps to rejected",
			input:        &smithy.GenericAPIError{Code: "MessageRejected", Message: "address blacklisted"},
			wantSentinel: email.ErrProviderRejected,
		},
		{
			name:         "invalid parameter value maps to rejected",
			input:        &smithy.GenericAPIError{Code: "InvalidParameterValue", Message: "invalid address"},
			wantSentinel: email.ErrProviderRejected,
		},
		{
			name:  "other API error passes through",
			input: &smithy.GenericAPIError{Code: "SomethingElse", Message: "unexpected"},
		},
		{
			name:  "generic error passes through",
			input: genericErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeSendMailError(test.input)

			if test.wantSentinel == nil {
				assert.Equal(t, test.input, got)
				assert.False(t, errors.Is(got, email.ErrProviderUnavailable))
				assert.False(t, errors.Is(got, email.ErrProviderRateLimited))
				return
			}

			assert.True(t, errors.Is(got, test.wantSentinel))
		})
	}
}

// snsWrap wraps an SES notification JSON string in the SNS envelope the SQS body carries.
func snsWrap(t *testing.T, notification string) string {
	t.Helper()
	body, err := json.Marshal(snsEnvelope{Type: "Notification", Message: notification})
	require.NoError(t, err)
	return string(body)
}

func TestParseFeedbackMessage(t *testing.T) {
	const ref = "018f0000-0000-7000-8000-000000000000"

	tagged := `"mail":{"tags":{"tharsis_email_recipient_id":["` + ref + `"]}}`

	tests := []struct {
		name        string
		body        string
		filter      string
		want        *email.FeedbackEvent
		wantErr     bool
		wantNotOurs bool
	}{
		{
			name: "delivery",
			body: snsWrap(t, `{"eventType":"Delivery",`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeDelivered},
		},
		{
			name: "permanent bounce with diagnostic",
			body: snsWrap(t, `{"eventType":"Bounce","bounce":{"bounceType":"Permanent","bouncedRecipients":[{"diagnosticCode":"smtp; 550 user unknown"}]},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeHardBounced, FailureReason: new("smtp; 550 user unknown")},
		},
		{
			name: "transient bounce is soft",
			body: snsWrap(t, `{"eventType":"Bounce","bounce":{"bounceType":"Transient","bouncedRecipients":[{"diagnosticCode":"smtp; 450 mailbox busy"}]},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeSoftBounced, FailureReason: new("smtp; 450 mailbox busy")},
		},
		{
			name: "undetermined bounce defaults to hard",
			body: snsWrap(t, `{"eventType":"Bounce","bounce":{"bounceType":"Undetermined","bouncedRecipients":[{"diagnosticCode":"smtp; 550 5.4.4 Invalid domain"}]},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeHardBounced, FailureReason: new("smtp; 550 5.4.4 Invalid domain")},
		},
		{
			name: "transient bounceType with a permanent enhanced status code is hard",
			body: snsWrap(t, `{"eventType":"Bounce","bounce":{"bounceType":"Transient","bouncedRecipients":[{"status":"5.4.4","diagnosticCode":"smtp; 550 5.4.4 Invalid domain"}]},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeHardBounced, FailureReason: new("smtp; 550 5.4.4 Invalid domain")},
		},
		{
			name: "permanent bounceType with a transient enhanced status code is soft",
			body: snsWrap(t, `{"eventType":"Bounce","bounce":{"bounceType":"Permanent","bouncedRecipients":[{"status":"4.2.2","diagnosticCode":"smtp; 550 4.2.2 mailbox full"}]},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeSoftBounced, FailureReason: new("smtp; 550 4.2.2 mailbox full")},
		},
		{
			name: "complaint",
			body: snsWrap(t, `{"eventType":"Complaint","complaint":{"complaintFeedbackType":"abuse"},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeComplaint, FailureReason: new("abuse")},
		},
		{
			name: "open",
			body: snsWrap(t, `{"eventType":"Open","open":{"isBotEvent":"Unlikely"},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeOpened},
		},
		{
			name: "open flagged as a likely bot yields no event",
			body: snsWrap(t, `{"eventType":"Open","open":{"isBotEvent":"Likely"},`+tagged+`}`),
			want: nil,
		},
		{
			name: "reject",
			body: snsWrap(t, `{"eventType":"Reject","reject":{"reason":"Bad content"},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeRejected, FailureReason: new("Bad content")},
		},
		{
			name: "delivery delay with diagnostic",
			body: snsWrap(t, `{"eventType":"DeliveryDelay","deliveryDelay":{"delayedRecipients":[{"diagnosticCode":"smtp; 450 4.2.1 mailbox busy"}]},`+tagged+`}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeDelayed, FailureReason: new("smtp; 450 4.2.1 mailbox busy")},
		},
		{
			name: "missing reference tag yields no event",
			body: snsWrap(t, `{"eventType":"Delivery","mail":{"tags":{}}}`),
			want: nil,
		},
		{
			name: "unhandled event type yields no event",
			body: snsWrap(t, `{"eventType":"Click",`+tagged+`}`),
			want: nil,
		},
		{
			name:    "malformed SNS envelope errors",
			body:    "not json",
			wantErr: true,
		},
		{
			name:    "malformed SES notification errors",
			body:    snsWrap(t, "not json"),
			wantErr: true,
		},
		{
			name:   "matching filter tag is handled",
			body:   snsWrap(t, `{"eventType":"Delivery","mail":{"tags":{"tharsis_email_recipient_id":["`+ref+`"],"tharsis_email_feedback_queue_filter":["mine"]}}}`),
			filter: "mine",
			want:   &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeDelivered},
		},
		{
			name:        "mismatched filter tag is not ours",
			body:        snsWrap(t, `{"eventType":"Delivery","mail":{"tags":{"tharsis_email_recipient_id":["`+ref+`"],"tharsis_email_feedback_queue_filter":["theirs"]}}}`),
			filter:      "mine",
			wantNotOurs: true,
		},
		{
			name:        "missing filter tag when a filter is configured is not ours",
			body:        snsWrap(t, `{"eventType":"Delivery",`+tagged+`}`),
			filter:      "mine",
			wantNotOurs: true,
		},
		{
			name: "filter tag ignored when no filter configured",
			body: snsWrap(t, `{"eventType":"Delivery","mail":{"tags":{"tharsis_email_recipient_id":["`+ref+`"],"tharsis_email_feedback_queue_filter":["theirs"]}}}`),
			want: &email.FeedbackEvent{CorrelationID: ref, Type: email.EventTypeDelivered},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseFeedbackMessage(test.body, test.filter)
			if test.wantNotOurs {
				assert.ErrorIs(t, err, errNotOurEvent)
				return
			}
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestBuildMessageTags(t *testing.T) {
	const ref = "018f0000-0000-7000-8000-000000000000"

	tagValues := func(tags []types.MessageTag) map[string]string {
		out := map[string]string{}
		for _, tag := range tags {
			out[*tag.Name] = *tag.Value
		}
		return out
	}

	t.Run("without feedback filter only tags the recipient reference", func(t *testing.T) {
		p := &sesProvider{}
		tags := p.buildMessageTags(ref)
		values := tagValues(tags)

		assert.Len(t, tags, 1)
		assert.Equal(t, ref, values[referenceTagName])
		_, hasFilter := values[feedbackQueueFilterTagName]
		assert.False(t, hasFilter)
	})

	t.Run("with feedback filter adds the feedback filter tag", func(t *testing.T) {
		p := &sesProvider{feedbackQueueFilter: "tharsis-internal"}
		tags := p.buildMessageTags(ref)
		values := tagValues(tags)

		assert.Len(t, tags, 2)
		assert.Equal(t, ref, values[referenceTagName])
		assert.Equal(t, "tharsis-internal", values[feedbackQueueFilterTagName])
	})
}
