// Package packageregistry implements the generic package registry service: CRUD for packages and
// their versions plus version content upload/download. Package-kind-specific behavior (e.g. OPA
// policies/evaluation) lives in the policy service, which shares logic via
// internal/core/packageregistry rather than depending on this service.
package packageregistry

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/go-version"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	corepkg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/packageregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/semver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// CreatePackageInput is the input for creating a package.
type CreatePackageInput struct {
	Name                 string
	Description          *string
	GroupID              string
	Kind                 models.PackageKind
	Visibility           models.PackageVisibility
	AllowMutableVersions bool
}

// UpdatePackageInput is the input for updating a package.
type UpdatePackageInput struct {
	ID                   string
	Description          *string
	Visibility           *models.PackageVisibility
	AllowMutableVersions *bool
}

// CreatePackageVersionInput is the input for creating a package version.
type CreatePackageVersionInput struct {
	SemanticVersion string
	PackageID       string
	SHASum          []byte
}

// GetPackagesInput is the input for listing packages.
type GetPackagesInput struct {
	Sort              *db.PackageSortableField
	PaginationOptions *pagination.Options
	Group             *models.Group
	Search            *string
	IncludeInherited  bool
}

// GetVisiblePackagesInput is the input for listing every package a group may reference.
type GetVisiblePackagesInput struct {
	Sort              *db.PackageSortableField
	PaginationOptions *pagination.Options
	Group             *models.Group
	Search            *string
}

// GetPackageVersionsInput is the input for listing package versions.
type GetPackageVersionsInput struct {
	Sort              *db.PackageVersionSortableField
	PaginationOptions *pagination.Options
	Status            *models.PackageVersionStatus
	SemanticVersion   *string
	Latest            *bool
	PackageID         string
	Search            *string
}

// ResolvePackageVersionInput is the input for resolving a version constraint to a concrete uploaded
// package version. An empty VersionConstraint resolves to the latest uploaded version.
type ResolvePackageVersionInput struct {
	PackageSource     string
	VersionConstraint string
}

// Service implements the generic package registry.
type Service interface {
	GetPackageByID(ctx context.Context, id string) (*models.Package, error)
	GetPackageByTRN(ctx context.Context, trn string) (*models.Package, error)
	GetPackagesByIDs(ctx context.Context, ids []string) ([]models.Package, error)
	GetPackages(ctx context.Context, input *GetPackagesInput) (*db.PackagesResult, error)
	GetVisiblePackages(ctx context.Context, input *GetVisiblePackagesInput) (*db.PackagesResult, error)
	CreatePackage(ctx context.Context, input *CreatePackageInput) (*models.Package, error)
	UpdatePackage(ctx context.Context, input *UpdatePackageInput) (*models.Package, error)
	DeletePackage(ctx context.Context, pkg *models.Package) error
	GetPackageVersionByID(ctx context.Context, id string) (*models.PackageVersion, error)
	GetPackageVersionByTRN(ctx context.Context, trn string) (*models.PackageVersion, error)
	GetPackageVersionsByIDs(ctx context.Context, ids []string) ([]models.PackageVersion, error)
	GetPackageVersions(ctx context.Context, input *GetPackageVersionsInput) (*db.PackageVersionsResult, error)
	ResolvePackageVersion(ctx context.Context, input *ResolvePackageVersionInput) (*models.PackageVersion, error)
	CreatePackageVersion(ctx context.Context, input *CreatePackageVersionInput) (*models.PackageVersion, error)
	DeletePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) error
	UploadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion, reader io.Reader) error
	DownloadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion) (string, error)
}

type service struct {
	logger       logger.Logger
	dbClient     *db.Client
	limitChecker limits.LimitChecker
	store        corepkg.PackageStore
}

// NewService creates an instance of Service.
func NewService(logger logger.Logger, dbClient *db.Client, limitChecker limits.LimitChecker, store corepkg.PackageStore) Service {
	return &service{logger: logger, dbClient: dbClient, limitChecker: limitChecker, store: store}
}

// authorizeViewPackage authorizes a read of pkg based on its visibility:
//   - global:     any authenticated subject may view it, so access is granted unconditionally.
//   - root_group: the caller must have inherited view access somewhere within the package's root
//     group hierarchy, so the check is run against the root group.
//   - private:    the caller must have inherited view access within the group the package lives in.
func (s *service) authorizeViewPackage(ctx context.Context, caller auth.Caller, pkg *models.Package) error {
	switch pkg.Visibility {
	case models.PackageVisibilityGlobal:
		// A globally visible package is viewable by any authenticated subject.
		return nil
	case models.PackageVisibilityRootGroup:
		rootGroup, err := s.dbClient.Groups.GetGroupByTRN(ctx, trn.TypeGroup.Build(pkg.GetRootGroupPath()))
		if err != nil {
			return err
		}
		if rootGroup == nil {
			return caller.UnauthorizedError(ctx, false)
		}
		return caller.RequireAccessToInheritableResource(ctx, types.PackageModelType, auth.WithGroupID(rootGroup.Metadata.ID))
	case models.PackageVisibilityPrivate:
		return caller.RequireAccessToInheritableResource(ctx, types.PackageModelType, auth.WithGroupID(pkg.GroupID))
	default:
		return caller.UnauthorizedError(ctx, false)
	}
}

func (s *service) GetPackageByID(ctx context.Context, id string) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackageByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return nil, err
	}

	if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return pkg, nil
}

func (s *service) GetPackageByTRN(ctx context.Context, trn string) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackageByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	pkg, err := s.dbClient.Packages.GetPackageByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by TRN")
		return nil, err
	}
	if pkg == nil {
		return nil, errors.New("package with trn %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return pkg, nil
}

// GetPackagesByIDs returns the packages with the given IDs. View-permission checks are deduplicated
// across packages that authorize identically, because a batch usually holds several packages from the
// same group and each distinct check hits the auth layer (and, for root-group packages, a group
// lookup).
func (s *service) GetPackagesByIDs(ctx context.Context, ids []string) ([]models.Package, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackagesByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.Packages.GetPackages(ctx, &db.GetPackagesInput{
		Filter: &db.PackageFilter{PackageIDs: ids},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get packages")
		return nil, err
	}

	// The authorization scope is what authorizeViewPackage keys on: a global package needs no check,
	// a root-group package is checked against its root group, and a private package against its owning
	// group. Deduplicate on that scope so N packages sharing one scope cost one check, not N.
	checkedScopes := map[string]struct{}{}
	for i := range result.Packages {
		pkg := &result.Packages[i]

		var scope string
		switch pkg.Visibility {
		case models.PackageVisibilityGlobal:
			// Viewable by any authenticated subject; no per-package check to run or dedupe.
			continue
		case models.PackageVisibilityRootGroup:
			scope = "root:" + pkg.GetRootGroupPath()
		case models.PackageVisibilityPrivate:
			scope = "private:" + pkg.GroupID
		default:
			// Unknown visibility authorizes per-package (authorizeViewPackage returns unauthorized);
			// key on the id so it is never collapsed with a real scope.
			scope = "unknown:" + pkg.Metadata.ID
		}

		if _, checked := checkedScopes[scope]; checked {
			continue
		}
		if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		checkedScopes[scope] = struct{}{}
	}

	return result.Packages, nil
}

func (s *service) GetPackages(ctx context.Context, input *GetPackagesInput) (*db.PackagesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackages")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.Group == nil {
		// Global (cross-group) listing. Non-admin callers are limited to packages they can access
		// via their root namespace memberships plus globally-visible packages; admins see all.
		dbInput := &db.GetPackagesInput{
			Sort:              input.Sort,
			PaginationOptions: input.PaginationOptions,
			Filter: &db.PackageFilter{
				Search: input.Search,
			},
		}

		if !caller.IsAdminModeActivated(ctx) {
			rootNamespaces, rErr := caller.GetRootNamespaceMemberships(ctx)
			if rErr != nil {
				tracing.RecordError(span, rErr, "failed to get root namespaces")
				return nil, rErr
			}
			dbInput.Filter.RootNamespaceMemberships = rootNamespaces
		}

		return s.dbClient.Packages.GetPackages(ctx, dbInput)
	}

	if err = caller.RequirePermission(ctx, models.ViewPackagePermission, auth.WithNamespacePath(input.Group.FullPath)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	dbInput := &db.GetPackagesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.PackageFilter{
			Search: input.Search,
		},
	}

	if input.IncludeInherited {
		// Every package owned by the group or one of its ancestors is visible to the group, so the
		// ancestor-or-self path list is a sufficient (and visibility-agnostic) scope.
		dbInput.Filter.NamespacePaths = input.Group.ExpandPath()
	} else {
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	}

	return s.dbClient.Packages.GetPackages(ctx, dbInput)
}

// GetVisiblePackages lists every package the group may reference in a policy: the packages it and its
// ancestors own, which is what GetPackages with IncludeInherited returns, plus the ones owned outside
// that ancestry whose own visibility opens them up to the group — root_group packages sharing its root
// group, and global packages anywhere. It is the listing counterpart of authorizeViewPackage, which
// applies the same rules to a single package.
//
// This is deliberately a separate method rather than another mode of GetPackages: the set is defined by
// the packages' visibility rather than by how far the caller opts into the group's ancestry, and it has
// no cross-group form, so it does not share GetPackages' nil-Group branch.
func (s *service) GetVisiblePackages(ctx context.Context, input *GetVisiblePackagesInput) (*db.PackagesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetVisiblePackages")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if input.Group == nil {
		err = errors.New("group is required", errors.WithErrorCode(errors.EInvalid))
		tracing.RecordError(span, err, "missing group")
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ViewPackagePermission, auth.WithNamespacePath(input.Group.FullPath)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return s.dbClient.Packages.GetPackages(ctx, &db.GetPackagesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.PackageFilter{
			Search:         input.Search,
			VisibleToGroup: &input.Group.FullPath,
		},
	})
}

func (s *service) CreatePackage(ctx context.Context, input *CreatePackageInput) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "svc.CreatePackage")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.CreatePackagePermission, auth.WithGroupID(input.GroupID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by ID")
		return nil, err
	}
	if group == nil {
		return nil, errors.New("group with id %s not found", input.GroupID, errors.WithErrorCode(errors.ENotFound))
	}

	// Resolve the package's root group so it can be cached on the row for root_group-visibility listings.
	rootGroupID := group.Metadata.ID
	if group.ParentID != "" {
		rootGroup, gErr := s.dbClient.Groups.GetGroupByTRN(ctx, trn.TypeGroup.Build(group.GetRootGroupPath()))
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get root group")
			return nil, gErr
		}
		if rootGroup == nil {
			return nil, errors.New("root group with path %s not found", group.GetRootGroupPath(), errors.WithErrorCode(errors.ENotFound))
		}
		rootGroupID = rootGroup.Metadata.ID
	}

	packageKind := input.Kind
	if packageKind == "" {
		packageKind = models.PackageKindOPAPolicy
	}

	visibility := input.Visibility
	if visibility == "" {
		visibility = models.PackageVisibilityPrivate
	}

	packageToCreate := &models.Package{
		Name:                 input.Name,
		Description:          input.Description,
		GroupID:              input.GroupID,
		RootGroupID:          rootGroupID,
		Kind:                 packageKind,
		Visibility:           visibility,
		CreatedBy:            caller.GetSubject(),
		AllowMutableVersions: input.AllowMutableVersions,
	}

	if vErr := packageToCreate.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate package model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreatePackage: %v", txErr)
		}
	}()

	createdPackage, err := s.dbClient.Packages.CreatePackage(txContext, packageToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create package")
		return nil, err
	}

	// Get the number of packages in the group to check whether we just violated the limit.
	pkgsResult, err := s.dbClient.Packages.GetPackages(txContext, &db.GetPackagesInput{
		Filter: &db.PackageFilter{
			GroupID: &input.GroupID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to query packages in group")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext, limits.ResourceLimitPackagesPerGroup, pkgsResult.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "failed to check limit for packages per group")
		return nil, err
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &group.FullPath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetPackage,
		TargetID:      createdPackage.Metadata.ID,
	}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Created a package.",
		"packageID", createdPackage.Metadata.ID,
		"packageName", createdPackage.Name,
		"packageTRN", createdPackage.Metadata.TRN,
	)

	return createdPackage, nil
}

func (s *service) UpdatePackage(ctx context.Context, input *UpdatePackageInput) (*models.Package, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdatePackage")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, input.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.UpdatePackagePermission, auth.WithGroupID(pkg.GroupID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if input.Description != nil {
		pkg.Description = input.Description
	}
	if input.Visibility != nil {
		pkg.Visibility = *input.Visibility
	}
	if input.AllowMutableVersions != nil {
		pkg.AllowMutableVersions = *input.AllowMutableVersions
	}

	if vErr := pkg.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate package model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdatePackage: %v", txErr)
		}
	}()

	updatedPackage, err := s.dbClient.Packages.UpdatePackage(txContext, pkg)
	if err != nil {
		tracing.RecordError(span, err, "failed to update package")
		return nil, err
	}

	groupPath := pkg.GetGroupPath()
	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &groupPath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetPackage,
		TargetID:      updatedPackage.Metadata.ID,
	}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Updated a package.",
		"packageID", updatedPackage.Metadata.ID,
		"packageName", updatedPackage.Name,
		"packageTRN", updatedPackage.Metadata.TRN,
	)

	return updatedPackage, nil
}

// DeletePackage deletes a package.
func (s *service) DeletePackage(ctx context.Context, pkg *models.Package) error {
	ctx, span := tracer.Start(ctx, "svc.DeletePackage")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	if err = caller.RequirePermission(ctx, models.DeletePackagePermission, auth.WithGroupID(pkg.GroupID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeletePackage: %v", txErr)
		}
	}()

	if err = s.dbClient.Packages.DeletePackage(txContext, pkg); err != nil {
		tracing.RecordError(span, err, "failed to delete package")
		return err
	}

	groupPath := pkg.GetGroupPath()

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      pkg.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: pkg.Name,
				ID:   pkg.Metadata.ID,
				Type: string(models.TargetPackage),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a package.",
		"packageID", pkg.Metadata.ID,
		"packageName", pkg.Name,
		"packageTRN", pkg.Metadata.TRN,
	)

	return nil
}

func (s *service) GetPackageVersionByID(ctx context.Context, id string) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackageVersionByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	packageVersion, err := s.getPackageVersionByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package version by ID")
		return nil, err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, packageVersion.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return nil, err
	}

	if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return packageVersion, nil
}

func (s *service) GetPackageVersionByTRN(ctx context.Context, trn string) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackageVersionByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	packageVersion, err := s.dbClient.PackageVersions.GetPackageVersionByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package version by TRN")
		return nil, err
	}
	if packageVersion == nil {
		return nil, errors.New("package version with trn %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, packageVersion.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return nil, err
	}

	if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	return packageVersion, nil
}

// GetPackageVersionsByIDs returns the package versions with the given IDs. The owning package lookup
// and its view-permission check are deduplicated because a batch usually holds several versions of the
// same package.
func (s *service) GetPackageVersionsByIDs(ctx context.Context, ids []string) ([]models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackageVersionsByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.PackageVersions.GetPackageVersions(ctx, &db.GetPackageVersionsInput{
		Filter: &db.PackageVersionFilter{PackageVersionIDs: ids},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get package versions")
		return nil, err
	}

	checkedPackages := map[string]struct{}{}
	for i := range result.PackageVersions {
		packageID := result.PackageVersions[i].PackageID

		if _, checked := checkedPackages[packageID]; checked {
			continue
		}

		pkg, pErr := corepkg.GetPackageByID(ctx, s.dbClient, packageID)
		if pErr != nil {
			tracing.RecordError(span, pErr, "failed to get package by ID")
			return nil, pErr
		}

		if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		checkedPackages[packageID] = struct{}{}
	}

	return result.PackageVersions, nil
}

func (s *service) GetPackageVersions(ctx context.Context, input *GetPackageVersionsInput) (*db.PackageVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPackageVersions")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, input.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return nil, err
	}

	if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	dbInput := &db.GetPackageVersionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.PackageVersionFilter{
			PackageID:       &input.PackageID,
			Status:          input.Status,
			SemanticVersion: input.SemanticVersion,
			Latest:          input.Latest,
			Search:          input.Search,
		},
	}

	return s.dbClient.PackageVersions.GetPackageVersions(ctx, dbInput)
}

// ResolvePackageVersion resolves a package source and version constraint to the concrete uploaded
// package version that satisfies it, returning ENotFound when none does. Authorization is the
// package's own view check, applied by GetPackageByTRN.
func (s *service) ResolvePackageVersion(ctx context.Context, input *ResolvePackageVersionInput) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.ResolvePackageVersion")
	defer span.End()

	// authz handled in GetPackageByTRN
	pkg, err := s.GetPackageByTRN(ctx, trn.TypePackage.Build(input.PackageSource))
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by TRN")
		return nil, err
	}

	packageVersion, err := corepkg.ResolveVersionConstraint(ctx, s.dbClient, pkg.Metadata.ID, input.VersionConstraint)
	if err != nil {
		tracing.RecordError(span, err, "failed to resolve package version constraint")
		return nil, err
	}
	if packageVersion == nil {
		return nil, errors.New(
			"no uploaded version of package %s satisfies %s", input.PackageSource, versionConstraintLabel(input.VersionConstraint),
			errors.WithErrorCode(errors.ENotFound),
		)
	}

	return packageVersion, nil
}

func (s *service) CreatePackageVersion(ctx context.Context, input *CreatePackageVersionInput) (*models.PackageVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.CreatePackageVersion")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, input.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.UpdatePackagePermission, auth.WithGroupID(pkg.GroupID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	semVersion, err := version.NewSemver(input.SemanticVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to verify semantic version")
		return nil, errors.Wrap(err, "invalid semantic version", errors.WithErrorCode(errors.EInvalid))
	}

	versionsResp, err := s.dbClient.PackageVersions.GetPackageVersions(ctx, &db.GetPackageVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.PackageVersionFilter{
			PackageID: &input.PackageID,
			Latest:    ptr.Bool(true),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get package versions")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreatePackageVersion: %v", txErr)
		}
	}()

	isLatest := false
	if len(versionsResp.PackageVersions) > 0 {
		prevLatest := versionsResp.PackageVersions[0]
		prevSemVersion, sErr := version.NewSemver(prevLatest.SemanticVersion)
		if sErr != nil {
			tracing.RecordError(span, sErr, "semver validation failed")
			return nil, sErr
		}
		if semver.IsSemverGreaterThan(semVersion, prevSemVersion) {
			isLatest = true
			prevLatest.Latest = false
			if _, uErr := s.dbClient.PackageVersions.UpdatePackageVersion(txContext, &prevLatest); uErr != nil {
				tracing.RecordError(span, uErr, "failed to update package version")
				return nil, uErr
			}
		}
	} else {
		isLatest = true
	}

	versionToCreate := &models.PackageVersion{
		PackageID:       input.PackageID,
		SemanticVersion: semVersion.String(),
		Latest:          isLatest,
		SHASum:          input.SHASum,
		Status:          models.PackageVersionStatusPending,
		CreatedBy:       caller.GetSubject(),
	}
	if vErr := versionToCreate.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "invalid package version")
		return nil, vErr
	}

	packageVersion, err := s.dbClient.PackageVersions.CreatePackageVersion(txContext, versionToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create package version")
		return nil, err
	}

	// Get the number of versions of this package within the time period to check whether we just
	// violated the limit.
	newVersions, err := s.dbClient.PackageVersions.GetPackageVersions(txContext, &db.GetPackageVersionsInput{
		Filter: &db.PackageVersionFilter{
			TimeRangeStart: ptr.Time(packageVersion.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			PackageID:      &packageVersion.PackageID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get package's versions")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitVersionsPerPackagePerTimePeriod, newVersions.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "failed to check limit for package versions per time period")
		return nil, err
	}

	groupPath := pkg.GetGroupPath()
	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &groupPath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetPackageVersion,
		TargetID:      packageVersion.Metadata.ID,
	}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Created a package version.",
		"packageID", input.PackageID,
		"packageVersion", packageVersion.SemanticVersion,
		"packageVersionTRN", packageVersion.Metadata.TRN,
	)

	return packageVersion, nil
}

// DeletePackageVersion deletes a single version of a package. Deleting the latest version is allowed:
// the highest remaining semantic version is promoted to latest, matching the module registry. The
// uploaded package objects are not removed here — they are linked into object_store_refs by the FK
// this delete nulls out, so the janitor reclaims them.
func (s *service) DeletePackageVersion(ctx context.Context, packageVersion *models.PackageVersion) error {
	ctx, span := tracer.Start(ctx, "svc.DeletePackageVersion")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, packageVersion.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return err
	}

	// A version is a child of the package, so removing one is an update to the package rather than a
	// package deletion; this is the permission CreatePackageVersion requires.
	if err = caller.RequirePermission(ctx, models.UpdatePackagePermission, auth.WithGroupID(pkg.GroupID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	// Find the version that takes over as latest, if the one being deleted holds the flag.
	var newLatestVersion *models.PackageVersion
	if packageVersion.Latest {
		versionsResp, gErr := s.dbClient.PackageVersions.GetPackageVersions(ctx, &db.GetPackageVersionsInput{
			Filter: &db.PackageVersionFilter{
				PackageID: &packageVersion.PackageID,
			},
		})
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get package versions")
			return gErr
		}

		for _, v := range versionsResp.PackageVersions {
			vCopy := v

			if v.Metadata.ID == packageVersion.Metadata.ID {
				continue
			}

			currentSemVersion, cErr := version.NewSemver(vCopy.SemanticVersion)
			if cErr != nil {
				tracing.RecordError(span, cErr, "semver validation failed")
				return cErr
			}

			if newLatestVersion == nil {
				newLatestVersion = &vCopy
				continue
			}

			latestSemVersion, lErr := version.NewSemver(newLatestVersion.SemanticVersion)
			if lErr != nil {
				tracing.RecordError(span, lErr, "semver validation failed")
				return lErr
			}

			if semver.IsSemverGreaterThan(currentSemVersion, latestSemVersion) {
				newLatestVersion = &vCopy
			}
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for DeletePackageVersion: %v", txErr)
		}
	}()

	// The delete must land before the promotion: index_package_versions_on_latest is a unique partial
	// index on (package_id, latest) where latest is true, so two latest rows cannot coexist.
	if err = s.dbClient.PackageVersions.DeletePackageVersion(txContext, packageVersion); err != nil {
		tracing.RecordError(span, err, "failed to delete package version")
		return err
	}

	if newLatestVersion != nil {
		newLatestVersion.Latest = true
		if _, err = s.dbClient.PackageVersions.UpdatePackageVersion(txContext, newLatestVersion); err != nil {
			tracing.RecordError(span, err, "failed to update package version")
			return err
		}
	}

	groupPath := pkg.GetGroupPath()

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      pkg.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: packageVersion.SemanticVersion,
				ID:   packageVersion.Metadata.ID,
				Type: string(models.TargetPackageVersion),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a package version.",
		"packageID", pkg.Metadata.ID,
		"packageVersion", packageVersion.SemanticVersion,
		"packageVersionTRN", packageVersion.Metadata.TRN,
	)

	return nil
}

// byteCounter counts the bytes written through it. It rides along the upload's checksum tee so the
// package size is measured from the stream rather than requiring the whole package in memory.
type byteCounter struct {
	n int
}

func (c *byteCounter) Write(p []byte) (int, error) {
	c.n += len(p)
	return len(p), nil
}

func (s *service) UploadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadPackageVersion")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, packageVersion.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return err
	}

	if err = caller.RequirePermission(ctx, models.UpdatePackagePermission, auth.WithGroupID(pkg.GroupID)); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if packageVersion.Status == models.PackageVersionStatusUploadInProgress {
		return errors.New("package upload is already in progress", errors.WithErrorCode(errors.EConflict))
	}

	// A version that already has a package (uploaded, or errored from a prior attempt) is a
	// re-upload: it overwrites the stable object key in place to edit its contents without bumping
	// the version, staying in the uploaded state throughout (no availability window). A first upload
	// of a pending version verifies the streamed content against the checksum pinned at create time.
	//
	// Re-uploading is only permitted when the package opts into mutable versions; otherwise an
	// uploaded version is immutable and a re-upload is rejected.
	reupload := packageVersion.Status != models.PackageVersionStatusPending
	if reupload && !pkg.AllowMutableVersions {
		return errors.New("package version %s has already been uploaded and the package does not allow mutable versions",
			packageVersion.SemanticVersion, errors.WithErrorCode(errors.EConflict))
	}

	checksum := sha256.New()
	counter := &byteCounter{}
	teeReader := io.TeeReader(reader, io.MultiWriter(checksum, counter))

	// Upload to object storage before opening the DB transaction so the transaction isn't held open
	// for the duration of a potentially large upload. The tracked object store records a pending ref
	// before writing the object; that ref is linked to the package version inside the transaction
	// below so they commit atomically. If the version is never linked (e.g., the tx fails or content
	// verification fails), the janitor reclaims the orphaned object after its grace period.
	retainRef, objectStoreKey, err := s.store.UploadPackageVersion(ctx, packageVersion, teeReader)
	if err != nil {
		tracing.RecordError(span, err, "failed to upload package")
		return err
	}

	shaSum := hex.EncodeToString(checksum.Sum(nil))
	s.logger.WithContextFields(ctx).Infof("Uploaded package version with sha checksum %s", shaSum)

	uploadStartedAt := time.Now()
	packageVersion.UploadStartedTimestamp = &uploadStartedAt

	// A first upload of a pending version verifies the streamed content against the checksum pinned
	// at create time. On a mismatch the version is marked errored and its object is left unlinked for
	// the janitor to reclaim (we don't record the key on the version).
	if !reupload && shaSum != packageVersion.GetSHASumHex() {
		errorMsg := fmt.Sprintf("Expected checksum of %s does not match received checksum %s", packageVersion.GetSHASumHex(), shaSum)
		packageVersion.Status = models.PackageVersionStatusErrored
		packageVersion.Error = &errorMsg
		if _, err = s.dbClient.PackageVersions.UpdatePackageVersion(ctx, packageVersion); err != nil {
			tracing.RecordError(span, err, "failed to set package version status to errored")
			s.logger.WithContextFields(ctx).Errorf("failed to set package version status to errored %v", err)
		}
		return nil
	}

	packageVersion.ObjectStoreKey = &objectStoreKey
	packageVersion.Status = models.PackageVersionStatusUploaded
	packageVersion.Error = nil
	// Set unconditionally: a re-upload replaces the object in place, so it adopts the new size just as
	// it adopts the new checksum below.
	packageVersion.Size = counter.n
	// On a re-upload the content can change under a fixed semantic version. Capture the prior checksum
	// (still on the model before we overwrite it) and whether it actually changed, so a content change
	// leaves a tamper trail below: a policy pinning this package by version but not by digest would
	// otherwise silently begin evaluating different Rego after a re-upload.
	priorSHASumHex := packageVersion.GetSHASumHex()
	contentChanged := false
	if reupload {
		// A re-upload overwrites the object in place and adopts the freshly computed checksum.
		contentChanged = shaSum != priorSHASumHex
		packageVersion.SHASum = checksum.Sum(nil)
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	updatedPackageVersion, err := s.dbClient.PackageVersions.UpdatePackageVersion(txContext, packageVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to update package version")
		return err
	}

	if err = retainRef(txContext, updatedPackageVersion.Metadata.ID); err != nil {
		tracing.RecordError(span, err, "failed to link package version object store ref")
		return err
	}

	// Record a durable tamper trail when a re-upload changed the version's content. The activity event
	// commits atomically with the version update, so the content change can never land without it.
	if contentChanged {
		groupPath := pkg.GetGroupPath()
		if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetPackageVersion,
			TargetID:      updatedPackageVersion.Metadata.ID,
		}); err != nil {
			tracing.RecordError(span, err, "failed to create package version re-upload activity event")
			return err
		}
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	if reupload {
		if contentChanged {
			// A re-upload that changes the checksum silently alters what a version-pinned (not
			// digest-pinned) policy evaluates, so surface it loudly in addition to the activity event.
			s.logger.WithContextFields(ctx).Warnf(
				"Re-upload changed the content of package version %s (%s %s): checksum %s -> %s by %s",
				updatedPackageVersion.Metadata.ID, updatedPackageVersion.PackageID,
				updatedPackageVersion.SemanticVersion, priorSHASumHex, shaSum, caller.GetSubject())
		}
		s.logger.WithContextFields(ctx).Infow("Re-uploaded a package version.",
			"packageVersionID", updatedPackageVersion.Metadata.ID,
			"packageID", updatedPackageVersion.PackageID,
			"packageVersionTRN", updatedPackageVersion.Metadata.TRN,
		)
	} else {
		s.logger.WithContextFields(ctx).Infow("Uploaded a package version.",
			"packageVersionID", updatedPackageVersion.Metadata.ID,
			"packageID", updatedPackageVersion.PackageID,
			"packageVersionTRN", updatedPackageVersion.Metadata.TRN,
		)
	}

	return nil
}

func (s *service) DownloadPackageVersion(ctx context.Context, packageVersion *models.PackageVersion) (string, error) {
	ctx, span := tracer.Start(ctx, "svc.DownloadPackageVersion")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return "", err
	}

	pkg, err := corepkg.GetPackageByID(ctx, s.dbClient, packageVersion.PackageID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package by ID")
		return "", err
	}

	if err = s.authorizeViewPackage(ctx, caller, pkg); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return "", err
	}

	downloadURL, err := s.store.GetPackageVersionPresignedURL(ctx, packageVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to get package presigned URL")
		return "", err
	}

	return downloadURL, nil
}

func (s *service) getPackageVersionByID(ctx context.Context, id string) (*models.PackageVersion, error) {
	packageVersion, err := s.dbClient.PackageVersions.GetPackageVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if packageVersion == nil {
		return nil, errors.New("package version with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}
	return packageVersion, nil
}

// versionConstraintLabel renders a version constraint for an error message, naming the empty constraint
// (which resolves to the latest uploaded version) rather than showing nothing.
func versionConstraintLabel(constraint string) string {
	if constraint == "" {
		return "latest"
	}
	return constraint
}
