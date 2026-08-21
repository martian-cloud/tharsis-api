package resolver

import (
	"context"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/policy"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// resolvePolicyPrincipalIDs converts a slice of principal GIDs to their model IDs.
func resolvePolicyPrincipalIDs(ctx context.Context, gids *[]string) ([]string, error) {
	if gids == nil {
		return nil, nil
	}
	serviceCatalog := getServiceCatalog(ctx)
	ids := make([]string, 0, len(*gids))
	for _, g := range *gids {
		id, err := serviceCatalog.FetchModelID(ctx, g)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

/* PolicyScopeRule Resolver */

// PolicyScopeRuleResolver resolves a policy scope rule.
type PolicyScopeRuleResolver struct {
	rule *models.ScopeRule
}

// Type resolver
func (r *PolicyScopeRuleResolver) Type() string {
	return toGraphqlEnum(string(r.rule.Type))
}

// Action resolver
func (r *PolicyScopeRuleResolver) Action() string {
	return toGraphqlEnum(string(r.rule.Action))
}

// Pattern resolver returns the glob pattern this rule matches with.
func (r *PolicyScopeRuleResolver) Pattern() string {
	return r.rule.Pattern
}

/* Policy Resolver */

// PolicyResolver resolves a policy (group-owned, with JSONB scope rules)
type PolicyResolver struct {
	policy *models.Policy
}

// ID resolver
func (r *PolicyResolver) ID() graphql.ID {
	return graphql.ID(r.policy.GetGlobalID())
}

// Metadata resolver
func (r *PolicyResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.policy.Metadata}
}

// GroupPath resolver returns the path of the owning group. Callers that only need to name the owner
// should prefer this over group.fullPath, which has to load the group.
func (r *PolicyResolver) GroupPath() string {
	return r.policy.GetGroupPath()
}

// Group resolver returns the owning group.
func (r *PolicyResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.policy.GroupID)
	if err != nil {
		return nil, err
	}
	return &GroupResolver{group: group}, nil
}

// Name resolver
func (r *PolicyResolver) Name() string {
	return r.policy.Name
}

// Description resolver
func (r *PolicyResolver) Description() string {
	if r.policy.Description == nil {
		return ""
	}
	return *r.policy.Description
}

// Kind resolver returns the policy engine kind.
func (r *PolicyResolver) Kind() string {
	return toGraphqlEnum(string(r.policy.Kind))
}

// OPAData resolver returns OPA-specific data, or nil for non-OPA policies.
func (r *PolicyResolver) OPAData() *OPAPolicyDataResolver {
	if r.policy.OPAData == nil {
		return nil
	}
	return &OPAPolicyDataResolver{data: r.policy.OPAData}
}

// Scope resolver returns the policy's scope rules.
func (r *PolicyResolver) Scope() []*PolicyScopeRuleResolver {
	resolvers := make([]*PolicyScopeRuleResolver, len(r.policy.Scope))
	for i, rule := range r.policy.Scope {
		resolvers[i] = &PolicyScopeRuleResolver{rule: rule}
	}
	return resolvers
}

/* OPAPolicyData Resolver */

// OPAPolicyDataResolver resolves OPA-specific policy data.
type OPAPolicyDataResolver struct {
	data *models.OPAPolicyData
}

// PackageSource resolver
func (r *OPAPolicyDataResolver) PackageSource() string {
	return r.data.PackageSource
}

// PackageVersionConstraint resolver returns nil when tracking the latest version.
func (r *OPAPolicyDataResolver) PackageVersionConstraint() *string {
	if r.data.PackageVersionConstraint == nil || *r.data.PackageVersionConstraint == "" {
		return nil
	}
	return r.data.PackageVersionConstraint
}

// PackageDigest resolver returns nil when unset.
func (r *OPAPolicyDataResolver) PackageDigest() *string {
	return r.data.PackageDigest
}

// Stage resolver
func (r *OPAPolicyDataResolver) Stage() string {
	return toGraphqlEnum(string(r.data.Stage))
}

// EnforcementLevel resolver
func (r *OPAPolicyDataResolver) EnforcementLevel() string {
	return toGraphqlEnum(string(r.data.EnforcementLevel))
}

// SpeculativeRunEnforcementLevel resolver returns the level used on a run with no apply.
func (r *OPAPolicyDataResolver) SpeculativeRunEnforcementLevel() string {
	return toGraphqlEnum(string(r.data.SpeculativeRunEnforcementLevel))
}

// RequiredApprovals resolver
func (r *PolicyResolver) RequiredApprovals() int32 {
	return int32(r.policy.RequiredApprovals)
}

// AllowedUsers resolver. A user that has been deleted is omitted.
func (r *PolicyResolver) AllowedUsers(ctx context.Context) ([]*UserResolver, error) {
	resolvers := []*UserResolver{}
	for _, id := range r.policy.AllowedUserIDs {
		user, err := loadUser(ctx, id)
		if err != nil {
			if errors.ErrorCode(err) == errors.ENotFound {
				continue
			}
			return nil, err
		}
		resolvers = append(resolvers, &UserResolver{user: user})
	}
	return resolvers, nil
}

// AllowedServiceAccounts resolver. A service account that has been deleted is omitted.
func (r *PolicyResolver) AllowedServiceAccounts(ctx context.Context) ([]*ServiceAccountResolver, error) {
	resolvers := []*ServiceAccountResolver{}
	for _, id := range r.policy.AllowedServiceAccountIDs {
		sa, err := loadServiceAccount(ctx, id)
		if err != nil {
			if errors.ErrorCode(err) == errors.ENotFound {
				continue
			}
			return nil, err
		}
		resolvers = append(resolvers, &ServiceAccountResolver{serviceAccount: sa})
	}
	return resolvers, nil
}

// AllowedTeams resolver. A team that has been deleted is omitted.
func (r *PolicyResolver) AllowedTeams(ctx context.Context) ([]*TeamResolver, error) {
	resolvers := []*TeamResolver{}
	for _, id := range r.policy.AllowedTeamIDs {
		team, err := loadTeam(ctx, id)
		if err != nil {
			if errors.ErrorCode(err) == errors.ENotFound {
				continue
			}
			return nil, err
		}
		resolvers = append(resolvers, &TeamResolver{team: team})
	}
	return resolvers, nil
}

// CreatedBy resolver
func (r *PolicyResolver) CreatedBy() string {
	return r.policy.CreatedBy
}

/* Connection resolvers */

// PolicyConnectionQueryArgs are used to query a policy connection.
type PolicyConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
}

// policyConnectionQuery resolves a group's policies as a paginated connection.
func policyConnectionQuery(ctx context.Context, args *PolicyConnectionQueryArgs, input *policy.GetPoliciesInput) (*PolicyConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input.PaginationOptions = &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before}
	if args.Sort != nil {
		sort := db.PolicySortableField(*args.Sort)
		input.Sort = &sort
	}
	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	result, err := getServiceCatalog(ctx).PolicyService.GetPolicies(ctx, input)
	if err != nil {
		return nil, err
	}

	return NewPolicyConnectionResolver(result), nil
}

// PolicyEdgeResolver resolves policy edges.
type PolicyEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor.
func (r *PolicyEdgeResolver) Cursor() (string, error) {
	policy, ok := r.edge.Node.(*models.Policy)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(policy)
	return *cursor, err
}

// Node returns a policy node.
func (r *PolicyEdgeResolver) Node() (*PolicyResolver, error) {
	policy, ok := r.edge.Node.(*models.Policy)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}
	return &PolicyResolver{policy: policy}, nil
}

// PolicyConnectionResolver resolves a policy connection.
type PolicyConnectionResolver struct {
	connection Connection
}

// NewPolicyConnectionResolver creates a new PolicyConnectionResolver from a db result.
func NewPolicyConnectionResolver(result *db.PoliciesResult) *PolicyConnectionResolver {
	policies := result.Policies

	edges := make([]Edge, len(policies))
	for i := range policies {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: policies[i]}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(policies) > 0 {
		pageInfo.StartCursor, _ = result.PageInfo.Cursor(policies[0])
		pageInfo.EndCursor, _ = result.PageInfo.Cursor(policies[len(policies)-1])
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &PolicyConnectionResolver{connection: connection}
}

// TotalCount returns the total result count.
func (r *PolicyConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the page info.
func (r *PolicyConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the policy edges.
func (r *PolicyConnectionResolver) Edges() *[]*PolicyEdgeResolver {
	resolvers := make([]*PolicyEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &PolicyEdgeResolver{edge: edge}
	}
	return &resolvers
}

/* Mutation payload */

// PolicyMutationPayload is the response payload for a policy mutation
type PolicyMutationPayload struct {
	ClientMutationID *string
	Policy           *models.Policy
	Problems         []Problem
}

// PolicyMutationPayloadResolver resolves a PolicyMutationPayload
type PolicyMutationPayloadResolver struct {
	PolicyMutationPayload
}

// Policy field resolver
func (r *PolicyMutationPayloadResolver) Policy() *PolicyResolver {
	if r.PolicyMutationPayload.Policy == nil {
		return nil
	}
	return &PolicyResolver{policy: r.PolicyMutationPayload.Policy}
}

/* Mutation inputs */

// PolicyScopeRuleInput is the GraphQL input for a single scope rule.
type PolicyScopeRuleInput struct {
	Type    string
	Action  string
	Pattern string
}

// toModelScopeRule converts a GraphQL PolicyScopeRuleInput to a models.ScopeRule. Every field is a
// value the rule stores verbatim, so an unusable combination — an unknown type, or an empty pattern —
// is left for the policy service's scope validation to report.
func (s *PolicyScopeRuleInput) toModelScopeRule() *models.ScopeRule {
	return &models.ScopeRule{
		Type:    models.ScopeRuleType(fromGraphqlEnum(s.Type)),
		Action:  models.ScopeRuleAction(fromGraphqlEnum(s.Action)),
		Pattern: s.Pattern,
	}
}

// toModelScopeRules converts a slice of PolicyScopeRuleInput pointers to model scope rules.
func toModelScopeRules(inputs *[]PolicyScopeRuleInput) []*models.ScopeRule {
	if inputs == nil {
		return nil
	}
	rules := make([]*models.ScopeRule, len(*inputs))
	for i, inp := range *inputs {
		rules[i] = inp.toModelScopeRule()
	}
	return rules
}

// OPAPolicyDataInput is the GraphQL input for OPA-specific policy configuration.
// PackageVersionConstraint and PackageDigest stay pointers because omitting them is meaningful: no
// version constraint tracks the latest version, and no digest leaves the policy content unpinned.
type OPAPolicyDataInput struct {
	PackageSource                  string
	PackageVersionConstraint       *string
	PackageDigest                  *string
	Stage                          string
	EnforcementLevel               string
	SpeculativeRunEnforcementLevel string
}

// CreatePolicyInput contains the input for creating a group-owned policy.
type CreatePolicyInput struct {
	ClientMutationID       *string
	Kind                   string
	GroupID                string
	Name                   string
	Description            *string
	OPAData                *OPAPolicyDataInput
	Scope                  *[]PolicyScopeRuleInput
	RequiredApprovals      *int32
	AllowedUsers           *[]string
	AllowedServiceAccounts *[]string
	AllowedTeams           *[]string
}

// DeletePolicyInput contains the input for removing a policy
type DeletePolicyInput struct {
	ClientMutationID *string
	ID               string
}

// UpdatePolicyInput contains the input for updating a policy's mutable fields.
// Name, GroupID, and Kind are immutable after creation.
type UpdatePolicyInput struct {
	ClientMutationID       *string
	ID                     string
	Description            *string
	OPAData                *OPAPolicyDataInput
	Scope                  *[]PolicyScopeRuleInput
	RequiredApprovals      *int32
	AllowedUsers           *[]string
	AllowedServiceAccounts *[]string
	AllowedTeams           *[]string
}

func handlePolicyMutationProblem(e error, clientMutationID *string) (*PolicyMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := PolicyMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &PolicyMutationPayloadResolver{PolicyMutationPayload: payload}, nil
}

func createPolicyMutation(ctx context.Context, input *CreatePolicyInput) (*PolicyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	groupID, err := serviceCatalog.FetchModelID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	allowedUserIDs, err := resolvePolicyPrincipalIDs(ctx, input.AllowedUsers)
	if err != nil {
		return nil, err
	}
	allowedServiceAccountIDs, err := resolvePolicyPrincipalIDs(ctx, input.AllowedServiceAccounts)
	if err != nil {
		return nil, err
	}
	allowedTeamIDs, err := resolvePolicyPrincipalIDs(ctx, input.AllowedTeams)
	if err != nil {
		return nil, err
	}

	var opaData *models.OPAPolicyData
	if input.OPAData != nil {
		opaData = &models.OPAPolicyData{
			PackageSource:                  input.OPAData.PackageSource,
			Stage:                          models.RunTaskStageName(fromGraphqlEnum(input.OPAData.Stage)),
			EnforcementLevel:               models.PolicyEnforcementLevel(fromGraphqlEnum(input.OPAData.EnforcementLevel)),
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementLevel(fromGraphqlEnum(input.OPAData.SpeculativeRunEnforcementLevel)),
			PackageVersionConstraint:       input.OPAData.PackageVersionConstraint,
			PackageDigest:                  input.OPAData.PackageDigest,
		}
	}

	svcInput := &policy.CreatePolicyInput{
		GroupID:                  groupID,
		Name:                     input.Name,
		Kind:                     models.PolicyKindOPA,
		OPAData:                  opaData,
		Scope:                    toModelScopeRules(input.Scope),
		AllowedUserIDs:           allowedUserIDs,
		AllowedServiceAccountIDs: allowedServiceAccountIDs,
		AllowedTeamIDs:           allowedTeamIDs,
	}
	if input.Description != nil {
		svcInput.Description = input.Description
	}
	if input.RequiredApprovals != nil {
		svcInput.RequiredApprovals = int(*input.RequiredApprovals)
	}

	created, err := serviceCatalog.PolicyService.CreatePolicy(ctx, svcInput)
	if err != nil {
		return nil, err
	}

	payload := PolicyMutationPayload{ClientMutationID: input.ClientMutationID, Policy: created, Problems: []Problem{}}
	return &PolicyMutationPayloadResolver{PolicyMutationPayload: payload}, nil
}

func deletePolicyMutation(ctx context.Context, input *DeletePolicyInput) (*PolicyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	policy, err := serviceCatalog.PolicyService.GetPolicyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := serviceCatalog.PolicyService.DeletePolicy(ctx, policy); err != nil {
		return nil, err
	}

	payload := PolicyMutationPayload{ClientMutationID: input.ClientMutationID, Policy: policy, Problems: []Problem{}}
	return &PolicyMutationPayloadResolver{PolicyMutationPayload: payload}, nil
}

func updatePolicyMutation(ctx context.Context, input *UpdatePolicyInput) (*PolicyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	allowedUserIDs, err := resolvePolicyPrincipalIDs(ctx, input.AllowedUsers)
	if err != nil {
		return nil, err
	}
	allowedServiceAccountIDs, err := resolvePolicyPrincipalIDs(ctx, input.AllowedServiceAccounts)
	if err != nil {
		return nil, err
	}
	allowedTeamIDs, err := resolvePolicyPrincipalIDs(ctx, input.AllowedTeams)
	if err != nil {
		return nil, err
	}

	var opaData *models.OPAPolicyData
	if input.OPAData != nil {
		// Supplying opaData replaces it wholesale, so the package source has to come with it.
		// String! guarantees the key is present, not that it holds anything.
		if input.OPAData.PackageSource == "" {
			return nil, errors.New("packageSource is required when opaData is supplied",
				errors.WithErrorCode(errors.EInvalid))
		}
		opaData = &models.OPAPolicyData{
			PackageSource:                  input.OPAData.PackageSource,
			Stage:                          models.RunTaskStageName(fromGraphqlEnum(input.OPAData.Stage)),
			EnforcementLevel:               models.PolicyEnforcementLevel(fromGraphqlEnum(input.OPAData.EnforcementLevel)),
			SpeculativeRunEnforcementLevel: models.PolicyEnforcementLevel(fromGraphqlEnum(input.OPAData.SpeculativeRunEnforcementLevel)),
			PackageVersionConstraint:       input.OPAData.PackageVersionConstraint,
			PackageDigest:                  input.OPAData.PackageDigest,
		}
	}

	svcInput := &policy.UpdatePolicyInput{
		ID:      id,
		OPAData: opaData,
	}
	if input.Description != nil {
		svcInput.Description = input.Description
	}
	// Only forward the partial-update fields the client actually supplied. Each of these GraphQL
	// fields is nullable, so a nil pointer means "leave unchanged" and a non-nil one (including an
	// empty list) replaces the stored value. Forwarding them unconditionally would let a client that
	// updates only e.g. the description silently clear the policy's scope and approval configuration.
	if input.Scope != nil {
		scope := toModelScopeRules(input.Scope)
		svcInput.Scope = &scope
	}
	if input.RequiredApprovals != nil {
		requiredApprovals := int(*input.RequiredApprovals)
		svcInput.RequiredApprovals = &requiredApprovals
	}
	if input.AllowedUsers != nil {
		svcInput.AllowedUserIDs = &allowedUserIDs
	}
	if input.AllowedServiceAccounts != nil {
		svcInput.AllowedServiceAccountIDs = &allowedServiceAccountIDs
	}
	if input.AllowedTeams != nil {
		svcInput.AllowedTeamIDs = &allowedTeamIDs
	}

	updated, err := serviceCatalog.PolicyService.UpdatePolicy(ctx, svcInput)
	if err != nil {
		return nil, err
	}

	payload := PolicyMutationPayload{ClientMutationID: input.ClientMutationID, Policy: updated, Problems: []Problem{}}
	return &PolicyMutationPayloadResolver{PolicyMutationPayload: payload}, nil
}

/* Policy loader */

const policyLoaderKey = "policy"

// RegisterPolicyLoader registers a policy loader function
func RegisterPolicyLoader(collection *loader.Collection) {
	collection.Register(policyLoaderKey, policyBatchFunc)
}

func loadPolicy(ctx context.Context, id string) (*models.Policy, error) {
	ldr, err := loader.Extract(ctx, policyLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	policy, ok := data.(models.Policy)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &policy, nil
}

func policyBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	policies, err := getServiceCatalog(ctx).PolicyService.GetPoliciesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range policies {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
