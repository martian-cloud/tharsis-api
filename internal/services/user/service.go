// Package user package
package user

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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

// UpdateAdminStatusForUserInput is the input for setting / unsetting users as admin.
type UpdateAdminStatusForUserInput struct {
	UserID string
	Admin  bool
}

// Service implements all user related functionality
type Service interface {
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUsers(ctx context.Context, input *GetUsersInput) (*db.UsersResult, error)
	GetUsersByIDs(ctx context.Context, idList []string) ([]models.User, error)
	UpdateAdminStatusForUser(ctx context.Context, input *UpdateAdminStatusForUserInput) (*models.User, error)
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return &service{logger, dbClient}
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

func (s *service) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.GetUserByUsername")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	user, err := s.dbClient.Users.GetUserByUsername(ctx, username)
	if err != nil {
		tracing.RecordError(span, err, "Failed to get user")
		return nil, errors.Wrap(
			err,
			"Failed to get user",
		)
	}

	if user == nil {
		tracing.RecordError(span, nil, "User with username %s not found", username)
		return nil, errors.New(
			"User with username %s not found", username,
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

	s.logger.Infow("Updated the admin status of a user.",
		"caller", caller.GetSubject(),
		"email", user.Email,
		"admin", input.Admin,
	)

	return updateUser, nil
}
