package namespace

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetRunnerTags(t *testing.T) {
	rootGroup := &models.Group{FullPath: "root", Metadata: models.ResourceMetadata{ID: "root-id"}}
	childGroup := &models.Group{FullPath: "root/child", ParentID: "root-id", Metadata: models.ResourceMetadata{ID: "child-id"}}

	t.Run("returns tags from namespace when set", func(t *testing.T) {
		group := &models.Group{
			FullPath:   "root/child",
			RunnerTags: []string{"tag1", "tag2"},
		}

		resolver := NewInheritedSettingResolver(&db.Client{})
		result, err := resolver.GetRunnerTags(t.Context(), group)

		require.NoError(t, err)
		assert.False(t, result.Inherited)
		assert.Equal(t, "root/child", result.NamespacePath)
		assert.Equal(t, []string{"tag1", "tag2"}, result.Value)
	})

	t.Run("returns empty tags for root group without tags", func(t *testing.T) {
		resolver := NewInheritedSettingResolver(&db.Client{})
		result, err := resolver.GetRunnerTags(t.Context(), rootGroup)

		require.NoError(t, err)
		assert.False(t, result.Inherited)
		assert.Equal(t, "root", result.NamespacePath)
		assert.Empty(t, result.Value)
	})

	t.Run("inherits tags from parent group", func(t *testing.T) {
		mockGroups := db.NewMockGroups(t)
		mockGroups.On("GetGroups", t.Context(), mock.MatchedBy(func(input *db.GetGroupsInput) bool {
			return input.Filter != nil && len(input.Filter.GroupPaths) == 1 && input.Filter.GroupPaths[0] == "root"
		})).Return(&db.GroupsResult{
			Groups: []models.Group{{FullPath: "root", RunnerTags: []string{"inherited-tag"}}},
		}, nil)

		resolver := NewInheritedSettingResolver(&db.Client{Groups: mockGroups})
		result, err := resolver.GetRunnerTags(t.Context(), childGroup)

		require.NoError(t, err)
		assert.True(t, result.Inherited)
		assert.Equal(t, "root", result.NamespacePath)
		assert.Equal(t, []string{"inherited-tag"}, result.Value)
	})
}

func TestGetDriftDetectionEnabled(t *testing.T) {
	rootGroup := &models.Group{FullPath: "root", Metadata: models.ResourceMetadata{ID: "root-id"}}
	childGroup := &models.Group{FullPath: "root/child", ParentID: "root-id", Metadata: models.ResourceMetadata{ID: "child-id"}}

	t.Run("returns value from namespace when set", func(t *testing.T) {
		group := &models.Group{
			FullPath:             "root/child",
			EnableDriftDetection: ptr.Bool(true),
		}

		resolver := NewInheritedSettingResolver(&db.Client{})
		result, err := resolver.GetDriftDetectionEnabled(t.Context(), group)

		require.NoError(t, err)
		assert.False(t, result.Inherited)
		assert.Equal(t, "root/child", result.NamespacePath)
		assert.True(t, result.Value)
	})

	t.Run("returns false for root group without setting", func(t *testing.T) {
		resolver := NewInheritedSettingResolver(&db.Client{})
		result, err := resolver.GetDriftDetectionEnabled(t.Context(), rootGroup)

		require.NoError(t, err)
		assert.False(t, result.Inherited)
		assert.Equal(t, "root", result.NamespacePath)
		assert.False(t, result.Value)
	})

	t.Run("inherits value from parent group", func(t *testing.T) {
		mockGroups := db.NewMockGroups(t)
		mockGroups.On("GetGroups", t.Context(), mock.MatchedBy(func(input *db.GetGroupsInput) bool {
			return input.Filter != nil && len(input.Filter.GroupPaths) == 1 && input.Filter.GroupPaths[0] == "root"
		})).Return(&db.GroupsResult{
			Groups: []models.Group{{FullPath: "root", EnableDriftDetection: ptr.Bool(true)}},
		}, nil)

		resolver := NewInheritedSettingResolver(&db.Client{Groups: mockGroups})
		result, err := resolver.GetDriftDetectionEnabled(t.Context(), childGroup)

		require.NoError(t, err)
		assert.True(t, result.Inherited)
		assert.Equal(t, "root", result.NamespacePath)
		assert.True(t, result.Value)
	})
}

func TestGetProviderMirrorEnabled(t *testing.T) {
	rootGroup := &models.Group{FullPath: "root", Metadata: models.ResourceMetadata{ID: "root-id"}}
	childGroup := &models.Group{FullPath: "root/child", ParentID: "root-id", Metadata: models.ResourceMetadata{ID: "child-id"}}

	t.Run("returns value from namespace when set", func(t *testing.T) {
		group := &models.Group{
			FullPath:             "root/child",
			EnableProviderMirror: ptr.Bool(true),
		}

		resolver := NewInheritedSettingResolver(&db.Client{})
		result, err := resolver.GetProviderMirrorEnabled(t.Context(), group)

		require.NoError(t, err)
		assert.False(t, result.Inherited)
		assert.Equal(t, "root/child", result.NamespacePath)
		assert.True(t, result.Value)
	})

	t.Run("returns false for root group without setting", func(t *testing.T) {
		resolver := NewInheritedSettingResolver(&db.Client{})
		result, err := resolver.GetProviderMirrorEnabled(t.Context(), rootGroup)

		require.NoError(t, err)
		assert.False(t, result.Inherited)
		assert.Equal(t, "root", result.NamespacePath)
		assert.False(t, result.Value)
	})

	t.Run("inherits value from parent group", func(t *testing.T) {
		mockGroups := db.NewMockGroups(t)
		mockGroups.On("GetGroups", t.Context(), mock.MatchedBy(func(input *db.GetGroupsInput) bool {
			return input.Filter != nil && len(input.Filter.GroupPaths) == 1 && input.Filter.GroupPaths[0] == "root"
		})).Return(&db.GroupsResult{
			Groups: []models.Group{{FullPath: "root", EnableProviderMirror: ptr.Bool(true)}},
		}, nil)

		resolver := NewInheritedSettingResolver(&db.Client{Groups: mockGroups})
		result, err := resolver.GetProviderMirrorEnabled(t.Context(), childGroup)

		require.NoError(t, err)
		assert.True(t, result.Inherited)
		assert.Equal(t, "root", result.NamespacePath)
		assert.True(t, result.Value)
	})
}

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
