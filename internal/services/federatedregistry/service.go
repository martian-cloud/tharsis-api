// Package federatedregistry package
package federatedregistry

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetFederatedRegistriesInput is the input for querying a list of federated registries
type GetFederatedRegistriesInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.FederatedRegistrySortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// Hostname filters to those with the specified registry hostname
	Hostname *string
	// GroupPath filters to those with the specified group path.
	GroupPath *string
}

// Token is the token returned for a federated registry host
type Token struct {
	Token    string
	Hostname string
}

// Service implements support for (client side) federated registries
type Service interface {
	GetFederatedRegistriesByIDs(ctx context.Context, ids []string) ([]*models.FederatedRegistry, error)
	GetFederatedRegistryByID(ctx context.Context, id string) (*models.FederatedRegistry, error)
	GetFederatedRegistries(ctx context.Context, input *GetFederatedRegistriesInput) (*db.FederatedRegistriesResult, error)
	CreateFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) (*models.FederatedRegistry, error)
	UpdateFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) (*models.FederatedRegistry, error)
	DeleteFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) error
	CreateFederatedRegistryTokensForJob(ctx context.Context, jobID string) ([]*Token, error)
}

type service struct {
	logger           logger.Logger
	dbClient         *db.Client
	limitChecker     limits.LimitChecker
	activityService  activityevent.Service
	identityProvider auth.IdentityProvider
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
	identityProvider auth.IdentityProvider,
) Service {
	return newService(
		logger,
		dbClient,
		limitChecker,
		activityService,
		identityProvider,
	)
}

func newService(
	logger logger.Logger,
	dbClient *db.Client,
	limitChecker limits.LimitChecker,
	activityService activityevent.Service,
	identityProvider auth.IdentityProvider,
) Service {
	return &service{
		logger,
		dbClient,
		limitChecker,
		activityService,
		identityProvider,
	}
}

func (s *service) CreateFederatedRegistryTokensForJob(ctx context.Context, jobID string) ([]*Token, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateFederatedRegistryTokensForJob")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, permissions.CreateFederatedRegistryTokenPermission, auth.WithJobID(jobID)); err != nil {
		return nil, errors.Wrap(err, "caller lacks permission to create federated registry tokens", errors.WithSpan(span))
	}

	job, err := s.dbClient.Jobs.GetJobByID(ctx, jobID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get job", errors.WithSpan(span))
	}
	if job == nil {
		return nil, errors.New("job with ID %s not found",
			jobID, errors.WithErrorCode(errors.ENotFound))
	}

	// Get workspace for the job.
	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, job.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace", errors.WithSpan(span))
	}
	if workspace == nil {
		return nil, errors.New("workspace with ID %s not found",
			job.WorkspaceID, errors.WithErrorCode(errors.ENotFound))
	}

	federatedRegistries, err := registry.GetFederatedRegistries(ctx, &registry.GetFederatedRegistriesInput{
		DBClient:  s.dbClient,
		GroupPath: workspace.GetGroupPath(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get federated registries")
	}

	response := []*Token{}
	for _, federatedRegistry := range federatedRegistries {
		// Create a token for the registry.
		token, err := registry.NewFederatedRegistryToken(ctx, &registry.FederatedRegistryTokenInput{
			FederatedRegistry: federatedRegistry,
			IdentityProvider:  s.identityProvider,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create federated registry token")
		}
		response = append(response, &Token{
			Token:    token,
			Hostname: federatedRegistry.Hostname,
		})
	}

	return response, nil
}

func (s *service) GetFederatedRegistriesByIDs(ctx context.Context, ids []string) ([]*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "svc.GetFederatedRegistriesByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	// Must get the registries in order to check permissions on any registry IDs being asked for.
	result, err := s.dbClient.FederatedRegistries.GetFederatedRegistries(ctx,
		&db.GetFederatedRegistriesInput{
			Filter: &db.FederatedRegistryFilter{
				FederatedRegistryIDs: ids,
			},
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get federated registries", errors.WithSpan(span))
	}

	// If querying by federated registry IDs, must check permission here.
	for _, registry := range result.FederatedRegistries {
		err = caller.RequireAccessToInheritableResource(ctx, permissions.FederatedRegistryResourceType,
			auth.WithGroupID(registry.GroupID))
		if err != nil {
			tracing.RecordError(span, err, "caller lacks permission to view the group for registry %s",
				gid.ToGlobalID(gid.FederatedRegistryType, registry.Metadata.ID))
			return nil, err
		}
	}

	return result.FederatedRegistries, nil
}

func (s *service) GetFederatedRegistryByID(ctx context.Context, id string) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "svc.GetFederatedRegistryByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	federatedRegistry, err := s.dbClient.FederatedRegistries.GetFederatedRegistryByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get federated registry", errors.WithSpan(span))
	}

	if federatedRegistry == nil {
		return nil, errors.New("federated registry with ID %s not found",
			id, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequireAccessToInheritableResource(ctx, permissions.FederatedRegistryResourceType,
		auth.WithGroupID(federatedRegistry.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller lacks permission to access the requested group")
		return nil, err
	}

	return federatedRegistry, nil
}

func (s *service) GetFederatedRegistries(ctx context.Context, input *GetFederatedRegistriesInput) (*db.FederatedRegistriesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetFederatedRegistries")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	// Only an admin user can ask for information on all registries.
	if input.GroupPath == nil && !caller.IsAdmin() {
		return nil, errors.New(
			"only admin users can query for federated registries without a local registry or group filter",
			errors.WithErrorCode(errors.EForbidden),
		)
	}

	// If seeking based on a group path, can check permissions here.
	var searchByGroupPaths []string
	if input.GroupPath != nil {
		rErr := caller.RequireAccessToInheritableResource(ctx, permissions.FederatedRegistryResourceType, auth.WithNamespacePath(*input.GroupPath))
		if rErr != nil {
			tracing.RecordError(span, err, "caller lacks permission to access the requested group")
			return nil, rErr
		}

		searchByGroupPaths = []string{*input.GroupPath}
	}

	// Must get the registries in order to check permissions on any registry IDs being asked for.
	result, err := s.dbClient.FederatedRegistries.GetFederatedRegistries(ctx,
		&db.GetFederatedRegistriesInput{
			Sort:              input.Sort,
			PaginationOptions: input.PaginationOptions,
			Filter: &db.FederatedRegistryFilter{
				Hostname:   input.Hostname,
				GroupPaths: searchByGroupPaths,
			},
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get federated registries", errors.WithSpan(span))
	}

	return result, nil
}

func (s *service) CreateFederatedRegistry(ctx context.Context,
	federatedRegistry *models.FederatedRegistry,
) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateFederatedRegistry")
	defer span.End()

	if federatedRegistry == nil {
		return nil, errors.New("federated registry cannot be nil", errors.WithSpan(span))
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, permissions.CreateFederatedRegistryPermission,
		auth.WithGroupID(federatedRegistry.GroupID),
	)
	if err != nil {
		tracing.RecordError(span, err, "caller lacks permission to view federated registry")
		return nil, err
	}

	s.logger.Infow("Requested creation of a federated registry.",
		"caller", caller.GetSubject(),
		"groupID", federatedRegistry.GroupID,
		"registryHostname", federatedRegistry.Hostname,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateFederatedRegistry: %v", txErr)
		}
	}()

	// Store federated registry in DB
	createdFederatedRegistry, err := s.dbClient.FederatedRegistries.
		CreateFederatedRegistry(txContext, federatedRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create federated registry", errors.WithSpan(span))
	}

	// Get the number of federated registries in the group to check whether we just violated the limit.
	newFederatedRegistries, gErr := s.dbClient.FederatedRegistries.
		GetFederatedRegistries(txContext, &db.GetFederatedRegistriesInput{
			Filter: &db.FederatedRegistryFilter{
				GroupID: &federatedRegistry.GroupID,
			},
		})
	if gErr != nil {
		return nil, errors.Wrap(gErr, "failed to get federated registries", errors.WithSpan(span))
	}
	if gErr = s.limitChecker.CheckLimit(txContext,
		limits.ResourceLimitFederatedRegistriesPerGroup, newFederatedRegistries.PageInfo.TotalCount); gErr != nil {
		return nil, errors.Wrap(gErr, "limit check failed", errors.WithSpan(span))
	}

	// Must get the group for its path.
	group, gErr := s.dbClient.Groups.GetGroupByID(txContext, federatedRegistry.GroupID)
	if gErr != nil {
		return nil, errors.Wrap(gErr, "failed to get group", errors.WithSpan(span))
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetFederatedRegistry,
			TargetID:      createdFederatedRegistry.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return createdFederatedRegistry, nil
}

func (s *service) UpdateFederatedRegistry(ctx context.Context,
	federatedRegistry *models.FederatedRegistry,
) (*models.FederatedRegistry, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateFederatedRegistry")
	defer span.End()

	if federatedRegistry == nil {
		return nil, errors.New("federated registry cannot be nil", errors.WithSpan(span))
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, permissions.UpdateFederatedRegistryPermission,
		auth.WithGroupID(federatedRegistry.GroupID),
	)
	if err != nil {
		tracing.RecordError(span, err, "caller lacks permission to update federated registry")
		return nil, err
	}

	s.logger.Infow("Requested update of a federated registry.",
		"caller", caller.GetSubject(),
		"groupID", federatedRegistry.GroupID,
		"registryHostname", federatedRegistry.Hostname,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateFederatedRegistry: %v", txErr)
		}
	}()

	// Update the federated registry in DB
	updatedFederatedRegistry, err := s.dbClient.FederatedRegistries.
		UpdateFederatedRegistry(txContext, federatedRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update federated registry", errors.WithSpan(span))
	}

	// Must get the group for its path.
	group, gErr := s.dbClient.Groups.GetGroupByID(txContext, federatedRegistry.GroupID)
	if gErr != nil {
		return nil, errors.Wrap(gErr, "failed to get group", errors.WithSpan(span))
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetFederatedRegistry,
			TargetID:      updatedFederatedRegistry.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to update activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return updatedFederatedRegistry, nil
}

func (s *service) DeleteFederatedRegistry(ctx context.Context, federatedRegistry *models.FederatedRegistry) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteFederatedRegistry")
	defer span.End()

	if federatedRegistry == nil {
		return errors.New("federated registry cannot be nil", errors.WithSpan(span))
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return errors.Wrap(err, "caller authorization failed", errors.WithSpan(span))
	}

	err = caller.RequirePermission(ctx, permissions.DeleteFederatedRegistryPermission,
		auth.WithGroupID(federatedRegistry.GroupID))
	if err != nil {
		tracing.RecordError(span, err, "caller lacks permission to delete federated registry")
		return err
	}

	s.logger.Infow("Requested deletion of a federated registry.",
		"caller", caller.GetSubject(),
		"groupID", federatedRegistry.GroupID,
		"federatedRegistryID", federatedRegistry.Metadata.ID,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer DeleteFederatedRegistry: %v", txErr)
		}
	}()

	err = s.dbClient.FederatedRegistries.DeleteFederatedRegistry(txContext, federatedRegistry)
	if err != nil {
		return errors.Wrap(err, "failed to delete federated registry", errors.WithSpan(span))
	}

	// Must get the group for its path.
	group, gErr := s.dbClient.Groups.GetGroupByID(txContext, federatedRegistry.GroupID)
	if gErr != nil {
		return errors.Wrap(gErr, "failed to get group", errors.WithSpan(span))
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			NamespacePath: &group.FullPath,
			Action:        models.ActionDeleteChildResource,
			TargetType:    models.TargetGroup,
			TargetID:      federatedRegistry.GroupID,
		}); err != nil {
		return errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	return nil
}
