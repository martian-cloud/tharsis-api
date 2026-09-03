package resolver

import (
	"context"
	"encoding/hex"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Run Query Resolvers */

// RunConnectionQueryArgs are used to query a run connection
type RunConnectionQueryArgs struct {
	ConnectionQueryArgs
	WorkspacePath       *string // DEPRECATED: use WorkspaceID with a TRN instead
	WorkspaceID         *string
	WorkspaceAssessment *bool
	IncludeNestedRuns   *bool
}

// RunQueryArgs are used to query a single run
// DEPRECATED: use node query instead
type RunQueryArgs struct {
	ID string
}

// RunNodeQueryArgs are used to query a single node of a run by its run-relative path
type RunNodeQueryArgs struct {
	RunID    string
	NodePath string
}

// RunEdgeResolver resolves run edges
type RunEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *RunEdgeResolver) Cursor() (string, error) {
	run, ok := r.edge.Node.(*models.Run)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(run)
	return *cursor, err
}

// Node returns a run node
func (r *RunEdgeResolver) Node() (*RunResolver, error) {
	run, ok := r.edge.Node.(*models.Run)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &RunResolver{run: run}, nil
}

// RunConnectionResolver resolves a run connection
type RunConnectionResolver struct {
	connection Connection
}

// NewRunConnectionResolver creates a new RunConnectionResolver
func NewRunConnectionResolver(ctx context.Context, input *run.GetRunsInput) (*RunConnectionResolver, error) {
	runService := getServiceCatalog(ctx).RunService

	result, err := runService.GetRuns(ctx, input)
	if err != nil {
		return nil, err
	}

	return newRunConnectionResolverFromResult(result)
}

// newRunConnectionResolverFromResult builds a RunConnectionResolver from an already-fetched runs
// result, so callers backed by different service queries can share the edge/pageInfo assembly.
func newRunConnectionResolverFromResult(result *db.RunsResult) (*RunConnectionResolver, error) {
	runs := result.Runs

	// Create edges
	edges := make([]Edge, len(runs))
	for i, run := range runs {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: run}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(runs) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(runs[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(runs[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &RunConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *RunConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the connection page information
func (r *RunConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *RunConnectionResolver) Edges() *[]*RunEdgeResolver {
	resolvers := make([]*RunEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &RunEdgeResolver{edge: edge}
	}
	return &resolvers
}

// RunVariableSensitiveValueResolver resolves a sensitive variable value
type RunVariableSensitiveValueResolver struct {
	VersionID string
	Value     string
}

// RunResolver resolves a run resource
type RunResolver struct {
	run *models.Run
}

// ID resolver
func (r *RunResolver) ID() graphql.ID {
	return graphql.ID(r.run.GetGlobalID())
}

// Status resolver
func (r *RunResolver) Status() string {
	return string(r.run.Status)
}

// IsDestroy resolver
func (r *RunResolver) IsDestroy() bool {
	return r.run.IsDestroy
}

// TargetAddresses resolver
func (r *RunResolver) TargetAddresses() []string {
	return r.run.TargetAddresses
}

// Refresh resolver
func (r *RunResolver) Refresh() bool {
	return r.run.Refresh
}

// RefreshOnly resolver
func (r *RunResolver) RefreshOnly() bool {
	return r.run.RefreshOnly
}

// Speculative resolver
func (r *RunResolver) Speculative() bool {
	return r.run.Speculative()
}

// AutoApply resolver
func (r *RunResolver) AutoApply() bool {
	return r.run.AutoApply
}

// Assessment resolver
func (r *RunResolver) Assessment() bool {
	return r.run.IsAssessmentRun
}

// Workspace resolver
func (r *RunResolver) Workspace(ctx context.Context) (*WorkspaceResolver, error) {
	workspace, err := loadWorkspace(ctx, r.run.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return &WorkspaceResolver{workspace: workspace}, nil
}

// CreatedBy resolver
func (r *RunResolver) CreatedBy() string {
	return r.run.CreatedBy
}

// Metadata resolver
func (r *RunResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.run.Metadata}
}

// ConfigurationVersion resolver
func (r *RunResolver) ConfigurationVersion(ctx context.Context) (*ConfigurationVersionResolver, error) {
	if r.run.ConfigurationVersionID == nil {
		return nil, nil
	}

	cv, err := loadConfigurationVersion(ctx, *r.run.ConfigurationVersionID)
	if err != nil {
		return nil, err
	}

	return &ConfigurationVersionResolver{configurationVersion: cv}, nil
}

// Apply resolver
func (r *RunResolver) Apply() (*ApplyResolver, error) {
	applyNode := r.run.Apply
	if applyNode == nil {
		return nil, nil
	}

	return &ApplyResolver{run: r.run}, nil
}

// Plan resolver
func (r *RunResolver) Plan() (*PlanResolver, error) {
	// A run always has a plan node, so this never returns nil.
	return &PlanResolver{run: r.run}, nil
}

// TaskStages resolver returns the run's task stage nodes in canonical stage order (pre_plan,
// post_plan, pre_apply, post_apply; empty when the run has no policies). Each stage owns its policy
// checks.
func (r *RunResolver) TaskStages() []*RunTaskStageResolver {
	stages := r.run.TaskStages
	resolvers := make([]*RunTaskStageResolver, len(stages))
	for i, stage := range stages {
		resolvers[i] = &RunTaskStageResolver{run: r.run, stage: stage}
	}
	return resolvers
}

// RunTaskStageResolver resolves a run's task stage node.
type RunTaskStageResolver struct {
	run   *models.Run
	stage *models.RunTaskStage
}

// ID resolver returns the task stage node ID as a GID.
func (r *RunTaskStageResolver) ID() string {
	return string(r.stage.GetGlobalID())
}

// StageName resolver returns the run stage this task stage evaluates (e.g. PRE_PLAN, POST_PLAN,
// PRE_APPLY, POST_APPLY).
func (r *RunTaskStageResolver) StageName() string {
	return toGraphqlEnum(string(r.stage.StageName))
}

// Status resolver returns the aggregate stage status.
func (r *RunTaskStageResolver) Status() string {
	return toGraphqlEnum(string(r.stage.Status))
}

// PolicyChecks resolver returns the policy checks owned by this stage.
func (r *RunTaskStageResolver) PolicyChecks() []*PolicyCheckResolver {
	resolvers := make([]*PolicyCheckResolver, len(r.stage.PolicyChecks))
	for i, check := range r.stage.PolicyChecks {
		resolvers[i] = &PolicyCheckResolver{run: r.run, check: check}
	}
	return resolvers
}

// PolicyCheckResolver resolves a run's policy check node.
type PolicyCheckResolver struct {
	run   *models.Run
	check *models.PolicyCheck
}

// ID resolver returns the policy check ID
func (r *PolicyCheckResolver) ID() string {
	return string(r.check.GetGlobalID())
}

// CheckType resolver returns the policy engine (e.g. opa).
func (r *PolicyCheckResolver) CheckType() string {
	return toGraphqlEnum(string(r.check.CheckType))
}

// StageName resolver returns the run stage the check evaluates at (e.g. PRE_PLAN, POST_PLAN).
func (r *PolicyCheckResolver) StageName() string {
	return toGraphqlEnum(string(r.check.StageName))
}

// NodePath resolver returns the check's node path within the run (e.g. "post_plan.opa"), which is
// what RetryRunNode takes to retry it. StageName and CheckType are upper-cased for the API, so a
// client cannot reconstruct this path from them.
func (r *PolicyCheckResolver) NodePath() string {
	return r.check.GetPath()
}

// Status resolver.
func (r *PolicyCheckResolver) Status() string {
	return toGraphqlEnum(string(r.check.Status))
}

// RunGate resolver returns the approval gate governing this check, or nil when it has none. A check
// only gets a gate when a soft-mandatory policy failed and declared approvers; an advisory or
// hard-mandatory failure, or a soft failure with no approvers, leaves the check overridable by
// permission alone.
func (r *PolicyCheckResolver) RunGate(ctx context.Context) (*RunGateResolver, error) {
	gate, err := loadRunGateForPolicyCheck(ctx, r.check.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &RunGateResolver{runGate: gate}, nil
}

// CurrentJob returns the current job for the policy check, or nil when no job exists.
func (r *PolicyCheckResolver) CurrentJob(ctx context.Context) (*JobResolver, error) {
	if r.check.LatestJobID == nil {
		return nil, nil
	}

	job, err := loadJob(ctx, *r.check.LatestJobID)
	if err != nil {
		return nil, err
	}

	return &JobResolver{job: job}, nil
}

// Jobs returns the connection of jobs associated with the policy check (its OPA evaluation jobs,
// including retries).
func (r *PolicyCheckResolver) Jobs(ctx context.Context, args *ConnectionQueryArgs) (*JobConnectionResolver, error) {
	return jobConnectionForPolicyCheck(ctx, r.run.WorkspaceID, r.check.ID, args)
}

// Policies resolver returns the per-policy-set snapshots the check evaluates, each with its result
// (empty until evaluated) and the origins that contributed the package — all taken from the
// check's own snapshot.
func (r *PolicyCheckResolver) Policies() []*PolicyCheckPolicyResolver {
	resolvers := make([]*PolicyCheckPolicyResolver, len(r.check.Policies))
	for i := range r.check.Policies {
		resolvers[i] = &PolicyCheckPolicyResolver{checkID: r.check.ID, policy: r.check.Policies[i]}
	}

	return resolvers
}

// MessagesSummary resolver returns the check's preview of the messages its policies reported. It is
// served from the check's own row, so a caller rendering many checks can show what failed without a
// read per policy from object storage.
func (r *PolicyCheckResolver) MessagesSummary() *PolicyCheckMessagesSummaryResolver {
	return &PolicyCheckMessagesSummaryResolver{summary: r.check.MessagesSummary}
}

// PolicyCheckMessagesSummaryResolver resolves a policy check's message summary. It tolerates a nil
// summary — a check that has not reported yet — so the field can stay non-null.
type PolicyCheckMessagesSummaryResolver struct {
	summary *models.PolicyCheckMessagesSummary
}

// Messages resolver returns the previewed messages, empty when the check has reported none.
func (r *PolicyCheckMessagesSummaryResolver) Messages() []string {
	if r.summary == nil {
		return []string{}
	}
	return r.summary.Messages
}

// Truncated resolver reports whether messages leaves some of what was reported out.
func (r *PolicyCheckMessagesSummaryResolver) Truncated() bool {
	return r.summary != nil && r.summary.Truncated
}

// PolicyCheckPolicyResolver resolves a single package the check evaluates, with its result. checkID
// is the policy check node this snapshot belongs to, which the messages resolver needs to address it.
type PolicyCheckPolicyResolver struct {
	checkID string
	policy  *models.PolicyCheckPolicy
}

// ID resolver returns the policy's ID within the run's policy check. It is the join key a run gate
// approval rule references by name.
func (r *PolicyCheckPolicyResolver) ID() string {
	return r.policy.ID
}

// Name resolver returns the policy's name as it was when the run was created. Empty on checks
// created before the name was snapshotted.
func (r *PolicyCheckPolicyResolver) Name() string {
	return r.policy.Name
}

// Description resolver returns the policy's description as it was when the run was created. Empty
// on checks created before the description was snapshotted, and for policies without one.
func (r *PolicyCheckPolicyResolver) Description() string {
	return r.policy.Description
}

// OPAData resolver returns the OPA-specific snapshot, null on a check of another type.
func (r *PolicyCheckPolicyResolver) OPAData() *PolicyCheckOPADataResolver {
	if r.policy.OPAData == nil {
		return nil
	}
	return &PolicyCheckOPADataResolver{data: r.policy.OPAData}
}

// ModuleAttestationData resolver returns the module-attestation snapshot, null on a check of another
// type.
func (r *PolicyCheckPolicyResolver) ModuleAttestationData() *PolicyCheckModuleAttestationDataResolver {
	if r.policy.ModuleAttestationData == nil {
		return nil
	}
	return &PolicyCheckModuleAttestationDataResolver{data: r.policy.ModuleAttestationData}
}

// PolicyCheckOPADataResolver resolves the OPA-specific snapshot of a policy a check evaluates.
type PolicyCheckOPADataResolver struct {
	data *models.PolicyCheckOPAData
}

// PackageSource resolver.
func (r *PolicyCheckOPADataResolver) PackageSource() string {
	return r.data.PackageSource
}

// PackageVersionConstraint resolver.
func (r *PolicyCheckOPADataResolver) PackageVersionConstraint() string {
	return r.data.PackageVersionConstraint
}

// PackageDigest resolver.
func (r *PolicyCheckOPADataResolver) PackageDigest() *string {
	return r.data.PackageDigest
}

// PolicyCheckModuleAttestationDataResolver resolves the module-attestation snapshot of a policy a
// check evaluates.
type PolicyCheckModuleAttestationDataResolver struct {
	data *models.PolicyCheckModuleAttestationData
}

// PublicKey resolver.
func (r *PolicyCheckModuleAttestationDataResolver) PublicKey() string {
	return r.data.PublicKey
}

// PredicateType resolver.
func (r *PolicyCheckModuleAttestationDataResolver) PredicateType() *string {
	return r.data.PredicateType
}

// VerifyStateLineage resolver.
func (r *PolicyCheckModuleAttestationDataResolver) VerifyStateLineage() bool {
	return r.data.VerifyStateLineage
}

// EnforcementLevel resolver returns the effective (strictest across sources) level.
func (r *PolicyCheckPolicyResolver) EnforcementLevel() string {
	return toGraphqlEnum(string(r.policy.EnforcementLevel))
}

// Status resolver returns the pass/fail result, empty until the check has been evaluated.
func (r *PolicyCheckPolicyResolver) Status() string {
	return toGraphqlEnum(string(r.policy.Status))
}

// Messages resolver returns every violation the policy reported, one entry each. The list is held in
// object storage, so this reads it per policy — the check's messagesSummary is the cheap alternative
// for anything rendering more than one check.
func (r *PolicyCheckPolicyResolver) Messages(ctx context.Context) ([]string, error) {
	return getServiceCatalog(ctx).RunService.GetPolicyCheckPolicyMessages(ctx, r.checkID, r.policy.ID)
}

// Provenance resolver returns the owner whose policy contributed this policy to the run.
func (r *PolicyCheckPolicyResolver) Provenance() *PolicyCheckPolicyProvenanceResolver {
	return &PolicyCheckPolicyProvenanceResolver{provenance: r.policy.Provenance}
}

// Policy resolver returns the policy resource this entry was snapshotted from. The snapshot's ID is
// the policy's own ID (see createPolicyChecks), so this is a plain by-ID load.
//
// It resolves to null rather than erroring when the policy is gone or unreadable: a run's snapshot
// outlives the policy it came from, and the caller may not hold ViewPolicy on the owning group. The
// snapshot fields on this type render either way. Note the returned policy reflects its current
// state, not its state when the run was created.
func (r *PolicyCheckPolicyResolver) Policy(ctx context.Context) (*PolicyResolver, error) {
	policy, err := loadPolicy(ctx, r.policy.ID)
	if err != nil {
		if code := errors.ErrorCode(err); code == errors.ENotFound || code == errors.EForbidden {
			return nil, nil
		}
		return nil, err
	}

	return &PolicyResolver{policy: policy}, nil
}

// PolicyCheckPolicyProvenanceResolver resolves the provenance of a policy check's policy.
type PolicyCheckPolicyProvenanceResolver struct {
	provenance models.PolicyCheckPolicyProvenance
}

// GroupID resolver returns the ID of the group that owns this policy attachment.
func (r *PolicyCheckPolicyProvenanceResolver) GroupID() string {
	return r.provenance.GroupID
}

// PolicyTRN resolver returns the stable TRN of the policy attachment itself.
func (r *PolicyCheckPolicyProvenanceResolver) PolicyTRN() string {
	return r.provenance.PolicyTRN
}

// Variables resolver
func (r *RunResolver) Variables(ctx context.Context) ([]*RunVariableResolver, error) {
	resolvers := []*RunVariableResolver{}

	variables, err := getServiceCatalog(ctx).RunService.GetRunVariables(ctx, r.run.Metadata.ID, false)
	if err != nil {
		return nil, err
	}

	for _, v := range variables {
		varCopy := v
		resolvers = append(resolvers, &RunVariableResolver{variable: &varCopy})
	}

	return resolvers, nil
}

// SensitiveVariableValues resolver
func (r *RunResolver) SensitiveVariableValues(ctx context.Context) ([]*RunVariableSensitiveValueResolver, error) {
	resolvers := []*RunVariableSensitiveValueResolver{}

	// Get run variables including sensitive values
	variables, err := getServiceCatalog(ctx).RunService.GetRunVariables(ctx, r.run.Metadata.ID, true)
	if err != nil {
		return nil, err
	}

	// Append sensitive variable values to resolvers
	for _, v := range variables {
		if v.Sensitive {
			// Verify that value and version id are not nil
			if v.Value == nil || v.VersionID == nil {
				return nil, errors.New("value and version id should be defined for sensitive variable version because includeSensitiveValues is true")
			}
			resolvers = append(resolvers, &RunVariableSensitiveValueResolver{
				VersionID: gid.ToGlobalID(types.VariableVersionModelType, *v.VersionID),
				Value:     *v.Value,
			})
		}
	}

	return resolvers, nil
}

// ModuleSource resolver
func (r *RunResolver) ModuleSource() *string {
	return r.run.ModuleSource
}

// ModuleVersion resolver
func (r *RunResolver) ModuleVersion() *string {
	return r.run.ModuleVersion
}

// ModuleDigest resolver
func (r *RunResolver) ModuleDigest() *string {
	if r.run.ModuleDigest == nil {
		return nil
	}
	return ptr.String(hex.EncodeToString(r.run.ModuleDigest))
}

// ForceCanceledBy resolver
func (r *RunResolver) ForceCanceledBy() *string {
	return r.run.ForceCanceledBy
}

// ForceCanceled resolver
func (r *RunResolver) ForceCanceled() bool {
	return r.run.ForceCanceled
}

// HasAdvisoryFailures resolver
func (r *RunResolver) HasAdvisoryFailures() bool {
	return r.run.HasAdvisoryFailures
}

// ForceCancelAvailableAt resolver
func (r *RunResolver) ForceCancelAvailableAt() *graphql.Time {
	if r.run.ForceCancelAvailableAt == nil {
		return nil
	}
	return &graphql.Time{Time: *r.run.ForceCancelAvailableAt}
}

// Comment resolver
func (r *RunResolver) Comment() string {
	return r.run.Comment
}

// TerraformVersion resolver
func (r *RunResolver) TerraformVersion() string {
	return r.run.TerraformVersion
}

// StateVersion resolver
func (r *RunResolver) StateVersion(ctx context.Context) (*StateVersionResolver, error) {
	sv, err := loadRunStateVersion(ctx, r.run.Metadata.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}

		return nil, err
	}

	return &StateVersionResolver{stateVersion: sv}, nil
}

// RunVariableResolver resolves a variable resource
type RunVariableResolver struct {
	variable *runvariables.Variable
}

// Category resolver
func (r *RunVariableResolver) Category() string {
	return string(r.variable.Category)
}

// NamespacePath resolver
func (r *RunVariableResolver) NamespacePath() *string {
	return r.variable.NamespacePath
}

// Key resolver
func (r *RunVariableResolver) Key() string {
	return r.variable.Key
}

// Value resolver
func (r *RunVariableResolver) Value() *string {
	return r.variable.Value
}

// Sensitive resolver
func (r *RunVariableResolver) Sensitive() bool {
	return r.variable.Sensitive
}

// VersionID resolver
func (r *RunVariableResolver) VersionID() *string {
	if r.variable.VersionID == nil {
		return nil
	}
	versionID := gid.ToGlobalID(types.VariableVersionModelType, *r.variable.VersionID)
	return &versionID
}

// IncludedInTFConfig resolver
func (r *RunVariableResolver) IncludedInTFConfig() *bool {
	return r.variable.IncludedInTFConfig
}

// DEPRECATED: use node query instead
func runQuery(ctx context.Context, args *RunQueryArgs) (*RunResolver, error) {
	model, err := getServiceCatalog(ctx).FetchModel(ctx, args.ID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	run, ok := model.(*models.Run)
	if !ok {
		return nil, fmt.Errorf("expected run model type, got %T", model)
	}

	return &RunResolver{run: run}, nil
}

// runNodeQuery resolves a single node of a run by its run-relative path. The run's nodes are loaded
// with the run itself, so fetching the run — which is what gates this on the workspace's view
// permission — is the only lookup this does.
func runNodeQuery(ctx context.Context, args *RunNodeQueryArgs) (*RunNodeResolver, error) {
	model, err := getServiceCatalog(ctx).FetchModel(ctx, args.RunID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	run, ok := model.(*models.Run)
	if !ok {
		return nil, fmt.Errorf("expected run model type, got %T", model)
	}

	node := run.NodeByPath(args.NodePath)
	if node == nil {
		return nil, nil
	}

	return &RunNodeResolver{run: run, node: node}, nil
}

// RunNodeResolver resolves the RunNode union: one of a run's typed nodes. Like NodeResolver, the
// concrete type is selected by whichever ToX method matches the node the path resolved to; the
// others report false so the client's inline fragments for them are skipped.
type RunNodeResolver struct {
	run  *models.Run
	node models.RunNode
}

// ToPlan resolver
func (r *RunNodeResolver) ToPlan() (*PlanResolver, bool) {
	if _, ok := r.node.(*models.Plan); ok {
		return &PlanResolver{run: r.run}, true
	}
	return nil, false
}

// ToApply resolver
func (r *RunNodeResolver) ToApply() (*ApplyResolver, bool) {
	if _, ok := r.node.(*models.Apply); ok {
		return &ApplyResolver{run: r.run}, true
	}
	return nil, false
}

// ToRunTaskStage resolver
func (r *RunNodeResolver) ToRunTaskStage() (*RunTaskStageResolver, bool) {
	if stage, ok := r.node.(*models.RunTaskStage); ok {
		return &RunTaskStageResolver{run: r.run, stage: stage}, true
	}
	return nil, false
}

// ToPolicyCheck resolver
func (r *RunNodeResolver) ToPolicyCheck() (*PolicyCheckResolver, bool) {
	if check, ok := r.node.(*models.PolicyCheck); ok {
		return &PolicyCheckResolver{run: r.run, check: check}, true
	}
	return nil, false
}

func runsQuery(ctx context.Context, args *RunConnectionQueryArgs) (*RunConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := run.GetRunsInput{
		PaginationOptions:   &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		WorkspaceAssessment: args.WorkspaceAssessment,
	}

	if args.WorkspaceID != nil && args.WorkspacePath != nil {
		return nil, errors.New("either workspaceId or workspacePath must be set, not both", errors.WithErrorCode(errors.EInvalid))
	}

	var workspaceFilterValueToResolver *string
	if args.WorkspacePath != nil {
		workspaceFilterValueToResolver = ptr.String(trn.TypeWorkspace.Build(*args.WorkspacePath))
	} else if args.WorkspaceID != nil {
		workspaceFilterValueToResolver = args.WorkspaceID
	}

	if workspaceFilterValueToResolver != nil {
		serviceCatalog := getServiceCatalog(ctx)

		workspaceID, err := serviceCatalog.FetchModelID(ctx, *workspaceFilterValueToResolver)
		if err != nil {
			return nil, err
		}

		workspace, err := serviceCatalog.WorkspaceService.GetWorkspaceByID(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		input.Workspace = workspace
	}

	if args.IncludeNestedRuns != nil {
		input.IncludeNestedRuns = args.IncludeNestedRuns
	}

	if args.Sort != nil {
		sort := db.RunSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewRunConnectionResolver(ctx, &input)
}

/* Run Mutations */

// RunMutationPayload is the response payload for a run mutation
type RunMutationPayload struct {
	ClientMutationID *string
	Run              *models.Run
	Problems         []Problem
}

// RunMutationPayloadResolver resolves a RunMutationPayload
type RunMutationPayloadResolver struct {
	RunMutationPayload
}

// Run field resolver
func (r *RunMutationPayloadResolver) Run() *RunResolver {
	if r.RunMutationPayload.Run == nil {
		return nil
	}
	return &RunResolver{run: r.RunMutationPayload.Run}
}

// CreateRunInput is the input for creating a run
type CreateRunInput struct {
	ClientMutationID       *string
	ConfigurationVersionID *string
	IsDestroy              *bool
	ModuleSource           *string
	ModuleVersion          *string
	Comment                *string
	Variables              *[]struct {
		Key      string
		Value    string
		Category string
		// DEPRECATED: HCL is DEPRECATED, to be removed in a future release.
		Hcl *bool
	}
	TerraformVersion         *string
	TargetAddresses          *[]string
	Refresh                  *bool
	RefreshOnly              *bool
	Speculative              *bool
	AutoApply                *bool
	IncludeModulePrereleases *bool
	WorkspaceID              *string
	WorkspacePath            *string // DEPRECATED: use workspaceID instead with a TRN
}

// ApplyRunInput is the input for applying a run
type ApplyRunInput struct {
	ClientMutationID *string
	Comment          *string
	RunID            string
}

// SetRunAutoApplyInput is the input for changing a run's auto-apply setting
type SetRunAutoApplyInput struct {
	ClientMutationID *string
	RunID            string
	AutoApply        bool
}

// CancelRunInput is the input for cancelling a run
type CancelRunInput struct {
	ClientMutationID *string
	Comment          *string
	Force            *bool
	RunID            string
}

// RetryRunNodeInput is the input for retrying a run's node, identified by RunID and NodePath ("plan",
// "apply", or a policy check path such as "post_plan.opa"). A plan or apply must be failed or canceled;
// a policy check may also be soft-failed.
type RetryRunNodeInput struct {
	ClientMutationID *string
	RunID            string
	NodePath         string
}

// DiscardRunInput is the input for discarding a planned run.
type DiscardRunInput struct {
	ClientMutationID *string
	RunID            string
}

// UndiscardRunInput is the input for undiscarding a discarded run.
type UndiscardRunInput struct {
	ClientMutationID *string
	RunID            string
}

// SetVariablesIncludedInTFConfigInput is the input for setting variables
// that are included in the Terraform config.
type SetVariablesIncludedInTFConfigInput struct {
	ClientMutationID *string
	RunID            string
	VariableKeys     []string
}

func handleRunMutationProblem(e error, clientMutationID *string) (*RunMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := RunMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func createRunMutation(ctx context.Context, input *CreateRunInput) (*RunMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	serviceCatalog := getServiceCatalog(ctx)

	var cvID *string
	if input.ConfigurationVersionID != nil {
		id, rErr := serviceCatalog.FetchModelID(ctx, *input.ConfigurationVersionID)
		if rErr != nil {
			return nil, rErr
		}

		cvID = &id
	}

	var terraformVersion string
	if input.TerraformVersion != nil {
		terraformVersion = *input.TerraformVersion
	}

	runOptions := &run.CreateRunInput{
		WorkspaceID:            workspaceID,
		ConfigurationVersionID: cvID,
		ModuleSource:           input.ModuleSource,
		ModuleVersion:          input.ModuleVersion,
		Comment:                input.Comment,
		TerraformVersion:       terraformVersion,
		Speculative:            input.Speculative,
		AutoApply:              ptr.ToBool(input.AutoApply),
	}

	if input.Variables != nil {
		variables := []runvariables.Variable{}

		for _, v := range *input.Variables {
			vCopy := v
			variables = append(variables, runvariables.Variable{
				Key:      v.Key,
				Value:    &vCopy.Value,
				Category: models.VariableCategory(v.Category),
			})
		}

		runOptions.Variables = variables
	}

	if input.IsDestroy != nil {
		runOptions.IsDestroy = *input.IsDestroy
	}

	if input.TargetAddresses != nil {
		runOptions.TargetAddresses = *input.TargetAddresses
	}

	// Pass the optional pointer through; run creation resolves nil to true (Terraform's default).
	runOptions.Refresh = input.Refresh

	runOptions.RefreshOnly = false // default to false unless the option was set
	if input.RefreshOnly != nil {
		runOptions.RefreshOnly = *input.RefreshOnly
	}

	if input.IncludeModulePrereleases != nil {
		runOptions.IncludeModulePrereleases = *input.IncludeModulePrereleases
	}

	run, err := serviceCatalog.RunService.CreateRun(ctx, runOptions)
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func applyRunMutation(ctx context.Context, input *ApplyRunInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	run, err := serviceCatalog.RunService.ApplyRun(ctx, runID, input.Comment)
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func setRunAutoApplyMutation(ctx context.Context, input *SetRunAutoApplyInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	run, err := serviceCatalog.RunService.SetRunAutoApply(ctx, runID, input.AutoApply)
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func cancelRunMutation(ctx context.Context, input *CancelRunInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	force := false
	if input.Force != nil {
		force = *input.Force
	}
	run, err := serviceCatalog.RunService.CancelRun(ctx, &run.CancelRunInput{
		RunID:   runID,
		Comment: input.Comment,
		Force:   force,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func retryRunNodeMutation(ctx context.Context, input *RetryRunNodeInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	retried, err := serviceCatalog.RunService.RetryRunNode(ctx, &run.RetryRunNodeInput{
		RunID:    runID,
		NodePath: input.NodePath,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: retried, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func discardRunMutation(ctx context.Context, input *DiscardRunInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	discarded, err := serviceCatalog.RunService.DiscardRun(ctx, &run.DiscardRunInput{
		RunID: runID,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: discarded, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

func undiscardRunMutation(ctx context.Context, input *UndiscardRunInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	undiscarded, err := serviceCatalog.RunService.UndiscardRun(ctx, &run.UndiscardRunInput{
		RunID: runID,
	})
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: undiscarded, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

// Deprecated: Use the gRPC API instead.
func setVariablesIncludedInTFConfigMutation(ctx context.Context, input *SetVariablesIncludedInTFConfigInput) (*RunMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	runID, err := serviceCatalog.FetchModelID(ctx, input.RunID)
	if err != nil {
		return nil, err
	}

	// Set variables
	if err = serviceCatalog.RunService.SetVariablesIncludedInTFConfig(ctx, &run.SetVariablesIncludedInTFConfigInput{
		RunID:        runID,
		VariableKeys: input.VariableKeys,
	}); err != nil {
		return nil, err
	}

	run, err := serviceCatalog.RunService.GetRunByID(ctx, runID)
	if err != nil {
		return nil, err
	}

	payload := RunMutationPayload{ClientMutationID: input.ClientMutationID, Run: run, Problems: []Problem{}}
	return &RunMutationPayloadResolver{RunMutationPayload: payload}, nil
}

/* Run Subscriptions */

// RunEventResolver resolves a run event
type RunEventResolver struct {
	event *run.Event
}

// Action resolves the event action
func (r *RunEventResolver) Action() string {
	return r.event.Action
}

// Run resolves the run
func (r *RunEventResolver) Run() *RunResolver {
	return &RunResolver{run: r.event.Run}
}

// RunSubscriptionInput is the input for subscribing to run events
type RunSubscriptionInput struct {
	RunID           *string
	WorkspaceID     *string
	WorkspacePath   *string // DEPRECATED: use workspaceID instead with a TRN
	AncestorGroupID *string
}

func (r RootResolver) workspaceRunEventsSubscription(ctx context.Context, input *RunSubscriptionInput) (<-chan *RunEventResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	subscriptionInput := &run.EventSubscriptionOptions{}

	if input.WorkspaceID != nil && input.WorkspacePath != nil {
		return nil, errors.New("workspaceId and workspacePath cannot both be set", errors.WithErrorCode(errors.EInvalid))
	}

	var workspaceValueToResolve *string
	if input.WorkspaceID != nil {
		workspaceValueToResolve = input.WorkspaceID
	} else if input.WorkspacePath != nil {
		workspaceValueToResolve = ptr.String(trn.TypeWorkspace.Build(*input.WorkspacePath))
	}

	if workspaceValueToResolve != nil {
		workspaceID, err := serviceCatalog.FetchModelID(ctx, *workspaceValueToResolve)
		if err != nil {
			return nil, err
		}

		subscriptionInput.WorkspaceID = &workspaceID
	}

	if input.RunID != nil {
		runID, err := serviceCatalog.FetchModelID(ctx, *input.RunID)
		if err != nil {
			return nil, err
		}

		subscriptionInput.RunID = &runID
	}
	if input.AncestorGroupID != nil {
		ancestorGroupID, err := serviceCatalog.FetchModelID(ctx, *input.AncestorGroupID)
		if err != nil {
			return nil, err
		}

		subscriptionInput.AncestorGroupID = &ancestorGroupID
	}
	events, err := serviceCatalog.RunService.SubscribeToRunEvents(ctx, subscriptionInput)
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *RunEventResolver)

	go func() {
		for event := range events {
			select {
			case <-ctx.Done():
			case outgoing <- &RunEventResolver{event: event}:
			}
		}

		close(outgoing)
	}()

	return outgoing, nil
}

/* Run loader */

const runLoaderKey = "run"

// RegisterRunLoader registers a run loader function
func RegisterRunLoader(collection *loader.Collection) {
	collection.Register(runLoaderKey, runBatchFunc)
}

func loadRun(ctx context.Context, id string) (*models.Run, error) {
	ldr, err := loader.Extract(ctx, runLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	run, ok := data.(*models.Run)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return run, nil
}

func runBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	runs, err := getServiceCatalog(ctx).RunService.GetRunsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range runs {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

/* Run state version loader */

const runStateVersionLoaderKey = "runStateVersion"

// RegisterRunStateVersionLoader registers a run state version loader function
func RegisterRunStateVersionLoader(collection *loader.Collection) {
	collection.Register(runStateVersionLoaderKey, runStateVersionBatchFunc)
}

func loadRunStateVersion(ctx context.Context, id string) (*models.StateVersion, error) {
	ldr, err := loader.Extract(ctx, runStateVersionLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	sv, ok := data.(models.StateVersion)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &sv, nil
}

func runStateVersionBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	stateVersions, err := getServiceCatalog(ctx).RunService.GetStateVersionsByRunIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range stateVersions {
		// Use run ID as the key since that is the ID which was
		// used to query the data
		batch[*result.RunID] = result
	}

	return batch, nil
}

/* Run gate by policy check loader */

const runGateByPolicyCheckLoaderKey = "runGateByPolicyCheck"

// RegisterRunGateByPolicyCheckLoader registers a run gate by policy check loader function
func RegisterRunGateByPolicyCheckLoader(collection *loader.Collection) {
	collection.Register(runGateByPolicyCheckLoaderKey, runGateByPolicyCheckBatchFunc)
}

func loadRunGateForPolicyCheck(ctx context.Context, policyCheckID string) (*models.RunGate, error) {
	ldr, err := loader.Extract(ctx, runGateByPolicyCheckLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(policyCheckID))()
	if err != nil {
		return nil, err
	}

	gate, ok := data.(*models.RunGate)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return gate, nil
}

func runGateByPolicyCheckBatchFunc(ctx context.Context, policyCheckIDs []string) (loader.DataBatch, error) {
	gates, err := getServiceCatalog(ctx).RunService.GetRunGatesByPolicyCheckIDs(ctx, policyCheckIDs)
	if err != nil {
		return nil, err
	}

	// Build map of results, keyed by policy check ID since that is what was queried. A check with no
	// gate is simply absent, and the resolver turns the loader's ENotFound into a null field.
	batch := loader.DataBatch{}
	for _, result := range gates {
		batch[result.PolicyCheckID] = result
	}

	return batch, nil
}

// TerraformCheckResultResolver resolves a check result
type TerraformCheckResultResolver struct {
	checkResult *corerun.CheckResult
}

// Name resolver
func (r *TerraformCheckResultResolver) Name() string {
	return r.checkResult.Name
}

// Status resolver
func (r *TerraformCheckResultResolver) Status() string {
	return toGraphqlEnum(r.checkResult.Status)
}

// Objects resolver
func (r *TerraformCheckResultResolver) Objects() []*TerraformCheckResultObjectResolver {
	resolvers := []*TerraformCheckResultObjectResolver{}
	for _, obj := range r.checkResult.Objects {
		objCopy := obj
		resolvers = append(resolvers, &TerraformCheckResultObjectResolver{
			address:         objCopy.Address,
			status:          objCopy.Status,
			failureMessages: objCopy.FailureMessages,
		})
	}
	return resolvers
}

// TerraformCheckResultObjectResolver resolves an individual checkable object within a check result
type TerraformCheckResultObjectResolver struct {
	address         string
	status          string
	failureMessages []string
}

// Address resolver
func (r *TerraformCheckResultObjectResolver) Address() string {
	return r.address
}

// Status resolver
func (r *TerraformCheckResultObjectResolver) Status() string {
	return toGraphqlEnum(r.status)
}

// FailureMessages resolver
func (r *TerraformCheckResultObjectResolver) FailureMessages() []string {
	if r.failureMessages == nil {
		return []string{}
	}
	return r.failureMessages
}
