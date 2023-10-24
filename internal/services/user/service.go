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
)

// GetUsersInput is the input for listing users
type GetUsersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.UserSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// UsernamePrefix filters user list by username prefix
	UsernamePrefix *string
}

// Service implements all user related functionality
type Service interface {
	GetUserByID(ctx context.Context, userID string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUsers(ctx context.Context, input *GetUsersInput) (*db.UsersResult, error)
	GetUsersByIDs(ctx context.Context, idList []string) ([]models.User, error)
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
			UsernamePrefix: input.UsernamePrefix,
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
