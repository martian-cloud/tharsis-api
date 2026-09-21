// Package email contains the service for querying the email delivery subsystem: outboxes, recipients, and suppressions.
package email

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// recipientClicks counts distinct email recipients who followed a tracked link back into the app.
var recipientClicks = metric.NewCounter("email_recipient_clicks_total", "Number of email recipients who clicked a tracked link back into the app.")

// Service is the interface for the email delivery subsystem.
type Service interface {
	GetOutboxItemByID(ctx context.Context, id string) (*models.EmailOutboxItem, error)
	GetOutboxItemByTRN(ctx context.Context, trn string) (*models.EmailOutboxItem, error)
	GetOutboxItemsByIDs(ctx context.Context, ids []string) ([]models.EmailOutboxItem, error)
	GetOutboxItems(ctx context.Context, input *db.GetEmailOutboxItemsInput) (*db.EmailOutboxItemsResult, error)
	GetRecipientByID(ctx context.Context, id string) (*models.EmailRecipient, error)
	GetRecipientByTRN(ctx context.Context, trn string) (*models.EmailRecipient, error)
	GetRecipients(ctx context.Context, input *db.GetEmailRecipientsInput) (*db.EmailRecipientsResult, error)
	GetRecipientStats(ctx context.Context, emailOutboxItemID string) (*db.EmailRecipientStatsResult, error)
	GetSuppressionByID(ctx context.Context, id string) (*models.EmailSuppression, error)
	GetSuppressionByTRN(ctx context.Context, trn string) (*models.EmailSuppression, error)
	GetSuppressions(ctx context.Context, input *db.GetEmailSuppressionsInput) (*db.EmailSuppressionsResult, error)
	DeleteSuppression(ctx context.Context, input *DeleteSuppressionInput) error
	MarkRecipientClicked(ctx context.Context, id string) (*models.EmailRecipient, error)
}

// DeleteSuppressionInput is the input for removing an address from the suppression list.
type DeleteSuppressionInput struct {
	MetadataVersion *int
	ID              string
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates a new email service.
func NewService(logger logger.Logger, dbClient *db.Client) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetOutboxItemByID(ctx context.Context, id string) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "svc.GetOutboxItemByID")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	outbox, err := s.dbClient.EmailOutboxItems.GetOutboxItemByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email outbox", errors.WithSpan(span))
	}

	if outbox == nil {
		return nil, errors.New("email outbox with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return outbox, nil
}

func (s *service) GetOutboxItemByTRN(ctx context.Context, trn string) (*models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "svc.GetOutboxItemByTRN")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	outbox, err := s.dbClient.EmailOutboxItems.GetOutboxItemByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email outbox", errors.WithSpan(span))
	}

	if outbox == nil {
		return nil, errors.New("email outbox with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return outbox, nil
}

func (s *service) GetOutboxItemsByIDs(ctx context.Context, ids []string) ([]models.EmailOutboxItem, error) {
	ctx, span := tracer.Start(ctx, "svc.GetOutboxItemsByIDs")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	result, err := s.dbClient.EmailOutboxItems.GetOutboxItems(ctx, &db.GetEmailOutboxItemsInput{
		Filter: &db.EmailOutboxItemFilter{
			OutboxItemIDs: ids,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email outbox items by IDs", errors.WithSpan(span))
	}

	return result.OutboxItems, nil
}

func (s *service) GetOutboxItems(ctx context.Context, input *db.GetEmailOutboxItemsInput) (*db.EmailOutboxItemsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetOutboxItems")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	result, err := s.dbClient.EmailOutboxItems.GetOutboxItems(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email outboxes", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) GetRecipientByID(ctx context.Context, id string) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRecipientByID")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	recipient, err := s.dbClient.EmailRecipients.GetRecipientByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email recipient", errors.WithSpan(span))
	}

	if recipient == nil {
		return nil, errors.New("email recipient with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return recipient, nil
}

// MarkRecipientClicked records that a recipient clicked a link back into the app; only the recipient themselves may confirm their own row.
func (s *service) MarkRecipientClicked(ctx context.Context, id string) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "svc.MarkRecipientClicked")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can mark an email recipient clicked", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	recipient, err := s.dbClient.EmailRecipients.GetRecipientByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email recipient", errors.WithSpan(span))
	}

	if recipient == nil {
		return nil, errors.New("email recipient with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if !strings.EqualFold(recipient.Address, userCaller.User.Email) {
		return nil, errors.New("caller is not the recipient", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	if !recipient.MarkClicked() {
		return recipient, nil
	}

	updated, err := s.dbClient.EmailRecipients.UpdateRecipient(ctx, recipient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update email recipient", errors.WithSpan(span))
	}

	recipientClicks.Inc()

	return updated, nil
}

func (s *service) GetRecipientByTRN(ctx context.Context, trn string) (*models.EmailRecipient, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRecipientByTRN")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	recipient, err := s.dbClient.EmailRecipients.GetRecipientByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email recipient", errors.WithSpan(span))
	}

	if recipient == nil {
		return nil, errors.New("email recipient with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return recipient, nil
}

func (s *service) GetRecipients(ctx context.Context, input *db.GetEmailRecipientsInput) (*db.EmailRecipientsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRecipients")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	result, err := s.dbClient.EmailRecipients.GetRecipients(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email recipients", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) GetRecipientStats(ctx context.Context, emailOutboxItemID string) (*db.EmailRecipientStatsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetRecipientStats")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	stats, err := s.dbClient.EmailRecipients.GetRecipientStats(ctx, emailOutboxItemID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email recipient stats", errors.WithSpan(span))
	}

	return stats, nil
}

func (s *service) GetSuppressionByID(ctx context.Context, id string) (*models.EmailSuppression, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSuppressionByID")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	suppression, err := s.dbClient.EmailSuppressions.GetSuppressionByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email suppression", errors.WithSpan(span))
	}

	if suppression == nil {
		return nil, errors.New("email suppression with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return suppression, nil
}

func (s *service) GetSuppressionByTRN(ctx context.Context, trn string) (*models.EmailSuppression, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSuppressionByTRN")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	suppression, err := s.dbClient.EmailSuppressions.GetSuppressionByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email suppression", errors.WithSpan(span))
	}

	if suppression == nil {
		return nil, errors.New("email suppression with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return suppression, nil
}

func (s *service) GetSuppressions(ctx context.Context, input *db.GetEmailSuppressionsInput) (*db.EmailSuppressionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSuppressions")
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	result, err := s.dbClient.EmailSuppressions.GetSuppressions(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get email suppressions", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) DeleteSuppression(ctx context.Context, input *DeleteSuppressionInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteSuppression")
	span.SetAttributes(attribute.String("suppressionID", input.ID))
	defer span.End()

	if err := s.requireAdmin(ctx); err != nil {
		return errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	suppression, err := s.dbClient.EmailSuppressions.GetSuppressionByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get email suppression", errors.WithSpan(span))
	}

	if suppression == nil {
		return errors.New("email suppression with id %s not found", input.ID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if input.MetadataVersion != nil {
		suppression.Metadata.Version = *input.MetadataVersion
	}

	if err = s.dbClient.EmailSuppressions.DeleteSuppression(ctx, suppression); err != nil {
		return errors.Wrap(err, "failed to delete email suppression", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("deleted email suppression.",
		"suppression_id", suppression.Metadata.ID,
		"address", suppression.Address,
	)

	return nil
}

// requireAdmin authorizes the caller and confirms admin mode is activated; email data is global and may contain PII.
func (s *service) requireAdmin(ctx context.Context) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return errors.New("only admins with admin mode activated can query email delivery data", errors.WithErrorCode(errors.EForbidden))
	}

	return nil
}
