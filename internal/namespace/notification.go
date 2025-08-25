package namespace

//go:generate go tool mockery --name NotificationManager --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// GetUsersToNotifyInput is the input for GetUsersToNotify
type GetUsersToNotifyInput struct {
	NamespacePath      string
	ParticipantUserIDs []string
	CustomEventCheck   func(*models.NotificationPreferenceCustomEvents) bool
}

// NotificationManager manages notifications for a namespace
type NotificationManager interface {
	// GetUsersToNotify returns the list of users to notify for a given namespace path
	GetUsersToNotify(ctx context.Context, input *GetUsersToNotifyInput) ([]string, error)
}

type notificationManager struct {
	dbClient                 *db.Client
	inheritedSettingResolver InheritedSettingResolver
	logger                   logger.Logger
}

// NewNotificationManager creates a new instance of NotificationManager
func NewNotificationManager(dbClient *db.Client, inheritedSettingResolver InheritedSettingResolver) NotificationManager {
	return &notificationManager{
		dbClient:                 dbClient,
		inheritedSettingResolver: inheritedSettingResolver,
	}
}

func (n *notificationManager) GetUsersToNotify(ctx context.Context, input *GetUsersToNotifyInput) ([]string, error) {
	// Get all users who have access to this workspace
	membershipsResponse, err := n.dbClient.NamespaceMemberships.GetNamespaceMemberships(ctx, &db.GetNamespaceMembershipsInput{
		Filter: &db.NamespaceMembershipFilter{
			NamespacePaths: utils.ExpandPath(input.NamespacePath),
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get memberships for namespace %s", input.NamespacePath)
	}
	userIDMap := map[string]struct{}{}
	for _, membership := range membershipsResponse.NamespaceMemberships {
		if membership.UserID != nil {
			userIDMap[*membership.UserID] = struct{}{}
		}
	}

	// Add participant user IDs to the map
	for _, participantUserID := range input.ParticipantUserIDs {
		userIDMap[participantUserID] = struct{}{}
	}

	userIDs := make([]string, 0, len(userIDMap))
	for userID := range userIDMap {
		userIDs = append(userIDs, userID)
	}

	// Get the notification settings for the user.
	notificationSettings, err := n.inheritedSettingResolver.GetNotificationPreferences(ctx, userIDs, &input.NamespacePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notification settings for namespace %s", input.NamespacePath)
	}

	participantIDMap := map[string]struct{}{}
	for _, id := range input.ParticipantUserIDs {
		participantIDMap[id] = struct{}{}
	}

	filteredUsersIDs := []string{}
	for _, userID := range userIDs {
		setting, ok := notificationSettings[userID]
		if !ok {
			n.logger.WithContextFields(ctx).Errorf("user notification preference not found for user %s", userID)
			continue
		}
		switch setting.Scope {
		case models.NotificationPreferenceScopeAll:
			filteredUsersIDs = append(filteredUsersIDs, userID)
		case models.NotificationPreferenceScopeParticipate:
			if _, ok := participantIDMap[userID]; ok {
				filteredUsersIDs = append(filteredUsersIDs, userID)
			}
		case models.NotificationPreferenceScopeCustom:
			if input.CustomEventCheck != nil && input.CustomEventCheck(setting.CustomEvents) {
				filteredUsersIDs = append(filteredUsersIDs, userID)
			}
		}
	}

	return filteredUsersIDs, nil
}
