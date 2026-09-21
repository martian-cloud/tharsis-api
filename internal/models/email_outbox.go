package models

import (
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*EmailOutboxItem)(nil)

// MaxEmailSubjectLength bounds the stored subject line for an outbox item.
const MaxEmailSubjectLength = 255

// EmailOutboxItemStatus tracks an outbox item's readiness to send, gated on async recipient materialization.
type EmailOutboxItemStatus string

// EmailOutboxItemStatus constants.
const (
	// EmailOutboxItemStatusPreparing means the recipients have not yet been materialized by the async worker.
	EmailOutboxItemStatusPreparing EmailOutboxItemStatus = "preparing"
	// EmailOutboxItemStatusReady means all recipient rows have been stored and the item is sendable.
	EmailOutboxItemStatusReady EmailOutboxItemStatus = "ready"
	// EmailOutboxItemStatusCompleted means every recipient has reached a final delivery status; the item is done.
	EmailOutboxItemStatusCompleted EmailOutboxItemStatus = "completed"
	// EmailOutboxItemStatusFailed means recipient materialization failed permanently.
	EmailOutboxItemStatusFailed EmailOutboxItemStatus = "failed"
)

// IsValid reports whether the status is one of the recognized outbox item statuses.
func (s EmailOutboxItemStatus) IsValid() bool {
	switch s {
	case EmailOutboxItemStatusPreparing,
		EmailOutboxItemStatusReady,
		EmailOutboxItemStatusCompleted,
		EmailOutboxItemStatusFailed:
		return true
	default:
		return false
	}
}

// EmailOutboxItem represents one logical email: a single rendered payload fanned out
// to one or more recipients tracked in email_recipients.
type EmailOutboxItem struct {
	Metadata              ResourceMetadata
	EmailType             builder.EmailType
	Subject               string
	Payload               []byte
	PayloadObjectStoreKey *string
	Status                EmailOutboxItemStatus
	RecipientUserIDs      []string
	RecipientTeamIDs      []string
	SendAt                *time.Time
	Ephemeral             bool
	SendToAllUsers        bool
	ClaimedAt             *time.Time
}

// GetID returns the Metadata ID.
func (e *EmailOutboxItem) GetID() string {
	return e.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (e *EmailOutboxItem) GetGlobalID() string {
	return gid.ToGlobalID(e.GetModelType(), e.Metadata.ID)
}

// GetModelType returns the type of the model.
func (e *EmailOutboxItem) GetModelType() types.ModelType {
	return types.EmailOutboxItemModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (e *EmailOutboxItem) ResolveMetadata(key string) (*string, error) {
	return e.Metadata.resolveFieldValue(key)
}

// Validate validates the email outbox item fields.
func (e *EmailOutboxItem) Validate() error {
	if !e.EmailType.Valid() {
		return errors.New("invalid email type: %s", e.EmailType, errors.WithErrorCode(errors.EInvalid))
	}

	if e.Subject == "" {
		return errors.New("email subject is required", errors.WithErrorCode(errors.EInvalid))
	}

	if len(e.Subject) > MaxEmailSubjectLength {
		return errors.New("email subject cannot be greater than %d characters", MaxEmailSubjectLength, errors.WithErrorCode(errors.EInvalid))
	}

	// The payload is stored inline for small emails and in object storage otherwise; exactly one must be set.
	if (len(e.Payload) == 0) == (e.PayloadObjectStoreKey == nil) {
		return errors.New("exactly one of email payload or payload object store key is required", errors.WithErrorCode(errors.EInvalid))
	}

	if !e.Status.IsValid() {
		return errors.New("invalid email outbox item status: %s", e.Status, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
