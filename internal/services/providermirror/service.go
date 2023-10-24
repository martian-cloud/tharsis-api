// Package providermirror implements the Terraform's
// Provider Network Mirror Protocol.
package providermirror

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/apparentlymart/go-versions/versions"
	"github.com/aws/smithy-go/ptr"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
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
	// IncludeInherited includes inherited provider version mirrors in the result.
	IncludeInherited bool
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

// listVersionsResponse is the response returned from the Terraform Registry API
// when querying for supported versions for a provider.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#list-available-versions
type listVersionsResponse struct {
	Versions []struct {
		Version   string `json:"version"`
		Platforms []struct {
			OS   string `json:"os"`
			Arch string `json:"arch"`
		} `json:"platforms"`
	} `json:"versions"`
	Warnings []string `json:"warnings"`
}

// packageQueryResponse is the response returned when querying for a particular
// installation package. It is used to find the SHA256SUMS, SHA256SUMS.sig files
// and the associated key files needed to verify their authenticity.
// https://developer.hashicorp.com/terraform/internals/provider-registry-protocol#find-a-provider-package
type packageQueryResponse struct {
	SHASumsURL          string `json:"shasums_url"`
	SHASumsSignatureURL string `json:"shasums_signature_url"`
	SigningKeys         struct {
		GPGPublicKeys []struct {
			ASCIIArmor string `json:"ascii_armor"`
		} `json:"gpg_public_keys"`
	} `json:"signing_keys"`
}

// Service implements all the Terraform provider mirror functionality.
type Service interface {
	GetProviderVersionMirrorByID(ctx context.Context, id string) (*models.TerraformProviderVersionMirror, error)
	GetProviderVersionMirrorByAddress(ctx context.Context, input *GetProviderVersionMirrorByAddressInput) (*models.TerraformProviderVersionMirror, error)
	GetProviderVersionMirrorsByIDs(ctx context.Context, idList []string) ([]models.TerraformProviderVersionMirror, error)
	GetProviderVersionMirrors(ctx context.Context, input *GetProviderVersionMirrorsInput) (*db.ProviderVersionMirrorsResult, error)
	CreateProviderVersionMirror(ctx context.Context, input *CreateProviderVersionMirrorInput) (*models.TerraformProviderVersionMirror, error)
	DeleteProviderVersionMirror(ctx context.Context, input *DeleteProviderVersionMirrorInput) error
	GetProviderPlatformMirrorByID(ctx context.Context, id string) (*models.TerraformProviderPlatformMirror, error)
	GetProviderPlatformMirrors(ctx context.Context, input *GetProviderPlatformMirrorsInput) (*db.ProviderPlatformMirrorsResult, error)
	DeleteProviderPlatformMirror(ctx context.Context, input *DeleteProviderPlatformMirrorInput) error
	UploadInstallationPackage(ctx context.Context, input *UploadInstallationPackageInput) error
	GetAvailableProviderVersions(ctx context.Context, input *GetAvailableProviderVersionsInput) (map[string]struct{}, error)
	GetAvailableInstallationPackages(ctx context.Context, input *GetAvailableInstallationPackagesInput) (map[string]any, error)
}

// serviceDiscoverer is an interface meant to facilitate in easier testing using the disco package.
type serviceDiscoverer interface {
	DiscoverServiceURL(hostname svchost.Hostname, serviceID string) (*url.URL, error)
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	httpClient      *http.Client
	discovery       serviceDiscoverer
	limitChecker    limits.LimitChecker
	activityService activityevent.Service
	mirrorStore     TerraformProviderMirrorStore
}

// NewService creates a new Service.
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	httpClient *http.Client,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
	mirrorStore TerraformProviderMirrorStore,
) Service {
	return &service{
		logger:          logger,
		dbClient:        dbClient,
		httpClient:      httpClient,
		limitChecker:    limitChecker,
		activityService: activityService,
		mirrorStore:     mirrorStore,
		discovery:       disco.New(),
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
	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	return versionMirror, nil
}

func (s *service) GetProviderVersionMirrorByAddress(ctx context.Context, input *GetProviderVersionMirrorByAddressInput) (*models.TerraformProviderVersionMirror, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionMirrorByAddress")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithNamespacePath(input.GroupPath))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	group, err := s.getGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		tracing.RecordError(span, err, "group not found")
		return nil, err
	}

	dbInput := &db.GetProviderVersionMirrorsInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(1)},
		Filter: &db.TerraformProviderVersionMirrorFilter{
			GroupID:           &group.Metadata.ID,
			RegistryHostname:  &input.RegistryHostname,
			RegistryNamespace: &input.RegistryNamespace,
			Type:              &input.Type,
			SemanticVersion:   &input.SemanticVersion,
		},
	}

	result, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, dbInput)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirrors")
		return nil, err
	}

	if result.PageInfo.TotalCount == 0 {
		tracing.RecordError(span, err, "terraform provider version mirror not found")
		return nil, errors.New("terraform provider version mirror with FQN %s/%s/%s not found", input.RegistryHostname, input.RegistryNamespace, input.Type, errors.WithErrorCode(errors.ENotFound))
	}

	return &result.VersionMirrors[0], nil
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
		err := caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithGroupID(m.GroupID))
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

	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithNamespacePath(input.NamespacePath))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	dbInput := &db.GetProviderVersionMirrorsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            &db.TerraformProviderVersionMirrorFilter{},
	}

	if input.IncludeInherited {
		pathParts := strings.Split(input.NamespacePath, "/")

		paths := []string{}
		for len(pathParts) > 0 {
			paths = append(paths, strings.Join(pathParts, "/"))
			// Remove last element
			pathParts = pathParts[:len(pathParts)-1]
		}

		dbInput.Filter.NamespacePaths = paths
	} else {
		dbInput.Filter.NamespacePaths = []string{input.NamespacePath}
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

	err = caller.RequirePermission(ctx, permissions.CreateTerraformProviderMirrorPermission, auth.WithNamespacePath(input.GroupPath))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	group, err := s.getGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		tracing.RecordError(span, err, "group not found")
		return nil, err
	}

	if group.ParentID != "" {
		tracing.RecordError(span, nil, "terraform provider version mirrors can only be created in a top-level group")
		return nil, errors.New("terraform provider version mirrors can only be created in a top-level group", errors.WithErrorCode(errors.EInvalid))
	}

	provider, err := parseProvider(input.RegistryHostname, input.RegistryNamespace, input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider")
		return nil, err
	}

	wantVersion, err := versions.ParseVersion(input.SemanticVersion)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider version")
		return nil, errors.Wrap(err, "invalid provider version", errors.WithErrorCode(errors.EInvalid))
	}

	// Discover the provider registry host and get the service URL.
	serviceURL, err := s.discovery.DiscoverServiceURL(provider.Hostname, "providers.v1")
	if err != nil {
		tracing.RecordError(span, err, "failed to discover provider registry's service URL")
		return nil, fmt.Errorf("failed to discover provider registry's service URL: %w", err)
	}

	// We will attempt to list all the supported versions for this provider, so we can find the
	// platforms it supports. It isn't possible to know this otherwise.
	availableVersions, err := s.listAvailableProviderVersions(ctx, serviceURL, provider)
	if err != nil {
		tracing.RecordError(span, err, "Failed to list available provider versions")
		return nil, errors.Wrap(err, "Failed to list available provider versions", errors.WithErrorCode(errors.ENotFound))
	}

	// Find a platform the provider supports. We only need one for our purposes.
	supportedPlatform, err := findSupportedPlatform(wantVersion, availableVersions)
	if err != nil {
		tracing.RecordError(span, err, "Unsupported provider version")
		return nil, errors.Wrap(err, "Unsupported version %s for provider %s", wantVersion, provider, errors.WithErrorCode(errors.EInvalid))
	}

	// Now, find the sha sums, signature URLs and the associated GPG key(s) by arbitrarily using
	// one of the supported platforms. The sha sums and signature URLs should be independent of
	// of the platform being queried for.
	packageResp, err := s.findProviderPackage(ctx, serviceURL, provider, wantVersion.String(), supportedPlatform)
	if err != nil {
		tracing.RecordError(span, err, "failed to find provider package")
		return nil, errors.Wrap(err, "Could not find package at provider registry API", errors.WithErrorCode(errors.ENotFound))
	}

	// Retrieve and verify the checksums from the response.
	digests, err := s.getChecksums(ctx, packageResp.SHASumsURL, packageResp.SHASumsSignatureURL, getGPGKeys(packageResp))
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
			s.logger.Errorf("failed to rollback tx for service layer CreateProviderVersionMirror: %v", txErr)
		}
	}()

	toCreate := &models.TerraformProviderVersionMirror{
		CreatedBy:         caller.GetSubject(),
		Type:              provider.Type,
		RegistryNamespace: provider.Namespace,
		RegistryHostname:  provider.Hostname.String(),
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

	s.logger.Infow("Created a terraform provider version mirror.",
		"caller", caller.GetSubject(),
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

	err = caller.RequirePermission(ctx, permissions.DeleteTerraformProviderMirrorPermission, auth.WithGroupID(input.VersionMirror.GroupID))
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
			s.logger.Errorf("failed to rollback tx for service layer DeleteProviderVersionMirror: %v", txErr)
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

	providerName := fmt.Sprintf("%s/%s/%s", input.VersionMirror.RegistryHostname, input.VersionMirror.RegistryNamespace, input.VersionMirror.Type)
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

	s.logger.Infow("Deleted a terraform provider version mirror.",
		"caller", caller.GetSubject(),
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

	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithGroupID(versionMirror.GroupID))
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

	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithGroupID(versionMirror.GroupID))
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

	err = caller.RequirePermission(ctx, permissions.DeleteTerraformProviderMirrorPermission, auth.WithGroupID(versionMirror.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return err
	}

	if err = s.dbClient.TerraformProviderPlatformMirrors.DeletePlatformMirror(ctx, input.PlatformMirror); err != nil {
		tracing.RecordError(span, err, "failed to delete provider platform mirror")
		return err
	}

	s.logger.Infow("Deleted a terraform provider platform mirror.",
		"caller", caller.GetSubject(),
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

	err = caller.RequirePermission(ctx, permissions.CreateTerraformProviderMirrorPermission, auth.WithGroupID(versionMirror.GroupID))
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

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UploadInstallationPackage: %v", txErr)
		}
	}()

	digestKey := getProviderPackageName(versionMirror.Type, versionMirror.SemanticVersion, input.OS, input.Architecture)

	// No point in continuing if we don't have a checksum for the package.
	expectDigest, ok := versionMirror.Digests[digestKey]
	if !ok {
		tracing.RecordError(span, nil, "no checksum available for provider package %s", digestKey)
		return fmt.Errorf("no checksum available for provider package %s", digestKey)
	}

	// This platform package hasn't been uploaded to the mirror yet, so we must upload it and
	// calculate the checksum for it afterwards.
	checksum := sha256.New()
	teeReader := io.TeeReader(
		// Wrap the package download in a limit reader so, we don't read indefinitely.
		io.LimitReader(input.Data, providerPlatformPackageMaxSizeLimit),
		checksum,
	)

	// Create a temp file we can download the package to. Needed to compute checksum prior to upload.
	f, err := os.CreateTemp("", "terraform-provider-package-*.zip")
	if err != nil {
		tracing.RecordError(span, err, "failed to create temporary package file")
		return err
	}
	defer os.Remove(f.Name())

	// Save the package to disk.
	if _, err = io.Copy(f, teeReader); err != nil {
		tracing.RecordError(span, err, "failed to save uploaded provider package file to disk")
		return fmt.Errorf("failed to save uploaded provider package file to disk: %w", err)
	}

	// Must calculate afterwards to avoid downloading the package into memory.
	calculatedSum := checksum.Sum(nil)

	if !bytes.Equal(expectDigest, calculatedSum) {
		tracing.RecordError(span, nil, "checksum of the uploaded provider platform package %x does not match the expected checksum %x", calculatedSum, expectDigest)
		return errors.New("checksum of the uploaded provider platform package %x does not match the expected checksum %x", calculatedSum, expectDigest, errors.WithErrorCode(errors.EInvalid))
	}

	toCreate := &models.TerraformProviderPlatformMirror{
		VersionMirrorID: versionMirror.Metadata.ID,
		OS:              input.OS,
		Architecture:    input.Architecture,
	}

	// Create the provider platform mirror before attempting an upload, incase the upload completes
	// and this fails to create. This allows us to rollback the transaction.
	if _, err = s.dbClient.TerraformProviderPlatformMirrors.CreatePlatformMirror(txContext, toCreate); err != nil {
		tracing.RecordError(span, err, "failed to create provider platform mirror")
		return fmt.Errorf("failed to create provider platform mirror: %w", err)
	}

	packageFile, err := os.Open(f.Name()) // nosemgrep: gosec.G304-1
	if err != nil {
		tracing.RecordError(span, err, "failed to open provider package file from disk")
		return fmt.Errorf("failed to open provider package file from disk: %w", err)
	}
	defer packageFile.Close()

	if err = s.mirrorStore.UploadProviderPlatformPackage(txContext, expectDigest, packageFile); err != nil {
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

	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	provider, err := parseProvider(input.RegistryHostname, input.RegistryNamespace, input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider")
		return nil, err
	}

	// Let the DB sort by semantic version ascending.
	sort := db.TerraformProviderVersionMirrorSortableFieldSemanticVersionAsc
	result, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, &db.GetProviderVersionMirrorsInput{
		Sort: &sort,
		Filter: &db.TerraformProviderVersionMirrorFilter{
			GroupID:           &group.Metadata.ID,
			RegistryHostname:  ptr.String(provider.Hostname.String()),
			RegistryNamespace: &provider.Namespace,
			Type:              &provider.Type,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version mirrors")
		return nil, err
	}

	// Per Terraform docs, must return a ENotFound when we have no mirrored provider versions.
	if result.PageInfo.TotalCount == 0 {
		tracing.RecordError(span, nil, "no versions are currently mirrored for Terraform provider %s", provider)
		return nil, errors.New("no versions are currently mirrored for Terraform provider %s", provider, errors.WithErrorCode(errors.ENotFound))
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

	err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderMirrorResourceType, auth.WithGroupID(group.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "caller permission check failed")
		return nil, err
	}

	provider, err := parseProvider(input.RegistryHostname, input.RegistryNamespace, input.Type)
	if err != nil {
		tracing.RecordError(span, err, "failed to parse provider")
		return nil, err
	}

	// Find the version mirror first.
	versionsResult, err := s.dbClient.TerraformProviderVersionMirrors.GetVersionMirrors(ctx, &db.GetProviderVersionMirrorsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.TerraformProviderVersionMirrorFilter{
			GroupID:           &group.Metadata.ID,
			RegistryHostname:  &input.RegistryHostname,
			RegistryNamespace: &input.RegistryNamespace,
			Type:              &input.Type,
			SemanticVersion:   &input.SemanticVersion,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get version mirror")
		return nil, err
	}

	if versionsResult.PageInfo.TotalCount == 0 {
		tracing.RecordError(span, nil, "version %s is currently not mirrored for Terraform provider %s", input.SemanticVersion, provider)
		return nil, errors.New("version %s is currently not mirrored for Terraform provider %s", input.SemanticVersion, provider, errors.WithErrorCode(errors.ENotFound))
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
		tracing.RecordError(span, nil, "no installation packages are currently mirrored for Terraform provider %s", provider)
		return nil, errors.New("no installation packages are currently mirrored for Terraform provider %s", provider, errors.WithErrorCode(errors.ENotFound))
	}

	// Build the list of supported packages.
	supportedPackages, err := s.buildSupportedPackages(ctx, &versionMirror, result.PlatformMirrors)
	if err != nil {
		tracing.RecordError(span, err, "failed to build supported packages")
		return nil, err
	}

	return supportedPackages, nil
}

// listAvailableProviderVersions lists the available provider versions and platforms
// they support by contacting the Terraform Registry API the provider is associated with.
func (s *service) listAvailableProviderVersions(
	ctx context.Context,
	serviceURL *url.URL,
	provider *tfaddr.Provider,
) (*listVersionsResponse, error) {
	result, err := url.Parse(path.Join(provider.Namespace, provider.Type, "versions"))
	if err != nil {
		return nil, err
	}

	endpoint := serviceURL.ResolveReference(result)

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate http request: %w", err)
	}

	r.Header.Add("Accept", "application/json")

	resp, err := s.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to perform http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode the payload to get the available provider versions.
	var decodedBody listVersionsResponse
	if err = json.NewDecoder(resp.Body).Decode(&decodedBody); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	// Make sure no warnings were returned.
	if len(decodedBody.Warnings) > 0 {
		return nil, fmt.Errorf("provider versions endpoint returned warnings: %s", strings.Join(decodedBody.Warnings, "; "))
	}

	if len(decodedBody.Versions) == 0 {
		return nil, fmt.Errorf("no versions found for provider %s", provider)
	}

	return &decodedBody, nil
}

// findProviderPackage attempts to locate the provider package at the provider's registry.
// It visits the endpoint for the target provider and parses the JSON response, which should
// give us access to the SHA256SUMS, SHA256SUMS.sig and GPG key used to sign the checksums file.
func (s *service) findProviderPackage(
	ctx context.Context,
	serviceURL *url.URL,
	provider *tfaddr.Provider,
	version,
	platform string,
) (*packageQueryResponse, error) {
	// Separate OS and arch.
	parts := strings.Split(platform, "_")

	// Build the URL to the provider's download endpoint which will give us access to the
	// SHASUMS, SHASUMS.sig and GPG key used to sign the checksums file. These are generally
	// hosted in a different location than the provider's registry.
	path := path.Join(
		provider.Namespace,
		provider.Type,
		version,
		"download",
		parts[0],
		parts[1],
	)

	result, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to build package download URL: %w", err)
	}

	endpoint := serviceURL.ResolveReference(result)

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build package download HTTP request: %w", err)
	}

	// We only want JSON.
	r.Header.Set("Accept", "application/json")

	response, err := s.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get package download URL: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var foundResp packageQueryResponse
	if err := json.NewDecoder(response.Body).Decode(&foundResp); err != nil {
		return nil, fmt.Errorf("failed to decode download package response body: %w", err)
	}

	return &foundResp, nil
}

func (s *service) getChecksums(
	ctx context.Context,
	shaSumsURL,
	shaSumsSignatureURL string,
	gpgKeys []string,
) (map[string][]byte, error) {
	// Parse the URL from the response.
	endpoint, err := url.Parse(shaSumsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checksums URL: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build download checksums request: %w", err)
	}

	checksumResp, err := s.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums file: %w", err)
	}
	defer checksumResp.Body.Close()

	if checksumResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status returned from checksums download: %d", checksumResp.StatusCode)
	}

	endpoint, err = url.Parse(shaSumsSignatureURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checksums signature URL: %w", err)
	}

	r, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build download signatures request: %w", err)
	}

	signatureResp, err := s.httpClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums signature file: %w", err)
	}
	defer signatureResp.Body.Close()

	if signatureResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status returned from checksums signature download: %d", signatureResp.StatusCode)
	}

	var (
		checksumReader io.Reader    // For building checksum map.
		buffer         bytes.Buffer // For validating checksum signature.
	)
	sigReader := io.TeeReader(checksumResp.Body, &buffer)
	checksumReader = &buffer

	// Verify the signature.
	if err = verifySumsSignature(sigReader, signatureResp.Body, gpgKeys); err != nil {
		return nil, fmt.Errorf("failed to verify checksum signature: %w", err)
	}

	checksumMap, err := toChecksumMap(checksumReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checksums: %w", err)
	}

	// Sanity check to make sure we actually have checksums.
	if len(checksumMap) == 0 {
		return nil, fmt.Errorf("no checksums found after parsing response")
	}

	return checksumMap, nil
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
		digestKey := getProviderPackageName(versionMirror.Type, versionMirror.SemanticVersion, mirror.OS, mirror.Architecture)

		digest, ok := versionMirror.Digests[digestKey]
		if !ok {
			// Shouldn't happen.
			return nil, fmt.Errorf("failed to get digest for provider package: %s", digestKey)
		}

		// Get the presignedURL for downloading this provider package.
		presignedURL, err := s.mirrorStore.GetProviderPlatformPackagePresignedURL(ctx, digest)
		if err != nil {
			return nil, fmt.Errorf("failed to get provider platform package presigned URL: %w", err)
		}

		// Use form "os_arch" as map key which is what Terraform CLI expects.
		key := fmt.Sprintf("%s_%s", mirror.OS, mirror.Architecture)

		supportedPackages[key] = map[string]any{
			"url": presignedURL,
			"hashes": []string{
				// Hash format complies with Terraform's zip hash format:
				// https://github.com/hashicorp/terraform/blob/d49e991c3c33c10b26c120465466d41f96e073de/internal/getproviders/hash.go#L330
				fmt.Sprintf("zh:%x", digest),
			},
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
	group, err := s.dbClient.Groups.GetGroupByFullPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if group == nil {
		return nil, errors.New("group not found", errors.WithErrorCode(errors.ENotFound))
	}

	return group, nil
}

// verifySumsSignature attempts to validate the signature on the checksum file.
func verifySumsSignature(checksums, signature io.Reader, gpgKeys []string) error {
	// Iterate through the keys and attempt to verify the signature. We only need a single key to match.
	// Per Terraform docs, at least one of the supplied keys must be used to sign the checksums.
	var matchFound bool
	for _, key := range gpgKeys {
		entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(key))
		if err != nil {
			return err
		}

		if _, err := openpgp.CheckDetachedSignature(entityList, checksums, signature, nil); err == nil {
			matchFound = true
			break
		}
	}

	if !matchFound {
		return fmt.Errorf("no matching key found for signature or signature mismatch")
	}

	return nil
}

// toChecksumMap returns a map of binary name --> SHA256SUM.
func toChecksumMap(reader io.Reader) (map[string][]byte, error) {
	checksumMap := make(map[string][]byte)
	scanner := bufio.NewScanner(reader)

	// Line by line process the checksum file.
	for scanner.Scan() {
		line := scanner.Text()

		// There should be exactly two parts for every line.
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected checksum line format: %s", line)
		}

		// Verify checksum only has hex values.
		hexBytes, err := hex.DecodeString(parts[0])
		if err != nil {
			return nil, err
		}

		// Check if it's the right length.
		if len(hexBytes) != sha256.Size {
			return nil, fmt.Errorf("unexpected checksum size. Expected %d, got %d", sha256.Size, len(hexBytes))
		}

		checksumMap[parts[1]] = hexBytes
	}

	return checksumMap, nil
}

// parseProvider parses a provider from the input. It validates
// all components to make sure they comply with Terraform's standards.
func parseProvider(hostname, namespace, providerType string) (*tfaddr.Provider, error) {
	// Must parse individual parts first to avoid any panics from NewProvider.
	ns, err := tfaddr.ParseProviderPart(namespace)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid registry namespace", errors.WithErrorCode(errors.EInvalid))
	}

	pType, err := tfaddr.ParseProviderPart(providerType)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid provider type", errors.WithErrorCode(errors.EInvalid))
	}

	convertedHostname, err := svchost.ForComparison(hostname)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid registry hostname", errors.WithErrorCode(errors.EInvalid))
	}

	provider := tfaddr.NewProvider(convertedHostname, ns, pType)

	return &provider, nil
}

// findSupportedPlatform finds the target provider version and returns a platform it supports.
func findSupportedPlatform(targetVersion versions.Version, versionsResp *listVersionsResponse) (string, error) {
	platform := ""

	for _, version := range versionsResp.Versions {
		v, err := versions.ParseVersion(version.Version)
		if err != nil {
			return "", fmt.Errorf("failed to parse provider version: %w", err)
		}

		if v.Same(targetVersion) {
			for _, p := range version.Platforms {
				// We will short-circuit here and just return the first platform we see
				// since that's all we really need.
				platform = fmt.Sprintf("%s_%s", p.OS, p.Arch)
				break
			}

			break
		}
	}

	// If this is empty then version is likely not supported by the provider.
	if platform == "" {
		return "", fmt.Errorf("no supported platforms found or provider version not supported")
	}

	return platform, nil
}

// getGPGKeys returns a slice of GPG keys from the packageQueryResponse.
func getGPGKeys(packageResp *packageQueryResponse) []string {
	gpgKeys := []string{}
	for _, key := range packageResp.SigningKeys.GPGPublicKeys {
		gpgKeys = append(gpgKeys, key.ASCIIArmor)
	}

	return gpgKeys
}

// Should conform to the following format as used by Terraform:
// terraform-provider-<provider_type>_<version>_<os>_<arch>.zip
func getProviderPackageName(providerType, version, os, arch string) string {
	return fmt.Sprintf("terraform-provider-%s_%s_%s_%s.zip", providerType, version, os, arch)
}
