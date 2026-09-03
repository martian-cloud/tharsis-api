// Package policy implements the CRUD logic for policies
package policy

//go:generate go tool mockery --name Service --inpackage --case underscore

import (
	"context"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	corepolicy "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/policy"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	nsutils "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// CreatePolicyInput is the input for creating a new policy. The policy is always owned by a group
// (GroupID is required). Kind identifies the policy engine, and the matching kind data must be
// provided: OPAData for OPA policies, ModuleAttestationData for module attestation policies. Scope
// controls where the policy fires; an empty Scope creates a policy that applies everywhere under the
// owning group.
type CreatePolicyInput struct {
	GroupID                  string // required: the owning group ID
	Name                     string // required: human-readable identifier, unique within the group
	Description              *string
	Kind                     models.PolicyKind
	OPAData                  *models.OPAPolicyData
	ModuleAttestationData    *models.ModuleAttestationPolicyData
	Scope                    []*models.ScopeRule
	RequiredApprovals        int
	AllowedUserIDs           []string
	AllowedServiceAccountIDs []string
	AllowedTeamIDs           []string
}

// GetPoliciesInput lists policies for a group.
type GetPoliciesInput struct {
	Sort              *db.PolicySortableField
	PaginationOptions *pagination.Options
	// Group scopes the results to a single group. When IncludeInherited is true, its full path is
	// expanded so policies owned by ancestor groups are returned as well.
	Group *models.Group
	// IncludeInherited additionally returns policies owned by ancestor groups.
	IncludeInherited bool
}

// UpdatePolicyInput is the input for updating the mutable fields of an existing policy.
type UpdatePolicyInput struct {
	ID                       string
	Description              *string
	OPAData                  *models.OPAPolicyData
	ModuleAttestationData    *models.ModuleAttestationPolicyData
	Scope                    *[]*models.ScopeRule
	RequiredApprovals        *int
	AllowedUserIDs           *[]string
	AllowedServiceAccountIDs *[]string
	AllowedTeamIDs           *[]string
}

// Validate returns an error if the input could not produce a usable policy: every scope rule must be
// complete, and OPAData — which replaces the stored data wholesale when it is supplied — must carry a
// package source and a stage the run engine evaluates policies at. The remaining fields are checked by
// the model's own Validate once they have been applied to it.
func (u *UpdatePolicyInput) Validate() error {
	if u.Scope != nil {
		if err := validateScopeRules(*u.Scope); err != nil {
			return err
		}
	}

	if u.OPAData != nil {
		if u.OPAData.PackageSource == "" {
			return errors.New("packageSource is required when opaData is supplied", errors.WithErrorCode(errors.EInvalid))
		}
		if !u.OPAData.Stage.IsValid() {
			return errors.New("policy stage %s is not a valid run stage", u.OPAData.Stage, errors.WithErrorCode(errors.EInvalid))
		}
	}

	if u.ModuleAttestationData != nil && u.ModuleAttestationData.PublicKey == "" {
		return errors.New("publicKey is required when moduleAttestationData is supplied",
			errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// Service implements policy-specific functionality over packages.
type Service interface {
	GetPolicies(ctx context.Context, input *GetPoliciesInput) (*db.PoliciesResult, error)
	GetPolicyByID(ctx context.Context, id string) (*models.Policy, error)
	GetPolicyByTRN(ctx context.Context, trn string) (*models.Policy, error)
	// GetPoliciesByIDs returns the policies with the given IDs, in no particular order. It backs the
	// GraphQL policy loader, so it exists to collapse a field resolved once per row into one query.
	GetPoliciesByIDs(ctx context.Context, ids []string) ([]models.Policy, error)
	// GetWorkspaceAssignedPolicies returns every policy that would fire on a run in the workspace,
	// filtered by each policy's scope rules.
	GetWorkspaceAssignedPolicies(ctx context.Context, workspaceID string) ([]models.Policy, error)
	// GetPoliciesReferencingManagedIdentity returns the policies that name the managed identity in a
	// managed identity scope rule
	GetPoliciesReferencingManagedIdentity(ctx context.Context, managedIdentityID string) ([]models.Policy, error)
	CreatePolicy(ctx context.Context, input *CreatePolicyInput) (*models.Policy, error)
	UpdatePolicy(ctx context.Context, input *UpdatePolicyInput) (*models.Policy, error)
	DeletePolicy(ctx context.Context, policy *models.Policy) error
}

type service struct {
	logger       logger.Logger
	dbClient     *db.Client
	limitChecker limits.LimitChecker
}

// NewService creates an instance of Service.
func NewService(logger logger.Logger, dbClient *db.Client, limitChecker limits.LimitChecker) Service {
	return &service{
		logger:       logger,
		dbClient:     dbClient,
		limitChecker: limitChecker,
	}
}

/* Policies */

func (s *service) GetPolicies(ctx context.Context, input *GetPoliciesInput) (*db.PoliciesResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPolicies")
	defer span.End()

	if input.Group == nil {
		return nil, errors.New("a group is required to list policies", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// A policy is an inheritable resource, like a managed identity: it fires on runs in every namespace
	// below the group that owns it, so view rights held anywhere in that group's subtree are enough to
	// read it. Everything this returns is owned by the group or one of its ancestors, so all of it is
	// inherited by the namespace where the caller holds the permission.
	if err = caller.RequireAccessToInheritableResource(ctx, types.PolicyModelType,
		auth.WithGroupID(input.Group.Metadata.ID)); err != nil {
		return nil, err
	}

	filter := &db.PolicyFilter{}

	if input.IncludeInherited {
		groupPaths := nsutils.ExpandPath(input.Group.FullPath)
		groupsResult, gErr := s.dbClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
			Filter: &db.GroupFilter{GroupPaths: groupPaths},
		})
		if gErr != nil {
			return nil, errors.Wrap(gErr, "failed to get ancestor groups", errors.WithSpan(span))
		}
		groupIDs := make([]string, len(groupsResult.Groups))
		for i := range groupsResult.Groups {
			groupIDs[i] = groupsResult.Groups[i].Metadata.ID
		}
		filter.GroupIDs = groupIDs
	} else {
		filter.GroupIDs = []string{input.Group.Metadata.ID}
	}

	return s.dbClient.Policies.GetPolicies(ctx, &db.GetPoliciesInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            filter,
	})
}

func (s *service) GetPolicyByID(ctx context.Context, id string) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPolicyByID")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	policy, err := s.dbClient.Policies.GetPolicyByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policy by ID", errors.WithSpan(span))
	}
	if policy == nil {
		return nil, errors.New("policy with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}

	// Inheritable: a member of a namespace below the owning group inherits this policy and so may read
	// it, even without rights in the owning group itself. That is the case a run's policy check hits
	// when the policy came from an ancestor group.
	if err = caller.RequireAccessToInheritableResource(ctx, types.PolicyModelType,
		auth.WithGroupID(policy.GroupID)); err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *service) GetPolicyByTRN(ctx context.Context, trnValue string) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPolicyByTRN")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	policy, err := s.dbClient.Policies.GetPolicyByTRN(ctx, trnValue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policy by TRN", errors.WithSpan(span))
	}
	if policy == nil {
		return nil, errors.New("policy with TRN %s not found", trnValue, errors.WithErrorCode(errors.ENotFound))
	}

	if err = caller.RequireAccessToInheritableResource(ctx, types.PolicyModelType,
		auth.WithGroupID(policy.GroupID)); err != nil {
		return nil, err
	}

	return policy, nil
}

// GetPoliciesByIDs is all-or-nothing on access, matching GetManagedIdentitiesByIDs: one unreadable
// policy fails the batch rather than being dropped from it, so a caller cannot use the loader to learn
// which of a set of IDs exist. In practice a batch is uniform — the policies on a run are all owned by
// the workspace's ancestor groups, and view rights held at the workspace satisfy the inheritable check
// for every one of them.
//
// IDs that match no policy are simply absent from the result; the loader turns those into ENotFound for
// the requesting field alone.
func (s *service) GetPoliciesByIDs(ctx context.Context, ids []string) ([]models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPoliciesByIDs")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.dbClient.Policies.GetPolicies(ctx, &db.GetPoliciesInput{
		Filter: &db.PolicyFilter{PolicyIDs: ids},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policies", errors.WithSpan(span))
	}

	// Deduplicated because a batch usually holds several policies from the same group, and each distinct
	// path costs a namespace membership query.
	seen := make(map[string]struct{}, len(result.Policies))
	groupPaths := make([]string, 0, len(result.Policies))
	for i := range result.Policies {
		groupPath := result.Policies[i].GetGroupPath()
		if _, ok := seen[groupPath]; ok {
			continue
		}
		seen[groupPath] = struct{}{}
		groupPaths = append(groupPaths, groupPath)
	}

	if len(groupPaths) > 0 {
		if err = caller.RequireAccessToInheritableResource(ctx, types.PolicyModelType,
			auth.WithNamespacePaths(groupPaths)); err != nil {
			return nil, err
		}
	}

	policies := make([]models.Policy, len(result.Policies))
	for i, p := range result.Policies {
		policies[i] = *p
	}

	return policies, nil
}

func (s *service) GetWorkspaceAssignedPolicies(ctx context.Context, workspaceID string) ([]models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetWorkspaceAssignedPolicies")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ViewPolicyPermission, auth.WithWorkspaceID(workspaceID)); err != nil {
		return nil, err
	}

	workspace, err := s.dbClient.Workspaces.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace", errors.WithSpan(span))
	}
	if workspace == nil {
		return nil, errors.New("workspace not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	managedIdentities, err := s.dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get managed identities for workspace", errors.WithSpan(span))
	}

	return corepolicy.GetWorkspaceAssignedPolicies(ctx, s.dbClient, workspace, managedIdentities, nil)
}

// GetPoliciesReferencingManagedIdentity is gated on viewing policies in the identity's own group, not on
// viewing the identity. The candidate policies come from that group, its ancestors and everything
// nested beneath it, and permission held in a group extends to every group below it, so this is the
// check that makes the whole result set readable by the caller: viewing the identity itself is
// inheritable, which a member of a subgroup satisfies without any right to that group's siblings.
//
// This is deliberately the one policy read that is not an inheritable-resource check. Unlike the other
// view functions, the result set reaches *downward* from the identity's group, and inheritance does not
// run that way — a caller holding rights only in one subgroup would be handed the policies of its
// siblings.
func (s *service) GetPoliciesReferencingManagedIdentity(ctx context.Context, managedIdentityID string) ([]models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.GetPoliciesReferencingManagedIdentity")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := s.dbClient.ManagedIdentities.GetManagedIdentityByID(ctx, managedIdentityID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get managed identity", errors.WithSpan(span))
	}
	if managedIdentity == nil {
		return nil, errors.New("managed identity not found", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if err = caller.RequirePermission(ctx, models.ViewPolicyPermission,
		auth.WithGroupID(managedIdentity.GroupID)); err != nil {
		return nil, err
	}

	return corepolicy.GetPoliciesReferencingManagedIdentity(ctx, s.dbClient, managedIdentity)
}

func (s *service) CreatePolicy(ctx context.Context, input *CreatePolicyInput) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.CreatePolicy")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if input.GroupID == "" {
		return nil, errors.New("groupID is required when creating a policy", errors.WithErrorCode(errors.EInvalid))
	}
	if input.Name == "" {
		return nil, errors.New("name is required when creating a policy", errors.WithErrorCode(errors.EInvalid))
	}

	ownerGroup, err := s.dbClient.Groups.GetGroupByID(ctx, input.GroupID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get owner group", errors.WithSpan(span))
	}
	if ownerGroup == nil {
		return nil, errors.New("group with id %s not found", input.GroupID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = caller.RequirePermission(ctx, models.CreatePolicyPermission, auth.WithGroupID(input.GroupID)); err != nil {
		return nil, err
	}

	switch input.Kind {
	case models.PolicyKindOPA:
		if input.OPAData == nil {
			return nil, errors.New("opaData is required when creating an OPA policy", errors.WithErrorCode(errors.EInvalid))
		}
		if !input.OPAData.Stage.IsValid() {
			return nil, errors.New("policy stage %s is not a valid run stage", input.OPAData.Stage, errors.WithErrorCode(errors.EInvalid))
		}
	case models.PolicyKindModuleAttestation:
		if input.ModuleAttestationData == nil {
			return nil, errors.New("moduleAttestationData is required when creating a module attestation policy",
				errors.WithErrorCode(errors.EInvalid))
		}
	default:
		return nil, errors.New("policy kind %s is not supported", input.Kind, errors.WithErrorCode(errors.EInvalid))
	}

	if err = validateScopeRules(input.Scope); err != nil {
		return nil, errors.Wrap(err, "invalid scope rules", errors.WithSpan(span))
	}

	if err = s.verifyServiceAccountAccessForGroup(ctx, input.AllowedServiceAccountIDs, ownerGroup.FullPath); err != nil {
		return nil, err
	}

	toCreate := &models.Policy{
		GroupID:                  input.GroupID,
		Name:                     input.Name,
		Description:              input.Description,
		Kind:                     input.Kind,
		OPAData:                  input.OPAData,
		ModuleAttestationData:    input.ModuleAttestationData,
		Scope:                    input.Scope,
		RequiredApprovals:        input.RequiredApprovals,
		AllowedUserIDs:           input.AllowedUserIDs,
		AllowedServiceAccountIDs: input.AllowedServiceAccountIDs,
		AllowedTeamIDs:           input.AllowedTeamIDs,
		CreatedBy:                caller.GetSubject(),
	}
	if vErr := toCreate.Validate(); vErr != nil {
		return nil, errors.Wrap(vErr, "failed to validate policy", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreatePolicy: %v", txErr)
		}
	}()

	created, err := s.dbClient.Policies.CreatePolicy(txContext, toCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create policy", errors.WithSpan(span))
	}

	policiesResult, pErr := s.dbClient.Policies.GetPolicies(txContext, &db.GetPoliciesInput{
		Filter: &db.PolicyFilter{GroupIDs: []string{input.GroupID}},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(0),
		},
	})
	if pErr != nil {
		return nil, errors.Wrap(pErr, "failed to query policies in namespace", errors.WithSpan(span))
	}
	if lErr := s.limitChecker.CheckLimit(txContext, limits.ResourceLimitPoliciesPerGroup, policiesResult.PageInfo.TotalCount); lErr != nil {
		return nil, errors.Wrap(lErr, "failed to check limit for policies per namespace", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &ownerGroup.FullPath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetPolicy,
		TargetID:      created.Metadata.ID,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a policy.",
		"name", input.Name,
		"kind", input.Kind,
		"ownerGroup", ownerGroup.FullPath,
		"policyTRN", created.Metadata.TRN,
	)

	return created, nil
}

func (s *service) UpdatePolicy(ctx context.Context, input *UpdatePolicyInput) (*models.Policy, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdatePolicy")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	policy, err := s.dbClient.Policies.GetPolicyByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get policy by ID", errors.WithSpan(span))
	}
	if policy == nil {
		return nil, errors.New("policy with id %s not found", input.ID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = caller.RequirePermission(ctx, models.UpdatePolicyPermission, auth.WithGroupID(policy.GroupID)); err != nil {
		return nil, err
	}

	ownerGroup, err := s.dbClient.Groups.GetGroupByID(ctx, policy.GroupID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get owner group", errors.WithSpan(span))
	}
	if ownerGroup == nil {
		return nil, errors.New("group with id %s not found", policy.GroupID, errors.WithErrorCode(errors.ENotFound))
	}

	if err = input.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid update policy input", errors.WithSpan(span))
	}

	if input.AllowedServiceAccountIDs != nil {
		if err = s.verifyServiceAccountAccessForGroup(ctx, *input.AllowedServiceAccountIDs, ownerGroup.FullPath); err != nil {
			return nil, err
		}
	}

	// A change to the enforcement level invalidates the approval configuration: approvals gate the
	// override of a run blocked at soft mandatory (see models.Policy.Validate, which rejects approvers
	// at any other level), so a level change either strands them or silently re-arms them. Whatever the
	// caller did not supply is therefore reset rather than carried over, which is what lets a caller
	// move a policy off soft mandatory by naming only the new level. Anything the caller did supply
	// still wins, so changing the level and setting approvals in one call is honoured as sent.
	//
	// Computed before the kind data below replaces the stored level, and only meaningful when the caller
	// supplied kind data at all — a partial update that leaves the level alone changes nothing here.
	var suppliedEnforcementLevel models.PolicyEnforcementLevel
	switch policy.Kind {
	case models.PolicyKindOPA:
		if input.OPAData != nil {
			suppliedEnforcementLevel = input.OPAData.EnforcementLevel
		}
	case models.PolicyKindModuleAttestation:
		if input.ModuleAttestationData != nil {
			suppliedEnforcementLevel = input.ModuleAttestationData.EnforcementLevel
		}
	}
	enforcementLevelChanged := suppliedEnforcementLevel != "" && suppliedEnforcementLevel != policy.EnforcementLevel()

	// Apply only the fields the caller supplied. Description is a pointer whose nil already means
	// "clear it" in this API, so it is assigned as-is; the remaining pointers distinguish "not
	// provided" (nil, leave the stored value) from "provided" (replace it, an empty slice clearing it).
	policy.Description = input.Description
	if input.Scope != nil {
		policy.Scope = *input.Scope
	}
	if input.RequiredApprovals != nil {
		policy.RequiredApprovals = *input.RequiredApprovals
	} else if enforcementLevelChanged {
		policy.RequiredApprovals = 0
	}
	if input.AllowedUserIDs != nil {
		policy.AllowedUserIDs = *input.AllowedUserIDs
	} else if enforcementLevelChanged {
		policy.AllowedUserIDs = nil
	}
	if input.AllowedServiceAccountIDs != nil {
		policy.AllowedServiceAccountIDs = *input.AllowedServiceAccountIDs
	} else if enforcementLevelChanged {
		policy.AllowedServiceAccountIDs = nil
	}
	if input.AllowedTeamIDs != nil {
		policy.AllowedTeamIDs = *input.AllowedTeamIDs
	} else if enforcementLevelChanged {
		policy.AllowedTeamIDs = nil
	}

	// Update the mutable kind-specific fields. A stage change only affects runs created afterwards:
	// a run's policy checks are built when the run starts, from the stage each policy had then. Kind
	// itself is immutable, so data of the other kind is ignored rather than switching the policy over.
	if policy.Kind == models.PolicyKindOPA && input.OPAData != nil && policy.OPAData != nil {
		policy.OPAData.PackageSource = input.OPAData.PackageSource
		policy.OPAData.PackageVersionConstraint = input.OPAData.PackageVersionConstraint
		policy.OPAData.PackageDigest = input.OPAData.PackageDigest
		policy.OPAData.EnforcementLevel = input.OPAData.EnforcementLevel
		policy.OPAData.SpeculativeRunEnforcementLevel = input.OPAData.SpeculativeRunEnforcementLevel
		policy.OPAData.Stage = input.OPAData.Stage
	}
	if policy.Kind == models.PolicyKindModuleAttestation && input.ModuleAttestationData != nil && policy.ModuleAttestationData != nil {
		policy.ModuleAttestationData.PublicKey = input.ModuleAttestationData.PublicKey
		policy.ModuleAttestationData.PredicateType = input.ModuleAttestationData.PredicateType
		policy.ModuleAttestationData.VerifyStateLineage = input.ModuleAttestationData.VerifyStateLineage
		policy.ModuleAttestationData.EnforcementLevel = input.ModuleAttestationData.EnforcementLevel
		policy.ModuleAttestationData.SpeculativeRunEnforcementLevel = input.ModuleAttestationData.SpeculativeRunEnforcementLevel
		policy.ModuleAttestationData.Stage = input.ModuleAttestationData.Stage
	}

	if vErr := policy.Validate(); vErr != nil {
		return nil, errors.Wrap(vErr, "failed to validate policy", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdatePolicy: %v", txErr)
		}
	}()

	updated, err := s.dbClient.Policies.UpdatePolicy(txContext, policy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update policy", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &ownerGroup.FullPath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetPolicy,
		TargetID:      updated.Metadata.ID,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a policy.",
		"policyID", input.ID,
		"policyTRN", updated.Metadata.TRN,
	)

	return updated, nil
}

func (s *service) DeletePolicy(ctx context.Context, policy *models.Policy) error {
	ctx, span := tracer.Start(ctx, "svc.DeletePolicy")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	if err = caller.RequirePermission(ctx, models.DeletePolicyPermission, auth.WithGroupID(policy.GroupID)); err != nil {
		return err
	}

	ownerGroup, err := s.dbClient.Groups.GetGroupByID(ctx, policy.GroupID)
	if err != nil {
		return errors.Wrap(err, "failed to get owner group", errors.WithSpan(span))
	}
	if ownerGroup == nil {
		return errors.New("group with id %s not found", policy.GroupID, errors.WithErrorCode(errors.ENotFound))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer DeletePolicy: %v", txErr)
		}
	}()

	if err = s.dbClient.Policies.DeletePolicy(txContext, policy); err != nil {
		return errors.Wrap(err, "failed to delete policy", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient, &activity.CreateActivityEventInput{
		NamespacePath: &ownerGroup.FullPath,
		Action:        models.ActionDeleteChildResource,
		TargetType:    models.TargetGroup,
		TargetID:      policy.GroupID,
		Payload: &models.ActivityEventDeleteChildResourcePayload{
			Name: policy.Name,
			ID:   policy.Metadata.ID,
			Type: string(models.TargetPolicy),
		},
	}); err != nil {
		return errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a policy.",
		"policyID", policy.Metadata.ID,
		"policyName", policy.Name,
		"policyTRN", policy.Metadata.TRN,
	)

	return nil
}

// verifyServiceAccountAccessForGroup rejects approver service accounts the owning group cannot reach,
// mirroring the managed identity service's check of the same name. A service account is in scope when it
// lives in the owning group itself or in one of its ancestors, which is the same direction inheritance
// runs: a descendant can use an ancestor's service account, never the other way round.
//
// Without this, a policy could name an approver from an unrelated subtree. That is not just untidy — it
// would let a group grant override authority to a principal its own members cannot see, and it makes the
// approver lists unresolvable for callers reading the policy from a descendant namespace.
func (s *service) verifyServiceAccountAccessForGroup(ctx context.Context, serviceAccountIDs []string, groupPath string) error {
	if len(serviceAccountIDs) == 0 {
		return nil
	}

	// One query for the whole approver list rather than a round trip per ID. Pagination is left nil
	// deliberately: the filter is an exact ID list, so every matching row is wanted.
	result, err := s.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
		Filter: &db.ServiceAccountFilter{ServiceAccountIDs: serviceAccountIDs},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get allowed service accounts")
	}

	found := make(map[string]*models.ServiceAccount, len(result.ServiceAccounts))
	for i := range result.ServiceAccounts {
		sa := &result.ServiceAccounts[i]
		found[sa.Metadata.ID] = sa
	}

	// Walked in input order, not result order, so a list with more than one bad ID always reports the
	// same one -- the query returns rows in ID order, which has nothing to do with what the caller sent.
	for _, id := range serviceAccountIDs {
		sa, ok := found[id]
		if !ok {
			return errors.New("service account with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
		}

		saGroupPath := sa.GetGroupPath()

		if groupPath != saGroupPath && !nsutils.IsDescendantOfPath(groupPath, saGroupPath) {
			return errors.New("service account %s is outside the scope of group %s", sa.GetResourcePath(), groupPath, errors.WithErrorCode(errors.EInvalid))
		}
	}

	return nil
}

// validateScopeRules validates each scope rule's type, action, and pattern. Every check here rejects
// a rule that would store successfully and then never fire.
func validateScopeRules(rules []*models.ScopeRule) error {
	for i, r := range rules {
		switch r.Action {
		case models.ScopeRuleActionInclude, models.ScopeRuleActionExclude:
		default:
			return errors.New("scope rule %d has invalid action %q", i, r.Action, errors.WithErrorCode(errors.EInvalid))
		}

		expectedTRNType, ok := r.Type.TRNType()
		if !ok {
			return errors.New("scope rule %d has invalid type %q", i, r.Type, errors.WithErrorCode(errors.EInvalid))
		}

		if r.Pattern == "" {
			return errors.New("scope rule %d of type %q requires a non-empty pattern", i, r.Type, errors.WithErrorCode(errors.EInvalid))
		}

		// A pattern may be written as a TRN, but only one naming the kind of resource the rule matches
		// against: anything else strips to a path of the wrong shape.
		if trn.IsTRN(r.Pattern) {
			parsed, err := trn.ParseAny(r.Pattern)
			if err != nil {
				return errors.New("scope rule %d has a malformed TRN pattern %q", i, r.Pattern, errors.WithErrorCode(errors.EInvalid))
			}
			if parsed.Type() != expectedTRNType {
				return errors.New("scope rule %d of type %q accepts a %q TRN, got %q", i, r.Type,
					expectedTRNType, parsed.Type(), errors.WithErrorCode(errors.EInvalid))
			}
		}
	}
	return nil
}
