package email

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewService(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	dbClient := &db.Client{}

	expect := &service{
		logger:   testLogger,
		dbClient: dbClient,
	}

	assert.Equal(t, expect, NewService(testLogger, dbClient))
}

func TestGetOutboxItemByID(t *testing.T) {
	sample := &models.EmailOutboxItem{Metadata: models.ResourceMetadata{ID: "outbox-1"}}

	type testCase struct {
		name            string
		result          *models.EmailOutboxItem
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			isAdmin:         false,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "outbox not found",
			withCaller:      true,
			isAdmin:         true,
			result:          nil,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin gets the outbox",
			withCaller: true,
			isAdmin:    true,
			result:     sample,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockOutboxItem := db.NewMockEmailOutboxItems(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockOutboxItem.On("GetOutboxItemByID", mock.Anything, "outbox-1").Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailOutboxItems: mockOutboxItem}}

			outbox, err := service.GetOutboxItemByID(ctx, "outbox-1")
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, outbox)
		})
	}
}

func TestGetOutboxItems(t *testing.T) {
	result := &db.EmailOutboxItemsResult{OutboxItems: []models.EmailOutboxItem{{Metadata: models.ResourceMetadata{ID: "outbox-1"}}}}

	type testCase struct {
		name            string
		result          *db.EmailOutboxItemsResult
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin lists outboxes",
			withCaller: true,
			isAdmin:    true,
			result:     result,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockOutboxItem := db.NewMockEmailOutboxItems(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockOutboxItem.On("GetOutboxItems", mock.Anything, mock.Anything).Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailOutboxItems: mockOutboxItem}}

			got, err := service.GetOutboxItems(ctx, &db.GetEmailOutboxItemsInput{})
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, got)
		})
	}
}

func TestGetRecipientByID(t *testing.T) {
	sample := &models.EmailRecipient{Metadata: models.ResourceMetadata{ID: "recipient-1"}}

	type testCase struct {
		name            string
		result          *models.EmailRecipient
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "recipient not found",
			withCaller:      true,
			isAdmin:         true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin gets the recipient",
			withCaller: true,
			isAdmin:    true,
			result:     sample,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRecipients := db.NewMockEmailRecipients(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockRecipients.On("GetRecipientByID", mock.Anything, "recipient-1").Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailRecipients: mockRecipients}}

			got, err := service.GetRecipientByID(ctx, "recipient-1")
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, got)
		})
	}
}

func TestGetRecipients(t *testing.T) {
	result := &db.EmailRecipientsResult{Recipients: []models.EmailRecipient{{Metadata: models.ResourceMetadata{ID: "recipient-1"}}}}

	type testCase struct {
		name            string
		result          *db.EmailRecipientsResult
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin lists recipients",
			withCaller: true,
			isAdmin:    true,
			result:     result,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRecipients := db.NewMockEmailRecipients(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockRecipients.On("GetRecipients", mock.Anything, mock.Anything).Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailRecipients: mockRecipients}}

			got, err := service.GetRecipients(ctx, &db.GetEmailRecipientsInput{})
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, got)
		})
	}
}

func TestGetRecipientStats(t *testing.T) {
	result := &db.EmailRecipientStatsResult{Total: 7, Delivered: 5}

	type testCase struct {
		name            string
		result          *db.EmailRecipientStatsResult
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin gets recipient stats",
			withCaller: true,
			isAdmin:    true,
			result:     result,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRecipients := db.NewMockEmailRecipients(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockRecipients.On("GetRecipientStats", mock.Anything, "outbox-1").Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailRecipients: mockRecipients}}

			got, err := service.GetRecipientStats(ctx, "outbox-1")
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, got)
		})
	}
}

func TestGetSuppressionByID(t *testing.T) {
	sample := &models.EmailSuppression{Metadata: models.ResourceMetadata{ID: "suppression-1"}}

	type testCase struct {
		name            string
		result          *models.EmailSuppression
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "suppression not found",
			withCaller:      true,
			isAdmin:         true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin gets the suppression",
			withCaller: true,
			isAdmin:    true,
			result:     sample,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSuppressions := db.NewMockEmailSuppressions(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockSuppressions.On("GetSuppressionByID", mock.Anything, "suppression-1").Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailSuppressions: mockSuppressions}}

			got, err := service.GetSuppressionByID(ctx, "suppression-1")
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, got)
		})
	}
}

func TestGetSuppressions(t *testing.T) {
	result := &db.EmailSuppressionsResult{Suppressions: []*models.EmailSuppression{{Metadata: models.ResourceMetadata{ID: "suppression-1"}}}}

	type testCase struct {
		name            string
		result          *db.EmailSuppressionsResult
		dbErr           error
		expectErrorCode errors.CodeType
		withCaller      bool
		isAdmin         bool
	}

	tests := []testCase{
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "non-admin caller is forbidden",
			withCaller:      true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "db error is internal",
			withCaller:      true,
			isAdmin:         true,
			dbErr:           errors.New("boom", errors.WithErrorCode(errors.EInternal)),
			expectErrorCode: errors.EInternal,
		},
		{
			name:       "admin lists suppressions",
			withCaller: true,
			isAdmin:    true,
			result:     result,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSuppressions := db.NewMockEmailSuppressions(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdminModeActivated", mock.Anything).Return(test.isAdmin)

				if test.isAdmin {
					mockSuppressions.On("GetSuppressions", mock.Anything, mock.Anything).Return(test.result, test.dbErr)
				}
			}

			service := &service{dbClient: &db.Client{EmailSuppressions: mockSuppressions}}

			got, err := service.GetSuppressions(ctx, &db.GetEmailSuppressionsInput{})
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.result, got)
		})
	}
}

func TestMarkRecipientClicked(t *testing.T) {
	recipientUser := &models.User{Metadata: models.ResourceMetadata{ID: "user-1"}, Email: "recipient@example.com"}
	recipient := &models.EmailRecipient{Metadata: models.ResourceMetadata{ID: "recipient-1"}, Address: "recipient@example.com"}
	clickedAt := time.Now()
	alreadyClicked := &models.EmailRecipient{Metadata: models.ResourceMetadata{ID: "recipient-1"}, Address: "recipient@example.com", ClickedAt: &clickedAt, OpenedAt: &clickedAt}

	tests := []struct {
		name            string
		caller          auth.Caller
		result          *models.EmailRecipient
		expectErrorCode errors.CodeType
		expectUpdate    bool
	}{
		{
			name:            "non-user caller is forbidden",
			caller:          auth.NewMockCaller(t),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "recipient not found",
			caller:          auth.NewUserCaller(recipientUser, nil, nil, nil, nil),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "caller is not the recipient",
			caller:          auth.NewUserCaller(&models.User{Email: "someone-else@example.com"}, nil, nil, nil, nil),
			result:          recipient,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:   "already clicked is a no-op",
			caller: auth.NewUserCaller(recipientUser, nil, nil, nil, nil),
			result: alreadyClicked,
		},
		{
			name:         "recipient marks their own email clicked",
			caller:       auth.NewUserCaller(recipientUser, nil, nil, nil, nil),
			result:       recipient,
			expectUpdate: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(t.Context(), test.caller)

			mockRecipients := db.NewMockEmailRecipients(t)

			if _, ok := test.caller.(*auth.UserCaller); ok {
				mockRecipients.On("GetRecipientByID", mock.Anything, "recipient-1").Return(test.result, nil)
			}

			if test.expectUpdate {
				mockRecipients.On("UpdateRecipient", mock.Anything, mock.Anything).Return(test.result, nil)
			}

			service := &service{dbClient: &db.Client{EmailRecipients: mockRecipients}}

			got, err := service.MarkRecipientClicked(ctx, "recipient-1")
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}
