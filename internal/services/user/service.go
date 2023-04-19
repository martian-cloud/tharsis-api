// Package user package
package user

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get user",
		)
	}

	if user == nil {
		return nil, errors.New(
			errors.ENotFound,
			"User with ID %s not found", userID,
		)
	}

	return user, nil
}

func (s *service) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	user, err := s.dbClient.Users.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, errors.Wrap(
			err,
			errors.EInternal,
			"Failed to get user",
		)
	}

	if user == nil {
		return nil, errors.New(
			errors.ENotFound,
			"User with username %s not found", username,
		)
	}

	return user, nil
}

func (s *service) GetUsers(ctx context.Context, input *GetUsersInput) (*db.UsersResult, error) {
	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
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
		return nil, err
	}

	return usersResult, nil
}

func (s *service) GetUsersByIDs(ctx context.Context, idList []string) ([]models.User, error) {
	// Any authenticated user can view basic user information
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{Filter: &db.UserFilter{UserIDs: idList}})
	if err != nil {
		return nil, err
	}

	return resp.Users, nil
}
