package namespace

//go:generate go tool mockery --name InheritedSettingResolver --inpackage --case underscore

import (
	"context"
	"slices"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const defaultNotificationPreferenceScope = models.NotificationPreferenceScopeParticipate

// RunnerTagsSetting contains the inherited setting for runner tags
type RunnerTagsSetting struct {
	Inherited     bool
	NamespacePath string
	Value         []string
}

// DriftDetectionEnabledSetting contains the inherited setting for enabling drift detection
type DriftDetectionEnabledSetting struct {
	Inherited     bool
	NamespacePath string
	Value         bool
}

// ProviderMirrorEnabledSetting contains the inherited setting for enabling provider mirror
type ProviderMirrorEnabledSetting struct {
	Inherited     bool
	NamespacePath string
	Value         bool
}

// NotificationPreferenceSetting contains the inherited setting for user notification preferences
type NotificationPreferenceSetting struct {
	Inherited     bool
	NamespacePath *string
	Scope         models.NotificationPreferenceScope
	CustomEvents  *models.NotificationPreferenceCustomEvents
}

// InheritedSettingResolver is used to resolve inherited settings by searching the group hierarchy
type InheritedSettingResolver interface {
	GetRunnerTags(ctx context.Context, namespace Namespace) (*RunnerTagsSetting, error)
	GetDriftDetectionEnabled(ctx context.Context, namespace Namespace) (*DriftDetectionEnabledSetting, error)
	GetProviderMirrorEnabled(ctx context.Context, namespace Namespace) (*ProviderMirrorEnabledSetting, error)
	GetNotificationPreference(ctx context.Context, userID string, namespacePath *string) (*NotificationPreferenceSetting, error)
	GetNotificationPreferences(ctx context.Context, userIDs []string, namespacePath *string) (map[string]*NotificationPreferenceSetting, error)
}

type getSettingFunc func(namespace Namespace) (any, bool)

type setting struct {
	inherited     bool
	namespacePath string
	value         any
}

type inheritedSettingsResolver struct {
	dbClient *db.Client
}

// NewInheritedSettingResolver creates a new instance
func NewInheritedSettingResolver(dbClient *db.Client) InheritedSettingResolver {
	return &inheritedSettingsResolver{
		dbClient: dbClient,
	}
}

func (r *inheritedSettingsResolver) GetNotificationPreference(ctx context.Context, userID string, namespacePath *string) (*NotificationPreferenceSetting, error) {
	settingsPerUser, err := r.GetNotificationPreferences(ctx, []string{userID}, namespacePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query notification preference from db")
	}
	notificationPreferenceSetting, ok := settingsPerUser[userID]
	if !ok {
		return nil, errors.New("failed to get notification preference setting for user %s", userID)
	}
	return notificationPreferenceSetting, nil
}

func (r *inheritedSettingsResolver) GetNotificationPreferences(ctx context.Context, userIDs []string, namespacePath *string) (map[string]*NotificationPreferenceSetting, error) {
	var globalFilter *bool

	if len(userIDs) == 0 {
		return map[string]*NotificationPreferenceSetting{}, nil
	}

	if namespacePath == nil {
		// If namespace path is nil, we need only need to get global preferences
		globalFilter = ptr.Bool(true)
	}

	response, err := r.dbClient.NotificationPreferences.GetNotificationPreferences(ctx, &db.GetNotificationPreferencesInput{
		Filter: &db.NotificationPreferenceFilter{
			UserIDs: userIDs,
			Global:  globalFilter,
		},
	})
	if err != nil {
		return nil, err
	}

	notificationPreferenceListPerUser := map[string][]models.NotificationPreference{}
	for _, np := range response.NotificationPreferences {
		if np.IsGlobal() || (namespacePath != nil && (*namespacePath == *np.NamespacePath || utils.IsDescendantOfPath(*namespacePath, *np.NamespacePath))) {
			notificationPreferenceListPerUser[np.UserID] = append(notificationPreferenceListPerUser[np.UserID], np)
		}
	}

	for _, userID := range userIDs {
		if _, ok := notificationPreferenceListPerUser[userID]; !ok {
			notificationPreferenceListPerUser[userID] = []models.NotificationPreference{}
		}
	}

	// Sort each list in descending order by namespace path with global preferences at the end
	for _, notificationPreferences := range notificationPreferenceListPerUser {
		slices.SortFunc(notificationPreferences, func(a, b models.NotificationPreference) int {
			if a.IsGlobal() {
				// Return 1 when namespace path for a is nil to sort global preferences at the end
				return 1
			}
			if b.IsGlobal() {
				// Return -1 when namespace path for b is nil to sort global preferences at the end
				return -1
			}
			if *a.NamespacePath < *b.NamespacePath {
				// Sort namespace path in descending order
				return 1
			}
			return -1
		})
	}

	notificationPreferencePerUser := map[string]*NotificationPreferenceSetting{}
	for userID, notificationPreferences := range notificationPreferenceListPerUser {
		// The first preference is the most specific one
		if len(notificationPreferences) > 0 {
			np := notificationPreferences[0]

			// This setting is inherited if the namespace path is not nil and the namespace path is different from the one in the preference
			inherited := namespacePath != nil && (np.IsGlobal() || *namespacePath != *np.NamespacePath)

			notificationPreferencePerUser[userID] = &NotificationPreferenceSetting{
				Inherited:     inherited,
				NamespacePath: np.NamespacePath,
				Scope:         np.Scope,
				CustomEvents:  np.CustomEvents,
			}
		} else {
			notificationPreferencePerUser[userID] = &NotificationPreferenceSetting{
				Inherited:     namespacePath != nil,
				NamespacePath: nil,
				Scope:         defaultNotificationPreferenceScope,
				CustomEvents:  nil,
			}
		}
	}

	return notificationPreferencePerUser, nil
}

func (r *inheritedSettingsResolver) GetRunnerTags(ctx context.Context, namespace Namespace) (*RunnerTagsSetting, error) {
	response, err := r.getInheritedSetting(ctx, namespace, func(namespace Namespace) (any, bool) {
		tags := namespace.GetRunnerTags()
		return tags, tags != nil
	})
	if err != nil {
		return nil, err
	}

	value := []string{}
	if response.value != nil {
		value = response.value.([]string)
	}

	return &RunnerTagsSetting{
		Inherited:     response.inherited,
		NamespacePath: response.namespacePath,
		Value:         value,
	}, nil
}

func (r *inheritedSettingsResolver) GetDriftDetectionEnabled(ctx context.Context, namespace Namespace) (*DriftDetectionEnabledSetting, error) {
	response, err := r.getInheritedSetting(ctx, namespace, func(namespace Namespace) (any, bool) {
		enabled := namespace.DriftDetectionEnabled()
		if enabled == nil {
			return false, false
		}
		return *enabled, true
	})
	if err != nil {
		return nil, err
	}

	value := false
	if response.value != nil {
		value = response.value.(bool)
	}

	return &DriftDetectionEnabledSetting{
		Inherited:     response.inherited,
		NamespacePath: response.namespacePath,
		Value:         value,
	}, nil
}

func (r *inheritedSettingsResolver) GetProviderMirrorEnabled(ctx context.Context, namespace Namespace) (*ProviderMirrorEnabledSetting, error) {
	response, err := r.getInheritedSetting(ctx, namespace, func(namespace Namespace) (any, bool) {
		enabled := namespace.ProviderMirrorEnabled()
		if enabled == nil {
			return false, false
		}
		return *enabled, true
	})
	if err != nil {
		return nil, err
	}

	value := false
	if response.value != nil {
		value = response.value.(bool)
	}

	return &ProviderMirrorEnabledSetting{
		Inherited:     response.inherited,
		NamespacePath: response.namespacePath,
		Value:         value,
	}, nil
}

func (r *inheritedSettingsResolver) getInheritedSetting(ctx context.Context, namespace Namespace, getSetting getSettingFunc) (*setting, error) {
	// The group sets its own tags.
	if s, ok := getSetting(namespace); ok {
		return &setting{
			inherited:     false,
			namespacePath: namespace.GetPath(),
			value:         s,
		}, nil
	}

	// A root group has no ancestors.
	// To avoid false positives, don't look for ancestor groups.
	if namespace.GetParentID() == "" {
		// At this point, we know group setting is nil.
		return &setting{
			inherited:     false,
			namespacePath: namespace.GetPath(),
		}, nil
	}

	sortLowestToHighest := db.GroupSortableFieldFullPathDesc
	parentGroupsResult, err := r.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
		Sort: &sortLowestToHighest,
		Filter: &db.GroupFilter{
			GroupPaths: namespace.ExpandPath()[1:],
		},
	})
	if err != nil {
		return nil, err
	}

	parentGroups := []*models.Group{}
	for _, g := range parentGroupsResult.Groups {
		copyGroup := g
		parentGroups = append(parentGroups, &copyGroup)
	}

	// Find the first/lowest group with the setting defined
	for _, g := range parentGroups {
		if s, ok := getSetting(g); ok {
			return &setting{
				inherited:     true,
				namespacePath: g.FullPath,
				value:         s,
			}, nil
		}
	}

	// No setting found in any ancestor group.
	// The last group in the list is a root group, so return its full path.
	return &setting{
		inherited:     true,
		namespacePath: parentGroups[len(parentGroups)-1].FullPath,
	}, nil
}
