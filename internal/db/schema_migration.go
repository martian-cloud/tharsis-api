package db

//go:generate go tool mockery --name SchemaMigrations --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// SchemaMigrations is an interface for managing schema migrations
type SchemaMigrations interface {
	GetCurrentMigration(ctx context.Context) (*SchemaMigration, error)
}

// SchemaMigration is a struct that represents a schema migration
type SchemaMigration struct {
	Version int
	Dirty   bool
}

// schemaMigrations is a struct that implements the SchemaMigrations interface
type schemaMigrations struct {
	dbClient *Client
}

var schemaMigrationFieldList = []interface{}{"version", "dirty"}

// NewSchemaMigrations creates a new SchemaMigrations struct
func NewSchemaMigrations(dbClient *Client) SchemaMigrations {
	return &schemaMigrations{dbClient}
}

// GetCurrentMigration returns the current schema migration
func (sm *schemaMigrations) GetCurrentMigration(ctx context.Context) (*SchemaMigration, error) {
	ctx, span := tracer.Start(ctx, "db.GetCurrentMigration")
	defer span.End()

	sql, args, err := dialect.From("schema_migrations").
		Prepared(true).
		Select(schemaMigrationFieldList...).
		ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build SQL", errors.WithSpan(span))
	}

	row := sm.dbClient.getConnection(ctx).QueryRow(ctx, sql, args...)

	var schemaMigration SchemaMigration
	if err = row.Scan(
		&schemaMigration.Version,
		&schemaMigration.Dirty,
	); err != nil {
		return nil, errors.Wrap(err, "failed to scan row", errors.WithSpan(span))
	}

	return &schemaMigration, nil
}
