package serviceaccount

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestNewSecretExpirationScheduler(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	mockEmailClient := email.NewMockClient(t)
	mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
	mockNotificationManager := namespace.NewMockNotificationManager(t)
	dbClient := &db.Client{}

	scheduler := NewSecretExpirationScheduler(
		dbClient,
		testLogger,
		mockEmailClient,
		mockMaintenanceMonitor,
		mockNotificationManager,
	)

	assert.NotNil(t, scheduler)
	assert.Equal(t, dbClient, scheduler.dbClient)
	assert.Equal(t, testLogger, scheduler.logger)
	assert.Equal(t, mockEmailClient, scheduler.emailClient)
	assert.Equal(t, mockMaintenanceMonitor, scheduler.maintenanceMonitor)
	assert.Equal(t, mockNotificationManager, scheduler.notificationManager)
}

func TestSecretExpirationScheduler_sendExpirationWarning(t *testing.T) {
	expiresAt := time.Now().Add(5 * 24 * time.Hour)

	sa := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:      "sa-1",
			Version: 1,
			TRN:     "trn:service_account:test-group/test-sa",
		},
		Name:                  "test-sa",
		GroupID:               "group-1",
		ClientSecretExpiresAt: &expiresAt,
	}

	type testCase struct {
		name          string
		userIDs       []string
		expectEmail   bool
		getUsersError error
		expectError   bool
	}

	testCases := []testCase{
		{
			name:        "sends email to owners",
			userIDs:     []string{"user-1", "user-2"},
			expectEmail: true,
		},
		{
			name:        "returns error when no owners",
			userIDs:     []string{},
			expectError: true,
		},
		{
			name:          "returns error on GetUsersToNotify failure",
			getUsersError: errors.New("some error"),
			expectError:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			testLogger, _ := logger.NewForTest()

			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockTransactions := db.NewMockTransactions(t)
			mockEmailClient := email.NewMockClient(t)
			mockNotificationManager := namespace.NewMockNotificationManager(t)

			mockNotificationManager.On("GetNamespaceMembersWithRole", mock.Anything, "test-group", models.OwnerRoleID.String()).
				Return(test.userIDs, nil).Maybe()
			mockNotificationManager.On("GetUsersToNotify", mock.Anything, mock.Anything).
				Return(test.userIDs, test.getUsersError).Maybe()

			if test.expectEmail {
				mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)
				mockServiceAccounts.On("UpdateServiceAccount", mock.Anything, mock.Anything).Return(sa, nil)
				mockEmailClient.On("SendMail", mock.Anything, mock.Anything).Return()
			}

			dbClient := &db.Client{
				ServiceAccounts: mockServiceAccounts,
				Transactions:    mockTransactions,
			}

			scheduler := &SecretExpirationScheduler{
				dbClient:            dbClient,
				logger:              testLogger,
				emailClient:         mockEmailClient,
				notificationManager: mockNotificationManager,
			}

			err := scheduler.sendExpirationWarning(ctx, sa)

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretExpirationScheduler_execute(t *testing.T) {
	expiresAt := time.Now().Add(5 * 24 * time.Hour)

	sa := models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID:      "sa-1",
			Version: 1,
			TRN:     "trn:service_account:test-group/test-sa",
		},
		Name:                  "test-sa",
		GroupID:               "group-1",
		ClientSecretExpiresAt: &expiresAt,
	}

	type testCase struct {
		name             string
		serviceAccounts  []models.ServiceAccount
		hasNextPage      bool
		expectNextCursor bool
		getAccountsError error
		expectError      bool
	}

	testCases := []testCase{
		{
			name:            "processes service accounts",
			serviceAccounts: []models.ServiceAccount{sa},
			hasNextPage:     false,
		},
		{
			name:             "returns cursor when more pages",
			serviceAccounts:  []models.ServiceAccount{sa},
			hasNextPage:      true,
			expectNextCursor: true,
		},
		{
			name:             "returns error on db failure",
			getAccountsError: errors.New("db error"),
			expectError:      true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			testLogger, _ := logger.NewForTest()

			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockTransactions := db.NewMockTransactions(t)
			mockEmailClient := email.NewMockClient(t)
			mockNotificationManager := namespace.NewMockNotificationManager(t)

			if test.getAccountsError != nil {
				mockServiceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).
					Return(nil, test.getAccountsError)
			} else {
				mockServiceAccounts.On("GetServiceAccounts", mock.Anything, mock.Anything).
					Return(&db.ServiceAccountsResult{
						ServiceAccounts: test.serviceAccounts,
						PageInfo: &pagination.PageInfo{
							HasNextPage: test.hasNextPage,
							Cursor: func(_ pagination.CursorPaginatable) (*string, error) {
								return ptr.String("next-cursor"), nil
							},
						},
					}, nil)

				if len(test.serviceAccounts) > 0 {
					mockNotificationManager.On("GetNamespaceMembersWithRole", mock.Anything, mock.Anything, mock.Anything).
						Return([]string{}, nil).Maybe()
					mockNotificationManager.On("GetUsersToNotify", mock.Anything, mock.Anything).
						Return([]string{}, nil).Maybe()
				}
			}

			dbClient := &db.Client{
				ServiceAccounts: mockServiceAccounts,
				Transactions:    mockTransactions,
			}

			scheduler := &SecretExpirationScheduler{
				dbClient:            dbClient,
				logger:              testLogger,
				emailClient:         mockEmailClient,
				notificationManager: mockNotificationManager,
			}

			cursor, err := scheduler.execute(ctx, nil)

			if test.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if test.expectNextCursor {
				require.NotNil(t, cursor)
			} else {
				require.Nil(t, cursor)
			}
		})
	}
}
