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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/semver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
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
	GetProviderByTRN(ctx context.Context, trn string) (*models.TerraformProvider, error)
	GetProviderByAddress(ctx context.Context, namespace string, name string) (*models.TerraformProvider, error)
	GetProvidersByIDs(ctx context.Context, ids []string) ([]models.TerraformProvider, error)
	GetProviders(ctx context.Context, input *GetProvidersInput) (*db.ProvidersResult, error)
	CreateProvider(ctx context.Context, input *CreateProviderInput) (*models.TerraformProvider, error)
	UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error)
	DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error
	GetProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error)
	GetProviderVersionByTRN(ctx context.Context, trn string) (*models.TerraformProviderVersion, error)
	GetProviderVersions(ctx context.Context, input *GetProviderVersionsInput) (*db.ProviderVersionsResult, error)
	GetProviderVersionsByIDs(ctx context.Context, ids []string) ([]models.TerraformProviderVersion, error)
	CreateProviderVersion(ctx context.Context, input *CreateProviderVersionInput) (*models.TerraformProviderVersion, error)
	DeleteProviderVersion(ctx context.Context, providerVersion *models.TerraformProviderVersion) error
	GetProviderVersionReadme(ctx context.Context, providerVersion *models.TerraformProviderVersion) (string, error)
	GetProviderPlatformByID(ctx context.Context, id string) (*models.TerraformProviderPlatform, error)
	GetProviderPlatformByTRN(ctx context.Context, trn string) (*models.TerraformProviderPlatform, error)
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
	limitChecker    limits.LimitChecker
	registryStore   RegistryStore
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	registryStore RegistryStore,
	activityService activityevent.Service,
) Service {
	return &service{
		logger,
		dbClient,
		limitChecker,
		registryStore,
		activityService,
	}
}

func (s *service) GetProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return provider, nil
}

func (s *service) GetProviderByTRN(ctx context.Context, trn string) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	provider, err := s.dbClient.TerraformProviders.GetProviderByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by TRN")
		return nil, err
	}

	if provider == nil {
		return nil, errors.New("provider with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return provider, nil
}

func (s *service) GetProviderByAddress(ctx context.Context, namespace string, name string) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderByAddress")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	rootGroup, err := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(namespace))
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by TRN")
		return nil, err
	}

	if rootGroup == nil {
		return nil, errors.New("namespace %s not found", namespace, errors.WithErrorCode(errors.ENotFound))
	}

	providerResult, err := s.dbClient.TerraformProviders.GetProviders(ctx, &db.GetProvidersInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(1)},
		Filter: &db.TerraformProviderFilter{
			RootGroupID: &rootGroup.Metadata.ID,
			Name:        &name,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get providers")
		return nil, err
	}

	if len(providerResult.Providers) == 0 {
		return nil, errors.New("provider with name %s not found in namespace %s", name, namespace, errors.WithErrorCode(errors.ENotFound))
	}

	provider := providerResult.Providers[0]

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return &provider, nil
}

func (s *service) GetProviders(ctx context.Context, input *GetProvidersInput) (*db.ProvidersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviders")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
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
		err = caller.RequirePermission(ctx, models.ViewTerraformProviderPermission, auth.WithNamespacePath(input.Group.FullPath))
		if err != nil {
			tracing.RecordError(span, err, "permission check failed")
			return nil, err
		}
		dbInput.Filter.GroupID = &input.Group.Metadata.ID
	} else {
		policy, napErr := caller.GetNamespaceAccessPolicy(ctx)
		if napErr != nil {
			tracing.RecordError(span, napErr, "failed to get namespace access policy")
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
				tracing.RecordError(span, err, "failed to set filters for non-admin authorization")
				return nil, err
			}
		}
	}

	return s.dbClient.TerraformProviders.GetProviders(ctx, &dbInput)
}

func (s *service) UpdateProvider(ctx context.Context, provider *models.TerraformProvider) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	if vErr := provider.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate terraform provider model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateProvider: %v", txErr)
		}
	}()

	updatedProvider, err := s.dbClient.TerraformProviders.UpdateProvider(txContext, provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to update provider")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedProvider, nil
}

func (s *service) CreateProvider(ctx context.Context, input *CreateProviderInput) (*models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.CreateTerraformProviderPermission, auth.WithGroupID(input.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	group, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by ID")
		return nil, err
	}

	if group == nil {
		return nil, fmt.Errorf("group with id %s not found", input.GroupID)
	}

	var rootGroupID string
	if group.ParentID == "" {
		rootGroupID = input.GroupID
	} else {
		rootGroup, gErr := s.dbClient.Groups.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(group.GetRootGroupPath()))
		if gErr != nil {
			tracing.RecordError(span, gErr, "failed to get group by full path")
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
		tracing.RecordError(span, vErr, "failed to validate terraform provider model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateProvider: %v", txErr)
		}
	}()

	createdProvider, err := s.dbClient.TerraformProviders.CreateProvider(txContext, providerToCreate)
	if err != nil {
		tracing.RecordError(span, err, "failed to create provider")
		return nil, err
	}

	// Get the number of providers in the group to check whether we just violated the limit.
	newProviders, err := s.dbClient.TerraformProviders.GetProviders(txContext, &db.GetProvidersInput{
		Filter: &db.TerraformProviderFilter{
			GroupID: &createdProvider.GroupID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get group's Terraform providers")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitTerraformProvidersPerGroup, newProviders.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformProvider,
			TargetID:      createdProvider.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdProvider, nil
}

func (s *service) DeleteProvider(ctx context.Context, provider *models.TerraformProvider) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteProvider")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
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
			s.logger.Errorf("failed to rollback tx for service layer DeleteProvider: %v", txErr)
		}
	}()

	err = s.dbClient.TerraformProviders.DeleteProvider(txContext, provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to delete provider")
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
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetProvidersByIDs(ctx context.Context, ids []string) ([]models.TerraformProvider, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProvidersByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	response, err := s.dbClient.TerraformProviders.GetProviders(ctx, &db.GetProvidersInput{
		Filter: &db.TerraformProviderFilter{
			TerraformProviderIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get providers")
		return nil, err
	}

	namespacePaths := []string{}
	for _, provider := range response.Providers {
		if provider.Private {
			namespacePaths = append(namespacePaths, provider.GetGroupPath())
		}
	}

	if len(namespacePaths) > 0 {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithNamespacePaths(namespacePaths))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return response.Providers, nil
}

func (s *service) GetProviderVersionReadme(ctx context.Context, providerVersion *models.TerraformProviderVersion) (string, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionReadme")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return "", err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return "", err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return "", err
		}
	}

	reader, err := s.registryStore.GetProviderVersionReadme(ctx, providerVersion, provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version readme")
		return "", err
	}
	defer reader.Close()

	buffer, err := io.ReadAll(reader)
	if err != nil {
		tracing.RecordError(span, err, "failed to create reder for provider module readme")
		return "", err
	}

	return string(buffer), nil
}

func (s *service) GetProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return providerVersion, nil
}

func (s *service) GetProviderVersionByTRN(ctx context.Context, trn string) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	providerVersion, err := s.dbClient.TerraformProviderVersions.GetProviderVersionByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by TRN")
		return nil, err
	}

	if providerVersion == nil {
		return nil, errors.New("provider version with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return providerVersion, nil
}

func (s *service) GetProviderVersions(ctx context.Context, input *GetProviderVersionsInput) (*db.ProviderVersionsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersions")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, input.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
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
	ctx, span := tracer.Start(ctx, "svc.GetProviderVersionsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	response, err := s.dbClient.TerraformProviderVersions.GetProviderVersions(ctx, &db.GetProviderVersionsInput{
		Filter: &db.TerraformProviderVersionFilter{
			ProviderVersionIDs: ids,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider versions")
		return nil, err
	}

	providerIDList := []string{}

	for _, version := range response.ProviderVersions {
		providerIDList = append(providerIDList, version.ProviderID)
	}

	// GetProvidersByIDs performs the authorization checks to verify that the subject
	// can view the requested providers
	if _, err := s.GetProvidersByIDs(ctx, providerIDList); err != nil {
		tracing.RecordError(span, err, "failed to get providers by IDs")
		return nil, err
	}

	return response.ProviderVersions, nil
}

func (s *service) CreateProviderVersion(ctx context.Context, input *CreateProviderVersionInput) (*models.TerraformProviderVersion, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateProviderVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, input.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Verify semantic version is valid
	semVersion, err := version.NewSemver(input.SemanticVersion)
	if err != nil {
		tracing.RecordError(span, err, "invalid semantic version")
		return nil, errors.Wrap(err, "invalid semantic version", errors.WithErrorCode(errors.EInvalid))
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
		tracing.RecordError(span, err, "failed to get provider versions")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
			tracing.RecordError(span, sErr, "failed to validate semver")
			return nil, sErr
		}
		if semver.IsSemverGreaterThan(semVersion, prevSemVersion) {
			isLatest = true
			// Remove latest from version
			prevLatest.Latest = false
			if _, uErr := s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, &prevLatest); uErr != nil {
				tracing.RecordError(span, uErr, "failed to update provider version")
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
		tracing.RecordError(span, err, "failed to create provider version")
		return nil, err
	}

	groupPath := provider.GetGroupPath()

	// Get the number of versions for the provider to check whether we just violated the limit.
	newVersions, err := s.dbClient.TerraformProviderVersions.GetProviderVersions(txContext, &db.GetProviderVersionsInput{
		Filter: &db.TerraformProviderVersionFilter{
			TimeRangeStart: ptr.Time(providerVersion.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			ProviderID:     &providerVersion.ProviderID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider's versions")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitVersionsPerTerraformProviderPerTimePeriod, newVersions.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &groupPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetTerraformProviderVersion,
			TargetID:      providerVersion.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.DeleteProviderVersion")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
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
			tracing.RecordError(span, err, "failed to get provider versions")
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
				tracing.RecordError(span, lsErr, "failed to validate latest semver")
				return lsErr
			}

			currentSemver, csErr := version.NewSemver(vCopy.SemanticVersion)
			if csErr != nil {
				tracing.RecordError(span, csErr, "failed to validate current semver")
				return csErr
			}

			if semver.IsSemverGreaterThan(currentSemver, latestSemver) {
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
			s.logger.Errorf("failed to rollback tx for DeleteProviderVersion: %v", txErr)
		}
	}()

	// Delete provider version from DB
	if err = s.dbClient.TerraformProviderVersions.DeleteProviderVersion(txContext, providerVersion); err != nil {
		tracing.RecordError(span, err, "failed to delete module version")
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
			tracing.RecordError(span, err, "failed to update provider version")
			return err
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
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
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatformByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	platform, err := s.getProviderPlatformByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform by ID")
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, platform.ProviderVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return platform, nil
}

func (s *service) GetProviderPlatformByTRN(ctx context.Context, trn string) (*models.TerraformProviderPlatform, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatformByTRN")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	platform, err := s.dbClient.TerraformProviderPlatforms.GetProviderPlatformByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform by TRN")
		return nil, err
	}

	if platform == nil {
		return nil, errors.New("provider platform with TRN %s not found", trn, errors.WithErrorCode(errors.ENotFound))
	}

	providerVersion, err := s.getProviderVersionByID(ctx, platform.ProviderVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	return platform, nil
}

func (s *service) GetProviderPlatforms(ctx context.Context, input *GetProviderPlatformsInput) (*db.ProviderPlatformsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatforms")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Verify at least one filter is set
	if input.ProviderID == nil && input.ProviderVersionID == nil {
		return nil, errors.New("the provider id or provider version id filter must be set when querying for provider platforms")
	}

	var provider *models.TerraformProvider
	if input.ProviderID != nil {
		provider, err = s.getProviderByID(ctx, *input.ProviderID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get provider by ID")
			return nil, err
		}
	} else if input.ProviderVersionID != nil {
		providerVersion, pvErr := s.getProviderVersionByID(ctx, *input.ProviderVersionID)
		if pvErr != nil {
			tracing.RecordError(span, err, "failed to get provider version by ID")
			return nil, err
		}

		provider, err = s.getProviderByID(ctx, providerVersion.ProviderID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get provider by ID")
			return nil, err
		}
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
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
		tracing.RecordError(span, err, "failed to get provider platforms")
		return nil, err
	}

	return response, nil
}

func (s *service) CreateProviderPlatform(ctx context.Context, input *CreateProviderPlatformInput) (*models.TerraformProviderPlatform, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateProviderPlatform")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, input.ProviderVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateProviderPlatform: %v", txErr)
		}
	}()

	createdPlatform, err := s.dbClient.TerraformProviderPlatforms.CreateProviderPlatform(txContext, &models.TerraformProviderPlatform{
		ProviderVersionID: input.ProviderVersionID,
		OperatingSystem:   input.OperatingSystem,
		Architecture:      input.Architecture,
		SHASum:            input.SHASum,
		Filename:          input.Filename,
		BinaryUploaded:    false,
		CreatedBy:         caller.GetSubject(),
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to create terraform provider platform")
		return nil, err
	}

	// Get the number of platforms for this provider version to check whether we just violated the limit.
	newPlatforms, err := s.dbClient.TerraformProviderPlatforms.GetProviderPlatforms(txContext, &db.GetProviderPlatformsInput{
		Filter: &db.TerraformProviderPlatformFilter{
			ProviderVersionID: &createdPlatform.ProviderVersionID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider's platforms")
		return nil, err
	}
	if err = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitPlatformsPerTerraformProviderVersion, newPlatforms.PageInfo.TotalCount); err != nil {
		tracing.RecordError(span, err, "limit check failed")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return createdPlatform, nil
}

func (s *service) DeleteProviderPlatform(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteProviderPlatform")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerPlatform.ProviderVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	return s.dbClient.TerraformProviderPlatforms.DeleteProviderPlatform(ctx, providerPlatform)
}

func (s *service) UploadProviderPlatformBinary(ctx context.Context, providerPlatformID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadProviderPlatformBinary")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	providerPlatform, err := s.getProviderPlatformByID(ctx, providerPlatformID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform by ID")
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerPlatform.ProviderVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if providerPlatform.BinaryUploaded {
		return errors.New("binary already uploaded", errors.WithErrorCode(errors.EConflict))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to update provider platform")
		return err
	}

	if err := s.registryStore.UploadProviderPlatformBinary(ctx, providerPlatform, providerVersion, provider, reader); err != nil {
		tracing.RecordError(span, err, "failed to upload provider platform binary")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) UploadProviderVersionReadme(ctx context.Context, providerVersionID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadProviderVersionReadme")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if providerVersion.ReadmeUploaded {
		return errors.New("README file already uploaded", errors.WithErrorCode(errors.EConflict))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to update provider version")
		return err
	}

	if err := s.registryStore.UploadProviderVersionReadme(ctx, providerVersion, provider, reader); err != nil {
		tracing.RecordError(span, err, "failed to upload provider version readme")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) UploadProviderVersionSHA256Sums(ctx context.Context, providerVersionID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadProviderVersionSHA256Sums")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if providerVersion.SHASumsUploaded {
		return errors.New("shasums file already uploaded", errors.WithErrorCode(errors.EConflict))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to update provider version")
		return err
	}

	if err := s.registryStore.UploadProviderVersionSHASums(ctx, providerVersion, provider, reader); err != nil {
		tracing.RecordError(span, err, "failed to upload provider version SHA sums")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) UploadProviderVersionSHA256SumsSignature(ctx context.Context, providerVersionID string, reader io.Reader) error {
	ctx, span := tracer.Start(ctx, "svc.UploadProviderVersionSHA256SumsSignature")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTerraformProviderPermission, auth.WithGroupID(provider.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if providerVersion.SHASumsSignatureUploaded {
		return errors.New("shasums signature file already uploaded", errors.WithErrorCode(errors.EConflict))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
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
		tracing.RecordError(span, err, "failed to read gpg signature")
		return errors.Wrap(err, "failed to read gpg signature", errors.WithErrorCode(errors.EInvalid))
	}

	key, ok := pkt.(*packet.Signature)
	if !ok {
		return errors.Wrap(err, "gpg signature is not in valid format", errors.WithErrorCode(errors.EInvalid))
	}

	// GPG key id is used to lookup the trusted GPG public key
	gpgKeyID := key.IssuerKeyId

	// Get the group that this provider is in
	group, err := s.dbClient.Groups.GetGroupByID(ctx, provider.GroupID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get group by ID")
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
		return errors.Wrap(err, "a trusted gpg key for key id %d does not exist", gpgKeyID, errors.WithErrorCode(errors.EInvalid))
	}

	gpgKey := searchKeyResult.GPGKeys[0]

	// Update DB before object storage. If the object storage write fails, the DB transaction will be rolled back
	providerVersion.GPGKeyID = &gpgKey.GPGKeyID
	providerVersion.GPGASCIIArmor = &gpgKey.ASCIIArmor
	providerVersion.SHASumsSignatureUploaded = true
	if _, err := s.dbClient.TerraformProviderVersions.UpdateProviderVersion(txContext, providerVersion); err != nil {
		tracing.RecordError(span, err, "failed to update provider version")
		return err
	}

	if err := s.registryStore.UploadProviderVersionSHASumsSignature(ctx, providerVersion, provider, &sigBuffer); err != nil {
		tracing.RecordError(span, err, "failed to upload provider version SHA sums signature")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) GetProviderPlatformDownloadURLs(ctx context.Context, providerPlatform *models.TerraformProviderPlatform) (*ProviderPlatformDownloadURLs, error) {
	ctx, span := tracer.Start(ctx, "svc.GetProviderPlatformDownloadURLs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	providerVersion, err := s.getProviderVersionByID(ctx, providerPlatform.ProviderVersionID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version by ID")
		return nil, err
	}

	provider, err := s.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider by ID")
		return nil, err
	}

	if provider.Private {
		err = caller.RequireAccessToInheritableResource(ctx, types.TerraformProviderModelType, auth.WithGroupID(provider.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "inheritable resource access check failed")
			return nil, err
		}
	}

	downloadURL, err := s.registryStore.GetProviderPlatformBinaryPresignedURL(ctx, providerPlatform, providerVersion, provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider platform binary presigned URL")
		return nil, err
	}

	shaSumsURL, err := s.registryStore.GetProviderVersionSHASumsPresignedURL(ctx, providerVersion, provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version SHA sums presigned URL")
		return nil, err
	}

	shaSumsSignatureURL, err := s.registryStore.GetProviderVersionSHASumsSignaturePresignedURL(ctx, providerVersion, provider)
	if err != nil {
		tracing.RecordError(span, err, "failed to get provider version SHA sums signature presigned URL")
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
		return nil, errors.New("provider platform with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	return platform, nil
}

func (s *service) getProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error) {
	provider, err := s.dbClient.TerraformProviders.GetProviderByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if provider == nil {
		return nil, errors.New("provider with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	return provider, nil
}

func (s *service) getProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	version, err := s.dbClient.TerraformProviderVersions.GetProviderVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if version == nil {
		return nil, errors.New("provider version with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	return version, nil
}
