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
			name: "participant user IDs treats matching members as participants",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"owner1", "owner2"},
			},
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("owner1"), RoleID: "owner"},
				{UserID: ptr.String("owner2"), RoleID: "owner"},
				{UserID: ptr.String("viewer1"), RoleID: "viewer"},
			},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"owner1":  {Scope: models.NotificationPreferenceScopeParticipate},
				"owner2":  {Scope: models.NotificationPreferenceScopeAll},
				"viewer1": {Scope: models.NotificationPreferenceScopeParticipate},
			},
			expectResponse: []string{"owner1", "owner2"}, // viewer1 excluded - PARTICIPATE but not participant
		},
		{
			name: "viewer with ALL scope gets notified even with participant filter",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"owner1"},
			},
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("owner1"), RoleID: "owner"},
				{UserID: ptr.String("viewer1"), RoleID: "viewer"},
			},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"owner1":  {Scope: models.NotificationPreferenceScopeParticipate},
				"viewer1": {Scope: models.NotificationPreferenceScopeAll},
			},
			expectResponse: []string{"owner1", "viewer1"},
		},
		{
			name: "multiple participant user IDs",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"creator1", "owner1"},
			},
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("owner1"), RoleID: "owner"},
				{UserID: ptr.String("creator1"), RoleID: "viewer"},
			},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"owner1":   {Scope: models.NotificationPreferenceScopeParticipate},
				"creator1": {Scope: models.NotificationPreferenceScopeParticipate},
			},
			expectResponse: []string{"owner1", "creator1"},
		},
		{
			name: "NONE scope excludes user regardless of participant status",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"owner1"},
			},
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("owner1"), RoleID: "owner"},
			},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"owner1": {Scope: models.NotificationPreferenceScopeNone},
			},
			expectResponse: []string{},
		},
		{
			name: "custom event check with nil custom events",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"user1"},
				CustomEventCheck: func(customEvents *models.NotificationPreferenceCustomEvents) bool {
					return customEvents != nil && customEvents.FailedRun
				},
			},
			memberships: []models.NamespaceMembership{},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"user1": {Scope: models.NotificationPreferenceScopeCustom, CustomEvents: nil},
			},
			expectResponse: []string{},
		},
		{
			name: "no custom event check function provided",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"user1"},
			},
			memberships: []models.NamespaceMembership{},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"user1": {Scope: models.NotificationPreferenceScopeCustom, CustomEvents: &models.NotificationPreferenceCustomEvents{FailedRun: true}},
			},
			expectResponse: []string{},
		},
		{
			name: "deduplicates users from participant IDs and memberships",
			input: &GetUsersToNotifyInput{
				NamespacePath:      "test/namespace",
				ParticipantUserIDs: []string{"user1"},
			},
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("user1")},
			},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{
				"user1": {Scope: models.NotificationPreferenceScopeAll},
			},
			expectResponse: []string{"user1"},
		},
		{
			name: "no users to notify",
			input: &GetUsersToNotifyInput{
				NamespacePath: "test/namespace",
			},
			memberships:        []models.NamespaceMembership{},
			preferencesPerUser: map[string]*NotificationPreferenceSetting{},
			expectResponse:     []string{},
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

			mockInheritedSettingResolver.On("GetNotificationPreferences", ctx, mock.Anything, &test.input.NamespacePath).Return(test.preferencesPerUser, nil).Maybe()

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

func TestGetNamespaceMembersWithRole(t *testing.T) {
	type testCase struct {
		name          string
		namespacePath string
		roleID        string
		memberships   []models.NamespaceMembership
		expectError   bool
		expectUserIDs []string
	}

	testCases := []testCase{
		{
			name:          "returns users with matching role",
			namespacePath: "test/namespace",
			roleID:        "owner",
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("user1"), RoleID: "owner"},
				{UserID: ptr.String("user2"), RoleID: "owner"},
			},
			expectUserIDs: []string{"user1", "user2"},
		},
		{
			name:          "returns empty when no users have role",
			namespacePath: "test/namespace",
			roleID:        "owner",
			memberships:   []models.NamespaceMembership{},
			expectUserIDs: []string{},
		},
		{
			name:          "excludes service account memberships",
			namespacePath: "test/namespace",
			roleID:        "owner",
			memberships: []models.NamespaceMembership{
				{UserID: ptr.String("user1"), RoleID: "owner"},
				{ServiceAccountID: ptr.String("sa1"), RoleID: "owner"},
			},
			expectUserIDs: []string{"user1"},
		},
		{
			name:          "returns empty for empty memberships",
			namespacePath: "test/namespace",
			roleID:        "owner",
			memberships:   []models.NamespaceMembership{},
			expectUserIDs: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)

			mockNamespaceMemberships.On("GetNamespaceMemberships", ctx, &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					NamespacePaths: utils.ExpandPath(test.namespacePath),
					RoleID:         &test.roleID,
				},
			}).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.memberships,
			}, nil)

			dbClient := &db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
			}

			notificationManager := NewNotificationManager(dbClient, nil)

			userIDs, err := notificationManager.GetNamespaceMembersWithRole(ctx, test.namespacePath, test.roleID)

			if test.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.ElementsMatch(t, test.expectUserIDs, userIDs)
		})
	}
}
