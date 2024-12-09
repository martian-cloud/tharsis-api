package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/version"
)

// VersionResolver resolves the version of different API components
type VersionResolver struct {
	versionInfo *version.Info
}

// APIVersion resolver
func (r *VersionResolver) APIVersion() string {
	return r.versionInfo.APIVersion
}

// DBMigrationVersion resolver
func (r *VersionResolver) DBMigrationVersion() string {
	return r.versionInfo.DBMigrationVersion
}

// DBMigrationDirty resolver
func (r *VersionResolver) DBMigrationDirty() bool {
	return r.versionInfo.DBMigrationDirty
}

func versionQuery(ctx context.Context) (*VersionResolver, error) {
	versionInfo, err := getVersionService(ctx).GetCurrentVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &VersionResolver{versionInfo}, nil
}
