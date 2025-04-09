package namespace

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetNotificationPreferences(t *testing.T) {

	type testCase struct {
		name            string
		userIDs         []string
		namespacePath   *string
		preferences     []models.NotificationPreference
		expectErrorCode errors.CodeType
		expectResponse  map[string]*NotificationPreferenceSetting
	}

	testCases := []testCase{
		{
			name:    "successfully get global notification preferences",
			userIDs: []string{"user1", "user2"},
			preferences: []models.NotificationPreference{
				{UserID: "user1", Scope: models.NotificationPreferenceScopeAll},
			},
			expectResponse: map[string]*NotificationPreferenceSetting{
				"user1": {
					Inherited: false,
					Scope:     models.NotificationPreferenceScopeAll,
				},
				"user2": {
					Inherited: false,
					Scope:     defaultNotificationPreferenceScope,
				},
			},
		},
		{
			name:          "successfully get namespace notification preferences",
			userIDs:       []string{"user1", "user2", "user3"},
			namespacePath: ptr.String("group1/subgroup1"),
			preferences: []models.NotificationPreference{
				{UserID: "user1", Scope: models.NotificationPreferenceScopeAll},
				{UserID: "user1", Scope: models.NotificationPreferenceScopeNone, NamespacePath: ptr.String("group1/subgroup1/workspace1")},
				{UserID: "user1", Scope: models.NotificationPreferenceScopeParticipate, NamespacePath: ptr.String("group1/subgroup1")},
				{UserID: "user2", Scope: models.NotificationPreferenceScopeNone, NamespacePath: ptr.String("group1/subgroup2")},
				{UserID: "user3", Scope: models.NotificationPreferenceScopeNone, NamespacePath: ptr.String("group2")},
				{UserID: "user3", Scope: models.NotificationPreferenceScopeCustom, CustomEvents: &models.NotificationPreferenceCustomEvents{}},
			},
			expectResponse: map[string]*NotificationPreferenceSetting{
				"user1": {
					Inherited:     false,
					NamespacePath: ptr.String("group1/subgroup1"),
					Scope:         models.NotificationPreferenceScopeParticipate,
				},
				"user2": {
					Inherited: true,
					Scope:     defaultNotificationPreferenceScope,
				},
				"user3": {
					Inherited:    true,
					Scope:        models.NotificationPreferenceScopeCustom,
					CustomEvents: &models.NotificationPreferenceCustomEvents{},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNotificationPreferences := db.NewMockNotificationPreferences(t)

			var globalFilter *bool
			if test.namespacePath == nil {
				globalFilter = ptr.Bool(true)
			}
			mockNotificationPreferences.On("GetNotificationPreferences", ctx, &db.GetNotificationPreferencesInput{
				Filter: &db.NotificationPreferenceFilter{
					UserIDs: test.userIDs,
					Global:  globalFilter,
				},
			}).Return(
				&db.NotificationPreferencesResult{
					NotificationPreferences: test.preferences,
				},
				nil,
			)

			dbClient := &db.Client{
				NotificationPreferences: mockNotificationPreferences,
			}

			settingResolver := NewInheritedSettingResolver(dbClient)

			actualResponse, err := settingResolver.GetNotificationPreferences(ctx, test.userIDs, test.namespacePath)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResponse, actualResponse)
		})
	}
}
