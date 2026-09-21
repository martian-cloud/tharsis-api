// Package announcement contains the service for managing announcements
package announcement

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

// defaultAnnouncementEmailSubject is used when the caller does not supply an email subject.
const defaultAnnouncementEmailSubject = "New Tharsis Announcement"

// GetAnnouncementsInput is the input for getting announcements
type GetAnnouncementsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.AnnouncementSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Active filters for announcements that are currently active
	Active *bool
}

// CreateAnnouncementInput is the input for creating an announcement
type CreateAnnouncementInput struct {
	Message      string
	StartTime    *time.Time
	EndTime      *time.Time
	Type         models.AnnouncementType
	Dismissible  bool
	SendEmail    bool
	EmailSubject *string
}

// UpdateAnnouncementInput is the input for updating an announcement
type UpdateAnnouncementInput struct {
	ID              string
	Message         *string
	StartTime       *time.Time
	EndTime         *time.Time
	Type            *models.AnnouncementType
	Dismissible     *bool
	MetadataVersion *int
}

// DeleteAnnouncementInput is the input for deleting an announcement
type DeleteAnnouncementInput struct {
	ID              string
	MetadataVersion *int
}

// Service is the interface for the announcement service
type Service interface {
	GetAnnouncementByID(ctx context.Context, id string) (*models.Announcement, error)
	GetAnnouncementByTRN(ctx context.Context, trn string) (*models.Announcement, error)
	GetAnnouncements(ctx context.Context, input *GetAnnouncementsInput) (*db.AnnouncementsResult, error)
	CreateAnnouncement(ctx context.Context, input *CreateAnnouncementInput) (*models.Announcement, error)
	UpdateAnnouncement(ctx context.Context, input *UpdateAnnouncementInput) (*models.Announcement, error)
	DeleteAnnouncement(ctx context.Context, input *DeleteAnnouncementInput) error
}

type service struct {
	logger        logger.Logger
	dbClient      *db.Client
	emailEnqueuer email.Enqueuer
}

// NewService creates a new announcement service
func NewService(logger logger.Logger, dbClient *db.Client, emailEnqueuer email.Enqueuer) Service {
	return &service{
		logger:        logger,
		dbClient:      dbClient,
		emailEnqueuer: emailEnqueuer,
	}
}

func (s *service) GetAnnouncementByID(ctx context.Context, id string) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "svc.GetAnnouncementByID")
	span.SetAttributes(attribute.String("announcementID", id))
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	announcement, err := s.dbClient.Announcements.GetAnnouncementByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get announcement", errors.WithSpan(span))
	}

	if announcement == nil {
		return nil, errors.New("announcement not found", errors.WithErrorCode(errors.ENotFound))
	}

	return announcement, nil
}

func (s *service) GetAnnouncementByTRN(ctx context.Context, trn string) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "svc.GetAnnouncementByTRN")
	span.SetAttributes(attribute.String("announcementTRN", trn))
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	announcement, err := s.dbClient.Announcements.GetAnnouncementByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get announcement", errors.WithSpan(span))
	}

	if announcement == nil {
		return nil, errors.New("announcement not found", errors.WithErrorCode(errors.ENotFound))
	}

	return announcement, nil
}

func (s *service) GetAnnouncements(ctx context.Context, input *GetAnnouncementsInput) (*db.AnnouncementsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetAnnouncements")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	dbInput := &db.GetAnnouncementsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
	}

	if input.Active != nil {
		dbInput.Filter = &db.AnnouncementFilter{
			Active: input.Active,
		}
	}

	result, err := s.dbClient.Announcements.GetAnnouncements(ctx, dbInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get announcements", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) CreateAnnouncement(ctx context.Context, input *CreateAnnouncementInput) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateAnnouncement")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return nil, errors.New("only admins with admin mode activated can create announcements", errors.WithErrorCode(errors.EForbidden))
	}

	if input.EmailSubject != nil && len(*input.EmailSubject) > models.MaxEmailSubjectLength {
		return nil, errors.New("email subject cannot be greater than %d characters", models.MaxEmailSubjectLength, errors.WithErrorCode(errors.EInvalid))
	}

	// Default start time to current time if not provided
	var startTime time.Time
	if input.StartTime != nil {
		startTime = *input.StartTime
	} else {
		startTime = time.Now().UTC()
	}

	toCreate := &models.Announcement{
		Message:     input.Message,
		StartTime:   startTime,
		EndTime:     input.EndTime,
		Type:        input.Type,
		Dismissible: input.Dismissible,
		CreatedBy:   caller.GetSubject(),
	}

	if err = toCreate.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate announcement model", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to roll back CreateAnnouncement transaction: %v", txErr)
		}
	}()

	created, err := s.dbClient.Announcements.CreateAnnouncement(txContext, toCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create announcement", errors.WithSpan(span))
	}

	if input.SendEmail {
		emailSubject := defaultAnnouncementEmailSubject
		if input.EmailSubject != nil && *input.EmailSubject != "" {
			emailSubject = *input.EmailSubject
		}

		if err = s.emailEnqueuer.EnqueueEmail(txContext, &email.EnqueueEmailInput{
			Builder: &builder.AnnouncementEmail{
				Message:  created.Message,
				Severity: string(created.Type),
			},
			Subject:        emailSubject,
			SendToAllUsers: true,
			Retain:         true, // Announcement emails are user-facing broadcasts kept for inspection
			SendAt:         &created.StartTime,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to enqueue announcement email", errors.WithSpan(span))
		}
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("created announcement.",
		"announcement_id", created.Metadata.ID,
		"announcement_type", created.Type,
		"sends_email", input.SendEmail,
	)

	return created, nil
}

func (s *service) UpdateAnnouncement(ctx context.Context, input *UpdateAnnouncementInput) (*models.Announcement, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateAnnouncement")
	span.SetAttributes(attribute.String("announcementID", input.ID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return nil, errors.New("only admins with admin mode activated can update announcements", errors.WithErrorCode(errors.EForbidden))
	}

	existing, err := s.dbClient.Announcements.GetAnnouncementByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get existing announcement", errors.WithSpan(span))
	}

	if existing == nil {
		return nil, errors.New("announcement not found", errors.WithErrorCode(errors.ENotFound))
	}

	existing.EndTime = input.EndTime

	if input.Message != nil {
		existing.Message = *input.Message
	}

	if input.StartTime != nil {
		existing.StartTime = *input.StartTime
	}

	if input.Type != nil {
		existing.Type = *input.Type
	}

	if input.Dismissible != nil {
		existing.Dismissible = *input.Dismissible
	}

	if input.MetadataVersion != nil {
		existing.Metadata.Version = *input.MetadataVersion
	}

	if err = existing.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate updated announcement model", errors.WithSpan(span))
	}

	updated, err := s.dbClient.Announcements.UpdateAnnouncement(ctx, existing)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update announcement", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated announcement.",
		"announcement_id", updated.Metadata.ID,
	)

	return updated, nil
}

func (s *service) DeleteAnnouncement(ctx context.Context, input *DeleteAnnouncementInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteAnnouncement")
	span.SetAttributes(attribute.String("announcementID", input.ID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return errors.New("only admins with admin mode activated can delete announcements", errors.WithErrorCode(errors.EForbidden))
	}

	announcement, err := s.dbClient.Announcements.GetAnnouncementByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get announcement", errors.WithSpan(span))
	}

	if announcement == nil {
		return errors.New("announcement not found", errors.WithErrorCode(errors.ENotFound))
	}

	if input.MetadataVersion != nil {
		announcement.Metadata.Version = *input.MetadataVersion
	}

	if err = s.dbClient.Announcements.DeleteAnnouncement(ctx, announcement); err != nil {
		return errors.Wrap(err, "failed to delete announcement", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted announcement.",
		"announcement_id", announcement.Metadata.ID,
	)

	return nil
}
