// Package user package
package user

import (
	"context"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/attribute"
)

// GetUsersInput is the input for listing users
type GetUsersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.UserSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Search filters user list by username prefix
	Search *string
}

// GetUserSessionsInput is the input for listing user sessions
type GetUserSessionsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.UserSessionSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// UserID is the ID of the user to get sessions for
	UserID string
}

// UpdateAdminStatusForUserInput is the input for setting / unsetting users as admin.
type UpdateAdminStatusForUserInput struct {
	UserID string
	Admin  bool
}

// RevokeUserSessionInput is the input for revoking a user session.
type RevokeUserSessionInput struct {
	UserSessionID string
}

// CreateUserInput is the input for creating a user.
type CreateUserInput struct {
	Username string
	Email    string
	Password *string
	Admin    bool
}

// DeleteUserInput is the input for deleting a user.
type DeleteUserInput struct {
	UserID string
}

// SetUserPasswordInput is the input for setting a user's password.
type SetUserPasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// SetNotificationPreferenceInput is the input for setting notification preferences
type SetNotificationPreferenceInput struct {
	Inherit       bool
	NamespacePath *string
	Scope         *models.NotificationPreferenceScope
	CustomEvents  *models.NotificationPreferenceCustomEvents
}

// Validate validates the SetNotificationPreferenceInput
func (s *SetNotificationPreferenceInput) Validate() error {
	if s.Inherit {
		if s.NamespacePath == nil {
			return errors.New("namespace path must be set if inherit is true", errors.WithErrorCode(errors.EInvalid))
		}
		if s.Scope != nil {
			return errors.New("scope must not be set if inherit is true", errors.WithErrorCode(errors.EInvalid))
		}
	}

	if s.NamespacePath != nil && *s.NamespacePath == "" {
		return errors.New("namespace path must not be empty", errors.WithErrorCode(errors.EInvalid))
	}

	if s.Scope != nil {
		if !s.Scope.Valid() {
			return errors.New("scope is invalid", errors.WithErrorCode(errors.EInvalid))
		}
		if *s.Scope == models.NotificationPreferenceScopeCustom && s.CustomEvents == nil {
			return errors.New("custom events must be set if scope is custom", errors.WithErrorCode(errors.EInvalid))
		}
		if *s.Scope != models.NotificationPreferenceScopeCustom && s.CustomEvents != nil {
			return errors.New("custom events must not be set if scope is not custom", errors.WithErrorCode(errors.EInvalid))
		}
	}

	return nil
}

// GetNotificationPreferenceInput is the input for getting notification preferences
type GetNotificationPreferenceInput struct {
	NamespacePath *string
}

// GetNamespaceFavoritesInput is the input for listing namespace favorites
type GetNamespaceFavoritesInput struct {
	Sort              *db.NamespaceFavoriteSortableField
	PaginationOptions *pagination.Options
	NamespacePath     *string
}

// CreateNamespaceFavoriteInput is the input for creating a namespace favorite
type CreateNamespaceFavoriteInput struct {
	NamespacePath string
	NamespaceType namespace.Type
}

// DeleteNamespaceFavoriteInput is the input for deleting a namespace favorite
type DeleteNamespaceFavoriteInput struct {
	NamespacePath string
	NamespaceType namespace.Type
}

// Service implements all user related functionality
type Service interface {
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetUserByTRN(ctx context.Context, trn string) (*models.User, error)
	GetUsers(ctx context.Context, input *GetUsersInput) (*db.UsersResult, error)
	GetUsersByIDs(ctx context.Context, idList []string) ([]models.User, error)
	GetUserSessions(ctx context.Context, input *GetUserSessionsInput) (*db.UserSessionsResult, error)
	GetUserSessionByID(ctx context.Context, userSessionID string) (*models.UserSession, error)
	GetUserSessionByTRN(ctx context.Context, trn string) (*models.UserSession, error)
	UpdateAdminStatusForUser(ctx context.Context, input *UpdateAdminStatusForUserInput) (*models.User, error)
	RevokeUserSession(ctx context.Context, input *RevokeUserSessionInput) error
	CreateUser(ctx context.Context, input *CreateUserInput) (*models.User, error)
	DeleteUser(ctx context.Context, input *DeleteUserInput) error
	SetUserPassword(ctx context.Context, input *SetUserPasswordInput) (*models.User, error)
	SetNotificationPreference(ctx context.Context, input *SetNotificationPreferenceInput) (*namespace.NotificationPreferenceSetting, error)
	GetNotificationPreference(ctx context.Context, input *GetNotificationPreferenceInput) (*namespace.NotificationPreferenceSetting, error)
	GetNamespaceFavoriteByID(ctx context.Context, id string) (*models.NamespaceFavorite, error)
	GetNamespaceFavoriteByTRN(ctx context.Context, trn string) (*models.NamespaceFavorite, error)
	GetNamespaceFavorites(ctx context.Context, input *GetNamespaceFavoritesInput) (*db.NamespaceFavoritesResult, error)
	CreateNamespaceFavorite(ctx context.Context, input *CreateNamespaceFavoriteInput) (*models.NamespaceFavorite, error)
	DeleteNamespaceFavorite(ctx context.Context, input *DeleteNamespaceFavoriteInput) error
}

type service struct {
	logger                    logger.Logger
	dbClient                  *db.Client
	inheritedSettingsResolver namespace.InheritedSettingResolver
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	inheritedSettingsResolver namespace.InheritedSettingResolver,
) Service {
	return &service{logger, dbClient, inheritedSettingsResolver}
}

func (s *service) SetNotificationPreference(ctx context.Context, input *SetNotificationPreferenceInput) (*namespace.NotificationPreferenceSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.SetNotificationPreference")
	defer span.End()

	// Any authenticated user can view basic user information
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can set notification preferences", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Validate the input
	if err = input.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid input", errors.WithSpan(span))
	}

	// If this is for a namespace, check if the user has access to it
	if input.NamespacePath != nil {
		namespacePath := *input.NamespacePath
		permission := models.ViewWorkspacePermission
		// Check if this namespace is a group, if it's not a group we can assume it's a workspace
		group, err := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(namespacePath))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get group by TRN", errors.WithSpan(span))
		}
		if group != nil {
			permission = models.ViewGroupPermission
		}
		if err := userCaller.RequirePermission(ctx, permission, auth.WithNamespacePath(namespacePath)); err != nil {
			return nil, err
		}
	}

	var global *bool
	if input.NamespacePath == nil {
		global = ptr.Bool(true)
	}

	// Check for existing notification preference
	notificationPreference, err := s.dbClient.NotificationPreferences.GetNotificationPreferences(ctx, &db.GetNotificationPreferencesInput{
		Filter: &db.NotificationPreferenceFilter{
			UserIDs:       []string{userCaller.User.Metadata.ID},
			NamespacePath: input.NamespacePath,
			Global:        global,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get notification preferences", errors.WithSpan(span))
	}

	if len(notificationPreference.NotificationPreferences) > 0 {
		// Update existing notification preference
		np := notificationPreference.NotificationPreferences[0]
		if input.Inherit {
			// The existing notification preference will need to be deleted
			if err = s.dbClient.NotificationPreferences.DeleteNotificationPreference(ctx, &np); err != nil {
				return nil, errors.Wrap(err, "failed to delete notification preference", errors.WithSpan(span))
			}
		} else {
			np.Scope = *input.Scope
			np.CustomEvents = input.CustomEvents

			if err = np.Validate(); err != nil {
				return nil, errors.Wrap(err, "invalid notification preference", errors.WithSpan(span))
			}

			_, err = s.dbClient.NotificationPreferences.UpdateNotificationPreference(ctx, &np)
			if err != nil {
				return nil, errors.Wrap(err, "failed to update notification preference", errors.WithSpan(span))
			}
		}
	} else {
		// There is nothing to do if inherit is true since no notification preference exists
		if !input.Inherit {
			np := models.NotificationPreference{
				UserID:        userCaller.User.Metadata.ID,
				NamespacePath: input.NamespacePath,
				Scope:         *input.Scope,
				CustomEvents:  input.CustomEvents,
			}

			if err = np.Validate(); err != nil {
				return nil, errors.Wrap(err, "invalid notification preference", errors.WithSpan(span))
			}

			if _, err := s.dbClient.NotificationPreferences.CreateNotificationPreference(ctx, &np); err != nil {
				return nil, errors.Wrap(err, "failed to create notification preference", errors.WithSpan(span))
			}
		}
	}

	return s.inheritedSettingsResolver.GetNotificationPreference(ctx, userCaller.User.Metadata.ID, input.NamespacePath)
}

func (s *service) GetNotificationPreference(ctx context.Context, input *GetNotificationPreferenceInput) (*namespace.NotificationPreferenceSetting, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNotificationPreference")
	defer span.End()

	// Any authenticated user can view basic user information
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can get notification preferences", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// If this is for a namespace, check if the user has access to it
	if input.NamespacePath != nil {
		namespacePath := *input.NamespacePath
		permission := models.ViewWorkspacePermission
		// Check if this namespace is a group, if it's not a group we can assume it's a workspace
		group, err := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(namespacePath))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get group by TRN", errors.WithSpan(span))
		}
		if group != nil {
			permission = models.ViewGroupPermission
		}
		if err := userCaller.RequirePermission(ctx, permission, auth.WithNamespacePath(namespacePath)); err != nil {
			return nil, err
		}
	}

	return s.inheritedSettingsResolver.GetNotificationPreference(ctx, userCaller.User.Metadata.ID, input.NamespacePath)
}

func (s *service) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUserByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, userID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get user by ID")
		return nil, errors.Wrap(
			err,
			"Failed to get user",
		)
	}

	if user == nil {
		tracing.RecordError(span, nil, "User with ID %s not found", userID)
		return nil, errors.New(
			"User with ID %s not found", userID,
			errors.WithErrorCode(errors.ENotFound))
	}

	return user, nil
}

func (s *service) GetUserByTRN(ctx context.Context, trn string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUserByTRN")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	user, err := s.dbClient.Users.GetUserByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get user")
		return nil, errors.Wrap(
			err,
			"Failed to get user",
		)
	}

	if user == nil {
		tracing.RecordError(span, nil, "User with TRN %s not found", trn)
		return nil, errors.New(
			"User with TRN %s not found", trn,
			errors.WithErrorCode(errors.ENotFound))
	}

	return user, nil
}

func (s *service) GetUsers(ctx context.Context, input *GetUsersInput) (*db.UsersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUsers")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	usersResult, err := s.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.UserFilter{
			Search: input.Search,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get users")
		return nil, err
	}

	return usersResult, nil
}

func (s *service) GetUsersByIDs(ctx context.Context, idList []string) ([]models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUsersByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	resp, err := s.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{Filter: &db.UserFilter{UserIDs: idList}})
	if err != nil {
		tracing.RecordError(span, err, "failed to get users")
		return nil, err
	}

	return resp.Users, nil
}

func (s *service) GetUserSessions(ctx context.Context, input *GetUserSessionsInput) (*db.UserSessionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUserSessions")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can query user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Only admins or the current user can query user sessions
	if !userCaller.IsAdmin() && userCaller.User.Metadata.ID != input.UserID {
		return nil, errors.New("only admins or the current user can query user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	userSessionsResult, err := s.dbClient.UserSessions.GetUserSessions(ctx, &db.GetUserSessionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.UserSessionFilter{
			UserID: &input.UserID,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user sessions", errors.WithSpan(span))
	}

	return userSessionsResult, nil
}

func (s *service) GetUserSessionByID(ctx context.Context, userSessionID string) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUserSessionByID")
	span.SetAttributes(attribute.String("session_id", userSessionID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can query user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	userSession, err := s.dbClient.UserSessions.GetUserSessionByID(ctx, userSessionID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user session", errors.WithSpan(span))
	}

	if userSession == nil {
		return nil, errors.New("user session with ID %s not found", userSessionID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Only admins or the session owner can access the user session
	if !userCaller.IsAdmin() && userCaller.User.Metadata.ID != userSession.UserID {
		return nil, errors.New("only admins or the session owner can access user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	return userSession, nil
}

func (s *service) GetUserSessionByTRN(ctx context.Context, trn string) (*models.UserSession, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUserSessionByTRN")
	span.SetAttributes(attribute.String("trn", trn))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can query user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	userSession, err := s.dbClient.UserSessions.GetUserSessionByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user session", errors.WithSpan(span))
	}

	if userSession == nil {
		return nil, errors.New("user session with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Only admins or the session owner can access the user session
	if !userCaller.IsAdmin() && userCaller.User.Metadata.ID != userSession.UserID {
		return nil, errors.New("only admins or the session owner can access user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	return userSession, nil
}

func (s *service) UpdateAdminStatusForUser(ctx context.Context, input *UpdateAdminStatusForUserInput) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateAdminStatusForUser")
	span.SetAttributes(attribute.String("userID", input.UserID))
	span.SetAttributes(attribute.Bool("admin", input.Admin))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can update admin status for other users", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	if !userCaller.IsAdmin() {
		return nil, errors.New("only admins users can alter admin status of other users", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Nothing wrong with this just prevents accidental changes to self.
	if input.UserID == userCaller.User.Metadata.ID {
		return nil, errors.New("a user cannot alter their own admin status", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, input.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user by ID", errors.WithSpan(span))
	}

	if user == nil {
		return nil, errors.New("user with id %s not found", input.UserID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if !user.Active && input.Admin {
		return nil, errors.New("user %s is not active in the system; only active users can be granted admin rights",
			user.Username,
			errors.WithErrorCode(errors.EInvalid),
			errors.WithSpan(span),
		)
	}

	if user.Admin == input.Admin {
		// Short-circuit since we're already updated.
		return user, nil
	}

	user.Admin = input.Admin

	updateUser, err := s.dbClient.Users.UpdateUser(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated the admin status of a user.",
		"email", user.Email,
		"admin", input.Admin,
	)

	return updateUser, nil
}

func (s *service) RevokeUserSession(ctx context.Context, input *RevokeUserSessionInput) error {
	ctx, span := tracer.Start(ctx, "svc.RevokeUserSession")
	span.SetAttributes(attribute.String("user_session_id", input.UserSessionID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return errors.New("only users can revoke user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Get the user session to verify ownership
	userSession, err := s.dbClient.UserSessions.GetUserSessionByID(ctx, input.UserSessionID)
	if err != nil {
		return errors.Wrap(err, "failed to get user session", errors.WithSpan(span))
	}

	if userSession == nil {
		return errors.New("user session with ID %s not found", input.UserSessionID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	// Only admins or the session owner can revoke the user session
	if !userCaller.IsAdmin() && userCaller.User.Metadata.ID != userSession.UserID {
		return errors.New("only admins or the session owner can revoke user sessions", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Delete the user session
	if err := s.dbClient.UserSessions.DeleteUserSession(ctx, userSession); err != nil {
		return errors.Wrap(err, "failed to revoke user session", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Revoked a user session.",
		"session_id", input.UserSessionID,
		"session_user_id", userSession.UserID,
	)

	return nil
}

func (s *service) CreateUser(ctx context.Context, input *CreateUserInput) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateUser")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	if !caller.IsAdmin() {
		return nil, errors.New("only admin users can create other users", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	user := &models.User{
		Username: input.Username,
		Email:    input.Email,
		Admin:    input.Admin,
		Active:   true,
	}

	if err = user.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid user", errors.WithSpan(span))
	}

	if input.Password != nil {
		if err = user.SetPassword(*input.Password); err != nil {
			return nil, errors.Wrap(err, "failed to set password", errors.WithSpan(span))
		}
	}

	createdUser, err := s.dbClient.Users.CreateUser(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a new user.",
		"username", input.Username,
		"admin", input.Admin,
	)

	return createdUser, nil
}

func (s *service) DeleteUser(ctx context.Context, input *DeleteUserInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteUser")
	span.SetAttributes(attribute.String("userID", input.UserID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	if !caller.IsAdmin() {
		return errors.New("only admin users can delete other users", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return errors.New("only users can delete other users", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Prevent self-deletion
	if input.UserID == userCaller.User.Metadata.ID {
		return errors.New("a user cannot delete themselves", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, input.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to get user by ID", errors.WithSpan(span))
	}

	if user == nil {
		return errors.New("user with id %s not found", input.UserID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err := s.dbClient.Users.DeleteUser(ctx, user); err != nil {
		return errors.Wrap(err, "failed to delete user", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a user.",
		"username", user.Username,
	)

	return nil
}

func (s *service) SetUserPassword(ctx context.Context, input *SetUserPasswordInput) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.SetUserPassword")
	span.SetAttributes(attribute.String("userID", input.UserID))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can set passwords", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	if userCaller.User.Metadata.ID != input.UserID {
		return nil, errors.New("users can only set their own password", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, input.UserID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user by ID", errors.WithSpan(span))
	}

	if user == nil {
		return nil, errors.New("user with id %s not found", input.UserID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if !user.VerifyPassword(input.CurrentPassword) {
		return nil, errors.New("current password is incorrect", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	if err = user.SetPassword(input.NewPassword); err != nil {
		return nil, errors.Wrap(err, "failed to set password", errors.WithSpan(span))
	}

	updatedUser, err := s.dbClient.Users.UpdateUser(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated user password.",
		"username", user.Username,
	)

	return updatedUser, nil
}

// NamespaceFavorite service methods
func (s *service) GetNamespaceFavoriteByID(ctx context.Context, id string) (*models.NamespaceFavorite, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceFavoriteByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can access namespace favorites", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	favorite, err := s.dbClient.NamespaceFavorites.GetNamespaceFavoriteByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get namespace favorite", errors.WithSpan(span))
	}

	if favorite == nil || favorite.UserID != userCaller.User.Metadata.ID {
		return nil, errors.New("namespace favorite with ID %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return favorite, nil
}

func (s *service) GetNamespaceFavoriteByTRN(ctx context.Context, trn string) (*models.NamespaceFavorite, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceFavoriteByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can access namespace favorites", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	favorite, err := s.dbClient.NamespaceFavorites.GetNamespaceFavoriteByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get namespace favorite", errors.WithSpan(span))
	}

	if favorite == nil || favorite.UserID != userCaller.User.Metadata.ID {
		return nil, errors.New("namespace favorite with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return favorite, nil
}

func (s *service) GetNamespaceFavorites(ctx context.Context, input *GetNamespaceFavoritesInput) (*db.NamespaceFavoritesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetNamespaceFavorites")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can get namespace favorites", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	return s.dbClient.NamespaceFavorites.GetNamespaceFavorites(ctx, &db.GetNamespaceFavoritesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.NamespaceFavoriteFilter{
			UserIDs:       []string{userCaller.User.Metadata.ID},
			NamespacePath: input.NamespacePath,
		},
	})
}

func (s *service) CreateNamespaceFavorite(ctx context.Context, input *CreateNamespaceFavoriteInput) (*models.NamespaceFavorite, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateNamespaceFavorite")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("only users can create namespace favorites", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	if input.NamespacePath == "" {
		return nil, errors.New("namespace path cannot be empty", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	namespacePath := input.NamespacePath
	var groupID *string
	var workspaceID *string

	switch input.NamespaceType {
	case namespace.TypeGroup:
		if err := userCaller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(namespacePath)); err != nil {
			return nil, err
		}
		group, err := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(namespacePath))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get group", errors.WithSpan(span))
		}
		if group == nil {
			return nil, errors.New("namespace path does not correspond to a group", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
		groupID = &group.Metadata.ID
	case namespace.TypeWorkspace:
		if err := userCaller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(namespacePath)); err != nil {
			return nil, err
		}
		workspace, err := s.dbClient.Workspaces.GetWorkspaceByTRN(ctx, types.WorkspaceModelType.BuildTRN(namespacePath))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get workspace", errors.WithSpan(span))
		}
		if workspace == nil {
			return nil, errors.New("namespace path does not correspond to a workspace", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
		}
		workspaceID = &workspace.Metadata.ID
	default:
		return nil, errors.New("invalid namespace type", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	favorite := &models.NamespaceFavorite{
		UserID:      userCaller.User.Metadata.ID,
		GroupID:     groupID,
		WorkspaceID: workspaceID,
	}

	if err = favorite.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid namespace favorite", errors.WithSpan(span))
	}

	createdFavorite, err := s.dbClient.NamespaceFavorites.CreateNamespaceFavorite(ctx, favorite)
	if err != nil {
		if errors.ErrorCode(err) == errors.EConflict {
			existingFavorites, getErr := s.dbClient.NamespaceFavorites.GetNamespaceFavorites(ctx, &db.GetNamespaceFavoritesInput{
				Filter: &db.NamespaceFavoriteFilter{
					UserIDs:       []string{userCaller.User.Metadata.ID},
					NamespacePath: &namespacePath,
				},
			})
			if getErr != nil {
				return nil, errors.Wrap(getErr, "failed to get existing namespace favorite", errors.WithSpan(span))
			}
			if len(existingFavorites.NamespaceFavorites) > 0 {
				return &existingFavorites.NamespaceFavorites[0], nil
			}
		}
		return nil, errors.Wrap(err, "failed to create namespace favorite", errors.WithSpan(span))
	}

	return createdFavorite, nil
}

func (s *service) DeleteNamespaceFavorite(ctx context.Context, input *DeleteNamespaceFavoriteInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteNamespaceFavorite")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return errors.Wrap(err, "caller not authenticated", errors.WithSpan(span))
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return errors.New("only users can delete namespace favorites", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	if input.NamespacePath == "" {
		return errors.New("namespace path cannot be empty", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	namespacePath := input.NamespacePath

	switch input.NamespaceType {
	case namespace.TypeGroup:
		if err := userCaller.RequirePermission(ctx, models.ViewGroupPermission, auth.WithNamespacePath(namespacePath)); err != nil {
			return errors.Wrap(err, "permission check failed", errors.WithSpan(span))
		}
	case namespace.TypeWorkspace:
		if err := userCaller.RequirePermission(ctx, models.ViewWorkspacePermission, auth.WithNamespacePath(namespacePath)); err != nil {
			return errors.Wrap(err, "permission check failed", errors.WithSpan(span))
		}
	default:
		return errors.New("invalid namespace type", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	favorites, err := s.dbClient.NamespaceFavorites.GetNamespaceFavorites(ctx, &db.GetNamespaceFavoritesInput{
		Filter: &db.NamespaceFavoriteFilter{
			UserIDs:       []string{userCaller.User.Metadata.ID},
			NamespacePath: &input.NamespacePath,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get namespace favorites", errors.WithSpan(span))
	}

	if len(favorites.NamespaceFavorites) == 0 {
		return nil
	}

	favorite := favorites.NamespaceFavorites[0]
	if err = s.dbClient.NamespaceFavorites.DeleteNamespaceFavorite(ctx, &favorite); err != nil {
		if errors.ErrorCode(err) == errors.EOptimisticLock {
			existingFavorites, getErr := s.dbClient.NamespaceFavorites.GetNamespaceFavorites(ctx, &db.GetNamespaceFavoritesInput{
				Filter: &db.NamespaceFavoriteFilter{
					UserIDs:       []string{userCaller.User.Metadata.ID},
					NamespacePath: &input.NamespacePath,
				},
			})
			if getErr != nil {
				return errors.Wrap(getErr, "failed to get namespace favorites", errors.WithSpan(span))
			}
			if len(existingFavorites.NamespaceFavorites) == 0 {
				return nil
			}
		}
		return errors.Wrap(err, "failed to delete namespace favorite", errors.WithSpan(span))
	}
	return nil
}
