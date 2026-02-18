// Package serviceaccount provides service account related services.
package serviceaccount

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// Check every 6-12 hours
	maxCheckInterval = 12 * time.Hour
	minCheckInterval = 6 * time.Hour
	// Warn when secret expires within this many days
	warningThresholdDays = 7
	// Batch size for processing
	batchSize = 100
)

// SecretExpirationScheduler sends email warnings for expiring client secrets.
type SecretExpirationScheduler struct {
	dbClient            *db.Client
	logger              logger.Logger
	emailClient         email.Client
	maintenanceMonitor  maintenance.Monitor
	notificationManager namespace.NotificationManager
}

// NewSecretExpirationScheduler creates a new scheduler.
func NewSecretExpirationScheduler(
	dbClient *db.Client,
	logger logger.Logger,
	emailClient email.Client,
	maintenanceMonitor maintenance.Monitor,
	notificationManager namespace.NotificationManager,
) *SecretExpirationScheduler {
	return &SecretExpirationScheduler{
		dbClient:            dbClient,
		logger:              logger,
		emailClient:         emailClient,
		maintenanceMonitor:  maintenanceMonitor,
		notificationManager: notificationManager,
	}
}

// Start starts the scheduler.
func (s *SecretExpirationScheduler) Start(ctx context.Context) {
	s.logger.Info("service account secret expiration scheduler started")

	go func() {
		var cursor *string

		for {
			// Continue immediately if there are more pages to process
			if cursor == nil {
				sleep := minCheckInterval + time.Duration(rand.Int64N(int64(maxCheckInterval-minCheckInterval)))
				select {
				case <-time.After(sleep):
				case <-ctx.Done():
					s.logger.Info("service account secret expiration scheduler stopped")
					return
				}
			}

			inMaintenance, err := s.maintenanceMonitor.InMaintenanceMode(ctx)
			if err != nil || inMaintenance {
				cursor = nil
				continue
			}

			nextCursor, err := s.execute(ctx, cursor)
			if err != nil {
				s.logger.Errorf("failed to process service account secret expiration scheduler: %v", err)
				cursor = nil
				continue
			}

			cursor = nextCursor
		}
	}()
}

func (s *SecretExpirationScheduler) execute(ctx context.Context, cursor *string) (*string, error) {
	expirationThreshold := time.Now().Add(time.Duration(warningThresholdDays) * 24 * time.Hour)

	result, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(batchSize),
			After: cursor,
		},
		Filter: &db.ServiceAccountFilter{
			PendingExpirationNotification: &expirationThreshold,
		},
	})
	if err != nil {
		return nil, err
	}

	for i := range result.ServiceAccounts {
		if err := s.sendExpirationWarning(ctx, &result.ServiceAccounts[i]); err != nil {
			s.logger.Errorf("failed to send expiration warning for service account %s: %v", result.ServiceAccounts[i].Name, err)
		}
	}

	if result.PageInfo.HasNextPage {
		nextCursor, cErr := result.PageInfo.Cursor(&result.ServiceAccounts[len(result.ServiceAccounts)-1])
		if cErr != nil {
			return nil, cErr
		}

		return nextCursor, nil
	}

	return nil, nil
}

func (s *SecretExpirationScheduler) sendExpirationWarning(ctx context.Context, sa *models.ServiceAccount) error {
	ownerUserIDs, err := s.notificationManager.GetNamespaceMembersWithRole(ctx, sa.GetGroupPath(), models.OwnerRoleID.String())
	if err != nil {
		return errors.Wrap(err, "failed to get namespace owners")
	}

	userIDs, err := s.notificationManager.GetUsersToNotify(ctx, &namespace.GetUsersToNotifyInput{
		NamespacePath:      sa.GetGroupPath(),
		ParticipantUserIDs: ownerUserIDs,
		CustomEventCheck: func(events *models.NotificationPreferenceCustomEvents) bool {
			return events.ServiceAccountSecretExpiration
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get users to notify")
	}

	if len(userIDs) == 0 {
		// A group should generally have at least one owner with notifications enabled
		return errors.New("no users to notify for service account %s", sa.Name)
	}

	txCtx, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	defer func() {
		if err := s.dbClient.Transactions.RollbackTx(txCtx); err != nil {
			s.logger.Errorf("failed to rollback transaction for service account secret expiration scheduler: %v", err)
		}
	}()

	// Mark as notified first to prevent duplicate emails
	sa.SecretExpirationEmailSentAt = ptr.Time(time.Now())
	if _, err := s.dbClient.ServiceAccounts.UpdateServiceAccount(txCtx, sa); err != nil {
		return errors.Wrap(err, "failed to update service account")
	}

	s.emailClient.SendMail(txCtx, &email.SendMailInput{
		UsersIDs: userIDs,
		Subject:  "Service Account Client Secret Expiring Soon",
		Builder: &builder.ServiceAccountSecretExpirationEmail{
			ServiceAccountName: sa.Name,
			ServiceAccountID:   sa.GetGlobalID(),
			GroupPath:          sa.GetGroupPath(),
			ExpiresAt:          *sa.ClientSecretExpiresAt, // ClientSecretExpiresAt is guaranteed non-nil by the DB filter.
		},
	})

	if err := s.dbClient.Transactions.CommitTx(txCtx); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}
