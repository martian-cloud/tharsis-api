package activityevent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type mockDBClient struct {
	*db.Client
	MockTransactions   *db.MockTransactions
	MockActivityEvents *db.MockActivityEvents
}

func buildDBClientWithMocks(t *testing.T) *mockDBClient {
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)

	mockActivityEvents := db.MockActivityEvents{}
	mockActivityEvents.Test(t)

	return &mockDBClient{
		Client: &db.Client{
			Transactions:   &mockTransactions,
			ActivityEvents: &mockActivityEvents,
		},
		MockTransactions:   &mockTransactions,
		MockActivityEvents: &mockActivityEvents,
	}
}

func TestGetActivityEvents(t *testing.T) {

	type testCase struct {
		name      string
		caller    string
		adminMode bool
	}

	// Test cases
	testCases := []testCase{
		{
			name:      "admin mode activated means no root namespace membership filter is applied",
			caller:    "user",
			adminMode: true,
		},
		{
			name:      "non-admin caller restricts results to root namespace memberships",
			caller:    "serviceAccount",
			adminMode: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			dbClient := buildDBClientWithMocks(t)

			mockAuthorizer := auth.MockAuthorizer{}
			mockAuthorizer.Test(t)

			mockAuthorizer.On("GetRootNamespaces", mock.Anything).Return([]models.MembershipNamespace{}, nil)

			var testCaller auth.Caller
			switch test.caller {
			case "user":
				adminModeExpiration := time.Now().UTC().Add(time.Hour)

				mockUsers := db.MockUsers{}
				mockUsers.Test(t)
				mockUsers.On("GetUserByID", mock.Anything, "123").Return(&models.User{
					Metadata:            models.ResourceMetadata{ID: "123"},
					Admin:               test.adminMode,
					AdminModeExpiration: &adminModeExpiration,
					Username:            "user1",
				}, nil)
				dbClient.Client.Users = &mockUsers

				testCaller = auth.NewUserCaller(
					&models.User{
						Metadata: models.ResourceMetadata{
							ID: "123",
						},
						Admin:    test.adminMode,
						Username: "user1",
					},
					&mockAuthorizer,
					dbClient.Client,
					nil,
					nil,
				)
			case "serviceAccount":
				testCaller = auth.NewServiceAccountCaller(
					"sa1",
					"groupA/sa1",
					&mockAuthorizer,
					nil,
					nil,
				)
			}

			dbClient.MockActivityEvents.On("GetActivityEvents", mock.Anything, mock.Anything).
				Return(func(_ context.Context, _ *db.GetActivityEventsInput) *db.ActivityEventsResult {
					return &db.ActivityEventsResult{
						ActivityEvents: []models.ActivityEvent{},
					}
				}, nil)

			logger, _ := logger.NewForTest()
			service := NewService(dbClient.Client, logger)

			// Call the service function.
			actualOutput, actualError := service.GetActivityEvents(auth.WithCaller(ctx, testCaller), &GetActivityEventsInput{})
			if actualError != nil {
				t.Fatal(actualError)
			}

			assert.Equal(t, &db.ActivityEventsResult{
				ActivityEvents: []models.ActivityEvent{},
			}, actualOutput)

			dbClient.MockActivityEvents.AssertCalled(t, "GetActivityEvents", mock.Anything, mock.MatchedBy(func(input *db.GetActivityEventsInput) bool {
				if test.adminMode {
					// Admin sees all: a nil slice means no membership filter.
					return input.Filter.RootNamespaceMemberships == nil
				}
				// Non-admin: a non-nil (possibly empty) slice restricts results.
				return input.Filter.RootNamespaceMemberships != nil
			}))
		})
	}
}
