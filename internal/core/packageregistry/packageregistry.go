package packageregistry

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/semver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// GetPackageByID loads a package by ID, returning ENotFound when it does not exist. It performs no
// authorization — callers are responsible for permission checks.
func GetPackageByID(ctx context.Context, dbClient *db.Client, id string) (*models.Package, error) {
	pkg, err := dbClient.Packages.GetPackageByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, errors.New("package with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}
	return pkg, nil
}

// GetUploadedPackageVersion returns the uploaded package version for the given package id and
// semantic version, or nil when no such uploaded version exists (e.g. the package/version was
// deleted). It performs no authorization.
func GetUploadedPackageVersion(ctx context.Context, dbClient *db.Client, packageID, semanticVersion string) (*models.PackageVersion, error) {
	uploaded := models.PackageVersionStatusUploaded
	result, err := dbClient.PackageVersions.GetPackageVersions(ctx, &db.GetPackageVersionsInput{
		Filter: &db.PackageVersionFilter{
			PackageID:       &packageID,
			SemanticVersion: &semanticVersion,
			Status:          &uploaded,
		},
	})
	if err != nil {
		return nil, err
	}
	if len(result.PackageVersions) == 0 {
		return nil, nil
	}
	return &result.PackageVersions[0], nil
}

// ResolveVersionConstraint returns the uploaded package version of packageID that best satisfies the
// given version constraint, or nil when none does. An empty constraint means the latest uploaded
// version. It performs no authorization.
//
// Three paths, cheapest first: an empty constraint uses the latest flag, falling back to a scan
// because that flag is assigned at registration and is not cleared if the upload later fails; a bare
// exact version is queried directly; and a genuine range is matched in application code, since the
// DB layer has no semver sort.
func ResolveVersionConstraint(ctx context.Context, dbClient *db.Client, packageID, constraint string) (*models.PackageVersion, error) {
	uploaded := models.PackageVersionStatusUploaded

	if constraint == "" {
		latest := true
		result, err := dbClient.PackageVersions.GetPackageVersions(ctx, &db.GetPackageVersionsInput{
			Filter: &db.PackageVersionFilter{PackageID: &packageID, Status: &uploaded, Latest: &latest},
		})
		if err != nil {
			return nil, err
		}
		if len(result.PackageVersions) > 0 {
			return &result.PackageVersions[0], nil
		}
		// Fall through to the highest-uploaded scan below.
	} else if exact, isExact := semver.ExactVersion(constraint); isExact {
		return GetUploadedPackageVersion(ctx, dbClient, packageID, exact)
	}

	result, err := dbClient.PackageVersions.GetPackageVersions(ctx, &db.GetPackageVersionsInput{
		Filter: &db.PackageVersionFilter{PackageID: &packageID, Status: &uploaded},
	})
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(result.PackageVersions))
	byVersion := make(map[string]*models.PackageVersion, len(result.PackageVersions))
	for i := range result.PackageVersions {
		pv := &result.PackageVersions[i]
		versions = append(versions, pv.SemanticVersion)
		byVersion[pv.SemanticVersion] = pv
	}

	resolved, ok := semver.HighestMatching(versions, constraint)
	if !ok {
		return nil, nil
	}
	return byVersion[resolved], nil
}

// IsPackageVisibleToNamespace reports whether pkg is visible to the namespace at namespacePath. A
// package is visible when its visibility is global, when its visibility is root_group and it shares
// a root group with the namespace, or when its visibility is private and its owning group is an
// ancestor-or-self of the namespace.
func IsPackageVisibleToNamespace(pkg *models.Package, namespacePath string) bool {
	ownerGroupPath := pkg.GetGroupPath()

	switch pkg.Visibility {
	case models.PackageVisibilityGlobal:
		return true
	case models.PackageVisibilityRootGroup:
		return utils.RootGroupPath(ownerGroupPath) == utils.RootGroupPath(namespacePath)
	case models.PackageVisibilityPrivate:
		return namespacePath == ownerGroupPath || utils.IsDescendantOfPath(namespacePath, ownerGroupPath)
	default:
		return false
	}
}
