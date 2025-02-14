package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestGetEvent(t *testing.T) {
	templateCtx := builder.NewTemplateContext("https://localhost", "custom footer")
	subject := "Test Email"
	body := "This is a test email"

	// Test cases
	tests := []struct {
		name             string
		userIDs          []string
		teamIDs          []string
		teamMembers      []models.TeamMember
		expectUsers      []models.User
		excludeUserIDs   []string
		expectRecipients []string
	}{
		{
			name:    "test email with users and teams with exclude",
			userIDs: []string{"user-1", "user-2"},
			teamIDs: []string{"team-1", "team-2"},
			teamMembers: []models.TeamMember{
				{UserID: "user-1"},
				{UserID: "user-2"},
				{UserID: "user-3"},
				{UserID: "user-4"},
			},
			expectUsers: []models.User{
				{Metadata: models.ResourceMetadata{ID: "user-1"}, Email: "user-1@test.com"},
				{Metadata: models.ResourceMetadata{ID: "user-2"}, Email: "user-2@test.com"},
				{Metadata: models.ResourceMetadata{ID: "user-3"}, Email: "user-3@test.com"},
			},
			excludeUserIDs: []string{"user-4"},
			expectRecipients: []string{
				"user-1@test.com",
				"user-2@test.com",
				"user-3@test.com",
			},
		},
		{
			name:    "test email with recipient list split into chunks",
			userIDs: []string{"user-1", "user-2"},
			teamIDs: []string{"team-1"},
			teamMembers: []models.TeamMember{
				{UserID: "user-1"},
				{UserID: "user-2"},
				{UserID: "user-3"},
				{UserID: "user-4"},
			},
			expectUsers: []models.User{
				{Metadata: models.ResourceMetadata{ID: "user-1"}, Email: "user-1@test.com"},
				{Metadata: models.ResourceMetadata{ID: "user-2"}, Email: "user-2@test.com"},
				{Metadata: models.ResourceMetadata{ID: "user-3"}, Email: "user-3@test.com"},
				{Metadata: models.ResourceMetadata{ID: "user-4"}, Email: "user-4@test.com"},
			},
			expectRecipients: []string{
				"user-1@test.com",
				"user-2@test.com",
				"user-3@test.com",
				"user-4@test.com",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockEmailProvider := email.NewMockProvider(t)
			mockTaskManager := asynctask.NewMockManager(t)

			mockUsers := db.NewMockUsers(t)
			mockTeamMembers := db.NewMockTeamMembers(t)

			mockBuilder := builder.NewMockEmailBuilder(t)

			mockBuilder.On("Build", templateCtx).Return(body, nil)

			mockTeamMembers.On("GetTeamMembers", mock.Anything, &db.GetTeamMembersInput{
				Filter: &db.TeamMemberFilter{
					TeamIDs: test.teamIDs,
				},
			}).Return(&db.TeamMembersResult{
				TeamMembers: test.teamMembers,
			}, nil)

			userIDs := []string{}
			for _, user := range test.expectUsers {
				userIDs = append(userIDs, user.Metadata.ID)
			}

			matcher := mock.MatchedBy(func(input *db.GetUsersInput) bool {
				return assert.ElementsMatch(t, input.Filter.UserIDs, userIDs, "userIDs do not match")
			})

			mockUsers.On("GetUsers", mock.Anything, matcher).Return(&db.UsersResult{
				Users: test.expectUsers,
			}, nil)

			for _, recipient := range test.expectRecipients {
				mockEmailProvider.On("SendMail", mock.Anything, []string{recipient}, subject, body).Return(nil).Once()
			}

			dbClient := &db.Client{
				Users:       mockUsers,
				TeamMembers: mockTeamMembers,
			}

			mockLogger, _ := logger.NewForTest()

			client := &client{
				emailProvider: mockEmailProvider,
				taskManager:   mockTaskManager,
				dbClient:      dbClient,
				logger:        mockLogger,
				templateCtx:   templateCtx,
			}

			if err := client.sendMail(ctx, &SendMailInput{
				UsersIDs:       test.userIDs,
				TeamsIDs:       test.teamIDs,
				ExcludeUserIDs: test.excludeUserIDs,
				Subject:        subject,
				Builder:        mockBuilder,
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
