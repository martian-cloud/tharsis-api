// Package maintenance contains the service for enabling/disabling maintenance mode
package maintenance

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// EnableMaintenanceModeInput is the input for enabling maintenance mode
type EnableMaintenanceModeInput struct {
	Message string
}

// Service is the interface for the maintenance service
type Service interface {
	GetMaintenanceMode(ctx context.Context) (*models.MaintenanceMode, error)
	EnableMaintenanceMode(ctx context.Context, input *EnableMaintenanceModeInput) (*models.MaintenanceMode, error)
	DisableMaintenanceMode(ctx context.Context) error
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates a new maintenance service
func NewService(logger logger.Logger, dbClient *db.Client) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetMaintenanceMode(ctx context.Context) (*models.MaintenanceMode, error) {
	ctx, span := tracer.Start(ctx, "svc.GetMaintenanceMode")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	maintenanceMode, err := s.dbClient.MaintenanceModes.GetMaintenanceMode(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to get maintenance mode")
		return nil, err
	}

	if maintenanceMode == nil {
		return nil, errors.New("maintenance mode is not enabled", errors.WithErrorCode(errors.ENotFound))
	}

	return maintenanceMode, nil
}

func (s *service) EnableMaintenanceMode(ctx context.Context, input *EnableMaintenanceModeInput) (*models.MaintenanceMode, error) {
	ctx, span := tracer.Start(ctx, "svc.EnableMaintenanceMode")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if !caller.IsAdmin() {
		tracing.RecordError(span, nil, "only system admins can enable maintenance mode")
		return nil, errors.New("only system admins can enable maintenance mode", errors.WithErrorCode(errors.EForbidden))
	}

	toCreate := &models.MaintenanceMode{
		CreatedBy: caller.GetSubject(),
		Message:   input.Message,
	}

	created, err := s.dbClient.MaintenanceModes.CreateMaintenanceMode(ctx, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create maintenance mode")
		return nil, err
	}

	s.logger.Infow("Enabled maintenance mode.",
		"caller", caller.GetSubject(),
		"maintenance_mode_id", created.Metadata.ID,
	)

	return created, nil
}

func (s *service) DisableMaintenanceMode(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "svc.DisableMaintenanceMode")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	if !caller.IsAdmin() {
		tracing.RecordError(span, nil, "only system admins can disable maintenance mode")
		return errors.New("only system admins can disable maintenance mode", errors.WithErrorCode(errors.EForbidden))
	}

	maintenanceMode, err := s.dbClient.MaintenanceModes.GetMaintenanceMode(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to get maintenance mode")
		return err
	}

	if maintenanceMode == nil {
		tracing.RecordError(span, nil, "maintenance mode is not enabled")
		return errors.New("maintenance mode is not enabled", errors.WithErrorCode(errors.EInvalid))
	}

	if err = s.dbClient.MaintenanceModes.DeleteMaintenanceMode(ctx, maintenanceMode); err != nil {
		tracing.RecordError(span, err, "failed to delete maintenance mode")
		return err
	}

	s.logger.Infow("Disabled maintenance mode.",
		"caller", caller.GetSubject(),
	)

	return nil
}
