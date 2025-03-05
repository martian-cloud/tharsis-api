package db

//go:generate go tool mockery --name ResourceLimits --inpackage --case underscore

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
)

// ResourceLimits encapsulates the logic to access resource-limits from the database
// Because resource limits are all created via the schema, there is no need for a create method or a delete method.
type ResourceLimits interface {
	GetResourceLimit(ctx context.Context, name string) (*models.ResourceLimit, error)
	GetResourceLimits(ctx context.Context) ([]models.ResourceLimit, error)
	UpdateResourceLimit(ctx context.Context, resourceLimit *models.ResourceLimit) (*models.ResourceLimit, error)
}

type resourceLimits struct {
	dbClient *Client
}

var resourceLimitFieldList = append(metadataFieldList, "name", "value")

// NewResourceLimits returns an instance of the ResourceLimits interface
func NewResourceLimits(dbClient *Client) ResourceLimits {
	return &resourceLimits{dbClient: dbClient}
}

func (t *resourceLimits) GetResourceLimit(ctx context.Context, name string) (*models.ResourceLimit, error) {
	ctx, span := tracer.Start(ctx, "db.GetResourceLimit")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	query := dialect.From(goqu.T("resource_limits")).
		Prepared(true).
		Select(resourceLimitFieldList...).
		Where(goqu.Ex{"resource_limits.name": name})

	sql, args, err := query.ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	resourceLimit, err := scanResourceLimit(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return resourceLimit, nil
}

// GetResourceLimits returns the limits in ascending order by name.
func (t *resourceLimits) GetResourceLimits(ctx context.Context) ([]models.ResourceLimit, error) {
	ctx, span := tracer.Start(ctx, "db.GetResourceLimits")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Get the whole table, so no where clause.
	query := dialect.From(goqu.T("resource_limits")).
		Prepared(true).
		Select(resourceLimitFieldList...).
		Order(goqu.I("name").Asc())

	sql, args, err := query.ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	rows, err := t.dbClient.getConnection(ctx).Query(ctx, sql, args...)
	if err != nil {
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	defer rows.Close()

	// Scan rows
	results := []models.ResourceLimit{}
	for rows.Next() {
		item, err := scanResourceLimit(rows)
		if err != nil {
			tracing.RecordError(span, err, "failed to scan row")
			return nil, err
		}

		results = append(results, *item)
	}

	return results, nil
}

// UpdateResourceLimit updates only the value.
func (t *resourceLimits) UpdateResourceLimit(ctx context.Context, resourceLimit *models.ResourceLimit) (*models.ResourceLimit, error) {
	ctx, span := tracer.Start(ctx, "db.UpdateResourceLimit")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	timestamp := currentTime()

	sql, args, err := dialect.Update("resource_limits").
		Prepared(true).
		Set(
			goqu.Record{
				"version":    goqu.L("? + ?", goqu.C("version"), 1),
				"updated_at": timestamp,
				"value":      resourceLimit.Value,
			},
		).Where(goqu.Ex{"id": resourceLimit.Metadata.ID, "version": resourceLimit.Metadata.Version}).Returning(resourceLimitFieldList...).ToSQL()
	if err != nil {
		tracing.RecordError(span, err, "failed to generate SQL")
		return nil, err
	}

	updatedResourceLimit, err := scanResourceLimit(t.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			tracing.RecordError(span, err, "optimistic lock error")
			return nil, ErrOptimisticLockError
		}
		tracing.RecordError(span, err, "failed to execute query")
		return nil, err
	}

	return updatedResourceLimit, nil
}

func scanResourceLimit(row scanner) (*models.ResourceLimit, error) {
	resourceLimit := &models.ResourceLimit{}

	fields := []interface{}{
		&resourceLimit.Metadata.ID,
		&resourceLimit.Metadata.CreationTimestamp,
		&resourceLimit.Metadata.LastUpdatedTimestamp,
		&resourceLimit.Metadata.Version,
		&resourceLimit.Name,
		&resourceLimit.Value,
	}

	err := row.Scan(fields...)
	if err != nil {
		return nil, err
	}

	return resourceLimit, nil
}
