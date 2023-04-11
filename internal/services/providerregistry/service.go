package providerregistry

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/go-version"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/semver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// CreateProviderInput is the input for creating a terraform provider
type CreateProviderInput struct {
	Name          string
	GroupID       string
	RepositoryURL string
	Private       bool
}

// CreateProviderVersionInput is the input for creating a terraform provider version
type CreateProviderVersionInput struct {
	SemanticVersion string
	ProviderID      string
	Protocols       []string
}

// CreateProviderPlatformInput is the input for creating a terraform provider platform
type CreateProviderPlatformInput struct {
	ProviderVersionID string
	OperatingSystem   string
	Architecture      string
	SHASum            string
	Filename          string
}

// GetProvidersInput is the input for getting a list of terraform providers
type GetProvidersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TerraformProviderSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Group filters providers be the specified group
	Group *models.Group
	// Search filters provider list by providers with a name that contains the search query
	Search *string
}

// GetProviderVersionsInput is the input for getting a list of provider versions
type GetProviderVersionsInput struct {
	Sort                     *db.TerraformProviderVersionSortableField
	PaginationOptions        *pagination.Options
	SHASumsUploaded          *bool
	SHASumsSignatureUploaded *bool
	SemanticVersion          *string
	Latest                   *bool
	ProviderID               string
}

// GetProviderPlatformsInput is the input for listing provider platforms
type GetProviderPlatformsInput struct {
	Sort              *db.TerraformProviderPlatformSortableField
	PaginationOptions *pagination.Options
	ProviderID        *string
	ProviderVersionID *string
	BinaryUploaded    *bool
	OperatingSystem   *string
	Architecture      *string
}

// ProviderPlatformDownloadURLs contains the signed URLs for downloading a particular provider platform
type ProviderPlatformDownloadURLs struct {
	DownloadURL         string
	SHASumsURL          string
	SHASumsSignatureURL string
}

// Service implements all provider registry functionality
type Service interface {
	GetProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error)
	GetProviderByPath(ctx context.Context, path string) (*models.TerraformProvider, error)
	GetProviderByAddress(ctx context.Context, namespace string, name string) (*models.TerraformProvider, error)
	GetProvidersByIDs(ctx context.Context, ids []string) ([]models.TerraformProvider, error)
	GetProviders(ctx context.Context, input *GetProvidersInput) (*db.ProvidersResult, error)
	CreateProvider(ctx context.Context, input *CreateProviderInput) (*models.TerraformProvider, error)
	UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error)
	DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error
	GetProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error)
	GetProviderVersions(ctx context.Context, input *GetProviderVersionsInput) (*db.ProviderVersionsResult, error)
	GetProviderVersionsByIDs(ctx context.Context, ids []string) ([]models.TerraformProviderVersion, error)
	CreateProviderVersion(ctx context.Context, input *CreateProviderVersionInput) (*models.TerraformProviderVersion, error)
	DeleteProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) error
	GetProviderVersionReadme(ctx context.Context, providerVersion *models.TerraformProviderVersion) (string, error)
	GetProviderPlatformByID(ctx context.Context, id string) (*models.TerraformProviderPlatform, error)
	GetProviderPlatforms(ctx context.Context, input *GetProviderPlatformsInput) (*db.ProviderPlatformsResult, error)
	CreateProviderPlatform(ctx context.Context, input *CreateProviderPlatformInput) (*models.TerraformProviderPlatform, error)
	DeleteProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) error
	UploadProviderPlatformBinary(ctx context.Context, providerPlatformID string, reader io.Reader) error
	UploadProviderVersionReadme(ctx context.Context, providerVersionID string, reader io.Reader) error
	UploadProviderVersionSHA256Sums(ctx context.Context, providerVersionID string, reader io.Reader) error
	UploadProviderVersionSHA256SumsSignature(ctx context.Context, providerVersionID string, reader io.Reader) error
	GetProviderPlatformDownloadURLs(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*ProviderPlatformDownloadURLs, error)
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	registryStore   RegistryStore
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	registryStore RegistryStore,
	activityService activityevent.Service,
) Service {
	return &service{
		logger,
		dbClient,
		registryStore,
		activityService,
	}
}

func (s *service) GetProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	return provider, nil
}

func (s *service) GetProviderByPath(ctx context.Context, path string) (*models.TerraformProvider, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := s.dbClient.TerraformProviders.GetProviderByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	if provider == nil {
		return nil, errors.New(errors.ENotFound, "provider with path %s not found", path)
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	return provider, nil
}

func (s *service) GetProviderByAddress(ctx context.Context, namespace string, name string) (*models.TerraformProvider, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	rootGroup, err := s.dbClient.Groups.GetGroupByFullPath(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if rootGroup == nil {
		return nil, errors.New(errors.ENotFound, "namespace %s not found", namespace)
	}

	providerResult, err := s.dbClient.TerraformProviders.GetProviders(ctx, &db.GetProvidersInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(1)},
		Filter: &db.TerraformProviderFilter{
			RootGroupID: &rootGroup.Metadata.ID,
			Name:        &name,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(providerResult.Providers) == 0 {
		return nil, errors.New(errors.ENotFound, "provider with name %s not found in namespace %s", name, namespace)
	}

	provider := providerResult.Providers[0]

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	return &provider, nil
}

func (s *service) GetProviders(ctx context.Context, input *GetProvidersInput) (*db.ProvidersResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	dbInput := db.GetProvidersInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformProviderFilter{
			Search: input.Search,
		},
	}

	if input.Group != nil {
		err = caller.RequirePermission(ctx, permissions.ViewTerraformProviderPermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			return nil, err
		}
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			return nil, napErr
		}

		if !policy.AllowAll {
			if err = auth.HandleCaller(
				ctx,
				func(_ context.Context, c *auth.UserCaller) error {
					dbInput.Filter.UserID = &c.User.Metadata.ID
					return nil
				},
				func(_ context.Context, c *auth.ServiceAccountCaller) error {
					dbInput.Filter.ServiceAccountID = &c.ServiceAccountID
					return nil
				},
			); err != nil {
				return nil, err
			}
		}
	}

	return s.dbClient.TerraformProviders.GetProviders(ctx, &dbInput)
}

func (s *service) UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return nil, err
	}

	if vErr := provider.Validate(); vErr != nil {
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateProvider: %v", txErr)
		}
	}()

	updatedProvider, err := s.dbClient.TerraformProviders.UpdateProvider(txContext, provider)
	if err != nil {
		return nil, err
	}

	groupPath := updatedProvider.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetTerraformProvider,
			TargetID:      updatedProvider.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return updatedProvider, nil
}

func (s *service) CreateProvider(ctx context.Context, input *CreateProviderInput) (*models.TerraformProvider, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.CreateTerraformProviderPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	if group == nil {
		return nil, fmt.Errorf("group with id %s not found", input.GroupID)
	}

	var rootGroupID string
	if group.ParentID == "" {
		rootGroupID = input.GroupID
	} else {
		rootGroup, gErr := s.dbClient.Groups.GetGroupByFullPath(ctx, group.GetRootGroupPath())
		if gErr != nil {
			return nil, gErr
		}

		if rootGroup == nil {
			return nil, fmt.Errorf("group with path %s not found", group.GetRootGroupPath())
		}
		rootGroupID = rootGroup.Metadata.ID
	}

	providerToCreate := &models.TerraformProvider{
		Name:          input.Name,
		GroupID:       input.GroupID,
		RootGroupID:   rootGroupID,
		Private:       input.Private,
		RepositoryURL: input.RepositoryURL,
		CreatedBy:     caller.GetSubject(),
	}

	if vErr := providerToCreate.Validate(); vErr != nil {
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateProvider: %v", txErr)
		}
	}()

	createdProvider, err := s.dbClient.TerraformProviders.CreateProvider(txContext, providerToCreate)
	if err != nil {
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformProvider,
			TargetID:      createdProvider.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return createdProvider, nil
}

func (s *service) DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteProvider: %v", txErr)
		}
	}()

	err = s.dbClient.TerraformProviders.DeleteProvider(txContext, provider)
	if err != nil {
		return err
	}

	groupPath := provider.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      provider.GroupID,
			Payload: &models.ActivityEventDeleteChildResourcePayload{
				Name: provider.Name,
				ID:   provider.Metadata.ID,
				Type: string(models.TargetTerraformProvider),
			},
		}); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetProvidersByIDs(ctx context.Context, ids []string) ([]models.TerraformProvider, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.dbClient.TerraformProviders.GetProviders(ctx, &db.GetProvidersInput{
		Filter: &db.TerraformProviderFilter{
			TerraformProviderIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	namespacePaths := []string{}
	for _, provider := range response.Providers {
		if provider.Private {
			namespacePaths = append(namespacePaths, provider.GetGroupPath())
		}
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			return nil, err
		}
	}

	return response.Providers, nil
}

func (s *service) GetProviderVersionReadme(ctx context.Context, providerVersion *models.TerraformProviderVersion) (string, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return "", err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return "", err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return "", err
		}
	}

	reader, err := s.registryStore.GetProviderVersionReadme(ctx, providerVersion, provider)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	buffer, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

func (s *service) GetProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	return providerVersion, nil
}

func (s *service) GetProviderVersions(ctx context.Context, input *GetProviderVersionsInput) (*db.ProviderVersionsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, input.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	dbInput := db.GetProviderVersionsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TerraformProviderVersionFilter{
			ProviderID:               &input.ProviderID,
			SHASumsUploaded:          input.SHASumsUploaded,
			SHASumsSignatureUploaded: input.SHASumsSignatureUploaded,
			SemanticVersion:          input.SemanticVersion,
			Latest:                   input.Latest,
		},
	}

	return s.dbClient.TerraformProviderVersions.GetProviderVersions(ctx, &dbInput)

}

func (s *service) GetProviderVersionsByIDs(ctx context.Context, ids []string) ([]models.TerraformProviderVersion, error) {
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	response, err := s.dbClient.TerraformProviderVersions.GetProviderVersions(ctx, &db.GetProviderVersionsInput{
		Filter: &db.TerraformProviderVersionFilter{
			ProviderVersionIDs: ids,
		},
	})
	if err != nil {
		return nil, err
	}

	providerIDList := []string{}

	for _, version := range response.ProviderVersions {
		providerIDList = append(providerIDList, version.ProviderID)
	}

	// GetProvidersByIDs performs the authorization checks to verify that the subject
	// can view the requested providers
	if _, err := s.GetProvidersByIDs(ctx, providerIDList); err != nil {
		return nil, err
	}

	return response.ProviderVersions, nil
}

func (s *service) CreateProviderVersion(ctx context.Context, input *CreateProviderVersionInput) (*models.TerraformProviderVersion, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, input.ProviderID)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return nil, err
	}

	// Verify semantic version is valid
	semVersion, err := version.NewSemver(input.SemanticVersion)
	if err != nil {
		return nil, errors.Wrap(err, errors.EInvalid, "invalid semantic version")
	}

	// Check if this version is greater than the previous latest
	versionsResp, err := s.dbClient.TerraformProviderVersions.GetProviderVersions(ctx, &db.GetProviderVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.TerraformProviderVersionFilter{
			ProviderID: &input.ProviderID,
			Latest:     ptr.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateProviderVersion: %v", txErr)
		}
	}()

	isLatest := false
	if len(versionsResp.ProviderVersions) > 0 {
		prevLatest := versionsResp.ProviderVersions[0]
		prevSemVersion, sErr := version.NewSemver(prevLatest.SemanticVersion)
		if sErr != nil {
			return nil, sErr
		}
		if semver.IsSemverGreaterThan(semVersion, prevSemVersion) {
			isLatest = true
			// Remove latest from version
			prevLatest.Latest = false
			if _, uErr := s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, &prevLatest); uErr != nil {
				return nil, uErr
			}
		}
	} else {
		// New version is the latest since it is the only version
		isLatest = true
	}

	providerVersion, err := s.dbClient.TerraformProviderVersions.CreateProviderVersion(txContext, &models.TerraformProviderVersion{
		ProviderID:      input.ProviderID,
		SemanticVersion: semVersion.String(),
		Protocols:       input.Protocols,
		Latest:          isLatest,
		CreatedBy:       caller.GetSubject(),
	})
	if err != nil {
		return nil, err
	}

	groupPath := provider.GetGroupPath()

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformProviderVersion,
			TargetID:      providerVersion.Metadata.ID,
		}); err != nil {
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	s.logger.Infow("Created a provider version.",
		"caller", caller.GetSubject(),
		"providerID", input.ProviderID,
		"providerVersion", providerVersion.SemanticVersion,
	)

	return providerVersion, nil
}

func (s *service) DeleteProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	// Reset latest flag if we're deleting the latest version
	var newLatestVersion *models.TerraformProviderVersion
	if providerVersion.Latest {
		versionsResp, gpErr := s.dbClient.TerraformProviderVersions.GetProviderVersions(ctx, &db.GetProviderVersionsInput{
			Filter: &db.TerraformProviderVersionFilter{
				ProviderID: &provider.Metadata.ID,
			},
		})
		if gpErr != nil {
			return err
		}

		for _, v := range versionsResp.ProviderVersions {
			vCopy := v

			// Skip if this is the provider version we're deleting
			if v.Metadata.ID == providerVersion.Metadata.ID {
				continue
			}

			if newLatestVersion == nil {
				newLatestVersion = &vCopy
				continue
			}

			latestSemver, lsErr := version.NewSemver(newLatestVersion.SemanticVersion)
			if lsErr != nil {
				return lsErr
			}

			currentSemver, csErr := version.NewSemver(vCopy.SemanticVersion)
			if csErr != nil {
				return csErr
			}

			if semver.IsSemverGreaterThan(currentSemver, latestSemver) {
				newLatestVersion = &vCopy
			}
		}
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for DeleteProviderVersion: %v", txErr)
		}
	}()

	// Delete provider version from DB
	if err = s.dbClient.TerraformProviderVersions.DeleteProviderVersion(txContext, providerVersion); err != nil {
		return err
	}

	if newLatestVersion != nil {
		s.logger.Infof(
			"Deleted latest provider version, latest flag is being set to latest version %s for provider %s",
			newLatestVersion.SemanticVersion,
			provider.GetRegistryNamespace(),
			provider.Name,
		)
		newLatestVersion.Latest = true
		if _, err = s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, newLatestVersion); err != nil {
			return err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return err
	}

	s.logger.Infow("Deleted a provider version.",
		"caller", caller.GetSubject(),
		"providerID", provider.Metadata.ID,
		"providerVersion", providerVersion.SemanticVersion,
	)

	return nil
}

func (s *service) GetProviderPlatformByID(ctx context.Context, id string) (*models.TerraformProviderPlatform, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	platform, err := s.getProviderPlatformByID(ctx, id)
	if err != nil {
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, platform.ProviderVersionID)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	return platform, nil
}

func (s *service) GetProviderPlatforms(ctx context.Context, input *GetProviderPlatformsInput) (*db.ProviderPlatformsResult, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Verify at least one filter is set
	if input.ProviderID == nil && input.ProviderVersionID == nil {
		return nil, errors.New(errors.EInternal, "the provider id or provider version id filter must be set when querying for provider platforms")
	}

	var provider *models.TerraformProvider
	if input.ProviderID != nil {
		provider, err = s.getProviderByID(ctx, *input.ProviderID)
		if err != nil {
			return nil, err
		}
	} else if input.ProviderVersionID != nil {
		providerVersion, pvErr := s.getProviderVersionByID(ctx, *input.ProviderVersionID)
		if pvErr != nil {
			return nil, err
		}

		provider, err = s.getProviderByID(ctx, providerVersion.ProviderID)
		if err != nil {
			return nil, err
		}
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	dbInput := db.GetProviderPlatformsInput{
		Filter: &db.TerraformProviderPlatformFilter{
			ProviderID:        input.ProviderID,
			ProviderVersionID: input.ProviderVersionID,
			BinaryUploaded:    input.BinaryUploaded,
			OperatingSystem:   input.OperatingSystem,
			Architecture:      input.Architecture,
		},
	}

	response, err := s.dbClient.TerraformProviderPlatforms.GetProviderPlatforms(ctx, &dbInput)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *service) CreateProviderPlatform(ctx context.Context, input *CreateProviderPlatformInput) (*models.TerraformProviderPlatform, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, input.ProviderVersionID)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return nil, err
	}

	return s.dbClient.TerraformProviderPlatforms.CreateProviderPlatform(ctx, &models.TerraformProviderPlatform{
		ProviderVersionID: input.ProviderVersionID,
		OperatingSystem:   input.OperatingSystem,
		Architecture:      input.Architecture,
		SHASum:            input.SHASum,
		Filename:          input.Filename,
		BinaryUploaded:    false,
		CreatedBy:         caller.GetSubject(),
	})
}

func (s *service) DeleteProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerPlatform.ProviderVersionID)
	if err != nil {
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	return s.dbClient.TerraformProviderPlatforms.DeleteProviderPlatform(ctx, providerPlatform)
}

func (s *service) UploadProviderPlatformBinary(ctx context.Context, providerPlatformID string, reader io.Reader) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	providerPlatform, err := s.getProviderPlatformByID(ctx, providerPlatformID)
	if err != nil {
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerPlatform.ProviderVersionID)
	if err != nil {
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	if providerPlatform.BinaryUploaded {
		return errors.New(errors.EConflict, "binary already uploaded")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	// Update DB before object storage. If the object storage write fails, the DB transaction will be rolled back
	providerPlatform.BinaryUploaded = true
	if _, err := s.dbClient.TerraformProviderPlatforms.UpdateProviderPlatform(txContext, providerPlatform); err != nil {
		return err
	}

	if err := s.registryStore.UploadProviderPlatformBinary(ctx, providerPlatform, providerVersion, provider, reader); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) UploadProviderVersionReadme(ctx context.Context, providerVersionID string, reader io.Reader) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	if providerVersion.ReadmeUploaded {
		return errors.New(errors.EConflict, "README file already uploaded")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	// Update DB before object storage. If the object storage write fails, the DB transaction will be rolled back
	providerVersion.ReadmeUploaded = true
	if _, err := s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, providerVersion); err != nil {
		return err
	}

	if err := s.registryStore.UploadProviderVersionReadme(ctx, providerVersion, provider, reader); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) UploadProviderVersionSHA256Sums(ctx context.Context, providerVersionID string, reader io.Reader) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	if providerVersion.SHASumsUploaded {
		return errors.New(errors.EConflict, "shasums file already uploaded")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	// Update DB before object storage. If the object storage write fails, the DB transaction will be rolled back
	providerVersion.SHASumsUploaded = true
	if _, err := s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, providerVersion); err != nil {
		return err
	}

	if err := s.registryStore.UploadProviderVersionSHASums(ctx, providerVersion, provider, reader); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) UploadProviderVersionSHA256SumsSignature(ctx context.Context, providerVersionID string, reader io.Reader) error {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		return err
	}

	if providerVersion.SHASumsSignatureUploaded {
		return errors.New(errors.EConflict, "shasums signature file already uploaded")
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx: %v", txErr)
		}
	}()

	var sigBuffer bytes.Buffer

	// Use Tee reader to read signature to get GPG key id
	packetReader := packet.NewReader(io.TeeReader(reader, &sigBuffer))

	pkt, err := packetReader.Next()
	if err != nil {
		return errors.Wrap(err, errors.EInvalid, "failed to read gpg signature")
	}

	key, ok := pkt.(*packet.Signature)
	if !ok {
		return errors.Wrap(err, errors.EInvalid, "gpg signature is not in valid format")
	}

	// GPG key id is used to lookup the trusted GPG public key
	gpgKeyID := key.IssuerKeyId

	// Get the group that this provider is in
	group, err := s.dbClient.Groups.GetGroupByID(ctx, provider.GroupID)
	if err != nil {
		return err
	}

	if group == nil {
		return fmt.Errorf("group with id %s not found", provider.GroupID)
	}

	// Find key by GPG key id
	searchKeyResult, err := s.dbClient.GPGKeys.GetGPGKeys(ctx, &db.GetGPGKeysInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1), // Only return first key that matches
		},
		Filter: &db.GPGKeyFilter{
			GPGKeyID:       gpgKeyID,
			NamespacePaths: group.ExpandPath(),
		}})

	if searchKeyResult.PageInfo.TotalCount == 0 {
		return errors.Wrap(err, errors.EInvalid, "a trusted gpg key for key id %d does not exist", gpgKeyID)
	}

	gpgKey := searchKeyResult.GPGKeys[0]

	// Update DB before object storage. If the object storage write fails, the DB transaction will be rolled back
	providerVersion.GPGKeyID = &gpgKey.GPGKeyID
	providerVersion.GPGASCIIArmor = &gpgKey.ASCIIArmor
	providerVersion.SHASumsSignatureUploaded = true
	if _, err := s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, providerVersion); err != nil {
		return err
	}

	if err := s.registryStore.UploadProviderVersionSHASumsSignature(ctx, providerVersion, provider, &sigBuffer); err != nil {
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetProviderPlatformDownloadURLs(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*ProviderPlatformDownloadURLs, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerPlatform.ProviderVersionID)
	if err != nil {
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.TerraformProviderResourceType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			return nil, err
		}
	}

	downloadURL, err := s.registryStore.GetProviderPlatformBinaryPresignedURL(ctx, providerPlatform, providerVersion, provider)
	if err != nil {
		return nil, err
	}

	shaSumsURL, err := s.registryStore.GetProviderVersionSHASumsPresignedURL(ctx, providerVersion, provider)
	if err != nil {
		return nil, err
	}

	shaSumsSignatureURL, err := s.registryStore.GetProviderVersionSHASumsSignaturePresignedURL(ctx, providerVersion, provider)
	if err != nil {
		return nil, err
	}

	return &ProviderPlatformDownloadURLs{
		DownloadURL:         downloadURL,
		SHASumsURL:          shaSumsURL,
		SHASumsSignatureURL: shaSumsSignatureURL,
	}, nil
}

func (s *service) getProviderPlatformByID(ctx context.Context, id string) (*models.TerraformProviderPlatform, error) {
	platform, err := s.dbClient.TerraformProviderPlatforms.GetProviderPlatformByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if platform == nil {
		return nil, errors.New(errors.ENotFound, "provider platform with id %s not found", id)
	}

	return platform, nil
}

func (s *service) getProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error) {
	provider, err := s.dbClient.TerraformProviders.GetProviderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if provider == nil {
		return nil, errors.New(errors.ENotFound, "provider with id %s not found", id)
	}

	return provider, nil
}

func (s *service) getProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	version, err := s.dbClient.TerraformProviderVersions.GetProviderVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.New(errors.ENotFound, "provider version with id %s not found", id)
	}

	return version, nil
}
