// Package providermirror implements the Terraform's
// Provider Network Mirror Protocol.
package providermirror

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/provider"
)

const (
	providerPlatformPackageMaxSizeLimit = 1024 * 1024 * 256 // ~ 244 MiB or 256 MB in bytes.
)

// GetProviderVersionMirrorsInput is the input for listing TerraformProviderVersionMirrors.
type GetProviderVersionMirrorsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TerraformProviderVersionMirrorSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// NamespacePath is the namespace to return provider version mirrors for.
	NamespacePath string
	// Search filters by hostname/namespace/type.
	Search *string
}

// GetProviderVersionMirrorByAddressInput is the input for GetProviderVersionMirrorByAddress.
type GetProviderVersionMirrorByAddressInput struct {
	RegistryHostname  string
	RegistryNamespace string
	Type              string
	SemanticVersion   string
	GroupPath         string
}

// CreateProviderVersionMirrorInput is the input for CreateProviderVersionMirror.
type CreateProviderVersionMirrorInput struct {
	Type              string
	RegistryNamespace string
	RegistryHostname  string
	SemanticVersion   string
	GroupPath         string
	RegistryToken     *string
}

// DeleteProviderVersionMirrorInput is the input for DeleteTerraformProviderVersionMirror.
type DeleteProviderVersionMirrorInput struct {
	VersionMirror *models.TerraformProviderVersionMirror
	Force         bool
}

// GetProviderPlatformMirrorsInput is the input listing provider platform mirrors.
type GetProviderPlatformMirrorsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TerraformProviderPlatformMirrorSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// OS is the OS to filter on.
	OS *string
	// Architecture is the architecture to filter on.
	Architecture *string
	// VersionMirrorID is the ID of the version mirror to filter on. Always required.
	VersionMirrorID string
}

// DeleteProviderPlatformMirrorInput is the input for DeleteProviderPlatformMirror.
type DeleteProviderPlatformMirrorInput struct {
	PlatformMirror *models.TerraformProviderPlatformMirror
}

// UploadInstallationPackageInput is the input for UploadInstallationPackage.
type UploadInstallationPackageInput struct {
	Data            io.Reader
	VersionMirrorID string
	OS              string
	Architecture    string
}

// GetAvailableProviderVersionsInput represents the input for GetAvailableProviderVersions.
type GetAvailableProviderVersionsInput struct {
	Type              string
	RegistryNamespace string
	RegistryHostname  string
	GroupPath         string
}

// GetAvailableInstallationPackagesInput represents the input for GetAvailableInstallationPackages.
type GetAvailableInstallationPackagesInput struct {
	Type              string
	RegistryNamespace string
	RegistryHostname  string
	SemanticVersion   string
	GroupPath         string
}

// GetInstallationPackageInput represents the input for GetInstallationPackage.
type GetInstallationPackageInput struct {
	Type              string
	RegistryNamespace string
	RegistryHostname  string
	SemanticVersion   string
	GroupPath         string
	OS                string
	Arch              string
}

// InstallationPackage represents a single platform's installation package info.
type InstallationPackage struct {
	URL    string
	Hashes []string
}

// Service implements all the Terraform provider mirror functionality.
type Service interface {
	GetProviderVersionMirrorByID(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error)
	GetProviderVersionMirrorByTRN(ctx context.Context, trn string) (*models.TerraformProviderVersionMirror, error)
	GetProviderVersionMirrorsByIDs(ctx context.Context, idList []string) ([]models.TerraformProviderVersionMirror, error)
	GetProviderVersionMirrors(ctx context.Context, input *GetProviderVersionMirrorsInput) (*db.ProviderVersionMirrorsResult, error)
	CreateProviderVersionMirror(ctx context.Context, input *CreateProviderVersionMirrorInput) (*models.TerraformProviderVersionMirror, error)
	DeleteProviderVersionMirror(ctx context.Context, input *DeleteProviderVersionMirrorInput) error
	GetProviderPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error)
	GetProviderPlatformMirrorByTRN(ctx context.Context, trn string) (*models.TerraformProviderPlatformMirror, error)
	GetProviderPlatformMirrors(ctx context.Context, input *GetProviderPlatformMirrorsInput) (*db.ProviderPlatformMirrorsResult, error)
	DeleteProviderPlatformMirror(ctx context.Context, input *DeleteProviderPlatformMirrorInput) error
	UploadInstallationPackage(ctx context.Context, input *UploadInstallationPackageInput) error
	GetAvailableProviderVersions(ctx context.Context, input *GetAvailableProviderVersionsInput) (map[string]struct{}, error)
	GetAvailableInstallationPackages(ctx context.Context, input *GetAvailableInstallationPackagesInput) (map[string]any, error)
	GetInstallationPackage(ctx context.Context, input *GetInstallationPackageInput) (*InstallationPackage, error)
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	registryClient  provider.RegistryProtocol
	limitChecker    limits.LimitChecker
	activityService activityevent.Service
	mirrorStore     TerraformProviderMirrorStore
}

// NewService creates a new Service.
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	registryClient provider.RegistryProtocol,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
	mirrorStore TerraformProviderMirrorStore,
) Service {
	return &service{
		logger:          logger,
		dbClient:        dbClient,
		registryClient:  registryClient,
		limitChecker:    limitChecker,
		activityService: activityService,
		mirrorStore:     mirrorStore,
	}
}

func (s *service) GetProviderVersionMirrorByID(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionMirrorByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	versionMirror, err := s.getVersionMirrorByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return nil, err
	}

	// Provider mirror is available to anyone within a group hierarchy.
	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	return versionMirror, nil
}

func (s *service) GetProviderVersionMirrorByTRN(ctx context.Context, trn string) (*models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionMirrorByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	versionMirror, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrorByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return nil, err
	}

	if versionMirror == nil {
		tracing.RecordError(span, nil, "provider version mirror not found")
		return nil, errors.New("provider version mirror not found", errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	return versionMirror, nil
}

func (s *service) GetProviderVersionMirrorsByIDs(ctx context.Context, idList []string) ([]models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionMirrorsByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	result, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, &db.GetProviderVersionMirrorsInput{
		Filter: &db.TerraformProviderVersionMirrorFilter{
			VersionMirrorIDs: idList,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirrors")
		return nil, err
	}

	for _, m := range result.VersionMirrors {
		err := caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(m.GroupID))
		if err != nil {
			return nil, err
		}
	}

	return result.VersionMirrors, nil
}

func (s *service) GetProviderVersionMirrors(ctx context.Context, input *GetProviderVersionMirrorsInput) (*db.ProviderVersionMirrorsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionMirrors")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	dbInput := &db.GetProviderVersionMirrorsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformProviderVersionMirrorFilter{
			Search: input.Search,
			// Version mirrors are always associated with a root group, so they must be inherited.
			NamespacePaths: utils.ExpandPath(input.NamespacePath),
		},
	}

	result, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirrors")
		return nil, err
	}

	return result, nil
}

func (s *service) CreateProviderVersionMirror(ctx context.Context, input *CreateProviderVersionMirrorInput) (*models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateProviderVersionMirror")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateTerraformProviderMirrorPermission, auth.WithNamespacePath(input.GroupPath))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(input.GroupPath))
	if err != nil {
		tracing.RecordError(span, err, "group not found")
		return nil, err
	}

	if group == nil {
		tracing.RecordError(span, nil, "group not found")
		return nil, errors.New("group %s not found", input.GroupPath, errors.WithErrorCode(errors.ENotFound))
	}

	if group.ParentID != "" {
		tracing.RecordError(span, nil, "terraform provider version mirrors can only be created in a top-level group")
		return nil, errors.New("terraform provider version mirrors can only be created in a top-level group", errors.WithErrorCode(errors.EInvalid))
	}

	prov, err := provider.NewProvider(input.RegistryHostname, input.RegistryNamespace, input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider")
		return nil, errors.Wrap(err, "invalid provider", errors.WithErrorCode(errors.EInvalid))
	}

	wantVersion, err := versions.ParseVersion(input.SemanticVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider version")
		return nil, errors.Wrap(err, "invalid provider version", errors.WithErrorCode(errors.EInvalid))
	}

	// Build request options with token if available.
	var reqOpts []provider.RequestOption
	if input.RegistryToken != nil {
		reqOpts = append(reqOpts, provider.WithToken(*input.RegistryToken))
	}

	// We will attempt to list all the supported versions for this provider, so we can find the
	// platforms it supports. It isn't possible to know this otherwise.
	availableVersions, err := s.registryClient.ListVersions(ctx, prov, reqOpts...)
	if err != nil {
		tracing.RecordError(span, err, "failed to list available provider versions")
		return nil, errors.Wrap(err, "failed to list available provider versions", errors.WithErrorCode(errors.ENotFound))
	}

	// Find a platform the provider supports. We only need one for our purposes.
	supportedPlatform, err := provider.GetPlatformForVersion(wantVersion.String(), availableVersions)
	if err != nil {
		tracing.RecordError(span, err, "unsupported provider version")
		return nil, errors.Wrap(err, "unsupported version %s for provider %s", wantVersion, prov, errors.WithErrorCode(errors.EInvalid))
	}

	// Now, find the sha sums, signature URLs and the associated GPG key(s) by arbitrarily using
	// one of the supported platforms. The sha sums and signature URLs should be independent of
	// of the platform being queried for.
	packageInfo, err := s.registryClient.GetPackageInfo(ctx, prov, wantVersion.String(), supportedPlatform.OS, supportedPlatform.Arch, reqOpts...)
	if err != nil {
		tracing.RecordError(span, err, "failed to find provider package")
		return nil, errors.Wrap(err, "could not find package at provider registry API", errors.WithErrorCode(errors.ENotFound))
	}

	// Retrieve and verify the checksums from the response.
	digests, err := s.registryClient.GetChecksums(ctx, packageInfo)
	if err != nil {
		tracing.RecordError(span, err, "failed to get checksums")
		return nil, fmt.Errorf("failed to get checksums: %w", err)
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateProviderVersionMirror: %v", txErr)
		}
	}()

	toCreate := &models.TerraformProviderVersionMirror{
		CreatedBy:         caller.GetSubject(),
		Type:              prov.Type,
		RegistryNamespace: prov.Namespace,
		RegistryHostname:  prov.Hostname,
		SemanticVersion:   wantVersion.String(),
		Digests:           digests,
		GroupID:           group.Metadata.ID,
	}

	created, err := s.dbClient.TerraformProviderVersionMirrors.CreateVersionMirror(txContext, toCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create provider version mirror")
		return nil, err
	}

	newMirrors, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(txContext, &db.GetProviderVersionMirrorsInput{
		Filter: &db.TerraformProviderVersionMirrorFilter{
			GroupID: &group.Metadata.ID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get a group's provider version mirrors")
		return nil, err
	}

	if err = s.limitChecker.CheckLimit(
		txContext,
		limits.ResourceLimitTerraformProviderVersionMirrorsPerGroup,
		newMirrors.PageInfo.TotalCount,
	); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformProviderVersionMirror,
			TargetID:      created.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.WithContextFields(ctx).Infow("Created a terraform provider version mirror.",
		"groupPath", input.GroupPath,
		"versionMirrorID", created.Metadata.ID,
	)

	return created, nil
}

func (s *service) DeleteProviderVersionMirror(ctx context.Context, input *DeleteProviderVersionMirrorInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteProviderVersionMirror")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteTerraformProviderMirrorPermission, auth.WithGroupID(input.VersionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return err
	}

	// Warn the end user if they try to delete a version mirror
	// that currently mirrors binaries for platforms.
	if !input.Force {
		result, pErr := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, &db.GetProviderPlatformMirrorsInput{
			PaginationOptions: &pagination.Options{
				First: ptr.Int32(0), // We just want the count.
			},
			Filter: &db.TerraformProviderPlatformMirrorFilter{
				VersionMirrorID: &input.VersionMirror.Metadata.ID,
			},
		})
		if pErr != nil {
			tracing.RecordError(span, pErr, "failed to get platform mirrors")
			return pErr
		}

		if result.PageInfo.TotalCount > 0 {
			tracing.RecordError(span, nil,
				"This provider version mirror can't be deleted because it currently mirrors %d platform(s). "+
					"Setting force to true will automatically remove all mirrored Terraform provider platform mirrors. ", result.PageInfo.TotalCount,
			)
			return errors.New(
				"This provider version mirror can't be deleted because it currently mirrors %d platform(s). "+
					"Setting force to true will automatically remove all mirrored Terraform provider platform mirrors. ", result.PageInfo.TotalCount,
				errors.WithErrorCode(errors.EConflict),
			)
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeleteProviderVersionMirror: %v", txErr)
		}
	}()

	if err = s.dbClient.TerraformProviderVersionMirrors.DeleteVersionMirror(txContext, input.VersionMirror); err != nil {
		tracing.RecordError(span, err, "failed to delete provider version mirror")
		return err
	}

	// Find the group so, we can get its path.
	group, err := s.dbClient.Groups.GetGroupByID(txContext, input.VersionMirror.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group associated with provider version mirror")
		return err
	}

	if group == nil {
		tracing.RecordError(span, nil, "failed to get group associated with version mirror")
		return fmt.Errorf("failed to get group associated with version mirror: %w", err)
	}

	provider := &provider.Provider{
		Hostname:  input.VersionMirror.RegistryHostname,
		Namespace: input.VersionMirror.RegistryNamespace,
		Type:      input.VersionMirror.Type,
	}

	providerName := fmt.Sprintf("%s/%s", provider, input.VersionMirror.SemanticVersion)
	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: providerName,
				ID:   input.VersionMirror.Metadata.ID,
				Type: string(models.TargetTerraformProviderVersionMirror),
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a terraform provider version mirror.",
		"groupID", input.VersionMirror.GroupID,
		"providerName", providerName,
		"semver", input.VersionMirror.SemanticVersion,
	)

	return nil
}

func (s *service) GetProviderPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatformMirrorByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	platformMirror, err := s.getPlatformMirrorByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform mirror")
		return nil, err
	}

	versionMirror, err := s.getVersionMirrorByID(ctx, platformMirror.VersionMirrorID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	return platformMirror, nil
}

func (s *service) GetProviderPlatformMirrorByTRN(ctx context.Context, trn string) (*models.TerraformProviderPlatformMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatformMirrorByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	platformMirror, err := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrorByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform mirror")
		return nil, err
	}

	if platformMirror == nil {
		tracing.RecordError(span, nil, "provider platform mirror not found")
		return nil, errors.New("provider platform mirror not found", errors.WithErrorCode(errors.ENotFound))
	}

	versionMirror, err := s.getVersionMirrorByID(ctx, platformMirror.VersionMirrorID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	return platformMirror, nil
}

func (s *service) GetProviderPlatformMirrors(ctx context.Context, input *GetProviderPlatformMirrorsInput) (*db.ProviderPlatformMirrorsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatformMirrors")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	versionMirror, err := s.getVersionMirrorByID(ctx, input.VersionMirrorID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	dbInput := &db.GetProviderPlatformMirrorsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformProviderPlatformMirrorFilter{
			OS:              input.OS,
			Architecture:    input.Architecture,
			VersionMirrorID: &versionMirror.Metadata.ID,
		},
	}

	result, err := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get platform mirrors")
		return nil, err
	}

	return result, nil
}

func (s *service) DeleteProviderPlatformMirror(ctx context.Context, input *DeleteProviderPlatformMirrorInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteProviderPlatformMirror")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	versionMirror, err := s.getVersionMirrorByID(ctx, input.PlatformMirror.VersionMirrorID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteTerraformProviderMirrorPermission, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return err
	}

	if err = s.dbClient.TerraformProviderPlatformMirrors.DeletePlatformMirror(ctx, input.PlatformMirror); err != nil {
		tracing.RecordError(span, err, "failed to delete provider platform mirror")
		return err
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a terraform provider platform mirror.",
		"groupID", versionMirror.GroupID,
		"versionMirrorID", versionMirror.Metadata.ID,
		"os", input.PlatformMirror.OS,
		"architecture", input.PlatformMirror.Architecture,
	)

	return nil
}

func (s *service) UploadInstallationPackage(ctx context.Context, input *UploadInstallationPackageInput) error {
	ctx, span := tracer.Start(ctx, "svc.UploadInstallationPackage")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	versionMirror, err := s.getVersionMirrorByID(ctx, input.VersionMirrorID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirror")
		return err
	}

	err = caller.RequirePermission(ctx, models.CreateTerraformProviderMirrorPermission, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return err
	}

	// Make sure the package does not already exist since we don't need to do anything if it does.
	result, err := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, &db.GetProviderPlatformMirrorsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
		Filter: &db.TerraformProviderPlatformMirrorFilter{
			VersionMirrorID: &input.VersionMirrorID,
			OS:              &input.OS,
			Architecture:    &input.Architecture,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform mirrors")
		return err
	}

	if result.PageInfo.TotalCount > 0 {
		tracing.RecordError(span, nil, "provider platform package is already mirrored")
		return errors.New("provider platform package is already mirrored", errors.WithErrorCode(errors.EConflict))
	}

	digestKey := provider.GetPackageName(versionMirror.Type, versionMirror.SemanticVersion, input.OS, input.Architecture)

	// No point in continuing if we don't have a checksum for the package.
	expectDigest, ok := versionMirror.Digests[digestKey]
	if !ok {
		tracing.RecordError(span, nil, "no checksum available for provider package %s", digestKey)
		return errors.New("no checksum available for provider package %s", digestKey)
	}

	checksum := sha256.New()
	teeReader := io.TeeReader(
		io.LimitReader(input.Data, providerPlatformPackageMaxSizeLimit),
		checksum,
	)

	packageFile, err := os.CreateTemp("", "terraform-provider-package-*.zip")
	if err != nil {
		tracing.RecordError(span, err, "failed to create temporary package file")
		return err
	}
	defer os.Remove(packageFile.Name())
	defer packageFile.Close()

	if _, err = io.Copy(packageFile, teeReader); err != nil {
		tracing.RecordError(span, err, "failed to save uploaded provider package file to disk")
		return fmt.Errorf("failed to save uploaded provider package file to disk: %w", err)
	}

	calculatedSum := checksum.Sum(nil)
	if !bytes.Equal(expectDigest, calculatedSum) {
		tracing.RecordError(span, nil, "checksum of the uploaded provider platform package %x does not match the expected checksum %x", calculatedSum, expectDigest)
		return errors.New("checksum of the uploaded provider platform package %x does not match the expected checksum %x", calculatedSum, expectDigest, errors.WithErrorCode(errors.EInvalid))
	}

	// Seek back to beginning for upload
	if _, err = packageFile.Seek(0, io.SeekStart); err != nil {
		tracing.RecordError(span, err, "failed to seek package file")
		return fmt.Errorf("failed to seek package file: %w", err)
	}

	// Start transaction for DB operations and upload
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UploadInstallationPackage: %v", txErr)
		}
	}()

	// Create DB records first before uploading
	platformMirror, err := s.dbClient.TerraformProviderPlatformMirrors.CreatePlatformMirror(txContext, &models.TerraformProviderPlatformMirror{
		VersionMirrorID: versionMirror.Metadata.ID,
		OS:              input.OS,
		Architecture:    input.Architecture,
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create provider platform mirror")
		return errors.Wrap(err, "failed to create provider platform mirror")
	}

	if err = s.mirrorStore.UploadProviderPlatformPackage(txContext, platformMirror.Metadata.ID, packageFile); err != nil {
		tracing.RecordError(span, err, "failed to upload provider platform package to object store")
		return fmt.Errorf("failed to upload provider platform package to object store: %w", err)
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return err
	}

	return nil
}

// Implements: https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol#list-available-versions
// Returns a map of strings to structs based on Terraform specification.
func (s *service) GetAvailableProviderVersions(ctx context.Context, input *GetAvailableProviderVersionsInput) (map[string]struct{}, error) {
	ctx, span := tracer.Start(ctx, "svc.GetAvailableProviderVersions")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	group, err := s.getGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	prov, err := provider.NewProvider(input.RegistryHostname, input.RegistryNamespace, input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider")
		return nil, errors.Wrap(err, "invalid provider", errors.WithErrorCode(errors.EInvalid))
	}

	// Only return versions with packages.
	sort := db.TerraformProviderVersionMirrorSortableFieldCreatedAtAsc
	result, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, &db.GetProviderVersionMirrorsInput{
		Sort: &sort,
		Filter: &db.TerraformProviderVersionMirrorFilter{
			GroupID:           &group.Metadata.ID,
			RegistryHostname:  &prov.Hostname,
			RegistryNamespace: &prov.Namespace,
			Type:              &prov.Type,
			HasPackages:       ptr.Bool(true),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirrors")
		return nil, err
	}

	// Per Terraform docs, must return a ENotFound when we have no mirrored provider versions.
	if result.PageInfo.TotalCount == 0 {
		tracing.RecordError(span, nil, "no versions are currently mirrored for Terraform provider %s", prov)
		return nil, errors.New("no versions are currently mirrored for Terraform provider %s", prov, errors.WithErrorCode(errors.ENotFound))
	}

	// Must convert to a map here as needed by Terraform CLI.
	versionsMap := make(map[string]struct{}, result.PageInfo.TotalCount)
	for _, v := range result.VersionMirrors {
		versionsMap[v.SemanticVersion] = struct{}{}
	}

	return versionsMap, nil
}

// Implements: https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol#list-available-installation-packages
func (s *service) GetAvailableInstallationPackages(ctx context.Context, input *GetAvailableInstallationPackagesInput) (map[string]any, error) {
	ctx, span := tracer.Start(ctx, "svc.GetAvailableInstallationPackages")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	group, err := s.getGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	prov, err := provider.NewProvider(input.RegistryHostname, input.RegistryNamespace, input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider")
		return nil, errors.Wrap(err, "invalid provider", errors.WithErrorCode(errors.EInvalid))
	}

	// Find the version mirror first.
	versionsResult, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, &db.GetProviderVersionMirrorsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.TerraformProviderVersionMirrorFilter{
			GroupID:           &group.Metadata.ID,
			RegistryHostname:  &prov.Hostname,
			RegistryNamespace: &prov.Namespace,
			Type:              &prov.Type,
			SemanticVersion:   &input.SemanticVersion,
			HasPackages:       ptr.Bool(true),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get version mirror")
		return nil, err
	}

	if versionsResult.PageInfo.TotalCount == 0 {
		tracing.RecordError(span, nil, "version %s is currently not mirrored for Terraform provider %s", input.SemanticVersion, prov)
		return nil, errors.New("version %s is currently not mirrored for Terraform provider %s", input.SemanticVersion, prov, errors.WithErrorCode(errors.ENotFound))
	}

	versionMirror := versionsResult.VersionMirrors[0]

	result, err := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrors(ctx, &db.GetProviderPlatformMirrorsInput{
		Filter: &db.TerraformProviderPlatformMirrorFilter{
			VersionMirrorID: &versionMirror.Metadata.ID,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get platform mirrors")
		return nil, err
	}

	if result.PageInfo.TotalCount == 0 {
		tracing.RecordError(span, nil, "no installation packages are currently mirrored for Terraform provider %s", prov)
		return nil, errors.New("no installation packages are currently mirrored for Terraform provider %s", prov, errors.WithErrorCode(errors.ENotFound))
	}

	// Build the list of supported packages.
	supportedPackages, err := s.buildSupportedPackages(ctx, &versionMirror, result.PlatformMirrors)
	if err != nil {
		tracing.RecordError(span, err, "failed to build supported packages")
		return nil, err
	}

	return supportedPackages, nil
}

func (s *service) GetInstallationPackage(ctx context.Context, input *GetInstallationPackageInput) (*InstallationPackage, error) {
	ctx, span := tracer.Start(ctx, "svc.GetInstallationPackage")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	group, err := s.getGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderMirrorModelType, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	// Build TRN for version mirror lookup.
	versionTRN := types.TerraformProviderVersionMirrorModelType.BuildTRN(
		input.GroupPath, input.RegistryHostname, input.RegistryNamespace, input.Type, input.SemanticVersion,
	)

	versionMirror, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrorByTRN(ctx, versionTRN)
	if err != nil {
		tracing.RecordError(span, err, "failed to get version mirror")
		return nil, err
	}
	if versionMirror == nil {
		return nil, errors.New("version %s is not mirrored for provider %s/%s/%s", input.SemanticVersion, input.RegistryHostname, input.RegistryNamespace, input.Type, errors.WithErrorCode(errors.ENotFound))
	}

	// Build TRN for platform mirror lookup.
	platformTRN := types.TerraformProviderPlatformMirrorModelType.BuildTRN(
		input.GroupPath, input.RegistryHostname, input.RegistryNamespace, input.Type, input.SemanticVersion, input.OS, input.Arch,
	)

	platformMirror, err := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrorByTRN(ctx, platformTRN)
	if err != nil {
		tracing.RecordError(span, err, "failed to get platform mirror")
		return nil, err
	}
	if platformMirror == nil {
		return nil, errors.New("platform %s_%s is not mirrored", input.OS, input.Arch, errors.WithErrorCode(errors.ENotFound))
	}

	digestKey := provider.GetPackageName(versionMirror.Type, versionMirror.SemanticVersion, platformMirror.OS, platformMirror.Architecture)
	hash, ok := provider.Checksums(versionMirror.Digests).GetZipHash(digestKey)
	if !ok {
		return nil, fmt.Errorf("digest not found for provider package: %s", digestKey)
	}

	presignedURL, err := s.mirrorStore.GetProviderPlatformPackagePresignedURL(ctx, platformMirror.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get presigned URL")
		return nil, err
	}

	return &InstallationPackage{
		URL:    presignedURL,
		Hashes: []string{hash},
	}, nil
}

// buildSupportedPackages builds a map of supported Terraform packages for each
// platform complete with the authentication token for the download endpoint.
func (s *service) buildSupportedPackages(
	ctx context.Context,
	versionMirror *models.TerraformProviderVersionMirror,
	platformMirrors []models.TerraformProviderPlatformMirror,
) (map[string]any, error) {
	supportedPackages := make(map[string]any, len(platformMirrors))

	for _, mirror := range platformMirrors {
		digestKey := provider.GetPackageName(versionMirror.Type, versionMirror.SemanticVersion, mirror.OS, mirror.Architecture)

		hash, ok := provider.Checksums(versionMirror.Digests).GetZipHash(digestKey)
		if !ok {
			// Shouldn't happen.
			return nil, fmt.Errorf("failed to get digest for provider package: %s", digestKey)
		}

		// Get the presignedURL for downloading this provider package.
		presignedURL, err := s.mirrorStore.GetProviderPlatformPackagePresignedURL(ctx, mirror.Metadata.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get provider platform package presigned URL: %w", err)
		}

		// Use form "os_arch" as map key which is what Terraform CLI expects.
		key := fmt.Sprintf("%s_%s", mirror.OS, mirror.Architecture)

		supportedPackages[key] = map[string]any{
			"url":    presignedURL,
			"hashes": []string{hash},
		}
	}

	return supportedPackages, nil
}

func (s *service) getVersionMirrorByID(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error) {
	versionMirror, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrorByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if versionMirror == nil {
		return nil, errors.New("provider version mirror not found", errors.WithErrorCode(errors.ENotFound))
	}

	return versionMirror, nil
}

func (s *service) getPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error) {
	platformMirror, err := s.dbClient.TerraformProviderPlatformMirrors.GetPlatformMirrorByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if platformMirror == nil {
		return nil, errors.New("terraform provider platform mirror not found", errors.WithErrorCode(errors.ENotFound))
	}

	return platformMirror, nil
}

func (s *service) getGroupByFullPath(ctx context.Context, path string) (*models.Group, error) {
	group, err := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(path))
	if err != nil {
		return nil, err
	}

	if group == nil {
		return nil, errors.New("group not found", errors.WithErrorCode(errors.ENotFound))
	}

	return group, nil
}
