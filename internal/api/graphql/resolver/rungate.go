package resolver

import (
	"context"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

/* RunGate Query Resolvers */

/* RunGate Resolver */

// RunGateResolver resolves a run gate.
type RunGateResolver struct {
	runGate *models.RunGate
}

// ID resolver.
func (r *RunGateResolver) ID() graphql.ID {
	return graphql.ID(r.runGate.GetGlobalID())
}

// Metadata resolver.
func (r *RunGateResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.runGate.Metadata}
}

// Status resolver.
func (r *RunGateResolver) Status() string {
	return toGraphqlEnum(string(r.runGate.Status))
}

// Type resolver returns the kind of policy check this gate governs.
func (r *RunGateResolver) Type() string {
	return toGraphqlEnum(string(r.runGate.Type))
}

// ApprovalRules resolver returns the gate's per-policy approval rules.
func (r *RunGateResolver) ApprovalRules() []*RunGateApprovalRuleResolver {
	resolvers := make([]*RunGateApprovalRuleResolver, len(r.runGate.ApprovalRules))
	for i := range r.runGate.ApprovalRules {
		resolvers[i] = &RunGateApprovalRuleResolver{rule: r.runGate.ApprovalRules[i]}
	}
	return resolvers
}

// OverriddenBy resolver returns the subject that bypassed the gate's approval requirements. Null
// unless the gate was overridden — a gate that reached approved by collecting its approvals was not
// overridden by anyone, and its decisions are listed under Approvals instead.
func (r *RunGateResolver) OverriddenBy() *string {
	return r.runGate.OverriddenBy
}

// OverrideComment resolver returns the reason given for the override, or an empty string when none
// was supplied or the gate was not overridden.
func (r *RunGateResolver) OverrideComment() string {
	if r.runGate.OverrideComment == nil {
		return ""
	}
	return *r.runGate.OverrideComment
}

/* RunGateApprovalRule Resolver */

// RunGateApprovalRuleResolver resolves a run gate approval rule.
type RunGateApprovalRuleResolver struct {
	rule *models.RunGateApprovalRule
}

// Name resolver returns the rule name (the run's PolicyCheckPolicy ID).
func (r *RunGateApprovalRuleResolver) Name() string {
	return r.rule.Name
}

// RequiredApprovals resolver.
func (r *RunGateApprovalRuleResolver) RequiredApprovals() int32 {
	return int32(r.rule.RequiredApprovals)
}

// AllowedUsers resolver. A subject whose user has been deleted is omitted (never fails the field).
func (r *RunGateApprovalRuleResolver) AllowedUsers(ctx context.Context) ([]*UserResolver, error) {
	resolvers := []*UserResolver{}
	for _, subject := range r.rule.AllowedSubjects {
		if subject.Type != models.RunGateSubjectUser {
			continue
		}
		user, err := loadUser(ctx, subject.ID)
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

// AllowedServiceAccounts resolver. A subject whose service account has been deleted is omitted.
func (r *RunGateApprovalRuleResolver) AllowedServiceAccounts(ctx context.Context) ([]*ServiceAccountResolver, error) {
	resolvers := []*ServiceAccountResolver{}
	for _, subject := range r.rule.AllowedSubjects {
		if subject.Type != models.RunGateSubjectServiceAccount {
			continue
		}
		sa, err := loadServiceAccount(ctx, subject.ID)
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

// AllowedTeams resolver. A subject whose team has been deleted is omitted.
func (r *RunGateApprovalRuleResolver) AllowedTeams(ctx context.Context) ([]*TeamResolver, error) {
	resolvers := []*TeamResolver{}
	for _, subject := range r.rule.AllowedSubjects {
		if subject.Type != models.RunGateSubjectTeam {
			continue
		}
		team, err := loadTeam(ctx, subject.ID)
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

// Approvals resolver.
func (r *RunGateResolver) Approvals(ctx context.Context) ([]*RunGateApprovalResolver, error) {
	approvals, err := getServiceCatalog(ctx).RunService.GetRunGateApprovalsByGateID(ctx, r.runGate.Metadata.ID)
	if err != nil {
		return nil, err
	}
	resolvers := make([]*RunGateApprovalResolver, len(approvals))
	for i := range approvals {
		resolvers[i] = &RunGateApprovalResolver{approval: &approvals[i]}
	}
	return resolvers, nil
}

// Run resolver.
func (r *RunGateResolver) Run(ctx context.Context) (*RunResolver, error) {
	run, err := loadRun(ctx, r.runGate.RunID)
	if err != nil {
		return nil, err
	}
	return &RunResolver{run: run}, nil
}

// PolicyCheck resolver returns the check this gate governs. The run carries its task stages and their
// checks in memory, so this needs no query beyond the batched run load Run() already performs.
func (r *RunGateResolver) PolicyCheck(ctx context.Context) (*PolicyCheckResolver, error) {
	run, err := loadRun(ctx, r.runGate.RunID)
	if err != nil {
		return nil, err
	}

	for _, stage := range run.TaskStages {
		for _, check := range stage.PolicyChecks {
			if check.ID == r.runGate.PolicyCheckID {
				return &PolicyCheckResolver{run: run, check: check}, nil
			}
		}
	}

	return nil, errors.New("policy check %s for run gate %s not found",
		r.runGate.PolicyCheckID, r.runGate.Metadata.ID, errors.WithErrorCode(errors.ENotFound))
}

/* RunGateApproval Resolver */

// RunGateApprovalResolver resolves a run gate approval.
type RunGateApprovalResolver struct {
	approval *models.RunGateApproval
}

// ID resolver.
func (r *RunGateApprovalResolver) ID() graphql.ID {
	return graphql.ID(r.approval.GetGlobalID())
}

// Metadata resolver.
func (r *RunGateApprovalResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.approval.Metadata}
}

// Decision resolver.
func (r *RunGateApprovalResolver) Decision() string {
	return toGraphqlEnum(string(r.approval.Decision))
}

// Comment resolver.
func (r *RunGateApprovalResolver) Comment() string {
	if r.approval.Comment == nil {
		return ""
	}
	return *r.approval.Comment
}

// CreatedBy resolver.
func (r *RunGateApprovalResolver) CreatedBy() string {
	return r.approval.CreatedBy
}

// CoveredRules resolver returns the names of the gate's approval rules this decision counts toward.
func (r *RunGateApprovalResolver) CoveredRules() []string {
	if r.approval.CoveredRules == nil {
		return []string{}
	}
	return r.approval.CoveredRules
}

// User resolver. Null when the approver was a service account or the user has been deleted.
func (r *RunGateApprovalResolver) User(ctx context.Context) (*UserResolver, error) {
	if r.approval.UserID == nil {
		return nil, nil
	}
	user, err := loadUser(ctx, *r.approval.UserID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &UserResolver{user: user}, nil
}

// ServiceAccount resolver. Null when the approver was a user or the SA has been deleted.
func (r *RunGateApprovalResolver) ServiceAccount(ctx context.Context) (*ServiceAccountResolver, error) {
	if r.approval.ServiceAccountID == nil {
		return nil, nil
	}
	sa, err := loadServiceAccount(ctx, *r.approval.ServiceAccountID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}
	return &ServiceAccountResolver{serviceAccount: sa}, nil
}

/* Query */

// RunGateConnectionQueryArgs are used to query a run gate connection.
type RunGateConnectionQueryArgs struct {
	ConnectionQueryArgs
}

// RunGateEdgeResolver resolves run gate edges.
type RunGateEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor.
func (r *RunGateEdgeResolver) Cursor() (string, error) {
	gate, ok := r.edge.Node.(models.RunGate)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&gate)
	return *cursor, err
}

// Node returns a run gate node.
func (r *RunGateEdgeResolver) Node() (*RunGateResolver, error) {
	gate, ok := r.edge.Node.(models.RunGate)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}
	return &RunGateResolver{runGate: &gate}, nil
}

// RunGateConnectionResolver resolves a run gate connection.
type RunGateConnectionResolver struct {
	connection Connection
}

// NewRunGateConnectionResolver creates a new RunGateConnectionResolver from a db result.
func NewRunGateConnectionResolver(result *db.RunGatesResult) (*RunGateConnectionResolver, error) {
	gates := result.RunGates

	edges := make([]Edge, len(gates))
	for i := range gates {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: gates[i]}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(gates) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&gates[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&gates[len(gates)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &RunGateConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count.
func (r *RunGateConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the page info.
func (r *RunGateConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the run gate edges.
func (r *RunGateConnectionResolver) Edges() *[]*RunGateEdgeResolver {
	resolvers := make([]*RunGateEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &RunGateEdgeResolver{edge: edge}
	}
	return &resolvers
}

// runGatesAwaitingMyDecisionQuery resolves the caller's approvals inbox: pending run gates the
// caller is an eligible, undecided approver for.
func runGatesAwaitingMyDecisionQuery(ctx context.Context, args *RunGateConnectionQueryArgs) (*RunGateConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &run.GetRunGatesAwaitingDecisionInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.Sort != nil {
		sort := db.RunGateSortableField(*args.Sort)
		input.Sort = &sort
	}

	result, err := getServiceCatalog(ctx).RunService.GetRunGatesAwaitingDecision(ctx, input)
	if err != nil {
		return nil, err
	}

	return NewRunGateConnectionResolver(result)
}

/* Mutations */

// RunGateMutationPayload is the response payload for a run gate mutation.
type RunGateMutationPayload struct {
	ClientMutationID *string
	RunGate          *models.RunGate
	Problems         []Problem
}

// RunGateMutationPayloadResolver resolves a RunGateMutationPayload.
type RunGateMutationPayloadResolver struct {
	RunGateMutationPayload
}

// RunGate field resolver.
func (r *RunGateMutationPayloadResolver) RunGate() *RunGateResolver {
	if r.RunGateMutationPayload.RunGate == nil {
		return nil
	}
	return &RunGateResolver{runGate: r.RunGateMutationPayload.RunGate}
}

// ApproveRunGateInput contains the input for approving or rejecting a run gate.
type ApproveRunGateInput struct {
	ClientMutationID *string
	GateID           string
	Decision         string
	Comment          *string
}

// OverrideRunGateInput contains the input for an admin override of a pending run gate.
type OverrideRunGateInput struct {
	ClientMutationID *string
	GateID           string
	Comment          *string
}

func handleRunGateMutationProblem(e error, clientMutationID *string) (*RunGateMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := RunGateMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &RunGateMutationPayloadResolver{RunGateMutationPayload: payload}, nil
}

func approveRunGateMutation(ctx context.Context, input *ApproveRunGateInput) (*RunGateMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	gateID, err := serviceCatalog.FetchModelID(ctx, input.GateID)
	if err != nil {
		return nil, err
	}

	gate, err := serviceCatalog.RunService.ApproveRunGate(ctx, &run.ApproveRunGateInput{
		GateID:   gateID,
		Decision: models.RunGateDecision(fromGraphqlEnum(input.Decision)),
		Comment:  input.Comment,
	})
	if err != nil {
		return nil, err
	}

	payload := RunGateMutationPayload{ClientMutationID: input.ClientMutationID, RunGate: gate, Problems: []Problem{}}
	return &RunGateMutationPayloadResolver{RunGateMutationPayload: payload}, nil
}

func overrideRunGateMutation(ctx context.Context, input *OverrideRunGateInput) (*RunGateMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	gateID, err := serviceCatalog.FetchModelID(ctx, input.GateID)
	if err != nil {
		return nil, err
	}

	gate, err := serviceCatalog.RunService.OverrideRunGate(ctx, gateID, input.Comment)
	if err != nil {
		return nil, err
	}

	payload := RunGateMutationPayload{ClientMutationID: input.ClientMutationID, RunGate: gate, Problems: []Problem{}}
	return &RunGateMutationPayloadResolver{RunGateMutationPayload: payload}, nil
}

/* RunGate loader */

const runGateLoaderKey = "runGate"

// RegisterRunGateLoader registers a run gate loader function
func RegisterRunGateLoader(collection *loader.Collection) {
	collection.Register(runGateLoaderKey, runGateBatchFunc)
}

func loadRunGate(ctx context.Context, id string) (*models.RunGate, error) {
	ldr, err := loader.Extract(ctx, runGateLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	gate, ok := data.(*models.RunGate)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return gate, nil
}

func runGateBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	gates, err := getServiceCatalog(ctx).RunService.GetRunGatesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range gates {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
