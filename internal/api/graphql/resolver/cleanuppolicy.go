package resolver

import (
	"context"
	"strconv"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cleanuppolicy"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// CleanupPolicyResolver resolves a cleanup policy resource
type CleanupPolicyResolver struct {
	policy *models.CleanupPolicy
}

// ID resolver
func (r *CleanupPolicyResolver) ID() graphql.ID {
	return graphql.ID(r.policy.GetGlobalID())
}

// Metadata resolver
func (r *CleanupPolicyResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.policy.Metadata}
}

// NamespacePath resolver
func (r *CleanupPolicyResolver) NamespacePath() string {
	return r.policy.NamespacePath()
}

// Kind resolver
func (r *CleanupPolicyResolver) Kind() string {
	return toGraphqlEnum(string(r.policy.Kind))
}

// Disabled resolver
func (r *CleanupPolicyResolver) Disabled() bool {
	return r.policy.Disabled
}

// LastSweepCompletedAt resolver
func (r *CleanupPolicyResolver) LastSweepCompletedAt() *graphql.Time {
	if r.policy.LastSweepCompletedAt == nil {
		return nil
	}

	return &graphql.Time{Time: *r.policy.LastSweepCompletedAt}
}

// TerraformModulePolicyData resolver
func (r *CleanupPolicyResolver) TerraformModulePolicyData() *TerraformModuleCleanupPolicyDataResolver {
	if r.policy.TerraformModulePolicyData == nil {
		return nil
	}

	return &TerraformModuleCleanupPolicyDataResolver{data: r.policy.TerraformModulePolicyData}
}

// TerraformProviderPolicyData resolver
func (r *CleanupPolicyResolver) TerraformProviderPolicyData() *TerraformProviderCleanupPolicyDataResolver {
	if r.policy.TerraformProviderPolicyData == nil {
		return nil
	}

	return &TerraformProviderCleanupPolicyDataResolver{data: r.policy.TerraformProviderPolicyData}
}

// RunPolicyData resolver
func (r *CleanupPolicyResolver) RunPolicyData() *RunCleanupPolicyDataResolver {
	if r.policy.RunPolicyData == nil {
		return nil
	}

	return &RunCleanupPolicyDataResolver{data: r.policy.RunPolicyData}
}

// TerraformModuleCleanupPolicyDataResolver resolves terraform module policy data.
type TerraformModuleCleanupPolicyDataResolver struct {
	data *models.TerraformModuleCleanupPolicyData
}

// Rules resolver
func (r *TerraformModuleCleanupPolicyDataResolver) Rules() []*TerraformModuleCleanupRuleResolver {
	resolvers := make([]*TerraformModuleCleanupRuleResolver, len(r.data.Rules))
	for i, rule := range r.data.Rules {
		resolvers[i] = &TerraformModuleCleanupRuleResolver{rule: rule}
	}

	return resolvers
}

// TerraformModuleCleanupRuleResolver resolves a Terraform module cleanup rule
type TerraformModuleCleanupRuleResolver struct {
	rule *models.TerraformModuleCleanupRule
}

// Strategy resolver
func (r *TerraformModuleCleanupRuleResolver) Strategy() string {
	return toGraphqlEnum(string(r.rule.Strategy))
}

// Description resolver
func (r *TerraformModuleCleanupRuleResolver) Description() string {
	return r.rule.Description
}

// NameGlob resolver
func (r *TerraformModuleCleanupRuleResolver) NameGlob() string {
	return string(r.rule.NameGlob)
}

// SystemGlob resolver
func (r *TerraformModuleCleanupRuleResolver) SystemGlob() string {
	return string(r.rule.SystemGlob)
}

// VersionGlob resolver
func (r *TerraformModuleCleanupRuleResolver) VersionGlob() string {
	return string(r.rule.VersionGlob)
}

// DeleteAfterDays resolver
func (r *TerraformModuleCleanupRuleResolver) DeleteAfterDays() int32 {
	return r.rule.DeleteAfterDays
}

// TerraformProviderCleanupPolicyDataResolver resolves terraform provider policy data.
type TerraformProviderCleanupPolicyDataResolver struct {
	data *models.TerraformProviderCleanupPolicyData
}

// Rules resolver
func (r *TerraformProviderCleanupPolicyDataResolver) Rules() []*TerraformProviderCleanupRuleResolver {
	resolvers := make([]*TerraformProviderCleanupRuleResolver, len(r.data.Rules))
	for i, rule := range r.data.Rules {
		resolvers[i] = &TerraformProviderCleanupRuleResolver{rule: rule}
	}

	return resolvers
}

// TerraformProviderCleanupRuleResolver resolves a Terraform provider cleanup rule
type TerraformProviderCleanupRuleResolver struct {
	rule *models.TerraformProviderCleanupRule
}

// Strategy resolver
func (r *TerraformProviderCleanupRuleResolver) Strategy() string {
	return toGraphqlEnum(string(r.rule.Strategy))
}

// Description resolver
func (r *TerraformProviderCleanupRuleResolver) Description() string {
	return r.rule.Description
}

// NameGlob resolver
func (r *TerraformProviderCleanupRuleResolver) NameGlob() string {
	return string(r.rule.NameGlob)
}

// VersionGlob resolver
func (r *TerraformProviderCleanupRuleResolver) VersionGlob() string {
	return string(r.rule.VersionGlob)
}

// DeleteAfterDays resolver
func (r *TerraformProviderCleanupRuleResolver) DeleteAfterDays() int32 {
	return r.rule.DeleteAfterDays
}

// RunCleanupPolicyDataResolver resolves run policy data.
type RunCleanupPolicyDataResolver struct {
	data *models.RunCleanupPolicyData
}

// Rules resolver
func (r *RunCleanupPolicyDataResolver) Rules() []*RunCleanupRuleResolver {
	resolvers := make([]*RunCleanupRuleResolver, len(r.data.Rules))
	for i, rule := range r.data.Rules {
		resolvers[i] = &RunCleanupRuleResolver{rule: rule}
	}

	return resolvers
}

// RunCleanupRuleResolver resolves a run cleanup rule
type RunCleanupRuleResolver struct {
	rule *models.RunCleanupRule
}

// Strategy resolver
func (r *RunCleanupRuleResolver) Strategy() string {
	return toGraphqlEnum(string(r.rule.Strategy))
}

// Description resolver
func (r *RunCleanupRuleResolver) Description() string {
	return r.rule.Description
}

// Speculative resolver
func (r *RunCleanupRuleResolver) Speculative() *bool {
	return r.rule.Speculative
}

// Assessment resolver
func (r *RunCleanupRuleResolver) Assessment() *bool {
	return r.rule.Assessment
}

// Status resolver
func (r *RunCleanupRuleResolver) Status() []models.RunStatus {
	return r.rule.Status
}

// KeepMin resolver
func (r *RunCleanupRuleResolver) KeepMin() int32 {
	return r.rule.KeepMin
}

// DeleteAfterDays resolver
func (r *RunCleanupRuleResolver) DeleteAfterDays() int32 {
	return r.rule.DeleteAfterDays
}

/* CleanupPolicy Query Resolvers */

// getCleanupPolicies fetches policies via the service and returns resolver slice.
func getCleanupPolicies(ctx context.Context, namespacePath string) ([]*CleanupPolicyResolver, error) {
	policies, err := getServiceCatalog(ctx).CleanupPolicyService.GetEffectiveCleanupPolicies(ctx, namespacePath)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*CleanupPolicyResolver, len(policies))
	for i, p := range policies {
		resolvers[i] = &CleanupPolicyResolver{policy: p}
	}

	return resolvers, nil
}

/* CleanupPolicy Mutation Resolvers */

// CleanupPolicyMutationPayload is the response payload for a cleanup policy mutation
type CleanupPolicyMutationPayload struct {
	ClientMutationID *string
	CleanupPolicy    *models.CleanupPolicy
	Problems         []Problem
}

// CleanupPolicyMutationPayloadResolver resolves a CleanupPolicyMutationPayload
type CleanupPolicyMutationPayloadResolver struct {
	CleanupPolicyMutationPayload
}

// CleanupPolicy field resolver
func (r *CleanupPolicyMutationPayloadResolver) CleanupPolicy() *CleanupPolicyResolver {
	if r.CleanupPolicyMutationPayload.CleanupPolicy == nil {
		return nil
	}

	return &CleanupPolicyResolver{policy: r.CleanupPolicyMutationPayload.CleanupPolicy}
}

// Namespace field resolver
func (r *CleanupPolicyMutationPayloadResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	if r.CleanupPolicyMutationPayload.CleanupPolicy == nil {
		return nil, nil
	}

	namespacePath := r.CleanupPolicyMutationPayload.CleanupPolicy.NamespacePath()
	serviceCatalog := getServiceCatalog(ctx)

	group, err := serviceCatalog.GroupService.GetGroupByTRN(ctx, trn.TypeGroup.Build(namespacePath))
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}

	if group != nil {
		return &NamespaceResolver{result: &GroupResolver{group: group}}, nil
	}

	ws, err := serviceCatalog.WorkspaceService.GetWorkspaceByTRN(ctx, trn.TypeWorkspace.Build(namespacePath))
	if err != nil {
		return nil, err
	}

	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ws}}, nil
}

// CreateCleanupPolicyInput contains the input for creating a cleanup policy
type CreateCleanupPolicyInput struct {
	ClientMutationID            *string
	NamespacePath               string
	Kind                        string
	Disabled                    *bool
	TerraformModulePolicyData   *TerraformModuleCleanupPolicyDataInput
	TerraformProviderPolicyData *TerraformProviderCleanupPolicyDataInput
	RunPolicyData               *RunCleanupPolicyDataInput
}

// UpdateCleanupPolicyInput contains the input for updating a cleanup policy
type UpdateCleanupPolicyInput struct {
	ClientMutationID            *string
	Metadata                    *MetadataInput
	ID                          string
	Disabled                    *bool
	TerraformModulePolicyData   *TerraformModuleCleanupPolicyDataInput
	TerraformProviderPolicyData *TerraformProviderCleanupPolicyDataInput
	RunPolicyData               *RunCleanupPolicyDataInput
}

// DeleteCleanupPolicyInput contains the input for deleting a cleanup policy
type DeleteCleanupPolicyInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

// TerraformModuleCleanupPolicyDataInput is the GraphQL input for terraform module policy data.
type TerraformModuleCleanupPolicyDataInput struct {
	Rules []*TerraformModuleCleanupRuleInput
}

// terraformModuleCleanupPolicyData converts GraphQL input to a model data struct, or nil for nil input.
func terraformModuleCleanupPolicyData(input *TerraformModuleCleanupPolicyDataInput) *models.TerraformModuleCleanupPolicyData {
	if input == nil {
		return nil
	}

	rules := make([]*models.TerraformModuleCleanupRule, len(input.Rules))
	for i, r := range input.Rules {
		rules[i] = r.toModel()
	}

	return &models.TerraformModuleCleanupPolicyData{Rules: rules}
}

// TerraformModuleCleanupRuleInput is the GraphQL input for a Terraform module cleanup rule. Strategy
// arrives in the wire's UPPER_SNAKE_CASE and is lowercased by toModel below, since models.Strategy no
// longer implements graphql.Unmarshaler.
type TerraformModuleCleanupRuleInput struct {
	Strategy        string
	Description     string
	NameGlob        string
	SystemGlob      string
	VersionGlob     string
	DeleteAfterDays int32
}

// toModel converts the GraphQL input to its model form.
func (i *TerraformModuleCleanupRuleInput) toModel() *models.TerraformModuleCleanupRule {
	return &models.TerraformModuleCleanupRule{
		Strategy:        models.CleanupStrategy(fromGraphqlEnum(i.Strategy)),
		Description:     i.Description,
		NameGlob:        models.CleanupGlob(i.NameGlob),
		SystemGlob:      models.CleanupGlob(i.SystemGlob),
		VersionGlob:     models.CleanupGlob(i.VersionGlob),
		DeleteAfterDays: i.DeleteAfterDays,
	}
}

// TerraformProviderCleanupPolicyDataInput is the GraphQL input for terraform provider policy data.
type TerraformProviderCleanupPolicyDataInput struct {
	Rules []*TerraformProviderCleanupRuleInput
}

// terraformProviderCleanupPolicyData converts GraphQL input to a model data struct, or nil for nil input.
func terraformProviderCleanupPolicyData(input *TerraformProviderCleanupPolicyDataInput) *models.TerraformProviderCleanupPolicyData {
	if input == nil {
		return nil
	}

	rules := make([]*models.TerraformProviderCleanupRule, len(input.Rules))
	for i, r := range input.Rules {
		rules[i] = r.toModel()
	}

	return &models.TerraformProviderCleanupPolicyData{Rules: rules}
}

// TerraformProviderCleanupRuleInput is the GraphQL input for a Terraform provider cleanup rule.
type TerraformProviderCleanupRuleInput struct {
	Strategy        string
	Description     string
	NameGlob        string
	VersionGlob     string
	DeleteAfterDays int32
}

// toModel converts the GraphQL input to its model form.
func (i *TerraformProviderCleanupRuleInput) toModel() *models.TerraformProviderCleanupRule {
	return &models.TerraformProviderCleanupRule{
		Strategy:        models.CleanupStrategy(fromGraphqlEnum(i.Strategy)),
		Description:     i.Description,
		NameGlob:        models.CleanupGlob(i.NameGlob),
		VersionGlob:     models.CleanupGlob(i.VersionGlob),
		DeleteAfterDays: i.DeleteAfterDays,
	}
}

// RunCleanupPolicyDataInput is the GraphQL input for run policy data.
type RunCleanupPolicyDataInput struct {
	Rules []*RunCleanupRuleInput
}

// runCleanupPolicyData converts GraphQL input to a model data struct, or nil for nil input.
func runCleanupPolicyData(input *RunCleanupPolicyDataInput) *models.RunCleanupPolicyData {
	if input == nil {
		return nil
	}

	rules := make([]*models.RunCleanupRule, len(input.Rules))
	for i, r := range input.Rules {
		rules[i] = r.toModel()
	}

	return &models.RunCleanupPolicyData{Rules: rules}
}

// RunCleanupRuleInput is the GraphQL input for a run cleanup rule.
type RunCleanupRuleInput struct {
	Strategy        string
	Description     string
	Speculative     *bool
	Assessment      *bool
	Status          []models.RunStatus
	KeepMin         int32
	DeleteAfterDays int32
}

// toModel converts the GraphQL input to its model form.
func (i *RunCleanupRuleInput) toModel() *models.RunCleanupRule {
	return &models.RunCleanupRule{
		Strategy:        models.CleanupStrategy(fromGraphqlEnum(i.Strategy)),
		Description:     i.Description,
		Speculative:     i.Speculative,
		Assessment:      i.Assessment,
		Status:          i.Status,
		KeepMin:         i.KeepMin,
		DeleteAfterDays: i.DeleteAfterDays,
	}
}

func handleCleanupPolicyMutationProblem(e error, clientMutationID *string) (*CleanupPolicyMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := CleanupPolicyMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &CleanupPolicyMutationPayloadResolver{CleanupPolicyMutationPayload: payload}, nil
}

func createCleanupPolicyMutation(ctx context.Context, input *CreateCleanupPolicyInput) (*CleanupPolicyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	toCreate := &cleanuppolicy.CreateCleanupPolicyInput{
		NamespacePath:               input.NamespacePath,
		Kind:                        models.CleanupRuleKind(fromGraphqlEnum(input.Kind)),
		TerraformModulePolicyData:   terraformModuleCleanupPolicyData(input.TerraformModulePolicyData),
		TerraformProviderPolicyData: terraformProviderCleanupPolicyData(input.TerraformProviderPolicyData),
		RunPolicyData:               runCleanupPolicyData(input.RunPolicyData),
	}

	if input.Disabled != nil {
		toCreate.Disabled = *input.Disabled
	}

	created, err := serviceCatalog.CleanupPolicyService.CreateCleanupPolicy(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	payload := CleanupPolicyMutationPayload{ClientMutationID: input.ClientMutationID, CleanupPolicy: created, Problems: []Problem{}}
	return &CleanupPolicyMutationPayloadResolver{CleanupPolicyMutationPayload: payload}, nil
}

func updateCleanupPolicyMutation(ctx context.Context, input *UpdateCleanupPolicyInput) (*CleanupPolicyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	toUpdate := &cleanuppolicy.UpdateCleanupPolicyInput{
		ID:                          id,
		Disabled:                    input.Disabled,
		TerraformModulePolicyData:   terraformModuleCleanupPolicyData(input.TerraformModulePolicyData),
		TerraformProviderPolicyData: terraformProviderCleanupPolicyData(input.TerraformProviderPolicyData),
		RunPolicyData:               runCleanupPolicyData(input.RunPolicyData),
	}

	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}
		toUpdate.Version = &v
	}

	updated, err := serviceCatalog.CleanupPolicyService.UpdateCleanupPolicy(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	payload := CleanupPolicyMutationPayload{ClientMutationID: input.ClientMutationID, CleanupPolicy: updated, Problems: []Problem{}}
	return &CleanupPolicyMutationPayloadResolver{CleanupPolicyMutationPayload: payload}, nil
}

func deleteCleanupPolicyMutation(ctx context.Context, input *DeleteCleanupPolicyInput) (*CleanupPolicyMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	policy, err := serviceCatalog.CleanupPolicyService.GetCleanupPolicyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	toDelete := &cleanuppolicy.DeleteCleanupPolicyInput{ID: id}

	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		toDelete.Version = &v
	}

	if err := serviceCatalog.CleanupPolicyService.DeleteCleanupPolicy(ctx, toDelete); err != nil {
		return nil, err
	}

	payload := CleanupPolicyMutationPayload{ClientMutationID: input.ClientMutationID, CleanupPolicy: policy, Problems: []Problem{}}
	return &CleanupPolicyMutationPayloadResolver{CleanupPolicyMutationPayload: payload}, nil
}

/* CleanupPolicy loader */

const cleanupPolicyLoaderKey = "cleanupPolicy"

// RegisterCleanupPolicyLoader registers a cleanupPolicy loader function
func RegisterCleanupPolicyLoader(collection *loader.Collection) {
	collection.Register(cleanupPolicyLoaderKey, cleanupPolicyBatchFunc)
}

func loadCleanupPolicy(ctx context.Context, id string) (*models.CleanupPolicy, error) {
	ldr, err := loader.Extract(ctx, cleanupPolicyLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	policy, ok := data.(*models.CleanupPolicy)
	if !ok {
		return nil, errors.New("wrong type")
	}

	return policy, nil
}

func cleanupPolicyBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	policies, err := getServiceCatalog(ctx).CleanupPolicyService.GetCleanupPoliciesByIDs(ctx, ids)
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
