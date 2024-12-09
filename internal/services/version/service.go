// Package version provides functionality to get the current version of the API and its components.
package version

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Info is a struct that represents version information of the API and its components
type Info struct {
	APIVersion         string
	DBMigrationVersion string
	DBMigrationDirty   bool
}

// Service is an interface for the version service
type Service interface {
	GetCurrentVersion(ctx context.Context) (*Info, error)
}

type service struct {
	dbClient   *db.Client
	apiVersion string
}

// NewService creates a new version service
func NewService(dbClient *db.Client, apiVersion string) Service {
	return &service{dbClient, apiVersion}
}

// GetCurrentVersion returns version information of the API and its components
func (s *service) GetCurrentVersion(ctx context.Context) (*Info, error) {
	ctx, span := tracer.Start(ctx, "svc.GetCurrentVersion")
	defer span.End()

	// Any authenticated caller can get the version info
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	dbMigration, err := s.dbClient.SchemaMigrations.GetCurrentMigration(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current migration", errors.WithSpan(span))
	}

	return &Info{
		DBMigrationVersion: strconv.Itoa(dbMigration.Version),
		DBMigrationDirty:   dbMigration.Dirty,
		APIVersion:         s.apiVersion,
	}, nil
}
