// Package cleanuppolicy package
package cleanuppolicy

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// CreateCleanupPolicyInput is the input for creating a cleanup policy.
type CreateCleanupPolicyInput struct {
	NamespacePath               string
	Disabled                    bool
	Kind                        models.CleanupRuleKind
	TerraformModulePolicyData   *models.TerraformModuleCleanupPolicyData
	TerraformProviderPolicyData *models.TerraformProviderCleanupPolicyData
	RunPolicyData               *models.RunCleanupPolicyData
}

// UpdateCleanupPolicyInput is the input for updating a cleanup policy.
type UpdateCleanupPolicyInput struct {
	ID                          string
	Version                     *int
	Disabled                    *bool
	TerraformModulePolicyData   *models.TerraformModuleCleanupPolicyData
	TerraformProviderPolicyData *models.TerraformProviderCleanupPolicyData
	RunPolicyData               *models.RunCleanupPolicyData
}

// DeleteCleanupPolicyInput is the input for deleting a cleanup policy.
type DeleteCleanupPolicyInput struct {
	ID      string
	Version *int
}

// Service implements all cleanup policy related functionality
type Service interface {
	GetCleanupPolicyByID(ctx context.Context, id string) (*models.CleanupPolicy, error)
	GetCleanupPoliciesByIDs(ctx context.Context, ids []string) ([]*models.CleanupPolicy, error)
	GetCleanupPolicyByTRN(ctx context.Context, trnValue string) (*models.CleanupPolicy, error)
	GetEffectiveCleanupPolicies(ctx context.Context, namespacePath string) ([]*models.CleanupPolicy, error)
	CreateCleanupPolicy(ctx context.Context, input *CreateCleanupPolicyInput) (*models.CleanupPolicy, error)
	UpdateCleanupPolicy(ctx context.Context, input *UpdateCleanupPolicyInput) (*models.CleanupPolicy, error)
	DeleteCleanupPolicy(ctx context.Context, input *DeleteCleanupPolicyInput) error
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

func (s *service) GetCleanupPolicyByID(ctx context.Context, id string) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetCleanupPolicyByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	policy, err := s.dbClient.CleanupPolicies.GetCleanupPolicyByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cleanup policy by ID", errors.WithSpan(span))
	}

	if policy == nil {
		return nil, errors.New("cleanup policy with id %s not found", id, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequireAccessToInheritableResource(ctx, types.CleanupPolicyModelType, auth.WithNamespacePath(policy.NamespacePath())); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	return policy, nil
}

func (s *service) GetCleanupPolicyByTRN(ctx context.Context, trnValue string) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetCleanupPolicyByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	policy, err := s.dbClient.CleanupPolicies.GetCleanupPolicyByTRN(ctx, trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cleanup policy by TRN", errors.WithSpan(span))
	}

	if policy == nil {
		return nil, errors.New("cleanup policy with TRN %s not found", trnValue, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequireAccessToInheritableResource(ctx, types.CleanupPolicyModelType, auth.WithNamespacePath(policy.NamespacePath())); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	return policy, nil
}

func (s *service) GetCleanupPoliciesByIDs(ctx context.Context, ids []string) ([]*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetCleanupPoliciesByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.CleanupPolicies.GetCleanupPolicies(ctx, &db.GetCleanupPoliciesInput{
		CleanupPolicyIDs: ids,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cleanup policies", errors.WithSpan(span))
	}

	for _, p := range result {
		if err = caller.RequireAccessToInheritableResource(ctx, types.CleanupPolicyModelType, auth.WithNamespacePath(p.NamespacePath())); err != nil {
			return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
		}
	}

	return result, nil
}

func (s *service) GetEffectiveCleanupPolicies(ctx context.Context, namespacePath string) ([]*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetEffectiveCleanupPolicies")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequireAccessToInheritableResource(ctx, types.CleanupPolicyModelType, auth.WithNamespacePath(namespacePath)); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	// A single query fetches policies for all levels; we then pick the closest per kind in memory.
	allPolicies, err := s.dbClient.CleanupPolicies.GetCleanupPolicies(ctx, &db.GetCleanupPoliciesInput{
		NamespacePaths: utils.ExpandPath(namespacePath),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cleanup policies", errors.WithSpan(span))
	}

	// Pick the closest-ancestor policy per kind.
	covered := map[models.CleanupRuleKind]*models.CleanupPolicy{}
	for _, p := range allPolicies {
		existing, ok := covered[p.Kind]
		if ok && !utils.IsDescendantOfPath(p.NamespacePath(), existing.NamespacePath()) {
			// If we already have a policy for this kind and
			// the candidate is not a descendant of (i.e. not more specific than) the existing one, skip it.
			continue
		}

		covered[p.Kind] = p
	}

	results := make([]*models.CleanupPolicy, 0, len(covered))
	for _, p := range covered {
		results = append(results, p)
	}

	return results, nil
}

func (s *service) CreateCleanupPolicy(ctx context.Context, input *CreateCleanupPolicyInput) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateCleanupPolicy")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.CreateCleanupPolicyPermission, auth.WithNamespacePath(input.NamespacePath)); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	ns, err := s.resolveNamespace(ctx, input.NamespacePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve namespace path", errors.WithSpan(span))
	}

	toCreate := &models.CleanupPolicy{
		Kind:                        input.Kind,
		Disabled:                    input.Disabled,
		TerraformModulePolicyData:   input.TerraformModulePolicyData,
		TerraformProviderPolicyData: input.TerraformProviderPolicyData,
		RunPolicyData:               input.RunPolicyData,
	}

	namespaceID := ns.GetID()
	if ns.GetModelType().Equals(types.GroupModelType) {
		toCreate.GroupID = &namespaceID
	} else {
		toCreate.WorkspaceID = &namespaceID
	}

	if vErr := toCreate.Validate(); vErr != nil {
		return nil, errors.Wrap(vErr, "cleanup policy validation failed", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateCleanupPolicy: %v", txErr)
		}
	}()

	created, err := s.dbClient.CleanupPolicies.CreateCleanupPolicy(txContext, toCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cleanup policy", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &input.NamespacePath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetCleanupPolicy,
		TargetID:      created.Metadata.ID,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a cleanup policy.", "policyTRN", created.Metadata.TRN)

	return created, nil
}

func (s *service) UpdateCleanupPolicy(ctx context.Context, input *UpdateCleanupPolicyInput) (*models.CleanupPolicy, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateCleanupPolicy")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	policy, err := s.dbClient.CleanupPolicies.GetCleanupPolicyByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cleanup policy by ID", errors.WithSpan(span))
	}

	if policy == nil {
		return nil, errors.New("cleanup policy with id %s not found", input.ID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, models.UpdateCleanupPolicyPermission, auth.WithNamespacePath(policy.NamespacePath())); err != nil {
		return nil, errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if input.Version != nil {
		policy.Metadata.Version = *input.Version
	}

	if input.Disabled != nil {
		policy.Disabled = *input.Disabled
	}

	switch policy.Kind {
	case models.CleanupRuleKindTerraformModules:
		policy.TerraformModulePolicyData = input.TerraformModulePolicyData
	case models.CleanupRuleKindTerraformProviders:
		policy.TerraformProviderPolicyData = input.TerraformProviderPolicyData
	case models.CleanupRuleKindRuns:
		policy.RunPolicyData = input.RunPolicyData
	}

	if vErr := policy.Validate(); vErr != nil {
		return nil, errors.Wrap(vErr, "cleanup policy validation failed", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateCleanupPolicy: %v", txErr)
		}
	}()

	updated, err := s.dbClient.CleanupPolicies.UpdateCleanupPolicy(txContext, policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update cleanup policy", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: new(policy.NamespacePath()),
		Action:        models.ActionUpdate,
		TargetType:    models.TargetCleanupPolicy,
		TargetID:      updated.Metadata.ID,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a cleanup policy.", "policyTRN", updated.Metadata.TRN)

	return updated, nil
}

func (s *service) DeleteCleanupPolicy(ctx context.Context, input *DeleteCleanupPolicyInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteCleanupPolicy")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	policy, err := s.dbClient.CleanupPolicies.GetCleanupPolicyByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get cleanup policy by ID", errors.WithSpan(span))
	}

	if policy == nil {
		return errors.New("cleanup policy with id %s not found", input.ID, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, models.DeleteCleanupPolicyPermission, auth.WithNamespacePath(policy.NamespacePath())); err != nil {
		return errors.Wrap(err, "permission check failed", errors.WithSpan(span))
	}

	if input.Version != nil {
		policy.Metadata.Version = *input.Version
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeleteCleanupPolicy: %v", txErr)
		}
	}()

	if err = s.dbClient.CleanupPolicies.DeleteCleanupPolicy(txContext, policy); err != nil {
		return errors.Wrap(err, "failed to delete cleanup policy", errors.WithSpan(span))
	}

	var ownerTargetType models.ActivityEventTargetType
	var ownerTargetID string
	if policy.GroupID != nil {
		ownerTargetType = models.TargetGroup
		ownerTargetID = *policy.GroupID
	} else {
		ownerTargetType = models.TargetWorkspace
		ownerTargetID = *policy.WorkspaceID
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: new(policy.NamespacePath()),
		Action:        models.ActionDeleteChildResource,
		TargetType:    ownerTargetType,
		TargetID:      ownerTargetID,
		Payload: &models.ActivityEventDeleteChildResourcePayload{
			Name: string(policy.Kind),
			ID:   policy.Metadata.ID,
			Type: string(models.TargetCleanupPolicy),
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) resolveNamespace(ctx context.Context, namespacePath string) (namespace.Namespace, error) {
	nsResult, err := s.dbClient.Namespaces.GetNamespace(ctx, namespacePath)
	if err != nil {
		return nil, err
	}

	if nsResult == nil {
		return nil, errors.New("namespace %s not found", namespacePath, errors.WithErrorCode(errors.ENotFound))
	}

	if nsResult.Type == models.NamespaceTypeGroup {
		return nsResult.Group, nil
	}

	return nsResult.Workspace, nil
}
