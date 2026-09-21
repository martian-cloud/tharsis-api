// Package email owns the durable email outbox: enqueuing, sending queued mail, and reclaiming finished ephemeral outboxes.
package email

//go:generate go tool mockery --name Enqueuer --inpackage --case underscore

import (
	"context"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// maxInlineRowSize bounds the combined inline size (payload + recipient ID arrays) before the payload is offloaded
// to object storage to keep the row under Postgres's TOAST threshold and pg_notify's 8000-byte cap.
const maxInlineRowSize = 1024

// uuidStringSize is the byte length of a recipient ID as stored inline: the IDs are JSON-encoded
// canonical UUID strings (36 chars), not the 16-byte binary form, so the footprint is 36 bytes each.
const uuidStringSize = 36

// EnqueueEmailInput is the input for enqueuing an email; recipients are the union of resolved UserIDs and TeamIDs members, minus suppressed addresses.
type EnqueueEmailInput struct {
	// Builder renders the email body; its Type() is the outbox email_type.
	Builder builder.EmailBuilder
	// Subject is the email subject line.
	Subject string
	// UserIDs are individual recipient user IDs.
	UserIDs []string
	// TeamIDs expand to their members.
	TeamIDs []string
	// SendToAllUsers fans out to every active user; when set, UserIDs/TeamIDs are ignored.
	SendToAllUsers bool
	// Retain keeps the outbox row after delivery instead of reclaiming it;
	// defaults to false, and the caller that sets it enforces who may.
	Retain bool
	// SendAt delays delivery until the given time; nil means send as soon as possible.
	SendAt *time.Time
}

// Enqueuer adds emails to the durable outbox; it is the only supported way to queue an email for delivery.
type Enqueuer interface {
	// EnqueueEmail resolves recipients, drops suppressed addresses, and writes the outbox and its queue rows.
	EnqueueEmail(ctx context.Context, input *EnqueueEmailInput) error
}

type enqueuer struct {
	dbClient *db.Client
	store    Store
	logger   logger.Logger
}

// NewEnqueuer creates an email outbox Enqueuer.
func NewEnqueuer(dbClient *db.Client, store Store, logger logger.Logger) Enqueuer {
	return &enqueuer{
		dbClient: dbClient,
		store:    store,
		logger:   logger,
	}
}

// EnqueueEmail stores the email payload and creates the outbox item synchronously on the caller's context
// (joining the caller's transaction when one is open) with a pending recipient status.
func (e *enqueuer) EnqueueEmail(ctx context.Context, input *EnqueueEmailInput) error {
	ctx, span := tracer.Start(ctx, "svc.EnqueueEmail")
	defer span.End()

	if input.Builder == nil {
		return errors.New("builder is required", errors.WithSpan(span))
	}

	payload, err := msgpack.Marshal(input.Builder)
	if err != nil {
		return errors.Wrap(err, "failed to marshal email payload", errors.WithSpan(span))
	}

	// Small payloads live inline in the outbox row; larger ones go to object storage to keep rows compact.
	var (
		inlinePayload []byte
		payloadKey    *string
		retainFunc    db.RetainObjectRefFunc
	)

	// Recipient ID arrays are stored inline and can't be offloaded, so count them toward the row budget.
	recipientIDsSize := (len(input.UserIDs) + len(input.TeamIDs)) * uuidStringSize

	if len(payload)+recipientIDsSize <= maxInlineRowSize {
		inlinePayload = payload
	} else {
		var key string
		retainFunc, key, err = e.store.UploadPayload(ctx, payload)
		if err != nil {
			return errors.Wrap(err, "failed to upload email payload", errors.WithSpan(span))
		}

		payloadKey = &key
	}

	txContext, err := e.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := e.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			e.logger.WithContextFields(ctx).Errorf("failed to roll back enqueueEmail transaction: %v", txErr)
		}
	}()

	outboxItem := &models.EmailOutboxItem{
		EmailType:             input.Builder.Type(),
		Subject:               input.Subject,
		Payload:               inlinePayload,
		PayloadObjectStoreKey: payloadKey,
		Ephemeral:             !input.Retain,
		Status:                models.EmailOutboxItemStatusPreparing,
		SendToAllUsers:        input.SendToAllUsers,
		RecipientUserIDs:      input.UserIDs,
		RecipientTeamIDs:      input.TeamIDs,
		SendAt:                input.SendAt,
	}

	if err = outboxItem.Validate(); err != nil {
		return errors.Wrap(err, "invalid email outbox item", errors.WithSpan(span))
	}

	outbox, err := e.dbClient.EmailOutboxItems.CreateOutboxItem(txContext, outboxItem)
	if err != nil {
		return errors.Wrap(err, "failed to create email outbox", errors.WithSpan(span))
	}

	// Only object-store payloads have a ref to link to the owning outbox row.
	if retainFunc != nil {
		if err = retainFunc(txContext, outbox.Metadata.ID); err != nil {
			return errors.Wrap(err, "failed to link email payload object", errors.WithSpan(span))
		}
	}

	if err = e.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit transaction", errors.WithSpan(span))
	}

	e.logger.WithContextFields(ctx).Infow("enqueued email.",
		"email_outbox_item_id", outbox.Metadata.ID,
		"email_type", outbox.EmailType,
		"ephemeral", outbox.Ephemeral,
		"send_to_all_users", outbox.SendToAllUsers,
	)

	return nil
}
