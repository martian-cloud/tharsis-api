package namespace

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetUsersToNotify(t *testing.T) {

	type testCase struct {
		name               string
		input              *GetUsersToNotifyInput
		memberships        []models.NamespaceMembership
		preferencesPerUser map[string]*NotificationPreferenceSetting
		expectErrorCode    errors.CodeType
		expectResponse     []string
	}

	testCases := []testCase{
		{
			name: "successfully get users to notify",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"user1", "user2"},
				CustomEventCheck: func(customEvents *models.NotificationPreferenceCustomEvents) bool {
					return customEvents.FailedRun
				},
			},
			memberships: []models.NamespaceMembership{
				{
					UserID: ptr.String("user1"),
				},
				{
					UserID: ptr.String("user3"),
				},
				{
					UserID: ptr.String("user4"),
				},
				{
					UserID: ptr.String("user5"),
				},
				{
					UserID: ptr.String("user6"),
				},
				{
					ServiceAccountID: ptr.String("sa1"),
				},
			},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"user1": {
					Inherited:     true,
					NamespacePath: ptr.String("test/namespace"),
					Scope:         models.NotificationPreferenceScopeParticipate,
				},
				"user2": {
					Scope: models.NotificationPreferenceScopeAll,
				},
				"user3": {
					Scope: models.NotificationPreferenceScopeNone,
				},
				"user4": {
					Scope: models.NotificationPreferenceScopeParticipate,
				},
				"user5": {
					Scope: models.NotificationPreferenceScopeCustom,
					CustomEvents: &models.NotificationPreferenceCustomEvents{
						FailedRun: true,
					},
				},
				"user6": {
					Scope: models.NotificationPreferenceScopeCustom,
					CustomEvents: &models.NotificationPreferenceCustomEvents{
						FailedRun: false,
					},
				},
			},
			expectResponse: []string{"user1", "user2", "user5"},
		},
		{
			name: "no users to notify",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
			},
			memberships: []models.NamespaceMembership{},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{},
			expectResponse: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)
			mockInheritedSettingResolver := NewMockInheritedSettingResolver(t)

			mockNamespaceMemberships.On("GetNamespaceMemberships", ctx, &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					NamespacePaths: utils.ExpandPath(test.input.NamespacePath),
				},
			}).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.memberships,
			}, nil)

			mockInheritedSettingResolver.On("GetNotificationPreferences", ctx, mock.Anything, &test.input.NamespacePath).Return(test.preferencesPerUser, nil)

			dbClient := &db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
			}

			notificationManager := NewNotificationManager(dbClient, mockInheritedSettingResolver)

			actualResponse, err := notificationManager.GetUsersToNotify(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.ElementsMatch(t, test.expectResponse, actualResponse)
		})
	}
}
