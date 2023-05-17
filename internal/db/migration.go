package db

import (
	"context"
	"embed"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx" // Instantiating migrate command
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// migrationSchema contains the embedded .sql files that are
// needed to migrate the DB automatically.
//
//go:embed migrations/*.sql
var migrationSchema embed.FS

// migrations implements methods necessary to migrate the DB automatically.
type migrations struct {
	logger      logger.Logger
	databaseURL string
}

// newMigrations returns an instance of migrations struct.
func newMigrations(logger logger.Logger, databaseURL string) (*migrations, error) {
	dbURL, err := url.Parse(databaseURL)
	if err != nil {
		return nil, err
	}
	dbURL.Scheme = "pgx" // Change the scheme.

	return &migrations{
		logger:      logger,
		databaseURL: dbURL.String(),
	}, nil
}

// migrateUp migrates the DB to the latest version.
func (m *migrations) migrateUp() error {
	_, span := tracer.Start(context.Background(), "db.migrateUp")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	fsDriver, err := iofs.New(migrationSchema, "migrations")
	if err != nil {
		tracing.RecordError(span, err, "failed to get new iofs driver")
		return err
	}
	defer fsDriver.Close()

	migrateCmd, err := migrate.NewWithSourceInstance("iofs", fsDriver, m.databaseURL)
	if err != nil {
		tracing.RecordError(span, err, "failed to build migration command")
		return err
	}

	defer func() {
		sourceErr, dbDriverErr := migrateCmd.Close()
		if sourceErr != nil {
			m.logger.Errorf("failed to close migrate command source driver: %v", err)
		}
		if dbDriverErr != nil {
			m.logger.Errorf("failed to close migrate command DB driver: %v", err)
		}
	}()

	// Perform the migration.
	return migrateCmd.Up()
}
