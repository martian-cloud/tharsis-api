package models

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

func TestEmailRecipientRecordDelivery(t *testing.T) {
	reason := "450 mailbox busy"

	tests := []struct {
		name           string
		current        EmailDeliveryStatus
		existingReason *string
		status         EmailDeliveryStatus
		reason         *string
		wantChanged    bool
		wantStatus     EmailDeliveryStatus
		wantReason     bool
	}{
		{
			name:        "accepted with no reason",
			current:     EmailDeliveryPending,
			status:      EmailDeliveryAccepted,
			wantChanged: true,
			wantStatus:  EmailDeliveryAccepted,
		},
		{
			name:        "completed with no reason",
			current:     EmailDeliveryAccepted,
			status:      EmailDeliveryCompleted,
			wantChanged: true,
			wantStatus:  EmailDeliveryCompleted,
		},
		{
			name:        "bounced records reason",
			current:     EmailDeliveryAccepted,
			status:      EmailDeliveryHardBounced,
			reason:      new("550 user unknown"),
			wantChanged: true,
			wantStatus:  EmailDeliveryHardBounced,
			wantReason:  true,
		},
		{
			name:        "failed records reason",
			current:     EmailDeliveryPending,
			status:      EmailDeliveryFailed,
			reason:      new("smtp error"),
			wantChanged: true,
			wantStatus:  EmailDeliveryFailed,
			wantReason:  true,
		},
		{
			name:        "same status with a new reason records it (soft bounce)",
			current:     EmailDeliveryAccepted,
			status:      EmailDeliveryAccepted,
			reason:      &reason,
			wantChanged: true,
			wantStatus:  EmailDeliveryAccepted,
			wantReason:  true,
		},
		{
			name:        "re-recording accepted with a nil reason is a no-op",
			current:     EmailDeliveryAccepted,
			status:      EmailDeliveryAccepted,
			wantChanged: false,
			wantStatus:  EmailDeliveryAccepted,
		},
		{
			name:           "re-recording accepted with an unchanged reason is a no-op",
			current:        EmailDeliveryAccepted,
			existingReason: &reason,
			status:         EmailDeliveryAccepted,
			reason:         &reason,
			wantChanged:    false,
			wantStatus:     EmailDeliveryAccepted,
			wantReason:     true,
		},
		{
			name:        "final status rejects further delivery",
			current:     EmailDeliveryHardBounced,
			status:      EmailDeliveryCompleted,
			wantChanged: false,
			wantStatus:  EmailDeliveryHardBounced,
		},
		{
			name:           "duplicate soft bounce with an unchanged reason is a no-op",
			current:        EmailDeliverySoftBounced,
			existingReason: &reason,
			status:         EmailDeliverySoftBounced,
			reason:         &reason,
			wantChanged:    false,
			wantStatus:     EmailDeliverySoftBounced,
			wantReason:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recipient := &EmailRecipient{DeliveryStatus: test.current, FailureReason: test.existingReason}

			changed := recipient.RecordDelivery(test.status, test.reason)

			assert.Equal(t, test.wantChanged, changed)
			assert.Equal(t, test.wantStatus, recipient.DeliveryStatus)
			assert.Equal(t, test.wantReason, recipient.FailureReason != nil)
		})
	}
}

func TestEmailRecipientRecordDeliverySchedulesRetryBackoff(t *testing.T) {
	for _, status := range []EmailDeliveryStatus{EmailDeliverySoftBounced, EmailDeliveryDelayed} {
		t.Run(string(status), func(t *testing.T) {
			recipient := &EmailRecipient{DeliveryStatus: EmailDeliveryAccepted, AvailableAt: time.Now().UTC()}

			before := time.Now().UTC()
			changed := recipient.RecordDelivery(status, nil)

			assert.True(t, changed)
			assert.True(t, recipient.AvailableAt.After(before))
		})
	}

	t.Run("hard bounce does not schedule a retry", func(t *testing.T) {
		fixed := time.Now().UTC()
		recipient := &EmailRecipient{DeliveryStatus: EmailDeliveryAccepted, AvailableAt: fixed}

		recipient.RecordDelivery(EmailDeliveryHardBounced, nil)

		assert.Equal(t, fixed, recipient.AvailableAt)
	})

	t.Run("a duplicate soft bounce does not re-pace the retry", func(t *testing.T) {
		recipient := &EmailRecipient{DeliveryStatus: EmailDeliveryAccepted, AvailableAt: time.Now().UTC()}

		// First soft bounce is a real transition: it changes and schedules a retry.
		assert.True(t, recipient.RecordDelivery(EmailDeliverySoftBounced, nil))
		paced := recipient.AvailableAt

		// A redelivered soft bounce is a no-op: no change and AvailableAt is left where the first set it.
		assert.False(t, recipient.RecordDelivery(EmailDeliverySoftBounced, nil))
		assert.Equal(t, paced, recipient.AvailableAt)
	})
}

func TestEmailRecipientRecordDeliveryTruncatesAndSanitizesReason(t *testing.T) {
	t.Run("truncates an oversized reason to the max length", func(t *testing.T) {
		long := strings.Repeat("x", maxFailureReasonLength+100)
		recipient := &EmailRecipient{DeliveryStatus: EmailDeliveryPending}

		changed := recipient.RecordDelivery(EmailDeliveryFailed, &long)

		assert.True(t, changed)
		require.NotNil(t, recipient.FailureReason)
		assert.Len(t, *recipient.FailureReason, maxFailureReasonLength)
	})

	t.Run("coerces invalid UTF-8 to the replacement character", func(t *testing.T) {
		invalid := "bounce \xff\xfe reason"
		recipient := &EmailRecipient{DeliveryStatus: EmailDeliveryPending}

		changed := recipient.RecordDelivery(EmailDeliveryHardBounced, &invalid)

		assert.True(t, changed)
		require.NotNil(t, recipient.FailureReason)
		assert.True(t, utf8.ValidString(*recipient.FailureReason))
		assert.NotContains(t, *recipient.FailureReason, "\xff")
	})
}

func TestEmailRecipientMarkOpened(t *testing.T) {
	t.Run("unopened records the timestamp", func(t *testing.T) {
		recipient := &EmailRecipient{}

		changed := recipient.MarkOpened()

		assert.True(t, changed)
		assert.NotNil(t, recipient.OpenedAt)
	})

	t.Run("already opened is a no-op", func(t *testing.T) {
		openedAt := time.Now()
		recipient := &EmailRecipient{OpenedAt: &openedAt}

		changed := recipient.MarkOpened()

		assert.False(t, changed)
		assert.True(t, recipient.OpenedAt.Equal(openedAt))
	})
}

func TestEmailRecipientMarkClicked(t *testing.T) {
	t.Run("unclicked records the timestamp and backfills OpenedAt with the same instant", func(t *testing.T) {
		recipient := &EmailRecipient{}

		changed := recipient.MarkClicked()

		assert.True(t, changed)
		require.NotNil(t, recipient.ClickedAt)
		require.NotNil(t, recipient.OpenedAt)
		assert.True(t, recipient.OpenedAt.Equal(*recipient.ClickedAt))
	})

	t.Run("already opened is preserved, not overwritten", func(t *testing.T) {
		openedAt := time.Now()
		recipient := &EmailRecipient{OpenedAt: &openedAt}

		recipient.MarkClicked()

		assert.True(t, recipient.OpenedAt.Equal(openedAt))
	})

	t.Run("already clicked is a no-op", func(t *testing.T) {
		clickedAt := time.Now()
		openedAt := time.Now()
		recipient := &EmailRecipient{ClickedAt: &clickedAt, OpenedAt: &openedAt}

		changed := recipient.MarkClicked()

		assert.False(t, changed)
		assert.True(t, recipient.ClickedAt.Equal(clickedAt))
	})
}

func TestEmailRecipientMarkComplained(t *testing.T) {
	t.Run("uncomplained records the timestamp", func(t *testing.T) {
		recipient := &EmailRecipient{}

		changed := recipient.MarkComplained()

		assert.True(t, changed)
		assert.NotNil(t, recipient.ComplainedAt)
	})

	t.Run("already complained is a no-op", func(t *testing.T) {
		complainedAt := time.Now()
		recipient := &EmailRecipient{ComplainedAt: &complainedAt}

		changed := recipient.MarkComplained()

		assert.False(t, changed)
		assert.True(t, recipient.ComplainedAt.Equal(complainedAt))
	})
}

func TestEmailRecipientEqual(t *testing.T) {
	now := time.Now().UTC()
	later := now.Add(time.Hour)
	reason := "bounced"
	otherReason := "complained"

	base := func() *EmailRecipient {
		return &EmailRecipient{
			DeliveryStatus: EmailDeliveryAccepted,
			FailureReason:  &reason,
			AvailableAt:    now,
			OpenedAt:       &now,
			ClickedAt:      &now,
			ComplainedAt:   &now,
			// Fields the mutators never touch must not affect equality.
			EmailOutboxItemID: "outbox-1",
			Address:           "user@example.com",
			AttemptCount:      3,
		}
	}

	tests := []struct {
		name   string
		base   func() *EmailRecipient
		mutate func(*EmailRecipient)
		want   bool
	}{
		{
			name:   "identical",
			mutate: func(*EmailRecipient) {},
			want:   true,
		},
		{
			name: "ignores non-mutator fields",
			mutate: func(r *EmailRecipient) {
				r.EmailOutboxItemID = "outbox-2"
				r.Address = "other@example.com"
				r.AttemptCount = 99
			},
			want: true,
		},
		{
			name:   "delivery status differs",
			mutate: func(r *EmailRecipient) { r.DeliveryStatus = EmailDeliveryCompleted },
			want:   false,
		},
		{
			name:   "failure reason value differs",
			mutate: func(r *EmailRecipient) { r.FailureReason = &otherReason },
			want:   false,
		},
		{
			name:   "failure reason nil vs set differs",
			mutate: func(r *EmailRecipient) { r.FailureReason = nil },
			want:   false,
		},
		{
			name:   "available at differs",
			mutate: func(r *EmailRecipient) { r.AvailableAt = later },
			want:   false,
		},
		{
			name:   "opened at differs",
			mutate: func(r *EmailRecipient) { r.OpenedAt = &later },
			want:   false,
		},
		{
			name:   "clicked at differs",
			mutate: func(r *EmailRecipient) { r.ClickedAt = &later },
			want:   false,
		},
		{
			name:   "complained at differs",
			mutate: func(r *EmailRecipient) { r.ComplainedAt = &later },
			want:   false,
		},
		{
			name:   "both nil pointers are equal",
			base:   func() *EmailRecipient { return &EmailRecipient{DeliveryStatus: EmailDeliveryPending} },
			mutate: func(*EmailRecipient) {},
			want:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			makeRecipient := base
			if test.base != nil {
				makeRecipient = test.base
			}

			other := makeRecipient()
			test.mutate(other)

			assert.Equal(t, test.want, makeRecipient().Equal(other))
		})
	}
}

func TestEmailRecipientSuppressionCause(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		delivery     EmailDeliveryStatus
		complainedAt *time.Time
		wantCause    EmailSuppressionCause
		wantOK       bool
	}{
		{
			name:      "bounced suppresses as bounce",
			delivery:  EmailDeliveryHardBounced,
			wantCause: EmailSuppressionCauseHardBounce,
			wantOK:    true,
		},
		{
			name:         "complained suppresses as complaint",
			delivery:     EmailDeliveryCompleted,
			complainedAt: &now,
			wantCause:    EmailSuppressionCauseComplaint,
			wantOK:       true,
		},
		{
			name:         "bounced and complained: bounce wins",
			delivery:     EmailDeliveryHardBounced,
			complainedAt: &now,
			wantCause:    EmailSuppressionCauseHardBounce,
			wantOK:       true,
		},
		{
			name:     "no terminal event does not suppress",
			delivery: EmailDeliveryCompleted,
			wantOK:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recipient := &EmailRecipient{DeliveryStatus: test.delivery, ComplainedAt: test.complainedAt}

			cause, ok := recipient.SuppressionCause()

			assert.Equal(t, test.wantOK, ok)
			assert.Equal(t, test.wantCause, cause)
		})
	}
}

func TestEmailRecipientIdentity(t *testing.T) {
	recipient := &EmailRecipient{}
	recipient.Metadata.ID = "rec-1"

	assert.Equal(t, "rec-1", recipient.GetID())
	assert.Equal(t, types.EmailRecipientModelType, recipient.GetModelType())
	assert.NotEmpty(t, recipient.GetGlobalID())
	assert.NoError(t, recipient.Validate())

	val, err := recipient.ResolveMetadata("id")
	require.NoError(t, err)
	require.NotNil(t, val)
	assert.Equal(t, "rec-1", *val)
}

func TestEmailDeliveryStatusPredicates(t *testing.T) {
	tests := []struct {
		status        EmailDeliveryStatus
		wantFinal     bool
		wantRetryable bool
		wantIssue     bool
	}{
		{status: EmailDeliveryPending, wantFinal: false, wantRetryable: true, wantIssue: false},
		{status: EmailDeliverySoftBounced, wantFinal: false, wantRetryable: true, wantIssue: true},
		{status: EmailDeliveryDelayed, wantFinal: false, wantRetryable: true, wantIssue: false},
		{status: EmailDeliveryAccepted, wantFinal: false, wantRetryable: true, wantIssue: false},
		{status: EmailDeliveryCompleted, wantFinal: true, wantRetryable: false, wantIssue: false},
		{status: EmailDeliveryFailed, wantFinal: true, wantRetryable: false, wantIssue: true},
		{status: EmailDeliveryHardBounced, wantFinal: true, wantRetryable: false, wantIssue: true},
		{status: EmailDeliveryAbandoned, wantFinal: true, wantRetryable: false, wantIssue: true},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			assert.Equal(t, test.wantFinal, test.status.IsFinal())
			assert.Equal(t, test.wantRetryable, test.status.IsRetryableStatus())
			assert.Equal(t, test.wantIssue, test.status.IsIssueStatus())
		})
	}
}

func TestEmailDeliveryStatusesMatching(t *testing.T) {
	tests := []struct {
		name      string
		predicate func(EmailDeliveryStatus) bool
		want      []EmailDeliveryStatus
	}{
		{
			name:      "issue statuses",
			predicate: EmailDeliveryStatus.IsIssueStatus,
			want:      []EmailDeliveryStatus{EmailDeliverySoftBounced, EmailDeliveryHardBounced, EmailDeliveryFailed, EmailDeliveryAbandoned},
		},
		{
			name:      "retryable statuses",
			predicate: EmailDeliveryStatus.IsRetryableStatus,
			want:      []EmailDeliveryStatus{EmailDeliveryPending, EmailDeliveryAccepted, EmailDeliverySoftBounced, EmailDeliveryDelayed},
		},
		{
			name:      "none match",
			predicate: func(EmailDeliveryStatus) bool { return false },
			want:      nil,
		},
		{
			name:      "all match",
			predicate: func(EmailDeliveryStatus) bool { return true },
			want:      allEmailDeliveryStatuses,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.ElementsMatch(t, test.want, EmailDeliveryStatusesMatching(test.predicate))
		})
	}
}
