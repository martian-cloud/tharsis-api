// Package resourcelimit package
package resourcelimit

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// UpdateResourceLimitInput is the input for updating (or creating) a (non-default) limit value.
type UpdateResourceLimitInput struct {
	MetadataVersion *int
	Name            string
	Value           int
}

// Service implements all resource limit related functionality
type Service interface {
	GetResourceLimits(ctx context.Context) ([]models.ResourceLimit, error)
	UpdateResourceLimit(ctx context.Context, input *UpdateResourceLimitInput) (*models.ResourceLimit, error)
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
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetResourceLimits(ctx context.Context) ([]models.ResourceLimit, error) {
	ctx, span := tracer.Start(ctx, "svc.GetResourceLimits")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	_, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Anyone is allowed to list the limits.

	result, err := s.dbClient.ResourceLimits.GetResourceLimits(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resource limits", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) UpdateResourceLimit(ctx context.Context, input *UpdateResourceLimitInput) (*models.ResourceLimit, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateResourceLimit")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New("Unsupported caller type, only users are allowed to update resource limits", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}
	// Only admins with admin mode activated are allowed to update resource limits.
	if !userCaller.IsAdminModeActivated(ctx) {
		return nil, errors.New("only admins with admin mode activated can update resource limits", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}

	// Validate the limit name/key.
	foundLimit, err := s.dbClient.ResourceLimits.GetResourceLimit(ctx, string(input.Name))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resource limit to validate name", errors.WithSpan(span))
	}
	if foundLimit == nil {
		return nil, errors.New("Invalid resource limit name", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	// Do an update DB operation.
	if input.MetadataVersion != nil {
		foundLimit.Metadata.Version = *input.MetadataVersion
	}
	foundLimit.Value = input.Value
	newLimit, err := s.dbClient.ResourceLimits.UpdateResourceLimit(ctx, foundLimit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update resource limit", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a resource limit.",
		"name", newLimit.Name,
		"value", newLimit.Value,
	)

	return newLimit, nil
}
