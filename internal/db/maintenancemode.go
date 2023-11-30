package db

//go:generate mockery --name MaintenanceModes --inpackage --case underscore

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// maintenanceModeUUID is the ID of the maintenance mode record
	// to ensure that only one record exists in the database.
	maintenanceModeUUID = "35b88b7f-8579-44d2-a1c0-f6b53b34fcd2"
)

// MaintenanceModes encapsulates the logic to access maintenance modes from the database
type MaintenanceModes interface {
	GetMaintenanceMode(ctx context.Context) (*models.MaintenanceMode, error)
	CreateMaintenanceMode(ctx context.Context, mode *models.MaintenanceMode) (*models.MaintenanceMode, error)
	DeleteMaintenanceMode(ctx context.Context, mode *models.MaintenanceMode) error
}

type maintenanceModes struct {
	dbClient *Client
}

var maintenanceModesFieldList = append(metadataFieldList, "created_by", "message")

// NewMaintenanceModes returns an instance of the MaintenanceModes interface.
func NewMaintenanceModes(dbClient *Client) MaintenanceModes {
	return &maintenanceModes{dbClient: dbClient}
}

func (s *maintenanceModes) GetMaintenanceMode(ctx context.Context) (*models.MaintenanceMode, error) {
	ctx, span := tracer.Start(ctx, "db.GetMaintenanceMode")
	defer span.End()

	sql, args, err := dialect.From("maintenance_mode").
		Prepared(true).
		Select(maintenanceModesFieldList...).
		ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	mode, err := scanMaintenanceMode(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return mode, nil
}

func (s *maintenanceModes) CreateMaintenanceMode(ctx context.Context, mode *models.MaintenanceMode) (*models.MaintenanceMode, error) {
	ctx, span := tracer.Start(ctx, "db.CreateMaintenanceMode")
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Insert("maintenance_mode").
		Prepared(true).
		Rows(goqu.Record{
			"id":         maintenanceModeUUID,
			"version":    initialResourceVersion,
			"created_at": timestamp,
			"updated_at": timestamp,
			"created_by": mode.CreatedBy,
			"message":    mode.Message,
		}).
		Returning(maintenanceModesFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	createdMode, err := scanMaintenanceMode(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if pgErr := asPgError(err); pgErr != nil {
			if isUniqueViolation(pgErr) {
				return nil, errors.New("maintenance mode already enabled", errors.WithErrorCode(errors.EConflict), errors.WithSpan(span))
			}
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return createdMode, nil
}

func (s *maintenanceModes) DeleteMaintenanceMode(ctx context.Context, mode *models.MaintenanceMode) error {
	ctx, span := tracer.Start(ctx, "db.DeleteMaintenanceMode")
	defer span.End()

	sql, args, err := dialect.Delete("maintenance_mode").
		Prepared(true).
		Where(
			goqu.Ex{
				"id":      mode.Metadata.ID,
				"version": mode.Metadata.Version,
			},
		).Returning(maintenanceModesFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return err
	}

	if _, err = scanMaintenanceMode(s.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)); err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return ErrOptimisticLockError
		}

		tracing.RecordError(span, err, "failed to execute query")
		return err
	}

	return nil
}

func scanMaintenanceMode(row scanner) (*models.MaintenanceMode, error) {
	mode := &models.MaintenanceMode{}

	fields := []interface{}{
		&mode.Metadata.ID,
		&mode.Metadata.CreationTimestamp,
		&mode.Metadata.LastUpdatedTimestamp,
		&mode.Metadata.Version,
		&mode.CreatedBy,
		&mode.Message,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return mode, nil
}
